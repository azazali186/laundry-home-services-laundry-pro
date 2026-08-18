// Package outbox implements the transactional outbox: events are persisted
// atomically with the business mutation and relayed by a background worker.
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"gorm.io/gorm"
)

// Event is a domain event awaiting relay.
type Event struct {
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       any
}

// Repo persists outbox events via GORM (no raw SQL).
type Repo struct {
	db       *gorm.DB
	maxRetry int
}

// NewRepo builds an outbox repository.
func NewRepo(db *gorm.DB, maxRetry int) *Repo {
	return &Repo{db: db, maxRetry: maxRetry}
}

// Enqueue inserts an event inside the caller's transaction.
func (r *Repo) Enqueue(ctx context.Context, tx *gorm.DB, ev Event) error {
	payload, err := json.Marshal(ev.Payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}
	now := time.Now()
	row := models.OutboxEvent{
		AggregateType: ev.AggregateType,
		AggregateID:   ev.AggregateID,
		EventType:     ev.EventType,
		Payload:       payload,
		Status:        models.OutboxPending,
		AvailableAt:   &now,
	}
	return tx.WithContext(ctx).Create(&row).Error
}

// Pending claims up to limit events that are ready for processing.
func (r *Repo) Pending(ctx context.Context, limit int) ([]models.OutboxEvent, error) {
	var events []models.OutboxEvent
	err := r.db.WithContext(ctx).
		Where("status = ? AND (available_at IS NULL OR available_at <= ?)", models.OutboxPending, time.Now()).
		Order("created_at asc").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// MarkSent records a successful relay.
func (r *Repo) MarkSent(ctx context.Context, eventID any) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.OutboxEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]any{"status": models.OutboxSent, "sent_at": &now, "last_error": ""}).Error
}

// MarkFailed records a failure. Retryable failures are rescheduled with
// exponential backoff; permanent exhaustion marks the event failed.
func (r *Repo) MarkFailed(ctx context.Context, event models.OutboxEvent, processErr error) error {
	nextRetry := event.RetryCount + 1
	updates := map[string]any{"last_error": processErr.Error()}

	if nextRetry <= r.maxRetry {
		delay := time.Duration(math.Pow(2, float64(nextRetry))) * time.Second
		if delay > time.Hour {
			delay = time.Hour
		}
		available := time.Now().Add(delay)
		updates["retry_count"] = nextRetry
		updates["available_at"] = &available
	} else {
		updates["status"] = models.OutboxFailed
	}
	return r.db.WithContext(ctx).Model(&models.OutboxEvent{}).Where("id = ?", event.ID).Updates(updates).Error
}

// ErrNoEvent is returned when an event id is unknown.
var ErrNoEvent = errors.New("outbox event not found")
