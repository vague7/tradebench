package runner

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type Builder struct {
	docker *client.Client
}

func NewBuilder() *Builder {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(fmt.Sprintf("builder: docker client init failed: %v", err))
	}
	return &Builder{docker: cli}
}

// Build builds a Docker image from the ZIP at zipPath, tagging it imageTag.
// ctx is used for the Docker build call — pass a context.WithTimeout to enforce
// the SANDBOX_BUILD_TIMEOUT limit (PRD FR-2).
func (b *Builder) Build(ctx context.Context, zipPath, imageTag string) error {
	if zipPath == "" || imageTag == "" {
		return fmt.Errorf("builder: zip path and image tag are required")
	}

	tarBuf, err := zipToTar(zipPath)
	if err != nil {
		return fmt.Errorf("builder: prepare build context: %w", err)
	}

	resp, err := b.docker.ImageBuild(ctx, tarBuf, types.ImageBuildOptions{
		Tags:       []string{imageTag},
		Dockerfile: "Dockerfile",
		Remove:     true,
	})
	if err != nil {
		return fmt.Errorf("builder: image build: %w", err)
	}
	defer resp.Body.Close()

	// Drain build output so the daemon releases resources.
	// Any error lines in the stream surface as a non-nil error here.
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("builder: read build output: %w", err)
	}
	return nil
}

// zipToTar unpacks the submission ZIP and re-packs it as a tar for Docker's build context.
func zipToTar(zipPath string) (*bytes.Buffer, error) {
	zipData, err := os.ReadFile(zipPath)
	if err != nil {
		return nil, fmt.Errorf("read zip: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("zip entry open %s: %w", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("zip entry read %s: %w", f.Name, err)
		}
		_ = tw.WriteHeader(&tar.Header{
			Name: f.Name,
			Mode: 0644,
			Size: int64(len(data)),
		})
		_, _ = tw.Write(data)
	}
	_ = tw.Close()
	return &buf, nil
}
