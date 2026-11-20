package search

import (
	"context"
	"errors"
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
}

func (f *fakeMCP) HasTool(name string) bool {
	return name == "browser_navigate" || name == "browser_evaluate"
}

func (f *fakeMCP) CallTool(_ context.Context, name string, _ any) (string, error) {
	switch name {
	case "browser_navigate":
		f.navigate++
		if f.navigateErr != nil {
			return "", f.navigateErr
		}
		return "Navigated to search results", nil
	case "browser_evaluate":
		f.evaluate++
		if f.evaluateErr != nil {
			return "", f.evaluateErr
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
	if mcp.navigate != 1 || mcp.evaluate != 1 {
		t.Fatalf("navigate/evaluate = %d/%d, want 1/1", mcp.navigate, mcp.evaluate)
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
	if mcp.navigate != 1 || mcp.evaluate != 1 {
		t.Fatalf("navigate/evaluate = %d/%d, want 1/1", mcp.navigate, mcp.evaluate)
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
