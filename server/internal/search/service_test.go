package search

import "testing"

func TestParseDuckDuckGo(t *testing.T) {
	html := `<!doctype html><html><body>
	<div class="result">
		<a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fa%3Fx%3D1&amp;rut=1">Example A</a>
		<a class="result__snippet">First result snippet.</a>
	</div>
	<div class="result">
		<a class="result__a" href="https://example.org/b">Example B</a>
		<div class="result__snippet">Second result <b>snippet</b>.</div>
	</div>
	<div class="result result--ad">
		<a class="result__a" href="https://ads.example.net">Ad</a>
	</div>
	</body></html>`

	items := parse([]byte(html), 5)
	if len(items) != 2 {
		t.Fatalf("got %d results, want 2", len(items))
	}
	if items[0].Title != "Example A" || items[0].URL != "https://example.com/a?x=1" {
		t.Fatalf("unexpected first result: %+v", items[0])
	}
	if items[1].URL != "https://example.org/b" {
		t.Fatalf("unexpected second result: %+v", items[1])
	}
}

func TestParseBing(t *testing.T) {
	html := `<html><body><ol id="b_results">
	<li class="b_algo"><h2><a href="https://example.com/bing">Bing result</a></h2>
	<div class="b_caption"><p>Bing snippet.</p></div></li>
	</ol></body></html>`

	items := parse([]byte(html), 5)
	if len(items) != 1 {
		t.Fatalf("got %d results, want 1", len(items))
	}
	if items[0].URL != "https://example.com/bing" || items[0].Snippet != "Bing snippet." {
		t.Fatalf("unexpected result: %+v", items[0])
	}
}

func TestResolveHrefRejectsNonHTTP(t *testing.T) {
	if got := resolveHref("javascript:alert(1)"); got != "" {
		t.Fatalf("javascript href should be rejected, got %q", got)
	}
	if got := resolveHref("https://example.com/a"); got != "https://example.com/a" {
		t.Fatalf("unexpected href: %q", got)
	}
}

func TestBuildURL(t *testing.T) {
	s := &Service{cfg: Config{EngineURL: "https://html.duckduckgo.com/html/?q={query}"}}
	if got := s.buildURL("hello world"); got != "https://html.duckduckgo.com/html/?q=hello+world" {
		t.Fatalf("unexpected url: %q", got)
	}
	s.cfg.EngineURL = "https://example.com/search"
	if got := s.buildURL("a&b"); got != "https://example.com/search?q=a%26b" {
		t.Fatalf("unexpected url: %q", got)
	}
}

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

func TestParseMultiEngines(t *testing.T) {
	sogou := `<html><body><div class="vrwrap"><h3><a href="https://sogou.example/a">Sogou A</a></h3><div class="str-text-info">Sogou snippet</div></div></body></html>`
	if items := parseEngine("sogou", []byte(sogou), 5, "https://www.sogou.com/web"); len(items) != 1 || items[0].URL != "https://sogou.example/a" {
		t.Fatalf("sogou parse failed: %+v", items)
	}
	baidu := `<html><body><div class="result c-container"><h3><a href="https://baidu.example/a">Baidu A</a></h3><div class="c-abstract">Baidu snippet</div></div></body></html>`
	if items := parseEngine("baidu", []byte(baidu), 5, "https://www.baidu.com/s"); len(items) != 1 || items[0].Snippet != "Baidu snippet" {
		t.Fatalf("baidu parse failed: %+v", items)
	}
	zhihu := `<html><body><div class="SearchResult-Card"><h2><a href="https://zhihu.example/a">Zhihu A</a></h2><div class="RichText">Zhihu snippet</div></div></body></html>`
	if items := parseEngine("zhihu", []byte(zhihu), 5, "https://www.zhihu.com/search"); len(items) != 1 || items[0].Title != "Zhihu A" {
		t.Fatalf("zhihu parse failed: %+v", items)
	}
}

func TestCanonicalURL(t *testing.T) {
	got := canonicalURL("https://Example.com/a/?utm_source=x&b=2&a=1#frag")
	if got != "https://example.com/a?a=1&b=2" {
		t.Fatalf("canonicalURL = %q", got)
	}
	if canonicalURL("javascript:alert(1)") != "" {
		t.Fatal("javascript URL should be rejected")
	}
}
