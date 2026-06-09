package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bench/shared/types"
	_ "github.com/jackc/pgx/v5/stdlib"
)

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

// Close closes the underlying database connection.
func (s *PostgresStore) Close() error {
	return s.db.Close()
}
