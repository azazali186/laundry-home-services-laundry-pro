package settings

import "time"

// UpdateRequest patches the singleton shop configuration. Every field is a
// pointer so partial updates can clear optional values.
type UpdateRequest struct {
	BusinessName  *string  `json:"business_name"`
	BusinessPhone *string  `json:"business_phone"`
	BusinessEmail *string  `json:"business_email"`
	Address       *string  `json:"address"`
	City          *string  `json:"city"`
	Currency      *string  `json:"currency"`
	TaxRate       *float64 `json:"tax_rate"`
	TaxID         *string  `json:"tax_id"`
	InvoicePrefix *string  `json:"invoice_prefix"`
	ReceiptFooter *string  `json:"receipt_footer"`
	Timezone      *string  `json:"timezone"`
}

// View is the public representation of the shop configuration.
type View struct {
	ID            string    `json:"id"`
	BusinessName  string    `json:"business_name"`
	BusinessPhone string    `json:"business_phone"`
	BusinessEmail string    `json:"business_email"`
	Address       string    `json:"address"`
	City          string    `json:"city"`
	Currency      string    `json:"currency"`
	TaxRate       float64   `json:"tax_rate"`
	TaxID         string    `json:"tax_id"`
	InvoicePrefix string    `json:"invoice_prefix"`
	ReceiptFooter string    `json:"receipt_footer"`
	Timezone      string    `json:"timezone"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
