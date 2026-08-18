package models

import "github.com/google/uuid"

// Combo is a priced bundle of services (e.g. a weekly pass or a family
// plan). The bundle price is set explicitly rather than derived, so
// discounts are first-class.
type Combo struct {
	Base
	Name        string  `gorm:"size:120;not null;uniqueIndex" json:"name"`
	Code        string  `gorm:"size:40;not null;uniqueIndex" json:"code"`
	Description string  `gorm:"size:500" json:"description"`
	Unit        string  `gorm:"size:24;not null;default:per_week" json:"unit"` // per_kg | per_week | per_month | flat
	Price       float64 `gorm:"type:decimal(12,2);not null;default:0" json:"price"`
	Currency    string  `gorm:"size:8;not null;default:INR" json:"currency"`
	// IsActive deliberately has no DB default: GORM would otherwise replace a
	// zero (false) value with the parsed default (true) at insert time, making
	// it impossible to create an inactive combo. The service always sets it.
	IsActive     bool        `gorm:"not null" json:"is_active"`
	IsFeatured   bool        `gorm:"not null;default:false" json:"is_featured"`
	ValidityDays int         `gorm:"not null;default:0" json:"validity_days"`           // 0 = no expiry
	RenewalType  string      `gorm:"size:16;not null;default:none" json:"renewal_type"` // none | auto | manual
	Items        []ComboItem `gorm:"constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

// TableName overrides GORM's default table name.
func (Combo) TableName() string { return "combos" }

// ComboItem links a service type into a combo with a quantity.
type ComboItem struct {
	Base
	ComboID       uuid.UUID `gorm:"type:uuid;index;not null" json:"combo_id"`
	ServiceTypeID uuid.UUID `gorm:"type:uuid;index;not null" json:"service_type_id"`
	Quantity      int       `gorm:"not null;default:1" json:"quantity"`
	// ServiceType is the referenced service, preloaded only for enriched read
	// views (item names, unit prices). Never serialized to API responses.
	ServiceType *ServiceType `gorm:"foreignKey:ServiceTypeID" json:"-"`
}

// TableName overrides GORM's default table name.
func (ComboItem) TableName() string { return "combo_items" }
