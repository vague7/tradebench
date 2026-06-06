package aggregate

import "sort"

func Percentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	copied := append([]float64(nil), values...)
	sort.Float64s(copied)
	if percentile <= 0 {
		return copied[0]
	}
	if percentile >= 100 {
		return copied[len(copied)-1]
	}
	index := int((percentile / 100.0) * float64(len(copied)-1))
	return copied[index]
}
