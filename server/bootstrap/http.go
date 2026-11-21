package bootstrap

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"deeptalk/apis"
	"deeptalk/apis/middleware"
	"deeptalk/internal/attachment"
	"deeptalk/internal/audit"
	"deeptalk/internal/chat"
	"deeptalk/internal/compliance"
	"deeptalk/internal/provider"
	"deeptalk/internal/search"
	"deeptalk/internal/setting"
	"deeptalk/internal/user"
)

// httpServer 包装标准库 http.Server，便于测试与关闭。
type httpServer struct {
	server *http.Server
}

// startHTTP 组装业务服务、路由并开始监听。
// 监听地址来自 app.addr，默认 0.0.0.0:9821。
func (a *App) startHTTP(ctx context.Context) error {
	cfg := a.Config
	// 业务服务按依赖方向手动注入，不使用全局单例
	settings := setting.NewService(a.DB)
	audits := audit.NewService(a.DB)
	users := user.NewService(a.DB, settings, audits, user.Options{
		SessionTTL:    cfg.Auth.SessionTTL(),
		LockThreshold: cfg.Auth.LockThreshold,
		LockMinutes:   cfg.Auth.LockMinutes,
		DefaultQuota:  cfg.Auth.DefaultQuotaTokens,
	})
	providers := provider.NewService(a.DB, cfg.Security.Secret, cfg.App.Env == "dev", settings, audits)
	attachments, err := attachment.NewService(a.DB, cfg.Uploads.Dir, int64(cfg.Uploads.MaxMB)<<20)
	if err != nil {
		return err
	}
	engines := make([]search.Engine, 0, len(cfg.WebSearch.Engines))
	for _, item := range cfg.WebSearch.Engines {
		engines = append(engines, search.Engine{Name: item.Name, URL: item.URL, Weight: item.Weight})
	}
	// nil interface 与 nil 指针不同：MCP 未启用时保持接口为 nil，检索服务才走本地引擎
	var mcpCaller search.ToolCaller
	if a.MCP != nil {
		mcpCaller = a.MCP
	}
	searcher := search.New(search.Config{
		Enabled:      cfg.WebSearch.Enabled,
		ObscuraBin:   cfg.WebSearch.ObscuraBin,
		EngineURL:    cfg.WebSearch.EngineURL,
		Engines:      engines,
		MaxResults:   cfg.WebSearch.MaxResults,
		Concurrency:  cfg.WebSearch.Concurrency,
		Timeout:      time.Duration(cfg.WebSearch.TimeoutSec) * time.Second,
		HTTPFallback: cfg.WebSearch.HTTPFallback,
		Stealth:      cfg.WebSearch.Stealth,
		MCP:          mcpCaller,
	})
	// 小程序端内容安全：只对 X-Client=mp-weixin 的请求生效
	checker := compliance.New(compliance.Config{
		Enabled:     cfg.WeChat.Security.Enabled,
		Scene:       cfg.WeChat.Security.Scene,
		FailOpen:    cfg.WeChat.Security.FailOpen,
		RequireBind: cfg.WeChat.Security.RequireBind,
	}, a.WeChat, users, a.Log)

	chats := chat.NewService(a.DB, providers, users, searcher, attachments, settings, chat.Config{
		MaxOutputTokens:  cfg.Upstream.MaxOutputTokens,
		FirstByteTimeout: time.Duration(cfg.Upstream.FirstByteTimeoutSec) * time.Second,
		ReasoningEffort:  cfg.Upstream.ReasoningEffort,
		CompactRatio:     cfg.Context.CompactRatio,
		KeepRatio:        cfg.Context.KeepRatio,
		CacheIdleHours:   cfg.Context.CacheIdleHours,
		IdleMinTokens:    cfg.Context.IdleMinTokens,
		Location:         cfg.App.Location,
		Timezone:         cfg.App.Timezone,
		Compliance:       checker,
	})
	chats.StartCompactionWorker(ctx)

	router := apis.NewRouter(apis.Config{
		Env:    cfg.App.Env,
		DBPath: cfg.Database.DSN,
		Auth: middleware.Config{
			CookieName:     cfg.Auth.CookieName,
			CookieSecure:   cfg.Auth.CookieSecure,
			SessionTTL:     cfg.Auth.SessionTTL(),
			AllowedOrigins: cfg.CORS.AllowedOrigins,
		},
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		WebSearchEnabled: cfg.WebSearch.Enabled,
	}, apis.Deps{
		Users:       users,
		Providers:   providers,
		Chats:       chats,
		Attachments: attachments,
		Settings:    settings,
		Audits:      audits,
		WeChat:      a.WeChat,
	}, a.Log)

	listener, err := net.Listen("tcp", cfg.App.Addr)
	if err != nil {
		return err
	}
	server := &http.Server{
		Handler: router,
		// SSE 长连接不能设置 WriteTimeout，否则生成中会被强制断开
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	a.server = &httpServer{server: server}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.Log.Error("http server stopped unexpectedly", "error", err)
		}
	}()
	a.Log.Info("http listening", "addr", listener.Addr().String(), "base_url", cfg.App.BaseURL)
	return nil
}

// stopHTTP 优雅关闭 HTTP 服务；超时后强制关闭，避免 SSE 阻塞退出。
func (a *App) stopHTTP(ctx context.Context) error {
	if a.server == nil {
		return nil
	}
	if err := a.server.server.Shutdown(ctx); err != nil {
		_ = a.server.server.Close()
		return err
	}
	return nil
}
