package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// 三种接入方式共用同一套事件映射：正文、思考、工具调用、用量。
// 这里只做协议转换，重试、超时、埋点仍由上层统一处理。

// buildAnthropicMessages 把内部消息转成 Anthropic Messages 结构。
// Anthropic 没有 tool 角色，工具结果要放进 user 消息的 tool_result 块里。
func buildAnthropicMessages(messages []ChatMessage) (system string, out []map[string]any) {
	for _, m := range messages {
		switch m.Role {
		case "system":
			if system != "" {
				system += "\n\n"
			}
			system += m.Content
		case "tool":
			out = append(out, map[string]any{
				"role": "user",
				"content": []map[string]any{{
					"type":        "tool_result",
					"tool_use_id": m.ToolCallID,
					"content":     m.Content,
				}},
			})
		case "assistant":
			blocks := make([]map[string]any, 0, 2)
			if strings.TrimSpace(m.Content) != "" {
				blocks = append(blocks, map[string]any{"type": "text", "text": m.Content})
			}
			for _, call := range m.ToolCalls {
				var input any = map[string]any{}
				if strings.TrimSpace(call.Arguments) != "" {
					_ = json.Unmarshal([]byte(call.Arguments), &input)
				}
				blocks = append(blocks, map[string]any{
					"type": "tool_use", "id": call.ID, "name": call.Name, "input": input,
				})
			}
			if len(blocks) == 0 {
				blocks = append(blocks, map[string]any{"type": "text", "text": ""})
			}
			out = append(out, map[string]any{"role": "assistant", "content": blocks})
		default: // user
			blocks := make([]map[string]any, 0, 1+len(m.Images))
			for _, image := range m.Images {
				if parsed := parseDataURL(image); parsed != nil {
					blocks = append(blocks, parsed)
				}
			}
			if m.Content != "" || len(blocks) == 0 {
				blocks = append(blocks, map[string]any{"type": "text", "text": m.Content})
			}
			out = append(out, map[string]any{"role": "user", "content": blocks})
		}
	}
	return system, out
}

// parseDataURL 把 data:image/...;base64,xxx 转成 Anthropic 图片块。
func parseDataURL(raw string) map[string]any {
	rest, ok := strings.CutPrefix(raw, "data:")
	if !ok {
		return nil
	}
	meta, data, ok := strings.Cut(rest, ",")
	if !ok || !strings.Contains(meta, "base64") {
		return nil
	}
	mediaType, _, _ := strings.Cut(meta, ";")
	return map[string]any{
		"type": "image",
		"source": map[string]any{
			"type": "base64", "media_type": mediaType, "data": data,
		},
	}
}

// buildResponsesInput 把内部消息转成 Responses API 的 input 数组。
func buildResponsesInput(messages []ChatMessage) (instructions string, input []map[string]any) {
	for _, m := range messages {
		switch m.Role {
		case "system":
			if instructions != "" {
				instructions += "\n\n"
			}
			instructions += m.Content
		case "tool":
			input = append(input, map[string]any{
				"type": "function_call_output", "call_id": m.ToolCallID, "output": m.Content,
			})
		case "assistant":
			if strings.TrimSpace(m.Content) != "" {
				input = append(input, map[string]any{
					"role":    "assistant",
					"content": []map[string]any{{"type": "output_text", "text": m.Content}},
				})
			}
			for _, call := range m.ToolCalls {
				input = append(input, map[string]any{
					"type": "function_call", "call_id": call.ID, "name": call.Name, "arguments": call.Arguments,
				})
			}
		default:
			parts := make([]map[string]any, 0, 1+len(m.Images))
			for _, image := range m.Images {
				parts = append(parts, map[string]any{"type": "input_image", "image_url": image})
			}
			if m.Content != "" || len(parts) == 0 {
				parts = append(parts, map[string]any{"type": "input_text", "text": m.Content})
			}
			input = append(input, map[string]any{"role": "user", "content": parts})
		}
	}
	return instructions, input
}

// responsesTools 把内部工具定义转成 Responses API 格式（function 是扁平结构）。
func responsesTools(tools []Tool) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		out = append(out, map[string]any{
			"type": "function", "name": tool.Function.Name,
			"description": tool.Function.Description, "parameters": tool.Function.Parameters,
		})
	}
	return out
}

// anthropicTools 把内部工具定义转成 Anthropic 格式（input_schema）。
func anthropicTools(tools []Tool) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		out = append(out, map[string]any{
			"name": tool.Function.Name, "description": tool.Function.Description,
			"input_schema": tool.Function.Parameters,
		})
	}
	return out
}

// sseLines 逐行读取上游 SSE，把 data 交给回调；空行与注释跳过。
func sseLines(body io.Reader, onData func(string) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") || !strings.HasPrefix(line, "data:") {
			continue
		}
		if err := onData(strings.TrimSpace(strings.TrimPrefix(line, "data:"))); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// postStream 发起一次流式请求并返回响应体（已校验状态码与 Content-Type）。
func (s *Service) postStream(ctx context.Context, p *Provider, endpoint, key string, payload any, auth func(*http.Request), timeout time.Duration, onEvent func(StreamEvent) error) (io.ReadCloser, *int32, func(), error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, nil, err
	}
	reqCtx, cancel := context.WithCancel(ctx)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		cancel()
		return nil, nil, nil, fmt.Errorf("上游地址不合法")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	auth(req)
	for name, value := range ProviderHeaders(p) {
		req.Header.Set(name, value)
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		cancel()
		if ctx.Err() != nil {
			return nil, nil, nil, ctx.Err()
		}
		return nil, nil, nil, fmt.Errorf("%w: %s", ErrUpstream, sanitizeNetErr(err))
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		cancel()
		return nil, nil, nil, fmt.Errorf("%w: %s", ErrUpstream, upstreamStatusMessage(resp.StatusCode, body))
	}
	firstByte := new(int32)
	stop := make(chan struct{})
	if timeout > 0 {
		go func() {
			select {
			case <-time.After(timeout):
				if atomic.LoadInt32(firstByte) == 0 {
					cancel()
				}
			case <-stop:
			}
		}()
	}
	cleanup := func() {
		close(stop)
		cancel()
	}
	return resp.Body, firstByte, cleanup, nil
}

// toolCallCollector 按 index 聚合流式工具调用参数。
type toolCallCollector struct {
	order []int
	calls map[int]*ToolCall
}

func newToolCollector() *toolCallCollector {
	return &toolCallCollector{calls: map[int]*ToolCall{}}
}

func (c *toolCallCollector) set(index int, id, name, argsDelta string) {
	call, ok := c.calls[index]
	if !ok {
		call = &ToolCall{}
		c.calls[index] = call
		c.order = append(c.order, index)
	}
	if id != "" {
		call.ID = id
	}
	if name != "" {
		call.Name = name
	}
	call.Arguments += argsDelta
}

func (c *toolCallCollector) all() []ToolCall {
	sort.Ints(c.order)
	out := make([]ToolCall, 0, len(c.order))
	for _, index := range c.order {
		out = append(out, *c.calls[index])
	}
	return out
}

// builtinSearchEnabled 决定是否把「模型内置联网」工具挂进请求。
// 由后台的 search_provider 控制：self/deepseek 走我们自己的检索，builtin 交给模型。
func (s *Service) builtinSearchEnabled(ctx context.Context) bool {
	if s.settings == nil {
		return false
	}
	return s.settings.Get(ctx, "search_provider", "self") == "builtin"
}
