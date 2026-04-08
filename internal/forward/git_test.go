package forward

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitMounts_WithGitconfig(t *testing.T) {
	tmp := t.TempDir()
	gitconfigPath := filepath.Join(tmp, ".gitconfig")
	require.NoError(t, os.WriteFile(gitconfigPath, []byte("[user]\n\tname = Test\n"), 0644))

	// Override the homeDir function for this test.
	origHomeDir := homeDir
	homeDir = func() (string, error) { return tmp, nil }
	t.Cleanup(func() { homeDir = origHomeDir })

	mounts, _, err := GitMounts()
	require.NoError(t, err)

	require.Len(t, mounts, 1)
	assert.Equal(t, gitconfigPath, mounts[0].Source)
	assert.Equal(t, "/root/.gitconfig", mounts[0].Target)
	assert.True(t, mounts[0].ReadOnly)
}

func TestGitMounts_WithXDGConfig(t *testing.T) {
	tmp := t.TempDir()
	xdgDir := filepath.Join(tmp, ".config", "git")
	require.NoError(t, os.MkdirAll(xdgDir, 0755))
	xdgConfig := filepath.Join(xdgDir, "config")
	require.NoError(t, os.WriteFile(xdgConfig, []byte("[user]\n\tname = Test\n"), 0644))

	origHomeDir := homeDir
	homeDir = func() (string, error) { return tmp, nil }
	t.Cleanup(func() { homeDir = origHomeDir })

	mounts, _, err := GitMounts()
	require.NoError(t, err)

	require.Len(t, mounts, 1)
	assert.Equal(t, xdgConfig, mounts[0].Source)
	assert.Equal(t, "/root/.config/git/config", mounts[0].Target)
	assert.True(t, mounts[0].ReadOnly)
}

func TestGitMounts_NoConfig(t *testing.T) {
	tmp := t.TempDir() // empty directory — no git config files

	origHomeDir := homeDir
	homeDir = func() (string, error) { return tmp, nil }
	t.Cleanup(func() { homeDir = origHomeDir })

	mounts, _, err := GitMounts()
	require.NoError(t, err)
	assert.Empty(t, mounts, "mounts should be empty when no git config exists")
}
