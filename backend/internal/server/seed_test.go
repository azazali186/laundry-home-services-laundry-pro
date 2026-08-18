package server

import (
	"context"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openSeedDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.ShopSetting{}, &models.ServiceType{}, &models.Combo{}, &models.ComboItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func countRows[T any](t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	var row T
	if err := db.Model(&row).Count(&count).Error; err != nil {
		t.Fatalf("count %T: %v", row, err)
	}
	return count
}

func TestSeedCatalogSeedsAndIsIdempotent(t *testing.T) {
	db := openSeedDB(t)
	ctx := context.Background()

	if err := SeedCatalog(ctx, db); err != nil {
		t.Fatalf("SeedCatalog: %v", err)
	}
	if got := countRows[models.ServiceType](t, db); got != 5 {
		t.Fatalf("service types = %d, want 5", got)
	}
	if got := countRows[models.Combo](t, db); got != 2 {
		t.Fatalf("combos = %d, want 2", got)
	}
	var items int64
	if err := db.Model(&models.ComboItem{}).Count(&items).Error; err != nil {
		t.Fatalf("count items: %v", err)
	}
	if items != 3 { // WEEKLY: WASH x4 · FAMILY: WASH x12 + LINEN x1
		t.Fatalf("combo items = %d, want 3", items)
	}

	// The weekly pass bundles 4 washes.
	var weekly models.Combo
	if err := db.Preload("Items").First(&weekly, "code = ?", "WEEKLY").Error; err != nil {
		t.Fatalf("weekly combo missing: %v", err)
	}
	if len(weekly.Items) != 1 || weekly.Items[0].Quantity != 4 {
		t.Fatalf("weekly items = %+v, want one item qty 4", weekly.Items)
	}

	// A second run must not duplicate anything.
	if err := SeedCatalog(ctx, db); err != nil {
		t.Fatalf("SeedCatalog second run: %v", err)
	}
	if got := countRows[models.ServiceType](t, db); got != 5 {
		t.Fatalf("service types after re-seed = %d, want 5", got)
	}
	if got := countRows[models.Combo](t, db); got != 2 {
		t.Fatalf("combos after re-seed = %d, want 2", got)
	}
}

func TestSeedCatalogUsesShopCurrency(t *testing.T) {
	db := openSeedDB(t)
	if err := db.Create(&models.ShopSetting{BusinessName: "Test Laundry", Currency: "USD", TaxRate: 0, InvoicePrefix: "INV"}).Error; err != nil {
		t.Fatalf("seed setting: %v", err)
	}
	if err := SeedCatalog(context.Background(), db); err != nil {
		t.Fatalf("SeedCatalog: %v", err)
	}
	var wash models.ServiceType
	if err := db.First(&wash, "code = ?", "WASH").Error; err != nil {
		t.Fatalf("wash missing: %v", err)
	}
	if wash.Currency != "USD" {
		t.Fatalf("seeded currency = %s, want USD from shop settings", wash.Currency)
	}
	var weekly models.Combo
	if err := db.First(&weekly, "code = ?", "WEEKLY").Error; err != nil {
		t.Fatalf("weekly missing: %v", err)
	}
	if weekly.Currency != "USD" {
		t.Fatalf("combo currency = %s, want USD from shop settings", weekly.Currency)
	}
}

func TestSeedCatalogMatchesPreExistingRowByName(t *testing.T) {
	db := openSeedDB(t)
	// A row with the canonical name but a different code (created earlier via
	// the API) must be adopted, not duplicated — the name unique constraint
	// must never be tripped by the seed.
	if err := db.Create(&models.ServiceType{Name: "Wash & Fold", Code: "WASHLEGACY", Unit: "kg", Price: 88, Currency: "INR"}).Error; err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}
	if err := SeedCatalog(context.Background(), db); err != nil {
		t.Fatalf("SeedCatalog with name conflict: %v", err)
	}
	if got := countRows[models.ServiceType](t, db); got != 5 {
		t.Fatalf("service types = %d, want 5 (no duplicate name)", got)
	}
	// Combo items must reference the adopted row's id.
	var wash models.ServiceType
	if err := db.First(&wash, "name = ?", "Wash & Fold").Error; err != nil {
		t.Fatalf("wash missing: %v", err)
	}
	var weekly models.Combo
	if err := db.Preload("Items").First(&weekly, "code = ?", "WEEKLY").Error; err != nil {
		t.Fatalf("weekly missing: %v", err)
	}
	if weekly.Items[0].ServiceTypeID != wash.ID {
		t.Fatalf("weekly item references %s, want adopted row %s", weekly.Items[0].ServiceTypeID, wash.ID)
	}
}
