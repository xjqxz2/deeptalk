package b

import (
	"deeptalk/apis/responses"
	"deeptalk/internal/audit"
	"deeptalk/internal/chat"
	"deeptalk/internal/provider"
	"deeptalk/internal/user"

	"github.com/gin-gonic/gin"
)

// OverviewController 汇总后台概览与最近变更。
type OverviewController struct {
	users     *user.Service     // 账号统计
	providers *provider.Service // 厂商与模型统计
	chats     *chat.Service     // 会话与消息统计
	audits    *audit.Service    // 审计记录
}

// NewOverviewController 创建概览控制器。
func NewOverviewController(users *user.Service, providers *provider.Service, chats *chat.Service, audits *audit.Service) *OverviewController {
	return &OverviewController{users: users, providers: providers, chats: chats, audits: audits}
}

// Overview 返回后台首页统计。
func (h *OverviewController) Overview(c *gin.Context) {
	ctx := c.Request.Context()
	stats, err := h.users.Stats(ctx)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	providersEnabled, modelsEnabled, err := h.providers.EnabledCounts(ctx)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	health, err := h.providers.Health(ctx)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	logs, err := h.audits.Recent(ctx, 6)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	conversations, err := h.chats.CountConversations(ctx)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	monthlyMessages, err := h.chats.CountMonthlyMessages(ctx)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	audits := make([]responses.AuditResponse, 0, len(logs))
	for _, item := range logs {
		audits = append(audits, responses.NewAudit(item))
	}
	responses.OK(c, responses.OverviewResponse{
		TotalUsers:        stats.Total,
		NewThisMonth:      stats.NewThisMonth,
		Active7D:          stats.Active7D,
		ActiveRate:        stats.ActiveRate,
		MonthlyTokens:     stats.MonthlyTokens,
		MonthlyGrowth:     stats.MonthlyGrowth,
		Restricted:        stats.Restricted,
		QuotaAlert:        stats.QuotaAlert,
		ProvidersEnabled:  providersEnabled,
		ModelsEnabled:     modelsEnabled,
		ConversationCount: conversations,
		MonthlyMessages:   monthlyMessages,
		Health:            health,
		RecentAudits:      audits,
	})
}

// AuditLogs 返回最近变更列表。
func (h *OverviewController) AuditLogs(c *gin.Context) {
	limit := 6
	if raw := c.Query("limit"); raw != "" {
		if n, err := parsePositive(raw); err == nil {
			limit = n
		}
	}
	logs, err := h.audits.Recent(c.Request.Context(), limit)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	items := make([]responses.AuditResponse, 0, len(logs))
	for _, item := range logs {
		items = append(items, responses.NewAudit(item))
	}
	responses.OK(c, gin.H{"items": items})
}

// UsageController 提供用量统计时间序列。
type UsageController struct {
	chats *chat.Service // 消息与用量
}

// NewUsageController 创建用量控制器。
func NewUsageController(chats *chat.Service) *UsageController {
	return &UsageController{chats: chats}
}

// Usage 返回最近 N 天的 tokens 与消息量。
func (h *UsageController) Usage(c *gin.Context) {
	days := 14
	if raw := c.Query("days"); raw != "" {
		if n, err := parsePositive(raw); err == nil {
			days = n
		}
	}
	items, err := h.chats.UsageTimeline(c.Request.Context(), days)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	var totalTokens, totalMessages int64
	for _, item := range items {
		totalTokens += item.Tokens
		totalMessages += item.Messages
	}
	responses.OK(c, gin.H{
		"days": items,
		"totals": gin.H{
			"tokens":   totalTokens,
			"messages": totalMessages,
		},
	})
}
