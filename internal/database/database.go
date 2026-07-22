package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Open creates a SQLite-backed GORM connection and applies startup settings.
func Open(databasePath string) (*gorm.DB, error) {
	if err := ensureDatabaseDir(databasePath); err != nil {
		return nil, err
	}

	dsn := databasePath
	if databasePath != ":memory:" {
		dsn = fmt.Sprintf("%s?_busy_timeout=5000&_foreign_keys=on", databasePath)
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql database handle: %w", err)
	}

	if databasePath != ":memory:" {
		if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
			return nil, fmt.Errorf("enable sqlite WAL mode: %w", err)
		}
	}
	if _, err := sqlDB.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		return nil, fmt.Errorf("set sqlite busy timeout: %w", err)
	}
	if _, err := sqlDB.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	return db, nil
}

// AutoMigrate applies the initial schema.
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.AdminUser{},
		&models.AppSetting{},
		&models.ShortLink{},
		&models.AuditLog{},
		&models.VisitorSession{},
		&models.PageView{},
		&models.RedirectEvent{},
	); err != nil {
		return fmt.Errorf("auto migrate database: %w", err)
	}

	return nil
}

// Ping verifies that the database is reachable.
func Ping(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql database handle: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	return nil
}

// Close closes the underlying sql.DB.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql database handle: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}

	return nil
}

func ensureDatabaseDir(databasePath string) error {
	if databasePath == ":memory:" {
		return nil
	}

	dir := filepath.Dir(databasePath)
	if dir == "." || dir == "" {
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create database directory %q: %w", dir, err)
	}

	return nil
}

// SQLDB returns the underlying database handle for controlled tests and tooling.
func SQLDB(db *gorm.DB) (*sql.DB, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql database handle: %w", err)
	}

	return sqlDB, nil
}
