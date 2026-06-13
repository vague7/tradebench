// Package trigger polls Redis for submission:{id}:ready keys set by the
// consumer after a successful health check, then calls BotFleet.StartBenchmark
// via gRPC to kick off load testing.
package trigger

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	gen "github.com/bench/shared/proto/gen"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	readyKeyPrefix    = "submission:"
	readyKeySuffix    = ":ready"
	pollInterval      = 2 * time.Second
	botFleetAddr      = "bot-fleet:9002"
	benchmarkBotCount = 10000
	benchmarkDuration = 270 // warm-up+ramp+sustained+spike+drain = 30+60+120+30+30
)

// Watcher polls Redis for submission:{id}:ready keys and fires
// BotFleet.StartBenchmark gRPC calls for each newly-ready submission.
type Watcher struct {
	rdb  *redis.Client
	seen map[string]struct{}
}

// NewWatcher constructs a Watcher using the shared Redis client.
func NewWatcher(rdb *redis.Client) *Watcher {
	return &Watcher{rdb: rdb, seen: make(map[string]struct{})}
}

// Run polls Redis every 2 seconds until ctx is cancelled.
func (w *Watcher) Run(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.scan(ctx)
		}
	}
}

func (w *Watcher) scan(ctx context.Context) {
	var cursor uint64
	for {
		keys, nextCursor, err := w.rdb.Scan(ctx, cursor, readyKeyPrefix+"*"+readyKeySuffix, 50).Result()
		if err != nil {
			slog.Error("trigger watcher: redis scan failed", "err", err)
			return
		}
		for _, key := range keys {
			subID := key[len(readyKeyPrefix) : len(key)-len(readyKeySuffix)]
			if _, already := w.seen[subID]; already {
				continue
			}
			targetHost, err := w.rdb.Get(ctx, key).Result()
			if err != nil {
				continue
			}
			w.seen[subID] = struct{}{}
			go w.startBenchmark(ctx, subID, targetHost)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}

func (w *Watcher) startBenchmark(ctx context.Context, submissionID, targetHost string) {
	conn, err := grpc.NewClient(botFleetAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("trigger: grpc dial failed", "submissionId", submissionID, "err", err)
		return
	}
	defer conn.Close()

	client := gen.NewBotFleetClient(conn)
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	shortID := submissionID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	_, err = client.StartBenchmark(callCtx, &gen.BenchmarkConfig{
		SubmissionId: submissionID,
		TargetHost:   fmt.Sprintf("submission-%s:8080", shortID),
		BotCount:     benchmarkBotCount,
		DurationSec:  benchmarkDuration,
	})
	if err != nil {
		slog.Error("trigger: StartBenchmark failed", "submissionId", submissionID, "err", err)
		return
	}
	slog.Info("trigger: benchmark started", "submissionId", submissionID, "targetHost", targetHost)
}
