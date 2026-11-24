package memo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// userIDKey 是从认证中间件传递到 MCP handler 的用户标识。
type userIDKey struct{}

// WithUser 把当前用户写进 request context，供 MCP handler 取用。
func WithUser(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// userFrom 取出请求里的用户；缺失时返回 0。
func userFrom(ctx context.Context) uint {
	if value, ok := ctx.Value(userIDKey{}).(uint); ok {
		return value
	}
	return 0
}

// Handler 返回一个挂在认证中间件之后的 MCP 端点（Streamable HTTP）。
// 外部 Agent（DSH / Claude Desktop 等）连上后即可写读灵感备忘录。
func Handler(service *Service) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		server := mcp.NewServer(&mcp.Implementation{Name: "deeptalk-memo", Version: "v1"}, nil)
		registerTools(server, service, userFrom(request.Context()))
		return server
	}, nil)
}

// registerTools 注册备忘录的四个工具。
func registerTools(server *mcp.Server, service *Service, userID uint) {
	server.AddTool(&mcp.Tool{
		Name:        "memo_save",
		Description: "把主人刚才提到的灵感/想法整理成一条备忘录保存下来。适合「记一下」「帮我记住」「这个想法不错」这类请求。",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title":   map[string]any{"type": "string", "description": "一句话主题，不超过 60 字"},
				"content": map[string]any{"type": "string", "description": "整理后的正文，保留关键细节与结论"},
				"tags":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "可选标签"},
			},
			"required": []string{"content"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Title   string   `json:"title"`
			Content string   `json:"content"`
			Tags    []string `json:"tags"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("参数不合法: %w", err)
		}
		item, err := service.Save(ctx, userID, SaveInput{Title: args.Title, Content: args.Content, Tags: args.Tags})
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{
			Text: fmt.Sprintf(`{"saved":true,"id":%d,"title":%q}`, item.ID, item.Title),
		}}}, nil
	})

	server.AddTool(&mcp.Tool{
		Name:        "memo_search",
		Description: "按关键词检索主人之前记下的灵感备忘录；关键词为空时返回最近的几条。",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "关键词；留空表示最近记录"},
				"limit": map[string]any{"type": "integer", "description": "返回条数，默认 10，最多 50"},
			},
			"required": []string{},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}
		_ = json.Unmarshal(request.Params.Arguments, &args)
		items, err := service.Search(ctx, userID, args.Query, args.Limit)
		if err != nil {
			return nil, err
		}
		raw, _ := json.Marshal(map[string]any{"items": items})
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}}}, nil
	})

	server.AddTool(&mcp.Tool{
		Name:        "memo_delete",
		Description: "删除一条备忘录（按 ID）。",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"id": map[string]any{"type": "integer", "description": "备忘录 ID"}},
			"required":   []string{"id"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			ID uint `json:"id"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("参数不合法: %w", err)
		}
		if err := service.Delete(ctx, userID, args.ID); err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: `{"deleted":true}`}}}, nil
	})
}
