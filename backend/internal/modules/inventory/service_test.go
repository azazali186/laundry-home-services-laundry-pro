package inventory

import (
	"context"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newEnv(t *testing.T) (*gorm.DB, *Service) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.InventoryItem{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	c := cache.NewFromClient(client, "test")
	return db, NewService(db, c, audit.NewWriter(db))
}

var testMeta = audit.Meta{UserID: uuid.New(), Method: "POST", Path: "/api/v1/inventory"}

func TestInventoryCreateNormalizesSKU(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{Name: "Detergent", SKU: "  det-01  ", QuantityOnHand: 10, ReorderLevel: 3}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if view.SKU != "DET-01" || !view.IsActive || view.Unit != "unit" {
		t.Fatalf("unexpected view: %+v", view)
	}
	if _, err := svc.Create(ctx, CreateRequest{SKU: "X"}, testMeta); err == nil {
		t.Fatal("missing name must fail")
	}
	if _, err := svc.Create(ctx, CreateRequest{Name: "X"}, testMeta); err == nil {
		t.Fatal("missing sku must fail")
	}
}

func TestInventoryAdjustStock(t *testing.T) {
	db, svc := newEnv(t)
	ctx := context.Background()
	view, err := svc.Create(ctx, CreateRequest{Name: "Detergent", SKU: "DET-01", QuantityOnHand: 10}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)

	// Consume 4 -> 6 remaining.
	after, err := svc.AdjustStock(ctx, AdjustRequest{ID: id, Delta: -4}, testMeta)
	if err != nil {
		t.Fatalf("AdjustStock: %v", err)
	}
	if after.QuantityOnHand != 6 {
		t.Fatalf("quantity = %v, want 6", after.QuantityOnHand)
	}
	// Consuming below zero is rejected.
	if _, err := svc.AdjustStock(ctx, AdjustRequest{ID: id, Delta: -100}, testMeta); err == nil {
		t.Fatal("oversell must fail")
	}
	var row models.InventoryItem
	db.First(&row, "id = ?", id)
	if row.QuantityOnHand != 6 {
		t.Fatalf("quantity unchanged after rejected adjust: %v", row.QuantityOnHand)
	}
}

func errCode(err error) int {
	if e := errors.As(err); e != nil {
		return e.Code
	}
	return 0
}

func TestInventoryUpdateAndDelete(t *testing.T) {
	db, svc := newEnv(t)
	ctx := context.Background()
	view, err := svc.Create(ctx, CreateRequest{Name: "Soap", SKU: "S-1", QuantityOnHand: 5, ReorderLevel: 2, UnitCost: 10}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)

	if _, err := svc.Update(ctx, UpdateRequest{ID: uuid.New()}, testMeta); errCode(err) != 404 {
		t.Fatalf("update unknown: want 404, got %v", err)
	}

	// Happy-path update: rename + reorder level.
	reorder := 9.0
	updated, err := svc.Update(ctx, UpdateRequest{ID: id, ReorderLevel: &reorder}, testMeta)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.ReorderLevel != 9 {
		t.Fatalf("reorder level not updated: %+v", updated)
	}
	if err := svc.Delete(ctx, IDRequest{ID: id}, testMeta); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var count int64
	db.Model(&models.InventoryItem{}).Where("id = ?", id).Count(&count)
	if count != 0 {
		t.Fatal("item not deleted")
	}
}

func TestInventoryDuplicateSKUConflict(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateRequest{Name: "A", SKU: "DUP-1"}, testMeta); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := svc.Create(ctx, CreateRequest{Name: "B", SKU: "dup-1"}, testMeta); errCode(err) != 409 {
		t.Fatalf("duplicate sku: want 409, got %v", err)
	}
}

func TestInventoryListAndSummary(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()
	for _, item := range []CreateRequest{
		{Name: "A", SKU: "A-1", Category: "chem", QuantityOnHand: 10, ReorderLevel: 3, UnitCost: 5},
		{Name: "B", SKU: "B-1", Category: "chem", QuantityOnHand: 2, ReorderLevel: 4, UnitCost: 10},
		{Name: "C", SKU: "C-1", Category: "pack", QuantityOnHand: 0, ReorderLevel: 1, UnitCost: 20},
	} {
		if _, err := svc.Create(ctx, item, testMeta); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	_, pg, summary, err := svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{Limit: 2}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pg.Total != 3 {
		t.Fatalf("total = %d, want 3", pg.Total)
	}
	// B: qty 2 <= reorder 4 -> low stock; C: qty 0 -> out of stock.
	if summary.TotalItems != 3 || summary.LowStock != 1 || summary.OutOfStock != 1 {
		t.Fatalf("summary stock mismatch: %+v", summary)
	}
	if summary.TotalValue != 10*5+2*10+0*20 {
		t.Fatalf("total_value = %v, want 70", summary.TotalValue)
	}
	if summary.ByCategory["chem"] != 2 || summary.ByCategory["pack"] != 1 {
		t.Fatalf("by_category mismatch: %+v", summary.ByCategory)
	}
}
