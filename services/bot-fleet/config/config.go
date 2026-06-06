package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	BotDefaultCount int
	BotTimeoutMs    int
	BotTargetTPS    int
	BotMaxP99Ms     int
	TelemetryAddr   string
}

func Load() *Config {
	return &Config{
		BotDefaultCount: requiredInt("BOT_DEFAULT_COUNT"),
		BotTimeoutMs:    requiredInt("BOT_TIMEOUT_MS"),
		BotTargetTPS:    requiredInt("BOT_TARGET_TPS"),
		BotMaxP99Ms:     requiredInt("BOT_MAX_P99_MS"),
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
