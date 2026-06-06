package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bench/telemetry-ingester/aggregate"
	"github.com/bench/telemetry-ingester/config"
	"github.com/bench/telemetry-ingester/correctness"
	"github.com/bench/telemetry-ingester/ingest"
	"github.com/bench/telemetry-ingester/scoring"
	"github.com/bench/telemetry-ingester/store"
	benchtypes "github.com/bench/shared/types"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	buffer := ingest.NewBuffer(10000)
	server := ingest.NewServer(buffer)
	windowManager := aggregate.NewWindowManager(cfg.TelemetryWindowSec)
	engine := scoring.NewEngine(float64(cfg.BotTargetTPS), float64(cfg.BotMaxP99Ms))
	validator := correctness.NewValidator()
	postgresStore := store.NewPostgresStore()

	snapshot := benchtypes.MetricSnapshot{
		SubmissionID:     "bootstrap-submission",
		WindowEnd:        windowManager.WindowEnd(time.Now().UTC()),
		P50LatencyMs:     0,
		P90LatencyMs:     0,
		P99LatencyMs:     0,
		TPS:              0,
		SuccessCount:     0,
		FailureCount:     0,
		TimeoutCount:     0,
		CorrectnessScore: validator.Score(nil, nil),
	}
	_ = server
	postgresStore.SaveSnapshot(snapshot)
	postgresStore.SaveScore(engine.Compute(snapshot))

	slog.Info("telemetry-ingester starting",
		"windowSec", cfg.TelemetryWindowSec,
		"postgresDSN", cfg.PostgresDSN,
		"redisAddr", cfg.RedisAddr,
	)

	<-ctx.Done()
	log.Println("telemetry-ingester stopped")
}
