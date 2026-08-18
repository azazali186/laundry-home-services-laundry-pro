// Package audit records before/after snapshots of entity mutations.
// Sensitive fields (passwords, tokens, card data) are redacted.
package audit

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// sensitiveKeys are JSON field names redacted from snapshots.
var sensitiveKeys = map[string]bool{
	"password": true, "token": true, "authorization": true,
	"transaction_id": true, "card_last4": true, "cvv": true, "secret": true,
}

// Redactor strips sensitive keys from a JSON payload.
func Redactor(data []byte) json.RawMessage {
	if len(data) == 0 {
		return nil
	}
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		return data
	}
	for k := range v {
		if sensitiveKeys[strings.ToLower(k)] {
			v[k] = "[redacted]"
		}
	}
	out, _ := json.Marshal(v)
	return out
}

// Writer persists audit log entries asynchronously (fire and forget).
type Writer struct {
	db *gorm.DB
}

// NewWriter builds the audit writer.
func NewWriter(db *gorm.DB) *Writer { return &Writer{db: db} }

// Meta carries request context for an audit entry.
type Meta struct {
	UserID    uuid.UUID
	IP        string
	UserAgent string
	Method    string
	Path      string
}

// MetaFrom extracts audit meta from a hertz request context.
func MetaFrom(c *app.RequestContext) Meta {
	meta := Meta{IP: c.ClientIP(), UserAgent: string(c.UserAgent()), Method: string(c.Method()), Path: string(c.Path())}
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(uuid.UUID); ok {
			meta.UserID = id
		}
	}
	return meta
}

// Before captures the pre-mutation snapshot of an entity.
func Before(entity any) json.RawMessage {
	return Redactor(mustJSON(entity))
}

// Log writes an audit entry in its own transaction.
func (w *Writer) Log(ctx context.Context, m Meta, entity, entityID, action string, before, after any) {
	row := models.AuditLog{
		UserID:    m.UserID,
		Entity:    entity,
		EntityID:  entityID,
		Action:    action,
		Method:    m.Method,
		Path:      m.Path,
		IP:        m.IP,
		UserAgent: m.UserAgent,
		Before:    Before(before),
		After:     Redactor(mustJSON(after)),
	}
	if err := w.db.WithContext(ctx).Create(&row).Error; err != nil {
		return // audit must never break the main flow
	}
}

// DiffBefore returns the redacted snapshot of the entity before mutation.
func DiffBefore(v any) json.RawMessage { return Before(v) }

// ChangedFields returns the top-level keys whose values differ between
// the before and after snapshots (excluding soft-delete bookkeeping).
func ChangedFields(before, after any) []string {
	b := mustMap(before)
	a := mustMap(after)
	var changed []string
	for k, av := range a {
		if sensitiveKeys[strings.ToLower(k)] {
			continue
		}
		if bv, ok := b[k]; !ok || !reflect.DeepEqual(bv, av) {
			changed = append(changed, k)
		}
	}
	return changed
}

func mustJSON(v any) []byte {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

func mustMap(v any) map[string]any {
	out := map[string]any{}
	if v == nil {
		return out
	}
	_ = json.Unmarshal(mustJSON(v), &out)
	return out
}

// Now is exposed for tests to control time.
var Now = time.Now
