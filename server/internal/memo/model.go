// Package memo 提供「小橘灵感录」：对话中随时记下灵感，需要时再取出来。
package memo

import (
	"time"

	"gorm.io/gorm"
)

// Memo 是一条灵感录，按用户隔离。
type Memo struct {
	ID             uint      `gorm:"primaryKey" json:"id"`                      // 主键
	UserID         uint      `gorm:"index:idx_memo_user;not null" json:"-"`     // 归属用户，不外发
	Title          string    `gorm:"size:120;not null;default:''" json:"title"` // 标题（模型总结出的简短主题）
	Content        string    `gorm:"type:text;not null" json:"content"`         // 正文
	Tags           string    `gorm:"size:255;not null;default:''" json:"tags"`  // 逗号分隔的标签
	ConversationID uint      `gorm:"not null;default:0" json:"conversation_id"` // 来源会话，0 表示手动创建
	CreatedAt      time.Time `gorm:"index:idx_memo_user" json:"created_at"`     // 创建时间
	UpdatedAt      time.Time `json:"updated_at"`                                // 更新时间
}

// Migrate 建表。
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Memo{})
}
