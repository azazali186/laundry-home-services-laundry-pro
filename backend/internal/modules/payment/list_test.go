package payment

import (
	"context"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/gateway"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/google/uuid"
)

// refundFailingGateway succeeds at charging but always fails refunds.
type refundFailingGateway struct{}

func (refundFailingGateway) Charge(_ context.Context, ref string, _ float64, _ string) (*gateway.ChargeResult, error) {
	return &gateway.ChargeResult{TransactionID: "TXN-" + ref, Status: "success"}, nil
}

func (refundFailingGateway) Refund(_ context.Context, _ string, _ float64) error {
	return gateway.ErrDeclined
}

func TestPaymentListAndMoneySummary(t *testing.T) {
	_, svc, _ := newPaymentEnv(t, &countingGateway{})
	ctx := context.Background()
	meta := audit.Meta{UserID: uuid.New()}

	// pending 500 (card), paid 300 (cash), refunded 200 (upi).
	if _, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Amount: 500, Method: "card", Currency: "INR"}, meta); err != nil {
		t.Fatalf("Create pending: %v", err)
	}
	v2, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Amount: 300, Method: "cash", Currency: "INR"}, meta)
	if err != nil {
		t.Fatalf("Create paid: %v", err)
	}
	v3, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Amount: 200, Method: "upi", Currency: "INR"}, meta)
	if err != nil {
		t.Fatalf("Create refunded: %v", err)
	}
	if _, err := svc.Capture(ctx, CaptureRequest{ID: mustParse(v2.ID)}, meta); err != nil {
		t.Fatalf("Capture paid: %v", err)
	}
	if _, err := svc.Capture(ctx, CaptureRequest{ID: mustParse(v3.ID)}, meta); err != nil {
		t.Fatalf("Capture refunded: %v", err)
	}
	if _, err := svc.Refund(ctx, RefundRequest{ID: mustParse(v3.ID), Amount: 200}, meta); err != nil {
		t.Fatalf("Refund: %v", err)
	}

	items, pg, summary, err := svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{Limit: 2}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pg.Total != 3 || len(items) != 2 {
		t.Fatalf("pagination total/len = %d/%d, want 3/2", pg.Total, len(items))
	}
	if summary.Total != 3 || summary.Pending != 1 || summary.Paid != 1 || summary.Failed != 0 || summary.Refunded != 1 {
		t.Fatalf("status counts wrong: %+v", summary)
	}
	if summary.Collected != 300 || summary.Outstanding != 500 || summary.RefundedSum != 200 {
		t.Fatalf("money summary wrong: collected=%v outstanding=%v refunded=%v, want 300/500/200",
			summary.Collected, summary.Outstanding, summary.RefundedSum)
	}
	if summary.ByMethod["card"] != 1 || summary.ByMethod["cash"] != 1 || summary.ByMethod["upi"] != 1 {
		t.Fatalf("by_method wrong: %+v", summary.ByMethod)
	}
}

func TestPaymentListFilterAndSearch(t *testing.T) {
	_, svc, _ := newPaymentEnv(t, &countingGateway{})
	ctx := context.Background()
	meta := audit.Meta{UserID: uuid.New()}

	for i := 0; i < 3; i++ {
		if _, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Amount: 100, Method: "cash", Currency: "INR"}, meta); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}
	v, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Amount: 400, Method: "card", Currency: "INR"}, meta)
	if err != nil {
		t.Fatalf("Create paid: %v", err)
	}
	if _, err := svc.Capture(ctx, CaptureRequest{ID: mustParse(v.ID)}, meta); err != nil {
		t.Fatalf("Capture: %v", err)
	}

	// Filter by status narrows both page and summary.
	items, pg, summary, err := svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{
		Filters: map[string]string{"status": models.PaymentPaid},
	}})
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if pg.Total != 1 || len(items) != 1 {
		t.Fatalf("filtered total/len = %d/%d, want 1/1", pg.Total, len(items))
	}
	if summary.Total != 1 || summary.Paid != 1 {
		t.Fatalf("filtered summary wrong: %+v", summary)
	}

	// Search across payment_no.
	_, pg, _, err = svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{Search: v.PaymentNo}})
	if err != nil {
		t.Fatalf("List search: %v", err)
	}
	if pg.Total != 1 {
		t.Fatalf("search total = %d, want 1", pg.Total)
	}
}

func TestPaymentNotFound(t *testing.T) {
	_, svc, _ := newPaymentEnv(t, &countingGateway{})
	ctx := context.Background()
	meta := audit.Meta{UserID: uuid.New()}
	missing := uuid.New()

	if _, err := svc.Update(ctx, UpdateRequest{ID: missing}, meta); err == nil {
		t.Fatal("update of missing payment must fail")
	} else if e := errors.As(err); e == nil || e.Code != 404 {
		t.Fatalf("update missing: %v", err)
	}
	if _, err := svc.Refund(ctx, RefundRequest{ID: missing, Amount: 10}, meta); err == nil {
		t.Fatal("refund of missing payment must fail")
	} else if e := errors.As(err); e == nil || e.Code != 404 {
		t.Fatalf("refund missing: %v", err)
	}
	if err := svc.Delete(ctx, IDRequest{ID: missing}, meta); err == nil {
		t.Fatal("delete of missing payment must fail")
	} else if e := errors.As(err); e == nil || e.Code != 404 {
		t.Fatalf("delete missing: %v", err)
	}
}

func TestPaymentRefundGatewayFailure(t *testing.T) {
	db, svc, _ := newPaymentEnv(t, &refundFailingGateway{})
	ctx := context.Background()
	meta := audit.Meta{UserID: uuid.New()}

	id := createPayment(t, svc)
	if _, err := svc.Capture(ctx, CaptureRequest{ID: mustParse(id)}, meta); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if _, err := svc.Refund(ctx, RefundRequest{ID: mustParse(id), Amount: 500}, meta); err == nil {
		t.Fatal("gateway refund failure must surface an error")
	} else if e := errors.As(err); e == nil || e.Code != 502 {
		t.Fatalf("refund failure code = %v, want 502", err)
	}
	// The payment must remain paid (not marked refunded) after the failure.
	var row models.Payment
	if err := db.First(&row, "id = ?", id).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if row.Status != models.PaymentPaid {
		t.Fatalf("status after failed refund = %s, want paid", row.Status)
	}
}
