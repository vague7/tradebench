package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bench/sandbox-engine/config"
	"github.com/bench/sandbox-engine/queue"
	"github.com/bench/sandbox-engine/runner"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	builder := runner.NewBuilder()
	spawner := runner.NewSpawner(cfg.BenchNetName)
	healthChecker := runner.NewHealthChecker(time.Duration(cfg.SandboxHealthTimeout) * time.Second)
	watchdog := runner.NewWatchdog(time.Duration(cfg.SandboxContainerTTL) * time.Second)
	consumer := queue.NewConsumer()

	slog.Info("sandbox-engine starting",
		"benchNetName", cfg.BenchNetName,
		"maxConcurrent", cfg.SandboxMaxConcurrent,
		"buildTimeoutSec", cfg.SandboxBuildTimeout,
		"healthTimeoutSec", cfg.SandboxHealthTimeout,
		"containerTTLsec", cfg.SandboxContainerTTL,
	)

	_ = builder
	_ = spawner
	_ = healthChecker
	_ = watchdog

	if err := consumer.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
