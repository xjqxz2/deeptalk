package responses

import ()

type SettingsResponse struct {
	AllowRegistration    bool   `json:"allow_registration"`     // 开放注册
	DefaultModelID       uint   `json:"default_model_id"`       // 默认模型
	FallbackModelID      uint   `json:"fallback_model_id"`      // 兜底模型
	MaxOutputTokens      int    `json:"max_output_tokens"`      // 单次输出预算
	SiteName             string `json:"site_name"`              // 站点名
	GenerationTimeoutSec int    `json:"generation_timeout_sec"` // 生成首字节超时（秒）
	SearchProvider       string `json:"search_provider"`        // self / deepseek
	SearchDeepSeekMasked string `json:"search_deepseek_masked"` // 已配置密钥的掩码
	HasSearchKey         bool   `json:"has_search_key"`         // 是否已配置密钥
	Version              string `json:"version"`                // 版本
	StorageUsedBytes     int64  `json:"storage_used_bytes"`     // 存储已用
	StorageTotalBytes    int64  `json:"storage_total_bytes"`    // 存储上限
}

// percent 计算百分比并向上取整。
func percent(used, quota int64) int {
	if quota <= 0 || used <= 0 {
		return 0
	}
	p := int((used*100 + quota - 1) / quota)
	if p > 100 {
		return 100
	}
	return p
}

// avatar 取昵称首字符作为头像。
