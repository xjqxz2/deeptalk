// Package search 提供联网检索能力。
// 检索页面由 Obscura 无头浏览器抓取，Obscura 不可用时可按配置回退到普通 HTTP；
// 返回结果只包含标题、链接与摘要，不把原始 HTML 暴露给上层。
package search

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

// 业务错误。
var (
	ErrDisabled = errors.New("web search is disabled")
	ErrNoResult = errors.New("no search result")
)

// Result 是一条检索结果。
type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// Engine 是一路搜索来源。
type Engine struct {
	Name   string  // bing / zhihu / sogou / baidu
	URL    string  // 搜索页模板，支持 {query}
	Weight float64 // RRF 融合权重
}

// Config 是检索服务的可配置项。
type Config struct {
	Enabled      bool          // 是否启用联网搜索
	ObscuraBin   string        // Obscura 可执行文件路径或 PATH 中的命令名
	EngineURL    string        // 兼容旧配置：单引擎搜索页模板
	Engines      []Engine      // 多引擎配置；非空时优先使用
	MaxResults   int           // 最多返回条数
	Timeout      time.Duration // 单次检索超时
	HTTPFallback bool          // Obscura 失败时是否允许普通 HTTP 抓取
	Stealth      bool          // 是否启用 Obscura 反检测模式
	UserAgent    string        // HTTP 回退时使用的浏览器 UA
}

// Service 是联网检索服务。
type Service struct {
	cfg    Config
	client *http.Client
}

// New 创建检索服务；配置不完整时返回 nil，调用方应按“未启用”处理。
func New(cfg Config) *Service {
	if !cfg.Enabled || (strings.TrimSpace(cfg.EngineURL) == "" && len(cfg.Engines) == 0) {
		return nil
	}
	if len(cfg.Engines) == 0 && strings.TrimSpace(cfg.EngineURL) != "" {
		cfg.Engines = []Engine{{Name: "bing", URL: cfg.EngineURL, Weight: 1}}
	}
	cleaned := make([]Engine, 0, len(cfg.Engines))
	for _, engine := range cfg.Engines {
		if strings.TrimSpace(engine.URL) == "" {
			continue
		}
		if strings.TrimSpace(engine.Name) == "" {
			engine.Name = "search"
		}
		if engine.Weight <= 0 {
			engine.Weight = 1
		}
		cleaned = append(cleaned, engine)
	}
	cfg.Engines = cleaned
	if len(cfg.Engines) == 0 {
		return nil
	}
	if cfg.MaxResults <= 0 {
		cfg.MaxResults = 5
	}
	if cfg.MaxResults > 8 {
		cfg.MaxResults = 8
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 25 * time.Second
	}
	if cfg.Timeout > 45*time.Second {
		cfg.Timeout = 45 * time.Second
	}
	if strings.TrimSpace(cfg.ObscuraBin) == "" {
		cfg.ObscuraBin = "obscura"
	}
	if strings.TrimSpace(cfg.UserAgent) == "" {
		cfg.UserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 " +
			"(KHTML, like Gecko) Chrome/124.0 Safari/537.36 DeepTalk/1.0"
	}
	return &Service{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

// Enabled 返回检索是否可用；页面可据此决定入口是否可点。
func (s *Service) Enabled() bool {
	return s != nil && s.cfg.Enabled
}

// Search 并发检索多个来源，用加权 RRF（Reciprocal Rank Fusion）融合排序。
// 单引擎失败不会拖垮整体；只有所有引擎都没有结果时才返回错误。
func (s *Service) Search(ctx context.Context, query string) ([]Result, error) {
	return s.searchWithEngines(ctx, query, nil)
}

// SearchWithEngines 只检索模型指定的来源；names 为空时使用全部默认来源。
func (s *Service) SearchWithEngines(ctx context.Context, query string, names []string) ([]Result, error) {
	if len(names) == 0 {
		return s.searchWithEngines(ctx, query, nil)
	}
	wanted := make(map[string]struct{}, len(names))
	for _, name := range names {
		wanted[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
	}
	filtered := make([]Engine, 0, len(names))
	for _, engine := range s.cfg.Engines {
		if _, ok := wanted[strings.ToLower(engine.Name)]; ok {
			filtered = append(filtered, engine)
		}
	}
	if len(filtered) == 0 {
		return s.searchWithEngines(ctx, query, nil)
	}
	return s.searchWithEngines(ctx, query, filtered)
}

func (s *Service) searchWithEngines(ctx context.Context, query string, engines []Engine) ([]Result, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	query = normalizeQuery(query)
	if query == "" {
		return nil, errors.New("search query is empty")
	}
	runes := []rune(query)
	if len(runes) > 200 {
		query = string(runes[:200])
	}
	searchCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()

	type engineResult struct {
		index  int
		engine Engine
		items  []Result
		err    error
	}
	if len(engines) == 0 {
		engines = s.cfg.Engines
	}
	results := make(chan engineResult, len(engines))
	sem := make(chan struct{}, 2) // 同时最多两个 Obscura，避免容器内存峰值过高
	var wg sync.WaitGroup
	for index, engine := range engines {
		target := buildEngineURL(engine.URL, query)
		if target == "" {
			continue
		}
		wg.Add(1)
		go func(index int, engine Engine, target string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-searchCtx.Done():
				results <- engineResult{index: index, engine: engine, err: searchCtx.Err()}
				return
			}
			items, err := s.searchEngine(searchCtx, engine, target)
			results <- engineResult{index: index, engine: engine, items: items, err: err}
		}(index, engine, target)
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	type candidate struct {
		result    Result
		score     float64
		firstSeen int
	}
	merged := make(map[string]*candidate)
	var firstErr error
	engineCount := 0
	for item := range results {
		if item.err != nil {
			if firstErr == nil {
				firstErr = item.err
			}
			continue
		}
		engineCount++
		for rank, result := range item.items {
			key := canonicalURL(result.URL)
			if key == "" {
				continue
			}
			score := item.engine.Weight / (60 + float64(rank+1))
			if current, ok := merged[key]; ok {
				current.score += score
				if current.result.Snippet == "" && result.Snippet != "" {
					current.result.Snippet = result.Snippet
				}
				if current.result.Title == "" && result.Title != "" {
					current.result.Title = result.Title
				}
				continue
			}
			merged[key] = &candidate{result: result, score: score, firstSeen: item.index*100 + rank}
		}
	}
	if len(merged) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, ErrNoResult
	}
	all := make([]*candidate, 0, len(merged))
	for _, item := range merged {
		all = append(all, item)
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].score == all[j].score {
			return all[i].firstSeen < all[j].firstSeen
		}
		return all[i].score > all[j].score
	})
	out := make([]Result, 0, s.cfg.MaxResults)
	for _, item := range all {
		out = append(out, item.result)
		if len(out) >= s.cfg.MaxResults {
			break
		}
	}
	if len(out) == 0 {
		return nil, ErrNoResult
	}
	_ = engineCount
	return out, nil
}

// searchEngine 调用 Obscura；失败且允许回退时再走普通 HTTP。
func (s *Service) searchEngine(ctx context.Context, engine Engine, target string) ([]Result, error) {
	var (
		raw []byte
		err error
	)
	if s.cfg.ObscuraBin != "" {
		raw, err = s.fetchWithObscura(ctx, target)
	}
	if err == nil && len(raw) > 0 {
		if items := parseEngine(engine.Name, raw, s.cfg.MaxResults, engine.URL); len(items) > 0 {
			return items, nil
		}
		err = ErrNoResult
	}
	if !s.cfg.HTTPFallback {
		if err == nil {
			err = ErrNoResult
		}
		return nil, err
	}
	raw, httpErr := s.fetchHTTP(ctx, target)
	if httpErr != nil {
		if err != nil {
			return nil, fmt.Errorf("obscura: %v; http: %w", err, httpErr)
		}
		return nil, httpErr
	}
	items := parseEngine(engine.Name, raw, s.cfg.MaxResults, engine.URL)
	if len(items) == 0 {
		return nil, ErrNoResult
	}
	return items, nil
}

// normalizeQuery 去掉“联网查一下关于”等口语前缀，避免搜索引擎把整句话当成关键词。
func normalizeQuery(query string) string {
	query = strings.TrimSpace(query)
	prefixes := []string{
		"联网查一下关于", "联网查一下", "帮我联网查一下",
		"帮我查一下关于", "帮我查一下", "帮我搜索一下", "帮我搜一下",
		"搜索一下关于", "搜索一下", "查一下关于", "查一下",
		"联网搜索一下", "联网搜索", "搜索关于", "搜索",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(query, prefix) {
			query = strings.TrimSpace(strings.TrimPrefix(query, prefix))
			break
		}
	}
	query = strings.TrimSpace(strings.TrimLeft(query, "：:，,。！？!? "))
	return query
}

// buildURL 把搜索词填入模板；模板没有占位符时按 q 参数补齐。
func (s *Service) buildURL(query string) string {
	engine := strings.TrimSpace(s.cfg.EngineURL)
	if engine == "" {
		return ""
	}
	escaped := url.QueryEscape(query)
	if strings.Contains(engine, "{query}") {
		return strings.ReplaceAll(engine, "{query}", escaped)
	}
	separator := "?"
	if strings.Contains(engine, "?") {
		separator = "&"
	}
	return engine + separator + "q=" + escaped
}

// buildEngineURL 把搜索词填入指定引擎模板。
func buildEngineURL(engine, query string) string {
	engine = strings.TrimSpace(engine)
	if engine == "" {
		return ""
	}
	escaped := url.QueryEscape(query)
	if strings.Contains(engine, "{query}") {
		return strings.ReplaceAll(engine, "{query}", escaped)
	}
	separator := "?"
	if strings.Contains(engine, "?") {
		separator = "&"
	}
	return engine + separator + "q=" + escaped
}

// fetchWithObscura 调用 Obscura CLI 渲染搜索页。
func (s *Service) fetchWithObscura(ctx context.Context, target string) ([]byte, error) {
	if _, err := exec.LookPath(s.cfg.ObscuraBin); err != nil {
		return nil, fmt.Errorf("obscura not found: %w", err)
	}
	timeoutSec := int(s.cfg.Timeout.Seconds())
	if timeoutSec < 5 {
		timeoutSec = 5
	}
	args := []string{
		"fetch", target,
		"--dump", "html",
		"--timeout", strconv.Itoa(timeoutSec),
		"--quiet",
	}
	if s.cfg.Stealth {
		args = append(args, "--stealth")
	}
	cmd := exec.CommandContext(ctx, s.cfg.ObscuraBin, args...)
	// 超时后确保 worker 子进程被回收，避免容器里越积越多
	cmd.WaitDelay = 3 * time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		message := strings.TrimSpace(stderr.String())
		if len(message) > 240 {
			message = message[:240]
		}
		return nil, fmt.Errorf("obscura failed: %w (%s)", err, message)
	}
	if stdout.Len() == 0 {
		return nil, errors.New("obscura returned empty page")
	}
	return stdout.Bytes(), nil
}

// fetchHTTP 是 Obscura 不可用时的降级路径，只抓取搜索页 HTML。
func (s *Service) fetchHTTP(ctx context.Context, target string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", s.cfg.UserAgent)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("search engine returned HTTP %d", resp.StatusCode)
	}
	// 搜索页通常在 100KB 内，限制 4MB 防止异常响应占满内存
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}

// parseEngine 按来源选择解析器；未知来源走通用解析。
func parseEngine(name string, raw []byte, limit int, base string) []Result {
	switch {
	case strings.Contains(strings.ToLower(name), "bing"):
		return parseBing(raw, limit, base)
	case strings.Contains(strings.ToLower(name), "zhihu"):
		return parseZhihu(raw, limit, base)
	case strings.Contains(strings.ToLower(name), "sogou"):
		return parseSogou(raw, limit, base)
	case strings.Contains(strings.ToLower(name), "baidu"):
		return parseBaidu(raw, limit, base)
	default:
		return parse(raw, limit)
	}
}

// parseBing 解析 cn.bing.com 的 b_algo 结果块。
func parseBing(raw []byte, limit int, base string) []Result {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	var items []Result
	seen := make(map[string]struct{})
	doc.Find("li.b_algo").EachWithBreak(func(_ int, item *goquery.Selection) bool {
		if len(items) >= limit {
			return false
		}
		link := item.Find("h2 a").First()
		if link.Length() == 0 {
			return true
		}
		href, _ := link.Attr("href")
		href = resolveHrefBase(href, base)
		title := cleanText(link.Text())
		if href == "" || title == "" {
			return true
		}
		if _, ok := seen[href]; ok {
			return true
		}
		seen[href] = struct{}{}
		items = append(items, Result{
			Title:   title,
			URL:     href,
			Snippet: truncate(cleanText(item.Find(".b_caption p, .b_snippet").First().Text()), 320),
		})
		return true
	})
	return items
}

// parseSogou 解析搜狗网页搜索结果。
func parseSogou(raw []byte, limit int, base string) []Result {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	var items []Result
	seen := make(map[string]struct{})
	doc.Find("div.vrwrap, div.rb").EachWithBreak(func(_ int, item *goquery.Selection) bool {
		if len(items) >= limit {
			return false
		}
		link := item.Find("h3 a, .vr-title a").First()
		if link.Length() == 0 {
			return true
		}
		href, _ := link.Attr("href")
		href = resolveHrefBase(href, base)
		title := cleanText(link.Text())
		if href == "" || title == "" {
			return true
		}
		if _, ok := seen[href]; ok {
			return true
		}
		seen[href] = struct{}{}
		items = append(items, Result{
			Title:   title,
			URL:     href,
			Snippet: truncate(cleanText(item.Find(".str-text-info, .text-layout, .str_info").First().Text()), 320),
		})
		return true
	})
	return items
}

// parseBaidu 解析百度结果容器；百度的跳转链接保留原样，避免解析失败丢结果。
func parseBaidu(raw []byte, limit int, base string) []Result {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	var items []Result
	seen := make(map[string]struct{})
	doc.Find("div.result, div.c-container").EachWithBreak(func(_ int, item *goquery.Selection) bool {
		if len(items) >= limit {
			return false
		}
		link := item.Find("h3 a").First()
		if link.Length() == 0 {
			return true
		}
		href, _ := link.Attr("href")
		href = resolveHrefBase(href, base)
		title := cleanText(link.Text())
		if href == "" || title == "" {
			return true
		}
		if _, ok := seen[href]; ok {
			return true
		}
		seen[href] = struct{}{}
		items = append(items, Result{
			Title:   title,
			URL:     href,
			Snippet: truncate(cleanText(item.Find(".c-abstract, .content-right_8Zs40").First().Text()), 320),
		})
		return true
	})
	return items
}

// parseZhihu 解析知乎搜索卡片；未登录时结果可能为空，交给其他引擎兜底。
func parseZhihu(raw []byte, limit int, base string) []Result {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	var items []Result
	seen := make(map[string]struct{})
	doc.Find("div.SearchResult-Card, div.List-item").EachWithBreak(func(_ int, item *goquery.Selection) bool {
		if len(items) >= limit {
			return false
		}
		link := item.Find("h2 a, .ContentItem-title a").First()
		if link.Length() == 0 {
			return true
		}
		href, _ := link.Attr("href")
		href = resolveHrefBase(href, base)
		title := cleanText(link.Text())
		if href == "" || title == "" {
			return true
		}
		if _, ok := seen[href]; ok {
			return true
		}
		seen[href] = struct{}{}
		items = append(items, Result{
			Title:   title,
			URL:     href,
			Snippet: truncate(cleanText(item.Find(".RichText, .SearchItem-meta").First().Text()), 320),
		})
		return true
	})
	return items
}

// parse 从 DuckDuckGo / Bing 搜索结果页提取条目。
func parse(raw []byte, limit int) []Result {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	var items []Result
	seen := make(map[string]struct{})
	add := func(title, href, snippet string) {
		title = cleanText(title)
		href = resolveHref(href)
		snippet = truncate(cleanText(snippet), 320)
		if title == "" || href == "" {
			return
		}
		if _, ok := seen[href]; ok {
			return
		}
		seen[href] = struct{}{}
		items = append(items, Result{Title: title, URL: href, Snippet: snippet})
	}
	// DuckDuckGo HTML 版
	doc.Find("div.result").EachWithBreak(func(_ int, item *goquery.Selection) bool {
		if len(items) >= limit {
			return false
		}
		if item.HasClass("result--ad") {
			return true
		}
		link := item.Find("a.result__a").First()
		if link.Length() == 0 {
			return true
		}
		href, _ := link.Attr("href")
		snippet := item.Find(".result__snippet").First().Text()
		add(link.Text(), href, snippet)
		return true
	})
	// Bing
	if len(items) == 0 {
		doc.Find("li.b_algo").EachWithBreak(func(_ int, item *goquery.Selection) bool {
			if len(items) >= limit {
				return false
			}
			link := item.Find("h2 a").First()
			if link.Length() == 0 {
				return true
			}
			href, _ := link.Attr("href")
			snippet := item.Find(".b_caption p, .b_snippet").First().Text()
			add(link.Text(), href, snippet)
			return true
		})
	}
	// 兜底：部分搜索页使用通用标题链接结构，只取明确的结果容器
	if len(items) == 0 {
		doc.Find("main a[href], #links a[href]").EachWithBreak(func(_ int, link *goquery.Selection) bool {
			if len(items) >= limit {
				return false
			}
			href, _ := link.Attr("href")
			title := cleanText(link.Text())
			if utf8.RuneCountInString(title) < 6 {
				return true
			}
			add(title, href, "")
			return true
		})
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

// resolveHrefBase 处理相对地址，并复用 DuckDuckGo 跳转解析。
func resolveHrefBase(href, base string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "//") {
		href = "https:" + href
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return resolveHref(href)
	}
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return resolveHref(baseURL.ResolveReference(ref).String())
}

// canonicalURL 用于 RRF 去重：忽略 fragment、tracking 参数，统一 host 大小写与 query 顺序。
func canonicalURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	q := u.Query()
	for key := range q {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "spm" || lower == "from" || lower == "ref" || lower == "source" {
			q.Del(key)
		}
	}
	u.RawQuery = q.Encode()
	if u.Path != "/" {
		u.Path = strings.TrimRight(u.Path, "/")
	}
	return u.String()
}

// resolveHref 解开 DuckDuckGo 的跳转链接，并过滤非 HTTP(S) 地址。
func resolveHref(href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "//") {
		href = "https:" + href
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if strings.Contains(u.Host, "duckduckgo.com") {
		if target := u.Query().Get("uddg"); target != "" {
			href = target
			u, err = url.Parse(href)
			if err != nil {
				return ""
			}
		}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	if u.Host == "" || strings.Contains(u.Host, "duckduckgo.com") {
		return ""
	}
	return u.String()
}

// cleanText 压缩空白并去掉零宽字符。
func cleanText(value string) string {
	value = strings.ReplaceAll(value, "\u200b", "")
	// 搜索引擎偶尔返回被截断的 UTF-8 字节，或把替换符写成字面量 \ufffd；
	// 两种形式都清掉，避免界面出现 � 或 “\ufffd”。
	value = strings.ReplaceAll(value, "\ufffd", "")
	value = strings.ReplaceAll(value, `\ufffd`, "")
	value = strings.ReplaceAll(value, `\uFFFD`, "")
	value = strings.ReplaceAll(value, "\xa0", " ")
	// 先把被截断的 UTF-8 字节换成空串，再做空白归一化，避免 JSON 编码后出现 �
	value = strings.ToValidUTF8(value, "")
	return strings.TrimSpace(strings.Join(strings.Fields(value), " "))
}

// truncate 按字符截断，避免摘要过长挤占上下文。
func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "…"
}
