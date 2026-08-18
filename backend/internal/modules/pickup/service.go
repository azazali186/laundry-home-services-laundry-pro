package pickup

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

// Service manages pickups and emits completion events.
type Service struct {
	db     *gorm.DB
	outbox *outbox.Repo
	bus    *bus.Bus
	audit  *audit.Writer
}

// NewService builds the pickup service.
func NewService(db *gorm.DB, outbox *outbox.Repo, bus *bus.Bus, audit *audit.Writer) *Service {
	return &Service{db: db, outbox: outbox, bus: bus, audit: audit}
}

var spec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"booking_id":   pagination.TypeUUID,
		"agent_id":     pagination.TypeUUID,
		"status":       pagination.TypeString,
		"scheduled_at": pagination.TypeTime,
	},
	map[string]pagination.ColumnType{
		"booking_id":   pagination.TypeUUID,
		"agent_id":     pagination.TypeUUID,
		"status":       pagination.TypeString,
		"scheduled_at": pagination.TypeTime,
	},
	"address",
)

// Create adds a pickup.
func (s *Service) Create(ctx context.Context, req CreateRequest, meta audit.Meta) (*View, error) {
	if req.BookingID == uuid.Nil {
		return nil, errors.New(400, "booking_id is required")
	}
	row := models.Pickup{
		BookingID:   req.BookingID,
		AgentID:     req.AgentID,
		ScheduledAt: req.ScheduledAt,
		Address:     req.Address,
		Status:      models.PickupScheduled,
		Notes:       req.Notes,
	}
	if row.ScheduledAt.IsZero() {
		row.ScheduledAt = time.Now()
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "pickup", row.ID.String(), "create", nil, &row)
	s.bus.Publish(bus.Event{Type: "pickup_scheduled", AggregateID: row.ID.String(), Data: row})
	return toView(&row), nil
}

// Update mutates pickup fields.
func (s *Service) Update(ctx context.Context, req UpdateRequest, meta audit.Meta) (*View, error) {
	var row models.Pickup
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	if row.Status == models.PickupCompleted {
		return nil, errors.New(409, "completed pickups cannot be edited")
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
	var after models.Pickup
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "pickup", row.ID.String(), "update", before, &after)
	return toView(&after), nil
}

// Complete marks a pickup completed and emits pickup_completed (saga step).
func (s *Service) Complete(ctx context.Context, req IDRequest, meta audit.Meta) (*View, error) {
	var row models.Pickup
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	if row.Status == models.PickupCompleted {
		return nil, errors.New(409, "pickup already completed")
	}
	before := audit.DiffBefore(&row)
	now := time.Now()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&row).Updates(map[string]any{"status": models.PickupCompleted, "completed_at": &now}).Error; err != nil {
			return err
		}
		return s.outbox.Enqueue(ctx, tx, outbox.Event{
			AggregateType: "pickup",
			AggregateID:   row.ID.String(),
			EventType:     "pickup_completed",
			Payload:       map[string]any{"pickup_id": row.ID.String(), "booking_id": row.BookingID.String()},
		})
	})
	if err != nil {
		return nil, err
	}
	var after models.Pickup
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "pickup", row.ID.String(), "update", before, &after)
	s.bus.Publish(bus.Event{Type: "pickup_completed", AggregateID: row.ID.String(), Data: after})
	return toView(&after), nil
}

// Delete removes a pickup.
func (s *Service) Delete(ctx context.Context, req IDRequest, meta audit.Meta) error {
	var row models.Pickup
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	if err := s.db.WithContext(ctx).Delete(&row).Error; err != nil {
		return err
	}
	s.audit.Log(ctx, meta, "pickup", row.ID.String(), "delete", before, nil)
	return nil
}

// List returns cursor-paged pickups with summary.
func (s *Service) List(ctx context.Context, req ListRequest) ([]View, *response.Pagination, *Summary, error) {
	items := []models.Pickup{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.Pickup{}), req.PageRequest, spec, &items)
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
	base := s.db.WithContext(ctx).Model(&models.Pickup{})
	filtered, err := pagination.FilteredQuery(base, req, spec)
	if err != nil {
		return nil, err
	}
	var sums struct {
		Total     int64 `gorm:"column:total"`
		Scheduled int64 `gorm:"column:scheduled"`
		Completed int64 `gorm:"column:completed"`
		Active    int64 `gorm:"column:active"`
	}
	if err := filtered.Select(`
		COUNT(*) AS total,
		SUM(CASE WHEN status = 'scheduled' OR status = 'in_progress' THEN 1 ELSE 0 END) AS active,
		SUM(CASE WHEN status = 'scheduled' THEN 1 ELSE 0 END) AS scheduled,
		SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) AS completed`).Scan(&sums).Error; err != nil {
		return nil, err
	}
	return &Summary{Total: sums.Total, Scheduled: sums.Scheduled, Completed: sums.Completed, Active: sums.Active}, nil
}

func toView(row *models.Pickup) *View {
	return &View{
		ID:          row.ID.String(),
		BookingID:   row.BookingID.String(),
		AgentID:     agentID(row.AgentID),
		ScheduledAt: row.ScheduledAt,
		CompletedAt: row.CompletedAt,
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
