package garment

import (
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/google/uuid"
)

// CreateRequest body.
type CreateRequest struct {
	OrderID      uuid.UUID `json:"order_id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Color        string    `json:"color"`
	Fabric       string    `json:"fabric"`
	Quantity     int       `json:"quantity"`
	Instructions string    `json:"instructions"`
	DamageNotes  string    `json:"damage_notes"`
}

// UpdateRequest body.
type UpdateRequest struct {
	ID           uuid.UUID `json:"id"`
	Name         *string   `json:"name"`
	Type         *string   `json:"type"`
	Color        *string   `json:"color"`
	Fabric       *string   `json:"fabric"`
	Quantity     *int      `json:"quantity"`
	Status       *string   `json:"status"`
	Instructions *string   `json:"instructions"`
	DamageNotes  *string   `json:"damage_notes"`
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
	ID           string    `json:"id"`
	OrderID      string    `json:"order_id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Color        string    `json:"color"`
	Fabric       string    `json:"fabric"`
	Quantity     int       `json:"quantity"`
	Status       string    `json:"status"`
	Instructions string    `json:"instructions"`
	DamageNotes  string    `json:"damage_notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Summary aggregate data.
type Summary struct {
	TotalItems int64            `json:"total_items"`
	Received   int64            `json:"received"`
	Cleaned    int64            `json:"cleaned"`
	Ironed     int64            `json:"ironed"`
	Ready      int64            `json:"ready"`
	Damaged    int64            `json:"damaged"`
	ByStatus   map[string]int64 `json:"by_status"`
}
