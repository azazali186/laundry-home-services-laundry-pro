package servicetype

import (
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/google/uuid"
)

// CreateRequest body.
type CreateRequest struct {
	Name          string  `json:"name"`
	Code          string  `json:"code"`
	Description   string  `json:"description"`
	Unit          string  `json:"unit"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency"`
	TaxRate       float64 `json:"tax_rate"`
	EstimatedDays int     `json:"estimated_days"`
	IsActive      *bool   `json:"is_active"`
	Icon          string  `json:"icon"`
}

// UpdateRequest body (id in body).
type UpdateRequest struct {
	ID            uuid.UUID `json:"id"`
	Name          *string   `json:"name"`
	Description   *string   `json:"description"`
	Unit          *string   `json:"unit"`
	Price         *float64  `json:"price"`
	Currency      *string   `json:"currency"`
	TaxRate       *float64  `json:"tax_rate"`
	EstimatedDays *int      `json:"estimated_days"`
	IsActive      *bool     `json:"is_active"`
	Icon          *string   `json:"icon"`
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
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Code          string    `json:"code"`
	Description   string    `json:"description"`
	Unit          string    `json:"unit"`
	Price         float64   `json:"price"`
	Currency      string    `json:"currency"`
	TaxRate       float64   `json:"tax_rate"`
	EstimatedDays int       `json:"estimated_days"`
	IsActive      bool      `json:"is_active"`
	Icon          string    `json:"icon"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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
