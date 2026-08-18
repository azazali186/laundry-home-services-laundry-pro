// Package migrate applies schema changes through a versioned migration
// runner. The baseline (20260810000000) creates the current schema via
// AutoMigrate; every later change is an explicit, ordered migration using
// only the GORM Migrator API (no raw SQL).
package migrate

import (
	"fmt"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"gorm.io/gorm"
)

// All returns every model in a deterministic order (parents before children
// where foreign keys exist).
func All() []any {
	return []any{
		&models.ServiceType{},
		&models.Combo{},
		&models.ComboItem{},
		&models.Role{},
		&models.Permission{},
		&models.User{},
		&models.Booking{},
		&models.Pickup{},
		&models.Order{},
		&models.Delivery{},
		&models.Payment{},
		&models.Garment{},
		&models.Invoice{},
		&models.Rating{},
		&models.InventoryItem{},
		&models.Notification{},
		&models.ShopSetting{},
		&models.LoginLog{},
		&models.AuditLog{},
		&models.OutboxEvent{},
		&models.SagaState{},
	}
}

// Run applies pending migrations in order and records them in
// schema_migrations. Idempotent and safe to call on every boot.
func Run(db *gorm.DB) error {
	if err := NewRunner(db).Run(Migrations()); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// Rollback reverts migrations down to (and including) the target version.
func Rollback(db *gorm.DB, targetVersion string) error {
	if err := NewRunner(db).Rollback(Migrations(), targetVersion); err != nil {
		return fmt.Errorf("rollback to %s: %w", targetVersion, err)
	}
	return nil
}

// Migrations is the ordered migration history. Append new schema changes
// here with a version greater than the last one; never edit history.
func Migrations() []Migration {
	return []Migration{
		{
			Version: "20260810000000",
			Name:    "initial_schema",
			// Baseline: AutoMigrate the full current model set. AutoMigrate
			// never drops columns, so it is safe as the initial schema and
			// as a compatibility check for older databases.
			Up: func(db *gorm.DB) error {
				return db.AutoMigrate(All()...)
			},
		},
		{
			Version: "20260811000000",
			Name:    "add_shop_settings",
			// Singleton shop configuration table (created via AutoMigrate, no
			// raw SQL); the settings service lazily seeds the default row.
			Up: func(db *gorm.DB) error {
				return db.AutoMigrate(&models.ShopSetting{})
			},
		},
		{
			Version: "20260811000001",
			Name:    "seed_shop_settings",
			// Deterministically create the singleton row before any traffic so
			// the service's lazy ensure() can never race two creations.
			Up: func(db *gorm.DB) error {
				var count int64
				if err := db.Model(&models.ShopSetting{}).Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					return nil
				}
				return db.Create(&models.ShopSetting{
					BusinessName:  "My Laundry",
					Currency:      "INR",
					InvoicePrefix: "INV",
					Timezone:      "Asia/Kolkata",
				}).Error
			},
		},
		{
			Version: "20260812000000",
			Name:    "add_combos",
			// Combo bundle pricing: a combo groups service types with quantities
			// and carries its own explicit bundle price.
			Up: func(db *gorm.DB) error {
				return db.AutoMigrate(&models.Combo{}, &models.ComboItem{})
			},
			Down: func(db *gorm.DB) error {
				return db.Migrator().DropTable(&models.ComboItem{}, &models.Combo{})
			},
		},
		{
			Version: "20260818000000",
			Name:    "add_combo_is_featured",
			Up: func(db *gorm.DB) error {
				if db.Migrator().HasColumn(&models.Combo{}, "is_featured") {
					return nil
				}
				return db.Migrator().AddColumn(&models.Combo{}, "is_featured")
			},
			Down: func(db *gorm.DB) error {
				if !db.Migrator().HasColumn(&models.Combo{}, "is_featured") {
					return nil
				}
				return db.Migrator().DropColumn(&models.Combo{}, "is_featured")
			},
		},
		{
			Version: "20260818000001",
			Name:    "add_combo_validity_renewal",
			Up: func(db *gorm.DB) error {
				m := db.Migrator()
				if !m.HasColumn(&models.Combo{}, "validity_days") {
					if err := m.AddColumn(&models.Combo{}, "validity_days"); err != nil {
						return err
					}
				}
				if !m.HasColumn(&models.Combo{}, "renewal_type") {
					if err := m.AddColumn(&models.Combo{}, "renewal_type"); err != nil {
						return err
					}
				}
				return nil
			},
			Down: func(db *gorm.DB) error {
				m := db.Migrator()
				if m.HasColumn(&models.Combo{}, "renewal_type") {
					if err := m.DropColumn(&models.Combo{}, "renewal_type"); err != nil {
						return err
					}
				}
				if m.HasColumn(&models.Combo{}, "validity_days") {
					return m.DropColumn(&models.Combo{}, "validity_days")
				}
				return nil
			},
		},
	}
}
