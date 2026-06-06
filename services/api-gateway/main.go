package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/bench/api-gateway/config"
	"github.com/bench/api-gateway/handlers"
	"github.com/bench/api-gateway/store"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()
	postgresStore := store.NewPostgresStore()
	redisClient := store.NewRedisClient()

	mux := http.NewServeMux()
	handlers.NewSubmissionHandler(cfg, postgresStore, redisClient).Register(mux)
	handlers.NewLeaderboardHandler(cfg, postgresStore, redisClient).Register(mux)
	handlers.NewAdminHandler(cfg, postgresStore, redisClient).Register(mux)

	slog.Info("api-gateway starting", "addr", ":8080")
	if err := http.ListenAndServe(":8080", corsMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}
