package search

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"deeptalk/internal/mcp"
)

func TestObscuraMCPBrowserSearchLive(t *testing.T) {
	bin := os.Getenv("OBSCURA_BIN")
	if bin == "" {
		t.Skip("OBSCURA_BIN is not set")
	}
	client, err := mcp.Start(context.Background(), mcp.Config{
		Command: bin, Args: []string{"mcp"}, ProtocolVersion: "2024-11-05", Timeout: 30 * time.Second,
	}, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	service := New(Config{
		Enabled:    true,
		Engines:    []Engine{{Name: "bing", URL: "https://cn.bing.com/search?q={query}&mkt=zh-CN&setlang=zh-Hans&cc=CN", Weight: 1}},
		MaxResults: 5, Timeout: 25 * time.Second, MCP: client,
	})
	start := time.Now()
	items, err := service.Search(context.Background(), "DeepSeek V4.1 Flash")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("no results")
	}
	t.Logf("got %d results in %s; first=%s", len(items), time.Since(start), items[0].URL)
}
