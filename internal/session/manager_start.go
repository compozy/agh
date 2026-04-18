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
	environmentID          string
	environment            *store.SessionEnvironmentMeta
	sessionName            string
	agentName              string
	workspace              workspacepkg.ResolvedWorkspace
	channel                string
	sessionType            Type
	postEvent              hookspkg.HookEvent
	startAction            string
	cleanupSessionDir      bool
	includePromptUpdatedAt bool
	preserveStopReason     bool
	createdAt              time.Time
	acpSessionID           string
	stopReason             store.StopReason
	stopDetail             string
}

type sessionStartRuntime struct {
	agent      aghconfig.ResolvedAgent
	mcpServers []aghconfig.MCPServer
}

type sessionStartStorage struct {
	sessionDir string
	metaPath   string
	dbPath     string
	recorder   EventRecorder
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

	agentName, err := aghconfig.ResolveAgentName(opts.AgentName, resolvedWorkspace.Config.Defaults)
	if err != nil {
		return sessionStartSpec{}, fmt.Errorf("session: resolve agent name: %w", err)
	}

	sessionID := strings.TrimSpace(m.newSessionID())
	if sessionID == "" {
		return sessionStartSpec{}, errors.New("session: session id generator returned empty id")
	}
	environmentID := strings.TrimSpace(m.newEnvironmentID())
	if environmentID == "" {
		return sessionStartSpec{}, errors.New("session: environment id generator returned empty id")
	}

	return sessionStartSpec{
		sessionID:         sessionID,
		environmentID:     environmentID,
		sessionName:       strings.TrimSpace(opts.Name),
		agentName:         strings.TrimSpace(agentName),
		workspace:         resolvedWorkspace,
		channel:           strings.TrimSpace(opts.Channel),
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
		environmentID:          sessionEnvironmentID(meta.Environment),
		environment:            cloneSessionEnvironmentMeta(meta.Environment),
		sessionName:            meta.Name,
		agentName:              meta.AgentName,
		workspace:              resolvedWorkspace,
		channel:                strings.TrimSpace(meta.Channel),
		sessionType:            normalizeSessionType(Type(meta.SessionType)),
		postEvent:              hookspkg.HookSessionPostResume,
		startAction:            "resume",
		includePromptUpdatedAt: true,
		preserveStopReason:     sessionMetaStopReason(meta) == store.StopAgentCrashed,
		createdAt:              meta.CreatedAt,
		acpSessionID:           derefString(meta.ACPSessionID),
		stopReason:             sessionMetaStopReason(meta),
		stopDetail:             strings.TrimSpace(meta.StopDetail),
	}, nil
}

func (m *Manager) startSession(ctx context.Context, spec *sessionStartSpec) (_ *Session, err error) {
	now := m.now()

	runtime, err := m.prepareSessionStartRuntime(ctx, spec, now)
	if err != nil {
		return nil, err
	}

	if err := m.reserve(spec.sessionID, m.effectiveMaxSessions(&spec.workspace.Config)); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			m.releaseReservation(spec.sessionID)
		}
	}()

	storage, err := m.openSessionStartStorage(ctx, spec)
	if err != nil {
		return nil, err
	}

	var proc *AgentProcess
	defer func() {
		if err == nil {
			return
		}

		cleanupDir := ""
		if spec.cleanupSessionDir {
			cleanupDir = storage.sessionDir
		}
		err = errors.Join(err, m.cleanupFailedStart(cleanupDir, storage.recorder, proc))
	}()

	session := spec.newStartingSession(runtime.agent, storage, now)

	startOpts := m.sessionStartOpts(spec, session, runtime.agent, runtime.mcpServers)
	startOpts, err = m.prepareEnvironmentForStart(ctx, spec, session, startOpts)
	if err != nil {
		return nil, err
	}
	startOpts, err = m.dispatchAgentPreStart(ctx, session, runtime.agent, startOpts)
	if err != nil {
		return nil, err
	}

	if err := m.writeMeta(session); err != nil {
		return nil, err
	}

	transportStarted := time.Now()
	proc, err = m.driver.Start(ctx, startOpts)
	if err != nil {
		m.logEnvironmentTransport(session, environmentEventTransportError, err, time.Since(transportStarted))
		return nil, fmt.Errorf("session: %s agent for %q: %w", spec.startAction, spec.sessionID, err)
	}
	m.logEnvironmentTransport(session, environmentEventTransportConnect, nil, time.Since(transportStarted))
	proc.configureRuntime(session.CurrentTurnSource)

	if err := m.activateAndWatch(
		ctx,
		session,
		proc,
		runtime.agent,
		spec.postEvent,
		spec.preserveStopReason,
	); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *sessionStartSpec) startupSessionContext(updatedAt time.Time) hookspkg.SessionContext {
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

func (s *sessionStartSpec) startupPromptContext() StartupPromptContext {
	ref := workref.NewRoot(s.workspace.ID, s.workspace.RootDir)
	return StartupPromptContext{
		SessionID:   strings.TrimSpace(s.sessionID),
		SessionName: strings.TrimSpace(s.sessionName),
		AgentName:   strings.TrimSpace(s.agentName),
		WorkspaceID: ref.WorkspaceID,
		Workspace:   ref.Workspace,
		Channel:     strings.TrimSpace(s.channel),
		SessionType: normalizeSessionType(s.sessionType),
	}
}

func (m *Manager) prepareSessionStartRuntime(
	ctx context.Context,
	spec *sessionStartSpec,
	updatedAt time.Time,
) (sessionStartRuntime, error) {
	agentDef, err := m.resolveWorkspaceAgent(spec.agentName, &spec.workspace)
	if err != nil {
		return sessionStartRuntime{}, fmt.Errorf("session: resolve workspace agent %q: %w", spec.agentName, err)
	}

	startupPrompt, err := m.startupPrompt(ctx, spec.startupSessionContext(updatedAt), agentDef, &spec.workspace)
	if err != nil {
		return sessionStartRuntime{}, err
	}
	if m.startupOverlay != nil {
		startupPrompt, err = m.startupOverlay.Apply(ctx, spec.startupPromptContext(), startupPrompt)
		if err != nil {
			return sessionStartRuntime{}, fmt.Errorf("session: apply startup prompt overlay: %w", err)
		}
	}
	agentDef.Prompt = startupPrompt

	resolved, err := spec.workspace.Config.ResolveAgent(agentDef)
	if err != nil {
		return sessionStartRuntime{}, fmt.Errorf("session: resolve agent %q: %w", spec.agentName, err)
	}

	startMCPServers, err := m.resolveStartMCPServers(ctx, &spec.workspace, resolved.MCPServers)
	if err != nil {
		return sessionStartRuntime{}, err
	}

	return sessionStartRuntime{
		agent:      resolved,
		mcpServers: startMCPServers,
	}, nil
}

func (m *Manager) openSessionStartStorage(
	ctx context.Context,
	spec *sessionStartSpec,
) (sessionStartStorage, error) {
	sessionDir := filepath.Join(m.homePaths.SessionsDir, spec.sessionID)
	if spec.cleanupSessionDir {
		if err := os.MkdirAll(sessionDir, 0o755); err != nil {
			return sessionStartStorage{}, fmt.Errorf("session: create session directory %q: %w", sessionDir, err)
		}
	}

	dbPath := store.SessionDBFile(sessionDir)
	recorder, err := m.openStore(ctx, spec.sessionID, dbPath)
	if err != nil {
		return sessionStartStorage{}, fmt.Errorf("session: open session store %q: %w", dbPath, err)
	}

	return sessionStartStorage{
		sessionDir: sessionDir,
		metaPath:   store.SessionMetaFile(sessionDir),
		dbPath:     dbPath,
		recorder:   recorder,
	}, nil
}

func (s *sessionStartSpec) newStartingSession(
	resolved aghconfig.ResolvedAgent,
	storage sessionStartStorage,
	now time.Time,
) *Session {
	createdAt := s.createdAt
	if createdAt.IsZero() {
		createdAt = now
	}

	return &Session{
		ID:                       s.sessionID,
		Name:                     s.sessionName,
		AgentName:                resolved.Name,
		WorkspaceID:              s.workspace.ID,
		Workspace:                s.workspace.RootDir,
		Channel:                  s.channel,
		Type:                     normalizeSessionType(s.sessionType),
		State:                    StateStarting,
		stopReason:               s.stopReason,
		stopDetail:               s.stopDetail,
		ACPSessionID:             s.acpSessionID,
		Environment:              cloneSessionEnvironmentMeta(s.environment),
		CreatedAt:                createdAt,
		UpdatedAt:                now,
		sessionDir:               storage.sessionDir,
		metaPath:                 storage.metaPath,
		dbPath:                   storage.dbPath,
		recorder:                 storage.recorder,
		environmentDestroyOnStop: s.workspace.Environment.DestroyOnStop,
	}
}

func (m *Manager) sessionStartOpts(
	s *sessionStartSpec,
	session *Session,
	resolved aghconfig.ResolvedAgent,
	mcpServers []aghconfig.MCPServer,
) acp.StartOpts {
	return acp.StartOpts{
		AgentName:       resolved.Name,
		Command:         resolved.Command,
		Cwd:             s.workspace.RootDir,
		AdditionalDirs:  append([]string(nil), s.workspace.AdditionalDirs...),
		Env:             sessionStartEnv(os.Environ(), session),
		MCPServers:      mcpServers,
		Permissions:     m.startPermissions(session.Type, resolved.Permissions),
		SystemPrompt:    resolved.Prompt,
		ResumeSessionID: s.acpSessionID,
	}
}

func sessionStartEnv(base []string, session *Session) []string {
	env := append([]string(nil), base...)
	if len(env) == 0 {
		env = os.Environ()
	}
	if session == nil {
		return env
	}

	env = setSessionStartEnvValue(env, "AGH_SESSION_ID", strings.TrimSpace(session.ID))
	env = unsetSessionStartEnvKeys(env, "AGH_SESSION_CHANNEL", "AGH_PEER_ID")

	channel := strings.TrimSpace(session.Channel)
	if channel == "" {
		return env
	}

	env = setSessionStartEnvValue(env, "AGH_SESSION_CHANNEL", channel)
	env = setSessionStartEnvValue(env, "AGH_PEER_ID", networkPeerID(session.AgentName, session.ID))
	return env
}

func setSessionStartEnvValue(env []string, key string, value string) []string {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return env
	}
	entry := trimmedKey + "=" + value
	for i, current := range env {
		existingKey, _, ok := strings.Cut(current, "=")
		if ok && existingKey == trimmedKey {
			env[i] = entry
			return env
		}
	}
	return append(env, entry)
}

func unsetSessionStartEnvKeys(env []string, keys ...string) []string {
	if len(env) == 0 || len(keys) == 0 {
		return env
	}

	blocked := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey != "" {
			blocked[trimmedKey] = struct{}{}
		}
	}
	if len(blocked) == 0 {
		return env
	}

	filtered := make([]string, 0, len(env))
	for _, current := range env {
		existingKey, _, ok := strings.Cut(current, "=")
		if ok {
			if _, blockedKey := blocked[existingKey]; blockedKey {
				continue
			}
		}
		filtered = append(filtered, current)
	}
	return filtered
}
