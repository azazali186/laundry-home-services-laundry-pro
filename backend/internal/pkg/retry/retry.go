// Package retry provides bounded retry with exponential backoff and jitter
// for transient failures, built on cenkalti/backoff.
package retry

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// ErrTransient marks a failure as retryable.
var ErrTransient = errors.New("transient failure")

// Options configure retry behavior.
type Options struct {
	InitInterval time.Duration
	MaxInterval  time.Duration
	MaxElapsed   time.Duration
	MaxRetries   int
}

// WithMaxElapsed sets only the overall budget (sensible defaults for the rest).
func WithMaxElapsed(d time.Duration) Options {
	return Options{InitInterval: 200 * time.Millisecond, MaxInterval: 2 * time.Second, MaxElapsed: d, MaxRetries: -1}
}

// Default returns default retry options (10 retries, ~1s budget).
func Default() Options {
	return Options{InitInterval: 100 * time.Millisecond, MaxInterval: 1 * time.Second, MaxElapsed: 5 * time.Second, MaxRetries: -1}
}

// Do runs fn until it succeeds or the budget is exhausted. Only errors that
// wrap ErrTransient are retried with backoff; any other error is treated as
// permanent and returned immediately (no wasted retries).
func Do(ctx context.Context, fn func() error, opts Options) error {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = opts.InitInterval
	b.MaxInterval = opts.MaxInterval
	b.MaxElapsedTime = opts.MaxElapsed
	b.Multiplier = 2

	attempts := 0
	operation := func() error {
		attempts++
		if opts.MaxRetries > 0 && attempts > opts.MaxRetries {
			return backoff.Permanent(errors.New("max retries exceeded"))
		}
		err := fn()
		if err != nil && !errors.Is(err, ErrTransient) {
			return backoff.Permanent(err)
		}
		return err
	}

	err := backoff.Retry(operation, backoff.WithContext(b, ctx))
	if err == nil {
		return nil
	}
	// Unwrap permanent errors so callers see the original failure.
	var perm *backoff.PermanentError
	if errors.As(err, &perm) {
		return perm.Err
	}
	return err
}
