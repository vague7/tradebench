package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	PostgresDSN             string
	RedisAddr               string
	UploadMaxBytes          int64
	LeaderboardSSEIntervalMs int
	AdminToken              string
}

func Load() *Config {
	return &Config{
		PostgresDSN:              required("POSTGRES_DSN"),
		RedisAddr:                required("REDIS_ADDR"),
		UploadMaxBytes:           parseInt64Required("UPLOAD_MAX_BYTES"),
		LeaderboardSSEIntervalMs: parseIntRequired("LEADERBOARD_SSE_INTERVAL_MS"),
		AdminToken:               required("ADMIN_TOKEN"),
	}
}

func required(name string) string {
	value := os.Getenv(name)
	if value == "" {
		panic(fmt.Sprintf("FATAL: required env var %s is not set", name))
	}
	return value
}

func parseIntRequired(name string) int {
	value := required(name)
	parsed, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("FATAL: required env var %s must be an integer: %v", name, err))
	}
	return parsed
}

func parseInt64Required(name string) int64 {
	value := required(name)
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("FATAL: required env var %s must be an integer: %v", name, err))
	}
	return parsed
}
