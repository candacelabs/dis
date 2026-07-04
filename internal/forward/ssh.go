package forward

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/docker/docker/api/types/mount"
)

// sshHomeDir is the function used to resolve the user's home directory for SSH.
// Override in tests to use a temp directory.
var sshHomeDir = os.UserHomeDir

// dockerDesktopSSHAgentSock is Docker Desktop for Mac's "magic" socket that
// proxies to the host's SSH agent. Bind-mounting it (rather than the host's
// real SSH_AUTH_SOCK path) is Docker Desktop's documented mechanism for SSH
// agent forwarding on macOS. See:
// https://docs.docker.com/desktop/features/networking/#ssh-agent-forwarding
const dockerDesktopSSHAgentSock = "/run/host-services/ssh-auth.sock"

// sshAgentMount describes how to forward the host's SSH agent socket into
// the container: which host path to bind-mount, which path to mount it at
// inside the container, and which SSH_AUTH_SOCK value the container should see.
// hostSock and containerSock are usually equal (Linux) but diverge on macOS,
// where Docker Desktop cannot bind-mount the host's real launchd socket.
type sshAgentMount struct {
	hostSock      string
	containerSock string
}

// resolveSSHAgentMount decides how to forward SSH_AUTH_SOCK into the
// container, given the host OS and the host's SSH_AUTH_SOCK value.
//
// On Linux, SSH_AUTH_SOCK is a regular Unix domain socket on disk and Docker
// can bind-mount it directly, so the container sees the same path.
//
// On macOS, SSH_AUTH_SOCK points at a launchd-managed socket (typically under
// /private/tmp/com.apple.launchd.*/Listeners) that Docker Desktop's Linux VM
// cannot bind-mount — container creation fails with "operation not supported".
// Docker Desktop instead exposes a magic socket at
// /run/host-services/ssh-auth.sock inside the VM that proxies to the host
// agent; dis mounts that instead and points the container's SSH_AUTH_SOCK at it.
//
// Returns ok=false if authSock is empty (nothing to forward).
func resolveSSHAgentMount(goos, authSock string) (sshAgentMount, bool) {
	if authSock == "" {
		return sshAgentMount{}, false
	}

	if goos == "darwin" {
		return sshAgentMount{
			hostSock:      dockerDesktopSSHAgentSock,
			containerSock: dockerDesktopSSHAgentSock,
		}, true
	}

	return sshAgentMount{
		hostSock:      authSock,
		containerSock: authSock,
	}, true
}

// SSHMounts returns Docker bind mounts and env vars for SSH agent forwarding.
// It mounts SSH_AUTH_SOCK (if set) and ~/.ssh/known_hosts (if present) into the container.
// Never returns an error — just returns whatever is available.
func SSHMounts() ([]mount.Mount, []string, error) {
	var mounts []mount.Mount
	var envVars []string

	authSock := os.Getenv("SSH_AUTH_SOCK")
	if agent, ok := resolveSSHAgentMount(runtime.GOOS, authSock); ok {
		// On Linux, hostSock is the real SSH_AUTH_SOCK path on this filesystem —
		// verify it actually exists before mounting it, same as before this fix.
		// On macOS, hostSock is Docker Desktop's magic socket path, which only
		// exists inside the Docker Desktop VM, not on the host filesystem, so
		// it can't be (and shouldn't be) checked with fileExists here.
		mountable := runtime.GOOS == "darwin" || fileExists(agent.hostSock)
		if mountable {
			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeBind,
				Source:   agent.hostSock,
				Target:   agent.containerSock,
				ReadOnly: false,
			})
			envVars = append(envVars, "SSH_AUTH_SOCK="+agent.containerSock)
		}
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
