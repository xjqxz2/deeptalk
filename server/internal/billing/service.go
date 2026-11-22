package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// 业务错误：限额触顶。
var (
	ErrSpendLimit   = errors.New("本月费用额度已用完")
	ErrShortLimit   = errors.New("5 小时内的请求次数已达上限")
	ErrWeekLimit    = errors.New("本周的请求次数已达上限")
	ErrMonthLimit   = errors.New("本月的请求次数已达上限")
	ErrPlanNotFound = errors.New("套餐不存在")
)

// LimitError 带上是哪一条限制、上限值与恢复时间，便于前端提示。
type LimitError struct {
	Kind     error     // ErrSpendLimit / ErrShortLimit / ...
	Limit    int64     // 上限值（费用为分，次数为次）
	Used     int64     // 当前已用
	ResetsAt time.Time // 最早恢复时间
	Window   string    // 5h / week / month
}

func (e *LimitError) Error() string {
	return e.Kind.Error()
}

func (e *LimitError) Unwrap() error { return e.Kind }

// Snapshot 是某个用户当前的用量快照。
type Snapshot struct {
	Plan            Plan      `json:"plan"`
	SpendCents      int64     `json:"spend_cents"`       // 本月已花费（分）
	SpendLimitCents int64     `json:"spend_limit_cents"` // 上限（分），0 不限
	RequestsShort   int       `json:"requests_short"`
	RequestsWeek    int       `json:"requests_week"`
	RequestsMonth   int       `json:"requests_month"`
	ShortResetsAt   time.Time `json:"short_resets_at"`
	WeekResetsAt    time.Time `json:"week_resets_at"`
	MonthResetsAt   time.Time `json:"month_resets_at"`
}

// Money 把「分的千分之一」转成展示用的元，保留两位小数。
func Money(milliCents int64) float64 {
	return float64(milliCents) / 100000
}

// Service 提供套餐查询、用量统计与限额校验。
type Service struct {
	db  *gorm.DB
	now func() time.Time
}

// NewService 创建计费服务。
func NewService(db *gorm.DB) *Service {
	return &Service{db: db, now: time.Now}
}

// Plans 返回全部启用的套餐，按排序值升序。
func (s *Service) Plans(ctx context.Context) ([]Plan, error) {
	var items []Plan
	if err := s.db.WithContext(ctx).Where("enabled = ?", true).Order("sort_order asc, id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// AllPlans 返回全部套餐（后台管理用）。
func (s *Service) AllPlans(ctx context.Context) ([]Plan, error) {
	var items []Plan
	if err := s.db.WithContext(ctx).Order("sort_order asc, id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// PlanByCode 按编码取套餐；找不到时回退到免费版。
func (s *Service) PlanByCode(ctx context.Context, code string) (Plan, error) {
	var plan Plan
	err := s.db.WithContext(ctx).Where("code = ?", code).First(&plan).Error
	if err == nil {
		return plan, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return Plan{}, err
	}
	if err := s.db.WithContext(ctx).Where("code = ?", PlanFree).First(&plan).Error; err != nil {
		return Plan{}, ErrPlanNotFound
	}
	return plan, nil
}

// Snapshot 统计用户当前窗口内的花费与请求次数。
func (s *Service) Snapshot(ctx context.Context, planCode string, userID uint) (Snapshot, error) {
	plan, err := s.PlanByCode(ctx, planCode)
	if err != nil {
		return Snapshot{}, err
	}
	now := s.now()
	snap := Snapshot{
		Plan:            plan,
		SpendLimitCents: plan.SpendLimitCents,
		ShortResetsAt:   now.Add(WindowShort),
		WeekResetsAt:    now.Add(WindowWeek),
		MonthResetsAt:   now.Add(WindowMonth),
	}
	// 花费：自然月口径
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	var spend struct{ Total int64 }
	if err := s.db.WithContext(ctx).Model(&Event{}).
		Select("COALESCE(SUM(cost_milli_cents), 0) AS total").
		Where("user_id = ? AND created_at >= ?", userID, monthStart).
		Scan(&spend).Error; err != nil {
		return Snapshot{}, err
	}
	snap.SpendCents = spend.Total / 1000

	count := func(since time.Time) (int, error) {
		var total int64
		err := s.db.WithContext(ctx).Model(&Event{}).
			Where("user_id = ? AND created_at >= ?", userID, since).
			Count(&total).Error
		return int(total), err
	}
	if snap.RequestsShort, err = count(now.Add(-WindowShort)); err != nil {
		return Snapshot{}, err
	}
	if snap.RequestsWeek, err = count(now.Add(-WindowWeek)); err != nil {
		return Snapshot{}, err
	}
	if snap.RequestsMonth, err = count(now.Add(-WindowMonth)); err != nil {
		return Snapshot{}, err
	}
	snap.ShortResetsAt = s.nextRelease(ctx, userID, now.Add(-WindowShort), WindowShort)
	snap.WeekResetsAt = s.nextRelease(ctx, userID, now.Add(-WindowWeek), WindowWeek)
	snap.MonthResetsAt = s.nextRelease(ctx, userID, now.Add(-WindowMonth), WindowMonth)
	return snap, nil
}

// nextRelease 估算窗口内最早释放名额的时间：取窗口内第 (count-limit+1) 条事件的时间 + 窗口长度。
func (s *Service) nextRelease(ctx context.Context, userID uint, since time.Time, window time.Duration) time.Time {
	var first Event
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND created_at >= ?", userID, since).
		Order("created_at asc").First(&first).Error; err != nil {
		return s.now().Add(window)
	}
	return first.CreatedAt.Add(window)
}

// Check 校验用户是否还能发起新一轮请求；触顶返回 *LimitError。
func (s *Service) Check(ctx context.Context, planCode string, userID uint) (*Snapshot, error) {
	snap, err := s.Snapshot(ctx, planCode, userID)
	if err != nil {
		return nil, err
	}
	plan := snap.Plan
	// 1) 费用上限
	if plan.SpendLimitCents > 0 && snap.SpendCents >= plan.SpendLimitCents {
		return &snap, &LimitError{
			Kind:     ErrSpendLimit,
			Limit:    plan.SpendLimitCents,
			Used:     snap.SpendCents,
			ResetsAt: s.monthStart().AddDate(0, 1, 0),
			Window:   "month",
		}
	}
	// 2) 请求次数上限：3 个窗口任一触顶即拒绝
	checks := []struct {
		limit  int
		used   int
		kind   error
		reset  time.Time
		window string
	}{
		{plan.LimitShort, snap.RequestsShort, ErrShortLimit, snap.ShortResetsAt, "5h"},
		{plan.LimitWeek, snap.RequestsWeek, ErrWeekLimit, snap.WeekResetsAt, "week"},
		{plan.LimitMonth, snap.RequestsMonth, ErrMonthLimit, snap.MonthResetsAt, "month"},
	}
	for _, item := range checks {
		if item.limit > 0 && item.used >= item.limit {
			return &snap, &LimitError{
				Kind:     item.kind,
				Limit:    int64(item.limit),
				Used:     int64(item.used),
				ResetsAt: item.reset,
				Window:   item.window,
			}
		}
	}
	return &snap, nil
}

// Record 记录一次成功生成的花费与请求次数。
func (s *Service) Record(ctx context.Context, event Event) error {
	if event.UserID == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Create(&event).Error
}

// CostMilliCents 按模型单价（元 / 百万 token）换算本次费用，单位「分的千分之一」。
func CostMilliCents(inputTokens, outputTokens int, priceInput, priceOutput float64) int64 {
	return CostMilliCentsCached(inputTokens, 0, outputTokens, priceInput, priceInput, priceOutput)
}

// CostMilliCentsCached 三段计价：缓存命中输入、普通输入、输出。
// 上游不给价格（如 OpenCode-GO 的免费模型）时，我们自己按 token 计这一层。
func CostMilliCentsCached(inputTokens, cachedTokens, outputTokens int, priceInput, priceCached, priceOutput float64) int64 {
	if cachedTokens < 0 {
		cachedTokens = 0
	}
	if cachedTokens > inputTokens {
		cachedTokens = inputTokens
	}
	if priceCached <= 0 {
		priceCached = priceInput
	}
	normalInput := inputTokens - cachedTokens
	cost := (float64(normalInput)*priceInput + float64(cachedTokens)*priceCached + float64(outputTokens)*priceOutput) / 1_000_000
	return int64(cost * 100000)
}

func (s *Service) monthStart() time.Time {
	now := s.now()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}

// CountHours 用于日志与测试。
func CountHours(hours int) string { return fmt.Sprintf("%dh", hours) }

// UpdateInput 是套餐更新的可选字段；nil 表示不改。
type UpdateInput struct {
	Name            string
	Description     string
	SpendLimitCents *int64
	LimitShort      *int
	LimitWeek       *int
	LimitMonth      *int
	Enabled         *bool
}

// Update 更新套餐配置；限额为 0 表示该条不限。
func (s *Service) Update(ctx context.Context, id uint, in UpdateInput) (Plan, error) {
	var plan Plan
	err := s.db.WithContext(ctx).First(&plan, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Plan{}, ErrPlanNotFound
	}
	if err != nil {
		return Plan{}, err
	}
	if in.Name != "" {
		plan.Name = in.Name
	}
	if in.Description != "" {
		plan.Description = in.Description
	}
	if in.SpendLimitCents != nil {
		plan.SpendLimitCents = *in.SpendLimitCents
	}
	if in.LimitShort != nil {
		plan.LimitShort = *in.LimitShort
	}
	if in.LimitWeek != nil {
		plan.LimitWeek = *in.LimitWeek
	}
	if in.LimitMonth != nil {
		plan.LimitMonth = *in.LimitMonth
	}
	if in.Enabled != nil {
		plan.Enabled = *in.Enabled
	}
	if err := s.db.WithContext(ctx).Save(&plan).Error; err != nil {
		return Plan{}, err
	}
	return plan, nil
}

// PurgeUser 删除某个用户的计费事件（管理员删号时调用）。
func (s *Service) PurgeUser(ctx context.Context, userID uint) error {
	return s.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&Event{}).Error
}
