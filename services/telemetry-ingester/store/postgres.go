package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bench/shared/types"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

// ErrNoSnapshot is returned when no metric snapshot exists for a given submission.
var ErrNoSnapshot = errors.New("store: no metric snapshot found")

// PostgresStore provides access to the PostgreSQL database for the telemetry-ingester.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore opens a database connection using the pgx driver and verifies connectivity.
func NewPostgresStore(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("store: failed to ping database: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

// InsertMetricSnapshot inserts a single MetricSnapshot row into the metric_snapshots table.
func (s *PostgresStore) InsertMetricSnapshot(ctx context.Context, snap types.MetricSnapshot) error {
	const query = `INSERT INTO metric_snapshots
		(submission_id, window_end, p50_latency_ms, p90_latency_ms, p99_latency_ms,
		 tps, success_count, failure_count, timeout_count, correctness)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`

	_, err := s.db.ExecContext(ctx, query,
		snap.SubmissionID,
		snap.WindowEnd,
		snap.P50LatencyMs,
		snap.P90LatencyMs,
		snap.P99LatencyMs,
		snap.TPS,
		snap.SuccessCount,
		snap.FailureCount,
		snap.TimeoutCount,
		snap.CorrectnessScore,
	)
	if err != nil {
		return fmt.Errorf("store: InsertMetricSnapshot for submission %s: %w", snap.SubmissionID, err)
	}

	return nil
}

// InsertScore inserts a single Score row into the scores table.
func (s *PostgresStore) InsertScore(ctx context.Context, score types.Score) error {
	const query = `INSERT INTO scores
		(submission_id, team_name, throughput_score, latency_score, correctness_score,
		 final_score, is_disqualified, disqualify_reason, computed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (submission_id) DO UPDATE SET
		  team_name         = EXCLUDED.team_name,
		  throughput_score  = EXCLUDED.throughput_score,
		  latency_score     = EXCLUDED.latency_score,
		  correctness_score = EXCLUDED.correctness_score,
		  final_score       = EXCLUDED.final_score,
		  is_disqualified   = EXCLUDED.is_disqualified,
		  disqualify_reason = EXCLUDED.disqualify_reason,
		  computed_at       = EXCLUDED.computed_at`

	_, err := s.db.ExecContext(ctx, query,
		score.SubmissionID,
		score.TeamName,
		score.ThroughputScore,
		score.LatencyScore,
		score.CorrectnessScore,
		score.FinalScore,
		score.IsDisqualified,
		score.DisqualifyReason,
		score.ComputedAt,
	)
	if err != nil {
		return fmt.Errorf("store: InsertScore for submission %s: %w", score.SubmissionID, err)
	}
	return nil
}

// GetLatestMetricSnapshot returns the most recent MetricSnapshot for the given submission.
// Returns ErrNoSnapshot if no rows exist.
func (s *PostgresStore) GetLatestMetricSnapshot(ctx context.Context, submissionID string) (types.MetricSnapshot, error) {
	const query = `SELECT submission_id, window_end, p50_latency_ms, p90_latency_ms, p99_latency_ms,
		tps, success_count, failure_count, timeout_count, correctness
		FROM metric_snapshots
		WHERE submission_id = $1
		ORDER BY window_end DESC LIMIT 1`

	var snap types.MetricSnapshot
	err := s.db.QueryRowContext(ctx, query, submissionID).Scan(
		&snap.SubmissionID,
		&snap.WindowEnd,
		&snap.P50LatencyMs,
		&snap.P90LatencyMs,
		&snap.P99LatencyMs,
		&snap.TPS,
		&snap.SuccessCount,
		&snap.FailureCount,
		&snap.TimeoutCount,
		&snap.CorrectnessScore, // DB column "correctness" maps to struct field CorrectnessScore
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.MetricSnapshot{}, ErrNoSnapshot
		}
		return types.MetricSnapshot{}, fmt.Errorf("store: GetLatestMetricSnapshot for submission %s: %w", submissionID, err)
	}

	return snap, nil
}

// GetSubmissionTeamName returns the team_name for the given submission ID.
func (s *PostgresStore) GetSubmissionTeamName(ctx context.Context, submissionID string) (string, error) {
	const query = `SELECT team_name FROM submissions WHERE id = $1`

	var teamName string
	err := s.db.QueryRowContext(ctx, query, submissionID).Scan(&teamName)
	if err != nil {
		return "", fmt.Errorf("store: GetSubmissionTeamName for submission %s: %w", submissionID, err)
	}

	return teamName, nil
}

// Close closes the underlying database connection.
func (s *PostgresStore) Close() error {
	return s.db.Close()
}

// ---------------------------------------------------------------------------
// RedisStore provides access to Redis for the telemetry-ingester.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a new RedisStore connected to the given address.
func NewRedisStore(addr string) *RedisStore {
	return &RedisStore{
		client: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
	}
}

// SetWithTTL writes a key-value pair to Redis with the given TTL.
func (r *RedisStore) SetWithTTL(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("store: RedisStore SetWithTTL for key %s: %w", key, err)
	}
	return nil
}

// Publish sends a message to a Redis Pub/Sub channel.
func (r *RedisStore) Publish(ctx context.Context, channel string, message []byte) error {
	if err := r.client.Publish(ctx, channel, message).Err(); err != nil {
		return fmt.Errorf("store: RedisStore Publish to %s: %w", channel, err)
	}
	return nil
}

// Close closes the underlying Redis connection.
func (r *RedisStore) Close() error {
	return r.client.Close()
}
