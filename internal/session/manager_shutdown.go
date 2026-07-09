package session

import "context"

// Shutdown releases session-manager background resources after active sessions stop.
func (m *Manager) Shutdown(ctx context.Context) error {
	if m == nil {
		return nil
	}
	return m.shutdownQueryStoreRuntime(ctx)
}
