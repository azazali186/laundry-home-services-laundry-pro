package servicetype

import (
	"context"
	"math"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/shopconfig"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newServiceEnv(t *testing.T, shop stubShop) *Service {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.ServiceType{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewService(db, audit.NewWriter(db), shop)
}

// stubShop is a fixed shop-configuration provider for hermetic tests.
type stubShop struct{ currency string }

func (s stubShop) Defaults(context.Context) (shopconfig.Defaults, error) {
	return shopconfig.Defaults{Currency: s.currency}, nil
}

var testMeta = audit.Meta{UserID: uuid.New(), Method: "POST", Path: "/api/v1/service-types"}

func TestServiceTypeCreateAppliesShopCurrencyDefault(t *testing.T) {
	svc := newServiceEnv(t, stubShop{currency: "USD"})

	view, err := svc.Create(context.Background(), CreateRequest{
		Name: "Wash & Fold", Code: "wash", Price: 50,
	}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if view.Currency != "USD" {
		t.Fatalf("currency = %s, want shop default USD", view.Currency)
	}
	if view.Code != "WASH" {
		t.Fatalf("code = %s, want normalized WASH", view.Code)
	}
	if view.Unit != "kg" || view.EstimatedDays != 2 {
		t.Fatalf("defaults not applied: %+v", view)
	}
}

func TestServiceTypeCreateValidation(t *testing.T) {
	svc := newServiceEnv(t, stubShop{currency: "INR"})
	ctx := context.Background()

	if _, err := svc.Create(ctx, CreateRequest{Code: "X", Price: 10}, testMeta); err == nil {
		t.Fatal("missing name must fail")
	}
	if _, err := svc.Create(ctx, CreateRequest{Name: "X"}, testMeta); err == nil {
		t.Fatal("missing code must fail")
	}
	neg := -5.0
	if _, err := svc.Create(ctx, CreateRequest{Name: "X", Code: "Y", Price: neg}, testMeta); err == nil {
		t.Fatal("negative price must fail")
	}
	// Duplicate code -> 409.
	if _, err := svc.Create(ctx, CreateRequest{Name: "First", Code: "dup", Price: 5}, testMeta); err != nil {
		t.Fatalf("seed first: %v", err)
	}
	if _, err := svc.Create(ctx, CreateRequest{Name: "Second", Code: "DUP", Price: 1}, testMeta); errCode(err) != 409 {
		t.Fatalf("duplicate code: want 409, got %v", err)
	}
}

func errCode(err error) int {
	if e := errors.As(err); e != nil {
		return e.Code
	}
	return 0
}

func TestServiceTypeUpdateAndDelete(t *testing.T) {
	svc := newServiceEnv(t, stubShop{currency: "INR"})
	ctx := context.Background()
	view, err := svc.Create(ctx, CreateRequest{Name: "Dry Clean", Code: "dry", Price: 80, TaxRate: 5}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)

	// Unknown id -> 404 on update/delete.
	newPrice := 99.0
	if _, err := svc.Update(ctx, UpdateRequest{ID: uuid.New(), Price: &newPrice}, testMeta); errCode(err) != 404 {
		t.Fatalf("update unknown: want 404, got %v", err)
	}
	if err := svc.Delete(ctx, DeleteRequest{ID: uuid.New()}, testMeta); errCode(err) != 404 {
		t.Fatalf("delete unknown: want 404, got %v", err)
	}

	// Happy-path update.
	updated, err := svc.Update(ctx, UpdateRequest{ID: id, Price: &newPrice}, testMeta)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Price != 99 || updated.Name != "Dry Clean" {
		t.Fatalf("update not applied: %+v", updated)
	}

	if err := svc.Delete(ctx, DeleteRequest{ID: id}, testMeta); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Update(ctx, UpdateRequest{ID: id, Price: &newPrice}, testMeta); errCode(err) != 404 {
		t.Fatal("deleted row must 404 on update")
	}
}

type fakeInvalidator struct{ calls int }

func (f *fakeInvalidator) InvalidateCatalog(context.Context) { f.calls++ }

func TestServiceTypeMutationsInvalidateCatalog(t *testing.T) {
	svc := newServiceEnv(t, stubShop{currency: "INR"})
	inv := &fakeInvalidator{}
	svc.SetCatalogInvalidator(inv)
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{Name: "Wash", Code: "WASH", Price: 50}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if inv.calls != 1 {
		t.Fatalf("create must invalidate the catalog, got %d calls", inv.calls)
	}
	id, _ := uuid.Parse(view.ID)

	newPrice := 99.0
	if _, err := svc.Update(ctx, UpdateRequest{ID: id, Price: &newPrice}, testMeta); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if inv.calls != 2 {
		t.Fatalf("update must invalidate the catalog, got %d calls", inv.calls)
	}

	// A failed mutation must not invalidate.
	if _, err := svc.Update(ctx, UpdateRequest{ID: uuid.New(), Price: &newPrice}, testMeta); err == nil {
		t.Fatal("unknown update must fail")
	}
	if inv.calls != 2 {
		t.Fatalf("failed mutation must not invalidate, got %d calls", inv.calls)
	}

	if err := svc.Delete(ctx, DeleteRequest{ID: id}, testMeta); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if inv.calls != 3 {
		t.Fatalf("delete must invalidate the catalog, got %d calls", inv.calls)
	}
}

func TestServiceTypeListAndSummary(t *testing.T) {
	svc := newServiceEnv(t, stubShop{currency: "INR"})
	ctx := context.Background()
	for _, st := range []CreateRequest{
		{Name: "A", Code: "a", Price: 10, Unit: "kg"},
		{Name: "B", Code: "b", Price: 30, Unit: "kg"},
		{Name: "C", Code: "c", Price: 60, Unit: "piece"},
	} {
		if _, err := svc.Create(ctx, st, testMeta); err != nil {
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
	if summary == nil {
		t.Fatal("summary missing")
	}
	if summary.TotalActive != 3 || math.Abs(summary.AvgPrice-33.3333) > 0.001 || summary.MaxPrice != 60 || summary.MinPrice != 10 {
		t.Fatalf("summary mismatch: %+v", summary)
	}
	if summary.ByUnit["kg"] != 2 || summary.ByUnit["piece"] != 1 {
		t.Fatalf("by_unit mismatch: %+v", summary.ByUnit)
	}
}
