package store

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	benchtypes "github.com/bench/shared/types"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return &PostgresStore{pool: pool}, nil
}

// ── Submissions ──────────────────────────────────────────────────────────────

func (s *PostgresStore) CreateSubmission(ctx context.Context, teamName, teamToken, zipHash, zipPath string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO submissions (team_name, team_token, zip_hash, zip_path, status, uploaded_at)
		VALUES ($1, $2, $3, $4, 'UPLOADED', NOW())
		RETURNING id`,
		teamName, teamToken, zipHash, zipPath,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("postgres: create submission: %w", err)
	}
	return id, nil
}

func (s *PostgresStore) GetSubmissionByHash(ctx context.Context, zipHash string) (SubmissionRecord, bool, error) {
	var r SubmissionRecord
	err := s.pool.QueryRow(ctx, `
		SELECT id, team_name, status, error_message, uploaded_at,
		       benchmark_start, benchmark_end, image_tag, container_id, container_port
		FROM submissions WHERE zip_hash = $1
		ORDER BY uploaded_at DESC LIMIT 1`,
		zipHash,
	).Scan(
		&r.ID, &r.TeamName, &r.Status, &r.ErrorMessage, &r.UploadedAt,
		&r.BenchmarkStart, &r.BenchmarkEnd, &r.ImageTag, &r.ContainerID, &r.ContainerPort,
	)
	if err != nil {
		if isNoRows(err) {
			return SubmissionRecord{}, false, nil
		}
		return SubmissionRecord{}, false, fmt.Errorf("postgres: get by hash: %w", err)
	}
	return r, true, nil
}

func (s *PostgresStore) GetSubmission(ctx context.Context, id string) (SubmissionRecord, bool, error) {
	var r SubmissionRecord
	err := s.pool.QueryRow(ctx, `
		SELECT id, team_name, status, error_message, uploaded_at,
		       benchmark_start, benchmark_end, image_tag, container_id, container_port
		FROM submissions WHERE id = $1`,
		id,
	).Scan(
		&r.ID, &r.TeamName, &r.Status, &r.ErrorMessage, &r.UploadedAt,
		&r.BenchmarkStart, &r.BenchmarkEnd, &r.ImageTag, &r.ContainerID, &r.ContainerPort,
	)
	if err != nil {
		if isNoRows(err) {
			return SubmissionRecord{}, false, nil
		}
		return SubmissionRecord{}, false, fmt.Errorf("postgres: get submission: %w", err)
	}
	return r, true, nil
}

func (s *PostgresStore) UpdateStatus(ctx context.Context, id string, status benchtypes.SubmissionStatus, errMsg string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE submissions SET status = $1, error_message = $2 WHERE id = $3`,
		string(status), errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("postgres: update status: %w", err)
	}
	return nil
}

func (s *PostgresStore) SetImageTag(ctx context.Context, id, imageTag string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE submissions SET image_tag = $1, built_at = NOW() WHERE id = $2`,
		imageTag, id,
	)
	return err
}

func (s *PostgresStore) SetContainerInfo(ctx context.Context, id, containerID string, port int) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE submissions SET container_id = $1, container_port = $2, started_at = NOW() WHERE id = $3`,
		containerID, port, id,
	)
	return err
}

func (s *PostgresStore) SetBenchmarkStart(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE submissions SET benchmark_start = NOW(), status = 'BENCHMARKING' WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("postgres: set benchmark start: %w", err)
	}
	return nil
}

func (s *PostgresStore) SetBenchmarkEnd(ctx context.Context, id string, status benchtypes.SubmissionStatus, errMsg string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE submissions SET benchmark_end = NOW(), scored_at = NOW(), status = $1, error_message = $2 WHERE id = $3`,
		string(status), errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("postgres: set benchmark end: %w", err)
	}
	return nil
}

// ── MetricSnapshots ──────────────────────────────────────────────────────────

func (s *PostgresStore) InsertMetricSnapshot(ctx context.Context, snap benchtypes.MetricSnapshot) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO metric_snapshots
		  (submission_id, window_end, p50_latency_ms, p90_latency_ms, p99_latency_ms,
		   tps, success_count, failure_count, timeout_count, correctness)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		snap.SubmissionID, snap.WindowEnd,
		snap.P50LatencyMs, snap.P90LatencyMs, snap.P99LatencyMs,
		snap.TPS, snap.SuccessCount, snap.FailureCount, snap.TimeoutCount,
		snap.CorrectnessScore,
	)
	if err != nil {
		return fmt.Errorf("postgres: insert metric snapshot: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetLatestSnapshot(ctx context.Context, submissionID string) (benchtypes.MetricSnapshot, bool, error) {
	var snap benchtypes.MetricSnapshot
	err := s.pool.QueryRow(ctx, `
		SELECT submission_id, window_end, p50_latency_ms, p90_latency_ms, p99_latency_ms,
		       tps, success_count, failure_count, timeout_count, correctness
		FROM metric_snapshots
		WHERE submission_id = $1
		ORDER BY window_end DESC LIMIT 1`,
		submissionID,
	).Scan(
		&snap.SubmissionID, &snap.WindowEnd,
		&snap.P50LatencyMs, &snap.P90LatencyMs, &snap.P99LatencyMs,
		&snap.TPS, &snap.SuccessCount, &snap.FailureCount, &snap.TimeoutCount,
		&snap.CorrectnessScore,
	)
	if err != nil {
		if isNoRows(err) {
			return benchtypes.MetricSnapshot{}, false, nil
		}
		return benchtypes.MetricSnapshot{}, false, fmt.Errorf("postgres: get latest snapshot: %w", err)
	}
	return snap, true, nil
}

// ── Scores ───────────────────────────────────────────────────────────────────

func (s *PostgresStore) InsertScore(ctx context.Context, score benchtypes.Score) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO scores
		  (submission_id, throughput_score, latency_score, correctness_score,
		   final_score, is_disqualified, disqualify_reason, computed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		score.SubmissionID, score.ThroughputScore, score.LatencyScore,
		score.CorrectnessScore, score.FinalScore,
		score.IsDisqualified, score.DisqualifyReason, score.ComputedAt,
	)
	if err != nil {
		return fmt.Errorf("postgres: insert score: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetLatestScore(ctx context.Context, submissionID string) (benchtypes.Score, bool, error) {
	var score benchtypes.Score
	err := s.pool.QueryRow(ctx, `
		SELECT submission_id, throughput_score, latency_score, correctness_score,
		       final_score, is_disqualified, disqualify_reason, computed_at
		FROM scores
		WHERE submission_id = $1
		ORDER BY computed_at DESC LIMIT 1`,
		submissionID,
	).Scan(
		&score.SubmissionID, &score.ThroughputScore, &score.LatencyScore,
		&score.CorrectnessScore, &score.FinalScore,
		&score.IsDisqualified, &score.DisqualifyReason, &score.ComputedAt,
	)
	if err != nil {
		if isNoRows(err) {
			return benchtypes.Score{}, false, nil
		}
		return benchtypes.Score{}, false, fmt.Errorf("postgres: get latest score: %w", err)
	}
	return score, true, nil
}

// ── Leaderboard ──────────────────────────────────────────────────────────────

func (s *PostgresStore) ListLeaderboard(ctx context.Context) ([]benchtypes.LeaderboardEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			sub.team_name,
			sub.status,
			COALESCE(sc.final_score, 0),
			COALESCE(sc.correctness_score, 0),
			COALESCE(sc.is_disqualified, false),
			COALESCE(snap.tps, 0),
			COALESCE(snap.p99_latency_ms, 0),
			COALESCE(snap.success_count, 0),
			COALESCE(snap.failure_count, 0),
			COALESCE(snap.timeout_count, 0)
		FROM submissions sub
		LEFT JOIN LATERAL (
			SELECT final_score, correctness_score, is_disqualified
			FROM scores WHERE submission_id = sub.id
			ORDER BY computed_at DESC LIMIT 1
		) sc ON true
		LEFT JOIN LATERAL (
			SELECT tps, p99_latency_ms, success_count, failure_count, timeout_count
			FROM metric_snapshots WHERE submission_id = sub.id
			ORDER BY window_end DESC LIMIT 1
		) snap ON true
		WHERE sub.status != 'UPLOADED'`)
	if err != nil {
		return nil, fmt.Errorf("postgres: list leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []benchtypes.LeaderboardEntry
	for rows.Next() {
		var e benchtypes.LeaderboardEntry
		var successCount, failureCount, timeoutCount int64
		var isDisqualified bool

		if err := rows.Scan(
			&e.TeamName, &e.Status,
			&e.FinalScore, &e.CorrectnessScore, &isDisqualified,
			&e.TPS, &e.P99LatencyMs,
			&successCount, &failureCount, &timeoutCount,
		); err != nil {
			return nil, fmt.Errorf("postgres: leaderboard scan: %w", err)
		}

		denom := successCount + failureCount + timeoutCount
		if denom > 0 {
			e.ErrorRate = float64(failureCount+timeoutCount) / float64(denom) * 100
		}
		e.FinalScore *= 100
		e.CorrectnessScore *= 100

		e.IsDisqualified = isDisqualified
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: leaderboard rows: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDisqualified != entries[j].IsDisqualified {
			return !entries[i].IsDisqualified
		}
		if entries[i].FinalScore != entries[j].FinalScore {
			return entries[i].FinalScore > entries[j].FinalScore
		}
		if entries[i].CorrectnessScore != entries[j].CorrectnessScore {
			return entries[i].CorrectnessScore > entries[j].CorrectnessScore
		}
		return entries[i].P99LatencyMs < entries[j].P99LatencyMs
	})
	for i := range entries {
		entries[i].Rank = i + 1
	}
	return entries, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

type SubmissionRecord struct {
	ID             string
	TeamName       string
	Status         benchtypes.SubmissionStatus
	ErrorMessage   *string
	UploadedAt     time.Time
	BenchmarkStart *time.Time
	BenchmarkEnd   *time.Time
	ImageTag       *string
	ContainerID    *string
	ContainerPort  *int
}

func isNoRows(err error) bool {
	return err != nil && err.Error() == "no rows in result set"
}

func (s *PostgresStore) UpdateZipPath(ctx context.Context, id, zipPath string) error {
_, err := s.pool.Exec(ctx, `UPDATE submissions SET zip_path = $1 WHERE id = $2`, zipPath, id)
if err != nil {
return fmt.Errorf("postgres: update zip path: %w", err)
}
return nil
}
