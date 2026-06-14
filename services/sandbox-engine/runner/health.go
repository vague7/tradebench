package runner

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type HealthChecker struct {
	Timeout      time.Duration
	BenchNetName string
	docker       *client.Client
}

func NewHealthChecker(timeout time.Duration, benchNetName string) *HealthChecker {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(fmt.Sprintf("healthchecker: docker client init failed: %v", err))
	}
	return &HealthChecker{Timeout: timeout, BenchNetName: benchNetName, docker: cli}
}

// WaitReady polls the container's /health endpoint until it returns 200 or the
// timeout expires. It reaches the container via its IP on bench-net directly,
// which is reachable from sandbox-engine when sandbox-engine is also on bench-net.
func (h *HealthChecker) WaitReady(ctx context.Context, containerID string) error {
	if containerID == "" {
		return fmt.Errorf("healthchecker: empty container ID")
	}

	deadline := time.Now().Add(h.Timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	httpClient := &http.Client{Timeout: 2 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("healthchecker: timed out waiting for container %s", containerID)
		}

		info, err := h.docker.ContainerInspect(ctx, containerID)
		if err != nil {
			continue
		}

		// Prefer direct container IP on bench-net (sandbox-engine is also on bench-net).
		ip := containerIPOnNetwork(info, h.BenchNetName)
		if ip == "" {
			// Fallback: any IP Docker assigned.
			ip = info.NetworkSettings.IPAddress
		}
		if ip == "" {
			continue
		}

		url := fmt.Sprintf("http://%s:8080/health", ip)
		resp, err := httpClient.Get(url)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return nil
		}
	}
}

// containerIPOnNetwork returns the container's IP on the named Docker network.
func containerIPOnNetwork(info types.ContainerJSON, networkName string) string {
	if info.NetworkSettings == nil {
		return ""
	}
	if ep, ok := info.NetworkSettings.Networks[networkName]; ok && ep != nil {
		return ep.IPAddress
	}
	return ""
}

// KillAndRemove stops and removes a container (used by watchdog and gRPC KillContainer).
func (h *HealthChecker) KillAndRemove(containerID string) error {
	ctx := context.Background()
	timeout := 10
	if err := h.docker.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("healthchecker: stop container: %w", err)
	}
	if err := h.docker.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("healthchecker: remove container: %w", err)
	}
	return nil
}
