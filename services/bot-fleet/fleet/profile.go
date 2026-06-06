package fleet

type Phase struct {
	Name             string
	DurationSec      int
	TargetConcurrency int
}

type Profile struct {
	Phases []Phase
}

func DefaultProfile() Profile {
	return Profile{
		Phases: []Phase{
			{Name: "warm-up", DurationSec: 30, TargetConcurrency: 500},
			{Name: "ramp", DurationSec: 60, TargetConcurrency: 10000},
			{Name: "sustained", DurationSec: 120, TargetConcurrency: 10000},
			{Name: "spike", DurationSec: 30, TargetConcurrency: 50000},
			{Name: "drain", DurationSec: 30, TargetConcurrency: 0},
		},
	}
}
