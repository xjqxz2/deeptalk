package chat

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"deeptalk/internal/search"
)

// TestGenerateTitle 校验兜底标题的前缀清理与长度上限。
// 正常情况下会用模型生成的语义标题覆盖它，这里只保证不会被生硬截断得太短。
func TestGenerateTitle(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"帮我把这段 SSE 代理的超时与重连逻辑改成可配置的", "这段 SSE 代理的超时与重连逻辑改成可配置的"},
		{"请帮我设计一个 Go JWT 中间件", "设计一个 Go JWT 中间件"},
		{"", "新对话"},
	}
	for _, item := range cases {
		got := GenerateTitle(item.input)
		if got != item.want {
			t.Fatalf("GenerateTitle(%q) = %q, want %q", item.input, got, item.want)
		}
		if len([]rune(got)) > 24 {
			t.Fatalf("title too long: %d runes", len([]rune(got)))
		}
	}
}

// TestDemoAnswer 校验演示回答在关键场景下包含代码块与思考内容。
func TestDemoAnswer(t *testing.T) {
	reasoning, content := demoAnswer("帮我优化 SSE 代理", true)
	if reasoning == "" {
		t.Fatal("deep thinking should produce reasoning")
	}
	if !strings.Contains(content, "```go") {
		t.Fatal("SSE answer should contain a go code block")
	}
	if _, plain := demoAnswer("随便聊聊", false); strings.Contains(plain, "```") {
		t.Fatal("plain answer should not contain code fences")
	}
}

// TestEstimateTokens 校验 tokens 估算对中文与空串的处理。
func TestEstimateTokens(t *testing.T) {
	if estimateTokens("") != 0 {
		t.Fatal("empty text should be 0 tokens")
	}
	if got := estimateTokens("你好世界"); got != 3 {
		t.Fatalf("estimateTokens = %d, want 3", got)
	}
}

// TestShouldSearch 校验联网搜索的轻量意图判断：
// 纯问候/短句不发起检索，实质问题才允许检索。
func TestShouldSearch(t *testing.T) {
	cases := []struct {
		query string
		want  bool
	}{
		{"你好呀", false},
		{"你好", false},
		{"hi", false},
		{"谢谢", false},
		{"你好呀，今天天气怎么样？", true},
		{"Go 1.26 有哪些新特性？", true},
		{"帮我对比 SSE 和 WebSocket", true},
	}
	for _, item := range cases {
		if got := shouldSearch(item.query); got != item.want {
			t.Fatalf("shouldSearch(%q) = %v, want %v", item.query, got, item.want)
		}
		if got := shouldThink(item.query); got != item.want {
			t.Fatalf("shouldThink(%q) = %v, want %v", item.query, got, item.want)
		}
	}
}

// TestSearchToolPayloadKeepsCitationIDsStable 锁定 sources[].id 与 sources_json 的顺序。
func TestSearchToolPayloadKeepsCitationIDsStable(t *testing.T) {
	raw := searchToolPayload([]search.Result{
		{Title: "A", URL: "https://example.com/a", Snippet: "first"},
		{Title: "B", URL: "https://example.com/b", Snippet: "second"},
	}, map[string]any{"pages": []search.Page{}})
	var payload struct {
		Sources []struct {
			ID  int    `json:"id"`
			URL string `json:"url"`
		} `json:"sources"`
		CitationRule string `json:"citation_rule"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if len(payload.Sources) != 2 || payload.Sources[0].ID != 1 || payload.Sources[1].ID != 2 {
		t.Fatalf("unexpected sources: %+v", payload.Sources)
	}
	if !strings.Contains(payload.CitationRule, "[1]") {
		t.Fatalf("missing citation rule: %q", payload.CitationRule)
	}
}

// TestModelErrorMessage 校验上游明确原因不会被折叠成“生成失败”。
func TestModelErrorMessage(t *testing.T) {
	cases := []struct {
		err   string
		model string
		want  string
	}{
		{
			err:   `upstream request failed: 上游服务异常（HTTP 503）：This model is temporarily unavailable. Please try again later.`,
			model: "MaikuaiDS41F",
			want:  "MaikuaiDS41F：上游模型暂时不可用，请稍后重试或切换到其它模型",
		},
		{
			err:   `upstream request failed: 上游返回错误（HTTP 400）：The model 'x' does not exist or is not available.`,
			model: "TestModel",
			want:  "TestModel：上游不存在该模型，请管理员检查模型 ID",
		},
		{
			err:   "something unexpected",
			model: "M",
			want:  "M：生成回答失败，请重试",
		},
	}
	for _, item := range cases {
		got := modelErrorMessage(errors.New(item.err), item.model)
		if got != item.want {
			t.Fatalf("modelErrorMessage(%q) = %q, want %q", item.err, got, item.want)
		}
	}
}
