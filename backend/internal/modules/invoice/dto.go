package invoice

import (
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/google/uuid"
)

// CreateRequest body.
type CreateRequest struct {
	OrderID     uuid.UUID  `json:"order_id"`
	DueDate     *time.Time `json:"due_date"`
	Subtotal    float64    `json:"subtotal"`
	TaxAmount   float64    `json:"tax_amount"`
	Discount    float64    `json:"discount"`
	BillingName string     `json:"billing_name"`
}

// UpdateRequest body.
type UpdateRequest struct {
	ID          uuid.UUID  `json:"id"`
	DueDate     *time.Time `json:"due_date"`
	Subtotal    *float64   `json:"subtotal"`
	TaxAmount   *float64   `json:"tax_amount"`
	Discount    *float64   `json:"discount"`
	BillingName *string    `json:"billing_name"`
	Status      *string    `json:"status"`
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
	ID          string     `json:"id"`
	OrderID     string     `json:"order_id"`
	InvoiceNo   string     `json:"invoice_no"`
	IssueDate   time.Time  `json:"issue_date"`
	DueDate     *time.Time `json:"due_date"`
	Subtotal    float64    `json:"subtotal"`
	TaxAmount   float64    `json:"tax_amount"`
	Discount    float64    `json:"discount"`
	Total       float64    `json:"total"`
	Status      string     `json:"status"`
	PaidAt      *time.Time `json:"paid_at"`
	BillingName string     `json:"billing_name"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Summary aggregate data.
type Summary struct {
	Total       int64            `json:"total"`
	Draft       int64            `json:"draft"`
	Issued      int64            `json:"issued"`
	Paid        int64            `json:"paid"`
	Void        int64            `json:"void"`
	Outstanding float64          `json:"outstanding"`
	Collected   float64          `json:"collected"`
	ByStatus    map[string]int64 `json:"by_status"`
}
