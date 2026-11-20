// Package c 提供 C 端（普通用户）接口控制器。
package c

import (
	"errors"

	"deeptalk/apis/middleware"
	"deeptalk/apis/requests"
	"deeptalk/apis/responses"
	"deeptalk/internal/provider"
	"deeptalk/internal/setting"
	"deeptalk/internal/user"

	"github.com/gin-gonic/gin"
)

// AuthController 处理注册、登录、退出、当前用户与用量。
type AuthController struct {
	users         *user.Service     // 账号能力
	providers     *provider.Service // 模型能力
	settings      *setting.Service  // 平台设置
	cfg           middleware.Config // Cookie 等配置
	searchEnabled bool              // 联网搜索是否可用
}

// NewAuthController 创建 C 端账号控制器。
func NewAuthController(users *user.Service, providers *provider.Service, settings *setting.Service, cfg middleware.Config, searchEnabled bool) *AuthController {
	return &AuthController{users: users, providers: providers, settings: settings, cfg: cfg, searchEnabled: searchEnabled}
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
		"site_name":          "DeepTalk",
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
	responses.OK(c, gin.H{"user": responses.NewUser(u), "token": token})
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
