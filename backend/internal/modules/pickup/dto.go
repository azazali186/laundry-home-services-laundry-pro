package pickup

import (
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/google/uuid"
)

// CreateRequest body.
type CreateRequest struct {
	BookingID   uuid.UUID  `json:"booking_id"`
	AgentID     *uuid.UUID `json:"agent_id"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	Address     string     `json:"address"`
	Notes       string     `json:"notes"`
}

// UpdateRequest body.
type UpdateRequest struct {
	ID          uuid.UUID  `json:"id"`
	AgentID     *uuid.UUID `json:"agent_id"`
	ScheduledAt *time.Time `json:"scheduled_at"`
	Address     *string    `json:"address"`
	Notes       *string    `json:"notes"`
}

// IDRequest body used by complete/delete.
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
	BookingID   string     `json:"booking_id"`
	AgentID     string     `json:"agent_id"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Address     string     `json:"address"`
	Status      string     `json:"status"`
	Notes       string     `json:"notes"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Summary aggregate data.
type Summary struct {
	Total     int64 `json:"total"`
	Scheduled int64 `json:"scheduled"`
	Completed int64 `json:"completed"`
	Active    int64 `json:"active"`
}
