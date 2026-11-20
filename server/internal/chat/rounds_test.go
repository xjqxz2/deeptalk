package chat

import (
	"context"
	"encoding/json"
	"fmt"
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

// TestGenerateAllowsManyToolRounds 校验模型可以连续多轮调用工具，
// 而不是像以前那样第二轮就被强制收口。
func TestGenerateAllowsManyToolRounds(t *testing.T) {
	const rounds = 6
	serverRounds := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverRounds++
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		if serverRounds <= rounds {
			payload, _ := json.Marshal(map[string]any{
				"choices": []map[string]any{{
					"delta": map[string]any{
						"tool_calls": []map[string]any{{
							"index": 0,
							"id":    fmt.Sprintf("call_%d", serverRounds),
							"type":  "function",
							"function": map[string]any{
								"name":      "web_search",
								"arguments": `{"query":"股票 涨幅榜"}`,
							},
						}},
					},
				}},
			})
			fmt.Fprintf(w, "data: %s\n\n", payload)
		} else {
			payload, _ := json.Marshal(map[string]any{
				"choices": []map[string]any{{"delta": map[string]any{"content": "最终回答。"}}},
			})
			fmt.Fprintf(w, "data: %s\n\n", payload)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	dsn := filepath.Join(t.TempDir(), "rounds.db") + "?_pragma=busy_timeout(5000)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	providers := provider.NewService(db, "s", true, nil, nil)
	enc, _ := provider.EncryptAPIKey("s", "sk")
	prov := &provider.Provider{Name: "fake", BaseURL: upstream.URL, APIKeyEnc: enc}
	model := &provider.Model{ModelID: "m", DisplayName: "M"}
	service := NewService(db, providers, nil, nil, nil, nil, Config{MaxOutputTokens: 1024, MaxToolRounds: 10})

	executed := 0
	var out strings.Builder
	_, _, err = service.generate(context.Background(), model, prov,
		[]provider.ChatMessage{{Role: RoleUser, Content: "股票"}},
		false, "", 1024, []provider.Tool{webSearchTool},
		func(call provider.ToolCall) (string, error) {
			executed++
			return `{"sources":[]}`, nil
		},
		func(kind, delta string) error {
			if kind == "content" {
				out.WriteString(delta)
			}
			return nil
		})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if executed != rounds {
		t.Fatalf("executed = %d, want %d", executed, rounds)
	}
	if !strings.Contains(out.String(), "最终回答") {
		t.Fatalf("missing final answer: %q", out.String())
	}
}
