package mcp

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestMCPHelperProcess 在子进程里模拟一个最小 MCP server。
// 用真实 SDK 的 stdio 传输验证协议握手、工具发现和工具调用，而不是测自研协议。
func TestMCPHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_MCP_HELPER") != "1" {
		return
	}
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "fake-obscura", Version: "1.0.0"}, nil)
	type searchArgs struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "web_search"}, func(_ context.Context, _ *mcpsdk.CallToolRequest, args searchArgs) (*mcpsdk.CallToolResult, any, error) {
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{
				Text: `{"title":"DeepSeek V4","url":"https://example.com/deepseek","snippet":"price"}`,
			}},
		}, nil, nil
	})
	session, err := server.Connect(context.Background(), &mcpsdk.StdioTransport{}, nil)
	if err != nil {
		os.Exit(2)
	}
	_ = session.Wait()
}

func TestClientStartListAndCall(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	client, err := Start(context.Background(), Config{
		Command:         os.Args[0],
		Args:            []string{"-test.run=TestMCPHelperProcess", "--"},
		Env:             map[string]string{"GO_WANT_MCP_HELPER": "1"},
		ProtocolVersion: DefaultProtocolVersion,
		Timeout:         5 * time.Second,
	}, logger)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	if !client.HasTool("web_search") {
		t.Fatalf("web_search not discovered: %v", client.ToolNames())
	}
	if client.HasTool("missing") {
		t.Fatal("unexpected missing tool")
	}
	text, err := client.CallTool(context.Background(), "web_search", map[string]any{"query": "DeepSeek", "limit": 2})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !strings.Contains(text, "example.com/deepseek") {
		t.Fatalf("unexpected tool text: %s", text)
	}
	if _, err := client.CallTool(context.Background(), "missing", nil); err == nil {
		t.Fatal("expected missing tool error")
	}
}
