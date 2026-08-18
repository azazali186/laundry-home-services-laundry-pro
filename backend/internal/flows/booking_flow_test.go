package flows

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/bus"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/modkit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/outbox"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/saga"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/shopconfig"
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newFlowEnv(t *testing.T) (*gorm.DB, *Flow, *saga.Registry) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Booking{}, &models.Pickup{}, &models.Order{}, &models.Delivery{},
		&models.Invoice{}, &models.OutboxEvent{}, &models.SagaState{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	c := cache.NewFromClient(client, "laundry-pro")

	repo := outbox.NewRepo(db, 5)
	flow := New(db, repo, saga.NewStore(db, c), bus.New(), nil)
	registry := saga.NewRegistry()
	flow.Register(registry)
	return db, flow, registry
}

// seedBooking creates a pending booking the saga will advance.
func seedBooking(t *testing.T, db *gorm.DB) models.Booking {
	t.Helper()
	row := models.Booking{
		CustomerID: uuid.New(), ServiceTypeID: uuid.New(),
		PickupAddress: "42 Flow St", Status: models.BookingPending,
		EstimatedAmount: 120, Currency: "INR",
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed booking: %v", err)
	}
	return row
}

func bookingPayload(b models.Booking) json.RawMessage {
	raw, _ := json.Marshal(bookingCreatedPayload{
		BookingID: b.ID.String(), BookingNo: b.BookingNo,
		CustomerID: b.CustomerID.String(), ServiceTypeID: b.ServiceTypeID.String(),
		PickupAddress: b.PickupAddress,
	})
	return raw
}

func TestBookingCreatedDrivesSaga(t *testing.T) {
	db, _, registry := newFlowEnv(t)
	ctx := context.Background()
	bk := seedBooking(t, db)

	ev := saga.Event{AggregateType: "booking", AggregateID: bk.ID.String(), EventType: "booking_created", Payload: bookingPayload(bk)}
	if err := registry.Dispatch(ctx, ev); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	// The saga must have scheduled a pickup for the booking.
	var pickup models.Pickup
	if err := db.First(&pickup, "booking_id = ?", bk.ID).Error; err != nil {
		t.Fatalf("pickup not created: %v", err)
	}
	if pickup.Status != models.PickupScheduled || pickup.Address != "42 Flow St" {
		t.Fatalf("unexpected pickup: %+v", pickup)
	}

	// Booking advanced to confirmed.
	var after models.Booking
	if err := db.First(&after, "id = ?", bk.ID).Error; err != nil {
		t.Fatalf("reload booking: %v", err)
	}
	if after.Status != models.BookingConfirmed {
		t.Fatalf("booking status = %s, want confirmed", after.Status)
	}

	// Saga state recorded the pickup_scheduled step.
	var sagaRow models.SagaState
	if err := db.First(&sagaRow, "aggregate_id = ?", bk.ID.String()).Error; err != nil {
		t.Fatalf("saga state missing: %v", err)
	}
	if sagaRow.Status != models.SagaRunning || sagaRow.Step != "pickup_scheduled" {
		t.Fatalf("unexpected saga state: status=%s step=%s", sagaRow.Status, sagaRow.Step)
	}

	// A pickup_scheduled event is queued in the outbox for relay.
	var queued models.OutboxEvent
	if err := db.First(&queued, "event_type = ?", "pickup_scheduled").Error; err != nil {
		t.Fatalf("pickup_scheduled outbox event missing: %v", err)
	}
	if queued.Status != models.OutboxPending {
		t.Fatalf("outbox status = %s, want pending", queued.Status)
	}
}

// TestInvalidBookingIDRejected verifies the parseUUID fix: a malformed id in
// the event payload must fail the handler instead of writing an orphaned
// pickup with a nil booking_id.
func TestInvalidBookingIDRejected(t *testing.T) {
	db, flow, _ := newFlowEnv(t)
	ctx := context.Background()

	ev := saga.Event{
		AggregateType: "booking", AggregateID: "not-a-uuid", EventType: "booking_created",
		Payload: json.RawMessage(`{"booking_id":"not-a-uuid","pickup_address":"x"}`),
	}
	if err := flow.onBookingCreated(ctx, ev); err == nil {
		t.Fatal("handler must reject a malformed booking id")
	}
	var count int64
	db.Model(&models.Pickup{}).Count(&count)
	if count != 0 {
		t.Fatalf("orphaned pickups created: %d", count)
	}
	var sagas int64
	db.Model(&models.SagaState{}).Count(&sagas)
	if sagas != 0 {
		t.Fatalf("saga started despite invalid id: %d", sagas)
	}
}

// TestFullSagaLifecycle drives the saga through pickup -> order -> delivery
// and verifies the saga state advances and finally completes.
func TestFullSagaLifecycle(t *testing.T) {
	db, flow, registry := newFlowEnv(t)
	ctx := context.Background()
	bk := seedBooking(t, db)

	// Step 1: booking created -> saga started, pickup scheduled.
	ev := saga.Event{AggregateType: "booking", AggregateID: bk.ID.String(), EventType: "booking_created", Payload: bookingPayload(bk)}
	if err := registry.Dispatch(ctx, ev); err != nil {
		t.Fatalf("Dispatch booking_created: %v", err)
	}

	// Step 2: pickup completed -> order created, saga advances.
	var pickup models.Pickup
	if err := db.First(&pickup, "booking_id = ?", bk.ID).Error; err != nil {
		t.Fatalf("pickup missing: %v", err)
	}
	if err := flow.onPickupCompleted(ctx, saga.Event{
		AggregateType: "pickup", AggregateID: pickup.ID.String(), EventType: "pickup_completed",
		Payload: json.RawMessage(`{"pickup_id":"` + pickup.ID.String() + `","booking_id":"` + bk.ID.String() + `"}`),
	}); err != nil {
		t.Fatalf("onPickupCompleted: %v", err)
	}
	var order models.Order
	if err := db.First(&order, "booking_id = ?", bk.ID).Error; err != nil {
		t.Fatalf("order missing: %v", err)
	}
	if order.OrderNo == "" {
		t.Fatal("order created without an order_no (unique constraint would collide)")
	}
	assertSagaStep(t, db, bk.ID.String(), "order_created", models.SagaRunning)

	// Step 3: order ready -> delivery scheduled, saga advances.
	if err := flow.onOrderStatusChanged(ctx, saga.Event{
		AggregateType: "order", AggregateID: order.ID.String(), EventType: "order_status_changed",
		Payload: json.RawMessage(`{"order_id":"` + order.ID.String() + `","to":"ready"}`),
	}); err != nil {
		t.Fatalf("onOrderStatusChanged: %v", err)
	}
	var delivery models.Delivery
	if err := db.First(&delivery, "order_id = ?", order.ID).Error; err != nil {
		t.Fatalf("delivery missing: %v", err)
	}
	assertSagaStep(t, db, bk.ID.String(), "delivery_scheduled", models.SagaRunning)

	// Step 4: delivery delivered -> invoice issued, saga completes.
	if err := flow.onDeliveryDelivered(ctx, saga.Event{
		AggregateType: "delivery", AggregateID: delivery.ID.String(), EventType: "delivery_delivered",
		Payload: json.RawMessage(`{"delivery_id":"` + delivery.ID.String() + `","order_id":"` + order.ID.String() + `"}`),
	}); err != nil {
		t.Fatalf("onDeliveryDelivered: %v", err)
	}
	var invoice models.Invoice
	if err := db.First(&invoice, "order_id = ?", order.ID).Error; err != nil {
		t.Fatalf("invoice missing: %v", err)
	}
	var sagaRow models.SagaState
	if err := db.First(&sagaRow, "aggregate_id = ?", bk.ID.String()).Error; err != nil {
		t.Fatalf("saga state missing: %v", err)
	}
	if sagaRow.Status != models.SagaCompleted || sagaRow.Step != "completed" {
		t.Fatalf("saga not completed: status=%s step=%s", sagaRow.Status, sagaRow.Step)
	}
}

func assertSagaStep(t *testing.T, db *gorm.DB, bookingID, step, status string) {
	t.Helper()
	var sagaRow models.SagaState
	if err := db.First(&sagaRow, "aggregate_id = ?", bookingID).Error; err != nil {
		t.Fatalf("saga state missing: %v", err)
	}
	if sagaRow.Step != step || sagaRow.Status != status {
		t.Fatalf("saga state = step:%s status:%s, want step:%s status:%s", sagaRow.Step, sagaRow.Status, step, status)
	}
}

// TestDeliveryDeliveredUsesShopInvoicePrefix proves saga-issued invoices
// inherit the shop-configured invoice prefix (not a hardcoded INV).
func TestDeliveryDeliveredUsesShopInvoicePrefix(t *testing.T) {
	db, flow, _ := newFlowEnv(t)
	ctx := context.Background()
	flow.shop = shopconfig.Static{InvoicePrefix: "BILL"}

	order := models.Order{
		Base:          models.Base{ID: uuid.New()},
		OrderNo:       modkit.NextNo("ORD"),
		CustomerID:    uuid.New(),
		ServiceTypeID: uuid.New(),
		Status:        models.OrderReady,
		TotalAmount:   100,
		TaxAmount:     7.5,
		GrandTotal:    107.5,
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("seed order: %v", err)
	}

	if err := flow.onDeliveryDelivered(ctx, saga.Event{
		AggregateType: "delivery", AggregateID: uuid.New().String(), EventType: "delivery_delivered",
		Payload: json.RawMessage(`{"delivery_id":"` + uuid.New().String() + `","order_id":"` + order.ID.String() + `"}`),
	}); err != nil {
		t.Fatalf("onDeliveryDelivered: %v", err)
	}

	var invoice models.Invoice
	if err := db.First(&invoice, "order_id = ?", order.ID).Error; err != nil {
		t.Fatalf("invoice missing: %v", err)
	}
	if !strings.HasPrefix(invoice.InvoiceNo, "BILL") {
		t.Fatalf("invoice_no = %q, want shop prefix BILL", invoice.InvoiceNo)
	}
}

// TestUnknownEventIsNoop verifies the relay treats unregistered event types
// as success (they may target other services).
func TestUnknownEventIsNoop(t *testing.T) {
	db, _, registry := newFlowEnv(t)
	ctx := context.Background()

	ev := saga.Event{AggregateType: "order", AggregateID: uuid.New().String(), EventType: "order_ready", Payload: json.RawMessage(`{}`)}
	if err := registry.Dispatch(ctx, ev); err != nil {
		t.Fatalf("unregistered event must not error: %v", err)
	}
	var count int64
	db.Model(&models.SagaState{}).Count(&count)
	if count != 0 {
		t.Fatalf("no saga should start for unregistered events, got %d", count)
	}
}
