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
