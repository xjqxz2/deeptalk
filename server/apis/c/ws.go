package c

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"deeptalk/apis/middleware"
	"deeptalk/apis/responses"
	"deeptalk/internal/chat"
	"deeptalk/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// StreamService 是 WebSocket 控制器依赖的会话能力，抽成接口便于测试替换。
type StreamService interface {
	ResolveRef(ctx context.Context, userID uint, ref string) (uint, error)
	Stream(ctx context.Context, userID, conversationID uint, in chat.StreamInput, emit chat.Emitter) error
	QueueInterjection(ctx context.Context, userID, conversationID uint, content string) (*chat.Interjection, error)
	PopInterjection(ctx context.Context, userID, conversationID uint) (*chat.Interjection, bool, error)
}

// WSController 处理小程序 / APP 的 WebSocket 长连接。
// 小程序端无法消费 SSE，这里把同样的事件用 JSON 帧下发。
type WSController struct {
	chats StreamService     // 会话与流式能力
	users *user.Service     // 会话校验
	cfg   middleware.Config // Cookie 与会话配置
}

// NewWSController 创建 WebSocket 控制器。
func NewWSController(chats StreamService, users *user.Service, cfg middleware.Config) *WSController {
	return &WSController{chats: chats, users: users, cfg: cfg}
}

// upgrader 允许跨域升级：小程序与 H5 调试页来源不同，鉴权靠 token 而不是 Origin。
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(*http.Request) bool { return true },
}

// wsRequest 是客户端发来的一个指令帧。
type wsRequest struct {
	Type           string `json:"type"`            // stream / stop / ping
	ConversationID string `json:"conversation_id"` // 会话 public_id 或数字 ID
	Content        string `json:"content"`         // 用户输入
	ModelID        uint   `json:"model_id"`        // 指定模型
	AttachmentIDs  []uint `json:"attachment_ids"`  // 图片附件
	Regenerate     bool   `json:"regenerate"`      // 重新生成
	Platform       string `json:"platform"`        // 客户端平台（可选，握手头优先）
}

// wsFrame 是服务端下发的统一帧结构：
// {"type":"delta","payload":{"delta":"..."}}，type 与 SSE 事件名一一对应。
type wsFrame struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

// Conn 处理一次 WebSocket 连接的生命周期。
func (h *WSController) Conn(c *gin.Context) {
	current, err := h.authenticate(c)
	if err != nil {
		responses.Fail(c, responses.CodeUnauthenticated, "登录已失效，请重新登录")
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 写操作可能来自流式协程，必须串行化
	var writeMu sync.Mutex
	write := func(frame wsFrame) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(frame)
	}

	// 关闭时让正在生成的回答立即停止
	connCtx, cancelConn := context.WithCancel(context.Background())
	defer cancelConn()

	var streamMu sync.Mutex
	var cancelStream context.CancelFunc
	running := false
	stopStream := func() {
		streamMu.Lock()
		if cancelStream != nil {
			cancelStream()
			cancelStream = nil
		}
		running = false
		streamMu.Unlock()
	}
	defer stopStream()

	conn.SetReadLimit(1 << 20)
	_ = conn.SetReadDeadline(time.Now().Add(wsIdleTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsIdleTimeout))
	})

	for {
		var req wsRequest
		if err := conn.ReadJSON(&req); err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(wsIdleTimeout))

		switch strings.ToLower(strings.TrimSpace(req.Type)) {
		case "", "ping":
			_ = write(wsFrame{Type: "pong"})
		case "interject":
			// 生成期间用户插话：先入队，当前轮结束后由下面的循环取走
			// 插话必须带会话：小程序端在生成中会带 conversation_id
			convRef := req.ConversationID
			var item *chat.Interjection
			var err error
			if strings.TrimSpace(convRef) == "" {
				err = chat.ErrNotGenerating
			} else {
				convID, resolveErr := h.chats.ResolveRef(c.Request.Context(), current.ID, convRef)
				if resolveErr != nil {
					err = resolveErr
				} else {
					item, err = h.chats.QueueInterjection(c.Request.Context(), current.ID, convID, req.Content)
				}
			}
			response := wsFrame{Type: "interjection_queued"}
			if err != nil {
				response = wsFrame{Type: "error", Payload: gin.H{"message": err.Error(), "code": 3005}}
			} else {
				response.Payload = gin.H{"id": item.ID, "content": item.Content}
			}
			_ = write(response)
		case "stop":
			stopStream()
			_ = write(wsFrame{Type: "stopped"})
		case "stream":
			streamMu.Lock()
			busy := running
			streamMu.Unlock()
			if busy {
				_ = write(wsFrame{Type: "error", Payload: gin.H{"message": "当前对话正在生成，请先停止", "code": 3002}})
				continue
			}
			conversationID, resolveErr := h.chats.ResolveRef(connCtx, current.ID, req.ConversationID)
			if resolveErr != nil {
				_ = write(wsFrame{Type: "error", Payload: gin.H{"message": "会话不存在或无权访问", "code": 3003}})
				continue
			}
			streamCtx, cancel := context.WithCancel(connCtx)
			streamMu.Lock()
			cancelStream = cancel
			running = true
			streamMu.Unlock()

			go func(convID uint, in chat.StreamInput, ctx context.Context) {
				emit := func(event string, payload any) error {
					return write(wsFrame{Type: event, Payload: payload})
				}
				for {
					err := h.chats.Stream(ctx, current.ID, convID, in, emit)
					if err != nil {
						_ = write(wsFrame{Type: "error", Payload: gin.H{"message": err.Error(), "code": 3001}})
						break
					}
					// 本轮结束后检查是否有插话：有就接着在同一连接里生成
					next, ok, popErr := h.chats.PopInterjection(ctx, current.ID, convID)
					if popErr != nil || !ok {
						_ = write(wsFrame{Type: "done", Payload: gin.H{"status": "complete"}})
						break
					}
					in = chat.StreamInput{
						Content:      next.Content,
						ModelID:      in.ModelID,
						ClientIP:     in.ClientIP,
						Platform:     in.Platform,
						Interjection: true,
					}
				}
				cancel()
				streamMu.Lock()
				running = false
				streamMu.Unlock()
			}(conversationID, chat.StreamInput{
				Content:       req.Content,
				ModelID:       req.ModelID,
				AttachmentIDs: req.AttachmentIDs,
				Regenerate:    req.Regenerate,
				ClientIP:      middleware.ClientIP(c),
				Platform:      wsPlatform(c, req.Platform),
			}, streamCtx)
		default:
			_ = write(wsFrame{Type: "error", Payload: gin.H{"message": "未知的指令类型", "code": 3004}})
		}
	}
}

// wsPlatform 识别小程序端：优先握手请求头，其次帧里的 platform 字段。
func wsPlatform(c *gin.Context, framePlatform string) string {
	value := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Client")))
	if value == "" {
		value = strings.ToLower(strings.TrimSpace(framePlatform))
	}
	switch value {
	case "mp-weixin", "weixin", "miniprogram", "mp":
		return "mp-weixin"
	case "app":
		return "app"
	default:
		return "web"
	}
}

// wsIdleTimeout 是连接空闲上限；客户端每 30 秒发一次 ping 即可续期。
const wsIdleTimeout = 120 * time.Second

// authenticate 支持三种凭据来源：Cookie、Authorization Bearer、X-Session-Token，
// 以及小程序 WebSocket 无法设置请求头时使用的 ?token=。
func (h *WSController) authenticate(c *gin.Context) (*user.User, error) {
	candidates := []string{}
	if cookie, err := c.Cookie(h.cfg.CookieName); err == nil && cookie != "" {
		candidates = append(candidates, cookie)
	}
	if header := strings.TrimSpace(c.GetHeader("Authorization")); strings.HasPrefix(strings.ToLower(header), "bearer ") {
		candidates = append(candidates, strings.TrimSpace(header[7:]))
	}
	if header := strings.TrimSpace(c.GetHeader("X-Session-Token")); header != "" {
		candidates = append(candidates, header)
	}
	if query := strings.TrimSpace(c.Query("token")); query != "" {
		candidates = append(candidates, query)
	}
	for _, token := range candidates {
		current, err := h.users.Authenticate(c.Request.Context(), token)
		if err == nil && current != nil {
			return current, nil
		}
	}
	return nil, errors.New("invalid session")
}
