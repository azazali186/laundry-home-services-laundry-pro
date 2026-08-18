package models

import (
	"time"

	"github.com/google/uuid"
)

// Delivery statuses.
const (
	DeliveryScheduled = "scheduled"
	DeliveryInTransit = "in_transit"
	DeliveryDelivered = "delivered"
	DeliveryCancelled = "cancelled"
)

// Delivery represents the physical handover of finished orders.
type Delivery struct {
	Base
	OrderID     uuid.UUID  `gorm:"type:uuid;index;not null" json:"order_id"`
	AgentID     *uuid.UUID `gorm:"type:uuid;index" json:"agent_id"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	DeliveredAt *time.Time `json:"delivered_at"`
	Address     string     `gorm:"size:500" json:"address"`
	Status      string     `gorm:"size:32;index;not null;default:scheduled" json:"status"`
	Notes       string     `gorm:"size:1000" json:"notes"`
}

// TableName overrides GORM's default table name.
func (Delivery) TableName() string { return "deliveries" }
