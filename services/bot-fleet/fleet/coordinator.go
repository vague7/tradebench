package fleet

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bench/bot-fleet/bot"
	"github.com/bench/bot-fleet/config"
	"github.com/bench/shared/types"
	"github.com/google/uuid"
)

// botEntry tracks an active bot's cancel function for individual bot lifecycle management.
type botEntry struct {
	id     string
	cancel context.CancelFunc
}

// Coordinator manages a dynamic pool of bot goroutines — adding and removing them as
// each phase demands — and orchestrates adversarial scenario injection.
type Coordinator struct {
	cfg          *config.Config
	profile      LoadProfile
	emitCh       chan<- types.BotEvent
	activeBots   []botEntry
	wg           sync.WaitGroup
	submissionID string
	targetURL    string
}

// NewCoordinator creates a new Coordinator.
func NewCoordinator(cfg *config.Config, profile LoadProfile, emitCh chan<- types.BotEvent) *Coordinator {
	return &Coordinator{
		cfg:     cfg,
		profile: profile,
		emitCh:  emitCh,
	}
}

// Run is the top-level benchmark executor. It iterates through load profile phases,
// executing each in order. After all phases complete or ctx is cancelled, it stops
// all bots, waits for them to exit, and closes the emit channel.
func (c *Coordinator) Run(ctx context.Context, submissionID, targetURL string) error {
	c.submissionID = submissionID
	c.targetURL = targetURL

	slog.Info("benchmark total duration", "submissionId", submissionID, "totalDuration", c.profile.TotalDuration().String())

	// Create a child context for all bot goroutines.
	// Cancelling botCtx stops all bots; cancelling parent ctx cascades into botCtx.
	botCtx, botCancel := context.WithCancel(ctx)
	defer botCancel()

	var runErr error
	for _, phase := range c.profile {
		select {
		case <-ctx.Done():
			runErr = ctx.Err()
			goto cleanup
		default:
		}

		slog.Info("phase started",
			"phase", phase.Name,
			"targetBots", phase.TargetBotCount,
			"durationSec", phase.DurationSec,
		)

		if err := c.executePhase(botCtx, phase); err != nil {
			if ctx.Err() != nil {
				runErr = ctx.Err()
				goto cleanup
			}
			runErr = err
			goto cleanup
		}

		slog.Info("phase completed",
			"phase", phase.Name,
			"actualBotsAtEnd", len(c.activeBots),
		)
	}

cleanup:
	// Cancel all bot contexts
	botCancel()

	// Wait for all bot goroutines to exit
	c.wg.Wait()

	// Close the emit channel — signals the streamer to flush and close
	close(c.emitCh)

	slog.Info("benchmark completed",
		"submissionId", submissionID,
	)

	return runErr
}

// executePhase executes one phase of the load profile.
func (c *Coordinator) executePhase(ctx context.Context, phase Phase) error {
	currentCount := len(c.activeBots)
	targetCount := phase.TargetBotCount
	delta := targetCount - currentCount
	duration := time.Duration(phase.DurationSec) * time.Second

	if phase.LinearRamp && phase.DurationSec > 0 && delta != 0 {
		// Linear ramp: spread spawn/kill operations evenly over the phase duration
		return c.linearRamp(ctx, phase, currentCount, targetCount, duration)
	}

	// Instant: immediately spawn or kill to reach target, then hold for duration
	if delta > 0 {
		c.spawnBots(ctx, delta)
	} else if delta < 0 {
		c.killBots(-delta)
	}

	// For sustained phase: inject adversarial scenarios at 30-second intervals
	if phase.Name == PhaseSustained {
		return c.sustainedWithAdversarial(ctx, duration)
	}

	// Hold at current count for the phase duration
	return c.holdForDuration(ctx, duration)
}

// linearRamp spreads bot spawn/kill operations evenly over the phase duration.
func (c *Coordinator) linearRamp(ctx context.Context, phase Phase, currentCount, targetCount int, duration time.Duration) error {
	delta := targetCount - currentCount
	isSpawning := delta > 0
	if !isSpawning {
		delta = -delta
	}

	// Use 1-second tick intervals
	tickInterval := time.Second
	totalTicks := int(duration / tickInterval)
	if totalTicks <= 0 {
		totalTicks = 1
	}

	botsPerTick := delta / totalTicks
	remainder := delta % totalTicks

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for tick := 0; tick < totalTicks; tick++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		count := botsPerTick
		if tick < remainder {
			count++
		}

		if isSpawning {
			c.spawnBots(ctx, count)
		} else {
			c.killBots(count)
		}
	}

	return nil
}

// sustainedWithAdversarial holds at current bot count while injecting adversarial
// scenarios at dynamically calculated intervals during the sustained phase.
func (c *Coordinator) sustainedWithAdversarial(ctx context.Context, duration time.Duration) error {
	intervalSec := c.cfg.SustainedDuration / 3
	if intervalSec < 1 {
		intervalSec = 1 // Floor at 1 second to prevent panic/spam
	}
	injectionInterval := time.Duration(intervalSec) * time.Second
	
	ticker := time.NewTicker(injectionInterval)
	defer ticker.Stop()

	deadline := time.After(duration)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return nil
		case <-ticker.C:
			// Inject all five adversarial scenarios concurrently
			c.injectAdversarialScenarios(ctx)
		}
	}
}

// injectAdversarialScenarios launches all five adversarial scenarios in goroutines.
// Each scenario runs once per injection cycle; they run concurrently.
func (c *Coordinator) injectAdversarialScenarios(ctx context.Context) {
	c.wg.Add(5)
	go func() { defer c.wg.Done(); bot.SimultaneousCrossingOrders(ctx, c.submissionID, c.targetURL, c.cfg.TimeoutMs, c.emitCh) }()
	go func() { defer c.wg.Done(); bot.RapidCancelReplace(ctx, c.submissionID, c.targetURL, c.cfg.TimeoutMs, c.emitCh) }()
	go func() { defer c.wg.Done(); bot.OrderBookFlood(ctx, c.submissionID, c.targetURL, c.cfg.TimeoutMs, c.emitCh) }()
	go func() { defer c.wg.Done(); bot.FatFinger(ctx, c.submissionID, c.targetURL, c.cfg.TimeoutMs, c.emitCh) }()
	go func() { defer c.wg.Done(); bot.StaleCancel(ctx, c.submissionID, c.targetURL, c.cfg.TimeoutMs, c.emitCh) }()
}

// holdForDuration waits for the specified duration while respecting ctx cancellation.
func (c *Coordinator) holdForDuration(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}

// spawnBots creates and launches the specified number of new bot goroutines.
func (c *Coordinator) spawnBots(ctx context.Context, count int) {
	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		botID := uuid.NewString()
		perBotCtx, perBotCancel := context.WithCancel(ctx)

		b := bot.NewBot(botID, c.submissionID, c.targetURL, c.cfg.TimeoutMs, c.emitCh)

		c.activeBots = append(c.activeBots, botEntry{
			id:     botID,
			cancel: perBotCancel,
		})

		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			b.Run(perBotCtx)
		}()
	}
}

// killBots cancels the specified number of bots from the active pool.
// Selects from the end of the slice for efficiency (no shifting needed).
func (c *Coordinator) killBots(count int) {
	if count > len(c.activeBots) {
		count = len(c.activeBots)
	}
	for i := 0; i < count; i++ {
		idx := len(c.activeBots) - 1
		c.activeBots[idx].cancel()
		c.activeBots = c.activeBots[:idx]
	}
}
