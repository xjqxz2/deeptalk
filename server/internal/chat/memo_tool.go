package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"deeptalk/internal/memo"
	"deeptalk/internal/provider"
)

// runMemoTool 执行灵感录工具：把对话中的灵感写下来，或检索已有记录。
func (s *Service) runMemoTool(ctx context.Context, userID uint, call provider.ToolCall, conversationID uint) (string, error) {
	if s.cfg.Memos == nil {
		return `{"error":"灵感录未启用"}`, nil
	}
	if userID == 0 {
		return `{"error":"缺少用户上下文"}`, nil
	}
	switch call.Name {
	case "memo_save":
		var args struct {
			Title   string   `json:"title"`
			Content string   `json:"content"`
			Tags    []string `json:"tags"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", fmt.Errorf("invalid memo_save arguments: %w", err)
		}
		item, err := s.cfg.Memos.Save(ctx, userID, memo.SaveInput{
			Title: args.Title, Content: args.Content, Tags: args.Tags, ConversationID: conversationID,
		})
		if err != nil {
			return fmt.Sprintf(`{"error":%q}`, err.Error()), nil
		}
		raw, _ := json.Marshal(map[string]any{
			"saved": true, "id": item.ID, "title": item.Title,
			"note": "已保存到主人的小橘灵感录，回复时用一句话确认即可。",
		})
		return string(raw), nil
	default:
		var args struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}
		_ = json.Unmarshal([]byte(call.Arguments), &args)
		limit := args.Limit
		if limit <= 0 {
			limit = 5
		}
		items, err := s.cfg.Memos.Search(ctx, userID, strings.TrimSpace(args.Query), limit)
		if err != nil {
			return fmt.Sprintf(`{"error":%q}`, err.Error()), nil
		}
		raw, _ := json.Marshal(map[string]any{"items": items})
		return string(raw), nil
	}
}
