package order

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/bus"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/outbox"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/shopconfig"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newOrderEnv(t *testing.T) (*gorm.DB, *Service, *bus.Bus) {
	return newOrderEnvWithShop(t, stubShop{currency: "INR", taxRate: 12})
}

func newOrderEnvWithShop(t *testing.T, shop stubShop) (*gorm.DB, *Service, *bus.Bus) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Order{}, &models.Payment{}, &models.OutboxEvent{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	b := bus.New()
	svc := NewService(db, outbox.NewRepo(db, 5), b, audit.NewWriter(db), shop)
	return db, svc, b
}

// stubShop is a fixed shop-configuration provider for hermetic tests.
type stubShop struct {
	currency, prefix string
	taxRate          float64
}

func (s stubShop) Defaults(context.Context) (shopconfig.Defaults, error) {
	return shopconfig.Defaults{Currency: s.currency, TaxRate: s.taxRate, InvoicePrefix: s.prefix}, nil
}

func seedOrder(t *testing.T, db *gorm.DB, status string) models.Order {
	t.Helper()
	row := models.Order{
		OrderNo:       uuid.New().String()[:12],
		CustomerID:    uuid.New(),
		ServiceTypeID: uuid.New(),
		Status:        status,
		Priority:      "normal",
		TotalAmount:   100,
		GrandTotal:    112,
		Currency:      "INR",
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed order: %v", err)
	}
	return row
}

func orderID(t *testing.T, row models.Order) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(row.ID.String())
	if err != nil {
		t.Fatalf("parse id: %v", err)
	}
	return id
}

func TestOrderCreateDerivesAmountsAndEmitsEvents(t *testing.T) {
	db, svc, b := newOrderEnv(t)
	ctx := context.Background()
	sub := b.Subscribe("order_created")

	view, err := svc.Create(ctx, CreateRequest{
		CustomerID: uuid.New(), ServiceTypeID: uuid.New(),
		TotalAmount: 100, TaxRate: f64(12), Currency: "INR",
	}, audit.Meta{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if view.Status != models.OrderCreated || view.OrderNo == "" {
		t.Fatalf("unexpected view: %+v", view)
	}
	if view.TaxAmount != 12 || view.GrandTotal != 112 {
		t.Fatalf("amounts not derived: tax=%v grand=%v", view.TaxAmount, view.GrandTotal)
	}

	var ev models.OutboxEvent
	if err := db.First(&ev, "event_type = ?", "order_created").Error; err != nil {
		t.Fatalf("order_created outbox event missing: %v", err)
	}
	select {
	case got := <-sub:
		if got.Type != "order_created" || got.AggregateID != view.ID {
			t.Fatalf("bus event mismatch: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("bus event never published")
	}
}

func TestOrderPipelineTransitions(t *testing.T) {
	db, svc, b := newOrderEnv(t)
	ctx := context.Background()
	row := seedOrder(t, db, models.OrderCreated)
	id := orderID(t, row)
	sub := b.Subscribe("order_status_changed")

	steps := []string{models.OrderProcessing, models.OrderQualityCheck, models.OrderReady, models.OrderOutForDelivery, models.OrderDelivered}
	for i, s := range steps {
		v, err := svc.UpdateStatus(ctx, UpdateStatusRequest{ID: id, Status: s}, audit.Meta{UserID: uuid.New()})
		if err != nil {
			t.Fatalf("transition to %s: %v", s, err)
		}
		if v.Status != s {
			t.Fatalf("status = %s, want %s", v.Status, s)
		}
		// The saga-driving event must be queued per transition.
		var evs []models.OutboxEvent
		if err := db.Where("event_type = ?", "order_status_changed").Order("created_at ASC").Find(&evs).Error; err != nil {
			t.Fatalf("order_status_changed events missing: %v", err)
		}
		var payload struct {
			To string `json:"to"`
		}
		if err := json.Unmarshal(evs[i].Payload, &payload); err != nil {
			t.Fatalf("decode payload %d: %v", i, err)
		}
		if payload.To != s {
			t.Fatalf("event %d to = %q, want %q", i, payload.To, s)
		}
	}
	select {
	case <-sub:
	default:
		t.Fatal("bus event not published")
	}
}

func TestOrderInvalidTransitionsRejected(t *testing.T) {
	db, svc, _ := newOrderEnv(t)
	ctx := context.Background()
	id := orderID(t, seedOrder(t, db, models.OrderCreated))

	// Skipping straight to ready is not allowed by the pipeline.
	if _, err := svc.UpdateStatus(ctx, UpdateStatusRequest{ID: id, Status: models.OrderReady}, audit.Meta{UserID: uuid.New()}); err == nil {
		t.Fatal("created -> ready must be rejected")
	}
	// Advancing two steps in one call is rejected (pipeline allows only one).
	id2 := orderID(t, seedOrder(t, db, models.OrderCreated))
	if _, err := svc.UpdateStatus(ctx, UpdateStatusRequest{ID: id2, Status: models.OrderQualityCheck}, audit.Meta{UserID: uuid.New()}); err == nil {
		t.Fatal("created -> quality_check must be rejected")
	}
	// Unknown orders 404.
	missing := uuid.New()
	if _, err := svc.UpdateStatus(ctx, UpdateStatusRequest{ID: missing, Status: models.OrderProcessing}, audit.Meta{UserID: uuid.New()}); err == nil {
		t.Fatal("unknown order must fail")
	}
}

func TestOrderCancelRules(t *testing.T) {
	db, svc, _ := newOrderEnv(t)
	ctx := context.Background()

	id := orderID(t, seedOrder(t, db, models.OrderCreated))
	if _, err := svc.UpdateStatus(ctx, UpdateStatusRequest{ID: id, Status: models.OrderCancelled}, audit.Meta{UserID: uuid.New()}); err != nil {
		t.Fatalf("cancel created order: %v", err)
	}

	id2 := orderID(t, seedOrder(t, db, models.OrderDelivered))
	if _, err := svc.UpdateStatus(ctx, UpdateStatusRequest{ID: id2, Status: models.OrderCancelled}, audit.Meta{UserID: uuid.New()}); err == nil {
		t.Fatal("cancelling a delivered order must fail")
	}
}

func TestOrderCreateAppliesShopTaxAndCurrencyDefaults(t *testing.T) {
	_, svc, _ := newOrderEnvWithShop(t, stubShop{currency: "USD", taxRate: 7.5})
	ctx := context.Background()

	// No currency and no tax rate on the request: shop defaults apply.
	view, err := svc.Create(ctx, CreateRequest{
		CustomerID: uuid.New(), ServiceTypeID: uuid.New(), TotalAmount: 200,
	}, audit.Meta{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if view.Currency != "USD" {
		t.Fatalf("currency = %s, want shop default USD", view.Currency)
	}
	if view.TaxAmount != 15 { // 200 * 7.5% = 15
		t.Fatalf("tax_amount = %v, want 15 (shop default rate)", view.TaxAmount)
	}
	if view.GrandTotal != 215 {
		t.Fatalf("grand_total = %v, want 215", view.GrandTotal)
	}

	// An explicit tax rate on the request still wins over the default.
	explicit, err := svc.Create(ctx, CreateRequest{
		CustomerID: uuid.New(), ServiceTypeID: uuid.New(), TotalAmount: 200, TaxRate: f64(10),
	}, audit.Meta{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("Create explicit: %v", err)
	}
	if explicit.TaxAmount != 20 {
		t.Fatalf("tax_amount = %v, want 20 (explicit rate wins)", explicit.TaxAmount)
	}

	// An explicit 0% must be honored as tax-exempt, not replaced by the default.
	exempt, err := svc.Create(ctx, CreateRequest{
		CustomerID: uuid.New(), ServiceTypeID: uuid.New(), TotalAmount: 200, TaxRate: f64(0),
	}, audit.Meta{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("Create exempt: %v", err)
	}
	if exempt.TaxAmount != 0 {
		t.Fatalf("tax_amount = %v, want 0 (explicit 0%% honored)", exempt.TaxAmount)
	}
	if err != nil {
		t.Fatalf("Create explicit: %v", err)
	}
	if explicit.TaxAmount != 20 {
		t.Fatalf("tax_amount = %v, want 20 (explicit rate wins)", explicit.TaxAmount)
	}
}

func f64(v float64) *float64 { return &v }

func TestOrderCreateValidation(t *testing.T) {
	_, svc, _ := newOrderEnv(t)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateRequest{ServiceTypeID: uuid.New()}, audit.Meta{UserID: uuid.New()}); err == nil {
		t.Fatal("missing customer_id must fail")
	}
	if _, err := svc.Create(ctx, CreateRequest{CustomerID: uuid.New()}, audit.Meta{UserID: uuid.New()}); err == nil {
		t.Fatal("missing service_type_id must fail")
	}
}

func TestPipelineMapExposed(t *testing.T) {
	if !IsValidPipeline(models.OrderCreated, models.OrderProcessing) {
		t.Fatal("created -> processing must be valid")
	}
	if IsValidPipeline(models.OrderCreated, models.OrderReady) {
		t.Fatal("created -> ready must be invalid")
	}
	if got := NormalizeStatus("  Ready "); got != "ready" {
		t.Fatalf("NormalizeStatus = %q", got)
	}
}

func errCode(err error) int {
	if e := errors.As(err); e != nil {
		return e.Code
	}
	return 0
}

func TestOrderUpdateAndDelete(t *testing.T) {
	db, svc, _ := newOrderEnv(t)
	ctx := context.Background()
	row := seedOrder(t, db, models.OrderCreated)
	id := orderID(t, row)

	if _, err := svc.Update(ctx, UpdateRequest{ID: uuid.New()}, audit.Meta{UserID: uuid.New()}); errCode(err) != 404 {
		t.Fatalf("update unknown: want 404, got %v", err)
	}
	note := "rush order"
	priority := "high"
	updated, err := svc.Update(ctx, UpdateRequest{ID: id, Note: &note, Priority: &priority}, audit.Meta{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Note != "rush order" || updated.Priority != "high" {
		t.Fatalf("update not applied: %+v", updated)
	}

	if err := svc.Delete(ctx, IDRequest{ID: uuid.New()}, audit.Meta{UserID: uuid.New()}); errCode(err) != 404 {
		t.Fatalf("delete unknown: want 404, got %v", err)
	}
	if err := svc.Delete(ctx, IDRequest{ID: id}, audit.Meta{UserID: uuid.New()}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var count int64
	db.Model(&models.Order{}).Where("id = ?", id).Count(&count)
	if count != 0 {
		t.Fatal("order not deleted")
	}
}

func TestOrderListAndSummary(t *testing.T) {
	db, svc, _ := newOrderEnv(t)
	ctx := context.Background()
	for i, st := range []string{models.OrderCreated, models.OrderReady, models.OrderDelivered} {
		row := seedOrder(t, db, st)
		row.GrandTotal = float64(100 * (i + 1))
		if err := db.Model(&row).Update("grand_total", row.GrandTotal).Error; err != nil {
			t.Fatalf("set grand total: %v", err)
		}
	}

	_, pg, summary, err := svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{Limit: 2}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pg.Total != 3 {
		t.Fatalf("total = %d, want 3", pg.Total)
	}
	if summary.Total != 3 || summary.Revenue != 600 || summary.AvgOrderValue != 200 {
		t.Fatalf("summary mismatch: %+v", summary)
	}
	if summary.ByStatus["created"] != 1 || summary.ByStatus["ready"] != 1 || summary.ByStatus["delivered"] != 1 {
		t.Fatalf("by_status mismatch: %+v", summary.ByStatus)
	}
}

func TestOrderCreateWithDiscount(t *testing.T) {
	_, svc, _ := newOrderEnv(t)
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{
		CustomerID: uuid.New(), ServiceTypeID: uuid.New(),
		TotalAmount: 100, TaxRate: f64(12), DiscountAmount: 10,
	}, audit.Meta{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// grand_total = total + tax - discount = 100 + 12 - 10 = 102.
	if view.GrandTotal != 102 {
		t.Fatalf("grand_total = %v, want 102 (total + tax - discount)", view.GrandTotal)
	}
}
