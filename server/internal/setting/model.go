// Package setting 维护运行期可调整的平台设置（注册开关、默认模型路由等）。
// 设置以 key/value 形式存储，新增配置项无需改表结构。
package setting

import (
	"context"
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
	KeySearchProvider    = "search_provider"        // 联网检索提供方：self / deepseek
	KeySearchDeepSeekKey = "search_deepseek_key"    // DeepSeek 检索密钥（AES-GCM 密文）
)

// Setting 是 key/value 形式的平台设置。
type Setting struct {
	Key       string    `gorm:"primaryKey;size:64"` // 设置键
	Value     string    `gorm:"size:255;not null"`  // 设置值（统一存字符串）
	UpdatedAt time.Time // 更新时间
}

// Migrate 创建设置表。
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Setting{})
}

// dalFindSetting 按 key 读取一条设置。
func dalFindSetting(ctx context.Context, db *gorm.DB, key string) (*Setting, error) {
	var item Setting
	if err := db.WithContext(ctx).First(&item, "key = ?", key).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// dalSaveSetting 写入或更新单条设置。
func dalSaveSetting(ctx context.Context, db *gorm.DB, item *Setting) error {
	return db.WithContext(ctx).Save(item).Error
}

// dalSaveSettings 在事务内批量写入设置。
func dalSaveSettings(ctx context.Context, db *gorm.DB, items []Setting) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for index := range items {
			if err := tx.Save(&items[index]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// dalAllSettings 读取全部设置。
func dalAllSettings(ctx context.Context, db *gorm.DB) ([]Setting, error) {
	var items []Setting
	err := db.WithContext(ctx).Find(&items).Error
	return items, err
}
