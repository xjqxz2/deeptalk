package search

import "testing"

// TestNormalizeQuery 校验口语前缀不会污染搜索引擎关键词。
func TestNormalizeQuery(t *testing.T) {
	cases := map[string]string{
		"联网查一下关于抽烟的危害":     "抽烟的危害",
		"帮我搜索一下 Go SSE 超时": "Go SSE 超时",
		"搜索 DeepSeek":      "DeepSeek",
		"抽烟的危害":            "抽烟的危害",
		"  hi  ":           "hi",
	}
	for input, want := range cases {
		if got := normalizeQuery(input); got != want {
			t.Fatalf("normalizeQuery(%q) = %q, want %q", input, got, want)
		}
	}
}
