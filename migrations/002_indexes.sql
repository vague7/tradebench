CREATE INDEX IF NOT EXISTS idx_submissions_status
    ON submissions(status);

CREATE INDEX IF NOT EXISTS idx_submissions_team_token
    ON submissions(team_token);

CREATE INDEX IF NOT EXISTS idx_metric_snapshots_submission
    ON metric_snapshots(submission_id, window_end DESC);

CREATE INDEX IF NOT EXISTS idx_scores_submission
    ON scores(submission_id);

CREATE INDEX IF NOT EXISTS idx_scores_final_score
    ON scores(final_score DESC)
    WHERE is_disqualified = FALSE;
