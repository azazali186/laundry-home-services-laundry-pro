// Package saga implements a pragmatic event-driven saga engine: flows are
// driven by outbox events dispatched through a registry. State is persisted
// in the saga_states table and hot-cached in Redis.
package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/logger"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/metrics"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/retry"
)

// Event is a domain event being dispatched to saga handlers.
type Event struct {
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       json.RawMessage
}

// Decode unmarshals the event payload into v.
func (e Event) Decode(v any) error {
	if len(e.Payload) == 0 {
		return nil
	}
	return json.Unmarshal(e.Payload, v)
}

// Handler processes one event type.
type Handler func(ctx context.Context, ev Event) error

// Registry maps event types to handlers with panic recovery and retry.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewRegistry builds an empty registry.
func NewRegistry() *Registry {
	return &Registry{handlers: map[string]Handler{}}
}

// Register binds a handler to an event type.
func (r *Registry) Register(eventType string, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[eventType] = h
}

// Has reports whether a handler exists for the event type.
func (r *Registry) Has(eventType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.handlers[eventType]
	return ok
}

// Dispatch runs the handler for ev with retry and panic recovery.
// Unknown events are treated as success (they may target other services).
func (r *Registry) Dispatch(ctx context.Context, ev Event) (err error) {
	h, ok := r.handlers[ev.EventType]
	if !ok {
		return nil
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("saga handler panic: %v", recovered)
			metrics.ObserveOutbox("panic")
		}
	}()
	err = retry.Do(ctx, func() error {
		if herr := h(ctx, ev); herr != nil {
			logger.Warn("saga handler %s failed: %v", ev.EventType, herr)
			return retry.ErrTransient
		}
		return nil
	}, retry.Default())
	return err
}
