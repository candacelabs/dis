package proxy

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// ContainerRunner is an interface for running commands in a container.
// This abstracts the container.Manager for testability.
type ContainerRunner interface {
	IsRunning(ctx context.Context) (bool, error)
	Exec(ctx context.Context, cmd []string, interactive bool) (int, error)
}

// ExitCodeError is returned when a command completes with a non-zero exit code.
type ExitCodeError struct {
	Code int
}

func (e *ExitCodeError) Error() string {
	return fmt.Sprintf("command exited with code %d", e.Code)
}

// errNotRunning is the user-facing error when the container isn't up.
const errNotRunning = "Dev container is not running. Run 'dis up' (or 'just dev') first."

// RunCommand executes a non-interactive command inside the dev container.
// It propagates SIGINT/SIGTERM via context cancellation.
func RunCommand(ctx context.Context, runner ContainerRunner, args []string) error {
	running, err := runner.IsRunning(ctx)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}
	if !running {
		return fmt.Errorf("%s", errNotRunning)
	}

	// Set up signal handling: cancel context on SIGINT or SIGTERM.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	exitCode, err := runner.Exec(ctx, args, false)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return &ExitCodeError{Code: exitCode}
	}

	return nil
}

// InteractiveShell opens an interactive shell session inside the dev container.
// The shell parameter specifies which shell to use (e.g., "/bin/bash").
func InteractiveShell(ctx context.Context, runner ContainerRunner, shell string) error {
	running, err := runner.IsRunning(ctx)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}
	if !running {
		return fmt.Errorf("%s", errNotRunning)
	}

	exitCode, err := runner.Exec(ctx, []string{shell}, true)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return &ExitCodeError{Code: exitCode}
	}

	return nil
}
