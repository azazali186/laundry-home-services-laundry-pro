package booking

import (
	"context"
	"strings"
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

func newBookingEnv(t *testing.T) (*gorm.DB, *Service, *bus.Bus) {
	return newBookingEnvWithShop(t, stubShop{currency: "INR"})
}

func newBookingEnvWithShop(t *testing.T, shop stubShop) (*gorm.DB, *Service, *bus.Bus) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Booking{}, &models.OutboxEvent{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	b := bus.New()
	svc := NewService(db, outbox.NewRepo(db, 5), b, audit.NewWriter(db), shop)
	return db, svc, b
}

// stubShop is a fixed shop-configuration provider for hermetic tests.
type stubShop struct{ currency, prefix string }

func (s stubShop) Defaults(context.Context) (shopconfig.Defaults, error) {
	return shopconfig.Defaults{Currency: s.currency, InvoicePrefix: s.prefix}, nil
}

var testMeta = audit.Meta{UserID: uuid.New(), Method: "POST", Path: "/api/v1/bookings"}

func TestBookingCreateEmitsOutboxAndBus(t *testing.T) {
	db, svc, b := newBookingEnv(t)
	ctx := context.Background()

	sub := b.Subscribe("booking_created")
	view, err := svc.Create(ctx, CreateRequest{
		CustomerID: uuid.New(), ServiceTypeID: uuid.New(),
		PickupAddress: "12 Test St", EstimatedAmount: 150, Currency: "INR",
	}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if view.Status != models.BookingPending || view.BookingNo == "" {
		t.Fatalf("unexpected view: %+v", view)
	}
	if view.Currency != "INR" || view.Source != "web" {
		t.Fatalf("defaults not applied: %+v", view)
	}

	// Outbox: a pending booking_created event must exist (drives the saga).
	var ev models.OutboxEvent
	if err := db.First(&ev, "event_type = ?", "booking_created").Error; err != nil {
		t.Fatalf("outbox event missing: %v", err)
	}
	if ev.Status != models.OutboxPending || ev.AggregateID != view.ID {
		t.Fatalf("unexpected outbox row: %+v", ev)
	}

	// Bus: the in-process event must have been published.
	select {
	case got := <-sub:
		if got.Type != "booking_created" || got.AggregateID != view.ID {
			t.Fatalf("bus event mismatch: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("bus event never published")
	}

	// Audit: a create audit entry must exist with no sensitive data.
	var auditRow models.AuditLog
	if err := db.First(&auditRow).Error; err != nil {
		t.Fatalf("audit entry missing: %v", err)
	}
	if auditRow.Action != "create" || len(auditRow.After) == 0 {
		t.Fatalf("unexpected audit row: %+v", auditRow)
	}
}

func TestBookingCreateValidation(t *testing.T) {
	_, svc, _ := newBookingEnv(t)
	ctx := context.Background()

	if _, err := svc.Create(ctx, CreateRequest{ServiceTypeID: uuid.New(), PickupAddress: "x"}, testMeta); err == nil {
		t.Fatal("missing customer_id must fail")
	}
	_, err := svc.Create(ctx, CreateRequest{CustomerID: uuid.New(), ServiceTypeID: uuid.New()}, testMeta)
	if err == nil {
		t.Fatal("missing pickup_address must fail")
	}
	appErr := errors.As(err)
	if appErr == nil || appErr.Code != 400 {
		t.Fatalf("expected 400, got %v", err)
	}
}

func TestBookingCancelEmitsEvent(t *testing.T) {
	db, svc, _ := newBookingEnv(t)
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{
		CustomerID: uuid.New(), ServiceTypeID: uuid.New(), PickupAddress: "addr",
	}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)
	if _, err := svc.Cancel(ctx, CancelRequest{ID: id, Reason: "customer request"}, testMeta); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	var ev models.OutboxEvent
	if err := db.First(&ev, "event_type = ?", "booking_cancelled").Error; err != nil {
		t.Fatalf("cancelled event missing: %v", err)
	}
	if !strings.Contains(string(ev.Payload), "customer request") {
		t.Fatalf("reason missing from payload: %s", ev.Payload)
	}
}

func TestBookingCreateAppliesShopCurrencyDefault(t *testing.T) {
	_, svc, _ := newBookingEnvWithShop(t, stubShop{currency: "USD"})
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{
		CustomerID: uuid.New(), ServiceTypeID: uuid.New(),
		PickupAddress: "addr", EstimatedAmount: 100,
	}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if view.Currency != "USD" {
		t.Fatalf("currency = %s, want shop default USD", view.Currency)
	}
}

func TestBookingUpdatePendingOnly(t *testing.T) {
	_, svc, _ := newBookingEnv(t)
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{
		CustomerID: uuid.New(), ServiceTypeID: uuid.New(), PickupAddress: "addr",
	}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)
	// Confirmed bookings can no longer be edited.
	if _, err := svc.Confirm(ctx, id, testMeta); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	newAddr := "changed"
	if _, err := svc.Update(ctx, UpdateRequest{ID: id, PickupAddress: &newAddr}, testMeta); err == nil {
		t.Fatal("update on a confirmed booking must fail")
	}
}

func errCode(err error) int {
	if e := errors.As(err); e != nil {
		return e.Code
	}
	return 0
}

func TestBookingConfirmRules(t *testing.T) {
	_, svc, _ := newBookingEnv(t)
	ctx := context.Background()
	view, err := svc.Create(ctx, CreateRequest{
		CustomerID: uuid.New(), ServiceTypeID: uuid.New(), PickupAddress: "addr",
	}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)

	if _, err := svc.Confirm(ctx, uuid.New(), testMeta); errCode(err) != 404 {
		t.Fatalf("confirm unknown: want 404, got %v", err)
	}
	confirmed, err := svc.Confirm(ctx, id, testMeta)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if confirmed.Status != models.BookingConfirmed {
		t.Fatalf("status = %s, want confirmed", confirmed.Status)
	}
	// Double confirm is rejected.
	if _, err := svc.Confirm(ctx, id, testMeta); errCode(err) != 409 {
		t.Fatalf("double confirm: want 409, got %v", err)
	}
}

func TestBookingCancelRules(t *testing.T) {
	_, svc, _ := newBookingEnv(t)
	ctx := context.Background()
	view, err := svc.Create(ctx, CreateRequest{
		CustomerID: uuid.New(), ServiceTypeID: uuid.New(), PickupAddress: "addr",
	}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)

	if _, err := svc.Cancel(ctx, CancelRequest{ID: uuid.New()}, testMeta); errCode(err) != 404 {
		t.Fatalf("cancel unknown: want 404, got %v", err)
	}
	if _, err := svc.Cancel(ctx, CancelRequest{ID: id}, testMeta); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	// Cancelling twice is rejected.
	if _, err := svc.Cancel(ctx, CancelRequest{ID: id}, testMeta); errCode(err) != 409 {
		t.Fatalf("double cancel: want 409, got %v", err)
	}
}

func TestBookingUpdateHappyPath(t *testing.T) {
	_, svc, _ := newBookingEnv(t)
	ctx := context.Background()
	view, err := svc.Create(ctx, CreateRequest{
		CustomerID: uuid.New(), ServiceTypeID: uuid.New(), PickupAddress: "addr",
	}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)

	if _, err := svc.Update(ctx, UpdateRequest{ID: uuid.New()}, testMeta); errCode(err) != 404 {
		t.Fatalf("update unknown: want 404, got %v", err)
	}
	newAddr := "2 Updated St"
	amount := 250.0
	updated, err := svc.Update(ctx, UpdateRequest{ID: id, PickupAddress: &newAddr, EstimatedAmount: &amount}, testMeta)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.PickupAddress != "2 Updated St" || updated.EstimatedAmount != 250 {
		t.Fatalf("update not applied: %+v", updated)
	}
}

func TestBookingDeleteAndList(t *testing.T) {
	db, svc, _ := newBookingEnv(t)
	ctx := context.Background()
	var ids []uuid.UUID
	for _, amt := range []float64{100, 200, 300} {
		v, err := svc.Create(ctx, CreateRequest{
			CustomerID: uuid.New(), ServiceTypeID: uuid.New(),
			PickupAddress: "addr", EstimatedAmount: amt,
		}, testMeta)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		id, _ := uuid.Parse(v.ID)
		ids = append(ids, id)
	}
	// One confirmed, one cancelled -> summary buckets.
	if _, err := svc.Confirm(ctx, ids[0], testMeta); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if _, err := svc.Cancel(ctx, CancelRequest{ID: ids[1]}, testMeta); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	_, pg, summary, err := svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{Limit: 2}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pg.Total != 3 {
		t.Fatalf("total = %d, want 3", pg.Total)
	}
	if summary.Total != 3 || summary.Pending != 1 || summary.Confirmed != 1 || summary.Cancelled != 1 {
		t.Fatalf("summary mismatch: %+v", summary)
	}
	if summary.EstimatedValue != 600 {
		t.Fatalf("estimated_value = %v, want 600", summary.EstimatedValue)
	}
	if summary.ByStatus["pending"] != 1 || summary.ByStatus["confirmed"] != 1 || summary.ByStatus["cancelled"] != 1 {
		t.Fatalf("by_status mismatch: %+v", summary.ByStatus)
	}

	// Delete unknown 404s; delete known removes the row.
	if err := svc.Delete(ctx, DeleteRequest{ID: uuid.New()}, testMeta); errCode(err) != 404 {
		t.Fatalf("delete unknown: want 404, got %v", err)
	}
	if err := svc.Delete(ctx, DeleteRequest{ID: ids[2]}, testMeta); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var count int64
	db.Model(&models.Booking{}).Where("id = ?", ids[2]).Count(&count)
	if count != 0 {
		t.Fatal("booking not deleted")
	}
}
