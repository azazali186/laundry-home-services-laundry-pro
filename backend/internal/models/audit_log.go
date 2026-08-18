package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

// AuditLog captures before/after snapshots of mutated entities.
// Sensitive fields are redacted before snapshots are written.
type AuditLog struct {
	Base
	UserID    uuid.UUID       `gorm:"type:uuid;index" json:"user_id"`
	Entity    string          `gorm:"size:64;index" json:"entity"`
	EntityID  string          `gorm:"size:64;index" json:"entity_id"`
	Action    string          `gorm:"size:32;index" json:"action"` // create | update | delete | login | logout
	Method    string          `gorm:"size:16" json:"method"`
	Path      string          `gorm:"size:255" json:"path"`
	IP        string          `gorm:"size:64" json:"ip"`
	UserAgent string          `gorm:"size:512" json:"user_agent"`
	Before    json.RawMessage `gorm:"type:jsonb" json:"before"`
	After     json.RawMessage `gorm:"type:jsonb" json:"after"`
}

// TableName overrides GORM's default table name.
func (AuditLog) TableName() string { return "audit_logs" }
