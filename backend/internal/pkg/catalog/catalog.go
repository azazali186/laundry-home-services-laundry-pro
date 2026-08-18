// Package catalog defines the contract for invalidating the public pricing
// catalog cache. It lives in pkg so the combo and servicetype modules can
// depend on it without importing pricing (which consumes them).
package catalog

import "context"

// Invalidator clears the cached public pricing catalog. Implemented by the
// pricing service; invoked by combo and servicetype services after any
// mutation so admin edits surface on the landing page immediately instead
// of waiting for the catalog cache TTL.
type Invalidator interface {
	InvalidateCatalog(ctx context.Context)
}
