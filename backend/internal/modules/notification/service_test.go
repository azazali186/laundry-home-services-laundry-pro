package notification

import (
	"context"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
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
	if err := db.AutoMigrate(&models.Notification{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db, NewService(db, audit.NewWriter(db))
}

var testMeta = audit.Meta{UserID: uuid.New(), Method: "POST", Path: "/api/v1/notifications"}

func TestNotificationCreateDefaultsAndValidation(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{UserID: uuid.New(), Title: "  Order ready  ", Body: "Pick it up"}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if view.Title != "Order ready" || view.Status != models.NotificationPending {
		t.Fatalf("unexpected view: %+v", view)
	}
	if view.Type != "push" || view.Channel != "push" {
		t.Fatalf("defaults not applied: %+v", view)
	}
	if _, err := svc.Create(ctx, CreateRequest{Title: "x"}, testMeta); err == nil {
		t.Fatal("missing user_id must fail")
	}
	if _, err := svc.Create(ctx, CreateRequest{UserID: uuid.New()}, testMeta); err == nil {
		t.Fatal("missing title must fail")
	}
}

func TestNotificationMarkRead(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()
	view, err := svc.Create(ctx, CreateRequest{UserID: uuid.New(), Title: "Hi"}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)

	read, err := svc.MarkRead(ctx, MarkReadRequest{ID: id})
	if err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if read.ReadAt == nil {
		t.Fatal("read_at not set")
	}
	// Unknown id 404s.
	if _, err := svc.MarkRead(ctx, MarkReadRequest{ID: uuid.New()}); err == nil {
		t.Fatal("unknown notification must fail")
	}
}

func TestNotificationListAndSummary(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()
	first, err := svc.Create(ctx, CreateRequest{UserID: uuid.New(), Title: "One"}, testMeta)
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}
	if _, err := svc.Create(ctx, CreateRequest{UserID: uuid.New(), Title: "Two"}, testMeta); err != nil {
		t.Fatalf("Create 2: %v", err)
	}
	id, _ := uuid.Parse(first.ID)
	if _, err := svc.MarkRead(ctx, MarkReadRequest{ID: id}); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}

	_, pg, summary, err := svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{Limit: 10}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pg.Total != 2 {
		t.Fatalf("total = %d, want 2", pg.Total)
	}
	// MarkRead sets read_at but leaves status pending, so both stay pending
	// while only one is unread.
	if summary.Total != 2 || summary.Pending != 2 || summary.Unread != 1 {
		t.Fatalf("summary mismatch: %+v", summary)
	}
}
