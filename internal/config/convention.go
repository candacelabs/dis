package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/candacelabs/dis/internal/container"
)

// DisConfig is the fully resolved configuration for a dis project.
type DisConfig struct {
	ProjectName    string
	DockerfilePath string // always ".dis/Dockerfile"
	WorkspaceDir   string // default "/workspace"
	Ports          []int
	EnvVars        []EnvVar
	ExtraMounts    []Mount
	Shell          string // default "/bin/bash"
	PostCreate     string
	ImageName      string // "dis-<ProjectName>"
	ContainerName  string // "dis-<ProjectName>-dev"
}

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

// ResolveConfig finds .dis/Dockerfile in projectRoot, parses labels, applies defaults,
// and returns a fully resolved DisConfig.
func ResolveConfig(projectRoot string) (*DisConfig, error) {
	dockerfilePath := filepath.Join(projectRoot, ".dis", "Dockerfile")

	if !DockerfileExists(projectRoot) {
		return nil, fmt.Errorf("No .dis/Dockerfile found in %s. Run 'dis init' to set up.", projectRoot)
	}

	labels, err := container.ParseLabelsFromDockerfile(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Dockerfile labels: %w", err)
	}

	projectName := sanitizeProjectName(filepath.Base(projectRoot))

	cfg := &DisConfig{
		ProjectName:    projectName,
		DockerfilePath: dockerfilePath,
		WorkspaceDir:   "/workspace",
		Shell:          "/bin/bash",
		ImageName:      "dis-" + projectName,
		ContainerName:  "dis-" + projectName + "-dev",
	}

	// Apply label overrides
	if v, ok := labels["dis.shell"]; ok && v != "" {
		cfg.Shell = v
	}

	if v, ok := labels["dis.workspace"]; ok && v != "" {
		cfg.WorkspaceDir = v
	}

	if v, ok := labels["dis.post_create"]; ok {
		cfg.PostCreate = v
	}

	// Parse ports
	if v, ok := labels["dis.ports"]; ok {
		ports, err := parsePorts(v)
		if err != nil {
			return nil, err
		}
		cfg.Ports = ports
	}

	// Parse env vars
	if v, ok := labels["dis.env"]; ok && v != "" {
		cfg.EnvVars = parseEnvVars(v)
	}

	// Parse mounts
	if v, ok := labels["dis.mounts"]; ok && v != "" {
		cfg.ExtraMounts = parseMounts(v)
	}

	return cfg, nil
}

// DockerfileExists checks if .dis/Dockerfile exists in the project root.
func DockerfileExists(projectRoot string) bool {
	dockerfilePath := filepath.Join(projectRoot, ".dis", "Dockerfile")
	info, err := os.Stat(dockerfilePath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// sanitizeProjectName lowercases a name and replaces spaces with hyphens.
func sanitizeProjectName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

// parsePorts splits a comma-separated port string and converts each to int.
func parsePorts(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	ports := make([]int, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		port, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid port %q: must be a number", p)
		}
		ports = append(ports, port)
	}

	return ports, nil
}

// parseEnvVars splits a comma-separated env string, each item split by first "=" into key/value.
func parseEnvVars(s string) []EnvVar {
	parts := strings.Split(s, ",")
	envs := make([]EnvVar, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		eqIdx := strings.Index(p, "=")
		if eqIdx == -1 {
			// Key only, no value
			envs = append(envs, EnvVar{Key: p, Value: ""})
			continue
		}
		envs = append(envs, EnvVar{
			Key:   p[:eqIdx],
			Value: p[eqIdx+1:],
		})
	}

	return envs
}

// parseMounts splits a comma-separated mount string, each item split by ":" into source/target.
func parseMounts(s string) []Mount {
	parts := strings.Split(s, ",")
	mounts := make([]Mount, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		colonIdx := strings.Index(p, ":")
		if colonIdx == -1 {
			// Source only, target defaults to same path
			mounts = append(mounts, Mount{Source: p, Target: p})
			continue
		}
		mounts = append(mounts, Mount{
			Source: p[:colonIdx],
			Target: p[colonIdx+1:],
		})
	}

	return mounts
}
