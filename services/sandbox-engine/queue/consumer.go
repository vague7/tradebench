package queue

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bench/sandbox-engine/runner"
	"github.com/redis/go-redis/v9"
)

const (
	jobStreamKey  = "stream:jobs"
	consumerGroup = "sandbox-engine"
	consumerName  = "sandbox-engine-1"
	blockDuration = 5 * time.Second

	// Redis key TTLs (PRD Section 4.4)
	statusKeyTTL = time.Hour
	readyKeyTTL  = 10 * time.Minute
)

// StatusUpdater is the interface the consumer uses to update submission state.
// PostgresStatusUpdater satisfies this interface.
type StatusUpdater interface {
	UpdateStatus(ctx context.Context, submissionID, status, errMsg string) error
	UpdateImageAndContainer(ctx context.Context, submissionID, imageTag, containerID string, containerPort int) error
}

// Consumer reads jobs from the Redis stream and drives each submission through
// the Build → Spawn → HealthCheck → Ready pipeline.
type Consumer struct {
	rdb          *redis.Client
	builder      *runner.Builder
	spawner      *runner.Spawner
	health       *runner.HealthChecker
	db           StatusUpdater
	registry     *runner.Registry
	sem          chan struct{} // concurrency limiter
	buildTimeout time.Duration
}

func NewConsumer(
	rdb *redis.Client,
	builder *runner.Builder,
	spawner *runner.Spawner,
	health *runner.HealthChecker,
	db StatusUpdater,
	registry *runner.Registry,
	maxConcurrent int,
	buildTimeoutSec int,
) *Consumer {
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}
	if buildTimeoutSec <= 0 {
		buildTimeoutSec = 120
	}
	return &Consumer{
		rdb:          rdb,
		builder:      builder,
		spawner:      spawner,
		health:       health,
		db:           db,
		registry:     registry,
		sem:          make(chan struct{}, maxConcurrent),
		buildTimeout: time.Duration(buildTimeoutSec) * time.Second,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	err := c.rdb.XGroupCreateMkStream(ctx, jobStreamKey, consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("consumer: create group: %w", err)
	}

	slog.Info("consumer: listening",
		"stream", jobStreamKey,
		"group", consumerGroup,
		"maxConcurrent", cap(c.sem),
		"buildTimeoutSec", c.buildTimeout.Seconds(),
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: consumerName,
			Streams:  []string{jobStreamKey, ">"},
			Count:    1,
			Block:    blockDuration,
		}).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			slog.Error("consumer: xreadgroup error", "err", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				// Acquire concurrency slot — blocks if at capacity.
				select {
				case c.sem <- struct{}{}:
				case <-ctx.Done():
					return ctx.Err()
				}

				go func(m redis.XMessage) {
					defer func() { <-c.sem }()

					if err := c.process(ctx, m); err != nil {
						slog.Error("consumer: process failed", "msgId", m.ID, "err", err)
						return // leave in PEL for redelivery
					}
					_ = c.rdb.XAck(ctx, jobStreamKey, consumerGroup, m.ID).Err()
				}(msg)
			}
		}
	}
}

func (c *Consumer) process(ctx context.Context, msg redis.XMessage) error {
	vals := msg.Values
	submissionID, _ := vals["submissionId"].(string)
	zipPath, _ := vals["zipPath"].(string)

	if submissionID == "" || zipPath == "" {
		return fmt.Errorf("consumer: missing submissionId or zipPath in message %s", msg.ID)
	}

	imageTag := "bench-submission-" + submissionID

	// ── BUILDING ─────────────────────────────────────────────────────────────
	c.setStatus(ctx, submissionID, "BUILDING", "")

	// Apply build timeout from config (PRD FR-2: context.WithTimeout 120s).
	buildCtx, buildCancel := context.WithTimeout(ctx, c.buildTimeout)
	defer buildCancel()

	if err := c.builder.Build(buildCtx, zipPath, imageTag); err != nil {
		c.setStatus(ctx, submissionID, "FAILED", err.Error())
		return fmt.Errorf("consumer: build: %w", err)
	}

	// ── RUNNING — spawn container ─────────────────────────────────────────────
	c.setStatus(ctx, submissionID, "RUNNING", "")

	containerID, port, err := c.spawner.Spawn(imageTag, submissionID)
	if err != nil {
		c.setStatus(ctx, submissionID, "FAILED", err.Error())
		return fmt.Errorf("consumer: spawn: %w", err)
	}

	// Write image_tag, container_id, container_port to Postgres (PRD schema).
	if err := c.db.UpdateImageAndContainer(ctx, submissionID, imageTag, containerID, port); err != nil {
		// Non-fatal — pipeline can continue; log for observability.
		slog.Error("consumer: UpdateImageAndContainer failed",
			"submissionId", submissionID,
			"err", err,
		)
	}

	// Register before health check so the watchdog can clean up on timeout.
	c.registry.Add(submissionID, containerID, port, time.Now())

	// ── HEALTH CHECK ─────────────────────────────────────────────────────────
	if err := c.health.WaitReady(ctx, containerID); err != nil {
		c.setStatus(ctx, submissionID, "FAILED", "health check failed: "+err.Error())
		c.registry.Remove(submissionID)
		return fmt.Errorf("consumer: health: %w", err)
	}

	// ── BENCHMARKING — publish ready key for trigger watcher ──────────────────
	c.setStatus(ctx, submissionID, "BENCHMARKING", "")

	readyKey := "submission:" + submissionID + ":ready"
	targetHost := fmt.Sprintf("submission-%s:8080", shortID(submissionID))
	_ = c.rdb.Set(ctx, readyKey, targetHost, readyKeyTTL).Err()

	slog.Info("consumer: container ready",
		"submissionId", submissionID,
		"containerID", containerID,
		"port", port,
		"imageTag", imageTag,
	)
	return nil
}

// setStatus updates both Postgres and the Redis submission:{id}:status key.
// Errors are logged but never returned — a status update failure must not abort
// the pipeline; the container state is authoritative.
func (c *Consumer) setStatus(ctx context.Context, submissionID, status, errMsg string) {
	if err := c.db.UpdateStatus(ctx, submissionID, status, errMsg); err != nil {
		slog.Error("consumer: UpdateStatus failed",
			"submissionId", submissionID,
			"status", status,
			"err", err,
		)
	}

	// Write Redis cache key: submission:{id}:status (PRD Section 4.4, TTL 1h).
	redisKey := "submission:" + submissionID + ":status"
	if err := c.rdb.Set(ctx, redisKey, status, statusKeyTTL).Err(); err != nil {
		slog.Error("consumer: redis SetStatus failed",
			"submissionId", submissionID,
			"status", status,
			"err", err,
		)
	}
}

// shortID returns the first 8 chars of a UUID (matches container naming convention).
func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
