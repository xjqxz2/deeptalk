package user

import (
	"context"
	"testing"
)

// TestSetRoleGuards 角色调整必须挡住两种会把后台锁死的操作：
// 取消自己、以及取消最后一个管理员。
func TestSetRoleGuards(t *testing.T) {
	svc, _, db := newTestService(t)
	ctx := context.Background()

	var admin User
	if err := db.Where("role = ?", RoleAdmin).First(&admin).Error; err != nil {
		t.Fatalf("migration should create an admin: %v", err)
	}

	// 取消自己：拒绝
	if _, err := svc.SetRole(ctx, admin.ID, admin.ID, RoleUser); err == nil {
		t.Fatal("self demotion must be rejected")
	}

	// 取消最后一个管理员：拒绝
	if _, err := svc.SetRole(ctx, admin.ID, admin.ID, RoleUser); err == nil {
		t.Fatal("demoting the last admin must be rejected")
	}

	// 提升普通用户，然后允许他取消原管理员（此时不止一个管理员）
	member, _, err := svc.Register(ctx, RegisterInput{Email: "member@deeptalk.local", Password: "Member123"}, RequestMeta{})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.SetRole(ctx, admin.ID, member.ID, RoleAdmin); err != nil {
		t.Fatalf("promote: %v", err)
	}
	promoted, err := svc.ByID(ctx, member.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if promoted.Role != RoleAdmin {
		t.Fatalf("role = %s, want admin", promoted.Role)
	}
	if _, err := svc.SetRole(ctx, member.ID, admin.ID, RoleUser); err != nil {
		t.Fatalf("demote when another admin exists should pass: %v", err)
	}
}

// TestRegisterNeverGrantsAdmin 注册与微信首登都不能产出管理员，
// 否则删掉管理员后第一个注册/登录的人就能拿到后台权限。
func TestRegisterNeverGrantsAdmin(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	u, _, err := svc.Register(ctx, RegisterInput{Email: "nobody@deeptalk.local", Password: "Nobody123"}, RequestMeta{})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if u.Role != RoleUser {
		t.Fatalf("role = %s, want user", u.Role)
	}
}
