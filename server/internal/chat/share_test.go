package chat

import (
	"context"
	"path/filepath"
	"testing"

	"deeptalk/internal/user"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newShareTestService 准备一个只含会话、消息与分享表的内存数据库。
func newShareTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "share.db") + "?_pragma=busy_timeout(5000)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate chat: %v", err)
	}
	if err := user.Migrate(db); err != nil {
		t.Fatalf("migrate user: %v", err)
	}
	return NewService(db, nil, nil, nil, nil, nil, Config{}), db
}

func TestShareLifecycle(t *testing.T) {
	service, db := newShareTestService(t)
	ctx := context.Background()
	const userID uint = 7

	conv := &Conversation{PublicID: "share-conv", UserID: userID, Title: "分享测试"}
	if err := db.Create(conv).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if err := db.Create(&Message{ConversationID: conv.ID, UserID: userID, Role: RoleUser, Content: "你好", Status: StatusComplete}).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}

	// 未创建时返回空
	if _, err := service.CurrentShare(ctx, userID, conv.ID); err != ErrShareNotFound {
		t.Fatalf("CurrentShare before create = %v, want ErrShareNotFound", err)
	}

	share, err := service.CreateShare(ctx, userID, conv.ID)
	if err != nil {
		t.Fatalf("CreateShare: %v", err)
	}
	if len(share.Token) != 32 {
		t.Fatalf("token length = %d, want 32", len(share.Token))
	}

	// 重复创建复用同一个 token
	again, err := service.CreateShare(ctx, userID, conv.ID)
	if err != nil || again.Token != share.Token {
		t.Fatalf("second CreateShare = %v, %v; want same token", again, err)
	}

	shared, err := service.GetShared(ctx, share.Token)
	if err != nil {
		t.Fatalf("GetShared: %v", err)
	}
	if len(shared.Messages) != 1 || shared.Conversation.Title != "分享测试" {
		t.Fatalf("unexpected shared payload: %+v", shared)
	}

	// 他人不能撤销
	if err := service.RevokeShare(ctx, userID+1, conv.ID); err == nil {
		t.Fatal("RevokeShare by another user should fail")
	}

	if err := service.RevokeShare(ctx, userID, conv.ID); err != nil {
		t.Fatalf("RevokeShare: %v", err)
	}
	if _, err := service.GetShared(ctx, share.Token); err != ErrShareNotFound {
		t.Fatalf("GetShared after revoke = %v, want ErrShareNotFound", err)
	}

	// 重新分享要换新 token，避免旧链接复活
	renewed, err := service.CreateShare(ctx, userID, conv.ID)
	if err != nil {
		t.Fatalf("recreate share: %v", err)
	}
	if renewed.Token == share.Token {
		t.Fatal("recreated share should use a new token")
	}
	if _, err := service.GetShared(ctx, share.Token); err != ErrShareNotFound {
		t.Fatal("old token must stay invalid")
	}
}

func TestShareRequiresOwnership(t *testing.T) {
	service, db := newShareTestService(t)
	ctx := context.Background()
	conv := &Conversation{PublicID: "owned", UserID: 1, Title: "别人的会话"}
	if err := db.Create(conv).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if _, err := service.CreateShare(ctx, 2, conv.ID); err != ErrNotFound {
		t.Fatalf("CreateShare on foreign conversation = %v, want ErrNotFound", err)
	}
	if _, err := service.GetShared(ctx, "not-a-real-token"); err != ErrShareNotFound {
		t.Fatalf("GetShared with unknown token = %v, want ErrShareNotFound", err)
	}
	if _, err := service.GetShared(ctx, "  "); err != ErrShareNotFound {
		t.Fatal("blank token must be rejected")
	}
}
