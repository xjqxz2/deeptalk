package user

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestWeChatColumnName 校验 openid 使用规范列名，登录查询不再报 no such column。
func TestWeChatColumnName(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:colcheck?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	openid := "oc-test-openid"
	u := &User{Email: "wx-test@wechat.local", Nickname: "", PasswordHash: "x", Role: RoleUser, Plan: PlanFree, WeChatOpenID: &openid}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	found, err := dalFindByWeChatOpenID(context.Background(), db, openid)
	if err != nil {
		t.Fatalf("find by openid: %v", err)
	}
	if found.ID != u.ID {
		t.Fatalf("found %d, want %d", found.ID, u.ID)
	}
	var count int64
	db.Table("users").Where("wechat_open_id = ?", openid).Count(&count)
	if count != 1 {
		t.Fatalf("wechat_open_id rows = %d, want 1", count)
	}
}
