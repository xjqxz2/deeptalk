package user

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"deeptalk/internal/audit"
	"deeptalk/internal/setting"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestService 创建基于临时 SQLite 文件的账号服务，避免测试之间互相污染。
func newTestService(t *testing.T) (*Service, *setting.Service, *gorm.DB) {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "test.db") + "?_pragma=busy_timeout(5000)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	for name, migrate := range map[string]func(*gorm.DB) error{
		"user":    Migrate,
		"setting": setting.Migrate,
		"audit":   audit.Migrate,
	} {
		if err := migrate(db); err != nil {
			t.Fatalf("migrate %s: %v", name, err)
		}
	}
	settings := setting.NewService(db)
	recorder := audit.NewService(db)
	svc := NewService(db, settings, recorder, Options{
		SessionTTL:    time.Hour,
		LockThreshold: 3,
		LockMinutes:   10,
		DefaultQuota:  4_000_000,
	})
	return svc, settings, db
}

// TestRegisterLoginAndRoles 校验注册即登录、注册账号为普通用户与邮箱唯一。
// 管理员由迁移写入（admin），不再由「第一个注册的人」担任。
func TestRegisterLoginAndRoles(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	meta := RequestMeta{IP: "127.0.0.1", UserAgent: "test"}

	first, token, err := svc.Register(ctx, RegisterInput{Email: "LMO@DeepTalk.app", Password: "DeepTalk123"}, meta)
	if err != nil {
		t.Fatalf("register first: %v", err)
	}
	if first.Role != RoleUser {
		t.Fatalf("registered user role = %s, want user", first.Role)
	}
	if first.Email != "lmo@deeptalk.app" {
		t.Fatalf("email should be normalized, got %s", first.Email)
	}
	if token == "" {
		t.Fatal("session token should not be empty")
	}
	authed, err := svc.Authenticate(ctx, token)
	if err != nil || authed.ID != first.ID {
		t.Fatalf("authenticate: %v, user=%v", err, authed)
	}

	second, _, err := svc.Register(ctx, RegisterInput{Email: "user@example.com", Password: "DeepTalk123"}, meta)
	if err != nil {
		t.Fatalf("register second: %v", err)
	}
	if second.Role != RoleUser {
		t.Fatalf("second user role = %s, want user", second.Role)
	}

	if _, _, err := svc.Register(ctx, RegisterInput{Email: "USER@example.com", Password: "DeepTalk123"}, meta); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("duplicate email err = %v, want ErrEmailTaken", err)
	}
	if _, _, err := svc.Register(ctx, RegisterInput{Email: "weak@example.com", Password: "12345678"}, meta); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("weak password err = %v, want ErrWeakPassword", err)
	}
	if _, _, err := svc.Login(ctx, first.Email, "DeepTalk123", meta); err != nil {
		t.Fatalf("login: %v", err)
	}
}

// TestLoginLockAndDisabled 校验连续失败锁定与限制登录后会话立即失效。
func TestLoginLockAndDisabled(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	meta := RequestMeta{IP: "127.0.0.1"}
	if _, _, err := svc.Register(ctx, RegisterInput{Email: "lock@example.com", Password: "DeepTalk123"}, meta); err != nil {
		t.Fatalf("register: %v", err)
	}
	for i := 0; i < 2; i++ {
		if _, _, err := svc.Login(ctx, "lock@example.com", "wrong-pass", meta); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d err = %v, want ErrInvalidCredentials", i+1, err)
		}
	}
	// 第 3 次失败即触发锁定
	if _, _, err := svc.Login(ctx, "lock@example.com", "wrong-pass", meta); !errors.Is(err, ErrLocked) {
		t.Fatalf("third failure err = %v, want ErrLocked", err)
	}
	if _, _, err := svc.Login(ctx, "lock@example.com", "DeepTalk123", meta); !errors.Is(err, ErrLocked) {
		t.Fatalf("locked login err = %v, want ErrLocked", err)
	}

	// 新建一个普通用户并限制登录，验证会话立即失效
	normal, token, err := svc.Register(ctx, RegisterInput{Email: "normal@example.com", Password: "DeepTalk123"}, meta)
	if err != nil {
		t.Fatalf("register normal: %v", err)
	}
	if _, err := svc.SetDisabled(ctx, normal.ID, true); err != nil {
		t.Fatalf("disable: %v", err)
	}
	// 受限时全部会话被撤销，因此表现为会话失效；登录侧返回 ErrDisabled
	if _, err := svc.Authenticate(ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("authenticate disabled err = %v, want ErrUnauthenticated", err)
	}
	if _, _, err := svc.Login(ctx, "normal@example.com", "DeepTalk123", meta); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled login err = %v, want ErrDisabled", err)
	}
}

// TestRegistrationSwitchAndReset 校验注册开关与管理员重置密码。
func TestRegistrationSwitchAndReset(t *testing.T) {
	svc, settings, _ := newTestService(t)
	ctx := context.Background()
	meta := RequestMeta{IP: "127.0.0.1"}
	if _, _, err := svc.Register(ctx, RegisterInput{Email: "admin@example.com", Password: "DeepTalk123"}, meta); err != nil {
		t.Fatalf("register admin: %v", err)
	}
	if err := settings.Set(ctx, setting.KeyAllowRegistration, "false"); err != nil {
		t.Fatalf("close registration: %v", err)
	}
	if _, _, err := svc.Register(ctx, RegisterInput{Email: "new@example.com", Password: "DeepTalk123"}, meta); !errors.Is(err, ErrRegistrationClosed) {
		t.Fatalf("closed registration err = %v, want ErrRegistrationClosed", err)
	}
	created, err := svc.CreateByAdmin(ctx, AdminCreateInput{
		Email: "created@example.com", Password: "DeepTalk123", Plan: PlanPro, QuotaTokens: 20_000_000,
	})
	if err != nil {
		t.Fatalf("admin create: %v", err)
	}
	password, err := svc.ResetPassword(ctx, created.ID)
	if err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if err := ValidatePassword(password); err != nil {
		t.Fatalf("generated password invalid: %v", err)
	}
	if _, _, err := svc.Login(ctx, "created@example.com", password, meta); err != nil {
		t.Fatalf("login with reset password: %v", err)
	}
}

// TestUsageAndQuota 校验用量百分比与额度下限校验。
func TestUsageAndQuota(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	u, _, err := svc.Register(ctx, RegisterInput{Email: "quota@example.com", Password: "DeepTalk123"}, RequestMeta{IP: "127.0.0.1"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := svc.ConsumeUsage(ctx, u.ID, 1_280_000); err != nil {
		t.Fatalf("consume: %v", err)
	}
	latest, err := svc.ByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("by id: %v", err)
	}
	usage := svc.Usage(latest)
	if usage.UsedPercent != 32 {
		t.Fatalf("used percent = %d, want 32", usage.UsedPercent)
	}
	if usage.ResetDays < 1 || usage.ResetDays > 31 {
		t.Fatalf("reset days = %d, want 1..31", usage.ResetDays)
	}
	if _, err := svc.UpdateQuota(ctx, u.ID, 500_000); err == nil {
		t.Fatal("quota below used should fail")
	}
	updated, err := svc.UpdateQuota(ctx, u.ID, 8_000_000)
	if err != nil {
		t.Fatalf("update quota: %v", err)
	}
	if updated.QuotaTokens != 8_000_000 {
		t.Fatalf("quota = %d, want 8000000", updated.QuotaTokens)
	}
}
