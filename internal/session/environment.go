package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	envpkg "github.com/pedronauck/agh/internal/environment"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/store"
)

const (
	environmentStateCreating  = "creating"
	environmentStatePrepared  = "prepared"
	environmentStateStopped   = "stopped"
	environmentStateDestroyed = "destroyed"

	environmentEventPrepareStart        = "environment.prepare.start"
	environmentEventPrepareComplete     = "environment.prepare.complete"
	environmentEventPrepareError        = "environment.prepare.error"
	environmentEventSyncStart           = "environment.sync.start"
	environmentEventSyncComplete        = "environment.sync.complete"
	environmentEventSyncError           = "environment.sync.error"
	environmentEventTransportConnect    = "environment.transport.connect"
	environmentEventTransportDisconnect = "environment.transport.disconnect"
	environmentEventTransportError      = "environment.transport.error"
	environmentEventDestroyStart        = "environment.destroy.start"
	environmentEventDestroyComplete     = "environment.destroy.complete"
	environmentEventDestroyError        = "environment.destroy.error"
)

// EnvironmentLifecycleEvent reports provider lifecycle timing to optional observers.
type EnvironmentLifecycleEvent struct {
	Name          string
	Span          string
	SessionID     string
	WorkspaceID   string
	EnvironmentID string
	Backend       string
	Profile       string
	InstanceID    string
	Reason        string
	Duration      time.Duration
	ErrorKind     string
	Error         string
	Timestamp     time.Time
}

// EnvironmentLifecycleNotifier is an optional notifier extension for environment lifecycle spans.
type EnvironmentLifecycleNotifier interface {
	OnEnvironmentLifecycleEvent(context.Context, EnvironmentLifecycleEvent)
}

func (m *Manager) prepareEnvironmentForStart(
	ctx context.Context,
	spec *sessionStartSpec,
	session *Session,
	opts acp.StartOpts,
) (acp.StartOpts, error) {
	if spec == nil {
		return acp.StartOpts{}, errors.New("session: start spec is required")
	}
	if session == nil {
		return acp.StartOpts{}, errors.New("session: session is required")
	}
	if m.environment == nil {
		return acp.StartOpts{}, errors.New("session: environment registry is required")
	}

	resolvedEnv := normalizeResolvedEnvironment(spec.workspace.Environment)
	provider, err := m.environment.Provider(resolvedEnv.Backend)
	if err != nil {
		return acp.StartOpts{}, fmt.Errorf("session: resolve environment provider %q: %w", resolvedEnv.Backend, err)
	}

	environmentID, meta, err := m.initializeEnvironmentMetaForStart(spec, session, resolvedEnv)
	if err != nil {
		return acp.StartOpts{}, err
	}

	req := envpkg.PrepareRequest{
		SessionID:           session.ID,
		WorkspaceID:         session.WorkspaceID,
		EnvironmentID:       environmentID,
		InstanceID:          meta.InstanceID,
		LocalRootDir:        spec.workspace.RootDir,
		LocalAdditionalDirs: append([]string(nil), spec.workspace.AdditionalDirs...),
		Environment:         resolvedEnv,
		AgentCommand:        opts.Command,
		AgentEnv:            environmentAgentEnv(opts.Env, resolvedEnv),
		Permissions:         string(opts.Permissions),
		ResumeACPState:      opts.ResumeSessionID,
		ProviderState:       cloneRawMessage(meta.ProviderState),
	}
	req, err = m.dispatchEnvironmentPrepare(ctx, session, req)
	if err != nil {
		return acp.StartOpts{}, err
	}

	prepared, prepareErr := m.callEnvironmentPrepare(ctx, provider, req, meta)
	if prepareErr != nil {
		return acp.StartOpts{}, prepareErr
	}

	state, err := normalizePreparedEnvironmentState(prepared, meta, resolvedEnv)
	if err != nil {
		return acp.StartOpts{}, err
	}
	meta = sessionEnvironmentMetaFromState(state, environmentStatePrepared)
	session.setEnvironment(meta, m.now())
	if err := m.writeMeta(session); err != nil {
		return acp.StartOpts{}, err
	}

	if err := m.syncEnvironmentToRuntime(ctx, provider, session, state, meta); err != nil {
		return acp.StartOpts{}, err
	}
	meta = cloneSessionEnvironmentMeta(session.Info().Environment)
	if err := m.dispatchEnvironmentReady(ctx, session, state, meta); err != nil {
		return acp.StartOpts{}, err
	}

	return environmentStartOpts(opts, prepared, state), nil
}

func (m *Manager) initializeEnvironmentMetaForStart(
	spec *sessionStartSpec,
	session *Session,
	resolvedEnv envpkg.Resolved,
) (string, *store.SessionEnvironmentMeta, error) {
	environmentID := strings.TrimSpace(spec.environmentID)
	if environmentID == "" {
		environmentID = sessionEnvironmentID(spec.environment)
	}
	if environmentID == "" {
		environmentID = strings.TrimSpace(m.newEnvironmentID())
	}
	if environmentID == "" {
		return "", nil, errors.New("session: environment id generator returned empty id")
	}
	spec.environmentID = environmentID

	meta := initialSessionEnvironmentMeta(environmentID, resolvedEnv, spec.environment)
	session.setEnvironment(meta, m.now())
	if err := m.writeMeta(session); err != nil {
		return "", nil, err
	}
	return environmentID, meta, nil
}

func (m *Manager) callEnvironmentPrepare(
	ctx context.Context,
	provider envpkg.Provider,
	req envpkg.PrepareRequest,
	meta *store.SessionEnvironmentMeta,
) (envpkg.Prepared, error) {
	started := time.Now()
	event := environmentEventFromMeta(meta, req.SessionID, req.WorkspaceID, environmentEventPrepareStart, "")
	m.logEnvironmentLifecycle(event)

	prepared, err := provider.Prepare(ctx, req)
	duration := time.Since(started)
	if err != nil {
		errorEvent := environmentEventFromMeta(meta, req.SessionID, req.WorkspaceID, environmentEventPrepareError, "")
		errorEvent.Duration = duration
		attachEnvironmentError(&errorEvent, err)
		m.logEnvironmentLifecycle(errorEvent)
		return envpkg.Prepared{}, fmt.Errorf(
			"session: prepare environment %q for %q: %w",
			req.EnvironmentID,
			req.SessionID,
			err,
		)
	}

	completeMeta := sessionEnvironmentMetaFromState(prepared.State, environmentStatePrepared)
	if completeMeta.EnvironmentID == "" {
		completeMeta.EnvironmentID = req.EnvironmentID
	}
	if completeMeta.Backend == "" {
		completeMeta.Backend = string(provider.Backend())
	}
	if completeMeta.Profile == "" {
		completeMeta.Profile = req.Environment.Profile
	}
	completeEvent := environmentEventFromMeta(
		completeMeta,
		req.SessionID,
		req.WorkspaceID,
		environmentEventPrepareComplete,
		"",
	)
	completeEvent.Duration = duration
	m.logEnvironmentLifecycle(completeEvent)
	return prepared, nil
}

func (m *Manager) dispatchEnvironmentPrepare(
	ctx context.Context,
	session *Session,
	req envpkg.PrepareRequest,
) (envpkg.PrepareRequest, error) {
	payload := hookspkg.EnvironmentPreparePayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookEnvironmentPrepare,
			Timestamp: m.now(),
		},
		SessionContext:      hookSessionContext(session),
		EnvironmentID:       strings.TrimSpace(req.EnvironmentID),
		Backend:             string(req.Environment.Backend),
		Profile:             environmentProfilePayload(req.Environment),
		LocalRootDir:        strings.TrimSpace(req.LocalRootDir),
		LocalAdditionalDirs: append([]string(nil), req.LocalAdditionalDirs...),
		AgentCommand:        strings.TrimSpace(req.AgentCommand),
		AgentEnv:            append([]string(nil), req.AgentEnv...),
		Permissions:         strings.TrimSpace(req.Permissions),
		ResumeACPState:      strings.TrimSpace(req.ResumeACPState),
	}
	patched, err := m.hooks.environment().DispatchEnvironmentPrepare(ctx, payload)
	if err != nil {
		return req, err
	}
	if patched.Denied {
		if reason := strings.TrimSpace(patched.DenyReason); reason != "" {
			return req, fmt.Errorf("session: environment prepare denied: %s", reason)
		}
		return req, errors.New("session: environment prepare denied")
	}
	if len(patched.EnvOverrides) == 0 {
		return req, nil
	}

	req.Environment.Env = mergeEnvironmentEnv(req.Environment.Env, patched.EnvOverrides)
	req.AgentEnv = applyEnvironmentEnvOverrides(req.AgentEnv, patched.EnvOverrides)
	return req, nil
}

func (m *Manager) dispatchEnvironmentReady(
	ctx context.Context,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionEnvironmentMeta,
) error {
	payload := hookspkg.EnvironmentReadyPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookEnvironmentReady,
			Timestamp: m.now(),
		},
		SessionContext:        hookSessionContext(session),
		EnvironmentID:         strings.TrimSpace(state.EnvironmentID),
		Backend:               string(state.Backend),
		Profile:               strings.TrimSpace(state.Profile),
		InstanceID:            strings.TrimSpace(state.InstanceID),
		RuntimeRootDir:        strings.TrimSpace(state.RuntimeRootDir),
		RuntimeAdditionalDirs: append([]string(nil), state.RuntimeAdditionalDirs...),
	}
	if meta != nil {
		if payload.EnvironmentID == "" {
			payload.EnvironmentID = strings.TrimSpace(meta.EnvironmentID)
		}
		if payload.Backend == "" {
			payload.Backend = strings.TrimSpace(meta.Backend)
		}
		if payload.Profile == "" {
			payload.Profile = strings.TrimSpace(meta.Profile)
		}
		if payload.InstanceID == "" {
			payload.InstanceID = strings.TrimSpace(meta.InstanceID)
		}
		if payload.RuntimeRootDir == "" {
			payload.RuntimeRootDir = strings.TrimSpace(meta.RuntimeRootDir)
		}
	}
	_, err := m.hooks.environment().DispatchEnvironmentReady(ctx, payload)
	return err
}

func (m *Manager) dispatchEnvironmentSyncBefore(
	ctx context.Context,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionEnvironmentMeta,
	direction envpkg.SyncDirection,
	reason envpkg.SyncReason,
) (hookspkg.EnvironmentSyncBeforePayload, error) {
	payload := hookspkg.EnvironmentSyncBeforePayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookEnvironmentSyncBefore,
			Timestamp: m.now(),
		},
		SessionContext: hookSessionContext(session),
		EnvironmentID:  strings.TrimSpace(state.EnvironmentID),
		Backend:        string(state.Backend),
		Profile:        strings.TrimSpace(state.Profile),
		InstanceID:     strings.TrimSpace(state.InstanceID),
		RuntimeRootDir: strings.TrimSpace(state.RuntimeRootDir),
		Direction:      string(direction),
		Reason:         string(reason),
	}
	if m.hooks.hasEnvironmentHooks() {
		payload.FileCount = m.environmentSyncFileCount(session, direction)
	}
	applyEnvironmentMetaFallbacks(&payload.EnvironmentID, &payload.Backend, &payload.Profile, &payload.InstanceID, meta)
	return m.hooks.environment().DispatchEnvironmentSyncBefore(ctx, payload)
}

func (m *Manager) dispatchEnvironmentSyncAfter(
	ctx context.Context,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionEnvironmentMeta,
	direction envpkg.SyncDirection,
	reason envpkg.SyncReason,
	duration time.Duration,
	result envpkg.SyncResult,
	errorsList []string,
) error {
	payload := hookspkg.EnvironmentSyncAfterPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookEnvironmentSyncAfter,
			Timestamp: m.now(),
		},
		SessionContext:   hookSessionContext(session),
		EnvironmentID:    strings.TrimSpace(state.EnvironmentID),
		Backend:          string(state.Backend),
		Profile:          strings.TrimSpace(state.Profile),
		InstanceID:       strings.TrimSpace(state.InstanceID),
		RuntimeRootDir:   strings.TrimSpace(state.RuntimeRootDir),
		Direction:        string(direction),
		Reason:           string(reason),
		FilesSynced:      result.FilesSynced,
		BytesTransferred: result.BytesTransferred,
		DurationMS:       duration.Milliseconds(),
		Errors:           append([]string(nil), errorsList...),
	}
	applyEnvironmentMetaFallbacks(&payload.EnvironmentID, &payload.Backend, &payload.Profile, &payload.InstanceID, meta)
	_, err := m.hooks.environment().DispatchEnvironmentSyncAfter(ctx, payload)
	return err
}

func (m *Manager) dispatchEnvironmentStop(
	ctx context.Context,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionEnvironmentMeta,
	reason envpkg.SyncReason,
	willDestroy bool,
) (hookspkg.EnvironmentStopPayload, error) {
	stopReason := string(reason)
	if info := session.Info(); info != nil && strings.TrimSpace(string(info.StopReason)) != "" {
		stopReason = string(info.StopReason)
	}
	payload := hookspkg.EnvironmentStopPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookEnvironmentStop,
			Timestamp: m.now(),
		},
		SessionContext: hookSessionContext(session),
		EnvironmentID:  strings.TrimSpace(state.EnvironmentID),
		Backend:        string(state.Backend),
		Profile:        strings.TrimSpace(state.Profile),
		InstanceID:     strings.TrimSpace(state.InstanceID),
		RuntimeRootDir: strings.TrimSpace(state.RuntimeRootDir),
		StopReason:     strings.TrimSpace(stopReason),
		WillDestroy:    willDestroy,
	}
	applyEnvironmentMetaFallbacks(&payload.EnvironmentID, &payload.Backend, &payload.Profile, &payload.InstanceID, meta)
	return m.hooks.environment().DispatchEnvironmentStop(ctx, payload)
}

func (m *Manager) syncEnvironmentToRuntime(
	ctx context.Context,
	provider envpkg.Provider,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionEnvironmentMeta,
) error {
	return m.syncEnvironmentRuntime(
		ctx,
		session,
		state,
		meta,
		envpkg.SyncDirectionToRuntime,
		envpkg.SyncReasonStart,
		provider.SyncToRuntime,
	)
}

type environmentSyncRunner func(context.Context, envpkg.SessionState, envpkg.SyncOptions) (envpkg.SyncResult, error)

func (m *Manager) syncEnvironmentRuntime(
	ctx context.Context,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionEnvironmentMeta,
	direction envpkg.SyncDirection,
	reason envpkg.SyncReason,
	run environmentSyncRunner,
) error {
	started := time.Now()
	m.logEnvironmentLifecycle(environmentEventFromMeta(
		meta,
		session.ID,
		session.WorkspaceID,
		environmentEventSyncStart,
		string(reason),
	))

	before, err := m.dispatchEnvironmentSyncBefore(ctx, session, state, meta, direction, reason)
	if err != nil {
		return err
	}
	if before.Denied {
		return nil
	}

	result, err := run(ctx, state, envpkg.SyncOptions{
		Reason:          reason,
		ExcludePatterns: append([]string(nil), before.ExcludePatterns...),
	})
	duration := time.Since(started)
	errorsList := syncResultErrors(result, err)
	now := m.now()
	meta = cloneSessionEnvironmentMeta(meta)
	meta.LastSyncAt = &now
	if err != nil {
		return m.finishEnvironmentSyncError(ctx, session, state, meta, direction, reason, environmentSyncOutcome{
			result:     result,
			duration:   duration,
			errorsList: errorsList,
			syncTime:   now,
			err:        err,
		})
	}
	return m.finishEnvironmentSyncSuccess(ctx, session, state, meta, direction, reason, environmentSyncOutcome{
		result:     result,
		duration:   duration,
		errorsList: errorsList,
		syncTime:   now,
	})
}

type environmentSyncOutcome struct {
	result     envpkg.SyncResult
	duration   time.Duration
	errorsList []string
	syncTime   time.Time
	err        error
}

func (m *Manager) finishEnvironmentSyncSuccess(
	ctx context.Context,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionEnvironmentMeta,
	direction envpkg.SyncDirection,
	reason envpkg.SyncReason,
	outcome environmentSyncOutcome,
) error {
	meta.LastSyncError = ""
	session.setEnvironment(meta, outcome.syncTime)
	if err := m.writeMeta(session); err != nil {
		return err
	}
	if err := m.dispatchEnvironmentSyncAfter(
		ctx,
		session,
		state,
		meta,
		direction,
		reason,
		outcome.duration,
		outcome.result,
		outcome.errorsList,
	); err != nil {
		return err
	}
	completeEvent := environmentEventFromMeta(
		meta,
		session.ID,
		session.WorkspaceID,
		environmentEventSyncComplete,
		string(reason),
	)
	completeEvent.Duration = outcome.duration
	m.logEnvironmentLifecycle(completeEvent)
	return nil
}

func (m *Manager) finishEnvironmentSyncError(
	ctx context.Context,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionEnvironmentMeta,
	direction envpkg.SyncDirection,
	reason envpkg.SyncReason,
	outcome environmentSyncOutcome,
) error {
	err := syncEnvironmentWriteError(m, session, meta, outcome)
	errorsList := syncResultErrors(outcome.result, err)
	if afterErr := m.dispatchEnvironmentSyncAfter(
		ctx,
		session,
		state,
		meta,
		direction,
		reason,
		outcome.duration,
		outcome.result,
		errorsList,
	); afterErr != nil {
		m.warnHookDispatch(ctx, session, hookspkg.HookEnvironmentSyncAfter, afterErr)
	}

	errorEvent := environmentEventFromMeta(
		meta,
		session.ID,
		session.WorkspaceID,
		environmentEventSyncError,
		string(reason),
	)
	errorEvent.Duration = outcome.duration
	attachEnvironmentError(&errorEvent, err)
	m.logEnvironmentLifecycle(errorEvent)
	return fmt.Errorf(
		"session: sync environment %q %s runtime for %q: %w",
		state.EnvironmentID,
		syncDirectionPreposition(direction),
		session.ID,
		err,
	)
}

func syncEnvironmentWriteError(
	m *Manager,
	session *Session,
	meta *store.SessionEnvironmentMeta,
	outcome environmentSyncOutcome,
) error {
	meta.LastSyncError = outcome.err.Error()
	session.setEnvironment(meta, outcome.syncTime)
	if writeErr := m.writeMeta(session); writeErr != nil {
		return errors.Join(outcome.err, writeErr)
	}
	return outcome.err
}

func syncDirectionPreposition(direction envpkg.SyncDirection) string {
	if direction == envpkg.SyncDirectionFromRuntime {
		return "from"
	}
	return "to"
}

func (m *Manager) finalizeEnvironment(
	ctx context.Context,
	session *Session,
	reason envpkg.SyncReason,
) error {
	if session == nil {
		return nil
	}
	meta := cloneSessionEnvironmentMeta(session.Info().Environment)
	if meta == nil {
		return nil
	}
	if m.environment == nil {
		return errors.New("session: environment registry is required")
	}

	provider, err := m.environment.Provider(envpkg.Backend(strings.TrimSpace(meta.Backend)))
	if err != nil {
		return fmt.Errorf("session: resolve environment provider %q: %w", meta.Backend, err)
	}

	state := sessionEnvironmentStateFromMeta(meta)
	var errs []error
	if syncErr := m.syncEnvironmentFromRuntime(ctx, provider, session, state, meta, reason); syncErr != nil {
		if reason == envpkg.SyncReasonCrash {
			m.sessionLogger(session).Warn("session: environment crash sync failed", "error", syncErr)
		} else {
			errs = append(errs, syncErr)
		}
		meta = cloneSessionEnvironmentMeta(session.Info().Environment)
		state = sessionEnvironmentStateFromMeta(meta)
	}

	shouldDestroy := session.environmentShouldDestroy()
	stopPayload, stopErr := m.dispatchEnvironmentStop(ctx, session, state, meta, reason, shouldDestroy)
	if stopErr != nil {
		errs = append(errs, stopErr)
		shouldDestroy = false
	}
	if stopPayload.Denied {
		shouldDestroy = false
	}

	if shouldDestroy {
		if destroyErr := m.destroyEnvironment(ctx, provider, session, state); destroyErr != nil {
			errs = append(errs, destroyErr)
		}
	} else {
		now := m.now()
		meta = cloneSessionEnvironmentMeta(session.Info().Environment)
		if meta != nil {
			meta.State = environmentStateStopped
			session.setEnvironment(meta, now)
			if err := m.writeMeta(session); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}

func (m *Manager) syncEnvironmentFromRuntime(
	ctx context.Context,
	provider envpkg.Provider,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionEnvironmentMeta,
	reason envpkg.SyncReason,
) error {
	return m.syncEnvironmentRuntime(
		ctx,
		session,
		state,
		meta,
		envpkg.SyncDirectionFromRuntime,
		reason,
		provider.SyncFromRuntime,
	)
}

func (m *Manager) destroyEnvironment(
	ctx context.Context,
	provider envpkg.Provider,
	session *Session,
	state envpkg.SessionState,
) error {
	meta := cloneSessionEnvironmentMeta(session.Info().Environment)
	started := time.Now()
	startEvent := environmentEventFromMeta(meta, session.ID, session.WorkspaceID, environmentEventDestroyStart, "")
	m.logEnvironmentLifecycle(startEvent)

	err := provider.Destroy(ctx, state)
	duration := time.Since(started)
	if err != nil {
		errorEvent := environmentEventFromMeta(meta, session.ID, session.WorkspaceID, environmentEventDestroyError, "")
		errorEvent.Duration = duration
		attachEnvironmentError(&errorEvent, err)
		m.logEnvironmentLifecycle(errorEvent)
		return fmt.Errorf("session: destroy environment %q for %q: %w", state.EnvironmentID, session.ID, err)
	}

	now := m.now()
	meta = cloneSessionEnvironmentMeta(meta)
	if meta != nil {
		meta.State = environmentStateDestroyed
		session.setEnvironment(meta, now)
		if err := m.writeMeta(session); err != nil {
			return err
		}
	}
	completeEvent := environmentEventFromMeta(
		meta,
		session.ID,
		session.WorkspaceID,
		environmentEventDestroyComplete,
		"",
	)
	completeEvent.Duration = duration
	m.logEnvironmentLifecycle(completeEvent)
	return nil
}

func (m *Manager) logEnvironmentTransport(session *Session, eventName string, err error, duration time.Duration) {
	if session == nil {
		return
	}
	meta := cloneSessionEnvironmentMeta(session.Info().Environment)
	event := environmentEventFromMeta(meta, session.ID, session.WorkspaceID, eventName, "")
	event.Duration = duration
	if err != nil {
		attachEnvironmentError(&event, err)
	}
	m.logEnvironmentLifecycle(event)
}

func (m *Manager) logEnvironmentLifecycle(event EnvironmentLifecycleEvent) {
	event.Name = strings.TrimSpace(event.Name)
	if event.Name == "" {
		return
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = m.now()
	}
	if event.Span == "" {
		event.Span = environmentSpanForEvent(event.Name, event.Reason)
	}
	logger := m.logger
	if logger == nil {
		logger = slog.Default()
	}

	args := []any{
		"backend", strings.TrimSpace(event.Backend),
		"profile", strings.TrimSpace(event.Profile),
		"environment_id", strings.TrimSpace(event.EnvironmentID),
		"instance_id", strings.TrimSpace(event.InstanceID),
		"workspace_id", strings.TrimSpace(event.WorkspaceID),
		"session_id", strings.TrimSpace(event.SessionID),
		"duration_ms", event.Duration.Milliseconds(),
	}
	if strings.TrimSpace(event.Reason) != "" {
		args = append(args, "reason", strings.TrimSpace(event.Reason))
	}
	if strings.TrimSpace(event.ErrorKind) != "" {
		args = append(args, "error_kind", strings.TrimSpace(event.ErrorKind))
	}
	if strings.TrimSpace(event.Error) != "" {
		args = append(args, "error", strings.TrimSpace(event.Error))
	}
	if strings.Contains(event.Name, ".error") {
		logger.Warn(event.Name, args...)
	} else {
		logger.Info(event.Name, args...)
	}

	if notifier, ok := m.notifier.(EnvironmentLifecycleNotifier); ok {
		notifier.OnEnvironmentLifecycleEvent(m.lifecycleCtx, event)
	}
}

func environmentStartOpts(
	opts acp.StartOpts,
	prepared envpkg.Prepared,
	state envpkg.SessionState,
) acp.StartOpts {
	next := opts
	if command := strings.TrimSpace(prepared.Launch.Command); command != "" {
		next.Command = command
	}
	if prepared.Launch.Env != nil {
		next.Env = append([]string(nil), prepared.Launch.Env...)
	}
	next.Cwd = strings.TrimSpace(prepared.RuntimeRootDir)
	if next.Cwd == "" {
		next.Cwd = strings.TrimSpace(state.RuntimeRootDir)
	}
	next.AdditionalDirs = append([]string(nil), prepared.RuntimeAdditionalDirs...)
	if next.AdditionalDirs == nil {
		next.AdditionalDirs = append([]string(nil), state.RuntimeAdditionalDirs...)
	}
	next.Launcher = prepared.Launcher
	next.ToolHost = prepared.ToolHost
	return next
}

func initialSessionEnvironmentMeta(
	environmentID string,
	resolved envpkg.Resolved,
	previous *store.SessionEnvironmentMeta,
) *store.SessionEnvironmentMeta {
	meta := cloneSessionEnvironmentMeta(previous)
	if meta == nil {
		meta = &store.SessionEnvironmentMeta{}
	}
	meta.EnvironmentID = strings.TrimSpace(environmentID)
	meta.Backend = string(resolved.Backend)
	meta.Profile = strings.TrimSpace(resolved.Profile)
	meta.State = environmentStateCreating
	return meta
}

func normalizePreparedEnvironmentState(
	prepared envpkg.Prepared,
	meta *store.SessionEnvironmentMeta,
	resolved envpkg.Resolved,
) (envpkg.SessionState, error) {
	state := prepared.State
	if strings.TrimSpace(state.EnvironmentID) == "" {
		state.EnvironmentID = strings.TrimSpace(meta.EnvironmentID)
	}
	if state.Backend == "" {
		state.Backend = resolved.Backend
	}
	if strings.TrimSpace(state.Profile) == "" {
		state.Profile = strings.TrimSpace(resolved.Profile)
	}
	if strings.TrimSpace(state.State) == "" {
		state.State = environmentStatePrepared
	}
	if strings.TrimSpace(state.RuntimeRootDir) == "" {
		state.RuntimeRootDir = strings.TrimSpace(prepared.RuntimeRootDir)
	}
	if strings.TrimSpace(state.RuntimeRootDir) == "" {
		state.RuntimeRootDir = strings.TrimSpace(prepared.Launch.Cwd)
	}
	if len(state.RuntimeAdditionalDirs) == 0 {
		state.RuntimeAdditionalDirs = append([]string(nil), prepared.RuntimeAdditionalDirs...)
	}
	if state.ProviderState == nil {
		state.ProviderState = cloneRawMessage(meta.ProviderState)
	}
	if strings.TrimSpace(state.EnvironmentID) == "" {
		return envpkg.SessionState{}, errors.New("session: prepared environment id is required")
	}
	if !state.Backend.Valid() {
		return envpkg.SessionState{}, fmt.Errorf("session: prepared environment backend %q is invalid", state.Backend)
	}
	if strings.TrimSpace(state.RuntimeRootDir) == "" {
		return envpkg.SessionState{}, errors.New("session: prepared runtime root dir is required")
	}
	return state, nil
}

func sessionEnvironmentMetaFromState(
	state envpkg.SessionState,
	fallbackState string,
) *store.SessionEnvironmentMeta {
	if strings.TrimSpace(state.EnvironmentID) == "" && state.Backend == "" {
		return nil
	}
	sessionState := strings.TrimSpace(state.State)
	if sessionState == "" {
		sessionState = fallbackState
	}
	return &store.SessionEnvironmentMeta{
		EnvironmentID:         strings.TrimSpace(state.EnvironmentID),
		Backend:               string(state.Backend),
		Profile:               strings.TrimSpace(state.Profile),
		State:                 sessionState,
		InstanceID:            strings.TrimSpace(state.InstanceID),
		RuntimeRootDir:        strings.TrimSpace(state.RuntimeRootDir),
		RuntimeAdditionalDirs: append([]string(nil), state.RuntimeAdditionalDirs...),
		ProviderState:         cloneRawMessage(state.ProviderState),
		SSHAccessExpiresAt:    cloneTimePointer(state.SSHAccessExpiresAt),
	}
}

func sessionEnvironmentStateFromMeta(meta *store.SessionEnvironmentMeta) envpkg.SessionState {
	if meta == nil {
		return envpkg.SessionState{}
	}
	return envpkg.SessionState{
		EnvironmentID:         strings.TrimSpace(meta.EnvironmentID),
		Backend:               envpkg.Backend(strings.TrimSpace(meta.Backend)),
		Profile:               strings.TrimSpace(meta.Profile),
		State:                 strings.TrimSpace(meta.State),
		InstanceID:            strings.TrimSpace(meta.InstanceID),
		RuntimeRootDir:        strings.TrimSpace(meta.RuntimeRootDir),
		RuntimeAdditionalDirs: append([]string(nil), meta.RuntimeAdditionalDirs...),
		ProviderState:         cloneRawMessage(meta.ProviderState),
		SSHAccessExpiresAt:    cloneTimePointer(meta.SSHAccessExpiresAt),
	}
}

func sessionEnvironmentID(meta *store.SessionEnvironmentMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.EnvironmentID)
}

func normalizeResolvedEnvironment(resolved envpkg.Resolved) envpkg.Resolved {
	if !resolved.Backend.Valid() {
		resolved.Backend = envpkg.BackendLocal
	}
	if strings.TrimSpace(resolved.Profile) == "" {
		resolved.Profile = string(resolved.Backend)
	}
	return resolved
}

func environmentAgentEnv(base []string, resolved envpkg.Resolved) []string {
	env := append([]string(nil), base...)
	if len(resolved.Env) == 0 {
		return env
	}
	keys := make([]string, 0, len(resolved.Env))
	for key := range resolved.Env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		env = setSessionStartEnvValue(env, key, resolved.Env[key])
	}
	return env
}

func environmentProfilePayload(resolved envpkg.Resolved) hookspkg.EnvironmentProfilePayload {
	return hookspkg.EnvironmentProfilePayload{
		Profile:        strings.TrimSpace(resolved.Profile),
		Backend:        string(resolved.Backend),
		SyncMode:       string(resolved.SyncMode),
		Persistence:    string(resolved.Persistence),
		RuntimeRootDir: strings.TrimSpace(resolved.RuntimeRootDir),
		DestroyOnStop:  resolved.DestroyOnStop,
		Env:            mergeEnvironmentEnv(nil, resolved.Env),
	}
}

func mergeEnvironmentEnv(base map[string]string, overrides map[string]string) map[string]string {
	if len(base) == 0 && len(overrides) == 0 {
		return nil
	}
	merged := make(map[string]string, len(base)+len(overrides))
	for key, value := range base {
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			merged[trimmed] = value
		}
	}
	for key, value := range overrides {
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			merged[trimmed] = value
		}
	}
	return merged
}

func applyEnvironmentEnvOverrides(env []string, overrides map[string]string) []string {
	next := append([]string(nil), env...)
	keys := make([]string, 0, len(overrides))
	values := make(map[string]string, len(overrides))
	for key, value := range overrides {
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			keys = append(keys, trimmed)
			values[trimmed] = value
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		next = setSessionStartEnvValue(next, key, values[key])
	}
	return next
}

func applyEnvironmentMetaFallbacks(
	environmentID *string,
	backend *string,
	profile *string,
	instanceID *string,
	meta *store.SessionEnvironmentMeta,
) {
	if meta == nil {
		return
	}
	if environmentID != nil && strings.TrimSpace(*environmentID) == "" {
		*environmentID = strings.TrimSpace(meta.EnvironmentID)
	}
	if backend != nil && strings.TrimSpace(*backend) == "" {
		*backend = strings.TrimSpace(meta.Backend)
	}
	if profile != nil && strings.TrimSpace(*profile) == "" {
		*profile = strings.TrimSpace(meta.Profile)
	}
	if instanceID != nil && strings.TrimSpace(*instanceID) == "" {
		*instanceID = strings.TrimSpace(meta.InstanceID)
	}
}

func (m *Manager) environmentSyncFileCount(session *Session, direction envpkg.SyncDirection) int {
	if direction != envpkg.SyncDirectionToRuntime || session == nil {
		return 0
	}
	info := session.Info()
	if info == nil {
		return 0
	}
	root := strings.TrimSpace(info.Workspace)
	if root == "" {
		return 0
	}
	count, err := countRegularFiles(root)
	if err != nil {
		m.sessionLogger(session).Warn("session: count environment sync files failed", "root", root, "error", err)
		return 0
	}
	return count
}

func countRegularFiles(root string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(_ string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry == nil || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			count++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

func syncResultErrors(result envpkg.SyncResult, err error) []string {
	if err == nil && len(result.Errors) == 0 {
		return nil
	}
	errorsList := make([]string, 0, len(result.Errors)+1)
	seen := make(map[string]struct{}, len(result.Errors)+1)
	for _, item := range result.Errors {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		errorsList = append(errorsList, trimmed)
	}
	if err != nil {
		trimmed := strings.TrimSpace(err.Error())
		if trimmed != "" {
			if _, ok := seen[trimmed]; !ok {
				errorsList = append(errorsList, trimmed)
			}
		}
	}
	return errorsList
}

func environmentEventFromMeta(
	meta *store.SessionEnvironmentMeta,
	sessionID string,
	workspaceID string,
	name string,
	reason string,
) EnvironmentLifecycleEvent {
	event := EnvironmentLifecycleEvent{
		Name:        strings.TrimSpace(name),
		SessionID:   strings.TrimSpace(sessionID),
		WorkspaceID: strings.TrimSpace(workspaceID),
		Reason:      strings.TrimSpace(reason),
	}
	if meta != nil {
		event.EnvironmentID = strings.TrimSpace(meta.EnvironmentID)
		event.Backend = strings.TrimSpace(meta.Backend)
		event.Profile = strings.TrimSpace(meta.Profile)
		event.InstanceID = strings.TrimSpace(meta.InstanceID)
	}
	return event
}

func attachEnvironmentError(event *EnvironmentLifecycleEvent, err error) {
	if event == nil || err == nil {
		return
	}
	event.Error = err.Error()
	event.ErrorKind = environmentErrorKind(err)
}

func environmentErrorKind(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "context_deadline_exceeded"
	default:
		return fmt.Sprintf("%T", err)
	}
}

func environmentSpanForEvent(name string, reason string) string {
	switch name {
	case environmentEventPrepareStart, environmentEventPrepareComplete, environmentEventPrepareError:
		return "environment.prepare"
	case environmentEventDestroyStart, environmentEventDestroyComplete, environmentEventDestroyError:
		return "environment.destroy"
	case environmentEventSyncStart, environmentEventSyncComplete, environmentEventSyncError:
		if reason == string(envpkg.SyncReasonStart) || reason == string(envpkg.SyncReasonTurn) {
			return "environment.sync.to_runtime"
		}
		return "environment.sync.from_runtime"
	default:
		return strings.TrimSpace(name)
	}
}

func cloneSessionEnvironmentMeta(meta *store.SessionEnvironmentMeta) *store.SessionEnvironmentMeta {
	if meta == nil {
		return nil
	}
	cloned := *meta
	cloned.RuntimeAdditionalDirs = append([]string(nil), meta.RuntimeAdditionalDirs...)
	cloned.ProviderState = cloneRawMessage(meta.ProviderState)
	cloned.SSHAccessExpiresAt = cloneTimePointer(meta.SSHAccessExpiresAt)
	cloned.LastSyncAt = cloneTimePointer(meta.LastSyncAt)
	return &cloned
}

func cloneRawMessage(value json.RawMessage) json.RawMessage {
	if value == nil {
		return nil
	}
	cloned := make(json.RawMessage, len(value))
	copy(cloned, value)
	return cloned
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
