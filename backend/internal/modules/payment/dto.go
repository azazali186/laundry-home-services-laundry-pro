package payment

import (
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/google/uuid"
)

// CreateRequest body. TransactionID/CardLast4 are encrypted at rest.
type CreateRequest struct {
	OrderID       uuid.UUID `json:"order_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Method        string    `json:"method"`
	Gateway       string    `json:"gateway"`
	TransactionID string    `json:"transaction_id"`
	CardLast4     string    `json:"card_last4"`
	Note          string    `json:"note"`
}

// UpdateRequest body.
type UpdateRequest struct {
	ID      uuid.UUID `json:"id"`
	Method  *string   `json:"method"`
	Gateway *string   `json:"gateway"`
	Note    *string   `json:"note"`
}

// CaptureRequest body.
type CaptureRequest struct {
	ID uuid.UUID `json:"id"`
}

// RefundRequest body.
type RefundRequest struct {
	ID     uuid.UUID `json:"id"`
	Amount float64   `json:"amount"`
}

// IDRequest body.
type IDRequest struct {
	ID uuid.UUID `json:"id"`
}

// ListRequest body.
type ListRequest struct {
	pagination.PageRequest
}

// View public representation (card data masked).
type View struct {
	ID        string     `json:"id"`
	OrderID   string     `json:"order_id"`
	PaymentNo string     `json:"payment_no"`
	Amount    float64    `json:"amount"`
	Currency  string     `json:"currency"`
	Method    string     `json:"method"`
	Status    string     `json:"status"`
	Gateway   string     `json:"gateway"`
	CardLast4 string     `json:"card_last4"` // masked: ************1234
	PaidAt    *time.Time `json:"paid_at"`
	Note      string     `json:"note"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Summary aggregate data.
type Summary struct {
	Total       int64            `json:"total"`
	Pending     int64            `json:"pending"`
	Paid        int64            `json:"paid"`
	Failed      int64            `json:"failed"`
	Refunded    int64            `json:"refunded"`
	Collected   float64          `json:"collected"`
	Outstanding float64          `json:"outstanding"`
	ByMethod    map[string]int64 `json:"by_method"`
	RefundedSum float64          `json:"refunded_sum"`
}
