// Package breaker provides circuit breakers for outbound calls, backed by
// sony/gobreaker, with Prometheus metrics.
package breaker

import (
	"context"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/metrics"
	"github.com/sony/gobreaker"
)

// Callable is a function executed through the breaker.
type Callable func(ctx context.Context) (any, error)

// Breaker wraps gobreaker with metrics.
type Breaker struct {
	cb   *gobreaker.CircuitBreaker
	name string
}

// New creates a named circuit breaker.
func New(name string) *Breaker {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: 5,                // half-open probe limit
		Interval:    30 * time.Second, // closed-state reset window
		Timeout:     15 * time.Second, // open -> half-open
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(_ string, from, to gobreaker.State) {
			metrics.SetBreakerState(name, from.String(), to.String())
		},
	})
	return &Breaker{cb: cb, name: name}
}

// Execute runs fn through the circuit breaker.
func (b *Breaker) Execute(ctx context.Context, fn Callable) (any, error) {
	start := time.Now()
	result, err := b.cb.Execute(func() (any, error) {
		return fn(ctx)
	})
	metrics.ObserveBreakerCall(b.name, time.Since(start), err == nil)
	return result, err
}

// State returns the current breaker state name.
func (b *Breaker) State() string { return b.cb.State().String() }
