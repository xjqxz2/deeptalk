package responses

import (
	"strings"
	"time"

	"deeptalk/internal/user"
)

type UserResponse struct {
	ID                uint       `json:"id"`                 // 用户 ID
	Email             string     `json:"email"`              // 邮箱
	Nickname          string     `json:"nickname"`           // 昵称
	Role              string     `json:"role"`               // admin / user
	Plan              string     `json:"plan"`               // free / pro / team
	QuotaTokens       int64      `json:"quota_tokens"`       // 本月额度
	UsedTokens        int64      `json:"used_tokens"`        // 本月已用
	UsedPercent       int        `json:"used_percent"`       // 使用百分比
	Disabled          bool       `json:"disabled"`           // 是否限制登录
	CreatedAt         time.Time  `json:"created_at"`         // 注册时间
	LastLoginAt       *time.Time `json:"last_login_at"`      // 最近登录
	LastLoginCity     string     `json:"last_login_city"`    // 最近登录城市
	LastLoginIP       string     `json:"last_login_ip"`      // 最近登录 IP
	AvatarText        string     `json:"avatar_text"`        // 头像文字
	AvatarURL         string     `json:"avatar_url"`         // 头像地址（小程序上传）
	DisplayName       string     `json:"display_name"`       // 展示名：昵称优先，否则邮箱
	EmailPlaceholder  bool       `json:"email_placeholder"`  // 邮箱是否为系统占位（待补全）
	ConversationCount int64      `json:"conversation_count"` // 会话数（后台列表）
	MessageCount      int64      `json:"message_count"`      // 消息数（后台详情）
	QuotaResetDays    int        `json:"quota_reset_days"`   // 距额度重置天数（后台详情）
}

// displayName 返回展示名：有昵称用昵称，否则回退邮箱。
func displayName(u *user.User) string {
	if u == nil {
		return ""
	}
	if strings.TrimSpace(u.Nickname) != "" {
		return u.Nickname
	}
	return u.Email
}

// NewUser 构造账号响应。
func NewUser(u *user.User) UserResponse {
	if u == nil {
		return UserResponse{}
	}
	return UserResponse{
		ID:               u.ID,
		Email:            u.Email,
		Nickname:         u.Nickname,
		Role:             u.Role,
		Plan:             u.Plan,
		QuotaTokens:      u.QuotaTokens,
		UsedTokens:       u.UsedTokens,
		UsedPercent:      percent(u.UsedTokens, u.QuotaTokens),
		Disabled:         u.Disabled,
		CreatedAt:        u.CreatedAt,
		LastLoginAt:      u.LastLoginAt,
		LastLoginCity:    u.LastLoginCity,
		LastLoginIP:      u.LastLoginIP,
		AvatarText:       avatar(displayName(u)),
		AvatarURL:        u.AvatarURL,
		DisplayName:      displayName(u),
		EmailPlaceholder: u.IsPlaceholderEmail(),
	}
}

// ModelResponse 是 C 端模型选择项。
func avatar(nickname string) string {
	runes := []rune(nickname)
	if len(runes) == 0 {
		return "U"
	}
	return string(runes[0])
}

// UsageResponse 保持扁平字段（兼容旧客户端），并附带套餐限额。
