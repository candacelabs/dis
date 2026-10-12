package container

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var _ = Describe("Up with dis.tmpfs", func() {
	It("mounts each path in memory, readable, writable and executable by every user", func(ctx SpecContext) {
		directory := GinkgoT().TempDir()
		Expect(os.MkdirAll(filepath.Join(directory, ".dis"), 0o755)).To(Succeed())
		dockerfile := filepath.Join(directory, ".dis", "Dockerfile")
		Expect(os.WriteFile(dockerfile, []byte("FROM ubuntu:22.04\n"), 0o644)).To(Succeed())

		var created *containertypes.HostConfig
		docker := &mockDockerClient{
			containerInspectFn: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
				return types.ContainerJSON{}, errdefs.NotFound(errors.New("not found"))
			},
			imageInspectFn: func(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
				return types.ImageInspect{}, nil, errdefs.NotFound(errors.New("not found"))
			},
			imageBuildFn: func(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
				return types.ImageBuildResponse{Body: io.NopCloser(strings.NewReader(`{"stream":"built"}`))}, nil
			},
			containerRemoveFn: func(ctx context.Context, containerID string, options containertypes.RemoveOptions) error {
				return errdefs.NotFound(errors.New("not found"))
			},
			containerCreateFn: func(ctx context.Context, config *containertypes.Config, hostConfig *containertypes.HostConfig,
				networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (containertypes.CreateResponse, error) {
				created = hostConfig
				return containertypes.CreateResponse{ID: "container-memory"}, nil
			},
		}
		manager := NewManagerWithClient(&Config{
			ProjectName:    "memory",
			DockerfilePath: dockerfile,
			WorkspaceDir:   "/src",
			Tmpfs:          []string{"/tmp", "/cache"},
			Shell:          "/bin/bash",
			ImageName:      "dis-memory",
			ContainerName:  "dis-memory-dev",
		}, docker)

		Expect(manager.Up(ctx)).To(Succeed())
		Expect(created.Tmpfs).To(Equal(map[string]string{
			"/tmp":   "rw,exec,mode=1777",
			"/cache": "rw,exec,mode=1777",
		}))
	})
})
