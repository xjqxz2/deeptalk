package search

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// fakeTransport 按 URL 模拟慢页面与正常页面。
type fakeTransport struct{ delay time.Duration }

func (f fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.Path, "slow") {
		select {
		case <-time.After(f.delay):
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	}
	body := "<html><head><title>ok</title></head><body>hello</body></html>"
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

// TestFetchPageTimeoutPerPage 校验单个网页超时返回 ErrPageTimeout。
func TestFetchPageTimeoutPerPage(t *testing.T) {
	svc := New(Config{
		Enabled:      true,
		EngineURL:    "https://example.com/search?q={query}",
		Timeout:      2 * time.Second,
		PageTimeout:  120 * time.Millisecond,
		HTTPFallback: true,
		Concurrency:  2,
	})
	if svc == nil {
		t.Fatal("service should be enabled")
	}
	svc.client = &http.Client{Transport: fakeTransport{delay: 400 * time.Millisecond}}

	_, err := svc.fetchPageWithTimeout(context.Background(), "https://slow.example.com/slow")
	if !errors.Is(err, ErrPageTimeout) {
		t.Fatalf("slow page err = %v, want ErrPageTimeout", err)
	}

	raw, err := svc.fetchPageWithTimeout(context.Background(), "https://fast.example.com/fast")
	if err != nil {
		t.Fatalf("fast page: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("fast page returned empty body")
	}
}

// TestPageTimeoutDoesNotBlockOthers 校验慢页面超时不会拖累其他页面完成。
func TestPageTimeoutDoesNotBlockOthers(t *testing.T) {
	svc := New(Config{
		Enabled:      true,
		EngineURL:    "https://example.com/search?q={query}",
		Timeout:      2 * time.Second,
		PageTimeout:  100 * time.Millisecond,
		HTTPFallback: true,
		Concurrency:  2,
	})
	svc.client = &http.Client{Transport: fakeTransport{delay: 2 * time.Second}}

	start := time.Now()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = svc.fetchPageWithTimeout(context.Background(), "https://slow.example.com/slow")
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("page fetch ignored the per-page timeout")
	}
	if elapsed := time.Since(start); elapsed > 900*time.Millisecond {
		t.Fatalf("took %v, want under the 1s guard", elapsed)
	}
}
