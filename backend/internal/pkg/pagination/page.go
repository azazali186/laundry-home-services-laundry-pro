// Package pagination implements offset-based pagination with dynamic column
// sorting and whitelisted filters, all driven from the request body
// (no query parameters). Supports date range filters and search.
package pagination

import (
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
)

const (
	// DefaultLimit is applied when no limit is provided.
	DefaultLimit = 20
	// MaxLimit caps the page size.
	MaxLimit = 100
)

// PageRequest is embedded in every list request DTO.
type PageRequest struct {
	Offset    int               `json:"offset"`
	Limit     int               `json:"limit"`
	SortBy    string            `json:"sort_by"`
	SortOrder string            `json:"sort_order"`
	Search    string            `json:"search"`
	Filters   map[string]string `json:"filters"`
}

// Normalize fills defaults and validates the request.
func (p *PageRequest) Normalize() error {
	if p.Offset < 0 {
		p.Offset = 0
	}
	if p.Limit <= 0 {
		p.Limit = DefaultLimit
	}
	if p.Limit > MaxLimit {
		p.Limit = MaxLimit
	}
	if p.SortBy == "" {
		p.SortBy = "created_at"
	}
	if p.SortOrder == "" {
		p.SortOrder = "desc"
	}
	if p.SortOrder != "asc" && p.SortOrder != "desc" {
		return errors.New(400, "sort_order must be asc or desc")
	}
	return nil
}

// Operator prefixes supported in filter values.
const (
	OpEq    = "eq:"   // exact match (default)
	OpGt    = "gt:"   // greater than
	OpGte   = "gte:"  // greater than or equal
	OpLt    = "lt:"   // less than
	OpLte   = "lte:"  // less than or equal
	OpLike  = "like:" // substring match
	OpRange = "_range"
)

// filterPrefixes lists recognized operator prefixes.
var filterPrefixes = []string{OpLike, OpGte, OpGt, OpLte, OpLt, OpEq}
