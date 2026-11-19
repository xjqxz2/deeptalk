package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestTestEndpointLargeModelList 回归：AUTODL 等网关会返回几十个模型，响应体超过 4KB。
// 之前读取上限只有 4096 字节，JSON 被截断后会误报“不是 OpenAI 兼容的模型列表”。
func TestTestEndpointLargeModelList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"object":"list","success":true,"data":[`)
		for i := 0; i < 100; i++ {
			if i > 0 {
				_, _ = fmt.Fprint(w, ",")
			}
			_, _ = fmt.Fprintf(w, `{"id":"model-%03d","object":"model","owned_by":"custom","member_discount":0.5}`, i)
		}
		_, _ = fmt.Fprint(w, `]}`)
	}))
	defer srv.Close()

	result := testEndpoint(context.Background(), srv.URL, "sk-test")
	if !result.OK {
		t.Fatalf("expected ok, got %+v", result)
	}
	if result.ModelsFound != 100 {
		t.Fatalf("expected 100 models, got %d", result.ModelsFound)
	}
}
