package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bench/telemetry-ingester/config"
)

func main() {
	// Structured JSON logger — PRD Section 9.1
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Load()

	slog.Info("telemetry-ingester starting",
		"grpcAddr", cfg.GRPCListenAddr,
		"windowSec", cfg.WindowSec,
		"postgresDSN", "[redacted]", // never log actual DSN
	)

	// Day 2: wire gRPC server here (ingest/server.go)
	// Day 2: connect to PostgreSQL (store/postgres.go)
	// Day 2: start aggregation pipeline (aggregate/window.go)
	slog.Info("telemetry-ingester gRPC server: wiring pending Day 2")

	// Block indefinitely — service must not exit on Day 1.
	// Simulates a long-running daemon so the Docker container stays alive.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
}
