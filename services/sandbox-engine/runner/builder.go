package runner

import "fmt"

type Builder struct{}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Build(zipPath, imageTag string) error {
	if zipPath == "" || imageTag == "" {
		return fmt.Errorf("builder: zip path and image tag are required")
	}
	return nil
}
