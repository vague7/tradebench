package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/bench/sandbox-engine/config"
	"github.com/bench/sandbox-engine/queue"
	"github.com/bench/sandbox-engine/runner"
)

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
	healthChecker := runner.NewHealthChecker(time.Duration(cfg.SandboxHealthTimeout) * time.Second)
	watchdog := runner.NewWatchdog(time.Duration(cfg.SandboxContainerTTL) * time.Second)
	registry := runner.NewRegistry()
	db := runner.NewPostgresStatusUpdater(os.Getenv("POSTGRES_DSN"))

	// Watchdog: poll registry every 10s, kill containers past TTL.
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
						slog.Info("watchdog: TTL exceeded, killing container",
							"submissionId", subID,
							"containerID", entry.ContainerID,
						)
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

	consumer := queue.NewConsumer(rdb, builder, spawner, healthChecker, db, registry, cfg.SandboxMaxConcurrent)

	slog.Info("sandbox-engine starting",
		"benchNetName", cfg.BenchNetName,
		"maxConcurrent", cfg.SandboxMaxConcurrent,
		"buildTimeoutSec", cfg.SandboxBuildTimeout,
		"healthTimeoutSec", cfg.SandboxHealthTimeout,
		"containerTTLsec", cfg.SandboxContainerTTL,
	)

	if err := consumer.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
