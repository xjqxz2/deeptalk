package b

import (
	"deeptalk/apis/requests"
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
	for i := range logs {
		audits = append(audits, responses.NewAudit(&logs[i]))
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
	var req requests.AuditLogsRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "参数不合法")
		return
	}
	logs, err := h.audits.Recent(c.Request.Context(), req.Limit)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	items := make([]responses.AuditResponse, 0, len(logs))
	for i := range logs {
		items = append(items, responses.NewAudit(&logs[i]))
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
	var req requests.UsageQueryRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "参数不合法")
		return
	}
	items, err := h.chats.UsageTimeline(c.Request.Context(), req.Days)
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
