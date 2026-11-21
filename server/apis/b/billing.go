package b

import (
	"strconv"

	"deeptalk/apis/responses"
	"deeptalk/internal/billing"

	"github.com/gin-gonic/gin"
)

// BillingController 管理套餐限额（后台可随时调整三档限制）。
type BillingController struct {
	billing *billing.Service
}

// NewBillingController 创建套餐控制器。
func NewBillingController(svc *billing.Service) *BillingController {
	return &BillingController{billing: svc}
}

// List 返回全部套餐（含停用）。
func (h *BillingController) List(c *gin.Context) {
	items, err := h.billing.AllPlans(c.Request.Context())
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, gin.H{"items": items})
}

// Update 更新单个套餐的限额。
func (h *BillingController) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "套餐 ID 不合法")
		return
	}
	var req struct {
		Name            string `json:"name"`
		Description     string `json:"description"`
		SpendLimitCents *int64 `json:"spend_limit_cents"`
		LimitShort      *int   `json:"limit_short"`
		LimitWeek       *int   `json:"limit_week"`
		LimitMonth      *int   `json:"limit_month"`
		Enabled         *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请求内容不合法")
		return
	}
	plan, err := h.billing.Update(c.Request.Context(), uint(id), billing.UpdateInput{
		Name: req.Name, Description: req.Description,
		SpendLimitCents: req.SpendLimitCents, LimitShort: req.LimitShort,
		LimitWeek: req.LimitWeek, LimitMonth: req.LimitMonth, Enabled: req.Enabled,
	})
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, gin.H{"plan": plan})
}
