package ingest

import (
	"github.com/bench/shared/types"
)

const ringBufferCapacity = 100_000

// RingBuffer is a fixed-capacity, goroutine-safe in-memory buffer for BotEvents.
// It is backed by a buffered channel, providing lock-free push/pop with
// Go's native backpressure semantics (FR-4.2).
type RingBuffer struct {
	ch chan types.BotEvent
}

// NewRingBuffer creates a new RingBuffer with a fixed capacity of 10,000 events.
func NewRingBuffer() *RingBuffer {
	return &RingBuffer{
		ch: make(chan types.BotEvent, ringBufferCapacity),
	}
}

// Push sends an event into the buffer. Returns true on success, false if the
// buffer is full (non-blocking). The caller is responsible for logging dropped events.
func (rb *RingBuffer) Push(e types.BotEvent) bool {
	select {
	case rb.ch <- e:
		return true
	default:
		return false
	}
}

// Drain reads up to n events from the buffer non-blocking.
// Returns whatever is available (may be fewer than n).
func (rb *RingBuffer) Drain(n int) []types.BotEvent {
	var events []types.BotEvent
	for i := 0; i < n; i++ {
		select {
		case e := <-rb.ch:
			events = append(events, e)
		default:
			return events
		}
	}
	return events
}

// C returns the underlying channel so the window manager can range over it.
func (rb *RingBuffer) C() <-chan types.BotEvent {
	return rb.ch
}
