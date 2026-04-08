package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRunner implements ContainerRunner for testing.
type mockRunner struct {
	running    bool
	runningErr error
	execCode   int
	execErr    error
	execCalls  []execCall // record what was called
}

type execCall struct {
	cmd         []string
	interactive bool
}

func (m *mockRunner) IsRunning(ctx context.Context) (bool, error) {
	return m.running, m.runningErr
}

func (m *mockRunner) Exec(ctx context.Context, cmd []string, interactive bool) (int, error) {
	m.execCalls = append(m.execCalls, execCall{cmd: cmd, interactive: interactive})
	return m.execCode, m.execErr
}

// --- RunCommand tests ---

func TestRunCommand_Success(t *testing.T) {
	runner := &mockRunner{running: true, execCode: 0}

	err := RunCommand(context.Background(), runner, []string{"go", "test", "./..."})

	require.NoError(t, err)
	require.Len(t, runner.execCalls, 1)
	assert.Equal(t, []string{"go", "test", "./..."}, runner.execCalls[0].cmd)
	assert.False(t, runner.execCalls[0].interactive)
}

func TestRunCommand_NotRunning(t *testing.T) {
	runner := &mockRunner{running: false}

	err := RunCommand(context.Background(), runner, []string{"echo", "hello"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
	assert.Contains(t, err.Error(), "dis up")
	// Exec should never be called if container is not running
	assert.Empty(t, runner.execCalls)
}

func TestRunCommand_NonZeroExit(t *testing.T) {
	runner := &mockRunner{running: true, execCode: 1}

	err := RunCommand(context.Background(), runner, []string{"false"})

	require.Error(t, err)
	var exitErr *ExitCodeError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 1, exitErr.Code)
}

func TestRunCommand_CommandPassthrough(t *testing.T) {
	runner := &mockRunner{running: true, execCode: 0}
	args := []string{"make", "-j4", "build", "VERBOSE=1"}

	err := RunCommand(context.Background(), runner, args)

	require.NoError(t, err)
	require.Len(t, runner.execCalls, 1)
	assert.Equal(t, args, runner.execCalls[0].cmd)
	assert.False(t, runner.execCalls[0].interactive, "RunCommand should not be interactive")
}

func TestRunCommand_ExecError(t *testing.T) {
	runner := &mockRunner{running: true, execErr: errors.New("docker daemon gone")}

	err := RunCommand(context.Background(), runner, []string{"ls"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "docker daemon gone")
	// Should NOT be an ExitCodeError — it's a real execution failure
	var exitErr *ExitCodeError
	assert.False(t, errors.As(err, &exitErr))
}

func TestRunCommand_IsRunningError(t *testing.T) {
	runner := &mockRunner{runningErr: errors.New("inspect failed")}

	err := RunCommand(context.Background(), runner, []string{"ls"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "inspect failed")
	assert.Empty(t, runner.execCalls)
}

func TestRunCommand_ExitCode42(t *testing.T) {
	runner := &mockRunner{running: true, execCode: 42}

	err := RunCommand(context.Background(), runner, []string{"exit", "42"})

	require.Error(t, err)
	var exitErr *ExitCodeError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 42, exitErr.Code)
	assert.Equal(t, "command exited with code 42", exitErr.Error())
}

// --- InteractiveShell tests ---

func TestInteractiveShell_Success(t *testing.T) {
	runner := &mockRunner{running: true, execCode: 0}

	err := InteractiveShell(context.Background(), runner, "/bin/bash")

	require.NoError(t, err)
	require.Len(t, runner.execCalls, 1)
	assert.Equal(t, []string{"/bin/bash"}, runner.execCalls[0].cmd)
	assert.True(t, runner.execCalls[0].interactive)
}

func TestInteractiveShell_NotRunning(t *testing.T) {
	runner := &mockRunner{running: false}

	err := InteractiveShell(context.Background(), runner, "/bin/bash")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
	assert.Contains(t, err.Error(), "dis up")
	assert.Empty(t, runner.execCalls)
}

func TestInteractiveShell_UsesConfigShell(t *testing.T) {
	runner := &mockRunner{running: true, execCode: 0}

	err := InteractiveShell(context.Background(), runner, "/bin/zsh")

	require.NoError(t, err)
	require.Len(t, runner.execCalls, 1)
	assert.Equal(t, []string{"/bin/zsh"}, runner.execCalls[0].cmd)
	assert.True(t, runner.execCalls[0].interactive, "InteractiveShell should be interactive")
}

func TestInteractiveShell_NonZeroExit(t *testing.T) {
	runner := &mockRunner{running: true, execCode: 130}

	err := InteractiveShell(context.Background(), runner, "/bin/bash")

	require.Error(t, err)
	var exitErr *ExitCodeError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 130, exitErr.Code)
}

func TestInteractiveShell_ExecError(t *testing.T) {
	runner := &mockRunner{running: true, execErr: errors.New("attach failed")}

	err := InteractiveShell(context.Background(), runner, "/bin/bash")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "attach failed")
}

// --- ExitCodeError tests ---

func TestExitCodeError_Message(t *testing.T) {
	err := &ExitCodeError{Code: 127}
	assert.Equal(t, "command exited with code 127", err.Error())
}

func TestExitCodeError_Unwrap(t *testing.T) {
	err := &ExitCodeError{Code: 1}
	// Verify it satisfies the error interface
	var genericErr error = err
	assert.NotNil(t, genericErr)
}
