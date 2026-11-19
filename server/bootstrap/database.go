package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"deeptalk/internal/attachment"
	"deeptalk/internal/audit"
	"deeptalk/internal/chat"
	"deeptalk/internal/provider"
	"deeptalk/internal/seed"
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
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
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
	}
	for _, migration := range migrations {
		if err := migration.fn(db); err != nil {
			return errors.New("migrate " + migration.name + ": " + err.Error())
		}
	}
	a.DB = db
	a.Log.Info("database ready", "dsn", cfg.Database.DSN)
	if cfg.Database.Seed {
		if err := seed.Run(ctx, db, seed.Config{
			Secret:         cfg.Security.Secret,
			DeepSeekAPIKey: cfg.Upstream.DeepSeekAPIKey,
		}); err != nil {
			return err
		}
		a.Log.Info("seed data ready")
	}
	return nil
}

// stopDatabase 关闭数据库连接。
func (a *App) stopDatabase(ctx context.Context) error {
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
