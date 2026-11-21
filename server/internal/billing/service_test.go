package billing

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&Plan{}, &Event{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Create(&Plan{Code: PlanFree, Name: "免费版", SpendLimitCents: 100, LimitShort: 3, LimitWeek: 5, LimitMonth: 10, Enabled: true})
	return db
}

func newTestService(t *testing.T, db *gorm.DB) *Service {
	svc := NewService(db)
	svc.now = func() time.Time { return time.Date(2026, 11, 21, 12, 0, 0, 0, time.Local) }
	return svc
}

// TestCheckPassesUnderLimits 校验未触顶时放行。
func TestCheckPassesUnderLimits(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(t, db)
	if _, err := svc.Check(context.Background(), PlanFree, 1); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

// TestShortWindowLimit 校验 5 小时窗口触顶。
func TestShortWindowLimit(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(t, db)
	for i := 0; i < 3; i++ {
		db.Create(&Event{UserID: 1, CreatedAt: svc.now().Add(-time.Hour)})
	}
	_, err := svc.Check(context.Background(), PlanFree, 1)
	var limit *LimitError
	if !errors.As(err, &limit) || !errors.Is(err, ErrShortLimit) {
		t.Fatalf("err = %v, want ErrShortLimit", err)
	}
	if limit.Limit != 3 || limit.Used != 3 {
		t.Fatalf("limit = %+v", limit)
	}
}

// TestWeekLimit 校验窗口外的事件不计入。
func TestWeekLimit(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(t, db)
	// 2 条在 5 小时内，3 条在 2 天前：5h 不触顶，week 达到 5
	for i := 0; i < 2; i++ {
		db.Create(&Event{UserID: 1, CreatedAt: svc.now().Add(-time.Hour)})
	}
	for i := 0; i < 3; i++ {
		db.Create(&Event{UserID: 1, CreatedAt: svc.now().Add(-48 * time.Hour)})
	}
	_, err := svc.Check(context.Background(), PlanFree, 1)
	if !errors.Is(err, ErrWeekLimit) {
		t.Fatalf("err = %v, want ErrWeekLimit", err)
	}
}

// TestSpendLimit 校验月消费触顶，且费用按分统计。
func TestSpendLimit(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(t, db)
	db.Create(&Event{UserID: 1, CostMilliCents: 60_000, CreatedAt: svc.now()})
	db.Create(&Event{UserID: 1, CostMilliCents: 50_000, CreatedAt: svc.now()})
	// 100 分 = 1 元
	_, err := svc.Check(context.Background(), PlanFree, 1)
	if !errors.Is(err, ErrSpendLimit) {
		t.Fatalf("err = %v, want ErrSpendLimit", err)
	}
}

// TestSnapshotWindows 校验三个窗口的计数与花费。
func TestSnapshotWindows(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(t, db)
	now := svc.now()
	db.Create(&Event{UserID: 1, CostMilliCents: 10_000, CreatedAt: now.Add(-time.Hour)})
	db.Create(&Event{UserID: 1, CostMilliCents: 20_000, CreatedAt: now.Add(-72 * time.Hour)})
	db.Create(&Event{UserID: 1, CostMilliCents: 30_000, CreatedAt: now.Add(-400 * time.Hour)})

	snap, err := svc.Snapshot(context.Background(), PlanFree, 1)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.RequestsShort != 1 || snap.RequestsWeek != 2 || snap.RequestsMonth != 3 {
		t.Fatalf("counts = %d/%d/%d", snap.RequestsShort, snap.RequestsWeek, snap.RequestsMonth)
	}
	// 10000+20000+30000 = 60000 milli-cents = 60 分
	if snap.SpendCents != 60 {
		t.Fatalf("spend = %d, want 60", snap.SpendCents)
	}
}

// TestCostMilliCents 校验费用换算：单价按元 / 百万 token。
func TestCostMilliCents(t *testing.T) {
	// 1M 输入 token，单价 2 元 => 2 元 = 200 分 => 200000 毫分
	if got := CostMilliCents(1_000_000, 0, 2, 8); got != 200_000 {
		t.Fatalf("got %d, want 200000", got)
	}
	// 1000 输出 token，单价 8 元 => 0.008 元
	if got := CostMilliCents(0, 1000, 2, 8); got != 800 {
		t.Fatalf("got %d, want 800", got)
	}
}

// TestMigrateSeedsDefaults 校验默认三档套餐存在，且与需求一致。
func TestMigrateSeedsDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := NewService(db)
	plans, err := svc.Plans(context.Background())
	if err != nil {
		t.Fatalf("Plans: %v", err)
	}
	want := map[string][4]int64{
		PlanFree:     {100, 50, 100, 200},
		PlanPro:      {1000, 150, 400, 1000},
		PlanUltimate: {5000, 400, 2100, 5000},
	}
	if len(plans) != len(want) {
		t.Fatalf("plans = %d, want %d", len(plans), len(want))
	}
	for _, plan := range plans {
		expected, ok := want[plan.Code]
		if !ok {
			t.Fatalf("unexpected plan %s", plan.Code)
		}
		got := [4]int64{plan.SpendLimitCents, int64(plan.LimitShort), int64(plan.LimitWeek), int64(plan.LimitMonth)}
		if got != expected {
			t.Fatalf("%s = %v, want %v", plan.Code, got, expected)
		}
	}
	fmt.Println("plans ok", len(plans))
}
