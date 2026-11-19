// Package chat 负责会话、消息与流式回答的业务编排。
// 会话与消息全部按 user_id 隔离；本包不直接访问上游，只通过 provider 模块能力调用。
package chat

import (
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
	UserID    uint           `gorm:"index;not null"`                 // 所属用户
	Title     string         `gorm:"size:60;not null;default:'新对话'"` // 会话标题
	ModelID   uint           `gorm:"not null;default:0"`             // 最近使用的模型
	CreatedAt time.Time      // 创建时间
	UpdatedAt time.Time      `gorm:"index"` // 最近更新时间
	DeletedAt gorm.DeletedAt `gorm:"index"` // 软删除时间
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
	AttachmentsJSON string    `gorm:"type:text;not null;default:''"`       // 图片附件 JSON，空表示无附件
	CreatedAt       time.Time `gorm:"index"`                               // 创建时间
	UpdatedAt       time.Time // 更新时间
}

// Summary 是会话列表项。
type Summary struct {
	ID           uint      `json:"id"`            // 会话 ID
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
}

// DailyUsage 是按天聚合的用量。
type DailyUsage struct {
	Date     string `json:"date"`     // 日期，YYYY-MM-DD
	Tokens   int64  `json:"tokens"`   // tokens 用量
	Messages int64  `json:"messages"` // 消息条数
}

// Migrate 创建会话与消息表。
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Conversation{}, &Message{})
}
