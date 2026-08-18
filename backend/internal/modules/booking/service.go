package booking

import (
	"context"
	"strings"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/bus"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/modkit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/outbox"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/shopconfig"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service manages bookings and emits booking events through the outbox.
type Service struct {
	db     *gorm.DB
	outbox *outbox.Repo
	bus    *bus.Bus
	audit  *audit.Writer
	shop   shopconfig.Provider
}

// NewService builds the booking service.
func NewService(db *gorm.DB, outbox *outbox.Repo, bus *bus.Bus, audit *audit.Writer, shop shopconfig.Provider) *Service {
	shop = shopconfig.Guard(shop)
	return &Service{db: db, outbox: outbox, bus: bus, audit: audit, shop: shop}
}

var spec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"booking_no":       pagination.TypeString,
		"status":           pagination.TypeString,
		"customer_id":      pagination.TypeUUID,
		"service_type_id":  pagination.TypeUUID,
		"estimated_amount": pagination.TypeNumber,
		"source":           pagination.TypeString,
	},
	map[string]pagination.ColumnType{
		"booking_no":       pagination.TypeString,
		"status":           pagination.TypeString,
		"customer_id":      pagination.TypeUUID,
		"service_type_id":  pagination.TypeUUID,
		"estimated_amount": pagination.TypeNumber,
		"source":           pagination.TypeString,
		"pickup_from":      pagination.TypeTime,
		"pickup_to":        pagination.TypeTime,
	},
	"booking_no", "pickup_address",
)

// Create adds a booking and queues booking_created for the saga.
func (s *Service) Create(ctx context.Context, req CreateRequest, meta audit.Meta) (*View, error) {
	if req.CustomerID == uuid.Nil || req.ServiceTypeID == uuid.Nil {
		return nil, errors.New(400, "customer_id and service_type_id are required")
	}
	if strings.TrimSpace(req.PickupAddress) == "" {
		return nil, errors.New(400, "pickup_address is required")
	}
	defaults, err := s.shop.Defaults(ctx)
	if err != nil {
		return nil, err
	}
	row := models.Booking{
		BookingNo:       modkit.NextNo("BKG"),
		CustomerID:      req.CustomerID,
		ServiceTypeID:   req.ServiceTypeID,
		PickupAddress:   strings.TrimSpace(req.PickupAddress),
		DropoffAddress:  req.DropoffAddress,
		PickupFrom:      req.PickupFrom,
		PickupTo:        req.PickupTo,
		Status:          models.BookingPending,
		Notes:           req.Notes,
		EstimatedAmount: req.EstimatedAmount,
		Currency:        modkit.OrDefault(req.Currency, defaults.Currency),
		Source:          modkit.OrDefault(req.Source, "web"),
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return s.outbox.Enqueue(ctx, tx, outbox.Event{
			AggregateType: "booking",
			AggregateID:   row.ID.String(),
			EventType:     "booking_created",
			Payload: map[string]any{
				"booking_id": row.ID.String(), "booking_no": row.BookingNo,
				"customer_id": row.CustomerID.String(), "service_type_id": row.ServiceTypeID.String(),
				"pickup_address": row.PickupAddress, "pickup_from": row.PickupFrom, "pickup_to": row.PickupTo,
			},
		})
	})
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "booking", row.ID.String(), "create", nil, &row)
	s.emit(ctx, "booking_created", row.ID, row)
	return toView(&row), nil
}

// Confirm moves a pending booking to confirmed.
func (s *Service) Confirm(ctx context.Context, id uuid.UUID, meta audit.Meta) (*View, error) {
	var row models.Booking
	if err := s.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	if row.Status != models.BookingPending {
		return nil, errors.New(409, "only pending bookings can be confirmed")
	}
	before := audit.DiffBefore(&row)
	row.Status = models.BookingConfirmed
	if err := s.db.WithContext(ctx).Model(&row).Update("status", row.Status).Error; err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "booking", row.ID.String(), "update", before, &row)
	return toView(&row), nil
}

// Cancel transitions a booking to cancelled.
func (s *Service) Cancel(ctx context.Context, req CancelRequest, meta audit.Meta) (*View, error) {
	var row models.Booking
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	if row.Status == models.BookingCancelled {
		return nil, errors.New(409, "booking already cancelled")
	}
	before := audit.DiffBefore(&row)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&row).Update("status", models.BookingCancelled).Error; err != nil {
			return err
		}
		return s.outbox.Enqueue(ctx, tx, outbox.Event{
			AggregateType: "booking",
			AggregateID:   row.ID.String(),
			EventType:     "booking_cancelled",
			Payload:       map[string]any{"booking_id": row.ID.String(), "reason": req.Reason},
		})
	})
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "booking", row.ID.String(), "update", before, &row)
	s.emit(ctx, "booking_cancelled", row.ID, row)
	return toView(&row), nil
}

// Update mutates editable booking fields.
func (s *Service) Update(ctx context.Context, req UpdateRequest, meta audit.Meta) (*View, error) {
	var row models.Booking
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	if row.Status != models.BookingPending {
		return nil, errors.New(409, "only pending bookings can be edited")
	}
	before := audit.DiffBefore(&row)
	updates := map[string]any{}
	if req.PickupAddress != nil {
		updates["pickup_address"] = strings.TrimSpace(*req.PickupAddress)
	}
	if req.DropoffAddress != nil {
		updates["dropoff_address"] = *req.DropoffAddress
	}
	if req.PickupFrom != nil {
		updates["pickup_from"] = *req.PickupFrom
	}
	if req.PickupTo != nil {
		updates["pickup_to"] = *req.PickupTo
	}
	if req.Notes != nil {
		updates["notes"] = *req.Notes
	}
	if req.EstimatedAmount != nil {
		updates["estimated_amount"] = *req.EstimatedAmount
	}
	if len(updates) == 0 {
		return toView(&row), nil
	}
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	var after models.Booking
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "booking", row.ID.String(), "update", before, &after)
	return toView(&after), nil
}

// Delete removes a booking.
func (s *Service) Delete(ctx context.Context, req DeleteRequest, meta audit.Meta) error {
	var row models.Booking
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	if err := s.db.WithContext(ctx).Delete(&row).Error; err != nil {
		return err
	}
	s.audit.Log(ctx, meta, "booking", row.ID.String(), "delete", before, nil)
	return nil
}

// List returns a cursor-paged booking list with summary data.
func (s *Service) List(ctx context.Context, req ListRequest) ([]View, *response.Pagination, *Summary, error) {
	items := []models.Booking{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.Booking{}), req.PageRequest, spec, &items)
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
	base := s.db.WithContext(ctx).Model(&models.Booking{})
	filtered, err := pagination.FilteredQuery(base, req, spec)
	if err != nil {
		return nil, err
	}
	var sums struct {
		Total          int64   `gorm:"column:total"`
		Pending        int64   `gorm:"column:pending"`
		Confirmed      int64   `gorm:"column:confirmed"`
		Cancelled      int64   `gorm:"column:cancelled"`
		EstimatedValue float64 `gorm:"column:estimated_value"`
		ThisMonth      int64   `gorm:"column:this_month"`
	}
	monthStart := time.Now().AddDate(0, -1, 0)
	if err := filtered.Select(`
		COUNT(*) AS total,
		SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) AS pending,
		SUM(CASE WHEN status = 'confirmed' THEN 1 ELSE 0 END) AS confirmed,
		SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) AS cancelled,
		COALESCE(SUM(estimated_amount), 0) AS estimated_value,
		SUM(CASE WHEN created_at >= ? THEN 1 ELSE 0 END) AS this_month`, monthStart).Scan(&sums).Error; err != nil {
		return nil, err
	}
	byStatus := map[string]int64{}
	type st struct {
		Status string `gorm:"column:status"`
		Count  int64  `gorm:"column:cnt"`
	}
	var rows []st
	s.db.WithContext(ctx).Model(&models.Booking{}).Select("status, COUNT(*) AS cnt").Group("status").Scan(&rows)
	for _, r := range rows {
		byStatus[r.Status] = r.Count
	}
	return &Summary{
		Total:          sums.Total,
		Pending:        sums.Pending,
		Confirmed:      sums.Confirmed,
		Cancelled:      sums.Cancelled,
		EstimatedValue: sums.EstimatedValue,
		ThisMonth:      sums.ThisMonth,
		ByStatus:       byStatus,
	}, nil
}

func (s *Service) emit(ctx context.Context, eventType string, id uuid.UUID, row any) {
	s.bus.Publish(bus.Event{Type: eventType, AggregateID: id.String(), Data: row})
}

func toView(row *models.Booking) *View {
	return &View{
		ID:              row.ID.String(),
		BookingNo:       row.BookingNo,
		CustomerID:      row.CustomerID.String(),
		ServiceTypeID:   row.ServiceTypeID.String(),
		PickupAddress:   row.PickupAddress,
		DropoffAddress:  row.DropoffAddress,
		PickupFrom:      row.PickupFrom,
		PickupTo:        row.PickupTo,
		Status:          row.Status,
		Notes:           row.Notes,
		EstimatedAmount: row.EstimatedAmount,
		Currency:        row.Currency,
		Source:          row.Source,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}
