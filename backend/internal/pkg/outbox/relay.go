package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/events"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/lock"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/logger"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/metrics"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/saga"
)

// RelayOptions configures the outbox relay worker.
type RelayOptions struct {
	PollInterval time.Duration
	BatchSize    int
	MaxRetries   int
}

// Relay drains the outbox: for each pending event it dispatches in-process
// saga handlers and publishes to the broker, then marks the event sent.
type Relay struct {
	repo      *Repo
	registry  *saga.Registry
	publisher events.Publisher
	cache     *cache.Cache
	opts      RelayOptions
	stop      chan struct{}
	done      chan struct{}
}

// NewRelay builds the relay worker.
func NewRelay(repo *Repo, registry *saga.Registry, publisher events.Publisher, cache *cache.Cache, opts RelayOptions) *Relay {
	return &Relay{
		repo:      repo,
		registry:  registry,
		publisher: publisher,
		cache:     cache,
		opts:      opts,
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
	}
}

// Start runs the relay loop until Stop is called.
func (r *Relay) Start(ctx context.Context) {
	go func() {
		defer close(r.done)
		logger.Info("outbox relay started (poll=%s batch=%d)", r.opts.PollInterval, r.opts.BatchSize)
		ticker := time.NewTicker(r.opts.PollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-r.stop:
				return
			case <-ticker.C:
				// Outer recovery: even infrastructure panics (locking, pending
				// query) must not crash the process — log, skip the batch, poll
				// again. Per-event containment in processBatch still applies
				// retry/backoff to misbehaving events.
				func() {
					defer func() {
						if rec := recover(); rec != nil {
							logger.Error("outbox relay batch panic recovered: %v", rec)
						}
					}()
					if err := r.processBatch(ctx); err != nil {
						logger.Warn("outbox relay batch error: %v", err)
					}
				}()
			}
		}
	}()
}

// Stop halts the relay loop.
func (r *Relay) Stop() {
	close(r.stop)
	<-r.done
}

// processBatch claims and processes one batch under a distributed lock so
// only one replica relays at a time. When no cache is configured (tests,
// single-instance mode) the lock is skipped.
func (r *Relay) processBatch(ctx context.Context) error {
	if r.cache != nil {
		mtx, err := lock.TryLock(ctx, r.cache, "outbox:relay", 30*time.Second)
		if err != nil {
			return nil // another replica is relaying
		}
		defer mtx.Release(ctx)
	}

	batch, err := r.repo.Pending(ctx, r.opts.BatchSize)
	if err != nil {
		return err
	}
	metrics.SetOutboxPending(len(batch))
	for _, ev := range batch {
		// Contain handler panics so a single broken event cannot crash the
		// process or kill the relay; the event is failed (retry/backoff applies)
		// and the rest of the batch still drains.
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("outbox event %s (%s) panic recovered: %v", ev.ID, ev.EventType, rec)
					_ = r.repo.MarkFailed(ctx, ev, fmt.Errorf("handler panic: %v", rec))
					metrics.ObserveOutbox("panic")
				}
			}()
			if err := r.processOne(ctx, ev); err != nil {
				logger.Warn("outbox event %s (%s) failed: %v", ev.ID, ev.EventType, err)
			}
		}()
	}
	return nil
}

func (r *Relay) processOne(ctx context.Context, ev models.OutboxEvent) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	event := saga.Event{
		AggregateType: ev.AggregateType,
		AggregateID:   ev.AggregateID,
		EventType:     ev.EventType,
		Payload:       ev.Payload,
	}
	// 1. In-process saga handlers drive business flows.
	if err := r.registry.Dispatch(ctx, event); err != nil {
		metrics.ObserveOutbox("failed")
		return r.repo.MarkFailed(ctx, ev, err)
	}
	// 2. Publish to the broker for external consumers.
	if err := r.publisher.Publish(ctx, events.Subject(ev.EventType), ev.Payload); err != nil {
		logger.Warn("publish %s failed: %v", ev.EventType, err)
	}
	if err := r.repo.MarkSent(ctx, ev.ID); err != nil {
		return err
	}
	metrics.ObserveOutbox("sent")
	return nil
}
