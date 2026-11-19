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
	"strconv"
	"strings"
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

// Config 是检索服务的可配置项。
type Config struct {
	Enabled      bool          // 是否启用联网搜索
	ObscuraBin   string        // Obscura 可执行文件路径或 PATH 中的命令名
	EngineURL    string        // 搜索页模板，支持 {query} 占位符
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
	if !cfg.Enabled || strings.TrimSpace(cfg.EngineURL) == "" {
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

// Search 执行一次检索。
// 优先调用 Obscura；Obscura 失败且开启回退时再走普通 HTTP。
func (s *Service) Search(ctx context.Context, query string) ([]Result, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	query = normalizeQuery(query)
	if query == "" {
		return nil, errors.New("search query is empty")
	}
	// 搜索词过长会被搜索引擎截断，这里主动限制，避免 URL 超长
	runes := []rune(query)
	if len(runes) > 200 {
		query = string(runes[:200])
	}
	target := s.buildURL(query)
	if target == "" {
		return nil, errors.New("invalid search engine url")
	}
	searchCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()

	var (
		raw []byte
		err error
	)
	if s.cfg.ObscuraBin != "" {
		raw, err = s.fetchWithObscura(searchCtx, target)
	}
	if err == nil && len(raw) > 0 {
		if items := parse(raw, s.cfg.MaxResults); len(items) > 0 {
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
	raw, httpErr := s.fetchHTTP(searchCtx, target)
	if httpErr != nil {
		if err != nil {
			return nil, fmt.Errorf("obscura: %v; http: %w", err, httpErr)
		}
		return nil, httpErr
	}
	items := parse(raw, s.cfg.MaxResults)
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
