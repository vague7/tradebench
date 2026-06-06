package ingest

import (
	"context"
	"fmt"
	"sync"
	"time"

	benchtypes "github.com/bench/shared/types"
)

type Server struct {
	mu      sync.Mutex
	buffer  *Buffer
	windows map[string][]benchtypes.BotEvent
}

func NewServer(buffer *Buffer) *Server {
	return &Server{buffer: buffer, windows: make(map[string][]benchtypes.BotEvent)}
}

func (s *Server) StreamEvent(ctx context.Context, event benchtypes.BotEvent) error {
	if s.buffer == nil {
		return fmt.Errorf("ingest: buffer is required")
	}
	if err := s.buffer.Push(ctx, event); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.windows[event.SubmissionID] = append(s.windows[event.SubmissionID], event)
	return nil
}

func (s *Server) Snapshot(submissionID string) []benchtypes.BotEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	events := s.windows[submissionID]
	clone := make([]benchtypes.BotEvent, len(events))
	copy(clone, events)
	return clone
}

func (s *Server) WindowEnd() time.Time {
	return time.Now().UTC()
}
