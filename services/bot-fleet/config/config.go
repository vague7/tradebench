package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all bot-fleet configuration loaded from environment variables.
type Config struct {
	BotDefaultCount int    // BOT_DEFAULT_COUNT: default concurrent bots
	BotTimeoutMs    int    // BOT_TIMEOUT_MS: per-request timeout in milliseconds
	TelemetryAddr   string // TELEMETRY_ADDR: gRPC address of telemetry-ingester
}

// Load reads all required environment variables and returns a Config.
// It panics immediately on any missing or unparseable required variable.
func Load() *Config {
	return &Config{
		BotDefaultCount: requiredInt("BOT_DEFAULT_COUNT"),
		BotTimeoutMs:    requiredInt("BOT_TIMEOUT_MS"),
		TelemetryAddr:   required("TELEMETRY_ADDR"),
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
