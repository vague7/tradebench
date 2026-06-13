package aggregate

import (
	"context"
	"log/slog"
	"time"

	"github.com/bench/shared/types"
	"github.com/bench/telemetry-ingester/ingest"
	"github.com/bench/telemetry-ingester/scoring"
	"github.com/bench/telemetry-ingester/store"
	"github.com/bench/telemetry-ingester/correctness"
)

// WindowManager manages the 10-second rolling window for metric aggregation.
// It drains events from the ring buffer on each tick, groups by SubmissionID,
// computes latency percentiles and TPS, and persists MetricSnapshots to Postgres.
type WindowManager struct {
	windowSec int
	buf       *ingest.RingBuffer
	store     *store.PostgresStore
	scorer    *scoring.Engine
	validator *correctness.Validator
}

// NewWindowManager creates a new WindowManager.
func NewWindowManager(windowSec int, buf *ingest.RingBuffer, store *store.PostgresStore, scorer *scoring.Engine, validator *correctness.Validator) *WindowManager {
	return &WindowManager{
		windowSec: windowSec,
		buf:       buf,
		store:     store,
		scorer:    scorer,
		validator: validator,
	}
}

// Run is the main loop. This is a blocking call intended to run in a goroutine.
// It ticks every windowSec seconds, drains all available events from the buffer,
// groups them by SubmissionID, and computes + persists metrics for each group.
// It exits cleanly when ctx is cancelled.
func (wm *WindowManager) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(wm.windowSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("window manager shutting down")
			return
		case <-ticker.C:
			windowEnd := time.Now().UTC()

			// Drain all available events from the buffer.
			events := wm.buf.Drain(ringBufferDrainSize)

			if len(events) == 0 {
				continue
			}

			// Group events by SubmissionID.
			grouped := make(map[string][]types.BotEvent)
			for _, e := range events {
				grouped[e.SubmissionID] = append(grouped[e.SubmissionID], e)
			}

			// Compute and persist metrics for each submission.
			for submissionID, subEvents := range grouped {
				wm.computeAndPersist(ctx, submissionID, subEvents, windowEnd)
			}
		}
	}
}

// ringBufferDrainSize is the maximum number of events to drain per tick.
// Set higher than the buffer capacity to ensure we drain everything available.
const ringBufferDrainSize = 20_000

// computeAndPersist computes latency percentiles, TPS, and success/failure/timeout counts,
// then persists a MetricSnapshot to the database and triggers scoring.
func (wm *WindowManager) computeAndPersist(ctx context.Context, submissionID string, events []types.BotEvent, windowEnd time.Time) {
	// Compute latencies.
	latencies := make([]float64, 0, len(events))
	for _, e := range events {
		latencyMs := float64(e.AckedAt.Sub(e.SentAt).Milliseconds())
		latencies = append(latencies, latencyMs)
	}

	// Compute percentiles.
	p50 := P50(latencies)
	p90 := P90(latencies)
	p99 := P99(latencies)

	// Compute TPS.
	tps := Compute(events, wm.windowSec)

	// Count success, failure, and timeout events.
	var successCount, failureCount, timeoutCount int64
	for _, e := range events {
		switch {
		case e.HTTPStatus == 200:
			successCount++
		case e.HTTPStatus >= 400 && e.HTTPStatus != 0:
			failureCount++
		case e.HTTPStatus == 0:
			timeoutCount++
		}
	}

	correctnessScore, err := wm.validator.Validate(ctx, events)
	if err != nil {
		slog.Warn("correctness validation failed, defaulting to 0.0", "submissionId", submissionID, "err", err)
		correctnessScore = 0.0
	}

	snapshot := types.MetricSnapshot{
		SubmissionID:     submissionID,
		WindowEnd:        windowEnd,
		P50LatencyMs:     p50,
		P90LatencyMs:     p90,
		P99LatencyMs:     p99,
		TPS:              tps,
		SuccessCount:     successCount,
		FailureCount:     failureCount,
		TimeoutCount:     timeoutCount,
		CorrectnessScore: correctnessScore,
	}

	if err := wm.store.InsertMetricSnapshot(ctx, snapshot); err != nil {
		slog.Error("failed to insert metric snapshot",
			"submissionId", snapshot.SubmissionID,
			"err", err,
		)
		return
	}

	// Trigger scoring after successful MetricSnapshot insert.
	if err := wm.scorer.Score(ctx, snapshot); err != nil {
		slog.Error("scoring engine failed",
			"submissionId", snapshot.SubmissionID,
			"err", err,
		)
		// A scoring failure must not crash the window manager or stop telemetry collection.
	}

	slog.Info("window tick processed",
		"submissionId", submissionID,
		"eventCount", len(events),
		"p99Ms", snapshot.P99LatencyMs,
		"tps", snapshot.TPS,
	)
}
