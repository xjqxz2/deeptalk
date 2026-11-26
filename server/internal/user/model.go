// Package user 负责账号、密码、会话与额度。
// 数据访问函数统一以 dal 前缀命名，业务规则集中在 Service，其他模块只调用导出方法。
package user

import (
	"context"
	"errors"
	"strings"
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
	ID               uint       `gorm:"primaryKey"`                                      // 主键
	Email            string     `gorm:"size:190;uniqueIndex;not null"`                   // 邮箱（小写唯一）
	Nickname         string     `gorm:"size:64;not null"`                                // 昵称
	PasswordHash     string     `gorm:"size:255;not null"`                               // Argon2id 哈希串
	Role             string     `gorm:"size:16;not null;default:'user';index"`           // admin / user
	Plan             string     `gorm:"size:16;not null;default:'free'"`                 // free / pro / team / unlimited
	QuotaTokens      int64      `gorm:"not null;default:4000000"`                        // 本月额度（tokens）
	UsedTokens       int64      `gorm:"not null;default:0"`                              // 本月已用（tokens）
	Disabled         bool       `gorm:"not null;default:false;index"`                    // 是否限制登录
	FailedAttempts   int        `gorm:"not null;default:0"`                              // 连续登录失败次数
	LockedUntil      *time.Time `gorm:"index"`                                           // 锁定截止时间
	LastLoginAt      *time.Time `gorm:"index"`                                           // 最近登录时间
	LastLoginIP      string     `gorm:"size:64;not null;default:''"`                     // 最近登录 IP
	LastLoginCity    string     `gorm:"size:32;not null;default:''"`                     // 最近登录城市（粗粒度）
	WeChatOpenID     *string    `gorm:"column:wechat_open_id;size:64;uniqueIndex"`       // 小程序 openid，用于内容安全
	EmailPlaceholder bool       `gorm:"column:email_placeholder;not null;default:false"` // 邮箱是系统占位，尚未由用户补全
	AvatarURL        string     `gorm:"column:avatar_url;size:512;not null;default:''"`  // 头像地址
	CreatedAt        time.Time  `gorm:"index"`                                           // 注册时间
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
	SignKeyID string     `gorm:"size:64;not null;default:''"`  // 请求签名密钥标识（密钥由主密钥派生）
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

// Migrate 创建账号与会话表。
func Migrate(db *gorm.DB) error {
	// 历史版本 GORM 把 WeChatOpenID 拆成了 we_chat_open_id，
	// 这里把数据搬到规范列名 wechat_open_id 上，避免登录时找不到列。
	if err := migrateLegacyWeChatColumn(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(&User{}, &Session{}); err != nil {
		return err
	}
	return ensureInitialAdmin(db)
}

// 初始管理员账号：只在库里还没有这个账号时写入一次。
// 放在迁移里而不是配置或 seed 代码里，避免账号密码散落在仓库文件中。
const (
	initialAdminEmail    = "admin"
	initialAdminPassword = "123456"
)

// ensureInitialAdmin 只在「空库首次部署」时写入初始管理员。
// 判断依据是 users 表完全没有数据：只要有任意账号（包括已有的生产数据），
// 就完全不动这张表，绝不会覆盖或新增用户。
func ensureInitialAdmin(db *gorm.DB) error {
	var count int64
	if err := db.Model(&User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := HashPassword(initialAdminPassword)
	if err != nil {
		return err
	}
	admin := User{
		Email:        initialAdminEmail,
		Nickname:     "admin",
		PasswordHash: hash,
		Role:         RoleAdmin,
		Plan:         PlanFree,
		QuotaTokens:  planQuota(PlanFree),
	}
	return db.Create(&admin).Error
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

// migrateLegacyWeChatColumn 兼容旧列名 we_chat_open_id：存在则回填后删除。
func migrateLegacyWeChatColumn(db *gorm.DB) error {
	type column struct {
		Name string
	}
	var columns []column
	if err := db.Raw("PRAGMA table_info(users)").Scan(&columns).Error; err != nil {
		return nil // 表还不存在，交给 AutoMigrate
	}
	has := func(name string) bool {
		for _, item := range columns {
			if item.Name == name {
				return true
			}
		}
		return false
	}
	if !has("we_chat_open_id") || !has("wechat_open_id") {
		return nil
	}
	if err := db.Exec("UPDATE users SET wechat_open_id = we_chat_open_id WHERE (wechat_open_id IS NULL OR wechat_open_id = '') AND we_chat_open_id IS NOT NULL").Error; err != nil {
		return err
	}
	_ = db.Exec("ALTER TABLE users DROP COLUMN we_chat_open_id").Error
	return nil
}

// dalUpdateWeChatOpenID 绑定小程序 openid。
func dalUpdateWeChatOpenID(ctx context.Context, db *gorm.DB, userID uint, openid string) error {
	return db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).
		Update("wechat_open_id", openid).Error
}

// dalSelectWeChatOpenID 读取用户的小程序 openid；未绑定时返回 nil。
func dalSelectWeChatOpenID(ctx context.Context, db *gorm.DB, userID uint) (*string, error) {
	var row struct{ WeChatOpenID *string }
	if err := db.WithContext(ctx).Model(&User{}).Select("wechat_open_id").
		Where("id = ?", userID).Scan(&row).Error; err != nil {
		return nil, err
	}
	return row.WeChatOpenID, nil
}

// dalCountAdmins 统计管理员数量（判断是否为最后一个管理员）。
func dalCountAdmins(ctx context.Context, db *gorm.DB) (int64, error) {
	var total int64
	err := db.WithContext(ctx).Model(&User{}).Where("role = ?", RoleAdmin).Count(&total).Error
	return total, err
}

// dalUpdateRole 更新用户角色。
func dalUpdateRole(ctx context.Context, db *gorm.DB, userID uint, role string) error {
	return db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Update("role", role).Error
}

// dalCountUsersWhere 按条件统计用户数。
func dalCountUsersWhere(ctx context.Context, db *gorm.DB, where string, args ...any) (int64, error) {
	query := db.WithContext(ctx).Model(&User{})
	if where != "" {
		query = query.Where(where, args...)
	}
	var total int64
	err := query.Count(&total).Error
	return total, err
}

// dalSumUsedTokens 汇总全部用户已用 token。
func dalSumUsedTokens(ctx context.Context, db *gorm.DB) (int64, error) {
	var row struct{ Total int64 }
	if err := db.WithContext(ctx).Model(&User{}).
		Select("COALESCE(SUM(used_tokens), 0) AS total").Scan(&row).Error; err != nil {
		return 0, err
	}
	return row.Total, nil
}

// dalListUsers 按条件分页查询用户；返回列表与总数。
func dalListUsers(ctx context.Context, db *gorm.DB, p ListParams) ([]User, int64, error) {
	filter := func(query *gorm.DB) *gorm.DB {
		if q := strings.TrimSpace(p.Query); q != "" {
			like := "%" + q + "%"
			query = query.Where("email LIKE ? OR nickname LIKE ?", like, like)
		}
		switch p.Status {
		case "normal":
			query = query.Where("disabled = ?", false)
		case "restricted":
			query = query.Where("disabled = ?", true)
		case "quota":
			query = query.Where("disabled = ? AND quota_tokens > 0 AND used_tokens * 10 >= quota_tokens * 9", false)
		}
		return query
	}
	var total int64
	if err := filter(db.WithContext(ctx).Model(&User{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	query := filter(db.WithContext(ctx).Model(&User{}))
	switch p.Sort {
	case "created":
		query = query.Order("created_at DESC, id DESC")
	case "usage":
		query = query.Order("used_tokens DESC, id ASC")
	default:
		// 最近活跃优先，从未登录的排最后
		query = query.Order("(last_login_at IS NULL) ASC, last_login_at DESC, id ASC")
	}
	var items []User
	err := query.Offset((p.Page - 1) * p.PageSize).Limit(p.PageSize).Find(&items).Error
	return items, total, err
}

// dalListAllUsers 返回全部用户（CSV 导出用）。
func dalListAllUsers(ctx context.Context, db *gorm.DB) ([]User, error) {
	var items []User
	err := db.WithContext(ctx).Order("id ASC").Find(&items).Error
	return items, err
}

// dalAddUsedTokens 累加用户已用 token。
func dalAddUsedTokens(ctx context.Context, db *gorm.DB, userID uint, tokens int64) error {
	return db.WithContext(ctx).Model(&User{}).
		Where("id = ?", userID).
		UpdateColumn("used_tokens", gorm.Expr("used_tokens + ?", tokens)).Error
}

// dalFindSessionByTokenHash 按 token 摘要读取未撤销的会话。
func dalFindSessionByTokenHash(ctx context.Context, db *gorm.DB, tokenHash string) (*Session, error) {
	var session Session
	if err := db.WithContext(ctx).Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// dalUpdateSessionSignKey 给会话补发签名 key id。
func dalUpdateSessionSignKey(ctx context.Context, db *gorm.DB, sessionID uint, keyID string) error {
	return db.WithContext(ctx).Model(&Session{}).Where("id = ?", sessionID).
		Update("sign_key_id", keyID).Error
}

// dalDeleteUserSessions 删除用户的全部会话记录。
func dalDeleteUserSessions(ctx context.Context, db *gorm.DB, userID uint) error {
	return db.WithContext(ctx).Where("user_id = ?", userID).Delete(&Session{}).Error
}

// dalDeleteUser 删除用户记录。
func dalDeleteUser(ctx context.Context, db *gorm.DB, userID uint) error {
	return db.WithContext(ctx).Delete(&User{}, userID).Error
}

// dalRecordLoginFailure 原子累加登录失败次数，达到阈值时直接锁定账号。
// 用单条 UPDATE 完成「自增 + 判定 + 锁定」，避免读改写竞态导致并发爆破绕过阈值。
// 返回本次之后账号是否处于锁定状态。
func dalRecordLoginFailure(ctx context.Context, db *gorm.DB, userID uint, threshold int, until time.Time) (bool, error) {
	var locked bool
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// UPDATE 的所有赋值都读取旧行值，两个 CASE 的判定条件因此保持一致
		if err := tx.Exec(`UPDATE users SET
			failed_attempts = CASE WHEN failed_attempts + 1 >= ? THEN 0 ELSE failed_attempts + 1 END,
			locked_until = CASE WHEN failed_attempts + 1 >= ? THEN ? ELSE locked_until END
			WHERE id = ?`, threshold, threshold, until, userID).Error; err != nil {
			return err
		}
		var row struct{ LockedUntil *time.Time }
		if err := tx.Model(&User{}).Select("locked_until").Where("id = ?", userID).Scan(&row).Error; err != nil {
			return err
		}
		locked = row.LockedUntil != nil && row.LockedUntil.After(time.Now())
		return nil
	})
	return locked, err
}

// LoginSuccess 是一次成功登录要落库的信息。
type LoginSuccess struct {
	At   time.Time // 登录时间
	IP   string    // 来源 IP
	City string    // 归属地
}

// dalRecordLoginSuccess 只更新登录相关字段。
// 不用 Save：整行保存会把并发累加中的 used_tokens 覆盖回旧值。
func dalRecordLoginSuccess(ctx context.Context, db *gorm.DB, userID uint, in LoginSuccess) error {
	return db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(map[string]any{
		"failed_attempts": 0,
		"locked_until":    nil,
		"last_login_at":   in.At,
		"last_login_ip":   in.IP,
		"last_login_city": in.City,
	}).Error
}
