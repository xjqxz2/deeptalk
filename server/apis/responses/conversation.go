package responses

import (
	"encoding/json"
	"time"

	"deeptalk/internal/attachment"
	"deeptalk/internal/chat"
	"deeptalk/internal/search"
)

type ConversationResponse struct {
	ID           uint      `json:"id"`            // 会话内部 ID
	PublicID     string    `json:"public_id"`     // URL 公开 ID
	Title        string    `json:"title"`         // 标题
	ModelID      uint      `json:"model_id"`      // 最近模型
	UpdatedAt    time.Time `json:"updated_at"`    // 最近更新时间
	MessageCount int       `json:"message_count"` // 消息条数
}

// NewConversation 由列表项构造响应。
func NewConversation(item *chat.Summary) ConversationResponse {
	return ConversationResponse{ID: item.ID, PublicID: item.PublicID, Title: item.Title, ModelID: item.ModelID, UpdatedAt: item.UpdatedAt, MessageCount: item.MessageCount}
}

// NewConversationDetail 由会话模型构造响应。
func NewConversationDetail(conv *chat.Conversation) ConversationResponse {
	if conv == nil {
		return ConversationResponse{}
	}
	return ConversationResponse{ID: conv.ID, PublicID: conv.PublicID, Title: conv.Title, ModelID: conv.ModelID, UpdatedAt: conv.UpdatedAt}
}

// MessageResponse 是消息展示结构。
type MessageResponse struct {
	ID           uint              `json:"id"`             // 消息 ID
	Role         string            `json:"role"`           // user / assistant
	Content      string            `json:"content"`        // 正文
	Reasoning    string            `json:"reasoning"`      // 深度思考
	Status       string            `json:"status"`         // complete / streaming / stopped / error
	ModelName    string            `json:"model_name"`     // 模型名
	InputTokens  int               `json:"input_tokens"`   // 输入 tokens
	OutputTokens int               `json:"output_tokens"`  // 输出 tokens
	DurationMS   int               `json:"duration_ms"`    // 生成耗时
	Feedback     string            `json:"feedback"`       // up / down
	Sources      []search.Result   `json:"sources"`        // 联网检索来源
	A2UI         []json.RawMessage `json:"a2ui,omitempty"` // A2UI v0.9 界面消息
	Attachments  []attachment.Info `json:"attachments"`    // 图片附件
	CreatedAt    time.Time         `json:"created_at"`     // 发送时间
}

// ShareResponse 是分享链接响应；前端用 token 拼出完整 URL。
type ShareResponse struct {
	Token     string    `json:"token"`      // 分享 token
	CreatedAt time.Time `json:"created_at"` // 创建时间
}

// NewShare 构造分享响应。
func NewShare(share *chat.Share) ShareResponse {
	return ShareResponse{Token: share.Token, CreatedAt: share.CreatedAt}
}

// SharedConversationResponse 是公开分享页的只读数据。
type SharedConversationResponse struct {
	Title    string            `json:"title"`     // 会话标题
	Owner    string            `json:"owner"`     // 分享者昵称
	SharedAt time.Time         `json:"shared_at"` // 分享时间
	Messages []MessageResponse `json:"messages"`  // 全部消息
}

// NewSharedConversation 构造分享页响应；只暴露标题、昵称与消息。
func NewSharedConversation(shared chat.SharedConversation) SharedConversationResponse {
	out := make([]MessageResponse, 0, len(shared.Messages))
	for i := range shared.Messages {
		out = append(out, NewMessage(&shared.Messages[i]))
	}
	return SharedConversationResponse{
		Title:    shared.Conversation.Title,
		Owner:    shared.Owner,
		SharedAt: shared.CreatedAt,
		Messages: out,
	}
}

// NewMessage 构造消息响应。
func NewMessage(m *chat.Message) MessageResponse {
	return MessageResponse{
		ID:   m.ID,
		Role: m.Role,
		// 历史消息可能残留上游的 DSML 工具标记，展示前统一清洗
		Content:      chat.SanitizeDSML(m.Content),
		Reasoning:    m.Reasoning,
		Status:       m.Status,
		ModelName:    m.ModelName,
		InputTokens:  m.InputTokens,
		OutputTokens: m.OutputTokens,
		DurationMS:   m.DurationMS,
		Feedback:     m.Feedback,
		Sources:      decodeSources(m.SourcesJSON),
		A2UI:         decodeA2UI(m.UIJSON),
		Attachments:  decodeAttachments(m.AttachmentsJSON),
		CreatedAt:    m.CreatedAt,
	}
}

// decodeSources 解析消息里的联网来源；历史消息或损坏数据返回空数组。
func decodeSources(raw string) []search.Result {
	if raw == "" {
		return []search.Result{}
	}
	var items []search.Result
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []search.Result{}
	}
	return items
}

// decodeA2UI 解析消息里的 A2UI envelope；损坏数据不影响整条消息。
func decodeA2UI(raw string) []json.RawMessage {
	if raw == "" {
		return nil
	}
	var messages []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &messages); err != nil {
		return nil
	}
	return messages
}

// decodeAttachments 解析消息里的附件快照；历史消息或损坏数据返回空数组。
func decodeAttachments(raw string) []attachment.Info {
	if raw == "" {
		return []attachment.Info{}
	}
	var items []attachment.Info
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []attachment.Info{}
	}
	return items
}

// AdminModelResponse 是后台模型路由表行。
