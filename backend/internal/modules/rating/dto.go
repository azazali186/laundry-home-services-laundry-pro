package rating

import (
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/google/uuid"
)

// CreateRequest body.
type CreateRequest struct {
	OrderID    uuid.UUID `json:"order_id"`
	CustomerID uuid.UUID `json:"customer_id"`
	Score      int       `json:"score"`
	Comment    string    `json:"comment"`
	IsPublic   bool      `json:"is_public"`
}

// UpdateRequest body.
type UpdateRequest struct {
	ID       uuid.UUID `json:"id"`
	Score    *int      `json:"score"`
	Comment  *string   `json:"comment"`
	IsPublic *bool     `json:"is_public"`
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
	ID         string    `json:"id"`
	OrderID    string    `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	Score      int       `json:"score"`
	Comment    string    `json:"comment"`
	IsPublic   bool      `json:"is_public"`
	CreatedAt  time.Time `json:"created_at"`
}

// Summary aggregate data.
type Summary struct {
	Total     int64   `json:"total"`
	AvgScore  float64 `json:"avg_score"`
	FiveStar  int64   `json:"five_star"`
	FourStar  int64   `json:"four_star"`
	ThreeStar int64   `json:"three_star"`
	TwoStar   int64   `json:"two_star"`
	OneStar   int64   `json:"one_star"`
}
