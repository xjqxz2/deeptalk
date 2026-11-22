package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"deeptalk/internal/setting"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestBuiltinSearchToolInResponses 校验 builtin 模式下 Responses 请求带 web_search 工具。
func TestBuiltinSearchToolInResponses(t *testing.T) {
	var gotTools []any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		if list, ok := payload["tools"].([]any); ok {
			gotTools = list
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
	}))
	defer srv.Close()

	db, _ := gorm.Open(sqlite.Open("file:builtin?mode=memory&cache=shared"), &gorm.Config{})
	_ = setting.Migrate(db)
	settings := setting.NewService(db)
	_ = settings.Set(context.Background(), setting.KeySearchProvider, "builtin")
	s := &Service{settings: settings}

	_, _, err := s.streamResponses(context.Background(), testProvider(srv.URL, APIStyleResponses), "k",
		"m", []ChatMessage{{Role: "user", Content: "hi"}}, false, "", 1024, 5*time.Second, nil,
		func(StreamEvent) error { return nil })
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	found := false
	for _, item := range gotTools {
		if m, ok := item.(map[string]any); ok && m["type"] == "web_search" {
			found = true
		}
	}
	if !found {
		t.Fatalf("tools = %+v, want a web_search tool", gotTools)
	}
}
