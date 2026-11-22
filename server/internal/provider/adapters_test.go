package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testProvider(baseURL, style string) *Provider {
	return &Provider{Name: "test", BaseURL: baseURL, APIStyle: style, TimeoutSec: 30}
}

// TestStreamResponses 校验 Responses API 的正文、思考、工具调用与用量映射。
func TestStreamResponses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Errorf("path = %s, want /responses", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer k-test" {
			t.Errorf("auth = %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		frames := []string{
			`{"type":"response.reasoning_summary_text.delta","delta":"先想想"}`,
			`{"type":"response.output_item.added","item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"web_search"}}`,
			`{"type":"response.function_call_arguments.delta","delta":"{\"queries\":[\"a\"]}"}`,
			`{"type":"response.output_text.delta","delta":"你好"}`,
			`{"type":"response.completed","response":{"usage":{"input_tokens":11,"output_tokens":7}}}`,
		}
		for _, frame := range frames {
			fmt.Fprintf(w, "data: %s\n\n", frame)
		}
	}))
	defer srv.Close()

	s := &Service{}
	var content, reasoning strings.Builder
	usage, calls, err := s.streamResponses(context.Background(), testProvider(srv.URL, APIStyleResponses), "k-test",
		"m", []ChatMessage{{Role: "user", Content: "hi"}}, false, "", 1024, 5*time.Second, nil,
		func(ev StreamEvent) error {
			switch ev.Kind {
			case "content":
				content.WriteString(ev.Delta)
			case "reasoning":
				reasoning.WriteString(ev.Delta)
			}
			return nil
		})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if content.String() != "你好" || reasoning.String() != "先想想" {
		t.Fatalf("content=%q reasoning=%q", content.String(), reasoning.String())
	}
	if len(calls) != 1 || calls[0].Name != "web_search" || calls[0].ID != "call_1" {
		t.Fatalf("calls = %+v", calls)
	}
	if !strings.Contains(calls[0].Arguments, "queries") {
		t.Fatalf("arguments = %q", calls[0].Arguments)
	}
	if usage.PromptTokens != 11 || usage.CompletionTokens != 7 {
		t.Fatalf("usage = %+v", usage)
	}
}

// TestStreamAnthropic 校验 Messages API 的正文、思考、工具调用与用量映射。
func TestStreamAnthropic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("path = %s, want /v1/messages", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "k-ant" {
			t.Errorf("x-api-key = %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Error("missing anthropic-version")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		frames := []string{
			`{"type":"message_start","message":{"usage":{"input_tokens":9}}}`,
			`{"type":"content_block_start","index":0,"content_block":{"type":"thinking"}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"推理一下"}}`,
			`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"web_search"}}`,
			`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"queries\":[\"b\"]}"}}`,
			`{"type":"content_block_start","index":2,"content_block":{"type":"text"}}`,
			`{"type":"content_block_delta","index":2,"delta":{"type":"text_delta","text":"世界"}}`,
			`{"type":"message_delta","usage":{"output_tokens":5}}`,
			`{"type":"message_stop"}`,
		}
		for _, frame := range frames {
			fmt.Fprintf(w, "data: %s\n\n", frame)
		}
	}))
	defer srv.Close()

	s := &Service{}
	var content, reasoning strings.Builder
	usage, calls, err := s.streamAnthropic(context.Background(), testProvider(srv.URL, APIStyleAnthropic), "k-ant",
		"claude", []ChatMessage{{Role: "system", Content: "sys"}, {Role: "user", Content: "hi"}}, true, "", 2048, 5*time.Second, nil,
		func(ev StreamEvent) error {
			switch ev.Kind {
			case "content":
				content.WriteString(ev.Delta)
			case "reasoning":
				reasoning.WriteString(ev.Delta)
			}
			return nil
		})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if content.String() != "世界" || reasoning.String() != "推理一下" {
		t.Fatalf("content=%q reasoning=%q", content.String(), reasoning.String())
	}
	if len(calls) != 1 || calls[0].Name != "web_search" || !strings.Contains(calls[0].Arguments, "queries") {
		t.Fatalf("calls = %+v", calls)
	}
	if usage.PromptTokens != 9 || usage.CompletionTokens != 5 {
		t.Fatalf("usage = %+v", usage)
	}
}

// TestStreamChatStyleKeepsOpenAIWire 校验默认 chat 方式仍发 /chat/completions。
func TestStreamChatStyleKeepsOpenAIWire(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	s := &Service{secret: "test-secret"}
	enc, encErr := EncryptAPIKey("test-secret", "k-chat")
	if encErr != nil {
		t.Fatalf("encrypt: %v", encErr)
	}
	prov := testProvider(srv.URL, APIStyleChat)
	prov.APIKeyEnc = enc
	var out strings.Builder
	_, _, err := s.StreamCompletionWithTools(context.Background(), prov, "m",
		[]ChatMessage{{Role: "user", Content: "hi"}}, false, "", 1024, 5*time.Second, nil,
		func(ev StreamEvent) error { out.WriteString(ev.Delta); return nil })
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if out.String() != "ok" {
		t.Fatalf("content = %q", out.String())
	}
}
