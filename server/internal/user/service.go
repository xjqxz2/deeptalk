package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"time"

	"deeptalk/internal/audit"
	"deeptalk/internal/geo"
	"deeptalk/internal/security"

	"gorm.io/gorm"
)

// RegistrationPolicy 是本模块消费的“注册开关”能力，由 setting.Service 实现。
// 接口定义在消费方，避免 user 依赖具体配置实现。
type RegistrationPolicy interface {
	AllowRegistration(ctx context.Context) (bool, error)
}

// AuditRecorder 是本模块消费的审计能力，由 audit.Service 实现。
type AuditRecorder interface {
	Record(ctx context.Context, in audit.Record) error
}

// Options 是账号服务的策略参数，全部来自配置。
type Options struct {
	SessionTTL    time.Duration // 会话有效期
	LockThreshold int           // 连续失败锁定阈值
	LockMinutes   int           // 锁定时长（分钟）
	DefaultQuota  int64         // 新用户默认额度
	Geo           *geo.Service  // IP 定位；为空时城市记为「未知」
}

// RequestMeta 记录会话签发时的客户端信息，用于安全审计。
type RequestMeta struct {
	IP        string // 客户端 IP
	UserAgent string // 客户端 UA
}

// RegisterInput 是注册所需的输入。
type RegisterInput struct {
	Email    string // 邮箱
	Password string // 明文密码（仅内存使用）
}

// AdminCreateInput 是管理员建号输入。
type AdminCreateInput struct {
	Email       string // 邮箱
	Password    string // 初始密码
	Nickname    string // 昵称，可空，默认取邮箱前缀
	Plan        string // 套餐
	QuotaTokens int64  // 额度，0 表示按套餐默认
}

// Service 提供账号、会话与额度相关业务。
type Service struct {
	geo    *geo.Service       // IP 定位
	db     *gorm.DB           // 数据库句柄
	policy RegistrationPolicy // 注册开关
	audit  AuditRecorder      // 审计记录器，可为 nil
	opt    Options            // 策略参数
}

// NewService 创建账号服务。
func NewService(db *gorm.DB, policy RegistrationPolicy, recorder AuditRecorder, opt Options) *Service {
	if opt.SessionTTL <= 0 {
		opt.SessionTTL = 7 * 24 * time.Hour
	}
	if opt.LockThreshold <= 0 {
		opt.LockThreshold = 5
	}
	if opt.LockMinutes <= 0 {
		opt.LockMinutes = 10
	}
	if opt.DefaultQuota <= 0 {
		opt.DefaultQuota = 4_000_000
	}
	return &Service{db: db, policy: policy, audit: recorder, opt: opt, geo: opt.Geo}
}

// Register 完成注册并直接创建会话（注册即登录）。
// 管理员账号由迁移创建（admin），注册出来的账号一律是普通用户：
// 不能再让「第一个注册的人」拿到管理员，否则删掉管理员行就等于开了提权后门。
// 返回用户、会话 token；错误包括邮箱冲突、注册关闭、密码不合规。
func (s *Service) Register(ctx context.Context, in RegisterInput, meta RequestMeta) (*User, string, error) {
	email := NormalizeEmail(in.Email)
	if err := ValidateEmail(email); err != nil {
		return nil, "", err
	}
	if err := ValidatePassword(in.Password); err != nil {
		return nil, "", err
	}
	// 只有库里已经有账号时才检查注册开关（空库首次注册必须放行）
	count, err := dalCountUsers(ctx, s.db)
	if err != nil {
		return nil, "", err
	}
	if count > 0 && s.policy != nil {
		allowed, err := s.policy.AllowRegistration(ctx)
		if err != nil {
			return nil, "", err
		}
		if !allowed {
			return nil, "", ErrRegistrationClosed
		}
	}
	if _, err := dalFindByEmail(ctx, s.db, email); err == nil {
		return nil, "", ErrEmailTaken
	} else if !errors.Is(err, ErrNotFound) {
		return nil, "", err
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		return nil, "", err
	}
	u := &User{
		Email:        email,
		Nickname:     nicknameFromEmail(email),
		PasswordHash: hash,
		Role:         RoleUser,
		Plan:         PlanFree,
		QuotaTokens:  s.opt.DefaultQuota,
	}
	token := ""
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dalCreateUser(ctx, tx, u); err != nil {
			// 唯一索引冲突仍按邮箱已占用处理，覆盖并发注册
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return ErrEmailTaken
			}
			return err
		}
		var err error
		token, err = s.issueSession(ctx, tx, u.ID, meta)
		return err
	})
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}

// Login 校验邮箱密码并签发会话。
// 统一错误提示不区分“邮箱不存在”和“密码错误”；连续失败达到阈值后锁定。
func (s *Service) Login(ctx context.Context, email, password string, meta RequestMeta) (*User, string, error) {
	email = NormalizeEmail(email)
	u, err := dalFindByEmail(ctx, s.db, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
	}
	now := time.Now()
	// 锁定期间直接返回锁定错误，避免继续消耗哈希计算
	if u.LockedUntil != nil && u.LockedUntil.After(now) {
		return nil, "", ErrLocked
	}
	if u.Disabled {
		return nil, "", ErrDisabled
	}
	if !VerifyPassword(u.PasswordHash, password) {
		until := now.Add(time.Duration(s.opt.LockMinutes) * time.Minute)
		locked, lockErr := dalRecordLoginFailure(ctx, s.db, u.ID, s.opt.LockThreshold, until)
		if lockErr != nil {
			return nil, "", lockErr
		}
		if locked {
			return nil, "", ErrLocked
		}
		return nil, "", ErrInvalidCredentials
	}
	// 登录成功：清零失败计数并刷新最近登录信息（只写这几列，避免覆盖并发累加的用量）
	u.FailedAttempts = 0
	u.LockedUntil = nil
	u.LastLoginAt = &now
	u.LastLoginIP = meta.IP
	u.LastLoginCity = s.cityFromIP(meta.IP)
	var token string
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dalRecordLoginSuccess(ctx, tx, u.ID, LoginSuccess{
			At:   now,
			IP:   u.LastLoginIP,
			City: u.LastLoginCity,
		}); err != nil {
			return err
		}
		var err error
		token, err = s.issueSession(ctx, tx, u.ID, meta)
		return err
	})
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}

// Authenticate 校验会话 token 并返回用户。
// 任何无效、过期、撤销、用户被限制的情况都返回 ErrUnauthenticated 或 ErrDisabled。
func (s *Service) Authenticate(ctx context.Context, token string) (*User, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrUnauthenticated
	}
	sum := sha256Hex(token)
	session, err := dalFindSession(ctx, s.db, sum)
	if err != nil {
		return nil, err
	}
	u, err := dalFindByID(ctx, s.db, session.UserID)
	if err != nil {
		return nil, ErrUnauthenticated
	}
	if u.Disabled {
		return nil, ErrDisabled
	}
	return u, nil
}

// SetWeChatOpenID 绑定小程序 openid；同一个 openid 不能绑定到多个账号。
func (s *Service) SetWeChatOpenID(ctx context.Context, userID uint, openid string) error {
	openid = strings.TrimSpace(openid)
	if openid == "" {
		return errors.New("wechat openid is required")
	}
	existing, err := dalFindByWeChatOpenID(ctx, s.db, openid)
	if err == nil && existing.ID != userID {
		return ErrForbidden
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	return dalUpdateWeChatOpenID(ctx, s.db, userID, openid)
}

// WeChatOpenID 返回用户绑定的小程序 openid；未绑定时返回空串。
func (s *Service) WeChatOpenID(ctx context.Context, userID uint) string {
	openid, err := dalSelectWeChatOpenID(ctx, s.db, userID)
	if err != nil || openid == nil {
		return ""
	}
	return *openid
}

// Logout 撤销当前会话。
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return dalRevokeSession(ctx, s.db, sha256Hex(token))
}

// ChangePassword 修改密码并撤销该用户全部会话（含当前会话）。
// 旧密码错误或新密码不合规时返回对应错误。
func (s *Service) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	u, err := dalFindByID(ctx, s.db, userID)
	if err != nil {
		return err
	}
	if !VerifyPassword(u.PasswordHash, oldPassword) {
		return ErrInvalidCredentials
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dalUpdateUser(ctx, tx, u); err != nil {
			return err
		}
		return dalRevokeUserSessions(ctx, tx, userID)
	})
}

// Usage 计算用量卡片所需数据。
func (s *Service) Usage(u *User) Usage {
	return Usage{
		UsedTokens:  u.UsedTokens,
		QuotaTokens: u.QuotaTokens,
		UsedPercent: quotaPercent(u.UsedTokens, u.QuotaTokens),
		ResetDays:   resetDays(time.Now()),
		Plan:        u.Plan,
	}
}

// ByID 按主键查询用户。
func (s *Service) ByID(ctx context.Context, id uint) (*User, error) {
	return dalFindByID(ctx, s.db, id)
}

// Count 返回账号总数，用于判断是否已有用户（首个账号为管理员）与注册页提示。
func (s *Service) Count(ctx context.Context) (int64, error) {
	return dalCountUsers(ctx, s.db)
}

// Stats 汇总后台用户统计口径。统计全部来自数据库，保证列表与卡片一致。
// 注意：GORM 的条件链会复用 Statement，因此每个统计项都必须从新的 Session 开始。
func (s *Service) Stats(ctx context.Context) (Stats, error) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	sevenDaysAgo := now.AddDate(0, 0, -7)
	var stats Stats
	var err error
	if stats.Total, err = dalCountUsersWhere(ctx, s.db, ""); err != nil {
		return stats, err
	}
	if stats.NewThisMonth, err = dalCountUsersWhere(ctx, s.db, "created_at >= ?", monthStart); err != nil {
		return stats, err
	}
	if stats.Active7D, err = dalCountUsersWhere(ctx, s.db, "last_login_at >= ?", sevenDaysAgo); err != nil {
		return stats, err
	}
	if stats.Restricted, err = dalCountUsersWhere(ctx, s.db, "disabled = ?", true); err != nil {
		return stats, err
	}
	// 额度将尽：使用率 ≥90% 且未被限制；SQL 中先乘后比避免浮点误差
	if stats.QuotaAlert, err = dalCountUsersWhere(ctx, s.db,
		"disabled = ? AND quota_tokens > 0 AND used_tokens * 10 >= quota_tokens * 9", false); err != nil {
		return stats, err
	}
	if stats.MonthlyTokens, err = dalSumUsedTokens(ctx, s.db); err != nil {
		return stats, err
	}
	if stats.Total > 0 {
		stats.ActiveRate = int((stats.Active7D*100 + stats.Total/2) / stats.Total)
	}
	// 环比为演示口径；真实环比需要历史快照表，P1 再补
	stats.MonthlyGrowth = 18
	return stats, nil
}

// List 按条件分页查询用户列表。
// 说明：统计总数与取列表使用独立查询链，避免 GORM 复用 Statement 造成条件污染。
func (s *Service) List(ctx context.Context, p ListParams) ([]User, int64, error) {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 8
	}
	if p.PageSize > 50 {
		p.PageSize = 50
	}
	return dalListUsers(ctx, s.db, p)
}

// ExportAll 返回全部账号，供 CSV 导出使用；只包含账号字段，不含对话正文。
func (s *Service) ExportAll(ctx context.Context) ([]User, error) {
	return dalListAllUsers(ctx, s.db)
}

// UpdateQuota 调整用户额度。
// 额度必须不小于已用量，范围 1M–100M，避免出现负数或异常大值。
func (s *Service) UpdateQuota(ctx context.Context, userID uint, quota int64) (*User, error) {
	u, err := dalFindByID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	if u.Plan == PlanUnlimited {
		return nil, errors.New("无限版无需单独调整额度")
	}
	if quota < u.UsedTokens || quota < 1_000_000 || quota > 100_000_000 {
		return nil, errors.New("invalid quota")
	}
	u.QuotaTokens = quota
	if err := dalUpdateUser(ctx, s.db, u); err != nil {
		return nil, err
	}
	return u, nil
}

// UpdatePlan 调整套餐，并按套餐默认值重置额度。
// 说明：已用量不做截断，若已超过新额度，后台会显示“额度将尽 / 已用超额”。
func (s *Service) UpdatePlan(ctx context.Context, userID uint, plan string) (*User, error) {
	u, err := dalFindByID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	if plan != PlanFree && plan != PlanPro && plan != PlanTeam && plan != PlanUnlimited {
		return nil, errors.New("invalid plan")
	}
	u.Plan = plan
	u.QuotaTokens = planQuota(plan)
	// 无限版固定 quota=0；其余套餐若已超额，至少保留到当前用量之上
	if plan != PlanUnlimited && u.QuotaTokens < u.UsedTokens {
		u.QuotaTokens = u.UsedTokens + 1_000_000
	}
	if err := dalUpdateUser(ctx, s.db, u); err != nil {
		return nil, err
	}
	return u, nil
}

// SetDisabled 限制或恢复登录。
// 管理员账号不可被限制；限制时立即撤销全部会话，恢复后需重新登录。
func (s *Service) SetDisabled(ctx context.Context, userID uint, disabled bool) (*User, error) {
	u, err := dalFindByID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	if u.Role == RoleAdmin {
		return nil, ErrForbidden
	}
	u.Disabled = disabled
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dalUpdateUser(ctx, tx, u); err != nil {
			return err
		}
		if disabled {
			return dalRevokeUserSessions(ctx, tx, userID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

// SetRole 调整用户角色（设为 / 取消管理员）。
// 保护规则：不能取消最后一个管理员，也不能取消自己，避免把后台锁死。
func (s *Service) SetRole(ctx context.Context, operatorID, userID uint, role string) (*User, error) {
	if role != RoleAdmin && role != RoleUser {
		return nil, errors.New("invalid role")
	}
	u, err := dalFindByID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	if u.Role == role {
		return u, nil
	}
	if role == RoleUser {
		if userID == operatorID {
			return nil, errors.New("不能取消自己的管理员权限")
		}
		total, err := dalCountAdmins(ctx, s.db)
		if err != nil {
			return nil, err
		}
		if total <= 1 {
			return nil, errors.New("至少要保留一个管理员")
		}
	}
	if err := dalUpdateRole(ctx, s.db, userID, role); err != nil {
		return nil, err
	}
	u.Role = role
	return u, nil
}

// ResetPassword 生成一次性强密码、更新哈希并撤销全部会话。
// 返回的明文密码只在此处返回一次，服务端不保存。
func (s *Service) ResetPassword(ctx context.Context, userID uint) (string, error) {
	u, err := dalFindByID(ctx, s.db, userID)
	if err != nil {
		return "", err
	}
	password, err := randomPassword()
	if err != nil {
		return "", err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return "", err
	}
	u.PasswordHash = hash
	// 管理员重置密码同时解除登录锁定，否则用户仍会被 10 分钟锁定挡住
	u.FailedAttempts = 0
	u.LockedUntil = nil
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dalUpdateUser(ctx, tx, u); err != nil {
			return err
		}
		return dalRevokeUserSessions(ctx, tx, userID)
	})
	if err != nil {
		return "", err
	}
	return password, nil
}

// RevokeAllSessions 强制退出用户全部设备。
func (s *Service) RevokeAllSessions(ctx context.Context, userID uint) error {
	return dalRevokeUserSessions(ctx, s.db, userID)
}

// CreateByAdmin 在“注册关闭”时由管理员创建账号。
// 与自助注册的区别：套餐与额度可指定，角色固定为普通用户。
func (s *Service) CreateByAdmin(ctx context.Context, in AdminCreateInput) (*User, error) {
	email := NormalizeEmail(in.Email)
	if err := ValidateEmail(email); err != nil {
		return nil, err
	}
	if err := ValidatePassword(in.Password); err != nil {
		return nil, err
	}
	if _, err := dalFindByEmail(ctx, s.db, email); err == nil {
		return nil, ErrEmailTaken
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	plan := in.Plan
	if plan != PlanPro && plan != PlanTeam && plan != PlanUnlimited {
		plan = PlanFree
	}
	quota := in.QuotaTokens
	if plan == PlanUnlimited {
		quota = 0
	} else if quota <= 0 {
		quota = planQuota(plan)
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	nickname := strings.TrimSpace(in.Nickname)
	if nickname == "" {
		nickname = nicknameFromEmail(email)
	}
	u := &User{
		Email:        email,
		Nickname:     nickname,
		PasswordHash: hash,
		Role:         RoleUser,
		Plan:         plan,
		QuotaTokens:  quota,
	}
	if err := dalCreateUser(ctx, s.db, u); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return u, nil
}

// ConsumeUsage 累加用户本月 token 用量，用于额度校验与用量卡片。
func (s *Service) ConsumeUsage(ctx context.Context, userID uint, tokens int64) error {
	if tokens <= 0 {
		return nil
	}
	return dalAddUsedTokens(ctx, s.db, userID, tokens)
}

// issueSession 生成 token 并写入会话表。
func (s *Service) issueSession(ctx context.Context, tx *gorm.DB, userID uint, meta RequestMeta) (string, error) {
	token, tokenHash, err := newSessionToken()
	if err != nil {
		return "", err
	}
	session := &Session{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(s.opt.SessionTTL),
		IP:        meta.IP,
		UserAgent: truncate(meta.UserAgent, 255),
		SignKeyID: security.NewKeyID(),
	}
	if err := dalCreateSession(ctx, tx, session); err != nil {
		return "", err
	}
	return token, nil
}

// sha256Hex 计算 token 的 SHA-256 十六进制串。
func sha256Hex(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// randomPassword 生成 12 位一次性密码：固定前缀 Dt- 后接字母与数字，并强制同时包含两类字符。
func randomPassword() (string, error) {
	const letters = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"
	const digits = "23456789"
	raw := make([]byte, 0, 10)
	// 先各取 5 位，保证字母和数字都存在
	for i := 0; i < 5; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		raw = append(raw, letters[idx.Int64()])
	}
	for i := 0; i < 5; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		raw = append(raw, digits[idx.Int64()])
	}
	// Fisher-Yates 洗牌，避免固定“字母在前”的规律
	for i := len(raw) - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", err
		}
		j := int(jBig.Int64())
		raw[i], raw[j] = raw[j], raw[i]
	}
	return "Dt-" + string(raw), nil
}

// cityFromIP 将登录 IP 解析为城市名；私网或查询失败时记为「本地」。
// 说明：MVP 不接入 IP 库；本机/内网显示“本地”，其余显示“未知”，避免伪造地理位置。
func (s *Service) cityFromIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" || ip == "::1" || ip == "127.0.0.1" || strings.HasPrefix(ip, "192.168.") ||
		strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "172.16.") {
		return "本地"
	}
	if s.geo == nil {
		return "未知"
	}
	location, ok := s.geo.Lookup(context.Background(), ip)
	if !ok {
		return "未知"
	}
	return location.Label()
}

// truncate 按字符数截断字符串，防止超长 UA 写库失败。
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

// WeChatPlaceholderDomain 是小程序自动创建账号时使用的占位邮箱域名。
const WeChatPlaceholderDomain = "wechat.local"

// IsPlaceholderEmail 判断邮箱是否是系统占位（小程序自动建号，尚未补全）。
func (u *User) IsPlaceholderEmail() bool {
	return u != nil && (u.EmailPlaceholder || strings.HasSuffix(u.Email, "@"+WeChatPlaceholderDomain))
}

// LoginByOpenID 用小程序 openid 登录；不存在时自动建号。
// 新账号使用占位邮箱与随机密码，用户可在小程序内补全邮箱密码后用于 PC 端登录。
func (s *Service) LoginByOpenID(ctx context.Context, openid, nickname string, meta RequestMeta) (*User, string, error) {
	openid = strings.TrimSpace(openid)
	if openid == "" {
		return nil, "", errors.New("wechat openid is required")
	}
	u, err := dalFindByWeChatOpenID(ctx, s.db, openid)
	if err == nil {
		if u.Disabled {
			return nil, "", ErrDisabled
		}
		now := time.Now()
		u.LastLoginAt = &now
		u.LastLoginIP = meta.IP
		u.LastLoginCity = s.cityFromIP(meta.IP)
		var token string
		if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := dalUpdateUser(ctx, tx, u); err != nil {
				return err
			}
			var issueErr error
			token, issueErr = s.issueSession(ctx, tx, u.ID, meta)
			return issueErr
		}); err != nil {
			return nil, "", err
		}
		return u, token, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, "", err
	}

	// 首次进入：建一个占位账号
	hash, err := HashPassword(randomSecret())
	if err != nil {
		return nil, "", err
	}
	// 不塞默认昵称：没有昵称时前端统一回退显示邮箱
	nickname = strings.TrimSpace(nickname)
	// 微信首次登录一律是普通用户：管理员只能由迁移创建或在后台授予，
	// 否则删掉管理员后第一个微信登录的人就能拿到后台权限。
	u = &User{
		Email:            placeholderEmail(openid),
		Nickname:         truncate(nickname, 24),
		PasswordHash:     hash,
		Role:             RoleUser,
		Plan:             PlanFree,
		QuotaTokens:      s.opt.DefaultQuota,
		WeChatOpenID:     &openid,
		EmailPlaceholder: true,
	}
	var token string
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dalCreateUser(ctx, tx, u); err != nil {
			// 并发首次登录：回读已创建的账号
			existing, findErr := dalFindByWeChatOpenID(ctx, tx, openid)
			if findErr != nil {
				return err
			}
			*u = *existing
			var issueErr error
			token, issueErr = s.issueSession(ctx, tx, u.ID, meta)
			return issueErr
		}
		var issueErr error
		token, issueErr = s.issueSession(ctx, tx, u.ID, meta)
		return issueErr
	})
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}

// CompleteProfile 补全邮箱与密码：小程序用户补全后即可在 PC / PWA 端登录。
func (s *Service) CompleteProfile(ctx context.Context, userID uint, email, password string) (*User, error) {
	email = NormalizeEmail(email)
	if err := ValidateEmail(email); err != nil {
		return nil, err
	}
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}
	u, err := s.ByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing, err := dalFindByEmail(ctx, s.db, email); err == nil && existing.ID != userID {
		return nil, ErrEmailTaken
	} else if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	u.Email = email
	u.PasswordHash = hash
	u.EmailPlaceholder = false
	u.FailedAttempts = 0
	u.LockedUntil = nil
	if err := dalUpdateUser(ctx, s.db, u); err != nil {
		return nil, err
	}
	return u, nil
}

// placeholderEmail 生成小程序占位邮箱；同一 openid 稳定可推导。
func placeholderEmail(openid string) string {
	sum := sha256.Sum256([]byte(openid))
	return "wx_" + hex.EncodeToString(sum[:8]) + "@" + WeChatPlaceholderDomain
}

// randomSecret 生成随机密码串（占位账号使用，用户补全后会被替换）。
func randomSecret() string {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "wx-" + time.Now().Format("20060102150405")
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// UpdateProfile 更新昵称与头像；两者都可单独更新，空值表示不改。
func (s *Service) UpdateProfile(ctx context.Context, userID uint, nickname, avatarURL string) (*User, error) {
	u, err := s.ByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	nickname = strings.TrimSpace(nickname)
	if nickname != "" {
		if len([]rune(nickname)) > 24 {
			return nil, errors.New("昵称最长 24 个字符")
		}
		u.Nickname = nickname
	}
	avatarURL = strings.TrimSpace(avatarURL)
	if avatarURL != "" {
		if len(avatarURL) > 512 {
			return nil, errors.New("头像地址过长")
		}
		u.AvatarURL = avatarURL
	}
	if err := dalUpdateUser(ctx, s.db, u); err != nil {
		return nil, err
	}
	return u, nil
}

// SessionSignKey 返回会话的签名 key id；老会话没有时补发一个。
func (s *Service) SessionSignKey(ctx context.Context, token string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", ErrUnauthenticated
	}
	sum := sha256Hex(token)
	session, err := dalFindSessionByTokenHash(ctx, s.db, sum)
	if err != nil {
		return "", ErrUnauthenticated
	}
	if session.SignKeyID != "" {
		return session.SignKeyID, nil
	}
	keyID := security.NewKeyID()
	if err := dalUpdateSessionSignKey(ctx, s.db, session.ID, keyID); err != nil {
		return "", err
	}
	return keyID, nil
}

// Delete 删除普通用户及其登录会话；会话与计费数据由上层按模块清理。
// 管理员账号不允许删除，避免把自己或最后一个管理员删掉。
func (s *Service) Delete(ctx context.Context, userID uint) error {
	u, err := dalFindByID(ctx, s.db, userID)
	if err != nil {
		return err
	}
	if u.Role == RoleAdmin {
		return ErrForbidden
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dalDeleteUserSessions(ctx, tx, userID); err != nil {
			return err
		}
		return dalDeleteUser(ctx, tx, userID)
	})
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
