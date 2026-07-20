package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/compozy/agh/internal/acp"
	aghconfig "github.com/compozy/agh/internal/config"
	hookspkg "github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/workref"
)

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

const (
	sessionStartActionCreate = "create"
	sessionStartActionResume = "resume"
)

func (m *Manager) prepareResumeStart(ctx context.Context, meta store.SessionMeta) (sessionStartSpec, error) {
	meta, err := m.dispatchSessionPreResume(ctx, meta)
	if err != nil {
		return sessionStartSpec{}, fmt.Errorf("session: dispatch pre-resume for %q: %w", meta.ID, err)
	}

	resolvedWorkspace, err := m.resolveResumeWorkspace(ctx, meta)
	if err != nil {
		return sessionStartSpec{}, fmt.Errorf("session: resolve resume workspace for %q: %w", meta.ID, err)
	}
	if err := validateSessionParticipationWorkspace(meta.NetworkSpecSnapshot(), resolvedWorkspace.ID); err != nil {
		return sessionStartSpec{}, fmt.Errorf("session: validate resume participation for %q: %w", meta.ID, err)
	}
	cwd, err := resumeSessionCWD(meta, resolvedWorkspace.RootDir)
	if err != nil {
		return sessionStartSpec{}, fmt.Errorf("session: validate resume cwd for %q: %w", meta.ID, err)
	}

	return sessionStartSpec{
		sessionID:               meta.ID,
		sandboxID:               sessionSandboxID(meta.Sandbox),
		sandbox:                 cloneSessionSandboxMeta(meta.Sandbox),
		sandboxDisabled:         meta.Sandbox == nil,
		sessionName:             meta.Name,
		agentName:               meta.AgentName,
		provider:                strings.TrimSpace(meta.Provider),
		model:                   strings.TrimSpace(meta.Model),
		reasoningEffort:         strings.TrimSpace(meta.ReasoningEffort),
		permissions:             aghconfig.PermissionMode(strings.TrimSpace(meta.EffectivePermissions)),
		workspace:               resolvedWorkspace,
		networkParticipation:    meta.NetworkSpecSnapshot(),
		networkOwnerKey:         meta.NetworkOwnerKeySnapshot(),
		cwd:                     cwd,
		sessionType:             normalizeSessionType(Type(meta.SessionType)),
		lineage:                 store.NormalizeSessionLineage(meta.ID, meta.Lineage),
		postEvent:               hookspkg.HookSessionPostResume,
		startAction:             sessionStartActionResume,
		includePromptUpdatedAt:  true,
		preserveStopReason:      sessionMetaStopReason(meta) == store.StopAgentCrashed,
		createdAt:               meta.CreatedAt,
		acpSessionID:            derefString(meta.ACPSessionID),
		stopReason:              sessionMetaStopReason(meta),
		stopDetail:              strings.TrimSpace(meta.StopDetail),
		failure:                 store.CloneSessionFailure(meta.Failure),
		soulSnapshotID:          strings.TrimSpace(meta.SoulSnapshotID),
		soulDigest:              strings.TrimSpace(meta.SoulDigest),
		parentSoulDigest:        strings.TrimSpace(meta.ParentSoulDigest),
		creationProfile:         cloneCreationProfile(meta.CreationProfile),
		creationOptions:         cloneCreationOptions(meta.CreationOptions),
		creationIdentity:        creationIdentityFromMeta(meta),
		creationIdentityPinned:  meta.CreationProfile != nil,
		creationIdentityEnabled: meta.CreationProfile != nil,
		advertisedCommands:      store.CloneSessionAdvertisedCommands(meta.AdvertisedCommands),
	}, nil
}

func resumeSessionCWD(meta store.SessionMeta, workspaceRoot string) (string, error) {
	requested := workspaceRoot
	if meta.CreationProfile != nil {
		requested = strings.TrimSpace(meta.CreationProfile.CWD)
	} else if cwd := strings.TrimSpace(meta.CWD); cwd != "" {
		requested = cwd
	}
	return ResolveSessionCWD(workspaceRoot, requested)
}

func (m *Manager) startSession(ctx context.Context, spec *sessionStartSpec) (_ *Session, err error) {
	now := m.now()

	runtime, err := m.prepareSessionStartRuntimeAndIdentity(ctx, spec, now)
	if err != nil {
		return nil, fmt.Errorf("session: prepare %s runtime for %q: %w", spec.startAction, spec.sessionID, err)
	}
	defer func() {
		if err != nil && m.hostedMCP != nil {
			m.hostedMCP.CancelLaunch(spec.sessionID)
		}
	}()

	if err := m.reserveStart(ctx, spec.sessionID, spec.workspace.ID); err != nil {
		return nil, fmt.Errorf("session: reserve %s session %q: %w", spec.startAction, spec.sessionID, err)
	}
	defer func() {
		if err != nil {
			m.releaseReservation(spec.sessionID)
		}
	}()

	storage, err := m.openSessionStartStorage(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("session: open %s storage for %q: %w", spec.startAction, spec.sessionID, err)
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
	if err := m.restoreAdvertisedCommands(ctx, session); err != nil {
		return nil, fmt.Errorf("session: restore advertised commands for %q: %w", spec.sessionID, err)
	}

	startOpts, err := m.prepareSessionLaunch(ctx, spec, session, &runtime)
	if err != nil {
		return nil, fmt.Errorf("session: prepare %s launch for %q: %w", spec.startAction, spec.sessionID, err)
	}

	proc, err = m.startAgentProcess(ctx, spec, session, startOpts)
	if err != nil {
		return nil, fmt.Errorf("session: start %s agent process for %q: %w", spec.startAction, spec.sessionID, err)
	}
	if err := m.persistResumeReplayMarker(ctx, spec, session); err != nil {
		return nil, err
	}

	if err := m.activateAndWatch(
		ctx,
		session,
		proc,
		strings.TrimSpace(startOpts.PreferredModel) == "",
		runtime.agent,
		runtime.networkCapabilities,
		spec.postEvent,
		spec.preserveStopReason,
	); err != nil {
		return nil, fmt.Errorf("session: activate %s session %q: %w", spec.startAction, spec.sessionID, err)
	}
	m.observeCommittedParticipation(ctx, spec.participationObservation)
	if spec.resumeReplay {
		m.stageResumeReplay(spec.sessionID, spec.resumeReplayBlock)
	}

	return session, nil
}

func (m *Manager) prepareSessionStartRuntimeAndIdentity(
	ctx context.Context,
	spec *sessionStartSpec,
	now time.Time,
) (sessionStartRuntime, error) {
	runtime, err := m.prepareSessionStartRuntime(ctx, spec, now)
	if err != nil {
		spec.startLogger(m).Warn(
			"session.start.runtime_prepare_failed",
			"phase",
			spec.startAction,
			"error",
			err,
		)
		return sessionStartRuntime{}, fmt.Errorf(
			"session: prepare %s runtime for %q: %w",
			spec.startAction,
			spec.sessionID,
			err,
		)
	}
	if err := prepareStartCreationIdentityIfEnabled(spec, runtime.agent); err != nil {
		return sessionStartRuntime{}, fmt.Errorf("session: prepare creation identity for %q: %w", spec.sessionID, err)
	}
	return runtime, nil
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
		SessionID:            strings.TrimSpace(s.sessionID),
		SessionName:          strings.TrimSpace(s.sessionName),
		AgentName:            strings.TrimSpace(s.agentName),
		Provider:             strings.TrimSpace(s.provider),
		WorkspaceID:          ref.WorkspaceID,
		Workspace:            ref.Workspace,
		NetworkParticipation: s.networkParticipation,
		SessionType:          normalizeSessionType(s.sessionType),
		SoulSnapshot:         cloneSoulSnapshotPointer(s.soulSnapshot),
		CreatedAt:            s.createdAt,
		UpdatedAt:            updatedAt,
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
		return sessionStartRuntime{}, fmt.Errorf("session: prepare soul for %q: %w", spec.sessionID, err)
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
		return sessionStartRuntime{}, fmt.Errorf("session: assemble startup prompt for %q: %w", spec.sessionID, err)
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
		return sessionStartRuntime{}, fmt.Errorf("session: validate runtime overrides for %q: %w", spec.sessionID, err)
	}
	if err := m.validateExplicitModel(ctx, spec, resolved); err != nil {
		return sessionStartRuntime{}, fmt.Errorf("session: validate model for %q: %w", spec.sessionID, err)
	}
	if err := spec.applyResolvedReasoningEffort(resolved); err != nil {
		return sessionStartRuntime{}, err
	}
	if err := spec.applyAllowedToolsOverride(&resolved, m.toolsetCatalog); err != nil {
		return sessionStartRuntime{}, fmt.Errorf("session: apply allowed tools for %q: %w", spec.sessionID, err)
	}

	startMCPServers, err := m.sessionMCPServers(ctx, spec, resolved)
	if err != nil {
		return sessionStartRuntime{}, fmt.Errorf("session: resolve MCP servers for %q: %w", spec.sessionID, err)
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
	if strings.EqualFold(spec.runtimeMode, RuntimeModeVerdictOnly) {
		return nil, nil
	}
	if !resolved.SessionMCP {
		spec.startLogger(m).Info(
			"session.mcp.skipped",
			"reason",
			"provider_session_mcp_disabled",
			"resolved_agent_name",
			strings.TrimSpace(resolved.Name),
			"resolved_provider",
			strings.TrimSpace(resolved.Provider),
		)
		return nil, nil
	}
	if m.hostedMCP == nil {
		spec.startLogger(m).Warn(
			"session.mcp.hosted_mcp_unavailable",
			"reason",
			"hosted_mcp_launcher_unavailable",
			"resolved_agent_name",
			strings.TrimSpace(resolved.Name),
			"resolved_provider",
			strings.TrimSpace(resolved.Provider),
			"configured_mcp_servers",
			len(resolved.MCPServers),
		)
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
