package eventbus

import (
	"context"
	"log"
	"sync"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

// EventHandler processes a single event. Return an error to signal failure
// (the bus logs it but continues processing subsequent events).
type EventHandler func(ctx context.Context, event domain.Event) error

// Bus is a simple in-process pub/sub event bus.
// Producers call Publish, consumers subscribe via Subscribe, and Run
// starts a background goroutine that drains the channel.
type Bus struct {
	mu       sync.RWMutex
	handlers map[domain.EventType][]EventHandler
	events   chan domain.Event
}

// New creates an event bus with the given channel buffer size.
// A buffer of 64 is a reasonable default for most workloads.
func New(bufSize int) *Bus {
	if bufSize <= 0 {
		bufSize = 64
	}
	return &Bus{
		handlers: make(map[domain.EventType][]EventHandler),
		events:   make(chan domain.Event, bufSize),
	}
}

// Publish enqueues an event. Returns immediately — handlers run asynchronously.
// If the channel is full the event is dropped and a warning is logged.
func (b *Bus) Publish(event domain.Event) {
	select {
	case b.events <- event:
	default:
		log.Printf("eventbus: channel full, dropping event %s", event.Type)
	}
}

// Subscribe registers a handler for the given event type.
// Handlers are called in order of subscription when the event is dispatched.
func (b *Bus) Subscribe(typ domain.EventType, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[typ] = append(b.handlers[typ], handler)
}

// Run starts the dispatch loop. It blocks until ctx is cancelled.
// Intended to be called in a goroutine from main.
func (b *Bus) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-b.events:
			b.dispatch(ctx, event)
		}
	}
}

// dispatch fans an event out to every subscribed handler.
func (b *Bus) dispatch(ctx context.Context, event domain.Event) {
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()

	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			log.Printf("eventbus: handler error for %s: %v", event.Type, err)
		}
	}
}
