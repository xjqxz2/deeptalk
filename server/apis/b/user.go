package b

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"deeptalk/apis/middleware"
	"deeptalk/apis/requests"
	"deeptalk/apis/responses"
	"deeptalk/internal/audit"
	"deeptalk/internal/billing"
	"deeptalk/internal/chat"
	"deeptalk/internal/user"

	"github.com/gin-gonic/gin"
)

// UserController 处理后台用户列表、详情与账号处置。
type UserController struct {
	users   *user.Service    // 账号能力
	chats   *chat.Service    // 会话统计
	audits  *audit.Service   // 审计
	billing *billing.Service // 计费数据清理
}

// NewUserController 创建用户管理控制器。
func NewUserController(users *user.Service, chats *chat.Service, audits *audit.Service, billingSvc *billing.Service) *UserController {
	return &UserController{users: users, chats: chats, audits: audits, billing: billingSvc}
}

// Stats 返回用户管理统计卡数据。
func (h *UserController) Stats(c *gin.Context) {
	stats, err := h.users.Stats(c.Request.Context())
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, stats)
}

// List 返回分页用户列表。
func (h *UserController) List(c *gin.Context) {
	var query requests.UserListQuery
	if err := query.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "查询参数不合法")
		return
	}
	ctx := c.Request.Context()
	items, total, err := h.users.List(ctx, user.ListParams{
		Query:    query.Q,
		Status:   query.Status,
		Sort:     query.Sort,
		Page:     query.Page,
		PageSize: query.PageSize,
	})
	if err != nil {
		responses.FromError(c, err)
		return
	}
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	counts, err := h.chats.CountByUsers(ctx, ids)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	out := make([]responses.UserResponse, 0, len(items))
	for i := range items {
		resp := responses.NewUser(&items[i])
		resp.ConversationCount = counts[items[i].ID]
		out = append(out, resp)
	}
	stats, err := h.users.Stats(ctx)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, gin.H{
		"items":     out,
		"total":     total,
		"page":      query.Page,
		"page_size": query.PageSize,
		"stats":     stats,
	})
}

// Detail 返回单个账号详情；不包含任何对话正文。
func (h *UserController) Detail(c *gin.Context) {
	var idReq requests.IDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "用户 ID 不合法")
		return
	}
	id := idReq.ID
	ctx := c.Request.Context()
	u, err := h.users.ByID(ctx, id)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	conversations, messages, err := h.chats.UserStats(ctx, id)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	usage := h.users.Usage(u)
	resp := responses.NewUser(u)
	resp.ConversationCount = conversations
	resp.MessageCount = messages
	resp.QuotaResetDays = usage.ResetDays
	responses.OK(c, gin.H{"user": resp})
}

// UpdateQuota 调整账号额度。
func (h *UserController) UpdateQuota(c *gin.Context) {
	var idReq requests.IDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "用户 ID 不合法")
		return
	}
	id := idReq.ID
	var req requests.QuotaRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "额度需在 1M–100M 之间")
		return
	}
	before, _ := h.users.ByID(c.Request.Context(), id)
	u, err := h.users.UpdateQuota(c.Request.Context(), id, req.QuotaTokens)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	oldQuota := int64(0)
	if before != nil {
		oldQuota = before.QuotaTokens
	}
	recordAudit(c, h.audits, "user.quota", "user", id,
		fmt.Sprintf("调整额度 %s → %s", formatTokens(oldQuota), formatTokens(u.QuotaTokens)))
	responses.OK(c, gin.H{"user": responses.NewUser(u)})
}

// UpdatePlan 调整账号套餐。
func (h *UserController) UpdatePlan(c *gin.Context) {
	var idReq requests.IDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "用户 ID 不合法")
		return
	}
	id := idReq.ID
	var req requests.PlanRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "套餐不合法")
		return
	}
	u, err := h.users.UpdatePlan(c.Request.Context(), id, req.Plan)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	recordAudit(c, h.audits, "user.plan", "user", id, "调整套餐为 "+u.Plan)
	responses.OK(c, gin.H{"user": responses.NewUser(u)})
}

// ToggleDisabled 限制 / 恢复账号登录。
func (h *UserController) ToggleDisabled(c *gin.Context) {
	var idReq requests.IDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "用户 ID 不合法")
		return
	}
	id := idReq.ID
	var req requests.DisabledRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "状态不合法")
		return
	}
	u, err := h.users.SetDisabled(c.Request.Context(), id, req.Disabled)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	if req.Disabled {
		recordAudit(c, h.audits, "user.disable", "user", id, "限制登录 "+u.Email)
	} else {
		recordAudit(c, h.audits, "user.enable", "user", id, "恢复登录 "+u.Email)
	}
	responses.OK(c, gin.H{"user": responses.NewUser(u)})
}

// SetRole 设为 / 取消管理员。
func (h *UserController) SetRole(c *gin.Context) {
	var idReq requests.IDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "用户 ID 不合法")
		return
	}
	var req requests.RoleRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "角色不合法")
		return
	}
	u, err := h.users.SetRole(c.Request.Context(), middleware.CurrentUser(c).ID, idReq.ID, req.Role)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	if req.Role == user.RoleAdmin {
		recordAudit(c, h.audits, "user.promote", "user", idReq.ID, "设为管理员 "+u.Email)
	} else {
		recordAudit(c, h.audits, "user.demote", "user", idReq.ID, "取消管理员 "+u.Email)
	}
	responses.OK(c, gin.H{"user": responses.NewUser(u)})
}

// Delete 删除用户及其会话、计费数据。
func (h *UserController) Delete(c *gin.Context) {
	var idReq requests.IDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "用户 ID 不合法")
		return
	}
	id := idReq.ID
	ctx := c.Request.Context()
	u, err := h.users.ByID(ctx, id)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	if err := h.chats.PurgeUser(ctx, id); err != nil {
		responses.FromError(c, err)
		return
	}
	if h.billing != nil {
		_ = h.billing.PurgeUser(ctx, id)
	}
	if err := h.users.Delete(ctx, id); err != nil {
		responses.FromError(c, err)
		return
	}
	recordAudit(c, h.audits, "user.delete", "user", id, "删除用户 "+u.Email)
	responses.OK(c, gin.H{"deleted": true})
}

// ResetPassword 重置密码并返回一次性密码。
func (h *UserController) ResetPassword(c *gin.Context) {
	var idReq requests.IDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "用户 ID 不合法")
		return
	}
	id := idReq.ID
	u, err := h.users.ByID(c.Request.Context(), id)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	password, err := h.users.ResetPassword(c.Request.Context(), id)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	recordAudit(c, h.audits, "user.reset_password", "user", id, "重置密码 "+u.Email)
	responses.OK(c, gin.H{
		"password":     password,
		"expires_hint": "仅显示一次，请立即转交用户",
	})
}

// LogoutAll 强制退出账号全部设备。
func (h *UserController) LogoutAll(c *gin.Context) {
	var idReq requests.IDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "用户 ID 不合法")
		return
	}
	id := idReq.ID
	u, err := h.users.ByID(c.Request.Context(), id)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	if err := h.users.RevokeAllSessions(c.Request.Context(), id); err != nil {
		responses.FromError(c, err)
		return
	}
	recordAudit(c, h.audits, "user.logout_all", "user", id, "强制退出全部设备 "+u.Email)
	responses.OK(c, nil)
}

// Create 管理员建号（注册关闭时使用）。
func (h *UserController) Create(c *gin.Context) {
	var req requests.AdminCreateUserRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请填写正确的邮箱与密码")
		return
	}
	u, err := h.users.CreateByAdmin(c.Request.Context(), user.AdminCreateInput{
		Email:       req.Email,
		Password:    req.Password,
		Nickname:    req.Nickname,
		Plan:        req.Plan,
		QuotaTokens: req.QuotaTokens,
	})
	if err != nil {
		responses.FromError(c, err)
		return
	}
	recordAudit(c, h.audits, "user.create", "user", u.ID, "创建账号 "+u.Email)
	responses.OK(c, gin.H{"user": responses.NewUser(u)})
}

// Export 导出账号 CSV；只含账号字段，不含对话正文。
func (h *UserController) Export(c *gin.Context) {
	items, err := h.users.ExportAll(c.Request.Context())
	if err != nil {
		responses.FromError(c, err)
		return
	}
	filename := "deeptalk-users-" + time.Now().Format("20060102") + ".csv"
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Status(http.StatusOK)
	// UTF-8 BOM 保证 Excel 打开中文不乱码
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"id", "email", "nickname", "role", "plan", "quota_tokens", "used_tokens", "disabled", "created_at", "last_login_at"})
	for _, u := range items {
		lastLogin := ""
		if u.LastLoginAt != nil {
			lastLogin = u.LastLoginAt.Format(time.RFC3339)
		}
		_ = writer.Write([]string{
			strconv.FormatUint(uint64(u.ID), 10),
			u.Email,
			u.Nickname,
			u.Role,
			u.Plan,
			strconv.FormatInt(u.QuotaTokens, 10),
			strconv.FormatInt(u.UsedTokens, 10),
			strconv.FormatBool(u.Disabled),
			u.CreatedAt.Format(time.RFC3339),
			lastLogin,
		})
	}
	writer.Flush()
	recordAudit(c, h.audits, "user.export", "user", 0, "导出账号 CSV（仅账号字段）")
}

// formatTokens 将 tokens 数格式化为可读的 M / K 单位，用于审计摘要。
func formatTokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.2gM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.2gK", float64(n)/1_000)
	default:
		return strconv.FormatInt(n, 10)
	}
}
