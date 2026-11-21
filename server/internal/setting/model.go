// Package setting 维护运行期可调整的平台设置（注册开关、默认模型路由等）。
// 设置以 key/value 形式存储，新增配置项无需改表结构。
package setting

import (
	"context"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// 平台设置键名。集中定义避免各处硬编码字符串。
const (
	KeyAllowRegistration = "allow_registration"     // 是否开放 C 端注册
	KeyDefaultModelID    = "default_model_id"       // 平台默认模型主键
	KeyFallbackModelID   = "fallback_model_id"      // 兜底模型主键
	KeyMaxOutputTokens   = "max_output_tokens"      // 单次回答输出预算（thinking + 正文）
	KeySiteName          = "site_name"              // 站点名称（后台可改）
	KeyGenerationTimeout = "generation_timeout_sec" // 生成首字节超时（秒），后台可调
)

// Setting 是 key/value 形式的平台设置。
type Setting struct {
	Key       string    `gorm:"primaryKey;size:64"` // 设置键
	Value     string    `gorm:"size:255;not null"`  // 设置值（统一存字符串）
	UpdatedAt time.Time // 更新时间
}

// Service 提供设置的读写能力。
type Service struct {
	db *gorm.DB // 数据库句柄
}

// NewService 创建设置服务。
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Get 读取字符串设置，不存在或查询失败时返回默认值。
func (s *Service) Get(ctx context.Context, key, def string) string {
	var item Setting
	err := s.db.WithContext(ctx).First(&item, "key = ?", key).Error
	if err != nil {
		return def
	}
	return item.Value
}

// Bool 读取布尔设置，仅 "true"/"1" 视为真。
func (s *Service) Bool(ctx context.Context, key string, def bool) bool {
	raw := s.Get(ctx, key, "")
	if raw == "" {
		return def
	}
	return raw == "true" || raw == "1"
}

// Int 读取整型设置，解析失败返回默认值。
func (s *Service) Int(ctx context.Context, key string, def int) int {
	raw := s.Get(ctx, key, "")
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}

// Set 写入或更新单个设置。
func (s *Service) Set(ctx context.Context, key, value string) error {
	item := Setting{Key: key, Value: value}
	// SQLite 的 upsert 语法由 GORM 统一生成
	return s.db.WithContext(ctx).Save(&item).Error
}

// SetMany 在一个事务内批量写入设置，保证注册开关与默认路由同时生效。
func (s *Service) SetMany(ctx context.Context, values map[string]string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, value := range values {
			item := Setting{Key: key, Value: value}
			if err := tx.Save(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// All 返回全部设置，用于后台系统设置页展示。
func (s *Service) All(ctx context.Context) map[string]string {
	var items []Setting
	_ = s.db.WithContext(ctx).Find(&items).Error
	out := make(map[string]string, len(items))
	for _, item := range items {
		out[item.Key] = item.Value
	}
	return out
}

// AllowRegistration 实现 user 包消费的注册策略接口。
func (s *Service) AllowRegistration(ctx context.Context) (bool, error) {
	return s.Bool(ctx, KeyAllowRegistration, true), nil
}

// Migrate 创建设置表。
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Setting{})
}
