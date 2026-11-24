package memo

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestMemoSaveSearchDelete 校验备忘录按用户隔离、可检索、可删除。
func TestMemoSaveSearchDelete(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:memo?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := NewService(db)
	ctx := context.Background()

	saved, err := svc.Save(ctx, 1, SaveInput{Content: "把检索超时改成空闲 180s\n细节：有数据就重置"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if saved.Title == "" {
		t.Fatal("title should default to the first line")
	}

	// 另一个用户看不到
	other, err := svc.Search(ctx, 2, "", 10)
	if err != nil {
		t.Fatalf("search other: %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("user isolation broken: %+v", other)
	}

	items, err := svc.Search(ctx, 1, "超时", 10)
	if err != nil || len(items) != 1 {
		t.Fatalf("search: %v %+v", err, items)
	}

	if _, err := svc.Update(ctx, 1, saved.ID, SaveInput{Tags: []string{"超时", "超时", "架构"}}); err != nil {
		t.Fatalf("update: %v", err)
	}
	tagged, _ := svc.Get(ctx, 1, saved.ID)
	if tagged.Tags != "超时,架构" {
		t.Fatalf("tags = %q, want deduplicated", tagged.Tags)
	}

	if err := svc.Delete(ctx, 2, saved.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-user delete err = %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, 1, saved.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
