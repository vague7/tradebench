package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	RedisAddr            string
	SandboxMaxConcurrent int
	SandboxBuildTimeout  int
	SandboxHealthTimeout int
	SandboxContainerTTL  int
	BenchNetName         string
}

func Load() *Config {
	return &Config{
		RedisAddr:            required("REDIS_ADDR"),
		SandboxMaxConcurrent: requiredInt("SANDBOX_MAX_CONCURRENT"),
		SandboxBuildTimeout:   requiredInt("SANDBOX_BUILD_TIMEOUT"),
		SandboxHealthTimeout:  requiredInt("SANDBOX_HEALTH_TIMEOUT"),
		SandboxContainerTTL:   requiredInt("SANDBOX_CONTAINER_TTL"),
		BenchNetName:          required("BENCH_NET_NAME"),
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
