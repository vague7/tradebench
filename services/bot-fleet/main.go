package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bench/bot-fleet/bot"
	"github.com/bench/bot-fleet/config"
	"github.com/bench/bot-fleet/emit"
	"github.com/bench/bot-fleet/fleet"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	streamer := emit.NewStreamer()
	generator := bot.NewGenerator()
	runner := bot.NewRunner(generator, streamer)
	coordinator := fleet.NewCoordinator(fleet.DefaultProfile(), runner)

	slog.Info("bot-fleet starting",
		"defaultCount", cfg.BotDefaultCount,
		"timeoutMs", cfg.BotTimeoutMs,
		"targetTPS", cfg.BotTargetTPS,
		"maxP99Ms", cfg.BotMaxP99Ms,
		"telemetryAddr", cfg.TelemetryAddr,
	)

	if err := coordinator.Run(ctx, "bootstrap-submission"); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
