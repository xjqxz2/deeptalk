package logging

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDailyWriterRotatesAndCompresses 校验跨天滚动、压缩与 7 天清理。
func TestDailyWriterRotatesAndCompresses(t *testing.T) {
	dir := t.TempDir()
	day := time.Date(2026, 11, 20, 23, 0, 0, 0, time.Local)
	writer, err := NewDailyWriter(dir, "db", 7)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	writer.now = func() time.Time { return day }
	if _, err := writer.Write([]byte("first\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	// 跨天：前一天的日志应被压缩
	writer.now = func() time.Time { return day.Add(24 * time.Hour) }
	if _, err := writer.Write([]byte("second\n")); err != nil {
		t.Fatalf("write day2: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "db-2026-11-20.log.gz")); err != nil {
		t.Fatalf("expected gz for day1: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "db-2026-11-21.log")); err != nil {
		t.Fatalf("expected current day file: %v", err)
	}

	// 8 天后：更早的文件应被清理
	writer.now = func() time.Time { return day.AddDate(0, 0, 8) }
	if _, err := writer.Write([]byte("later\n")); err != nil {
		t.Fatalf("write later: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "db-2026-11-20.log.gz")); !os.IsNotExist(err) {
		t.Fatalf("old log should be pruned, err = %v", err)
	}
}
