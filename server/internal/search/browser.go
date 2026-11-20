package search

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// browserSnapshotChars 是回传给模型的页面文本预算。
// 太小会看不清页面结构，太大会把上下文吃掉，4000 字左右比较平衡。
const browserSnapshotChars = 4000

// browserActionTimeout 是单个浏览器动作的等待上限。
const browserActionTimeout = 20 * time.Second

// BrowserAction 描述一次浏览器交互。
type BrowserAction struct {
	Action    string // open / click / fill / press / scroll / back / read
	URL       string // open 使用
	Ref       string // snapshot 返回的元素引用，如 e3
	Selector  string // CSS 选择器，ref 不可用时使用
	Text      string // fill 输入的内容
	Key       string // press 的按键，如 Enter
	Direction string // scroll 方向，或 read 的模式（text/links/markdown）
}

// BrowserAvailable 判断当前是否能做浏览器交互。
func (s *Service) BrowserAvailable() bool {
	if s == nil || s.cfg.MCP == nil {
		return false
	}
	return s.cfg.MCP.HasTool("browser_navigate") && s.cfg.MCP.HasTool("browser_snapshot") && s.cfg.MCP.HasTool("browser_click")
}

// Browser 执行一次浏览器动作，并返回动作后的页面快照。
// 浏览器只有一个活动页面，所以所有动作串行执行。
func (s *Service) Browser(ctx context.Context, act BrowserAction) (string, error) {
	if !s.BrowserAvailable() {
		return "", errors.New("browser is not available")
	}
	if err := s.lockBrowser(ctx); err != nil {
		return "", err
	}
	defer s.mcpMu.Unlock()

	action := strings.ToLower(strings.TrimSpace(act.Action))
	switch action {
	case "open":
		target := strings.TrimSpace(act.URL)
		if target == "" {
			return "", errors.New("open 需要 url")
		}
		if _, err := s.cfg.MCP.CallTool(ctx, "browser_navigate", map[string]any{
			"url":       target,
			"waitUntil": "domcontentloaded",
		}); err != nil {
			return "", err
		}
	case "click":
		if err := s.browserTarget(ctx, act, "browser_click"); err != nil {
			return "", err
		}
	case "fill":
		if strings.TrimSpace(act.Text) == "" {
			return "", errors.New("fill 需要 text")
		}
		args := map[string]any{"value": act.Text}
		s.applyTarget(args, act)
		if _, err := s.cfg.MCP.CallTool(ctx, "browser_fill", args); err != nil {
			return "", err
		}
	case "press":
		key := strings.TrimSpace(act.Key)
		if key == "" {
			key = "Enter"
		}
		args := map[string]any{"key": key}
		if strings.TrimSpace(act.Selector) != "" {
			args["selector"] = act.Selector
		}
		if _, err := s.cfg.MCP.CallTool(ctx, "browser_press_key", args); err != nil {
			return "", err
		}
	case "scroll":
		direction := strings.ToLower(strings.TrimSpace(act.Direction))
		if direction == "" {
			direction = "down"
		}
		if _, err := s.cfg.MCP.CallTool(ctx, "browser_scroll", map[string]any{"direction": direction}); err != nil {
			return "", err
		}
	case "back":
		if _, err := s.cfg.MCP.CallTool(ctx, "browser_back", map[string]any{}); err != nil {
			return "", err
		}
	case "read":
		return s.browserRead(ctx, act)
	default:
		return "", fmt.Errorf("不支持的浏览器动作: %s", action)
	}
	return s.browserSnapshot(ctx)
}

// browserTarget 处理需要 ref/selector 的动作（click）。
func (s *Service) browserTarget(ctx context.Context, act BrowserAction, tool string) error {
	args := map[string]any{}
	s.applyTarget(args, act)
	if len(args) == 0 {
		return errors.New("click 需要 ref 或 selector")
	}
	_, err := s.cfg.MCP.CallTool(ctx, tool, args)
	return err
}

// applyTarget 把 ref / selector 填进参数，ref 优先。
func (s *Service) applyTarget(args map[string]any, act BrowserAction) {
	if ref := strings.TrimSpace(act.Ref); ref != "" {
		args["ref"] = ref
		return
	}
	if selector := strings.TrimSpace(act.Selector); selector != "" {
		args["selector"] = selector
	}
}

// browserRead 只读取当前页面，不做交互。
func (s *Service) browserRead(ctx context.Context, act BrowserAction) (string, error) {
	switch strings.ToLower(strings.TrimSpace(act.Direction)) {
	case "links":
		raw, err := s.cfg.MCP.CallTool(ctx, "browser_links", map[string]any{"limit": 40})
		if err != nil {
			return "", err
		}
		return trimRunes(mcpStringValue(raw), browserSnapshotChars), nil
	case "markdown":
		raw, err := s.cfg.MCP.CallTool(ctx, "browser_markdown", map[string]any{"max_chars": browserSnapshotChars})
		if err != nil {
			return "", err
		}
		return trimRunes(mcpStringValue(raw), browserSnapshotChars), nil
	default:
		return s.browserSnapshot(ctx)
	}
}

// browserSnapshot 取页面快照：既包含可见正文，也包含可点击元素的 ref。
func (s *Service) browserSnapshot(ctx context.Context) (string, error) {
	raw, err := s.cfg.MCP.CallTool(ctx, "browser_snapshot", map[string]any{
		"max_chars": browserSnapshotChars,
	})
	if err != nil {
		return "", err
	}
	return trimRunes(mcpStringValue(raw), browserSnapshotChars*2), nil
}

// lockBrowser 争用唯一的浏览器页面；等待而不是立刻失败，
// 因为点击/输入必须按顺序发生在同一个页面上。
func (s *Service) lockBrowser(ctx context.Context) error {
	deadline := time.Now().Add(browserActionTimeout)
	for !s.mcpMu.TryLock() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Now().After(deadline) {
			return errors.New("浏览器正忙，请稍后再试")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(120 * time.Millisecond):
		}
	}
	return nil
}

// trimRunes 按字符截断，避免把半个 UTF-8 字符返回给模型。
func trimRunes(text string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(text) <= limit {
		return text
	}
	runes := []rune(text)
	return string(runes[:limit]) + "\n…（内容已截断）"
}
