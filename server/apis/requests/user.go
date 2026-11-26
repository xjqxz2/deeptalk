package requests

import (
	"github.com/gin-gonic/gin"
)

type QuotaRequest struct {
	QuotaTokens int64 `json:"quota_tokens" binding:"required,min=1000000,max=100000000"` // 新额度
}

// Bind 绑定额度请求。
func (r *QuotaRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// PlanRequest 是套餐调整请求。
type DisabledRequest struct {
	Disabled bool `json:"disabled"` // true 限制，false 恢复
}

// Bind 绑定限制登录请求。
func (r *DisabledRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// AdminCreateUserRequest 是管理员建号请求。
type AdminCreateUserRequest struct {
	Email       string `json:"email" binding:"required,email,max=190"`    // 邮箱
	Password    string `json:"password" binding:"required,min=8,max=128"` // 初始密码
	Nickname    string `json:"nickname" binding:"max=32"`                 // 昵称
	Plan        string `json:"plan"`                                      // 套餐
	QuotaTokens int64  `json:"quota_tokens"`                              // 额度
}

// Bind 绑定管理员建号请求。
func (r *AdminCreateUserRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// UserListQuery 是后台用户列表查询参数。
type UserListQuery struct {
	Q        string `form:"q"`         // 搜索关键词
	Status   string `form:"status"`    // all / normal / restricted / quota
	Sort     string `form:"sort"`      // recent / created / usage
	Page     int    `form:"page"`      // 页码
	PageSize int    `form:"page_size"` // 每页条数
}

// Bind 绑定查询参数并补齐默认值。
func (q *UserListQuery) Bind(c *gin.Context) error {
	if err := c.ShouldBindQuery(q); err != nil {
		return err
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 8
	}
	if q.Status == "" {
		q.Status = "all"
	}
	if q.Sort == "" {
		q.Sort = "recent"
	}
	return nil
}

// SettingsRequest 是系统设置更新请求，字段均可选。

// RoleRequest 是调整用户角色的请求。
type RoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin user"` // admin / user
}

// Bind 绑定角色请求。
func (r *RoleRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}
