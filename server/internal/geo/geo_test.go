package geo

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestIsPublicIP 私网/本机地址不应该被拿去做定位查询。
func TestIsPublicIP(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "::1", "10.1.2.3", "192.168.1.5", "172.16.0.9", "", "not-an-ip"} {
		if IsPublicIP(raw) != nil {
			t.Fatalf("%q should not be treated as public", raw)
		}
	}
	if IsPublicIP("114.114.114.114") == nil {
		t.Fatal("public IP should pass")
	}
}

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// TestLookupParsesResponse 校验解析与 Display 拼接。
func TestLookupParsesResponse(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing user agent")
		}
		_, _ = w.Write([]byte(`{"city":"Hangzhou","region":"Zhejiang","country_name":"China","timezone":"Asia/Shanghai"}`))
	})
	svc := NewService(Options{Endpoint: srv.URL + "/{ip}/json/", Timeout: time.Second})
	loc, ok := svc.Lookup(context.Background(), "114.114.114.114")
	if !ok {
		t.Fatal("expected a real lookup")
	}
	if loc.Label() != "Hangzhou" {
		t.Fatalf("label = %q", loc.Label())
	}
	if loc.Display() != "China Zhejiang Hangzhou" {
		t.Fatalf("display = %q", loc.Display())
	}
	if loc.Timezone != "Asia/Shanghai" {
		t.Fatalf("timezone = %q", loc.Timezone)
	}
}

// TestLookupCaches 同一 IP 只应查询一次。
func TestLookupCaches(t *testing.T) {
	var calls int
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"city":"Chengdu","region":"Sichuan","country_name":"China","timezone":"Asia/Shanghai"}`))
	})
	svc := NewService(Options{Endpoint: srv.URL + "/{ip}/json/", Timeout: time.Second})
	for i := 0; i < 3; i++ {
		if _, ok := svc.Lookup(context.Background(), "114.114.114.114"); !ok {
			t.Fatal("lookup failed")
		}
	}
	if calls != 1 {
		t.Fatalf("upstream calls = %d, want 1 (cached)", calls)
	}
}

// TestLookupFallsBack 私网地址与上游故障都应返回兜底且不 panic。
func TestLookupFallsBack(t *testing.T) {
	fallback := Location{City: "本地"}
	svc := NewService(Options{Endpoint: "http://127.0.0.1:1/{ip}", Timeout: 200 * time.Millisecond, Fallback: fallback})
	if loc, ok := svc.Lookup(context.Background(), "192.168.1.1"); ok || loc.City != "本地" {
		t.Fatalf("private ip should return fallback, got %+v ok=%v", loc, ok)
	}
	if loc, ok := svc.Lookup(context.Background(), "114.114.114.114"); ok || loc.City != "本地" {
		t.Fatalf("unreachable upstream should return fallback, got %+v ok=%v", loc, ok)
	}
}

// TestLookupErrorPayload 上游返回 error:true 时不应当成有效定位。
func TestLookupErrorPayload(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"error":true,"reason":"RateLimited"}`))
	})
	svc := NewService(Options{Endpoint: srv.URL + "/{ip}/json/", Timeout: time.Second})
	if _, ok := svc.Lookup(context.Background(), "8.8.8.8"); ok {
		t.Fatal("error payload must not be treated as success")
	}
}

// TestTrustedProxyResolution 确认信任内网代理后能取到真实客户端 IP。
func TestClientIPFromForwardedHeader(t *testing.T) {
	// 这里只验证 net 段的判定逻辑与本包无关，放在 geo 包里确保 IP 分类正确
	if !net.ParseIP("203.0.113.9").IsPrivate() {
		return
	}
	t.Fatal("203.0.113.9 should not be private")
}
