package container

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDockerClient implements the client.APIClient interface for testing.
// Only the methods used by the Manager are overridden; all others panic.
type mockDockerClient struct {
	client.APIClient // embed to satisfy the full interface

	pingFn               func(ctx context.Context) (types.Ping, error)
	containerInspectFn   func(ctx context.Context, containerID string) (types.ContainerJSON, error)
	containerCreateFn    func(ctx context.Context, config *containertypes.Config, hostConfig *containertypes.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (containertypes.CreateResponse, error)
	containerStartFn     func(ctx context.Context, containerID string, options containertypes.StartOptions) error
	containerStopFn      func(ctx context.Context, containerID string, options containertypes.StopOptions) error
	containerRemoveFn    func(ctx context.Context, containerID string, options containertypes.RemoveOptions) error
	containerLogsFn      func(ctx context.Context, containerID string, options containertypes.LogsOptions) (io.ReadCloser, error)
	containerExecCreateFn  func(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error)
	containerExecAttachFn  func(ctx context.Context, execID string, config types.ExecStartCheck) (types.HijackedResponse, error)
	containerExecInspectFn func(ctx context.Context, execID string) (types.ContainerExecInspect, error)
	imageBuildFn         func(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error)
	imageInspectFn       func(ctx context.Context, imageID string) (types.ImageInspect, []byte, error)
	imageRemoveFn        func(ctx context.Context, imageID string, options types.ImageRemoveOptions) ([]imagetypes.DeleteResponse, error)
}

func (m *mockDockerClient) Ping(ctx context.Context) (types.Ping, error) {
	if m.pingFn != nil {
		return m.pingFn(ctx)
	}
	return types.Ping{}, nil
}

func (m *mockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	if m.containerInspectFn != nil {
		return m.containerInspectFn(ctx, containerID)
	}
	return types.ContainerJSON{}, errdefs.NotFound(errors.New("container not found"))
}

func (m *mockDockerClient) ContainerStop(ctx context.Context, containerID string, options containertypes.StopOptions) error {
	if m.containerStopFn != nil {
		return m.containerStopFn(ctx, containerID, options)
	}
	return nil
}

func (m *mockDockerClient) ContainerRemove(ctx context.Context, containerID string, options containertypes.RemoveOptions) error {
	if m.containerRemoveFn != nil {
		return m.containerRemoveFn(ctx, containerID, options)
	}
	return nil
}

func (m *mockDockerClient) ContainerLogs(ctx context.Context, containerID string, options containertypes.LogsOptions) (io.ReadCloser, error) {
	if m.containerLogsFn != nil {
		return m.containerLogsFn(ctx, containerID, options)
	}
	return io.NopCloser(strings.NewReader("mock logs")), nil
}

func (m *mockDockerClient) ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error) {
	if m.containerExecCreateFn != nil {
		return m.containerExecCreateFn(ctx, container, config)
	}
	return types.IDResponse{ID: "exec-123"}, nil
}

func (m *mockDockerClient) ContainerExecAttach(ctx context.Context, execID string, config types.ExecStartCheck) (types.HijackedResponse, error) {
	if m.containerExecAttachFn != nil {
		return m.containerExecAttachFn(ctx, execID, config)
	}
	// Return a no-op hijacked response with a pipe connection.
	serverConn, clientConn := net.Pipe()
	serverConn.Close() // close server side so reads return EOF
	return types.NewHijackedResponse(clientConn, "application/vnd.docker.raw-stream"), nil
}

func (m *mockDockerClient) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	if m.containerExecInspectFn != nil {
		return m.containerExecInspectFn(ctx, execID)
	}
	return types.ContainerExecInspect{ExitCode: 0}, nil
}

func (m *mockDockerClient) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	if m.imageInspectFn != nil {
		return m.imageInspectFn(ctx, imageID)
	}
	return types.ImageInspect{}, nil, errdefs.NotFound(errors.New("image not found"))
}

func (m *mockDockerClient) ImageRemove(ctx context.Context, imageID string, options types.ImageRemoveOptions) ([]imagetypes.DeleteResponse, error) {
	if m.imageRemoveFn != nil {
		return m.imageRemoveFn(ctx, imageID, options)
	}
	return nil, nil
}

func (m *mockDockerClient) ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	if m.imageBuildFn != nil {
		return m.imageBuildFn(ctx, buildContext, options)
	}
	return types.ImageBuildResponse{
		Body: io.NopCloser(strings.NewReader(`{"stream":"built"}`)),
	}, nil
}

func (m *mockDockerClient) ContainerCreate(ctx context.Context, config *containertypes.Config, hostConfig *containertypes.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (containertypes.CreateResponse, error) {
	if m.containerCreateFn != nil {
		return m.containerCreateFn(ctx, config, hostConfig, networkingConfig, platform, containerName)
	}
	return containertypes.CreateResponse{ID: "container-123"}, nil
}

func (m *mockDockerClient) ContainerStart(ctx context.Context, containerID string, options containertypes.StartOptions) error {
	if m.containerStartFn != nil {
		return m.containerStartFn(ctx, containerID, options)
	}
	return nil
}

// --- Helper to create a test config ---

func testConfig(t *testing.T) *Config {
	t.Helper()
	dir := t.TempDir()
	disDir := filepath.Join(dir, ".dis")
	require.NoError(t, os.MkdirAll(disDir, 0755))

	dockerfilePath := filepath.Join(disDir, "Dockerfile")
	require.NoError(t, os.WriteFile(dockerfilePath, []byte(`FROM ubuntu:22.04
LABEL dis.shell=/bin/bash
`), 0644))

	return &Config{
		ProjectName:    "testproject",
		DockerfilePath: dockerfilePath,
		WorkspaceDir:   "/workspace",
		Shell:          "/bin/bash",
		ImageName:      "dis-testproject",
		ContainerName:  "dis-testproject-dev",
	}
}

// =============================================================================
// IsRunning tests
// =============================================================================

func TestIsRunning_Running(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		containerInspectFn: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{Running: true},
				},
			}, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	running, err := mgr.IsRunning(context.Background())
	require.NoError(t, err)
	assert.True(t, running)
}

func TestIsRunning_Stopped(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		containerInspectFn: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{Running: false},
				},
			}, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	running, err := mgr.IsRunning(context.Background())
	require.NoError(t, err)
	assert.False(t, running)
}

func TestIsRunning_NotFound(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		containerInspectFn: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
			return types.ContainerJSON{}, errdefs.NotFound(errors.New("not found"))
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	running, err := mgr.IsRunning(context.Background())
	require.NoError(t, err)
	assert.False(t, running)
}

// =============================================================================
// NeedsRebuild tests
// =============================================================================

func TestNeedsRebuild_NoImage(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		imageInspectFn: func(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
			return types.ImageInspect{}, nil, errdefs.NotFound(errors.New("not found"))
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	needs, reason, err := mgr.NeedsRebuild(context.Background())
	require.NoError(t, err)
	assert.True(t, needs)
	assert.Equal(t, "image not found", reason)
}

func TestNeedsRebuild_HashMatch(t *testing.T) {
	cfg := testConfig(t)

	// Compute the current hash of the test Dockerfile.
	currentHash, err := dockerfileHash(cfg.DockerfilePath)
	require.NoError(t, err)

	mock := &mockDockerClient{
		imageInspectFn: func(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
			return types.ImageInspect{
				Config: &containertypes.Config{
					Labels: map[string]string{
						"dis.dockerfile.hash": currentHash,
					},
				},
			}, nil, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	needs, reason, err := mgr.NeedsRebuild(context.Background())
	require.NoError(t, err)
	assert.False(t, needs)
	assert.Empty(t, reason)
}

func TestNeedsRebuild_HashMismatch(t *testing.T) {
	cfg := testConfig(t)

	mock := &mockDockerClient{
		imageInspectFn: func(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
			return types.ImageInspect{
				Config: &containertypes.Config{
					Labels: map[string]string{
						"dis.dockerfile.hash": "stale-hash-value",
					},
				},
			}, nil, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	needs, reason, err := mgr.NeedsRebuild(context.Background())
	require.NoError(t, err)
	assert.True(t, needs)
	assert.Equal(t, "Dockerfile changed", reason)
}

func TestNeedsRebuild_NoLabelsOnImage(t *testing.T) {
	cfg := testConfig(t)

	mock := &mockDockerClient{
		imageInspectFn: func(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
			return types.ImageInspect{
				Config: &containertypes.Config{},
			}, nil, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	needs, reason, err := mgr.NeedsRebuild(context.Background())
	require.NoError(t, err)
	assert.True(t, needs)
	assert.Equal(t, "Dockerfile changed", reason)
}

// =============================================================================
// Down tests (idempotency)
// =============================================================================

func TestDown_Running(t *testing.T) {
	cfg := testConfig(t)
	stopCalled := false
	removeCalled := false

	mock := &mockDockerClient{
		containerStopFn: func(ctx context.Context, containerID string, options containertypes.StopOptions) error {
			stopCalled = true
			assert.Equal(t, cfg.ContainerName, containerID)
			return nil
		},
		containerRemoveFn: func(ctx context.Context, containerID string, options containertypes.RemoveOptions) error {
			removeCalled = true
			assert.Equal(t, cfg.ContainerName, containerID)
			assert.True(t, options.Force)
			return nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	err := mgr.Down(context.Background())
	require.NoError(t, err)
	assert.True(t, stopCalled)
	assert.True(t, removeCalled)
}

func TestDown_NotFound(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		containerStopFn: func(ctx context.Context, containerID string, options containertypes.StopOptions) error {
			return errdefs.NotFound(errors.New("no such container"))
		},
		containerRemoveFn: func(ctx context.Context, containerID string, options containertypes.RemoveOptions) error {
			return errdefs.NotFound(errors.New("no such container"))
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	// Should NOT return an error — Down is idempotent.
	err := mgr.Down(context.Background())
	require.NoError(t, err)
}

func TestDown_StopError(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		containerStopFn: func(ctx context.Context, containerID string, options containertypes.StopOptions) error {
			return errors.New("connection refused")
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	err := mgr.Down(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stop container")
}

// =============================================================================
// Status tests
// =============================================================================

func TestStatus_Running(t *testing.T) {
	cfg := testConfig(t)
	startedAt := time.Now().Add(-5 * time.Minute).Format(time.RFC3339Nano)

	mock := &mockDockerClient{
		containerInspectFn: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Running:   true,
						StartedAt: startedAt,
					},
					Image: "sha256:abc123",
				},
				Mounts: []types.MountPoint{
					{Destination: "/workspace"},
					{Destination: "/root/.bash_history"},
				},
			}, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	status, err := mgr.Status(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "running", status.State)
	assert.Equal(t, cfg.ContainerName, status.ContainerName)
	assert.Equal(t, "sha256:abc123", status.Image)
	assert.Contains(t, status.Mounts, "/workspace")
	assert.Contains(t, status.Mounts, "/root/.bash_history")
	assert.NotEmpty(t, status.Uptime)
}

func TestStatus_NotFound(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		containerInspectFn: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
			return types.ContainerJSON{}, errdefs.NotFound(errors.New("not found"))
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	status, err := mgr.Status(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "not_found", status.State)
	assert.Equal(t, cfg.ContainerName, status.ContainerName)
}

func TestStatus_Stopped(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		containerInspectFn: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{Running: false},
					Image: "sha256:abc123",
				},
			}, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	status, err := mgr.Status(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "stopped", status.State)
}

// =============================================================================
// assembleMounts tests
// =============================================================================

func TestAssembleMounts_WorkspaceMount(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{}
	mgr := NewManagerWithClient(cfg, mock)

	mounts, _, err := mgr.assembleMounts()
	require.NoError(t, err)

	// First mount should be the workspace bind mount.
	require.NotEmpty(t, mounts)
	assert.Equal(t, mount.TypeBind, mounts[0].Type)
	assert.Equal(t, "/workspace", mounts[0].Target)
}

func TestAssembleMounts_HistoryVolume(t *testing.T) {
	cfg := testConfig(t)
	cfg.Shell = "/bin/bash"
	mock := &mockDockerClient{}
	mgr := NewManagerWithClient(cfg, mock)

	mounts, _, err := mgr.assembleMounts()
	require.NoError(t, err)

	// Find the history volume mount.
	var found bool
	for _, m := range mounts {
		if m.Type == mount.TypeVolume && m.Target == "/root/.bash_history" {
			assert.Equal(t, "dis-testproject-history", m.Source)
			found = true
			break
		}
	}
	assert.True(t, found, "should include history volume mount")
}

func TestAssembleMounts_ExtraMounts(t *testing.T) {
	cfg := testConfig(t)
	cfg.ExtraMounts = []Mount{
		{Source: "/host/data", Target: "/container/data", ReadOnly: true},
		{Source: "/host/cache", Target: "/container/cache", ReadOnly: false},
	}
	mock := &mockDockerClient{}
	mgr := NewManagerWithClient(cfg, mock)

	mounts, _, err := mgr.assembleMounts()
	require.NoError(t, err)

	// Find extra mounts.
	var dataMount, cacheMount bool
	for _, m := range mounts {
		if m.Target == "/container/data" {
			assert.Equal(t, "/host/data", m.Source)
			assert.True(t, m.ReadOnly)
			dataMount = true
		}
		if m.Target == "/container/cache" {
			assert.Equal(t, "/host/cache", m.Source)
			assert.False(t, m.ReadOnly)
			cacheMount = true
		}
	}
	assert.True(t, dataMount, "should include data mount")
	assert.True(t, cacheMount, "should include cache mount")
}

func TestAssembleMounts_ZshHistoryVolume(t *testing.T) {
	cfg := testConfig(t)
	cfg.Shell = "/bin/zsh"
	mock := &mockDockerClient{}
	mgr := NewManagerWithClient(cfg, mock)

	mounts, _, err := mgr.assembleMounts()
	require.NoError(t, err)

	var found bool
	for _, m := range mounts {
		if m.Type == mount.TypeVolume && m.Target == "/root/.zsh_history" {
			found = true
			break
		}
	}
	assert.True(t, found, "should use zsh history target for zsh shell")
}

// =============================================================================
// assembleEnv tests
// =============================================================================

func TestAssembleEnv_ConfigVars(t *testing.T) {
	cfg := testConfig(t)
	cfg.EnvVars = []EnvVar{
		{Key: "NODE_ENV", Value: "development"},
		{Key: "DEBUG", Value: "true"},
	}
	mock := &mockDockerClient{}
	mgr := NewManagerWithClient(cfg, mock)

	env := mgr.assembleEnv(nil)
	assert.Contains(t, env, "NODE_ENV=development")
	assert.Contains(t, env, "DEBUG=true")
}

func TestAssembleEnv_ForwardVars(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{}
	mgr := NewManagerWithClient(cfg, mock)

	forwardEnv := []string{"GIT_AUTHOR_NAME=test", "SSH_AUTH_SOCK=/tmp/ssh"}
	env := mgr.assembleEnv(forwardEnv)
	assert.Contains(t, env, "GIT_AUTHOR_NAME=test")
	assert.Contains(t, env, "SSH_AUTH_SOCK=/tmp/ssh")
}

func TestAssembleEnv_Combined(t *testing.T) {
	cfg := testConfig(t)
	cfg.EnvVars = []EnvVar{
		{Key: "APP_ENV", Value: "dev"},
	}
	mock := &mockDockerClient{}
	mgr := NewManagerWithClient(cfg, mock)

	forwardEnv := []string{"GIT_AUTHOR_NAME=test"}
	env := mgr.assembleEnv(forwardEnv)

	assert.Len(t, env, 2)
	assert.Contains(t, env, "APP_ENV=dev")
	assert.Contains(t, env, "GIT_AUTHOR_NAME=test")
}

// =============================================================================
// dockerfileHash tests
// =============================================================================

func TestDockerfileHash_Consistent(t *testing.T) {
	dir := t.TempDir()
	df := filepath.Join(dir, "Dockerfile")
	require.NoError(t, os.WriteFile(df, []byte("FROM ubuntu:22.04\n"), 0644))

	hash1, err := dockerfileHash(df)
	require.NoError(t, err)

	hash2, err := dockerfileHash(df)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2, "same content should produce same hash")
	assert.Len(t, hash1, 64, "SHA256 hex digest should be 64 chars")
}

func TestDockerfileHash_DifferentContent(t *testing.T) {
	dir := t.TempDir()
	df := filepath.Join(dir, "Dockerfile")

	require.NoError(t, os.WriteFile(df, []byte("FROM ubuntu:22.04\n"), 0644))
	hash1, err := dockerfileHash(df)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(df, []byte("FROM ubuntu:24.04\n"), 0644))
	hash2, err := dockerfileHash(df)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "different content should produce different hash")
}

func TestDockerfileHash_MissingFile(t *testing.T) {
	_, err := dockerfileHash("/nonexistent/Dockerfile")
	assert.Error(t, err)
}

// =============================================================================
// createTarArchive tests
// =============================================================================

func TestCreateTarArchive(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM ubuntu\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "setup.sh"), []byte("#!/bin/sh\necho hi\n"), 0644))

	buf, err := createTarArchive(dir)
	require.NoError(t, err)
	assert.True(t, buf.Len() > 0, "tar archive should not be empty")
}

// =============================================================================
// Up tests
// =============================================================================

func TestUp_AlreadyRunning(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		containerInspectFn: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{Running: true},
				},
			}, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	// Capture stdout to check the "already running" message.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := mgr.Up(context.Background())
	require.NoError(t, err)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	assert.Contains(t, buf.String(), "already running")
}

func TestUp_FullLifecycle(t *testing.T) {
	cfg := testConfig(t)
	createCalled := false
	startCalled := false
	inspectCallCount := 0

	mock := &mockDockerClient{
		containerInspectFn: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
			inspectCallCount++
			// First call is IsRunning check — not found.
			return types.ContainerJSON{}, errdefs.NotFound(errors.New("not found"))
		},
		imageInspectFn: func(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
			// Image not found → triggers build.
			return types.ImageInspect{}, nil, errdefs.NotFound(errors.New("not found"))
		},
		imageBuildFn: func(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
			assert.Equal(t, []string{"dis-testproject"}, options.Tags)
			assert.NotEmpty(t, options.Labels["dis.dockerfile.hash"])
			return types.ImageBuildResponse{
				Body: io.NopCloser(strings.NewReader(`{"stream":"built"}`)),
			}, nil
		},
		containerCreateFn: func(ctx context.Context, config *containertypes.Config, hostConfig *containertypes.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (containertypes.CreateResponse, error) {
			createCalled = true
			assert.Equal(t, "dis-testproject", config.Image)
			assert.Equal(t, "/workspace", config.WorkingDir)
			assert.True(t, config.Tty)
			assert.True(t, config.OpenStdin)
			assert.Equal(t, "dis-testproject-dev", containerName)
			return containertypes.CreateResponse{ID: "container-abc"}, nil
		},
		containerStartFn: func(ctx context.Context, containerID string, options containertypes.StartOptions) error {
			startCalled = true
			assert.Equal(t, "container-abc", containerID)
			return nil
		},
		containerRemoveFn: func(ctx context.Context, containerID string, options containertypes.RemoveOptions) error {
			return errdefs.NotFound(errors.New("not found"))
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	// Suppress stdout.
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := mgr.Up(context.Background())
	w.Close()
	os.Stdout = old

	require.NoError(t, err)
	assert.True(t, createCalled, "ContainerCreate should be called")
	assert.True(t, startCalled, "ContainerStart should be called")
}

func TestUp_WithPostCreate(t *testing.T) {
	cfg := testConfig(t)
	cfg.PostCreate = "npm install"
	execCreated := false

	mock := &mockDockerClient{
		containerInspectFn: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
			return types.ContainerJSON{}, errdefs.NotFound(errors.New("not found"))
		},
		imageInspectFn: func(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
			return types.ImageInspect{}, nil, errdefs.NotFound(errors.New("not found"))
		},
		imageBuildFn: func(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
			return types.ImageBuildResponse{
				Body: io.NopCloser(strings.NewReader(`{"stream":"built"}`)),
			}, nil
		},
		containerRemoveFn: func(ctx context.Context, containerID string, options containertypes.RemoveOptions) error {
			return errdefs.NotFound(errors.New("not found"))
		},
		containerExecCreateFn: func(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error) {
			execCreated = true
			assert.Equal(t, []string{"sh", "-c", "npm install"}, config.Cmd)
			return types.IDResponse{ID: "exec-post"}, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := mgr.Up(context.Background())
	w.Close()
	os.Stdout = old

	require.NoError(t, err)
	assert.True(t, execCreated, "post-create exec should be called")
}

func TestUp_WithPorts(t *testing.T) {
	cfg := testConfig(t)
	cfg.Ports = []int{3000, 8080}

	mock := &mockDockerClient{
		containerInspectFn: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
			return types.ContainerJSON{}, errdefs.NotFound(errors.New("not found"))
		},
		imageInspectFn: func(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
			return types.ImageInspect{}, nil, errdefs.NotFound(errors.New("not found"))
		},
		imageBuildFn: func(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
			return types.ImageBuildResponse{
				Body: io.NopCloser(strings.NewReader(`{"stream":"built"}`)),
			}, nil
		},
		containerRemoveFn: func(ctx context.Context, containerID string, options containertypes.RemoveOptions) error {
			return errdefs.NotFound(errors.New("not found"))
		},
		containerCreateFn: func(ctx context.Context, config *containertypes.Config, hostConfig *containertypes.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (containertypes.CreateResponse, error) {
			// Verify ports are configured.
			assert.Len(t, config.ExposedPorts, 2, "should have 2 exposed ports")
			assert.Len(t, hostConfig.PortBindings, 2, "should have 2 port bindings")
			return containertypes.CreateResponse{ID: "container-abc"}, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := mgr.Up(context.Background())
	w.Close()
	os.Stdout = old

	require.NoError(t, err)
}

// =============================================================================
// Exec tests
// =============================================================================

func TestExec_NonInteractive(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		containerExecCreateFn: func(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error) {
			assert.Equal(t, []string{"ls", "-la"}, config.Cmd)
			assert.True(t, config.AttachStdout)
			assert.True(t, config.AttachStderr)
			assert.False(t, config.AttachStdin)
			assert.False(t, config.Tty)
			return types.IDResponse{ID: "exec-ls"}, nil
		},
		containerExecInspectFn: func(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
			return types.ContainerExecInspect{ExitCode: 0}, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	// Suppress stdout.
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	exitCode, err := mgr.Exec(context.Background(), []string{"ls", "-la"}, false)
	w.Close()
	os.Stdout = old

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
}

func TestExec_NonZeroExitCode(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		containerExecInspectFn: func(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
			return types.ContainerExecInspect{ExitCode: 42}, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	exitCode, err := mgr.Exec(context.Background(), []string{"false"}, false)
	w.Close()
	os.Stdout = old

	require.NoError(t, err)
	assert.Equal(t, 42, exitCode)
}

func TestExec_CreateError(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		containerExecCreateFn: func(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error) {
			return types.IDResponse{}, errors.New("container not running")
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	_, err := mgr.Exec(context.Background(), []string{"ls"}, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create exec")
}

// =============================================================================
// Logs tests
// =============================================================================

func TestLogs_DefaultTail(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		containerLogsFn: func(ctx context.Context, containerID string, options containertypes.LogsOptions) (io.ReadCloser, error) {
			assert.Equal(t, "100", options.Tail, "default tail should be 100")
			assert.True(t, options.ShowStdout)
			assert.True(t, options.ShowStderr)
			assert.False(t, options.Follow)
			return io.NopCloser(strings.NewReader("log line 1\nlog line 2\n")), nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := mgr.Logs(context.Background(), false, "")
	w.Close()
	os.Stdout = old

	require.NoError(t, err)
}

func TestLogs_Follow(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		containerLogsFn: func(ctx context.Context, containerID string, options containertypes.LogsOptions) (io.ReadCloser, error) {
			assert.True(t, options.Follow)
			assert.Equal(t, "50", options.Tail)
			return io.NopCloser(strings.NewReader("log")), nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := mgr.Logs(context.Background(), true, "50")
	w.Close()
	os.Stdout = old

	require.NoError(t, err)
}

// =============================================================================
// WaitForReady tests (health.go)
// =============================================================================

func TestWaitForReady_AlreadyRunning(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		containerInspectFn: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{Running: true},
				},
			}, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	err := mgr.WaitForReady(context.Background())
	require.NoError(t, err)
}

func TestWaitForReady_BecomesReady(t *testing.T) {
	cfg := testConfig(t)
	callCount := 0
	mock := &mockDockerClient{
		containerInspectFn: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
			callCount++
			if callCount < 3 {
				return types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{Running: false},
					},
				}, nil
			}
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{Running: true},
				},
			}, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	err := mgr.WaitForReady(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, callCount, 3)
}

func TestWaitForReady_ContextCancelled(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{
		containerInspectFn: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{Running: false},
				},
			}, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := mgr.WaitForReady(ctx)
	assert.Error(t, err)
}

// =============================================================================
// Rebuild tests
// =============================================================================

func TestRebuild_CallsDownThenUp(t *testing.T) {
	cfg := testConfig(t)
	downCalled := false
	imageRemoveCalled := false

	mock := &mockDockerClient{
		containerStopFn: func(ctx context.Context, containerID string, options containertypes.StopOptions) error {
			downCalled = true
			return errdefs.NotFound(errors.New("not found"))
		},
		containerRemoveFn: func(ctx context.Context, containerID string, options containertypes.RemoveOptions) error {
			return errdefs.NotFound(errors.New("not found"))
		},
		imageRemoveFn: func(ctx context.Context, imageID string, options types.ImageRemoveOptions) ([]imagetypes.DeleteResponse, error) {
			imageRemoveCalled = true
			assert.Equal(t, cfg.ImageName, imageID)
			assert.True(t, options.Force)
			return nil, nil
		},
		// Up() flow: IsRunning → NeedsRebuild → build → create → start
		containerInspectFn: func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
			return types.ContainerJSON{}, errdefs.NotFound(errors.New("not found"))
		},
		imageInspectFn: func(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
			return types.ImageInspect{}, nil, errdefs.NotFound(errors.New("not found"))
		},
		imageBuildFn: func(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
			return types.ImageBuildResponse{
				Body: io.NopCloser(strings.NewReader(`{"stream":"built"}`)),
			}, nil
		},
	}
	mgr := NewManagerWithClient(cfg, mock)

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := mgr.Rebuild(context.Background())
	w.Close()
	os.Stdout = old

	require.NoError(t, err)
	assert.True(t, downCalled, "Down should be called")
	assert.True(t, imageRemoveCalled, "ImageRemove should be called")
}

// =============================================================================
// NewManagerWithClient tests
// =============================================================================

func TestNewManagerWithClient(t *testing.T) {
	cfg := testConfig(t)
	mock := &mockDockerClient{}

	mgr := NewManagerWithClient(cfg, mock)
	assert.NotNil(t, mgr)
	assert.Equal(t, cfg, mgr.config)
}
