package runner

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetStatus reads the current status and error_message for a submission.
// Returns ("", "", nil) when the row is not found (caller interprets as NotFound).
func (p *PostgresStatusUpdater) GetStatus(ctx context.Context, submissionID string) (statusStr, errMsg string, err error) {
	row := p.pool.QueryRow(ctx,
		`SELECT status, COALESCE(error_message, '') FROM submissions WHERE id = $1`,
		submissionID,
	)
	if scanErr := row.Scan(&statusStr, &errMsg); scanErr != nil {
		if scanErr == pgx.ErrNoRows {
			return "", "", nil
		}
		return "", "", fmt.Errorf("postgres: GetStatus for submission %s: %w", submissionID, scanErr)
	}
	return statusStr, errMsg, nil
}
