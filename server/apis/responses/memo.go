package responses

import (
	"strings"
	"time"

	"deeptalk/internal/memo"
)

// MemoResponse 是灵感录的外发结构；tags 保持逗号分隔字符串，与前端契约一致。
type MemoResponse struct {
	ID             uint      `json:"id"`              // 主键
	Title          string    `json:"title"`           // 标题
	Content        string    `json:"content"`         // 正文
	Tags           string    `json:"tags"`            // 逗号分隔标签
	ConversationID uint      `json:"conversation_id"` // 来源会话
	CreatedAt      time.Time `json:"created_at"`      // 创建时间
	UpdatedAt      time.Time `json:"updated_at"`      // 更新时间
}

// NewMemo 把模型转成响应结构。
func NewMemo(item memo.Memo) MemoResponse {
	return MemoResponse{
		ID:             item.ID,
		Title:          strings.TrimSpace(item.Title),
		Content:        item.Content,
		Tags:           item.Tags,
		ConversationID: item.ConversationID,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

// NewMemos 批量转换。
func NewMemos(items []memo.Memo) []MemoResponse {
	out := make([]MemoResponse, 0, len(items))
	for _, item := range items {
		out = append(out, NewMemo(item))
	}
	return out
}

// MemoListResponse 是灵感录列表响应。
type MemoListResponse struct {
	Items []MemoResponse `json:"items"` // 列表
}

// AdminMemoResponse 是后台灵感录条目，额外带作者邮箱。
type AdminMemoResponse struct {
	MemoResponse
	UserEmail string `json:"user_email"` // 作者邮箱
}

// NewAdminMemos 批量转换后台条目。
func NewAdminMemos(items []memo.AdminItem) []AdminMemoResponse {
	out := make([]AdminMemoResponse, 0, len(items))
	for _, item := range items {
		out = append(out, AdminMemoResponse{
			MemoResponse: NewMemo(item.Memo),
			UserEmail:    item.UserEmail,
		})
	}
	return out
}

// AdminMemoListResponse 是后台分页列表响应。
type AdminMemoListResponse struct {
	Items []AdminMemoResponse `json:"items"` // 列表
	Total int64               `json:"total"` // 总数
	Page  int                 `json:"page"`  // 页码
	Size  int                 `json:"size"`  // 每页条数
}
