package aggregate

func TPS(successCount int64, windowSeconds int) float64 {
	if windowSeconds <= 0 {
		return 0
	}
	return float64(successCount) / float64(windowSeconds)
}
