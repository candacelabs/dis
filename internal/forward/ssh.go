package forward

import (
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/mount"
)

// sshHomeDir is the function used to resolve the user's home directory for SSH.
// Override in tests to use a temp directory.
var sshHomeDir = os.UserHomeDir

// SSHMounts returns Docker bind mounts and env vars for SSH agent forwarding.
// It mounts SSH_AUTH_SOCK (if set) and ~/.ssh/known_hosts (if present) into the container.
// Never returns an error — just returns whatever is available.
func SSHMounts() ([]mount.Mount, []string, error) {
	var mounts []mount.Mount
	var envVars []string

	// Forward SSH agent socket if available.
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock != "" && fileExists(authSock) {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   authSock,
			Target:   authSock,
			ReadOnly: false,
		})
		envVars = append(envVars, "SSH_AUTH_SOCK="+authSock)
	}

	// Mount known_hosts if present.
	home, err := sshHomeDir()
	if err != nil {
		return mounts, envVars, nil
	}

	knownHosts := filepath.Join(home, ".ssh", "known_hosts")
	if fileExists(knownHosts) {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   knownHosts,
			Target:   "/root/.ssh/known_hosts",
			ReadOnly: true,
		})
	}

	return mounts, envVars, nil
}
