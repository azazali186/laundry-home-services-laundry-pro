package models

import "time"

// User represents an operator, agent or customer account.
type User struct {
	Base
	Name        string     `gorm:"size:120;not null" json:"name"`
	Email       string     `gorm:"size:255;not null;uniqueIndex" json:"email"`
	Phone       string     `gorm:"size:64" json:"phone"` // encrypted at rest
	Password    string     `gorm:"size:255;not null" json:"-"`
	IsActive    bool       `gorm:"not null;default:true" json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at"`
	Roles       []Role     `gorm:"many2many:user_roles;" json:"roles,omitempty"`
}

// TableName overrides GORM's default table name.
func (User) TableName() string { return "users" }
