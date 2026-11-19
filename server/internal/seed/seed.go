// Package seed 只负责首次启动时的最小可用数据：
// 一个平台管理员、DeepSeek 一家预置厂商、deepseek-flash 一个模型，以及默认路由。
// 不再写入演示用户、演示会话或审计记录；对应表为空时才会写入，重复启动不会产生重复记录。
package seed

import (
	"context"
	"fmt"
	"time"

	"deeptalk/internal/provider"
	"deeptalk/internal/setting"
	"deeptalk/internal/user"

	"gorm.io/gorm"
)

// Config 是种子数据的输入参数。
type Config struct {
	Secret         string // 用于加密预置厂商的 Key
	DeepSeekAPIKey string // 真实上游 Key，可为空；为空时对话走本地演示回答
}

// 平台管理员初始账号。密码仅用于首次登录，登录后应尽快在“账号与密码”中修改。
const (
	adminEmail    = "xjqxz_2@163.com"
	adminNickname = "Wind"
	adminPassword = "123456"
)

// 预置模型标识。
const deepSeekModelID = "deepseek-flash"

// Run 幂等写入最小种子数据。
func Run(ctx context.Context, db *gorm.DB, cfg Config) error {
	var userCount int64
	if err := db.WithContext(ctx).Model(&user.User{}).Count(&userCount).Error; err != nil {
		return err
	}
	if userCount == 0 {
		if err := seedAdmin(ctx, db); err != nil {
			return fmt.Errorf("seed admin: %w", err)
		}
	}
	var providerCount int64
	if err := db.WithContext(ctx).Model(&provider.Provider{}).Count(&providerCount).Error; err != nil {
		return err
	}
	if providerCount == 0 {
		if err := seedProviders(ctx, db, cfg); err != nil {
			return fmt.Errorf("seed providers: %w", err)
		}
	}
	var settingCount int64
	if err := db.WithContext(ctx).Model(&setting.Setting{}).Count(&settingCount).Error; err != nil {
		return err
	}
	if settingCount == 0 {
		if err := seedSettings(ctx, db); err != nil {
			return fmt.Errorf("seed settings: %w", err)
		}
	}
	return nil
}

// seedAdmin 写入唯一的管理员账号。
func seedAdmin(ctx context.Context, db *gorm.DB) error {
	hash, err := user.HashPassword(adminPassword)
	if err != nil {
		return err
	}
	admin := user.User{
		Email:        adminEmail,
		Nickname:     adminNickname,
		PasswordHash: hash,
		Role:         user.RoleAdmin,
		Plan:         user.PlanFree,
		QuotaTokens:  4_000_000,
		UsedTokens:   0,
		CreatedAt:    time.Now(),
	}
	return db.WithContext(ctx).Create(&admin).Error
}

// seedProviders 写入 DeepSeek 预置厂商与 deepseek-flash 模型。
// Key 使用与服务端相同的 AES-GCM 密钥加密；没有 Key 时仍保留厂商，便于后续在后台补录。
func seedProviders(ctx context.Context, db *gorm.DB, cfg Config) error {
	masked := provider.MaskAPIKey(cfg.DeepSeekAPIKey)
	var encrypted string
	if cfg.DeepSeekAPIKey != "" {
		value, err := provider.EncryptAPIKey(cfg.Secret, cfg.DeepSeekAPIKey)
		if err != nil {
			return err
		}
		encrypted = value
	}
	now := time.Now()
	deepSeek := provider.Provider{
		Name:           "DeepSeek",
		Code:           "deepseek",
		Kind:           provider.KindDeepSeek,
		BaseURL:        "https://api.deepseek.com",
		APIKeyEnc:      encrypted,
		APIKeyMasked:   masked,
		TimeoutSec:     60,
		MaxConcurrency: 8,
		Enabled:        true,
		Verified:       cfg.DeepSeekAPIKey != "",
		IsBuiltin:      true,
		IsDefault:      true,
		SortOrder:      1,
	}
	if cfg.DeepSeekAPIKey != "" {
		deepSeek.LastTestedAt = &now
		deepSeek.LastLatencyMS = 320
	}
	if err := db.WithContext(ctx).Create(&deepSeek).Error; err != nil {
		return err
	}
	model := provider.Model{
		ProviderID:     deepSeek.ID,
		ModelID:        deepSeekModelID,
		DisplayName:    "DeepSeek-V4.1-Flash",
		Description:    "DeepSeek V4.1 Flash · 通用对话",
		ContextWindow:  1_048_576,
		MaxConcurrency: 8,
		Enabled:        true,
		IsDefault:      true,
		SortOrder:      1,
	}
	return db.WithContext(ctx).Create(&model).Error
}

// seedSettings 写入注册开关与默认路由；只有一家厂商时不设置兜底模型。
func seedSettings(ctx context.Context, db *gorm.DB) error {
	var defaultModel provider.Model
	if err := db.WithContext(ctx).Where("model_id = ?", deepSeekModelID).First(&defaultModel).Error; err != nil {
		return err
	}
	rows := []setting.Setting{
		{Key: setting.KeyAllowRegistration, Value: "true"},
		{Key: setting.KeyDefaultModelID, Value: fmt.Sprintf("%d", defaultModel.ID)},
		{Key: setting.KeyFallbackModelID, Value: "0"},
	}
	return db.WithContext(ctx).Create(&rows).Error
}
