package chat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ErrShareNotFound 表示分享不存在或已失效。
var ErrShareNotFound = errors.New("share not found")

// Share 是一条对话分享；同一会话只保留一个有效分享。
// 分享本身是只读快照：访客只能读取消息，不能继续对话。
type Share struct {
	ID             uint       `gorm:"primaryKey"`
	ConversationID uint       `gorm:"uniqueIndex;not null"` // 一个会话一个分享
	UserID         uint       `gorm:"index;not null"`       // 分享创建者，便于审计
	Token          string     `gorm:"size:64;uniqueIndex;not null"`
	CreatedAt      time.Time  // 创建时间
	RevokedAt      *time.Time // 撤销时间；非空表示已失效
}

// TableName 固定表名，避免依赖结构体复数的约定。
func (Share) TableName() string { return "conversation_shares" }

// newShareToken 生成不可枚举的分享 token。
func newShareToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}

// MigrateShares 创建分享表；随 chat 模块的迁移一起执行。
func MigrateShares(db *gorm.DB) error {
	return db.AutoMigrate(&Share{})
}

// CreateShare 创建或复用当前会话的分享链接。
// 已撤销的分享会重新生成 token，避免旧链接被猜中后复用。
func (s *Service) CreateShare(ctx context.Context, userID, conversationID uint) (*Share, error) {
	if _, err := s.findConversation(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	var share Share
	err := s.db.WithContext(ctx).Where("conversation_id = ?", conversationID).First(&share).Error
	switch {
	case err == nil:
		if share.RevokedAt == nil && share.UserID == userID {
			return &share, nil
		}
		token := newShareToken()
		if err := dalRotateShare(ctx, s.db, share.ID, ShareUpdate{
			Token:        &token,
			UserID:       &userID,
			ClearRevoked: true,
		}); err != nil {
			return nil, err
		}
		share.Token = token
		share.RevokedAt = nil
		share.UserID = userID
		return &share, nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		share = Share{
			ConversationID: conversationID,
			UserID:         userID,
			Token:          newShareToken(),
		}
		if err := s.db.WithContext(ctx).Create(&share).Error; err != nil {
			return nil, err
		}
		return &share, nil
	default:
		return nil, err
	}
}

// CurrentShare 返回会话当前有效的分享；没有则返回 ErrShareNotFound。
func (s *Service) CurrentShare(ctx context.Context, userID, conversationID uint) (*Share, error) {
	if _, err := s.findConversation(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	var share Share
	if err := s.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ? AND revoked_at IS NULL", conversationID, userID).
		First(&share).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShareNotFound
		}
		return nil, err
	}
	return &share, nil
}

// RevokeShare 撤销会话分享；不存在时视为已撤销。
func (s *Service) RevokeShare(ctx context.Context, userID, conversationID uint) error {
	if _, err := s.findConversation(ctx, userID, conversationID); err != nil {
		return err
	}
	now := time.Now()
	return s.db.WithContext(ctx).Model(&Share{}).
		Where("conversation_id = ? AND user_id = ? AND revoked_at IS NULL", conversationID, userID).
		Update("revoked_at", &now).Error
}

// SharedConversation 是公开分享页需要的只读数据。
type SharedConversation struct {
	Conversation Conversation // 会话元信息（仅暴露标题与时间）
	Messages     []Message    // 全部消息
	Owner        string       // 分享者展示名
	CreatedAt    time.Time    // 分享创建时间
}

// GetShared 通过 token 读取只读对话，不需要登录。
func (s *Service) GetShared(ctx context.Context, token string) (*SharedConversation, error) {
	token = trimToken(token)
	if token == "" {
		return nil, ErrShareNotFound
	}
	var share Share
	if err := s.db.WithContext(ctx).
		Where("token = ? AND revoked_at IS NULL", token).First(&share).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShareNotFound
		}
		return nil, err
	}
	var conv Conversation
	if err := s.db.WithContext(ctx).Where("id = ?", share.ConversationID).First(&conv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShareNotFound
		}
		return nil, err
	}
	var messages []Message
	if err := s.db.WithContext(ctx).Where("conversation_id = ?", conv.ID).
		Order("id ASC").Find(&messages).Error; err != nil {
		return nil, err
	}
	return &SharedConversation{
		Conversation: conv,
		Messages:     messages,
		Owner:        s.ownerName(ctx, share.UserID),
		CreatedAt:    share.CreatedAt,
	}, nil
}

// ownerName 返回分享者昵称，分享页不暴露邮箱。
func (s *Service) ownerName(ctx context.Context, userID uint) string {
	if s.users == nil {
		return ""
	}
	u, err := s.users.ByID(ctx, userID)
	if err != nil || u == nil {
		return ""
	}
	return u.Nickname
}

func trimToken(value string) string {
	return strings.TrimSpace(value)
}
