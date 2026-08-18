// Package bus is a tiny in-process pub/sub used to fan out domain events
// to the WebSocket hub. NATS remains the cross-service transport; this bus
// avoids wiring latency for in-process consumers.
package bus

import "sync"

// Event is a domain event broadcast on the bus.
type Event struct {
	Type        string `json:"type"`
	AggregateID string `json:"aggregate_id"`
	Data        any    `json:"data,omitempty"`
}

// Bus fans events out to subscribers.
type Bus struct {
	mu   sync.RWMutex
	subs map[string][]chan Event
}

// New builds an empty bus.
func New() *Bus {
	return &Bus{subs: map[string][]chan Event{}}
}

// Publish delivers an event to all subscribers of its type.
// Delivery is non-blocking; slow consumers drop events rather than stall.
func (b *Bus) Publish(ev Event) {
	b.mu.RLock()
	chans := b.subs[ev.Type]
	b.mu.RUnlock()
	for _, ch := range chans {
		select {
		case ch <- ev:
		default:
		}
	}
}

// Subscribe returns a buffered channel for events of the given type.
func (b *Bus) Subscribe(eventType string) <-chan Event {
	ch := make(chan Event, 64)
	b.mu.Lock()
	b.subs[eventType] = append(b.subs[eventType], ch)
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel from the bus.
func (b *Bus) Unsubscribe(eventType string, ch <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	chans := b.subs[eventType]
	for i, c := range chans {
		if c == ch {
			b.subs[eventType] = append(chans[:i], chans[i+1:]...)
			break
		}
	}
}
