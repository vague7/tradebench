package main

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bench/bot-fleet/bot"
	"github.com/bench/bot-fleet/config"
)

// day1BotCap limits goroutine count for Day 1 safety.
const day1BotCap = 100

func main() {
	// Structured JSON logger — PRD Section 9.1
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Load()

	// Day 1 local test overrides — not permanent config fields.
	// Exception to config-only rule: temporary smoke-test harness.
	targetURL := os.Getenv("BOT_TARGET_URL")
	if targetURL == "" {
		targetURL = "http://localhost:8080"
	}

	submissionID := os.Getenv("BOT_TEST_SUBMISSION_ID")
	if submissionID == "" {
		submissionID = "test-submission-day1"
	}

	slog.Info("bot-fleet starting",
		"botCount", cfg.BotDefaultCount,
		"targetURL", targetURL,
		"timeoutMs", cfg.BotTimeoutMs,
	)

	botCount := min(cfg.BotDefaultCount, day1BotCap)

	// Day 1: 30-second timeout for the entire run
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var totalEvents, successCount, failureCount, timeoutCount int64
	start := time.Now()

	for i := 0; i < botCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Per-bot state: each bot maintains its own open order list
			var openOrderIDs []string
			var mu sync.Mutex

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				event, err := bot.RunBot(ctx, submissionID, targetURL, cfg.BotTimeoutMs, &openOrderIDs, &mu)
				if err != nil {
					// If context is done, exit cleanly
					if ctx.Err() != nil {
						return
					}
					slog.Debug("bot run error",
						"submissionId", submissionID,
						"err", err,
					)
					continue
				}

				atomic.AddInt64(&totalEvents, 1)
				switch {
				case event.HTTPStatus == 0:
					atomic.AddInt64(&timeoutCount, 1)
				case event.HTTPStatus >= 200 && event.HTTPStatus < 300:
					atomic.AddInt64(&successCount, 1)
				default:
					atomic.AddInt64(&failureCount, 1)
				}

				slog.Debug("bot event",
					"submissionId", event.SubmissionID,
					"botId", event.BotID,
					"orderId", event.OrderID,
					"orderType", event.OrderType,
					"httpStatus", event.HTTPStatus,
					"durationMs", event.AckedAt.Sub(event.SentAt).Milliseconds(),
				)
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	slog.Info("bot-fleet day1 run complete",
		"totalEvents", atomic.LoadInt64(&totalEvents),
		"successCount", atomic.LoadInt64(&successCount),
		"failureCount", atomic.LoadInt64(&failureCount),
		"timeoutCount", atomic.LoadInt64(&timeoutCount),
		"durationMs", elapsed.Milliseconds(),
	)

	os.Exit(0)
}
