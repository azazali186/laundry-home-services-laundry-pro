package combo

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

type stubShop struct{ currency string }

func (s stubShop) Defaults(context.Context) (shopconfig.Defaults, error) {
	return shopconfig.Defaults{Currency: s.currency}, nil
}

var testMeta = audit.Meta{UserID: uuid.New(), Method: "POST", Path: "/api/v1/combos"}

func newEnv(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.ServiceType{}, &models.Combo{}, &models.ComboItem{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewService(db, audit.NewWriter(db), stubShop{currency: "INR"}), db
}

func errCode(err error) int {
	if e := errors.As(err); e != nil {
		return e.Code
	}
	return 0
}

func seedServiceTypes(t *testing.T, db *gorm.DB) (wash, linen uuid.UUID) {
	t.Helper()
	w := models.ServiceType{Name: "Wash & Fold", Code: "WASH", Unit: "kg", Price: 99, Currency: "INR"}
	l := models.ServiceType{Name: "Curtains & Home Linen", Code: "LINEN", Unit: "per_item", Price: 199, Currency: "INR"}
	if err := db.Create(&w).Error; err != nil {
		t.Fatalf("seed wash: %v", err)
	}
	if err := db.Create(&l).Error; err != nil {
		t.Fatalf("seed linen: %v", err)
	}
	return w.ID, l.ID
}

func TestComboCreateWithItemsAndShopCurrency(t *testing.T) {
	svc, db := newEnv(t)
	wash, _ := seedServiceTypes(t, db)
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{
		Name: "Weekly Pass", Code: "weekly", Unit: "per_week", Price: 499,
		Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 4}},
	}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if view.Currency != "INR" {
		t.Fatalf("currency = %s, want shop default INR", view.Currency)
	}
	if view.Code != "WEEKLY" {
		t.Fatalf("code = %s, want normalized WEEKLY", view.Code)
	}
	if len(view.Items) != 1 || view.Items[0].Quantity != 4 {
		t.Fatalf("items not attached: %+v", view.Items)
	}
	if !view.IsActive {
		t.Fatal("combo must default to active")
	}
}

func TestComboCreateValidation(t *testing.T) {
	svc, db := newEnv(t)
	wash, _ := seedServiceTypes(t, db)
	ctx := context.Background()

	if _, err := svc.Create(ctx, CreateRequest{Code: "X", Price: 10, Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 1}}}, testMeta); err == nil {
		t.Fatal("missing name must fail")
	}
	if _, err := svc.Create(ctx, CreateRequest{Name: "X", Price: 10, Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 1}}}, testMeta); err == nil {
		t.Fatal("missing code must fail")
	}
	if _, err := svc.Create(ctx, CreateRequest{Name: "X", Code: "Y", Price: -1, Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 1}}}, testMeta); err == nil {
		t.Fatal("negative price must fail")
	}
	if _, err := svc.Create(ctx, CreateRequest{Name: "X", Code: "Y", Price: 10}, testMeta); err == nil {
		t.Fatal("empty items must fail")
	}
	if _, err := svc.Create(ctx, CreateRequest{Name: "X", Code: "Y", Price: 10, Items: []ItemRequest{{ServiceTypeID: uuid.New(), Quantity: 1}}}, testMeta); errCode(err) != 400 {
		t.Fatalf("unknown service type: want 400, got %v", err)
	}
	if _, err := svc.Create(ctx, CreateRequest{Name: "X", Code: "Y", Price: 10, Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 0}}}, testMeta); errCode(err) != 400 {
		t.Fatalf("zero quantity: want 400, got %v", err)
	}
	// Duplicate code -> 409.
	if _, err := svc.Create(ctx, CreateRequest{Name: "First", Code: "dup", Price: 10, Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 1}}}, testMeta); err != nil {
		t.Fatalf("seed first: %v", err)
	}
	if _, err := svc.Create(ctx, CreateRequest{Name: "Second", Code: "DUP", Price: 20, Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 1}}}, testMeta); errCode(err) != 409 {
		t.Fatalf("duplicate code: want 409, got %v", err)
	}
}

func TestComboUpdateReplacesItemsAndDelete(t *testing.T) {
	svc, db := newEnv(t)
	wash, linen := seedServiceTypes(t, db)
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{
		Name: "Weekly Pass", Code: "WEEKLY", Price: 499,
		Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 4}},
	}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)

	// Unknown id -> 404.
	newPrice := 549.0
	if _, err := svc.Update(ctx, UpdateRequest{ID: uuid.New(), Price: &newPrice}, testMeta); errCode(err) != 404 {
		t.Fatalf("update unknown: want 404, got %v", err)
	}
	if err := svc.Delete(ctx, DeleteRequest{ID: uuid.New()}, testMeta); errCode(err) != 404 {
		t.Fatalf("delete unknown: want 404, got %v", err)
	}

	// Happy-path update: price + item replacement.
	updated, err := svc.Update(ctx, UpdateRequest{
		ID:    id,
		Price: &newPrice,
		Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 4}, {ServiceTypeID: linen, Quantity: 1}},
	}, testMeta)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Price != 549 || len(updated.Items) != 2 {
		t.Fatalf("update not applied: %+v", updated)
	}

	// Clearing items must be rejected.
	if _, err := svc.Update(ctx, UpdateRequest{ID: id, Items: []ItemRequest{}}, testMeta); errCode(err) != 400 {
		t.Fatalf("empty items: want 400, got %v", err)
	}

	if err := svc.Delete(ctx, DeleteRequest{ID: id}, testMeta); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Update(ctx, UpdateRequest{ID: id, Price: &newPrice}, testMeta); errCode(err) != 404 {
		t.Fatal("deleted combo must 404 on update")
	}
}

func TestComboListAndSummary(t *testing.T) {
	svc, db := newEnv(t)
	wash, _ := seedServiceTypes(t, db)
	ctx := context.Background()
	for _, c := range []CreateRequest{
		{Name: "Starter", Code: "a", Price: 199, Unit: "per_week", Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 2}}},
		{Name: "Weekly", Code: "b", Price: 499, Unit: "per_week", Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 4}}},
		{Name: "Family", Code: "c", Price: 1299, Unit: "per_month", Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 12}}},
	} {
		if _, err := svc.Create(ctx, c, testMeta); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	views, pg, summary, err := svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{Limit: 2}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pg.Total != 3 {
		t.Fatalf("total = %d, want 3", pg.Total)
	}
	if len(views) != 2 || len(views[0].Items) != 1 {
		t.Fatalf("page/items mismatch: %d rows", len(views))
	}
	if summary == nil {
		t.Fatal("summary missing")
	}
	if summary.TotalActive != 3 || math.Abs(summary.AvgPrice-665.6666) > 0.001 || summary.MaxPrice != 1299 || summary.MinPrice != 199 {
		t.Fatalf("summary mismatch: %+v", summary)
	}
	if summary.ByUnit["per_week"] != 2 || summary.ByUnit["per_month"] != 1 {
		t.Fatalf("by_unit mismatch: %+v", summary.ByUnit)
	}
}

func TestComboPricingBreakdown(t *testing.T) {
	svc, db := newEnv(t)
	wash, linen := seedServiceTypes(t, db)
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{
		Name: "Family Plan", Code: "FAMILY", Unit: "per_month", Price: 1299,
		Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 12}, {ServiceTypeID: linen, Quantity: 1}},
	}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// À la carte: 12 × ₹99 + 1 × ₹199 = ₹1387 → savings ₹88.
	if view.RegularPrice != 1387 {
		t.Fatalf("regular_price = %v, want 1387", view.RegularPrice)
	}
	if view.Savings != 88 {
		t.Fatalf("savings = %v, want 88", view.Savings)
	}
	if len(view.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(view.Items))
	}
	first := view.Items[0]
	if first.ServiceName != "Wash & Fold" || first.Unit != "kg" || first.Price != 99 || first.Currency != "INR" {
		t.Fatalf("item snapshot missing: %+v", first)
	}

	// A combo priced above its à-la-carte value must report zero savings.
	expensive, err := svc.Create(ctx, CreateRequest{
		Name: "Premium", Code: "PREMIUM", Unit: "per_week", Price: 500,
		Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 4}},
	}, testMeta)
	if err != nil {
		t.Fatalf("Create premium: %v", err)
	}
	if expensive.RegularPrice != 396 || expensive.Savings != 0 {
		t.Fatalf("premium breakdown = regular %v savings %v, want 396/0", expensive.RegularPrice, expensive.Savings)
	}
}

type fakeInvalidator struct{ calls int }

func (f *fakeInvalidator) InvalidateCatalog(context.Context) { f.calls++ }

func TestComboMutationsInvalidateCatalog(t *testing.T) {
	svc, db := newEnv(t)
	inv := &fakeInvalidator{}
	svc.SetCatalogInvalidator(inv)
	wash, _ := seedServiceTypes(t, db)
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{
		Name: "Weekly Pass", Code: "WEEKLY", Price: 499,
		Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 4}},
	}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if inv.calls != 1 {
		t.Fatalf("create must invalidate the catalog, got %d calls", inv.calls)
	}
	id, _ := uuid.Parse(view.ID)

	newPrice := 549.0
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

func TestComboListActiveOnly(t *testing.T) {
	svc, db := newEnv(t)
	wash, _ := seedServiceTypes(t, db)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateRequest{
		Name: "Active Combo", Code: "act", Price: 199, Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 2}},
	}, testMeta); err != nil {
		t.Fatalf("seed active: %v", err)
	}
	disabled := false
	if _, err := svc.Create(ctx, CreateRequest{
		Name: "Retired Combo", Code: "ret", Price: 299, Items: []ItemRequest{{ServiceTypeID: wash, Quantity: 3}}, IsActive: &disabled,
	}, testMeta); err != nil {
		t.Fatalf("seed retired: %v", err)
	}
	views, err := svc.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(views) != 1 || views[0].Code != "ACT" {
		t.Fatalf("ListActive = %d rows, want only the active combo: %+v", len(views), views)
	}
	if len(views[0].Items) != 1 {
		t.Fatalf("active combo items missing: %+v", views[0].Items)
	}
}
