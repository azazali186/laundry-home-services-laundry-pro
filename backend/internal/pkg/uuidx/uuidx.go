// Package uuidx provides UUID v7 helpers used for all primary keys.
package uuidx

import (
	"strings"

	"github.com/google/uuid"
)

// New returns a new UUID v7. Panics only when the runtime is broken
// (astronomically unlikely); callers that care can use NewErr.
func New() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return id
}

// NewErr returns a new UUID v7 with an error.
func NewErr() (uuid.UUID, error) { return uuid.NewV7() }

// Parse parses a UUID string, also accepting short/uppercase forms.
func Parse(s string) (uuid.UUID, error) {
	return uuid.Parse(strings.ToLower(strings.TrimSpace(s)))
}

// String renders the uuid as lowercase string.
func String(id uuid.UUID) string { return id.String() }
