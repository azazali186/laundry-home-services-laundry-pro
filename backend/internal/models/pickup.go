package models

import (
	"time"

	"github.com/google/uuid"
)

// Pickup statuses.
const (
	PickupScheduled  = "scheduled"
	PickupInProgress = "in_progress"
	PickupCompleted  = "completed"
	PickupCancelled  = "cancelled"
)

// Pickup represents the physical collection of garments.
type Pickup struct {
	Base
	BookingID   uuid.UUID  `gorm:"type:uuid;index;not null" json:"booking_id"`
	AgentID     *uuid.UUID `gorm:"type:uuid;index" json:"agent_id"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Address     string     `gorm:"size:500" json:"address"`
	Status      string     `gorm:"size:32;index;not null;default:scheduled" json:"status"`
	Notes       string     `gorm:"size:1000" json:"notes"`
}

// TableName overrides GORM's default table name.
func (Pickup) TableName() string { return "pickups" }
