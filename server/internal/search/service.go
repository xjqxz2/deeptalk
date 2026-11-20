// Package search 提供联网检索能力。
// 检索页面优先由 Obscura MCP 的浏览器工具抓取；MCP 不可用时回退 Obscura CLI，
// 最后才走普通 HTTP。返回结果只包含标题、链接与摘要，不把原始 HTML 暴露给上层。
package search

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
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
// Viewed 表示模型不仅看到摘要，还真正打开了网页正文；前端据此区分“查看的网页”和搜索候选。
type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Viewed  bool   `json:"viewed,omitempty"`
}

// Page 是模型指定 URL 后抓取到的正文。
type Page struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// ToolCaller 是检索服务消费的 MCP 能力，定义在消费方以便测试替换。
type ToolCaller interface {
	HasTool(name string) bool
	CallTool(ctx context.Context, name string, args any) (string, error)
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
	Concurrency  int           // 多引擎并发数
	Timeout      time.Duration // 单次检索超时
	HTTPFallback bool          // Obscura 失败时是否允许普通 HTTP 抓取
	Stealth      bool          // 是否启用 Obscura 反检测模式
	UserAgent    string        // HTTP 回退时使用的浏览器 UA
	MCP          ToolCaller    // Obscura MCP client；为 nil 时走本地检索链路
}

// Service 是联网检索服务。
type Service struct {
	cfg    Config
	client *http.Client
	// Obscura MCP 只有一个活动页面；navigate + evaluate 必须原子执行。
	mcpMu sync.Mutex
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
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 5
	}
	if cfg.Concurrency > 10 {
		cfg.Concurrency = 10
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

// FetchPages 抓取模型明确指定的公开 URL，返回正文供主模型阅读。
func (s *Service) FetchPages(ctx context.Context, rawURLs []string) ([]Page, error) {
	return s.FetchPagesWithProgress(ctx, rawURLs, nil)
}

// FetchPagesWithProgress 在每抓完一个页面时回调，供 SSE 展示“正在阅读哪个网页”。
// 回调在汇总协程中串行执行，调用方无需再加锁。
func (s *Service) FetchPagesWithProgress(ctx context.Context, rawURLs []string, onPage func(Page)) ([]Page, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	seen := make(map[string]struct{})
	urls := make([]string, 0, len(rawURLs))
	for _, raw := range rawURLs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		normalized, err := s.validatePublicURL(ctx, raw)
		if err != nil {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		urls = append(urls, normalized)
		if len(urls) >= 5 {
			break
		}
	}
	if len(urls) == 0 {
		return nil, errors.New("没有可抓取的公开 URL")
	}
	type pageResult struct {
		page Page
		err  error
	}
	results := make(chan pageResult, len(urls))
	sem := make(chan struct{}, s.cfg.Concurrency)
	var wg sync.WaitGroup
	for _, target := range urls {
		wg.Add(1)
		go func(target string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results <- pageResult{err: ctx.Err()}
				return
			}
			raw, err := s.fetchPage(ctx, target)
			if err != nil {
				results <- pageResult{err: err}
				return
			}
			title, content := extractPage(raw)
			if strings.TrimSpace(content) == "" {
				results <- pageResult{err: ErrNoResult}
				return
			}
			results <- pageResult{page: Page{URL: target, Title: title, Content: content}}
		}(target)
	}
	go func() { wg.Wait(); close(results) }()
	pages := make([]Page, 0, len(urls))
	var firstErr error
	for item := range results {
		if item.err != nil {
			if firstErr == nil {
				firstErr = item.err
			}
			continue
		}
		pages = append(pages, item.page)
		if onPage != nil {
			onPage(item.page)
		}
	}
	if len(pages) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, ErrNoResult
	}
	return pages, nil
}

func extractPage(raw []byte) (string, string) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(raw))
	if err != nil {
		return "", ""
	}
	title := cleanText(doc.Find("title").Text())
	body := doc.Find("body").Text()
	if strings.TrimSpace(body) == "" {
		body = doc.Text()
	}
	return title, truncate(cleanText(body), 12000)
}

func (s *Service) validatePublicURL(ctx context.Context, raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return "", errors.New("URL 不合法")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("只允许 HTTP(S) URL")
	}
	if u.User != nil {
		return "", errors.New("URL 不能包含账号信息")
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", u.Hostname())
	if err != nil {
		return "", errors.New("URL 域名无法解析")
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return "", errors.New("URL 指向私网地址")
		}
	}
	return u.String(), nil
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
	sem := make(chan struct{}, s.cfg.Concurrency) // 默认 5 路并发，可通过配置调整
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
		relevant  bool
	}
	terms := queryTerms(query)
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
			relevant := isRelevantResult(result, terms)
			score := item.engine.Weight / (60 + float64(rank+1))
			if current, ok := merged[key]; ok {
				current.score += score
				current.relevant = current.relevant || relevant
				if current.result.Snippet == "" && result.Snippet != "" {
					current.result.Snippet = result.Snippet
				}
				if current.result.Title == "" && result.Title != "" {
					current.result.Title = result.Title
				}
				continue
			}
			merged[key] = &candidate{result: result, score: score, firstSeen: item.index*100 + rank, relevant: relevant}
		}
	}
	if len(merged) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, ErrNoResult
	}
	all := make([]*candidate, 0, len(merged))
	hasRelevant := false
	for _, item := range merged {
		if item.relevant {
			hasRelevant = true
			break
		}
	}
	if !hasRelevant {
		// 全部来源都与问题词无关时，不再把它们塞进来源面板；
		// 这比展示“我 / 我的世界”这类乱码结果更可信。
		return nil, ErrNoResult
	}
	for _, item := range merged {
		if item.relevant {
			all = append(all, item)
		}
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

// searchEngine 优先用 Obscura MCP 浏览器打开搜索页，再复用现有解析与排序；
// MCP 无法启动时退回 Obscura CLI，最后才走普通 HTTP。
func (s *Service) searchEngine(ctx context.Context, engine Engine, target string) ([]Result, error) {
	raw, err := s.fetchPage(ctx, target)
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
			return nil, fmt.Errorf("mcp/obscura: %v; http: %w", err, httpErr)
		}
		return nil, httpErr
	}
	items := parseEngine(engine.Name, raw, s.cfg.MaxResults, engine.URL)
	if len(items) == 0 {
		return nil, ErrNoResult
	}
	return items, nil
}

// fetchPage 按 MCP -> Obscura CLI -> HTTP 的顺序取页面，返回第一个成功结果。
func (s *Service) fetchPage(ctx context.Context, target string) ([]byte, error) {
	var errs []string
	if s.cfg.MCP != nil && s.cfg.MCP.HasTool("browser_navigate") && s.cfg.MCP.HasTool("browser_evaluate") {
		// MCP 只有一个活动页面。拿不到锁时不要排队，直接走 CLI/HTTP，
		// 既保留并发搜索，也避免一个慢页面拖住所有引擎。
		if s.mcpMu.TryLock() {
			raw, err := s.fetchWithMCPUnlocked(ctx, target)
			s.mcpMu.Unlock()
			if err == nil && len(raw) > 0 {
				return raw, nil
			}
			if err != nil {
				errs = append(errs, "mcp: "+err.Error())
			}
		}
	}
	if s.cfg.ObscuraBin != "" {
		raw, err := s.fetchWithObscura(ctx, target)
		if err == nil && len(raw) > 0 {
			return raw, nil
		}
		if err != nil {
			errs = append(errs, "obscura: "+err.Error())
		}
	}
	if s.cfg.HTTPFallback {
		raw, err := s.fetchHTTP(ctx, target)
		if err == nil && len(raw) > 0 {
			return raw, nil
		}
		if err != nil {
			errs = append(errs, "http: "+err.Error())
		}
	}
	if len(errs) == 0 {
		errs = append(errs, "no fetch backend available")
	}
	return nil, errors.New(strings.Join(errs, "; "))
}

// fetchWithMCP 通过 Obscura MCP 的浏览器工具取得渲染后的 HTML。
func (s *Service) fetchWithMCP(ctx context.Context, target string) ([]byte, error) {
	s.mcpMu.Lock()
	defer s.mcpMu.Unlock()
	return s.fetchWithMCPUnlocked(ctx, target)
}

// fetchWithMCPUnlocked 要求调用方已持有 mcpMu，保证 navigate + evaluate 原子执行。
func (s *Service) fetchWithMCPUnlocked(ctx context.Context, target string) ([]byte, error) {
	if s.cfg.MCP == nil {
		return nil, errors.New("mcp is not configured")
	}
	if _, err := s.cfg.MCP.CallTool(ctx, "browser_navigate", map[string]any{
		"url":       target,
		"waitUntil": "domcontentloaded",
	}); err != nil {
		return nil, err
	}
	raw, err := s.cfg.MCP.CallTool(ctx, "browser_evaluate", map[string]any{
		"expression": "document.documentElement.outerHTML",
		"timeoutMs":  8000,
	})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) == "" {
		return nil, ErrNoResult
	}
	return []byte(raw), nil
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

var queryTermPattern = regexp.MustCompile(`[a-zA-Z0-9]{2,}|[\p{Han}]{2,}`)

func isHanRune(r rune) bool {
	return r >= 0x4e00 && r <= 0x9fff
}

// queryTerms 提取用于相关性过滤的词；不追求分词精度，只挡住完全跑题的引擎结果。
func queryTerms(query string) []string {
	raw := strings.ToLower(query)
	stop := map[string]struct{}{
		"什么": {}, "怎么": {}, "多少": {}, "请问": {}, "一下": {}, "可以": {}, "需要": {},
		"为什么": {}, "如何": {}, "是否": {}, "以及": {}, "然后": {},
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, 8)
	for _, term := range queryTermPattern.FindAllString(raw, -1) {
		if _, ok := stop[term]; ok {
			continue
		}
		runes := []rune(term)
		// 中文长句必须切成 bigram 再做相关性判断；
		// 否则“今天热点新闻”会要求标题完整包含这六个字，几乎必然被误杀。
		if len(runes) > 2 && isHanRune(runes[0]) {
			for i := 0; i+1 < len(runes); i++ {
				bgram := string(runes[i : i+2])
				if _, ok := stop[bgram]; ok {
					continue
				}
				if _, ok := seen[bgram]; ok {
					continue
				}
				seen[bgram] = struct{}{}
				out = append(out, bgram)
			}
			continue
		}
		if len(runes) < 2 {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		out = append(out, term)
	}
	return out
}

// isRelevantResult 要求标题/摘要/URL 至少包含一个问题词。
func isRelevantResult(result Result, terms []string) bool {
	if len(terms) == 0 {
		return true
	}
	haystack := strings.ToLower(result.Title + " " + result.Snippet + " " + result.URL)
	for _, term := range terms {
		if strings.Contains(haystack, term) {
			return true
		}
	}
	return false
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
