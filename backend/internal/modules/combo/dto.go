package combo

import (
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/google/uuid"
)

// ItemRequest links a service type into a combo with a quantity.
type ItemRequest struct {
	ServiceTypeID uuid.UUID `json:"service_type_id"`
	Quantity      int       `json:"quantity"`
}

// CreateRequest body.
type CreateRequest struct {
	Name         string        `json:"name"`
	Code         string        `json:"code"`
	Description  string        `json:"description"`
	Unit         string        `json:"unit"`
	Price        float64       `json:"price"`
	Currency     string        `json:"currency"`
	IsActive     *bool         `json:"is_active"`
	IsFeatured   *bool         `json:"is_featured"`
	ValidityDays *int          `json:"validity_days"` // 0 = no expiry
	RenewalType  *string       `json:"renewal_type"`  // none | auto | manual
	Items        []ItemRequest `json:"items"`
}

// UpdateRequest body (id in body).
type UpdateRequest struct {
	ID           uuid.UUID     `json:"id"`
	Code         *string       `json:"code"`
	Name         *string       `json:"name"`
	Description  *string       `json:"description"`
	Unit         *string       `json:"unit"`
	Price        *float64      `json:"price"`
	Currency     *string       `json:"currency"`
	IsActive     *bool         `json:"is_active"`
	IsFeatured   *bool         `json:"is_featured"`
	ValidityDays *int          `json:"validity_days"`
	RenewalType  *string       `json:"renewal_type"`
	Items        []ItemRequest `json:"items"`
}

// DeleteRequest body.
type DeleteRequest struct {
	ID uuid.UUID `json:"id"`
}

// ListRequest body.
type ListRequest struct {
	pagination.PageRequest
}

// ItemView public representation of a combo item, enriched with the service
// snapshot (name, unit, unit price) so clients can render exactly what a
// bundle includes without joining against the services catalog.
type ItemView struct {
	ServiceTypeID string  `json:"service_type_id"`
	Quantity      int     `json:"quantity"`
	ServiceName   string  `json:"service_name"`
	Unit          string  `json:"unit"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency"`
	Icon          string  `json:"icon"`
}

// View public representation.
type View struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Code         string     `json:"code"`
	Description  string     `json:"description"`
	Unit         string     `json:"unit"`
	Price        float64    `json:"price"`
	Currency     string     `json:"currency"`
	IsActive     bool       `json:"is_active"`
	IsFeatured   bool       `json:"is_featured"`
	ValidityDays int        `json:"validity_days"`
	RenewalType  string     `json:"renewal_type"`
	ValidityText string     `json:"validity_text"` // computed human-readable text
	Items        []ItemView `json:"items"`
	// RegularPrice is the à-la-carte value of the included items (qty × unit
	// price summed). Savings is what the bundle saves vs buying separately
	// (never negative). Both are computed server-side so pricing is consistent
	// everywhere the catalog is rendered.
	RegularPrice float64   `json:"regular_price"`
	Savings      float64   `json:"savings"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Summary aggregate data for the list endpoint.
type Summary struct {
	TotalActive   int64            `json:"total_active"`
	TotalInactive int64            `json:"total_inactive"`
	AvgPrice      float64          `json:"avg_price"`
	MaxPrice      float64          `json:"max_price"`
	MinPrice      float64          `json:"min_price"`
	ByUnit        map[string]int64 `json:"by_unit"`
}
