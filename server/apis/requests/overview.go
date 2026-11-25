package requests

import (
	"github.com/gin-gonic/gin"
)

// AuditLogsRequest 是最近变更列表查询。
type AuditLogsRequest struct {
	Limit int `form:"limit" binding:"omitempty,min=1,max=200"` // 条数
}

// Bind 绑定查询参数并补齐默认值。
func (r *AuditLogsRequest) Bind(c *gin.Context) error {
	if err := c.ShouldBindQuery(r); err != nil {
		return err
	}
	if r.Limit <= 0 {
		r.Limit = 6
	}
	return nil
}

// UsageQueryRequest 是用量时间序列查询。
type UsageQueryRequest struct {
	Days int `form:"days" binding:"omitempty,min=1,max=365"` // 天数
}

// Bind 绑定查询参数并补齐默认值。
func (r *UsageQueryRequest) Bind(c *gin.Context) error {
	if err := c.ShouldBindQuery(r); err != nil {
		return err
	}
	if r.Days <= 0 {
		r.Days = 14
	}
	return nil
}
