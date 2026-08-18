package garment

import (
	"context"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
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
	if err := db.AutoMigrate(&models.Garment{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db, NewService(db, audit.NewWriter(db))
}

var testMeta = audit.Meta{UserID: uuid.New(), Method: "POST", Path: "/api/v1/garments"}

func TestGarmentCreateAndDefaults(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Name: "  Shirt  ", Quantity: 3}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if view.Name != "Shirt" || view.Status != models.GarmentReceived {
		t.Fatalf("unexpected view: %+v", view)
	}
	if view.Quantity != 3 {
		t.Fatalf("quantity = %d, want 3", view.Quantity)
	}
	// Non-positive quantity defaults to 1.
	one, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Name: "Sock"}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if one.Quantity != 1 {
		t.Fatalf("quantity = %d, want 1", one.Quantity)
	}
}

func TestGarmentCreateValidation(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateRequest{Name: "x"}, testMeta); err == nil {
		t.Fatal("missing order_id must fail")
	}
	if _, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New()}, testMeta); err == nil {
		t.Fatal("missing name must fail")
	}
}

func TestGarmentUpdateStatusAndDelete(t *testing.T) {
	db, svc := newEnv(t)
	ctx := context.Background()
	view, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Name: "Shirt"}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)

	status := models.GarmentCleaned
	updated, err := svc.Update(ctx, UpdateRequest{ID: id, Status: &status}, testMeta)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Status != models.GarmentCleaned {
		t.Fatalf("status = %s, want cleaned", updated.Status)
	}

	if err := svc.Delete(ctx, IDRequest{ID: id}, testMeta); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var count int64
	db.Model(&models.Garment{}).Count(&count)
	if count != 0 {
		t.Fatalf("rows after delete = %d, want 0", count)
	}
}

func TestGarmentUpdateFieldsAndNotFound(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Name: "Shirt", Color: "Blue"}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)

	name, color, qty := "Oxford Shirt", "White", 4
	instr, notes := "hang dry", "loose button"
	updated, err := svc.Update(ctx, UpdateRequest{
		ID: id, Name: &name, Color: &color, Quantity: &qty,
		Instructions: &instr, DamageNotes: &notes,
	}, testMeta)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != name || updated.Color != color || updated.Quantity != qty ||
		updated.Instructions != instr || updated.DamageNotes != notes {
		t.Fatalf("fields not applied: %+v", updated)
	}

	// Updating a missing garment is a 404 (fail loudly if the error is not an AppError).
	if _, err := svc.Update(ctx, UpdateRequest{ID: uuid.New(), Name: &name}, testMeta); err == nil {
		t.Fatal("update of missing garment must fail")
	} else if e := errors.As(err); e == nil || e.Code != 404 {
		t.Fatalf("expected 404, got %v", err)
	}
}

func TestGarmentDeleteNotFound(t *testing.T) {
	_, svc := newEnv(t)
	if err := svc.Delete(context.Background(), IDRequest{ID: uuid.New()}, testMeta); err == nil {
		t.Fatal("delete of missing garment must fail")
	} else if e := errors.As(err); e == nil || e.Code != 404 {
		t.Fatalf("expected 404, got %v", err)
	}
}

func TestGarmentListAndSummary(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()
	orderID := uuid.New()

	if _, err := svc.Create(ctx, CreateRequest{OrderID: orderID, Name: "Shirt", Quantity: 2}, testMeta); err != nil {
		t.Fatalf("seed A: %v", err)
	}
	b, err := svc.Create(ctx, CreateRequest{OrderID: orderID, Name: "Jeans", Quantity: 3}, testMeta)
	if err != nil {
		t.Fatalf("seed B: %v", err)
	}
	if _, err := svc.Create(ctx, CreateRequest{OrderID: orderID, Name: "Coat", Quantity: 1, DamageNotes: "stain at cuff"}, testMeta); err != nil {
		t.Fatalf("seed C: %v", err)
	}

	bid, _ := uuid.Parse(b.ID)
	cleaned := models.GarmentCleaned
	if _, err := svc.Update(ctx, UpdateRequest{ID: bid, Status: &cleaned}, testMeta); err != nil {
		t.Fatalf("mark cleaned: %v", err)
	}

	items, pg, summary, err := svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{Limit: 2}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pg.Total != 3 {
		t.Fatalf("total = %d, want 3", pg.Total)
	}
	if len(items) != 2 {
		t.Fatalf("page size = %d, want 2", len(items))
	}
	if summary == nil {
		t.Fatal("summary missing")
	}
	// TotalItems sums quantities across all rows (2 + 3 + 1).
	if summary.TotalItems != 6 {
		t.Fatalf("total_items = %d, want 6", summary.TotalItems)
	}
	// Received sums quantity of received garments: A(2) + C(1, never moved).
	if summary.Received != 3 || summary.Cleaned != 3 {
		t.Fatalf("received=%d cleaned=%d, want 3/3", summary.Received, summary.Cleaned)
	}
	if summary.Damaged != 1 {
		t.Fatalf("damaged = %d, want 1", summary.Damaged)
	}
	// ByStatus counts garments (not quantity): A+C received, B cleaned.
	if summary.ByStatus[models.GarmentReceived] != 2 || summary.ByStatus[models.GarmentCleaned] != 1 {
		t.Fatalf("by_status = %+v", summary.ByStatus)
	}
}

func TestGarmentListFilterByStatus(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()
	for _, n := range []string{"A", "B", "C"} {
		if _, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Name: n}, testMeta); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	items, pg, _, err := svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{
		Filters: map[string]string{"status": string(models.GarmentReceived)},
	}})
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if pg.Total != 3 || len(items) != 3 {
		t.Fatalf("filtered total=%d page=%d, want 3/3", pg.Total, len(items))
	}
	for _, it := range items {
		if it.Status != models.GarmentReceived {
			t.Fatalf("filter leaked status %s", it.Status)
		}
	}
}
