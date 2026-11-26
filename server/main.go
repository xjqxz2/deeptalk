// DeepTalk 服务端入口：只负责拉起 bootstrap，不写路由、不连数据库。
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"deeptalk/bootstrap"
	"deeptalk/internal/sshmcp"
)

// 由 Makefile 通过 -ldflags 注入的构建信息。
var (
	version   = "dev"     // 版本号
	commit    = "unknown" // 提交号
	buildTime = "unknown" // 构建时间
)

// main 启动服务并等待退出信号；`ssh-mcp` 子命令以 stdio MCP server 运行。
func main() {
	if len(os.Args) > 1 && os.Args[1] == "ssh-mcp" {
		if err := runSSHMCP(); err != nil {
			fmt.Fprintf(os.Stderr, "ssh-mcp exited: %v\n", err)
			os.Exit(1)
		}
		return
	}
	fmt.Fprintf(os.Stderr, "DeepTalk %s (%s, %s)\n", version, commit, buildTime)
	app := bootstrap.New()
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "startup failed: %v\n", err)
		os.Exit(1)
	}
	app.Waiting()
}

// runSSHMCP 运行只做 SSH 的 MCP server。
// 参数优先读 app.yaml 的 mcp.ssh（与主服务同一份配置），环境变量仅作容器覆盖。
func runSSHMCP() error {
	cfg := sshmcp.Config{DataDir: os.Getenv("DEEPTALK_DATA_DIR")}
	configDir := os.Getenv("DEEPTALK_CONFIG_DIR")
	if configDir == "" {
		configDir = "configs"
	}
	if file, err := bootstrap.LoadConfig(configDir, ""); err == nil {
		cfg.KeyBits = file.MCP.SSH.KeyBits
		cfg.DefaultTimeout = time.Duration(file.MCP.SSH.TimeoutSec) * time.Second
		if cfg.DataDir == "" {
			cfg.DataDir = filepath.Dir(file.Database.DSN)
		}
	}
	if raw := os.Getenv("DEEPTALK_SSH_KEY_BITS"); raw != "" {
		if bits, err := strconv.Atoi(raw); err == nil {
			cfg.KeyBits = bits
		}
	}
	if raw := os.Getenv("DEEPTALK_SSH_TIMEOUT_SEC"); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			cfg.DefaultTimeout = time.Duration(seconds) * time.Second
		}
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return sshmcp.Run(context.Background(), cfg, logger)
}
