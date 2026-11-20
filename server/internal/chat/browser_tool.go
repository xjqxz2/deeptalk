package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"deeptalk/internal/provider"
	"deeptalk/internal/search"
)

// browserActionsPerTurn 是一次回答里浏览器动作的上限，
// 防止模型在同一个页面里反复点击打转。
const browserActionsPerTurn = 40

// runBrowserTool 执行模型请求的浏览器动作。
// 每个动作都会回传最新页面快照，模型据此决定下一步点击哪个元素。
func (s *Service) runBrowserTool(ctx context.Context, call provider.ToolCall, emitActivity func(status, label, detail, url string)) (string, error) {
	if s.search == nil || !s.search.BrowserAvailable() {
		return `{"error":"browser is not available"}`, nil
	}
	var args struct {
		Action    string `json:"action"`
		URL       string `json:"url"`
		Ref       string `json:"ref"`
		Selector  string `json:"selector"`
		Text      string `json:"text"`
		Key       string `json:"key"`
		Direction string `json:"direction"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return "", fmt.Errorf("invalid browser arguments: %w", err)
	}
	action := strings.ToLower(strings.TrimSpace(args.Action))
	label := browserActionLabel(action, args.URL)
	emitActivity("reading", label, browserActionDetail(args), args.URL)

	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := s.search.Browser(runCtx, search.BrowserAction{
		Action:    action,
		URL:       args.URL,
		Ref:       args.Ref,
		Selector:  args.Selector,
		Text:      args.Text,
		Key:       args.Key,
		Direction: args.Direction,
	})
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		emitActivity("found", "浏览器操作失败", err.Error(), "")
		return fmt.Sprintf(`{"error":%q}`, err.Error()), nil
	}
	emitActivity("found", "已更新页面", "", "")
	payload, marshalErr := json.Marshal(map[string]any{
		"action": action,
		"page":   out,
		"note":   "这是操作后的页面快照；请用其中的 ref 继续操作，或直接回答用户。",
	})
	if marshalErr != nil {
		return out, nil
	}
	return string(payload), nil
}

// browserActionLabel 生成思考链里展示的一句话。
func browserActionLabel(action, target string) string {
	switch action {
	case "open":
		return "正在打开网页…"
	case "click":
		return "正在点击页面元素…"
	case "fill":
		return "正在填写表单…"
	case "press":
		return "正在提交…"
	case "scroll":
		return "正在向下翻阅页面…"
	case "back":
		return "正在返回上一页…"
	case "read":
		return "正在阅读当前页面…"
	default:
		return "正在操作浏览器…"
	}
}

// browserActionDetail 给活动提示补充一点上下文。
func browserActionDetail(args struct {
	Action    string `json:"action"`
	URL       string `json:"url"`
	Ref       string `json:"ref"`
	Selector  string `json:"selector"`
	Text      string `json:"text"`
	Key       string `json:"key"`
	Direction string `json:"direction"`
}) string {
	switch args.Action {
	case "open":
		return args.URL
	case "click":
		if args.Ref != "" {
			return args.Ref
		}
		return args.Selector
	case "fill":
		return truncateRunes(args.Text, 40)
	case "press":
		return args.Key
	default:
		return ""
	}
}
