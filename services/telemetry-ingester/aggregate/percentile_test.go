package aggregate

import (
	"testing"
)

func TestPercentiles(t *testing.T) {
	values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}

	tests := []struct {
		name string
		fn   func([]float64) float64
		want float64
	}{
		{"P50", P50, 50.0},
		{"P90", P90, 90.0},
		{"P99", P99, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(values)
			if got != tt.want {
				t.Errorf("%s(%v) = %v, want %v", tt.name, values, got, tt.want)
			}
		})
	}
}

func TestPercentilesEmpty(t *testing.T) {
	var empty []float64

	tests := []struct {
		name string
		fn   func([]float64) float64
	}{
		{"P50", P50},
		{"P90", P90},
		{"P99", P99},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_empty", func(t *testing.T) {
			got := tt.fn(empty)
			if got != 0.0 {
				t.Errorf("%s(empty) = %v, want 0.0", tt.name, got)
			}
		})
	}
}

func TestPercentileDoesNotMutateInput(t *testing.T) {
	original := []float64{100, 90, 80, 70, 60, 50, 40, 30, 20, 10}
	inputCopy := make([]float64, len(original))
	copy(inputCopy, original)

	P50(original)
	P90(original)
	P99(original)

	for i := range original {
		if original[i] != inputCopy[i] {
			t.Errorf("input was mutated at index %d: got %v, want %v", i, original[i], inputCopy[i])
		}
	}
}
