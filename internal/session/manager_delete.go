package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Delete removes one session from active runtime state and persisted history.
func (m *Manager) Delete(ctx context.Context, id string) error {
	if m == nil {
		return errors.New("session: manager is required")
	}
	if ctx == nil {
		return errors.New("session: delete context is required")
	}

	target, err := normalizeStoredSessionID(id)
	if err != nil {
		return err
	}

	if _, ok := m.Get(target); ok {
		if err := m.StopWithCause(ctx, target, CauseUserRequested, "session deleted"); err != nil {
			return err
		}
	}

	sessionDir := filepath.Join(m.homePaths.SessionsDir, target)
	if _, err := os.Stat(sessionDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrSessionNotFound, target)
		}
		return fmt.Errorf("session: stat session directory %q: %w", sessionDir, err)
	}

	if err := os.RemoveAll(sessionDir); err != nil {
		return fmt.Errorf("session: delete session directory %q: %w", sessionDir, err)
	}

	m.remove(target)
	return nil
}
