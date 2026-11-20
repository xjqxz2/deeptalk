package chat

import "testing"

// TestCleanTitle 校验模型输出会被清洗成可直接展示的短标题。
func TestCleanTitle(t *testing.T) {
	cases := map[string]string{
		"零信任组网方案":         "零信任组网方案",
		"  “零信任组网方案”。 ":   "零信任组网方案",
		"标题：小红书商品链接":      "小红书商品链接",
		"第一行标题\n多余的解释":    "第一行标题 多余的解释",
		"《DeepSeek 使用技巧》": "DeepSeek 使用技巧",
		"":                "",
		"   ":             "",
	}
	for input, want := range cases {
		if got := cleanTitle(input); got != want {
			t.Fatalf("cleanTitle(%q) = %q, want %q", input, got, want)
		}
	}
	// 过长标题会被压到 24 字
	long := cleanTitle("这是一个非常非常非常非常非常非常非常长的标题需要被截断处理")
	if len([]rune(long)) != 24 {
		t.Fatalf("long title length = %d, want 24", len([]rune(long)))
	}
}

// TestDefaultTitleSeed 校验标题种子只取前 400 字，图片对话有兜底文案。
func TestDefaultTitleSeed(t *testing.T) {
	if got := defaultTitleSeed("  ", true); got != "图片对话" {
		t.Fatalf("image seed = %q", got)
	}
	if got := defaultTitleSeed(" 你好 ", false); got != "你好" {
		t.Fatalf("text seed = %q", got)
	}
	long := make([]rune, 600)
	for i := range long {
		long[i] = '字'
	}
	if got := defaultTitleSeed(string(long), false); len([]rune(got)) != 400 {
		t.Fatalf("long seed length = %d, want 400", len([]rune(got)))
	}
	if !isDefaultTitle("新对话") || !isDefaultTitle(" ") || isDefaultTitle("已有标题") {
		t.Fatal("isDefaultTitle mismatch")
	}
}
