package session

import (
	"context"
	"errors"
)

// CreateAccepted validates and durably registers a starting session before
// launching provider startup under the manager lifecycle context.
func (m *Manager) CreateAccepted(ctx context.Context, opts CreateOpts) (*Info, error) {
	if ctx == nil {
		return nil, errors.New("session: create accepted context is required")
	}
	if err := m.checkNewWorkAdmission(ctx); err != nil {
		return nil, err
	}
	spec, err := m.prepareCreateStart(ctx, opts)
	if err != nil {
		return nil, err
	}
	accepted, err := m.acceptSessionStart(ctx, m.lifecycleCtx, &spec)
	if err != nil {
		return nil, err
	}
	accepted.async = true
	accepted.persistFailure = true
	info := accepted.session.Info()
	go func() {
		if runErr := m.runAcceptedSessionStart(accepted); runErr != nil {
			m.sessionLogger(accepted.session).Warn(
				"session.start.background_failed",
				"error", runErr,
			)
		}
	}()
	return info, nil
}
