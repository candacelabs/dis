package forward

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/mount"
)

// homeDir is the function used to resolve the user's home directory.
// Override in tests to use a temp directory.
var homeDir = os.UserHomeDir

// GitMounts returns Docker bind mounts and env vars for git credential forwarding.
// It mounts ~/.gitconfig (or ~/.config/git/config) read-only into the container
// and extracts GIT_AUTHOR_NAME, GIT_AUTHOR_EMAIL, GIT_COMMITTER_NAME,
// GIT_COMMITTER_EMAIL from the host's git config.
// Never returns an error for missing config — just returns less.
func GitMounts() ([]mount.Mount, []string, error) {
	var mounts []mount.Mount
	var envVars []string

	home, err := homeDir()
	if err != nil {
		// Can't determine home dir — return empty, not an error.
		return mounts, envVars, nil
	}

	// Try ~/.gitconfig first, then XDG location.
	gitconfig := filepath.Join(home, ".gitconfig")
	if fileExists(gitconfig) {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   gitconfig,
			Target:   "/root/.gitconfig",
			ReadOnly: true,
		})
	} else {
		xdgConfig := filepath.Join(home, ".config", "git", "config")
		if fileExists(xdgConfig) {
			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeBind,
				Source:   xdgConfig,
				Target:   "/root/.config/git/config",
				ReadOnly: true,
			})
		}
	}

	// Extract git user identity from host config.
	name := gitConfigValue("user.name")
	email := gitConfigValue("user.email")

	if name != "" {
		envVars = append(envVars, fmt.Sprintf("GIT_AUTHOR_NAME=%s", name))
		envVars = append(envVars, fmt.Sprintf("GIT_COMMITTER_NAME=%s", name))
	}
	if email != "" {
		envVars = append(envVars, fmt.Sprintf("GIT_AUTHOR_EMAIL=%s", email))
		envVars = append(envVars, fmt.Sprintf("GIT_COMMITTER_EMAIL=%s", email))
	}

	return mounts, envVars, nil
}

// gitConfigValue runs `git config --global <key>` and returns the trimmed output.
// Returns empty string if git is not available or the key is not set.
func gitConfigValue(key string) string {
	cmd := exec.Command("git", "config", "--global", key)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// fileExists checks whether a path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
