// Package bootstrap 负责装配服务并管理启动 / 关闭顺序。
package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"deeptalk/internal/mcp"
	"deeptalk/internal/wechat"

	"gorm.io/gorm"
)

// App 保存进程级共享依赖，按启动顺序逐步填充。
type App struct {
	Config *Config         // 生效配置
	Log    *slog.Logger    // 结构化日志
	DB     *gorm.DB        // 数据库句柄
	MCP    *mcp.Client     // MCP client；未启用或启动失败时为 nil
	WeChat *wechat.Manager // 小程序 access_token 管理器；未配置凭据时为 nil
	server *httpServer     // HTTP 设备
	cfgDir string          // 配置文件目录
}

// step 描述一个启动步骤及其逆序关闭动作。
type step struct {
	name  string                      // 步骤名，用于日志
	start func(context.Context) error // 启动
	stop  func(context.Context) error // 关闭，可为 nil
}

// Boost 按顺序启动设备，并在退出时逆序关闭。
type Boost struct {
	app     *App   // 共享依赖
	steps   []step // 启动步骤
	started int    // 已成功启动的步骤数
}

// New 创建启动器；Add 顺序即启动顺序：配置 -> 日志 -> 数据库 -> MCP -> HTTP。
func New() *Boost {
	app := &App{cfgDir: configDir()}
	boost := &Boost{app: app}
	boost.steps = []step{
		{name: "config", start: app.startConfig},
		{name: "logger", start: app.startLogger},
		{name: "database", start: app.startDatabase, stop: app.stopDatabase},
		{name: "mcp", start: app.startMCP, stop: app.stopMCP},
		{name: "wechat", start: app.startWeChat, stop: app.stopWeChat},
		{name: "http", start: app.startHTTP, stop: app.stopHTTP},
	}
	return boost
}

// Run 依次启动全部设备；任一失败则逆序回滚已启动的设备。
func (b *Boost) Run() error {
	ctx := context.Background()
	for index, item := range b.steps {
		if err := item.start(ctx); err != nil {
			b.rollback(index)
			return err
		}
		b.started = index + 1
		if b.app.Log != nil {
			b.app.Log.Info("device started", "name", item.name)
		}
	}
	return nil
}

// Waiting 阻塞等待退出信号，随后在超时内逆序关闭设备。
func (b *Boost) Waiting() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	if b.app.Log != nil {
		b.app.Log.Info("shutdown signal received")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	for index := b.started - 1; index >= 0; index-- {
		item := b.steps[index]
		if item.stop == nil {
			continue
		}
		if err := item.stop(ctx); err != nil && b.app.Log != nil {
			b.app.Log.Error("device stop failed", "name", item.name, "error", err)
		}
	}
	if b.app.Log != nil {
		b.app.Log.Info("shutdown complete")
	}
}

// rollback 逆序关闭已启动的设备，忽略关闭错误。
func (b *Boost) rollback(failedIndex int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for index := failedIndex - 1; index >= 0; index-- {
		item := b.steps[index]
		if item.stop == nil {
			continue
		}
		_ = item.stop(ctx)
	}
}

// startConfig 加载配置；失败直接终止启动。
func (a *App) startConfig(ctx context.Context) error {
	cfg, err := LoadConfig(a.cfgDir, "")
	if err != nil {
		return err
	}
	a.Config = cfg
	return nil
}

// startLogger 初始化结构化日志，并设置为全局默认 logger。
func (a *App) startLogger(ctx context.Context) error {
	level := slog.LevelInfo
	if a.Config.App.Env == "dev" {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler).With("app", a.Config.App.Name, "env", a.Config.App.Env)
	slog.SetDefault(logger)
	a.Log = logger
	return nil
}

// configDir 返回配置目录，允许通过环境变量覆盖以适配容器挂载。
func configDir() string {
	if dir := os.Getenv("DEEPTALK_CONFIG_DIR"); dir != "" {
		return dir
	}
	return "configs"
}
