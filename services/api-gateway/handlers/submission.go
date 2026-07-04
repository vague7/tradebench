package handlers

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bench/api-gateway/config"
	"github.com/bench/api-gateway/middleware"
	"github.com/bench/api-gateway/store"
	benchtypes "github.com/bench/shared/types"
)

const submissionsBasePath = "/data/submissions"

type SubmissionHandler struct {
	cfg   *config.Config
	store *store.PostgresStore
	redis *store.RedisClient
}

func NewSubmissionHandler(cfg *config.Config, store *store.PostgresStore, redis *store.RedisClient) *SubmissionHandler {
	return &SubmissionHandler{cfg: cfg, store: store, redis: redis}
}

func (h *SubmissionHandler) Register(mux *http.ServeMux) {
	mux.Handle("POST /api/submissions",
		middleware.TeamAuth(
			middleware.UploadLimit(h.cfg.UploadMaxBytes,
				http.HandlerFunc(h.handleCreateSubmission))))
	mux.Handle("GET /api/submissions/",
		middleware.TeamAuth(http.HandlerFunc(h.handleSubmissionAction)))
}

type createSubmissionResponse struct {
	SubmissionID string `json:"submissionId"`
}

type submissionStatusResponse struct {
	ID             string `json:"id"`
	TeamName       string `json:"teamName"`
	Status         string `json:"status"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
	UploadedAt     string `json:"uploadedAt"`
	BenchmarkStart string `json:"benchmarkStart,omitempty"`
	BenchmarkEnd   string `json:"benchmarkEnd,omitempty"`
}

// submissionResultsResponse uses concrete types so JSON encoding produces the full
// typed structure the frontend expects (Score and MetricSnapshot fields), not a
// Go struct string representation.
type submissionResultsResponse struct {
	Snapshot benchtypes.MetricSnapshot `json:"snapshot"`
	Score    benchtypes.Score          `json:"score"`
}

func (h *SubmissionHandler) handleCreateSubmission(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ── 1. Parse multipart form ───────────────────────────────────────────
	if err := r.ParseMultipartForm(h.cfg.UploadMaxBytes); err != nil {
		middleware.WriteAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid multipart form")
		return
	}

	teamName := strings.TrimSpace(r.FormValue("teamName"))
	if teamName == "" {
		middleware.WriteAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "teamName is required")
		return
	}

	// extract team token from Authorization header
	teamToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	teamToken = strings.TrimSpace(teamToken)

	// Accept both "zipFile" (frontend) and "file" (legacy) field names.
	file, _, err := r.FormFile("zipFile")
	if err != nil {
		file, _, err = r.FormFile("file")
		if err != nil {
			middleware.WriteAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "zipFile field is required")
			return
		}
	}
	defer file.Close()

	// ── 2. Read file bytes + compute SHA-256 ─────────────────────────────
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		slog.Error("failed to read uploaded file", "err", err)
		middleware.WriteAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to read file")
		return
	}

	hash := sha256.Sum256(fileBytes)
	zipHash := hex.EncodeToString(hash[:])

	// ── 3. SHA-256 dedup check ────────────────────────────────────────────
	existing, found, err := h.store.GetSubmissionByHash(ctx, zipHash)
	if err != nil {
		slog.Error("dedup check failed", "err", err)
		middleware.WriteAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "dedup check failed")
		return
	}
	if found {
		slog.Info("duplicate submission, reusing existing",
			"submissionId", existing.ID,
			"zipHash", zipHash,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(createSubmissionResponse{SubmissionID: existing.ID})
		return
	}

	// ── 4. Create DB record to get UUID ───────────────────────────────────
	submissionID, err := h.store.CreateSubmission(ctx, teamName, teamToken, zipHash, "pending")
	if err != nil {
		slog.Error("failed to create submission record", "err", err)
		middleware.WriteAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create submission")
		return
	}

	// ── 5. Save ZIP to disk ───────────────────────────────────────────────
	submissionDir := filepath.Join(submissionsBasePath, submissionID)
	if err := os.MkdirAll(submissionDir, 0755); err != nil {
		slog.Error("failed to create submission dir", "submissionId", submissionID, "err", err)
		middleware.WriteAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to save file")
		return
	}

	zipPath := filepath.Join(submissionDir, "submission.zip")
	if err := os.WriteFile(zipPath, fileBytes, 0644); err != nil {
		slog.Error("failed to write zip file", "submissionId", submissionID, "err", err)
		middleware.WriteAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to save file")
		return
	}

	// ── 5b. Validate ZIP contains a Dockerfile ────────────────────────────
	if err := validateZipHasDockerfile(fileBytes); err != nil {
		slog.Warn("submission rejected: no Dockerfile in ZIP", "submissionId", submissionID)
		_ = os.Remove(zipPath)
		_ = os.Remove(submissionDir)
		middleware.WriteAPIError(w, http.StatusUnprocessableEntity, "INVALID_SUBMISSION", "ZIP must contain a Dockerfile")
		return
	}

	// ── 6. Update zip_path in DB ──────────────────────────────────────────
	if err := h.store.UpdateZipPath(ctx, submissionID, zipPath); err != nil {
		slog.Error("failed to update zip path", "submissionId", submissionID, "err", err)
		// non-fatal — sandbox-engine will still get the path via convention
	}

	// ── 7. Enqueue job to Redis Stream ────────────────────────────────────
	if err := h.redis.EnqueueJob(ctx, map[string]string{
		"submissionId": submissionID,
		"teamName":     teamName,
		"zipPath":      zipPath,
		"uploadedAt":   time.Now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		slog.Error("failed to enqueue job", "submissionId", submissionID, "err", err)
		middleware.WriteAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to enqueue job")
		return
	}

	slog.Info("submission uploaded",
		"submissionId", submissionID,
		"teamName", teamName,
		"zipHash", zipHash,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(createSubmissionResponse{SubmissionID: submissionID})
}

func (h *SubmissionHandler) handleSubmissionAction(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/submissions/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 {
		middleware.WriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return
	}

	submissionID, action := parts[0], parts[1]
	switch action {
	case "status":
		if r.Method != http.MethodGet {
			middleware.WriteAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		h.handleGetStatus(w, r, submissionID)
	case "results":
		if r.Method != http.MethodGet {
			middleware.WriteAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		h.handleGetResults(w, r, submissionID)
	case "history":
		if r.Method != http.MethodGet {
			middleware.WriteAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		h.handleGetHistory(w, r, submissionID)
	default:
		middleware.WriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
	}
}

func (h *SubmissionHandler) handleGetStatus(w http.ResponseWriter, r *http.Request, submissionID string) {
	ctx := r.Context()
	record, found, err := h.store.GetSubmission(ctx, submissionID)
	if err != nil {
		slog.Error("get submission failed", "submissionId", submissionID, "err", err)
		middleware.WriteAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch submission")
		return
	}
	if !found {
		middleware.WriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "submission not found")
		return
	}

	resp := submissionStatusResponse{
		ID:         record.ID,
		TeamName:   record.TeamName,
		Status:     string(record.Status),
		UploadedAt: record.UploadedAt.UTC().Format(time.RFC3339Nano),
	}

	if record.ErrorMessage != nil {
		resp.ErrorMessage = *record.ErrorMessage
	}
	if record.BenchmarkStart != nil {
		resp.BenchmarkStart = record.BenchmarkStart.UTC().Format(time.RFC3339Nano)
	}
	if record.BenchmarkEnd != nil {
		resp.BenchmarkEnd = record.BenchmarkEnd.UTC().Format(time.RFC3339Nano)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *SubmissionHandler) handleGetResults(w http.ResponseWriter, r *http.Request, submissionID string) {
	ctx := r.Context()

	snapshot, snapFound, err := h.store.GetLatestSnapshot(ctx, submissionID)
	if err != nil {
		slog.Error("get snapshot failed", "submissionId", submissionID, "err", err)
		middleware.WriteAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch results")
		return
	}

	score, scoreFound, err := h.store.GetLatestScore(ctx, submissionID)
	if err != nil {
		slog.Error("get score failed", "submissionId", submissionID, "err", err)
		middleware.WriteAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch results")
		return
	}

	if !snapFound && !scoreFound {
		middleware.WriteAPIError(w, http.StatusNotFound, "NOT_READY", "results are not available yet")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(submissionResultsResponse{
		Snapshot: snapshot,
		Score:    score,
	})
}

func (h *SubmissionHandler) handleGetHistory(w http.ResponseWriter, r *http.Request, submissionID string) {
	ctx := r.Context()

	history, err := h.store.GetSnapshotHistory(ctx, submissionID)
	if err != nil {
		slog.Error("get snapshot history failed", "submissionId", submissionID, "err", err)
		middleware.WriteAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch snapshot history")
		return
	}

	if len(history) == 0 {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(history)
}

// validateZipHasDockerfile returns an error if the ZIP bytes contain no "Dockerfile" entry.
func validateZipHasDockerfile(data []byte) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("invalid zip: %w", err)
	}
	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name == "Dockerfile" {
			return nil
		}
	}
	return fmt.Errorf("no Dockerfile found in ZIP")
}
