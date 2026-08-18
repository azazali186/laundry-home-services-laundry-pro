package pricing

import (
	"context"
	"testing"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/modules/combo"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/servicetype"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/shopconfig"
)

// stubServices returns a fixed active-services list.
type stubServices struct{ err error }

func (s stubServices) ListActive(context.Context) ([]servicetype.View, error) {
	if s.err != nil {
		return nil, s.err
	}
	return []servicetype.View{
		{ID: "1", Name: "Wash & Fold", Code: "WASH", Unit: "kg", Price: 99, Currency: "INR", IsActive: true},
		{ID: "2", Name: "Dry Cleaning", Code: "DRY", Unit: "per_item", Price: 149, Currency: "INR", IsActive: true},
	}, nil
}

// stubCombos returns a fixed active-combos list.
type stubCombos struct{ err error }

func (s stubCombos) ListActive(context.Context) ([]combo.View, error) {
	if s.err != nil {
		return nil, s.err
	}
	return []combo.View{
		{ID: "c1", Name: "Weekly Pass", Code: "WEEKLY", Unit: "per_week", Price: 499, Currency: "INR", IsActive: true,
			RegularPrice: 396, Savings: 0,
			Items: []combo.ItemView{{ServiceTypeID: "1", Quantity: 4, ServiceName: "Wash & Fold", Unit: "kg", Price: 99, Currency: "INR"}}},
	}, nil
}

type stubShop struct{}

func (stubShop) Defaults(context.Context) (shopconfig.Defaults, error) {
	return shopconfig.Defaults{BusinessName: "My Laundry", Currency: "INR", TaxRate: 5}, nil
}

func TestPricingPublicReturnsCatalog(t *testing.T) {
	svc := NewService(stubServices{}, stubCombos{}, stubShop{}, nil)
	view, err := svc.Public(context.Background())
	if err != nil {
		t.Fatalf("Public: %v", err)
	}
	if view.BusinessName != "My Laundry" || view.Currency != "INR" || view.TaxRate != 5 {
		t.Fatalf("shop defaults missing: %+v", view)
	}
	if len(view.Services) != 2 || view.Services[0].Price != 99 {
		t.Fatalf("services mismatch: %+v", view.Services)
	}
	if len(view.Combos) != 1 || view.Combos[0].Price != 499 || len(view.Combos[0].Items) != 1 {
		t.Fatalf("combos mismatch: %+v", view.Combos)
	}
	if view.Combos[0].RegularPrice != 396 || view.Combos[0].Savings != 0 {
		t.Fatalf("combo breakdown missing: %+v", view.Combos[0])
	}
	if item := view.Combos[0].Items[0]; item.ServiceName != "Wash & Fold" || item.Price != 99 {
		t.Fatalf("combo item snapshot missing: %+v", item)
	}
}

func TestPricingPublicPropagatesReaderErrors(t *testing.T) {
	if _, err := NewService(stubServices{err: errBoom}, stubCombos{}, stubShop{}, nil).Public(context.Background()); err == nil {
		t.Fatal("services reader error must propagate")
	}
	if _, err := NewService(stubServices{}, stubCombos{err: errBoom}, stubShop{}, nil).Public(context.Background()); err == nil {
		t.Fatal("combos reader error must propagate")
	}
}

func TestPricingPublicNilSafeShop(t *testing.T) {
	svc := NewService(stubServices{}, stubCombos{}, nil, nil)
	view, err := svc.Public(context.Background())
	if err != nil {
		t.Fatalf("Public with nil shop: %v", err)
	}
	if view.Currency != "" {
		t.Fatalf("nil shop must degrade to empty defaults, got %+v", view)
	}
	if len(view.Services) != 2 {
		t.Fatalf("services missing with nil shop: %+v", view)
	}
}

// countingServices wraps the stub so tests can observe reader invocations.
type countingServices struct {
	stubServices
	calls int
}

func (s *countingServices) ListActive(ctx context.Context) ([]servicetype.View, error) {
	s.calls++
	return s.stubServices.ListActive(ctx)
}

type fakeCache struct {
	stored *View
}

func (f *fakeCache) GetJSON(_ context.Context, _ string, v any) error {
	if f.stored == nil {
		return errBoom // cache miss
	}
	out := v.(*View)
	*out = *f.stored
	return nil
}

func (f *fakeCache) SetJSON(_ context.Context, _ string, v any, _ time.Duration) error {
	view := v.(*View)
	copy := *view
	f.stored = &copy
	return nil
}

func (f *fakeCache) Del(_ context.Context, _ ...string) error {
	f.stored = nil
	return nil
}

func TestPricingInvalidateCatalogRebuilds(t *testing.T) {
	counter := &countingServices{}
	svc := NewService(counter, stubCombos{}, stubShop{}, &fakeCache{})
	ctx := context.Background()
	if _, err := svc.Public(ctx); err != nil {
		t.Fatalf("first Public: %v", err)
	}
	if counter.calls != 1 {
		t.Fatalf("first call must hit the readers once, got %d", counter.calls)
	}
	svc.InvalidateCatalog(ctx)
	if _, err := svc.Public(ctx); err != nil {
		t.Fatalf("post-invalidate Public: %v", err)
	}
	if counter.calls != 2 {
		t.Fatalf("catalog must rebuild after invalidation, readers called %d times", counter.calls)
	}
}

func TestPricingPublicCachesCatalog(t *testing.T) {
	counter := &countingServices{}
	svc := NewService(counter, stubCombos{}, stubShop{}, &fakeCache{})
	ctx := context.Background()
	if _, err := svc.Public(ctx); err != nil {
		t.Fatalf("first Public: %v", err)
	}
	if counter.calls != 1 {
		t.Fatalf("first call must hit the readers once, got %d", counter.calls)
	}
	if _, err := svc.Public(ctx); err != nil {
		t.Fatalf("cached Public: %v", err)
	}
	if counter.calls != 1 {
		t.Fatalf("second call must be served from cache, readers called %d times", counter.calls)
	}
}

var errBoom = &fakeError{}

type fakeError struct{}

func (f *fakeError) Error() string { return "boom" }
