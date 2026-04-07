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
)

// Create resolves an agent definition, opens the session store, and starts a new runtime session.
func (m *Manager) Create(ctx context.Context, opts CreateOpts) (_ *Session, err error) {
	if ctx == nil {
		return nil, errors.New("session: create context is required")
	}

	resolvedWorkspace, err := m.resolveCreateWorkspace(ctx, opts)
	if err != nil {
		return nil, err
	}

	agentName, err := aghconfig.ResolveAgentName(opts.AgentName, resolvedWorkspace.Config)
	if err != nil {
		return nil, fmt.Errorf("session: resolve agent name: %w", err)
	}

	agentDef, err := resolveWorkspaceAgent(agentName, resolvedWorkspace)
	if err != nil {
		return nil, fmt.Errorf("session: resolve workspace agent %q: %w", agentName, err)
	}
	startupPrompt, err := m.startupPrompt(ctx, agentName, agentDef, resolvedWorkspace)
	if err != nil {
		return nil, err
	}
	agentDef.Prompt = startupPrompt

	resolved, err := resolvedWorkspace.Config.ResolveAgent(agentDef)
	if err != nil {
		return nil, fmt.Errorf("session: resolve agent %q: %w", agentName, err)
	}

	sessionID := strings.TrimSpace(m.newSessionID())
	if sessionID == "" {
		return nil, errors.New("session: session id generator returned empty id")
	}

	if err := m.reserve(sessionID, m.effectiveMaxSessions(resolvedWorkspace.Config)); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			m.releaseReservation(sessionID)
		}
	}()

	sessionDir := filepath.Join(m.homePaths.SessionsDir, sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, fmt.Errorf("session: create session directory %q: %w", sessionDir, err)
	}

	dbPath := store.SessionDBFile(sessionDir)
	recorder, err := m.openStore(ctx, sessionID, dbPath)
	if err != nil {
		return nil, fmt.Errorf("session: open session store %q: %w", dbPath, err)
	}

	var proc *AgentProcess
	defer func() {
		if err == nil {
			return
		}
		err = errors.Join(err, m.cleanupFailedStart(sessionDir, recorder, proc))
	}()

	now := m.now()
	session := &Session{
		ID:          sessionID,
		Name:        strings.TrimSpace(opts.Name),
		AgentName:   resolved.Name,
		WorkspaceID: resolvedWorkspace.ID,
		Workspace:   resolvedWorkspace.RootDir,
		Type:        normalizeSessionType(opts.Type),
		State:       StateStarting,
		CreatedAt:   now,
		UpdatedAt:   now,
		sessionDir:  sessionDir,
		metaPath:    store.SessionMetaFile(sessionDir),
		dbPath:      dbPath,
		recorder:    recorder,
	}

	if err := m.writeMeta(session); err != nil {
		return nil, err
	}

	proc, err = m.driver.Start(ctx, acp.StartOpts{
		AgentName:      resolved.Name,
		Command:        resolved.Command,
		Cwd:            resolvedWorkspace.RootDir,
		AdditionalDirs: append([]string(nil), resolvedWorkspace.AdditionalDirs...),
		MCPServers:     append([]aghconfig.MCPServer(nil), resolved.MCPServers...),
		Permissions:    m.startPermissions(session.Type, resolved.Permissions),
		SystemPrompt:   resolved.Prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("session: start agent for %q: %w", sessionID, err)
	}

	session.updateFromProcess(proc, m.now())
	if err := session.activate(m.now()); err != nil {
		return nil, err
	}
	if err := m.writeMeta(session); err != nil {
		return nil, err
	}
	if err := m.activate(session); err != nil {
		return nil, err
	}

	m.watchProcess(session)
	if m.notifier != nil {
		m.notifier.OnSessionCreated(ctx, session)
	}

	return session, nil
}

// Stop stops an active session and persists the stopped state to disk.
func (m *Manager) Stop(ctx context.Context, id string) error {
	if ctx == nil {
		return errors.New("session: stop context is required")
	}

	session, err := m.lookup(id)
	if err != nil {
		return err
	}

	writeMeta, promptSetupDone, err := session.prepareStop(m.now())
	if err != nil {
		return err
	}
	if writeMeta {
		if err := m.writeMeta(session); err != nil {
			return err
		}
	}
	if err := waitForPromptSetup(ctx, session, promptSetupDone); err != nil {
		return err
	}

	state := session.Info().State
	if state == StateStopped {
		return nil
	}

	proc := session.processHandle()
	if proc == nil {
		return m.finalizeStopped(ctx, session, nil)
	}

	stopErr := m.driver.Stop(ctx, proc)
	if !isProcessDone(proc) {
		return stopErr
	}

	return errors.Join(stopErr, m.finalizeStopped(ctx, session, nil))
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

	sessionDir := filepath.Join(m.homePaths.SessionsDir, target)
	metaPath := store.SessionMetaFile(sessionDir)
	meta, err := store.ReadSessionMeta(metaPath)
	if err != nil {
		return nil, fmt.Errorf("session: read session meta %q: %w", metaPath, err)
	}

	resolvedWorkspace, err := m.resolveResumeWorkspace(ctx, meta)
	if err != nil {
		return nil, err
	}

	agentDef, err := resolveWorkspaceAgent(meta.AgentName, resolvedWorkspace)
	if err != nil {
		return nil, fmt.Errorf("session: resolve workspace agent %q: %w", meta.AgentName, err)
	}
	startupPrompt, err := m.startupPrompt(ctx, meta.AgentName, agentDef, resolvedWorkspace)
	if err != nil {
		return nil, err
	}
	agentDef.Prompt = startupPrompt

	resolved, err := resolvedWorkspace.Config.ResolveAgent(agentDef)
	if err != nil {
		return nil, fmt.Errorf("session: resolve agent %q: %w", meta.AgentName, err)
	}

	if err := m.reserve(meta.ID, m.effectiveMaxSessions(resolvedWorkspace.Config)); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			m.releaseReservation(meta.ID)
		}
	}()

	dbPath := store.SessionDBFile(sessionDir)
	recorder, err := m.openStore(ctx, meta.ID, dbPath)
	if err != nil {
		return nil, fmt.Errorf("session: open session store %q: %w", dbPath, err)
	}

	var proc *AgentProcess
	defer func() {
		if err == nil {
			return
		}
		err = errors.Join(err, m.cleanupFailedStart("", recorder, proc))
	}()

	createdAt := meta.CreatedAt
	if createdAt.IsZero() {
		createdAt = m.now()
	}
	session := &Session{
		ID:           meta.ID,
		Name:         meta.Name,
		AgentName:    meta.AgentName,
		WorkspaceID:  strings.TrimSpace(meta.WorkspaceID),
		Workspace:    resolvedWorkspace.RootDir,
		Type:         normalizeSessionType(SessionType(meta.SessionType)),
		State:        StateStarting,
		ACPSessionID: derefString(meta.ACPSessionID),
		CreatedAt:    createdAt,
		UpdatedAt:    m.now(),
		sessionDir:   sessionDir,
		metaPath:     metaPath,
		dbPath:       dbPath,
		recorder:     recorder,
	}

	if err := m.writeMeta(session); err != nil {
		return nil, err
	}

	proc, err = m.driver.Start(ctx, acp.StartOpts{
		AgentName:       resolved.Name,
		Command:         resolved.Command,
		Cwd:             resolvedWorkspace.RootDir,
		AdditionalDirs:  append([]string(nil), resolvedWorkspace.AdditionalDirs...),
		MCPServers:      append([]aghconfig.MCPServer(nil), resolved.MCPServers...),
		Permissions:     m.startPermissions(session.Type, resolved.Permissions),
		SystemPrompt:    resolved.Prompt,
		ResumeSessionID: derefString(meta.ACPSessionID),
	})
	if err != nil {
		return nil, fmt.Errorf("session: resume agent for %q: %w", meta.ID, err)
	}

	session.updateFromProcess(proc, m.now())
	if err := session.activate(m.now()); err != nil {
		return nil, err
	}
	if err := m.writeMeta(session); err != nil {
		return nil, err
	}
	if err := m.activate(session); err != nil {
		return nil, err
	}

	m.watchProcess(session)
	if m.notifier != nil {
		m.notifier.OnSessionCreated(ctx, session)
	}

	return session, nil
}

func (m *Manager) watchProcess(session *Session) {
	proc := session.processHandle()
	if proc == nil {
		return
	}

	go func() {
		waitErr := proc.Wait()
		if err := m.handleProcessExit(session, waitErr); err != nil {
			m.sessionLogger(session).Warn("session: process exit handling failed", "error", err)
		}
	}()
}

func (m *Manager) handleProcessExit(session *Session, waitErr error) error {
	if session == nil {
		return nil
	}

	state := session.Info().State
	if state != StateActive && state != StateStopping {
		return nil
	}

	return m.finalizeStopped(context.Background(), session, waitErr)
}

func (m *Manager) finalizeStopped(ctx context.Context, session *Session, waitErr error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if session == nil {
		return nil
	}
	if !m.claimFinalization(session) {
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

	if waitErr != nil {
		event := acp.AgentEvent{
			Type:      acp.EventTypeError,
			TurnID:    newID("turn"),
			Timestamp: m.now(),
			Error:     waitErr.Error(),
			Text:      session.processHandle().Stderr(),
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
		Type:      EventTypeSessionStopped,
		TurnID:    newID("turn"),
		Timestamp: m.now(),
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

	if recorder := session.recorderHandle(); recorder != nil {
		closeCtx, cancel := context.WithTimeout(context.Background(), defaultLifecycleTimeout)
		if err := recorder.Close(closeCtx); err != nil {
			errs = append(errs, err)
		}
		cancel()
		session.setRecorder(nil)
	}

	session.clearProcess(m.now())
	if err := session.markStopped(m.now()); err != nil {
		errs = append(errs, err)
	} else if err := m.writeMeta(session); err != nil {
		errs = append(errs, err)
	}

	m.remove(session.ID)
	if m.notifier != nil {
		m.notifier.OnSessionStopped(ctx, session)
	}

	return errors.Join(errs...)
}

func (m *Manager) cleanupFailedStart(sessionDir string, recorder EventRecorder, proc *AgentProcess) error {
	var errs []error
	if proc != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), defaultLifecycleTimeout)
		if err := m.driver.Stop(stopCtx, proc); err != nil {
			errs = append(errs, err)
		}
		cancel()
	}
	if recorder != nil {
		closeCtx, cancel := context.WithTimeout(context.Background(), defaultLifecycleTimeout)
		if err := recorder.Close(closeCtx); err != nil {
			errs = append(errs, err)
		}
		cancel()
	}
	if strings.TrimSpace(sessionDir) != "" {
		if err := os.RemoveAll(sessionDir); err != nil {
			errs = append(errs, fmt.Errorf("session: remove failed session directory %q: %w", sessionDir, err))
		}
	}
	return errors.Join(errs...)
}
