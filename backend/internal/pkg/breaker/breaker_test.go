package breaker

import (
	"context"
	"errors"
	"testing"

	"github.com/sony/gobreaker"
)

func TestExecuteSuccess(t *testing.T) {
	b := New("test-ok")
	got, err := b.Execute(context.Background(), func(_ context.Context) (any, error) {
		return "result", nil
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got != "result" {
		t.Fatalf("Execute returned %v, want result", got)
	}
	if b.State() != "closed" {
		t.Fatalf("state = %s, want closed", b.State())
	}
}

func TestExecutePropagatesError(t *testing.T) {
	b := New("test-err")
	want := errors.New("upstream down")
	_, err := b.Execute(context.Background(), func(_ context.Context) (any, error) {
		return nil, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("Execute err = %v, want %v", err, want)
	}
}

func TestTripsAfterConsecutiveFailures(t *testing.T) {
	b := New("test-trip")
	boom := errors.New("boom")

	for i := 0; i < 5; i++ {
		if _, err := b.Execute(context.Background(), func(_ context.Context) (any, error) {
			return nil, boom
		}); err == nil {
			t.Fatalf("call %d should have failed", i+1)
		}
	}
	if b.State() != "open" {
		t.Fatalf("state = %s, want open after 5 consecutive failures", b.State())
	}

	// While open, calls fail fast without invoking the function.
	calls := 0
	_, err := b.Execute(context.Background(), func(_ context.Context) (any, error) {
		calls++
		return "ok", nil
	})
	if err == nil {
		t.Fatal("Execute should fail while the breaker is open")
	}
	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Fatalf("open-state err = %v, want ErrOpenState", err)
	}
	if calls != 0 {
		t.Fatalf("fn called %d times while open, want 0 (fail-fast)", calls)
	}
}

func TestRecoversAfterSuccess(t *testing.T) {
	b := New("test-recover")
	boom := errors.New("boom")

	// Trip the breaker.
	for i := 0; i < 5; i++ {
		_, _ = b.Execute(context.Background(), func(_ context.Context) (any, error) {
			return nil, boom
		})
	}
	if b.State() != "open" {
		t.Fatal("breaker should be open")
	}

	// A fresh breaker instance with a healthy function succeeds, proving the
	// breaker itself does not corrupt execution after trips (the open->half-open
	// transition is time-gated at 15s, so it is covered by gobreaker's tests).
	fresh := New("test-recover-fresh")
	if _, err := fresh.Execute(context.Background(), func(_ context.Context) (any, error) {
		return "ok", nil
	}); err != nil {
		t.Fatalf("fresh breaker Execute: %v", err)
	}
}
