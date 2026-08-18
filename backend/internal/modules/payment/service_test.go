package payment

import (
	"context"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/bus"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/crypto"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/gateway"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/shopconfig"
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newPaymentEnv(t *testing.T, gw gateway.Client) (*gorm.DB, *Service, *crypto.Cipher) {
	return newPaymentEnvWithShop(t, gw, stubShop{currency: "INR"})
}

func newPaymentEnvWithShop(t *testing.T, gw gateway.Client, shop stubShop) (*gorm.DB, *Service, *crypto.Cipher) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Payment{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	c := cache.NewFromClient(client, "laundry-pro")
	cipher, err := crypto.NewCipher("test-encryption-key")
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	svc := NewService(db, c, cipher, gw, audit.NewWriter(db), bus.New(), shop)
	return db, svc, cipher
}

// stubShop is a fixed shop-configuration provider for hermetic tests.
type stubShop struct{ currency, prefix string }

func (s stubShop) Defaults(context.Context) (shopconfig.Defaults, error) {
	return shopconfig.Defaults{Currency: s.currency, InvoicePrefix: s.prefix}, nil
}

// countingGateway records charge attempts so idempotency is verifiable.
type countingGateway struct {
	charges int
	decline bool
	lastRef string
	lastAmt float64
	lastCur string
}

func (g *countingGateway) Charge(_ context.Context, ref string, amount float64, currency string) (*gateway.ChargeResult, error) {
	g.charges++
	g.lastRef, g.lastAmt, g.lastCur = ref, amount, currency
	if g.decline {
		return nil, gateway.ErrDeclined
	}
	return &gateway.ChargeResult{TransactionID: "TXN-" + ref, Status: "success"}, nil
}

func (g *countingGateway) Refund(_ context.Context, transactionID string, _ float64) error {
	if transactionID == "" {
		return gateway.ErrDeclined
	}
	return nil
}

func createPayment(t *testing.T, svc *Service) string {
	t.Helper()
	view, err := svc.Create(context.Background(), CreateRequest{
		OrderID: uuid.New(), Amount: 500, Currency: "INR", Method: "card",
		TransactionID: "TXN-SECRET-42", CardLast4: "4242", Gateway: "stub",
	}, audit.Meta{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return view.ID
}

func TestPaymentCreateEncryptsSensitiveFields(t *testing.T) {
	db, svc, cipher := newPaymentEnv(t, &countingGateway{})
	id := createPayment(t, svc)

	var row models.Payment
	if err := db.First(&row, "id = ?", id).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	// Sensitive values must never be stored in plaintext.
	if row.TransactionID == "TXN-SECRET-42" || row.TransactionID == "" {
		t.Fatalf("transaction_id not encrypted at rest: %q", row.TransactionID)
	}
	if row.CardLast4 == "4242" || row.CardLast4 == "" {
		t.Fatalf("card_last4 not encrypted at rest: %q", row.CardLast4)
	}
	// ... but they must decrypt back.
	plain, err := cipher.Decrypt(row.TransactionID)
	if err != nil || plain != "TXN-SECRET-42" {
		t.Fatalf("decrypt failed: %q %v", plain, err)
	}
}

func TestPaymentCaptureSuccess(t *testing.T) {
	_, svc, _ := newPaymentEnv(t, &countingGateway{})
	id := createPayment(t, svc)

	view, err := svc.Capture(context.Background(), CaptureRequest{ID: mustParse(id)}, audit.Meta{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if view.Status != models.PaymentPaid || view.PaidAt == nil {
		t.Fatalf("payment not marked paid: %+v", view)
	}
	if view.CardLast4 == "4242" || view.CardLast4 == "" {
		t.Fatalf("view must expose a masked card, got %q", view.CardLast4)
	}
}

func TestPaymentCaptureIdempotent(t *testing.T) {
	gw := &countingGateway{}
	_, svc, _ := newPaymentEnv(t, gw)
	id := createPayment(t, svc)
	meta := audit.Meta{UserID: uuid.New()}

	if _, err := svc.Capture(context.Background(), CaptureRequest{ID: mustParse(id)}, meta); err != nil {
		t.Fatalf("first Capture: %v", err)
	}
	view, err := svc.Capture(context.Background(), CaptureRequest{ID: mustParse(id)}, meta)
	if err != nil {
		t.Fatalf("second Capture: %v", err)
	}
	if view.Status != models.PaymentPaid {
		t.Fatalf("status after re-capture: %s", view.Status)
	}
	if gw.charges != 1 {
		t.Fatalf("gateway charged %d times, want exactly 1", gw.charges)
	}
}

func TestPaymentCaptureDeclined(t *testing.T) {
	gw := &countingGateway{decline: true}
	db, svc, _ := newPaymentEnv(t, gw)
	id := createPayment(t, svc)

	_, err := svc.Capture(context.Background(), CaptureRequest{ID: mustParse(id)}, audit.Meta{UserID: uuid.New()})
	if err == nil {
		t.Fatal("declined capture must return an error")
	}
	var row models.Payment
	if err := db.First(&row, "id = ?", id).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if row.Status != models.PaymentFailed {
		t.Fatalf("status after decline = %s, want failed", row.Status)
	}
}

func TestPaymentRefund(t *testing.T) {
	_, svc, _ := newPaymentEnv(t, &countingGateway{})
	id := createPayment(t, svc)

	if _, err := svc.Capture(context.Background(), CaptureRequest{ID: mustParse(id)}, audit.Meta{UserID: uuid.New()}); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	view, err := svc.Refund(context.Background(), RefundRequest{ID: mustParse(id), Amount: 500}, audit.Meta{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("Refund: %v", err)
	}
	if view.Status != models.PaymentRefunded {
		t.Fatalf("status after refund = %s, want refunded", view.Status)
	}
	// Double refund must be rejected.
	if _, err := svc.Refund(context.Background(), RefundRequest{ID: mustParse(id), Amount: 500}, audit.Meta{UserID: uuid.New()}); err == nil {
		t.Fatal("second refund must fail")
	}
}

func TestPaymentCreateRequiresOrderIDAndAmount(t *testing.T) {
	_, svc, _ := newPaymentEnv(t, &countingGateway{})
	ctx := context.Background()
	meta := audit.Meta{UserID: uuid.New()}

	if _, err := svc.Create(ctx, CreateRequest{Amount: 10}, meta); err == nil {
		t.Fatal("missing order_id must fail")
	}
	if _, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New()}, meta); err == nil {
		t.Fatal("non-positive amount must fail")
	}
}

func TestPaymentCreateAppliesShopCurrencyDefault(t *testing.T) {
	_, svc, _ := newPaymentEnvWithShop(t, &countingGateway{}, stubShop{currency: "USD"})
	view, err := svc.Create(context.Background(), CreateRequest{
		OrderID: uuid.New(), Amount: 100, Method: "cash",
	}, audit.Meta{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if view.Currency != "USD" {
		t.Fatalf("currency = %s, want shop default USD", view.Currency)
	}
}

func TestPaymentUpdateAndDelete(t *testing.T) {
	db, svc, _ := newPaymentEnv(t, &countingGateway{})
	ctx := context.Background()
	id := createPayment(t, svc)
	meta := audit.Meta{UserID: uuid.New()}

	note := "walk-in"
	updated, err := svc.Update(ctx, UpdateRequest{ID: mustParse(id), Note: &note}, meta)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Note != note {
		t.Fatalf("note = %q, want %q", updated.Note, note)
	}

	// Settled payments cannot be edited after capture.
	if _, err := svc.Capture(ctx, CaptureRequest{ID: mustParse(id)}, meta); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if _, err := svc.Update(ctx, UpdateRequest{ID: mustParse(id), Note: &note}, meta); err == nil {
		t.Fatal("editing a paid payment must fail")
	}

	// Delete removes the row entirely.
	if err := svc.Delete(ctx, IDRequest{ID: mustParse(id)}, meta); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var count int64
	db.Model(&models.Payment{}).Count(&count)
	if count != 0 {
		t.Fatalf("rows after delete = %d, want 0", count)
	}
}

func mustParse(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
