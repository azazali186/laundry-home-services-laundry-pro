package models

import (
	"time"

	"github.com/google/uuid"
)

// Notification statuses.
const (
	NotificationPending = "pending"
	NotificationSent    = "sent"
	NotificationFailed  = "failed"
)

// Notification is an outbound message (push/sms/email) for a user.
type Notification struct {
	Base
	UserID  uuid.UUID  `gorm:"type:uuid;index" json:"user_id"`
	Title   string     `gorm:"size:255" json:"title"`
	Body    string     `gorm:"size:2000" json:"body"`
	Type    string     `gorm:"size:32;index;not null;default:push" json:"type"` // push | sms | email
	Channel string     `gorm:"size:16;not null;default:push" json:"channel"`
	Status  string     `gorm:"size:16;index;not null;default:pending" json:"status"`
	SentAt  *time.Time `json:"sent_at"`
	ReadAt  *time.Time `json:"read_at"`
}

// TableName overrides GORM's default table name.
func (Notification) TableName() string { return "notifications" }
