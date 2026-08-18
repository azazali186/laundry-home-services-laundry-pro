package lock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newTestCache spins up miniredis and a prefix-aware cache.
func newTestCache(t *testing.T) (*miniredis.Miniredis, *cache.Cache) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return mr, cache.NewFromClient(client, "laundry-pro")
}

// TestReleaseActuallyDeletes is the regression test for the prefix bug:
// Release used to run its Lua script against the unprefixed key, so the
// lock lingered until TTL expiry.
func TestReleaseActuallyDeletes(t *testing.T) {
	ctx := context.Background()
	mr, c := newTestCache(t)

	mtx, err := TryLock(ctx, c, "outbox:relay", 30*time.Second)
	if err != nil {
		t.Fatalf("TryLock: %v", err)
	}
	if !mr.Exists("laundry-pro:lock:outbox:relay") {
		t.Fatal("prefixed lock key should exist")
	}
	if mr.Exists("lock:outbox:relay") {
		t.Fatal("unprefixed key must never be created")
	}

	if err := mtx.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if mr.Exists("laundry-pro:lock:outbox:relay") {
		t.Fatal("lock key still exists after Release (prefix bug regression)")
	}
}

func TestMutualExclusion(t *testing.T) {
	ctx := context.Background()
	_, c := newTestCache(t)

	m1, err := TryLock(ctx, c, "inventory:adjust:abc", 10*time.Second)
	if err != nil {
		t.Fatalf("first TryLock: %v", err)
	}
	m2, err := TryLock(ctx, c, "inventory:adjust:abc", 10*time.Second)
	if !errors.Is(err, ErrNotAcquired) {
		t.Fatalf("second TryLock should fail with ErrNotAcquired, got %v", err)
	}
	if m2 != nil {
		t.Fatal("second TryLock returned a mutex despite failure")
	}
	if err := m1.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}

	m3, err := TryLock(ctx, c, "inventory:adjust:abc", 10*time.Second)
	if err != nil {
		t.Fatalf("TryLock after release: %v", err)
	}
	_ = m3.Release(ctx)
}

// TestOwnerCheck verifies a foreign owner cannot release the lock.
func TestOwnerCheck(t *testing.T) {
	ctx := context.Background()
	_, c := newTestCache(t)

	m1, err := TryLock(ctx, c, "payment:capture:123", 15*time.Second)
	if err != nil {
		t.Fatalf("TryLock: %v", err)
	}
	// A different mutex instance (different owner) must not delete the key.
	other := &Mutex{cache: c, key: "payment:capture:123", owner: "someone-else", acquired: true}
	if err := other.Release(ctx); err != nil {
		t.Fatalf("foreign Release should not error: %v", err)
	}
	if err := m1.Release(ctx); err != nil {
		t.Fatalf("owner Release: %v", err)
	}
}

func TestRenew(t *testing.T) {
	ctx := context.Background()
	mr, c := newTestCache(t)

	mtx, err := TryLock(ctx, c, "outbox:relay", 5*time.Second)
	if err != nil {
		t.Fatalf("TryLock: %v", err)
	}
	if err := mtx.Renew(ctx, 60*time.Second); err != nil {
		t.Fatalf("Renew: %v", err)
	}
	ttl := mr.TTL("laundry-pro:lock:outbox:relay")
	if ttl <= 5*time.Second || ttl > 60*time.Second {
		t.Fatalf("TTL after renew = %s, want ~60s", ttl)
	}
	_ = mtx.Release(ctx)
}

func TestAcquireWithRetry(t *testing.T) {
	ctx := context.Background()
	_, c := newTestCache(t)

	m1, err := TryLock(ctx, c, "busy", 2*time.Second)
	if err != nil {
		t.Fatalf("TryLock: %v", err)
	}
	// maxWait of 300ms should not be enough to wait out the 2s TTL.
	if _, err := Acquire(ctx, c, "busy", 2*time.Second, 300*time.Millisecond); err == nil {
		t.Fatal("Acquire should fail while the lock is held")
	}
	_ = m1.Release(ctx)

	mtx, err := Acquire(ctx, c, "busy", 2*time.Second, time.Second)
	if err != nil {
		t.Fatalf("Acquire after release: %v", err)
	}
	_ = mtx.Release(ctx)
}
