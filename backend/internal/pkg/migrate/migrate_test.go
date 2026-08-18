package migrate

import (
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openMigrateDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func TestRunnerAppliesInOrderAndIsIdempotent(t *testing.T) {
	db := openMigrateDB(t)
	runner := NewRunner(db)

	var order []string
	migs := []Migration{
		{Version: "20260000000002", Name: "second", Up: func(db *gorm.DB) error { order = append(order, "2"); return nil }},
		{Version: "20260000000001", Name: "first", Up: func(db *gorm.DB) error { order = append(order, "1"); return nil }},
	}
	if err := runner.Run(migs); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if len(order) != 2 || order[0] != "1" || order[1] != "2" {
		t.Fatalf("migrations applied out of order: %v", order)
	}
	// The versions table must record both.
	var count int64
	if err := db.Model(&migrationRow{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("recorded versions = %d, want 2", count)
	}
	// Re-running must be a no-op.
	order = nil
	if err := runner.Run(migs); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if len(order) != 0 {
		t.Fatalf("migrations re-applied on idempotent run: %v", order)
	}
}

func TestRunnerRollsBackFailingMigration(t *testing.T) {
	db := openMigrateDB(t)
	runner := NewRunner(db)

	migs := []Migration{
		{Version: "20260000000001", Name: "ok", Up: func(db *gorm.DB) error { return nil }},
		{Version: "20260000000002", Name: "boom", Up: func(db *gorm.DB) error {
			return gorm.ErrRecordNotFound
		}},
	}
	if err := runner.Run(migs); err == nil {
		t.Fatal("failing migration must return an error")
	}
	// The first migration committed; the failed version must not be recorded
	// (its transaction rolled back).
	var failed int64
	db.Model(&migrationRow{}).Where("version = ?", "20260000000002").Count(&failed)
	if failed != 0 {
		t.Fatalf("failed migration recorded %d rows, want 0 (transaction rollback)", failed)
	}
	var ok int64
	db.Model(&migrationRow{}).Where("version = ?", "20260000000001").Count(&ok)
	if ok != 1 {
		t.Fatalf("successful migration rows = %d, want 1", ok)
	}
}

func TestBaselineMigrationsIncludeInitialSchema(t *testing.T) {
	db := openMigrateDB(t)
	if err := Run(db); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Every migration version must be recorded exactly once.
	var count int64
	db.Model(&migrationRow{}).Count(&count)
	if count != int64(len(Migrations())) {
		t.Fatalf("recorded versions = %d, want %d (full migration set)", count, len(Migrations()))
	}
	// The applied schema must include both the baseline entities and any
	// later additive migrations (e.g. shop settings).
	for _, model := range []any{
		&models.User{}, &models.Role{}, &models.Permission{},
		&models.Booking{}, &models.ShopSetting{},
	} {
		if !db.Migrator().HasTable(model) {
			t.Fatalf("migrations did not create table for %T", model)
		}
	}
	// Idempotent on a second call.
	if err := Run(db); err != nil {
		t.Fatalf("second Run: %v", err)
	}
}

func TestRollbackRevertsToTargetVersion(t *testing.T) {
	db := openMigrateDB(t)
	if err := Run(db); err != nil {
		t.Fatalf("Run: %v", err)
	}
	all := Migrations()
	if len(all) < 2 {
		t.Fatal("need at least 2 migrations for rollback test")
	}
	// Roll back to the second migration — combos table should be dropped.
	target := all[len(all)-1].Version
	if err := Rollback(db, target); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	// The combo tables should be gone (Down drops them).
	if db.Migrator().HasTable(&models.Combo{}) {
		t.Fatal("Combo table should not exist after rollback")
	}
	// The migration record for the target must be removed.
	var count int64
	db.Model(&migrationRow{}).Where("version = ?", target).Count(&count)
	if count != 0 {
		t.Fatalf("rolled-back migration still recorded: %d rows", count)
	}
}

func TestRollbackRejectsUnappliedVersion(t *testing.T) {
	db := openMigrateDB(t)
	if err := Run(db); err != nil {
		t.Fatalf("Run: %v", err)
	}
	err := Rollback(db, "99999999999999")
	if err == nil {
		t.Fatal("rollback of unapplied version must fail")
	}
}
