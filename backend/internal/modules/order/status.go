package order

import (
	"context"
	"strings"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/bus"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/outbox"
	"gorm.io/gorm"
)

// validPipeline defines the allowed forward status transitions.
var validPipeline = map[string]string{
	models.OrderCreated:        models.OrderProcessing,
	models.OrderProcessing:     models.OrderQualityCheck,
	models.OrderQualityCheck:   models.OrderReady,
	models.OrderReady:          models.OrderOutForDelivery,
	models.OrderOutForDelivery: models.OrderDelivered,
}

// Update mutates editable order fields.
func (s *Service) Update(ctx context.Context, req UpdateRequest, meta audit.Meta) (*View, error) {
	var row models.Order
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	updates := map[string]any{}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.ItemCount != nil {
		updates["item_count"] = *req.ItemCount
	}
	if req.TotalAmount != nil {
		updates["total_amount"] = *req.TotalAmount
	}
	if req.TaxAmount != nil {
		updates["tax_amount"] = *req.TaxAmount
	}
	if req.DiscountAmount != nil {
		updates["discount_amount"] = *req.DiscountAmount
	}
	if req.Note != nil {
		updates["note"] = *req.Note
	}
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	var after models.Order
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	after.GrandTotal = after.TotalAmount + after.TaxAmount - after.DiscountAmount
	s.db.WithContext(ctx).Model(&after).Update("grand_total", after.GrandTotal)
	s.audit.Log(ctx, meta, "order", row.ID.String(), "update", before, &after)
	return toView(&after), nil
}

// UpdateStatus transitions an order along the pipeline (or cancels).
func (s *Service) UpdateStatus(ctx context.Context, req UpdateStatusRequest, meta audit.Meta) (*View, error) {
	var row models.Order
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	if req.Status == models.OrderCancelled {
		if row.Status == models.OrderDelivered || row.Status == models.OrderCancelled {
			return nil, errors.New(409, "cannot cancel a "+row.Status+" order")
		}
	} else if validPipeline[row.Status] != req.Status {
		return nil, errors.New(409, "invalid transition from "+row.Status+" to "+req.Status)
	}
	before := audit.DiffBefore(&row)
	from := row.Status
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&row).Update("status", req.Status).Error; err != nil {
			return err
		}
		return s.outbox.Enqueue(ctx, tx, outbox.Event{
			AggregateType: "order",
			AggregateID:   row.ID.String(),
			EventType:     "order_status_changed",
			Payload: map[string]any{
				"order_id": row.ID.String(), "order_no": row.OrderNo,
				"from": from, "to": req.Status, "reason": req.Reason,
			},
		})
	})
	if err != nil {
		return nil, err
	}
	var after models.Order
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "order", row.ID.String(), "update", before, &after)
	s.bus.Publish(bus.Event{Type: "order_status_changed", AggregateID: row.ID.String(), Data: after})
	return toView(&after), nil
}

// Delete removes an order.
func (s *Service) Delete(ctx context.Context, req IDRequest, meta audit.Meta) error {
	var row models.Order
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	if err := s.db.WithContext(ctx).Delete(&row).Error; err != nil {
		return err
	}
	s.audit.Log(ctx, meta, "order", row.ID.String(), "delete", before, nil)
	return nil
}

// IsValidPipeline exposes the transition map for tests.
func IsValidPipeline(from, to string) bool { return validPipeline[from] == to }

// NormalizeStatus trims and lowercases a status input.
func NormalizeStatus(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
