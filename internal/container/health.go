package container

import (
	"context"
	"fmt"
	"time"
)

// WaitForReady polls the container until it reaches running state or times out.
// Default timeout is 30 seconds with 500ms polling interval.
func (m *Manager) WaitForReady(ctx context.Context) error {
	timeout := 30 * time.Second
	interval := 500 * time.Millisecond

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		running, err := m.IsRunning(ctx)
		if err != nil {
			return fmt.Errorf("failed to check container status: %w", err)
		}
		if running {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
			// Continue polling.
		}
	}

	return fmt.Errorf("container %s did not become ready within %s", m.config.ContainerName, timeout)
}
