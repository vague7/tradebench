package emit

import (
	"context"
	"sync"

	benchtypes "github.com/bench/shared/types"
)

type Streamer struct {
	mu     sync.Mutex
	events []benchtypes.BotEvent
}

func NewStreamer() *Streamer {
	return &Streamer{}
}

func (s *Streamer) Send(_ context.Context, event benchtypes.BotEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, event)
	return nil
}

func (s *Streamer) Events() []benchtypes.BotEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]benchtypes.BotEvent, len(s.events))
	copy(out, s.events)
	return out
}
