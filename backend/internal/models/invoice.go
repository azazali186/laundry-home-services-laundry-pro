package models

import (
	"time"

	"github.com/google/uuid"
)

// Invoice statuses.
const (
	InvoiceDraft  = "draft"
	InvoiceIssued = "issued"
	InvoicePaid   = "paid"
	InvoiceVoid   = "void"
)

// Invoice is the billable document for an order.
type Invoice struct {
	Base
	OrderID     uuid.UUID  `gorm:"type:uuid;index;not null" json:"order_id"`
	InvoiceNo   string     `gorm:"size:40;not null;uniqueIndex" json:"invoice_no"`
	IssueDate   time.Time  `json:"issue_date"`
	DueDate     *time.Time `json:"due_date"`
	Subtotal    float64    `gorm:"type:decimal(12,2);not null;default:0" json:"subtotal"`
	TaxAmount   float64    `gorm:"type:decimal(12,2);not null;default:0" json:"tax_amount"`
	Discount    float64    `gorm:"type:decimal(12,2);not null;default:0" json:"discount"`
	Total       float64    `gorm:"type:decimal(12,2);not null;default:0" json:"total"`
	Status      string     `gorm:"size:32;index;not null;default:draft" json:"status"`
	PaidAt      *time.Time `json:"paid_at"`
	BillingName string     `gorm:"size:160" json:"billing_name"`
}

// TableName overrides GORM's default table name.
func (Invoice) TableName() string { return "invoices" }
