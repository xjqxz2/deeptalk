package requests

import (
	"github.com/gin-gonic/gin"
)

// MemoCreateRequest 是新建灵感录请求。
type MemoCreateRequest struct {
	Title          string   `json:"title" binding:"max=120"`           // 标题
	Content        string   `json:"content" binding:"required"`        // 正文
	Tags           []string `json:"tags" binding:"max=12,dive,max=24"` // 标签
	ConversationID uint     `json:"conversation_id"`                   // 来源会话
}

// Bind 绑定新建请求。
func (r *MemoCreateRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// MemoUpdateRequest 是修改灵感录请求。
type MemoUpdateRequest struct {
	Title   string   `json:"title" binding:"max=120"`           // 标题
	Content string   `json:"content" binding:"required"`        // 正文
	Tags    []string `json:"tags" binding:"max=12,dive,max=24"` // 标签
}

// Bind 绑定修改请求。
func (r *MemoUpdateRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// MemoIDRequest 是灵感录路径参数。
type MemoIDRequest struct {
	ID uint `uri:"id" binding:"required,min=1"` // 灵感录 ID
}

// Bind 绑定路径参数。
func (r *MemoIDRequest) Bind(c *gin.Context) error {
	return c.ShouldBindUri(r)
}

// MemoListRequest 是 C 端灵感录列表查询。
type MemoListRequest struct {
	Query string `form:"q" binding:"max=120"` // 关键词
	Limit int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

// Bind 绑定查询参数。
func (r *MemoListRequest) Bind(c *gin.Context) error {
	if err := c.ShouldBindQuery(r); err != nil {
		return err
	}
	if r.Limit <= 0 {
		r.Limit = 20
	}
	return nil
}

// AdminMemoListRequest 是后台灵感录分页查询。
type AdminMemoListRequest struct {
	Query string `form:"q" binding:"max=120"`                    // 关键词
	Page  int    `form:"page" binding:"omitempty,min=1"`         // 页码
	Size  int    `form:"size" binding:"omitempty,min=1,max=100"` // 每页条数
}

// Bind 绑定查询参数并补齐默认值。
func (r *AdminMemoListRequest) Bind(c *gin.Context) error {
	if err := c.ShouldBindQuery(r); err != nil {
		return err
	}
	if r.Page <= 0 {
		r.Page = 1
	}
	if r.Size <= 0 {
		r.Size = 20
	}
	return nil
}
