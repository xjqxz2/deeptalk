package requests

import (
	"github.com/gin-gonic/gin"
)

type ConversationCreateRequest struct {
	Title string `json:"title" binding:"max=60"` // 可选标题
}

// Bind 绑定新建会话请求；允许空 body。
func (r *ConversationCreateRequest) Bind(c *gin.Context) error {
	if c.Request.ContentLength == 0 {
		return nil
	}
	return c.ShouldBindJSON(r)
}

// ConversationRenameRequest 是重命名会话请求。
type ConversationRenameRequest struct {
	Title string `json:"title" binding:"required,min=1,max=60"` // 新标题
}

// Bind 绑定重命名请求。
func (r *ConversationRenameRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// StreamRequest 是发起流式回答的请求。
type StreamRequest struct {
	Content       string `json:"content" binding:"max=8000"` // 用户输入
	ModelID       uint   `json:"model_id"`                   // 模型主键
	DeepThinking  bool   `json:"deep_thinking"`              // 深度思考
	WebSearch     bool   `json:"web_search"`                 // 联网搜索
	AttachmentIDs []uint `json:"attachment_ids"`             // 图片附件 ID
	Regenerate    bool   `json:"regenerate"`                 // 重新生成
}

// Bind 绑定流式请求；重新生成时 content 可为空，业务层再做校验。
func (r *StreamRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// FeedbackRequest 是消息反馈请求。
type FeedbackRequest struct {
	Rating string `json:"rating" binding:"max=8"` // up / down / 空
}

// Bind 绑定反馈请求。
func (r *FeedbackRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// ProviderSaveRequest 是新建 / 编辑厂商请求。
