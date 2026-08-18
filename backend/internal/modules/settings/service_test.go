package settings

import (
	"context"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newSettingsEnv(t *testing.T) (*gorm.DB, *Service) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.ShopSetting{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db, NewService(db, audit.NewWriter(db))
}

var testMeta = audit.Meta{UserID: uuid.New(), Method: "PATCH", Path: "/api/v1/settings"}

func TestGetCreatesDefaultRow(t *testing.T) {
	db, svc := newSettingsEnv(t)
	ctx := context.Background()

	view, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if view.BusinessName != "My Laundry" || view.Currency != "INR" || view.TaxRate != 0 {
		t.Fatalf("unexpected default view: %+v", view)
	}
	if view.InvoicePrefix != "INV" || view.Timezone != "Asia/Kolkata" {
		t.Fatalf("unexpected default view: %+v", view)
	}

	var count int64
	db.Model(&models.ShopSetting{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected a single settings row, got %d", count)
	}
}

func TestUpdatePersistsAndAudits(t *testing.T) {
	db, svc := newSettingsEnv(t)
	ctx := context.Background()

	name := "Sparkle Laundry"
	currency := "usd"
	taxRate := 7.5
	prefix := "spk"
	view, err := svc.Update(ctx, UpdateRequest{
		BusinessName:  &name,
		Currency:      &currency,
		TaxRate:       &taxRate,
		InvoicePrefix: &prefix,
	}, testMeta)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if view.BusinessName != name || view.Currency != "USD" || view.TaxRate != 7.5 {
		t.Fatalf("update not applied: %+v", view)
	}
	if view.InvoicePrefix != "SPK" {
		t.Fatalf("prefix not normalized: %+v", view)
	}

	// Audit entry must capture the before/after snapshots.
	var row models.AuditLog
	if err := db.First(&row, "entity = ?", "shop_setting").Error; err != nil {
		t.Fatalf("audit entry missing: %v", err)
	}
	if row.Action != "update" || len(row.Before) == 0 || len(row.After) == 0 {
		t.Fatalf("unexpected audit row: %+v", row)
	}
}

func TestUpdateValidation(t *testing.T) {
	_, svc := newSettingsEnv(t)
	ctx := context.Background()

	empty := "   "
	if _, err := svc.Update(ctx, UpdateRequest{BusinessName: &empty}, testMeta); err == nil {
		t.Fatal("empty business name must fail")
	}
	badTax := 150.0
	if _, err := svc.Update(ctx, UpdateRequest{TaxRate: &badTax}, testMeta); err == nil {
		t.Fatal("tax rate over 100 must fail")
	}
	badCurrency := "IN"
	_, err := svc.Update(ctx, UpdateRequest{Currency: &badCurrency}, testMeta)
	if err == nil {
		t.Fatal("invalid currency must fail")
	}
	appErr := errors.As(err)
	if appErr == nil || appErr.Code != 400 {
		t.Fatalf("expected 400, got %v", err)
	}
}

func TestDefaultsReflectSettings(t *testing.T) {
	_, svc := newSettingsEnv(t)
	ctx := context.Background()

	// Initial defaults before any update.
	d, err := svc.Defaults(ctx)
	if err != nil {
		t.Fatalf("Defaults: %v", err)
	}
	if d.Currency != "INR" || d.TaxRate != 0 || d.InvoicePrefix != "INV" {
		t.Fatalf("unexpected initial defaults: %+v", d)
	}

	// After an update the money-flow defaults must follow the settings row.
	currency := "USD"
	tax := 7.5
	prefix := "SPK"
	if _, err := svc.Update(ctx, UpdateRequest{
		Currency: &currency, TaxRate: &tax, InvoicePrefix: &prefix,
	}, testMeta); err != nil {
		t.Fatalf("Update: %v", err)
	}
	d, err = svc.Defaults(ctx)
	if err != nil {
		t.Fatalf("Defaults: %v", err)
	}
	if d.Currency != "USD" || d.TaxRate != 7.5 || d.InvoicePrefix != "SPK" {
		t.Fatalf("defaults not synced to settings: %+v", d)
	}
}

func TestUpdateClearsOptionalFields(t *testing.T) {
	_, svc := newSettingsEnv(t)
	ctx := context.Background()

	city := "Mumbai"
	if _, err := svc.Update(ctx, UpdateRequest{City: &city}, testMeta); err != nil {
		t.Fatalf("Update: %v", err)
	}
	// Clearing a pointer field back to an empty string must persist.
	clear := ""
	view, err := svc.Update(ctx, UpdateRequest{City: &clear}, testMeta)
	if err != nil {
		t.Fatalf("clear Update: %v", err)
	}
	if view.City != "" {
		t.Fatalf("city = %q, want cleared", view.City)
	}
}
