// Package billing 负责套餐限额与用量计费。
//
// 每个套餐有两类锁止条件：
//  1. 自然月内的费用上限（元）；
//  2. 滚动窗口内的请求次数上限：5 小时 / 7 天 / 30 天。
//
// 任何一条触顶都会拒绝新一轮请求，并告诉用户是哪一条、什么时候恢复。
package billing

import (
	"time"

	"gorm.io/gorm"
)

// 套餐编码：与 user.Plan 保持一致。
const (
	PlanFree     = "free"
	PlanPro      = "pro"
	PlanTeam     = "team"
	PlanUltimate = "ultimate"
)

// 滚动窗口。
const (
	WindowShort = 5 * time.Hour
	WindowWeek  = 7 * 24 * time.Hour
	WindowMonth = 30 * 24 * time.Hour
)

// Plan 是可在后台调整的套餐配置。
type Plan struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Code            string    `gorm:"size:32;uniqueIndex;not null" json:"code"`
	Name            string    `gorm:"size:64;not null" json:"name"`
	Description     string    `gorm:"size:255;not null;default:''" json:"description"`
	SpendLimitCents int64     `gorm:"not null;default:0" json:"spend_limit_cents"`
	LimitShort      int       `gorm:"not null;default:0" json:"limit_short"`
	LimitWeek       int       `gorm:"not null;default:0" json:"limit_week"`
	LimitMonth      int       `gorm:"not null;default:0" json:"limit_month"`
	Enabled         bool      `gorm:"not null;default:true;index" json:"enabled"`
	SortOrder       int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Event 是一次成功生成的计费事件。
type Event struct {
	ID           uint   `gorm:"primaryKey"`
	UserID       uint   `gorm:"index:idx_billing_user_time;not null"`
	ModelID      uint   `gorm:"not null;default:0"`
	ModelName    string `gorm:"size:64;not null;default:''"`
	InputTokens  int    `gorm:"not null;default:0"`
	OutputTokens int    `gorm:"not null;default:0"`
	CachedTokens int    `gorm:"column:cached_tokens;not null;default:0"` // 命中缓存的输入 tokens
	// 本次费用，单位「分」的千分之一（避免精度丢失），展示时 /1000
	CostMilliCents int64     `gorm:"not null;default:0"`
	CreatedAt      time.Time `gorm:"index:idx_billing_user_time"`
}

// Migrate 建表并写入默认套餐。
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Plan{}, &Event{}); err != nil {
		return err
	}
	defaults := []Plan{
		{Code: PlanFree, Name: "免费版", Description: "体验对话与联网搜索", SpendLimitCents: 100, LimitShort: 50, LimitWeek: 100, LimitMonth: 200, Enabled: true, SortOrder: 10},
		{Code: PlanPro, Name: "专业版", Description: "更高频率与更长对话", SpendLimitCents: 1000, LimitShort: 150, LimitWeek: 400, LimitMonth: 1000, Enabled: true, SortOrder: 20},
		{Code: PlanUltimate, Name: "旗舰版", Description: "重度使用、几乎无感限流", SpendLimitCents: 5000, LimitShort: 400, LimitWeek: 2100, LimitMonth: 5000, Enabled: true, SortOrder: 30},
	}
	for _, plan := range defaults {
		var count int64
		if err := db.Model(&Plan{}).Where("code = ?", plan.Code).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := db.Create(&plan).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
