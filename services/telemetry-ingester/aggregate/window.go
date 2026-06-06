package aggregate

import "time"

type WindowManager struct {
	WindowSeconds int
}

func NewWindowManager(windowSeconds int) *WindowManager {
	if windowSeconds <= 0 {
		windowSeconds = 10
	}
	return &WindowManager{WindowSeconds: windowSeconds}
}

func (w *WindowManager) WindowEnd(now time.Time) time.Time {
	return now.UTC().Truncate(time.Duration(w.WindowSeconds) * time.Second)
}
