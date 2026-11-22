package requests

import (
	"github.com/gin-gonic/gin"

	"deeptalk/internal/provider"
)

type EnabledRequest struct {
	Enabled bool `json:"enabled"` // 目标状态
}

// Bind 绑定启用状态请求。
func (r *EnabledRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// ModelSaveRequest 是新增 / 编辑模型请求。
type ModelSaveRequest struct {
	ProviderID       uint    `json:"provider_id"`                         // 所属厂商
	ModelID          string  `json:"model_id" binding:"required,max=128"` // 上游模型 ID
	DisplayName      string  `json:"display_name" binding:"max=64"`       // 展示名
	Description      string  `json:"description" binding:"max=128"`       // 用途
	ContextWindow    int     `json:"context_window"`                      // 上下文窗口
	MaxConcurrency   int     `json:"max_concurrency"`                     // 并发
	Enabled          bool    `json:"enabled"`                             // 是否启用
	PriceInput       float64 `json:"price_input"`                         // 输入单价（元 / 百万 token）
	PriceCachedInput float64 `json:"price_cached_input"`                  // 缓存命中单价
	PriceOutput      float64 `json:"price_output"`                        // 输出单价
}

// Bind 绑定模型保存请求。
func (r *ModelSaveRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// ToInput 转换为领域输入。
func (r *ModelSaveRequest) ToInput() provider.ModelInput {
	return provider.ModelInput{
		ModelID:          r.ModelID,
		DisplayName:      r.DisplayName,
		Description:      r.Description,
		ContextWindow:    r.ContextWindow,
		MaxConcurrency:   r.MaxConcurrency,
		Enabled:          r.Enabled,
		PriceInput:       r.PriceInput,
		PriceCachedInput: r.PriceCachedInput,
		PriceOutput:      r.PriceOutput,
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
