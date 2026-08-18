package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"gorm.io/gorm"
)

// DashboardRequest body: date range in body (no query params).
type DashboardRequest struct {
	From string `json:"from"` // RFC3339 or YYYY-MM-DD
	To   string `json:"to"`
}

// RevenuePoint is a revenue-per-day bucket.
type RevenuePoint struct {
	Day     string  `json:"day"`
	Revenue float64 `json:"revenue"`
	Orders  int64   `json:"orders"`
}

// StatusCount is a status bucket.
type StatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// TopService is a top service type bucket.
type TopService struct {
	ServiceTypeID string  `json:"service_type_id"`
	Orders        int64   `json:"orders"`
	Revenue       float64 `json:"revenue"`
}

// Dashboard is the aggregate analytics payload.
type Dashboard struct {
	Period          PeriodSummary  `json:"period"`
	RevenueTrend    []RevenuePoint `json:"revenue_trend"`
	OrdersByStatus  []StatusCount  `json:"orders_by_status"`
	BookingsByMonth []StatusCount  `json:"bookings_by_month"`
	TopServices     []TopService   `json:"top_services"`
	Operational     Operational    `json:"operational"`
}

// PeriodSummary covers the requested range.
type PeriodSummary struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Revenue  float64 `json:"revenue"`
	Orders   int64   `json:"orders"`
	Bookings int64   `json:"bookings"`
	AvgOrder float64 `json:"avg_order_value"`
}

// Operational captures live pipeline counts.
type Operational struct {
	PickupsPending  int64 `json:"pickups_pending"`
	DeliveriesOut   int64 `json:"deliveries_out"`
	GarmentsInQueue int64 `json:"garments_in_queue"`
	UnreadInvoices  int64 `json:"unread_invoices"`
	LowStockItems   int64 `json:"low_stock_items"`
}

// Service computes cross-entity analytics via GORM only.
type Service struct{ db *gorm.DB }

// NewService builds the analytics service.
func NewService(db *gorm.DB) *Service { return &Service{db: db} }

// Dashboard aggregates the operational picture for a date range.
func (s *Service) Dashboard(ctx context.Context, req DashboardRequest) (*Dashboard, error) {
	from, to := parseRange(req.From, req.To)

	base := s.db.WithContext(ctx)

	var period PeriodSummary
	if err := base.Model(&models.Order{}).
		Where("created_at BETWEEN ? AND ?", from, to).
		Select("COALESCE(SUM(grand_total), 0) AS revenue, COUNT(*) AS orders, COALESCE(AVG(grand_total), 0) AS avg_order").
		Scan(&period).Error; err != nil {
		return nil, fmt.Errorf("query period summary: %w", err)
	}
	// Scan resets dest fields; set the window after so from/to always survive.
	period.From = from.Format(time.RFC3339)
	period.To = to.Format(time.RFC3339)
	base.Model(&models.Booking{}).Where("created_at BETWEEN ? AND ?", from, to).Count(&period.Bookings)

	var trend []RevenuePoint
	base.Model(&models.Order{}).
		Where("created_at BETWEEN ? AND ?", from, to).
		Select("DATE(created_at) AS day, COALESCE(SUM(grand_total), 0) AS revenue, COUNT(*) AS orders").
		Group("DATE(created_at)").Order("day ASC").Scan(&trend)

	var byStatus []StatusCount
	base.Model(&models.Order{}).
		Where("created_at BETWEEN ? AND ?", from, to).
		Select("status, COUNT(*) AS count").Group("status").Scan(&byStatus)

	var byMonth []StatusCount
	base.Model(&models.Booking{}).
		Where("created_at BETWEEN ? AND ?", from, to).
		Select("strftime('%Y-%m', created_at) AS status, COUNT(*) AS count").Group("status").Order("status ASC").Scan(&byMonth)
	// Postgres uses to_char; normalize for portability.
	if len(byMonth) == 0 {
		base.Model(&models.Booking{}).
			Where("created_at BETWEEN ? AND ?", from, to).
			Select("to_char(created_at, 'YYYY-MM') AS status, COUNT(*) AS count").Group("status").Order("status ASC").Scan(&byMonth)
	}

	var top []TopService
	base.Model(&models.Order{}).
		Where("created_at BETWEEN ? AND ?", from, to).
		Select("service_type_id, COUNT(*) AS orders, COALESCE(SUM(grand_total), 0) AS revenue").
		Group("service_type_id").Order("revenue DESC").Limit(5).Scan(&top)

	var ops Operational
	base.Model(&models.Pickup{}).Where("status = ?", models.PickupScheduled).Count(&ops.PickupsPending)
	base.Model(&models.Delivery{}).Where("status IN ?", []string{models.DeliveryScheduled, models.DeliveryInTransit}).Count(&ops.DeliveriesOut)
	base.Model(&models.Garment{}).Where("status IN ?", []string{models.GarmentReceived, models.GarmentCleaned, models.GarmentIroned}).Count(&ops.GarmentsInQueue)
	base.Model(&models.Invoice{}).Where("status = ?", models.InvoiceIssued).Count(&ops.UnreadInvoices)
	base.Model(&models.InventoryItem{}).Where("quantity_on_hand > 0 AND quantity_on_hand <= reorder_level").Count(&ops.LowStockItems)

	return &Dashboard{
		Period:          period,
		RevenueTrend:    trend,
		OrdersByStatus:  byStatus,
		BookingsByMonth: byMonth,
		TopServices:     top,
		Operational:     ops,
	}, nil
}

func parseRange(from, to string) (time.Time, time.Time) {
	layout := "2006-01-02"
	if len(from) > 10 {
		layout = time.RFC3339
	}
	f, err := time.Parse(layout, from)
	if err != nil {
		f = time.Now().AddDate(0, -1, 0)
	}
	t, err := time.Parse(layout, to)
	if err != nil {
		t = time.Now()
	}
	return f.UTC(), t.UTC()
}
