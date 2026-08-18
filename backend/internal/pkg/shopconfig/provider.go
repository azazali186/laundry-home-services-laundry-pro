// Package shopconfig defines the contract that money-flow modules use to
// source shop-wide defaults (currency, default tax, invoice prefix) from the
// singleton settings row instead of hardcoding them.
package shopconfig

import "context"

// Defaults are the shop-wide defaults applied to new money flows when the
// request does not override them. BusinessName is surfaced by the public
// pricing catalog.
type Defaults struct {
	BusinessName  string
	Currency      string
	TaxRate       float64
	InvoicePrefix string
	ReceiptFooter string
}

// Provider supplies shop configuration defaults. Implemented by the settings
// module; tests substitute a stub.
type Provider interface {
	Defaults(ctx context.Context) (Defaults, error)
}

// Static is a nil-safe provider that returns fixed defaults without touching
// the database. Used as a fallback when no settings provider is wired (tests,
// partial embeddings) so money-flow modules never nil-dereference.
type Static Defaults

// Defaults returns the static defaults.
func (s Static) Defaults(context.Context) (Defaults, error) { return Defaults(s), nil }

// Guard returns the provider, substituting an empty Static provider for nil so
// callers can safely invoke Defaults.
func Guard(p Provider) Provider {
	if p == nil {
		return Static{}
	}
	return p
}
