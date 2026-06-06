package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/bench/api-gateway/config"
	"github.com/bench/api-gateway/middleware"
	"github.com/bench/api-gateway/store"
	benchtypes "github.com/bench/shared/types"
)

type AdminHandler struct {
	cfg   *config.Config
	store *store.PostgresStore
	redis *store.RedisClient
}

func NewAdminHandler(cfg *config.Config, store *store.PostgresStore, redis *store.RedisClient) *AdminHandler {
	return &AdminHandler{cfg: cfg, store: store, redis: redis}
}

func (h *AdminHandler) Register(mux *http.ServeMux) {
	mux.Handle("POST /api/admin/benchmark/", middleware.AdminAuth(h.cfg, http.HandlerFunc(h.handleBenchmarkAction)))
}

type adminAck struct {
	OK bool `json:"ok"`
}

func (h *AdminHandler) handleBenchmarkAction(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/benchmark/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 {
		middleware.WriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return
	}

	submissionID, action := parts[0], parts[1]
	now := time.Now().UTC()
	switch action {
	case "start":
		if !h.store.SetBenchmarkStart(submissionID, now) {
			middleware.WriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "submission not found")
			return
		}
		_ = h.store.UpdateStatus(submissionID, benchtypes.StatusBenchmarking, "")
	case "stop":
		if !h.store.SetBenchmarkEnd(submissionID, now, benchtypes.StatusFailed) {
			middleware.WriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "submission not found")
			return
		}
		_ = h.store.UpdateStatus(submissionID, benchtypes.StatusFailed, "benchmark stopped by admin")
	default:
		middleware.WriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(adminAck{OK: true})
}
