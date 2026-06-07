package queue

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/bench/sandbox-engine/runner"
)

const (
	jobStreamKey  = "stream:jobs"
	consumerGroup = "sandbox-engine"
	consumerName  = "sandbox-engine-1"
	blockDuration = 5 * time.Second
)

type StatusUpdater interface {
	UpdateStatus(ctx context.Context, submissionID, status, errMsg string) error
}

type Consumer struct {
	rdb      *redis.Client
	builder  *runner.Builder
	spawner  *runner.Spawner
	health   *runner.HealthChecker
	db       StatusUpdater
	registry *runner.Registry
	sem      chan struct{} // concurrency limiter
}

func NewConsumer(
	rdb *redis.Client,
	builder *runner.Builder,
	spawner *runner.Spawner,
	health *runner.HealthChecker,
	db StatusUpdater,
	registry *runner.Registry,
	maxConcurrent int,
) *Consumer {
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}
	return &Consumer{
		rdb:      rdb,
		builder:  builder,
		spawner:  spawner,
		health:   health,
		db:       db,
		registry: registry,
		sem:      make(chan struct{}, maxConcurrent),
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	err := c.rdb.XGroupCreateMkStream(ctx, jobStreamKey, consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("consumer: create group: %w", err)
	}

	slog.Info("consumer: listening", "stream", jobStreamKey, "group", consumerGroup, "maxConcurrent", cap(c.sem))

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
				// Acquire slot — blocks if at capacity.
				select {
				case c.sem <- struct{}{}:
				case <-ctx.Done():
					return ctx.Err()
				}

				go func(m redis.XMessage) {
					defer func() { <-c.sem }() // release slot when done

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

	_ = c.db.UpdateStatus(ctx, submissionID, "BUILDING", "")
	if err := c.builder.Build(zipPath, imageTag); err != nil {
		_ = c.db.UpdateStatus(ctx, submissionID, "FAILED", err.Error())
		return fmt.Errorf("consumer: build: %w", err)
	}

	_ = c.db.UpdateStatus(ctx, submissionID, "RUNNING", "")
	containerID, port, err := c.spawner.Spawn(imageTag, submissionID)
	if err != nil {
		_ = c.db.UpdateStatus(ctx, submissionID, "FAILED", err.Error())
		return fmt.Errorf("consumer: spawn: %w", err)
	}

	// Register before health check so watchdog can clean up on timeout.
	c.registry.Add(submissionID, containerID, port, time.Now())

	if err := c.health.WaitReady(ctx, containerID); err != nil {
		_ = c.db.UpdateStatus(ctx, submissionID, "FAILED", "health check failed: "+err.Error())
		c.registry.Remove(submissionID)
		return fmt.Errorf("consumer: health: %w", err)
	}

	_ = c.db.UpdateStatus(ctx, submissionID, "BENCHMARKING", "")

	// Publish ready event for api-gateway to trigger bot-fleet.
	readyKey := "submission:" + submissionID + ":ready"
	_ = c.rdb.Set(ctx, readyKey, fmt.Sprintf("%s:%d", containerID, port), 10*time.Minute).Err()

	slog.Info("consumer: container ready",
		"submissionId", submissionID,
		"containerID", containerID,
		"port", port,
	)
	return nil
}
