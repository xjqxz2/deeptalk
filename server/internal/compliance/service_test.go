package compliance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"deeptalk/internal/wechat"
)

// fakeUsers 返回固定的 openid。
type fakeUsers struct{ openid string }

func (f fakeUsers) WeChatOpenID(context.Context, uint) string { return f.openid }

// newWeChat 构造一个能取 token 且返回指定判定结果的微信客户端。
func newWeChat(t *testing.T, suggest string) *wechat.Manager {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tk", "expires_in": 7200})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"errcode": 0,
				"result":  map[string]any{"suggest": suggest, "label": 100},
			})
		}
	}))
	t.Cleanup(server.Close)
	return wechat.New(wechat.Config{
		AppID: "a", Secret: "b",
		Endpoint:            server.URL + "/token",
		MsgSecCheckEndpoint: server.URL + "/check",
	}, nil)
}

// TestCheckTextOnlyForMPWeChat 校验只有小程序端触发检测。
func TestCheckTextOnlyForMPWeChat(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 0, "result": map[string]any{"suggest": "pass"}})
	}))
	defer server.Close()
	wx := wechat.New(wechat.Config{AppID: "a", Secret: "b", Endpoint: server.URL}, nil)
	service := New(Config{Enabled: true}, wx, fakeUsers{openid: "o1"}, nil)

	// 没有 token 时 Enabled 为 true，但决定是否真的调用看平台
	if err := service.CheckText(context.Background(), 1, "你好", "web"); err != nil {
		t.Fatalf("web should skip: %v", err)
	}
	if calls != 0 {
		t.Fatalf("calls = %d, want 0 for web", calls)
	}
}

// TestCheckTextBlocksRisky 校验违规内容返回 ErrBlocked。
func TestCheckTextBlocksRisky(t *testing.T) {
	service := New(Config{Enabled: true}, newWeChat(t, "risky"), fakeUsers{openid: "o1"}, nil)
	if err := service.CheckText(context.Background(), 1, "违规内容", PlatformMPWeChat); err != ErrBlocked {
		t.Fatalf("err = %v, want ErrBlocked", err)
	}
}

// TestCheckTextPassesClean 校验正常内容放行。
func TestCheckTextPassesClean(t *testing.T) {
	service := New(Config{Enabled: true}, newWeChat(t, "pass"), fakeUsers{openid: "o1"}, nil)
	if err := service.CheckText(context.Background(), 1, "今天天气不错", PlatformMPWeChat); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}

// TestCheckTextRequiresOpenID 校验未绑定 openid 时的两种策略。
func TestCheckTextRequiresOpenID(t *testing.T) {
	skip := New(Config{Enabled: true, FailOpen: true}, newWeChat(t, "pass"), fakeUsers{}, nil)
	if err := skip.CheckText(context.Background(), 1, "你好", PlatformMPWeChat); err != nil {
		t.Fatalf("should skip without openid: %v", err)
	}
	strict := New(Config{Enabled: true, RequireBind: true}, newWeChat(t, "pass"), fakeUsers{}, nil)
	if err := strict.CheckText(context.Background(), 1, "你好", PlatformMPWeChat); err != ErrNotBound {
		t.Fatalf("err = %v, want ErrNotBound", err)
	}
}

// TestClampRunes 校验超长文本按字符截断。
func TestClampRunes(t *testing.T) {
	long := ""
	for i := 0; i < 3000; i++ {
		long += "字"
	}
	if got := len([]rune(clampRunes(long, maxCheckRunes))); got != maxCheckRunes {
		t.Fatalf("len = %d, want %d", got, maxCheckRunes)
	}
}
