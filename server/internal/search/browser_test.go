package search

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// recordingMCP 记录模型触发的浏览器动作，用于校验参数映射。
type recordingMCP struct {
	calls []struct {
		tool string
		args map[string]any
	}
	result string
}

func (r *recordingMCP) HasTool(name string) bool {
	return name == "browser_navigate" || name == "browser_snapshot" || name == "browser_click" ||
		name == "browser_fill" || name == "browser_press_key" || name == "browser_scroll" ||
		name == "browser_back" || name == "browser_links" || name == "browser_markdown"
}

func (r *recordingMCP) CallTool(_ context.Context, name string, args any) (string, error) {
	values, _ := args.(map[string]any)
	r.calls = append(r.calls, struct {
		tool string
		args map[string]any
	}{name, values})
	if name == "browser_snapshot" {
		return "页面正文 e1 搜索框 e2 提交按钮", nil
	}
	return r.result, nil
}

func newBrowserService(mcp *recordingMCP) *Service {
	return New(Config{
		Enabled:    true,
		Engines:    []Engine{{Name: "bing", URL: "https://cn.bing.com/search?q={query}", Weight: 1}},
		MaxResults: 5,
		MCP:        mcp,
	})
}

func TestBrowserOpenReturnsSnapshot(t *testing.T) {
	mcp := &recordingMCP{}
	service := newBrowserService(mcp)
	out, err := service.Browser(context.Background(), BrowserAction{Action: "open", URL: "https://example.com"})
	if err != nil {
		t.Fatalf("Browser: %v", err)
	}
	if !strings.Contains(out, "页面正文") {
		t.Fatalf("snapshot missing: %q", out)
	}
	if len(mcp.calls) != 2 || mcp.calls[0].tool != "browser_navigate" || mcp.calls[1].tool != "browser_snapshot" {
		t.Fatalf("calls = %+v", mcp.calls)
	}
	if mcp.calls[0].args["url"] != "https://example.com" {
		t.Fatalf("navigate args = %+v", mcp.calls[0].args)
	}
}

func TestBrowserClickPrefersRef(t *testing.T) {
	mcp := &recordingMCP{}
	service := newBrowserService(mcp)
	if _, err := service.Browser(context.Background(), BrowserAction{Action: "click", Ref: "e3", Selector: "#submit"}); err != nil {
		t.Fatalf("Browser: %v", err)
	}
	click := mcp.calls[0]
	if click.tool != "browser_click" || click.args["ref"] != "e3" {
		t.Fatalf("click args = %+v", click.args)
	}
	if _, ok := click.args["selector"]; ok {
		t.Fatal("ref 可用时不应同时传 selector")
	}
}

func TestBrowserClickRequiresTarget(t *testing.T) {
	service := newBrowserService(&recordingMCP{})
	if _, err := service.Browser(context.Background(), BrowserAction{Action: "click"}); err == nil {
		t.Fatal("click without ref/selector should fail")
	}
}

func TestBrowserFillAndPress(t *testing.T) {
	mcp := &recordingMCP{}
	service := newBrowserService(mcp)
	if _, err := service.Browser(context.Background(), BrowserAction{Action: "fill", Selector: "#q", Text: "股票"}); err != nil {
		t.Fatalf("fill: %v", err)
	}
	if mcp.calls[0].tool != "browser_fill" || mcp.calls[0].args["value"] != "股票" {
		t.Fatalf("fill args = %+v", mcp.calls[0].args)
	}
	if _, err := service.Browser(context.Background(), BrowserAction{Action: "press", Key: "Enter"}); err != nil {
		t.Fatalf("press: %v", err)
	}
	if mcp.calls[2].tool != "browser_press_key" || mcp.calls[2].args["key"] != "Enter" {
		t.Fatalf("press args = %+v", mcp.calls[2].args)
	}
}

func TestBrowserReadLinks(t *testing.T) {
	mcp := &recordingMCP{result: "- 行情中心 https://example.com/quote"}
	service := newBrowserService(mcp)
	out, err := service.Browser(context.Background(), BrowserAction{Action: "read", Direction: "links"})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(out, "行情中心") || mcp.calls[0].tool != "browser_links" {
		t.Fatalf("links read = %q / %+v", out, mcp.calls[0])
	}
}

func TestBrowserUnavailableWithoutMCP(t *testing.T) {
	service := New(Config{
		Enabled:    true,
		Engines:    []Engine{{Name: "bing", URL: "https://cn.bing.com/search?q={query}", Weight: 1}},
		MaxResults: 5,
	})
	if service.BrowserAvailable() {
		t.Fatal("should be unavailable without MCP")
	}
	if _, err := service.Browser(context.Background(), BrowserAction{Action: "open", URL: "https://example.com"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestBrowserUnknownAction(t *testing.T) {
	service := newBrowserService(&recordingMCP{})
	_, err := service.Browser(context.Background(), BrowserAction{Action: "type"})
	if err == nil || !strings.Contains(err.Error(), "不支持") {
		t.Fatalf("err = %v", err)
	}
}

// 保证 lockBrowser 在上下文取消时能退出
func TestBrowserLockHonoursContext(t *testing.T) {
	service := newBrowserService(&recordingMCP{})
	service.mcpMu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := service.Browser(ctx, BrowserAction{Action: "open", URL: "https://example.com"})
	service.mcpMu.Unlock()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
