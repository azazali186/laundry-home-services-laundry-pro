package models

import (
	"time"

	"github.com/google/uuid"
)

// Booking statuses.
const (
	BookingPending   = "pending"
	BookingConfirmed = "confirmed"
	BookingCancelled = "cancelled"
)

// Booking captures a customer request for pickup/dropoff of laundry.
type Booking struct {
	Base
	BookingNo       string     `gorm:"size:40;not null;uniqueIndex" json:"booking_no"`
	CustomerID      uuid.UUID  `gorm:"type:uuid;index" json:"customer_id"`
	ServiceTypeID   uuid.UUID  `gorm:"type:uuid;index" json:"service_type_id"`
	PickupAddress   string     `gorm:"size:500" json:"pickup_address"`
	DropoffAddress  string     `gorm:"size:500" json:"dropoff_address"`
	PickupFrom      *time.Time `json:"pickup_from"`
	PickupTo        *time.Time `json:"pickup_to"`
	Status          string     `gorm:"size:32;index;not null;default:pending" json:"status"`
	Notes           string     `gorm:"size:1000" json:"notes"`
	EstimatedAmount float64    `gorm:"type:decimal(12,2);not null;default:0" json:"estimated_amount"`
	Currency        string     `gorm:"size:8;not null;default:INR" json:"currency"`
	Source          string     `gorm:"size:32;not null;default:web" json:"source"`
}

// TableName overrides GORM's default table name.
func (Booking) TableName() string { return "bookings" }
