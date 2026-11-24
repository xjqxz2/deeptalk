package memo

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

// ErrNotFound 表示灵感录不存在（或不属于当前用户）。
var ErrNotFound = errors.New("memo not found")

// Service 提供灵感录读写。
type Service struct {
	db *gorm.DB
}

// NewService 创建灵感录服务。
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// SaveInput 是新建 / 更新灵感录的输入。
type SaveInput struct {
	Title          string   `json:"title"`
	Content        string   `json:"content"`
	Tags           []string `json:"tags"`
	ConversationID uint     `json:"conversation_id"`
}

// Save 新建一条灵感录。
func (s *Service) Save(ctx context.Context, userID uint, in SaveInput) (*Memo, error) {
	content := strings.TrimSpace(in.Content)
	if content == "" {
		return nil, errors.New("memo content is required")
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = firstLine(content)
	}
	if len([]rune(title)) > 60 {
		title = string([]rune(title)[:60])
	}
	item := &Memo{
		UserID:         userID,
		Title:          title,
		Content:        content,
		Tags:           strings.Join(cleanTags(in.Tags), ","),
		ConversationID: in.ConversationID,
	}
	if err := s.db.WithContext(ctx).Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

// Update 修改标题 / 正文 / 标签。
func (s *Service) Update(ctx context.Context, userID, id uint, in SaveInput) (*Memo, error) {
	var item Memo
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if title := strings.TrimSpace(in.Title); title != "" {
		item.Title = title
	}
	if content := strings.TrimSpace(in.Content); content != "" {
		item.Content = content
	}
	if in.Tags != nil {
		item.Tags = strings.Join(cleanTags(in.Tags), ",")
	}
	if err := s.db.WithContext(ctx).Save(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// Delete 删除一条灵感录。
func (s *Service) Delete(ctx context.Context, userID, id uint) error {
	result := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&Memo{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Get 读取一条灵感录。
func (s *Service) Get(ctx context.Context, userID, id uint) (*Memo, error) {
	var item Memo
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

// Search 按关键词检索标题、正文与标签；query 为空时按时间倒序返回。
func (s *Service) Search(ctx context.Context, userID uint, query string, limit int) ([]Memo, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	db := s.db.WithContext(ctx).Where("user_id = ?", userID)
	if text := strings.TrimSpace(query); text != "" {
		like := "%" + text + "%"
		db = db.Where("title LIKE ? OR content LIKE ? OR tags LIKE ?", like, like, like)
	}
	var items []Memo
	if err := db.Order("updated_at DESC").Limit(limit).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func firstLine(text string) string {
	if index := strings.IndexByte(text, '\n'); index >= 0 {
		text = text[:index]
	}
	return strings.TrimSpace(text)
}

func cleanTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		value := strings.TrimSpace(tag)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
