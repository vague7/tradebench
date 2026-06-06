package correctness

import benchtypes "github.com/bench/shared/types"

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) Score(expected []benchtypes.Fill, actual []benchtypes.Fill) float64 {
	if len(expected) == 0 {
		return 0
	}
	matches := 0
	limit := len(expected)
	if len(actual) < limit {
		limit = len(actual)
	}
	for i := 0; i < limit; i++ {
		if expected[i] == actual[i] {
			matches++
		}
	}
	return float64(matches) / float64(len(expected))
}
