package c

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"deeptalk/apis/middleware"
	"deeptalk/apis/requests"
	"deeptalk/apis/responses"
	"deeptalk/internal/chat"

	"github.com/gin-gonic/gin"
)

// ChatController 处理会话、消息与 SSE 流式回答。
type ChatController struct {
	chats *chat.Service // 会话能力
}

// NewChatController 创建 C 端会话控制器。
func NewChatController(chats *chat.Service) *ChatController {
	return &ChatController{chats: chats}
}

// List 返回当前用户的会话列表，支持标题搜索。
func (h *ChatController) List(c *gin.Context) {
	current := middleware.CurrentUser(c)
	items, err := h.chats.List(c.Request.Context(), current.ID, c.Query("keyword"))
	if err != nil {
		responses.FromError(c, err)
		return
	}
	out := make([]responses.ConversationResponse, 0, len(items))
	for _, item := range items {
		out = append(out, responses.NewConversation(item))
	}
	responses.OK(c, gin.H{"items": out})
}

// Create 新建会话。
func (h *ChatController) Create(c *gin.Context) {
	var req requests.ConversationCreateRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "标题不合法")
		return
	}
	current := middleware.CurrentUser(c)
	conv, err := h.chats.Create(c.Request.Context(), current.ID)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, gin.H{"conversation": responses.NewConversationDetail(conv)})
}

// Get 返回会话详情与全部消息。
func (h *ChatController) Get(c *gin.Context) {
	current := middleware.CurrentUser(c)
	id, err := h.chats.ResolveRef(c.Request.Context(), current.ID, c.Param("id"))
	if err != nil {
		responses.FromError(c, err)
		return
	}
	conv, messages, err := h.chats.Get(c.Request.Context(), current.ID, id)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	out := make([]responses.MessageResponse, 0, len(messages))
	for _, message := range messages {
		out = append(out, responses.NewMessage(message))
	}
	responses.OK(c, gin.H{
		"conversation": responses.NewConversationDetail(conv),
		"messages":     out,
	})
}

// Rename 重命名会话。
func (h *ChatController) Rename(c *gin.Context) {
	current := middleware.CurrentUser(c)
	id, err := h.chats.ResolveRef(c.Request.Context(), current.ID, c.Param("id"))
	if err != nil {
		responses.FromError(c, err)
		return
	}
	var req requests.ConversationRenameRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "标题需为 1–60 个字符")
		return
	}
	conv, err := h.chats.Rename(c.Request.Context(), current.ID, id, req.Title)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, gin.H{"conversation": responses.NewConversationDetail(conv)})
}

// Delete 删除会话。
func (h *ChatController) Delete(c *gin.Context) {
	current := middleware.CurrentUser(c)
	id, err := h.chats.ResolveRef(c.Request.Context(), current.ID, c.Param("id"))
	if err != nil {
		responses.FromError(c, err)
		return
	}
	if err := h.chats.Delete(c.Request.Context(), current.ID, id); err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, nil)
}

// clientPlatform 读取客户端声明的平台标识，用于小程序端的内容安全校验。
func clientPlatform(c *gin.Context) string {
	value := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Client")))
	switch value {
	case "mp-weixin", "weixin", "miniprogram", "mp":
		return "mp-weixin"
	case "app":
		return "app"
	default:
		return "web"
	}
}

// Stream 发起 SSE 流式回答。
// 说明：只有在业务层确认可生成后才写出响应头，因此额度不足、模型不可用等错误仍返回 JSON。
func (h *ChatController) Stream(c *gin.Context) {
	current := middleware.CurrentUser(c)
	id, err := h.chats.ResolveRef(c.Request.Context(), current.ID, c.Param("id"))
	if err != nil {
		responses.FromError(c, err)
		return
	}
	var req requests.StreamRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请求内容不合法")
		return
	}
	var (
		mu      sync.Mutex // 保护 started 与 ResponseWriter，心跳与业务事件可能并发
		started bool
	)
	startedNow := func() bool {
		mu.Lock()
		defer mu.Unlock()
		return started
	}
	emit := func(event string, payload any) error {
		mu.Lock()
		defer mu.Unlock()
		if !started {
			header := c.Writer.Header()
			header.Set("Content-Type", "text/event-stream; charset=utf-8")
			header.Set("Cache-Control", "no-cache, no-transform")
			header.Set("Connection", "keep-alive")
			// 关闭 Nginx 等反向代理的缓冲，保证逐字输出
			header.Set("X-Accel-Buffering", "no")
			c.Writer.WriteHeader(http.StatusOK)
			started = true
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}

	// 长连接心跳：15 秒无事件时发送 ping，避免网关主动断开
	stopHeartbeat := make(chan struct{})
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopHeartbeat:
				return
			case <-ticker.C:
				if !startedNow() {
					continue
				}
				_ = emit("ping", map[string]any{"ts": time.Now().Unix()})
			}
		}
	}()

	clientIP := strings.TrimSpace(c.GetHeader("X-Real-IP"))
	if clientIP == "" {
		clientIP = c.ClientIP()
	}
	input := chat.StreamInput{
		Content:       req.Content,
		ModelID:       req.ModelID,
		DeepThinking:  req.DeepThinking,
		WebSearch:     req.WebSearch,
		AttachmentIDs: req.AttachmentIDs,
		Regenerate:    req.Regenerate,
		ClientIP:      clientIP,
		Platform:      clientPlatform(c),
	}
	for {
		err = h.chats.Stream(c.Request.Context(), current.ID, id, input, emit)
		if err != nil {
			// 本轮异常（用户停止 / 上游报错）时，队列里的插话不会再被处理：
			// 取出来并告知前端，避免界面永久停在「排队中」。
			if next, ok, _ := h.chats.PopInterjection(c.Request.Context(), current.ID, id); ok && next != nil {
				_ = emit("notice", gin.H{
					"message":      "插话未处理，请重新发送",
					"interjection": next.Content,
				})
			}
			break
		}
		// 生成期间用户可能插了话：当前轮结束后接着处理，不打断连接
		next, ok, popErr := h.chats.PopInterjection(c.Request.Context(), current.ID, id)
		if popErr != nil || !ok {
			break
		}
		input = chat.StreamInput{
			Content:      next.Content,
			ModelID:      req.ModelID,
			DeepThinking: req.DeepThinking,
			WebSearch:    req.WebSearch,
			ClientIP:     clientIP,
			Platform:     clientPlatform(c),
			Interjection: true,
		}
	}

	close(stopHeartbeat)
	<-heartbeatDone
	if err != nil && !startedNow() {
		// 客户端已断开时不再写响应，避免无效日志
		if c.Request.Context().Err() != nil {
			return
		}
		responses.FromError(c, err)
	}
}

// Interject 在生成过程中插入一句话：先入队，等当前轮结束后由同一条流继续处理。
func (h *ChatController) Interject(c *gin.Context) {
	current := middleware.CurrentUser(c)
	id, err := h.chats.ResolveRef(c.Request.Context(), current.ID, c.Param("id"))
	if err != nil {
		responses.FromError(c, err)
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请求内容不合法")
		return
	}
	item, err := h.chats.QueueInterjection(c.Request.Context(), current.ID, id, req.Content)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, gin.H{"queued": true, "id": item.ID, "content": item.Content})
}

// Feedback 记录消息赞 / 踩。
func (h *ChatController) Feedback(c *gin.Context) {
	current := middleware.CurrentUser(c)
	id, ok := parseID(c.Param("id"))
	if !ok {
		responses.Fail(c, responses.CodeInvalidArgument, "消息 ID 不合法")
		return
	}
	var req requests.FeedbackRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "反馈内容不合法")
		return
	}
	if err := h.chats.Feedback(c.Request.Context(), current.ID, id, req.Rating); err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, nil)
}

// parseID 解析路径中的正整数主键。
func parseID(raw string) (uint, bool) {
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || n == 0 {
		return 0, false
	}
	return uint(n), true
}
