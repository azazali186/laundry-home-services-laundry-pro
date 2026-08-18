package models

// Role groups permissions for RBAC.
type Role struct {
	Base
	Name        string       `gorm:"size:80;not null;uniqueIndex" json:"name"`
	Code        string       `gorm:"size:80;not null;uniqueIndex" json:"code"`
	Description string       `gorm:"size:255" json:"description"`
	IsSystem    bool         `gorm:"not null;default:false" json:"is_system"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

// TableName overrides GORM's default table name.
func (Role) TableName() string { return "roles" }
