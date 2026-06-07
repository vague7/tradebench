package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/bench/api-gateway/config"
	"github.com/bench/api-gateway/handlers"
	"github.com/bench/api-gateway/store"
	"github.com/bench/api-gateway/trigger"

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
	ctx := context.Background()

	postgresStore, err := store.NewPostgresStore(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("FATAL: postgres init failed: %v", err)
	}

	redisClient, err := store.NewRedisClient(cfg.RedisAddr)
	if err != nil {
		log.Fatalf("FATAL: redis init failed: %v", err)
	}
	defer redisClient.Close()

	// Start the ready-event watcher; triggers bot-fleet when a container is healthy.
	watcher := trigger.NewWatcher(redisClient.Client())
	go watcher.Run(ctx)

	mux := http.NewServeMux()
	handlers.NewSubmissionHandler(cfg, postgresStore, redisClient).Register(mux)
	handlers.NewLeaderboardHandler(cfg, postgresStore, redisClient).Register(mux)
	handlers.NewAdminHandler(cfg, postgresStore, redisClient).Register(mux)

	slog.Info("api-gateway starting", "addr", ":8080")
	if err := http.ListenAndServe(":8080",corsMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}
