// Package models defines all GORM entities. Every entity embeds Base which
// provides a UUID v7 primary key, timestamps and soft delete.
package models

import (
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/uuidx"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base is embedded by every entity.
type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate assigns a UUID v7 when absent.
func (b *Base) BeforeCreate(_ *gorm.DB) error {
	if uuid.Nil == b.ID {
		id, err := uuidx.NewErr()
		if err != nil {
			return err
		}
		b.ID = id
	}
	return nil
}
