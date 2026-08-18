package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestCache(t *testing.T) (*miniredis.Miniredis, *Cache) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return mr, NewFromClient(client, "laundry-pro")
}

func TestSetGet(t *testing.T) {
	ctx := context.Background()
	mr, c := newTestCache(t)

	if err := c.Set(ctx, "name", "LaundryPro", time.Hour); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := c.Get(ctx, "name")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "LaundryPro" {
		t.Fatalf("Get = %q", got)
	}
	if !mr.Exists("laundry-pro:name") {
		t.Fatal("key must be stored with the prefix")
	}
	if mr.Exists("name") {
		t.Fatal("unprefixed key must never be created")
	}
}

func TestGetMissing(t *testing.T) {
	_, c := newTestCache(t)
	if _, err := c.Get(context.Background(), "missing"); err == nil {
		t.Fatal("Get must error for a missing key")
	}
}

func TestGetWithTTL(t *testing.T) {
	ctx := context.Background()
	_, c := newTestCache(t)

	if err := c.Set(ctx, "ttl-key", "v", time.Hour); err != nil {
		t.Fatalf("Set: %v", err)
	}
	val, ttl, err := c.GetWithTTL(ctx, "ttl-key")
	if err != nil {
		t.Fatalf("GetWithTTL: %v", err)
	}
	if val != "v" {
		t.Fatalf("val = %q", val)
	}
	if ttl <= 0 || ttl > time.Hour {
		t.Fatalf("ttl = %s, want ~1h", ttl)
	}
	// No-expiry keys report a ttl of ~0 (persistent key).
	if err := c.Set(ctx, "no-ttl", "v", 0); err != nil {
		t.Fatalf("Set(no ttl): %v", err)
	}
	if _, ttl, err := c.GetWithTTL(ctx, "no-ttl"); err != nil || ttl != 0 {
		t.Fatalf("no-ttl: ttl=%s err=%v, want 0", ttl, err)
	}
}

func TestDel(t *testing.T) {
	ctx := context.Background()
	_, c := newTestCache(t)
	if err := c.Set(ctx, "a", "1", 0); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := c.Set(ctx, "b", "2", 0); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := c.Del(ctx, "a", "b"); err != nil {
		t.Fatalf("Del: %v", err)
	}
	if _, err := c.Get(ctx, "a"); err == nil {
		t.Fatal("key a must be gone")
	}
	if _, err := c.Get(ctx, "b"); err == nil {
		t.Fatal("key b must be gone")
	}
}

func TestJSONRoundtrip(t *testing.T) {
	ctx := context.Background()
	_, c := newTestCache(t)

	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	in := payload{Name: "Wash & Fold", Count: 3}
	if err := c.SetJSON(ctx, "obj", in, time.Minute); err != nil {
		t.Fatalf("SetJSON: %v", err)
	}
	var out payload
	if err := c.GetJSON(ctx, "obj", &out); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if out != in {
		t.Fatalf("JSON roundtrip mismatch: %+v", out)
	}
	if err := c.GetJSON(ctx, "missing", &out); err == nil {
		t.Fatal("GetJSON must error for a missing key")
	}
}

func TestSetNX(t *testing.T) {
	ctx := context.Background()
	_, c := newTestCache(t)

	ok, err := c.SetNX(ctx, "lock", "owner-1", time.Minute)
	if err != nil || !ok {
		t.Fatalf("first SetNX = %v, %v; want true", ok, err)
	}
	ok, err = c.SetNX(ctx, "lock", "owner-2", time.Minute)
	if err != nil || ok {
		t.Fatalf("second SetNX = %v, %v; want false", ok, err)
	}
}

func TestIncrWithTTL(t *testing.T) {
	ctx := context.Background()
	mr, c := newTestCache(t)

	n, err := c.Incr(ctx, "failures:u1", time.Minute)
	if err != nil || n != 1 {
		t.Fatalf("first Incr = %d, %v; want 1", n, err)
	}
	n, _ = c.Incr(ctx, "failures:u1", time.Minute)
	if n != 2 {
		t.Fatalf("second Incr = %d, want 2", n)
	}
	if ttl := mr.TTL("laundry-pro:failures:u1"); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("TTL = %s, want ~1m", ttl)
	}
}

func TestKeyPrefixing(t *testing.T) {
	_, c := newTestCache(t)
	if got := c.Key("raw"); got != "laundry-pro:raw" {
		t.Fatalf("Key() = %q, want prefixed", got)
	}
}
