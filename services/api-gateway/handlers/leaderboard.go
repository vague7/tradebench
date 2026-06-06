package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bench/api-gateway/config"
	"github.com/bench/api-gateway/middleware"
	"github.com/bench/api-gateway/store"
	benchtypes "github.com/bench/shared/types"
)

type LeaderboardHandler struct {
	cfg   *config.Config
	store *store.PostgresStore
	redis *store.RedisClient
}

func NewLeaderboardHandler(cfg *config.Config, store *store.PostgresStore, redis *store.RedisClient) *LeaderboardHandler {
	return &LeaderboardHandler{cfg: cfg, store: store, redis: redis}
}

func (h *LeaderboardHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/leaderboard", h.handleLeaderboard)
	mux.HandleFunc("GET /api/leaderboard/stream", h.handleLeaderboardStream)
}

type leaderboardStreamEnvelope struct {
	Event     string                     `json:"event"`
	Timestamp string                     `json:"timestamp"`
	Rankings  []benchtypes.LeaderboardEntry `json:"rankings"`
}

func (h *LeaderboardHandler) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	entries := h.store.ListLeaderboard()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entries)
}

func (h *LeaderboardHandler) handleLeaderboardStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		middleware.WriteAPIError(w, http.StatusInternalServerError, "STREAM_UNAVAILABLE", "streaming is unavailable")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(time.Duration(h.cfg.LeaderboardSSEIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	send := func() {
		payload := leaderboardStreamEnvelope{
			Event:     "leaderboard_update",
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Rankings:  h.store.ListLeaderboard(),
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return
		}
		_, _ = fmt.Fprintf(w, "event: leaderboard_update\ndata: %s\n\n", data)
		flusher.Flush()
	}

	send()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			send()
		}
	}
}
