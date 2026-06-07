package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all telemetry-ingester configuration loaded from environment variables.
type Config struct {
	PostgresDSN    string // POSTGRES_DSN: PostgreSQL connection string
	WindowSec      int    // TELEMETRY_WINDOW_SEC: aggregation window size in seconds
	GRPCListenAddr string // TELEMETRY_GRPC_ADDR: address to listen on (e.g. ":9003")
}

// Load reads all required environment variables and returns a Config.
// It panics immediately on any missing or unparseable required variable.
func Load() *Config {
	return &Config{
		PostgresDSN:    required("POSTGRES_DSN"),
		WindowSec:      requiredInt("TELEMETRY_WINDOW_SEC"),
		GRPCListenAddr: required("TELEMETRY_GRPC_ADDR"),
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
