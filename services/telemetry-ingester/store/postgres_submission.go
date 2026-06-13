package store

import (
	"context"
	"fmt"
	"time"
)

// UpdateSubmissionStatus writes status (and optional scored_at timestamp) back to
// the submissions table. Called by the scoring engine after a score is persisted.
func (s *PostgresStore) UpdateSubmissionStatus(ctx context.Context, submissionID, status string) error {
	var err error
	if status == "SCORED" {
		_, err = s.db.ExecContext(ctx,
			`UPDATE submissions SET status = $1, scored_at = $2 WHERE id = $3`,
			status, time.Now().UTC(), submissionID,
		)
	} else {
		_, err = s.db.ExecContext(ctx,
			`UPDATE submissions SET status = $1 WHERE id = $2`,
			status, submissionID,
		)
	}
	if err != nil {
		return fmt.Errorf("store: UpdateSubmissionStatus(%s) for submission %s: %w", status, submissionID, err)
	}
	return nil
}
