package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestDeepSeekSearchMapping 校验 DeepSeek Search API 的请求与结果映射。
func TestDeepSeekSearchMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ds-key" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["query"] != "测试关键词" || body["need_content"] != true {
			t.Errorf("body = %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{{
				"title": "标题", "url": "https://example.com/a",
				"snippet": "短摘要", "summary": "长摘要内容", "content": "",
			}},
		})
	}))
	defer srv.Close()

	svc := New(Config{Enabled: true, Timeout: 5 * time.Second, PageTimeout: 5 * time.Second})
	// 测试直接打本地服务：临时替换端点
	svc.cfg.MaxResults = 10
	items, err := svc.searchDeepSeekAt(context.Background(), srv.URL, "ds-key", "测试关键词")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(items) != 1 || items[0].URL != "https://example.com/a" || items[0].Snippet == "" {
		t.Fatalf("items = %+v", items)
	}
}

// TestDeepSeekProviderSwitch 校验选择 deepseek 时才走官方接口。
func TestDeepSeekProviderSwitch(t *testing.T) {
	svc := New(Config{Enabled: true, Timeout: time.Second, PageTimeout: time.Second})
	svc.SetDeepSeekProvider(func(ctx context.Context) (string, string) {
		return ProviderSelf, ""
	})
	if key := svc.deepseekKey(context.Background()); key != "" {
		t.Fatalf("self mode should not return a key, got %q", key)
	}
	svc.SetDeepSeekProvider(func(ctx context.Context) (string, string) {
		return ProviderDeepSeek, "ds-key"
	})
	if key := svc.deepseekKey(context.Background()); key != "ds-key" {
		t.Fatalf("deepseek mode key = %q", key)
	}
}
