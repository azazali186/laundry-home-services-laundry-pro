package payment

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/bus"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/gateway"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/lock"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/modkit"
)

// Capture charges a pending payment through the gateway (breaker + retry),
// guarded by a distributed lock to guarantee idempotency.
func (s *Service) Capture(ctx context.Context, req CaptureRequest, meta audit.Meta) (*View, error) {
	mtx, err := lock.Acquire(ctx, s.cache, "payment:capture:"+req.ID.String(), 15*time.Second, 5*time.Second)
	if err != nil {
		return nil, errors.New(409, "payment capture already in progress")
	}
	defer mtx.Release(ctx)

	var row models.Payment
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	if row.Status == models.PaymentPaid {
		return toView(&row, s.cipher), nil // idempotent
	}
	if row.Status != models.PaymentPending {
		return nil, errors.New(409, "only pending payments can be captured")
	}
	before := audit.DiffBefore(&row)

	ref := row.PaymentNo + "-" + row.ID.String()
	result, err := s.chargeWithResilience(ctx, ref, row.Amount, row.Currency)
	if err != nil {
		row.Status = models.PaymentFailed
		s.db.WithContext(ctx).Model(&row).Update("status", row.Status)
		s.audit.Log(ctx, meta, "payment", row.ID.String(), "capture_failed", before, &row)
		s.bus.Publish(bus.Event{Type: "payment_failed", AggregateID: row.ID.String(), Data: row})
		return toView(&row, s.cipher), errors.New(402, "payment capture failed")
	}

	now := time.Now()
	txID, _ := s.cipher.Encrypt(result.TransactionID)
	updates := map[string]any{
		"status": models.PaymentPaid, "transaction_id": txID,
		"gateway": modkit.OrDefault(row.Gateway, "stub"), "paid_at": &now,
	}
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	var after models.Payment
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "payment", row.ID.String(), "update", before, &after)
	s.bus.Publish(bus.Event{Type: "payment_paid", AggregateID: row.ID.String(), Data: after})
	return toView(&after, s.cipher), nil
}

// Refund reverses a paid payment.
func (s *Service) Refund(ctx context.Context, req RefundRequest, meta audit.Meta) (*View, error) {
	var row models.Payment
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	if row.Status != models.PaymentPaid {
		return nil, errors.New(409, "only paid payments can be refunded")
	}
	before := audit.DiffBefore(&row)
	txID, err := s.cipher.Decrypt(row.TransactionID)
	if err != nil {
		return nil, err
	}
	_, err = s.breaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return nil, s.gateway.Refund(ctx, txID, req.Amount)
	})
	if err != nil {
		return nil, errors.New(502, "refund failed at gateway")
	}
	if err := s.db.WithContext(ctx).Model(&row).Update("status", models.PaymentRefunded).Error; err != nil {
		return nil, err
	}
	var after models.Payment
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "payment", row.ID.String(), "refund", before, &after)
	s.bus.Publish(bus.Event{Type: "payment_refunded", AggregateID: row.ID.String(), Data: after})
	return toView(&after, s.cipher), nil
}

// chargeWithResilience runs the gateway call through the breaker; retry is
// handled inside the breaker's Execute for transient errors.
func (s *Service) chargeWithResilience(ctx context.Context, ref string, amount float64, currency string) (*gateway.ChargeResult, error) {
	result, err := s.breaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return s.gateway.Charge(ctx, ref, amount, currency)
	})
	if err != nil {
		return nil, err
	}
	res, ok := result.(*gateway.ChargeResult)
	if !ok || stderrors.Is(err, gateway.ErrDeclined) {
		return nil, errors.New(402, "payment declined")
	}
	return res, nil
}
