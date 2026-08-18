package retry

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func fast() Options {
	return Options{InitInterval: time.Millisecond, MaxInterval: 2 * time.Millisecond, MaxElapsed: 200 * time.Millisecond, MaxRetries: -1}
}

func TestDoImmediateSuccess(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func() error {
		calls++
		return nil
	}, fast())
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if calls != 1 {
		t.Fatalf("fn called %d times, want 1", calls)
	}
}

func TestDoRetriesTransientThenSucceeds(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func() error {
		calls++
		if calls < 3 {
			return ErrTransient
		}
		return nil
	}, fast())
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if calls != 3 {
		t.Fatalf("fn called %d times, want 3 (2 retries)", calls)
	}
}

func TestDoAbortsOnPermanentError(t *testing.T) {
	calls := 0
	boom := errors.New("permanent: invalid input")
	err := Do(context.Background(), func() error {
		calls++
		return boom
	}, fast())
	if !errors.Is(err, boom) {
		t.Fatalf("Do err = %v, want %v", err, boom)
	}
	if calls != 1 {
		t.Fatalf("fn called %d times, want 1 (no retry for permanent errors)", calls)
	}
}

func TestDoMaxRetries(t *testing.T) {
	calls := 0
	opts := fast()
	opts.MaxRetries = 2
	err := Do(context.Background(), func() error {
		calls++
		return ErrTransient
	}, opts)
	if err == nil {
		t.Fatal("Do should fail after max retries")
	}
	if !strings.Contains(err.Error(), "max retries exceeded") {
		t.Fatalf("err = %v, want max-retries message", err)
	}
	if calls != 2 {
		t.Fatalf("fn called %d times, want 2 (MaxRetries=2)", calls)
	}
}

func TestDoContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled
	err := Do(ctx, func() error {
		return ErrTransient
	}, fast())
	if err == nil {
		t.Fatal("Do should surface context cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestErrTransientIsSentinel(t *testing.T) {
	if !errors.Is(ErrTransient, ErrTransient) {
		t.Fatal("ErrTransient sentinel broken")
	}
	if errors.Is(errors.New("x"), ErrTransient) {
		t.Fatal("unrelated errors must not match ErrTransient")
	}
}

func TestWithMaxElapsedDefaults(t *testing.T) {
	o := WithMaxElapsed(time.Second)
	if o.InitInterval != 200*time.Millisecond || o.MaxInterval != 2*time.Second || o.MaxElapsed != time.Second {
		t.Fatalf("WithMaxElapsed defaults wrong: %+v", o)
	}
}
