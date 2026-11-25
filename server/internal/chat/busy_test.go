package chat

import "testing"

// TestBusySetRejectsSecondRun 同一会话只允许一轮生成占用。
func TestBusySetRejectsSecondRun(t *testing.T) {
	var set busySet
	if !set.tryAdd(7) {
		t.Fatal("first tryAdd should succeed")
	}
	if set.tryAdd(7) {
		t.Fatal("second tryAdd must be rejected while generating")
	}
	if !set.tryAdd(8) {
		t.Fatal("different conversation should not be blocked")
	}
	set.done(7)
	if !set.tryAdd(7) {
		t.Fatal("tryAdd should succeed after done")
	}
}
