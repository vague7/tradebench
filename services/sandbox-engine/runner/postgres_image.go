package runner

import (
	"context"
	"fmt"
)

// UpdateImageAndContainer writes the image_tag and container_id columns back to
// the submissions table after a successful build and spawn. These fields are
// required by the PRD schema (migrations/001_initial_schema.sql) and are read
// by the admin stop endpoint and the SandboxEngine gRPC GetStatus response.
func (p *PostgresStatusUpdater) UpdateImageAndContainer(
	ctx context.Context,
	submissionID, imageTag, containerID string,
	containerPort int,
) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE submissions
		    SET image_tag = $1, container_id = $2, container_port = $3
		  WHERE id = $4`,
		imageTag, containerID, containerPort, submissionID,
	)
	if err != nil {
		return fmt.Errorf("postgres: UpdateImageAndContainer for submission %s: %w", submissionID, err)
	}
	return nil
}
