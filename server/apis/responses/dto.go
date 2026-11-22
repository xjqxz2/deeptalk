package responses

import (
	"encoding/json"
	"strings"
	"time"

	"deeptalk/internal/attachment"
	"deeptalk/internal/audit"
	"deeptalk/internal/billing"
	"deeptalk/internal/chat"
	"deeptalk/internal/provider"
	"deeptalk/internal/search"
	"deeptalk/internal/user"
)

// UserResponse 是 C / B 端共用的账号展示结构。
// 注意：永远不包含密码哈希、上游 Key 或对话正文。
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
type ModelResponse struct {
	ID            uint   `json:"id"`             // 模型主键
	ModelID       string `json:"model_id"`       // 上游模型 ID
	DisplayName   string `json:"display_name"`   // 展示名
	Description   string `json:"description"`    // 用途说明
	ProviderName  string `json:"provider_name"`  // 厂商名
	ContextWindow int    `json:"context_window"` // 上下文窗口
	IsDefault     bool   `json:"is_default"`     // 平台默认
}

// NewModel 构造 C 端模型响应。
func NewModel(m provider.Model) ModelResponse {
	return ModelResponse{
		ID:            m.ID,
		ModelID:       m.ModelID,
		DisplayName:   m.DisplayName,
		Description:   m.Description,
		ProviderName:  m.ProviderName,
		ContextWindow: m.ContextWindow,
		IsDefault:     m.IsDefault,
	}
}

// ConversationResponse 是会话展示结构。
type ConversationResponse struct {
	ID           uint      `json:"id"`            // 会话内部 ID
	PublicID     string    `json:"public_id"`     // URL 公开 ID
	Title        string    `json:"title"`         // 标题
	ModelID      uint      `json:"model_id"`      // 最近模型
	UpdatedAt    time.Time `json:"updated_at"`    // 最近更新时间
	MessageCount int       `json:"message_count"` // 消息条数
}

// NewConversation 由列表项构造响应。
func NewConversation(item chat.Summary) ConversationResponse {
	return ConversationResponse{ID: item.ID, PublicID: item.PublicID, Title: item.Title, ModelID: item.ModelID, UpdatedAt: item.UpdatedAt, MessageCount: item.MessageCount}
}

// NewConversationDetail 由会话模型构造响应。
func NewConversationDetail(conv *chat.Conversation) ConversationResponse {
	if conv == nil {
		return ConversationResponse{}
	}
	return ConversationResponse{ID: conv.ID, PublicID: conv.PublicID, Title: conv.Title, ModelID: conv.ModelID, UpdatedAt: conv.UpdatedAt}
}

// MessageResponse 是消息展示结构。
type MessageResponse struct {
	ID           uint              `json:"id"`             // 消息 ID
	Role         string            `json:"role"`           // user / assistant
	Content      string            `json:"content"`        // 正文
	Reasoning    string            `json:"reasoning"`      // 深度思考
	Status       string            `json:"status"`         // complete / streaming / stopped / error
	ModelName    string            `json:"model_name"`     // 模型名
	InputTokens  int               `json:"input_tokens"`   // 输入 tokens
	OutputTokens int               `json:"output_tokens"`  // 输出 tokens
	DurationMS   int               `json:"duration_ms"`    // 生成耗时
	Feedback     string            `json:"feedback"`       // up / down
	Sources      []search.Result   `json:"sources"`        // 联网检索来源
	A2UI         []json.RawMessage `json:"a2ui,omitempty"` // A2UI v0.9 界面消息
	Attachments  []attachment.Info `json:"attachments"`    // 图片附件
	CreatedAt    time.Time         `json:"created_at"`     // 发送时间
}

// ShareResponse 是分享链接响应；前端用 token 拼出完整 URL。
type ShareResponse struct {
	Token     string    `json:"token"`      // 分享 token
	CreatedAt time.Time `json:"created_at"` // 创建时间
}

// NewShare 构造分享响应。
func NewShare(share chat.Share) ShareResponse {
	return ShareResponse{Token: share.Token, CreatedAt: share.CreatedAt}
}

// SharedConversationResponse 是公开分享页的只读数据。
type SharedConversationResponse struct {
	Title    string            `json:"title"`     // 会话标题
	Owner    string            `json:"owner"`     // 分享者昵称
	SharedAt time.Time         `json:"shared_at"` // 分享时间
	Messages []MessageResponse `json:"messages"`  // 全部消息
}

// NewSharedConversation 构造分享页响应；只暴露标题、昵称与消息。
func NewSharedConversation(shared chat.SharedConversation) SharedConversationResponse {
	out := make([]MessageResponse, 0, len(shared.Messages))
	for _, message := range shared.Messages {
		out = append(out, NewMessage(message))
	}
	return SharedConversationResponse{
		Title:    shared.Conversation.Title,
		Owner:    shared.Owner,
		SharedAt: shared.CreatedAt,
		Messages: out,
	}
}

// NewMessage 构造消息响应。
func NewMessage(m chat.Message) MessageResponse {
	return MessageResponse{
		ID:   m.ID,
		Role: m.Role,
		// 历史消息可能残留上游的 DSML 工具标记，展示前统一清洗
		Content:      chat.SanitizeDSML(m.Content),
		Reasoning:    m.Reasoning,
		Status:       m.Status,
		ModelName:    m.ModelName,
		InputTokens:  m.InputTokens,
		OutputTokens: m.OutputTokens,
		DurationMS:   m.DurationMS,
		Feedback:     m.Feedback,
		Sources:      decodeSources(m.SourcesJSON),
		A2UI:         decodeA2UI(m.UIJSON),
		Attachments:  decodeAttachments(m.AttachmentsJSON),
		CreatedAt:    m.CreatedAt,
	}
}

// decodeSources 解析消息里的联网来源；历史消息或损坏数据返回空数组。
func decodeSources(raw string) []search.Result {
	if raw == "" {
		return []search.Result{}
	}
	var items []search.Result
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []search.Result{}
	}
	return items
}

// decodeA2UI 解析消息里的 A2UI envelope；损坏数据不影响整条消息。
func decodeA2UI(raw string) []json.RawMessage {
	if raw == "" {
		return nil
	}
	var messages []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &messages); err != nil {
		return nil
	}
	return messages
}

// decodeAttachments 解析消息里的附件快照；历史消息或损坏数据返回空数组。
func decodeAttachments(raw string) []attachment.Info {
	if raw == "" {
		return []attachment.Info{}
	}
	var items []attachment.Info
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []attachment.Info{}
	}
	return items
}

// AdminModelResponse 是后台模型路由表行。
type AdminModelResponse struct {
	ID             uint   `json:"id"`              // 模型主键
	ProviderID     uint   `json:"provider_id"`     // 厂商 ID
	ProviderName   string `json:"provider_name"`   // 厂商名
	ModelID        string `json:"model_id"`        // 上游模型 ID
	DisplayName    string `json:"display_name"`    // 展示名
	Description    string `json:"description"`     // 用途说明
	ContextWindow  int    `json:"context_window"`  // 上下文窗口
	MaxConcurrency int    `json:"max_concurrency"` // 并发
	Enabled        bool   `json:"enabled"`         // 是否启用
	IsDefault      bool   `json:"is_default"`      // 默认模型
	IsFallback     bool   `json:"is_fallback"`     // 兜底模型
}

// NewAdminModel 构造后台模型响应。
func NewAdminModel(m provider.Model) AdminModelResponse {
	return AdminModelResponse{
		ID:             m.ID,
		ProviderID:     m.ProviderID,
		ProviderName:   m.ProviderName,
		ModelID:        m.ModelID,
		DisplayName:    m.DisplayName,
		Description:    m.Description,
		ContextWindow:  m.ContextWindow,
		MaxConcurrency: m.MaxConcurrency,
		Enabled:        m.Enabled,
		IsDefault:      m.IsDefault,
		IsFallback:     m.IsFallback,
	}
}

// ProviderResponse 是后台厂商卡片数据。
type ProviderResponse struct {
	ID             uint                 `json:"id"`              // 厂商 ID
	Name           string               `json:"name"`            // 名称
	Code           string               `json:"code"`            // 唯一标识
	Kind           string               `json:"kind"`            // 厂商类型
	APIStyle       string               `json:"api_style"`       // 接入方式
	BaseURL        string               `json:"base_url"`        // Base URL
	APIKeyMasked   string               `json:"api_key_masked"`  // Key 掩码
	HasAPIKey      bool                 `json:"has_api_key"`     // 是否已配置 Key
	TimeoutSec     int                  `json:"timeout_sec"`     // 超时
	MaxConcurrency int                  `json:"max_concurrency"` // 并发
	Enabled        bool                 `json:"enabled"`         // 是否启用
	Verified       bool                 `json:"verified"`        // 是否验证通过
	Status         string               `json:"status"`          // 状态胶囊
	IsBuiltin      bool                 `json:"is_builtin"`      // 预置厂商
	IsDefault      bool                 `json:"is_default"`      // 平台默认
	ModelCount     int                  `json:"model_count"`     // 模型数
	LastLatencyMS  int                  `json:"last_latency_ms"` // 最近延迟
	LastError      string               `json:"last_error"`      // 最近失败原因
	Headers        map[string]string    `json:"headers"`         // 自定义请求头
	Models         []AdminModelResponse `json:"models"`          // 模型列表
}

// NewProvider 构造后台厂商响应；status 由调用方传入，避免响应层依赖业务方法。
func NewProvider(p provider.Provider, status string) ProviderResponse {
	models := make([]AdminModelResponse, 0, len(p.Models))
	for _, m := range p.Models {
		m.ProviderName = p.Name
		models = append(models, NewAdminModel(m))
	}
	return ProviderResponse{
		ID:             p.ID,
		Name:           p.Name,
		Code:           p.Code,
		Kind:           p.Kind,
		APIStyle:       provider.NormalizeAPIStyle(p.APIStyle),
		BaseURL:        p.BaseURL,
		APIKeyMasked:   p.APIKeyMasked,
		HasAPIKey:      p.APIKeyEnc != "",
		TimeoutSec:     p.TimeoutSec,
		MaxConcurrency: p.MaxConcurrency,
		Enabled:        p.Enabled,
		Verified:       p.Verified,
		Status:         status,
		IsBuiltin:      p.IsBuiltin,
		IsDefault:      p.IsDefault,
		ModelCount:     len(p.Models),
		LastLatencyMS:  p.LastLatencyMS,
		LastError:      p.LastError,
		Headers:        provider.ProviderHeaders(&p),
		Models:         models,
	}
}

// AuditResponse 是最近变更条目。
type AuditResponse struct {
	ID        uint      `json:"id"`         // 记录 ID
	ActorName string    `json:"actor_name"` // 操作者
	Action    string    `json:"action"`     // 动作
	Detail    string    `json:"detail"`     // 摘要
	CreatedAt time.Time `json:"created_at"` // 时间
}

// NewAudit 构造审计条目响应。
func NewAudit(l audit.Log) AuditResponse {
	return AuditResponse{ID: l.ID, ActorName: l.ActorName, Action: l.Action, Detail: l.Detail, CreatedAt: l.CreatedAt}
}

// OverviewResponse 是后台概览数据。
type OverviewResponse struct {
	TotalUsers        int64           `json:"total_users"`        // 总用户
	NewThisMonth      int64           `json:"new_this_month"`     // 本月新增
	Active7D          int64           `json:"active_7d"`          // 7 日活跃
	ActiveRate        int             `json:"active_rate"`        // 周活跃率
	MonthlyTokens     int64           `json:"monthly_tokens"`     // 本月 tokens
	MonthlyGrowth     int             `json:"monthly_growth"`     // 环比
	Restricted        int64           `json:"restricted"`         // 已限制
	QuotaAlert        int64           `json:"quota_alert"`        // 额度将尽
	ProvidersEnabled  int64           `json:"providers_enabled"`  // 启用厂商
	ModelsEnabled     int64           `json:"models_enabled"`     // 启用模型
	ConversationCount int64           `json:"conversation_count"` // 平台会话总数
	MonthlyMessages   int64           `json:"monthly_messages"`   // 本月消息数
	Health            provider.Health `json:"health"`             // 接入健康
	RecentAudits      []AuditResponse `json:"recent_audits"`      // 最近变更
}

// SettingsResponse 是后台系统设置。
type SettingsResponse struct {
	AllowRegistration    bool   `json:"allow_registration"`     // 开放注册
	DefaultModelID       uint   `json:"default_model_id"`       // 默认模型
	FallbackModelID      uint   `json:"fallback_model_id"`      // 兜底模型
	MaxOutputTokens      int    `json:"max_output_tokens"`      // 单次输出预算
	SiteName             string `json:"site_name"`              // 站点名
	GenerationTimeoutSec int    `json:"generation_timeout_sec"` // 生成首字节超时（秒）
	Version              string `json:"version"`                // 版本
	StorageUsedBytes     int64  `json:"storage_used_bytes"`     // 存储已用
	StorageTotalBytes    int64  `json:"storage_total_bytes"`    // 存储上限
}

// percent 计算百分比并向上取整。
func percent(used, quota int64) int {
	if quota <= 0 || used <= 0 {
		return 0
	}
	p := int((used*100 + quota - 1) / quota)
	if p > 100 {
		return 100
	}
	return p
}

// avatar 取昵称首字符作为头像。
func avatar(nickname string) string {
	runes := []rune(nickname)
	if len(runes) == 0 {
		return "U"
	}
	return string(runes[0])
}

// UsageResponse 保持扁平字段（兼容旧客户端），并附带套餐限额。
type UsageResponse struct {
	UsedTokens  int64             `json:"used_tokens"`
	QuotaTokens int64             `json:"quota_tokens"`
	UsedPercent int               `json:"used_percent"`
	ResetDays   int               `json:"reset_days"`
	Plan        string            `json:"plan"`
	Limits      *billing.Snapshot `json:"limits,omitempty"`
}

// NewUsageResponse 组装用量与限额。
func NewUsage(usage UsageResponse, limits *billing.Snapshot) UsageResponse {
	usage.Limits = limits
	return usage
}
