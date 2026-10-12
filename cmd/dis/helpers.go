package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/candacelabs/dis/internal/config"
	"github.com/candacelabs/dis/internal/container"
	"github.com/candacelabs/dis/internal/proxy"
)

// mustResolveConfig resolves config or exits with helpful error.
func mustResolveConfig() *config.DisConfig {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	cfg, err := config.ResolveConfig(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return cfg
}

// mustCreateManager creates a container manager or exits with helpful error.
func mustCreateManager(cfg *config.DisConfig) *container.Manager {
	containerCfg := toContainerConfig(cfg)
	mgr, err := container.NewManager(containerCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return mgr
}

// toContainerConfig converts config.DisConfig → container.Config to avoid
// the import cycle (config imports container for ParseLabelsFromDockerfile,
// so container cannot import config).
func toContainerConfig(cfg *config.DisConfig) *container.Config {
	envVars := make([]container.EnvVar, len(cfg.EnvVars))
	for i, ev := range cfg.EnvVars {
		envVars[i] = container.EnvVar{Key: ev.Key, Value: ev.Value}
	}
	mounts := make([]container.Mount, len(cfg.ExtraMounts))
	for i, m := range cfg.ExtraMounts {
		mounts[i] = container.Mount{Source: m.Source, Target: m.Target, ReadOnly: m.ReadOnly}
	}
	return &container.Config{
		ProjectName:    cfg.ProjectName,
		DockerfilePath: cfg.DockerfilePath,
		WorkspaceDir:   cfg.WorkspaceDir,
		Ports:          cfg.Ports,
		EnvVars:        envVars,
		ExtraMounts:    mounts,
		Tmpfs:          cfg.Tmpfs,
		Shell:          cfg.Shell,
		PostCreate:     cfg.PostCreate,
		ImageName:      cfg.ImageName,
		ContainerName:  cfg.ContainerName,
	}
}

// handleExitCode checks if err is a proxy.ExitCodeError and exits with that code.
func handleExitCode(err error) {
	if err == nil {
		return
	}
	var exitErr *proxy.ExitCodeError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.Code)
	}
}
