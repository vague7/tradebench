package aggregate

import (
	"github.com/bench/shared/types"
)

// Compute calculates TPS (transactions per second) as the count of successful
// events (HTTPStatus == 200) divided by the window duration in seconds.
// Returns 0.0 if windowSec <= 0 or events is empty.
func Compute(events []types.BotEvent, windowSec int) float64 {
	if windowSec <= 0 || len(events) == 0 {
		return 0.0
	}

	var successCount int
	for _, e := range events {
		if e.HTTPStatus == 200 {
			successCount++
		}
	}

	return float64(successCount) / float64(windowSec)
}
