package outbox

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/events"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/saga"
	"github.com/glebarez/sqlite" // pure-Go driver, no CGO required
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Unique DSN per test: with cache=shared, a bare :memory: DSN would be
	// shared across every test in the process.
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.OutboxEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestRelaySurvivesPanickingHandler verifies a handler panic is contained:
// the relay keeps draining and a subsequent healthy event is still sent.
func TestRelaySurvivesPanickingHandler(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	registry := saga.NewRegistry()
	registry.Register("boom", func(_ context.Context, _ saga.Event) error { panic("handler boom") })
	registry.Register("ok", func(_ context.Context, _ saga.Event) error { return nil })
	repo := NewRepo(db, 0) // MaxRetries 0: the first panic fails the event immediately.
	relay := NewRelay(repo, registry, events.NoopPublisher{}, nil, RelayOptions{
		PollInterval: 10 * time.Millisecond, BatchSize: 10, MaxRetries: 0,
	})

	enqueue := func(typ string) {
		t.Helper()
		if err := db.Transaction(func(tx *gorm.DB) error {
			return repo.Enqueue(ctx, tx, Event{
				AggregateType: "test", AggregateID: uuid.New().String(),
				EventType: typ, Payload: json.RawMessage(`{}`),
			})
		}); err != nil {
			t.Fatalf("enqueue %s: %v", typ, err)
		}
	}

	enqueue("boom")
	relay.Start(ctx)
	t.Cleanup(func() { relay.Stop() }) // no goroutine leak on early failure
	time.Sleep(150 * time.Millisecond) // let the panicking event hit the relay

	enqueue("ok")
	deadline := time.Now().Add(3 * time.Second)
	sent := false
	for time.Now().Before(deadline) {
		var ev models.OutboxEvent
		if err := db.Where("event_type = ?", "ok").First(&ev).Error; err == nil && ev.Status == models.OutboxSent {
			sent = true
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !sent {
		t.Fatal("relay did not survive handler panic: follow-up event was never sent")
	}

	// The panicking event must have been failed (not left pending forever).
	var boom models.OutboxEvent
	if err := db.Where("event_type = ?", "boom").First(&boom).Error; err != nil {
		t.Fatalf("boom event missing: %v", err)
	}
	if boom.Status != models.OutboxFailed {
		t.Fatalf("boom event status = %s, want failed (retry/backoff applied)", boom.Status)
	}
}

func TestEnqueueAndPending(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := NewRepo(db, 3)

	aggID := uuid.New().String()
	if err := db.Transaction(func(tx *gorm.DB) error {
		return repo.Enqueue(ctx, tx, Event{
			AggregateType: "booking", AggregateID: aggID, EventType: "booking_created",
			Payload: map[string]any{"booking_id": aggID},
		})
	}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	pending, err := repo.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending = %d, want 1", len(pending))
	}
	if pending[0].EventType != "booking_created" || pending[0].Status != models.OutboxPending {
		t.Fatalf("unexpected event: %+v", pending[0])
	}

	if err := repo.MarkSent(ctx, pending[0].ID); err != nil {
		t.Fatalf("MarkSent: %v", err)
	}
	remaining, err := repo.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("Pending after sent: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("pending after sent = %d, want 0", len(remaining))
	}
}

func TestMarkFailedBackoff(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := NewRepo(db, 3)

	var id any
	_ = db.Transaction(func(tx *gorm.DB) error {
		return repo.Enqueue(ctx, tx, Event{AggregateType: "booking", AggregateID: uuid.New().String(), EventType: "x"})
	})
	pending, _ := repo.Pending(ctx, 1)
	id = pending[0].ID

	if err := repo.MarkFailed(ctx, pending[0], context.DeadlineExceeded); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}
	var row models.OutboxEvent
	if err := db.First(&row, "id = ?", id).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if row.Status != models.OutboxPending {
		t.Fatalf("status = %s, want pending (retryable)", row.Status)
	}
	if row.RetryCount != 1 {
		t.Fatalf("retry_count = %d, want 1", row.RetryCount)
	}
	if row.AvailableAt == nil || row.AvailableAt.After(time.Now().Add(10*time.Second)) {
		t.Fatalf("backoff window wrong: %v", row.AvailableAt)
	}
	if row.LastError == "" {
		t.Fatal("last_error not recorded")
	}
	// Not visible to Pending until the backoff elapses.
	if backoffPending, _ := repo.Pending(ctx, 10); len(backoffPending) != 0 {
		t.Fatal("event with future available_at was returned by Pending")
	}
}

func TestMarkFailedPermanent(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := NewRepo(db, 0) // zero retries -> immediately permanent

	var ev models.OutboxEvent
	_ = db.Transaction(func(tx *gorm.DB) error {
		return repo.Enqueue(ctx, tx, Event{AggregateType: "booking", AggregateID: uuid.New().String(), EventType: "x"})
	})
	pending, _ := repo.Pending(ctx, 1)
	ev = pending[0]

	if err := repo.MarkFailed(ctx, ev, context.DeadlineExceeded); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}
	var row models.OutboxEvent
	if err := db.First(&row, "id = ?", ev.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if row.Status != models.OutboxFailed {
		t.Fatalf("status = %s, want failed", row.Status)
	}
}

// fakePublisher records published subjects without touching a broker.
type fakePublisher struct {
	subjects []string
}

func (f *fakePublisher) Publish(_ context.Context, subject string, _ []byte) error {
	f.subjects = append(f.subjects, subject)
	return nil
}

func (f *fakePublisher) Available() bool { return true }

// TestRelayDispatchesAndMarksSent exercises the relay's batch path: a
// registered saga handler runs, the event is published and marked sent.
func TestRelayDispatchesAndMarksSent(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := NewRepo(db, 3)

	registry := saga.NewRegistry()
	handled := make(chan string, 1)
	registry.Register("booking_created", func(_ context.Context, ev saga.Event) error {
		handled <- ev.AggregateID
		return nil
	})

	aggID := uuid.New().String()
	_ = db.Transaction(func(tx *gorm.DB) error {
		return repo.Enqueue(ctx, tx, Event{
			AggregateType: "booking", AggregateID: aggID, EventType: "booking_created",
			Payload: map[string]any{"booking_id": aggID},
		})
	})

	pub := &fakePublisher{}
	relay := NewRelay(repo, registry, pub, nil, RelayOptions{PollInterval: time.Millisecond, BatchSize: 10, MaxRetries: 3})

	// Simulate one poll cycle (lock is nil-guarded: cache is unused here).
	if err := relay.processBatch(ctx); err != nil {
		t.Fatalf("processBatch: %v", err)
	}
	select {
	case got := <-handled:
		if got != aggID {
			t.Fatalf("handler got %s, want %s", got, aggID)
		}
	default:
		t.Fatal("saga handler was never invoked")
	}
	wantSubject := "laundrypro.events.booking_created"
	if len(pub.subjects) != 1 || pub.subjects[0] != wantSubject {
		t.Fatalf("published subjects = %v, want [%s]", pub.subjects, wantSubject)
	}
	left, _ := repo.Pending(ctx, 10)
	if len(left) != 0 {
		t.Fatal("event not marked sent after relay")
	}
}

// TestRelayRetriesOnHandlerFailure verifies a failing handler marks the
// event for backoff instead of marking it sent.
func TestRelayRetriesOnHandlerFailure(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := NewRepo(db, 3)

	registry := saga.NewRegistry()
	registry.Register("booking_created", func(context.Context, saga.Event) error {
		return context.DeadlineExceeded
	})
	aggID := uuid.New().String()
	_ = db.Transaction(func(tx *gorm.DB) error {
		return repo.Enqueue(ctx, tx, Event{AggregateType: "booking", AggregateID: aggID, EventType: "booking_created"})
	})

	relay := NewRelay(repo, registry, &fakePublisher{}, nil, RelayOptions{BatchSize: 10, MaxRetries: 3})
	if err := relay.processBatch(ctx); err != nil {
		t.Fatalf("processBatch should absorb handler errors: %v", err)
	}
	var row models.OutboxEvent
	if err := db.First(&row, "aggregate_id = ?", aggID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if row.Status != models.OutboxPending || row.RetryCount == 0 {
		t.Fatalf("expected retryable pending event, got status=%s retries=%d", row.Status, row.RetryCount)
	}
}
