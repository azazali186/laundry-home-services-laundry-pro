package invoice

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/bus"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/shopconfig"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newInvoiceEnv(t *testing.T, prefix string) (*gorm.DB, *Service, *bus.Bus) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Invoice{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	b := bus.New()
	svc := NewService(db, b, audit.NewWriter(db), stubShop{prefix: prefix})
	return db, svc, b
}

// stubShop is a fixed shop-configuration provider for hermetic tests.
type stubShop struct{ prefix string }

func (s stubShop) Defaults(context.Context) (shopconfig.Defaults, error) {
	return shopconfig.Defaults{InvoicePrefix: s.prefix}, nil
}

var testMeta = audit.Meta{UserID: uuid.New(), Method: "POST", Path: "/api/v1/invoices"}

func TestInvoiceCreateUsesSettingsPrefix(t *testing.T) {
	_, svc, _ := newInvoiceEnv(t, "SPK")
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Subtotal: 100, TaxAmount: 7, Discount: 5}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !strings.HasPrefix(view.InvoiceNo, "SPK") {
		t.Fatalf("invoice_no %q does not use the settings prefix", view.InvoiceNo)
	}
	if view.Total != 102 {
		t.Fatalf("total = %v, want 102 (100 + 7 - 5)", view.Total)
	}
	if view.Status != models.InvoiceDraft {
		t.Fatalf("status = %s, want draft", view.Status)
	}
}

func TestInvoiceCreateFallsBackToInvPrefix(t *testing.T) {
	_, svc, _ := newInvoiceEnv(t, "")
	view, err := svc.Create(context.Background(), CreateRequest{OrderID: uuid.New(), Subtotal: 10}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !strings.HasPrefix(view.InvoiceNo, "INV") {
		t.Fatalf("empty prefix must fall back to INV, got %q", view.InvoiceNo)
	}
}

func TestInvoiceCreateRequiresOrderID(t *testing.T) {
	_, svc, _ := newInvoiceEnv(t, "INV")
	if _, err := svc.Create(context.Background(), CreateRequest{Subtotal: 10}, testMeta); err == nil {
		t.Fatal("missing order_id must fail")
	}
}

func TestInvoiceMarkPaidEmitsEvent(t *testing.T) {
	db, svc, b := newInvoiceEnv(t, "INV")
	ctx := context.Background()
	sub := b.Subscribe("invoice_paid")

	view, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Subtotal: 100}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)
	paid, err := svc.MarkPaid(ctx, IDRequest{ID: id}, testMeta)
	if err != nil {
		t.Fatalf("MarkPaid: %v", err)
	}
	if paid.Status != models.InvoicePaid || paid.PaidAt == nil {
		t.Fatalf("invoice not marked paid: %+v", paid)
	}
	select {
	case got := <-sub:
		if got.Type != "invoice_paid" || got.AggregateID != view.ID {
			t.Fatalf("bus event mismatch: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("invoice_paid event never published")
	}

	// Double mark-paid must be idempotent (same view, no error).
	again, err := svc.MarkPaid(ctx, IDRequest{ID: id}, testMeta)
	if err != nil || again.Status != models.InvoicePaid {
		t.Fatalf("second MarkPaid: %v %+v", err, again)
	}
	var count int64
	db.Model(&models.Invoice{}).Count(&count)
	if count != 1 {
		t.Fatalf("rows = %d, want 1", count)
	}
}

func TestInvoiceUpdateRecomputesTotal(t *testing.T) {
	_, svc, _ := newInvoiceEnv(t, "INV")
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Subtotal: 100, TaxAmount: 10}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)
	newSub := 200.0
	updated, err := svc.Update(ctx, UpdateRequest{ID: id, Subtotal: &newSub}, testMeta)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Total != 210 {
		t.Fatalf("total after update = %v, want 210", updated.Total)
	}
}

func TestInvoiceDelete(t *testing.T) {
	db, svc, _ := newInvoiceEnv(t, "INV")
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Subtotal: 10}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)
	if err := svc.Delete(ctx, IDRequest{ID: id}, testMeta); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var count int64
	db.Model(&models.Invoice{}).Count(&count)
	if count != 0 {
		t.Fatalf("rows after delete = %d, want 0", count)
	}
}

func TestInvoiceListAndMoneySummary(t *testing.T) {
	_, svc, _ := newInvoiceEnv(t, "INV")
	ctx := context.Background()

	// draft 100, issued 200, paid 300.
	v1, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Subtotal: 100}, testMeta)
	if err != nil {
		t.Fatalf("Create draft: %v", err)
	}
	v2, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Subtotal: 200}, testMeta)
	if err != nil {
		t.Fatalf("Create issued: %v", err)
	}
	v3, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Subtotal: 300}, testMeta)
	if err != nil {
		t.Fatalf("Create paid: %v", err)
	}

	id2, _ := uuid.Parse(v2.ID)
	issued := models.InvoiceIssued
	if _, err := svc.Update(ctx, UpdateRequest{ID: id2, Status: &issued}, testMeta); err != nil {
		t.Fatalf("issue invoice: %v", err)
	}
	id3, _ := uuid.Parse(v3.ID)
	if _, err := svc.MarkPaid(ctx, IDRequest{ID: id3}, testMeta); err != nil {
		t.Fatalf("pay invoice: %v", err)
	}

	items, pg, summary, err := svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{Limit: 2}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pg.Total != 3 || len(items) != 2 {
		t.Fatalf("pagination total/len = %d/%d, want 3/2", pg.Total, len(items))
	}
	if summary.Total != 3 || summary.Draft != 1 || summary.Issued != 1 || summary.Paid != 1 || summary.Void != 0 {
		t.Fatalf("summary status counts wrong: %+v", summary)
	}
	if summary.Outstanding != 200 || summary.Collected != 300 {
		t.Fatalf("money summary wrong: outstanding=%v collected=%v, want 200/300", summary.Outstanding, summary.Collected)
	}
	if summary.ByStatus[models.InvoiceDraft] != 1 || summary.ByStatus[models.InvoiceIssued] != 1 || summary.ByStatus[models.InvoicePaid] != 1 {
		t.Fatalf("by_status wrong: %+v", summary.ByStatus)
	}
	if v1.ID == "" || v1.Total != 100 {
		t.Fatalf("draft view wrong: %+v", v1)
	}
}

func TestInvoiceUpdateStatusAndBilling(t *testing.T) {
	_, svc, _ := newInvoiceEnv(t, "INV")
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Subtotal: 50, BillingName: "Old Co"}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)
	name := "New Co"
	void := models.InvoiceVoid
	updated, err := svc.Update(ctx, UpdateRequest{ID: id, BillingName: &name, Status: &void}, testMeta)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.BillingName != "New Co" || updated.Status != models.InvoiceVoid {
		t.Fatalf("billing/status not updated: %+v", updated)
	}
}

func TestInvoiceUpdatePaidRejected(t *testing.T) {
	_, svc, _ := newInvoiceEnv(t, "INV")
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Subtotal: 100}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)
	if _, err := svc.MarkPaid(ctx, IDRequest{ID: id}, testMeta); err != nil {
		t.Fatalf("MarkPaid: %v", err)
	}

	sub := 500.0
	if _, err := svc.Update(ctx, UpdateRequest{ID: id, Subtotal: &sub}, testMeta); err == nil {
		t.Fatal("updating a paid invoice must fail")
	} else if e := errors.As(err); e == nil || e.Code != 409 {
		t.Fatalf("paid update error = %v, want 409", err)
	}
}

func TestInvoiceNotFound(t *testing.T) {
	_, svc, _ := newInvoiceEnv(t, "INV")
	ctx := context.Background()
	missing := uuid.New()

	if _, err := svc.Update(ctx, UpdateRequest{ID: missing}, testMeta); err == nil {
		t.Fatal("update of missing invoice must fail")
	} else if e := errors.As(err); e == nil || e.Code != 404 {
		t.Fatalf("update missing: %v", err)
	}
	if _, err := svc.MarkPaid(ctx, IDRequest{ID: missing}, testMeta); err == nil {
		t.Fatal("mark-paid of missing invoice must fail")
	} else if e := errors.As(err); e == nil || e.Code != 404 {
		t.Fatalf("mark-paid missing: %v", err)
	}
	if err := svc.Delete(ctx, IDRequest{ID: missing}, testMeta); err == nil {
		t.Fatal("delete of missing invoice must fail")
	} else if e := errors.As(err); e == nil || e.Code != 404 {
		t.Fatalf("delete missing: %v", err)
	}
}
