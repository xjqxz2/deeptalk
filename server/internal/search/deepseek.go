package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// DeepSeekProvider 返回当前生效的检索提供方：self（自建）或 deepseek，以及 DeepSeek 密钥。
// 由 bootstrap 注入，每次调用都读设置，后台改完立即生效。
type DeepSeekProvider func(ctx context.Context) (mode string, key string)

// deepSeekBaseURL 便于测试覆盖；生产固定为官方地址。
var deepSeekBaseURL = "https://api.deepseek.com"

const (
	ProviderSelf     = "self"     // 自建检索（Obscura / 多引擎）
	ProviderDeepSeek = "deepseek" // DeepSeek 官方 Search / Fetch API
	ProviderBuiltin  = "builtin"  // 模型内置联网（Responses / Anthropic 服务端 web_search）
)

// SetDeepSeekProvider 注入提供方解析函数。
func (s *Service) SetDeepSeekProvider(provider DeepSeekProvider) {
	s.deepseek = provider
}

// deepseekKey 返回 DeepSeek 密钥；未选择 DeepSeek 时返回空串。
func (s *Service) deepseekKey(ctx context.Context) string {
	if s == nil || s.deepseek == nil {
		return ""
	}
	mode, key := s.deepseek(ctx)
	if strings.ToLower(strings.TrimSpace(mode)) != ProviderDeepSeek {
		return ""
	}
	return strings.TrimSpace(key)
}

type deepSeekSearchResponse struct {
	Results []struct {
		Title     string  `json:"title"`
		URL       string  `json:"url"`
		Snippet   string  `json:"snippet"`
		Summary   string  `json:"summary"`
		Content   string  `json:"content"`
		SiteName  *string `json:"site_name"`
		SiteIcon  *string `json:"site_icon"`
		Date      *string `json:"date"`
		Authority int     `json:"authority"`
	} `json:"results"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// searchDeepSeek 调用 POST /v1/tools/search，返回结构化结果。
func (s *Service) searchDeepSeek(ctx context.Context, key, query string) ([]Result, error) {
	return s.searchDeepSeekAt(ctx, deepSeekBaseURL+"/v1/tools/search", key, query)
}

// searchDeepSeekAt 允许测试指向本地端点。
func (s *Service) searchDeepSeekAt(ctx context.Context, endpoint, key, query string) ([]Result, error) {
	body, err := json.Marshal(map[string]any{
		"query":        query,
		"count":        10,
		"need_content": true,
	})
	if err != nil {
		return nil, err
	}
	searchCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(searchCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("deepseek search: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("deepseek search: HTTP %d %s", resp.StatusCode, deepSeekError(raw))
	}
	var parsed deepSeekSearchResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("deepseek search: 响应解析失败")
	}
	out := make([]Result, 0, len(parsed.Results))
	for index, item := range parsed.Results {
		summary := item.Summary
		if summary == "" {
			summary = item.Content
		}
		if summary == "" {
			summary = item.Snippet
		}
		result := Result{
			Title:   item.Title,
			URL:     item.URL,
			Snippet: truncate(summary, 300),
		}
		if index < 10 {
			out = append(out, result)
		}
	}
	if len(out) == 0 {
		return nil, ErrNoResult
	}
	return out, nil
}

type deepSeekFetchResponse struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Error   struct {
		Message string `json:"message"`
	} `json:"error"`
}

// fetchDeepSeek 调用 POST /v1/tools/fetch 抓取单个网页。
func (s *Service) fetchDeepSeek(ctx context.Context, key, target string) (Page, error) {
	body, _ := json.Marshal(map[string]any{"url": target})
	pageCtx, cancel := context.WithTimeout(ctx, s.cfg.PageTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(pageCtx, http.MethodPost, "https://api.deepseek.com/v1/tools/fetch", bytes.NewReader(body))
	if err != nil {
		return Page{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := s.client.Do(req)
	if err != nil {
		if errors.Is(pageCtx.Err(), context.DeadlineExceeded) {
			return Page{}, ErrPageTimeout
		}
		return Page{}, fmt.Errorf("deepseek fetch: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return Page{}, fmt.Errorf("deepseek fetch: HTTP %d %s", resp.StatusCode, deepSeekError(raw))
	}
	var parsed deepSeekFetchResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Page{}, fmt.Errorf("deepseek fetch: 响应解析失败")
	}
	if strings.TrimSpace(parsed.Content) == "" {
		return Page{}, ErrNoResult
	}
	return Page{URL: target, Title: parsed.Title, Content: parsed.Content}, nil
}

// fetchPagesDeepSeek 并发抓取多个网页（每个网页独立 15s 预算）。
func (s *Service) fetchPagesDeepSeek(ctx context.Context, key string, urls []string, onPage func(Page)) ([]Page, error) {
	type outcome struct {
		page Page
		err  error
	}
	results := make([]outcome, len(urls))
	var wg sync.WaitGroup
	for index, target := range urls {
		wg.Add(1)
		go func(index int, target string) {
			defer wg.Done()
			page, err := s.fetchDeepSeek(ctx, key, target)
			results[index] = outcome{page: page, err: err}
		}(index, target)
	}
	wg.Wait()
	pages := make([]Page, 0, len(urls))
	var firstErr error
	for _, item := range results {
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

func deepSeekError(raw []byte) string {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &parsed) == nil && parsed.Error.Message != "" {
		return parsed.Error.Message
	}
	return truncate(string(raw), 160)
}
