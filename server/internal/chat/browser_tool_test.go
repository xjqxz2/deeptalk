package chat

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"deeptalk/internal/provider"
	"deeptalk/internal/search"
)

// browserMCP 记录 chat 层下发的浏览器动作。
type browserMCP struct {
	tools []string
	last  map[string]any
}

func (b *browserMCP) HasTool(name string) bool {
	switch name {
	case "browser_navigate", "browser_snapshot", "browser_click", "browser_fill",
		"browser_press_key", "browser_scroll", "browser_back", "browser_links", "browser_markdown":
		return true
	}
	return false
}

func (b *browserMCP) CallTool(_ context.Context, name string, args any) (string, error) {
	b.tools = append(b.tools, name)
	if values, ok := args.(map[string]any); ok {
		b.last = values
	}
	if name == "browser_snapshot" {
		return "搜索结果列表 e1 第一条 e2 下一页", nil
	}
	return "", nil
}

// TestAvailableToolsGatesBrowser 校验没有浏览器能力时不下发 browser 工具。
func TestAvailableToolsGatesBrowser(t *testing.T) {
	plain := NewService(nil, nil, nil, nil, nil, nil, Config{})
	for _, tool := range plain.availableTools() {
		if tool.Function.Name == "browser" {
			t.Fatal("browser tool should not be exposed without MCP")
		}
	}

	mcp := &browserMCP{}
	searcher := search.New(search.Config{
		Enabled:    true,
		Engines:    []search.Engine{{Name: "bing", URL: "https://cn.bing.com/search?q={query}", Weight: 1}},
		MaxResults: 5,
		MCP:        mcp,
	})
	withBrowser := NewService(nil, nil, nil, searcher, nil, nil, Config{})
	found := false
	for _, tool := range withBrowser.availableTools() {
		if tool.Function.Name == "browser" {
			found = true
		}
	}
	if !found {
		t.Fatal("browser tool should be exposed when MCP supports it")
	}
}

// TestRunBrowserToolReturnsSnapshot 校验点击后会把新快照回传给模型。
func TestRunBrowserToolReturnsSnapshot(t *testing.T) {
	mcp := &browserMCP{}
	searcher := search.New(search.Config{
		Enabled:    true,
		Engines:    []search.Engine{{Name: "bing", URL: "https://cn.bing.com/search?q={query}", Weight: 1}},
		MaxResults: 5,
		MCP:        mcp,
	})
	service := NewService(nil, nil, nil, searcher, nil, nil, Config{})

	args, _ := json.Marshal(map[string]any{"action": "click", "ref": "e2"})
	var labels []string
	out, err := service.runBrowserTool(context.Background(), provider.ToolCall{Name: "browser", Arguments: string(args)},
		func(status, label, detail, url string) { labels = append(labels, label) })
	if err != nil {
		t.Fatalf("runBrowserTool: %v", err)
	}
	var payload struct {
		Action string `json:"action"`
		Page   string `json:"page"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("payload: %v (%s)", err, out)
	}
	if payload.Action != "click" || !strings.Contains(payload.Page, "搜索结果列表") {
		t.Fatalf("payload = %+v", payload)
	}
	if len(labels) == 0 || !strings.Contains(labels[0], "点击") {
		t.Fatalf("activity labels = %v", labels)
	}
	if len(mcp.tools) != 2 || mcp.tools[0] != "browser_click" || mcp.tools[1] != "browser_snapshot" {
		t.Fatalf("tools = %v", mcp.tools)
	}
}

// TestRunBrowserToolErrorIsRecoverable 校验失败会变成模型可读的错误，而不是中断回答。
func TestRunBrowserToolErrorIsRecoverable(t *testing.T) {
	mcp := &browserMCP{}
	searcher := search.New(search.Config{
		Enabled:    true,
		Engines:    []search.Engine{{Name: "bing", URL: "https://cn.bing.com/search?q={query}", Weight: 1}},
		MaxResults: 5,
		MCP:        mcp,
	})
	service := NewService(nil, nil, nil, searcher, nil, nil, Config{})
	args, _ := json.Marshal(map[string]any{"action": "click"})
	out, err := service.runBrowserTool(context.Background(), provider.ToolCall{Name: "browser", Arguments: string(args)},
		func(string, string, string, string) {})
	if err != nil {
		t.Fatalf("should not fail the turn: %v", err)
	}
	if !strings.Contains(out, "error") {
		t.Fatalf("out = %s", out)
	}
}
