// Package c 提供 C 端（普通用户）接口控制器。
package c

import (
	"encoding/hex"
	"errors"
	"strings"

	"deeptalk/apis/middleware"
	"deeptalk/apis/requests"
	"deeptalk/apis/responses"
	"deeptalk/internal/provider"
	"deeptalk/internal/security"
	"deeptalk/internal/setting"
	"deeptalk/internal/user"
	"deeptalk/internal/wechat"

	"github.com/gin-gonic/gin"
)

// AuthController 处理注册、登录、退出、当前用户与用量。
type AuthController struct {
	users         *user.Service      // 账号能力
	wechat        *wechat.Manager    // 小程序凭据（code2session）
	sign          *security.Verifier // 请求签名密钥派生
	providers     *provider.Service  // 模型能力
	settings      *setting.Service   // 平台设置
	cfg           middleware.Config  // Cookie 等配置
	searchEnabled bool               // 联网搜索是否可用
}

// NewAuthController 创建 C 端账号控制器。
func NewAuthController(users *user.Service, providers *provider.Service, settings *setting.Service, cfg middleware.Config, searchEnabled bool, wx *wechat.Manager, sign *security.Verifier) *AuthController {
	return &AuthController{users: users, wechat: wx, sign: sign, providers: providers, settings: settings, cfg: cfg, searchEnabled: searchEnabled}
}

// WeChatLogin 用小程序 wx.login 的 code 登录：openid 不存在时自动建号。
func (h *AuthController) WeChatLogin(c *gin.Context) {
	if h.wechat == nil || !h.wechat.Enables() {
		responses.Fail(c, responses.CodeInternal, "小程序凭据未配置")
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Code) == "" {
		responses.Fail(c, responses.CodeInvalidArgument, "缺少登录凭证")
		return
	}
	session, err := h.wechat.Code2Session(c.Request.Context(), req.Code)
	if err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "小程序授权失败，请重试")
		return
	}
	current, token, err := h.users.LoginByOpenID(c.Request.Context(), session.OpenID, "", user.RequestMeta{
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		responses.FromError(c, err)
		return
	}
	middleware.SetSessionCookie(c, h.cfg, token)
	payload := gin.H{
		"user":  responses.NewUser(current),
		"token": token,
	}
	h.attachSignKey(c, token, payload)
	responses.OK(c, payload)
}

// UpdateProfile 更新昵称与头像（小程序端在授权后调用）。
func (h *AuthController) UpdateProfile(c *gin.Context) {
	current := middleware.CurrentUser(c)
	var req struct {
		Nickname  string `json:"nickname"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请求内容不合法")
		return
	}
	updated, err := h.users.UpdateProfile(c.Request.Context(), current.ID, req.Nickname, req.AvatarURL)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, gin.H{"user": responses.NewUser(updated)})
}

// CompleteProfile 补全邮箱与密码：补全后可用邮箱在 PC / PWA 端登录。
func (h *AuthController) CompleteProfile(c *gin.Context) {
	current := middleware.CurrentUser(c)
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请求内容不合法")
		return
	}
	updated, err := h.users.CompleteProfile(c.Request.Context(), current.ID, req.Email, req.Password)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, gin.H{"user": responses.NewUser(updated)})
}

// BindWeChat 用小程序 wx.login 的 code 绑定 openid，内容安全检测需要它。
func (h *AuthController) BindWeChat(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if h.wechat == nil || !h.wechat.Enables() {
		responses.Fail(c, responses.CodeInternal, "小程序凭据未配置")
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Code) == "" {
		responses.Fail(c, responses.CodeInvalidArgument, "缺少登录凭证")
		return
	}
	session, err := h.wechat.Code2Session(c.Request.Context(), req.Code)
	if err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "小程序授权失败，请重试")
		return
	}
	if err := h.users.SetWeChatOpenID(c.Request.Context(), current.ID, session.OpenID); err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, gin.H{"bound": true})
}

// attachSignKey 把本次会话的签名密钥放进响应：密钥由服务端主密钥派生，
// 客户端只用于给写请求签名，服务端不存明文。
func (h *AuthController) attachSignKey(c *gin.Context, token string, payload gin.H) {
	if h.sign == nil || h.sign.Disabled() {
		return
	}
	keyID, err := h.users.SessionSignKey(c.Request.Context(), token)
	if err != nil || keyID == "" {
		return
	}
	payload["sign_key_id"] = keyID
	payload["sign_key"] = hex.EncodeToString(h.sign.DeriveKey(keyID))
}

// SignKey 重新下发当前会话的签名密钥（刷新页面后客户端内存丢失时用）。
func (h *AuthController) SignKey(c *gin.Context) {
	if h.sign == nil || h.sign.Disabled() {
		responses.OK(c, gin.H{})
		return
	}
	keyID, err := h.users.SessionSignKey(c.Request.Context(), middleware.SessionToken(c))
	if err != nil || keyID == "" {
		responses.Fail(c, responses.CodeUnauthenticated, "登录已失效，请重新登录")
		return
	}
	responses.OK(c, gin.H{
		"sign_key_id": keyID,
		"sign_key":    hex.EncodeToString(h.sign.DeriveKey(keyID)),
	})
}

// Config 返回注册开关等公开配置，供登录 / 注册页使用。
func (h *AuthController) Config(c *gin.Context) {
	allow, _ := h.settings.AllowRegistration(c.Request.Context())
	count, err := h.users.Count(c.Request.Context())
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, gin.H{
		"allow_registration": allow,
		"has_users":          count > 0,
		"site_name":          "小橘树洞",
		"web_search_enabled": h.searchEnabled,
	})
}

// Register 注册并直接登录。
func (h *AuthController) Register(c *gin.Context) {
	var req requests.RegisterRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请填写正确的邮箱与密码")
		return
	}
	u, token, err := h.users.Register(c.Request.Context(), user.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
	}, user.RequestMeta{IP: c.ClientIP(), UserAgent: c.Request.UserAgent()})
	if err != nil {
		responses.FromError(c, err)
		return
	}
	middleware.SetSessionCookie(c, h.cfg, token)
	// token 同时下发给小程序 / APP：它们无法可靠保存 Cookie
	responses.OK(c, gin.H{"user": responses.NewUser(u), "token": token})
}

// Login 校验账号密码并签发会话。
func (h *AuthController) Login(c *gin.Context) {
	var req requests.LoginRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请填写邮箱与密码")
		return
	}
	u, token, err := h.users.Login(c.Request.Context(), req.Email, req.Password,
		user.RequestMeta{IP: c.ClientIP(), UserAgent: c.Request.UserAgent()})
	if err != nil {
		responses.FromError(c, err)
		return
	}
	middleware.SetSessionCookie(c, h.cfg, token)
	payload := gin.H{"user": responses.NewUser(u), "token": token}
	h.attachSignKey(c, token, payload)
	responses.OK(c, payload)
}

// Logout 退出登录并撤销当前会话。
func (h *AuthController) Logout(c *gin.Context) {
	token, _ := c.Cookie(h.cfg.CookieName)
	if token != "" {
		_ = h.users.Logout(c.Request.Context(), token)
	}
	middleware.ClearSessionCookie(c, h.cfg)
	responses.OK(c, nil)
}

// Me 返回当前登录用户。
func (h *AuthController) Me(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if current == nil {
		responses.Fail(c, responses.CodeUnauthenticated, "登录已失效，请重新登录")
		return
	}
	responses.OK(c, gin.H{"user": responses.NewUser(current)})
}

// ChangePassword 修改密码；成功后撤销全部会话并清除 Cookie。
func (h *AuthController) ChangePassword(c *gin.Context) {
	current := middleware.CurrentUser(c)
	var req requests.ChangePasswordRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "新密码至少 8 位，且需同时包含字母与数字")
		return
	}
	if err := h.users.ChangePassword(c.Request.Context(), current.ID, req.OldPassword, req.NewPassword); err != nil {
		if errors.Is(err, user.ErrInvalidCredentials) {
			responses.Fail(c, responses.CodeInvalidArgument, "当前密码不正确")
			return
		}
		responses.FromError(c, err)
		return
	}
	middleware.ClearSessionCookie(c, h.cfg)
	responses.OK(c, nil)
}

// Usage 返回当前用户本月用量。
func (h *AuthController) Usage(c *gin.Context) {
	current := middleware.CurrentUser(c)
	responses.OK(c, h.users.Usage(current))
}

// Models 返回 C 端可选择的模型（仅启用项）。
func (h *AuthController) Models(c *gin.Context) {
	items, err := h.providers.EnabledModels(c.Request.Context())
	if err != nil {
		responses.FromError(c, err)
		return
	}
	out := make([]responses.ModelResponse, 0, len(items))
	for _, item := range items {
		out = append(out, responses.NewModel(item))
	}
	responses.OK(c, gin.H{"items": out})
}
