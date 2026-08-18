package models

import (
	"encoding/json"
	"time"
)

// Outbox event statuses.
const (
	OutboxPending = "pending"
	OutboxSent    = "sent"
	OutboxFailed  = "failed"
)

// OutboxEvent is the transactional outbox: business events written in the
// same DB transaction as the mutation, then relayed to NATS by a worker.
type OutboxEvent struct {
	Base
	AggregateType string          `gorm:"size:64;index" json:"aggregate_type"`
	AggregateID   string          `gorm:"size:64;index" json:"aggregate_id"`
	EventType     string          `gorm:"size:128;index;not null" json:"event_type"`
	Payload       json.RawMessage `gorm:"type:jsonb" json:"payload"`
	Status        string          `gorm:"size:16;index;not null;default:pending" json:"status"`
	RetryCount    int             `gorm:"not null;default:0" json:"retry_count"`
	AvailableAt   *time.Time      `json:"available_at"`
	SentAt        *time.Time      `json:"sent_at"`
	LastError     string          `gorm:"size:512" json:"-"`
}

// TableName overrides GORM's default table name.
func (OutboxEvent) TableName() string { return "outbox_events" }
