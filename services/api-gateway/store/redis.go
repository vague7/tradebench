package store

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const jobStreamKey = "stream:jobs"

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(addr string) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis: ping: %w", err)
	}
	return &RedisClient{client: rdb}, nil
}

func (r *RedisClient) EnqueueJob(ctx context.Context, fields map[string]string) error {
	args := &redis.XAddArgs{
		Stream: jobStreamKey,
		Values: fields,
	}
	if err := r.client.XAdd(ctx, args).Err(); err != nil {
		return fmt.Errorf("redis: enqueue job: %w", err)
	}
	return nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Client exposes the underlying redis client for use outside the store package.
func (r *RedisClient) Client() *redis.Client {
	return r.client
}
