package requests

import (
	"github.com/gin-gonic/gin"
)

type SettingsRequest struct {
	AllowRegistration    *bool   `json:"allow_registration"`     // 开放注册
	DefaultModelID       *uint   `json:"default_model_id"`       // 默认模型
	FallbackModelID      *uint   `json:"fallback_model_id"`      // 兜底模型
	MaxOutputTokens      *int    `json:"max_output_tokens"`      // 单次输出预算
	SiteName             *string `json:"site_name"`              // 站点名称
	GenerationTimeoutSec *int    `json:"generation_timeout_sec"` // 生成首字节超时（秒）
	SearchProvider       *string `json:"search_provider"`        // self / deepseek
	SearchDeepSeekKey    *string `json:"search_deepseek_key"`    // DeepSeek 检索密钥，留空表示不修改
}

// Bind 绑定系统设置请求。
func (r *SettingsRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}
