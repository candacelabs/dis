package container

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/candacelabs/dis/internal/forward"
	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"
	"golang.org/x/term"
)

// EnvVar represents a key=value environment variable.
type EnvVar struct {
	Key   string
	Value string
}

// Mount represents a bind mount from host to container.
type Mount struct {
	Source   string
	Target   string
	ReadOnly bool
}

// Config holds the fully resolved configuration for a dis container manager.
// This mirrors config.DisConfig but lives in the container package to avoid
// an import cycle (config imports container for ParseLabelsFromDockerfile).
type Config struct {
	ProjectName    string
	DockerfilePath string // e.g., "/path/to/project/.dis/Dockerfile"
	WorkspaceDir   string // default "/workspace"
	Ports          []int
	EnvVars        []EnvVar
	ExtraMounts    []Mount
	Tmpfs          []string // container paths mounted in memory
	Shell          string   // default "/bin/bash"
	PostCreate     string
	ImageName      string // "dis-<ProjectName>"
	ContainerName  string // "dis-<ProjectName>-dev"
}

// Manager handles the Docker container lifecycle for a dis project.
type Manager struct {
	client client.APIClient
	config *Config
}

// ContainerStatus holds the inspected state of a dis container.
type ContainerStatus struct {
	ContainerName string
	State         string // "running", "stopped", "not_found"
	Image         string
	Ports         []int
	Mounts        []string
	Uptime        string
}

// NewManager creates a Manager by connecting to the local Docker daemon.
// Returns a user-friendly error if Docker is not running.
func NewManager(cfg *Config) (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	host := ResolveDockerHost(os.Getenv, home)

	// FromEnv first (TLS config + DOCKER_HOST + API version from env), then
	// WithHost so our resolved host (context-aware, with the DOCKER_HOST
	// short-circuit already applied in ResolveDockerHost) always wins over
	// FromEnv's own DOCKER_HOST handling — they agree when DOCKER_HOST is set,
	// and WithHost only takes over when DOCKER_HOST is unset.
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("Docker is not running (tried %s). Start Docker Desktop (or dockerd) and try again.", host)
	}

	return &Manager{client: cli, config: cfg}, nil
}

// NewManagerWithClient creates a Manager with a provided Docker client, for testing.
func NewManagerWithClient(cfg *Config, cli client.APIClient) *Manager {
	return &Manager{client: cli, config: cfg}
}

// Up starts the dev container. It builds the image if needed, creates and starts
// the container, and runs the post-create command if configured.
func (m *Manager) Up(ctx context.Context) error {
	running, err := m.IsRunning(ctx)
	if err != nil {
		return err
	}
	if running {
		fmt.Println("✅ Container already running:", m.config.ContainerName)
		return nil
	}

	needsBuild, reason, err := m.NeedsRebuild(ctx)
	if err != nil {
		return err
	}
	if needsBuild {
		fmt.Println("🔨 Building image:", reason)
		if err := m.buildImage(ctx); err != nil {
			return err
		}
	}

	// Remove any stopped container with same name.
	if err := m.removeContainerIfExists(ctx); err != nil {
		return err
	}

	// Assemble mounts.
	mounts, envFromMounts, err := m.assembleMounts()
	if err != nil {
		return fmt.Errorf("failed to assemble mounts: %w", err)
	}

	// Assemble ports.
	portMap, portSet, err := forward.PortBindings(m.config.Ports)
	if err != nil {
		return fmt.Errorf("failed to configure ports: %w", err)
	}

	// Assemble env vars.
	env := m.assembleEnv(envFromMounts)

	// Create container.
	containerCfg := &containertypes.Config{
		Image:        m.config.ImageName,
		Tty:          true,
		OpenStdin:    true,
		WorkingDir:   m.config.WorkspaceDir,
		Env:          env,
		ExposedPorts: portSet,
		Labels: map[string]string{
			"dis.managed": "true",
			"dis.project": m.config.ProjectName,
		},
	}

	hostCfg := &containertypes.HostConfig{
		Mounts:       mounts,
		PortBindings: portMap,
		Tmpfs:        tmpfsMounts(m.config.Tmpfs),
	}

	resp, err := m.client.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, m.config.ContainerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start container.
	if err := m.client.ContainerStart(ctx, resp.ID, containertypes.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Run post-create command if configured.
	if m.config.PostCreate != "" {
		fmt.Println("🔧 Running post-create command...")
		exitCode, err := m.Exec(ctx, []string{"sh", "-c", m.config.PostCreate}, false)
		if err != nil {
			return fmt.Errorf("post-create command failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("post-create command exited with code %d", exitCode)
		}
	}

	fmt.Println("🚀 Container started:", m.config.ContainerName)
	if len(m.config.Ports) > 0 {
		portStrs := make([]string, len(m.config.Ports))
		for i, p := range m.config.Ports {
			portStrs[i] = fmt.Sprintf("%d", p)
		}
		fmt.Println("   Ports:", strings.Join(portStrs, ", "))
	}

	return nil
}

// Down stops and removes the container. Idempotent — no error if not found.
func (m *Manager) Down(ctx context.Context) error {
	timeout := 10
	err := m.client.ContainerStop(ctx, m.config.ContainerName, containertypes.StopOptions{Timeout: &timeout})
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	err = m.client.ContainerRemove(ctx, m.config.ContainerName, containertypes.RemoveOptions{Force: true})
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
}

// Exec runs a command inside the running container.
// If interactive is true, it attaches stdin and enables TTY mode.
// Returns the exit code of the command.
func (m *Manager) Exec(ctx context.Context, cmd []string, interactive bool) (int, error) {
	execCfg := types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  interactive,
		Tty:          interactive,
	}

	execResp, err := m.client.ContainerExecCreate(ctx, m.config.ContainerName, execCfg)
	if err != nil {
		return -1, fmt.Errorf("failed to create exec: %w", err)
	}

	attachResp, err := m.client.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{
		Tty: interactive,
	})
	if err != nil {
		return -1, fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer attachResp.Close()

	if interactive {
		// Set terminal to raw mode for interactive sessions.
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return -1, fmt.Errorf("failed to set raw terminal: %w", err)
		}
		defer term.Restore(int(os.Stdin.Fd()), oldState)

		// Copy stdin → container and container → stdout concurrently.
		done := make(chan struct{})
		go func() {
			defer close(done)
			io.Copy(os.Stdout, attachResp.Reader)
		}()
		go func() {
			io.Copy(attachResp.Conn, os.Stdin)
			attachResp.CloseWrite()
		}()
		<-done
	} else {
		// Non-interactive: demux stdout/stderr.
		_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, attachResp.Reader)
		if err != nil {
			return -1, fmt.Errorf("failed to copy exec output: %w", err)
		}
	}

	// Get exit code.
	inspect, err := m.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return -1, fmt.Errorf("failed to inspect exec: %w", err)
	}

	return inspect.ExitCode, nil
}

// Status returns the current state of the dis container.
func (m *Manager) Status(ctx context.Context) (*ContainerStatus, error) {
	info, err := m.client.ContainerInspect(ctx, m.config.ContainerName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return &ContainerStatus{
				ContainerName: m.config.ContainerName,
				State:         "not_found",
			}, nil
		}
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	state := "stopped"
	if info.State != nil && info.State.Running {
		state = "running"
	}

	// Extract port numbers from bindings.
	var ports []int
	if info.NetworkSettings != nil {
		for port := range info.NetworkSettings.Ports {
			p := port.Int()
			if p > 0 {
				ports = append(ports, p)
			}
		}
	}

	// Extract mount destinations.
	var mountPaths []string
	for _, mp := range info.Mounts {
		mountPaths = append(mountPaths, mp.Destination)
	}

	// Calculate uptime.
	var uptime string
	if info.State != nil && info.State.Running && info.State.StartedAt != "" {
		startedAt, err := time.Parse(time.RFC3339Nano, info.State.StartedAt)
		if err == nil {
			uptime = time.Since(startedAt).Truncate(time.Second).String()
		}
	}

	return &ContainerStatus{
		ContainerName: m.config.ContainerName,
		State:         state,
		Image:         info.Image,
		Ports:         ports,
		Mounts:        mountPaths,
		Uptime:        uptime,
	}, nil
}

// Rebuild force-removes the current image and rebuilds it via Up.
func (m *Manager) Rebuild(ctx context.Context) error {
	// Stop and remove container first.
	if err := m.Down(ctx); err != nil {
		return err
	}

	// Remove image.
	_, err := m.client.ImageRemove(ctx, m.config.ImageName, types.ImageRemoveOptions{Force: true})
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to remove image: %w", err)
	}

	return m.Up(ctx)
}

// Logs streams container logs to stdout.
func (m *Manager) Logs(ctx context.Context, follow bool, tail string) error {
	if tail == "" {
		tail = "100"
	}

	reader, err := m.client.ContainerLogs(ctx, m.config.ContainerName, containertypes.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tail,
	})
	if err != nil {
		return fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	return err
}

// NeedsRebuild checks if the Dockerfile has changed since the image was built.
// Returns (needsRebuild, reason, error).
func (m *Manager) NeedsRebuild(ctx context.Context) (bool, string, error) {
	currentHash, err := dockerfileHash(m.config.DockerfilePath)
	if err != nil {
		return true, "cannot read Dockerfile", err
	}

	imageInfo, _, err := m.client.ImageInspectWithRaw(ctx, m.config.ImageName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return true, "image not found", nil
		}
		return false, "", fmt.Errorf("failed to inspect image: %w", err)
	}

	storedHash := ""
	if imageInfo.Config != nil && imageInfo.Config.Labels != nil {
		storedHash = imageInfo.Config.Labels["dis.dockerfile.hash"]
	}

	if storedHash != currentHash {
		return true, "Dockerfile changed", nil
	}

	return false, "", nil
}

// IsRunning returns true if the container exists and is in running state.
func (m *Manager) IsRunning(ctx context.Context) (bool, error) {
	info, err := m.client.ContainerInspect(ctx, m.config.ContainerName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to inspect container: %w", err)
	}

	return info.State != nil && info.State.Running, nil
}

// assembleMounts constructs all mount points and returns any extra env vars from forwarding.
func (m *Manager) assembleMounts() ([]mount.Mount, []string, error) {
	var allMounts []mount.Mount
	var allEnv []string

	// 1. Workspace bind mount: project root → /workspace
	projectRoot := filepath.Dir(filepath.Dir(m.config.DockerfilePath)) // go up from .dis/Dockerfile
	allMounts = append(allMounts, mount.Mount{
		Type:   mount.TypeBind,
		Source: projectRoot,
		Target: m.config.WorkspaceDir,
	})

	// 2. Git credential forwarding.
	gitMounts, gitEnv, err := forward.GitMounts()
	if err != nil {
		return nil, nil, fmt.Errorf("git mounts: %w", err)
	}
	allMounts = append(allMounts, gitMounts...)
	allEnv = append(allEnv, gitEnv...)

	// 3. SSH agent forwarding.
	sshMounts, sshEnv, err := forward.SSHMounts()
	if err != nil {
		return nil, nil, fmt.Errorf("ssh mounts: %w", err)
	}
	allMounts = append(allMounts, sshMounts...)
	allEnv = append(allEnv, sshEnv...)

	// 3b. GitHub CLI login forwarding.
	ghMounts, ghEnv, err := forward.GHMounts()
	if err != nil {
		return nil, nil, fmt.Errorf("gh mounts: %w", err)
	}
	allMounts = append(allMounts, ghMounts...)
	allEnv = append(allEnv, ghEnv...)

	// 4. Shell history volume.
	allMounts = append(allMounts, forward.HistoryVolume(m.config.ProjectName, m.config.Shell))

	// 5. Extra mounts from config.
	for _, em := range m.config.ExtraMounts {
		allMounts = append(allMounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   em.Source,
			Target:   em.Target,
			ReadOnly: em.ReadOnly,
		})
	}

	return allMounts, allEnv, nil
}

// tmpfsMounts is the in-memory mounts for the container's host config:
// readable, writable and executable by every user, so a toolchain can run
// what it builds there. Docker mounts tmpfs noexec unless told otherwise.
func tmpfsMounts(targets []string) map[string]string {
	if len(targets) == 0 {
		return nil
	}
	mounts := make(map[string]string, len(targets))
	for _, target := range targets {
		mounts[target] = "rw,exec,mode=1777"
	}
	return mounts
}

// assembleEnv combines config env vars with forwarding env vars.
func (m *Manager) assembleEnv(forwardEnv []string) []string {
	var env []string

	// Config-defined env vars.
	for _, ev := range m.config.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", ev.Key, ev.Value))
	}

	// Forwarding env vars (git, ssh).
	env = append(env, forwardEnv...)

	return env
}

// buildImage creates a Docker image from the .dis/ directory.
func (m *Manager) buildImage(ctx context.Context) error {
	disDir := filepath.Dir(m.config.DockerfilePath) // .dis/

	tarBuf, err := createTarArchive(disDir)
	if err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}

	currentHash, err := dockerfileHash(m.config.DockerfilePath)
	if err != nil {
		return fmt.Errorf("failed to hash Dockerfile: %w", err)
	}

	buildResp, err := m.client.ImageBuild(ctx, tarBuf, types.ImageBuildOptions{
		Tags:       []string{m.config.ImageName},
		Dockerfile: "Dockerfile",
		Remove:     true,
		Labels: map[string]string{
			"dis.dockerfile.hash": currentHash,
			"dis.project":         m.config.ProjectName,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer buildResp.Body.Close()

	// Drain the build output so the build actually completes.
	_, err = io.Copy(os.Stdout, buildResp.Body)
	if err != nil {
		return fmt.Errorf("error reading build output: %w", err)
	}

	return nil
}

// removeContainerIfExists removes a container by name, ignoring not-found errors.
func (m *Manager) removeContainerIfExists(ctx context.Context) error {
	err := m.client.ContainerRemove(ctx, m.config.ContainerName, containertypes.RemoveOptions{Force: true})
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to remove existing container: %w", err)
	}
	return nil
}

// dockerfileHash returns the SHA256 hex digest of a Dockerfile's contents.
func dockerfileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read Dockerfile: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// createTarArchive creates an in-memory tar archive of the given directory.
func createTarArchive(dir string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})

	return buf, err
}
