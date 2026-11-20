package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

	result := testEndpoint(context.Background(), srv.URL, "sk-test", nil)
	if !result.OK {
		t.Fatalf("expected ok, got %+v", result)
	}
	if result.ModelsFound != 100 {
		t.Fatalf("expected 100 models, got %d", result.ModelsFound)
	}
}

// TestUpstreamStatusMessageKeepsDetail 校验 5xx 会保留服务商给的原因。
func TestUpstreamStatusMessageKeepsDetail(t *testing.T) {
	body := []byte(`{"error":{"message":"This model is temporarily unavailable. Please try again later.","code":"model_unavailable"}}`)
	got := upstreamStatusMessage(503, body)
	if !strings.Contains(got, "503") || !strings.Contains(got, "temporarily unavailable") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(upstreamStatusMessage(500, []byte("oops")), "oops") {
		t.Fatal("non-JSON body should not be echoed")
	}
}
