package user

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestDeleteUserGuardsAdmin 校验管理员不可删除，普通用户可删且连带会话。
func TestDeleteUserGuardsAdmin(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:deluser?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := NewService(db, nil, nil, Options{})

	admin := &User{Email: "admin@test.local", PasswordHash: "x", Role: RoleAdmin, Plan: PlanFree}
	plain := &User{Email: "plain@test.local", PasswordHash: "x", Role: RoleUser, Plan: PlanFree}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if err := db.Create(plain).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := svc.Delete(context.Background(), admin.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("delete admin err = %v, want ErrForbidden", err)
	}
	if err := svc.Delete(context.Background(), plain.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	if _, err := dalFindByID(context.Background(), db, plain.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("user should be gone, err = %v", err)
	}
}
