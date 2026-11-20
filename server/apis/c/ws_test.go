package c

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"deeptalk/apis/middleware"
	"deeptalk/internal/audit"
	"deeptalk/internal/chat"
	"deeptalk/internal/setting"
	"deeptalk/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// fakeStream 用一个可控的 Stream 实现替换真实对话服务。
type fakeStream struct {
	called  bool
	content string
	stopReq <-chan struct{}
}

func (f *fakeStream) ResolveRef(_ context.Context, _ uint, ref string) (uint, error) {
	if strings.TrimSpace(ref) == "" {
		return 0, chat.ErrNotFound
	}
	return 42, nil
}

func (f *fakeStream) Stream(ctx context.Context, _ uint, _ uint, in chat.StreamInput, emit chat.Emitter) error {
	f.called = true
	f.content = in.Content
	if err := emit("meta", map[string]any{"assistant_message_id": 1, "model_name": "M"}); err != nil {
		return err
	}
	if err := emit("delta", map[string]any{"delta": "你好"}); err != nil {
		return err
	}
	if f.stopReq != nil {
		select {
		case <-f.stopReq:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return emit("usage", map[string]any{"output_tokens": 2})
}

// newTestUsers 准备一个带账号的 user.Service，并返回一个有效 token。
func newTestUsers(t *testing.T) (*user.Service, string, string) {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "ws.db") + "?_pragma=busy_timeout(5000)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	for name, migrate := range map[string]func(*gorm.DB) error{
		"user": user.Migrate, "setting": setting.Migrate, "audit": audit.Migrate,
	} {
		if err := migrate(db); err != nil {
			t.Fatalf("migrate %s: %v", name, err)
		}
	}
	settings := setting.NewService(db)
	users := user.NewService(db, settings, audit.NewService(db), user.Options{SessionTTL: time.Hour, LockThreshold: 3, LockMinutes: 10})
	_, token, err := users.Register(context.Background(), user.RegisterInput{Email: "ws@example.com", Password: "pass-1234"}, user.RequestMeta{})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return users, token, "ws@example.com"
}

// newWSServer 启动一个只挂载 WebSocket 路由的测试服务器。
func newWSServer(t *testing.T, service StreamService, users *user.Service) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	controller := NewWSController(service, users, middleware.Config{CookieName: "dt_session"})
	router.GET("/api/v1/c/ws", controller.Conn)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server
}

// TestWSStreamsWithToken 校验用 ?token= 建连后能收到与 SSE 对齐的事件帧。
func TestWSStreamsWithToken(t *testing.T) {
	users, token, _ := newTestUsers(t)
	stream := &fakeStream{}
	server := newWSServer(t, stream, users)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/c/ws?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(wsRequest{Type: "stream", ConversationID: "abc", Content: "你好"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	want := []string{"meta", "delta", "usage", "done"}
	for _, expected := range want {
		var frame wsFrame
		if err := conn.ReadJSON(&frame); err != nil {
			t.Fatalf("read %s: %v", expected, err)
		}
		if frame.Type != expected {
			t.Fatalf("frame = %q, want %q", frame.Type, expected)
		}
	}
	if !stream.called || stream.content != "你好" {
		t.Fatalf("stream not called correctly: %+v", stream)
	}
}

// TestWSRejectsInvalidToken 校验没有有效凭据时不会升级连接。
func TestWSRejectsInvalidToken(t *testing.T) {
	users, _, _ := newTestUsers(t)
	server := newWSServer(t, &fakeStream{}, users)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/c/ws?token=bad-token"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		_ = conn.Close()
		t.Fatal("expected dial to fail")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %v, want 401", resp)
	}
}

// TestWSPingPong 校验心跳，客户端据此保活。
func TestWSPingPong(t *testing.T) {
	users, token, _ := newTestUsers(t)
	server := newWSServer(t, &fakeStream{}, users)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/c/ws?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(wsRequest{Type: "ping"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	var frame wsFrame
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if err := conn.ReadJSON(&frame); err != nil {
		t.Fatalf("read: %v", err)
	}
	if frame.Type != "pong" {
		t.Fatalf("frame = %q, want pong", frame.Type)
	}
}
