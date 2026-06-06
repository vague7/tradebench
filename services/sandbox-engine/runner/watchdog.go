package runner

import "time"

type Watchdog struct {
	TTL time.Duration
}

func NewWatchdog(ttl time.Duration) *Watchdog {
	return &Watchdog{TTL: ttl}
}

func (w *Watchdog) ShouldKill(startedAt time.Time, now time.Time) bool {
	if w.TTL <= 0 {
		return false
	}
	return now.Sub(startedAt) >= w.TTL
}
