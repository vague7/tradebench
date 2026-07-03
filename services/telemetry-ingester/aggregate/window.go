package aggregate

import (
	"context"
	"log/slog"
	"runtime"
	"sync"
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
	// Start async validator worker pool with WaitGroup for graceful drain.
	var workerWg sync.WaitGroup
	jobCh := make(chan ValidationJob, 1000)
	workerCount := runtime.NumCPU()
	if workerCount < 2 {
		workerCount = 2
	}
	for i := 0; i < workerCount; i++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			for job := range jobCh {
				wm.processValidationJob(ctx, job)
			}
		}()
	}

	ticker := time.NewTicker(time.Duration(wm.windowSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("window manager shutting down, draining validation workers")
			close(jobCh)
			workerWg.Wait()
			slog.Info("window manager shutdown complete")
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

			// Compute metrics and dispatch to worker pool for validation and scoring.
			for submissionID, subEvents := range grouped {
				wm.dispatchValidation(ctx, jobCh, submissionID, subEvents, windowEnd)
			}
		}
	}
}

// ringBufferDrainSize is the maximum number of events to drain per tick.
// Set higher than the buffer capacity to ensure we drain everything available.
const ringBufferDrainSize = 200_000

// ValidationJob encapsulates the work for the async validation pool.
type ValidationJob struct {
	events   []types.BotEvent
	snapshot types.MetricSnapshot
}

// dispatchValidation computes latency percentiles and TPS,
// then prepares a ValidationJob and sends it to the async worker pool.
func (wm *WindowManager) dispatchValidation(ctx context.Context, jobCh chan<- ValidationJob, submissionID string, events []types.BotEvent, windowEnd time.Time) {
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
	}

	select {
	case jobCh <- ValidationJob{events: events, snapshot: snapshot}:
	case <-time.After(5 * time.Second):
		slog.Error("validation job channel full after 5s, dropping window metrics", "submissionId", submissionID)
	}
}

// processValidationJob executes the C++ validator, inserts the snapshot, and triggers scoring.
func (wm *WindowManager) processValidationJob(ctx context.Context, job ValidationJob) {
	correctnessScore, err := wm.validator.Validate(ctx, job.events)
	if err != nil {
		slog.Warn("correctness validation failed, defaulting to 0.0", "submissionId", job.snapshot.SubmissionID, "err", err)
		correctnessScore = 0.0
	}

	job.snapshot.CorrectnessScore = correctnessScore

	if err := wm.store.InsertMetricSnapshot(ctx, job.snapshot); err != nil {
		slog.Error("failed to insert metric snapshot",
			"submissionId", job.snapshot.SubmissionID,
			"err", err,
		)
		return
	}

	// Trigger scoring after successful MetricSnapshot insert.
	if err := wm.scorer.Score(ctx, job.snapshot); err != nil {
		slog.Error("scoring engine failed",
			"submissionId", job.snapshot.SubmissionID,
			"err", err,
		)
		// A scoring failure must not crash the window manager or stop telemetry collection.
	}

	slog.Info("window tick processed async",
		"submissionId", job.snapshot.SubmissionID,
		"eventCount", len(job.events),
		"p99Ms", job.snapshot.P99LatencyMs,
		"tps", job.snapshot.TPS,
	)
}
