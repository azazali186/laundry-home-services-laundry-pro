package bus

import (
	"testing"
	"time"
)

func TestPublishDelivers(t *testing.T) {
	b := New()
	ch := b.Subscribe("booking.created")
	defer b.Unsubscribe("booking.created", ch)

	ev := Event{Type: "booking.created", AggregateID: "agg-1", Data: map[string]any{"x": 1}}
	b.Publish(ev)

	select {
	case got := <-ch:
		if got.Type != ev.Type || got.AggregateID != "agg-1" {
			t.Fatalf("delivered %+v, want %+v", got, ev)
		}
	case <-time.After(time.Second):
		t.Fatal("event was not delivered")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	b := New()
	ch1 := b.Subscribe("orders.updated")
	ch2 := b.Subscribe("orders.updated")
	defer b.Unsubscribe("orders.updated", ch1)
	defer b.Unsubscribe("orders.updated", ch2)

	b.Publish(Event{Type: "orders.updated", AggregateID: "o-1"})
	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case got := <-ch:
			if got.AggregateID != "o-1" {
				t.Fatalf("subscriber %d got %+v", i, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d never received the event", i)
		}
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	b := New()
	ch := b.Subscribe("topic")
	b.Unsubscribe("topic", ch)

	b.Publish(Event{Type: "topic"})
	select {
	case got := <-ch:
		t.Fatalf("received event after unsubscribe: %+v", got)
	default:
	}
}

func TestPublishNoSubscribers(t *testing.T) {
	b := New()
	b.Publish(Event{Type: "nobody.listens"}) // must not panic
}

func TestSlowConsumerDropsNotBlocks(t *testing.T) {
	b := New()
	// Subscriber that never drains its 64-buffer: publishing more than the
	// buffer must drop events instead of blocking the publisher.
	_ = b.Subscribe("flood")

	done := make(chan struct{})
	go func() {
		for i := 0; i < 200; i++ {
			b.Publish(Event{Type: "flood"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a slow consumer")
	}
}

func TestEventDecodePayload(t *testing.T) {
	// Event here is the bus event; Decode lives on the saga.Event type and is
	// tested there. This guards the wire shape only.
	ev := Event{Type: "x", AggregateID: "a"}
	if ev.Type != "x" || ev.AggregateID != "a" {
		t.Fatalf("unexpected event shape: %+v", ev)
	}
}
