// Package lock provides a Redis-backed distributed lock with automatic
// release via a Lua script and bounded retry with backoff.
package lock

import (
	"context"
	"errors"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/retry"
	"github.com/google/uuid"
)

const (
	releaseScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0`

	renewScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("pexpire", KEYS[1], ARGV[2])
end
return 0`
)

var (
	// ErrNotAcquired is returned when the lock cannot be acquired within attempts.
	ErrNotAcquired = errors.New("could not acquire lock")
	// ErrLost is returned when a renewal fails because the lock was lost.
	ErrLost = errors.New("lock lost")
)

// Mutex is a distributed mutex instance.
type Mutex struct {
	cache    *cache.Cache
	key      string
	owner    string
	ttl      time.Duration
	acquired bool
}

// TryLock attempts to acquire the lock once.
func TryLock(ctx context.Context, c *cache.Cache, key string, ttl time.Duration) (*Mutex, error) {
	owner := uuid.NewString()
	ok, err := c.SetNX(ctx, "lock:"+key, owner, ttl)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotAcquired
	}
	return &Mutex{cache: c, key: key, owner: owner, ttl: ttl, acquired: true}, nil
}

// Acquire tries to acquire the lock, retrying with exponential backoff until
// maxWait elapses.
func Acquire(ctx context.Context, c *cache.Cache, key string, ttl time.Duration, maxWait time.Duration) (*Mutex, error) {
	var mtx *Mutex
	err := retry.Do(ctx, func() error {
		m, err := TryLock(ctx, c, key, ttl)
		if err == nil {
			mtx = m
			return nil
		}
		if errors.Is(err, ErrNotAcquired) {
			return retry.ErrTransient
		}
		return err
	}, retry.WithMaxElapsed(maxWait))
	if err != nil {
		return nil, err
	}
	return mtx, nil
}

// Renew extends the lock TTL while still held.
func (m *Mutex) Renew(ctx context.Context, ttl time.Duration) error {
	if !m.acquired {
		return ErrNotAcquired
	}
	ok, err := m.cache.Client().Eval(ctx, renewScript, []string{m.cache.Key("lock:" + m.key)}, m.owner, ttl.Milliseconds()).Result()
	if err != nil {
		return err
	}
	if ok.(int64) == 0 {
		return ErrLost
	}
	m.ttl = ttl
	return nil
}

// Release releases the lock if still owned.
func (m *Mutex) Release(ctx context.Context) error {
	if !m.acquired {
		return nil
	}
	m.acquired = false
	_, err := m.cache.Client().Eval(ctx, releaseScript, []string{m.cache.Key("lock:" + m.key)}, m.owner).Result()
	return err
}
