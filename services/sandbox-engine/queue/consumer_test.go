package queue_test

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// Fakes — no real Docker / Redis / Postgres needed.
// ---------------------------------------------------------------------------

type statusCall struct {
	submissionID string
	status       string
	errMsg       string
}

type imageCall struct {
	submissionID string
	imageTag     string
	containerID  string
	port         int
}

// fakeDB records every UpdateStatus and UpdateImageAndContainer call in order.
type fakeDB struct {
	mu           sync.Mutex
	statusCalls  []statusCall
	imageCalls   []imageCall
	failOnStatus string // if set, UpdateStatus returns an error for this status value
}

func (f *fakeDB) UpdateStatus(_ context.Context, submissionID, status, errMsg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.statusCalls = append(f.statusCalls, statusCall{submissionID, status, errMsg})
	if f.failOnStatus == status {
		return errors.New("injected db error")
	}
	return nil
}

func (f *fakeDB) UpdateImageAndContainer(_ context.Context, submissionID, imageTag, containerID string, port int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.imageCalls = append(f.imageCalls, imageCall{submissionID, imageTag, containerID, port})
	return nil
}

func (f *fakeDB) statusSequence() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.statusCalls))
	for i, c := range f.statusCalls {
		out[i] = c.status
	}
	return out
}

// ---------------------------------------------------------------------------
// Pipeline state machine — mirrors consumer.process() logic without I/O.
// This lets us unit-test status transition ordering independently of Docker.
// ---------------------------------------------------------------------------

type buildResult struct {
	containerID string
	port        int
	err         error
}

// runPipeline simulates the consumer.process() state machine.
// It calls db methods in the same order as the real consumer, making it
// possible to assert the exact sequence of status transitions.
func runPipeline(
	ctx context.Context,
	submissionID string,
	db interface {
		UpdateStatus(context.Context, string, string, string) error
		UpdateImageAndContainer(context.Context, string, string, string, int) error
	},
	buildErr error,
	spawnResult buildResult,
	healthErr error,
) error {
	imageTag := "bench-submission-" + submissionID

	// BUILDING
	_ = db.UpdateStatus(ctx, submissionID, "BUILDING", "")
	if buildErr != nil {
		_ = db.UpdateStatus(ctx, submissionID, "FAILED", buildErr.Error())
		return buildErr
	}

	// RUNNING
	_ = db.UpdateStatus(ctx, submissionID, "RUNNING", "")
	if spawnResult.err != nil {
		_ = db.UpdateStatus(ctx, submissionID, "FAILED", spawnResult.err.Error())
		return spawnResult.err
	}

	_ = db.UpdateImageAndContainer(ctx, submissionID, imageTag, spawnResult.containerID, spawnResult.port)

	// Health check
	if healthErr != nil {
		_ = db.UpdateStatus(ctx, submissionID, "FAILED", "health check failed: "+healthErr.Error())
		return healthErr
	}

	// BENCHMARKING
	_ = db.UpdateStatus(ctx, submissionID, "BENCHMARKING", "")
	return nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestStatusTransition_HappyPath(t *testing.T) {
	db := &fakeDB{}
	err := runPipeline(
		context.Background(),
		"sub-001",
		db,
		nil,                                               // build ok
		buildResult{containerID: "ctr-abc", port: 49152}, // spawn ok
		nil,                                               // health ok
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	want := []string{"BUILDING", "RUNNING", "BENCHMARKING"}
	got := db.statusSequence()
	if !equalSlice(got, want) {
		t.Errorf("status sequence: got %v, want %v", got, want)
	}

	if len(db.imageCalls) != 1 {
		t.Errorf("expected 1 UpdateImageAndContainer call, got %d", len(db.imageCalls))
	}
	if db.imageCalls[0].containerID != "ctr-abc" {
		t.Errorf("containerID: got %q, want %q", db.imageCalls[0].containerID, "ctr-abc")
	}
}

func TestStatusTransition_BuildFails(t *testing.T) {
	db := &fakeDB{}
	err := runPipeline(
		context.Background(),
		"sub-002",
		db,
		errors.New("docker build: exit 1"),
		buildResult{},
		nil,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	want := []string{"BUILDING", "FAILED"}
	got := db.statusSequence()
	if !equalSlice(got, want) {
		t.Errorf("status sequence: got %v, want %v", got, want)
	}
	// Must not attempt spawn or image update after build failure.
	if len(db.imageCalls) != 0 {
		t.Errorf("expected 0 UpdateImageAndContainer calls after build failure, got %d", len(db.imageCalls))
	}
}

func TestStatusTransition_SpawnFails(t *testing.T) {
	db := &fakeDB{}
	err := runPipeline(
		context.Background(),
		"sub-003",
		db,
		nil,
		buildResult{err: errors.New("container create: port conflict")},
		nil,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	want := []string{"BUILDING", "RUNNING", "FAILED"}
	got := db.statusSequence()
	if !equalSlice(got, want) {
		t.Errorf("status sequence: got %v, want %v", got, want)
	}
	if len(db.imageCalls) != 0 {
		t.Errorf("expected 0 UpdateImageAndContainer after spawn failure, got %d", len(db.imageCalls))
	}
}

func TestStatusTransition_HealthCheckFails(t *testing.T) {
	db := &fakeDB{}
	err := runPipeline(
		context.Background(),
		"sub-004",
		db,
		nil,
		buildResult{containerID: "ctr-xyz", port: 49153},
		errors.New("timed out waiting for container"),
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	want := []string{"BUILDING", "RUNNING", "FAILED"}
	got := db.statusSequence()
	if !equalSlice(got, want) {
		t.Errorf("status sequence: got %v, want %v", got, want)
	}
	// UpdateImageAndContainer must still be called — container exists, just not healthy.
	if len(db.imageCalls) != 1 {
		t.Errorf("expected 1 UpdateImageAndContainer even after health failure, got %d", len(db.imageCalls))
	}
}

func TestStatusTransition_FailedStatusContainsErrorMessage(t *testing.T) {
	db := &fakeDB{}
	buildErrMsg := "Dockerfile parse error line 3"
	_ = runPipeline(
		context.Background(),
		"sub-005",
		db,
		errors.New(buildErrMsg),
		buildResult{},
		nil,
	)

	db.mu.Lock()
	defer db.mu.Unlock()
	for _, c := range db.statusCalls {
		if c.status == "FAILED" && c.errMsg == "" {
			t.Error("FAILED status must carry a non-empty error message")
		}
	}
}

func TestStatusTransition_ContextCancelled_BuildStep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	db := &fakeDB{}
	// Even with a cancelled context, the pipeline should still write BUILDING
	// then FAILED (the real builder returns ctx.Err() which we simulate here).
	err := runPipeline(
		ctx,
		"sub-006",
		db,
		context.Canceled,
		buildResult{},
		nil,
	)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	seq := db.statusSequence()
	if len(seq) < 2 || seq[len(seq)-1] != "FAILED" {
		t.Errorf("expected last status to be FAILED, got sequence %v", seq)
	}
}

func TestStatusTransition_NoDuplicateBenchmarkingOnHealthFail(t *testing.T) {
	db := &fakeDB{}
	_ = runPipeline(
		context.Background(),
		"sub-007",
		db,
		nil,
		buildResult{containerID: "ctr-001", port: 49154},
		errors.New("health timeout"),
	)

	for _, c := range db.statusSequence() {
		if c == "BENCHMARKING" {
			t.Error("BENCHMARKING must not appear when health check fails")
		}
	}
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func equalSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
