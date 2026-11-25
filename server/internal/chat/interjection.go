package chat

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// 插话相关的业务错误。
var (
	ErrNotGenerating  = errors.New("当前没有正在生成的回答")
	ErrEmptyInterject = errors.New("插话内容不能为空")
	ErrBusy           = errors.New("conversation is generating")
)

// 生成中标记：按会话记录当前是否有流式回答在跑。
// 只需要进程内可见：插话必须落在同一轮生成期间才有意义。
type busySet struct {
	mu   sync.Mutex
	refs map[uint]int
}

// tryAdd 尝试占用会话；已经在生成时返回 false，用于拒绝并发重复生成。
func (b *busySet) tryAdd(conversationID uint) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.refs == nil {
		b.refs = make(map[uint]int)
	}
	if b.refs[conversationID] > 0 {
		return false
	}
	b.refs[conversationID] = 1
	return true
}

func (b *busySet) add(conversationID uint) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.refs == nil {
		b.refs = make(map[uint]int)
	}
	b.refs[conversationID]++
}

func (b *busySet) done(conversationID uint) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.refs == nil {
		return
	}
	if b.refs[conversationID] <= 1 {
		delete(b.refs, conversationID)
		return
	}
	b.refs[conversationID]--
}

// busy 判断会话是否正在生成。
func (b *busySet) busy(conversationID uint) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.refs[conversationID] > 0
}

// QueueInterjection 记录一条插话，等当前轮结束后由控制器取走继续生成。
func (s *Service) QueueInterjection(ctx context.Context, userID, conversationID uint, content string) (*Interjection, error) {
	conv, err := s.findConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrEmptyInterject
	}
	if len([]rune(content)) > 2000 {
		return nil, errors.New("插话内容过长")
	}
	if !s.busy.busy(conv.ID) {
		return nil, ErrNotGenerating
	}
	item := &Interjection{
		ConversationID: conv.ID,
		UserID:         userID,
		Content:        content,
	}
	if err := s.db.WithContext(ctx).Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

// PopInterjection 取出一条尚未处理的插话并标记已消费。
func (s *Service) PopInterjection(ctx context.Context, userID, conversationID uint) (*Interjection, bool, error) {
	var item Interjection
	err := s.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ? AND consumed_at IS NULL", conversationID, userID).
		Order("id asc").First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&Interjection{}).Where("id = ?", item.ID).
		Update("consumed_at", now).Error; err != nil {
		return nil, false, err
	}
	item.ConsumedAt = &now
	return &item, true, nil
}

// Busy 返回会话是否正在生成。
func (s *Service) Busy(conversationID uint) bool { return s.busy.busy(conversationID) }
