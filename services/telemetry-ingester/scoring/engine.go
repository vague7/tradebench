package scoring

import (
	"math"
	"time"

	benchtypes "github.com/bench/shared/types"
)

type Engine struct {
	TargetTPS float64
	MaxP99Ms  float64
}

func NewEngine(targetTPS, maxP99Ms float64) *Engine {
	if targetTPS <= 0 {
		targetTPS = 50000
	}
	if maxP99Ms <= 0 {
		maxP99Ms = 1000
	}
	return &Engine{TargetTPS: targetTPS, MaxP99Ms: maxP99Ms}
}

func (e *Engine) Compute(snapshot benchtypes.MetricSnapshot) benchtypes.Score {
	throughput := math.Min(snapshot.TPS/e.TargetTPS, 1)
	latency := math.Max(0, 1-(snapshot.P99LatencyMs/e.MaxP99Ms))
	correctness := snapshot.CorrectnessScore
	final := 0.4*throughput + 0.4*latency + 0.2*correctness
	return benchtypes.Score{
		SubmissionID:    snapshot.SubmissionID,
		ThroughputScore:  throughput,
		LatencyScore:    latency,
		CorrectnessScore: correctness,
		FinalScore:      final,
		ComputedAt:      time.Now().UTC(),
	}
}
