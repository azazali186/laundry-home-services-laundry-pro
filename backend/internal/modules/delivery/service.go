package delivery

import (
	"context"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/bus"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/modkit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/outbox"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service manages deliveries and emits completion events.
type Service struct {
	db     *gorm.DB
	outbox *outbox.Repo
	bus    *bus.Bus
	audit  *audit.Writer
}

// NewService builds the delivery service.
func NewService(db *gorm.DB, outbox *outbox.Repo, bus *bus.Bus, audit *audit.Writer) *Service {
	return &Service{db: db, outbox: outbox, bus: bus, audit: audit}
}

var spec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"order_id":     pagination.TypeUUID,
		"agent_id":     pagination.TypeUUID,
		"status":       pagination.TypeString,
		"scheduled_at": pagination.TypeTime,
	},
	map[string]pagination.ColumnType{
		"order_id":     pagination.TypeUUID,
		"agent_id":     pagination.TypeUUID,
		"status":       pagination.TypeString,
		"scheduled_at": pagination.TypeTime,
	},
	"address",
)

// Create adds a delivery.
func (s *Service) Create(ctx context.Context, req CreateRequest, meta audit.Meta) (*View, error) {
	if req.OrderID == uuid.Nil {
		return nil, errors.New(400, "order_id is required")
	}
	row := models.Delivery{
		OrderID:     req.OrderID,
		AgentID:     req.AgentID,
		ScheduledAt: req.ScheduledAt,
		Address:     req.Address,
		Status:      models.DeliveryScheduled,
		Notes:       req.Notes,
	}
	if row.ScheduledAt.IsZero() {
		row.ScheduledAt = time.Now()
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return s.outbox.Enqueue(ctx, tx, outbox.Event{
			AggregateType: "delivery",
			AggregateID:   row.ID.String(),
			EventType:     "delivery_scheduled",
			Payload:       map[string]any{"delivery_id": row.ID.String(), "order_id": row.OrderID.String()},
		})
	})
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "delivery", row.ID.String(), "create", nil, &row)
	s.bus.Publish(bus.Event{Type: "delivery_scheduled", AggregateID: row.ID.String(), Data: row})
	return toView(&row), nil
}

// Update mutates delivery fields.
func (s *Service) Update(ctx context.Context, req UpdateRequest, meta audit.Meta) (*View, error) {
	var row models.Delivery
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	if row.Status == models.DeliveryDelivered {
		return nil, errors.New(409, "delivered deliveries cannot be edited")
	}
	before := audit.DiffBefore(&row)
	updates := map[string]any{}
	if req.AgentID != nil {
		updates["agent_id"] = *req.AgentID
	}
	if req.ScheduledAt != nil {
		updates["scheduled_at"] = *req.ScheduledAt
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}
	if req.Notes != nil {
		updates["notes"] = *req.Notes
	}
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	var after models.Delivery
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "delivery", row.ID.String(), "update", before, &after)
	return toView(&after), nil
}

// Complete marks a delivery delivered and emits delivery_delivered (saga step).
func (s *Service) Complete(ctx context.Context, req IDRequest, meta audit.Meta) (*View, error) {
	var row models.Delivery
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	if row.Status == models.DeliveryDelivered {
		return nil, errors.New(409, "delivery already completed")
	}
	before := audit.DiffBefore(&row)
	now := time.Now()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&row).Updates(map[string]any{"status": models.DeliveryDelivered, "delivered_at": &now}).Error; err != nil {
			return err
		}
		return s.outbox.Enqueue(ctx, tx, outbox.Event{
			AggregateType: "delivery",
			AggregateID:   row.ID.String(),
			EventType:     "delivery_delivered",
			Payload:       map[string]any{"delivery_id": row.ID.String(), "order_id": row.OrderID.String()},
		})
	})
	if err != nil {
		return nil, err
	}
	var after models.Delivery
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "delivery", row.ID.String(), "update", before, &after)
	s.bus.Publish(bus.Event{Type: "delivery_delivered", AggregateID: row.ID.String(), Data: after})
	return toView(&after), nil
}

// Delete removes a delivery.
func (s *Service) Delete(ctx context.Context, req IDRequest, meta audit.Meta) error {
	var row models.Delivery
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	if err := s.db.WithContext(ctx).Delete(&row).Error; err != nil {
		return err
	}
	s.audit.Log(ctx, meta, "delivery", row.ID.String(), "delete", before, nil)
	return nil
}

// List returns cursor-paged deliveries with summary.
func (s *Service) List(ctx context.Context, req ListRequest) ([]View, *response.Pagination, *Summary, error) {
	items := []models.Delivery{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.Delivery{}), req.PageRequest, spec, &items)
	if err != nil {
		return nil, nil, nil, err
	}
	views := make([]View, 0, len(items))
	for i := range items {
		views = append(views, *toView(&items[i]))
	}
	summary, err := s.summary(ctx, req.PageRequest)
	if err != nil {
		return nil, nil, nil, err
	}
	return views, modkit.PageRes(res), summary, nil
}

func (s *Service) summary(ctx context.Context, req pagination.PageRequest) (*Summary, error) {
	base := s.db.WithContext(ctx).Model(&models.Delivery{})
	filtered, err := pagination.FilteredQuery(base, req, spec)
	if err != nil {
		return nil, err
	}
	var sums struct {
		Total     int64 `gorm:"column:total"`
		Scheduled int64 `gorm:"column:scheduled"`
		InTransit int64 `gorm:"column:in_transit"`
		Delivered int64 `gorm:"column:delivered"`
		Cancelled int64 `gorm:"column:cancelled"`
	}
	if err := filtered.Select(`
		COUNT(*) AS total,
		SUM(CASE WHEN status = 'scheduled' THEN 1 ELSE 0 END) AS scheduled,
		SUM(CASE WHEN status = 'in_transit' THEN 1 ELSE 0 END) AS in_transit,
		SUM(CASE WHEN status = 'delivered' THEN 1 ELSE 0 END) AS delivered,
		SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) AS cancelled`).Scan(&sums).Error; err != nil {
		return nil, err
	}
	return &Summary{
		Total: sums.Total, Scheduled: sums.Scheduled,
		InTransit: sums.InTransit, Delivered: sums.Delivered, Cancelled: sums.Cancelled,
	}, nil
}

func toView(row *models.Delivery) *View {
	return &View{
		ID:          row.ID.String(),
		OrderID:     row.OrderID.String(),
		AgentID:     agentID(row.AgentID),
		ScheduledAt: row.ScheduledAt,
		DeliveredAt: row.DeliveredAt,
		Address:     row.Address,
		Status:      row.Status,
		Notes:       row.Notes,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func agentID(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
