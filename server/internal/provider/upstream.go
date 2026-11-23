package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// ChatMessage 是发送给上游的一条对话消息。
type ChatMessage struct {
	Role       string     // system / user / assistant / tool
	Content    string     // 消息正文
	Images     []string   // 图片 data URL，仅当前用户消息携带
	ToolCalls  []ToolCall // assistant 发起的工具调用
	ToolCallID string     // role=tool 时对应的调用 ID
}

// ToolCall 是一次模型发起的工具调用。
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool 是 OpenAI 兼容的工具声明。
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction 描述工具名称、用途与参数 schema。
type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

// StreamEvent 是增量输出事件：reasoning 表示深度思考，content 表示正式回答。
type StreamEvent struct {
	Kind  string // reasoning / content
	Delta string // 增量文本
}

// CompletionUsage 是上游返回的 token 用量。
type CompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`     // 输入 tokens（含缓存命中）
	CompletionTokens int `json:"completion_tokens"` // 输出 tokens
	CachedTokens     int `json:"cached_tokens"`     // 其中命中缓存的输入 tokens
}

// ErrUpstream 表示上游调用失败，具体原因由返回值携带。
var ErrUpstream = errors.New("upstream request failed")

// StreamCompletion 调用 OpenAI 兼容的 /chat/completions 流式接口。
// 通过 onEvent 逐段回调；返回最终 token 用量。
// 注意：调用方必须处理 ctx 取消（客户端停止生成），本函数会立即关闭上游连接。
func (s *Service) StreamCompletion(
	ctx context.Context,
	p *Provider,
	upstreamModel string,
	messages []ChatMessage,
	deepThinking bool,
	reasoningEffort string,
	maxTokens int,
	firstByteTimeout time.Duration,
	onEvent func(StreamEvent) error,
) (CompletionUsage, error) {
	usage, _, err := s.StreamCompletionWithTools(ctx, p, upstreamModel, messages, deepThinking, reasoningEffort, maxTokens, firstByteTimeout, nil, onEvent)
	return usage, err
}

// StreamCompletionWithTools 与 StreamCompletion 相同，但额外支持工具调用。
func (s *Service) StreamCompletionWithTools(
	ctx context.Context,
	p *Provider,
	upstreamModel string,
	messages []ChatMessage,
	deepThinking bool,
	reasoningEffort string,
	maxTokens int,
	firstByteTimeout time.Duration,
	tools []Tool,
	onEvent func(StreamEvent) error,
) (CompletionUsage, []ToolCall, error) {
	usage := CompletionUsage{}
	toolCalls := make([]ToolCall, 0)
	key, err := DecryptAPIKey(s.secret, p.APIKeyEnc)
	if err != nil {
		return usage, toolCalls, fmt.Errorf("密钥解密失败，请重新配置 API Key")
	}
	if strings.TrimSpace(key) == "" {
		return usage, toolCalls, ErrNoAPIKey
	}
	if maxTokens <= 0 {
		maxTokens = 384000
	}
	// 深度思考与正文共用 max_tokens；预算太低会出现“只想完、没正文”
	if deepThinking && maxTokens < 8192 {
		maxTokens = 8192
	}
	if firstByteTimeout <= 0 {
		firstByteTimeout = 30 * time.Second
	}
	// 按厂商选择的接入方式分发：chat / responses / anthropic
	switch NormalizeAPIStyle(p.APIStyle) {
	case APIStyleResponses:
		return s.streamResponses(ctx, p, key, upstreamModel, messages, deepThinking, reasoningEffort, maxTokens, firstByteTimeout, tools, onEvent)
	case APIStyleAnthropic:
		return s.streamAnthropic(ctx, p, key, upstreamModel, messages, deepThinking, reasoningEffort, maxTokens, firstByteTimeout, tools, onEvent)
	}

	payload := map[string]any{
		"model":       upstreamModel,
		"messages":    toUpstreamMessages(messages),
		"stream":      true,
		"max_tokens":  maxTokens,
		"temperature": 0.7,
		// 让上游在最后一帧返回 usage，便于准确记账
		"stream_options": map[string]any{"include_usage": true},
	}
	if s.builtinSearchEnabled(ctx) {
		// DeepSeek 联网工具在 Chat Completions 里也支持，且不能与普通工具混用
		payload["tools"] = deepSeekWebTools(false)
	} else if len(tools) > 0 {
		payload["tools"] = tools
		payload["tool_choice"] = "auto"
	}
	if deepThinking {
		// DeepSeek 兼容 reasoning_effort；不支持的厂商会忽略该字段。
		// 强度由服务端配置统一控制，避免前端硬编码 high。
		effort := strings.TrimSpace(reasoningEffort)
		if effort == "" {
			effort = "medium"
		}
		payload["reasoning_effort"] = effort
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return usage, toolCalls, err
	}
	endpoint := strings.TrimRight(p.BaseURL, "/") + "/chat/completions"
	reqCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return usage, toolCalls, fmt.Errorf("上游地址不合法")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+key)
	// 自定义请求头：部分网关需要 HTTP-Referer / X-Title 之类的标识
	for name, value := range ProviderHeaders(p) {
		req.Header.Set(name, value)
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return usage, toolCalls, ctx.Err()
		}
		return usage, toolCalls, fmt.Errorf("%w: %s", ErrUpstream, sanitizeNetErr(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return usage, toolCalls, fmt.Errorf("%w: %s", ErrUpstream, upstreamStatusMessage(resp.StatusCode, body))
	}
	// 部分聚合网关会用 HTTP 200 + JSON error 返回上游故障；必须在进入 SSE 扫描前识别
	if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		if message := upstreamBodyError(body); message != "" {
			return usage, toolCalls, fmt.Errorf("%w: %s", ErrUpstream, message)
		}
		return usage, toolCalls, fmt.Errorf("%w: 上游返回了非流式响应", ErrUpstream)
	}

	// 空闲超时：只要上游还有数据（含 keep-alive 注释行）就不算超时；
	// 连续 idleTimeout 没有任何字节流过才中止。这样慢模型边想边搜不会被误杀，
	// 真正卡死的连接也不会让前端一直空等。
	var firstByte atomic.Bool
	var lastSeen atomic.Int64
	lastSeen.Store(time.Now().UnixNano())
	stalled := atomic.Bool{}
	streamDone := make(chan struct{})
	defer close(streamDone)
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-streamDone:
				return
			case <-ticker.C:
				idle := time.Since(time.Unix(0, lastSeen.Load()))
				if idle > firstByteTimeout {
					stalled.Store(true)
					cancel()
					return
				}
			}
		}
	}()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	finishReason := ""
	hadContent := false
	toolMap := make(map[int]*ToolCall)
	for scanner.Scan() {
		// 任何一行（包括 ": keep-alive"）都说明流还活着，重置空闲计时
		lastSeen.Store(time.Now().UnixNano())
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		if data == "" {
			continue
		}
		firstByte.Store(true)
		var chunk upstreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// 单帧解析失败不应中断整段回答
			continue
		}
		if chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0 {
			usage.PromptTokens = chunk.Usage.PromptTokens
			usage.CompletionTokens = chunk.Usage.CompletionTokens
			usage.CachedTokens = chunk.Usage.PromptCacheHit
			if chunk.Usage.PromptTokensDetails.CachedTokens > usage.CachedTokens {
				usage.CachedTokens = chunk.Usage.PromptTokensDetails.CachedTokens
			}
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.ReasoningContent != "" {
				if err := onEvent(StreamEvent{Kind: "reasoning", Delta: choice.Delta.ReasoningContent}); err != nil {
					return usage, toolCalls, err
				}
			}
			if choice.Delta.Content != "" {
				hadContent = true
				if err := onEvent(StreamEvent{Kind: "content", Delta: choice.Delta.Content}); err != nil {
					return usage, toolCalls, err
				}
			}
			for _, call := range choice.Delta.ToolCalls {
				current := toolMap[call.Index]
				if current == nil {
					current = &ToolCall{}
					toolMap[call.Index] = current
				}
				if call.ID != "" {
					current.ID = call.ID
				}
				if call.Function.Name != "" {
					current.Name = call.Function.Name
				}
				current.Arguments += call.Function.Arguments
			}
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
		}
	}
	if err := scanner.Err(); err != nil {
		if stalled.Load() {
			if !firstByte.Load() {
				return usage, toolCalls, fmt.Errorf("%w: 首字节超时（%s）", ErrUpstream, firstByteTimeout)
			}
			return usage, toolCalls, fmt.Errorf("%w: 超过 %s 没有收到新数据，已中止", ErrUpstream, firstByteTimeout)
		}
		if ctx.Err() != nil {
			if stalled.Load() {
				if !firstByte.Load() {
					return usage, toolCalls, fmt.Errorf("%w: 首字节超时（%s）", ErrUpstream, firstByteTimeout)
				}
				return usage, toolCalls, fmt.Errorf("%w: 超过 %s 没有收到新数据，已中止", ErrUpstream, firstByteTimeout)
			}
			return usage, toolCalls, ctx.Err()
		}
		return usage, toolCalls, fmt.Errorf("%w: 流式响应中断", ErrUpstream)
	}
	if !firstByte.Load() {
		return usage, toolCalls, fmt.Errorf("%w: 上游未返回任何内容", ErrUpstream)
	}
	indexes := make([]int, 0, len(toolMap))
	for index := range toolMap {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	for _, index := range indexes {
		toolCalls = append(toolCalls, *toolMap[index])
	}
	if finishReason == "length" && !hadContent && len(toolCalls) == 0 {
		return usage, toolCalls, fmt.Errorf("%w: 输出达到 max_tokens 上限，思考内容占满了预算", ErrUpstream)
	}
	return usage, toolCalls, nil
}

// upstreamChunk 是 OpenAI 兼容流式响应的最小结构。
type upstreamChunk struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`           // 正式回答增量
			ReasoningContent string `json:"reasoning_content"` // 深度思考增量
			ToolCalls        []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"` // stop / length
	} `json:"choices"`
	Usage struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		PromptCacheHit      int `json:"prompt_cache_hit_tokens"`
		PromptTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	} `json:"usage"` // 最后一帧的用量
}

// toUpstreamMessages 转换为上游协议需要的消息数组。
// 有图片时使用 OpenAI 兼容的多模态 content 数组；纯文本仍保持 string，兼容性最好。
func toUpstreamMessages(messages []ChatMessage) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		if m.Role == "" {
			continue
		}
		if m.Role == "tool" {
			out = append(out, map[string]any{
				"role":         "tool",
				"tool_call_id": m.ToolCallID,
				"content":      m.Content,
			})
			continue
		}
		if len(m.ToolCalls) > 0 {
			calls := make([]map[string]any, 0, len(m.ToolCalls))
			for _, call := range m.ToolCalls {
				calls = append(calls, map[string]any{
					"id":   call.ID,
					"type": "function",
					"function": map[string]any{
						"name":      call.Name,
						"arguments": call.Arguments,
					},
				})
			}
			out = append(out, map[string]any{"role": "assistant", "content": m.Content, "tool_calls": calls})
			continue
		}
		if len(m.Images) == 0 {
			if strings.TrimSpace(m.Content) == "" {
				continue
			}
			out = append(out, map[string]any{"role": m.Role, "content": m.Content})
			continue
		}
		text := strings.TrimSpace(m.Content)
		if text == "" {
			text = "请分析用户上传的图片。"
		}
		parts := make([]map[string]any, 0, len(m.Images)+1)
		parts = append(parts, map[string]any{"type": "text", "text": text})
		for _, image := range m.Images {
			parts = append(parts, map[string]any{
				"type":      "image_url",
				"image_url": map[string]any{"url": image},
			})
		}
		out = append(out, map[string]any{"role": m.Role, "content": parts})
	}
	return out
}

// upstreamStatusMessage 将上游 HTTP 状态映射为可读提示。
// 5xx 会带上服务商返回的具体原因：聚合网关常用 503 + model_unavailable 说明模型临时不可用，
// 这比一句“上游异常”对排查更有用。
// upstreamBodyError 解析 HTTP 200 响应里的 OpenAI 兼容错误体。
func upstreamBodyError(body []byte) string {
	var payload struct {
		Error struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	message := strings.TrimSpace(payload.Error.Message)
	if len([]rune(message)) > 240 {
		message = string([]rune(message)[:240])
	}
	return message
}

func upstreamStatusMessage(status int, body []byte) string {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return fmt.Sprintf("上游密钥无效或无权限（HTTP %d）", status)
	case status == http.StatusTooManyRequests:
		return "上游限流，请稍后再试（HTTP 429）"
	case status >= 500:
		if detail := upstreamBodyError(body); detail != "" {
			return fmt.Sprintf("上游服务异常（HTTP %d）：%s", status, detail)
		}
		return fmt.Sprintf("上游服务异常（HTTP %d）", status)
	default:
		return fmt.Sprintf("上游返回错误（HTTP %d）：%s", status, sanitizeBody(body))
	}
}
