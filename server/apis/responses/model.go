package responses

import (
	"deeptalk/internal/provider"
)

type ModelResponse struct {
	ID            uint   `json:"id"`             // 模型主键
	ModelID       string `json:"model_id"`       // 上游模型 ID
	DisplayName   string `json:"display_name"`   // 展示名
	Description   string `json:"description"`    // 用途说明
	ProviderName  string `json:"provider_name"`  // 厂商名
	ContextWindow int    `json:"context_window"` // 上下文窗口
	IsDefault     bool   `json:"is_default"`     // 平台默认
}

// NewModel 构造 C 端模型响应。
func NewModel(m provider.Model) ModelResponse {
	return ModelResponse{
		ID:            m.ID,
		ModelID:       m.ModelID,
		DisplayName:   m.DisplayName,
		Description:   m.Description,
		ProviderName:  m.ProviderName,
		ContextWindow: m.ContextWindow,
		IsDefault:     m.IsDefault,
	}
}

// ConversationResponse 是会话展示结构。
type AdminModelResponse struct {
	ID               uint    `json:"id"`                 // 模型主键
	ProviderID       uint    `json:"provider_id"`        // 厂商 ID
	ProviderName     string  `json:"provider_name"`      // 厂商名
	ModelID          string  `json:"model_id"`           // 上游模型 ID
	DisplayName      string  `json:"display_name"`       // 展示名
	Description      string  `json:"description"`        // 用途说明
	ContextWindow    int     `json:"context_window"`     // 上下文窗口
	MaxConcurrency   int     `json:"max_concurrency"`    // 并发
	Enabled          bool    `json:"enabled"`            // 是否启用
	IsDefault        bool    `json:"is_default"`         // 默认模型
	IsFallback       bool    `json:"is_fallback"`        // 兜底模型
	PriceInput       float64 `json:"price_input"`        // 输入单价（元 / 百万 token）
	PriceCachedInput float64 `json:"price_cached_input"` // 缓存命中单价
	PriceOutput      float64 `json:"price_output"`       // 输出单价
}

// NewAdminModel 构造后台模型响应。
func NewAdminModel(m provider.Model) AdminModelResponse {
	return AdminModelResponse{
		ID:               m.ID,
		ProviderID:       m.ProviderID,
		ProviderName:     m.ProviderName,
		ModelID:          m.ModelID,
		DisplayName:      m.DisplayName,
		Description:      m.Description,
		ContextWindow:    m.ContextWindow,
		MaxConcurrency:   m.MaxConcurrency,
		Enabled:          m.Enabled,
		IsDefault:        m.IsDefault,
		IsFallback:       m.IsFallback,
		PriceInput:       m.PriceInput,
		PriceCachedInput: m.PriceCachedInput,
		PriceOutput:      m.PriceOutput,
	}
}

// ProviderResponse 是后台厂商卡片数据。
