package setting

import (
	"context"
	"strconv"

	"gorm.io/gorm"
)

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
	item, err := dalFindSetting(ctx, s.db, key)
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
	return dalSaveSetting(ctx, s.db, &item)
}

// SetRouting 在一个事务内写入默认模型与兜底模型，保证两者同时生效。
// 用 0 表示清空路由。
func (s *Service) SetRouting(ctx context.Context, defaultModelID, fallbackModelID uint) error {
	return dalSaveSettings(ctx, s.db, []Setting{
		{Key: KeyDefaultModelID, Value: strconv.FormatUint(uint64(defaultModelID), 10)},
		{Key: KeyFallbackModelID, Value: strconv.FormatUint(uint64(fallbackModelID), 10)},
	})
}

// All 返回全部设置，用于后台系统设置页展示。
func (s *Service) All(ctx context.Context) map[string]string {
	items, err := dalAllSettings(ctx, s.db)
	if err != nil {
		return map[string]string{}
	}
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
