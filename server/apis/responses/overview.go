package responses

import (
	"time"

	"deeptalk/internal/audit"
	"deeptalk/internal/provider"
)

type AuditResponse struct {
	ID        uint      `json:"id"`         // 记录 ID
	ActorName string    `json:"actor_name"` // 操作者
	Action    string    `json:"action"`     // 动作
	Detail    string    `json:"detail"`     // 摘要
	CreatedAt time.Time `json:"created_at"` // 时间
}

// NewAudit 构造审计条目响应。
func NewAudit(l *audit.Log) AuditResponse {
	return AuditResponse{ID: l.ID, ActorName: l.ActorName, Action: l.Action, Detail: l.Detail, CreatedAt: l.CreatedAt}
}

// OverviewResponse 是后台概览数据。
type OverviewResponse struct {
	TotalUsers        int64           `json:"total_users"`        // 总用户
	NewThisMonth      int64           `json:"new_this_month"`     // 本月新增
	Active7D          int64           `json:"active_7d"`          // 7 日活跃
	ActiveRate        int             `json:"active_rate"`        // 周活跃率
	MonthlyTokens     int64           `json:"monthly_tokens"`     // 本月 tokens
	MonthlyGrowth     int             `json:"monthly_growth"`     // 环比
	Restricted        int64           `json:"restricted"`         // 已限制
	QuotaAlert        int64           `json:"quota_alert"`        // 额度将尽
	ProvidersEnabled  int64           `json:"providers_enabled"`  // 启用厂商
	ModelsEnabled     int64           `json:"models_enabled"`     // 启用模型
	ConversationCount int64           `json:"conversation_count"` // 平台会话总数
	MonthlyMessages   int64           `json:"monthly_messages"`   // 本月消息数
	Health            provider.Health `json:"health"`             // 接入健康
	RecentAudits      []AuditResponse `json:"recent_audits"`      // 最近变更
}

// SettingsResponse 是后台系统设置。
