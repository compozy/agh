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
	"github.com/pedronauck/agh/internal/diagnostics"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	envpkg "github.com/pedronauck/agh/internal/sandbox"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/vault"
)

const (
	sandboxSandboxPreparePath = "sandbox.prepare"
)

const (
	sandboxStateCreating  = "creating"
	sandboxStatePrepared  = "prepared"
	sandboxStateStopped   = "stopped"
	sandboxStateDestroyed = "destroyed"

	sandboxEventPrepareStart        = "sandbox.prepare.start"
	sandboxEventPrepareComplete     = "sandbox.prepare.complete"
	sandboxEventPrepareError        = "sandbox.prepare.error"
	sandboxEventSyncStart           = "sandbox.sync.start"
	sandboxEventSyncComplete        = "sandbox.sync.complete"
	sandboxEventSyncError           = "sandbox.sync.error"
	sandboxEventTransportConnect    = "sandbox.transport.connect"
	sandboxEventTransportDisconnect = "sandbox.transport.disconnect"
	sandboxEventTransportError      = "sandbox.transport.error"
	sandboxEventDestroyStart        = "sandbox.destroy.start"
	sandboxEventDestroyComplete     = "sandbox.destroy.complete"
	sandboxEventDestroyError        = "sandbox.destroy.error"
)

// SandboxLifecycleEvent reports provider lifecycle timing to optional observers.
type SandboxLifecycleEvent struct {
	Name        string
	Span        string
	SessionID   string
	WorkspaceID string
	SandboxID   string
	Backend     string
	Profile     string
	InstanceID  string
	Reason      string
	Duration    time.Duration
	ErrorKind   string
	Error       string
	Timestamp   time.Time
}

// SandboxLifecycleNotifier is an optional notifier extension for sandbox lifecycle spans.
type SandboxLifecycleNotifier interface {
	OnSandboxLifecycleEvent(context.Context, SandboxLifecycleEvent)
}

func (m *Manager) prepareSandboxForStart(
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
	if spec.sandboxDisabled {
		return opts, nil
	}
	if m.sandbox == nil {
		return acp.StartOpts{}, errors.New("session: sandbox registry is required")
	}

	resolvedEnv := normalizeResolvedSandbox(spec.workspace.Sandbox)
	provider, err := m.sandbox.Provider(resolvedEnv.Backend)
	if err != nil {
		return acp.StartOpts{}, fmt.Errorf("session: resolve sandbox provider %q: %w", resolvedEnv.Backend, err)
	}

	sandboxID, meta, err := m.initializeSandboxMetaForStart(spec, session, resolvedEnv)
	if err != nil {
		return acp.StartOpts{}, err
	}

	req := envpkg.PrepareRequest{
		SessionID:           session.ID,
		WorkspaceID:         session.WorkspaceID,
		SandboxID:           sandboxID,
		InstanceID:          meta.InstanceID,
		LocalRootDir:        spec.workspace.RootDir,
		LocalAdditionalDirs: append([]string(nil), spec.workspace.AdditionalDirs...),
		Sandbox:             resolvedEnv,
		AgentCommand:        opts.Command,
		AgentEnv:            sandboxAgentEnv(opts.Env, resolvedEnv),
		Permissions:         string(opts.Permissions),
		ResumeACPState:      opts.ResumeSessionID,
		ProviderState:       cloneRawMessage(meta.ProviderState),
	}
	req, err = m.dispatchSandboxPrepare(ctx, session, req)
	if err != nil {
		return acp.StartOpts{}, err
	}
	req, err = m.resolveSandboxSecretEnv(ctx, session, req)
	if err != nil {
		return acp.StartOpts{}, err
	}

	prepared, prepareErr := m.callSandboxPrepare(ctx, provider, req, meta)
	if prepareErr != nil {
		return acp.StartOpts{}, prepareErr
	}

	state, err := normalizePreparedSandboxState(prepared, meta, resolvedEnv)
	if err != nil {
		return acp.StartOpts{}, err
	}
	meta = sessionSandboxMetaFromState(state, sandboxStatePrepared)
	session.setSandbox(meta, m.now())
	if err := m.writeMeta(session); err != nil {
		return acp.StartOpts{}, err
	}

	if err := m.syncSandboxToRuntime(ctx, provider, session, state, meta); err != nil {
		return acp.StartOpts{}, err
	}
	meta = cloneSessionSandboxMeta(session.Info().Sandbox)
	if err := m.dispatchSandboxReady(ctx, session, state, meta); err != nil {
		return acp.StartOpts{}, err
	}

	return sandboxStartOpts(opts, prepared, state), nil
}

func (m *Manager) initializeSandboxMetaForStart(
	spec *sessionStartSpec,
	session *Session,
	resolvedEnv envpkg.Resolved,
) (string, *store.SessionSandboxMeta, error) {
	sandboxID := strings.TrimSpace(spec.sandboxID)
	if sandboxID == "" {
		sandboxID = sessionSandboxID(spec.sandbox)
	}
	if sandboxID == "" {
		sandboxID = strings.TrimSpace(m.newSandboxID())
	}
	if sandboxID == "" {
		return "", nil, errors.New("session: sandbox id generator returned empty id")
	}
	spec.sandboxID = sandboxID

	meta := initialSessionSandboxMeta(sandboxID, resolvedEnv, spec.sandbox)
	session.setSandbox(meta, m.now())
	if err := m.writeMeta(session); err != nil {
		return "", nil, err
	}
	return sandboxID, meta, nil
}

func (m *Manager) callSandboxPrepare(
	ctx context.Context,
	provider envpkg.Provider,
	req envpkg.PrepareRequest,
	meta *store.SessionSandboxMeta,
) (envpkg.Prepared, error) {
	started := time.Now()
	event := sandboxEventFromMeta(meta, req.SessionID, req.WorkspaceID, sandboxEventPrepareStart, "")
	m.logSandboxLifecycle(event)

	prepared, err := provider.Prepare(ctx, req)
	duration := time.Since(started)
	if err != nil {
		errorEvent := sandboxEventFromMeta(meta, req.SessionID, req.WorkspaceID, sandboxEventPrepareError, "")
		errorEvent.Duration = duration
		attachSandboxError(&errorEvent, err)
		m.logSandboxLifecycle(errorEvent)
		return envpkg.Prepared{}, fmt.Errorf(
			"session: prepare sandbox %q for %q: %w",
			req.SandboxID,
			req.SessionID,
			err,
		)
	}

	completeMeta := sessionSandboxMetaFromState(prepared.State, sandboxStatePrepared)
	if completeMeta.SandboxID == "" {
		completeMeta.SandboxID = req.SandboxID
	}
	if completeMeta.Backend == "" {
		completeMeta.Backend = string(provider.Backend())
	}
	if completeMeta.Profile == "" {
		completeMeta.Profile = req.Sandbox.Profile
	}
	completeEvent := sandboxEventFromMeta(
		completeMeta,
		req.SessionID,
		req.WorkspaceID,
		sandboxEventPrepareComplete,
		"",
	)
	completeEvent.Duration = duration
	m.logSandboxLifecycle(completeEvent)
	return prepared, nil
}

func (m *Manager) dispatchSandboxPrepare(
	ctx context.Context,
	session *Session,
	req envpkg.PrepareRequest,
) (envpkg.PrepareRequest, error) {
	payload := hookspkg.SandboxPreparePayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookSandboxPrepare,
			Timestamp: m.now(),
		},
		SessionContext:      hookSessionContext(session),
		SandboxID:           strings.TrimSpace(req.SandboxID),
		Backend:             string(req.Sandbox.Backend),
		Profile:             sandboxProfilePayload(req.Sandbox),
		LocalRootDir:        strings.TrimSpace(req.LocalRootDir),
		LocalAdditionalDirs: append([]string(nil), req.LocalAdditionalDirs...),
		AgentCommand:        strings.TrimSpace(req.AgentCommand),
		AgentEnv:            append([]string(nil), req.AgentEnv...),
		Permissions:         strings.TrimSpace(req.Permissions),
		ResumeACPState:      strings.TrimSpace(req.ResumeACPState),
	}
	patched, err := m.hooks.sandbox().DispatchSandboxPrepare(ctx, payload)
	if err != nil {
		return req, err
	}
	if patched.Denied {
		if reason := strings.TrimSpace(patched.DenyReason); reason != "" {
			return req, fmt.Errorf("session: sandbox prepare denied: %s", reason)
		}
		return req, errors.New("session: sandbox prepare denied")
	}
	if len(patched.EnvOverrides) == 0 {
		return req, nil
	}

	req.Sandbox.Env = mergeSandboxEnv(req.Sandbox.Env, patched.EnvOverrides)
	req.AgentEnv = applySandboxEnvOverrides(req.AgentEnv, patched.EnvOverrides)
	return req, nil
}

func (m *Manager) resolveSandboxSecretEnv(
	ctx context.Context,
	session *Session,
	req envpkg.PrepareRequest,
) (envpkg.PrepareRequest, error) {
	if len(req.Sandbox.SecretEnv) == 0 {
		return req, nil
	}
	if ctx == nil {
		return req, errors.New("session: sandbox secret env context is required")
	}
	if m.providerSecrets == nil {
		return req, errors.New("session: sandbox secret resolver is not configured")
	}
	values := make(map[string]string, len(req.Sandbox.SecretEnv))
	cleanups := []func(){}
	keys := make([]string, 0, len(req.Sandbox.SecretEnv))
	for key := range req.Sandbox.SecretEnv {
		keys = append(keys, strings.TrimSpace(key))
	}
	sort.Strings(keys)
	for _, key := range keys {
		ref := vault.NormalizeRef(req.Sandbox.SecretEnv[key])
		value, err := m.providerSecrets.ResolveRef(ctx, ref)
		if err != nil {
			runProviderSecretRedactions(cleanups)
			return req, fmt.Errorf("session: resolve sandbox secret_env.%s: %w", key, err)
		}
		values[key] = value
		cleanups = append(cleanups, diagnostics.RegisterDynamicSecret(value))
	}
	req.Sandbox.Env = mergeSandboxEnv(req.Sandbox.Env, values)
	req.AgentEnv = applySandboxEnvOverrides(req.AgentEnv, values)
	if session != nil {
		session.addProviderSecretRedactions(cleanups)
	}
	return req, nil
}

func (m *Manager) dispatchSandboxReady(
	ctx context.Context,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionSandboxMeta,
) error {
	payload := hookspkg.SandboxReadyPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookSandboxReady,
			Timestamp: m.now(),
		},
		SessionContext:        hookSessionContext(session),
		SandboxID:             strings.TrimSpace(state.SandboxID),
		Backend:               string(state.Backend),
		Profile:               strings.TrimSpace(state.Profile),
		InstanceID:            strings.TrimSpace(state.InstanceID),
		RuntimeRootDir:        strings.TrimSpace(state.RuntimeRootDir),
		RuntimeAdditionalDirs: append([]string(nil), state.RuntimeAdditionalDirs...),
	}
	if meta != nil {
		if payload.SandboxID == "" {
			payload.SandboxID = strings.TrimSpace(meta.SandboxID)
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
	_, err := m.hooks.sandbox().DispatchSandboxReady(ctx, payload)
	return err
}

func (m *Manager) dispatchSandboxSyncBefore(
	ctx context.Context,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionSandboxMeta,
	direction envpkg.SyncDirection,
	reason envpkg.SyncReason,
) (hookspkg.SandboxSyncBeforePayload, error) {
	payload := hookspkg.SandboxSyncBeforePayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookSandboxSyncBefore,
			Timestamp: m.now(),
		},
		SessionContext: hookSessionContext(session),
		SandboxID:      strings.TrimSpace(state.SandboxID),
		Backend:        string(state.Backend),
		Profile:        strings.TrimSpace(state.Profile),
		InstanceID:     strings.TrimSpace(state.InstanceID),
		RuntimeRootDir: strings.TrimSpace(state.RuntimeRootDir),
		Direction:      string(direction),
		Reason:         string(reason),
	}
	if m.hooks.hasSandboxHooks() {
		payload.FileCount = m.sandboxSyncFileCount(session, direction)
	}
	applySandboxMetaFallbacks(&payload.SandboxID, &payload.Backend, &payload.Profile, &payload.InstanceID, meta)
	return m.hooks.sandbox().DispatchSandboxSyncBefore(ctx, payload)
}

func (m *Manager) dispatchSandboxSyncAfter(
	ctx context.Context,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionSandboxMeta,
	direction envpkg.SyncDirection,
	reason envpkg.SyncReason,
	duration time.Duration,
	result envpkg.SyncResult,
	errorsList []string,
) error {
	payload := hookspkg.SandboxSyncAfterPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookSandboxSyncAfter,
			Timestamp: m.now(),
		},
		SessionContext:   hookSessionContext(session),
		SandboxID:        strings.TrimSpace(state.SandboxID),
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
	applySandboxMetaFallbacks(&payload.SandboxID, &payload.Backend, &payload.Profile, &payload.InstanceID, meta)
	_, err := m.hooks.sandbox().DispatchSandboxSyncAfter(ctx, payload)
	return err
}

func (m *Manager) dispatchSandboxStop(
	ctx context.Context,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionSandboxMeta,
	reason envpkg.SyncReason,
	willDestroy bool,
) (hookspkg.SandboxStopPayload, error) {
	stopReason := string(reason)
	if info := session.Info(); info != nil && strings.TrimSpace(string(info.StopReason)) != "" {
		stopReason = string(info.StopReason)
	}
	payload := hookspkg.SandboxStopPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookSandboxStop,
			Timestamp: m.now(),
		},
		SessionContext: hookSessionContext(session),
		SandboxID:      strings.TrimSpace(state.SandboxID),
		Backend:        string(state.Backend),
		Profile:        strings.TrimSpace(state.Profile),
		InstanceID:     strings.TrimSpace(state.InstanceID),
		RuntimeRootDir: strings.TrimSpace(state.RuntimeRootDir),
		StopReason:     strings.TrimSpace(stopReason),
		WillDestroy:    willDestroy,
	}
	applySandboxMetaFallbacks(&payload.SandboxID, &payload.Backend, &payload.Profile, &payload.InstanceID, meta)
	return m.hooks.sandbox().DispatchSandboxStop(ctx, payload)
}

func (m *Manager) syncSandboxToRuntime(
	ctx context.Context,
	provider envpkg.Provider,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionSandboxMeta,
) error {
	return m.syncSandboxRuntime(
		ctx,
		session,
		state,
		meta,
		envpkg.SyncDirectionToRuntime,
		envpkg.SyncReasonStart,
		provider.SyncToRuntime,
	)
}

type sandboxSyncRunner func(context.Context, envpkg.SessionState, envpkg.SyncOptions) (envpkg.SyncResult, error)

func (m *Manager) syncSandboxRuntime(
	ctx context.Context,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionSandboxMeta,
	direction envpkg.SyncDirection,
	reason envpkg.SyncReason,
	run sandboxSyncRunner,
) error {
	started := time.Now()
	m.logSandboxLifecycle(sandboxEventFromMeta(
		meta,
		session.ID,
		session.WorkspaceID,
		sandboxEventSyncStart,
		string(reason),
	))

	before, err := m.dispatchSandboxSyncBefore(ctx, session, state, meta, direction, reason)
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
	meta = cloneSessionSandboxMeta(meta)
	meta.LastSyncAt = &now
	if err != nil {
		return m.finishSandboxSyncError(ctx, session, state, meta, direction, reason, sandboxSyncOutcome{
			result:     result,
			duration:   duration,
			errorsList: errorsList,
			syncTime:   now,
			err:        err,
		})
	}
	return m.finishSandboxSyncSuccess(ctx, session, state, meta, direction, reason, sandboxSyncOutcome{
		result:     result,
		duration:   duration,
		errorsList: errorsList,
		syncTime:   now,
	})
}

type sandboxSyncOutcome struct {
	result     envpkg.SyncResult
	duration   time.Duration
	errorsList []string
	syncTime   time.Time
	err        error
}

func (m *Manager) finishSandboxSyncSuccess(
	ctx context.Context,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionSandboxMeta,
	direction envpkg.SyncDirection,
	reason envpkg.SyncReason,
	outcome sandboxSyncOutcome,
) error {
	meta.LastSyncError = ""
	session.setSandbox(meta, outcome.syncTime)
	if err := m.writeMeta(session); err != nil {
		return err
	}
	if err := m.dispatchSandboxSyncAfter(
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
	completeEvent := sandboxEventFromMeta(
		meta,
		session.ID,
		session.WorkspaceID,
		sandboxEventSyncComplete,
		string(reason),
	)
	completeEvent.Duration = outcome.duration
	m.logSandboxLifecycle(completeEvent)
	return nil
}

func (m *Manager) finishSandboxSyncError(
	ctx context.Context,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionSandboxMeta,
	direction envpkg.SyncDirection,
	reason envpkg.SyncReason,
	outcome sandboxSyncOutcome,
) error {
	err := syncSandboxWriteError(m, session, meta, outcome)
	errorsList := syncResultErrors(outcome.result, err)
	if afterErr := m.dispatchSandboxSyncAfter(
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
		m.warnHookDispatch(ctx, session, hookspkg.HookSandboxSyncAfter, afterErr)
	}

	errorEvent := sandboxEventFromMeta(
		meta,
		session.ID,
		session.WorkspaceID,
		sandboxEventSyncError,
		string(reason),
	)
	errorEvent.Duration = outcome.duration
	attachSandboxError(&errorEvent, err)
	m.logSandboxLifecycle(errorEvent)
	return fmt.Errorf(
		"session: sync sandbox %q %s runtime for %q: %w",
		state.SandboxID,
		syncDirectionPreposition(direction),
		session.ID,
		err,
	)
}

func syncSandboxWriteError(
	m *Manager,
	session *Session,
	meta *store.SessionSandboxMeta,
	outcome sandboxSyncOutcome,
) error {
	meta.LastSyncError = outcome.err.Error()
	session.setSandbox(meta, outcome.syncTime)
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

func (m *Manager) finalizeSandbox(
	ctx context.Context,
	session *Session,
	reason envpkg.SyncReason,
) error {
	if session == nil {
		return nil
	}
	meta := cloneSessionSandboxMeta(session.Info().Sandbox)
	if meta == nil {
		return nil
	}
	if m.sandbox == nil {
		return errors.New("session: sandbox registry is required")
	}

	provider, err := m.sandbox.Provider(envpkg.Backend(strings.TrimSpace(meta.Backend)))
	if err != nil {
		return fmt.Errorf("session: resolve sandbox provider %q: %w", meta.Backend, err)
	}

	state := sessionSandboxStateFromMeta(meta)
	var errs []error
	if syncErr := m.syncSandboxFromRuntime(ctx, provider, session, state, meta, reason); syncErr != nil {
		if reason == envpkg.SyncReasonCrash {
			m.sessionLogger(session).Warn("session: sandbox crash sync failed", "error", syncErr)
		} else {
			errs = append(errs, syncErr)
		}
		meta = cloneSessionSandboxMeta(session.Info().Sandbox)
		state = sessionSandboxStateFromMeta(meta)
	}

	shouldDestroy := session.sandboxShouldDestroy()
	stopPayload, stopErr := m.dispatchSandboxStop(ctx, session, state, meta, reason, shouldDestroy)
	if stopErr != nil {
		errs = append(errs, stopErr)
		shouldDestroy = false
	}
	if stopPayload.Denied {
		shouldDestroy = false
	}

	if shouldDestroy {
		if destroyErr := m.destroySandbox(ctx, provider, session, state); destroyErr != nil {
			errs = append(errs, destroyErr)
		}
	} else {
		now := m.now()
		meta = cloneSessionSandboxMeta(session.Info().Sandbox)
		if meta != nil {
			meta.State = sandboxStateStopped
			session.setSandbox(meta, now)
			if err := m.writeMeta(session); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}

func (m *Manager) syncSandboxFromRuntime(
	ctx context.Context,
	provider envpkg.Provider,
	session *Session,
	state envpkg.SessionState,
	meta *store.SessionSandboxMeta,
	reason envpkg.SyncReason,
) error {
	return m.syncSandboxRuntime(
		ctx,
		session,
		state,
		meta,
		envpkg.SyncDirectionFromRuntime,
		reason,
		provider.SyncFromRuntime,
	)
}

func (m *Manager) destroySandbox(
	ctx context.Context,
	provider envpkg.Provider,
	session *Session,
	state envpkg.SessionState,
) error {
	meta := cloneSessionSandboxMeta(session.Info().Sandbox)
	started := time.Now()
	startEvent := sandboxEventFromMeta(meta, session.ID, session.WorkspaceID, sandboxEventDestroyStart, "")
	m.logSandboxLifecycle(startEvent)

	err := provider.Destroy(ctx, state)
	duration := time.Since(started)
	if err != nil {
		errorEvent := sandboxEventFromMeta(meta, session.ID, session.WorkspaceID, sandboxEventDestroyError, "")
		errorEvent.Duration = duration
		attachSandboxError(&errorEvent, err)
		m.logSandboxLifecycle(errorEvent)
		return fmt.Errorf("session: destroy sandbox %q for %q: %w", state.SandboxID, session.ID, err)
	}

	now := m.now()
	meta = cloneSessionSandboxMeta(meta)
	if meta != nil {
		meta.State = sandboxStateDestroyed
		session.setSandbox(meta, now)
		if err := m.writeMeta(session); err != nil {
			return err
		}
	}
	completeEvent := sandboxEventFromMeta(
		meta,
		session.ID,
		session.WorkspaceID,
		sandboxEventDestroyComplete,
		"",
	)
	completeEvent.Duration = duration
	m.logSandboxLifecycle(completeEvent)
	return nil
}

func (m *Manager) logSandboxTransport(session *Session, eventName string, err error, duration time.Duration) {
	if session == nil {
		return
	}
	meta := cloneSessionSandboxMeta(session.Info().Sandbox)
	event := sandboxEventFromMeta(meta, session.ID, session.WorkspaceID, eventName, "")
	event.Duration = duration
	if err != nil {
		attachSandboxError(&event, err)
	}
	m.logSandboxLifecycle(event)
}

func (m *Manager) logSandboxLifecycle(event SandboxLifecycleEvent) {
	event.Name = strings.TrimSpace(event.Name)
	if event.Name == "" {
		return
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = m.now()
	}
	if event.Span == "" {
		event.Span = sandboxSpanForEvent(event.Name, event.Reason)
	}
	logger := m.logger
	if logger == nil {
		logger = slog.Default()
	}

	args := []any{
		"backend", strings.TrimSpace(event.Backend),
		"profile", strings.TrimSpace(event.Profile),
		"sandbox_id", strings.TrimSpace(event.SandboxID),
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

	if notifier, ok := m.notifier.(SandboxLifecycleNotifier); ok {
		notifier.OnSandboxLifecycleEvent(m.lifecycleCtx, event)
	}
}

func sandboxStartOpts(
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

func initialSessionSandboxMeta(
	sandboxID string,
	resolved envpkg.Resolved,
	previous *store.SessionSandboxMeta,
) *store.SessionSandboxMeta {
	meta := cloneSessionSandboxMeta(previous)
	if meta == nil {
		meta = &store.SessionSandboxMeta{}
	}
	meta.SandboxID = strings.TrimSpace(sandboxID)
	meta.Backend = string(resolved.Backend)
	meta.Profile = strings.TrimSpace(resolved.Profile)
	meta.State = sandboxStateCreating
	return meta
}

func normalizePreparedSandboxState(
	prepared envpkg.Prepared,
	meta *store.SessionSandboxMeta,
	resolved envpkg.Resolved,
) (envpkg.SessionState, error) {
	state := prepared.State
	if strings.TrimSpace(state.SandboxID) == "" {
		state.SandboxID = strings.TrimSpace(meta.SandboxID)
	}
	if state.Backend == "" {
		state.Backend = resolved.Backend
	}
	if strings.TrimSpace(state.Profile) == "" {
		state.Profile = strings.TrimSpace(resolved.Profile)
	}
	if strings.TrimSpace(state.State) == "" {
		state.State = sandboxStatePrepared
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
	if strings.TrimSpace(state.SandboxID) == "" {
		return envpkg.SessionState{}, errors.New("session: prepared sandbox id is required")
	}
	if !state.Backend.Valid() {
		return envpkg.SessionState{}, fmt.Errorf("session: prepared sandbox backend %q is invalid", state.Backend)
	}
	if strings.TrimSpace(state.RuntimeRootDir) == "" {
		return envpkg.SessionState{}, errors.New("session: prepared runtime root dir is required")
	}
	return state, nil
}

func sessionSandboxMetaFromState(
	state envpkg.SessionState,
	fallbackState string,
) *store.SessionSandboxMeta {
	if strings.TrimSpace(state.SandboxID) == "" && state.Backend == "" {
		return nil
	}
	sessionState := strings.TrimSpace(state.State)
	if sessionState == "" {
		sessionState = fallbackState
	}
	return &store.SessionSandboxMeta{
		SandboxID:             strings.TrimSpace(state.SandboxID),
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

func sessionSandboxStateFromMeta(meta *store.SessionSandboxMeta) envpkg.SessionState {
	if meta == nil {
		return envpkg.SessionState{}
	}
	return envpkg.SessionState{
		SandboxID:             strings.TrimSpace(meta.SandboxID),
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

func sessionSandboxID(meta *store.SessionSandboxMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.SandboxID)
}

func normalizeResolvedSandbox(resolved envpkg.Resolved) envpkg.Resolved {
	if !resolved.Backend.Valid() {
		resolved.Backend = envpkg.BackendLocal
	}
	if strings.TrimSpace(resolved.Profile) == "" {
		resolved.Profile = string(resolved.Backend)
	}
	return resolved
}

func sandboxAgentEnv(base []string, resolved envpkg.Resolved) []string {
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

func sandboxProfilePayload(resolved envpkg.Resolved) hookspkg.SandboxProfilePayload {
	return hookspkg.SandboxProfilePayload{
		Profile:        strings.TrimSpace(resolved.Profile),
		Backend:        string(resolved.Backend),
		SyncMode:       string(resolved.SyncMode),
		Persistence:    string(resolved.Persistence),
		RuntimeRootDir: strings.TrimSpace(resolved.RuntimeRootDir),
		DestroyOnStop:  resolved.DestroyOnStop,
		Env:            mergeSandboxEnv(nil, resolved.Env),
		SecretEnv:      mergeSandboxEnv(nil, resolved.SecretEnv),
	}
}

func mergeSandboxEnv(base map[string]string, overrides map[string]string) map[string]string {
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

func applySandboxEnvOverrides(env []string, overrides map[string]string) []string {
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

func applySandboxMetaFallbacks(
	sandboxID *string,
	backend *string,
	profile *string,
	instanceID *string,
	meta *store.SessionSandboxMeta,
) {
	if meta == nil {
		return
	}
	if sandboxID != nil && strings.TrimSpace(*sandboxID) == "" {
		*sandboxID = strings.TrimSpace(meta.SandboxID)
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

func (m *Manager) sandboxSyncFileCount(session *Session, direction envpkg.SyncDirection) int {
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
		m.sessionLogger(session).Warn("session: count sandbox sync files failed", "root", root, "error", err)
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

func sandboxEventFromMeta(
	meta *store.SessionSandboxMeta,
	sessionID string,
	workspaceID string,
	name string,
	reason string,
) SandboxLifecycleEvent {
	event := SandboxLifecycleEvent{
		Name:        strings.TrimSpace(name),
		SessionID:   strings.TrimSpace(sessionID),
		WorkspaceID: strings.TrimSpace(workspaceID),
		Reason:      strings.TrimSpace(reason),
	}
	if meta != nil {
		event.SandboxID = strings.TrimSpace(meta.SandboxID)
		event.Backend = strings.TrimSpace(meta.Backend)
		event.Profile = strings.TrimSpace(meta.Profile)
		event.InstanceID = strings.TrimSpace(meta.InstanceID)
	}
	return event
}

func attachSandboxError(event *SandboxLifecycleEvent, err error) {
	if event == nil || err == nil {
		return
	}
	event.Error = err.Error()
	event.ErrorKind = sandboxErrorKind(err)
}

func sandboxErrorKind(err error) string {
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

func sandboxSpanForEvent(name string, reason string) string {
	switch name {
	case sandboxEventPrepareStart, sandboxEventPrepareComplete, sandboxEventPrepareError:
		return sandboxSandboxPreparePath
	case sandboxEventDestroyStart, sandboxEventDestroyComplete, sandboxEventDestroyError:
		return "sandbox.destroy"
	case sandboxEventSyncStart, sandboxEventSyncComplete, sandboxEventSyncError:
		if reason == string(envpkg.SyncReasonStart) || reason == string(envpkg.SyncReasonTurn) {
			return "sandbox.sync.to_runtime"
		}
		return "sandbox.sync.from_runtime"
	default:
		return strings.TrimSpace(name)
	}
}

func cloneSessionSandboxMeta(meta *store.SessionSandboxMeta) *store.SessionSandboxMeta {
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
