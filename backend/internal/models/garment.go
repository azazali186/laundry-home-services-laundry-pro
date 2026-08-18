package models

import "github.com/google/uuid"

// Garment statuses.
const (
	GarmentReceived = "received"
	GarmentCleaned  = "cleaned"
	GarmentIroned   = "ironed"
	GarmentReady    = "ready"
)

// Garment tracks an individual item within an order.
type Garment struct {
	Base
	OrderID      uuid.UUID `gorm:"type:uuid;index;not null" json:"order_id"`
	Name         string    `gorm:"size:120;not null" json:"name"`
	Type         string    `gorm:"size:64" json:"type"`
	Color        string    `gorm:"size:32" json:"color"`
	Fabric       string    `gorm:"size:32" json:"fabric"`
	Quantity     int       `gorm:"not null;default:1" json:"quantity"`
	Status       string    `gorm:"size:32;index;not null;default:received" json:"status"`
	Instructions string    `gorm:"size:500" json:"instructions"`
	DamageNotes  string    `gorm:"size:500" json:"damage_notes"`
}

// TableName overrides GORM's default table name.
func (Garment) TableName() string { return "garments" }
