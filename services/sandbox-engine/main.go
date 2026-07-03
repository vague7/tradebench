package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	"github.com/bench/sandbox-engine/config"
	sbgrpc "github.com/bench/sandbox-engine/grpc"
	"github.com/bench/sandbox-engine/queue"
	"github.com/bench/sandbox-engine/runner"
	gen "github.com/bench/shared/proto/gen"
)

const sandboxGRPCPort = ":9001"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("FATAL: redis ping failed: %v", err)
	}
	defer rdb.Close()

	builder := runner.NewBuilder()
	spawner := runner.NewSpawner(cfg.BenchNetName)
	// Pass BenchNetName so the health checker reaches submission containers via
	// their bench-net IP, not via 127.0.0.1 (which is unreachable inside Docker).
	healthChecker := runner.NewHealthChecker(
		time.Duration(cfg.SandboxHealthTimeout)*time.Second,
		cfg.BenchNetName,
	)
	watchdog := runner.NewWatchdog(time.Duration(cfg.SandboxContainerTTL) * time.Second)
	registry := runner.NewRegistry()
	db := runner.NewPostgresStatusUpdater(os.Getenv("POSTGRES_DSN"))

	// ── Watchdog ─────────────────────────────────────────────────────────────
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				for subID, entry := range registry.Snapshot() {
					if watchdog.ShouldKill(entry.StartedAt, now) {
						slog.Info("watchdog: TTL exceeded, killing", "submissionId", subID)
						if err := healthChecker.KillAndRemove(entry.ContainerID); err != nil {
							slog.Error("watchdog: kill failed", "containerID", entry.ContainerID, "err", err)
						}
						registry.Remove(subID)
						_ = db.UpdateStatus(context.Background(), subID, "FAILED", "container TTL exceeded")
					}
				}
			}
		}
	}()

	// ── SandboxEngine gRPC server on :9001 ───────────────────────────────────
	lis, err := net.Listen("tcp", sandboxGRPCPort)
	if err != nil {
		log.Fatalf("FATAL: sandbox gRPC listen on %s failed: %v", sandboxGRPCPort, err)
	}
	grpcServer := grpc.NewServer()
	gen.RegisterSandboxEngineServer(grpcServer, sbgrpc.NewSandboxServer(registry, db, healthChecker))
	go func() {
		slog.Info("sandbox-engine gRPC server listening", "port", sandboxGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("sandbox gRPC server error", "err", err)
		}
	}()
	go func() { <-ctx.Done(); grpcServer.GracefulStop() }()

	// ── Redis stream consumer ─────────────────────────────────────────────────
	consumer := queue.NewConsumer(rdb, builder, spawner, healthChecker, db, registry,
		cfg.SandboxMaxConcurrent, cfg.SandboxBuildTimeout)

	slog.Info("sandbox-engine starting",
		"benchNetName", cfg.BenchNetName,
		"maxConcurrent", cfg.SandboxMaxConcurrent,
		"buildTimeoutSec", cfg.SandboxBuildTimeout,
		"healthTimeoutSec", cfg.SandboxHealthTimeout,
		"containerTTLsec", cfg.SandboxContainerTTL,
		"grpcPort", sandboxGRPCPort,
	)

	if err := consumer.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
