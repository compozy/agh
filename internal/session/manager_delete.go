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
		return fmt.Errorf("session: normalize delete id %q: %w", id, err)
	}

	if _, ok := m.Get(target); ok {
		if err := stopSessionBeforeDelete(ctx, target, m.StopWithCause); err != nil {
			return fmt.Errorf("session: stop %q before delete: %w", target, err)
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

func stopSessionBeforeDelete(
	ctx context.Context,
	target string,
	stop func(context.Context, string, StopCause, string) error,
) error {
	err := stop(ctx, target, CauseUserRequested, "session deleted")
	if err == nil || errors.Is(err, ErrSessionNotFound) {
		return nil
	}
	return err
}
