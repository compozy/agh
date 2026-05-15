package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/procutil"
	"github.com/pedronauck/agh/internal/soul"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/workref"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type sessionStartSpec struct {
	sessionID              string
	sandboxID              string
	sandbox                *store.SessionSandboxMeta
	sessionName            string
	agentName              string
	provider               string
	model                  string
	reasoningEffort        string
	sandboxDisabled        bool
	workspace              workspacepkg.ResolvedWorkspace
	channel                string
	promptOverlay          string
	sessionType            Type
	lineage                *store.SessionLineage
	postEvent              hookspkg.HookEvent
	startAction            string
	cleanupSessionDir      bool
	includePromptUpdatedAt bool
	preserveStopReason     bool
	createdAt              time.Time
	acpSessionID           string
	stopReason             store.StopReason
	stopDetail             string
	failure                *store.SessionFailure
	soulSnapshotID         string
	soulDigest             string
	parentSoulDigest       string
	soulSnapshot           *soul.Snapshot
}

type sessionStartRuntime struct {
	agent               aghconfig.ResolvedAgent
	mcpServers          []aghconfig.MCPServer
	networkCapabilities []NetworkPeerCapability
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
	sandboxDisabled, err := applyCreateSandboxOverride(&resolvedWorkspace, opts)
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
	sandboxID := strings.TrimSpace(m.newSandboxID())
	if sandboxID == "" {
		return sessionStartSpec{}, errors.New("session: sandbox id generator returned empty id")
	}
	lineage, err := m.normalizeCreateLineage(ctx, sessionID, opts.Type, opts.Lineage)
	if err != nil {
		return sessionStartSpec{}, err
	}

	return sessionStartSpec{
		sessionID:         sessionID,
		sandboxID:         sandboxID,
		sessionName:       strings.TrimSpace(opts.Name),
		agentName:         strings.TrimSpace(agentName),
		provider:          strings.TrimSpace(opts.Provider),
		model:             strings.TrimSpace(opts.Model),
		reasoningEffort:   strings.TrimSpace(opts.ReasoningEffort),
		sandboxDisabled:   sandboxDisabled,
		workspace:         resolvedWorkspace,
		channel:           strings.TrimSpace(opts.Channel),
		promptOverlay:     strings.TrimSpace(opts.PromptOverlay),
		sessionType:       normalizeSessionType(opts.Type),
		lineage:           lineage,
		parentSoulDigest:  strings.TrimSpace(opts.ParentSoulDigest),
		postEvent:         hookspkg.HookSessionPostCreate,
		startAction:       "create",
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
		sandboxID:              sessionSandboxID(meta.Sandbox),
		sandbox:                cloneSessionSandboxMeta(meta.Sandbox),
		sandboxDisabled:        meta.Sandbox == nil,
		sessionName:            meta.Name,
		agentName:              meta.AgentName,
		provider:               strings.TrimSpace(meta.Provider),
		model:                  strings.TrimSpace(meta.Model),
		reasoningEffort:        strings.TrimSpace(meta.ReasoningEffort),
		workspace:              resolvedWorkspace,
		channel:                strings.TrimSpace(meta.Channel),
		sessionType:            normalizeSessionType(Type(meta.SessionType)),
		lineage:                store.NormalizeSessionLineage(meta.ID, meta.Lineage),
		postEvent:              hookspkg.HookSessionPostResume,
		startAction:            "resume",
		includePromptUpdatedAt: true,
		preserveStopReason:     sessionMetaStopReason(meta) == store.StopAgentCrashed,
		createdAt:              meta.CreatedAt,
		acpSessionID:           derefString(meta.ACPSessionID),
		stopReason:             sessionMetaStopReason(meta),
		stopDetail:             strings.TrimSpace(meta.StopDetail),
		failure:                store.CloneSessionFailure(meta.Failure),
		soulSnapshotID:         strings.TrimSpace(meta.SoulSnapshotID),
		soulDigest:             strings.TrimSpace(meta.SoulDigest),
		parentSoulDigest:       strings.TrimSpace(meta.ParentSoulDigest),
	}, nil
}

func (m *Manager) startSession(ctx context.Context, spec *sessionStartSpec) (_ *Session, err error) {
	now := m.now()

	runtime, err := m.prepareSessionStartRuntime(ctx, spec, now)
	if err != nil {
		spec.startLogger(m).Warn("session.start.runtime_prepare_failed", "phase", spec.startAction, "error", err)
		return nil, err
	}
	defer func() {
		if err != nil && m.hostedMCP != nil {
			m.hostedMCP.CancelLaunch(spec.sessionID)
		}
	}()

	if err := m.reserve(spec.sessionID); err != nil {
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
	defer cleanupProviderRedactionsOnStartError(session, &err)

	startOpts := m.sessionStartOpts(spec, session, runtime.agent, runtime.mcpServers)
	startOpts, err = m.prepareProviderForStart(ctx, session, runtime.agent, startOpts)
	if err != nil {
		return nil, m.failSessionStart(ctx, spec, session, "session provider startup failed", err)
	}
	startOpts, err = m.prepareSandboxForStart(ctx, spec, session, startOpts)
	if err != nil {
		return nil, m.failSessionStart(ctx, spec, session, "session sandbox startup failed", err)
	}
	startOpts, err = m.dispatchAgentPreStart(ctx, session, runtime.agent, startOpts)
	if err != nil {
		return nil, m.failSessionStart(ctx, spec, session, "session pre-start hook failed", err)
	}

	if err := m.writeMeta(session); err != nil {
		m.sessionLogger(session).Warn("session.start.meta_write_failed", "phase", spec.startAction, "error", err)
		return nil, err
	}

	proc, err = m.startAgentProcess(ctx, spec, session, startOpts)
	if err != nil {
		return nil, err
	}

	if err := m.activateAndWatch(
		ctx,
		session,
		proc,
		runtime.agent,
		runtime.networkCapabilities,
		spec.postEvent,
		spec.preserveStopReason,
	); err != nil {
		return nil, err
	}

	return session, nil
}

func cleanupProviderRedactionsOnStartError(session *Session, err *error) {
	if err != nil && *err != nil {
		session.clearProviderSecretRedactions()
	}
}

func (m *Manager) failSessionStart(
	ctx context.Context,
	spec *sessionStartSpec,
	session *Session,
	summary string,
	err error,
) error {
	startErr := acp.WrapFailure(store.FailureStartup, summary, err)
	spec.cleanupSessionDir = false
	return errors.Join(startErr, m.persistFailedStart(ctx, session, startErr))
}

func (m *Manager) startAgentProcess(
	ctx context.Context,
	spec *sessionStartSpec,
	session *Session,
	startOpts acp.StartOpts,
) (*AgentProcess, error) {
	transportStarted := time.Now()
	proc, err := m.driver.Start(ctx, startOpts)
	if err != nil {
		m.sessionLogger(session).Warn("session.start.driver_start_failed", "phase", spec.startAction, "error", err)
		m.logSandboxTransport(session, sandboxEventTransportError, err, time.Since(transportStarted))
		return proc, m.failSessionStart(
			ctx,
			spec,
			session,
			"agent runtime startup failed",
			fmt.Errorf("session: %s agent for %q: %w", spec.startAction, spec.sessionID, err),
		)
	}
	m.logSandboxTransport(session, sandboxEventTransportConnect, nil, time.Since(transportStarted))
	proc.configureRuntime(session.CurrentTurnSource)
	return proc, nil
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
		SessionSoulContext: hookSessionSoulContext(
			s.soulSnapshotID,
			s.soulDigest,
		),
		CreatedAt: s.createdAt,
	}
	if s.includePromptUpdatedAt {
		ctx.UpdatedAt = updatedAt
	}
	return ctx
}

func (s *sessionStartSpec) startupPromptContext(updatedAt time.Time) StartupPromptContext {
	ref := workref.NewRoot(s.workspace.ID, s.workspace.RootDir)
	return StartupPromptContext{
		SessionID:    strings.TrimSpace(s.sessionID),
		SessionName:  strings.TrimSpace(s.sessionName),
		AgentName:    strings.TrimSpace(s.agentName),
		Provider:     strings.TrimSpace(s.provider),
		WorkspaceID:  ref.WorkspaceID,
		Workspace:    ref.Workspace,
		Channel:      strings.TrimSpace(s.channel),
		SessionType:  normalizeSessionType(s.sessionType),
		SoulSnapshot: cloneSoulSnapshotPointer(s.soulSnapshot),
		CreatedAt:    s.createdAt,
		UpdatedAt:    updatedAt,
	}
}

func (m *Manager) prepareSessionStartRuntime(
	ctx context.Context,
	spec *sessionStartSpec,
	updatedAt time.Time,
) (sessionStartRuntime, error) {
	artifacts, err := m.resolveWorkspaceAgentArtifactsForSession(spec.agentName, spec.sessionType, &spec.workspace)
	if err != nil {
		return sessionStartRuntime{}, fmt.Errorf("session: resolve workspace agent %q: %w", spec.agentName, err)
	}
	agentDef := artifacts.Agent

	if err := m.prepareSessionStartSoul(ctx, spec, artifacts, updatedAt); err != nil {
		return sessionStartRuntime{}, err
	}

	startupCtx := spec.startupPromptContext(updatedAt)
	if strings.TrimSpace(startupCtx.AgentName) == "" {
		startupCtx.AgentName = strings.TrimSpace(agentDef.Name)
	}
	if strings.TrimSpace(startupCtx.Provider) == "" {
		startupCtx.Provider = strings.TrimSpace(agentDef.Provider)
	}
	startupPrompt, err := m.startupPrompt(
		ctx,
		spec.startupSessionContext(updatedAt),
		startupCtx,
		agentDef,
		&spec.workspace,
	)
	if err != nil {
		return sessionStartRuntime{}, err
	}
	if m.startupOverlay != nil {
		startupPrompt, err = m.startupOverlay.Apply(ctx, startupCtx, startupPrompt)
		if err != nil {
			return sessionStartRuntime{}, fmt.Errorf("session: apply startup prompt overlay: %w", err)
		}
	}
	agentDef.Prompt = startupPrompt
	if overlay := strings.TrimSpace(spec.promptOverlay); overlay != "" {
		if strings.TrimSpace(agentDef.Prompt) == "" {
			agentDef.Prompt = overlay
		} else {
			agentDef.Prompt = strings.TrimSpace(agentDef.Prompt) + "\n\n" + overlay
		}
	}

	resolved, err := spec.workspace.Config.ResolveSessionAgentWithRuntime(agentDef, spec.provider, spec.model)
	if err != nil {
		return sessionStartRuntime{}, fmt.Errorf("session: resolve session agent %q: %w", spec.agentName, err)
	}
	if err := spec.validateRuntimeOverrides(); err != nil {
		return sessionStartRuntime{}, err
	}

	startMCPServers, err := m.sessionMCPServers(ctx, spec, resolved)
	if err != nil {
		return sessionStartRuntime{}, err
	}

	return sessionStartRuntime{
		agent:               resolved,
		mcpServers:          startMCPServers,
		networkCapabilities: networkPeerCapabilities(agentDef.Capabilities),
	}, nil
}

func (s *sessionStartSpec) validateRuntimeOverrides() error {
	providerOverride := strings.TrimSpace(s.provider)
	modelOverride := strings.TrimSpace(s.model)
	reasoningEffort := strings.TrimSpace(s.reasoningEffort)
	if modelOverride != "" && providerOverride == "" {
		return fmt.Errorf("%w: provider is required when model is set", ErrInvalidRuntimeOverride)
	}
	if reasoningEffort == "" {
		return nil
	}
	if providerOverride == "" {
		return fmt.Errorf("%w: provider is required when reasoning_effort is set", ErrInvalidRuntimeOverride)
	}
	if err := ValidateReasoningEffort(reasoningEffort); err != nil {
		return err
	}
	return nil
}

func (m *Manager) sessionMCPServers(
	ctx context.Context,
	spec *sessionStartSpec,
	resolved aghconfig.ResolvedAgent,
) ([]aghconfig.MCPServer, error) {
	if !resolved.SessionMCP {
		return nil, nil
	}
	if m.hostedMCP == nil {
		return m.resolveStartMCPServers(ctx, &spec.workspace, resolved.Name, resolved.MCPServers)
	}
	hosted, err := m.hostedMCP.Launch(ctx, HostedMCPLaunchRequest{
		SessionID:   spec.sessionID,
		WorkspaceID: spec.workspace.ID,
		AgentName:   resolved.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("session: mint hosted MCP launch for %q: %w", spec.sessionID, err)
	}
	return []aghconfig.MCPServer{hosted}, nil
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
		ID:                   s.sessionID,
		Name:                 s.sessionName,
		AgentName:            resolved.Name,
		Provider:             strings.TrimSpace(resolved.Provider),
		Model:                strings.TrimSpace(resolved.Model),
		ReasoningEffort:      strings.TrimSpace(s.reasoningEffort),
		WorkspaceID:          s.workspace.ID,
		Workspace:            s.workspace.RootDir,
		Channel:              s.channel,
		Type:                 normalizeSessionType(s.sessionType),
		Lineage:              store.CloneSessionLineage(s.lineage),
		State:                StateStarting,
		stopReason:           s.stopReason,
		stopDetail:           s.stopDetail,
		failure:              store.CloneSessionFailure(s.failure),
		ACPSessionID:         s.acpSessionID,
		Sandbox:              cloneSessionSandboxMeta(s.sandbox),
		SoulSnapshotID:       s.soulSnapshotID,
		SoulDigest:           s.soulDigest,
		ParentSoulDigest:     s.parentSoulDigest,
		CreatedAt:            createdAt,
		UpdatedAt:            now,
		sessionDir:           storage.sessionDir,
		metaPath:             storage.metaPath,
		dbPath:               storage.dbPath,
		recorder:             storage.recorder,
		sandboxDestroyOnStop: !s.sandboxDisabled && s.workspace.Sandbox.DestroyOnStop,
	}
}

func (m *Manager) normalizeCreateLineage(
	ctx context.Context,
	sessionID string,
	sessionType Type,
	lineage *store.SessionLineage,
) (*store.SessionLineage, error) {
	normalizedType := normalizeSessionType(sessionType)
	normalized := store.NormalizeSessionLineage(sessionID, lineage)
	if err := store.ValidateSessionLineage(sessionID, normalized); err != nil {
		return nil, fmt.Errorf("session: validate session lineage: %w", err)
	}

	hasParent := strings.TrimSpace(normalized.ParentSessionID) != ""
	switch {
	case normalizedType == SessionTypeSpawned && !hasParent:
		return nil, errors.New("session: spawned session lineage requires a parent session id")
	case hasParent && normalizedType != SessionTypeSpawned:
		return nil, errors.New("session: only spawned sessions may have a parent session id")
	case normalizedType == SessionTypeCoordinator && hasParent:
		return nil, errors.New("session: coordinator sessions must be root sessions")
	}

	requiresTTL := normalizedType == SessionTypeSpawned || normalizedType == SessionTypeCoordinator
	if requiresTTL && normalized.TTLExpiresAt == nil {
		return nil, errors.New("session: spawned and coordinator sessions require a ttl deadline")
	}
	if normalized.TTLExpiresAt != nil {
		now := m.now()
		if !normalized.TTLExpiresAt.After(now) {
			return nil, errors.New("session: ttl deadline must be in the future")
		}
		if normalized.SpawnBudget.TTLSeconds <= 0 {
			ttlSeconds := int64(normalized.TTLExpiresAt.Sub(now).Seconds())
			if ttlSeconds <= 0 {
				ttlSeconds = 1
			}
			normalized.SpawnBudget.TTLSeconds = ttlSeconds
		}
	}
	if err := m.validateCreateLineageReferences(ctx, normalized); err != nil {
		return nil, err
	}

	return normalized, nil
}

func (m *Manager) validateCreateLineageReferences(ctx context.Context, lineage *store.SessionLineage) error {
	if lineage == nil || strings.TrimSpace(lineage.ParentSessionID) == "" {
		return nil
	}
	if _, err := m.Status(ctx, lineage.ParentSessionID); err != nil {
		return fmt.Errorf("session: validate parent lineage %q: %w", lineage.ParentSessionID, err)
	}
	rootID := strings.TrimSpace(lineage.RootSessionID)
	if rootID == "" || rootID == strings.TrimSpace(lineage.ParentSessionID) {
		return nil
	}
	if _, err := m.Status(ctx, rootID); err != nil {
		return fmt.Errorf("session: validate root lineage %q: %w", rootID, err)
	}
	return nil
}

func (s *sessionStartSpec) startLogger(m *Manager) *slog.Logger {
	logger := slog.Default()
	if m != nil && m.logger != nil {
		logger = m.logger
	}
	return logger.With(
		"session_id", strings.TrimSpace(s.sessionID),
		"agent_name", strings.TrimSpace(s.agentName),
		"provider", strings.TrimSpace(s.provider),
		"workspace_id", strings.TrimSpace(s.workspace.ID),
	)
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
		Env:             sessionStartEnvForProvider(os.Environ(), session, resolved.EnvPolicy),
		MCPServers:      mcpServers,
		Permissions:     m.startPermissions(session.Type, resolved.Permissions),
		SystemPrompt:    resolved.Prompt,
		PreferredModel:  preferredACPModel(resolved),
		ReasoningEffort: strings.TrimSpace(session.ReasoningEffort),
		ResumeSessionID: s.acpSessionID,
		ToolGateway:     newProviderNativeToolGateway(m, session),
	}
}

func preferredACPModel(resolved aghconfig.ResolvedAgent) string {
	if resolved.Harness != aghconfig.ProviderHarnessPiACP ||
		resolved.AuthMode != aghconfig.ProviderAuthModeNativeCLI {
		return ""
	}
	model := strings.TrimSpace(resolved.Model)
	if model == "" {
		return ""
	}
	runtimeProvider := strings.TrimSpace(resolved.RuntimeProvider)
	if runtimeProvider == "" {
		runtimeProvider = strings.TrimSpace(resolved.Provider)
	}
	if runtimeProvider == "" || strings.HasPrefix(model, runtimeProvider+"/") {
		return model
	}
	return runtimeProvider + "/" + model
}

func sessionStartEnv(base []string, session *Session) []string {
	return sessionStartEnvForProvider(base, session, aghconfig.ProviderEnvPolicyFiltered)
}

func sessionStartEnvForProvider(
	base []string,
	session *Session,
	envPolicy aghconfig.ProviderEnvPolicy,
) []string {
	env := procutil.FilteredDaemonEnv(base)
	if envPolicy == aghconfig.ProviderEnvPolicyIsolated {
		env = procutil.IsolatedDaemonEnv(base)
	}
	if session == nil {
		return env
	}

	env = setSessionStartEnvValue(env, "AGH_SESSION_ID", strings.TrimSpace(session.ID))
	env = setSessionStartEnvValue(env, "AGH_AGENT", strings.TrimSpace(session.AgentName))
	env = setSessionStartEnvValue(env, "AGH_AGENT_NAME", strings.TrimSpace(session.AgentName))
	env = unsetSessionStartEnvKeys(env, "AGH_SESSION_CHANNEL", "AGH_PEER_ID")

	if effort := strings.TrimSpace(session.ReasoningEffort); effort != "" {
		env = setSessionStartEnvValue(env, "AGH_REASONING_EFFORT", effort)
	} else {
		env = unsetSessionStartEnvKeys(env, "AGH_REASONING_EFFORT")
	}

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
