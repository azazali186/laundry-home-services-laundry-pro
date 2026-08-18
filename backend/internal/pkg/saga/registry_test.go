package saga

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRegisterAndHas(t *testing.T) {
	r := NewRegistry()
	if r.Has("booking.created") {
		t.Fatal("new registry must have no handlers")
	}
	r.Register("booking.created", func(context.Context, Event) error { return nil })
	if !r.Has("booking.created") {
		t.Fatal("registered handler not found")
	}
	r.Register("booking.created", func(context.Context, Event) error { return nil }) // overwrite
	if !r.Has("booking.created") {
		t.Fatal("overwritten handler not found")
	}
}

func TestDispatchUnknownEvent(t *testing.T) {
	r := NewRegistry()
	// Unknown events are treated as success (may target other services).
	if err := r.Dispatch(context.Background(), Event{EventType: "unknown"}); err != nil {
		t.Fatalf("Dispatch(unknown) = %v, want nil", err)
	}
}

func TestDispatchRunsHandler(t *testing.T) {
	r := NewRegistry()
	var got string
	r.Register("booking.created", func(_ context.Context, ev Event) error {
		got = ev.AggregateID
		return nil
	})
	err := r.Dispatch(context.Background(), Event{
		AggregateType: "booking", AggregateID: "agg-1", EventType: "booking.created",
		Payload: json.RawMessage(`{"booking_id":"agg-1"}`),
	})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if got != "agg-1" {
		t.Fatalf("handler got aggregate %q, want agg-1", got)
	}
}

func TestDispatchDecode(t *testing.T) {
	r := NewRegistry()
	var payload struct {
		BookingID string `json:"booking_id"`
	}
	r.Register("booking.created", func(_ context.Context, ev Event) error {
		return ev.Decode(&payload)
	})
	if err := r.Dispatch(context.Background(), Event{
		EventType: "booking.created", Payload: json.RawMessage(`{"booking_id":"b-9"}`),
	}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if payload.BookingID != "b-9" {
		t.Fatalf("decoded booking_id = %q", payload.BookingID)
	}
}

func TestDispatchPanicRecovered(t *testing.T) {
	r := NewRegistry()
	r.Register("boom", func(context.Context, Event) error { panic("handler exploded") })
	err := r.Dispatch(context.Background(), Event{EventType: "boom"})
	if err == nil {
		t.Fatal("Dispatch must surface a panic as an error")
	}
	if !strings.Contains(err.Error(), "panic") {
		t.Fatalf("err = %v, want panic mention", err)
	}
}

func TestDispatchRetriesTransientFailure(t *testing.T) {
	r := NewRegistry()
	calls := 0
	r.Register("flaky", func(context.Context, Event) error {
		calls++
		if calls < 2 {
			return errors.New("transient hiccup")
		}
		return nil
	})
	err := r.Dispatch(context.Background(), Event{EventType: "flaky"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if calls != 2 {
		t.Fatalf("handler called %d times, want 2 (one retry)", calls)
	}
}

func TestDispatchPermanentFailureAborts(t *testing.T) {
	r := NewRegistry()
	calls := 0
	r.Register("always-fails", func(context.Context, Event) error {
		calls++
		return errors.New("always failing")
	})
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	err := r.Dispatch(ctx, Event{EventType: "always-fails"})
	if err == nil {
		t.Fatal("Dispatch must eventually return an error for a permanently failing handler")
	}
	if calls < 2 {
		t.Fatalf("handler called %d times, want retries before timeout", calls)
	}
}
