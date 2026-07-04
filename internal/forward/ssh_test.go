package forward

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSSHMounts_WithAuthSock exercises SSHMounts end-to-end using the real
// runtime.GOOS. On Linux, Docker can bind-mount the real SSH_AUTH_SOCK path
// directly, so the mount source/target/env should match it exactly. On macOS,
// Docker Desktop cannot bind-mount the host's real (launchd) socket path, so
// dis substitutes Docker Desktop's magic ssh-auth.sock instead — see
// TestResolveSSHAgentMount for the exhaustive GOOS x SSH_AUTH_SOCK matrix.
func TestSSHMounts_WithAuthSock(t *testing.T) {
	tmp := t.TempDir()
	sockPath := filepath.Join(tmp, "agent.sock")
	// Create a regular file to represent the socket for testing.
	require.NoError(t, os.WriteFile(sockPath, nil, 0600))

	t.Setenv("SSH_AUTH_SOCK", sockPath)

	// Override sshHomeDir so known_hosts isn't picked up from the real home.
	origHomeDir := sshHomeDir
	sshHomeDir = func() (string, error) { return tmp, nil }
	t.Cleanup(func() { sshHomeDir = origHomeDir })

	mounts, envVars, err := SSHMounts()
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(mounts), 1)

	wantSock := sockPath
	if runtime.GOOS == "darwin" {
		wantSock = dockerDesktopSSHAgentSock
	}

	assert.Equal(t, wantSock, mounts[0].Source)
	assert.Equal(t, wantSock, mounts[0].Target)
	assert.False(t, mounts[0].ReadOnly)

	assert.Contains(t, envVars, "SSH_AUTH_SOCK="+wantSock)
}

func TestSSHMounts_NoAuthSock(t *testing.T) {
	tmp := t.TempDir()

	t.Setenv("SSH_AUTH_SOCK", "")

	origHomeDir := sshHomeDir
	sshHomeDir = func() (string, error) { return tmp, nil }
	t.Cleanup(func() { sshHomeDir = origHomeDir })

	mounts, envVars, err := SSHMounts()
	require.NoError(t, err)
	assert.Empty(t, mounts, "no mounts when SSH_AUTH_SOCK is unset")
	assert.Empty(t, envVars, "no env vars when SSH_AUTH_SOCK is unset")
}

func TestSSHMounts_WithKnownHosts(t *testing.T) {
	tmp := t.TempDir()
	sshDir := filepath.Join(tmp, ".ssh")
	require.NoError(t, os.MkdirAll(sshDir, 0700))
	khPath := filepath.Join(sshDir, "known_hosts")
	require.NoError(t, os.WriteFile(khPath, []byte("github.com ssh-rsa AAAA...\n"), 0644))

	// No auth sock — only testing known_hosts mount.
	t.Setenv("SSH_AUTH_SOCK", "")

	origHomeDir := sshHomeDir
	sshHomeDir = func() (string, error) { return tmp, nil }
	t.Cleanup(func() { sshHomeDir = origHomeDir })

	mounts, _, err := SSHMounts()
	require.NoError(t, err)

	require.Len(t, mounts, 1)
	assert.Equal(t, khPath, mounts[0].Source)
	assert.Equal(t, "/root/.ssh/known_hosts", mounts[0].Target)
	assert.True(t, mounts[0].ReadOnly)
}

// TestResolveSSHAgentMount covers the pure decision function across the full
// GOOS x SSH_AUTH_SOCK-set/unset matrix, independent of the real host OS or
// filesystem. This is the regression test for the macOS SSH agent forwarding
// bug: Docker Desktop for Mac cannot bind-mount a host launchd socket like
// /private/tmp/com.apple.launchd.*/Listeners (container creation fails with
// "invalid mount config ... operation not supported"), so on darwin dis must
// mount Docker Desktop's magic socket instead, regardless of what
// SSH_AUTH_SOCK's actual host path looks like.
func TestResolveSSHAgentMount(t *testing.T) {
	const hostSock = "/private/tmp/com.apple.launchd.ABCDEF/Listeners"

	tests := []struct {
		name     string
		goos     string
		authSock string
		wantOK   bool
		wantHost string
		wantCtr  string
	}{
		{
			name:     "linux with SSH_AUTH_SOCK set mounts the real path unchanged",
			goos:     "linux",
			authSock: "/tmp/ssh-agent.sock",
			wantOK:   true,
			wantHost: "/tmp/ssh-agent.sock",
			wantCtr:  "/tmp/ssh-agent.sock",
		},
		{
			name:     "linux with SSH_AUTH_SOCK unset forwards nothing",
			goos:     "linux",
			authSock: "",
			wantOK:   false,
		},
		{
			name:     "darwin with SSH_AUTH_SOCK set substitutes Docker Desktop's magic socket",
			goos:     "darwin",
			authSock: hostSock,
			wantOK:   true,
			wantHost: dockerDesktopSSHAgentSock,
			wantCtr:  dockerDesktopSSHAgentSock,
		},
		{
			name:     "darwin with SSH_AUTH_SOCK unset forwards nothing",
			goos:     "darwin",
			authSock: "",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, ok := resolveSSHAgentMount(tt.goos, tt.authSock)
			require.Equal(t, tt.wantOK, ok)
			if !tt.wantOK {
				return
			}
			assert.Equal(t, tt.wantHost, agent.hostSock)
			assert.Equal(t, tt.wantCtr, agent.containerSock)
		})
	}
}
