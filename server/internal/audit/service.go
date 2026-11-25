package audit

import (
	"context"

	"gorm.io/gorm"
)

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
	return dalCreateLog(ctx, s.db, &entry)
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
	return dalRecentLogs(ctx, s.db, limit)
}
