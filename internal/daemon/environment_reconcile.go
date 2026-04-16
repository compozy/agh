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

	"github.com/pedronauck/agh/internal/environment"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	environmentReconcileStatePrepared  = "prepared"
	environmentReconcileStateDestroyed = "destroyed"
)

type environmentReconcileSession struct {
	metaPath string
	meta     store.SessionMeta
}

func (d *Daemon) reconcileDaemonEnvironments(ctx context.Context, state *bootState) {
	logger := environmentReconcileLogger(state)
	if ctx == nil {
		ctx = context.Background()
	}
	if state == nil {
		logger.Warn("daemon: environment reconciliation skipped", "error", "boot state is required")
		return
	}
	if state.environmentRegistry == nil {
		logger.Warn("daemon: environment reconciliation skipped", "error", "environment registry is required")
		return
	}

	sessions, err := d.loadEnvironmentReconcileSessions(state)
	if err != nil {
		logger.Warn("daemon: environment reconciliation failed to load sessions", "error", err)
		return
	}

	for _, candidate := range sessions {
		if err := ctx.Err(); err != nil {
			logger.Warn("daemon: environment reconciliation canceled", "error", err)
			return
		}
		d.reconcileDaemonEnvironmentSession(ctx, state, candidate)
	}

	logger.Info("daemon: environment reconciliation complete", "sessions", len(sessions))
}

func (d *Daemon) loadEnvironmentReconcileSessions(
	state *bootState,
) ([]environmentReconcileSession, error) {
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

	logger := environmentReconcileLogger(state)
	sessions := make([]environmentReconcileSession, 0, len(entries))
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
				"daemon: environment reconciliation skipped unreadable session metadata",
				"session_id", strings.TrimSpace(entry.Name()),
				"path", metaPath,
				"error", err,
			)
			continue
		}
		if !sessionHasRemoteEnvironment(meta) {
			continue
		}
		sessions = append(sessions, environmentReconcileSession{metaPath: metaPath, meta: meta})
	}
	return sessions, nil
}

func (d *Daemon) reconcileDaemonEnvironmentSession(
	ctx context.Context,
	state *bootState,
	candidate environmentReconcileSession,
) {
	meta := candidate.meta
	envMeta := cloneDaemonSessionEnvironmentMeta(meta.Environment)
	logger := environmentReconcileLogger(state)
	if envMeta == nil {
		return
	}

	backend := environment.Backend(strings.TrimSpace(envMeta.Backend))
	provider, err := state.environmentRegistry.Provider(backend)
	if err != nil {
		logger.Warn(
			"daemon: environment reconciliation provider unavailable",
			environmentReconcileLogAttrs(meta, envMeta, 0, err)...,
		)
		return
	}

	resolvedWorkspace, workspaceResolved := d.resolveEnvironmentReconcileWorkspace(ctx, state, meta, envMeta)
	resolvedEnv := environmentReconcileResolvedEnvironment(envMeta, resolvedWorkspace, workspaceResolved)
	localRoot, localAdditional := environmentReconcileLocalRoots(resolvedWorkspace, workspaceResolved)

	stateForProvider := environmentSessionStateFromMeta(envMeta)
	if strings.TrimSpace(stateForProvider.InstanceID) == "" && strings.TrimSpace(envMeta.EnvironmentID) != "" {
		if found, ok := d.findDaemonEnvironment(
			ctx,
			state,
			provider,
			meta,
			envMeta,
			resolvedEnv,
			localRoot,
			localAdditional,
		); ok {
			stateForProvider = mergeEnvironmentSessionState(stateForProvider, found)
			envMeta = environmentMetaFromSessionState(stateForProvider, envMeta.State)
		}
	}
	if strings.TrimSpace(stateForProvider.InstanceID) == "" {
		logger.Warn(
			"daemon: environment reconciliation skipped missing instance",
			environmentReconcileLogAttrs(
				meta,
				envMeta,
				0,
				errors.New("environment instance id is required"),
			)...,
		)
		return
	}

	if !sessionStateRecoverable(meta.State) {
		d.destroyDaemonEnvironment(ctx, state, provider, candidate, envMeta, stateForProvider)
		return
	}

	d.reattachDaemonEnvironment(
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

func (d *Daemon) findDaemonEnvironment(
	ctx context.Context,
	state *bootState,
	provider environment.Provider,
	meta store.SessionMeta,
	envMeta *store.SessionEnvironmentMeta,
	resolvedEnv environment.Resolved,
	localRoot string,
	localAdditional []string,
) (environment.SessionState, bool) {
	finder, ok := provider.(environment.Finder)
	if !ok {
		return environment.SessionState{}, false
	}

	labels := map[string]string{
		"agh_environment_id": strings.TrimSpace(envMeta.EnvironmentID),
	}
	found, err := finder.FindEnvironment(ctx, environment.FindEnvironmentRequest{
		SessionID:           strings.TrimSpace(meta.ID),
		WorkspaceID:         strings.TrimSpace(meta.WorkspaceID),
		EnvironmentID:       strings.TrimSpace(envMeta.EnvironmentID),
		LocalRootDir:        localRoot,
		LocalAdditionalDirs: append([]string(nil), localAdditional...),
		Environment:         resolvedEnv,
		ProviderState:       cloneDaemonRawMessage(envMeta.ProviderState),
		Labels:              labels,
	})
	if err != nil {
		attrs := environmentReconcileLogAttrs(meta, envMeta, 0, err)
		if errors.Is(err, environment.ErrEnvironmentNotFound) {
			environmentReconcileLogger(state).Info("daemon: environment reconciliation remote not found", attrs...)
		} else {
			environmentReconcileLogger(state).Warn("daemon: environment reconciliation remote lookup failed", attrs...)
		}
		return environment.SessionState{}, false
	}
	return found, true
}

func (d *Daemon) reattachDaemonEnvironment(
	ctx context.Context,
	state *bootState,
	provider environment.Provider,
	candidate environmentReconcileSession,
	envMeta *store.SessionEnvironmentMeta,
	providerState environment.SessionState,
	resolvedEnv environment.Resolved,
	localRoot string,
	localAdditional []string,
) {
	meta := candidate.meta
	started := time.Now()
	prepared, err := provider.Prepare(ctx, environment.PrepareRequest{
		SessionID:           strings.TrimSpace(meta.ID),
		WorkspaceID:         strings.TrimSpace(meta.WorkspaceID),
		EnvironmentID:       strings.TrimSpace(envMeta.EnvironmentID),
		InstanceID:          strings.TrimSpace(providerState.InstanceID),
		LocalRootDir:        localRoot,
		LocalAdditionalDirs: append([]string(nil), localAdditional...),
		Environment:         resolvedEnv,
		ProviderState:       cloneDaemonRawMessage(providerState.ProviderState),
	})
	duration := time.Since(started)
	if err != nil {
		environmentReconcileLogger(state).Warn(
			"daemon: environment reattach failed",
			environmentReconcileLogAttrs(meta, envMeta, duration, err)...,
		)
		if strings.TrimSpace(providerState.InstanceID) != "" {
			d.destroyDaemonEnvironment(ctx, state, provider, candidate, envMeta, providerState)
		}
		return
	}

	nextState := mergeEnvironmentSessionState(providerState, prepared.State)
	nextMeta := environmentMetaFromSessionState(nextState, environmentReconcileStatePrepared)
	if nextMeta.Backend == "" {
		nextMeta.Backend = envMeta.Backend
	}
	if nextMeta.Profile == "" {
		nextMeta.Profile = envMeta.Profile
	}
	d.persistEnvironmentReconcileMeta(ctx, state, candidate, nextMeta)
	environmentReconcileLogger(state).Info(
		"daemon: environment reattach complete",
		environmentReconcileLogAttrs(meta, nextMeta, duration, nil)...,
	)
}

func (d *Daemon) destroyDaemonEnvironment(
	ctx context.Context,
	state *bootState,
	provider environment.Provider,
	candidate environmentReconcileSession,
	envMeta *store.SessionEnvironmentMeta,
	providerState environment.SessionState,
) {
	meta := candidate.meta
	logger := environmentReconcileLogger(state)
	if strings.TrimSpace(providerState.InstanceID) == "" {
		logger.Warn(
			"daemon: environment destroy skipped",
			environmentReconcileLogAttrs(meta, envMeta, 0, errors.New("environment instance id is required"))...,
		)
		return
	}

	started := time.Now()
	err := provider.Destroy(ctx, providerState)
	duration := time.Since(started)
	if err != nil {
		logger.Warn(
			"daemon: environment destroy failed",
			environmentReconcileLogAttrs(meta, envMeta, duration, err)...,
		)
		return
	}

	nextMeta := environmentMetaFromSessionState(providerState, environmentReconcileStateDestroyed)
	if nextMeta.Backend == "" {
		nextMeta.Backend = envMeta.Backend
	}
	if nextMeta.Profile == "" {
		nextMeta.Profile = envMeta.Profile
	}
	nextMeta.State = environmentReconcileStateDestroyed
	d.persistEnvironmentReconcileMeta(ctx, state, candidate, nextMeta)
	logger.Info(
		"daemon: environment destroy complete",
		environmentReconcileLogAttrs(meta, nextMeta, duration, nil)...,
	)
}

func (d *Daemon) persistEnvironmentReconcileMeta(
	ctx context.Context,
	state *bootState,
	candidate environmentReconcileSession,
	envMeta *store.SessionEnvironmentMeta,
) {
	logger := environmentReconcileLogger(state)
	next := candidate.meta
	next.Environment = cloneDaemonSessionEnvironmentMeta(envMeta)
	next.UpdatedAt = d.now().UTC()
	if next.CreatedAt.IsZero() {
		next.CreatedAt = next.UpdatedAt
	}
	if err := store.WriteSessionMeta(candidate.metaPath, next); err != nil {
		logger.Warn(
			"daemon: environment reconciliation metadata write failed",
			environmentReconcileLogAttrs(candidate.meta, envMeta, 0, err)...,
		)
		return
	}
	if state == nil || state.registry == nil {
		return
	}
	if err := state.registry.RegisterSession(ctx, sessionInfoFromEnvironmentReconcileMeta(next)); err != nil {
		logger.Warn(
			"daemon: environment reconciliation session index update failed",
			environmentReconcileLogAttrs(candidate.meta, envMeta, 0, err)...,
		)
	}
}

func (d *Daemon) resolveEnvironmentReconcileWorkspace(
	ctx context.Context,
	state *bootState,
	meta store.SessionMeta,
	envMeta *store.SessionEnvironmentMeta,
) (*workspacepkg.ResolvedWorkspace, bool) {
	if state == nil || state.workspaceResolver == nil {
		return nil, false
	}
	resolved, err := state.workspaceResolver.Resolve(ctx, strings.TrimSpace(meta.WorkspaceID))
	if err != nil {
		environmentReconcileLogger(state).Warn(
			"daemon: environment reconciliation workspace resolve failed",
			environmentReconcileLogAttrs(meta, envMeta, 0, err)...,
		)
		return nil, false
	}
	return &resolved, true
}

func sessionHasRemoteEnvironment(meta store.SessionMeta) bool {
	if meta.Environment == nil {
		return false
	}
	backend := strings.TrimSpace(meta.Environment.Backend)
	if backend == "" {
		backend = string(environment.BackendLocal)
	}
	return environment.Backend(backend) != environment.BackendLocal
}

func sessionStateRecoverable(state string) bool {
	switch session.State(strings.TrimSpace(state)) {
	case session.StateStarting, session.StateActive, session.StateStopping:
		return true
	default:
		return false
	}
}

func environmentReconcileResolvedEnvironment(
	envMeta *store.SessionEnvironmentMeta,
	resolvedWorkspace *workspacepkg.ResolvedWorkspace,
	workspaceResolved bool,
) environment.Resolved {
	resolved := environment.Resolved{}
	if workspaceResolved && resolvedWorkspace != nil {
		resolved = resolvedWorkspace.Environment
	}
	backend := environment.Backend(strings.TrimSpace(envMeta.Backend))
	if backend.Valid() {
		resolved.Backend = backend
	}
	if !resolved.Backend.Valid() {
		resolved.Backend = environment.BackendLocal
	}
	if profile := strings.TrimSpace(envMeta.Profile); profile != "" {
		resolved.Profile = profile
	}
	if strings.TrimSpace(resolved.Profile) == "" {
		resolved.Profile = string(resolved.Backend)
	}
	return resolved
}

func environmentReconcileLocalRoots(
	resolvedWorkspace *workspacepkg.ResolvedWorkspace,
	workspaceResolved bool,
) (string, []string) {
	if !workspaceResolved || resolvedWorkspace == nil {
		return "", nil
	}
	return strings.TrimSpace(resolvedWorkspace.RootDir), append([]string(nil), resolvedWorkspace.AdditionalDirs...)
}

func environmentSessionStateFromMeta(meta *store.SessionEnvironmentMeta) environment.SessionState {
	if meta == nil {
		return environment.SessionState{}
	}
	return environment.SessionState{
		EnvironmentID:         strings.TrimSpace(meta.EnvironmentID),
		Backend:               environment.Backend(strings.TrimSpace(meta.Backend)),
		Profile:               strings.TrimSpace(meta.Profile),
		State:                 strings.TrimSpace(meta.State),
		InstanceID:            strings.TrimSpace(meta.InstanceID),
		RuntimeRootDir:        strings.TrimSpace(meta.RuntimeRootDir),
		RuntimeAdditionalDirs: append([]string(nil), meta.RuntimeAdditionalDirs...),
		ProviderState:         cloneDaemonRawMessage(meta.ProviderState),
		SSHAccessExpiresAt:    cloneDaemonTimePointer(meta.SSHAccessExpiresAt),
	}
}

func environmentMetaFromSessionState(
	state environment.SessionState,
	fallbackState string,
) *store.SessionEnvironmentMeta {
	envState := strings.TrimSpace(state.State)
	if envState == "" || envState == "found" || envState == "ready" {
		envState = fallbackState
	}
	return &store.SessionEnvironmentMeta{
		EnvironmentID:         strings.TrimSpace(state.EnvironmentID),
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

func mergeEnvironmentSessionState(
	base environment.SessionState,
	overlay environment.SessionState,
) environment.SessionState {
	next := base
	if strings.TrimSpace(overlay.EnvironmentID) != "" {
		next.EnvironmentID = strings.TrimSpace(overlay.EnvironmentID)
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

func sessionInfoFromEnvironmentReconcileMeta(meta store.SessionMeta) store.SessionInfo {
	stopReason := store.StopReason("")
	if meta.StopReason != nil {
		stopReason = *meta.StopReason
	}
	return store.SessionInfo{
		ID:           strings.TrimSpace(meta.ID),
		Name:         strings.TrimSpace(meta.Name),
		AgentName:    strings.TrimSpace(meta.AgentName),
		WorkspaceID:  strings.TrimSpace(meta.WorkspaceID),
		Channel:      strings.TrimSpace(meta.Channel),
		SessionType:  strings.TrimSpace(meta.SessionType),
		State:        strings.TrimSpace(meta.State),
		ACPSessionID: cloneDaemonStringPointer(meta.ACPSessionID),
		StopReason:   stopReason,
		StopDetail:   strings.TrimSpace(meta.StopDetail),
		Environment:  cloneDaemonSessionEnvironmentMeta(meta.Environment),
		CreatedAt:    meta.CreatedAt,
		UpdatedAt:    meta.UpdatedAt,
	}
}

func environmentReconcileLogAttrs(
	meta store.SessionMeta,
	envMeta *store.SessionEnvironmentMeta,
	duration time.Duration,
	err error,
) []any {
	attrs := []any{
		"session_id", strings.TrimSpace(meta.ID),
		"workspace_id", strings.TrimSpace(meta.WorkspaceID),
		"session_state", strings.TrimSpace(meta.State),
		"duration_ms", duration.Milliseconds(),
	}
	if envMeta != nil {
		attrs = append(
			attrs,
			"backend", strings.TrimSpace(envMeta.Backend),
			"profile", strings.TrimSpace(envMeta.Profile),
			"environment_id", strings.TrimSpace(envMeta.EnvironmentID),
			"instance_id", strings.TrimSpace(envMeta.InstanceID),
		)
	}
	if err != nil {
		attrs = append(attrs, "error", err)
	}
	return attrs
}

func environmentReconcileLogger(state *bootState) *slog.Logger {
	if state != nil && state.logger != nil {
		return state.logger
	}
	return slog.Default()
}

func cloneDaemonSessionEnvironmentMeta(
	meta *store.SessionEnvironmentMeta,
) *store.SessionEnvironmentMeta {
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
