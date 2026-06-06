package fleet

import (
	"context"
	"fmt"
	"time"

	"github.com/bench/bot-fleet/bot"
)

type Coordinator struct {
	Profile  Profile
	Runner   *bot.Runner
}

func NewCoordinator(profile Profile, runner *bot.Runner) *Coordinator {
	return &Coordinator{Profile: profile, Runner: runner}
}

func (c *Coordinator) Run(ctx context.Context, submissionID string) error {
	if c.Runner == nil {
		return fmt.Errorf("coordinator: runner is required")
	}
	for _, phase := range c.Profile.Phases {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := c.Runner.Run(ctx, submissionID, phase.Name, phase.TargetConcurrency); err != nil {
			return err
		}
		if phase.DurationSec > 0 {
			time.Sleep(time.Millisecond)
		}
	}
	return nil
}
