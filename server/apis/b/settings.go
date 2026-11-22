package b

import (
	"os"
	"strconv"
	"strings"

	"deeptalk/apis/requests"
	"deeptalk/apis/responses"
	"deeptalk/internal/audit"
	"deeptalk/internal/provider"
	"deeptalk/internal/search"
	"deeptalk/internal/setting"

	"github.com/gin-gonic/gin"
)

// SettingsController 处理系统设置：注册开关、默认路由与存储信息。
type SettingsController struct {
	settings  *setting.Service  // 设置读写
	providers *provider.Service // 路由能力
	audits    *audit.Service    // 审计
	dbPath    string            // SQLite 文件路径，用于展示存储占用
	secret    string            // 用于加密 DeepSeek 检索密钥
}

// NewSettingsController 创建设置控制器。
func NewSettingsController(settings *setting.Service, providers *provider.Service, audits *audit.Service, dbPath, secret string) *SettingsController {
	return &SettingsController{settings: settings, providers: providers, audits: audits, dbPath: dbPath, secret: secret}
}

// Get 返回当前系统设置。
func (h *SettingsController) Get(c *gin.Context) {
	responses.OK(c, h.snapshot(c))
}

// Update 更新注册开关与默认路由。
func (h *SettingsController) Update(c *gin.Context) {
	var req requests.SettingsRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "设置内容不合法")
		return
	}
	ctx := c.Request.Context()
	if req.AllowRegistration != nil {
		value := "false"
		if *req.AllowRegistration {
			value = "true"
		}
		if err := h.settings.Set(ctx, setting.KeyAllowRegistration, value); err != nil {
			responses.FromError(c, err)
			return
		}
		detail := "关闭开放注册"
		if *req.AllowRegistration {
			detail = "开放注册"
		}
		recordAudit(c, h.audits, "setting.registration", "setting", 0, detail)
	}
	if req.SearchProvider != nil {
		mode := strings.ToLower(strings.TrimSpace(*req.SearchProvider))
		if mode != search.ProviderSelf && mode != search.ProviderDeepSeek && mode != search.ProviderBuiltin {
			responses.Fail(c, responses.CodeInvalidArgument, "检索来源只能是 自建、DeepSeek 或 模型内置")
			return
		}
		if err := h.settings.Set(ctx, setting.KeySearchProvider, mode); err != nil {
			responses.FromError(c, err)
			return
		}
		recordAudit(c, h.audits, "setting.search_provider", "setting", 0, "联网检索来源改为 "+mode)
	}
	if req.SearchDeepSeekKey != nil {
		key := strings.TrimSpace(*req.SearchDeepSeekKey)
		if key != "" {
			enc, err := provider.EncryptAPIKey(h.secret, key)
			if err != nil {
				responses.FromError(c, err)
				return
			}
			if err := h.settings.Set(ctx, setting.KeySearchDeepSeekKey, enc); err != nil {
				responses.FromError(c, err)
				return
			}
			recordAudit(c, h.audits, "setting.search_key", "setting", 0, "更新 DeepSeek 检索密钥")
		}
	}
	if req.GenerationTimeoutSec != nil {
		value := *req.GenerationTimeoutSec
		if value < 10 || value > 300 {
			responses.Fail(c, responses.CodeInvalidArgument, "生成超时需在 10–300 秒之间")
			return
		}
		if err := h.settings.Set(ctx, setting.KeyGenerationTimeout, strconv.Itoa(value)); err != nil {
			responses.FromError(c, err)
			return
		}
		recordAudit(c, h.audits, "setting.generation_timeout", "setting", 0, "生成首字节超时调整为 "+strconv.Itoa(value)+" 秒")
	}
	if req.SiteName != nil {
		name := strings.TrimSpace(*req.SiteName)
		if name == "" || len([]rune(name)) > 32 {
			responses.Fail(c, responses.CodeInvalidArgument, "站点名称需为 1–32 个字符")
			return
		}
		if err := h.settings.Set(ctx, setting.KeySiteName, name); err != nil {
			responses.FromError(c, err)
			return
		}
		recordAudit(c, h.audits, "setting.site_name", "setting", 0, "修改站点名称为 "+name)
	}
	if req.MaxOutputTokens != nil {
		if *req.MaxOutputTokens < 1024 || *req.MaxOutputTokens > 1_000_000 {
			responses.Fail(c, responses.CodeInvalidArgument, "输出预算需在 1024–1,000,000 tokens 之间")
			return
		}
		if err := h.settings.Set(ctx, setting.KeyMaxOutputTokens, strconv.Itoa(*req.MaxOutputTokens)); err != nil {
			responses.FromError(c, err)
			return
		}
		recordAudit(c, h.audits, "setting.output_budget", "setting", 0,
			"调整单次输出预算为 "+strconv.Itoa(*req.MaxOutputTokens)+" tokens")
	}
	if req.DefaultModelID != nil || req.FallbackModelID != nil {
		current := h.providers.Routing(ctx)
		defaultID := current.DefaultModelID
		fallbackID := current.FallbackModelID
		if req.DefaultModelID != nil {
			defaultID = *req.DefaultModelID
		}
		if req.FallbackModelID != nil {
			fallbackID = *req.FallbackModelID
		}
		if err := h.providers.SetRouting(ctx, defaultID, fallbackID); err != nil {
			responses.FromError(c, err)
			return
		}
		recordAudit(c, h.audits, "routing.update", "setting", 0, "更新默认 / 兜底模型路由")
	}
	responses.OK(c, h.snapshot(c))
}

// snapshot 组装设置响应，包含存储占用。
func (h *SettingsController) snapshot(c *gin.Context) responses.SettingsResponse {
	ctx := c.Request.Context()
	routing := h.providers.Routing(ctx)
	used := fileSize(h.dbPath) + fileSize(h.dbPath+"-wal")
	return responses.SettingsResponse{
		AllowRegistration:    h.settings.Bool(ctx, setting.KeyAllowRegistration, true),
		DefaultModelID:       routing.DefaultModelID,
		FallbackModelID:      routing.FallbackModelID,
		MaxOutputTokens:      h.settings.Int(ctx, setting.KeyMaxOutputTokens, 384000),
		SiteName:             h.settings.Get(ctx, setting.KeySiteName, "小橘"),
		GenerationTimeoutSec: h.settings.Int(ctx, setting.KeyGenerationTimeout, 30),
		SearchProvider:       h.settings.Get(ctx, setting.KeySearchProvider, search.ProviderSelf),
		SearchDeepSeekMasked: maskSearchKey(h.settings.Get(ctx, setting.KeySearchDeepSeekKey, "")),
		HasSearchKey:         h.settings.Get(ctx, setting.KeySearchDeepSeekKey, "") != "",
		Version:              "v1.0.0",
		StorageUsedBytes:     used,
		StorageTotalBytes:    10 * 1024 * 1024 * 1024,
	}
}

// fileSize 返回文件字节数，文件不存在时返回 0。
func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// maskSearchKey 只回显密文长度，前端永远拿不到明文密钥。
func maskSearchKey(encoded string) string {
	if encoded == "" {
		return ""
	}
	return "已配置（" + strconv.Itoa(len(encoded)) + " 字符密文）"
}
