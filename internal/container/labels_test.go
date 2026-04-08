package container

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLabels_SingleLabel(t *testing.T) {
	dir := t.TempDir()
	df := filepath.Join(dir, "Dockerfile")
	content := `FROM ubuntu:22.04
LABEL dis.shell=/bin/zsh
RUN apt-get update
`
	require.NoError(t, os.WriteFile(df, []byte(content), 0644))

	labels, err := ParseLabelsFromDockerfile(df)
	require.NoError(t, err)

	assert.Equal(t, "/bin/zsh", labels["dis.shell"])
	assert.Len(t, labels, 1)
}

func TestParseLabels_MultiLabel(t *testing.T) {
	dir := t.TempDir()
	df := filepath.Join(dir, "Dockerfile")
	content := `FROM ubuntu:22.04
LABEL dis.shell=/bin/zsh dis.ports="3000,8080" dis.workspace=/home/dev
`
	require.NoError(t, os.WriteFile(df, []byte(content), 0644))

	labels, err := ParseLabelsFromDockerfile(df)
	require.NoError(t, err)

	assert.Equal(t, "/bin/zsh", labels["dis.shell"])
	assert.Equal(t, "3000,8080", labels["dis.ports"])
	assert.Equal(t, "/home/dev", labels["dis.workspace"])
	assert.Len(t, labels, 3)
}

func TestParseLabels_QuotedValues(t *testing.T) {
	dir := t.TempDir()
	df := filepath.Join(dir, "Dockerfile")
	content := `FROM ubuntu:22.04
LABEL dis.env="NODE_ENV=development,DEBUG=true"
LABEL dis.shell='/bin/zsh'
LABEL dis.ports="3000,8080"
`
	require.NoError(t, os.WriteFile(df, []byte(content), 0644))

	labels, err := ParseLabelsFromDockerfile(df)
	require.NoError(t, err)

	assert.Equal(t, "NODE_ENV=development,DEBUG=true", labels["dis.env"])
	assert.Equal(t, "/bin/zsh", labels["dis.shell"])
	assert.Equal(t, "3000,8080", labels["dis.ports"])
}

func TestParseLabels_ContinuationLines(t *testing.T) {
	dir := t.TempDir()
	df := filepath.Join(dir, "Dockerfile")
	content := `FROM ubuntu:22.04
LABEL dis.shell=/bin/zsh \
      dis.ports="3000,8080" \
      dis.workspace=/workspace
RUN echo hello
`
	require.NoError(t, os.WriteFile(df, []byte(content), 0644))

	labels, err := ParseLabelsFromDockerfile(df)
	require.NoError(t, err)

	assert.Equal(t, "/bin/zsh", labels["dis.shell"])
	assert.Equal(t, "3000,8080", labels["dis.ports"])
	assert.Equal(t, "/workspace", labels["dis.workspace"])
	assert.Len(t, labels, 3)
}

func TestParseLabels_NonDisLabels(t *testing.T) {
	dir := t.TempDir()
	df := filepath.Join(dir, "Dockerfile")
	content := `FROM ubuntu:22.04
LABEL maintainer="dev@example.com"
LABEL version="1.0"
LABEL dis.shell=/bin/zsh
LABEL org.opencontainers.image.title="myimage"
`
	require.NoError(t, os.WriteFile(df, []byte(content), 0644))

	labels, err := ParseLabelsFromDockerfile(df)
	require.NoError(t, err)

	assert.Equal(t, "/bin/zsh", labels["dis.shell"])
	assert.Len(t, labels, 1, "only dis.* labels should be extracted")
}

func TestParseLabels_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	df := filepath.Join(dir, "Dockerfile")
	require.NoError(t, os.WriteFile(df, []byte(""), 0644))

	labels, err := ParseLabelsFromDockerfile(df)
	require.NoError(t, err)
	assert.Empty(t, labels)
}

func TestParseLabels_MissingFile(t *testing.T) {
	labels, err := ParseLabelsFromDockerfile("/nonexistent/path/Dockerfile")
	assert.Error(t, err)
	assert.Nil(t, labels)
}
