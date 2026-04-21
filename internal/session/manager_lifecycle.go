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
	"github.com/pedronauck/agh/internal/environment"
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

	return m.startSession(ctx, &spec)
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

	session, err := m.startSession(ctx, &spec)
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

	fallbackSession, fallbackErr := m.startSession(ctx, &fallbackSpec)
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
			session.setStopCause(CauseCompleted)
		default:
			session.setStopCause(CauseProcessExited)
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
	owned, err := m.claimOrWaitFinalization(ctx, session)
	if err != nil || !owned {
		return err
	}

	defer m.finishFinalization(session.ID)

	var errs []error
	errs = appendLifecycleErr(errs, m.beginStoppingSession(session))
	errs = appendLifecycleErr(errs, m.persistStopClassification(session, waitErr))
	errs = appendLifecycleErr(errs, m.recordProcessExitEvent(ctx, session, waitErr))
	errs = appendLifecycleErr(errs, m.recordSessionStoppedEvent(ctx, session, waitErr))

	m.dispatchAgentStopped(ctx, session, session.processHandle(), waitErr)

	m.logEnvironmentTransport(session, environmentEventTransportDisconnect, nil, 0)
	errs = appendLifecycleErr(errs, m.finalizeEnvironment(ctx, session, environmentSyncReasonForStop(session)))

	errs = appendLifecycleErr(errs, m.closeSessionRecorder(session))
	errs = appendLifecycleErr(errs, m.markSessionStopped(session))
	errs = appendLifecycleErr(errs, m.leaveSessionNetwork(ctx, session))
	m.failQueuedSyntheticPrompts(session.ID, ErrSessionNotActive)

	m.removeActive(session.ID)
	m.dispatchSessionPostStop(ctx, session)
	if m.notifier != nil {
		m.notifier.OnSessionStopped(ctx, session)
	}

	return errors.Join(errs...)
}

func (m *Manager) resolveStartMCPServers(
	ctx context.Context,
	resolvedWorkspace *workspacepkg.ResolvedWorkspace,
	base []aghconfig.MCPServer,
) ([]aghconfig.MCPServer, error) {
	switch {
	case m.skillRegistry == nil && m.mcpResolver == nil:
		return append([]aghconfig.MCPServer(nil), base...), nil
	case m.skillRegistry == nil || m.mcpResolver == nil:
		return nil, errors.New("session: skill registry and MCP resolver must be configured together")
	}

	activeSkills, err := m.skillRegistry.ForWorkspace(ctx, resolvedWorkspace)
	if err != nil {
		workspaceID := ""
		if resolvedWorkspace != nil {
			workspaceID = resolvedWorkspace.ID
		}
		return nil, fmt.Errorf("session: resolve active skills for workspace %q: %w", workspaceID, err)
	}

	return aghconfig.MergeMCPServers(base, m.mcpResolver.Resolve(activeSkills)), nil
}

func (m *Manager) claimOrWaitFinalization(ctx context.Context, session *Session) (bool, error) {
	owned, done := m.claimFinalization(session)
	if owned || done == nil {
		return owned, nil
	}

	select {
	case <-done:
		return false, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

func appendLifecycleErr(errs []error, err error) []error {
	if err == nil {
		return errs
	}
	return append(errs, err)
}

func (m *Manager) beginStoppingSession(session *Session) error {
	if session.Info().State != StateActive {
		return nil
	}
	if err := session.beginStopping(m.now()); err != nil {
		return err
	}
	return m.writeMeta(session)
}

func (m *Manager) persistStopClassification(session *Session, waitErr error) error {
	stopCause, stopDetailHint := session.stopCauseDetail()
	stopReason, stopDetail := classifyStopReason(stopCause, waitErr, stopDetailHint)
	session.setStopClassification(stopReason, stopDetail)
	session.markExited(m.now())
	return m.writeMeta(session)
}

func environmentSyncReasonForStop(session *Session) environment.SyncReason {
	if session == nil {
		return environment.SyncReasonStop
	}
	info := session.Info()
	if info != nil && info.StopReason == store.StopAgentCrashed {
		return environment.SyncReasonCrash
	}
	return environment.SyncReasonStop
}

func (m *Manager) recordProcessExitEvent(ctx context.Context, session *Session, waitErr error) error {
	if waitErr == nil {
		return nil
	}

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
		return err
	}
	m.notifyAgentEvent(ctx, session, normalized)
	return nil
}

func (m *Manager) recordSessionStoppedEvent(ctx context.Context, session *Session, waitErr error) error {
	stopReason := store.StopReason("")
	if info := session.Info(); info != nil {
		stopReason = info.StopReason
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
		return err
	}
	m.notifyAgentEvent(ctx, session, normalizedStop)
	return nil
}

func (m *Manager) closeSessionRecorder(session *Session) error {
	recorder := session.recorderHandle()
	if recorder == nil {
		return nil
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), defaultLifecycleTimeout)
	defer cancel()
	err := recorder.Close(closeCtx)
	session.setRecorder(nil)
	return err
}

func (m *Manager) markSessionStopped(session *Session) error {
	now := m.now()
	session.clearProcess(now)
	if err := session.markStopped(now); err != nil {
		return err
	}
	return m.writeMeta(session)
}

func (m *Manager) leaveSessionNetwork(ctx context.Context, session *Session) error {
	if err := m.leaveNetworkPeer(ctx, session); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			m.sessionLogger(session).Warn("session: leave network channel canceled", "error", err)
			return nil
		}
		return fmt.Errorf("session: leave network channel for %q: %w", session.ID, err)
	}
	return nil
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
