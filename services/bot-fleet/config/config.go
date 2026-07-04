package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all bot-fleet configuration loaded from environment variables.
// Field names match PRD Section 7 exactly.
type Config struct {
	TimeoutMs        int     // BOT_TIMEOUT_MS: per-request timeout in milliseconds
	TargetTPS        float64 // BOT_TARGET_TPS: target TPS for observability logging
	TelemetryGRPCAddr string // TELEMETRY_GRPC_ADDR: gRPC address of telemetry-ingester
	FleetGRPCPort    string  // BOT_FLEET_GRPC_PORT: port for bot-fleet's own gRPC server

	WarmupDuration int
	WarmupCount    int

	RampDuration int
	RampCount    int

	SustainedDuration int
	SustainedCount    int

	SpikeDuration int
	SpikeCount    int

	DrainDuration int
}

// Load reads all required environment variables and returns a Config.
// It panics immediately on any missing or unparseable required variable.
// Called once at startup in main.go before anything else.
func Load() *Config {
	port := os.Getenv("BOT_FLEET_GRPC_PORT")
	if port == "" {
		port = ":9002"
	}

	return &Config{
		TimeoutMs:         requiredInt("BOT_TIMEOUT_MS"),
		TargetTPS:         requiredFloat64("BOT_TARGET_TPS"),
		TelemetryGRPCAddr: required("TELEMETRY_GRPC_ADDR"),
		FleetGRPCPort:     port,
		WarmupDuration: requiredInt("BOT_WARMUP_DURATION"),
		WarmupCount: requiredInt("BOT_WARMUP_COUNT"),
		RampDuration: requiredInt("BOT_RAMP_DURATION"),
		RampCount: requiredInt("BOT_RAMP_COUNT"),
		SustainedDuration: requiredInt("BOT_SUSTAINED_DURATION"),
		SustainedCount: requiredInt("BOT_SUSTAINED_COUNT"),
		SpikeDuration: requiredInt("BOT_SPIKE_DURATION"),
		SpikeCount: requiredInt("BOT_SPIKE_COUNT"),
		DrainDuration: requiredInt("BOT_DRAIN_DURATION"),
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
		panic(fmt.Sprintf("FATAL: required env var %s must be a float: %v", name, err))
	}
	return parsed
}
