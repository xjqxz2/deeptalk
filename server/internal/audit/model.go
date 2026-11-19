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

// Record 是写入审计日志所需的字段。
type Record struct {
	ActorID    uint   // 操作者 ID
	ActorName  string // 操作者昵称
	ActorEmail string // 操作者邮箱
	Action     string // 动作标识
	TargetType string // 目标类型
	TargetID   uint   // 目标主键
	Detail     string // 变更摘要
	IP         string // 来源 IP
}

// Service 提供审计日志的写入与查询能力。
type Service struct {
	db *gorm.DB // 数据库句柄
}

// NewService 创建审计服务。
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Record 写入一条审计日志；审计失败不影响主流程，由调用方决定是否忽略错误。
func (s *Service) Record(ctx context.Context, in Record) error {
	entry := Log{
		ActorID:    in.ActorID,
		ActorName:  in.ActorName,
		ActorEmail: in.ActorEmail,
		Action:     in.Action,
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
		Detail:     in.Detail,
		IP:         in.IP,
	}
	return s.db.WithContext(ctx).Create(&entry).Error
}

// Recent 返回最近的审计记录，按时间倒序。
// limit 会被限制在 1..50，避免一次拉取过多数据。
func (s *Service) Recent(ctx context.Context, limit int) ([]Log, error) {
	if limit <= 0 {
		limit = 6
	}
	if limit > 50 {
		limit = 50
	}
	var items []Log
	err := s.db.WithContext(ctx).Order("created_at DESC, id DESC").Limit(limit).Find(&items).Error
	return items, err
}

// Migrate 创建审计表。
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Log{})
}
