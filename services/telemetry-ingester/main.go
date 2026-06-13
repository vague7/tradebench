package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/bench/shared/proto/gen"
	"github.com/bench/telemetry-ingester/aggregate"
	"github.com/bench/telemetry-ingester/config"
	"github.com/bench/telemetry-ingester/ingest"
	"github.com/bench/telemetry-ingester/scoring"
	"github.com/bench/telemetry-ingester/store"
	"github.com/bench/telemetry-ingester/correctness"
)

func main() {
	// Structured JSON logger — PRD Section 9.1
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Construction order: config → postgres → redis → scoring engine → buffer → ingest server → window manager → gRPC → signals.

	cfg := config.Load()

	// Connect to PostgreSQL.
	pgStore, err := store.NewPostgresStore(cfg.PostgresDSN)
	if err != nil {
		slog.Error("failed to connect to postgres", "err", err)
		os.Exit(1)
	}

	// Connect to Redis.
	redisStore := store.NewRedisStore(cfg.RedisAddr)

	// Create scoring engine.
	scorer := scoring.NewEngine(cfg, pgStore, redisStore)

	// Create correctness validator.
	validator := correctness.NewValidator("./correctness/reference/bin/ref-engine")

	// Create ring buffer and gRPC server.
	buf := ingest.NewRingBuffer()
	srv := ingest.NewServer(buf)

	// Create window manager.
	wm := aggregate.NewWindowManager(cfg.WindowSec, buf, pgStore, scorer, validator)

	// Create a cancellable context for the window manager.
	ctx, cancel := context.WithCancel(context.Background())

	// Start the window manager in a goroutine.
	go wm.Run(ctx)

	// Set up gRPC server.
	grpcServer := grpc.NewServer()
	gen.RegisterTelemetryIngesterServer(grpcServer, srv)

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		slog.Error("failed to listen on gRPC port", "port", cfg.GRPCPort, "err", err)
		os.Exit(1)
	}

	slog.Info("telemetry-ingester gRPC listening", "port", cfg.GRPCPort)

	// Handle OS signals for graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("received shutdown signal")
		cancel()
		grpcServer.GracefulStop()
		pgStore.Close()
		redisStore.Close()
		slog.Info("telemetry-ingester shutdown complete")
	}()

	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("gRPC server failed", "err", err)
		os.Exit(1)
	}
}
