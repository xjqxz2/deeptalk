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

// responses 协议里我们需要的最小事件结构。
type responsesEvent struct {
	Type     string `json:"type"`
	Delta    string `json:"delta"`
	Text     string `json:"text"`
	ItemID   string `json:"item_id"`
	Argument string `json:"arguments"`
	Item     struct {
		Type   string `json:"type"`
		ID     string `json:"id"`
		CallID string `json:"call_id"`
		Name   string `json:"name"`
	} `json:"item"`
	Response struct {
		Usage struct {
			InputTokens        int `json:"input_tokens"`
			OutputTokens       int `json:"output_tokens"`
			InputTokensDetails struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"input_tokens_details"`
		} `json:"usage"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	} `json:"response"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// streamResponses 走 OpenAI Responses API（/responses）。
// 事件映射：output_text → 正文，reasoning_summary_text → 思考，
// function_call_arguments → 工具参数，completed → 用量。
func (s *Service) streamResponses(
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
	instructions, input := buildResponsesInput(messages)
	payload := map[string]any{
		"model":             upstreamModel,
		"input":             input,
		"stream":            true,
		"max_output_tokens": maxTokens,
	}
	if instructions != "" {
		payload["instructions"] = instructions
	}
	if len(tools) > 0 {
		payload["tools"] = responsesTools(tools)
		payload["tool_choice"] = "auto"
	}
	if deepThinking {
		effort := strings.TrimSpace(reasoningEffort)
		if effort == "" {
			effort = "medium"
		}
		payload["reasoning"] = map[string]any{"effort": effort, "summary": "auto"}
	}
	endpoint := strings.TrimRight(p.BaseURL, "/") + "/responses"
	body, firstByte, cleanup, err := s.postStream(ctx, p, endpoint, key, payload, func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+key)
	}, firstByteTimeout, onEvent)
	if err != nil {
		return usage, nil, err
	}
	defer body.Close()
	defer cleanup()

	collector := newToolCollector()
	var callIndex int
	err = sseLines(body, func(data string) error {
		if data == "[DONE]" {
			return nil
		}
		var event responsesEvent
		if json.Unmarshal([]byte(data), &event) != nil {
			return nil
		}
		atomic.StoreInt32(firstByte, 1)
		switch event.Type {
		case "response.output_text.delta":
			if event.Delta != "" {
				return onEvent(StreamEvent{Kind: "content", Delta: event.Delta})
			}
		case "response.reasoning_summary_text.delta", "response.reasoning_text.delta":
			if event.Delta != "" {
				return onEvent(StreamEvent{Kind: "reasoning", Delta: event.Delta})
			}
		case "response.output_item.added":
			if event.Item.Type == "function_call" {
				callIndex++
				id := event.Item.CallID
				if id == "" {
					id = event.Item.ID
				}
				collector.set(callIndex, id, event.Item.Name, "")
			}
		case "response.function_call_arguments.delta":
			delta := event.Delta
			if delta == "" {
				delta = event.Argument
			}
			if callIndex > 0 {
				collector.set(callIndex, "", "", delta)
			}
		case "response.completed":
			usage.PromptTokens = event.Response.Usage.InputTokens
			usage.CompletionTokens = event.Response.Usage.OutputTokens
			usage.CachedTokens = event.Response.Usage.InputTokensDetails.CachedTokens
		case "response.failed", "error":
			message := event.Response.Error.Message
			if message == "" {
				message = event.Error.Message
			}
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
