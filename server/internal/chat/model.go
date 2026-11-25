// Package chat 负责会话、消息与流式回答的业务编排。
// 会话与消息全部按 user_id 隔离；本包不直接访问上游，只通过 provider 模块能力调用。
package chat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"gorm.io/gorm"
)

// 消息角色。
const (
	RoleUser      = "user"      // 用户
	RoleAssistant = "assistant" // 助手
)

// 消息状态。
const (
	StatusComplete   = "complete"   // 已完成
	StatusStreaming  = "streaming"  // 生成中
	StatusStopped    = "stopped"    // 用户停止，保留已输出内容
	StatusError      = "error"      // 生成失败
	StatusRegenerate = "regenerate" // 兼容字段，不在数据库中出现
)

// 业务错误。
var (
	ErrNotFound      = errors.New("conversation not found")
	ErrMessageNotOwn = errors.New("message not found")
	ErrQuotaExceeded = errors.New("quota exceeded")
	ErrNoModel       = errors.New("no available model")
)

// Conversation 是用户会话。软删除便于审计，列表查询自动过滤。
type Conversation struct {
	ID        uint           `gorm:"primaryKey"`                     // 主键
	PublicID  string         `gorm:"size:64;index" json:"public_id"` // URL 使用的不可枚举公开 ID
	UserID    uint           `gorm:"index;not null"`                 // 所属用户
	Title     string         `gorm:"size:60;not null;default:'新对话'"` // 会话标题
	ModelID   uint           `gorm:"not null;default:0"`             // 最近使用的模型
	CreatedAt time.Time      // 创建时间
	UpdatedAt time.Time      `gorm:"index"` // 最近更新时间
	DeletedAt gorm.DeletedAt `gorm:"index"` // 软删除时间

	// 上下文压缩：Summary 是 prefix 的结构化记忆，SummaryUntilID 之前的消息不再原样发送。
	Summary          string     `gorm:"type:text;not null;default:''" json:"-"`
	SummaryUntilID   uint       `gorm:"not null;default:0" json:"-"`
	SummaryTokens    int        `gorm:"not null;default:0" json:"-"`
	SummaryUpdatedAt *time.Time `json:"-"`
}

// newPublicID 生成不可枚举的会话公开 ID。
func newPublicID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}

// Message 是一条会话消息。
// reasoning 保存深度思考内容；status=stopped 时正文保留用户已看到的部分。
type Message struct {
	ID              uint      `gorm:"primaryKey"`                          // 主键
	ConversationID  uint      `gorm:"index;not null"`                      // 所属会话
	UserID          uint      `gorm:"index;not null"`                      // 所属用户（冗余便于隔离校验）
	Role            string    `gorm:"size:16;not null"`                    // user / assistant
	Content         string    `gorm:"type:text;not null;default:''"`       // 正文 markdown
	Reasoning       string    `gorm:"type:text;not null;default:''"`       // 深度思考内容
	Status          string    `gorm:"size:16;not null;default:'complete'"` // complete / streaming / stopped / error
	ModelName       string    `gorm:"size:64;not null;default:''"`         // 生成该消息的模型展示名
	InputTokens     int       `gorm:"not null;default:0"`                  // 输入 tokens
	OutputTokens    int       `gorm:"not null;default:0"`                  // 输出 tokens
	DurationMS      int       `gorm:"not null;default:0"`                  // 生成耗时（毫秒）
	Feedback        string    `gorm:"size:8;not null;default:''"`          // up / down / 空
	SourcesJSON     string    `gorm:"type:text;not null;default:''"`       // 联网来源 JSON，空表示未联网
	UIJSON          string    `gorm:"type:text;not null;default:''"`       // A2UI 消息 JSON，空表示未生成界面
	AttachmentsJSON string    `gorm:"type:text;not null;default:''"`       // 图片附件 JSON，空表示无附件
	Interjection    bool      `gorm:"not null;default:false"`              // 是否是生成过程中插入的插话
	CreatedAt       time.Time `gorm:"index"`                               // 创建时间
	UpdatedAt       time.Time // 更新时间
}

// Summary 是会话列表项。
type Summary struct {
	ID           uint      `json:"id"`            // 会话 ID
	PublicID     string    `json:"public_id"`     // URL 公开 ID
	Title        string    `json:"title"`         // 标题
	ModelID      uint      `json:"model_id"`      // 最近模型
	UpdatedAt    time.Time `json:"updated_at"`    // 最近更新时间
	MessageCount int       `json:"message_count"` // 消息条数
}

// StreamInput 是发起流式回答的输入。
type StreamInput struct {
	Content       string // 用户输入
	ModelID       uint   // 模型主键；0 表示平台默认
	DeepThinking  bool   // 是否开启深度思考
	WebSearch     bool   // 是否在生成前执行联网检索
	AttachmentIDs []uint // 本次消息的图片附件
	Regenerate    bool   // 是否重新生成
	ClientIP      string // 客户端公网 IP，用于注入当前位置
	Interjection  bool   // 本轮是插话：用户消息允许在上一轮生成结束后紧接着问
	Platform      string // 客户端平台标识：mp-weixin / web / app
}

// DailyUsage 是按天聚合的用量。
type DailyUsage struct {
	Date     string `json:"date"`     // 日期，YYYY-MM-DD
	Tokens   int64  `json:"tokens"`   // tokens 用量
	Messages int64  `json:"messages"` // 消息条数
}

// Migrate 创建会话与消息表，并为旧会话补齐 URL 公开 ID。
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Conversation{}, &Message{}, &Share{}, &Interjection{}); err != nil {
		return err
	}
	var rows []Conversation
	if err := db.Where("public_id = '' OR public_id IS NULL").Find(&rows).Error; err != nil {
		return err
	}
	for _, conv := range rows {
		if err := db.Model(&Conversation{}).Where("id = ?", conv.ID).Update("public_id", newPublicID()).Error; err != nil {
			return err
		}
	}
	return nil
}

// Interjection 是生成过程中用户插入的一句话，等当前轮结束后再处理。
type Interjection struct {
	ID             uint       `gorm:"primaryKey"`
	ConversationID uint       `gorm:"index;not null"`
	UserID         uint       `gorm:"index;not null"`
	Content        string     `gorm:"type:text;not null"`
	CreatedAt      time.Time  `gorm:"index"`
	ConsumedAt     *time.Time `gorm:"index"` // 非空表示已经被下一轮消费
}

// dalListSummaries 返回会话列表（带消息数），支持标题关键词。
func dalListSummaries(ctx context.Context, db *gorm.DB, userID uint, keyword string) ([]Summary, error) {
	query := db.WithContext(ctx).Model(&Conversation{}).
		Select("conversations.id, conversations.public_id, conversations.title, conversations.model_id, conversations.updated_at, "+
			"(SELECT COUNT(*) FROM messages WHERE messages.conversation_id = conversations.id) AS message_count").
		Where("conversations.user_id = ? AND conversations.deleted_at IS NULL", userID)
	if keyword != "" {
		query = query.Where("conversations.title LIKE ?", "%"+keyword+"%")
	}
	var items []Summary
	err := query.Order("conversations.updated_at DESC, conversations.id DESC").Scan(&items).Error
	return items, err
}

// dalCreateConversation 新建会话。
func dalCreateConversation(ctx context.Context, db *gorm.DB, conv *Conversation) error {
	return db.WithContext(ctx).Create(conv).Error
}

// dalSaveConversation 保存会话。
func dalSaveConversation(ctx context.Context, db *gorm.DB, conv *Conversation) error {
	return db.WithContext(ctx).Save(conv).Error
}

// dalDeleteConversationMessages 删除会话下的全部消息。
func dalDeleteConversationMessages(ctx context.Context, db *gorm.DB, conversationID uint) error {
	return db.WithContext(ctx).Where("conversation_id = ?", conversationID).Delete(&Message{}).Error
}

// dalDeleteConversationsMessages 批量删除多个会话下的消息。
func dalDeleteConversationsMessages(ctx context.Context, db *gorm.DB, ids []uint) error {
	return db.WithContext(ctx).Where("conversation_id IN ?", ids).Delete(&Message{}).Error
}

// dalDeleteConversation 删除单个会话。
func dalDeleteConversation(ctx context.Context, db *gorm.DB, id uint) error {
	return db.WithContext(ctx).Delete(&Conversation{}, id).Error
}

// dalDeleteConversations 批量删除会话。
func dalDeleteConversations(ctx context.Context, db *gorm.DB, ids []uint) error {
	return db.WithContext(ctx).Where("id IN ?", ids).Delete(&Conversation{}).Error
}

// dalOwnedConversationIDs 过滤出属于该用户的会话 ID。
func dalOwnedConversationIDs(ctx context.Context, db *gorm.DB, userID uint, ids []uint) ([]uint, error) {
	var owned []uint
	err := db.WithContext(ctx).Model(&Conversation{}).
		Where("user_id = ? AND id IN ?", userID, ids).
		Pluck("id", &owned).Error
	return owned, err
}

// dalListMessages 按 ID 升序返回会话的全部消息。
func dalListMessages(ctx context.Context, db *gorm.DB, conversationID uint) ([]Message, error) {
	var items []Message
	err := db.WithContext(ctx).Where("conversation_id = ?", conversationID).
		Order("id ASC").Find(&items).Error
	return items, err
}

// dalFindOwnedAssistantMessage 读取属于该用户的助手消息。
func dalFindOwnedAssistantMessage(ctx context.Context, db *gorm.DB, userID, messageID uint) (*Message, error) {
	var msg Message
	if err := db.WithContext(ctx).Where("id = ? AND user_id = ? AND role = ?", messageID, userID, RoleAssistant).
		First(&msg).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

// ptr 返回值的地址，用来构造「可选字段」。
func ptr[T any](value T) *T { return &value }

// MessageUpdate 是消息的部分更新；字段为 nil 表示不改该列。
// 用结构体而不是 map，拼错列名或漏字段在编译期就能发现。
type MessageUpdate struct {
	Content      *string // 正文
	Reasoning    *string // 思考链
	Status       *string // 状态：streaming / complete / stopped / error
	ModelName    *string // 模型展示名
	InputTokens  *int    // 输入 token
	OutputTokens *int    // 输出 token
	DurationMS   *int    // 生成耗时（毫秒）
	Feedback     *string // 赞 / 踩
	SourcesJSON  *string // 来源列表 JSON
	UIJSON       *string // A2UI 结构 JSON
}

// fields 只保留被设置的列。
func (u MessageUpdate) fields() map[string]any {
	out := make(map[string]any, 10)
	if u.Content != nil {
		out["content"] = *u.Content
	}
	if u.Reasoning != nil {
		out["reasoning"] = *u.Reasoning
	}
	if u.Status != nil {
		out["status"] = *u.Status
	}
	if u.ModelName != nil {
		out["model_name"] = *u.ModelName
	}
	if u.InputTokens != nil {
		out["input_tokens"] = *u.InputTokens
	}
	if u.OutputTokens != nil {
		out["output_tokens"] = *u.OutputTokens
	}
	if u.DurationMS != nil {
		out["duration_ms"] = *u.DurationMS
	}
	if u.Feedback != nil {
		out["feedback"] = *u.Feedback
	}
	if u.SourcesJSON != nil {
		out["sources_json"] = *u.SourcesJSON
	}
	if u.UIJSON != nil {
		out["ui_json"] = *u.UIJSON
	}
	return out
}

// ConversationUpdate 是会话的部分更新；字段为 nil 表示不改该列。
type ConversationUpdate struct {
	Title            *string    // 标题
	ModelID          *uint      // 当前模型
	UpdatedAt        *time.Time // 更新时间
	Summary          *string    // 上下文压缩摘要
	SummaryUntilID   *uint      // 摘要覆盖到的消息 ID
	SummaryTokens    *int       // 摘要 token 数
	SummaryUpdatedAt *time.Time // 摘要生成时间
}

// fields 只保留被设置的列。
func (u ConversationUpdate) fields() map[string]any {
	out := make(map[string]any, 7)
	if u.Title != nil {
		out["title"] = *u.Title
	}
	if u.ModelID != nil {
		out["model_id"] = *u.ModelID
	}
	if u.UpdatedAt != nil {
		out["updated_at"] = *u.UpdatedAt
	}
	if u.Summary != nil {
		out["summary"] = *u.Summary
	}
	if u.SummaryUntilID != nil {
		out["summary_until_id"] = *u.SummaryUntilID
	}
	if u.SummaryTokens != nil {
		out["summary_tokens"] = *u.SummaryTokens
	}
	if u.SummaryUpdatedAt != nil {
		out["summary_updated_at"] = *u.SummaryUpdatedAt
	}
	return out
}

// dalUpdateMessage 按 MessageUpdate 更新一条消息。
func dalUpdateMessage(ctx context.Context, db *gorm.DB, messageID uint, update MessageUpdate) error {
	fields := update.fields()
	if len(fields) == 0 {
		return nil
	}
	return db.WithContext(ctx).Model(&Message{}).Where("id = ?", messageID).Updates(fields).Error
}

// dalUpdateConversation 按 ConversationUpdate 更新一条会话。
func dalUpdateConversation(ctx context.Context, db *gorm.DB, conversationID uint, update ConversationUpdate) error {
	fields := update.fields()
	if len(fields) == 0 {
		return nil
	}
	return db.WithContext(ctx).Model(&Conversation{}).Where("id = ?", conversationID).Updates(fields).Error
}

// dalDeleteAssistantAfter 删除某条用户消息之后的助手消息（重新生成用）。
func dalDeleteAssistantAfter(ctx context.Context, db *gorm.DB, conversationID, afterMessageID uint) error {
	return db.WithContext(ctx).
		Where("conversation_id = ? AND id > ? AND role = ?", conversationID, afterMessageID, RoleAssistant).
		Delete(&Message{}).Error
}

// dalCreateMessage 新增一条消息。
func dalCreateMessage(ctx context.Context, db *gorm.DB, msg *Message) error {
	return db.WithContext(ctx).Create(msg).Error
}

// dalUpdateSources 更新消息的来源列表 JSON。
func dalUpdateSources(ctx context.Context, db *gorm.DB, messageID uint, raw string) error {
	return dalUpdateMessage(ctx, db, messageID, MessageUpdate{SourcesJSON: &raw})
}

// dalUpdateUI 更新消息的 A2UI 结构 JSON。
func dalUpdateUI(ctx context.Context, db *gorm.DB, messageID uint, raw string) error {
	return dalUpdateMessage(ctx, db, messageID, MessageUpdate{UIJSON: &raw})
}

// dalUpdatePartialContent 更新生成中的正文与思考（用于断线后回看进度）。
func dalUpdatePartialContent(ctx context.Context, db *gorm.DB, messageID uint, content, reasoning string) error {
	return dalUpdateMessage(ctx, db, messageID, MessageUpdate{Content: &content, Reasoning: &reasoning})
}

// dalRecentContextMessages 取会话最近几条有效消息，供检索/推理路由参考。
func dalRecentContextMessages(ctx context.Context, db *gorm.DB, conversationID uint, limit int) ([]Message, error) {
	var items []Message
	err := db.WithContext(ctx).
		Where("conversation_id = ? AND status <> ? AND content <> ''", conversationID, StatusError).
		Order("id DESC").Limit(limit).Find(&items).Error
	return items, err
}

// dalCountUserConversations 统计用户会话数。
func dalCountUserConversations(ctx context.Context, db *gorm.DB, userID uint) (int64, error) {
	var total int64
	err := db.WithContext(ctx).Model(&Conversation{}).Where("user_id = ?", userID).Count(&total).Error
	return total, err
}

// dalCountUserMessages 统计用户消息数。
func dalCountUserMessages(ctx context.Context, db *gorm.DB, userID uint) (int64, error) {
	var total int64
	err := db.WithContext(ctx).Model(&Message{}).Where("user_id = ?", userID).Count(&total).Error
	return total, err
}

// dalCountConversations 统计全部会话数。
func dalCountConversations(ctx context.Context, db *gorm.DB) (int64, error) {
	var total int64
	err := db.WithContext(ctx).Model(&Conversation{}).Count(&total).Error
	return total, err
}

// dalCountMessagesSince 统计自 since 起的消息数。
func dalCountMessagesSince(ctx context.Context, db *gorm.DB, since time.Time) (int64, error) {
	var total int64
	err := db.WithContext(ctx).Model(&Message{}).Where("created_at >= ?", since).Count(&total).Error
	return total, err
}

// dalCountConversationsByUsers 批量统计多个用户的会话数。
func dalCountConversationsByUsers(ctx context.Context, db *gorm.DB, userIDs []uint) (map[uint]int64, error) {
	out := make(map[uint]int64, len(userIDs))
	var rows []struct {
		UserID uint
		Total  int64
	}
	err := db.WithContext(ctx).Model(&Conversation{}).
		Select("user_id, COUNT(*) AS total").
		Where("user_id IN ?", userIDs).
		Group("user_id").Scan(&rows).Error
	if err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.UserID] = row.Total
	}
	return out, nil
}

// dalUsageTimeline 按天聚合 tokens 与消息数。
func dalUsageTimeline(ctx context.Context, db *gorm.DB, since time.Time) ([]DailyUsage, error) {
	var rows []DailyUsage
	err := db.WithContext(ctx).Model(&Message{}).
		Select("date(created_at) AS date, COALESCE(SUM(input_tokens + output_tokens), 0) AS tokens, COUNT(*) AS messages").
		Where("created_at >= ? AND role = ?", since, RoleAssistant).
		Group("date(created_at)").
		Scan(&rows).Error
	return rows, err
}

// dalFindConversation 按主键与归属用户读取会话。
func dalFindConversation(ctx context.Context, db *gorm.DB, userID, conversationID uint) (*Conversation, error) {
	var conv Conversation
	if err := db.WithContext(ctx).Where("id = ? AND user_id = ?", conversationID, userID).
		First(&conv).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// dalFindConversationByRef 按公开 ID 或数字 ID 解析会话。
func dalFindConversationByRef(ctx context.Context, db *gorm.DB, userID uint, publicID string, numericID *uint) (*Conversation, error) {
	var conv Conversation
	query := db.WithContext(ctx).Where("user_id = ? AND deleted_at IS NULL", userID)
	if numericID != nil {
		query = query.Where("id = ?", *numericID)
	} else {
		query = query.Where("public_id = ?", publicID)
	}
	if err := query.First(&conv).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// dalFindConversationByID 按主键读取会话（不校验归属，内部使用）。
func dalFindConversationByID(ctx context.Context, db *gorm.DB, conversationID uint) (*Conversation, error) {
	var conv Conversation
	if err := db.WithContext(ctx).First(&conv, conversationID).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// dalLastUserMessage 取会话中最后一条用户消息。
func dalLastUserMessage(ctx context.Context, db *gorm.DB, conversationID uint) (*Message, error) {
	var msg Message
	if err := db.WithContext(ctx).
		Where("conversation_id = ? AND role = ?", conversationID, RoleUser).
		Order("id DESC").First(&msg).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

// dalMessagesBefore 取某条消息之前、摘要边界之后的有效消息，用于压缩。
func dalMessagesBefore(ctx context.Context, db *gorm.DB, conversationID, afterID, untilID uint) ([]Message, error) {
	var rows []Message
	err := db.WithContext(ctx).
		Where("conversation_id = ? AND id > ? AND id <= ? AND status <> ? AND (content <> '' OR attachments_json <> '')",
			conversationID, afterID, untilID, StatusError).
		Order("id ASC").Find(&rows).Error
	return rows, err
}

// dalUpdateSummaryIfNewer 乐观写入摘要边界，返回是否真的更新。
func dalUpdateSummaryIfNewer(ctx context.Context, db *gorm.DB, conversationID, boundaryID uint, update ConversationUpdate) (bool, error) {
	result := db.WithContext(ctx).Model(&Conversation{}).
		Where("id = ? AND summary_until_id < ?", conversationID, boundaryID).
		Updates(update.fields())
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// dalIdleConversations 返回更新时间早于 cutoff 的会话，供空闲压缩扫描。
func dalIdleConversations(ctx context.Context, db *gorm.DB, cutoff time.Time, limit int) ([]Conversation, error) {
	var items []Conversation
	err := db.WithContext(ctx).
		Where("deleted_at IS NULL AND updated_at < ?", cutoff).
		Order("updated_at ASC").Limit(limit).Find(&items).Error
	return items, err
}

// dalMaxMessageID 返回会话中最大的消息 ID。
func dalMaxMessageID(ctx context.Context, db *gorm.DB, conversationID, afterID uint) (uint, error) {
	var lastID uint
	err := db.WithContext(ctx).Model(&Message{}).
		Where("conversation_id = ? AND id > ?", conversationID, afterID).
		Select("COALESCE(MAX(id), 0)").Scan(&lastID).Error
	return lastID, err
}

// dalConversationIDsByUser 返回某用户的全部会话 ID。
func dalConversationIDsByUser(ctx context.Context, db *gorm.DB, userID uint) ([]uint, error) {
	var ids []uint
	err := db.WithContext(ctx).Model(&Conversation{}).Where("user_id = ?", userID).Pluck("id", &ids).Error
	return ids, err
}

// ShareUpdate 是分享记录的部分更新；字段为 nil 表示不改该列。
type ShareUpdate struct {
	Token        *string    // 新的分享令牌
	RevokedAt    *time.Time // 撤销时间；显式置空用 ClearRevoked
	UserID       *uint      // 归属用户
	ClearRevoked bool       // 是否把 revoked_at 置为 NULL
}

// dalRotateShare 轮换分享令牌并恢复分享。
func dalRotateShare(ctx context.Context, db *gorm.DB, shareID uint, update ShareUpdate) error {
	fields := make(map[string]any, 3)
	if update.Token != nil {
		fields["token"] = *update.Token
	}
	if update.UserID != nil {
		fields["user_id"] = *update.UserID
	}
	if update.ClearRevoked {
		fields["revoked_at"] = nil
	} else if update.RevokedAt != nil {
		fields["revoked_at"] = *update.RevokedAt
	}
	if len(fields) == 0 {
		return nil
	}
	return db.WithContext(ctx).Model(&Share{}).Where("id = ?", shareID).Updates(fields).Error
}
