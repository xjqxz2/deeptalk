package responses

import (
	"deeptalk/internal/billing"
)

type UsageResponse struct {
	UsedTokens  int64             `json:"used_tokens"`
	QuotaTokens int64             `json:"quota_tokens"`
	UsedPercent int               `json:"used_percent"`
	ResetDays   int               `json:"reset_days"`
	Plan        string            `json:"plan"`
	Limits      *billing.Snapshot `json:"limits,omitempty"`
}

// NewUsageResponse 组装用量与限额。
func NewUsage(usage UsageResponse, limits *billing.Snapshot) UsageResponse {
	usage.Limits = limits
	return usage
}
