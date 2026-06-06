package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	PostgresDSN       string
	RedisAddr         string
	TelemetryWindowSec int
	BotTargetTPS      int
	BotMaxP99Ms       int
}

func Load() *Config {
	return &Config{
		PostgresDSN:        required("POSTGRES_DSN"),
		RedisAddr:          required("REDIS_ADDR"),
		TelemetryWindowSec: requiredInt("TELEMETRY_WINDOW_SEC"),
		BotTargetTPS:       requiredInt("BOT_TARGET_TPS"),
		BotMaxP99Ms:        requiredInt("BOT_MAX_P99_MS"),
	}
}

func required(name string) string {
	value := os.Getenv(name)
	if value == "" {
		panic(fmt.Sprintf("FATAL: required env var %s is not set", name))
	}
	return value
}

func requiredInt(name string) int {
	value := required(name)
	parsed, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("FATAL: required env var %s must be an integer: %v", name, err))
	}
	return parsed
}
