package forward

import (
	"testing"

	"github.com/docker/docker/api/types/mount"
	"github.com/stretchr/testify/assert"
)

func TestHistoryVolume_Bash(t *testing.T) {
	m := HistoryVolume("myproject", "/bin/bash")

	assert.Equal(t, mount.TypeVolume, m.Type)
	assert.Equal(t, "dis-myproject-history", m.Source)
	assert.Equal(t, "/root/.bash_history", m.Target)
}

func TestHistoryVolume_Zsh(t *testing.T) {
	m := HistoryVolume("myproject", "/bin/zsh")

	assert.Equal(t, mount.TypeVolume, m.Type)
	assert.Equal(t, "dis-myproject-history", m.Source)
	assert.Equal(t, "/root/.zsh_history", m.Target)
}

func TestHistoryVolume_VolumeName(t *testing.T) {
	m := HistoryVolume("cool-app", "bash")

	assert.Equal(t, "dis-cool-app-history", m.Source, "volume name should be dis-<project>-history")
	assert.Equal(t, mount.TypeVolume, m.Type, "must be a named volume, not a bind mount")
}
