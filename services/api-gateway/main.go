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
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
