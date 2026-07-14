package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type sessionFinalization struct {
	done chan struct{}
	err  error
}

func (m *Manager) claimOrWaitFinalization(ctx context.Context, session *Session) (bool, error) {
	owned, finalization := m.claimFinalization(session)
	if owned || finalization == nil {
		return owned, nil
	}

	select {
	case <-finalization.done:
		return false, finalization.err
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

func (m *Manager) finishFinalization(id string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if finalization, ok := m.finalizing[id]; ok {
		finalization.err = err
		close(finalization.done)
	}
	delete(m.finalizing, id)
}

func (m *Manager) stopTarget(id string) (*Session, *sessionFinalization, error) {
	target := strings.TrimSpace(id)
	if target == "" {
		return nil, nil, errors.New("session: session id is required")
	}

	m.mu.RLock()
	session := m.sessions[target]
	finalization := m.finalizing[target]
	m.mu.RUnlock()
	if session == nil && finalization == nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrSessionNotFound, target)
	}
	return session, finalization, nil
}

func waitForSessionFinalization(ctx context.Context, finalization *sessionFinalization) error {
	select {
	case <-finalization.done:
		return finalization.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Manager) claimFinalization(session *Session) (bool, *sessionFinalization) {
	if session == nil {
		return false, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if finalization, ok := m.finalizing[session.ID]; ok {
		return false, finalization
	}

	current, ok := m.sessions[session.ID]
	if !ok || current != session {
		return false, nil
	}

	finalization := &sessionFinalization{done: make(chan struct{})}
	m.finalizing[session.ID] = finalization
	return true, finalization
}

// WaitForFinalizations blocks until all in-flight finalization routines finish.
func (m *Manager) WaitForFinalizations(ctx context.Context) error {
	if m == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("session: wait for finalizations context is required")
	}

	for {
		m.mu.RLock()
		pending := make([]<-chan struct{}, 0, len(m.finalizing))
		for _, finalization := range m.finalizing {
			if finalization != nil {
				pending = append(pending, finalization.done)
			}
		}
		m.mu.RUnlock()

		if len(pending) == 0 {
			return nil
		}

		for _, done := range pending {
			select {
			case <-done:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
