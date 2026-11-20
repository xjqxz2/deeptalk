package search

import "testing"

// TestNormalizeQueryStripsFillers 校验口语词会被去掉，时间限定词会保留。
func TestNormalizeQueryStripsFillers(t *testing.T) {
	cases := map[string]string{
		"现在股市在涨的股票":          "股市在涨的股票",
		"现在 沪深股市 涨幅榜":        "沪深股市 涨幅榜",
		"请问今天天气怎么样":          "今天天气怎么样",
		"帮我查一下 DeepSeek 价格":  "DeepSeek 价格",
		"能不能告诉我 2026 年春节是哪天": "2026 年春节是哪天",
		"今天深圳天气":             "今天深圳天气",
		"":                   "",
	}
	for input, want := range cases {
		if got := normalizeQuery(input); got != want {
			t.Fatalf("normalizeQuery(%q) = %q, want %q", input, got, want)
		}
	}
}

// TestNormalizeQueryKeepsMeaningfulTime 确保“今天/最近”这类限定词不被当成填充词删掉。
func TestNormalizeQueryKeepsMeaningfulTime(t *testing.T) {
	for _, query := range []string{"今天 新闻", "最近 有什么新政策", "本周 A股 行情"} {
		got := normalizeQuery(query)
		if got != query {
			t.Fatalf("normalizeQuery(%q) = %q, 时间限定词被误删", query, got)
		}
	}
}
