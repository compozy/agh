package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/compozy/agh/internal/store"
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
	_, statErr := os.Stat(sessionDir)
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("session: stat session directory %q: %w", sessionDir, statErr)
	}
	resumeQueries, err := m.quiesceSessionQueries(ctx, target, sessionDir)
	if err != nil {
		return fmt.Errorf("session: quiesce stored readers for %q: %w", target, err)
	}
	defer resumeQueries()

	catalogDeleted := false
	if m.sessionCatalog != nil {
		err := m.sessionCatalog.DeleteSession(ctx, target)
		switch {
		case err == nil:
			catalogDeleted = true
		case errors.Is(err, store.ErrSessionNotFound):
			// A prior attempt can commit catalog deletion before filesystem removal fails.
		case err != nil:
			return fmt.Errorf("session: delete catalog state for %q: %w", target, err)
		}
	}

	if errors.Is(statErr, os.ErrNotExist) {
		if !catalogDeleted {
			return fmt.Errorf("%w: %s", ErrSessionNotFound, target)
		}
		m.remove(target)
		return nil
	}

	if err := os.RemoveAll(sessionDir); err != nil {
		return fmt.Errorf("session: delete session directory %q: %w", sessionDir, err)
	}

	m.remove(target)
	return nil
}

func (m *Manager) quiesceSessionQueries(
	ctx context.Context,
	target string,
	sessionDir string,
) (func(), error) {
	if m.queryStoreRuntime == nil {
		return func() {}, nil
	}
	return m.queryStoreRuntime.Quiesce(ctx, target, store.SessionDBFile(sessionDir))
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
