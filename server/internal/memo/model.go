// Package memo 提供「小橘灵感录」：对话中随时记下灵感，需要时再取出来。
package memo

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Memo 是一条灵感录，按用户隔离。
type Memo struct {
	ID             uint      `gorm:"primaryKey" json:"id"`                      // 主键
	UserID         uint      `gorm:"index:idx_memo_user;not null" json:"-"`     // 归属用户，不外发
	Title          string    `gorm:"size:120;not null;default:''" json:"title"` // 标题（模型总结出的简短主题）
	Content        string    `gorm:"type:text;not null" json:"content"`         // 正文
	Tags           string    `gorm:"size:255;not null;default:''" json:"tags"`  // 逗号分隔的标签
	ConversationID uint      `gorm:"not null;default:0" json:"conversation_id"` // 来源会话，0 表示手动创建
	CreatedAt      time.Time `gorm:"index:idx_memo_user" json:"created_at"`     // 创建时间
	UpdatedAt      time.Time `json:"updated_at"`                                // 更新时间
}

// Migrate 建表。
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Memo{})
}

// dalCreateMemo 写入一条灵感录。
func dalCreateMemo(ctx context.Context, db *gorm.DB, item *Memo) error {
	return db.WithContext(ctx).Create(item).Error
}

// dalFindMemo 按 ID 与归属用户读取一条灵感录。
func dalFindMemo(ctx context.Context, db *gorm.DB, userID, id uint) (*Memo, error) {
	var item Memo
	if err := db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// dalSaveMemo 保存灵感录的全部字段。
func dalSaveMemo(ctx context.Context, db *gorm.DB, item *Memo) error {
	return db.WithContext(ctx).Save(item).Error
}

// dalDeleteMemo 按归属用户删除一条灵感录，返回受影响行数。
func dalDeleteMemo(ctx context.Context, db *gorm.DB, userID, id uint) (int64, error) {
	result := db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&Memo{})
	return result.RowsAffected, result.Error
}

// dalSearchMemos 按关键词检索某个用户的灵感录；keyword 为空时按更新时间倒序。
func dalSearchMemos(ctx context.Context, db *gorm.DB, userID uint, keyword string, limit int) ([]Memo, error) {
	query := db.WithContext(ctx).Where("user_id = ?", userID)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ? OR tags LIKE ?", like, like, like)
	}
	var items []Memo
	if err := query.Order("updated_at DESC").Limit(limit).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// dalAdminListMemos 分页查询全部用户的灵感录，并返回总数。
func dalAdminListMemos(ctx context.Context, db *gorm.DB, keyword string, page, size int) ([]AdminItem, int64, error) {
	base := db.WithContext(ctx).Table("memos").
		Joins("LEFT JOIN users ON users.id = memos.user_id")
	if keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where("memos.title LIKE ? OR memos.content LIKE ? OR memos.tags LIKE ? OR users.email LIKE ?",
			like, like, like, like)
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

// dalDeleteMemoByID 按 ID 删除任意用户的灵感录（后台），返回受影响行数。
func dalDeleteMemoByID(ctx context.Context, db *gorm.DB, id uint) (int64, error) {
	result := db.WithContext(ctx).Delete(&Memo{}, id)
	return result.RowsAffected, result.Error
}

// dalCountMemos 统计符合条件的灵感录数量。
func dalCountMemos(ctx context.Context, db *gorm.DB, where string, args ...any) (int64, error) {
	query := db.WithContext(ctx).Model(&Memo{})
	if where != "" {
		query = query.Where(where, args...)
	}
	var total int64
	err := query.Count(&total).Error
	return total, err
}

// dalCountMemoUsers 统计有灵感录的用户数。
func dalCountMemoUsers(ctx context.Context, db *gorm.DB) (int64, error) {
	var total int64
	err := db.WithContext(ctx).Model(&Memo{}).Distinct("user_id").Count(&total).Error
	return total, err
}

// dalRecentDays 返回最近 N 天的日期列表（从旧到新），用于补齐趋势的空白天。
func dalRecentDays(ctx context.Context, db *gorm.DB, days int) ([]string, error) {
	rows, err := db.WithContext(ctx).Raw(
		"SELECT date('now', '-' || (n - 1) || ' days') FROM (SELECT 1 n UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7) ORDER BY n DESC",
	).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0, days)
	for rows.Next() {
		var day string
		if err := rows.Scan(&day); err != nil {
			return nil, err
		}
		out = append(out, day)
	}
	return out, rows.Err()
}

// dalMemoDailyCounts 返回最近 N 天的每日数量。
func dalMemoDailyCounts(ctx context.Context, db *gorm.DB, where string) ([]DailyCount, error) {
	var rows []DailyCount
	err := db.WithContext(ctx).Model(&Memo{}).
		Select("date(created_at) as date, count(*) as count").
		Where(where).
		Group("date(created_at)").Order("date").
		Scan(&rows).Error
	return rows, err
}
