package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"deeptalk/internal/mcp"
)

// startMCP 启动 stdio MCP server；失败时只告警并回退原有检索链路，
// 不让一个可选的浏览器工具阻断整个对话服务。
func (a *App) startMCP(ctx context.Context) error {
	cfg := a.Config.MCP
	// SSH 与浏览器两个 MCP 互相独立：obscura 缺失不应该把 SSH 一起关掉
	defer a.startSSHMCP(ctx)
	if !cfg.Enabled {
		return nil
	}
	args := append([]string(nil), cfg.Args...)
	if a.Config.WebSearch.Stealth {
		// 保持与 Obscura CLI 抓取一致的 stealth 行为
		hasStealth := false
		for _, arg := range args {
			if arg == "--stealth" {
				hasStealth = true
				break
			}
		}
		if !hasStealth {
			args = append(args, "--stealth")
		}
	}
	client, err := mcp.Start(ctx, mcp.Config{
		Command:         cfg.Command,
		Args:            args,
		Env:             cfg.Env,
		ProtocolVersion: cfg.ProtocolVersion,
		Timeout:         time.Duration(cfg.TimeoutSec) * time.Second,
	}, a.Log)
	if err != nil {
		a.Log.Warn("mcp server unavailable; web search will fall back to the local engine path", "error", err)
		return nil
	}
	if !client.HasTool("browser_navigate") || !client.HasTool("browser_evaluate") {
		a.Log.Warn("mcp browser tools are not advertised; web search will use Obscura CLI or HTTP fallback", "tools", client.ToolNames())
	}
	a.MCP = client
	return nil
}

// startSSHMCP 启动内置 SSH MCP server（主程序的 ssh-mcp 子命令）。
// 它只提供远端 SSH 能力，不提供本机 shell；启动失败只告警，不影响主服务。
func (a *App) startSSHMCP(ctx context.Context) {
	cfg := a.Config.MCP.SSH
	if !cfg.Enabled {
		a.Log.Info("ssh mcp disabled by config")
		return
	}
	self, err := os.Executable()
	if err != nil {
		a.Log.Warn("ssh mcp disabled: cannot resolve own executable", "error", err)
		return
	}
	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	// 与主服务共用数据目录：密钥落在 <data>/ssh/<owner>/ 下
	dataDir := filepath.Dir(a.Config.Database.DSN)
	client, startErr := mcp.Start(ctx, mcp.Config{
		Command: self,
		Args:    []string{"ssh-mcp"},
		Env:     map[string]string{"DEEPTALK_DATA_DIR": dataDir},
		Timeout: timeout,
	}, a.Log)
	if startErr != nil {
		a.Log.Warn("ssh mcp unavailable; ssh tools are hidden from the model", "error", startErr)
		return
	}
	if len(client.Tools()) == 0 {
		a.Log.Warn("ssh mcp advertised no tools", "tools", client.ToolNames())
		_ = client.Close()
		return
	}
	a.SSHMCP = client
	a.Log.Info("ssh mcp ready", "tools", client.ToolNames(), "data_dir", dataDir)
}

// stopMCP 关闭 MCP 会话和 Obscura 子进程；HTTP 已先停止，不会再有新工具调用。
func (a *App) stopMCP(ctx context.Context) error {
	if a.SSHMCP != nil {
		_ = a.SSHMCP.Close()
		a.SSHMCP = nil
	}
	if a.MCP == nil {
		return nil
	}
	err := a.MCP.Close()
	a.MCP = nil
	return err
}
