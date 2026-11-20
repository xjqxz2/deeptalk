package chat

import (
	"strings"
	"testing"
)

// 生产环境里出现的真实 DSML 片段（全角竖线）
const dsmlSample = `我来查一下目前各家生图模型的定价和口碑。<｜｜DSML｜｜ calls>
<｜｜DSML｜｜ invoke name="web_search">
<｜｜DSML｜｜ parameter name="query" string="true">通义万相 豆包 Seedream API 价格 每张 2026 硅基流动</｜｜DSML｜｜ parameter>
</｜｜DSML｜｜ invoke>
</｜｜DSML｜｜ calls>`

func TestParseDSMLCalls(t *testing.T) {
	calls, clean := ParseDSMLCalls(dsmlSample)
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	if calls[0].Name != "web_search" {
		t.Fatalf("name = %q", calls[0].Name)
	}
	if !strings.Contains(calls[0].Arguments, "通义万相") {
		t.Fatalf("arguments = %s", calls[0].Arguments)
	}
	if strings.Contains(clean, "DSML") || strings.Contains(clean, "invoke") {
		t.Fatalf("markup leaked: %q", clean)
	}
	if !strings.HasPrefix(clean, "我来查一下") {
		t.Fatalf("clean text = %q", clean)
	}
}

func TestParseDSMLCallsNumberParam(t *testing.T) {
	text := `<｜｜DSML｜｜ invoke name="open_url">
<｜｜DSML｜｜ parameter name="count" string="false">3</｜｜DSML｜｜ parameter>
</｜｜DSML｜｜ invoke>`
	calls, _ := ParseDSMLCalls(text)
	if len(calls) != 1 || calls[0].Arguments != `{"count":3}` {
		t.Fatalf("calls = %+v", calls)
	}
}

// TestDSMLStreamHidesMarkup 校验流式输出不会把标记漏给用户，并且能拆出调用。
func TestDSMLStreamHidesMarkup(t *testing.T) {
	var out strings.Builder
	filter := newDSMLStream(func(_ string, delta string) error {
		out.WriteString(delta)
		return nil
	})
	// 逐字符写入，模拟真实 token 切分
	for _, r := range dsmlSample {
		if err := filter.Write(string(r)); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	calls := filter.Finish()
	if len(calls) != 1 || calls[0].Name != "web_search" {
		t.Fatalf("calls = %+v", calls)
	}
	if strings.Contains(out.String(), "DSML") {
		t.Fatalf("markup leaked to client: %q", out.String())
	}
	if out.String() != "我来查一下目前各家生图模型的定价和口碑。" {
		t.Fatalf("visible text = %q", out.String())
	}
	if filter.Spoken() != out.String() {
		t.Fatalf("spoken mismatch: %q", filter.Spoken())
	}
}

// TestDSMLStreamPlainText 普通文本必须原样透传，不受过滤影响。
func TestDSMLStreamPlainText(t *testing.T) {
	var out strings.Builder
	filter := newDSMLStream(func(_ string, delta string) error {
		out.WriteString(delta)
		return nil
	})
	parts := []string{"普通回答，", "带 `code`", " 和 <html> 标签。"}
	for _, p := range parts {
		if err := filter.Write(p); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	if calls := filter.Finish(); len(calls) != 0 {
		t.Fatalf("unexpected calls: %+v", calls)
	}
	if out.String() != strings.Join(parts, "") {
		t.Fatalf("text = %q", out.String())
	}
}

func TestSanitizeDSML(t *testing.T) {
	if got := SanitizeDSML("普通文本"); got != "普通文本" {
		t.Fatalf("plain text changed: %q", got)
	}
	if got := SanitizeDSML(dsmlSample); strings.Contains(got, "DSML") {
		t.Fatalf("markup left: %q", got)
	}
}
