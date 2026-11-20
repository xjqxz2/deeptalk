package search

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
)

type fakeMCP struct {
	html        string
	navigate    int
	evaluate    int
	navigateErr error
	evaluateErr error
	host        string // 最近一次导航目标的 host，用于回答 location.host
}

func (f *fakeMCP) HasTool(name string) bool {
	return name == "browser_navigate" || name == "browser_evaluate"
}

func (f *fakeMCP) CallTool(_ context.Context, name string, args any) (string, error) {
	switch name {
	case "browser_navigate":
		f.navigate++
		if f.navigateErr != nil {
			return "", f.navigateErr
		}
		if values, ok := args.(map[string]any); ok {
			if raw, ok := values["url"].(string); ok {
				if parsed, err := url.Parse(raw); err == nil {
					f.host = parsed.Host
				}
			}
		}
		return "Navigated to search results", nil
	case "browser_evaluate":
		f.evaluate++
		if f.evaluateErr != nil {
			return "", f.evaluateErr
		}
		if values, ok := args.(map[string]any); ok && values["expression"] == "location.host" {
			return f.host, nil
		}
		return f.html, nil
	default:
		return "", errors.New("unexpected tool")
	}
}

const bingFixture = `<!doctype html><html><body>
<li class="b_algo">
  <h2><a href="https://example.com/deepseek-v4">DeepSeek V4 价格公布</a></h2>
  <div class="b_caption"><p>DeepSeek V4 价格与发布信息。</p></div>
</li>
<li class="b_algo">
  <h2><a href="https://example.com/weather">今日天气</a></h2>
  <div class="b_caption"><p>与问题无关。</p></div>
</li>
</body></html>`

func TestFetchWithMCPUsesNavigateAndEvaluate(t *testing.T) {
	mcp := &fakeMCP{html: bingFixture}
	service := New(Config{
		Enabled:    true,
		EngineURL:  "https://example.invalid/?q={query}",
		MaxResults: 5,
		Timeout:    time.Second,
		MCP:        mcp,
	})
	raw, err := service.fetchWithMCP(context.Background(), "https://cn.bing.com/search?q=test")
	if err != nil {
		t.Fatalf("fetchWithMCP: %v", err)
	}
	if string(raw) != bingFixture {
		t.Fatalf("unexpected html")
	}
	if mcp.navigate != 1 || mcp.evaluate != 2 {
		t.Fatalf("navigate/evaluate = %d/%d, want 1/2", mcp.navigate, mcp.evaluate)
	}
}

func TestSearchUsesMCPRenderedHTML(t *testing.T) {
	mcp := &fakeMCP{html: bingFixture}
	service := New(Config{
		Enabled:   true,
		EngineURL: "https://cn.bing.com/search?q={query}",
		Engines: []Engine{{
			Name:   "bing",
			URL:    "https://cn.bing.com/search?q={query}",
			Weight: 1,
		}},
		MaxResults: 5,
		Timeout:    time.Second,
		MCP:        mcp,
	})
	items, err := service.Search(context.Background(), "DeepSeek V4 价格")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if mcp.navigate != 1 || mcp.evaluate != 2 {
		t.Fatalf("navigate/evaluate = %d/%d, want 1/2", mcp.navigate, mcp.evaluate)
	}
	if len(items) != 1 || items[0].URL != "https://example.com/deepseek-v4" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestFetchPageFallsBackWhenMCPFails(t *testing.T) {
	mcp := &fakeMCP{navigateErr: errors.New("browser failed")}
	service := New(Config{
		Enabled:      true,
		EngineURL:    "https://example.invalid/?q={query}",
		MaxResults:   5,
		Timeout:      100 * time.Millisecond,
		HTTPFallback: false,
		MCP:          mcp,
	})
	_, err := service.fetchPage(context.Background(), "https://example.invalid/search")
	if err == nil || !strings.Contains(err.Error(), "mcp:") {
		t.Fatalf("expected mcp error, got %v", err)
	}
}
