package models

import (
	"time"

	"github.com/google/uuid"
)

// Payment statuses.
const (
	PaymentPending  = "pending"
	PaymentPaid     = "paid"
	PaymentFailed   = "failed"
	PaymentRefunded = "refunded"
)

// Payment captures money movement for an order.
// TransactionID and CardLast4 are encrypted at rest by the payment service.
type Payment struct {
	Base
	OrderID       uuid.UUID  `gorm:"type:uuid;index;not null" json:"order_id"`
	PaymentNo     string     `gorm:"size:40;not null;uniqueIndex" json:"payment_no"`
	Amount        float64    `gorm:"type:decimal(12,2);not null" json:"amount"`
	Currency      string     `gorm:"size:8;not null;default:INR" json:"currency"`
	Method        string     `gorm:"size:24;not null;default:cash" json:"method"` // cash | card | upi | wallet
	Status        string     `gorm:"size:32;index;not null;default:pending" json:"status"`
	Gateway       string     `gorm:"size:64" json:"gateway"`
	TransactionID string     `gorm:"size:512" json:"-"` // encrypted at rest
	CardLast4     string     `gorm:"size:512" json:"-"` // encrypted at rest
	PaidAt        *time.Time `json:"paid_at"`
	Note          string     `gorm:"size:500" json:"note"`
}

// TableName overrides GORM's default table name.
func (Payment) TableName() string { return "payments" }
