package models

// Permission maps an API route+method to an RBAC grantable unit.
type Permission struct {
	Base
	Name    string `gorm:"size:160;not null" json:"name"`
	Route   string `gorm:"size:255;not null;uniqueIndex" json:"route"` // "POST /api/v1/orders"
	Path    string `gorm:"size:255;not null" json:"path"`
	Method  string `gorm:"size:16;not null" json:"method"`
	Service string `gorm:"size:64;not null;default:api-gateway" json:"service"`
	Guard   string `gorm:"size:32;not null;default:API" json:"guard"`
}

// TableName overrides GORM's default table name.
func (Permission) TableName() string { return "permissions" }
