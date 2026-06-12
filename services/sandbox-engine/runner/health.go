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
	Timeout time.Duration
	docker  *client.Client
}

func NewHealthChecker(timeout time.Duration) *HealthChecker {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(fmt.Sprintf("healthchecker: docker client init failed: %v", err))
	}
	return &HealthChecker{Timeout: timeout, docker: cli}
}

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

		// Inspect to get port binding.
		info, err := h.docker.ContainerInspect(ctx, containerID)
		if err != nil {
			continue
		}
		bindings, ok := info.NetworkSettings.Ports["8080/tcp"]
		if !ok || len(bindings) == 0 {
			continue
		}
		hostPort := bindings[0].HostPort
		url := fmt.Sprintf("http://127.0.0.1:%s/health", hostPort)

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

// KillAndRemove stops and removes a container (used by the watchdog).
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
