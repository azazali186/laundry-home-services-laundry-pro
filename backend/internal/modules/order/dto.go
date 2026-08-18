package order

import (
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/google/uuid"
)

// CreateRequest body.
type CreateRequest struct {
	BookingID      uuid.UUID  `json:"booking_id"`
	CustomerID     uuid.UUID  `json:"customer_id"`
	ServiceTypeID  uuid.UUID  `json:"service_type_id"`
	PickupID       *uuid.UUID `json:"pickup_id"`
	Priority       string     `json:"priority"`
	ItemCount      int        `json:"item_count"`
	TotalAmount    float64    `json:"total_amount"`
	TaxRate        *float64   `json:"tax_rate"` // nil = shop default; explicit 0 = tax-exempt
	DiscountAmount float64    `json:"discount_amount"`
	Currency       string     `json:"currency"`
	Note           string     `json:"note"`
}

// UpdateRequest body.
type UpdateRequest struct {
	ID             uuid.UUID `json:"id"`
	Priority       *string   `json:"priority"`
	ItemCount      *int      `json:"item_count"`
	TotalAmount    *float64  `json:"total_amount"`
	TaxAmount      *float64  `json:"tax_amount"`
	DiscountAmount *float64  `json:"discount_amount"`
	Note           *string   `json:"note"`
}

// UpdateStatusRequest body: PATCH-like transition via POST body.
type UpdateStatusRequest struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
	Reason string    `json:"reason"`
}

// IDRequest body.
type IDRequest struct {
	ID uuid.UUID `json:"id"`
}

// ListRequest body.
type ListRequest struct {
	pagination.PageRequest
}

// View public representation.
type View struct {
	ID             string    `json:"id"`
	OrderNo        string    `json:"order_no"`
	BookingID      string    `json:"booking_id"`
	CustomerID     string    `json:"customer_id"`
	ServiceTypeID  string    `json:"service_type_id"`
	PickupID       string    `json:"pickup_id"`
	Status         string    `json:"status"`
	Priority       string    `json:"priority"`
	ItemCount      int       `json:"item_count"`
	TotalAmount    float64   `json:"total_amount"`
	TaxAmount      float64   `json:"tax_amount"`
	DiscountAmount float64   `json:"discount_amount"`
	GrandTotal     float64   `json:"grand_total"`
	Currency       string    `json:"currency"`
	Note           string    `json:"note"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Summary aggregate data (with date range awareness via filters).
type Summary struct {
	Total          int64            `json:"total"`
	ByStatus       map[string]int64 `json:"by_status"`
	Revenue        float64          `json:"revenue"`       // sum grand_total
	Collected      float64          `json:"collected"`     // sum paid payments
	PendingValue   float64          `json:"pending_value"` // sum grand_total non-delivered
	AvgOrderValue  float64          `json:"avg_order_value"`
	ItemsProcessed int64            `json:"items_processed"`
}
