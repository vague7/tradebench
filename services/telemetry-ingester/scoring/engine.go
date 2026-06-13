package scoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/bench/shared/types"
	"github.com/bench/telemetry-ingester/config"
	"github.com/bench/telemetry-ingester/store"
)

// Engine computes composite scores from MetricSnapshots and persists results
// to the scores table and Redis for the SSE leaderboard pipeline.
type Engine struct {
	cfg   *config.Config
	store *store.PostgresStore
	redis *store.RedisStore
}

// NewEngine creates a new scoring Engine with the given configuration and stores.
func NewEngine(cfg *config.Config, store *store.PostgresStore, redis *store.RedisStore) *Engine {
	return &Engine{
		cfg:   cfg,
		store: store,
		redis: redis,
	}
}

// computeScores applies the composite score formula (FR-6.1) to a MetricSnapshot.
// This is an unexported helper to make the formula testable without a live Engine instance.
//
// Formula:
//
//	throughputScore  = min(TPS / targetTPS, 1.0)
//	latencyScore     = max(0.0, 1.0 - P99LatencyMs / maxP99Ms)
//	correctnessScore = CorrectnessScore / 100.0  (normalise from 0–100 to 0–1)
//	finalScore       = 0.40 * throughputScore + 0.40 * latencyScore + 0.20 * correctnessScore
func computeScores(snap types.MetricSnapshot, targetTPS, maxP99Ms float64) (throughput, latency, correctness, final float64) {
	throughput = math.Min(snap.TPS/targetTPS, 1.0)
	latency = math.Max(0.0, 1.0-snap.P99LatencyMs/maxP99Ms)
	correctness = snap.CorrectnessScore / 100.0
	final = 0.40*throughput + 0.40*latency + 0.20*correctness
	return
}

// Score runs the full scoring pipeline for a single MetricSnapshot:
//  1. Compute component scores using the composite formula.
//  2. Apply disqualification rule (CorrectnessScore < 30.0).
//  3. Fetch team name from the submissions table.
//  4. Persist the Score to the scores table.
//  5. Write submission status → SCORED in the submissions table. ← E2E blocker fix
//  6. Publish the MetricSnapshot JSON to Redis for SSE reads.
//  7. Log the outcome.
func (e *Engine) Score(ctx context.Context, snap types.MetricSnapshot) error {
	// 1. Apply composite formula.
	throughputScore, latencyScore, correctnessScore, finalScore := computeScores(snap, e.cfg.TargetTPS, e.cfg.MaxP99Ms)

	// 2. Apply disqualification rule.
	isDisqualified := false
	disqualifyReason := ""
	if snap.CorrectnessScore < 30.0 {
		isDisqualified = true
		disqualifyReason = "correctness_score below threshold (< 30)"
	}

	// 3. Fetch team name from DB.
	teamName, err := e.store.GetSubmissionTeamName(ctx, snap.SubmissionID)
	if err != nil {
		slog.Error("failed to get team name for scoring",
			"submissionId", snap.SubmissionID,
			"err", err,
		)
		return fmt.Errorf("scoring: get team name for submission %s: %w", snap.SubmissionID, err)
	}

	// 4. Construct and persist the Score.
	score := types.Score{
		SubmissionID:     snap.SubmissionID,
		TeamName:         teamName,
		ThroughputScore:  throughputScore,
		LatencyScore:     latencyScore,
		CorrectnessScore: correctnessScore,
		FinalScore:       finalScore,
		IsDisqualified:   isDisqualified,
		DisqualifyReason: disqualifyReason,
		ComputedAt:       time.Now().UTC(),
	}

	if err := e.store.InsertScore(ctx, score); err != nil {
		slog.Error("failed to insert score",
			"submissionId", snap.SubmissionID,
			"err", err,
		)
		return fmt.Errorf("scoring: insert score for submission %s: %w", snap.SubmissionID, err)
	}

	// 5. Mark submission as SCORED in the submissions table.
	// This is what the E2E test polls for via GET /api/submissions/:id/status.
	if err := e.store.UpdateSubmissionStatus(ctx, snap.SubmissionID, "SCORED"); err != nil {
		slog.Error("failed to mark submission as SCORED",
			"submissionId", snap.SubmissionID,
			"err", err,
		)
		// Score is already persisted — log but don't abort. The leaderboard
		// will still show the score; only the status polling will be stale.
		// This is recoverable: a retry on the next window tick will re-run Score().
	}

	// 6. Publish MetricSnapshot JSON to Redis (submission:{id}:snapshot, TTL 30s).
	jsonBytes, err := json.Marshal(snap)
	if err != nil {
		slog.Warn("failed to marshal metric snapshot for Redis",
			"submissionId", snap.SubmissionID,
			"err", err,
		)
	} else {
		redisKey := fmt.Sprintf("submission:%s:snapshot", snap.SubmissionID)
		if err := e.redis.SetWithTTL(ctx, redisKey, jsonBytes, 30*time.Second); err != nil {
			slog.Warn("failed to write metric snapshot to Redis",
				"submissionId", snap.SubmissionID,
				"key", redisKey,
				"err", err,
			)
			// Redis write failure must not abort the scoring pipeline.
		}
	}

	// 7. Log the outcome.
	slog.Info("score computed",
		"submissionId", snap.SubmissionID,
		"finalScore", finalScore,
		"throughputScore", throughputScore,
		"latencyScore", latencyScore,
		"correctnessScore", correctnessScore,
		"isDisqualified", isDisqualified,
	)

	return nil
}
