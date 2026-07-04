package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
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
	Event     string                        `json:"event"`
	Timestamp string                        `json:"timestamp"`
	Rankings  []benchtypes.LeaderboardEntry `json:"rankings"`
}

func (h *LeaderboardHandler) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := h.store.ListLeaderboard(r.Context())
	if err != nil {
		slog.Error("leaderboard query failed", "err", err)
		middleware.WriteAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch leaderboard")
		return
	}
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

	// Subscribe to Redis Pub/Sub for real-time score updates.
	pubsub := h.redis.Subscribe(r.Context(), "channel:leaderboard")
	defer pubsub.Close()

	msgCh := pubsub.Channel()

	// Send the full leaderboard immediately on connect.
	send := func() {
		entries, err := h.store.ListLeaderboard(r.Context())
		if err != nil {
			slog.Error("leaderboard stream query failed", "err", err)
			return
		}
		payload := leaderboardStreamEnvelope{
			Event:     "leaderboard_update",
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Rankings:  entries,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return
		}
		_, _ = fmt.Fprintf(w, "event: leaderboard_update\ndata: %s\n\n", data)
		flusher.Flush()
	}

	send()

	// 30-second SSE keepalive prevents proxies/CDNs from closing idle connections.
	keepalive := time.NewTicker(30 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case _, ok := <-msgCh:
			if !ok {
				return // subscription closed
			}
			// A score was updated — re-query Postgres for the full ranked leaderboard.
			send()
		case <-keepalive.C:
			_, _ = fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

