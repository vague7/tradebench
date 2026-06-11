package fleet

import "time"

// Phase name constants — FR-3.2.
const (
	PhaseWarmUp   = "warm-up"
	PhaseRamp     = "ramp"
	PhaseSustained = "sustained"
	PhaseSpike    = "spike"
	PhaseDrain    = "drain"
)

// Phase defines a single phase of the load profile.
type Phase struct {
	Name           string // one of the Phase* constants
	DurationSec    int    // how long this phase lasts
	TargetBotCount int    // target number of concurrent bots at end of phase
	LinearRamp     bool   // true = linear ramp to target; false = instant jump
}

// LoadProfile is an ordered sequence of phases defining the benchmark load curve.
type LoadProfile []Phase

// DefaultProfile returns the five-phase load profile exactly as specified in FR-3.2.
// These values are fixed in the PRD and are not configurable at runtime.
func DefaultProfile() LoadProfile {
	return LoadProfile{
		{Name: PhaseWarmUp, DurationSec: 30, TargetBotCount: 500, LinearRamp: true},
		{Name: PhaseRamp, DurationSec: 60, TargetBotCount: 10_000, LinearRamp: true},
		{Name: PhaseSustained, DurationSec: 120, TargetBotCount: 10_000, LinearRamp: false},
		{Name: PhaseSpike, DurationSec: 30, TargetBotCount: 50_000, LinearRamp: false},
		{Name: PhaseDrain, DurationSec: 30, TargetBotCount: 0, LinearRamp: true},
	}
}

// TotalDuration returns the sum of all phase durations.
// Used by the coordinator to pre-compute the benchmark's total runtime for logging.
func (lp LoadProfile) TotalDuration() time.Duration {
	var total int
	for _, p := range lp {
		total += p.DurationSec
	}
	return time.Duration(total) * time.Second
}
