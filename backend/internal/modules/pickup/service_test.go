package pickup

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

func newEnv(t *testing.T) (*gorm.DB, *Service, *bus.Bus) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Pickup{}, &models.OutboxEvent{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	b := bus.New()
	return db, NewService(db, outbox.NewRepo(db, 5), b, audit.NewWriter(db)), b
}

var testMeta = audit.Meta{UserID: uuid.New(), Method: "POST", Path: "/api/v1/pickups"}

func TestPickupCreateDefaultsAndValidation(t *testing.T) {
	_, svc, _ := newEnv(t)
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{BookingID: uuid.New(), Address: "1 St"}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if view.Status != models.PickupScheduled || view.ScheduledAt.IsZero() {
		t.Fatalf("defaults not applied: %+v", view)
	}
	if _, err := svc.Create(ctx, CreateRequest{Address: "x"}, testMeta); err == nil {
		t.Fatal("missing booking_id must fail")
	}
}

func TestPickupCompleteEmitsSagaEvent(t *testing.T) {
	db, svc, b := newEnv(t)
	ctx := context.Background()
	sub := b.Subscribe("pickup_completed")

	view, err := svc.Create(ctx, CreateRequest{BookingID: uuid.New(), Address: "1 St"}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)
	done, err := svc.Complete(ctx, IDRequest{ID: id}, testMeta)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if done.Status != models.PickupCompleted || done.CompletedAt == nil {
		t.Fatalf("pickup not completed: %+v", done)
	}

	var ev models.OutboxEvent
	if err := db.First(&ev, "event_type = ?", "pickup_completed").Error; err != nil {
		t.Fatalf("outbox event missing: %v", err)
	}
	select {
	case got := <-sub:
		if got.Type != "pickup_completed" || got.AggregateID != view.ID {
			t.Fatalf("bus event mismatch: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("bus event never published")
	}

	// Double-complete is rejected.
	if _, err := svc.Complete(ctx, IDRequest{ID: id}, testMeta); err == nil {
		t.Fatal("second complete must fail")
	}
	// Unknown pickup 404s.
	if _, err := svc.Complete(ctx, IDRequest{ID: uuid.New()}, testMeta); err == nil {
		t.Fatal("unknown pickup must fail")
	}
}

func errCode(err error) int {
	if e := errors.As(err); e != nil {
		return e.Code
	}
	return 0
}

func TestPickupUpdateAndDelete(t *testing.T) {
	db, svc, _ := newEnv(t)
	ctx := context.Background()
	view, err := svc.Create(ctx, CreateRequest{BookingID: uuid.New(), Address: "1 St"}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)

	if _, err := svc.Update(ctx, UpdateRequest{ID: uuid.New()}, testMeta); errCode(err) != 404 {
		t.Fatalf("update unknown: want 404, got %v", err)
	}
	addr := "2 St"
	updated, err := svc.Update(ctx, UpdateRequest{ID: id, Address: &addr}, testMeta)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Address != "2 St" {
		t.Fatalf("address not updated: %+v", updated)
	}

	// Completed pickups cannot be edited.
	if _, err := svc.Complete(ctx, IDRequest{ID: id}, testMeta); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if _, err := svc.Update(ctx, UpdateRequest{ID: id, Address: &addr}, testMeta); errCode(err) != 409 {
		t.Fatalf("edit completed pickup: want 409, got %v", err)
	}

	// Delete then verify gone.
	if err := svc.Delete(ctx, IDRequest{ID: id}, testMeta); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var count int64
	db.Model(&models.Pickup{}).Where("id = ?", id).Count(&count)
	if count != 0 {
		t.Fatal("pickup not deleted")
	}
}

func TestPickupListAndSummary(t *testing.T) {
	db, svc, _ := newEnv(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := svc.Create(ctx, CreateRequest{BookingID: uuid.New(), Address: "x"}, testMeta); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	var first models.Pickup
	db.First(&first)
	if _, err := svc.Complete(ctx, IDRequest{ID: first.ID}, testMeta); err != nil {
		t.Fatalf("complete one: %v", err)
	}

	_, pg, summary, err := svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{Limit: 2}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pg.Total != 3 {
		t.Fatalf("total = %d, want 3", pg.Total)
	}
	if summary.Total != 3 || summary.Completed != 1 || summary.Active != 2 {
		t.Fatalf("summary mismatch: %+v", summary)
	}
}
