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

// SetMany 在一个事务内批量写入设置，保证注册开关与默认路由同时生效。
func (s *Service) SetMany(ctx context.Context, values map[string]string) error {
	items := make([]Setting, 0, len(values))
	for key, value := range values {
		items = append(items, Setting{Key: key, Value: value})
	}
	return dalSaveSettings(ctx, s.db, items)
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
