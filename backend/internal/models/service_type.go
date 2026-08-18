package models

// ServiceType defines a priced laundry service (e.g. wash & fold, dry clean).
type ServiceType struct {
	Base
	Name          string  `gorm:"size:120;not null;uniqueIndex" json:"name"`
	Code          string  `gorm:"size:40;not null;uniqueIndex" json:"code"`
	Description   string  `gorm:"size:500" json:"description"`
	Unit          string  `gorm:"size:16;not null;default:kg" json:"unit"` // kg | piece
	Price         float64 `gorm:"type:decimal(12,2);not null;default:0" json:"price"`
	Currency      string  `gorm:"size:8;not null;default:INR" json:"currency"`
	TaxRate       float64 `gorm:"type:decimal(5,2);not null;default:0" json:"tax_rate"`
	EstimatedDays int     `gorm:"not null;default:2" json:"estimated_days"`
	// IsActive deliberately has no DB default: GORM would otherwise replace a
	// zero (false) value with the parsed default (true) at insert time, making
	// it impossible to create an inactive row. The service always sets it.
	IsActive bool   `gorm:"not null" json:"is_active"`
	Icon     string `gorm:"size:255" json:"icon"`
}

// TableName overrides GORM's default table name.
func (ServiceType) TableName() string { return "service_types" }
