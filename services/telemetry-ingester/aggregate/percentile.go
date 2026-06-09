package aggregate

import (
	"math"
	"sort"
)

// P50 returns the 50th percentile of the given latency values.
// Returns 0.0 if the input slice is empty.
func P50(latenciesMs []float64) float64 {
	return percentile(latenciesMs, 50)
}

// P90 returns the 90th percentile of the given latency values.
// Returns 0.0 if the input slice is empty.
func P90(latenciesMs []float64) float64 {
	return percentile(latenciesMs, 90)
}

// P99 returns the 99th percentile of the given latency values.
// Returns 0.0 if the input slice is empty.
func P99(latenciesMs []float64) float64 {
	return percentile(latenciesMs, 99)
}

// percentile computes the p-th percentile using a sorted-slice approach.
// It sorts a copy of the input (never mutates the original).
// Index formula: idx = int(math.Ceil(p/100.0 * float64(len(sorted)))) - 1, clamped to [0, len-1].
func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	idx := int(math.Ceil(p/100.0*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}
