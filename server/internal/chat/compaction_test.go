package chat

import "testing"

// TestChooseCompactionBoundary 校验压缩边界会保留最近上下文，且尽量落在 assistant 之后。
func TestChooseCompactionBoundary(t *testing.T) {
	rows := []Message{
		{ID: 1, Role: RoleUser, Content: "第一轮问题，大概十个字左右"},
		{ID: 2, Role: RoleAssistant, Content: "第一轮回答，内容比较长，用来占一些 token"},
		{ID: 3, Role: RoleUser, Content: "第二轮问题"},
		{ID: 4, Role: RoleAssistant, Content: "第二轮回答"},
		{ID: 5, Role: RoleUser, Content: "最近的问题"},
		{ID: 6, Role: RoleAssistant, Content: "最近的回答"},
	}
	boundary := chooseCompactionBoundary(rows, 30)
	if boundary <= 0 || boundary >= len(rows) {
		t.Fatalf("boundary = %d, want within prefix", boundary)
	}
	if rows[boundary-1].Role != RoleAssistant {
		t.Fatalf("boundary should end after assistant, got role %q", rows[boundary-1].Role)
	}
	if boundary < 2 {
		t.Fatalf("should compress at least two messages, got %d", boundary)
	}
	if got := chooseCompactionBoundary(rows[:2], 100); got != 0 {
		t.Fatalf("short history should not compact, got %d", got)
	}
	if got := chooseCompactionBoundary(rows, 100000); got != 0 {
		t.Fatalf("history inside budget should not compact, got %d", got)
	}
}
