package inventory

import (
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/google/uuid"
)

// CreateRequest body.
type CreateRequest struct {
	Name           string  `json:"name"`
	SKU            string  `json:"sku"`
	Category       string  `json:"category"`
	QuantityOnHand float64 `json:"quantity_on_hand"`
	ReorderLevel   float64 `json:"reorder_level"`
	Unit           string  `json:"unit"`
	UnitCost       float64 `json:"unit_cost"`
	IsActive       *bool   `json:"is_active"`
}

// UpdateRequest body.
type UpdateRequest struct {
	ID           uuid.UUID `json:"id"`
	Name         *string   `json:"name"`
	Category     *string   `json:"category"`
	ReorderLevel *float64  `json:"reorder_level"`
	Unit         *string   `json:"unit"`
	UnitCost     *float64  `json:"unit_cost"`
	IsActive     *bool     `json:"is_active"`
}

// AdjustRequest body: signed delta, e.g. +5 restock / -2 consumption.
type AdjustRequest struct {
	ID     uuid.UUID `json:"id"`
	Delta  float64   `json:"delta"`
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
	Name           string    `json:"name"`
	SKU            string    `json:"sku"`
	Category       string    `json:"category"`
	QuantityOnHand float64   `json:"quantity_on_hand"`
	ReorderLevel   float64   `json:"reorder_level"`
	Unit           string    `json:"unit"`
	UnitCost       float64   `json:"unit_cost"`
	IsActive       bool      `json:"is_active"`
	LowStock       bool      `json:"low_stock"`
	StockValue     float64   `json:"stock_value"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Summary aggregate data.
type Summary struct {
	TotalItems  int64            `json:"total_items"`
	LowStock    int64            `json:"low_stock"`
	OutOfStock  int64            `json:"out_of_stock"`
	TotalValue  float64          `json:"total_value"`
	ActiveItems int64            `json:"active_items"`
	ByCategory  map[string]int64 `json:"by_category"`
}
