package models

// InventoryItem tracks consumables (detergent, packaging, hangers, etc.).
type InventoryItem struct {
	Base
	Name           string  `gorm:"size:120;not null" json:"name"`
	SKU            string  `gorm:"size:64;not null;uniqueIndex" json:"sku"`
	Category       string  `gorm:"size:64;index" json:"category"`
	QuantityOnHand float64 `gorm:"type:decimal(12,2);not null;default:0" json:"quantity_on_hand"`
	ReorderLevel   float64 `gorm:"type:decimal(12,2);not null;default:0" json:"reorder_level"`
	Unit           string  `gorm:"size:16;not null;default:unit" json:"unit"`
	UnitCost       float64 `gorm:"type:decimal(12,2);not null;default:0" json:"unit_cost"`
	IsActive       bool    `gorm:"not null;default:true" json:"is_active"`
}

// TableName overrides GORM's default table name.
func (InventoryItem) TableName() string { return "inventory_items" }
