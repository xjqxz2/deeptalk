package bootstrap

import (
	"context"
	"errors"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"deeptalk/internal/attachment"
	"deeptalk/internal/audit"
	"deeptalk/internal/billing"
	"deeptalk/internal/chat"
	"deeptalk/internal/logging"
	"deeptalk/internal/memo"
	"deeptalk/internal/provider"
	"deeptalk/internal/setting"
	"deeptalk/internal/user"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// startDatabase 打开 SQLite、执行迁移并写入演示数据。
// SQLite 使用 WAL 与 busy_timeout，兼顾单机读写与并发访问。
func (a *App) startDatabase(ctx context.Context) error {
	cfg := a.Config
	if err := ensureDataDir(cfg.Database.DSN); err != nil {
		return err
	}
	dsn := withSQLitePragmas(cfg.Database.DSN)
	// SQL 日志写到 data/logs/db-YYYY-MM-DD.log：按天滚动、次日 gzip、只保留 7 天。
	// 控制台保持干净，排查时再去翻文件。
	sqlOut := io.Writer(os.Stdout)
	logDir := filepath.Join(filepath.Dir(cfg.Database.DSN), "logs")
	if rolling, err := logging.NewDailyWriter(logDir, "db", 7); err == nil {
		sqlOut = rolling
		a.dbLogCloser = rolling
	} else {
		slog.Warn("sql log file disabled", "dir", logDir, "error", err)
	}
	dbLogger := logger.New(log.New(sqlOut, "", log.LstdFlags), logger.Config{
		SlowThreshold:             500 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: false, // 记到文件里，便于回溯，但不再刷控制台
		ParameterizedQueries:      false,
		Colorful:                  false,
	})
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: dbLogger,
	})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(8)
	sqlDB.SetMaxIdleConns(4)
	if err := sqlDB.PingContext(ctx); err != nil {
		return err
	}
	// 迁移顺序与业务表依赖一致；生产环境以 migrations/0001_init.sql 为准
	migrations := []struct {
		name string
		fn   func(*gorm.DB) error
	}{
		{"user", user.Migrate},
		{"setting", setting.Migrate},
		{"audit", audit.Migrate},
		{"attachment", attachment.Migrate},
		{"provider", provider.Migrate},
		{"chat", chat.Migrate},
		{"billing", billing.Migrate},
		{"memo", memo.Migrate},
	}
	for _, migration := range migrations {
		if err := migration.fn(db); err != nil {
			return errors.New("migrate " + migration.name + ": " + err.Error())
		}
	}
	// 进程重启后，上次「正在生成」的回答已经没有对应的后台任务：
	// 收尾成 stopped，避免前端一直显示「生成中」转圈。
	if err := db.Model(&chat.Message{}).
		Where("status = ?", chat.StatusStreaming).
		Update("status", chat.StatusStopped).Error; err != nil {
		return errors.New("cleanup streaming messages: " + err.Error())
	}
	a.DB = db
	a.Log.Info("database ready", "dsn", cfg.Database.DSN)
	return nil
}

// stopDatabase 关闭数据库连接。
func (a *App) stopDatabase(ctx context.Context) error {
	if a.dbLogCloser != nil {
		_ = a.dbLogCloser.Close()
	}
	if a.DB == nil {
		return nil
	}
	sqlDB, err := a.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// ensureDataDir 创建 SQLite 文件所在目录。
func ensureDataDir(dsn string) error {
	path := dsn
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

// withSQLitePragmas 补齐 WAL、超时与外键约束；调用方已带参数时不重复追加。
func withSQLitePragmas(dsn string) string {
	if strings.Contains(dsn, "_pragma=") {
		return dsn
	}
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	return dsn + separator + "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
}
