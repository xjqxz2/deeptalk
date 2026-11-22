package search

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestRunEnginesReturnsEarly 校验慢引擎不会拖住整轮（早退生效）。
func TestRunEnginesReturnsEarly(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		fmt.Fprint(w, `<html><body><ol><li class="b_algo"><h2><a href="https://slow.example.com/a">关键词 测试 慢结果</a></h2><p>关键词 测试 慢结果摘要，内容足够长用于通过相关性判断。</p></li></ol></body></html>`)
	}))
	defer slow.Close()
	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><ol><li class="b_algo"><h2><a href="https://fast.example.com/a">关键词 测试 快结果</a></h2><p>关键词 测试 快结果摘要，内容足够长用于通过相关性判断。</p></li></ol></body></html>`)
	}))
	defer fast.Close()

	svc := New(Config{
		Enabled:          true,
		Engines:          []Engine{{Name: "bing", URL: fast.URL + "?q={query}", Weight: 1}, {Name: "bing", URL: slow.URL + "?q={query}", Weight: 1}},
		Timeout:          10 * time.Second,
		PageTimeout:      5 * time.Second,
		EarlyReturnAfter: 300 * time.Millisecond,
		HTTPFallback:     true,
		Concurrency:      4,
		MaxResults:       5,
	})
	start := time.Now()
	items, err := svc.Search(context.Background(), "关键词 测试")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected results")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("took %v, early return should cut the slow engine", elapsed)
	}
}
