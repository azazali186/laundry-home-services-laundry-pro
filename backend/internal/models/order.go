package models

import "github.com/google/uuid"

// Order statuses (processing pipeline).
const (
	OrderCreated        = "created"
	OrderProcessing     = "processing"
	OrderQualityCheck   = "quality_check"
	OrderReady          = "ready"
	OrderOutForDelivery = "out_for_delivery"
	OrderDelivered      = "delivered"
	OrderCancelled      = "cancelled"
)

// Order is the central work unit of the laundry pipeline.
type Order struct {
	Base
	OrderNo        string     `gorm:"size:40;not null;uniqueIndex" json:"order_no"`
	BookingID      uuid.UUID  `gorm:"type:uuid;index" json:"booking_id"`
	CustomerID     uuid.UUID  `gorm:"type:uuid;index" json:"customer_id"`
	ServiceTypeID  uuid.UUID  `gorm:"type:uuid;index" json:"service_type_id"`
	PickupID       *uuid.UUID `gorm:"type:uuid;index" json:"pickup_id"`
	Status         string     `gorm:"size:32;index;not null;default:created" json:"status"`
	Priority       string     `gorm:"size:16;not null;default:normal" json:"priority"`
	ItemCount      int        `gorm:"not null;default:0" json:"item_count"`
	TotalAmount    float64    `gorm:"type:decimal(12,2);not null;default:0" json:"total_amount"`
	TaxAmount      float64    `gorm:"type:decimal(12,2);not null;default:0" json:"tax_amount"`
	DiscountAmount float64    `gorm:"type:decimal(12,2);not null;default:0" json:"discount_amount"`
	GrandTotal     float64    `gorm:"type:decimal(12,2);not null;default:0" json:"grand_total"`
	Currency       string     `gorm:"size:8;not null;default:INR" json:"currency"`
	Note           string     `gorm:"size:1000" json:"note"`
}

// TableName overrides GORM's default table name.
func (Order) TableName() string { return "orders" }
