// Package requests 集中定义各端请求体与查询参数，并提供统一的 Bind 方法。
// 控制器只调用 Bind，不直接写 ShouldBindJSON / ShouldBindQuery。
package requests

import (
	"deeptalk/internal/provider"

	"github.com/gin-gonic/gin"
)

// RegisterRequest 是 C 端注册请求。
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=190"`    // 邮箱
	Password string `json:"password" binding:"required,min=8,max=128"` // 密码
}

// Bind 绑定并校验注册请求。
func (r *RegisterRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// LoginRequest 是 C 端登录请求。
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=190"`    // 邮箱
	Password string `json:"password" binding:"required,min=1,max=128"` // 密码
}

// Bind 绑定登录请求。
func (r *LoginRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// ChangePasswordRequest 是修改密码请求。
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`               // 当前密码
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"` // 新密码
}

// Bind 绑定修改密码请求。
func (r *ChangePasswordRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// ConversationCreateRequest 是新建会话请求，标题可空。
type ConversationCreateRequest struct {
	Title string `json:"title" binding:"max=60"` // 可选标题
}

// Bind 绑定新建会话请求；允许空 body。
func (r *ConversationCreateRequest) Bind(c *gin.Context) error {
	if c.Request.ContentLength == 0 {
		return nil
	}
	return c.ShouldBindJSON(r)
}

// ConversationRenameRequest 是重命名会话请求。
type ConversationRenameRequest struct {
	Title string `json:"title" binding:"required,min=1,max=60"` // 新标题
}

// Bind 绑定重命名请求。
func (r *ConversationRenameRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// StreamRequest 是发起流式回答的请求。
type StreamRequest struct {
	Content       string `json:"content" binding:"max=8000"` // 用户输入
	ModelID       uint   `json:"model_id"`                   // 模型主键
	DeepThinking  bool   `json:"deep_thinking"`              // 深度思考
	WebSearch     bool   `json:"web_search"`                 // 联网搜索
	AttachmentIDs []uint `json:"attachment_ids"`             // 图片附件 ID
	Regenerate    bool   `json:"regenerate"`                 // 重新生成
}

// Bind 绑定流式请求；重新生成时 content 可为空，业务层再做校验。
func (r *StreamRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// FeedbackRequest 是消息反馈请求。
type FeedbackRequest struct {
	Rating string `json:"rating" binding:"max=8"` // up / down / 空
}

// Bind 绑定反馈请求。
func (r *FeedbackRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// ProviderSaveRequest 是新建 / 编辑厂商请求。
type ProviderSaveRequest struct {
	Name           string                 `json:"name" binding:"required,max=64"`      // 名称
	Kind           string                 `json:"kind"`                                // 厂商类型
	BaseURL        string                 `json:"base_url" binding:"required,max=255"` // Base URL
	APIKey         string                 `json:"api_key" binding:"max=512"`           // API Key，编辑时可为空
	TimeoutSec     int                    `json:"timeout_sec"`                         // 超时
	MaxConcurrency int                    `json:"max_concurrency"`                     // 并发
	Enabled        bool                   `json:"enabled"`                             // 是否启用
	IsDefault      bool                   `json:"is_default"`                          // 平台默认
	Headers        map[string]string      `json:"headers"`                             // 自定义请求头；nil 表示保持不变
	Models         *[]provider.ModelInput `json:"models"`                              // 模型列表
}

// Bind 绑定厂商保存请求。
func (r *ProviderSaveRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// ToInput 转换为领域输入。
func (r *ProviderSaveRequest) ToInput() provider.SaveInput {
	return provider.SaveInput{
		Name:           r.Name,
		Kind:           r.Kind,
		BaseURL:        r.BaseURL,
		APIKey:         r.APIKey,
		TimeoutSec:     r.TimeoutSec,
		MaxConcurrency: r.MaxConcurrency,
		Headers:        r.Headers,
		Enabled:        r.Enabled,
		IsDefault:      r.IsDefault,
		Models:         r.Models,
	}
}

// ProviderTestRequest 是未保存配置的连接测试请求。
type ProviderTestRequest struct {
	ID uint `json:"id"` // 已保存厂商 ID，可为 0
	ProviderSaveRequest
}

// Bind 绑定连接测试请求。
func (r *ProviderTestRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// EnabledRequest 是通用的启用 / 停用请求。
type EnabledRequest struct {
	Enabled bool `json:"enabled"` // 目标状态
}

// Bind 绑定启用状态请求。
func (r *EnabledRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// ModelSaveRequest 是新增 / 编辑模型请求。
type ModelSaveRequest struct {
	ProviderID     uint   `json:"provider_id"`                         // 所属厂商
	ModelID        string `json:"model_id" binding:"required,max=128"` // 上游模型 ID
	DisplayName    string `json:"display_name" binding:"max=64"`       // 展示名
	Description    string `json:"description" binding:"max=128"`       // 用途
	ContextWindow  int    `json:"context_window"`                      // 上下文窗口
	MaxConcurrency int    `json:"max_concurrency"`                     // 并发
	Enabled        bool   `json:"enabled"`                             // 是否启用
}

// Bind 绑定模型保存请求。
func (r *ModelSaveRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// ToInput 转换为领域输入。
func (r *ModelSaveRequest) ToInput() provider.ModelInput {
	return provider.ModelInput{
		ModelID:        r.ModelID,
		DisplayName:    r.DisplayName,
		Description:    r.Description,
		ContextWindow:  r.ContextWindow,
		MaxConcurrency: r.MaxConcurrency,
		Enabled:        r.Enabled,
	}
}

// RoutingRequest 是默认 / 兜底模型路由请求。
type RoutingRequest struct {
	DefaultModelID  uint `json:"default_model_id" binding:"required,min=1"` // 默认模型
	FallbackModelID uint `json:"fallback_model_id"`                         // 兜底模型
}

// Bind 绑定路由请求。
func (r *RoutingRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// QuotaRequest 是额度调整请求。
type QuotaRequest struct {
	QuotaTokens int64 `json:"quota_tokens" binding:"required,min=1000000,max=100000000"` // 新额度
}

// Bind 绑定额度请求。
func (r *QuotaRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// PlanRequest 是套餐调整请求。
type PlanRequest struct {
	Plan string `json:"plan" binding:"required,oneof=free pro team unlimited"` // free / pro / team / unlimited
}

// Bind 绑定套餐请求。
func (r *PlanRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// DisabledRequest 是限制登录请求。
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
type SettingsRequest struct {
	AllowRegistration *bool   `json:"allow_registration"` // 开放注册
	DefaultModelID    *uint   `json:"default_model_id"`   // 默认模型
	FallbackModelID   *uint   `json:"fallback_model_id"`  // 兜底模型
	MaxOutputTokens   *int    `json:"max_output_tokens"`  // 单次输出预算
	SiteName          *string `json:"site_name"`          // 站点名称
}

// Bind 绑定系统设置请求。
func (r *SettingsRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}
