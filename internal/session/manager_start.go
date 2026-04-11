package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/workref"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type sessionStartSpec struct {
	sessionID              string
	sessionName            string
	agentName              string
	workspace              workspacepkg.ResolvedWorkspace
	space                  string
	sessionType            SessionType
	postEvent              hookspkg.HookEvent
	startAction            string
	cleanupSessionDir      bool
	includePromptUpdatedAt bool
	createdAt              time.Time
	acpSessionID           string
	stopReason             store.StopReason
	stopDetail             string
}

func (m *Manager) prepareCreateStart(ctx context.Context, opts CreateOpts) (sessionStartSpec, error) {
	opts, err := m.dispatchSessionPreCreate(ctx, opts)
	if err != nil {
		return sessionStartSpec{}, err
	}

	resolvedWorkspace, err := m.resolveCreateWorkspace(ctx, opts)
	if err != nil {
		return sessionStartSpec{}, err
	}

	agentName, err := aghconfig.ResolveAgentName(opts.AgentName, resolvedWorkspace.Config)
	if err != nil {
		return sessionStartSpec{}, fmt.Errorf("session: resolve agent name: %w", err)
	}

	sessionID := strings.TrimSpace(m.newSessionID())
	if sessionID == "" {
		return sessionStartSpec{}, errors.New("session: session id generator returned empty id")
	}

	return sessionStartSpec{
		sessionID:         sessionID,
		sessionName:       strings.TrimSpace(opts.Name),
		agentName:         strings.TrimSpace(agentName),
		workspace:         resolvedWorkspace,
		space:             strings.TrimSpace(opts.Space),
		sessionType:       normalizeSessionType(opts.Type),
		postEvent:         hookspkg.HookSessionPostCreate,
		startAction:       "start",
		cleanupSessionDir: true,
	}, nil
}

func (m *Manager) prepareResumeStart(ctx context.Context, meta store.SessionMeta) (sessionStartSpec, error) {
	meta, err := m.dispatchSessionPreResume(ctx, meta)
	if err != nil {
		return sessionStartSpec{}, err
	}

	resolvedWorkspace, err := m.resolveResumeWorkspace(ctx, meta)
	if err != nil {
		return sessionStartSpec{}, err
	}

	return sessionStartSpec{
		sessionID:              meta.ID,
		sessionName:            meta.Name,
		agentName:              meta.AgentName,
		workspace:              resolvedWorkspace,
		space:                  strings.TrimSpace(meta.Space),
		sessionType:            normalizeSessionType(SessionType(meta.SessionType)),
		postEvent:              hookspkg.HookSessionPostResume,
		startAction:            "resume",
		includePromptUpdatedAt: true,
		createdAt:              meta.CreatedAt,
		acpSessionID:           derefString(meta.ACPSessionID),
		stopReason:             sessionMetaStopReason(meta),
		stopDetail:             strings.TrimSpace(meta.StopDetail),
	}, nil
}

func (m *Manager) startSession(ctx context.Context, spec sessionStartSpec) (_ *Session, err error) {
	agentDef, err := resolveWorkspaceAgent(spec.agentName, spec.workspace)
	if err != nil {
		return nil, fmt.Errorf("session: resolve workspace agent %q: %w", spec.agentName, err)
	}

	startupPrompt, err := m.startupPrompt(ctx, spec.startupSessionContext(m.now()), agentDef, spec.workspace)
	if err != nil {
		return nil, err
	}
	agentDef.Prompt = startupPrompt

	resolved, err := spec.workspace.Config.ResolveAgent(agentDef)
	if err != nil {
		return nil, fmt.Errorf("session: resolve agent %q: %w", spec.agentName, err)
	}

	startMCPServers, err := m.resolveStartMCPServers(ctx, spec.workspace, resolved.MCPServers)
	if err != nil {
		return nil, err
	}

	if err := m.reserve(spec.sessionID, m.effectiveMaxSessions(spec.workspace.Config)); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			m.releaseReservation(spec.sessionID)
		}
	}()

	sessionDir := filepath.Join(m.homePaths.SessionsDir, spec.sessionID)
	if spec.cleanupSessionDir {
		if err := os.MkdirAll(sessionDir, 0o755); err != nil {
			return nil, fmt.Errorf("session: create session directory %q: %w", sessionDir, err)
		}
	}

	metaPath := store.SessionMetaFile(sessionDir)
	dbPath := store.SessionDBFile(sessionDir)
	recorder, err := m.openStore(ctx, spec.sessionID, dbPath)
	if err != nil {
		return nil, fmt.Errorf("session: open session store %q: %w", dbPath, err)
	}

	var proc *AgentProcess
	defer func() {
		if err == nil {
			return
		}

		cleanupDir := ""
		if spec.cleanupSessionDir {
			cleanupDir = sessionDir
		}
		err = errors.Join(err, m.cleanupFailedStart(cleanupDir, recorder, proc))
	}()

	now := m.now()
	createdAt := spec.createdAt
	if createdAt.IsZero() {
		createdAt = now
	}

	session := &Session{
		ID:           spec.sessionID,
		Name:         spec.sessionName,
		AgentName:    resolved.Name,
		WorkspaceID:  spec.workspace.ID,
		Workspace:    spec.workspace.RootDir,
		Space:        spec.space,
		Type:         normalizeSessionType(spec.sessionType),
		State:        StateStarting,
		stopReason:   spec.stopReason,
		stopDetail:   spec.stopDetail,
		ACPSessionID: spec.acpSessionID,
		CreatedAt:    createdAt,
		UpdatedAt:    now,
		sessionDir:   sessionDir,
		metaPath:     metaPath,
		dbPath:       dbPath,
		recorder:     recorder,
	}

	startOpts := acp.StartOpts{
		AgentName:       resolved.Name,
		Command:         resolved.Command,
		Cwd:             spec.workspace.RootDir,
		AdditionalDirs:  append([]string(nil), spec.workspace.AdditionalDirs...),
		MCPServers:      startMCPServers,
		Permissions:     m.startPermissions(session.Type, resolved.Permissions),
		SystemPrompt:    resolved.Prompt,
		ResumeSessionID: spec.acpSessionID,
	}
	startOpts, err = m.dispatchAgentPreStart(ctx, session, resolved, startOpts)
	if err != nil {
		return nil, err
	}

	if err := m.writeMeta(session); err != nil {
		return nil, err
	}

	proc, err = m.driver.Start(ctx, startOpts)
	if err != nil {
		return nil, fmt.Errorf("session: %s agent for %q: %w", spec.startAction, spec.sessionID, err)
	}
	proc.configureRuntime(session.CurrentTurnSource)

	if err := m.activateAndWatch(ctx, session, proc, resolved, spec.postEvent); err != nil {
		return nil, err
	}

	return session, nil
}

func (s sessionStartSpec) startupSessionContext(updatedAt time.Time) hookspkg.SessionContext {
	ref := workref.NewRoot(s.workspace.ID, s.workspace.RootDir)
	ctx := hookspkg.SessionContext{
		SessionID:    strings.TrimSpace(s.sessionID),
		SessionName:  strings.TrimSpace(s.sessionName),
		SessionType:  string(normalizeSessionType(s.sessionType)),
		AgentName:    strings.TrimSpace(s.agentName),
		WorkspaceID:  ref.WorkspaceID,
		Workspace:    ref.Workspace,
		ACPSessionID: strings.TrimSpace(s.acpSessionID),
		State:        string(StateStarting),
		CreatedAt:    s.createdAt,
	}
	if s.includePromptUpdatedAt {
		ctx.UpdatedAt = updatedAt
	}
	return ctx
}
