package saga

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newSagaEnv(t *testing.T) (*Store, *miniredis.Miniredis) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.SagaState{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewStore(db, cache.NewFromClient(client, "laundry-pro")), mr
}

func TestStartAndGet(t *testing.T) {
	ctx := context.Background()
	s, _ := newSagaEnv(t)

	state := map[string]any{"booking_id": "b-1", "amount": 499}
	id, err := s.Start(ctx, "booking", "agg-1", state)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if id == "" {
		t.Fatal("Start returned an empty saga id")
	}

	got, err := s.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != models.SagaRunning || got.Step != "started" || got.Name != "booking" {
		t.Fatalf("unexpected saga: %+v", got)
	}
	var st map[string]any
	if err := json.Unmarshal(got.State, &st); err != nil || st["amount"].(float64) != 499 {
		t.Fatalf("state not persisted: %v, %v", st, err)
	}
}

func TestAdvance(t *testing.T) {
	ctx := context.Background()
	s, _ := newSagaEnv(t)

	id, _ := s.Start(ctx, "booking", "agg-1", map[string]any{"step": 0})
	if err := s.Advance(ctx, id, "payment_authorized", map[string]any{"step": 1}); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	got, _ := s.Get(ctx, id)
	if got.Step != "payment_authorized" || got.Status != models.SagaRunning {
		t.Fatalf("after advance: step=%s status=%s", got.Step, got.Status)
	}
}

func TestCompleteAndFail(t *testing.T) {
	ctx := context.Background()
	s, mr := newSagaEnv(t)

	id, _ := s.Start(ctx, "booking", "agg-1", map[string]any{})
	if err := s.Complete(ctx, id, map[string]any{"done": true}); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	// Eviction must be asserted before Get, which re-caches from the DB by design.
	if mr.Exists("laundry-pro:saga:" + id) {
		t.Fatal("completed saga must be evicted from cache")
	}
	got, _ := s.Get(ctx, id)
	if got.Status != models.SagaCompleted || got.Step != "completed" || got.FinishedAt == nil {
		t.Fatalf("after complete: %+v", got)
	}

	id2, _ := s.Start(ctx, "booking", "agg-2", map[string]any{})
	if err := s.Fail(ctx, id2, context.DeadlineExceeded); err != nil {
		t.Fatalf("Fail: %v", err)
	}
	failed, _ := s.Get(ctx, id2)
	if failed.Status != models.SagaFailed || failed.LastError == "" {
		t.Fatalf("after fail: %+v", failed)
	}
	if failed.LastError != context.DeadlineExceeded.Error() {
		t.Fatalf("last_error = %q", failed.LastError)
	}
}

func TestFindByAggregateReturnsRunning(t *testing.T) {
	ctx := context.Background()
	s, _ := newSagaEnv(t)

	// One completed and one running saga for the same aggregate.
	cid, _ := s.Start(ctx, "booking", "agg-9", map[string]any{})
	_ = s.Complete(ctx, cid, nil)
	rid, _ := s.Start(ctx, "booking", "agg-9", map[string]any{})

	got, err := s.FindByAggregate(ctx, "agg-9")
	if err != nil {
		t.Fatalf("FindByAggregate: %v", err)
	}
	if got.SagaID != rid {
		t.Fatalf("found %s, want the running saga %s", got.SagaID, rid)
	}
	if got.Status != models.SagaRunning {
		t.Fatalf("status = %s, want running", got.Status)
	}
}

func TestFindByAggregateMissing(t *testing.T) {
	ctx := context.Background()
	s, _ := newSagaEnv(t)
	if _, err := s.FindByAggregate(ctx, "no-such-agg"); err == nil {
		t.Fatal("FindByAggregate must error when nothing is running")
	}
}

func TestGetServesFromCache(t *testing.T) {
	ctx := context.Background()
	s, _ := newSagaEnv(t)

	id, _ := s.Start(ctx, "booking", "agg-1", map[string]any{"x": 1})
	// Start cached the row; delete the DB row to prove Get reads the hot cache.
	if err := s.db.Where("saga_id = ?", id).Delete(&models.SagaState{}).Error; err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err := s.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get from cache: %v", err)
	}
	if got.SagaID != id {
		t.Fatalf("cache miss: got %+v", got)
	}
}

func TestStartUniqueSagaIDs(t *testing.T) {
	ctx := context.Background()
	s, _ := newSagaEnv(t)
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		id, err := s.Start(ctx, "test", "agg-"+uuid.New().String(), nil)
		if err != nil {
			t.Fatalf("Start: %v", err)
		}
		if seen[id] {
			t.Fatalf("duplicate saga id %q", id)
		}
		seen[id] = true
	}
}
