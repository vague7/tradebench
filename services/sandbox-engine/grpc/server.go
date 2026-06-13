// Package grpc implements the SandboxEngine gRPC service defined in
// shared/proto/sandbox.proto. It exposes two RPCs:
//
//   - GetStatus  – returns the current status of a submission container
//   - KillContainer – immediately stops and removes a running container
//
// Both operations are read from the in-process Registry first (O(1), no DB
// round-trip for running containers), falling back to PostgreSQL for
// non-running submissions.
package grpc

import (
	"context"
	"fmt"
	"log/slog"

	gen "github.com/bench/shared/proto/gen"
	"github.com/bench/sandbox-engine/runner"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SandboxServer implements gen.SandboxEngineServer (generated from sandbox.proto).
type SandboxServer struct {
	gen.UnimplementedSandboxEngineServer

	registry *runner.Registry
	db       *runner.PostgresStatusUpdater
	health   *runner.HealthChecker
}

// NewSandboxServer constructs a SandboxServer from the shared runtime components.
// No new connections are opened; the same registry, db, and health instances
// used by the consumer goroutine are reused here.
func NewSandboxServer(
	registry *runner.Registry,
	db *runner.PostgresStatusUpdater,
	health *runner.HealthChecker,
) *SandboxServer {
	return &SandboxServer{
		registry: registry,
		db:       db,
		health:   health,
	}
}

// GetStatus returns the current status of a submission's container.
// Running containers are served from the in-memory Registry without a DB query.
// All other statuses (BUILDING, FAILED, SCORED, etc.) are fetched from Postgres.
func (s *SandboxServer) GetStatus(ctx context.Context, req *gen.StatusRequest) (*gen.StatusResponse, error) {
	submissionID := req.GetSubmissionId()
	if submissionID == "" {
		return nil, status.Error(codes.InvalidArgument, "submission_id is required")
	}

	slog.Info("grpc: GetStatus", "submissionId", submissionID)

	// Fast path: check in-memory registry for live containers.
	if entry, ok := s.registry.Snapshot()[submissionID]; ok {
		return &gen.StatusResponse{
			SubmissionId: submissionID,
			Status:       "RUNNING",
			ContainerId:  entry.ContainerID,
		}, nil
	}

	// Slow path: query Postgres for non-running submissions.
	dbStatus, errMsg, err := s.db.GetStatus(ctx, submissionID)
	if err != nil {
		slog.Error("grpc: GetStatus db query failed", "submissionId", submissionID, "err", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("db query failed: %v", err))
	}
	if dbStatus == "" {
		return nil, status.Error(codes.NotFound, "submission not found")
	}

	return &gen.StatusResponse{
		SubmissionId: submissionID,
		Status:       dbStatus,
		ErrorMessage: errMsg,
	}, nil
}

// KillContainer stops and removes the contestant container for the given submission.
// It updates the DB to FAILED and removes the entry from the Registry so the
// watchdog loop does not attempt a double-kill. KillContainer is idempotent:
// if the container is already gone, it returns Ok=true without an error.
func (s *SandboxServer) KillContainer(ctx context.Context, req *gen.KillRequest) (*gen.SandboxAck, error) {
	submissionID := req.GetSubmissionId()
	if submissionID == "" {
		return nil, status.Error(codes.InvalidArgument, "submission_id is required")
	}

	slog.Info("grpc: KillContainer requested", "submissionId", submissionID)

	entry, ok := s.registry.Snapshot()[submissionID]
	if !ok {
		// Already gone — treat as idempotent success.
		slog.Warn("grpc: KillContainer: container not in registry (already gone?)", "submissionId", submissionID)
		return &gen.SandboxAck{Ok: true}, nil
	}

	if err := s.health.KillAndRemove(entry.ContainerID); err != nil {
		slog.Error("grpc: KillContainer failed",
			"submissionId", submissionID,
			"containerID", entry.ContainerID,
			"err", err,
		)
		return nil, status.Error(codes.Internal, fmt.Sprintf("kill failed: %v", err))
	}

	s.registry.Remove(submissionID)

	// Best-effort DB update — container is already dead even if this fails.
	if err := s.db.UpdateStatus(ctx, submissionID, "FAILED", "killed via gRPC KillContainer"); err != nil {
		slog.Error("grpc: KillContainer db update failed", "submissionId", submissionID, "err", err)
	}

	slog.Info("grpc: KillContainer succeeded",
		"submissionId", submissionID,
		"containerID", entry.ContainerID,
	)
	return &gen.SandboxAck{Ok: true}, nil
}
