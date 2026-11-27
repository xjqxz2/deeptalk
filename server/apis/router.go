// Package apis 负责路由汇总与 HTTP 层装配。
package apis

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"deeptalk/apis/b"
	capi "deeptalk/apis/c"
	"deeptalk/apis/middleware"
	"deeptalk/apis/responses"
	"deeptalk/internal/attachment"
	"deeptalk/internal/audit"
	"deeptalk/internal/billing"
	"deeptalk/internal/chat"
	"deeptalk/internal/memo"
	"deeptalk/internal/provider"
	"deeptalk/internal/security"
	"deeptalk/internal/setting"
	"deeptalk/internal/user"
	"deeptalk/internal/wechat"
	"deeptalk/web"

	"github.com/gin-gonic/gin"
)

// Config 是 HTTP 层所需配置，避免 apis 反向依赖 bootstrap。
type Config struct {
	Env              string            // 运行环境
	DBPath           string            // SQLite 路径（设置页展示存储占用）
	Auth             middleware.Config // Cookie 与会话配置
	AllowedOrigins   []string          // 开发期跨域来源
	SignMode         string            // 请求签名模式：off / log / enforce
	SecuritySecret   string            // 签名密钥派生源（与 AES-GCM 同源）
	WebSearchEnabled bool              // 联网搜索是否可用（下发给 C 端）
}

// Deps 是控制器依赖的业务服务集合。
type Deps struct {
	Users       *user.Service       // 账号
	Providers   *provider.Service   // 厂商
	Chats       *chat.Service       // 会话
	Attachments *attachment.Service // 图片附件
	Settings    *setting.Service    // 设置
	Audits      *audit.Service      // 审计
	WeChat      *wechat.Manager     // 小程序 access_token（未配置时为 nil）
	Billing     *billing.Service    // 套餐限额与计费
	Memos       *memo.Service       // 小橘灵感录
}

// NewRouter 装配全部路由、中间件与静态资源。
func NewRouter(cfg Config, deps Deps, log *slog.Logger) *gin.Engine {
	if cfg.Env != "dev" {
		gin.SetMode(gin.ReleaseMode)
	}
	// 请求签名：密钥按会话派生，服务端只存 key id
	signVerifier := security.NewVerifier(cfg.SecuritySecret)
	router := gin.New()
	// 只信任本机与私有网段的代理（Nginx / Docker 网关）：
	// 这样能拿到 Nginx 透传的真实客户端 IP，外网直连伪造的 X-Forwarded-For 会被忽略。
	_ = router.SetTrustedProxies([]string{"127.0.0.1/32", "::1/128", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})
	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery(log))
	router.Use(middleware.AccessLog(log))
	router.Use(middleware.CORS(cfg.AllowedOrigins))

	router.GET("/healthz", func(c *gin.Context) {
		responses.OK(c, gin.H{"status": "ok"})
	})

	auth := middleware.Auth(deps.Users, cfg.Auth)
	admin := middleware.AdminOnly()

	authController := capi.NewAuthController(deps.Users, deps.Providers, deps.Settings, cfg.Auth, cfg.WebSearchEnabled, deps.WeChat, signVerifier, deps.Billing)
	chatController := capi.NewChatController(deps.Chats)
	wsController := capi.NewWSController(deps.Chats, deps.Users, cfg.Auth)
	memoController := capi.NewMemoController(deps.Memos)
	bMemoController := b.NewMemoController(deps.Memos, deps.Audits)
	memoMCP := memo.Handler(deps.Memos)
	attachmentController := capi.NewAttachmentController(deps.Attachments)

	overviewController := b.NewOverviewController(deps.Users, deps.Providers, deps.Chats, deps.Audits)
	usageController := b.NewUsageController(deps.Chats)
	providerController := b.NewProviderController(deps.Providers, deps.Audits)
	billingController := b.NewBillingController(deps.Billing)
	userController := b.NewUserController(deps.Users, deps.Chats, deps.Audits, deps.Billing)
	settingsController := b.NewSettingsController(deps.Settings, deps.Providers, deps.Audits, cfg.DBPath, cfg.SecuritySecret)

	api := router.Group("/api/v1")

	// 公开接口：登录、注册与注册开关
	cPublic := api.Group("/c/auth")
	{
		cPublic.GET("/config", authController.Config)
		cPublic.GET("/plans", authController.Plans)
		cPublic.POST("/register", authController.Register)
		cPublic.POST("/login", authController.Login)
		cPublic.POST("/wechat/login", authController.WeChatLogin)
	}

	// C 端受保护接口
	cGroup := api.Group("/c", auth, middleware.Signature(signVerifier, cfg.SignMode, log))
	{
		cGroup.GET("/auth/me", authController.Me)
		cGroup.POST("/auth/logout", authController.Logout)
		cGroup.PUT("/auth/password", authController.ChangePassword)
		cGroup.GET("/auth/sign-key", authController.SignKey)
		cGroup.POST("/auth/wechat/bind", authController.BindWeChat)
		cGroup.POST("/auth/profile/complete", authController.CompleteProfile)
		cGroup.PATCH("/auth/profile", authController.UpdateProfile)
		cGroup.GET("/usage", authController.Usage)
		cGroup.GET("/models", authController.Models)
		cGroup.GET("/memos", memoController.List)
		cGroup.POST("/memos", memoController.Create)
		cGroup.PATCH("/memos/:id", memoController.Update)
		cGroup.DELETE("/memos/:id", memoController.Delete)
		// MCP 端点：外部 Agent 通过 Streamable HTTP 直接读写灵感录。
		// handler 必须复用同一实例，否则每次请求都会新建会话表，Mcp-Session-Id 失效。
		cGroup.Any("/memo/mcp", func(c *gin.Context) {
			ctx := memo.WithUser(c.Request.Context(), middleware.CurrentUser(c).ID)
			c.Request = c.Request.WithContext(ctx)
			memoMCP.ServeHTTP(c.Writer, c.Request)
		})

		cGroup.POST("/attachments", attachmentController.Upload)
		cGroup.GET("/attachments/:id", attachmentController.Get)

		cGroup.GET("/conversations", chatController.List)
		cGroup.POST("/conversations", chatController.Create)
		cGroup.GET("/conversations/:id", chatController.Get)
		cGroup.PATCH("/conversations/:id", chatController.Rename)
		cGroup.DELETE("/conversations/:id", chatController.Delete)
		cGroup.POST("/conversations/batch-delete", chatController.BatchDelete)
		cGroup.POST("/conversations/:id/stream", chatController.Stream)
		cGroup.POST("/conversations/:id/stop", chatController.Stop)
		cGroup.GET("/conversations/:id/stream-state", chatController.StreamState)
		cGroup.POST("/conversations/:id/interjections", chatController.Interject)
		cGroup.POST("/messages/:id/feedback", chatController.Feedback)

		// 分享只在登录态下创建与撤销；访客只能读取公开快照
		cGroup.GET("/conversations/:id/share", chatController.GetShare)
		cGroup.POST("/conversations/:id/share", chatController.CreateShare)
		cGroup.DELETE("/conversations/:id/share", chatController.RevokeShare)
	}

	// WebSocket 长连接：供小程序 / APP 使用，鉴权在处理器内部完成
	// （小程序无法消费 SSE，也不方便携带 Cookie）
	api.GET("/c/ws", wsController.Conn)

	// 公开分享页：只读，不经过 Auth 中间件，也不提供任何回复接口
	cPublicGroup := api.Group("/c/public")
	{
		cPublicGroup.GET("/shares/:token", chatController.PublicShare)
	}

	// B 端管理接口，需管理员身份
	bGroup := api.Group("/b", auth, admin, middleware.Signature(signVerifier, cfg.SignMode, log))
	{
		bGroup.GET("/overview", overviewController.Overview)
		bGroup.GET("/audit-logs", overviewController.AuditLogs)
		bGroup.GET("/usage", usageController.Usage)

		bGroup.GET("/plans", billingController.List)
		bGroup.PUT("/plans/:id", billingController.Update)
		bGroup.GET("/providers", providerController.List)
		bGroup.POST("/providers", providerController.Create)
		bGroup.POST("/providers/test", providerController.TestDraft)
		bGroup.PUT("/providers/:id", providerController.Update)
		bGroup.DELETE("/providers/:id", providerController.Delete)
		bGroup.PATCH("/providers/:id/enabled", providerController.Toggle)
		bGroup.POST("/providers/:id/test", providerController.Test)

		bGroup.GET("/models", providerController.Models)
		bGroup.POST("/models", providerController.CreateModel)
		bGroup.PUT("/models/:id", providerController.UpdateModel)
		bGroup.PATCH("/models/:id/enabled", providerController.ToggleModel)
		bGroup.DELETE("/models/:id", providerController.DeleteModel)
		bGroup.GET("/routing", providerController.Routing)
		bGroup.PUT("/routing", providerController.SetRouting)
		bGroup.GET("/health", providerController.Health)

		bGroup.GET("/users/stats", userController.Stats)
		bGroup.GET("/users/export.csv", userController.Export)
		bGroup.GET("/users", userController.List)
		bGroup.POST("/users", userController.Create)
		bGroup.GET("/users/:id", userController.Detail)
		bGroup.PATCH("/users/:id/quota", userController.UpdateQuota)
		bGroup.PATCH("/users/:id/plan", userController.UpdatePlan)
		bGroup.PATCH("/users/:id/disabled", userController.ToggleDisabled)
		bGroup.PATCH("/users/:id/role", userController.SetRole)
		bGroup.DELETE("/users/:id", userController.Delete)
		bGroup.GET("/memos", bMemoController.List)
		bGroup.GET("/memos/stats", bMemoController.Stats)
		bGroup.DELETE("/memos/:id", bMemoController.Delete)
		bGroup.POST("/users/:id/reset-password", userController.ResetPassword)
		bGroup.POST("/users/:id/logout-all", userController.LogoutAll)

		bGroup.GET("/settings", settingsController.Get)
		bGroup.PUT("/settings", settingsController.Update)
	}

	mountFrontend(router)
	return router
}

// mountFrontend 托管两个前端构建产物：C 端在根路径，B 端在 /admin。
// 未命中静态文件时回退到各自 index.html，支持 history 路由刷新。
func mountFrontend(router *gin.Engine) {
	cFS, err := fs.Sub(web.FS, "dist/c")
	if err != nil {
		return
	}
	bFS, err := fs.Sub(web.FS, "dist/b")
	if err != nil {
		return
	}
	router.GET("/", func(c *gin.Context) {
		serveIndex(c, cFS)
	})
	// 只注册 catch-all：/admin 会由 Gin 自动重定向到 /admin/
	router.GET("/admin/*path", func(c *gin.Context) {
		relative := strings.TrimPrefix(c.Param("path"), "/")
		if relative != "" && fileExistsIn(bFS, relative) {
			serveFrontendFile(c, bFS, relative, false)
			return
		}
		serveIndex(c, bFS)
	})
	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") {
			responses.Fail(c, responses.CodeNotFound, "接口不存在")
			return
		}
		relative := strings.TrimPrefix(path, "/")
		if relative != "" && fileExistsIn(cFS, relative) {
			serveFrontendFile(c, cFS, relative, true)
			return
		}
		serveIndex(c, cFS)
	})
}

// serveFrontendFile 输出前端静态文件。
// Service Worker、manifest 与离线页必须短缓存或禁缓存，否则发版后会继续使用旧版本；
// 带内容哈希的 /assets 才能长期 immutable。admin 端没有 PWA 文件，沿用长期缓存。
func serveFrontendFile(c *gin.Context, fsys fs.FS, relative string, pwa bool) {
	if !pwa {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.FileFromFS(relative, http.FS(fsys))
		return
	}

	switch relative {
	case "sw.js":
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Service-Worker-Allowed", "/")
		c.Header("Content-Type", "text/javascript; charset=utf-8")
		c.Header("X-Content-Type-Options", "nosniff")
	case "manifest.webmanifest":
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Content-Type", "application/manifest+json; charset=utf-8")
	case "offline.html":
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Content-Type", "text/html; charset=utf-8")
	default:
		if strings.HasPrefix(relative, "icons/") {
			c.Header("Cache-Control", "public, max-age=604800")
			break
		}
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	}
	c.FileFromFS(relative, http.FS(fsys))
}

// serveIndex 输出 SPA 入口，并禁止 HTML 缓存，保证发版后能立即生效。
// 说明：不直接使用 http.FileServer，因为它会把 index.html 301 重定向到 "./"。
func serveIndex(c *gin.Context, fsys fs.FS) {
	data, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		responses.Fail(c, responses.CodeInternal, "前端资源未就绪，请先构建 ui")
		return
	}
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

// fileExistsIn 判断嵌入文件系统中是否存在该普通文件。
func fileExistsIn(fsys fs.FS, name string) bool {
	info, err := fs.Stat(fsys, name)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
