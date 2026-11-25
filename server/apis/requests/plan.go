package requests

import (
	"github.com/gin-gonic/gin"
)

type PlanRequest struct {
	Plan string `json:"plan" binding:"required,oneof=free pro team unlimited"` // free / pro / team / unlimited
}

// Bind 绑定套餐请求。
func (r *PlanRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// DisabledRequest 是限制登录请求。

// PlanUpdateRequest 是修改套餐请求；指针字段表示「不修改」。
type PlanUpdateRequest struct {
	Name            string `json:"name" binding:"max=60"`         // 套餐名
	Description     string `json:"description" binding:"max=200"` // 描述
	SpendLimitCents *int64 `json:"spend_limit_cents"`             // 消费上限（分）
	LimitShort      *int   `json:"limit_short"`                   // 5 小时请求上限
	LimitWeek       *int   `json:"limit_week"`                    // 本周上限
	LimitMonth      *int   `json:"limit_month"`                   // 本月上限
	Enabled         *bool  `json:"enabled"`                       // 是否启用
}

// Bind 绑定套餐修改请求。
func (r *PlanUpdateRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}
