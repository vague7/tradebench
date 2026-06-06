-- Enable TimescaleDB
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- submissions
CREATE TABLE IF NOT EXISTS submissions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_name        TEXT NOT NULL,
    team_token       TEXT NOT NULL,
    zip_hash         TEXT NOT NULL,
    zip_path         TEXT NOT NULL,
    image_tag        TEXT,
    container_id     TEXT,
    container_port   INT,
    status           TEXT NOT NULL DEFAULT 'UPLOADED',
    error_message    TEXT,
    uploaded_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    built_at         TIMESTAMPTZ,
    started_at       TIMESTAMPTZ,
    benchmark_start  TIMESTAMPTZ,
    benchmark_end    TIMESTAMPTZ,
    scored_at        TIMESTAMPTZ
);

-- metric_snapshots (time-series)
CREATE TABLE IF NOT EXISTS metric_snapshots (
    id               BIGSERIAL,
    submission_id    UUID NOT NULL REFERENCES submissions(id),
    window_end       TIMESTAMPTZ NOT NULL,
    p50_latency_ms   DOUBLE PRECISION,
    p90_latency_ms   DOUBLE PRECISION,
    p99_latency_ms   DOUBLE PRECISION,
    tps              DOUBLE PRECISION,
    success_count    BIGINT,
    failure_count    BIGINT,
    timeout_count    BIGINT,
    correctness      DOUBLE PRECISION,
    PRIMARY KEY (id, window_end)
);
SELECT create_hypertable('metric_snapshots', 'window_end', if_not_exists => TRUE);

-- scores
CREATE TABLE IF NOT EXISTS scores (
    id                  BIGSERIAL PRIMARY KEY,
    submission_id       UUID NOT NULL REFERENCES submissions(id),
    throughput_score    DOUBLE PRECISION,
    latency_score       DOUBLE PRECISION,
    correctness_score   DOUBLE PRECISION,
    final_score         DOUBLE PRECISION,
    is_disqualified     BOOLEAN DEFAULT FALSE,
    disqualify_reason   TEXT,
    computed_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
