package memo

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// AdminItem 是后台列表里的一条灵感，附带记录人邮箱。
type AdminItem struct {
	Memo
	UserEmail string `json:"user_email"`
}

// DailyCount 是某一天的灵感数量。
type DailyCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// Stats 是小橘灵感录的后台统计。
type Stats struct {
	Total   int64        `json:"total"`    // 全部灵感数
	Today   int64        `json:"today"`    // 今日新增
	Last7   int64        `json:"last7"`    // 近 7 天新增
	Users   int64        `json:"users"`    // 有记录的用户数
	Daily   []DailyCount `json:"daily"`    // 近 7 天每日新增
	TopTags []DailyCount `json:"top_tags"` // 预留：热门标签
}

// AdminList 分页列出所有用户的灵感；query 非空时按标题/正文/邮箱检索。
func (s *Service) AdminList(ctx context.Context, query string, page, size int) ([]AdminItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	base := s.db.WithContext(ctx).Table("memos").
		Joins("LEFT JOIN users ON users.id = memos.user_id")
	if text := strings.TrimSpace(query); text != "" {
		like := "%" + text + "%"
		base = base.Where("memos.title LIKE ? OR memos.content LIKE ? OR memos.tags LIKE ? OR users.email LIKE ?", like, like, like, like)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []AdminItem
	err := base.Select("memos.*, COALESCE(users.email, '') AS user_email").
		Order("memos.created_at DESC").
		Offset((page - 1) * size).Limit(size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// AdminDelete 按 ID 删除任意用户的灵感（后台操作）。
func (s *Service) AdminDelete(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&Memo{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// StatsOf 统计总量、今日、近 7 天与每日趋势。
func (s *Service) StatsOf(ctx context.Context) (*Stats, error) {
	db := s.db.WithContext(ctx)
	stats := &Stats{}
	if err := db.Model(&Memo{}).Count(&stats.Total).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&Memo{}).Where("created_at >= date('now', 'start of day')").Count(&stats.Today).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&Memo{}).Where("created_at >= date('now', '-6 days', 'start of day')").Count(&stats.Last7).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&Memo{}).Distinct("user_id").Count(&stats.Users).Error; err != nil {
		return nil, err
	}
	rows, err := db.Model(&Memo{}).
		Select("date(created_at) as date, count(*) as count").
		Where("created_at >= date('now', '-6 days', 'start of day')").
		Group("date(created_at)").Order("date").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int64, 7)
	for rows.Next() {
		var day string
		var count int64
		if err := rows.Scan(&day, &count); err != nil {
			return nil, err
		}
		counts[day] = count
	}
	stats.Daily = fillDays(db, counts, 7)
	return stats, nil
}

// fillDays 把稀疏的按天计数补齐成连续 7 天（缺失记 0），供前端画趋势。
func fillDays(db *gorm.DB, counts map[string]int64, days int) []DailyCount {
	out := make([]DailyCount, 0, days)
	rows, err := db.Raw(
		"SELECT date('now', '-' || (n - 1) || ' days') FROM (SELECT 1 n UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7) ORDER BY n DESC",
	).Rows()
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var day string
		if err := rows.Scan(&day); err != nil {
			return nil
		}
		out = append(out, DailyCount{Date: day, Count: counts[day]})
	}
	return out
}
