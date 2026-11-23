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

// TestIdleTimeoutAbortsStalledStream 校验「180s 无数据」这类空闲超时会被触发。
func TestIdleTimeoutAbortsStalledStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"开头"}}]}`+"\n\n")
		w.(http.Flusher).Flush()
		time.Sleep(1500 * time.Millisecond) // 之后没有任何数据
	}))
	defer srv.Close()

	s := &Service{secret: "test-secret"}
	enc, _ := EncryptAPIKey("test-secret", "k")
	prov := testProvider(srv.URL, APIStyleChat)
	prov.APIKeyEnc = enc

	start := time.Now()
	_, _, err := s.StreamCompletionWithTools(context.Background(), prov, "m",
		[]ChatMessage{{Role: "user", Content: "hi"}}, false, "", 1024, 300*time.Millisecond, nil,
		func(StreamEvent) error { return nil })
	if err == nil {
		t.Fatal("expected idle timeout error")
	}
	if !strings.Contains(err.Error(), "没有收到新数据") && !strings.Contains(err.Error(), "首字节超时") {
		t.Fatalf("err = %v", err)
	}
	if elapsed := time.Since(start); elapsed > 1200*time.Millisecond {
		t.Fatalf("took %v, idle timeout should fire fast", elapsed)
	}
}

// TestIdleTimeoutIgnoredWhileDataFlows 校验持续有数据时不会超时。
func TestIdleTimeoutIgnoredWhileDataFlows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for i := 0; i < 5; i++ {
			fmt.Fprintf(w, ": keep-alive\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"块%d\"}}]}\n\n", i)
			w.(http.Flusher).Flush()
			time.Sleep(120 * time.Millisecond)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	s := &Service{secret: "test-secret"}
	enc, _ := EncryptAPIKey("test-secret", "k")
	prov := testProvider(srv.URL, APIStyleChat)
	prov.APIKeyEnc = enc

	var out strings.Builder
	_, _, err := s.StreamCompletionWithTools(context.Background(), prov, "m",
		[]ChatMessage{{Role: "user", Content: "hi"}}, false, "", 1024, 400*time.Millisecond, nil,
		func(ev StreamEvent) error { out.WriteString(ev.Delta); return nil })
	if err != nil {
		t.Fatalf("stream should survive while data flows: %v", err)
	}
	if !strings.Contains(out.String(), "块4") {
		t.Fatalf("content = %q", out.String())
	}
}
