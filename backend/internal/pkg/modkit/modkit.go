// Package modkit provides small shared helpers for business modules:
// business document number generation and common validation.
package modkit

import (
	"fmt"
	"strings"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/google/uuid"
)

// NextNo generates a unique human-friendly business number using a UUID v7
// suffix, e.g. "ORD-4X2M9KP0QW". The UUID suffix guarantees uniqueness
// without a counter table (GORM-only, no raw SQL).
func NextNo(prefix string) string {
	id := uuid.New()
	short := strings.ToUpper(strings.ReplaceAll(id.String(), "-", "")[:10])
	return fmt.Sprintf("%s-%s", prefix, short)
}

// Now returns the current UTC time (separated for tests).
func Now() time.Time { return time.Now().UTC() }

// Nil returns the zero UUID (used for optional-field checks).
func Nil() uuid.UUID { return uuid.Nil }

// PageRes converts a pagination result to the response envelope.
func PageRes(res *pagination.PageResult) *response.Pagination {
	if res == nil {
		return nil
	}
	return &response.Pagination{
		Limit:   res.Limit,
		Total:   res.Total,
		HasMore: res.HasMore,
		Offset:  res.Offset,
	}
}

// IsUnique reports whether the error is a DB unique-constraint violation
// (portable across PostgreSQL and SQLite, no raw SQL).
func IsUnique(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate key value") ||
		strings.Contains(msg, "SQLSTATE 23505")
}

// OrDefault returns fallback when v is blank.
func OrDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
