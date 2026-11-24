package c

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
	hub   *streamHub    // SSE 断线续传缓冲
}

// NewChatController 创建 C 端会话控制器。
func NewChatController(chats *chat.Service) *ChatController {
	return &ChatController{chats: chats, hub: newStreamHub()}
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

// Stream 返回 SSE 流：生成过程跑在独立的 goroutine 里，写入 streamHub 缓冲，
// 连接断开不会中断生成；客户端带 resume + last_seq 重连即可补齐漏掉的事件。
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
	key := h.hub.key(current.ID, id)

	// 断线重连：把已有缓冲接上，从 last_seq 之后补发
	if req.Resume {
		if run := h.hub.get(key); run != nil {
			h.attach(c, run, req.LastSeq)
			return
		}
		responses.Fail(c, responses.CodeInvalidArgument, "上一次回答已经结束，请重新发送")
		return
	}

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

	run := h.hub.start(key)
	ctx, cancel := context.WithCancel(context.Background())
	run.setCancel(cancel)
	go h.generate(ctx, run, key, current.ID, id, input)
	h.attach(c, run, 0)
}

// attach 把一条 SSE 连接挂到正在进行的回答上：先补发积压事件，再实时推送。
func (h *ChatController) attach(c *gin.Context, run *streamRun, afterSeq int) {
	header := c.Writer.Header()
	header.Set("Content-Type", "text/event-stream; charset=utf-8")
	header.Set("Cache-Control", "no-cache, no-transform")
	header.Set("Connection", "keep-alive")
	// 关闭 Nginx 等反向代理的缓冲，保证逐字输出
	header.Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	backlog, ch, finished := run.subscribe(afterSeq)
	if ch != nil {
		defer run.unsubscribe(ch)
	}
	write := func(event streamEvent) error {
		if _, err := fmt.Fprintf(c.Writer, "id: %d\nevent: %s\ndata: %s\n\n", event.seq, event.name, event.data); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}
	for _, event := range backlog {
		if err := write(event); err != nil {
			return
		}
	}
	if finished {
		return
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	clientGone := c.Request.Context().Done()
	for {
		select {
		case <-clientGone:
			// 浏览器断开：生成继续，等它带 last_seq 回来
			return
		case <-ticker.C:
			if _, err := fmt.Fprintf(c.Writer, "event: ping\ndata: {\"ts\":%d}\n\n", time.Now().Unix()); err != nil {
				return
			}
			c.Writer.Flush()
		case event, ok := <-ch:
			if !ok {
				return
			}
			if err := write(event); err != nil {
				return
			}
		}
	}
}

// generate 在独立 context 里跑完一轮（可能包含多次插话），事件写进缓冲。
func (h *ChatController) generate(ctx context.Context, run *streamRun, key string, userID, conversationID uint, input chat.StreamInput) {
	publish := func(event string, payload any) error {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		run.publish(event, data)
		return nil
	}
	defer func() {
		run.finish()
		h.hub.removeAfter(key, run, streamRunTTL)
	}()
	// 没人监听超过 2 分钟就取消，避免用户关页面后还在烧 token
	go func() {
		var idleFrom time.Time
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if run.hasSubscribers() {
					idleFrom = time.Time{}
					continue
				}
				if idleFrom.IsZero() {
					idleFrom = time.Now()
					continue
				}
				if time.Since(idleFrom) > 2*time.Minute {
					run.cancelRun()
					return
				}
			}
		}
	}()

	for {
		err := h.chats.Stream(ctx, userID, conversationID, input, publish)
		if err != nil {
			// 本轮异常（用户停止 / 上游报错）时，队列里的插话不会再被处理：
			// 取出来并告知前端，避免界面永久停在「排队中」。
			if next, ok, _ := h.chats.PopInterjection(ctx, userID, conversationID); ok && next != nil {
				_ = publish("notice", gin.H{
					"message":      "插话未处理，请重新发送",
					"interjection": next.Content,
				})
			}
			if ctx.Err() != nil {
				_ = publish("error", gin.H{"message": "生成已停止"})
			}
			return
		}
		// 生成期间用户可能插了话：当前轮结束后接着处理，不打断连接
		next, ok, popErr := h.chats.PopInterjection(ctx, userID, conversationID)
		if popErr != nil || !ok {
			return
		}
		input = chat.StreamInput{
			Content:      next.Content,
			ModelID:      input.ModelID,
			DeepThinking: input.DeepThinking,
			WebSearch:    input.WebSearch,
			ClientIP:     input.ClientIP,
			Platform:     input.Platform,
			Interjection: true,
		}
	}
}


// Stop 取消当前会话正在进行的生成。
// 生成已经不绑定 HTTP 连接，所以「停止生成」必须显式通知服务端，否则会继续烧 token。
func (h *ChatController) Stop(c *gin.Context) {
	current := middleware.CurrentUser(c)
	id, err := h.chats.ResolveRef(c.Request.Context(), current.ID, c.Param("id"))
	if err != nil {
		responses.FromError(c, err)
		return
	}
	run := h.hub.get(h.hub.key(current.ID, id))
	if run != nil {
		run.cancelRun()
	}
	responses.OK(c, gin.H{"stopped": run != nil})
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
