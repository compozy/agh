package session

import (
	"context"
	"errors"
	"fmt"
)

// Shutdown releases session-manager background resources after active sessions stop.
func (m *Manager) Shutdown(ctx context.Context) error {
	if m == nil {
		return nil
	}
	var shutdownErr error
	if err := m.WaitForPromptDrains(ctx); err != nil {
		shutdownErr = errors.Join(shutdownErr, fmt.Errorf("session: wait for prompt drains during shutdown: %w", err))
	}
	if err := m.shutdownQueryStoreRuntime(ctx); err != nil {
		shutdownErr = errors.Join(shutdownErr, fmt.Errorf("session: shut down query store runtime: %w", err))
	}
	return shutdownErr
}
