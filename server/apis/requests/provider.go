package requests

import (
	"github.com/gin-gonic/gin"

	"deeptalk/internal/provider"
)

type ProviderSaveRequest struct {
	Name           string                 `json:"name" binding:"required,max=64"`      // 名称
	Kind           string                 `json:"kind"`                                // 厂商类型
	APIStyle       string                 `json:"api_style"`                           // 接入方式：chat / responses / anthropic
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
		APIStyle:       r.APIStyle,
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

// ProviderListRequest 是厂商列表筛选。
type ProviderListRequest struct {
	Filter string `form:"filter" binding:"omitempty,oneof=all enabled disabled"` // 筛选
}

// Bind 绑定查询参数。
func (r *ProviderListRequest) Bind(c *gin.Context) error {
	return c.ShouldBindQuery(r)
}
