-- Add team_name column to scores table
ALTER TABLE scores ADD COLUMN IF NOT EXISTS team_name TEXT NOT NULL DEFAULT '';

-- Add unique constraint on submission_id so ON CONFLICT (submission_id) works.
-- Scores are upserted on every window tick, not inserted fresh each time.
ALTER TABLE scores DROP CONSTRAINT IF EXISTS scores_submission_id_unique;
ALTER TABLE scores ADD CONSTRAINT scores_submission_id_unique UNIQUE (submission_id);
