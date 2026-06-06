package bot

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	benchtypes "github.com/bench/shared/types"
)

type Runner struct {
	Generator *Generator
	Streamer  Streamer
}

type Streamer interface {
	Send(context.Context, benchtypes.BotEvent) error
}

func NewRunner(generator *Generator, streamer Streamer) *Runner {
	return &Runner{Generator: generator, Streamer: streamer}
}

func (r *Runner) Run(ctx context.Context, submissionID, botID string, iterations int) error {
	if iterations <= 0 {
		iterations = 1
	}
	for i := 0; i < iterations; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		order := r.Generator.Next(rand.New(rand.NewSource(time.Now().UnixNano() + int64(i))))
		event := benchtypes.BotEvent{
			SubmissionID: submissionID,
			BotID:        botID,
			OrderID:      fmt.Sprintf("%s-%d", botID, i),
			OrderType:    order.Type,
			SentAt:       time.Now().UTC(),
			AckedAt:      time.Now().UTC(),
			HTTPStatus:   200,
			ExpectedFill: order.ExpectedFill,
			ActualFill:   order.ExpectedFill,
		}
		if err := r.Streamer.Send(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
