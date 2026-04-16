package session

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/environment"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestSessionEnvironmentStartPrepareSyncAndLaunchSequence(t *testing.T) {
	t.Parallel()

	runtimeRoot := filepath.Join(t.TempDir(), "runtime-root")
	runtimeAdditional := []string{filepath.Join(t.TempDir(), "runtime-extra")}
	providerState := json.RawMessage(`{"prepared":true}`)
	var (
		h     *harness
		mu    sync.Mutex
		order []string
	)
	appendOrder := func(entry string) {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, entry)
	}

	provider := &recordingEnvironmentProvider{
		runtimeRoot:       runtimeRoot,
		runtimeAdditional: runtimeAdditional,
		instanceID:        "instance-start",
		providerState:     providerState,
		prepareHook: func(req environment.PrepareRequest) error {
			appendOrder("prepare")
			meta := readMeta(t, store.SessionMetaFile(filepath.Join(h.homePaths.SessionsDir, req.SessionID)))
			if meta.Environment == nil {
				t.Fatal("persisted environment before Prepare = nil, want creating metadata")
			}
			if got, want := meta.Environment.EnvironmentID, "env-1"; got != want {
				t.Fatalf("creating environment id = %q, want %q", got, want)
			}
			if got, want := meta.Environment.State, environmentStateCreating; got != want {
				t.Fatalf("creating environment state = %q, want %q", got, want)
			}
			return nil
		},
		syncToHook: func(state environment.SessionState, opts environment.SyncOptions) (environment.SyncResult, error) {
			appendOrder("sync_to")
			if got, want := opts.Reason, environment.SyncReasonStart; got != want {
				t.Fatalf("SyncToRuntime reason = %q, want %q", got, want)
			}
			if got, want := state.EnvironmentID, "env-1"; got != want {
				t.Fatalf("SyncToRuntime environment id = %q, want %q", got, want)
			}
			if got, want := state.RuntimeRootDir, runtimeRoot; got != want {
				t.Fatalf("SyncToRuntime runtime root = %q, want %q", got, want)
			}
			return environment.SyncResult{}, nil
		},
	}
	registry := newRegistryForProvider(t, provider)
	h = newHarness(t, WithEnvironmentRegistry(registry), WithEnvironmentIDGenerator(sequentialIDGenerator("env")))
	h.driver.startHook = func(opts acp.StartOpts, _ int) (*fakeProcess, error) {
		appendOrder("launch")
		if got, want := opts.Cwd, runtimeRoot; got != want {
			t.Fatalf("StartOpts.Cwd = %q, want runtime root %q", got, want)
		}
		if !reflect.DeepEqual(opts.AdditionalDirs, runtimeAdditional) {
			t.Fatalf("StartOpts.AdditionalDirs = %#v, want %#v", opts.AdditionalDirs, runtimeAdditional)
		}
		if opts.Launcher != nil {
			t.Fatal("fake provider launcher = non-nil, want fake-driver fallback")
		}
		meta := readMeta(t, store.SessionMetaFile(filepath.Join(h.homePaths.SessionsDir, "sess-1")))
		if got, want := meta.Environment.InstanceID, "instance-start"; got != want {
			t.Fatalf("persisted instance id before launch = %q, want %q", got, want)
		}
		assertJSONEqual(t, meta.Environment.ProviderState, providerState, "persisted provider state before launch")
		if meta.Environment.LastSyncAt == nil {
			t.Fatal("persisted LastSyncAt before launch = nil, want sync timestamp")
		}
		return newFakeProcess(opts.AgentName, opts.Command, opts.Cwd, "acp-start"), nil
	}

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspaceID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	if got, want := provider.prepareRequests[0].EnvironmentID, "env-1"; got != want {
		t.Fatalf("PrepareRequest.EnvironmentID = %q, want %q", got, want)
	}
	if got, want := provider.prepareRequests[0].SessionID, session.ID; got != want {
		t.Fatalf("PrepareRequest.SessionID = %q, want %q", got, want)
	}
	if got, want := provider.prepareRequests[0].WorkspaceID, h.workspaceID; got != want {
		t.Fatalf("PrepareRequest.WorkspaceID = %q, want %q", got, want)
	}
	if got, want := orderSnapshot(&mu, order), []string{"prepare", "sync_to", "launch"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("environment order = %#v, want %#v", got, want)
	}
	if info := session.Info(); info.Environment == nil || info.Environment.EnvironmentID != "env-1" {
		t.Fatalf("session.Info().Environment = %#v, want env-1", info.Environment)
	}
}

func TestSessionEnvironmentStopSyncsBeforeRecorderCloseAndDestroyPolicy(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name          string
		destroyOnStop bool
		wantState     string
		wantOrder     []string
	}{
		{
			name:          "keeps environment when destroy on stop is false",
			destroyOnStop: false,
			wantState:     environmentStateStopped,
			wantOrder:     []string{"sync_from", "close"},
		},
		{
			name:          "destroys environment when destroy on stop is true",
			destroyOnStop: true,
			wantState:     environmentStateDestroyed,
			wantOrder:     []string{"sync_from", "destroy", "close"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				mu    sync.Mutex
				order []string
			)
			appendOrder := func(entry string) {
				mu.Lock()
				defer mu.Unlock()
				order = append(order, entry)
			}
			provider := &recordingEnvironmentProvider{
				syncFromHook: func(_ environment.SessionState, opts environment.SyncOptions) (environment.SyncResult, error) {
					appendOrder("sync_from")
					if got, want := opts.Reason, environment.SyncReasonStop; got != want {
						t.Fatalf("SyncFromRuntime reason = %q, want %q", got, want)
					}
					return environment.SyncResult{}, nil
				},
				destroyHook: func(environment.SessionState) error {
					appendOrder("destroy")
					return nil
				},
			}
			h := newHarness(
				t,
				WithEnvironmentRegistry(newRegistryForProvider(t, provider)),
				WithStore(func(context.Context, string, string) (EventRecorder, error) {
					return &orderingRecorder{onClose: func() { appendOrder("close") }}, nil
				}),
			)
			setHarnessEnvironment(t, h, tt.destroyOnStop)
			session := createSession(t, h)

			if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
				t.Fatalf("Stop() error = %v", err)
			}

			if got := orderSnapshot(&mu, order); !reflect.DeepEqual(got, tt.wantOrder) {
				t.Fatalf("stop environment order = %#v, want %#v", got, tt.wantOrder)
			}
			meta := readMeta(t, session.MetaPath())
			if got := meta.Environment.State; got != tt.wantState {
				t.Fatalf("environment state after stop = %q, want %q", got, tt.wantState)
			}
		})
	}
}

func TestSessionEnvironmentCrashSyncIsBestEffort(t *testing.T) {
	t.Parallel()

	syncErr := errors.New("runtime sync failed")
	provider := &recordingEnvironmentProvider{
		syncFromHook: func(_ environment.SessionState, opts environment.SyncOptions) (environment.SyncResult, error) {
			if got, want := opts.Reason, environment.SyncReasonCrash; got != want {
				t.Fatalf("SyncFromRuntime reason = %q, want %q", got, want)
			}
			return environment.SyncResult{}, syncErr
		},
	}
	h := newHarness(t, WithEnvironmentRegistry(newRegistryForProvider(t, provider)))
	session := createSession(t, h)

	h.driver.lastProcess().crash(errors.New("boom"), "stderr trace")
	waitForCondition(t, "session stopped after crash sync failure", func() bool {
		return h.notifier.stoppedCount() == 1
	})

	meta := readMeta(t, session.MetaPath())
	if got, want := meta.State, string(StateStopped); got != want {
		t.Fatalf("session state after crash = %q, want %q", got, want)
	}
	if meta.StopReason == nil || *meta.StopReason != store.StopAgentCrashed {
		t.Fatalf("StopReason after crash = %#v, want agent_crashed", meta.StopReason)
	}
	if got := meta.Environment.LastSyncError; got != syncErr.Error() {
		t.Fatalf("environment LastSyncError = %q, want %q", got, syncErr.Error())
	}
}

func TestSessionEnvironmentResumeRestoresProviderState(t *testing.T) {
	t.Parallel()

	provider := &recordingEnvironmentProvider{}
	h := newHarness(t, WithEnvironmentRegistry(newRegistryForProvider(t, provider)))
	session := createSession(t, h)
	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	meta := readMeta(t, session.MetaPath())
	meta.Environment.EnvironmentID = "env-resume"
	meta.Environment.InstanceID = "instance-resume"
	meta.Environment.ProviderState = json.RawMessage(`{"resume":true}`)
	if err := store.WriteSessionMeta(session.MetaPath(), meta); err != nil {
		t.Fatalf("WriteSessionMeta() error = %v", err)
	}

	resumed, err := h.manager.Resume(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), resumed.ID)
	})

	if got, want := len(provider.prepareRequests), 2; got != want {
		t.Fatalf("Prepare calls = %d, want %d", got, want)
	}
	req := provider.prepareRequests[1]
	if got, want := req.EnvironmentID, "env-resume"; got != want {
		t.Fatalf("resume PrepareRequest.EnvironmentID = %q, want %q", got, want)
	}
	if got, want := req.InstanceID, "instance-resume"; got != want {
		t.Fatalf("resume PrepareRequest.InstanceID = %q, want %q", got, want)
	}
	assertJSONEqual(t, req.ProviderState, json.RawMessage(`{"resume":true}`), "resume PrepareRequest.ProviderState")
}

func TestSessionEnvironmentLifecycleObserverReceivesRequiredFields(t *testing.T) {
	t.Parallel()

	observer := &recordingEnvironmentNotifier{}
	h := newHarness(t, WithNotifier(observer))
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	events := observer.eventsSnapshot()
	if len(events) == 0 {
		t.Fatal("environment lifecycle events = empty, want events")
	}
	for _, event := range events {
		if event.SessionID != session.ID {
			t.Fatalf("event.SessionID = %q, want %q in %#v", event.SessionID, session.ID, event)
		}
		if event.WorkspaceID != h.workspaceID {
			t.Fatalf("event.WorkspaceID = %q, want %q in %#v", event.WorkspaceID, h.workspaceID, event)
		}
		if event.EnvironmentID == "" || event.Backend == "" || event.Profile == "" {
			t.Fatalf("event missing environment fields: %#v", event)
		}
		if event.Name == "" || event.Span == "" {
			t.Fatalf("event missing name/span: %#v", event)
		}
	}
}

func TestSessionEnvironmentHooksDispatchPayloadsAcrossLifecycle(t *testing.T) {
	t.Parallel()

	if err := os.WriteFile(filepath.Join(t.TempDir(), "unrelated.txt"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("WriteFile(unrelated) error = %v", err)
	}
	provider := &recordingEnvironmentProvider{
		instanceID: "instance-hooked",
		syncToResult: environment.SyncResult{
			FilesSynced:      2,
			BytesTransferred: 17,
			Errors:           []string{"provider warning"},
		},
		syncFromResult: environment.SyncResult{
			FilesSynced:      1,
			BytesTransferred: 9,
		},
	}
	hooks := &recordingEnvironmentHooks{}
	h := newHarness(
		t,
		WithEnvironmentRegistry(newRegistryForProvider(t, provider)),
		WithEnvironmentIDGenerator(sequentialIDGenerator("env")),
		WithHookSet(HookSet{Environment: hooks}),
	)
	if err := os.WriteFile(filepath.Join(h.workspace, "tracked.txt"), []byte("workspace"), 0o644); err != nil {
		t.Fatalf("WriteFile(tracked) error = %v", err)
	}

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspaceID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	events := hooks.eventsSnapshot()
	wantStart := []string{
		"environment.prepare",
		"environment.sync.before:to_runtime",
		"environment.sync.after:to_runtime",
		"environment.ready",
	}
	if !reflect.DeepEqual(events, wantStart) {
		t.Fatalf("start environment hook events = %#v, want %#v", events, wantStart)
	}

	prepare := hooks.prepareSnapshot()[0]
	if prepare.EnvironmentID != "env-1" || prepare.WorkspaceID != h.workspaceID || prepare.AgentName != "coder" {
		t.Fatalf("environment.prepare payload = %#v, want env/session context", prepare)
	}
	if prepare.Profile.Profile == "" || prepare.Backend != string(environment.BackendLocal) {
		t.Fatalf("environment.prepare profile/backend = %#v/%q, want local profile", prepare.Profile, prepare.Backend)
	}

	syncBefore := hooks.syncBeforeSnapshot()[0]
	if syncBefore.Direction != string(environment.SyncDirectionToRuntime) ||
		syncBefore.Reason != string(environment.SyncReasonStart) {
		t.Fatalf("sync.before start direction/reason = %q/%q", syncBefore.Direction, syncBefore.Reason)
	}
	if syncBefore.FileCount != 1 {
		t.Fatalf("sync.before file_count = %d, want 1", syncBefore.FileCount)
	}
	syncAfter := hooks.syncAfterSnapshot()[0]
	if syncAfter.FilesSynced != 2 || syncAfter.BytesTransferred != 17 {
		t.Fatalf("sync.after stats = files %d bytes %d, want 2/17", syncAfter.FilesSynced, syncAfter.BytesTransferred)
	}
	if !reflect.DeepEqual(syncAfter.Errors, []string{"provider warning"}) {
		t.Fatalf("sync.after errors = %#v, want provider warning", syncAfter.Errors)
	}
	if syncAfter.DurationMS < 0 {
		t.Fatalf("sync.after duration_ms = %d, want non-negative", syncAfter.DurationMS)
	}
	ready := hooks.readySnapshot()[0]
	if ready.EnvironmentID != "env-1" || ready.InstanceID != "instance-hooked" {
		t.Fatalf("environment.ready payload = %#v, want env-1/instance-hooked", ready)
	}

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	events = hooks.eventsSnapshot()
	wantAll := []string{
		"environment.prepare",
		"environment.sync.before:to_runtime",
		"environment.sync.after:to_runtime",
		"environment.ready",
		"environment.sync.before:from_runtime",
		"environment.sync.after:from_runtime",
		"environment.stop",
	}
	if !reflect.DeepEqual(events, wantAll) {
		t.Fatalf("environment hook events = %#v, want %#v", events, wantAll)
	}
	stop := hooks.stopSnapshot()[0]
	if stop.EnvironmentID != "env-1" || stop.InstanceID != "instance-hooked" {
		t.Fatalf("environment.stop payload = %#v, want env-1/instance-hooked", stop)
	}
}

func TestSessionEnvironmentPrepareHookDenyAbortsSessionCreation(t *testing.T) {
	t.Parallel()

	provider := &recordingEnvironmentProvider{
		prepareHook: func(environment.PrepareRequest) error {
			t.Fatal("Prepare() called after environment.prepare denial")
			return nil
		},
	}
	hooks := &recordingEnvironmentHooks{
		prepareFn: func(
			_ context.Context,
			payload hookspkg.EnvironmentPreparePayload,
		) (hookspkg.EnvironmentPreparePayload, error) {
			payload.Denied = true
			payload.DenyReason = "policy"
			return payload, nil
		},
	}
	h := newHarness(
		t,
		WithEnvironmentRegistry(newRegistryForProvider(t, provider)),
		WithHookSet(HookSet{Environment: hooks}),
	)

	_, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspaceID,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want prepare denied error")
	}
	if !strings.Contains(err.Error(), "environment prepare denied") {
		t.Fatalf("Create() error = %v, want environment prepare denied", err)
	}
	if got := len(provider.prepareRequests); got != 0 {
		t.Fatalf("provider Prepare calls = %d, want 0", got)
	}
}

func TestSessionEnvironmentPrepareHookEnvOverridesMergeIntoEnvironmentConfig(t *testing.T) {
	t.Parallel()

	provider := &recordingEnvironmentProvider{}
	hooks := &recordingEnvironmentHooks{
		prepareFn: func(
			_ context.Context,
			payload hookspkg.EnvironmentPreparePayload,
		) (hookspkg.EnvironmentPreparePayload, error) {
			payload.EnvOverrides = map[string]string{
				"BASE":   "patched",
				"SECRET": "token",
			}
			return payload, nil
		},
	}
	h := newHarness(
		t,
		WithEnvironmentRegistry(newRegistryForProvider(t, provider)),
		WithHookSet(HookSet{Environment: hooks}),
	)
	resolved, err := h.resolver.Resolve(context.Background(), h.workspaceID)
	if err != nil {
		t.Fatalf("Resolve(%q) error = %v", h.workspaceID, err)
	}
	resolved.Environment.Env = map[string]string{"BASE": "original"}
	h.resolver.upsert(&resolved)

	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	req := provider.prepareRequests[0]
	if got := req.Environment.Env["BASE"]; got != "patched" {
		t.Fatalf("PrepareRequest.Environment.Env[BASE] = %q, want patched", got)
	}
	if got := req.Environment.Env["SECRET"]; got != "token" {
		t.Fatalf("PrepareRequest.Environment.Env[SECRET] = %q, want token", got)
	}
	if !stringSliceContains(req.AgentEnv, "BASE=patched") || !stringSliceContains(req.AgentEnv, "SECRET=token") {
		t.Fatalf("PrepareRequest.AgentEnv = %#v, want patched env overrides", req.AgentEnv)
	}
}

func TestSessionEnvironmentSyncBeforeDenySkipsSyncOperation(t *testing.T) {
	t.Parallel()

	provider := &recordingEnvironmentProvider{
		syncToHook: func(environment.SessionState, environment.SyncOptions) (environment.SyncResult, error) {
			t.Fatal("SyncToRuntime() called after environment.sync.before denial")
			return environment.SyncResult{}, nil
		},
	}
	hooks := &recordingEnvironmentHooks{
		syncBeforeFn: func(
			_ context.Context,
			payload hookspkg.EnvironmentSyncBeforePayload,
		) (hookspkg.EnvironmentSyncBeforePayload, error) {
			if payload.Direction == string(environment.SyncDirectionToRuntime) {
				payload.Denied = true
				payload.DenyReason = "skip initial sync"
			}
			return payload, nil
		},
	}
	h := newHarness(
		t,
		WithEnvironmentRegistry(newRegistryForProvider(t, provider)),
		WithHookSet(HookSet{Environment: hooks}),
	)

	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})
	if got := len(provider.syncToReasons); got != 0 {
		t.Fatalf("SyncToRuntime calls = %d, want 0", got)
	}
	if before := hooks.syncBeforeSnapshot()[0]; !before.Denied || before.DenyReason != "skip initial sync" {
		t.Fatalf("sync.before denied payload = %#v, want skip denial", before)
	}
}

func TestSessionEnvironmentSyncBeforeExcludePatternsPassToProvider(t *testing.T) {
	t.Parallel()

	wantPatterns := []string{"node_modules/**", "*.log"}
	provider := &recordingEnvironmentProvider{
		syncToHook: func(_ environment.SessionState, opts environment.SyncOptions) (environment.SyncResult, error) {
			if !reflect.DeepEqual(opts.ExcludePatterns, wantPatterns) {
				t.Fatalf("SyncToRuntime ExcludePatterns = %#v, want %#v", opts.ExcludePatterns, wantPatterns)
			}
			return environment.SyncResult{}, nil
		},
	}
	hooks := &recordingEnvironmentHooks{
		syncBeforeFn: func(
			_ context.Context,
			payload hookspkg.EnvironmentSyncBeforePayload,
		) (hookspkg.EnvironmentSyncBeforePayload, error) {
			if payload.Direction == string(environment.SyncDirectionToRuntime) {
				payload.ExcludePatterns = append([]string(nil), wantPatterns...)
			}
			return payload, nil
		},
	}
	h := newHarness(
		t,
		WithEnvironmentRegistry(newRegistryForProvider(t, provider)),
		WithHookSet(HookSet{Environment: hooks}),
	)

	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})
	if got := len(provider.syncToOptions); got != 1 {
		t.Fatalf("SyncToRuntime calls = %d, want 1", got)
	}
	if !reflect.DeepEqual(provider.syncToOptions[0].ExcludePatterns, wantPatterns) {
		t.Fatalf("recorded ExcludePatterns = %#v, want %#v", provider.syncToOptions[0].ExcludePatterns, wantPatterns)
	}
}

func TestSessionEnvironmentStopDenyPreventsDestroyButStopsSession(t *testing.T) {
	t.Parallel()

	provider := &recordingEnvironmentProvider{
		destroyHook: func(environment.SessionState) error {
			t.Fatal("Destroy() called after environment.stop denial")
			return nil
		},
	}
	hooks := &recordingEnvironmentHooks{
		stopFn: func(
			_ context.Context,
			payload hookspkg.EnvironmentStopPayload,
		) (hookspkg.EnvironmentStopPayload, error) {
			payload.Denied = true
			payload.DenyReason = "retain sandbox"
			return payload, nil
		},
	}
	h := newHarness(
		t,
		WithEnvironmentRegistry(newRegistryForProvider(t, provider)),
		WithHookSet(HookSet{Environment: hooks}),
	)
	setHarnessEnvironment(t, h, true)
	session := createSession(t, h)

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	info, err := h.manager.Status(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Status(%q) error = %v", session.ID, err)
	}
	if info.State != StateStopped {
		t.Fatalf("session state = %q, want stopped", info.State)
	}
	meta := readMeta(t, session.MetaPath())
	if got := meta.Environment.State; got != environmentStateStopped {
		t.Fatalf("environment state = %q, want %q", got, environmentStateStopped)
	}
	if got := len(provider.destroyStates); got != 0 {
		t.Fatalf("Destroy calls = %d, want 0", got)
	}
	stop := hooks.stopSnapshot()[0]
	if !stop.Denied || stop.WillDestroy != true {
		t.Fatalf("environment.stop payload = %#v, want denied with initial will_destroy", stop)
	}
}

func TestManagerExecEnvironmentUsesPreparedToolHost(t *testing.T) {
	t.Parallel()

	runtimeRoot := filepath.Join(t.TempDir(), "runtime")
	toolHost := &recordingEnvironmentToolHost{exitCode: 7, output: "terminal output"}
	provider := &recordingEnvironmentProvider{
		runtimeRoot: runtimeRoot,
		toolHost:    toolHost,
	}
	h := newHarness(t, WithEnvironmentRegistry(newRegistryForProvider(t, provider)))
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	result, err := h.manager.ExecEnvironment(testutil.Context(t), EnvironmentExecRequest{
		SessionID: session.ID,
		Command:   " echo ready ",
		Timeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("ExecEnvironment() error = %v", err)
	}
	if result.ExitCode != 7 || result.Stdout != "terminal output" || result.Stderr != "" {
		t.Fatalf("ExecEnvironment() result = %#v, want exit/output without stderr", result)
	}

	toolHost.mu.Lock()
	defer toolHost.mu.Unlock()
	if len(toolHost.createRequests) != 1 {
		t.Fatalf("CreateTerminal calls = %d, want 1", len(toolHost.createRequests))
	}
	req := toolHost.createRequests[0]
	if req.Command != "echo ready" {
		t.Fatalf("CreateTerminal command = %q, want trimmed command", req.Command)
	}
	if req.Cwd == nil || *req.Cwd != runtimeRoot {
		t.Fatalf("CreateTerminal cwd = %v, want %q", req.Cwd, runtimeRoot)
	}
	if got, want := toolHost.waitIDs, []string{"term-1"}; !slices.Equal(got, want) {
		t.Fatalf("WaitForTerminalExit ids = %v, want %v", got, want)
	}
	if got, want := toolHost.outputIDs, []string{"term-1"}; !slices.Equal(got, want) {
		t.Fatalf("TerminalOutput ids = %v, want %v", got, want)
	}
	if got, want := toolHost.releaseIDs, []string{"term-1"}; !slices.Equal(got, want) {
		t.Fatalf("ReleaseTerminal ids = %v, want %v", got, want)
	}
}

func TestManagerExecEnvironmentValidationErrors(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	tests := []struct {
		name string
		ctx  context.Context
		req  EnvironmentExecRequest
	}{
		{
			name: "nil context",
			req:  EnvironmentExecRequest{SessionID: session.ID, Command: "pwd"},
		},
		{
			name: "blank session id",
			ctx:  testutil.Context(t),
			req:  EnvironmentExecRequest{Command: "pwd"},
		},
		{
			name: "blank command",
			ctx:  testutil.Context(t),
			req:  EnvironmentExecRequest{SessionID: session.ID},
		},
		{
			name: "missing session",
			ctx:  testutil.Context(t),
			req:  EnvironmentExecRequest{SessionID: "missing", Command: "pwd"},
		},
		{
			name: "missing tool host",
			ctx:  testutil.Context(t),
			req:  EnvironmentExecRequest{SessionID: session.ID, Command: "pwd"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := h.manager.ExecEnvironment(tc.ctx, tc.req); err == nil {
				t.Fatal("ExecEnvironment() error = nil, want error")
			}
		})
	}
}

type recordingEnvironmentProvider struct {
	mu                sync.Mutex
	prepareRequests   []environment.PrepareRequest
	syncToReasons     []environment.SyncReason
	syncFromReasons   []environment.SyncReason
	syncToOptions     []environment.SyncOptions
	syncFromOptions   []environment.SyncOptions
	destroyStates     []environment.SessionState
	runtimeRoot       string
	runtimeAdditional []string
	instanceID        string
	providerState     json.RawMessage
	syncToResult      environment.SyncResult
	syncFromResult    environment.SyncResult
	toolHost          environment.ToolHost
	prepareHook       func(environment.PrepareRequest) error
	syncToHook        func(environment.SessionState, environment.SyncOptions) (environment.SyncResult, error)
	syncFromHook      func(environment.SessionState, environment.SyncOptions) (environment.SyncResult, error)
	destroyHook       func(environment.SessionState) error
}

func (p *recordingEnvironmentProvider) Backend() environment.Backend {
	return environment.BackendLocal
}

func (p *recordingEnvironmentProvider) Prepare(
	_ context.Context,
	req environment.PrepareRequest,
) (environment.Prepared, error) {
	p.mu.Lock()
	p.prepareRequests = append(p.prepareRequests, clonePrepareRequest(req))
	p.mu.Unlock()

	if p.prepareHook != nil {
		if err := p.prepareHook(req); err != nil {
			return environment.Prepared{}, err
		}
	}

	runtimeRoot := p.runtimeRoot
	if runtimeRoot == "" {
		runtimeRoot = req.LocalRootDir
	}
	runtimeAdditional := append([]string(nil), p.runtimeAdditional...)
	if runtimeAdditional == nil {
		runtimeAdditional = append([]string(nil), req.LocalAdditionalDirs...)
	}
	instanceID := p.instanceID
	if instanceID == "" {
		instanceID = req.InstanceID
	}
	providerState := append(json.RawMessage(nil), p.providerState...)
	if providerState == nil {
		providerState = append(json.RawMessage(nil), req.ProviderState...)
	}
	state := environment.SessionState{
		EnvironmentID:         req.EnvironmentID,
		Backend:               environment.BackendLocal,
		Profile:               req.Environment.Profile,
		State:                 environmentStatePrepared,
		InstanceID:            instanceID,
		RuntimeRootDir:        runtimeRoot,
		RuntimeAdditionalDirs: append([]string(nil), runtimeAdditional...),
		ProviderState:         providerState,
		PreparedAt:            time.Now().UTC(),
	}
	return environment.Prepared{
		State:                 state,
		RuntimeRootDir:        runtimeRoot,
		RuntimeAdditionalDirs: append([]string(nil), runtimeAdditional...),
		ToolHost:              p.toolHost,
		Launch: environment.LaunchSpec{
			Command:        req.AgentCommand,
			Cwd:            runtimeRoot,
			AdditionalDirs: append([]string(nil), runtimeAdditional...),
			Env:            append([]string(nil), req.AgentEnv...),
		},
	}, nil
}

func (p *recordingEnvironmentProvider) SyncToRuntime(
	_ context.Context,
	state environment.SessionState,
	opts environment.SyncOptions,
) (environment.SyncResult, error) {
	p.mu.Lock()
	p.syncToReasons = append(p.syncToReasons, opts.Reason)
	p.syncToOptions = append(p.syncToOptions, cloneSyncOptions(opts))
	p.mu.Unlock()
	if p.syncToHook != nil {
		return p.syncToHook(state, opts)
	}
	return cloneSyncResult(p.syncToResult), nil
}

func (p *recordingEnvironmentProvider) SyncFromRuntime(
	_ context.Context,
	state environment.SessionState,
	opts environment.SyncOptions,
) (environment.SyncResult, error) {
	p.mu.Lock()
	p.syncFromReasons = append(p.syncFromReasons, opts.Reason)
	p.syncFromOptions = append(p.syncFromOptions, cloneSyncOptions(opts))
	p.mu.Unlock()
	if p.syncFromHook != nil {
		return p.syncFromHook(state, opts)
	}
	return cloneSyncResult(p.syncFromResult), nil
}

func (p *recordingEnvironmentProvider) Destroy(
	_ context.Context,
	state environment.SessionState,
) error {
	p.mu.Lock()
	p.destroyStates = append(p.destroyStates, state)
	p.mu.Unlock()
	if p.destroyHook != nil {
		return p.destroyHook(state)
	}
	return nil
}

type recordingEnvironmentToolHost struct {
	mu             sync.Mutex
	createRequests []acpsdk.CreateTerminalRequest
	waitIDs        []string
	outputIDs      []string
	releaseIDs     []string
	exitCode       int
	output         string
}

func (h *recordingEnvironmentToolHost) ReadTextFile(context.Context, string) (string, error) {
	return "", errors.New("test: ReadTextFile not implemented")
}

func (h *recordingEnvironmentToolHost) WriteTextFile(context.Context, string, string) error {
	return errors.New("test: WriteTextFile not implemented")
}

func (h *recordingEnvironmentToolHost) ResolvePath(path string) (string, error) {
	return path, nil
}

func (h *recordingEnvironmentToolHost) Authorize(environment.PermissionOperation) error {
	return nil
}

func (h *recordingEnvironmentToolHost) PermissionDecision(
	acpsdk.RequestPermissionRequest,
) (environment.PermissionDecision, bool) {
	return "", false
}

func (h *recordingEnvironmentToolHost) CreateTerminal(
	_ context.Context,
	req acpsdk.CreateTerminalRequest,
) (acpsdk.CreateTerminalResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.createRequests = append(h.createRequests, req)
	return acpsdk.CreateTerminalResponse{TerminalId: "term-1"}, nil
}

func (h *recordingEnvironmentToolHost) KillTerminal(string) error {
	return nil
}

func (h *recordingEnvironmentToolHost) TerminalOutput(id string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.outputIDs = append(h.outputIDs, id)
	return h.output, nil
}

func (h *recordingEnvironmentToolHost) WaitForTerminalExit(_ context.Context, id string) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.waitIDs = append(h.waitIDs, id)
	return h.exitCode, nil
}

func (h *recordingEnvironmentToolHost) ReleaseTerminal(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.releaseIDs = append(h.releaseIDs, id)
	return nil
}

type recordingEnvironmentHooks struct {
	mu           sync.Mutex
	events       []string
	prepare      []hookspkg.EnvironmentPreparePayload
	ready        []hookspkg.EnvironmentReadyPayload
	syncBefore   []hookspkg.EnvironmentSyncBeforePayload
	syncAfter    []hookspkg.EnvironmentSyncAfterPayload
	stop         []hookspkg.EnvironmentStopPayload
	prepareFn    func(context.Context, hookspkg.EnvironmentPreparePayload) (hookspkg.EnvironmentPreparePayload, error)
	readyFn      func(context.Context, hookspkg.EnvironmentReadyPayload) (hookspkg.EnvironmentReadyPayload, error)
	syncBeforeFn func(context.Context, hookspkg.EnvironmentSyncBeforePayload) (hookspkg.EnvironmentSyncBeforePayload, error)
	syncAfterFn  func(context.Context, hookspkg.EnvironmentSyncAfterPayload) (hookspkg.EnvironmentSyncAfterPayload, error)
	stopFn       func(context.Context, hookspkg.EnvironmentStopPayload) (hookspkg.EnvironmentStopPayload, error)
}

func (h *recordingEnvironmentHooks) DispatchEnvironmentPrepare(
	ctx context.Context,
	payload hookspkg.EnvironmentPreparePayload,
) (hookspkg.EnvironmentPreparePayload, error) {
	result := payload
	var err error
	if h.prepareFn != nil {
		result, err = h.prepareFn(ctx, payload)
	}
	h.mu.Lock()
	h.events = append(h.events, string(hookspkg.HookEnvironmentPrepare))
	h.prepare = append(h.prepare, cloneEnvironmentPreparePayload(result))
	h.mu.Unlock()
	return result, err
}

func (h *recordingEnvironmentHooks) DispatchEnvironmentReady(
	ctx context.Context,
	payload hookspkg.EnvironmentReadyPayload,
) (hookspkg.EnvironmentReadyPayload, error) {
	result := payload
	var err error
	if h.readyFn != nil {
		result, err = h.readyFn(ctx, payload)
	}
	h.mu.Lock()
	h.events = append(h.events, string(hookspkg.HookEnvironmentReady))
	h.ready = append(h.ready, cloneEnvironmentReadyPayload(result))
	h.mu.Unlock()
	return result, err
}

func (h *recordingEnvironmentHooks) DispatchEnvironmentSyncBefore(
	ctx context.Context,
	payload hookspkg.EnvironmentSyncBeforePayload,
) (hookspkg.EnvironmentSyncBeforePayload, error) {
	result := payload
	var err error
	if h.syncBeforeFn != nil {
		result, err = h.syncBeforeFn(ctx, payload)
	}
	h.mu.Lock()
	h.events = append(h.events, string(hookspkg.HookEnvironmentSyncBefore)+":"+result.Direction)
	h.syncBefore = append(h.syncBefore, cloneEnvironmentSyncBeforePayload(result))
	h.mu.Unlock()
	return result, err
}

func (h *recordingEnvironmentHooks) DispatchEnvironmentSyncAfter(
	ctx context.Context,
	payload hookspkg.EnvironmentSyncAfterPayload,
) (hookspkg.EnvironmentSyncAfterPayload, error) {
	result := payload
	var err error
	if h.syncAfterFn != nil {
		result, err = h.syncAfterFn(ctx, payload)
	}
	h.mu.Lock()
	h.events = append(h.events, string(hookspkg.HookEnvironmentSyncAfter)+":"+result.Direction)
	h.syncAfter = append(h.syncAfter, cloneEnvironmentSyncAfterPayload(result))
	h.mu.Unlock()
	return result, err
}

func (h *recordingEnvironmentHooks) DispatchEnvironmentStop(
	ctx context.Context,
	payload hookspkg.EnvironmentStopPayload,
) (hookspkg.EnvironmentStopPayload, error) {
	result := payload
	var err error
	if h.stopFn != nil {
		result, err = h.stopFn(ctx, payload)
	}
	h.mu.Lock()
	h.events = append(h.events, string(hookspkg.HookEnvironmentStop))
	h.stop = append(h.stop, result)
	h.mu.Unlock()
	return result, err
}

func (h *recordingEnvironmentHooks) eventsSnapshot() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.events...)
}

func (h *recordingEnvironmentHooks) prepareSnapshot() []hookspkg.EnvironmentPreparePayload {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]hookspkg.EnvironmentPreparePayload(nil), h.prepare...)
}

func (h *recordingEnvironmentHooks) readySnapshot() []hookspkg.EnvironmentReadyPayload {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]hookspkg.EnvironmentReadyPayload(nil), h.ready...)
}

func (h *recordingEnvironmentHooks) syncBeforeSnapshot() []hookspkg.EnvironmentSyncBeforePayload {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]hookspkg.EnvironmentSyncBeforePayload(nil), h.syncBefore...)
}

func (h *recordingEnvironmentHooks) syncAfterSnapshot() []hookspkg.EnvironmentSyncAfterPayload {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]hookspkg.EnvironmentSyncAfterPayload(nil), h.syncAfter...)
}

func (h *recordingEnvironmentHooks) stopSnapshot() []hookspkg.EnvironmentStopPayload {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]hookspkg.EnvironmentStopPayload(nil), h.stop...)
}

type orderingRecorder struct {
	onClose func()
}

func (r *orderingRecorder) Record(context.Context, store.SessionEvent) error {
	return nil
}

func (r *orderingRecorder) RecordTokenUsage(context.Context, store.TokenUsage) error {
	return nil
}

func (r *orderingRecorder) Query(context.Context, store.EventQuery) ([]store.SessionEvent, error) {
	return nil, nil
}

func (r *orderingRecorder) History(context.Context, store.EventQuery) ([]store.TurnHistory, error) {
	return nil, nil
}

func (r *orderingRecorder) Close(context.Context) error {
	if r.onClose != nil {
		r.onClose()
	}
	return nil
}

type recordingEnvironmentNotifier struct {
	mu     sync.Mutex
	events []EnvironmentLifecycleEvent
}

func (n *recordingEnvironmentNotifier) OnSessionCreated(context.Context, *Session) {}

func (n *recordingEnvironmentNotifier) OnSessionStopped(context.Context, *Session) {}

func (n *recordingEnvironmentNotifier) OnAgentEvent(context.Context, string, any) {}

func (n *recordingEnvironmentNotifier) OnEnvironmentLifecycleEvent(
	_ context.Context,
	event EnvironmentLifecycleEvent,
) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.events = append(n.events, event)
}

func (n *recordingEnvironmentNotifier) eventsSnapshot() []EnvironmentLifecycleEvent {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]EnvironmentLifecycleEvent(nil), n.events...)
}

func newRegistryForProvider(t *testing.T, provider environment.Provider) *environment.Registry {
	t.Helper()
	registry, err := environment.NewRegistry(provider)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	return registry
}

func setHarnessEnvironment(t *testing.T, h *harness, destroyOnStop bool) {
	t.Helper()
	resolved, err := h.resolver.Resolve(context.Background(), h.workspaceID)
	if err != nil {
		t.Fatalf("Resolve(%q) error = %v", h.workspaceID, err)
	}
	resolved.Environment.DestroyOnStop = destroyOnStop
	h.resolver.upsert(&resolved)
}

func clonePrepareRequest(req environment.PrepareRequest) environment.PrepareRequest {
	cloned := req
	cloned.LocalAdditionalDirs = append([]string(nil), req.LocalAdditionalDirs...)
	cloned.AgentEnv = append([]string(nil), req.AgentEnv...)
	cloned.ProviderState = append(json.RawMessage(nil), req.ProviderState...)
	cloned.Environment.Env = cloneStringMapForEnvironmentTests(req.Environment.Env)
	return cloned
}

func cloneSyncOptions(opts environment.SyncOptions) environment.SyncOptions {
	cloned := opts
	cloned.ExcludePatterns = append([]string(nil), opts.ExcludePatterns...)
	return cloned
}

func cloneSyncResult(result environment.SyncResult) environment.SyncResult {
	cloned := result
	cloned.Errors = append([]string(nil), result.Errors...)
	return cloned
}

func cloneEnvironmentPreparePayload(
	payload hookspkg.EnvironmentPreparePayload,
) hookspkg.EnvironmentPreparePayload {
	cloned := payload
	cloned.LocalAdditionalDirs = append([]string(nil), payload.LocalAdditionalDirs...)
	cloned.AgentEnv = append([]string(nil), payload.AgentEnv...)
	cloned.EnvOverrides = cloneStringMapForEnvironmentTests(payload.EnvOverrides)
	cloned.Profile.Env = cloneStringMapForEnvironmentTests(payload.Profile.Env)
	return cloned
}

func cloneEnvironmentReadyPayload(payload hookspkg.EnvironmentReadyPayload) hookspkg.EnvironmentReadyPayload {
	cloned := payload
	cloned.RuntimeAdditionalDirs = append([]string(nil), payload.RuntimeAdditionalDirs...)
	return cloned
}

func cloneEnvironmentSyncBeforePayload(
	payload hookspkg.EnvironmentSyncBeforePayload,
) hookspkg.EnvironmentSyncBeforePayload {
	cloned := payload
	cloned.ExcludePatterns = append([]string(nil), payload.ExcludePatterns...)
	return cloned
}

func cloneEnvironmentSyncAfterPayload(
	payload hookspkg.EnvironmentSyncAfterPayload,
) hookspkg.EnvironmentSyncAfterPayload {
	cloned := payload
	cloned.Errors = append([]string(nil), payload.Errors...)
	return cloned
}

func cloneStringMapForEnvironmentTests(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	maps.Copy(cloned, values)
	return cloned
}

func stringSliceContains(values []string, want string) bool {
	return slices.Contains(values, want)
}

func orderSnapshot(mu *sync.Mutex, order []string) []string {
	mu.Lock()
	defer mu.Unlock()
	return append([]string(nil), order...)
}

func assertJSONEqual(t *testing.T, got json.RawMessage, want json.RawMessage, label string) {
	t.Helper()
	var gotValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("%s invalid JSON %s: %v", label, string(got), err)
	}
	var wantValue any
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("%s invalid expected JSON %s: %v", label, string(want), err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("%s = %#v, want %#v", label, gotValue, wantValue)
	}
}

var _ environment.Provider = (*recordingEnvironmentProvider)(nil)
var _ EventRecorder = (*orderingRecorder)(nil)
var _ EnvironmentLifecycleNotifier = (*recordingEnvironmentNotifier)(nil)
var _ EnvironmentHooks = (*recordingEnvironmentHooks)(nil)
var _ workspacepkg.RuntimeResolver = (*fakeWorkspaceResolver)(nil)
