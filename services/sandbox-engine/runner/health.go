package runner

import (
	"context"
	"time"
)

type HealthChecker struct {
	Timeout time.Duration
}

func NewHealthChecker(timeout time.Duration) *HealthChecker {
	return &HealthChecker{Timeout: timeout}
}

func (h *HealthChecker) WaitReady(ctx context.Context, containerID string) error {
	if containerID == "" {
		return context.Canceled
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Millisecond):
		return nil
	}
}
