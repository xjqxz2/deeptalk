// Package user 负责账号、密码、会话与额度。
// 数据访问函数统一以 dal 前缀命名，业务规则集中在 Service，其他模块只调用导出方法。
package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"gorm.io/gorm"
)

// 角色常量。MVP 只有平台管理员与普通用户两种。
const (
	RoleAdmin = "admin" // 平台管理员，可进入 B 端
	RoleUser  = "user"  // 普通用户，仅能使用 C 端
)

// 套餐常量。套餐只影响额度与模型权限，MVP 不做支付。
const (
	PlanFree      = "free"      // 免费版：4M tokens / 月
	PlanPro       = "pro"       // 专业版：20M tokens / 月
	PlanTeam      = "team"      // 团队版：50M tokens / 月
	PlanUnlimited = "unlimited" // 无限版：不限制月度 tokens（quota=0）
)

// 业务错误。控制器按 errors.Is 映射到统一错误码。
var (
	ErrNotFound           = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrLocked             = errors.New("account locked")
	ErrDisabled           = errors.New("account disabled")
	ErrEmailTaken         = errors.New("email already registered")
	ErrRegistrationClosed = errors.New("registration closed")
	ErrUnauthenticated    = errors.New("session invalid")
	ErrForbidden          = errors.New("operation forbidden")
)

// User 是 C 端账号模型。
// 邮箱统一小写且唯一；用量按自然月统计在 used_tokens 上。
type User struct {
	ID               uint       `gorm:"primaryKey"`                            // 主键
	Email            string     `gorm:"size:190;uniqueIndex;not null"`         // 邮箱（小写唯一）
	Nickname         string     `gorm:"size:64;not null"`                      // 昵称
	PasswordHash     string     `gorm:"size:255;not null"`                     // Argon2id 哈希串
	Role             string     `gorm:"size:16;not null;default:'user';index"` // admin / user
	Plan             string     `gorm:"size:16;not null;default:'free'"`       // free / pro / team / unlimited
	QuotaTokens      int64      `gorm:"not null;default:4000000"`              // 本月额度（tokens）
	UsedTokens       int64      `gorm:"not null;default:0"`                    // 本月已用（tokens）
	Disabled         bool       `gorm:"not null;default:false;index"`          // 是否限制登录
	FailedAttempts   int        `gorm:"not null;default:0"`                    // 连续登录失败次数
	LockedUntil      *time.Time `gorm:"index"`                                 // 锁定截止时间
	LastLoginAt      *time.Time `gorm:"index"`                                 // 最近登录时间
	LastLoginIP      string     `gorm:"size:64;not null;default:''"`           // 最近登录 IP
	LastLoginCity    string     `gorm:"size:32;not null;default:''"`           // 最近登录城市（粗粒度）
	WeChatOpenID     *string    `gorm:"size:64;uniqueIndex"`                   // 小程序 openid，用于内容安全
	EmailPlaceholder bool       `gorm:"not null;default:false"`                // 邮箱是系统占位，尚未由用户补全
	AvatarURL        string     `gorm:"size:512;not null;default:''"`          // 头像地址
	CreatedAt        time.Time  `gorm:"index"`                                 // 注册时间
	UpdatedAt        time.Time  // 更新时间
}

// Session 是服务端会话（只保存 token 的 SHA-256，泄露数据库也无法直接使用）。
type Session struct {
	ID        uint       `gorm:"primaryKey"`                   // 主键
	UserID    uint       `gorm:"index;not null"`               // 所属用户
	TokenHash string     `gorm:"size:64;uniqueIndex;not null"` // token 的 SHA-256 十六进制
	ExpiresAt time.Time  `gorm:"index;not null"`               // 过期时间
	RevokedAt *time.Time `gorm:"index"`                        // 撤销时间，非空表示失效
	IP        string     `gorm:"size:64;not null;default:''"`  // 签发 IP
	UserAgent string     `gorm:"size:255;not null;default:''"` // 客户端 UA
	CreatedAt time.Time  // 签发时间
}

// Usage 是 C 端用量卡片的展示结构。
type Usage struct {
	UsedTokens  int64  `json:"used_tokens"`  // 本月已用 tokens
	QuotaTokens int64  `json:"quota_tokens"` // 本月额度 tokens
	UsedPercent int    `json:"used_percent"` // 使用百分比（向上取整，0–100）
	ResetDays   int    `json:"reset_days"`   // 距额度重置剩余天数
	Plan        string `json:"plan"`         // 套餐标识
}

// Stats 是后台用户管理的统计口径。
type Stats struct {
	Total         int64 `json:"total"`          // 用户总数
	NewThisMonth  int64 `json:"new_this_month"` // 本月新增
	Active7D      int64 `json:"active_7d"`      // 7 日活跃（按最近登录）
	ActiveRate    int   `json:"active_rate"`    // 周活跃率百分比
	MonthlyTokens int64 `json:"monthly_tokens"` // 本月 tokens 总量
	Restricted    int64 `json:"restricted"`     // 已限制登录数
	QuotaAlert    int64 `json:"quota_alert"`    // 额度将尽数（≥90% 且未受限）
	MonthlyGrowth int   `json:"monthly_growth"` // 本月用量环比（演示数据固定 +18）
}

// ListParams 是后台用户列表的查询条件。
type ListParams struct {
	Query    string // 邮箱或昵称关键词
	Status   string // all / normal / restricted / quota
	Sort     string // recent / created / usage
	Page     int    // 页码，从 1 开始
	PageSize int    // 每页条数
}

// dalFindByEmail 按小写邮箱查询用户。
func dalFindByEmail(ctx context.Context, db *gorm.DB, email string) (*User, error) {
	var u User
	err := db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// dalFindByID 按主键查询用户。
func dalFindByID(ctx context.Context, db *gorm.DB, id uint) (*User, error) {
	var u User
	err := db.WithContext(ctx).First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// dalCountUsers 统计账号总数，用于判断“首个账号即管理员”。
func dalCountUsers(ctx context.Context, db *gorm.DB) (int64, error) {
	var count int64
	err := db.WithContext(ctx).Model(&User{}).Count(&count).Error
	return count, err
}

// dalCreateUser 创建账号。
func dalCreateUser(ctx context.Context, db *gorm.DB, u *User) error {
	return db.WithContext(ctx).Create(u).Error
}

// dalUpdateUser 保存账号全部字段。
func dalUpdateUser(ctx context.Context, db *gorm.DB, u *User) error {
	return db.WithContext(ctx).Save(u).Error
}

// dalCreateSession 写入会话。
func dalCreateSession(ctx context.Context, db *gorm.DB, s *Session) error {
	return db.WithContext(ctx).Create(s).Error
}

// dalFindSession 按 token 哈希查询未撤销且未过期的会话。
func dalFindSession(ctx context.Context, db *gorm.DB, tokenHash string) (*Session, error) {
	var s Session
	now := time.Now()
	err := db.WithContext(ctx).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, now).
		First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUnauthenticated
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// dalRevokeSession 撤销单个会话。
func dalRevokeSession(ctx context.Context, db *gorm.DB, tokenHash string) error {
	now := time.Now()
	return db.WithContext(ctx).Model(&Session{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Update("revoked_at", now).Error
}

// dalRevokeUserSessions 撤销某用户的全部有效会话。
func dalRevokeUserSessions(ctx context.Context, db *gorm.DB, userID uint) error {
	now := time.Now()
	return db.WithContext(ctx).Model(&Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

// newSessionToken 生成 32 字节随机 token 及其 SHA-256。
// 返回值 token 只下发给浏览器，数据库只保存 hash。
func newSessionToken() (token string, tokenHash string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(sum[:]), nil
}

// quotaPercent 计算使用百分比，向上取整，封顶 100。
func quotaPercent(used, quota int64) int {
	if quota <= 0 {
		return 0
	}
	if used <= 0 {
		return 0
	}
	percent := int((used*100 + quota - 1) / quota)
	if percent > 100 {
		return 100
	}
	return percent
}

// resetDays 返回到下月 1 日的剩余天数（至少 1），对应设计图“重置 12 天”。
func resetDays(now time.Time) int {
	next := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	days := int(next.Sub(now).Hours() / 24)
	if days < 1 {
		return 1
	}
	return days
}

// planQuota 返回套餐默认额度。
func planQuota(plan string) int64 {
	switch plan {
	case PlanPro:
		return 20_000_000
	case PlanTeam:
		return 50_000_000
	case PlanUnlimited:
		return 0
	default:
		return 4_000_000
	}
}

// Migrate 创建账号与会话表。
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Session{})
}

// dalFindByWeChatOpenID 按 openid 查询用户。
func dalFindByWeChatOpenID(ctx context.Context, db *gorm.DB, openid string) (*User, error) {
	var u User
	err := db.WithContext(ctx).Where("wechat_open_id = ?", openid).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}
