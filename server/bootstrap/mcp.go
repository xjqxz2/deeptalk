package bootstrap

import (
	"context"
	"time"

	"deeptalk/internal/mcp"
)

// startMCP 启动 stdio MCP server；失败时只告警并回退原有检索链路，
// 不让一个可选的浏览器工具阻断整个对话服务。
func (a *App) startMCP(ctx context.Context) error {
	cfg := a.Config.MCP
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

// stopMCP 关闭 MCP 会话和 Obscura 子进程；HTTP 已先停止，不会再有新工具调用。
func (a *App) stopMCP(ctx context.Context) error {
	if a.MCP == nil {
		return nil
	}
	err := a.MCP.Close()
	a.MCP = nil
	return err
}
