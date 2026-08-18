package models

// ShopSetting holds the singleton shop-wide configuration (business
// identity, default currency and default tax rate). Exactly one row is
// maintained; the settings service lazily creates it on first access.
type ShopSetting struct {
	Base
	BusinessName  string  `gorm:"size:120;not null" json:"business_name"`
	BusinessPhone string  `gorm:"size:30" json:"business_phone"`
	BusinessEmail string  `gorm:"size:120" json:"business_email"`
	Address       string  `gorm:"size:500" json:"address"`
	City          string  `gorm:"size:80" json:"city"`
	Currency      string  `gorm:"size:8;not null;default:INR" json:"currency"`
	TaxRate       float64 `gorm:"type:decimal(5,2);not null;default:0" json:"tax_rate"` // default tax %
	TaxID         string  `gorm:"size:60" json:"tax_id"`
	InvoicePrefix string  `gorm:"size:12;not null;default:INV" json:"invoice_prefix"`
	ReceiptFooter string  `gorm:"size:500" json:"receipt_footer"`
	Timezone      string  `gorm:"size:40;not null;default:Asia/Kolkata" json:"timezone"`
}

// TableName overrides GORM's default table name.
func (ShopSetting) TableName() string { return "shop_settings" }
