// Package mcp 封装官方 Go MCP SDK，负责 Obscura 等 stdio MCP server 的
// 生命周期、工具发现和工具调用。
//
// 说明：这里不手写 JSON-RPC。协议握手、stdio 分帧、请求取消和子进程关闭
// 都交给 github.com/modelcontextprotocol/go-sdk。
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// DefaultProtocolVersion 是 Obscura 0.2.x 支持的 MCP 版本。
	DefaultProtocolVersion = "2024-11-05"
	// DefaultTimeout 覆盖一次 MCP 工具调用；搜索请求本身还会受上层 ctx 约束。
	DefaultTimeout = 30 * time.Second
)

// Config 描述一个 stdio MCP server。
type Config struct {
	Command         string            // 可执行文件，默认 obscura
	Args            []string          // 启动参数，默认 ["mcp"]
	Env             map[string]string // 追加或覆盖的环境变量
	ProtocolVersion string            // MCP 协议版本，默认 2024-11-05
	Timeout         time.Duration     // 连接与单次工具调用超时
}

// Client 是进程级 MCP 会话；并发工具调用会串行化，避免共享浏览器状态互相干扰。
type Client struct {
	session *mcpsdk.ClientSession
	log     *slog.Logger
	timeout time.Duration

	toolsMu  sync.RWMutex
	tools    map[string]struct{}
	toolList []string
	defs     []ToolInfo

	callMu    sync.Mutex
	closeOnce sync.Once
	closeErr  error
}

// Start 启动 MCP server、完成 initialize 握手并读取工具列表。
func Start(ctx context.Context, cfg Config, logger *slog.Logger) (*Client, error) {
	normalizeConfig(&cfg)
	if logger == nil {
		logger = slog.Default()
	}

	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Env = buildEnv(cfg.Env)
	// MCP 的 stdout 被 SDK 占用；stderr 直接透传到容器日志，便于排查启动失败。
	cmd.Stderr = os.Stderr

	sdkClient := mcpsdk.NewClient(&mcpsdk.Implementation{
		Name:    "DeepTalk",
		Version: "1.0.0",
	}, &mcpsdk.ClientOptions{Logger: logger})

	// 启动握手不能占用整个工具调用预算：Obscura 异常时最多等 10 秒，
	// 避免 Docker healthcheck 在 HTTP 端口就绪前把容器判死。
	connectTimeout := cfg.Timeout
	if connectTimeout > 10*time.Second {
		connectTimeout = 10 * time.Second
	}
	if connectTimeout < 3*time.Second {
		connectTimeout = 3 * time.Second
	}
	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	session, err := sdkClient.Connect(connectCtx, &mcpsdk.CommandTransport{
		Command:           cmd,
		TerminateDuration: 3 * time.Second,
	}, &mcpsdk.ClientSessionOptions{ProtocolVersion: cfg.ProtocolVersion})
	if err != nil {
		return nil, fmt.Errorf("connect mcp server %q: %w", cfg.Command, err)
	}

	client := &Client{
		session: session,
		log:     logger,
		timeout: cfg.Timeout,
		tools:   make(map[string]struct{}),
	}

	listCtx, cancelList := context.WithTimeout(ctx, connectTimeout)
	defer cancelList()
	result, err := session.ListTools(listCtx, nil)
	if err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("list mcp tools: %w", err)
	}
	for _, tool := range result.Tools {
		if tool == nil || strings.TrimSpace(tool.Name) == "" {
			continue
		}
		client.tools[tool.Name] = struct{}{}
		client.toolList = append(client.toolList, tool.Name)
		client.defs = append(client.defs, ToolInfo{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}
	sort.Strings(client.toolList)
	logger.Info("mcp server connected",
		"command", cfg.Command,
		"protocol", cfg.ProtocolVersion,
		"tools", client.toolList,
	)
	return client, nil
}

// HasTool 判断 server 是否声明了指定工具。
func (c *Client) HasTool(name string) bool {
	if c == nil {
		return false
	}
	c.toolsMu.RLock()
	defer c.toolsMu.RUnlock()
	_, ok := c.tools[name]
	return ok
}

// ToolNames 返回已发现的工具名，用于启动日志和诊断。
func (c *Client) ToolNames() []string {
	if c == nil {
		return nil
	}
	c.toolsMu.RLock()
	defer c.toolsMu.RUnlock()
	return append([]string(nil), c.toolList...)
}

// ToolInfo 是 MCP server 声明的一个工具，用于转成模型可调用的 tool。
type ToolInfo struct {
	Name        string // 工具名
	Description string // 说明
	InputSchema any    // JSON Schema（由 SDK 反序列化，原样透传）
}

// Tools 返回工具定义，供上层转换成模型工具。
func (c *Client) Tools() []ToolInfo {
	if c == nil {
		return nil
	}
	c.toolsMu.RLock()
	defer c.toolsMu.RUnlock()
	return append([]ToolInfo(nil), c.defs...)
}

// CallTool 调用一个 MCP 工具，并返回 text content 拼接后的文本。
func (c *Client) CallTool(ctx context.Context, name string, args any) (string, error) {
	if c == nil {
		return "", errors.New("mcp client is not configured")
	}
	if !c.HasTool(name) {
		return "", fmt.Errorf("mcp tool %q is not available", name)
	}

	// Obscura 的浏览器状态是单个会话，串行化工具调用避免 tab/页面互相覆盖。
	c.callMu.Lock()
	defer c.callMu.Unlock()

	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	result, err := c.session.CallTool(callCtx, &mcpsdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return "", fmt.Errorf("call mcp tool %q: %w", name, err)
	}

	text := contentText(result.Content)
	if result.IsError {
		if strings.TrimSpace(text) == "" {
			text = "mcp tool returned an error"
		}
		return "", errors.New(text)
	}
	if strings.TrimSpace(text) == "" && result.StructuredContent != nil {
		if raw, marshalErr := json.Marshal(result.StructuredContent); marshalErr == nil {
			text = string(raw)
		}
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("mcp tool %q returned empty content", name)
	}
	return text, nil
}

// Close 关闭 MCP 会话和底层的 Obscura 子进程。
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.closeOnce.Do(func() {
		if c.session != nil {
			c.closeErr = c.session.Close()
		}
	})
	return c.closeErr
}

func contentText(content []mcpsdk.Content) string {
	parts := make([]string, 0, len(content))
	for _, item := range content {
		if text, ok := item.(*mcpsdk.TextContent); ok && text != nil {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func normalizeConfig(cfg *Config) {
	if strings.TrimSpace(cfg.Command) == "" {
		cfg.Command = "obscura"
	}
	if len(cfg.Args) == 0 {
		cfg.Args = []string{"mcp"}
	}
	if strings.TrimSpace(cfg.ProtocolVersion) == "" {
		cfg.ProtocolVersion = DefaultProtocolVersion
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.Timeout > 2*time.Minute {
		cfg.Timeout = 2 * time.Minute
	}
}

func buildEnv(overrides map[string]string) []string {
	// 不把 DEEPTALK_SECRET / DEEPTALK_DEEPSEEK_KEY 等应用密钥传给浏览器子进程；
	// PATH、代理、glibc 兼容层等运行环境仍然继承。
	env := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "DEEPTALK_") {
			continue
		}
		env = append(env, entry)
	}
	for key, value := range overrides {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		entry := key + "=" + value
		prefix := key + "="
		replaced := false
		for index, current := range env {
			if strings.HasPrefix(current, prefix) {
				env[index] = entry
				replaced = true
				break
			}
		}
		if !replaced {
			env = append(env, entry)
		}
	}
	return env
}
