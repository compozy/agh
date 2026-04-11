package daemon

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	channelspkg "github.com/pedronauck/agh/internal/channels"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/testutil"
)

type recordingChannelSecretResolver struct {
	values map[string]string
	calls  []channelspkg.ChannelSecretBinding
	err    error
}

func (r *recordingChannelSecretResolver) ResolveChannelSecret(_ context.Context, binding channelspkg.ChannelSecretBinding) (string, error) {
	r.calls = append(r.calls, binding)
	if r.err != nil {
		return "", r.err
	}
	if value, ok := r.values[binding.BindingName]; ok {
		return value, nil
	}
	return "resolved-" + binding.BindingName, nil
}

func TestComposeChannelRuntime(t *testing.T) {
	t.Run("ShouldReturnNilWhenRegistryDoesNotSupportChannelPersistence", func(t *testing.T) {
		t.Parallel()

		homePaths := testHomePaths(t)
		cfg := testConfig(t, homePaths)
		d := newTestDaemon(t, homePaths, cfg)
		d.now = func() time.Time {
			return time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
		}

		state := &bootState{
			logger:   discardLogger(),
			registry: &recordingRegistry{path: homePaths.DatabaseFile},
		}
		if runtime := d.composeChannelRuntime(state, &bootCleanup{}); runtime != nil {
			t.Fatalf("composeChannelRuntime(recordingRegistry) = %#v, want nil", runtime)
		}
	})

	t.Run("ShouldBuildRuntimeWhenRegistrySupportsChannelPersistence", func(t *testing.T) {
		t.Parallel()

		homePaths := testHomePaths(t)
		cfg := testConfig(t, homePaths)
		d := newTestDaemon(t, homePaths, cfg)
		d.now = func() time.Time {
			return time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
		}

		db := openDaemonTestGlobalDB(t)
		state := &bootState{
			logger:   discardLogger(),
			registry: db,
		}

		runtime := d.composeChannelRuntime(state, &bootCleanup{})
		if runtime == nil {
			t.Fatal("composeChannelRuntime(globaldb) = nil, want non-nil")
		}
		if runtime.Broker() == nil {
			t.Fatal("composeChannelRuntime(globaldb) broker = nil, want non-nil")
		}
		if runtime.store != db {
			t.Fatalf("composeChannelRuntime(globaldb) store = %#v, want global db", runtime.store)
		}
	})
}

func TestWithChannelSecretResolver(t *testing.T) {
	t.Run("ShouldStoreResolverOnDaemon", func(t *testing.T) {
		t.Parallel()

		resolver := &recordingChannelSecretResolver{}
		d := &Daemon{}

		WithChannelSecretResolver(resolver)(d)

		if d.channelSecretResolver != resolver {
			t.Fatalf("WithChannelSecretResolver() stored %#v, want %#v", d.channelSecretResolver, resolver)
		}
	})
}

func TestBootExtensions(t *testing.T) {
	t.Run("ShouldInjectChannelRuntimeDependencies", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 12, 15, 0, 0, time.UTC)
		channels := newChannelRuntime(db, discardLogger(), func() time.Time { return now }, nil)
		if channels == nil {
			t.Fatal("newChannelRuntime() = nil, want non-nil")
		}

		manager := &fakeExtensionRuntime{}
		var captured extensionManagerDeps

		homePaths := testHomePaths(t)
		cfg := testConfig(t, homePaths)
		d := newTestDaemon(t, homePaths, cfg)
		d.newExtensionManager = func(deps extensionManagerDeps) extensionRuntime {
			captured = deps
			return manager
		}

		state := &bootState{
			logger:            discardLogger(),
			registry:          db,
			sessions:          &fakeSessionManager{},
			observer:          &fakeObserver{},
			workspaceResolver: nil,
			channels:          channels,
		}

		if err := d.bootExtensions(testutil.Context(t), state, &bootCleanup{}); err != nil {
			t.Fatalf("bootExtensions() error = %v", err)
		}

		if captured.ChannelRegistry != channels {
			t.Fatalf("extension deps channel registry = %#v, want channel runtime", captured.ChannelRegistry)
		}
		if captured.ChannelDedupStore == nil {
			t.Fatal("extension deps channel dedup store = nil, want non-nil")
		}
		if captured.ChannelBroker != channels.Broker() {
			t.Fatalf("extension deps channel broker = %#v, want runtime broker", captured.ChannelBroker)
		}
		if captured.ChannelRuntime != channels {
			t.Fatalf("extension deps channel runtime = %#v, want channel runtime", captured.ChannelRuntime)
		}
		if manager.startCount != 1 {
			t.Fatalf("extension manager start count = %d, want 1", manager.startCount)
		}
	})
}

func TestChannelRuntimeStartInstance(t *testing.T) {
	t.Run("ShouldReturnNilRuntimeWhenStoreIsMissing", func(t *testing.T) {
		t.Parallel()

		if runtime := newChannelRuntime(nil, nil, nil, nil); runtime != nil {
			t.Fatalf("newChannelRuntime(nil store) = %#v, want nil", runtime)
		}
	})

	t.Run("ShouldHandleNilBrokerAccess", func(t *testing.T) {
		t.Parallel()

		var nilRuntime *channelRuntime
		if broker := nilRuntime.Broker(); broker != nil {
			t.Fatalf("(*channelRuntime)(nil).Broker() = %#v, want nil", broker)
		}
		nilRuntime.Close()
	})

	t.Run("ShouldTransitionDisabledInstanceToStarting", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		runtime := newChannelRuntime(db, nil, nil, nil)
		extensions := &fakeExtensionRuntime{}
		runtime.setExtensionRuntime(extensions)

		instance := mustCreateDaemonChannelInstance(t, runtime, channelspkg.CreateInstanceRequest{
			ID:            "chan-start",
			Scope:         channelspkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-start",
			DisplayName:   "Start Channel",
			Enabled:       false,
			Status:        channelspkg.ChannelStatusDisabled,
			RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
		})

		updated, err := runtime.StartInstance(testutil.Context(t), instance.ID)
		if err != nil {
			t.Fatalf("StartInstance() error = %v", err)
		}
		if !updated.Enabled {
			t.Fatal("StartInstance() left instance disabled, want enabled")
		}
		if got, want := updated.Status, channelspkg.ChannelStatusStarting; got != want {
			t.Fatalf("StartInstance().Status = %q, want %q", got, want)
		}
		if got, want := extensions.reloadCount, 1; got != want {
			t.Fatalf("extension reload count = %d, want %d", got, want)
		}
		if runtime.Broker() == nil {
			t.Fatal("newChannelRuntime() broker = nil, want non-nil")
		}
	})
}

func TestChannelRuntimeCreateInstance(t *testing.T) {
	t.Run("ShouldReloadExtensionsWhenEnabled", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 12, 20, 0, 0, time.UTC)
		runtime := newChannelRuntime(db, discardLogger(), func() time.Time { return now }, nil)
		extensions := &fakeExtensionRuntime{}
		runtime.setExtensionRuntime(extensions)

		created, err := runtime.CreateInstance(testutil.Context(t), channelspkg.CreateInstanceRequest{
			ID:            "chan-create",
			Scope:         channelspkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-create",
			DisplayName:   "Create Channel",
			Enabled:       true,
			Status:        channelspkg.ChannelStatusStarting,
			RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
		})
		if err != nil {
			t.Fatalf("CreateInstance() error = %v", err)
		}
		if created == nil {
			t.Fatal("CreateInstance() = nil, want non-nil")
		}
		if got, want := extensions.reloadCount, 1; got != want {
			t.Fatalf("extension reload count after enabled create = %d, want %d", got, want)
		}
	})

	t.Run("ShouldRollBackToDisabledWhenReloadFails", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 12, 21, 0, 0, time.UTC)
		reloadErr := errors.New("reload boom")
		extensions := &fakeExtensionRuntime{reloadErr: reloadErr}
		runtime := newChannelRuntime(db, discardLogger(), func() time.Time { return now }, nil)
		runtime.setExtensionRuntime(extensions)

		_, err := runtime.CreateInstance(testutil.Context(t), channelspkg.CreateInstanceRequest{
			ID:            "chan-create-rollback",
			Scope:         channelspkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-create-rollback",
			DisplayName:   "Create Rollback Channel",
			Enabled:       true,
			Status:        channelspkg.ChannelStatusStarting,
			RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
		})
		if !errors.Is(err, reloadErr) {
			t.Fatalf("CreateInstance() error = %v, want wrapped reload failure", err)
		}
		if got, want := extensions.reloadCount, 1; got != want {
			t.Fatalf("extension reload count after failed create = %d, want %d", got, want)
		}

		created, getErr := runtime.GetInstance(testutil.Context(t), "chan-create-rollback")
		if getErr != nil {
			t.Fatalf("GetInstance() error = %v", getErr)
		}
		if created.Enabled {
			t.Fatal("GetInstance() after failed create left instance enabled, want disabled rollback")
		}
		if got, want := created.Status, channelspkg.ChannelStatusDisabled; got != want {
			t.Fatalf("GetInstance().Status after failed create = %q, want %q", got, want)
		}
	})
}

func TestChannelRuntimeResolveChannelRuntime(t *testing.T) {
	t.Run("ShouldResolveBoundSecrets", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 12, 30, 0, 0, time.UTC)
		resolver := &recordingChannelSecretResolver{
			values: map[string]string{
				"bot_token": "secret-value",
			},
		}

		runtime := newChannelRuntime(db, discardLogger(), func() time.Time { return now }, resolver)
		instance := mustCreateDaemonChannelInstance(t, runtime, channelspkg.CreateInstanceRequest{
			ID:            "chan-secret",
			Scope:         channelspkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-secret",
			DisplayName:   "Secret Channel",
			Enabled:       true,
			Status:        channelspkg.ChannelStatusReady,
			RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
		})
		if err := db.PutChannelSecretBinding(testutil.Context(t), channelspkg.ChannelSecretBinding{
			ChannelInstanceID: instance.ID,
			BindingName:       "bot_token",
			VaultRef:          "vault://channels/ext-secret/bot-token",
			Kind:              "bot_token",
			CreatedAt:         now,
			UpdatedAt:         now,
		}); err != nil {
			t.Fatalf("PutChannelSecretBinding() error = %v", err)
		}

		launch, err := runtime.ResolveChannelRuntime(testutil.Context(t), "ext-secret")
		if err != nil {
			t.Fatalf("ResolveChannelRuntime() error = %v", err)
		}
		if launch == nil {
			t.Fatal("ResolveChannelRuntime() = nil, want non-nil")
		}
		if got, want := launch.Instance.ID, instance.ID; got != want {
			t.Fatalf("ResolveChannelRuntime().Instance.ID = %q, want %q", got, want)
		}
		if got := launch.BoundSecrets; len(got) != 1 || got[0].BindingName != "bot_token" || got[0].Value != "secret-value" {
			t.Fatalf("ResolveChannelRuntime().BoundSecrets = %#v, want resolved bot_token binding", got)
		}
		if len(resolver.calls) != 1 || resolver.calls[0].BindingName != "bot_token" {
			t.Fatalf("ResolveChannelSecret() calls = %#v, want bot_token binding", resolver.calls)
		}

		updated, err := runtime.GetInstance(testutil.Context(t), instance.ID)
		if err != nil {
			t.Fatalf("GetInstance() error = %v", err)
		}
		if got, want := updated.Status, channelspkg.ChannelStatusStarting; got != want {
			t.Fatalf("instance status after launch = %q, want %q", got, want)
		}
	})

	t.Run("ShouldRequireSecretResolverWhenBindingsExist", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 12, 35, 0, 0, time.UTC)
		runtime := newChannelRuntime(db, discardLogger(), func() time.Time { return now }, nil)
		instance := mustCreateDaemonChannelInstance(t, runtime, channelspkg.CreateInstanceRequest{
			ID:            "chan-secret-missing",
			Scope:         channelspkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-secret-missing",
			DisplayName:   "Secret Missing",
			Enabled:       true,
			Status:        channelspkg.ChannelStatusReady,
			RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
		})
		if err := db.PutChannelSecretBinding(testutil.Context(t), channelspkg.ChannelSecretBinding{
			ChannelInstanceID: instance.ID,
			BindingName:       "bot_token",
			VaultRef:          "vault://channels/ext-secret-missing/bot-token",
			Kind:              "bot_token",
			CreatedAt:         now,
			UpdatedAt:         now,
		}); err != nil {
			t.Fatalf("PutChannelSecretBinding() error = %v", err)
		}

		_, err := runtime.ResolveChannelRuntime(testutil.Context(t), instance.ExtensionName)
		if !errors.Is(err, errChannelSecretResolverRequired) {
			t.Fatalf("ResolveChannelRuntime() error = %v, want missing secret resolver sentinel", err)
		}
	})

	t.Run("ShouldNotPersistStartingWhenSecretResolutionFails", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 12, 36, 0, 0, time.UTC)
		resolverErr := errors.New("vault boom")
		resolver := &recordingChannelSecretResolver{err: resolverErr}
		runtime := newChannelRuntime(db, discardLogger(), func() time.Time { return now }, resolver)

		instance := mustCreateDaemonChannelInstance(t, runtime, channelspkg.CreateInstanceRequest{
			ID:            "chan-secret-fail",
			Scope:         channelspkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-secret-fail",
			DisplayName:   "Secret Failure",
			Enabled:       true,
			Status:        channelspkg.ChannelStatusReady,
			RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
		})
		if err := db.PutChannelSecretBinding(testutil.Context(t), channelspkg.ChannelSecretBinding{
			ChannelInstanceID: instance.ID,
			BindingName:       "bot_token",
			VaultRef:          "vault://channels/ext-secret-fail/bot-token",
			Kind:              "bot_token",
			CreatedAt:         now,
			UpdatedAt:         now,
		}); err != nil {
			t.Fatalf("PutChannelSecretBinding() error = %v", err)
		}

		_, err := runtime.ResolveChannelRuntime(testutil.Context(t), instance.ExtensionName)
		if !errors.Is(err, resolverErr) {
			t.Fatalf("ResolveChannelRuntime() error = %v, want wrapped resolver error", err)
		}

		updated, getErr := runtime.GetInstance(testutil.Context(t), instance.ID)
		if getErr != nil {
			t.Fatalf("GetInstance() error = %v", getErr)
		}
		if got, want := updated.Status, channelspkg.ChannelStatusReady; got != want {
			t.Fatalf("instance status after failed secret resolution = %q, want %q", got, want)
		}
	})

	t.Run("ShouldDeferWhenNoEnabledInstanceExistsForExtension", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		runtime := newChannelRuntime(db, discardLogger(), nil, nil)

		_, err := runtime.ResolveChannelRuntime(testutil.Context(t), "ext-missing")
		if !errors.Is(err, extensionpkg.ErrChannelRuntimeDeferred) {
			t.Fatalf("ResolveChannelRuntime() error = %v, want deferred sentinel", err)
		}
	})
}

func TestChannelRuntimeStopInstance(t *testing.T) {
	t.Run("ShouldBlockIngressAndPreserveRoutes", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 12, 45, 0, 0, time.UTC)
		runtime := newChannelRuntime(db, discardLogger(), func() time.Time { return now }, nil)
		extensions := &fakeExtensionRuntime{}
		runtime.setExtensionRuntime(extensions)

		instance := mustCreateDaemonChannelInstance(t, runtime, channelspkg.CreateInstanceRequest{
			ID:            "chan-stop",
			Scope:         channelspkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-stop",
			DisplayName:   "Stop Channel",
			Enabled:       true,
			Status:        channelspkg.ChannelStatusReady,
			RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
		})
		extensions.reloadCount = 0
		route := mustUpsertDaemonChannelRoute(t, runtime, channelspkg.ChannelRoute{
			Scope:             channelspkg.ScopeGlobal,
			ChannelInstanceID: instance.ID,
			PeerID:            "peer-stop",
			SessionID:         "sess-stop",
			AgentName:         "coder",
			LastActivityAt:    now,
		})

		updated, err := runtime.StopInstance(testutil.Context(t), instance.ID)
		if err != nil {
			t.Fatalf("StopInstance() error = %v", err)
		}
		if updated.Enabled {
			t.Fatal("StopInstance() left instance enabled, want disabled")
		}
		if got, want := updated.Status, channelspkg.ChannelStatusDisabled; got != want {
			t.Fatalf("StopInstance().Status = %q, want %q", got, want)
		}

		routes, err := runtime.ListRoutes(testutil.Context(t), instance.ID)
		if err != nil {
			t.Fatalf("ListRoutes() error = %v", err)
		}
		if len(routes) != 1 || routes[0].RoutingKeyHash != route.RoutingKeyHash {
			t.Fatalf("ListRoutes() = %#v, want preserved route %#v", routes, route)
		}

		if _, err := runtime.ResolveRoute(testutil.Context(t), route.RoutingKey()); !errors.Is(err, channelspkg.ErrChannelInstanceUnavailable) {
			t.Fatalf("ResolveRoute(disabled) error = %v, want ErrChannelInstanceUnavailable", err)
		}
	})
}

func TestChannelRuntimeRestartInstance(t *testing.T) {
	t.Run("ShouldPreserveRoutesAndReloadExtensions", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 13, 0, 0, 0, time.UTC)
		runtime := newChannelRuntime(db, discardLogger(), func() time.Time { return now }, nil)
		extensions := &fakeExtensionRuntime{}
		runtime.setExtensionRuntime(extensions)

		instance := mustCreateDaemonChannelInstance(t, runtime, channelspkg.CreateInstanceRequest{
			ID:            "chan-restart",
			Scope:         channelspkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-restart",
			DisplayName:   "Restart Channel",
			Enabled:       true,
			Status:        channelspkg.ChannelStatusReady,
			RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
		})
		extensions.reloadCount = 0
		route := mustUpsertDaemonChannelRoute(t, runtime, channelspkg.ChannelRoute{
			Scope:             channelspkg.ScopeGlobal,
			ChannelInstanceID: instance.ID,
			PeerID:            "peer-restart",
			SessionID:         "sess-restart",
			AgentName:         "coder",
			LastActivityAt:    now,
		})

		updated, err := runtime.RestartInstance(testutil.Context(t), instance.ID)
		if err != nil {
			t.Fatalf("RestartInstance() error = %v", err)
		}
		if got, want := extensions.reloadCount, 1; got != want {
			t.Fatalf("extension reload count = %d, want %d", got, want)
		}
		if !updated.Enabled {
			t.Fatal("RestartInstance() disabled the instance, want enabled")
		}
		if got, want := updated.Status, channelspkg.ChannelStatusStarting; got != want {
			t.Fatalf("RestartInstance().Status = %q, want %q", got, want)
		}

		resolved, err := runtime.ResolveRoute(testutil.Context(t), route.RoutingKey())
		if err != nil {
			t.Fatalf("ResolveRoute(after restart) error = %v", err)
		}
		if got, want := resolved.RoutingKeyHash, route.RoutingKeyHash; got != want {
			t.Fatalf("ResolveRoute(after restart).RoutingKeyHash = %q, want %q", got, want)
		}

		routes, err := runtime.ListRoutes(testutil.Context(t), instance.ID)
		if err != nil {
			t.Fatalf("ListRoutes(after restart) error = %v", err)
		}
		if len(routes) != 1 || routes[0].RoutingKeyHash != route.RoutingKeyHash {
			t.Fatalf("ListRoutes(after restart) = %#v, want preserved route %#v", routes, route)
		}
	})
}

func TestChannelRuntimeTransition(t *testing.T) {
	t.Run("ShouldRestorePreviousStateWhenReloadFails", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 4, 11, 13, 5, 0, 0, time.UTC)

		testCases := []struct {
			name       string
			request    channelspkg.CreateInstanceRequest
			transition func(*channelRuntime, context.Context, string) (*channelspkg.ChannelInstance, error)
			wantState  channelspkg.ChannelStatus
			wantEnable bool
		}{
			{
				name: "ShouldRollbackStart",
				request: channelspkg.CreateInstanceRequest{
					ID:            "chan-start-rollback",
					Scope:         channelspkg.ScopeGlobal,
					Platform:      "slack",
					ExtensionName: "ext-start-rollback",
					DisplayName:   "Start Rollback",
					Enabled:       false,
					Status:        channelspkg.ChannelStatusDisabled,
					RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
				},
				transition: func(runtime *channelRuntime, ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
					return runtime.StartInstance(ctx, id)
				},
				wantState:  channelspkg.ChannelStatusDisabled,
				wantEnable: false,
			},
			{
				name: "ShouldRollbackStop",
				request: channelspkg.CreateInstanceRequest{
					ID:            "chan-stop-rollback",
					Scope:         channelspkg.ScopeGlobal,
					Platform:      "slack",
					ExtensionName: "ext-stop-rollback",
					DisplayName:   "Stop Rollback",
					Enabled:       true,
					Status:        channelspkg.ChannelStatusReady,
					RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
				},
				transition: func(runtime *channelRuntime, ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
					return runtime.StopInstance(ctx, id)
				},
				wantState:  channelspkg.ChannelStatusReady,
				wantEnable: true,
			},
			{
				name: "ShouldRollbackRestart",
				request: channelspkg.CreateInstanceRequest{
					ID:            "chan-restart-rollback",
					Scope:         channelspkg.ScopeGlobal,
					Platform:      "slack",
					ExtensionName: "ext-restart-rollback",
					DisplayName:   "Restart Rollback",
					Enabled:       true,
					Status:        channelspkg.ChannelStatusReady,
					RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
				},
				transition: func(runtime *channelRuntime, ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
					return runtime.RestartInstance(ctx, id)
				},
				wantState:  channelspkg.ChannelStatusReady,
				wantEnable: true,
			},
		}

		for _, tt := range testCases {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				db := openDaemonTestGlobalDB(t)
				reloadErr := errors.New("reload boom")
				runtime := newChannelRuntime(db, discardLogger(), func() time.Time { return now }, nil)

				instance := mustCreateDaemonChannelInstance(t, runtime, tt.request)
				runtime.setExtensionRuntime(&fakeExtensionRuntime{reloadErr: reloadErr})

				_, err := tt.transition(runtime, testutil.Context(t), instance.ID)
				if !errors.Is(err, reloadErr) {
					t.Fatalf("transition() error = %v, want wrapped reload failure", err)
				}

				updated, getErr := runtime.GetInstance(testutil.Context(t), instance.ID)
				if getErr != nil {
					t.Fatalf("GetInstance() error = %v", getErr)
				}
				if got, want := updated.Enabled, tt.wantEnable; got != want {
					t.Fatalf("GetInstance().Enabled = %t, want %t", got, want)
				}
				if got, want := updated.Status, tt.wantState; got != want {
					t.Fatalf("GetInstance().Status = %q, want %q", got, want)
				}
			})
		}
	})

	t.Run("ShouldSerializeConcurrentLifecycleOperationsDuringReloadRollback", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 13, 10, 0, 0, time.UTC)
		reloadErr := errors.New("reload boom")
		runtime := newChannelRuntime(db, discardLogger(), func() time.Time { return now }, nil)

		instance := mustCreateDaemonChannelInstance(t, runtime, channelspkg.CreateInstanceRequest{
			ID:            "chan-race",
			Scope:         channelspkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-race",
			DisplayName:   "Race Channel",
			Enabled:       true,
			Status:        channelspkg.ChannelStatusReady,
			RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
		})
		extensions := newBlockingReloadExtensionRuntime(reloadErr)
		runtime.setExtensionRuntime(extensions)

		ctx := testutil.Context(t)
		restartErrCh := make(chan error, 1)
		go func() {
			_, err := runtime.RestartInstance(ctx, instance.ID)
			restartErrCh <- err
		}()

		select {
		case <-extensions.firstStarted:
		case <-time.After(time.Second):
			t.Fatal("RestartInstance() did not enter reload")
		}

		stopErrCh := make(chan error, 1)
		go func() {
			_, err := runtime.StopInstance(ctx, instance.ID)
			stopErrCh <- err
		}()

		select {
		case <-extensions.secondStarted:
			t.Fatal("StopInstance() entered reload before in-flight restart completed")
		case err := <-stopErrCh:
			t.Fatalf("StopInstance() returned before in-flight restart completed: %v", err)
		case <-time.After(200 * time.Millisecond):
		}

		close(extensions.releaseFirst)

		if err := <-restartErrCh; !errors.Is(err, reloadErr) {
			t.Fatalf("RestartInstance() error = %v, want wrapped reload failure", err)
		}
		if err := <-stopErrCh; err != nil {
			t.Fatalf("StopInstance() error = %v", err)
		}

		updated, err := runtime.GetInstance(ctx, instance.ID)
		if err != nil {
			t.Fatalf("GetInstance() error = %v", err)
		}
		if updated.Enabled {
			t.Fatal("GetInstance() after concurrent restart/stop left instance enabled, want disabled")
		}
		if got, want := updated.Status, channelspkg.ChannelStatusDisabled; got != want {
			t.Fatalf("GetInstance().Status after concurrent restart/stop = %q, want %q", got, want)
		}
	})
}

func mustCreateDaemonChannelInstance(
	t *testing.T,
	runtime *channelRuntime,
	req channelspkg.CreateInstanceRequest,
) *channelspkg.ChannelInstance {
	t.Helper()

	instance, err := runtime.CreateInstance(testutil.Context(t), req)
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}
	return instance
}

func mustUpsertDaemonChannelRoute(
	t *testing.T,
	runtime *channelRuntime,
	route channelspkg.ChannelRoute,
) *channelspkg.ChannelRoute {
	t.Helper()

	resolved, err := runtime.UpsertRoute(testutil.Context(t), route)
	if err != nil {
		t.Fatalf("UpsertRoute() error = %v", err)
	}
	return resolved
}

type blockingReloadExtensionRuntime struct {
	mu             sync.Mutex
	reloadCount    int
	reloadErr      error
	firstStarted   chan struct{}
	secondStarted  chan struct{}
	releaseFirst   chan struct{}
	firstStartOnce sync.Once
}

func newBlockingReloadExtensionRuntime(reloadErr error) *blockingReloadExtensionRuntime {
	return &blockingReloadExtensionRuntime{
		reloadErr:     reloadErr,
		firstStarted:  make(chan struct{}),
		secondStarted: make(chan struct{}),
		releaseFirst:  make(chan struct{}),
	}
}

func (r *blockingReloadExtensionRuntime) Start(context.Context) error {
	return nil
}

func (r *blockingReloadExtensionRuntime) Stop(context.Context) error {
	return nil
}

func (r *blockingReloadExtensionRuntime) Reload(ctx context.Context) error {
	r.mu.Lock()
	r.reloadCount++
	call := r.reloadCount
	r.mu.Unlock()

	switch call {
	case 1:
		r.firstStartOnce.Do(func() {
			close(r.firstStarted)
		})
		select {
		case <-r.releaseFirst:
		case <-ctx.Done():
			return ctx.Err()
		}
		return r.reloadErr
	case 2:
		close(r.secondStarted)
	}

	return nil
}

func (r *blockingReloadExtensionRuntime) Get(string) (*extensionpkg.Extension, error) {
	return nil, nil
}

func (r *blockingReloadExtensionRuntime) HookDeclarations(context.Context) ([]hookspkg.HookDecl, error) {
	return nil, nil
}
