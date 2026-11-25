package user

import (
	"context"
	"sync"
	"testing"
)

// TestConcurrentLoginFailuresAccumulate 并发失败登录必须逐次累加，
// 否则爆破者可以靠并发绕过锁定阈值。
func TestConcurrentLoginFailuresAccumulate(t *testing.T) {
	svc, _, db := newTestService(t)
	ctx := context.Background()
	u, _, err := svc.Register(ctx, RegisterInput{Email: "race@deeptalk.local", Password: "Race-pass-2026"}, RequestMeta{})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	const attempts = 8
	var wg sync.WaitGroup
	wg.Add(attempts)
	for i := 0; i < attempts; i++ {
		go func() {
			defer wg.Done()
			_, _, _ = svc.Login(ctx, "race@deeptalk.local", "wrong-password", RequestMeta{})
		}()
	}
	wg.Wait()

	var row User
	if err := db.First(&row, u.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	// 阈值 3：8 次失败至少应触发两次锁定，并最终处于锁定态
	if row.LockedUntil == nil {
		t.Fatalf("expected account to be locked after %d concurrent failures", attempts)
	}
	if _, _, err := svc.Login(ctx, "race@deeptalk.local", "Race-pass-2026", RequestMeta{}); err == nil {
		t.Fatal("expected locked account to reject even the correct password")
	}
}
