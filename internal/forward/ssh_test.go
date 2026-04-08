package forward

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	// Should have the auth sock mount.
	require.GreaterOrEqual(t, len(mounts), 1)
	assert.Equal(t, sockPath, mounts[0].Source)
	assert.Equal(t, sockPath, mounts[0].Target)
	assert.False(t, mounts[0].ReadOnly)

	assert.Contains(t, envVars, "SSH_AUTH_SOCK="+sockPath)
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
