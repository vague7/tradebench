package runner

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type Spawner struct {
	BenchNetName string
	docker       *client.Client
}

func NewSpawner(benchNetName string) *Spawner {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		panic(fmt.Sprintf(
			"spawner: docker client init failed: %v",
			err,
		))
	}

	return &Spawner{
		BenchNetName: benchNetName,
		docker:       cli,
	}
}

func (s *Spawner) Spawn(
	imageTag,
	submissionID string,
) (string, int, error) {

	if imageTag == "" || submissionID == "" {
		return "", 0, fmt.Errorf(
			"spawner: image tag and submission id are required",
		)
	}

	shortID := submissionID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	containerName := "submission-" + shortID

	ctx := context.Background()

	resp, err := s.docker.ContainerCreate(
		ctx,

		&container.Config{
			Image:    imageTag,
			Hostname: containerName,
			ExposedPorts: nat.PortSet{
    			"8080/tcp": struct{}{},
			},
		},

		&container.HostConfig{
			ReadonlyRootfs: true,

			SecurityOpt: []string{
				"no-new-privileges:true",
			},

			CapDrop: []string{
				"ALL",
			},

			Tmpfs: map[string]string{
				"/tmp": "size=64m",
			},

			Resources: container.Resources{
				Memory:    512 * 1024 * 1024,
				NanoCPUs:  1_000_000_000,
				PidsLimit: int64Ptr(128),
			},
		},

		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				s.BenchNetName: {},
			},
		},

		nil,

		containerName,
	)

	if err != nil {
		return "", 0,
			fmt.Errorf(
				"spawner: container create: %w",
				err,
			)
	}

	if err := s.docker.ContainerStart(
		ctx,
		resp.ID,
		types.ContainerStartOptions{},
	); err != nil {

		return "", 0,
			fmt.Errorf(
				"spawner: container start: %w",
				err,
			)
	}

	return resp.ID, 8080, nil
}

func int64Ptr(v int64) *int64 {
	return &v
}
