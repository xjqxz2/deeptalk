package chat

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"deeptalk/internal/provider"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestStartTitleSummaryUsesModel 用本地假上游验证标题确实来自模型，并且会被清洗。
func TestStartTitleSummaryUsesModel(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		chunks := []string{
			`{"choices":[{"delta":{"content":"标题：\"零信任组网"}}]}`,
			`{"choices":[{"delta":{"content":"方案\"。"}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
		}
		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			if flusher != nil {
				flusher.Flush()
			}
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	dsn := filepath.Join(t.TempDir(), "title.db") + "?_pragma=busy_timeout(5000)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	providers := provider.NewService(db, "test-secret", true, nil, nil)
	enc, err := provider.EncryptAPIKey("test-secret", "sk-test")
	if err != nil {
		t.Fatalf("encrypt key: %v", err)
	}
	prov := &provider.Provider{Name: "fake", BaseURL: upstream.URL, APIKeyEnc: enc}
	service := NewService(db, providers, nil, nil, nil, nil, Config{})

	ch := service.startTitleSummary(context.Background(), prov, "deepseek-chat", "帮我看看企业内网的零信任组网怎么选")
	if ch == nil {
		t.Fatal("first round should start title summary")
	}
	select {
	case got := <-ch:
		if got != "零信任组网方案" {
			t.Fatalf("title = %q, want 零信任组网方案", got)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("title summary timed out")
	}

	// 没有厂商或没有问题文本时不发起请求
	if ch := service.startTitleSummary(context.Background(), nil, "m", "问题"); ch != nil {
		t.Fatal("nil provider should not start summary")
	}
	if ch := service.startTitleSummary(context.Background(), prov, "m", "   "); ch != nil {
		t.Fatal("blank seed should not start summary")
	}
}
