package delivery

import (
	"context"
	"testing"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/bus"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/outbox"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newDeliveryEnv(t *testing.T) (*gorm.DB, *Service, *bus.Bus) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Delivery{}, &models.OutboxEvent{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	b := bus.New()
	svc := NewService(db, outbox.NewRepo(db, 5), b, audit.NewWriter(db))
	return db, svc, b
}

func seedDelivery(t *testing.T, db *gorm.DB) models.Delivery {
	t.Helper()
	row := models.Delivery{
		OrderID:     uuid.New(),
		ScheduledAt: time.Now().Add(time.Hour),
		Address:     "1 Delivery St",
		Status:      models.DeliveryScheduled,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed delivery: %v", err)
	}
	return row
}

func deliveryID(t *testing.T, row models.Delivery) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(row.ID.String())
	if err != nil {
		t.Fatalf("parse id: %v", err)
	}
	return id
}

func TestDeliveryCreateDefaultsAndEvent(t *testing.T) {
	db, svc, b := newDeliveryEnv(t)
	ctx := context.Background()
	sub := b.Subscribe("delivery_scheduled")

	// Zero scheduled_at must default to now.
	view, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Address: "x"}, audit.Meta{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if view.Status != models.DeliveryScheduled || view.ScheduledAt.IsZero() {
		t.Fatalf("unexpected view: %+v", view)
	}
	var ev models.OutboxEvent
	if err := db.First(&ev, "event_type = ?", "delivery_scheduled").Error; err != nil {
		t.Fatalf("delivery_scheduled event missing: %v", err)
	}
	select {
	case got := <-sub:
		if got.Type != "delivery_scheduled" || got.AggregateID != view.ID {
			t.Fatalf("bus event mismatch: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("bus event never published")
	}
}

func TestDeliveryCompleteEmitsSagaStep(t *testing.T) {
	db, svc, b := newDeliveryEnv(t)
	ctx := context.Background()
	row := seedDelivery(t, db)
	id := deliveryID(t, row)
	sub := b.Subscribe("delivery_delivered")

	view, err := svc.Complete(ctx, IDRequest{ID: id}, audit.Meta{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if view.Status != models.DeliveryDelivered || view.DeliveredAt == nil {
		t.Fatalf("unexpected view: %+v", view)
	}
	// The delivery_delivered event drives the saga's final step.
	var ev models.OutboxEvent
	if err := db.First(&ev, "event_type = ?", "delivery_delivered").Error; err != nil {
		t.Fatalf("delivery_delivered event missing: %v", err)
	}
	if len(ev.Payload) == 0 {
		t.Fatal("event payload must carry delivery_id/order_id")
	}
	select {
	case <-sub:
	default:
		t.Fatal("bus event not published")
	}
}

func TestDeliveryCompleteIdempotency(t *testing.T) {
	db, svc, _ := newDeliveryEnv(t)
	ctx := context.Background()
	row := seedDelivery(t, db)
	id := deliveryID(t, row)

	if _, err := svc.Complete(ctx, IDRequest{ID: id}, audit.Meta{UserID: uuid.New()}); err != nil {
		t.Fatalf("first Complete: %v", err)
	}
	if _, err := svc.Complete(ctx, IDRequest{ID: id}, audit.Meta{UserID: uuid.New()}); err == nil {
		t.Fatal("second Complete must be rejected")
	}
	// Only one saga event should ever exist.
	var count int64
	db.Model(&models.OutboxEvent{}).Where("event_type = ?", "delivery_delivered").Count(&count)
	if count != 1 {
		t.Fatalf("delivery_delivered events = %d, want 1", count)
	}
}

func TestDeliveryEditAfterDeliveredRejected(t *testing.T) {
	db, svc, _ := newDeliveryEnv(t)
	ctx := context.Background()
	row := seedDelivery(t, db)
	id := deliveryID(t, row)

	if _, err := svc.Complete(ctx, IDRequest{ID: id}, audit.Meta{UserID: uuid.New()}); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	addr := "changed"
	if _, err := svc.Update(ctx, UpdateRequest{ID: id, Address: &addr}, audit.Meta{UserID: uuid.New()}); err == nil {
		t.Fatal("editing a delivered delivery must fail")
	}
}

func TestDeliveryCreateValidation(t *testing.T) {
	_, svc, _ := newDeliveryEnv(t)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateRequest{}, audit.Meta{UserID: uuid.New()}); err == nil {
		t.Fatal("missing order_id must fail")
	}
	if _, err := svc.Complete(ctx, IDRequest{ID: uuid.New()}, audit.Meta{UserID: uuid.New()}); err == nil {
		t.Fatal("unknown delivery must fail")
	}
}

func TestDeliveryUpdateAndDelete(t *testing.T) {
	db, svc, _ := newDeliveryEnv(t)
	ctx := context.Background()
	row := seedDelivery(t, db)
	id := deliveryID(t, row)

	if _, err := svc.Update(ctx, UpdateRequest{ID: uuid.New()}, audit.Meta{UserID: uuid.New()}); errCode(err) != 404 {
		t.Fatalf("update unknown: want 404, got %v", err)
	}
	addr := "2 Delivery St"
	updated, err := svc.Update(ctx, UpdateRequest{ID: id, Address: &addr}, audit.Meta{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Address != "2 Delivery St" {
		t.Fatalf("address not updated: %+v", updated)
	}

	if err := svc.Delete(ctx, IDRequest{ID: id}, audit.Meta{UserID: uuid.New()}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var count int64
	db.Model(&models.Delivery{}).Where("id = ?", id).Count(&count)
	if count != 0 {
		t.Fatal("delivery not deleted")
	}
}

func errCode(err error) int {
	if e := errors.As(err); e != nil {
		return e.Code
	}
	return 0
}

func TestDeliveryListAndSummary(t *testing.T) {
	db, svc, _ := newDeliveryEnv(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if err := db.Create(&models.Delivery{
			OrderID: uuid.New(), ScheduledAt: time.Now().Add(time.Hour),
			Status: models.DeliveryScheduled,
		}).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	var first models.Delivery
	db.First(&first)
	if _, err := svc.Complete(ctx, IDRequest{ID: first.ID}, audit.Meta{UserID: uuid.New()}); err != nil {
		t.Fatalf("complete one: %v", err)
	}

	_, pg, summary, err := svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{Limit: 2}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pg.Total != 3 {
		t.Fatalf("total = %d, want 3", pg.Total)
	}
	if summary.Total != 3 || summary.Delivered != 1 || summary.Scheduled != 2 {
		t.Fatalf("summary mismatch: %+v", summary)
	}
}
