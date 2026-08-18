package invoice

import (
	"context"
	"strings"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/bus"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/modkit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/shopconfig"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service manages invoices.
type Service struct {
	db    *gorm.DB
	bus   *bus.Bus
	audit *audit.Writer
	shop  shopconfig.Provider
}

// NewService builds the invoice service.
func NewService(db *gorm.DB, bus *bus.Bus, audit *audit.Writer, shop shopconfig.Provider) *Service {
	shop = shopconfig.Guard(shop)
	return &Service{db: db, bus: bus, audit: audit, shop: shop}
}

var spec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"invoice_no": pagination.TypeString,
		"order_id":   pagination.TypeUUID,
		"status":     pagination.TypeString,
		"total":      pagination.TypeNumber,
		"issue_date": pagination.TypeTime,
	},
	map[string]pagination.ColumnType{
		"invoice_no": pagination.TypeString,
		"order_id":   pagination.TypeUUID,
		"status":     pagination.TypeString,
		"total":      pagination.TypeNumber,
		"issue_date": pagination.TypeTime,
	},
	"invoice_no", "billing_name",
)

// Create adds an invoice for an order.
func (s *Service) Create(ctx context.Context, req CreateRequest, meta audit.Meta) (*View, error) {
	if req.OrderID == uuid.Nil {
		return nil, errors.New(400, "order_id is required")
	}
	defaults, err := s.shop.Defaults(ctx)
	if err != nil {
		return nil, err
	}
	prefix := strings.TrimSpace(defaults.InvoicePrefix)
	if prefix == "" {
		prefix = "INV"
	}
	total := req.Subtotal + req.TaxAmount - req.Discount
	row := models.Invoice{
		OrderID:     req.OrderID,
		InvoiceNo:   modkit.NextNo(prefix),
		IssueDate:   time.Now(),
		DueDate:     req.DueDate,
		Subtotal:    req.Subtotal,
		TaxAmount:   req.TaxAmount,
		Discount:    req.Discount,
		Total:       total,
		Status:      models.InvoiceDraft,
		BillingName: req.BillingName,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "invoice", row.ID.String(), "create", nil, &row)
	return toView(&row), nil
}

// Update mutates invoice fields.
func (s *Service) Update(ctx context.Context, req UpdateRequest, meta audit.Meta) (*View, error) {
	var row models.Invoice
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	if row.Status == models.InvoicePaid {
		return nil, errors.New(409, "paid invoices cannot be edited")
	}
	before := audit.DiffBefore(&row)
	updates := map[string]any{}
	if req.DueDate != nil {
		updates["due_date"] = *req.DueDate
	}
	if req.Subtotal != nil {
		updates["subtotal"] = *req.Subtotal
	}
	if req.TaxAmount != nil {
		updates["tax_amount"] = *req.TaxAmount
	}
	if req.Discount != nil {
		updates["discount"] = *req.Discount
	}
	if req.BillingName != nil {
		updates["billing_name"] = *req.BillingName
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	// Recompute total when money fields change.
	if req.Subtotal != nil || req.TaxAmount != nil || req.Discount != nil {
		sub := row.Subtotal
		tax := row.TaxAmount
		disc := row.Discount
		if req.Subtotal != nil {
			sub = *req.Subtotal
		}
		if req.TaxAmount != nil {
			tax = *req.TaxAmount
		}
		if req.Discount != nil {
			disc = *req.Discount
		}
		updates["total"] = sub + tax - disc
	}
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	var after models.Invoice
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "invoice", row.ID.String(), "update", before, &after)
	return toView(&after), nil
}

// MarkPaid settles an invoice.
func (s *Service) MarkPaid(ctx context.Context, req IDRequest, meta audit.Meta) (*View, error) {
	var row models.Invoice
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	if row.Status == models.InvoicePaid {
		return toView(&row), nil
	}
	before := audit.DiffBefore(&row)
	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&row).Updates(map[string]any{"status": models.InvoicePaid, "paid_at": &now}).Error; err != nil {
		return nil, err
	}
	var after models.Invoice
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "invoice", row.ID.String(), "update", before, &after)
	s.bus.Publish(bus.Event{Type: "invoice_paid", AggregateID: row.ID.String(), Data: after})
	return toView(&after), nil
}

// Delete removes an invoice.
func (s *Service) Delete(ctx context.Context, req IDRequest, meta audit.Meta) error {
	var row models.Invoice
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	if err := s.db.WithContext(ctx).Delete(&row).Error; err != nil {
		return err
	}
	s.audit.Log(ctx, meta, "invoice", row.ID.String(), "delete", before, nil)
	return nil
}

// List returns cursor-paged invoices with money summary.
func (s *Service) List(ctx context.Context, req ListRequest) ([]View, *response.Pagination, *Summary, error) {
	items := []models.Invoice{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.Invoice{}), req.PageRequest, spec, &items)
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
	base := s.db.WithContext(ctx).Model(&models.Invoice{})
	filtered, err := pagination.FilteredQuery(base, req, spec)
	if err != nil {
		return nil, err
	}
	var sums struct {
		Total       int64   `gorm:"column:total"`
		Draft       int64   `gorm:"column:draft"`
		Issued      int64   `gorm:"column:issued"`
		Paid        int64   `gorm:"column:paid"`
		Void        int64   `gorm:"column:void"`
		Outstanding float64 `gorm:"column:outstanding"`
		Collected   float64 `gorm:"column:collected"`
	}
	if err := filtered.Select(`
		COUNT(*) AS total,
		SUM(CASE WHEN status = 'draft' THEN 1 ELSE 0 END) AS draft,
		SUM(CASE WHEN status = 'issued' THEN 1 ELSE 0 END) AS issued,
		SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END) AS paid,
		SUM(CASE WHEN status = 'void' THEN 1 ELSE 0 END) AS void,
		COALESCE(SUM(CASE WHEN status = 'issued' THEN total ELSE 0 END), 0) AS outstanding,
		COALESCE(SUM(CASE WHEN status = 'paid' THEN total ELSE 0 END), 0) AS collected`).Scan(&sums).Error; err != nil {
		return nil, err
	}
	byStatus := map[string]int64{}
	type st struct {
		Status string `gorm:"column:status"`
		Count  int64  `gorm:"column:cnt"`
	}
	var rows []st
	s.db.WithContext(ctx).Model(&models.Invoice{}).Select("status, COUNT(*) AS cnt").Group("status").Scan(&rows)
	for _, r := range rows {
		byStatus[r.Status] = r.Count
	}
	return &Summary{
		Total: sums.Total, Draft: sums.Draft, Issued: sums.Issued,
		Paid: sums.Paid, Void: sums.Void,
		Outstanding: sums.Outstanding, Collected: sums.Collected, ByStatus: byStatus,
	}, nil
}

func toView(row *models.Invoice) *View {
	return &View{
		ID:          row.ID.String(),
		OrderID:     row.OrderID.String(),
		InvoiceNo:   row.InvoiceNo,
		IssueDate:   row.IssueDate,
		DueDate:     row.DueDate,
		Subtotal:    row.Subtotal,
		TaxAmount:   row.TaxAmount,
		Discount:    row.Discount,
		Total:       row.Total,
		Status:      row.Status,
		PaidAt:      row.PaidAt,
		BillingName: row.BillingName,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
