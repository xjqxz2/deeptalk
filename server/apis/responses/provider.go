package responses

import (
	"deeptalk/internal/provider"
)

type ProviderResponse struct {
	ID             uint                 `json:"id"`              // 厂商 ID
	Name           string               `json:"name"`            // 名称
	Code           string               `json:"code"`            // 唯一标识
	Kind           string               `json:"kind"`            // 厂商类型
	APIStyle       string               `json:"api_style"`       // 接入方式
	BaseURL        string               `json:"base_url"`        // Base URL
	APIKeyMasked   string               `json:"api_key_masked"`  // Key 掩码
	HasAPIKey      bool                 `json:"has_api_key"`     // 是否已配置 Key
	TimeoutSec     int                  `json:"timeout_sec"`     // 超时
	MaxConcurrency int                  `json:"max_concurrency"` // 并发
	Enabled        bool                 `json:"enabled"`         // 是否启用
	Verified       bool                 `json:"verified"`        // 是否验证通过
	Status         string               `json:"status"`          // 状态胶囊
	IsBuiltin      bool                 `json:"is_builtin"`      // 预置厂商
	IsDefault      bool                 `json:"is_default"`      // 平台默认
	ModelCount     int                  `json:"model_count"`     // 模型数
	LastLatencyMS  int                  `json:"last_latency_ms"` // 最近延迟
	LastError      string               `json:"last_error"`      // 最近失败原因
	Headers        map[string]string    `json:"headers"`         // 自定义请求头
	Models         []AdminModelResponse `json:"models"`          // 模型列表
}

// NewProvider 构造后台厂商响应；status 由调用方传入，避免响应层依赖业务方法。
func NewProvider(p provider.Provider, status string) ProviderResponse {
	models := make([]AdminModelResponse, 0, len(p.Models))
	for _, m := range p.Models {
		m.ProviderName = p.Name
		models = append(models, NewAdminModel(m))
	}
	return ProviderResponse{
		ID:             p.ID,
		Name:           p.Name,
		Code:           p.Code,
		Kind:           p.Kind,
		APIStyle:       provider.NormalizeAPIStyle(p.APIStyle),
		BaseURL:        p.BaseURL,
		APIKeyMasked:   p.APIKeyMasked,
		HasAPIKey:      p.APIKeyEnc != "",
		TimeoutSec:     p.TimeoutSec,
		MaxConcurrency: p.MaxConcurrency,
		Enabled:        p.Enabled,
		Verified:       p.Verified,
		Status:         status,
		IsBuiltin:      p.IsBuiltin,
		IsDefault:      p.IsDefault,
		ModelCount:     len(p.Models),
		LastLatencyMS:  p.LastLatencyMS,
		LastError:      p.LastError,
		Headers:        provider.ProviderHeaders(&p),
		Models:         models,
	}
}

// AuditResponse 是最近变更条目。
