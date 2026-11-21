package wechat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeWeChat 模拟稳定版接口：记录调用次数与 force_refresh 取值。
type fakeWeChat struct {
	mu       sync.Mutex
	calls    int
	lastBody stableTokenRequest
	token    string
	expires  int
	failWith int
}

func (f *fakeWeChat) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body stableTokenRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		f.calls++
		f.lastBody = body
		failWith := f.failWith
		token := f.token
		expires := f.expires
		f.mu.Unlock()

		if failWith != 0 {
			_ = json.NewEncoder(w).Encode(stableTokenResponse{ErrCode: failWith, ErrMsg: "boom"})
			return
		}
		_ = json.NewEncoder(w).Encode(stableTokenResponse{AccessToken: token, ExpiresIn: expires})
	})
}

func (f *fakeWeChat) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func newTestManager(t *testing.T, fake *fakeWeChat) *Manager {
	t.Helper()
	server := httptest.NewServer(fake.handler())
	t.Cleanup(server.Close)
	return New(Config{
		AppID:    "wx-test-appid",
		Secret:   "test-secret",
		Endpoint: server.URL,
		Buffer:   5 * time.Minute,
	}, nil)
}

// TestTokenCached 校验同一 token 在有效期内不会重复请求。
func TestTokenCached(t *testing.T) {
	fake := &fakeWeChat{token: "tk-1", expires: 7200}
	manager := newTestManager(t, fake)

	first, err := manager.Token(context.Background())
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	second, err := manager.Token(context.Background())
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if first != "tk-1" || second != first {
		t.Fatalf("token = %q / %q, want tk-1", first, second)
	}
	if calls := fake.callCount(); calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

// TestTokenRefreshesNearExpiry 校验进入刷新窗口后会重新获取。
func TestTokenRefreshesNearExpiry(t *testing.T) {
	fake := &fakeWeChat{token: "tk-1", expires: 7200}
	manager := newTestManager(t, fake)
	now := time.Now()
	manager.now = func() time.Time { return now }

	if _, err := manager.Token(context.Background()); err != nil {
		t.Fatalf("Token: %v", err)
	}

	// 拨到刷新窗口内（有效期 7200，提前 300 秒刷新）
	fake.mu.Lock()
	fake.token = "tk-2"
	fake.mu.Unlock()
	now = now.Add(7000 * time.Second)

	token, err := manager.Token(context.Background())
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if token != "tk-2" {
		t.Fatalf("token = %q, want tk-2", token)
	}
	if calls := fake.callCount(); calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

// TestInvalidateForcesRefresh 校验失效后下次取用会带 force_refresh。
func TestInvalidateForcesRefresh(t *testing.T) {
	fake := &fakeWeChat{token: "tk-1", expires: 7200}
	manager := newTestManager(t, fake)

	if _, err := manager.Token(context.Background()); err != nil {
		t.Fatalf("Token: %v", err)
	}
	manager.Invalidate()

	fake.mu.Lock()
	fake.token = "tk-2"
	fake.mu.Unlock()

	if _, err := manager.Token(context.Background()); err != nil {
		t.Fatalf("Token: %v", err)
	}
	fake.mu.Lock()
	forced := fake.lastBody.ForceRefresh
	fake.mu.Unlock()
	if !forced {
		t.Fatal("expected force_refresh=true after Invalidate")
	}
	if calls := fake.callCount(); calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

// TestTokenErrorPropagates 校验第一次获取失败会返回错误。
func TestTokenErrorPropagates(t *testing.T) {
	fake := &fakeWeChat{failWith: 40001}
	manager := newTestManager(t, fake)
	if _, err := manager.Token(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

// TestDisabledWithoutCredentials 校验缺少凭据时返回 ErrDisabled。
func TestDisabledWithoutCredentials(t *testing.T) {
	manager := New(Config{}, nil)
	if manager.Enables() {
		t.Fatal("should be disabled")
	}
	if _, err := manager.Token(context.Background()); err != ErrDisabled {
		t.Fatalf("err = %v, want ErrDisabled", err)
	}
}

// TestConcurrentCallersShareOneRefresh 校验并发取 token 只打一次接口。
func TestConcurrentCallersShareOneRefresh(t *testing.T) {
	fake := &fakeWeChat{token: "tk-1", expires: 7200}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		fake.handler().ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	manager := New(Config{AppID: "a", Secret: "b", Endpoint: server.URL}, nil)

	var wg sync.WaitGroup
	var success int32
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := manager.Token(context.Background()); err == nil {
				atomic.AddInt32(&success, 1)
			}
		}()
	}
	wg.Wait()
	if success != 8 {
		t.Fatalf("success = %d, want 8", success)
	}
	if calls := fake.callCount(); calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}
