package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all telemetry-ingester configuration loaded from environment variables.
type Config struct {
	PostgresDSN string  // POSTGRES_DSN: PostgreSQL connection string
	WindowSec   int     // TELEMETRY_WINDOW_SEC: aggregation window size in seconds
	TargetTPS   float64 // BOT_TARGET_TPS: used by scoring engine
	MaxP99Ms    float64 // BOT_MAX_P99_MS: used by scoring engine
	GRPCPort    string  // TELEMETRY_GRPC_PORT: gRPC listen address (e.g. ":9003")
}

// Load reads all required environment variables and returns a Config.
// It panics immediately on any missing or unparseable required variable.
func Load() *Config {
	grpcPort := os.Getenv("TELEMETRY_GRPC_PORT")
	if grpcPort == "" {
		grpcPort = ":9003"
	}

	return &Config{
		PostgresDSN: required("POSTGRES_DSN"),
		WindowSec:   requiredInt("TELEMETRY_WINDOW_SEC"),
		TargetTPS:   requiredFloat64("BOT_TARGET_TPS"),
		MaxP99Ms:    requiredFloat64("BOT_MAX_P99_MS"),
		GRPCPort:    grpcPort,
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

func requiredFloat64(name string) float64 {
	value := required(name)
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		panic(fmt.Sprintf("FATAL: required env var %s must be a float64: %v", name, err))
	}
	return parsed
}
