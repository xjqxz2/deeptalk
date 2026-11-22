// Package logging 提供按天滚动、自动压缩并限制保留天数的文件日志。
package logging

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// DailyWriter 把写入内容按天落到 dir/<prefix>-YYYY-MM-DD.log。
// 跨天时把前一天的文件 gzip 压缩，并清理超过 KeepDays 的旧文件。
type DailyWriter struct {
	mu       sync.Mutex
	dir      string
	prefix   string
	keepDays int
	now      func() time.Time

	day  string
	file *os.File
}

// NewDailyWriter 创建按天滚动的写入器；目录不可写时返回错误，调用方应回退到 stdout。
func NewDailyWriter(dir, prefix string, keepDays int) (*DailyWriter, error) {
	if keepDays <= 0 {
		keepDays = 7
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	writer := &DailyWriter{dir: dir, prefix: prefix, keepDays: keepDays, now: time.Now}
	if err := writer.rotate(); err != nil {
		return nil, err
	}
	return writer, nil
}

// Write 实现 io.Writer，按当前日期追加到当天文件。
func (w *DailyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if day := w.now().Format("2006-01-02"); day != w.day {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	if w.file == nil {
		return 0, fmt.Errorf("log file not open")
	}
	return w.file.Write(p)
}

// Close 关闭当天文件。
func (w *DailyWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

// rotate 切换到当天文件，并压缩、清理历史文件。
func (w *DailyWriter) rotate() error {
	day := w.now().Format("2006-01-02")
	if w.file != nil {
		prev := w.file.Name()
		_ = w.file.Close()
		w.file = nil
		if !strings.HasSuffix(prev, day+".log") {
			_ = gzipFile(prev)
		}
	}
	path := filepath.Join(w.dir, fmt.Sprintf("%s-%s.log", w.prefix, day))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	w.file = file
	w.day = day
	w.prune()
	return nil
}

// prune 删除超过保留天数的日志与压缩包。
func (w *DailyWriter) prune() {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return
	}
	type item struct {
		path string
		day  string
	}
	items := make([]item, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, w.prefix+"-") {
			continue
		}
		day := strings.TrimSuffix(strings.TrimSuffix(strings.TrimPrefix(name, w.prefix+"-"), ".gz"), ".log")
		if _, err := time.Parse("2006-01-02", day); err != nil {
			continue
		}
		items = append(items, item{path: filepath.Join(w.dir, name), day: day})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].day < items[j].day })
	cutoff := w.now().AddDate(0, 0, -w.keepDays).Format("2006-01-02")
	for _, entry := range items {
		if entry.day >= cutoff {
			continue
		}
		_ = os.Remove(entry.path)
	}
}

// gzipFile 把滚动出去的日志压缩成 .gz，失败时保留原文件以便排查。
func gzipFile(path string) error {
	source, err := os.Open(path)
	if err != nil {
		return err
	}
	defer source.Close()

	target, err := os.Create(path + ".gz")
	if err != nil {
		return err
	}
	writer := gzip.NewWriter(target)
	if _, err := io.Copy(writer, source); err != nil {
		_ = writer.Close()
		_ = target.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		_ = target.Close()
		return err
	}
	if err := target.Close(); err != nil {
		return err
	}
	return os.Remove(path)
}
