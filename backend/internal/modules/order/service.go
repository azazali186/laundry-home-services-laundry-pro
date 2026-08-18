package order

import (
	"context"

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

// Service manages orders and emits status-change events.
type Service struct {
	db     *gorm.DB
	outbox *outbox.Repo
	bus    *bus.Bus
	audit  *audit.Writer
	shop   shopconfig.Provider
}

// NewService builds the order service.
func NewService(db *gorm.DB, outbox *outbox.Repo, bus *bus.Bus, audit *audit.Writer, shop shopconfig.Provider) *Service {
	shop = shopconfig.Guard(shop)
	return &Service{db: db, outbox: outbox, bus: bus, audit: audit, shop: shop}
}

var spec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"order_no":        pagination.TypeString,
		"status":          pagination.TypeString,
		"priority":        pagination.TypeString,
		"customer_id":     pagination.TypeUUID,
		"service_type_id": pagination.TypeUUID,
		"booking_id":      pagination.TypeUUID,
		"grand_total":     pagination.TypeNumber,
	},
	map[string]pagination.ColumnType{
		"order_no":        pagination.TypeString,
		"status":          pagination.TypeString,
		"priority":        pagination.TypeString,
		"customer_id":     pagination.TypeUUID,
		"service_type_id": pagination.TypeUUID,
		"booking_id":      pagination.TypeUUID,
		"grand_total":     pagination.TypeNumber,
	},
	"order_no", "note",
)

// Create adds an order; amounts are derived when omitted.
func (s *Service) Create(ctx context.Context, req CreateRequest, meta audit.Meta) (*View, error) {
	if req.CustomerID == uuid.Nil || req.ServiceTypeID == uuid.Nil {
		return nil, errors.New(400, "customer_id and service_type_id are required")
	}
	defaults, err := s.shop.Defaults(ctx)
	if err != nil {
		return nil, err
	}
	taxRate := defaults.TaxRate
	if req.TaxRate != nil {
		taxRate = *req.TaxRate
	}
	taxAmount := req.TotalAmount * taxRate / 100
	grand := req.TotalAmount + taxAmount - req.DiscountAmount
	row := models.Order{
		OrderNo:        modkit.NextNo("ORD"),
		BookingID:      req.BookingID,
		CustomerID:     req.CustomerID,
		ServiceTypeID:  req.ServiceTypeID,
		PickupID:       req.PickupID,
		Status:         models.OrderCreated,
		Priority:       modkit.OrDefault(req.Priority, "normal"),
		ItemCount:      req.ItemCount,
		TotalAmount:    req.TotalAmount,
		TaxAmount:      taxAmount,
		DiscountAmount: req.DiscountAmount,
		GrandTotal:     grand,
		Currency:       modkit.OrDefault(req.Currency, defaults.Currency),
		Note:           req.Note,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return s.outbox.Enqueue(ctx, tx, outbox.Event{
			AggregateType: "order",
			AggregateID:   row.ID.String(),
			EventType:     "order_created",
			Payload: map[string]any{
				"order_id": row.ID.String(), "order_no": row.OrderNo,
				"booking_id": row.BookingID.String(), "customer_id": row.CustomerID.String(),
			},
		})
	})
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "order", row.ID.String(), "create", nil, &row)
	s.bus.Publish(bus.Event{Type: "order_created", AggregateID: row.ID.String(), Data: row})
	return toView(&row), nil
}

// List returns offset-paged orders with revenue summary.
func (s *Service) List(ctx context.Context, req ListRequest) ([]View, *response.Pagination, *Summary, error) {
	items := []models.Order{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.Order{}), req.PageRequest, spec, &items)
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
	base := s.db.WithContext(ctx).Model(&models.Order{})
	filtered, err := pagination.FilteredQuery(base, req, spec)
	if err != nil {
		return nil, err
	}
	var sums struct {
		Total        int64   `gorm:"column:total"`
		Revenue      float64 `gorm:"column:revenue"`
		PendingValue float64 `gorm:"column:pending_value"`
		AvgOrder     float64 `gorm:"column:avg_order"`
		Items        int64   `gorm:"column:items"`
	}
	if err := filtered.Select(`
		COUNT(*) AS total,
		COALESCE(SUM(grand_total), 0) AS revenue,
		COALESCE(SUM(CASE WHEN status NOT IN ('delivered','cancelled') THEN grand_total ELSE 0 END), 0) AS pending_value,
		COALESCE(AVG(grand_total), 0) AS avg_order,
		COALESCE(SUM(item_count), 0) AS items`).Scan(&sums).Error; err != nil {
		return nil, err
	}
	byStatus := map[string]int64{}
	type st struct {
		Status string `gorm:"column:status"`
		Count  int64  `gorm:"column:cnt"`
	}
	var rows []st
	s.db.WithContext(ctx).Model(&models.Order{}).Select("status, COUNT(*) AS cnt").Group("status").Scan(&rows)
	for _, r := range rows {
		byStatus[r.Status] = r.Count
	}
	var collected float64
	s.db.WithContext(ctx).Model(&models.Payment{}).Where("status = ?", models.PaymentPaid).Select("COALESCE(SUM(amount), 0)").Scan(&collected)
	return &Summary{
		Total: sums.Total, ByStatus: byStatus, Revenue: sums.Revenue,
		Collected: collected, PendingValue: sums.PendingValue,
		AvgOrderValue: sums.AvgOrder, ItemsProcessed: sums.Items,
	}, nil
}

func toView(row *models.Order) *View {
	return &View{
		ID:             row.ID.String(),
		OrderNo:        row.OrderNo,
		BookingID:      row.BookingID.String(),
		CustomerID:     row.CustomerID.String(),
		ServiceTypeID:  row.ServiceTypeID.String(),
		PickupID:       strOrEmpty(row.PickupID),
		Status:         row.Status,
		Priority:       row.Priority,
		ItemCount:      row.ItemCount,
		TotalAmount:    row.TotalAmount,
		TaxAmount:      row.TaxAmount,
		DiscountAmount: row.DiscountAmount,
		GrandTotal:     row.GrandTotal,
		Currency:       row.Currency,
		Note:           row.Note,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func strOrEmpty(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
