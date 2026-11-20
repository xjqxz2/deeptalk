package mcp

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

func TestObscuraBrowserFetchLive(t *testing.T) {
	bin := os.Getenv("OBSCURA_BIN")
	if bin == "" {
		t.Skip("OBSCURA_BIN is not set")
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	client, err := Start(context.Background(), Config{
		Command: bin, Args: []string{"mcp"},
		ProtocolVersion: DefaultProtocolVersion, Timeout: 30 * time.Second,
	}, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	out, err := client.CallTool(ctx, "browser_navigate", map[string]any{"url": "https://cn.bing.com/search?q=DeepSeek+V4.1+Flash&mkt=zh-CN&setlang=zh-Hans&cc=CN", "waitUntil": "domcontentloaded"})
	if err != nil {
		t.Fatalf("navigate: %v", err)
	}
	t.Logf("navigate: %s", out)
	html, err := client.CallTool(ctx, "browser_evaluate", map[string]any{"expression": "document.documentElement.outerHTML", "timeoutMs": 8000})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !strings.Contains(html, "<html") {
		t.Fatalf("not html: %.200s", html)
	}
	t.Logf("html bytes: %d", len(html))
}
