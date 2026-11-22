package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// anthropicEvent 是 Messages API 流式事件里我们需要的最小结构。
type anthropicEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		Thinking    string `json:"thinking"`
		PartialJSON string `json:"partial_json"`
		StopReason  string `json:"stop_reason"`
	} `json:"delta"`
	ContentBlock struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"content_block"`
	Message struct {
		Usage struct {
			InputTokens          int `json:"input_tokens"`
			CacheReadInputTokens int `json:"cache_read_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// streamAnthropic 走 Anthropic Messages API（/v1/messages）。
func (s *Service) streamAnthropic(
	ctx context.Context,
	p *Provider,
	key string,
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
	system, input := buildAnthropicMessages(messages)
	payload := map[string]any{
		"model":      upstreamModel,
		"messages":   input,
		"max_tokens": maxTokens,
		"stream":     true,
	}
	if system != "" {
		payload["system"] = system
	}
	toolList := anthropicTools(tools)
	if s.builtinSearchEnabled(ctx) {
		// 模型内置联网：Anthropic 服务端 web_search 工具
		toolList = append(toolList, map[string]any{
			"type": "web_search_20250305", "name": "web_search", "max_uses": 5,
		})
	}
	if len(toolList) > 0 {
		payload["tools"] = toolList
	}
	if deepThinking {
		budget := maxTokens / 2
		if budget < 2048 {
			budget = 2048
		}
		payload["thinking"] = map[string]any{"type": "enabled", "budget_tokens": budget}
	}
	endpoint := strings.TrimRight(p.BaseURL, "/")
	if !strings.HasSuffix(endpoint, "/v1") {
		endpoint += "/v1"
	}
	endpoint += "/messages"

	body, firstByte, cleanup, err := s.postStream(ctx, p, endpoint, key, payload, func(req *http.Request) {
		req.Header.Set("x-api-key", key)
		req.Header.Set("anthropic-version", "2023-06-01")
	}, firstByteTimeout, onEvent)
	if err != nil {
		return usage, nil, err
	}
	defer body.Close()
	defer cleanup()

	collector := newToolCollector()
	// Anthropic 用 content block 的 index 区分工具调用，顺序与出现顺序一致。
	blockToTool := map[int]int{}
	err = sseLines(body, func(data string) error {
		var event anthropicEvent
		if json.Unmarshal([]byte(data), &event) != nil {
			return nil
		}
		atomic.StoreInt32(firstByte, 1)
		switch event.Type {
		case "message_start":
			usage.PromptTokens = event.Message.Usage.InputTokens
			usage.CachedTokens = event.Message.Usage.CacheReadInputTokens
		case "content_block_start":
			if event.ContentBlock.Type == "tool_use" {
				blockToTool[event.Index] = event.Index
				collector.set(event.Index, event.ContentBlock.ID, event.ContentBlock.Name, "")
			}
		case "content_block_delta":
			switch event.Delta.Type {
			case "text_delta":
				if event.Delta.Text != "" {
					return onEvent(StreamEvent{Kind: "content", Delta: event.Delta.Text})
				}
			case "thinking_delta":
				if event.Delta.Thinking != "" {
					return onEvent(StreamEvent{Kind: "reasoning", Delta: event.Delta.Thinking})
				}
			case "input_json_delta":
				if _, ok := blockToTool[event.Index]; ok {
					collector.set(event.Index, "", "", event.Delta.PartialJSON)
				}
			}
		case "message_delta":
			if event.Usage.OutputTokens > 0 {
				usage.CompletionTokens = event.Usage.OutputTokens
			}
		case "error":
			message := event.Error.Message
			if message == "" {
				message = "上游返回失败"
			}
			return fmt.Errorf("%w: %s", ErrUpstream, message)
		}
		return nil
	})
	if err != nil {
		if ctx.Err() != nil {
			return usage, nil, ctx.Err()
		}
		return usage, collector.all(), err
	}
	return usage, collector.all(), nil
}
