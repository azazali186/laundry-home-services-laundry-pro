package models

import "github.com/google/uuid"

// Rating captures customer feedback for a delivered order.
type Rating struct {
	Base
	OrderID    uuid.UUID `gorm:"type:uuid;index;not null" json:"order_id"`
	CustomerID uuid.UUID `gorm:"type:uuid;index" json:"customer_id"`
	Score      int       `gorm:"not null" json:"score"` // 1..5
	Comment    string    `gorm:"size:1000" json:"comment"`
	IsPublic   bool      `gorm:"not null;default:false" json:"is_public"`
}

// TableName overrides GORM's default table name.
func (Rating) TableName() string { return "ratings" }
