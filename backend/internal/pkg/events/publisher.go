// Package events defines the outbound event publisher abstraction.
// NATS is optional: when unreachable the app degrades to a no-op publisher
// while events remain durable in the outbox table.
package events

import (
	"context"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/breaker"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/logger"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/retry"
	"github.com/nats-io/nats.go"
)

// Publisher publishes domain events to a broker.
type Publisher interface {
	// Publish sends an event payload to the subject.
	Publish(ctx context.Context, subject string, payload []byte) error
	// Available reports whether the broker is currently reachable.
	Available() bool
}

// Subject derives the NATS subject for an event type.
func Subject(eventType string) string {
	return "laundrypro.events." + eventType
}

// NoopPublisher is used when the broker is disabled/unreachable.
type NoopPublisher struct{}

// Publish always succeeds without side effects.
func (NoopPublisher) Publish(context.Context, string, []byte) error { return nil }

// Available always returns true (nothing to connect to).
func (NoopPublisher) Available() bool { return true }

// NATSPublisher publishes to NATS core with reconnects, a circuit breaker
// and bounded retry. A failed publish does not lose the event: the outbox
// relay retains it for a later attempt.
type NATSPublisher struct {
	nc      *nats.Conn
	breaker *breaker.Breaker
}

// NewNATSPublisher connects to NATS. Returns an error only when the URL is
// invalid; connectivity is retried in the background.
func NewNATSPublisher(url string) (*NATSPublisher, error) {
	nc, err := nats.Connect(url,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				logger.Warn("nats disconnected: %v", err)
			}
		}),
	)
	if err != nil {
		return nil, err
	}
	return &NATSPublisher{nc: nc, breaker: breaker.New("nats-publish")}, nil
}

// Publish sends an event through the breaker with retry.
func (p *NATSPublisher) Publish(ctx context.Context, subject string, payload []byte) error {
	_, err := p.breaker.Execute(ctx, func(ctx context.Context) (any, error) {
		err := retry.Do(ctx, func() error {
			if !p.nc.IsConnected() {
				return retry.ErrTransient
			}
			return p.nc.Publish(subject, payload)
		}, retry.WithMaxElapsed(3*time.Second))
		return nil, err
	})
	return err
}

// Available reports whether the connection is live.
func (p *NATSPublisher) Available() bool {
	return p.nc != nil && p.nc.IsConnected()
}

// Close drains and closes the connection.
func (p *NATSPublisher) Close() {
	if p.nc != nil {
		_ = p.nc.Drain()
		p.nc.Close()
	}
}
