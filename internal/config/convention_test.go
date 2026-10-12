package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeDockerfile(t *testing.T, dir string, content string) string {
	t.Helper()
	disDir := filepath.Join(dir, ".dis")
	require.NoError(t, os.MkdirAll(disDir, 0755))
	df := filepath.Join(disDir, "Dockerfile")
	require.NoError(t, os.WriteFile(df, []byte(content), 0644))
	return df
}

func TestResolveConfig_FullLabels(t *testing.T) {
	dir := t.TempDir()
	// Create a subdirectory with a known name so we can predict ProjectName
	projectDir := filepath.Join(dir, "my-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	writeDockerfile(t, projectDir, `FROM node:20
LABEL dis.ports="3000,8080"
LABEL dis.env="NODE_ENV=development,DEBUG=true"
LABEL dis.mounts="./data:/data,./config:/etc/app"
LABEL dis.shell=/bin/zsh
LABEL dis.post_create="npm install"
LABEL dis.workspace=/home/dev
`)

	cfg, err := ResolveConfig(projectDir)
	require.NoError(t, err)

	assert.Equal(t, "my-project", cfg.ProjectName)
	assert.Equal(t, filepath.Join(projectDir, ".dis", "Dockerfile"), cfg.DockerfilePath)
	assert.Equal(t, "/home/dev", cfg.WorkspaceDir)
	assert.Equal(t, []int{3000, 8080}, cfg.Ports)
	assert.Equal(t, "/bin/zsh", cfg.Shell)
	assert.Equal(t, "npm install", cfg.PostCreate)
	assert.Equal(t, "dis-my-project", cfg.ImageName)
	assert.Equal(t, "dis-my-project-dev", cfg.ContainerName)

	require.Len(t, cfg.EnvVars, 2)
	assert.Equal(t, "NODE_ENV", cfg.EnvVars[0].Key)
	assert.Equal(t, "development", cfg.EnvVars[0].Value)
	assert.Equal(t, "DEBUG", cfg.EnvVars[1].Key)
	assert.Equal(t, "true", cfg.EnvVars[1].Value)

	require.Len(t, cfg.ExtraMounts, 2)
	assert.Equal(t, "./data", cfg.ExtraMounts[0].Source)
	assert.Equal(t, "/data", cfg.ExtraMounts[0].Target)
	assert.Equal(t, "./config", cfg.ExtraMounts[1].Source)
	assert.Equal(t, "/etc/app", cfg.ExtraMounts[1].Target)
}

func TestResolveConfig_Defaults(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "bare-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	writeDockerfile(t, projectDir, `FROM ubuntu:22.04
RUN apt-get update
`)

	cfg, err := ResolveConfig(projectDir)
	require.NoError(t, err)

	assert.Equal(t, "bare-project", cfg.ProjectName)
	assert.Equal(t, "/workspace", cfg.WorkspaceDir)
	assert.Equal(t, "/bin/bash", cfg.Shell)
	assert.Empty(t, cfg.Ports)
	assert.Empty(t, cfg.EnvVars)
	assert.Empty(t, cfg.ExtraMounts)
	assert.Equal(t, "", cfg.PostCreate)
	assert.Equal(t, "dis-bare-project", cfg.ImageName)
	assert.Equal(t, "dis-bare-project-dev", cfg.ContainerName)
}

func TestResolveConfig_MissingDockerfile(t *testing.T) {
	dir := t.TempDir()

	cfg, err := ResolveConfig(dir)
	assert.Nil(t, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No .dis/Dockerfile found")
	assert.Contains(t, err.Error(), "dis init")
}

func TestResolveConfig_ProjectNaming(t *testing.T) {
	tests := []struct {
		dirName       string
		wantProject   string
		wantImage     string
		wantContainer string
	}{
		{
			dirName:       "My Cool Project",
			wantProject:   "my-cool-project",
			wantImage:     "dis-my-cool-project",
			wantContainer: "dis-my-cool-project-dev",
		},
		{
			dirName:       "SimpleApp",
			wantProject:   "simpleapp",
			wantImage:     "dis-simpleapp",
			wantContainer: "dis-simpleapp-dev",
		},
		{
			dirName:       "UPPER CASE NAME",
			wantProject:   "upper-case-name",
			wantImage:     "dis-upper-case-name",
			wantContainer: "dis-upper-case-name-dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.dirName, func(t *testing.T) {
			dir := t.TempDir()
			projectDir := filepath.Join(dir, tt.dirName)
			require.NoError(t, os.MkdirAll(projectDir, 0755))
			writeDockerfile(t, projectDir, "FROM ubuntu:22.04\n")

			cfg, err := ResolveConfig(projectDir)
			require.NoError(t, err)

			assert.Equal(t, tt.wantProject, cfg.ProjectName)
			assert.Equal(t, tt.wantImage, cfg.ImageName)
			assert.Equal(t, tt.wantContainer, cfg.ContainerName)
		})
	}
}

func TestResolveConfig_InvalidPort(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "bad-ports")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	writeDockerfile(t, projectDir, `FROM ubuntu:22.04
LABEL dis.ports="3000,abc,8080"
`)

	cfg, err := ResolveConfig(projectDir)
	assert.Nil(t, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid port")
	assert.Contains(t, err.Error(), "abc")
}

func TestResolveConfig_EmptyPorts(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "empty-ports")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	writeDockerfile(t, projectDir, `FROM ubuntu:22.04
LABEL dis.ports=""
`)

	cfg, err := ResolveConfig(projectDir)
	require.NoError(t, err)
	assert.Empty(t, cfg.Ports)
}
