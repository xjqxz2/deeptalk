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
	return dalAdminListMemos(ctx, s.db, strings.TrimSpace(query), page, size)
}

// AdminDelete 按 ID 删除任意用户的灵感（后台操作）。
func (s *Service) AdminDelete(ctx context.Context, id uint) error {
	affected, err := dalDeleteMemoByID(ctx, s.db, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// StatsOf 统计总量、今日、近 7 天与每日趋势。
func (s *Service) StatsOf(ctx context.Context) (*Stats, error) {
	stats := &Stats{}
	total, err := dalCountMemos(ctx, s.db, "")
	if err != nil {
		return nil, err
	}
	stats.Total = total
	today, err := dalCountMemos(ctx, s.db, "created_at >= date('now', 'start of day')")
	if err != nil {
		return nil, err
	}
	stats.Today = today
	last7, err := dalCountMemos(ctx, s.db, "created_at >= date('now', '-6 days', 'start of day')")
	if err != nil {
		return nil, err
	}
	stats.Last7 = last7
	users, err := dalCountMemoUsers(ctx, s.db)
	if err != nil {
		return nil, err
	}
	stats.Users = users
	days, err := dalMemoDailyCounts(ctx, s.db, "created_at >= date('now', '-6 days', 'start of day')")
	if err != nil {
		return nil, err
	}
	stats.Daily, err = fillDays(ctx, s.db, days, 7)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// fillDays 把稀疏的按天计数补齐成连续 7 天（缺失记 0），供前端画趋势。
func fillDays(ctx context.Context, db *gorm.DB, rows []DailyCount, days int) ([]DailyCount, error) {
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.Date] = row.Count
	}
	dates, err := dalRecentDays(ctx, db, days)
	if err != nil {
		return nil, err
	}
	out := make([]DailyCount, 0, len(dates))
	for _, day := range dates {
		out = append(out, DailyCount{Date: day, Count: counts[day]})
	}
	return out, nil
}
