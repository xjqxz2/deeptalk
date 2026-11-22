// Package provider 管理上游模型厂商与模型路由。
// 上游 Key 只以 AES-GCM 密文入库，接口只返回掩码；所有厂商调用都由服务端发起。
package provider

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// 厂商类型常量，用于前端展示与图标区分。
const (
	KindDeepSeek = "deepseek" // 深度求索
	KindOpenAI   = "openai"   // OpenAI
	KindAliyun   = "aliyun"   // 阿里云百炼
	KindMoonshot = "moonshot" // Moonshot
	KindCustom   = "custom"   // 自定义 OpenAI 兼容接口

	// 接入方式（三选一）
	APIStyleChat      = "chat"      // OpenAI Chat Completions
	APIStyleResponses = "responses" // OpenAI Responses API
	APIStyleAnthropic = "anthropic" // Anthropic Messages API
)

// 卡片状态常量，对应设计图“已启用 / 待验证 / 连接失败 / 已停用”。
const (
	StatusActive   = "active"   // 已启用且验证通过
	StatusPending  = "pending"  // 已启用但未验证
	StatusFailed   = "failed"   // 验证失败
	StatusDisabled = "disabled" // 已停用
)

// 业务错误。
var (
	ErrNotFound       = errors.New("provider not found")
	ErrBuiltinDelete  = errors.New("builtin provider cannot be deleted")
	ErrInvalidBaseURL = errors.New("invalid base url")
	ErrNoAPIKey       = errors.New("api key is not configured")
	ErrModelNotFound  = errors.New("model not found")
)

// Provider 是上游厂商配置。
type Provider struct {
	ID             uint       `gorm:"primaryKey"`                                       // 主键
	Name           string     `gorm:"size:64;not null"`                                 // 显示名称
	Code           string     `gorm:"size:64;uniqueIndex;not null"`                     // 唯一标识
	Kind           string     `gorm:"size:16;not null;default:'custom'"`                // 厂商类型
	APIStyle       string     `gorm:"column:api_style;size:16;not null;default:'chat'"` // 接入方式：chat / responses / anthropic
	BaseURL        string     `gorm:"size:255;not null"`                                // 上游 Base URL
	APIKeyEnc      string     `gorm:"size:512;not null;default:''"`                     // AES-GCM 密文
	APIKeyMasked   string     `gorm:"size:64;not null;default:''"`                      // 掩码，前端只读
	HeadersJSON    string     `gorm:"type:text;not null;default:''"`                    // 自定义请求头 JSON，空表示不附加
	TimeoutSec     int        `gorm:"not null;default:60"`                              // 调用超时（秒）
	MaxConcurrency int        `gorm:"not null;default:4"`                               // 最大并发
	Enabled        bool       `gorm:"not null;default:false;index"`                     // 是否启用
	Verified       bool       `gorm:"not null;default:false"`                           // 连接测试是否通过
	IsBuiltin      bool       `gorm:"not null;default:false"`                           // 是否预置厂商
	IsDefault      bool       `gorm:"not null;default:false"`                           // 是否平台默认厂商
	SortOrder      int        `gorm:"not null;default:0"`                               // 展示排序
	LastTestedAt   *time.Time `gorm:"index"`                                            // 最近测试时间
	LastLatencyMS  int        `gorm:"not null;default:0"`                               // 最近测试延迟
	LastError      string     `gorm:"size:255;not null;default:''"`                     // 最近测试失败原因
	CreatedAt      time.Time  // 创建时间
	UpdatedAt      time.Time  // 更新时间
	Models         []Model    `gorm:"foreignKey:ProviderID"` // 该厂商的模型
}

// Model 是可供用户选择的模型及其路由信息。
type Model struct {
	ID               uint      `gorm:"primaryKey"`                                      // 主键
	ProviderID       uint      `gorm:"uniqueIndex:uk_provider_model;not null"`          // 所属厂商
	ModelID          string    `gorm:"size:128;uniqueIndex:uk_provider_model;not null"` // 上游模型 ID
	DisplayName      string    `gorm:"size:64;not null"`                                // 展示名
	Description      string    `gorm:"size:128;not null;default:''"`                    // 用途说明
	ContextWindow    int       `gorm:"not null;default:32768"`                          // 上下文窗口（tokens）
	MaxConcurrency   int       `gorm:"not null;default:2"`                              // 模型级并发
	Enabled          bool      `gorm:"not null;default:false;index"`                    // 是否启用
	IsDefault        bool      `gorm:"not null;default:false"`                          // 平台默认模型
	IsFallback       bool      `gorm:"not null;default:false"`                          // 兜底模型
	SortOrder        int       `gorm:"not null;default:0"`                              // 排序
	PriceInput       float64   `gorm:"not null;default:0"`                              // 输入单价（元 / 百万 token）
	PriceCachedInput float64   `gorm:"column:price_cached_input;not null;default:0"`    // 缓存命中输入单价（元 / 百万 token）
	PriceOutput      float64   `gorm:"not null;default:0"`                              // 输出单价（元 / 百万 token）
	CreatedAt        time.Time // 创建时间
	UpdatedAt        time.Time // 更新时间

	// ProviderName 是联表查询后的展示字段，不落库。
	ProviderName string `gorm:"->"`
	// ProviderKind 用于前端图标区分，不落库。
	ProviderKind string `gorm:"->"`
}

// Counts 是厂商列表的筛选计数。
type Counts struct {
	All     int64 `json:"all"`     // 全部厂商
	Builtin int64 `json:"builtin"` // 预置厂商
	Custom  int64 `json:"custom"`  // 自定义厂商
	Enabled int64 `json:"enabled"` // 已启用厂商
}

// TestResult 是连接测试结果。
type TestResult struct {
	OK          bool     `json:"ok"`           // 是否成功
	LatencyMS   int      `json:"latency_ms"`   // 耗时（毫秒）
	ModelsFound int      `json:"models_found"` // 发现模型数
	Models      []string `json:"models"`       // 模型 ID 列表
	Message     string   `json:"message"`      // 可读结果说明
}

// HealthItem 是单个上游的健康状态。
type HealthItem struct {
	ProviderID uint   `json:"provider_id"` // 厂商 ID
	Name       string `json:"name"`        // 厂商名称
	OK         bool   `json:"ok"`          // 是否可用
	LatencyMS  int    `json:"latency_ms"`  // 最近延迟
}

// Health 是接入健康面板数据。
type Health struct {
	UpstreamsTotal     int          `json:"upstreams_total"`     // 启用的上游总数
	UpstreamsAvailable int          `json:"upstreams_available"` // 可用上游数
	Errors1H           int64        `json:"errors_1h"`           // 近 1 小时测试失败数
	FirstByteP95MS     int          `json:"first_byte_p95_ms"`   // 首字节 P95（毫秒）
	UpdatedAt          time.Time    `json:"updated_at"`          // 采集时间
	Items              []HealthItem `json:"items"`               // 各上游明细
}

// ModelInput 是保存厂商时提交的模型项。
type ModelInput struct {
	ModelID        string `json:"model_id"`        // 上游模型 ID
	DisplayName    string `json:"display_name"`    // 展示名
	Description    string `json:"description"`     // 用途说明
	ContextWindow  int    `json:"context_window"`  // 上下文窗口
	MaxConcurrency int    `json:"max_concurrency"` // 模型级并发
	Enabled        bool   `json:"enabled"`         // 是否启用
	// 价格（元 / 百万 token），由我们自己的计费层使用；上游不给价时也能算钱
	PriceInput       float64 `json:"price_input"`        // 普通输入
	PriceCachedInput float64 `json:"price_cached_input"` // 缓存命中输入
	PriceOutput      float64 `json:"price_output"`       // 输出
}

// SaveInput 是新建 / 编辑厂商的输入。
type SaveInput struct {
	Name           string            `json:"name"`            // 显示名称
	Kind           string            `json:"kind"`            // 厂商类型
	APIStyle       string            `json:"api_style"`       // 接入方式：chat / responses / anthropic
	BaseURL        string            `json:"base_url"`        // Base URL
	APIKey         string            `json:"api_key"`         // 明文 Key；编辑时为空表示保留
	Headers        map[string]string `json:"headers"`         // 自定义请求头；nil 表示保持不变
	TimeoutSec     int               `json:"timeout_sec"`     // 超时秒数
	MaxConcurrency int               `json:"max_concurrency"` // 最大并发
	Enabled        bool              `json:"enabled"`         // 是否启用
	IsDefault      bool              `json:"is_default"`      // 是否平台默认
	Models         *[]ModelInput     `json:"models"`          // 全量模型列表；nil 表示不变
}

// Routing 是默认与兜底模型路由。
type Routing struct {
	DefaultModelID  uint `json:"default_model_id"`  // 默认模型主键
	FallbackModelID uint `json:"fallback_model_id"` // 兜底模型主键
}

// dalListProviders 查询厂商列表并预加载模型。
func dalListProviders(ctx context.Context, db *gorm.DB, builtin *bool) ([]Provider, error) {
	query := db.WithContext(ctx).Preload("Models", func(tx *gorm.DB) *gorm.DB {
		return tx.Order("sort_order ASC, id ASC")
	})
	if builtin != nil {
		query = query.Where("is_builtin = ?", *builtin)
	}
	var items []Provider
	err := query.Order("sort_order ASC, id ASC").Find(&items).Error
	return items, err
}

// dalFindProvider 按主键查询厂商。
func dalFindProvider(ctx context.Context, db *gorm.DB, id uint) (*Provider, error) {
	var p Provider
	if err := db.WithContext(ctx).Preload("Models", func(tx *gorm.DB) *gorm.DB {
		return tx.Order("sort_order ASC, id ASC")
	}).First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// dalCounts 统计厂商筛选数量。
func dalCounts(ctx context.Context, db *gorm.DB) (Counts, error) {
	var counts Counts
	base := db.WithContext(ctx).Model(&Provider{})
	if err := base.Count(&counts.All).Error; err != nil {
		return counts, err
	}
	if err := db.WithContext(ctx).Model(&Provider{}).Where("is_builtin = ?", true).Count(&counts.Builtin).Error; err != nil {
		return counts, err
	}
	if err := db.WithContext(ctx).Model(&Provider{}).Where("is_builtin = ?", false).Count(&counts.Custom).Error; err != nil {
		return counts, err
	}
	if err := db.WithContext(ctx).Model(&Provider{}).Where("enabled = ?", true).Count(&counts.Enabled).Error; err != nil {
		return counts, err
	}
	return counts, nil
}

// Migrate 创建厂商与模型表。
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Provider{}, &Model{})
}
