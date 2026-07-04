package container

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noEnv returns an env lookup function that always returns "".
func noEnv(string) string { return "" }

// envWith returns an env lookup function returning val for key, "" otherwise.
func envWith(key, val string) func(string) string {
	return func(k string) string {
		if k == key {
			return val
		}
		return ""
	}
}

// writeContextMeta writes a Docker CLI context meta.json fixture for contextName
// under homeDir/.docker/contexts/meta/<sha256hex(contextName)>/meta.json, with
// the given daemon host recorded in Endpoints.docker.Host.
func writeContextMeta(t *testing.T, homeDir, contextName, host string) {
	t.Helper()
	sum := sha256.Sum256([]byte(contextName))
	hash := hex.EncodeToString(sum[:])
	dir := filepath.Join(homeDir, ".docker", "contexts", "meta", hash)
	require.NoError(t, os.MkdirAll(dir, 0755))

	content := `{"Name":"` + contextName + `","Endpoints":{"docker":{"Host":"` + host + `"}}}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "meta.json"), []byte(content), 0644))
}

// writeDockerConfig writes ~/.docker/config.json with the given currentContext.
func writeDockerConfig(t *testing.T, homeDir, currentContext string) {
	t.Helper()
	dockerDir := filepath.Join(homeDir, ".docker")
	require.NoError(t, os.MkdirAll(dockerDir, 0755))
	content := `{"currentContext":"` + currentContext + `"}`
	require.NoError(t, os.WriteFile(filepath.Join(dockerDir, "config.json"), []byte(content), 0644))
}

func TestResolveDockerHost_DockerHostEnvWins(t *testing.T) {
	home := t.TempDir()
	// Even with a context configured, DOCKER_HOST must win outright.
	writeDockerConfig(t, home, "desktop-linux")
	writeContextMeta(t, home, "desktop-linux", "unix:///home/user/.docker/run/docker.sock")

	host := ResolveDockerHost(envWith("DOCKER_HOST", "tcp://1.2.3.4:2375"), home)
	assert.Equal(t, "tcp://1.2.3.4:2375", host)
}

func TestResolveDockerHost_CurrentContextResolvesToMetaHost(t *testing.T) {
	home := t.TempDir()
	writeDockerConfig(t, home, "desktop-linux")
	writeContextMeta(t, home, "desktop-linux", "unix:///Users/someone/.docker/run/docker.sock")

	host := ResolveDockerHost(noEnv, home)
	assert.Equal(t, "unix:///Users/someone/.docker/run/docker.sock", host)
}

func TestResolveDockerHost_DefaultContextUsesDefaultSocket(t *testing.T) {
	home := t.TempDir()
	writeDockerConfig(t, home, "default")

	host := ResolveDockerHost(noEnv, home)
	assert.Equal(t, defaultDockerHost(), host)
}

func TestResolveDockerHost_MissingConfigFallsBack(t *testing.T) {
	home := t.TempDir() // no .docker/config.json at all

	host := ResolveDockerHost(noEnv, home)
	assert.Equal(t, defaultDockerHost(), host)
}

func TestResolveDockerHost_CorruptConfigFallsBack(t *testing.T) {
	home := t.TempDir()
	dockerDir := filepath.Join(home, ".docker")
	require.NoError(t, os.MkdirAll(dockerDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dockerDir, "config.json"), []byte("{not valid json"), 0644))

	host := ResolveDockerHost(noEnv, home)
	assert.Equal(t, defaultDockerHost(), host)
}

func TestResolveDockerHost_MissingMetaJSONFallsBack(t *testing.T) {
	home := t.TempDir()
	// currentContext points at a context whose meta.json was never written.
	writeDockerConfig(t, home, "ghost-context")

	host := ResolveDockerHost(noEnv, home)
	assert.Equal(t, defaultDockerHost(), host)
}

func TestResolveDockerHost_CorruptMetaJSONFallsBack(t *testing.T) {
	home := t.TempDir()
	writeDockerConfig(t, home, "desktop-linux")

	sum := sha256.Sum256([]byte("desktop-linux"))
	hash := hex.EncodeToString(sum[:])
	dir := filepath.Join(home, ".docker", "contexts", "meta", hash)
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "meta.json"), []byte("{not valid json"), 0644))

	host := ResolveDockerHost(noEnv, home)
	assert.Equal(t, defaultDockerHost(), host)
}

func TestResolveDockerHost_EmptyMetaHostFallsBack(t *testing.T) {
	home := t.TempDir()
	writeDockerConfig(t, home, "desktop-linux")
	writeContextMeta(t, home, "desktop-linux", "")

	host := ResolveDockerHost(noEnv, home)
	assert.Equal(t, defaultDockerHost(), host)
}

// TestResolveDockerHost_PrefersUserSocketWhenDefaultMissing reproduces the
// real-world macOS scenario: no usable context info, and the platform default
// socket path doesn't exist/isn't reachable, but Docker Desktop's per-user
// socket convention ($HOME/.docker/run/docker.sock) does exist. dis should
// prefer the user socket rather than giving up.
//
// This only exercises the Unix branch since defaultDockerHost() on Windows
// is a named pipe, not a filesystem path we can fake with a temp dir.
func TestResolveDockerHost_PrefersUserSocketWhenDefaultMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("default docker host is a named pipe on windows, not applicable")
	}

	home := t.TempDir()
	// No .docker/config.json at all -> falls through to fallbackDockerHost.
	runDir := filepath.Join(home, ".docker", "run")
	require.NoError(t, os.MkdirAll(runDir, 0755))
	userSock := filepath.Join(runDir, "docker.sock")
	require.NoError(t, os.WriteFile(userSock, nil, 0600))

	host := ResolveDockerHost(noEnv, home)
	assert.Equal(t, "unix://"+userSock, host)
}

func TestResolveDockerHost_FallsBackToDefaultWhenNeitherSocketExists(t *testing.T) {
	home := t.TempDir() // no config, no user socket either

	host := ResolveDockerHost(noEnv, home)
	assert.Equal(t, defaultDockerHost(), host)
}
