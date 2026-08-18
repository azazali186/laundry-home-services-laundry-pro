package models

import (
	"encoding/json"
	"time"
)

// Saga state statuses.
const (
	SagaRunning     = "running"
	SagaCompleted   = "completed"
	SagaFailed      = "failed"
	SagaCompensated = "compensated"
)

// SagaState persists saga execution state so flows survive restarts.
// Hot state is also cached in Redis under saga:<id>.
type SagaState struct {
	Base
	SagaID      string          `gorm:"size:64;not null;uniqueIndex" json:"saga_id"`
	Name        string          `gorm:"size:64;not null" json:"name"`
	AggregateID string          `gorm:"size:64;index" json:"aggregate_id"`
	Status      string          `gorm:"size:16;index;not null;default:running" json:"status"`
	Step        string          `gorm:"size:64" json:"step"`
	State       json.RawMessage `gorm:"type:jsonb" json:"state"`
	StartedAt   time.Time       `json:"started_at"`
	FinishedAt  *time.Time      `json:"finished_at"`
	LastError   string          `gorm:"size:512" json:"-"`
}

// TableName overrides GORM's default table name.
func (SagaState) TableName() string { return "saga_states" }
