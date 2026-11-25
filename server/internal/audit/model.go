// Package audit 记录管理后台的关键操作，用于追溯“谁在什么时候改了什么”。
// 审计日志只写入、不提供修改入口，且不记录任何敏感值（如 API Key、密码）。
package audit

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Log 是后台操作审计记录。
type Log struct {
	ID         uint      `gorm:"primaryKey"`                   // 主键
	ActorID    uint      `gorm:"index;not null"`               // 操作者用户 ID
	ActorName  string    `gorm:"size:64;not null"`             // 操作者昵称快照
	ActorEmail string    `gorm:"size:190;not null"`            // 操作者邮箱快照
	Action     string    `gorm:"size:64;not null"`             // 动作标识，如 provider.update
	TargetType string    `gorm:"size:32;not null"`             // 目标类型，如 provider/user
	TargetID   uint      `gorm:"not null;default:0"`           // 目标主键，0 表示无具体对象
	Detail     string    `gorm:"size:255;not null;default:''"` // 变更摘要（不含密钥）
	IP         string    `gorm:"size:64;not null;default:''"`  // 操作来源 IP
	CreatedAt  time.Time `gorm:"index"`                        // 记录时间
}

// Migrate 创建审计表。
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Log{})
}

// dalCreateLog 写入一条审计记录。
func dalCreateLog(ctx context.Context, db *gorm.DB, entry *Log) error {
	return db.WithContext(ctx).Create(entry).Error
}

// dalRecentLogs 按时间倒序取最近的审计记录。
func dalRecentLogs(ctx context.Context, db *gorm.DB, limit int) ([]Log, error) {
	var items []Log
	err := db.WithContext(ctx).Order("created_at DESC, id DESC").Limit(limit).Find(&items).Error
	return items, err
}
