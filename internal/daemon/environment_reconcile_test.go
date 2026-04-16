package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/environment"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestReconcileDaemonEnvironmentsWithNoRemoteSessionsIsNoop(t *testing.T) {
	daemon, state, provider, _ := newEnvironmentReconcileHarness(t)
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-local",
		state:  session.StateActive,
		env:    remoteMeta("env-local", environment.BackendLocal, "local", ""),
		agent:  "coder",
		worker: "ws-local",
	})

	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)

	if got := len(provider.prepareRequests); got != 0 {
		t.Fatalf("Prepare calls = %d, want 0", got)
	}
	if got := len(provider.findRequests); got != 0 {
		t.Fatalf("FindEnvironment calls = %d, want 0", got)
	}
	if got := len(provider.destroyStates); got != 0 {
		t.Fatalf("Destroy calls = %d, want 0", got)
	}
}

func TestReconcileDaemonEnvironmentsHandlesBootEdgeCases(t *testing.T) {
	daemon, state, provider, logs := newEnvironmentReconcileHarness(t)

	var nilBootContext context.Context
	daemon.reconcileDaemonEnvironments(nilBootContext, nil)

	logs.Reset()
	state.environmentRegistry = nil
	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)
	if !strings.Contains(logs.String(), "environment registry is required") {
		t.Fatalf("logs missing nil registry warning: %s", logs.String())
	}

	logs.Reset()
	registry, err := environment.NewRegistry(provider)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	state.environmentRegistry = registry
	daemon.homePaths.SessionsDir = filepath.Join(t.TempDir(), "missing-sessions")
	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)
	if got := len(provider.prepareRequests); got != 0 {
		t.Fatalf("Prepare calls = %d, want 0 for missing sessions dir", got)
	}

	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-canceled",
		state:  session.StateActive,
		env:    remoteMeta("env-canceled", environment.BackendDaytona, "daytona", "sandbox-canceled"),
		agent:  "coder",
		worker: "ws-canceled",
	})
	ctx, cancel := context.WithCancel(testutil.Context(t))
	cancel()
	daemon.reconcileDaemonEnvironments(ctx, state)
	if got := len(provider.prepareRequests); got != 0 {
		t.Fatalf("Prepare calls = %d, want 0 after context cancellation", got)
	}
	if !strings.Contains(logs.String(), "daemon: environment reconciliation canceled") {
		t.Fatalf("logs missing cancellation warning: %s", logs.String())
	}
}

func TestLoadEnvironmentReconcileSessionsSkipsUnreadableMetadata(t *testing.T) {
	daemon, state, _, logs := newEnvironmentReconcileHarness(t)
	sessionsDir := daemon.homePaths.SessionsDir
	if err := os.MkdirAll(filepath.Join(sessionsDir, "sess-bad"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(
		store.SessionMetaFile(filepath.Join(sessionsDir, "sess-bad")),
		[]byte("{not-json"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionsDir, "not-a-session-dir"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("WriteFile(non-dir) error = %v", err)
	}
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-good",
		state:  session.StateActive,
		env:    remoteMeta("env-good", environment.BackendDaytona, "daytona", "sandbox-good"),
		agent:  "coder",
		worker: "ws-good",
	})

	sessions, err := daemon.loadEnvironmentReconcileSessions(state)
	if err != nil {
		t.Fatalf("loadEnvironmentReconcileSessions() error = %v", err)
	}
	if got := len(sessions); got != 1 {
		t.Fatalf("remote sessions = %d, want 1", got)
	}
	if got := sessions[0].meta.ID; got != "sess-good" {
		t.Fatalf("remote session ID = %q, want sess-good", got)
	}
	if !strings.Contains(logs.String(), "skipped unreadable session metadata") {
		t.Fatalf("logs missing unreadable metadata warning: %s", logs.String())
	}
}

func TestReconcileDaemonEnvironmentsReattachesRecoverableSession(t *testing.T) {
	daemon, state, provider, _ := newEnvironmentReconcileHarness(t)
	provider.prepareState = environment.SessionState{
		EnvironmentID: "env-remote",
		Backend:       environment.BackendDaytona,
		Profile:       "daytona",
		State:         "ready",
		InstanceID:    "sandbox-reattached",
		ProviderState: json.RawMessage(`{"reattached":true}`),
	}
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-active",
		state:  session.StateActive,
		env:    remoteMeta("env-remote", environment.BackendDaytona, "daytona", "sandbox-remote"),
		agent:  "coder",
		worker: "ws-active",
	})

	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)

	if got := len(provider.prepareRequests); got != 1 {
		t.Fatalf("Prepare calls = %d, want 1", got)
	}
	req := provider.prepareRequests[0]
	if req.EnvironmentID != "env-remote" {
		t.Fatalf("PrepareRequest.EnvironmentID = %q, want env-remote", req.EnvironmentID)
	}
	if req.InstanceID != "sandbox-remote" {
		t.Fatalf("PrepareRequest.InstanceID = %q, want sandbox-remote", req.InstanceID)
	}
	assertEnvironmentReconcileJSON(
		t,
		req.ProviderState,
		json.RawMessage(`{"seed":true}`),
		"PrepareRequest.ProviderState",
	)

	meta := readEnvironmentReconcileMeta(t, daemon, "sess-active")
	if meta.Environment.InstanceID != "sandbox-reattached" {
		t.Fatalf("persisted InstanceID = %q, want sandbox-reattached", meta.Environment.InstanceID)
	}
	assertEnvironmentReconcileJSON(
		t,
		meta.Environment.ProviderState,
		json.RawMessage(`{"reattached":true}`),
		"persisted ProviderState",
	)
	if meta.Environment.State != environmentReconcileStatePrepared {
		t.Fatalf("persisted environment state = %q, want %q", meta.Environment.State, environmentReconcileStatePrepared)
	}
}

func TestReconcileDaemonEnvironmentsFindsAndAttachesPartialCreate(t *testing.T) {
	daemon, state, provider, _ := newEnvironmentReconcileHarness(t)
	provider.findState = environment.SessionState{
		EnvironmentID: "env-partial",
		Backend:       environment.BackendDaytona,
		Profile:       "daytona",
		State:         "found",
		InstanceID:    "sandbox-found",
		ProviderState: json.RawMessage(`{"found":true}`),
	}
	provider.prepareState = environment.SessionState{
		EnvironmentID: "env-partial",
		Backend:       environment.BackendDaytona,
		Profile:       "daytona",
		State:         "ready",
		InstanceID:    "sandbox-found",
		ProviderState: json.RawMessage(`{"attached":true}`),
	}
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-partial",
		state:  session.StateActive,
		env:    remoteMeta("env-partial", environment.BackendDaytona, "daytona", ""),
		agent:  "coder",
		worker: "ws-partial",
	})

	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)

	if got := len(provider.findRequests); got != 1 {
		t.Fatalf("FindEnvironment calls = %d, want 1", got)
	}
	if got := provider.findRequests[0].Labels["agh_environment_id"]; got != "env-partial" {
		t.Fatalf("FindEnvironment label agh_environment_id = %q, want env-partial", got)
	}
	if got := len(provider.prepareRequests); got != 1 {
		t.Fatalf("Prepare calls = %d, want 1", got)
	}
	if got := provider.prepareRequests[0].InstanceID; got != "sandbox-found" {
		t.Fatalf("PrepareRequest.InstanceID = %q, want sandbox-found", got)
	}
	assertEnvironmentReconcileJSON(
		t,
		provider.prepareRequests[0].ProviderState,
		json.RawMessage(`{"found":true}`),
		"PrepareRequest.ProviderState",
	)

	meta := readEnvironmentReconcileMeta(t, daemon, "sess-partial")
	if meta.Environment.InstanceID != "sandbox-found" {
		t.Fatalf("persisted InstanceID = %q, want sandbox-found", meta.Environment.InstanceID)
	}
	assertEnvironmentReconcileJSON(
		t,
		meta.Environment.ProviderState,
		json.RawMessage(`{"attached":true}`),
		"persisted ProviderState",
	)
}

func TestReconcileDaemonEnvironmentsDestroysUnrecoverablePartialCreate(t *testing.T) {
	daemon, state, provider, logs := newEnvironmentReconcileHarness(t)
	provider.findState = environment.SessionState{
		EnvironmentID: "env-stopped-partial",
		Backend:       environment.BackendDaytona,
		Profile:       "daytona",
		State:         "found",
		InstanceID:    "sandbox-stopped-partial",
		ProviderState: json.RawMessage(`{"found":true}`),
	}
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-stopped-partial",
		state:  session.StateStopped,
		env:    remoteMeta("env-stopped-partial", environment.BackendDaytona, "daytona", ""),
		agent:  "coder",
		worker: "ws-stopped-partial",
	})

	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)

	if got := len(provider.prepareRequests); got != 0 {
		t.Fatalf("Prepare calls = %d, want 0", got)
	}
	if got := len(provider.destroyStates); got != 1 {
		t.Fatalf("Destroy calls = %d, want 1", got)
	}
	if got := provider.destroyStates[0].InstanceID; got != "sandbox-stopped-partial" {
		t.Fatalf("Destroy state InstanceID = %q, want sandbox-stopped-partial", got)
	}
	meta := readEnvironmentReconcileMeta(t, daemon, "sess-stopped-partial")
	if meta.Environment.State != environmentReconcileStateDestroyed {
		t.Fatalf("persisted environment state = %q, want destroyed", meta.Environment.State)
	}
	if !strings.Contains(logs.String(), "daemon: environment destroy complete") {
		t.Fatalf("logs missing destroy completion: %s", logs.String())
	}
}

func TestReconcileDaemonEnvironmentsDestroysUnrecoverableSession(t *testing.T) {
	daemon, state, provider, _ := newEnvironmentReconcileHarness(t)
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-stopped",
		state:  session.StateStopped,
		env:    remoteMeta("env-stopped", environment.BackendDaytona, "daytona", "sandbox-stopped"),
		agent:  "coder",
		worker: "ws-stopped",
	})

	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)

	if got := len(provider.destroyStates); got != 1 {
		t.Fatalf("Destroy calls = %d, want 1", got)
	}
	if got := provider.destroyStates[0].InstanceID; got != "sandbox-stopped" {
		t.Fatalf("Destroy state InstanceID = %q, want sandbox-stopped", got)
	}
}

func TestReconcileDaemonEnvironmentsUsesResolvedWorkspaceInputs(t *testing.T) {
	daemon, state, provider, _ := newEnvironmentReconcileHarness(t)
	rootDir := t.TempDir()
	additionalDir := t.TempDir()
	expectedRootDir, err := filepath.EvalSymlinks(rootDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(root) error = %v", err)
	}
	expectedRootDir, err = filepath.Abs(expectedRootDir)
	if err != nil {
		t.Fatalf("Abs(root) error = %v", err)
	}
	resolver, err := workspacepkg.NewResolver(
		&environmentReconcileWorkspaceStore{
			workspaces: map[string]workspacepkg.Workspace{
				"ws-resolved": {
					ID:             "ws-resolved",
					RootDir:        rootDir,
					AdditionalDirs: []string{additionalDir},
					Name:           "resolved",
					EnvironmentRef: "daytona-dev",
					CreatedAt:      time.Date(2026, 4, 16, 9, 0, 0, 0, time.UTC),
					UpdatedAt:      time.Date(2026, 4, 16, 9, 0, 0, 0, time.UTC),
				},
			},
		},
		workspacepkg.WithConfigLoader(func(string) (aghconfig.Config, error) {
			return aghconfig.Config{
				Environments: map[string]aghconfig.EnvironmentProfile{
					"daytona-dev": {
						Backend:  string(environment.BackendDaytona),
						SyncMode: string(environment.SyncModeNone),
						Daytona: aghconfig.DaytonaProfile{
							Snapshot: "snap-reconcile",
						},
					},
				},
			}, nil
		}),
		workspacepkg.WithLogger(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))),
	)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}
	state.workspaceResolver = resolver
	provider.prepareState = environment.SessionState{
		EnvironmentID: "env-resolved",
		Backend:       environment.BackendDaytona,
		Profile:       "daytona-dev",
		State:         "ready",
		InstanceID:    "sandbox-resolved",
	}
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-resolved",
		state:  session.StateActive,
		env:    remoteMeta("env-resolved", environment.BackendDaytona, "daytona-dev", "sandbox-resolved"),
		agent:  "coder",
		worker: "ws-resolved",
	})

	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)

	if got := len(provider.prepareRequests); got != 1 {
		t.Fatalf("Prepare calls = %d, want 1", got)
	}
	req := provider.prepareRequests[0]
	if req.LocalRootDir != expectedRootDir {
		t.Fatalf("PrepareRequest.LocalRootDir = %q, want %q", req.LocalRootDir, expectedRootDir)
	}
	if len(req.LocalAdditionalDirs) != 1 || req.LocalAdditionalDirs[0] != additionalDir {
		t.Fatalf("PrepareRequest.LocalAdditionalDirs = %#v, want [%q]", req.LocalAdditionalDirs, additionalDir)
	}
	if req.Environment.Profile != "daytona-dev" ||
		req.Environment.Backend != environment.BackendDaytona ||
		req.Environment.SyncMode != environment.SyncModeNone {
		t.Fatalf("PrepareRequest.Environment = %#v, want resolved Daytona profile", req.Environment)
	}
}

func TestReconcileDaemonEnvironmentsUnavailableProviderLogsAndContinues(t *testing.T) {
	daemon, state, _, logs := newEnvironmentReconcileHarness(t)
	emptyRegistry, err := environment.NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	state.environmentRegistry = emptyRegistry
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-provider-missing",
		state:  session.StateActive,
		env:    remoteMeta("env-provider-missing", environment.BackendDaytona, "daytona", "sandbox-missing"),
		agent:  "coder",
		worker: "ws-provider-missing",
	})

	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)
	if !strings.Contains(logs.String(), "daemon: environment reconciliation provider unavailable") {
		t.Fatalf("logs missing provider unavailable warning: %s", logs.String())
	}
}

func TestReconcileDaemonEnvironmentsFailureDoesNotBlockBoot(t *testing.T) {
	daemon, state, provider, logs := newEnvironmentReconcileHarness(t)
	provider.prepareErr = errors.New("provider unavailable")
	provider.destroyErr = errors.New("destroy unavailable")
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-failure",
		state:  session.StateActive,
		env:    remoteMeta("env-failure", environment.BackendDaytona, "daytona", "sandbox-failure"),
		agent:  "coder",
		worker: "ws-failure",
	})

	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)
	if got := len(provider.prepareRequests); got != 1 {
		t.Fatalf("Prepare calls = %d, want 1", got)
	}
	if got := len(provider.destroyStates); got != 1 {
		t.Fatalf("Destroy calls = %d, want 1 cleanup attempt", got)
	}
	if !strings.Contains(logs.String(), "daemon: environment reattach failed") ||
		!strings.Contains(logs.String(), "daemon: environment destroy failed") {
		t.Fatalf("logs missing non-blocking failure evidence: %s", logs.String())
	}
}

func TestReconcileDaemonEnvironmentsPersistsSessionIndexBestEffort(t *testing.T) {
	daemon, state, provider, _ := newEnvironmentReconcileHarness(t)
	registry := &environmentReconcileRegistry{}
	state.registry = registry
	provider.prepareState = environment.SessionState{
		EnvironmentID: "env-indexed",
		Backend:       environment.BackendDaytona,
		Profile:       "daytona",
		State:         "ready",
		InstanceID:    "sandbox-indexed",
		ProviderState: json.RawMessage(`{"indexed":true}`),
	}
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-indexed",
		state:  session.StateActive,
		env:    remoteMeta("env-indexed", environment.BackendDaytona, "daytona", "sandbox-seed"),
		agent:  "coder",
		worker: "ws-indexed",
	})

	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)

	if got := len(registry.sessions); got != 1 {
		t.Fatalf("RegisterSession calls = %d, want 1", got)
	}
	indexed := registry.sessions[0]
	if indexed.ID != "sess-indexed" || indexed.Environment == nil {
		t.Fatalf("indexed session = %#v, want session with environment", indexed)
	}
	if indexed.Environment.InstanceID != "sandbox-indexed" {
		t.Fatalf("indexed environment instance = %q, want sandbox-indexed", indexed.Environment.InstanceID)
	}
}

func TestReconcileDaemonEnvironmentsLogsSessionIndexFailure(t *testing.T) {
	daemon, state, provider, logs := newEnvironmentReconcileHarness(t)
	state.registry = &failingEnvironmentReconcileRegistry{}
	provider.prepareState = environment.SessionState{
		EnvironmentID: "env-index-fails",
		Backend:       environment.BackendDaytona,
		Profile:       "daytona",
		State:         "ready",
		InstanceID:    "sandbox-index-fails",
	}
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-index-fails",
		state:  session.StateActive,
		env:    remoteMeta("env-index-fails", environment.BackendDaytona, "daytona", "sandbox-seed"),
		agent:  "coder",
		worker: "ws-index-fails",
	})

	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)

	if !strings.Contains(logs.String(), "daemon: environment reconciliation session index update failed") {
		t.Fatalf("logs missing session index failure: %s", logs.String())
	}
}

func TestReconcileDaemonEnvironmentsPartialCreateNotFoundDoesNotBlock(t *testing.T) {
	daemon, state, provider, logs := newEnvironmentReconcileHarness(t)
	provider.findErr = environment.ErrEnvironmentNotFound
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-not-found",
		state:  session.StateActive,
		env:    remoteMeta("env-not-found", environment.BackendDaytona, "daytona", ""),
		agent:  "coder",
		worker: "ws-not-found",
	})

	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)
	if got := len(provider.prepareRequests); got != 0 {
		t.Fatalf("Prepare calls = %d, want 0 without a discovered instance", got)
	}
	if !strings.Contains(logs.String(), "daemon: environment reconciliation remote not found") {
		t.Fatalf("logs missing remote not found info: %s", logs.String())
	}
}

func TestReconcileDaemonEnvironmentsDestroySkippedWithoutInstance(t *testing.T) {
	daemon, state, provider, logs := newEnvironmentReconcileHarness(t)
	writeEnvironmentReconcileMeta(t, daemon, environmentReconcileMeta{
		id:     "sess-no-instance",
		state:  session.StateStopped,
		env:    remoteMeta("env-no-instance", environment.BackendDaytona, "daytona", ""),
		agent:  "coder",
		worker: "ws-no-instance",
	})

	daemon.reconcileDaemonEnvironments(testutil.Context(t), state)
	if got := len(provider.destroyStates); got != 0 {
		t.Fatalf("Destroy calls = %d, want 0", got)
	}
	if !strings.Contains(logs.String(), "daemon: environment reconciliation skipped missing instance") {
		t.Fatalf("logs missing missing instance warning: %s", logs.String())
	}
}

type environmentReconcileMeta struct {
	id     string
	state  session.State
	env    *store.SessionEnvironmentMeta
	agent  string
	worker string
}

func newEnvironmentReconcileHarness(
	t *testing.T,
) (*Daemon, *bootState, *recordingEnvironmentReconcileProvider, *bytes.Buffer) {
	t.Helper()

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	provider := &recordingEnvironmentReconcileProvider{backend: environment.BackendDaytona}
	registry, err := environment.NewRegistry(provider)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	daemon := &Daemon{
		homePaths: aghconfig.HomePaths{SessionsDir: filepath.Join(t.TempDir(), "sessions")},
		now: func() time.Time {
			return time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
		},
	}
	state := &bootState{
		logger:              logger,
		environmentRegistry: registry,
	}
	return daemon, state, provider, logs
}

func writeEnvironmentReconcileMeta(t *testing.T, daemon *Daemon, spec environmentReconcileMeta) {
	t.Helper()
	meta := store.SessionMeta{
		ID:          spec.id,
		AgentName:   spec.agent,
		WorkspaceID: spec.worker,
		SessionType: string(session.SessionTypeUser),
		State:       string(spec.state),
		Environment: spec.env,
		CreatedAt:   time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
	}
	path := store.SessionMetaFile(filepath.Join(daemon.homePaths.SessionsDir, spec.id))
	if err := store.WriteSessionMeta(path, meta); err != nil {
		t.Fatalf("WriteSessionMeta() error = %v", err)
	}
}

func readEnvironmentReconcileMeta(t *testing.T, daemon *Daemon, sessionID string) store.SessionMeta {
	t.Helper()
	meta, err := store.ReadSessionMeta(store.SessionMetaFile(filepath.Join(daemon.homePaths.SessionsDir, sessionID)))
	if err != nil {
		t.Fatalf("ReadSessionMeta() error = %v", err)
	}
	return meta
}

func remoteMeta(
	environmentID string,
	backend environment.Backend,
	profile string,
	instanceID string,
) *store.SessionEnvironmentMeta {
	return &store.SessionEnvironmentMeta{
		EnvironmentID: environmentID,
		Backend:       string(backend),
		Profile:       profile,
		State:         "creating",
		InstanceID:    instanceID,
		ProviderState: json.RawMessage(`{"seed":true}`),
	}
}

func assertEnvironmentReconcileJSON(t *testing.T, got json.RawMessage, want json.RawMessage, label string) {
	t.Helper()
	var gotValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("%s got invalid JSON %s: %v", label, got, err)
	}
	var wantValue any
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("%s want invalid JSON %s: %v", label, want, err)
	}
	if !jsonValuesEqual(gotValue, wantValue) {
		t.Fatalf("%s = %s, want %s", label, got, want)
	}
}

func jsonValuesEqual(left any, right any) bool {
	leftRaw, leftErr := json.Marshal(left)
	rightRaw, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftRaw, rightRaw)
}

type recordingEnvironmentReconcileProvider struct {
	backend         environment.Backend
	prepareRequests []environment.PrepareRequest
	findRequests    []environment.FindEnvironmentRequest
	destroyStates   []environment.SessionState
	prepareState    environment.SessionState
	findState       environment.SessionState
	prepareErr      error
	findErr         error
	destroyErr      error
}

type environmentReconcileRegistry struct {
	recordingRegistry
	sessions []store.SessionInfo
}

func (r *environmentReconcileRegistry) RegisterSession(_ context.Context, session store.SessionInfo) error {
	r.sessions = append(r.sessions, session)
	return nil
}

type failingEnvironmentReconcileRegistry struct {
	recordingRegistry
}

func (r *failingEnvironmentReconcileRegistry) RegisterSession(context.Context, store.SessionInfo) error {
	return errors.New("index unavailable")
}

type environmentReconcileWorkspaceStore struct {
	workspaces map[string]workspacepkg.Workspace
}

func (s *environmentReconcileWorkspaceStore) InsertWorkspace(context.Context, workspacepkg.Workspace) error {
	return nil
}

func (s *environmentReconcileWorkspaceStore) UpdateWorkspace(_ context.Context, ws workspacepkg.Workspace) error {
	s.workspaces[ws.ID] = ws
	return nil
}

func (s *environmentReconcileWorkspaceStore) DeleteWorkspace(context.Context, string) error {
	return nil
}

func (s *environmentReconcileWorkspaceStore) GetWorkspace(
	_ context.Context,
	id string,
) (workspacepkg.Workspace, error) {
	if ws, ok := s.workspaces[id]; ok {
		return ws, nil
	}
	return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (s *environmentReconcileWorkspaceStore) GetWorkspaceByPath(
	_ context.Context,
	rootDir string,
) (workspacepkg.Workspace, error) {
	for _, ws := range s.workspaces {
		if ws.RootDir == rootDir {
			return ws, nil
		}
	}
	return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (s *environmentReconcileWorkspaceStore) GetWorkspaceByName(
	_ context.Context,
	name string,
) (workspacepkg.Workspace, error) {
	for _, ws := range s.workspaces {
		if ws.Name == name {
			return ws, nil
		}
	}
	return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (s *environmentReconcileWorkspaceStore) ListWorkspaces(context.Context) ([]workspacepkg.Workspace, error) {
	workspaces := make([]workspacepkg.Workspace, 0, len(s.workspaces))
	for _, ws := range s.workspaces {
		workspaces = append(workspaces, ws)
	}
	return workspaces, nil
}

func (p *recordingEnvironmentReconcileProvider) Backend() environment.Backend {
	return p.backend
}

func (p *recordingEnvironmentReconcileProvider) Prepare(
	_ context.Context,
	req environment.PrepareRequest,
) (environment.Prepared, error) {
	p.prepareRequests = append(p.prepareRequests, clonePrepareRequest(req))
	if p.prepareErr != nil {
		return environment.Prepared{}, p.prepareErr
	}
	state := p.prepareState
	if strings.TrimSpace(state.EnvironmentID) == "" {
		state.EnvironmentID = req.EnvironmentID
	}
	if !state.Backend.Valid() {
		state.Backend = p.backend
	}
	if strings.TrimSpace(state.Profile) == "" {
		state.Profile = req.Environment.Profile
	}
	if strings.TrimSpace(state.InstanceID) == "" {
		state.InstanceID = req.InstanceID
	}
	if len(state.ProviderState) == 0 {
		state.ProviderState = append(json.RawMessage(nil), req.ProviderState...)
	}
	return environment.Prepared{State: state}, nil
}

func (p *recordingEnvironmentReconcileProvider) SyncToRuntime(
	context.Context,
	environment.SessionState,
	environment.SyncOptions,
) (environment.SyncResult, error) {
	return environment.SyncResult{}, nil
}

func (p *recordingEnvironmentReconcileProvider) SyncFromRuntime(
	context.Context,
	environment.SessionState,
	environment.SyncOptions,
) (environment.SyncResult, error) {
	return environment.SyncResult{}, nil
}

func (p *recordingEnvironmentReconcileProvider) Destroy(
	_ context.Context,
	state environment.SessionState,
) error {
	p.destroyStates = append(p.destroyStates, cloneSessionState(state))
	return p.destroyErr
}

func (p *recordingEnvironmentReconcileProvider) FindEnvironment(
	_ context.Context,
	req environment.FindEnvironmentRequest,
) (environment.SessionState, error) {
	p.findRequests = append(p.findRequests, cloneFindRequest(req))
	if p.findErr != nil {
		return environment.SessionState{}, p.findErr
	}
	return cloneSessionState(p.findState), nil
}

func clonePrepareRequest(req environment.PrepareRequest) environment.PrepareRequest {
	cloned := req
	cloned.LocalAdditionalDirs = append([]string(nil), req.LocalAdditionalDirs...)
	cloned.AgentEnv = append([]string(nil), req.AgentEnv...)
	cloned.ProviderState = append(json.RawMessage(nil), req.ProviderState...)
	return cloned
}

func cloneFindRequest(req environment.FindEnvironmentRequest) environment.FindEnvironmentRequest {
	cloned := req
	cloned.LocalAdditionalDirs = append([]string(nil), req.LocalAdditionalDirs...)
	cloned.ProviderState = append(json.RawMessage(nil), req.ProviderState...)
	if req.Labels != nil {
		cloned.Labels = make(map[string]string, len(req.Labels))
		maps.Copy(cloned.Labels, req.Labels)
	}
	return cloned
}

func cloneSessionState(state environment.SessionState) environment.SessionState {
	cloned := state
	cloned.RuntimeAdditionalDirs = append([]string(nil), state.RuntimeAdditionalDirs...)
	cloned.ProviderState = append(json.RawMessage(nil), state.ProviderState...)
	if state.SSHAccessExpiresAt != nil {
		expires := *state.SSHAccessExpiresAt
		cloned.SSHAccessExpiresAt = &expires
	}
	return cloned
}
