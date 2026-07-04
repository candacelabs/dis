package container

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// defaultDockerHost returns the platform-appropriate default Docker socket,
// matching the Docker CLI's own fallback when no context/env override applies.
func defaultDockerHost() string {
	if runtime.GOOS == "windows" {
		return "npipe:////./pipe/docker_engine"
	}
	return "unix:///var/run/docker.sock"
}

// dockerCLIConfig mirrors the subset of ~/.docker/config.json that dis cares about.
type dockerCLIConfig struct {
	CurrentContext string `json:"currentContext"`
}

// dockerContextMeta mirrors the subset of ~/.docker/contexts/meta/<hash>/meta.json
// that dis cares about.
type dockerContextMeta struct {
	Endpoints struct {
		Docker struct {
			Host string `json:"Host"`
		} `json:"docker"`
	} `json:"Endpoints"`
}

// ResolveDockerHost determines which Docker daemon socket dis should connect to.
// It mirrors the resolution order used by the Docker CLI itself:
//
//  1. DOCKER_HOST env var, if set — always wins, unchanged behavior.
//  2. The current Docker CLI context (~/.docker/config.json "currentContext"),
//     resolved via ~/.docker/contexts/meta/<sha256(name)>/meta.json's
//     Endpoints.docker.Host. An absent or "default" context means the
//     platform default socket.
//  3. On any lookup/parse failure, fall back to the platform default socket.
//     If that default path doesn't exist but $HOME/.docker/run/docker.sock
//     does (Docker Desktop's per-user socket convention on macOS), prefer that.
//
// ResolveDockerHost is pure given its inputs so it can be table-tested against
// fixture directories instead of the real $HOME.
func ResolveDockerHost(env func(string) string, homeDir string) string {
	if dockerHost := env("DOCKER_HOST"); dockerHost != "" {
		return dockerHost
	}

	if host, ok := resolveFromContext(homeDir); ok {
		return host
	}

	return fallbackDockerHost(homeDir)
}

// resolveFromContext reads ~/.docker/config.json and, if a non-default context
// is configured, follows it to the context's meta.json to get the daemon host.
// Returns ok=false if no usable context could be resolved (caller should fall back).
func resolveFromContext(homeDir string) (string, bool) {
	configPath := filepath.Join(homeDir, ".docker", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", false
	}

	var cfg dockerCLIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", false
	}

	if cfg.CurrentContext == "" || cfg.CurrentContext == "default" {
		return "", false
	}

	metaPath := contextMetaPath(homeDir, cfg.CurrentContext)
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return "", false
	}

	var meta dockerContextMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return "", false
	}

	if meta.Endpoints.Docker.Host == "" {
		return "", false
	}

	return meta.Endpoints.Docker.Host, true
}

// contextMetaPath returns the path to a Docker CLI context's meta.json,
// keyed by the SHA256 hex digest of the context name (matches Docker CLI layout).
func contextMetaPath(homeDir, contextName string) string {
	sum := sha256.Sum256([]byte(contextName))
	hash := hex.EncodeToString(sum[:])
	return filepath.Join(homeDir, ".docker", "contexts", "meta", hash, "meta.json")
}

// fallbackDockerHost returns the platform default Docker socket, preferring
// Docker Desktop's per-user socket on macOS if the platform default isn't present.
func fallbackDockerHost(homeDir string) string {
	def := defaultDockerHost()

	if runtime.GOOS != "windows" {
		defPath := strings.TrimPrefix(def, "unix://")

		if !socketExists(defPath) {
			userSock := filepath.Join(homeDir, ".docker", "run", "docker.sock")
			if socketExists(userSock) {
				return "unix://" + userSock
			}
		}
	}

	return def
}

// socketExists reports whether path exists and is reachable on disk. It uses
// Stat (which follows symlinks) rather than Lstat so that a dangling symlink
// — e.g. a stale /var/run/docker.sock left behind pointing at another user's
// now-deleted home directory, as happens with some Docker Desktop upgrades on
// macOS — correctly reports as not existing rather than "exists".
func socketExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
