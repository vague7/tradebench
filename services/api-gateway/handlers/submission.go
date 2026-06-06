package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bench/api-gateway/config"
	"github.com/bench/api-gateway/middleware"
	"github.com/bench/api-gateway/store"
	benchtypes "github.com/bench/shared/types"
)

type SubmissionHandler struct {
	cfg   *config.Config
	store *store.PostgresStore
	redis *store.RedisClient
}

func NewSubmissionHandler(cfg *config.Config, store *store.PostgresStore, redis *store.RedisClient) *SubmissionHandler {
	return &SubmissionHandler{cfg: cfg, store: store, redis: redis}
}

func (h *SubmissionHandler) Register(mux *http.ServeMux) {
	mux.Handle("POST /api/submissions", middleware.TeamAuth(middleware.UploadLimit(h.cfg.UploadMaxBytes, http.HandlerFunc(h.handleCreateSubmission))))
	mux.Handle("GET /api/submissions/", middleware.TeamAuth(http.HandlerFunc(h.handleSubmissionAction)))
}

type createSubmissionRequest struct {
	TeamName string `json:"teamName"`
}

type createSubmissionResponse struct {
	SubmissionID string `json:"submissionId"`
}

type submissionStatusResponse struct {
	ID             string                        `json:"id"`
	TeamName       string                        `json:"teamName"`
	Status         benchtypes.SubmissionStatus   `json:"status"`
	ErrorMessage   string                        `json:"errorMessage,omitempty"`
	UploadedAt     string                        `json:"uploadedAt"`
	BenchmarkStart string                        `json:"benchmarkStart,omitempty"`
	BenchmarkEnd   string                        `json:"benchmarkEnd,omitempty"`
}

type submissionResultsResponse struct {
	Snapshot benchtypes.MetricSnapshot `json:"snapshot"`
	Score    benchtypes.Score          `json:"score"`
}

func (h *SubmissionHandler) handleCreateSubmission(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	teamName := r.FormValue("teamName")
	if teamName == "" {
		var req createSubmissionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			teamName = strings.TrimSpace(req.TeamName)
		}
	}
	if teamName == "" {
		teamName = "Unknown Team"
	}

	submissionID := fmt.Sprintf("sub-%d", time.Now().UnixNano())
	h.store.CreateSubmission(submissionID, teamName, time.Now().UTC())
	h.redis.EnqueueJob(map[string]string{
		"submissionId": submissionID,
		"teamName":     teamName,
		"uploadedAt":   time.Now().UTC().Format(time.RFC3339Nano),
	})

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
		h.handleGetSubmissionStatus(w, submissionID)
	case "results":
		if r.Method != http.MethodGet {
			middleware.WriteAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		h.handleGetSubmissionResults(w, submissionID)
	default:
		middleware.WriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
	}
}

func (h *SubmissionHandler) handleGetSubmissionStatus(w http.ResponseWriter, submissionID string) {
	record, ok := h.store.GetSubmission(submissionID)
	if !ok {
		middleware.WriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "submission not found")
		return
	}

	response := submissionStatusResponse{
		ID:           record.ID,
		TeamName:     record.TeamName,
		Status:       record.Status,
		ErrorMessage: record.ErrorMessage,
		UploadedAt:   record.UploadedAt.UTC().Format(time.RFC3339Nano),
	}
	if record.BenchmarkStart != nil {
		response.BenchmarkStart = record.BenchmarkStart.UTC().Format(time.RFC3339Nano)
	}
	if record.BenchmarkEnd != nil {
		response.BenchmarkEnd = record.BenchmarkEnd.UTC().Format(time.RFC3339Nano)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h *SubmissionHandler) handleGetSubmissionResults(w http.ResponseWriter, submissionID string) {
	record, ok := h.store.GetSubmission(submissionID)
	if !ok {
		middleware.WriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "submission not found")
		return
	}
	if record.Snapshot == nil || record.FinalScore == nil {
		middleware.WriteAPIError(w, http.StatusNotFound, "NOT_READY", "results are not available yet")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(submissionResultsResponse{
		Snapshot: *record.Snapshot,
		Score:    *record.FinalScore,
	})
}
