package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMarshalHeadersValidates 校验请求头名称与换行会被拦截。
func TestMarshalHeadersValidates(t *testing.T) {
	raw, err := marshalHeaders(map[string]string{
		"HTTP-Referer":   "https://deeptalk.peon.top",
		"X-Title":        "DeepTalk",
		"Content-Length": "123", // 客户端管理的头应被忽略
	})
	if err != nil {
		t.Fatalf("marshalHeaders: %v", err)
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed["HTTP-Referer"] != "https://deeptalk.peon.top" || parsed["X-Title"] != "DeepTalk" {
		t.Fatalf("parsed = %+v", parsed)
	}
	if _, ok := parsed["Content-Length"]; ok {
		t.Fatal("Content-Length should be dropped")
	}

	if _, err := marshalHeaders(map[string]string{"Bad Name": "x"}); err == nil {
		t.Fatal("header name with space should be rejected")
	}
	if _, err := marshalHeaders(map[string]string{"X-Test": "a\r\nInjected: 1"}); err == nil {
		t.Fatal("newlines in value should be rejected")
	}
	if got, err := marshalHeaders(nil); err != nil || got != "" {
		t.Fatalf("empty headers = %q, %v", got, err)
	}
}

// TestStreamAppliesCustomHeaders 校验调用上游时会带上自定义请求头。
func TestStreamAppliesCustomHeaders(t *testing.T) {
	var gotReferer, gotTitle, gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReferer = r.Header.Get("HTTP-Referer")
		gotTitle = r.Header.Get("X-Title")
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n"))
	}))
	defer server.Close()

	service := NewService(nil, "test-secret", true, nil, nil)
	enc, err := EncryptAPIKey("test-secret", "sk-test")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	headersJSON, err := marshalHeaders(map[string]string{
		"HTTP-Referer": "https://deeptalk.peon.top",
		"X-Title":      "DeepTalk",
	})
	if err != nil {
		t.Fatalf("marshalHeaders: %v", err)
	}
	prov := &Provider{Name: "fake", BaseURL: server.URL, APIKeyEnc: enc, HeadersJSON: headersJSON}

	_, _, err = service.StreamCompletionWithTools(context.Background(), prov, "m",
		[]ChatMessage{{Role: "user", Content: "hi"}}, false, "", 16, 0, nil, func(StreamEvent) error { return nil })
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if gotReferer != "https://deeptalk.peon.top" || gotTitle != "DeepTalk" {
		t.Fatalf("headers not applied: referer=%q title=%q", gotReferer, gotTitle)
	}
	if !strings.HasPrefix(gotAuth, "Bearer ") {
		t.Fatalf("authorization = %q", gotAuth)
	}
}
