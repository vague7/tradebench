package fleet

import (
	"time"
	
	"github.com/bench/bot-fleet/config"
)

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
func DefaultProfile(
	cfg *config.Config,
) LoadProfile {

	return LoadProfile{

		{
			Name:           PhaseWarmUp,
			DurationSec:    cfg.WarmupDuration,
			TargetBotCount: cfg.WarmupCount,
			LinearRamp:     true,
		},

		{
			Name:           PhaseRamp,
			DurationSec:    cfg.RampDuration,
			TargetBotCount: cfg.RampCount,
			LinearRamp:     true,
		},

		{
			Name:           PhaseSustained,
			DurationSec:    cfg.SustainedDuration,
			TargetBotCount: cfg.SustainedCount,
			LinearRamp:     false,
		},

		{
			Name:           PhaseSpike,
			DurationSec:    cfg.SpikeDuration,
			TargetBotCount: cfg.SpikeCount,
			LinearRamp:     false,
		},

		{
			Name:           PhaseDrain,
			DurationSec:    cfg.DrainDuration,
			TargetBotCount: 0,
			LinearRamp:     true,
		},
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
