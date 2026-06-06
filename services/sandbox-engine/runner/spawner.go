package runner

import "fmt"

type Spawner struct {
	BenchNetName string
}

func NewSpawner(benchNetName string) *Spawner {
	return &Spawner{BenchNetName: benchNetName}
}

func (s *Spawner) Spawn(imageTag, submissionID string) (string, int, error) {
	if imageTag == "" || submissionID == "" {
		return "", 0, fmt.Errorf("spawner: image tag and submission id are required")
	}
	return "container-" + submissionID, 8080, nil
}
