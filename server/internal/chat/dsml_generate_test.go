package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"deeptalk/internal/provider"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestGenerateHandlesDSMLToolCalls 端到端验证：
// 上游把工具调用写进正文时，用户看不到标记，并且会用工具结果再答一次。
func TestGenerateHandlesDSMLToolCalls(t *testing.T) {
	var round int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		round++
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		send := func(content string) {
			payload, _ := json.Marshal(map[string]any{
				"choices": []map[string]any{{"delta": map[string]any{"content": content}}},
			})
			fmt.Fprintf(w, "data: %s\n\n", payload)
		}
		if strings.Contains(string(body), "我调用了工具") {
			send("综合搜索结果，最便宜的是豆包 Seedream。")
		} else {
			send("我来查一下各家生图模型的价格。")
			send(dsmlSample[strings.Index(dsmlSample, "<｜｜DSML｜｜"):])
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	dsn := filepath.Join(t.TempDir(), "dsml.db") + "?_pragma=busy_timeout(5000)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	providers := provider.NewService(db, "test-secret", true, nil, nil)
	enc, err := provider.EncryptAPIKey("test-secret", "sk-test")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	prov := &provider.Provider{Name: "fake", BaseURL: upstream.URL, APIKeyEnc: enc}
	model := &provider.Model{ModelID: "fake-model", DisplayName: "Fake"}
	service := NewService(db, providers, nil, nil, nil, nil, Config{MaxOutputTokens: 1024})

	var visible strings.Builder
	toolCalls := 0
	_, _, err = service.generate(context.Background(), model, prov,
		[]provider.ChatMessage{{Role: RoleUser, Content: "找个便宜的生图模型"}},
		false, "", 1024, []provider.Tool{webSearchTool},
		func(call provider.ToolCall) (string, error) {
			toolCalls++
			if call.Name != "web_search" {
				t.Fatalf("unexpected tool: %s", call.Name)
			}
			return `{"sources":[{"title":"豆包","url":"https://example.com"}]}`, nil
		},
		func(kind, delta string) error {
			if kind == "content" {
				visible.WriteString(delta)
			}
			return nil
		})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if toolCalls != 1 {
		t.Fatalf("tool calls = %d, want 1", toolCalls)
	}
	if round < 2 {
		t.Fatalf("rounds = %d, want at least 2", round)
	}
	text := visible.String()
	if strings.Contains(text, "DSML") || strings.Contains(text, "invoke") {
		t.Fatalf("markup leaked: %q", text)
	}
	if !strings.Contains(text, "最便宜的是豆包") {
		t.Fatalf("second round answer missing: %q", text)
	}
}
