package notification

import (
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/google/uuid"
)

// CreateRequest body.
type CreateRequest struct {
	UserID  uuid.UUID `json:"user_id"`
	Title   string    `json:"title"`
	Body    string    `json:"body"`
	Type    string    `json:"type"`
	Channel string    `json:"channel"`
}

// MarkReadRequest body.
type MarkReadRequest struct {
	ID uuid.UUID `json:"id"`
}

// ListRequest body.
type ListRequest struct {
	pagination.PageRequest
}

// View public representation.
type View struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Type      string     `json:"type"`
	Channel   string     `json:"channel"`
	Status    string     `json:"status"`
	SentAt    *time.Time `json:"sent_at"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// Summary aggregate data.
type Summary struct {
	Total   int64 `json:"total"`
	Pending int64 `json:"pending"`
	Sent    int64 `json:"sent"`
	Failed  int64 `json:"failed"`
	Unread  int64 `json:"unread"`
}
