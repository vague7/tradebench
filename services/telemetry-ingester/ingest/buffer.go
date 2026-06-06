package ingest

import (
	"context"
	"sync"

	benchtypes "github.com/bench/shared/types"
)

type Buffer struct {
	mu      sync.Mutex
	limit   int
	events  []benchtypes.BotEvent
}

func NewBuffer(limit int) *Buffer {
	if limit <= 0 {
		limit = 1
	}
	return &Buffer{limit: limit}
}

func (b *Buffer) Push(ctx context.Context, event benchtypes.BotEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.events) >= b.limit {
		b.events = b.events[1:]
	}
	b.events = append(b.events, event)
	return nil
}

func (b *Buffer) Events() []benchtypes.BotEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	clone := make([]benchtypes.BotEvent, len(b.events))
	copy(clone, b.events)
	return clone
}
