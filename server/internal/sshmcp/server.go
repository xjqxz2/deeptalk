// Package sshmcp 实现一个只做 SSH 的 MCP server。
//
// 设计约束：
//   - 只提供 SSH 能力，不执行任何本机命令，也不暴露本机文件系统；
//   - 每个 owner（用户）一把独立密钥，密钥按需生成在 <data>/ssh/<owner>/id_rsa；
//   - 会话绑定 owner，跨用户无法复用；
//   - 主机指纹首次连接按 TOFU 记录，之后不一致直接拒绝。
package sshmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ownerInjected 是宿主注入的字段说明：模型不该自己设置 owner。
const ownerNote = "由宿主自动填写，模型不要设置。"

// serverInstructions 会写进 MCP 初始化结果，帮助模型理解使用顺序。
const serverInstructions = `你可以通过 SSH 操作远端服务器，但不能操作本机。
使用顺序：
1. 先调用 ssh_public_key 拿到公钥，把公钥和配置方法原样告诉用户，等用户把它加到目标机器的 ~/.ssh/authorized_keys；
2. 用户确认后再调用 ssh_connect（host / username 由用户提供），拿到 session_id；
3. 用 ssh_exec 在会话里执行命令，每次只做一件事，读结果再决定下一步；
4. 用完调用 ssh_disconnect。
主机指纹首次会自动记录；若提示指纹不一致，必须停下来告诉用户，不要重试。`

// Run 启动 stdio MCP server，直到 ctx 取消或 stdin 关闭。
func Run(ctx context.Context, cfg Config, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	manager := NewManager(cfg)
	defer manager.Close()

	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "deeptalk-ssh",
		Version: "1.0.0",
	}, &mcpsdk.ServerOptions{
		Instructions: serverInstructions,
	})

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "ssh_public_key",
		Description: "返回本 AI 的 SSH 公钥（authorized_keys 行）、指纹以及给用户的配置步骤。" +
			"第一次连接某台机器前必须先调用它，并把公钥原样展示给用户。",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, args publicKeyArgs) (*mcpsdk.CallToolResult, any, error) {
		pair, err := manager.PublicKey(args.Owner)
		if err != nil {
			return errResult(err), nil, nil
		}
		return jsonResult(map[string]any{
			"public_key":  pair.PublicKey,
			"fingerprint": pair.Fingerprint,
			"instructions": "把 public_key 整行追加到目标机器的 ~/.ssh/authorized_keys（目录 700，文件 600），" +
				"然后告诉我 host 与 username，我来连接。",
		}), nil, nil
	})

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "ssh_connect",
		Description: "用本 AI 的密钥连接一台远端机器。host 与 username 必须来自用户的明确输入。" +
			"可选 workdir 会把会话钉在某个目录（每条命令都会先 cd 进去）。返回 session_id。",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, args connectArgs) (*mcpsdk.CallToolResult, any, error) {
		session, err := manager.Connect(ctx, args.Owner, args.Host, args.Port, args.Username, args.Workdir)
		if err != nil {
			return errResult(err), nil, nil
		}
		logger.Info("ssh session opened",
			"owner", args.Owner, "host", session.Host, "port", session.Port, "user", session.User, "session", session.ID)
		return jsonResult(map[string]any{
			"session_id": session.ID,
			"host":       session.Host,
			"port":       session.Port,
			"username":   session.User,
			"workdir":    session.Workdir,
			"message":    "连接已建立，可以调用 ssh_exec 了",
		}), nil, nil
	})

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "ssh_exec",
		Description: "在已建立的会话上执行一条 shell 命令，返回 stdout、stderr 与退出码。" +
			"命令在远端执行，本机不受影响；单条命令默认 60 秒超时，超时会断开该会话。",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, args execArgs) (*mcpsdk.CallToolResult, any, error) {
		timeout := time.Duration(args.TimeoutSec) * time.Second
		result, err := manager.Exec(ctx, args.Owner, args.SessionID, args.Command, timeout)
		if err != nil {
			return errResult(err), nil, nil
		}
		return jsonResult(map[string]any{
			"stdout":    result.Stdout,
			"stderr":    result.Stderr,
			"exit_code": result.ExitCode,
			"truncated": result.Truncated,
		}), nil, nil
	})

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "ssh_disconnect",
		Description: "关闭一个 SSH 会话，任务完成后调用。",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, args disconnectArgs) (*mcpsdk.CallToolResult, any, error) {
		if err := manager.Disconnect(args.Owner, args.SessionID); err != nil {
			return errResult(err), nil, nil
		}
		return jsonResult(map[string]any{"closed": true}), nil, nil
	})

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "ssh_sessions",
		Description: "列出当前仍然打开的 SSH 会话。",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, args sessionsArgs) (*mcpsdk.CallToolResult, any, error) {
		items := make([]map[string]any, 0)
		for _, session := range manager.Sessions(args.Owner) {
			items = append(items, map[string]any{
				"session_id": session.ID,
				"host":       session.Host,
				"port":       session.Port,
				"username":   session.User,
				"workdir":    session.Workdir,
				"created_at": session.Created.Format(time.RFC3339),
			})
		}
		return jsonResult(map[string]any{"sessions": items}), nil, nil
	})

	return server.Run(ctx, &mcpsdk.StdioTransport{})
}

// publicKeyArgs 是 ssh_public_key 的参数。
type publicKeyArgs struct {
	Owner string `json:"owner" jsonschema:"宿主注入的 owner，模型不要设置"`
}

// connectArgs 是 ssh_connect 的参数。
type connectArgs struct {
	Host     string `json:"host" jsonschema:"目标主机名或 IP，来自用户输入"`
	Port     int    `json:"port,omitempty" jsonschema:"SSH 端口，默认 22"`
	Username string `json:"username" jsonschema:"登录用户名，来自用户输入"`
	Workdir  string `json:"workdir,omitempty" jsonschema:"可选：把会话固定在这个绝对路径下"`
	Owner    string `json:"owner" jsonschema:"宿主注入的 owner，模型不要设置"`
}

// execArgs 是 ssh_exec 的参数。
type execArgs struct {
	SessionID  string `json:"session_id" jsonschema:"ssh_connect 返回的会话 ID"`
	Command    string `json:"command" jsonschema:"要在远端执行的命令"`
	TimeoutSec int    `json:"timeout_sec,omitempty" jsonschema:"超时秒数，默认 60，最大 300"`
	Owner      string `json:"owner" jsonschema:"宿主注入的 owner，模型不要设置"`
}

// disconnectArgs 是 ssh_disconnect 的参数。
type disconnectArgs struct {
	SessionID string `json:"session_id" jsonschema:"要关闭的会话 ID"`
	Owner     string `json:"owner" jsonschema:"宿主注入的 owner，模型不要设置"`
}

// sessionsArgs 是 ssh_sessions 的参数。
type sessionsArgs struct {
	Owner string `json:"owner" jsonschema:"宿主注入的 owner，模型不要设置"`
}

// jsonResult 把结构体序列化成一条文本结果。
func jsonResult(value any) *mcpsdk.CallToolResult {
	data, err := json.Marshal(value)
	if err != nil {
		return errResult(err)
	}
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(data)}},
	}
}

// errResult 把错误转成模型可读的文本，而不是协议错误。
func errResult(err error) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		IsError: true,
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: fmt.Sprintf("失败：%v", err)}},
	}
}
