// Package cache wraps go-redis with key prefixing, JSON convenience helpers,
// TTL-aware operations and a distributed lock primitive.
package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache is a thin, prefix-aware Redis client.
type Cache struct {
	client *redis.Client
	prefix string
}

// New connects to Redis at the given URL.
func New(url, prefix string) (*Cache, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &Cache{client: client, prefix: prefix}, nil
}

// NewFromClient wraps an existing redis client (used in tests).
func NewFromClient(client *redis.Client, prefix string) *Cache {
	return &Cache{client: client, prefix: prefix}
}

// Client exposes the underlying redis client.
func (c *Cache) Client() *redis.Client { return c.client }

// Ping checks connectivity to Redis (used by the readiness probe).
func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Cache) key(k string) string { return c.prefix + ":" + k }

// Key returns the fully-prefixed Redis key for k. Consumers running raw
// scripts (e.g. distributed locks) must use this instead of building keys
// themselves, otherwise their commands miss the prefix.
func (c *Cache) Key(k string) string { return c.key(k) }

// Set stores a string value with TTL (0 = no expiry).
func (c *Cache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.client.Set(ctx, c.key(key), value, ttl).Err()
}

// Get fetches a string value.
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, c.key(key)).Result()
}

// GetWithTTL fetches a value and its remaining TTL.
func (c *Cache) GetWithTTL(ctx context.Context, key string) (string, time.Duration, error) {
	pipe := c.client.Pipeline()
	g := pipe.Get(ctx, c.key(key))
	t := pipe.TTL(ctx, c.key(key))
	if _, err := pipe.Exec(ctx); err != nil {
		return "", 0, err
	}
	val, err := g.Result()
	if err != nil {
		return "", 0, err
	}
	ttl, _ := t.Result()
	// Redis returns -1 for persistent keys; normalize to 0 so callers get a
	// consistent "no expiry" signal instead of a negative duration.
	if ttl < 0 {
		ttl = 0
	}
	return val, ttl, nil
}

// Del removes keys.
func (c *Cache) Del(ctx context.Context, keys ...string) error {
	full := make([]string, 0, len(keys))
	for _, k := range keys {
		full = append(full, c.key(k))
	}
	return c.client.Del(ctx, full...).Err()
}

// SetJSON marshals v and stores it with TTL.
func (c *Cache) SetJSON(ctx context.Context, key string, v any, ttl time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Set(ctx, key, string(b), ttl)
}

// GetJSON unmarshals the value at key into v. Returns redis.Nil when absent.
func (c *Cache) GetJSON(ctx context.Context, key string, v any) error {
	raw, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), v)
}

// SetNX atomically sets the value only if the key is absent.
func (c *Cache) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return c.client.SetNX(ctx, c.key(key), value, ttl).Result()
}

// Incr atomically increments a counter and sets TTL when the counter is new.
func (c *Cache) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := c.client.Pipeline()
	incr := pipe.Incr(ctx, c.key(key))
	pipe.Expire(ctx, c.key(key), ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Result()
}
