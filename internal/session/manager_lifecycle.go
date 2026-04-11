package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

// Create resolves an agent definition, opens the session store, and starts a new runtime session.
func (m *Manager) Create(ctx context.Context, opts CreateOpts) (_ *Session, err error) {
	if ctx == nil {
		return nil, errors.New("session: create context is required")
	}

	spec, err := m.prepareCreateStart(ctx, opts)
	if err != nil {
		return nil, err
	}

	return m.startSession(ctx, spec)
}

// Stop stops an active session and persists the stopped state to disk.
func (m *Manager) Stop(ctx context.Context, id string) error {
	return m.StopWithCause(ctx, id, CauseUserRequested, "")
}

// Resume restarts a stopped session from its persisted metadata and event history.
func (m *Manager) Resume(ctx context.Context, id string) (_ *Session, err error) {
	if ctx == nil {
		return nil, errors.New("session: resume context is required")
	}

	target := strings.TrimSpace(id)
	if target == "" {
		return nil, errors.New("session: session id is required")
	}

	if session, ok := m.Get(target); ok {
		return session, nil
	}

	meta, err := m.readMeta(target)
	if err != nil {
		return nil, err
	}
	if validationErrs := m.validateInfrastructure(ctx, meta); len(validationErrs) > 0 {
		m.logResumeValidationFailures(meta, validationErrs)
		return nil, fmt.Errorf(
			"session: validate resume infrastructure for %q: %w",
			target,
			errors.Join(validationErrs...),
		)
	}

	spec, err := m.prepareResumeStart(ctx, meta)
	if err != nil {
		return nil, err
	}

	session, err := m.startSession(ctx, spec)
	if err == nil {
		return session, nil
	}

	metaPath := store.SessionMetaFile(filepath.Join(m.homePaths.SessionsDir, target))
	clearACP := acp.IsLoadSessionResourceMissing(err)
	restoredMeta, restoreErr := m.restoreFailedResumeStart(metaPath, meta, clearACP)
	if restoreErr != nil {
		return nil, errors.Join(err, restoreErr)
	}
	if !clearACP {
		return nil, err
	}

	m.resumeLogger(meta).Warn(
		"session.resume.load_session_missing_fallback",
		"error", err,
	)

	fallbackSpec, fallbackSpecErr := m.prepareResumeStart(ctx, restoredMeta)
	if fallbackSpecErr != nil {
		return nil, errors.Join(err, fallbackSpecErr)
	}

	fallbackSession, fallbackErr := m.startSession(ctx, fallbackSpec)
	if fallbackErr != nil {
		return nil, errors.Join(err, fallbackErr)
	}
	return fallbackSession, nil
}

func (m *Manager) watchProcess(ctx context.Context, session *Session) {
	proc := session.processHandle()
	if proc == nil {
		return
	}

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-proc.Done():
		}
		waitErr := proc.Wait()
		if err := m.handleProcessExit(ctx, session, waitErr); err != nil {
			m.sessionLogger(session).Warn("session: process exit handling failed", "error", err)
		}
	}()
}

func (m *Manager) handleProcessExit(ctx context.Context, session *Session, waitErr error) error {
	if session == nil {
		return nil
	}

	state := session.Info().State
	if state != StateActive && state != StateStopping {
		return nil
	}

	if !session.stopWasRequested() {
		switch waitErr {
		case nil:
			session.setStopCause(CauseCompleted, "")
		default:
			session.setStopCause(CauseProcessExited, "")
		}
	}

	return m.finalizeStopped(ctx, session, waitErr)
}

func (m *Manager) finalizeStopped(ctx context.Context, session *Session, waitErr error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if session == nil {
		return nil
	}
	owned, done := m.claimFinalization(session)
	if !owned {
		if done == nil {
			return nil
		}
		select {
		case <-done:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if done == nil {
		return nil
	}

	var errs []error
	state := session.Info().State
	if state == StateActive {
		if err := session.beginStopping(m.now()); err != nil {
			errs = append(errs, err)
		} else if err := m.writeMeta(session); err != nil {
			errs = append(errs, err)
		}
	}

	stopCause, stopDetailHint := session.stopCauseDetail()
	stopReason, stopDetail := classifyStopReason(stopCause, waitErr, stopDetailHint)
	session.setStopClassification(stopReason, stopDetail)
	if err := m.writeMeta(session); err != nil {
		errs = append(errs, err)
	}

	if waitErr != nil {
		m.dispatchAgentCrashed(ctx, session, session.processHandle(), waitErr)

		stderr := ""
		if proc := session.processHandle(); proc != nil {
			stderr = proc.Stderr()
		}
		event := acp.AgentEvent{
			Type:      acp.EventTypeError,
			TurnID:    newID("turn"),
			Timestamp: m.now(),
			Error:     waitErr.Error(),
			Text:      stderr,
		}
		normalized := m.normalizeEvent(session, event.TurnID, event)
		if err := m.recordEvent(ctx, session, normalized); err != nil {
			errs = append(errs, err)
		}
		if m.notifier != nil {
			m.notifier.OnAgentEvent(ctx, session.ID, normalized)
		}
	}

	stopEvent := acp.AgentEvent{
		Type:       EventTypeSessionStopped,
		TurnID:     newID("turn"),
		Timestamp:  m.now(),
		StopReason: string(stopReason),
	}
	if waitErr != nil {
		stopEvent.Error = waitErr.Error()
		if proc := session.processHandle(); proc != nil {
			stopEvent.Text = proc.Stderr()
		}
	}
	normalizedStop := m.normalizeEvent(session, stopEvent.TurnID, stopEvent)
	if err := m.recordEvent(ctx, session, normalizedStop); err != nil {
		errs = append(errs, err)
	}
	if m.notifier != nil {
		m.notifier.OnAgentEvent(ctx, session.ID, normalizedStop)
	}

	m.dispatchAgentStopped(ctx, session, session.processHandle(), waitErr)

	if recorder := session.recorderHandle(); recorder != nil {
		func() {
			closeCtx, cancel := context.WithTimeout(context.Background(), defaultLifecycleTimeout)
			defer cancel()
			if err := recorder.Close(closeCtx); err != nil {
				errs = append(errs, err)
			}
		}()
		session.setRecorder(nil)
	}

	session.clearProcess(m.now())
	if err := session.markStopped(m.now()); err != nil {
		errs = append(errs, err)
	} else if err := m.writeMeta(session); err != nil {
		errs = append(errs, err)
	}

	m.remove(session.ID)
	m.dispatchSessionPostStop(ctx, session)
	if m.notifier != nil {
		m.notifier.OnSessionStopped(ctx, session)
	}

	return errors.Join(errs...)
}

func (m *Manager) resolveStartMCPServers(ctx context.Context, resolvedWorkspace workspacepkg.ResolvedWorkspace, base []aghconfig.MCPServer) ([]aghconfig.MCPServer, error) {
	switch {
	case m.skillRegistry == nil && m.mcpResolver == nil:
		return append([]aghconfig.MCPServer(nil), base...), nil
	case m.skillRegistry == nil || m.mcpResolver == nil:
		return nil, errors.New("session: skill registry and MCP resolver must be configured together")
	}

	activeSkills, err := m.skillRegistry.ForWorkspace(ctx, resolvedWorkspace)
	if err != nil {
		return nil, fmt.Errorf("session: resolve active skills for workspace %q: %w", resolvedWorkspace.ID, err)
	}

	return aghconfig.MergeMCPServers(base, m.mcpResolver.Resolve(activeSkills)), nil
}

func (m *Manager) cleanupFailedStart(sessionDir string, recorder EventRecorder, proc *AgentProcess) error {
	var errs []error
	if proc != nil {
		func() {
			stopCtx, cancel := context.WithTimeout(context.Background(), defaultLifecycleTimeout)
			defer cancel()
			if err := m.driver.Stop(stopCtx, proc); err != nil {
				errs = append(errs, err)
			}
		}()
	}
	if recorder != nil {
		func() {
			closeCtx, cancel := context.WithTimeout(context.Background(), defaultLifecycleTimeout)
			defer cancel()
			if err := recorder.Close(closeCtx); err != nil {
				errs = append(errs, err)
			}
		}()
	}
	if strings.TrimSpace(sessionDir) != "" {
		if err := os.RemoveAll(sessionDir); err != nil {
			errs = append(errs, fmt.Errorf("session: remove failed session directory %q: %w", sessionDir, err))
		}
	}
	return errors.Join(errs...)
}
