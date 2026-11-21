// Package bootstrap 负责服务的装配与生命周期管理。
// 启动顺序固定为：配置 -> 日志 -> 数据库 -> 业务服务 -> 种子数据 -> HTTP；
// 关闭顺序完全相反，保证连接先于数据库释放。
package bootstrap

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 是服务端的全部可配置项。
// 字段均为值类型或切片，加载后只读，避免运行期被并发修改。
type Config struct {
	App       AppConfig       `yaml:"app"`        // 应用基础信息
	Database  DatabaseConfig  `yaml:"database"`   // 数据库配置
	Auth      AuthConfig      `yaml:"auth"`       // 账号与会话策略
	Security  SecurityConfig  `yaml:"security"`   // 加密与签名密钥
	CORS      CORSConfig      `yaml:"cors"`       // 跨域来源
	Upstream  UpstreamConfig  `yaml:"upstream"`   // 上游模型调用策略
	WebSearch WebSearchConfig `yaml:"web_search"` // 联网检索策略
	MCP       MCPConfig       `yaml:"mcp"`        // MCP client（当前对接 Obscura）
	Uploads   UploadsConfig   `yaml:"uploads"`    // 图片附件存储
	WeChat    WeChatConfig    `yaml:"wechat"`     // 小程序凭据（access_token 管理）
	Context   ContextConfig   `yaml:"context"`    // 长上下文压缩
}

// AppConfig 保存进程级基础配置。
type AppConfig struct {
	Name     string `yaml:"name"`     // 应用名
	Env      string `yaml:"env"`      // dev / prod
	Addr     string `yaml:"addr"`     // 监听地址
	BaseURL  string `yaml:"base_url"` // 对外访问地址
	Location string `yaml:"location"` // 位置兜底；IP 定位失败时注入
	Timezone string `yaml:"timezone"` // 时区兜底；IP 定位失败时使用
}

// DatabaseConfig 保存 SQLite 相关配置。
type DatabaseConfig struct {
	DSN  string `yaml:"dsn"`  // 数据库文件路径
	Seed bool   `yaml:"seed"` // 是否写入演示种子数据
}

// AuthConfig 保存登录、会话与锁定策略。
type AuthConfig struct {
	CookieName         string `yaml:"cookie_name"`          // 会话 Cookie 名
	CookieSecure       bool   `yaml:"cookie_secure"`        // 是否仅 HTTPS 传输
	SessionTTLHours    int    `yaml:"session_ttl_hours"`    // 会话有效期（小时）
	LockThreshold      int    `yaml:"lock_threshold"`       // 连续失败锁定阈值
	LockMinutes        int    `yaml:"lock_minutes"`         // 锁定时长（分钟）
	DefaultQuotaTokens int64  `yaml:"default_quota_tokens"` // 新用户默认额度
}

// SecurityConfig 保存敏感配置。
type SecurityConfig struct {
	Secret   string `yaml:"secret"`    // AES-GCM 与签名密钥派生源
	SignMode string `yaml:"sign_mode"` // off / log / enforce
}

// CORSConfig 保存允许跨域的来源列表。
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"` // 允许携带 Cookie 的来源
}

// UpstreamConfig 保存上游模型的调用策略。
// ContextConfig 是长上下文压缩策略。
type ContextConfig struct {
	CompactRatio   float64 `yaml:"compact_ratio"`    // 达到上下文窗口的百分比时触发压缩
	KeepRatio      float64 `yaml:"keep_ratio"`       // 压缩后保留的最近上下文百分比
	CacheIdleHours int     `yaml:"cache_idle_hours"` // 空闲多久后尝试在缓存失效前压缩
	IdleMinTokens  int     `yaml:"idle_min_tokens"`  // 空闲压缩的最小上下文 token 数
}

type UpstreamConfig struct {
	MaxOutputTokens     int    `yaml:"max_output_tokens"`      // 单次回答最大输出 tokens
	FirstByteTimeoutSec int    `yaml:"first_byte_timeout_sec"` // 首字节超时（秒）
	ReasoningEffort     string `yaml:"reasoning_effort"`       // 深度思考强度：low / medium / high
	DeepSeekAPIKey      string `yaml:"deepseek_api_key"`       // 仅本地种子使用，加密入库后不再读取
}

// WebSearchConfig 保存联网检索配置。
// 检索页由 Obscura 抓取；Obscura 不可用时可降级到普通 HTTP，便于本地开发。
type SearchEngineConfig struct {
	Name   string  `yaml:"name"`
	URL    string  `yaml:"url"`
	Weight float64 `yaml:"weight"`
}

type WebSearchConfig struct {
	Enabled      bool                 `yaml:"enabled"`     // 是否启用联网搜索
	ObscuraBin   string               `yaml:"obscura_bin"` // Obscura 可执行文件
	EngineURL    string               `yaml:"engine_url"`
	Engines      []SearchEngineConfig `yaml:"engines"`       // 搜索页模板，支持 {query}
	MaxResults   int                  `yaml:"max_results"`   // 最多引用条数
	Concurrency  int                  `yaml:"concurrency"`   // 多引擎并发数
	TimeoutSec   int                  `yaml:"timeout_sec"`   // 单次检索超时（秒）
	HTTPFallback bool                 `yaml:"http_fallback"` // Obscura 失败时是否允许 HTTP 回退
	Stealth      bool                 `yaml:"stealth"`       // 是否启用 Obscura 反检测
}

// MCPConfig 保存 stdio MCP server 的启动参数。
// 当前只连接一个 server（Obscura），后续需要多 server 时再扩展为列表。
type MCPConfig struct {
	Enabled         bool              `yaml:"enabled"`          // 是否启用 MCP client
	Command         string            `yaml:"command"`          // MCP server 可执行文件
	Args            []string          `yaml:"args"`             // 启动参数
	Env             map[string]string `yaml:"env"`              // 追加/覆盖子进程环境变量
	ProtocolVersion string            `yaml:"protocol_version"` // MCP 协议版本
	TimeoutSec      int               `yaml:"timeout_sec"`      // 连接与单次工具调用超时
}

// WeChatConfig 保存微信小程序相关配置。
// AppID / Secret 属于凭据，生产环境请用环境变量覆盖，不要写进提交到仓库的文件。
type WeChatConfig struct {
	AppID       string `yaml:"app_id"`       // 小程序 AppID
	Secret      string `yaml:"secret"`       // 小程序 AppSecret
	TokenBuffer int    `yaml:"token_buffer"` // 提前刷新秒数
	TimeoutSec  int    `yaml:"timeout_sec"`  // 单次请求超时秒数
	Enabled     bool   `yaml:"enabled"`      // 是否启用（缺凭据时自动关闭）

	Security SecurityCheckConfig `yaml:"security"` // 内容安全策略
}

// SecurityCheckConfig 保存内容安全策略。
type SecurityCheckConfig struct {
	Enabled     bool `yaml:"enabled"`      // 是否对小程序端输入做检测
	Scene       int  `yaml:"scene"`        // 1 资料 / 2 评论 / 3 论坛 / 4 社交日志
	FailOpen    bool `yaml:"fail_open"`    // 微信接口异常时是否放行
	RequireBind bool `yaml:"require_bind"` // 未绑定 openid 时是否拦截
}

// UploadsConfig 保存图片附件配置。
type UploadsConfig struct {
	Dir   string `yaml:"dir"`    // 存储目录
	MaxMB int    `yaml:"max_mb"` // 单张图片上限（MB）
}

// SessionTTL 返回会话有效期，非法配置时回退到 7 天。
func (c AuthConfig) SessionTTL() time.Duration {
	hours := c.SessionTTLHours
	if hours <= 0 {
		hours = 168
	}
	return time.Duration(hours) * time.Hour
}

// LoadConfig 按“默认配置 -> 环境配置 -> 本地密钥 -> 环境变量”的顺序加载配置。
// 入参 env 为空时读取 DEEPTALK_ENV，再回退到 dev。
// 返回的 *Config 会在校验失败时返回错误，调用方应直接终止进程。
func LoadConfig(configDir, env string) (*Config, error) {
	cfg := &Config{}
	// 1. 默认配置文件，缺失视为启动错误，避免静默使用零值配置
	if err := mergeYAML(cfg, configDir+"/app.yaml"); err != nil {
		return nil, fmt.Errorf("load app.yaml: %w", err)
	}
	// 2. 环境覆盖文件可选，例如 app-prod.yaml
	if env == "" {
		env = os.Getenv("DEEPTALK_ENV")
	}
	if env == "" {
		env = cfg.App.Env
	}
	envFile := fmt.Sprintf("%s/app-%s.yaml", configDir, env)
	if fileExists(envFile) {
		if err := mergeYAML(cfg, envFile); err != nil {
			return nil, fmt.Errorf("load %s: %w", envFile, err)
		}
	}
	// 3. 本地密钥文件（不入库、不提交），用于验收环境的上游 Key
	if fileExists(configDir + "/secret.local.yaml") {
		if err := mergeYAML(cfg, configDir+"/secret.local.yaml"); err != nil {
			return nil, fmt.Errorf("load secret.local.yaml: %w", err)
		}
	}
	cfg.App.Env = env
	// 4. 环境变量优先级最高，便于容器化部署
	applyEnvOverrides(cfg)
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// mergeYAML 将文件内容覆盖到目标结构体上。
// yaml.v3 只覆盖文件中出现的键，未出现的字段保持原值，正好满足“默认 -> 环境 -> 密钥”的分层覆盖。
func mergeYAML(dst *Config, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(raw, dst)
}

// applyEnvOverrides 读取关键环境变量，容器部署时无需修改 YAML。
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("DEEPTALK_ADDR"); v != "" {
		cfg.App.Addr = v
	}
	if v := os.Getenv("DEEPTALK_BASE_URL"); v != "" {
		cfg.App.BaseURL = v
	}
	if v := os.Getenv("DEEPTALK_LOCATION"); v != "" {
		cfg.App.Location = v
	}
	if v := os.Getenv("DEEPTALK_TIMEZONE"); v != "" {
		cfg.App.Timezone = v
	}
	if v := os.Getenv("DEEPTALK_DB"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("DEEPTALK_SIGN_MODE"); v != "" {
		cfg.Security.SignMode = v
	}
	if v := os.Getenv("DEEPTALK_SECRET"); v != "" {
		cfg.Security.Secret = v
	}
	if v := os.Getenv("DEEPTALK_DEEPSEEK_KEY"); v != "" {
		cfg.Upstream.DeepSeekAPIKey = v
	}
	if v := os.Getenv("DEEPTALK_COOKIE_SECURE"); v != "" {
		cfg.Auth.CookieSecure = v == "true" || v == "1"
	}
	if v := os.Getenv("DEEPTALK_SEED"); v != "" {
		cfg.Database.Seed = v == "true" || v == "1"
	}
	if v := os.Getenv("DEEPTALK_SESSION_TTL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Auth.SessionTTLHours = n
		}
	}
	if v := os.Getenv("DEEPTALK_SEARCH_ENABLED"); v != "" {
		cfg.WebSearch.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("DEEPTALK_OBSCURA_BIN"); v != "" {
		cfg.WebSearch.ObscuraBin = v
	}
	if v := os.Getenv("DEEPTALK_SEARCH_ENGINE"); v != "" {
		cfg.WebSearch.EngineURL = v
	}
	if v := os.Getenv("DEEPTALK_SEARCH_HTTP_FALLBACK"); v != "" {
		cfg.WebSearch.HTTPFallback = v == "true" || v == "1"
	}
	if v := os.Getenv("DEEPTALK_MAX_OUTPUT_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Upstream.MaxOutputTokens = n
		}
	}
	if v := os.Getenv("DEEPTALK_REASONING_EFFORT"); v != "" {
		cfg.Upstream.ReasoningEffort = strings.ToLower(v)
	}
	if v := os.Getenv("DEEPTALK_WECHAT_APPID"); v != "" {
		cfg.WeChat.AppID = v
	}
	if v := os.Getenv("DEEPTALK_WECHAT_SECRET"); v != "" {
		cfg.WeChat.Secret = v
	}
	if v := os.Getenv("DEEPTALK_WECHAT_SECURITY"); v != "" {
		cfg.WeChat.Security.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("DEEPTALK_WECHAT_SECURITY_REQUIRE_BIND"); v != "" {
		cfg.WeChat.Security.RequireBind = v == "true" || v == "1"
	}
	if v := os.Getenv("DEEPTALK_WECHAT_SECURITY_FAIL_CLOSED"); v != "" {
		cfg.WeChat.Security.FailOpen = !(v == "true" || v == "1")
	}
	if v := os.Getenv("DEEPTALK_WECHAT_TOKEN_BUFFER"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.WeChat.TokenBuffer = n
		}
	}
	if v := os.Getenv("DEEPTALK_UPLOAD_DIR"); v != "" {
		cfg.Uploads.Dir = v
	}
	if v := os.Getenv("DEEPTALK_UPLOAD_MAX_MB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Uploads.MaxMB = n
		}
	}
	if v := os.Getenv("DEEPTALK_SEARCH_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.WebSearch.TimeoutSec = n
		}
	}
	if v := os.Getenv("DEEPTALK_MCP_ENABLED"); v != "" {
		cfg.MCP.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("DEEPTALK_MCP_COMMAND"); v != "" {
		cfg.MCP.Command = v
	}
	if v := os.Getenv("DEEPTALK_MCP_PROTOCOL_VERSION"); v != "" {
		cfg.MCP.ProtocolVersion = v
	}
	if v := os.Getenv("DEEPTALK_MCP_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MCP.TimeoutSec = n
		}
	}
}

// validateConfig 校验启动必需项，任何一项不合法都直接失败。
func validateConfig(cfg *Config) error {
	if cfg.App.Addr == "" {
		return fmt.Errorf("app.addr is required")
	}
	if cfg.Database.DSN == "" {
		return fmt.Errorf("database.dsn is required")
	}
	if len(cfg.Security.Secret) < 16 {
		return fmt.Errorf("security.secret must be at least 16 characters")
	}
	if cfg.App.Env == "prod" {
		if cfg.Auth.CookieSecure == false {
			return fmt.Errorf("auth.cookie_secure must be true in production")
		}
		if strings.Contains(cfg.Security.Secret, "local-secret") {
			return fmt.Errorf("security.secret must be overridden in production")
		}
	}
	if cfg.Auth.DefaultQuotaTokens <= 0 {
		cfg.Auth.DefaultQuotaTokens = 4_000_000
	}
	if cfg.Upstream.MaxOutputTokens <= 0 {
		cfg.Upstream.MaxOutputTokens = 384000
	}
	if cfg.Upstream.MaxOutputTokens > 1_000_000 {
		cfg.Upstream.MaxOutputTokens = 1_000_000
	}
	if cfg.Upstream.FirstByteTimeoutSec <= 0 {
		cfg.Upstream.FirstByteTimeoutSec = 30
	}
	if cfg.Context.CompactRatio <= 0 || cfg.Context.CompactRatio >= 1 {
		cfg.Context.CompactRatio = 0.85
	}
	if cfg.Context.KeepRatio <= 0 || cfg.Context.KeepRatio >= cfg.Context.CompactRatio {
		cfg.Context.KeepRatio = 0.30
	}
	if cfg.Context.CacheIdleHours <= 0 {
		cfg.Context.CacheIdleHours = 2
	}
	if cfg.Context.IdleMinTokens <= 0 {
		cfg.Context.IdleMinTokens = 16000
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Upstream.ReasoningEffort)) {
	case "low", "medium", "high":
		cfg.Upstream.ReasoningEffort = strings.ToLower(strings.TrimSpace(cfg.Upstream.ReasoningEffort))
	default:
		// 默认中等强度：只有用户主动开启深度思考时才会发送给上游
		cfg.Upstream.ReasoningEffort = "medium"
	}
	if cfg.WebSearch.ObscuraBin == "" {
		cfg.WebSearch.ObscuraBin = "obscura"
	}
	if cfg.WebSearch.EngineURL == "" {
		cfg.WebSearch.EngineURL = "https://html.duckduckgo.com/html/?q={query}"
	}
	if cfg.WebSearch.MaxResults <= 0 {
		cfg.WebSearch.MaxResults = 5
	}
	if cfg.WebSearch.Concurrency <= 0 {
		cfg.WebSearch.Concurrency = 5
	}
	if cfg.WebSearch.Concurrency > 10 {
		cfg.WebSearch.Concurrency = 10
	}
	if cfg.WebSearch.MaxResults > 8 {
		cfg.WebSearch.MaxResults = 8
	}
	if cfg.WebSearch.TimeoutSec <= 0 {
		cfg.WebSearch.TimeoutSec = 25
	}
	if cfg.WebSearch.TimeoutSec > 45 {
		cfg.WebSearch.TimeoutSec = 45
	}
	if cfg.MCP.Command == "" {
		cfg.MCP.Command = "obscura"
	}
	if len(cfg.MCP.Args) == 0 {
		cfg.MCP.Args = []string{"mcp"}
	}
	if cfg.MCP.ProtocolVersion == "" {
		cfg.MCP.ProtocolVersion = "2024-11-05"
	}
	if cfg.MCP.TimeoutSec <= 0 {
		cfg.MCP.TimeoutSec = 30
	}
	if cfg.MCP.TimeoutSec > 120 {
		cfg.MCP.TimeoutSec = 120
	}
	if cfg.MCP.Env == nil {
		cfg.MCP.Env = make(map[string]string)
	}
	if cfg.Uploads.Dir == "" {
		cfg.Uploads.Dir = "data/uploads"
	}
	if cfg.Security.SignMode == "" {
		cfg.Security.SignMode = "log"
	}
	if strings.TrimSpace(cfg.WeChat.AppID) != "" && strings.TrimSpace(cfg.WeChat.Secret) != "" {
		cfg.WeChat.Enabled = true
	}
	if cfg.WeChat.TokenBuffer <= 0 {
		cfg.WeChat.TokenBuffer = 300
	}
	if cfg.WeChat.Security.Scene <= 0 {
		cfg.WeChat.Security.Scene = 2 // 评论场景，适合对话输入
	}
	if cfg.WeChat.TimeoutSec <= 0 {
		cfg.WeChat.TimeoutSec = 10
	}
	if cfg.Uploads.MaxMB <= 0 {
		cfg.Uploads.MaxMB = 8
	}
	if cfg.Uploads.MaxMB > 20 {
		cfg.Uploads.MaxMB = 20
	}
	return nil
}

// fileExists 判断文件是否存在且不是目录。
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
