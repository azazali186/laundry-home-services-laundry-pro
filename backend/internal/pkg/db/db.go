// Package db initializes GORM with PostgreSQL (primary) or SQLite (tests).
// Auto-migration is managed here; no raw SQL is used anywhere in the app.
package db

import (
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Options configures the GORM connection.
type Options struct {
	DSN             string
	SQLite          bool
	LogLevel        logger.LogLevel
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DefaultOptions returns sensible production defaults.
func DefaultOptions(dsn string) Options {
	return Options{
		DSN:             dsn,
		LogLevel:        logger.Warn,
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 30 * time.Minute,
	}
}

// Open opens GORM using the Postgres driver (or SQLite when opts.SQLite).
func Open(opts Options) (*gorm.DB, error) {
	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(opts.LogLevel)}
	var dialector gorm.Dialector
	if opts.SQLite {
		dialector = sqlite.Open(opts.DSN)
	} else {
		dialector = postgres.Open(opts.DSN)
	}
	gdb, err := gorm.Open(dialector, gormCfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
	sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(opts.ConnMaxLifetime)
	return gdb, nil
}
