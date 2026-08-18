package booking

import (
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/google/uuid"
)

// CreateRequest body.
type CreateRequest struct {
	CustomerID      uuid.UUID  `json:"customer_id"`
	ServiceTypeID   uuid.UUID  `json:"service_type_id"`
	PickupAddress   string     `json:"pickup_address"`
	DropoffAddress  string     `json:"dropoff_address"`
	PickupFrom      *time.Time `json:"pickup_from"`
	PickupTo        *time.Time `json:"pickup_to"`
	Notes           string     `json:"notes"`
	EstimatedAmount float64    `json:"estimated_amount"`
	Currency        string     `json:"currency"`
	Source          string     `json:"source"`
}

// UpdateRequest body.
type UpdateRequest struct {
	ID              uuid.UUID  `json:"id"`
	PickupAddress   *string    `json:"pickup_address"`
	DropoffAddress  *string    `json:"dropoff_address"`
	PickupFrom      *time.Time `json:"pickup_from"`
	PickupTo        *time.Time `json:"pickup_to"`
	Notes           *string    `json:"notes"`
	EstimatedAmount *float64   `json:"estimated_amount"`
}

// CancelRequest body (id in body).
type CancelRequest struct {
	ID     uuid.UUID `json:"id"`
	Reason string    `json:"reason"`
}

// DeleteRequest body.
type DeleteRequest struct {
	ID uuid.UUID `json:"id"`
}

// ListRequest body.
type ListRequest struct {
	pagination.PageRequest
}

// View public representation.
type View struct {
	ID              string     `json:"id"`
	BookingNo       string     `json:"booking_no"`
	CustomerID      string     `json:"customer_id"`
	ServiceTypeID   string     `json:"service_type_id"`
	PickupAddress   string     `json:"pickup_address"`
	DropoffAddress  string     `json:"dropoff_address"`
	PickupFrom      *time.Time `json:"pickup_from"`
	PickupTo        *time.Time `json:"pickup_to"`
	Status          string     `json:"status"`
	Notes           string     `json:"notes"`
	EstimatedAmount float64    `json:"estimated_amount"`
	Currency        string     `json:"currency"`
	Source          string     `json:"source"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Summary aggregate data.
type Summary struct {
	Total          int64            `json:"total"`
	Pending        int64            `json:"pending"`
	Confirmed      int64            `json:"confirmed"`
	Cancelled      int64            `json:"cancelled"`
	EstimatedValue float64          `json:"estimated_value"`
	ThisMonth      int64            `json:"this_month"`
	ByStatus       map[string]int64 `json:"by_status"`
}
