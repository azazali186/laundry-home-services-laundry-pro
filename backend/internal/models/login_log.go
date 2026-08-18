package models

import "github.com/google/uuid"

// LoginLog records authentication attempts. Never stores credentials.
type LoginLog struct {
	Base
	UserID    uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	Email     string    `gorm:"size:255;index" json:"email"`
	IP        string    `gorm:"size:64" json:"ip"`
	UserAgent string    `gorm:"size:512" json:"user_agent"`
	Status    string    `gorm:"size:16;index;not null" json:"status"` // success | failed
	Reason    string    `gorm:"size:255" json:"reason"`
}

// TableName overrides GORM's default table name.
func (LoginLog) TableName() string { return "login_logs" }
