// Package pricing assembles the public pricing catalog: active service
// types, active combos and shop-wide defaults (currency, tax, business name).
// Everything a customer-facing page displays is sourced here — never from
// hardcoded client-side constants.
package pricing

import (
	"context"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/modules/combo"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/servicetype"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/shopconfig"
)

// catalogCacheTTL bounds how long a catalog is reused. Combo and service
// type edits invalidate it immediately (see InvalidateCatalog); the TTL is a
// safety net for anything that slips through.
const catalogCacheTTL = 60 * time.Second

const catalogCacheKey = "public-pricing-catalog"

// Cacher stores serialized catalog snapshots. Implemented by the cache
// package; tests substitute a stub or nil (cache bypass).
type Cacher interface {
	GetJSON(ctx context.Context, key string, v any) error
	SetJSON(ctx context.Context, key string, v any, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

// ActiveServiceReader supplies active service types. Implemented by the
// servicetype service; tests substitute a stub.
type ActiveServiceReader interface {
	ListActive(ctx context.Context) ([]servicetype.View, error)
}

// ActiveComboReader supplies active combos. Implemented by the combo service;
// tests substitute a stub.
type ActiveComboReader interface {
	ListActive(ctx context.Context) ([]combo.View, error)
}

// Service builds the public pricing catalog.
type Service struct {
	services ActiveServiceReader
	combos   ActiveComboReader
	shop     shopconfig.Provider
	cache    Cacher
}

// NewService wires the catalog readers. A nil cache disables caching.
func NewService(services ActiveServiceReader, combos ActiveComboReader, shop shopconfig.Provider, cache Cacher) *Service {
	return &Service{services: services, combos: combos, shop: shopconfig.Guard(shop), cache: cache}
}

// Public returns the catalog of sellable services and combos with the
// shop's money defaults, serving a short-lived cached snapshot when possible.
func (s *Service) Public(ctx context.Context) (*View, error) {
	if s.cache != nil {
		cached := &View{}
		if err := s.cache.GetJSON(ctx, catalogCacheKey, cached); err == nil && (len(cached.Services) > 0 || len(cached.Combos) > 0) {
			return cached, nil
		}
	}
	view, err := s.build(ctx)
	if err != nil {
		return nil, err
	}
	// Best-effort: a cache write failure must never fail the endpoint.
	if s.cache != nil {
		_ = s.cache.SetJSON(ctx, catalogCacheKey, view, catalogCacheTTL)
	}
	return view, nil
}

// InvalidateCatalog drops the cached catalog snapshot so the next Public
// call rebuilds from live data. Best-effort: a cache failure must never
// fail the caller's mutation. Satisfies catalog.Invalidator.
func (s *Service) InvalidateCatalog(ctx context.Context) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Del(ctx, catalogCacheKey)
}

// build assembles the catalog from the live readers.
func (s *Service) build(ctx context.Context) (*View, error) {
	services, err := s.services.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	combos, err := s.combos.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	defaults, err := s.shop.Defaults(ctx)
	if err != nil {
		return nil, err
	}
	return &View{
		BusinessName: defaults.BusinessName,
		Currency:     defaults.Currency,
		TaxRate:      defaults.TaxRate,
		Services:     services,
		Combos:       combos,
	}, nil
}
