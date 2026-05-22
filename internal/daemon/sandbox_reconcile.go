package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/compozy/agh/internal/sandbox"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/store"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

const (
	sandboxReconcileFoundKey       = "found"
	sandboxReconcileWorkspaceIDKey = "workspace_id"
)

const (
	sandboxReconcileStatePrepared  = "prepared"
	sandboxReconcileStateDestroyed = "destroyed"
)

type sandboxReconcileSession struct {
	metaPath string
	meta     store.SessionMeta
}

func (d *Daemon) reconcileDaemonSandboxes(ctx context.Context, state *bootState) {
	logger := sandboxReconcileLogger(state)
	if ctx == nil {
		ctx = context.Background()
	}
	if state == nil {
		logger.Warn("daemon: sandbox reconciliation skipped", "error", "boot state is required")
		return
	}
	if state.sandboxRegistry == nil {
		logger.Warn("daemon: sandbox reconciliation skipped", "error", "sandbox registry is required")
		return
	}

	sessions, err := d.loadSandboxReconcileSessions(state)
	if err != nil {
		logger.Warn("daemon: sandbox reconciliation failed to load sessions", "error", err)
		return
	}

	for _, candidate := range sessions {
		if err := ctx.Err(); err != nil {
			logger.Warn("daemon: sandbox reconciliation canceled", "error", err)
			return
		}
		d.reconcileDaemonSandboxSession(ctx, state, candidate)
	}

	logger.Info("daemon: sandbox reconciliation complete", "sessions", len(sessions))
}

func (d *Daemon) loadSandboxReconcileSessions(
	state *bootState,
) ([]sandboxReconcileSession, error) {
	entries, err := os.ReadDir(d.homePaths.SessionsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	logger := sandboxReconcileLogger(state)
	sessions := make([]sandboxReconcileSession, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metaPath := store.SessionMetaFile(filepath.Join(d.homePaths.SessionsDir, entry.Name()))
		meta, err := store.ReadSessionMeta(metaPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			logger.Warn(
				"daemon: sandbox reconciliation skipped unreadable session metadata",
				"session_id", strings.TrimSpace(entry.Name()),
				"path", metaPath,
				"error", err,
			)
			continue
		}
		if !sessionHasRemoteSandbox(meta) {
			continue
		}
		sessions = append(sessions, sandboxReconcileSession{metaPath: metaPath, meta: meta})
	}
	return sessions, nil
}

func (d *Daemon) reconcileDaemonSandboxSession(
	ctx context.Context,
	state *bootState,
	candidate sandboxReconcileSession,
) {
	meta := candidate.meta
	envMeta := cloneDaemonSessionSandboxMeta(meta.Sandbox)
	logger := sandboxReconcileLogger(state)
	if envMeta == nil {
		return
	}

	backend := sandbox.Backend(strings.TrimSpace(envMeta.Backend))
	provider, err := state.sandboxRegistry.Provider(backend)
	if err != nil {
		logger.Warn(
			"daemon: sandbox reconciliation provider unavailable",
			sandboxReconcileLogAttrs(meta, envMeta, 0, err)...,
		)
		return
	}

	resolvedWorkspace, workspaceResolved := d.resolveSandboxReconcileWorkspace(ctx, state, meta, envMeta)
	resolvedEnv := sandboxReconcileResolvedSandbox(envMeta, resolvedWorkspace, workspaceResolved)
	localRoot, localAdditional := sandboxReconcileLocalRoots(resolvedWorkspace, workspaceResolved)

	stateForProvider := sandboxSessionStateFromMeta(envMeta)
	if strings.TrimSpace(stateForProvider.InstanceID) == "" && strings.TrimSpace(envMeta.SandboxID) != "" {
		if found, ok := d.findDaemonSandbox(
			ctx,
			state,
			provider,
			meta,
			envMeta,
			resolvedEnv,
			localRoot,
			localAdditional,
		); ok {
			stateForProvider = mergeSandboxSessionState(stateForProvider, found)
			envMeta = sandboxMetaFromSessionState(stateForProvider, envMeta.State)
		}
	}
	if strings.TrimSpace(stateForProvider.InstanceID) == "" {
		logger.Warn(
			"daemon: sandbox reconciliation skipped missing instance",
			sandboxReconcileLogAttrs(
				meta,
				envMeta,
				0,
				errors.New("sandbox instance id is required"),
			)...,
		)
		return
	}

	if !sessionStateRecoverable(meta.State) {
		d.destroyDaemonSandbox(ctx, state, provider, candidate, envMeta, stateForProvider)
		return
	}

	d.reattachDaemonSandbox(
		ctx,
		state,
		provider,
		candidate,
		envMeta,
		stateForProvider,
		resolvedEnv,
		localRoot,
		localAdditional,
	)
}

func (d *Daemon) findDaemonSandbox(
	ctx context.Context,
	state *bootState,
	provider sandbox.Provider,
	meta store.SessionMeta,
	envMeta *store.SessionSandboxMeta,
	resolvedEnv sandbox.Resolved,
	localRoot string,
	localAdditional []string,
) (sandbox.SessionState, bool) {
	finder, ok := provider.(sandbox.Finder)
	if !ok {
		return sandbox.SessionState{}, false
	}

	labels := map[string]string{
		"agh_sandbox_id": strings.TrimSpace(envMeta.SandboxID),
	}
	found, err := finder.FindSandbox(ctx, sandbox.FindSandboxRequest{
		SessionID:           strings.TrimSpace(meta.ID),
		WorkspaceID:         strings.TrimSpace(meta.WorkspaceID),
		SandboxID:           strings.TrimSpace(envMeta.SandboxID),
		LocalRootDir:        localRoot,
		LocalAdditionalDirs: append([]string(nil), localAdditional...),
		Sandbox:             resolvedEnv,
		ProviderState:       cloneDaemonRawMessage(envMeta.ProviderState),
		Labels:              labels,
	})
	if err != nil {
		attrs := sandboxReconcileLogAttrs(meta, envMeta, 0, err)
		if errors.Is(err, sandbox.ErrSandboxNotFound) {
			sandboxReconcileLogger(state).Info("daemon: sandbox reconciliation remote not found", attrs...)
		} else {
			sandboxReconcileLogger(state).Warn("daemon: sandbox reconciliation remote lookup failed", attrs...)
		}
		return sandbox.SessionState{}, false
	}
	return found, true
}

func (d *Daemon) reattachDaemonSandbox(
	ctx context.Context,
	state *bootState,
	provider sandbox.Provider,
	candidate sandboxReconcileSession,
	envMeta *store.SessionSandboxMeta,
	providerState sandbox.SessionState,
	resolvedEnv sandbox.Resolved,
	localRoot string,
	localAdditional []string,
) {
	meta := candidate.meta
	started := time.Now()
	prepared, err := provider.Prepare(ctx, sandbox.PrepareRequest{
		SessionID:           strings.TrimSpace(meta.ID),
		WorkspaceID:         strings.TrimSpace(meta.WorkspaceID),
		SandboxID:           strings.TrimSpace(envMeta.SandboxID),
		InstanceID:          strings.TrimSpace(providerState.InstanceID),
		LocalRootDir:        localRoot,
		LocalAdditionalDirs: append([]string(nil), localAdditional...),
		Sandbox:             resolvedEnv,
		ProviderState:       cloneDaemonRawMessage(providerState.ProviderState),
	})
	duration := time.Since(started)
	if err != nil {
		sandboxReconcileLogger(state).Warn(
			"daemon: sandbox reattach failed",
			sandboxReconcileLogAttrs(meta, envMeta, duration, err)...,
		)
		if strings.TrimSpace(providerState.InstanceID) != "" {
			d.destroyDaemonSandbox(ctx, state, provider, candidate, envMeta, providerState)
		}
		return
	}

	nextState := mergeSandboxSessionState(providerState, prepared.State)
	nextMeta := sandboxMetaFromSessionState(nextState, sandboxReconcileStatePrepared)
	if nextMeta.Backend == "" {
		nextMeta.Backend = envMeta.Backend
	}
	if nextMeta.Profile == "" {
		nextMeta.Profile = envMeta.Profile
	}
	d.persistSandboxReconcileMeta(ctx, state, candidate, nextMeta)
	sandboxReconcileLogger(state).Info(
		"daemon: sandbox reattach complete",
		sandboxReconcileLogAttrs(meta, nextMeta, duration, nil)...,
	)
}

func (d *Daemon) destroyDaemonSandbox(
	ctx context.Context,
	state *bootState,
	provider sandbox.Provider,
	candidate sandboxReconcileSession,
	envMeta *store.SessionSandboxMeta,
	providerState sandbox.SessionState,
) {
	meta := candidate.meta
	logger := sandboxReconcileLogger(state)
	if strings.TrimSpace(providerState.InstanceID) == "" {
		logger.Warn(
			"daemon: sandbox destroy skipped",
			sandboxReconcileLogAttrs(meta, envMeta, 0, errors.New("sandbox instance id is required"))...,
		)
		return
	}

	started := time.Now()
	err := provider.Destroy(ctx, providerState)
	duration := time.Since(started)
	if err != nil {
		logger.Warn(
			"daemon: sandbox destroy failed",
			sandboxReconcileLogAttrs(meta, envMeta, duration, err)...,
		)
		return
	}

	nextMeta := sandboxMetaFromSessionState(providerState, sandboxReconcileStateDestroyed)
	if nextMeta.Backend == "" {
		nextMeta.Backend = envMeta.Backend
	}
	if nextMeta.Profile == "" {
		nextMeta.Profile = envMeta.Profile
	}
	nextMeta.State = sandboxReconcileStateDestroyed
	d.persistSandboxReconcileMeta(ctx, state, candidate, nextMeta)
	logger.Info(
		"daemon: sandbox destroy complete",
		sandboxReconcileLogAttrs(meta, nextMeta, duration, nil)...,
	)
}

func (d *Daemon) persistSandboxReconcileMeta(
	ctx context.Context,
	state *bootState,
	candidate sandboxReconcileSession,
	envMeta *store.SessionSandboxMeta,
) {
	logger := sandboxReconcileLogger(state)
	next := candidate.meta
	next.Sandbox = cloneDaemonSessionSandboxMeta(envMeta)
	next.UpdatedAt = d.now().UTC()
	if next.CreatedAt.IsZero() {
		next.CreatedAt = next.UpdatedAt
	}
	if err := store.WriteSessionMeta(candidate.metaPath, next); err != nil {
		logger.Warn(
			"daemon: sandbox reconciliation metadata write failed",
			sandboxReconcileLogAttrs(candidate.meta, envMeta, 0, err)...,
		)
		return
	}
	if state == nil || state.registry == nil {
		return
	}
	if err := state.registry.RegisterSession(ctx, sessionInfoFromSandboxReconcileMeta(next)); err != nil {
		logger.Warn(
			"daemon: sandbox reconciliation session index update failed",
			sandboxReconcileLogAttrs(candidate.meta, envMeta, 0, err)...,
		)
	}
}

func (d *Daemon) resolveSandboxReconcileWorkspace(
	ctx context.Context,
	state *bootState,
	meta store.SessionMeta,
	envMeta *store.SessionSandboxMeta,
) (*workspacepkg.ResolvedWorkspace, bool) {
	if state == nil || state.workspaceResolver == nil {
		return nil, false
	}
	resolved, err := state.workspaceResolver.Resolve(ctx, strings.TrimSpace(meta.WorkspaceID))
	if err != nil {
		sandboxReconcileLogger(state).Warn(
			"daemon: sandbox reconciliation workspace resolve failed",
			sandboxReconcileLogAttrs(meta, envMeta, 0, err)...,
		)
		return nil, false
	}
	return &resolved, true
}

func sessionHasRemoteSandbox(meta store.SessionMeta) bool {
	if meta.Sandbox == nil {
		return false
	}
	backend := strings.TrimSpace(meta.Sandbox.Backend)
	if backend == "" {
		backend = string(sandbox.BackendLocal)
	}
	return sandbox.Backend(backend) != sandbox.BackendLocal
}

func sessionStateRecoverable(state string) bool {
	switch session.State(strings.TrimSpace(state)) {
	case session.StateStarting, session.StateActive, session.StateStopping:
		return true
	default:
		return false
	}
}

func sandboxReconcileResolvedSandbox(
	envMeta *store.SessionSandboxMeta,
	resolvedWorkspace *workspacepkg.ResolvedWorkspace,
	workspaceResolved bool,
) sandbox.Resolved {
	resolved := sandbox.Resolved{}
	if workspaceResolved && resolvedWorkspace != nil {
		resolved = resolvedWorkspace.Sandbox
	}
	backend := sandbox.Backend(strings.TrimSpace(envMeta.Backend))
	if backend.Valid() {
		resolved.Backend = backend
	}
	if !resolved.Backend.Valid() {
		resolved.Backend = sandbox.BackendLocal
	}
	if profile := strings.TrimSpace(envMeta.Profile); profile != "" {
		resolved.Profile = profile
	}
	if strings.TrimSpace(resolved.Profile) == "" {
		resolved.Profile = string(resolved.Backend)
	}
	return resolved
}

func sandboxReconcileLocalRoots(
	resolvedWorkspace *workspacepkg.ResolvedWorkspace,
	workspaceResolved bool,
) (string, []string) {
	if !workspaceResolved || resolvedWorkspace == nil {
		return "", nil
	}
	return strings.TrimSpace(resolvedWorkspace.RootDir), append([]string(nil), resolvedWorkspace.AdditionalDirs...)
}

func sandboxSessionStateFromMeta(meta *store.SessionSandboxMeta) sandbox.SessionState {
	if meta == nil {
		return sandbox.SessionState{}
	}
	return sandbox.SessionState{
		SandboxID:             strings.TrimSpace(meta.SandboxID),
		Backend:               sandbox.Backend(strings.TrimSpace(meta.Backend)),
		Profile:               strings.TrimSpace(meta.Profile),
		State:                 strings.TrimSpace(meta.State),
		InstanceID:            strings.TrimSpace(meta.InstanceID),
		RuntimeRootDir:        strings.TrimSpace(meta.RuntimeRootDir),
		RuntimeAdditionalDirs: append([]string(nil), meta.RuntimeAdditionalDirs...),
		ProviderState:         cloneDaemonRawMessage(meta.ProviderState),
		SSHAccessExpiresAt:    cloneDaemonTimePointer(meta.SSHAccessExpiresAt),
	}
}

func sandboxMetaFromSessionState(
	state sandbox.SessionState,
	fallbackState string,
) *store.SessionSandboxMeta {
	envState := strings.TrimSpace(state.State)
	if envState == "" || envState == sandboxReconcileFoundKey || envState == string(RestartStatusReady) {
		envState = fallbackState
	}
	return &store.SessionSandboxMeta{
		SandboxID:             strings.TrimSpace(state.SandboxID),
		Backend:               string(state.Backend),
		Profile:               strings.TrimSpace(state.Profile),
		State:                 envState,
		InstanceID:            strings.TrimSpace(state.InstanceID),
		RuntimeRootDir:        strings.TrimSpace(state.RuntimeRootDir),
		RuntimeAdditionalDirs: append([]string(nil), state.RuntimeAdditionalDirs...),
		ProviderState:         cloneDaemonRawMessage(state.ProviderState),
		SSHAccessExpiresAt:    cloneDaemonTimePointer(state.SSHAccessExpiresAt),
	}
}

func mergeSandboxSessionState(
	base sandbox.SessionState,
	overlay sandbox.SessionState,
) sandbox.SessionState {
	next := base
	if strings.TrimSpace(overlay.SandboxID) != "" {
		next.SandboxID = strings.TrimSpace(overlay.SandboxID)
	}
	if overlay.Backend.Valid() {
		next.Backend = overlay.Backend
	}
	if strings.TrimSpace(overlay.Profile) != "" {
		next.Profile = strings.TrimSpace(overlay.Profile)
	}
	if strings.TrimSpace(overlay.State) != "" {
		next.State = strings.TrimSpace(overlay.State)
	}
	if strings.TrimSpace(overlay.InstanceID) != "" {
		next.InstanceID = strings.TrimSpace(overlay.InstanceID)
	}
	if strings.TrimSpace(overlay.RuntimeRootDir) != "" {
		next.RuntimeRootDir = strings.TrimSpace(overlay.RuntimeRootDir)
	}
	if len(overlay.RuntimeAdditionalDirs) > 0 {
		next.RuntimeAdditionalDirs = append([]string(nil), overlay.RuntimeAdditionalDirs...)
	}
	if len(overlay.ProviderState) > 0 {
		next.ProviderState = cloneDaemonRawMessage(overlay.ProviderState)
	}
	if overlay.SSHAccessExpiresAt != nil {
		next.SSHAccessExpiresAt = cloneDaemonTimePointer(overlay.SSHAccessExpiresAt)
	}
	if !overlay.PreparedAt.IsZero() {
		next.PreparedAt = overlay.PreparedAt
	}
	return next
}

func sessionInfoFromSandboxReconcileMeta(meta store.SessionMeta) store.SessionInfo {
	stopReason := store.StopReason("")
	if meta.StopReason != nil {
		stopReason = *meta.StopReason
	}
	return store.SessionInfo{
		ID:           strings.TrimSpace(meta.ID),
		Name:         strings.TrimSpace(meta.Name),
		AgentName:    strings.TrimSpace(meta.AgentName),
		Provider:     strings.TrimSpace(meta.Provider),
		WorkspaceID:  strings.TrimSpace(meta.WorkspaceID),
		Channel:      strings.TrimSpace(meta.Channel),
		SessionType:  strings.TrimSpace(meta.SessionType),
		Lineage:      store.NormalizeSessionLineage(meta.ID, meta.Lineage),
		State:        strings.TrimSpace(meta.State),
		ACPSessionID: cloneDaemonStringPointer(meta.ACPSessionID),
		StopReason:   stopReason,
		StopDetail:   strings.TrimSpace(meta.StopDetail),
		Sandbox:      cloneDaemonSessionSandboxMeta(meta.Sandbox),
		CreatedAt:    meta.CreatedAt,
		UpdatedAt:    meta.UpdatedAt,
	}
}

func sandboxReconcileLogAttrs(
	meta store.SessionMeta,
	envMeta *store.SessionSandboxMeta,
	duration time.Duration,
	err error,
) []any {
	attrs := []any{
		"session_id", strings.TrimSpace(meta.ID),
		sandboxReconcileWorkspaceIDKey, strings.TrimSpace(meta.WorkspaceID),
		"session_state", strings.TrimSpace(meta.State),
		"duration_ms", duration.Milliseconds(),
	}
	if envMeta != nil {
		attrs = append(
			attrs,
			"backend", strings.TrimSpace(envMeta.Backend),
			"profile", strings.TrimSpace(envMeta.Profile),
			"sandbox_id", strings.TrimSpace(envMeta.SandboxID),
			"instance_id", strings.TrimSpace(envMeta.InstanceID),
		)
	}
	if err != nil {
		attrs = append(attrs, "error", err)
	}
	return attrs
}

func sandboxReconcileLogger(state *bootState) *slog.Logger {
	if state != nil && state.logger != nil {
		return state.logger
	}
	return slog.Default()
}

func cloneDaemonSessionSandboxMeta(
	meta *store.SessionSandboxMeta,
) *store.SessionSandboxMeta {
	if meta == nil {
		return nil
	}
	cloned := *meta
	cloned.RuntimeAdditionalDirs = append([]string(nil), meta.RuntimeAdditionalDirs...)
	cloned.ProviderState = cloneDaemonRawMessage(meta.ProviderState)
	cloned.SSHAccessExpiresAt = cloneDaemonTimePointer(meta.SSHAccessExpiresAt)
	cloned.LastSyncAt = cloneDaemonTimePointer(meta.LastSyncAt)
	return &cloned
}

func cloneDaemonRawMessage(value json.RawMessage) json.RawMessage {
	if value == nil {
		return nil
	}
	cloned := make(json.RawMessage, len(value))
	copy(cloned, value)
	return cloned
}

func cloneDaemonTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneDaemonStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := strings.TrimSpace(*value)
	return &cloned
}
