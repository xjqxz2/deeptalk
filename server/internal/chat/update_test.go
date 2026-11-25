package chat

import (
	"context"
	"testing"
	"time"
)

// TestMessageUpdateOnlyWritesSetColumns 类型化更新必须只写被设置的列，
// 未设置的列（这里是正文）不能被零值覆盖。
func TestMessageUpdateOnlyWritesSetColumns(t *testing.T) {
	_, db := newShareTestService(t)
	ctx := context.Background()
	msg := &Message{
		ConversationID: 1,
		UserID:         1,
		Role:           RoleAssistant,
		Content:        "原始正文",
		Reasoning:      "原始思考",
		Status:         StatusComplete,
	}
	if err := dalCreateMessage(ctx, db, msg); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := dalUpdateMessage(ctx, db, msg.ID, MessageUpdate{Feedback: ptr("up")}); err != nil {
		t.Fatalf("update: %v", err)
	}

	var row Message
	if err := db.First(&row, msg.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if row.Feedback != "up" {
		t.Fatalf("feedback = %q, want up", row.Feedback)
	}
	if row.Content != "原始正文" || row.Reasoning != "原始思考" || row.Status != StatusComplete {
		t.Fatalf("untouched columns were overwritten: %+v", row)
	}
}

// TestMessageUpdateEmptyIsNoop 空更新不应该发出 SQL（避免无意义写入）。
func TestMessageUpdateEmptyIsNoop(t *testing.T) {
	_, db := newShareTestService(t)
	ctx := context.Background()
	msg := &Message{ConversationID: 2, UserID: 1, Role: RoleUser, Content: "x", Status: StatusComplete}
	if err := dalCreateMessage(ctx, db, msg); err != nil {
		t.Fatalf("create: %v", err)
	}
	before := msg.UpdatedAt
	if err := dalUpdateMessage(ctx, db, msg.ID, MessageUpdate{}); err != nil {
		t.Fatalf("empty update: %v", err)
	}
	var row Message
	if err := db.First(&row, msg.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !row.UpdatedAt.Equal(before) {
		t.Fatalf("empty update touched the row: %v -> %v", before, row.UpdatedAt)
	}
}

// TestConversationUpdateFields 只生成被设置的列名。
func TestConversationUpdateFields(t *testing.T) {
	now := time.Now()
	upd := ConversationUpdate{
		Title:     ptr("新标题"),
		UpdatedAt: ptr(now),
	}
	fields := upd.fields()
	if len(fields) != 2 {
		t.Fatalf("fields = %v, want 2 entries", fields)
	}
	if fields["title"] != "新标题" {
		t.Fatalf("title = %v", fields["title"])
	}
	if _, ok := fields["summary"]; ok {
		t.Fatal("summary should not be written when unset")
	}
}
