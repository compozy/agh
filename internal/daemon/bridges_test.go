package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/testutil"
)

var errUnexpectedBridgeRuntimeStoreCall = errors.New("unexpected bridge runtime store stub call")

type recordingBridgeSecretResolver struct {
	values map[string]string
	calls  []bridgepkg.BridgeSecretBinding
	err    error
}

type bridgeRuntimeStoreStub struct {
	bridgeRuntimeStore
	listBridgeSecretBindingsFn  func(context.Context, string) ([]bridgepkg.BridgeSecretBinding, error)
	putBridgeSecretBindingFn    func(context.Context, bridgepkg.BridgeSecretBinding) error
	deleteBridgeSecretBindingFn func(context.Context, string, string) error
}

func (s bridgeRuntimeStoreStub) ListBridgeSecretBindings(ctx context.Context, bridgeInstanceID string) ([]bridgepkg.BridgeSecretBinding, error) {
	if s.listBridgeSecretBindingsFn != nil {
		return s.listBridgeSecretBindingsFn(ctx, bridgeInstanceID)
	}
	return nil, fmt.Errorf("%w: ListBridgeSecretBindings(%q)", errUnexpectedBridgeRuntimeStoreCall, bridgeInstanceID)
}

func (s bridgeRuntimeStoreStub) PutBridgeSecretBinding(ctx context.Context, binding bridgepkg.BridgeSecretBinding) error {
	if s.putBridgeSecretBindingFn != nil {
		return s.putBridgeSecretBindingFn(ctx, binding)
	}
	return fmt.Errorf("%w: PutBridgeSecretBinding(%q, %q)", errUnexpectedBridgeRuntimeStoreCall, binding.BridgeInstanceID, binding.BindingName)
}

func (s bridgeRuntimeStoreStub) DeleteBridgeSecretBinding(ctx context.Context, bridgeInstanceID string, bindingName string) error {
	if s.deleteBridgeSecretBindingFn != nil {
		return s.deleteBridgeSecretBindingFn(ctx, bridgeInstanceID, bindingName)
	}
	return fmt.Errorf("%w: DeleteBridgeSecretBinding(%q, %q)", errUnexpectedBridgeRuntimeStoreCall, bridgeInstanceID, bindingName)
}

func (r *recordingBridgeSecretResolver) ResolveBridgeSecret(_ context.Context, binding bridgepkg.BridgeSecretBinding) (string, error) {
	r.calls = append(r.calls, binding)
	if r.err != nil {
		return "", r.err
	}
	if value, ok := r.values[binding.BindingName]; ok {
		return value, nil
	}
	return "resolved-" + binding.BindingName, nil
}

func TestComposeBridgeRuntime(t *testing.T) {
	t.Run("ShouldReturnNilWhenRegistryDoesNotSupportBridgePersistence", func(t *testing.T) {
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
		if runtime := d.composeBridgeRuntime(state, &bootCleanup{}); runtime != nil {
			t.Fatalf("composeBridgeRuntime(recordingRegistry) = %#v, want nil", runtime)
		}
	})

	t.Run("ShouldBuildRuntimeWhenRegistrySupportsBridgePersistence", func(t *testing.T) {
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

		runtime := d.composeBridgeRuntime(state, &bootCleanup{})
		if runtime == nil {
			t.Fatal("composeBridgeRuntime(globaldb) = nil, want non-nil")
		}
		if runtime.Broker() == nil {
			t.Fatal("composeBridgeRuntime(globaldb) broker = nil, want non-nil")
		}
		if runtime.store != db {
			t.Fatalf("composeBridgeRuntime(globaldb) store = %#v, want global db", runtime.store)
		}
		if runtime.registry == nil {
			t.Fatal("composeBridgeRuntime(globaldb) registry = nil, want extension registry")
		}
	})
}

func TestWithBridgeSecretResolver(t *testing.T) {
	t.Run("ShouldStoreResolverOnDaemon", func(t *testing.T) {
		t.Parallel()

		resolver := &recordingBridgeSecretResolver{}
		d := &Daemon{}

		WithBridgeSecretResolver(resolver)(d)

		if d.bridgeSecretResolver != resolver {
			t.Fatalf("WithBridgeSecretResolver() stored %#v, want %#v", d.bridgeSecretResolver, resolver)
		}
	})
}

func TestBootExtensions(t *testing.T) {
	t.Run("ShouldInjectBridgeRuntimeDependencies", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 12, 15, 0, 0, time.UTC)
		bridges := newBridgeRuntime(db, discardLogger(), func() time.Time { return now }, nil)
		if bridges == nil {
			t.Fatal("newBridgeRuntime() = nil, want non-nil")
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
			bridges:           bridges,
		}

		if err := d.bootExtensions(testutil.Context(t), state, &bootCleanup{}); err != nil {
			t.Fatalf("bootExtensions() error = %v", err)
		}

		if captured.BridgeRegistry != bridges {
			t.Fatalf("extension deps bridge registry = %#v, want bridge runtime", captured.BridgeRegistry)
		}
		if captured.BridgeDedupStore == nil {
			t.Fatal("extension deps bridge dedup store = nil, want non-nil")
		}
		if captured.BridgeBroker != bridges.Broker() {
			t.Fatalf("extension deps bridge broker = %#v, want runtime broker", captured.BridgeBroker)
		}
		if captured.BridgeRuntime != bridges {
			t.Fatalf("extension deps bridge runtime = %#v, want bridge runtime", captured.BridgeRuntime)
		}
		if manager.startCount != 1 {
			t.Fatalf("extension manager start count = %d, want 1", manager.startCount)
		}
	})
}

func TestBridgeRuntimeStartInstance(t *testing.T) {
	t.Run("ShouldReturnNilRuntimeWhenStoreIsMissing", func(t *testing.T) {
		t.Parallel()

		if runtime := newBridgeRuntime(nil, nil, nil, nil); runtime != nil {
			t.Fatalf("newBridgeRuntime(nil store) = %#v, want nil", runtime)
		}
	})

	t.Run("ShouldHandleNilBrokerAccess", func(t *testing.T) {
		t.Parallel()

		var nilRuntime *bridgeRuntime
		if broker := nilRuntime.Broker(); broker != nil {
			t.Fatalf("(*bridgeRuntime)(nil).Broker() = %#v, want nil", broker)
		}
		nilRuntime.Close()
	})

	t.Run("ShouldTransitionDisabledInstanceToStarting", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		runtime := newBridgeRuntime(db, nil, nil, nil)
		extensions := &fakeExtensionRuntime{}
		runtime.setExtensionRuntime(extensions)

		instance := mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
			ID:            "brg-start",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-start",
			DisplayName:   "Start Bridge",
			Enabled:       false,
			Status:        bridgepkg.BridgeStatusDisabled,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})

		updated, err := runtime.StartInstance(testutil.Context(t), instance.ID)
		if err != nil {
			t.Fatalf("StartInstance() error = %v", err)
		}
		if !updated.Enabled {
			t.Fatal("StartInstance() left instance disabled, want enabled")
		}
		if got, want := updated.Status, bridgepkg.BridgeStatusStarting; got != want {
			t.Fatalf("StartInstance().Status = %q, want %q", got, want)
		}
		if got, want := extensions.reloadCount, 1; got != want {
			t.Fatalf("extension reload count = %d, want %d", got, want)
		}
		if runtime.Broker() == nil {
			t.Fatal("newBridgeRuntime() broker = nil, want non-nil")
		}
	})
}

func TestBridgeRuntimeCreateInstance(t *testing.T) {
	t.Run("ShouldReloadExtensionsWhenEnabled", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 12, 20, 0, 0, time.UTC)
		runtime := newBridgeRuntime(db, discardLogger(), func() time.Time { return now }, nil)
		extensions := &fakeExtensionRuntime{}
		runtime.setExtensionRuntime(extensions)

		created, err := runtime.CreateInstance(testutil.Context(t), bridgepkg.CreateInstanceRequest{
			ID:            "brg-create",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-create",
			DisplayName:   "Create Bridge",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusStarting,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
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
		runtime := newBridgeRuntime(db, discardLogger(), func() time.Time { return now }, nil)
		runtime.setExtensionRuntime(extensions)

		_, err := runtime.CreateInstance(testutil.Context(t), bridgepkg.CreateInstanceRequest{
			ID:            "brg-create-rollback",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-create-rollback",
			DisplayName:   "Create Rollback Bridge",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusStarting,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})
		if !errors.Is(err, reloadErr) {
			t.Fatalf("CreateInstance() error = %v, want wrapped reload failure", err)
		}
		if got, want := extensions.reloadCount, 1; got != want {
			t.Fatalf("extension reload count after failed create = %d, want %d", got, want)
		}

		created, getErr := runtime.GetInstance(testutil.Context(t), "brg-create-rollback")
		if getErr != nil {
			t.Fatalf("GetInstance() error = %v", getErr)
		}
		if created.Enabled {
			t.Fatal("GetInstance() after failed create left instance enabled, want disabled rollback")
		}
		if got, want := created.Status, bridgepkg.BridgeStatusDisabled; got != want {
			t.Fatalf("GetInstance().Status after failed create = %q, want %q", got, want)
		}
	})
}

func TestBridgeRuntimeListProviders(t *testing.T) {
	t.Run("ShouldProjectInstalledBridgeProvidersFromExtensionRegistry", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 12, 25, 0, 0, time.UTC)
		runtime := newBridgeRuntime(db, discardLogger(), func() time.Time { return now }, nil)
		if runtime == nil {
			t.Fatal("newBridgeRuntime() = nil, want non-nil")
		}
		if runtime.registry == nil {
			t.Fatal("runtime.registry = nil, want extension registry")
		}

		bridgeInfo := mustInstallDaemonExtension(t, runtime.registry, daemonExtensionFixture{
			name:              "telegram-reference",
			description:       "Reference Telegram bridge adapter",
			capabilities:      []string{"bridge.adapter"},
			bridgePlatform:    "telegram",
			bridgeDisplayName: "Telegram",
			bridgeSecretSlots: `
[[bridge.secret_slots]]
name = "bot_token"
description = "Bot API token"
required = true
`,
			bridgeConfigSchema:  "agh.bridge.telegram",
			bridgeConfigVersion: "v1",
			enabled:             true,
		})
		mustInstallDaemonExtension(t, runtime.registry, daemonExtensionFixture{
			name:         "memory-only",
			description:  "Memory backend",
			capabilities: []string{"memory.backend"},
			enabled:      true,
		})

		runtime.setExtensionRuntime(&fakeExtensionRuntime{
			getExt: &extensionpkg.Extension{
				Info: *bridgeInfo,
				Status: extensionpkg.ExtensionStatus{
					Name:          bridgeInfo.Name,
					Version:       bridgeInfo.Version,
					Source:        bridgeInfo.Source,
					Enabled:       bridgeInfo.Enabled,
					Registered:    true,
					Active:        true,
					Healthy:       true,
					HealthMessage: "connected",
					LastStartedAt: now.Add(-time.Minute),
				},
			},
		})

		providers, err := runtime.ListProviders(testutil.Context(t))
		if err != nil {
			t.Fatalf("ListProviders() error = %v", err)
		}
		if got, want := len(providers), 1; got != want {
			t.Fatalf("len(providers) = %d, want %d", got, want)
		}
		if got, want := providers[0].Platform, "telegram"; got != want {
			t.Fatalf("provider platform = %q, want %q", got, want)
		}
		if got, want := providers[0].DisplayName, "Telegram"; got != want {
			t.Fatalf("provider display name = %q, want %q", got, want)
		}
		if got, want := len(providers[0].SecretSlots), 1; got != want {
			t.Fatalf("len(provider secret slots) = %d, want %d", got, want)
		}
		if got, want := providers[0].SecretSlots[0].Name, "bot_token"; got != want {
			t.Fatalf("provider secret slot name = %q, want %q", got, want)
		}
		if providers[0].ConfigSchema == nil {
			t.Fatal("provider config schema = nil, want value")
		}
		if got, want := providers[0].ConfigSchema.Schema, "agh.bridge.telegram"; got != want {
			t.Fatalf("provider config schema id = %q, want %q", got, want)
		}
		if got, want := providers[0].ConfigSchema.Version, "v1"; got != want {
			t.Fatalf("provider config schema version = %q, want %q", got, want)
		}
		if got, want := providers[0].State, "active"; got != want {
			t.Fatalf("provider state = %q, want %q", got, want)
		}
		if got, want := providers[0].Health, "healthy"; got != want {
			t.Fatalf("provider health = %q, want %q", got, want)
		}
		if got, want := providers[0].HealthMessage, "connected"; got != want {
			t.Fatalf("provider health message = %q, want %q", got, want)
		}
	})

	t.Run("ShouldSkipBridgeProvidersWithUnreadableManifestSnapshots", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		runtime := newBridgeRuntime(db, discardLogger(), func() time.Time {
			return time.Date(2026, 4, 11, 12, 35, 0, 0, time.UTC)
		}, nil)
		if runtime == nil || runtime.registry == nil {
			t.Fatal("newBridgeRuntime() missing registry")
		}

		goodInfo := mustInstallDaemonExtension(t, runtime.registry, daemonExtensionFixture{
			name:              "telegram-reference",
			description:       "Reference Telegram bridge adapter",
			capabilities:      []string{"bridge.adapter"},
			bridgePlatform:    "telegram",
			bridgeDisplayName: "Telegram",
			enabled:           true,
		})
		badInfo := mustInstallDaemonExtension(t, runtime.registry, daemonExtensionFixture{
			name:              "slack-broken",
			description:       "Broken Slack bridge adapter",
			capabilities:      []string{"bridge.adapter"},
			bridgePlatform:    "slack",
			bridgeDisplayName: "Slack",
			enabled:           true,
		})
		if err := os.Remove(badInfo.ManifestPath); err != nil {
			t.Fatalf("os.Remove(%s) error = %v", badInfo.ManifestPath, err)
		}

		providers, err := runtime.ListProviders(testutil.Context(t))
		if err != nil {
			t.Fatalf("ListProviders() error = %v", err)
		}
		if got, want := len(providers), 1; got != want {
			t.Fatalf("len(providers) = %d, want %d", got, want)
		}
		if got, want := providers[0].ExtensionName, goodInfo.Name; got != want {
			t.Fatalf("provider extension_name = %q, want %q", got, want)
		}
	})
}

func TestBridgeRuntimeResolveBridgeRuntime(t *testing.T) {
	t.Run("ShouldResolveBoundSecrets", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 12, 30, 0, 0, time.UTC)
		resolver := &recordingBridgeSecretResolver{
			values: map[string]string{
				"bot_token": "secret-value",
			},
		}

		runtime := newBridgeRuntime(db, discardLogger(), func() time.Time { return now }, resolver)
		instance := mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
			ID:            "brg-secret",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-secret",
			DisplayName:   "Secret Bridge",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusReady,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})
		if err := db.PutBridgeSecretBinding(testutil.Context(t), bridgepkg.BridgeSecretBinding{
			BridgeInstanceID: instance.ID,
			BindingName:      "bot_token",
			VaultRef:         "vault://bridges/ext-secret/bot-token",
			Kind:             "bot_token",
			CreatedAt:        now,
			UpdatedAt:        now,
		}); err != nil {
			t.Fatalf("PutBridgeSecretBinding() error = %v", err)
		}

		launch, err := runtime.ResolveBridgeRuntime(testutil.Context(t), "ext-secret")
		if err != nil {
			t.Fatalf("ResolveBridgeRuntime() error = %v", err)
		}
		if launch == nil {
			t.Fatal("ResolveBridgeRuntime() = nil, want non-nil")
		}
		managed, ok := launch.ManagedInstance(instance.ID)
		if !ok {
			t.Fatalf("ResolveBridgeRuntime() missing managed instance %q", instance.ID)
		}
		if got, want := managed.Instance.ID, instance.ID; got != want {
			t.Fatalf("ResolveBridgeRuntime().ManagedInstance(%q).Instance.ID = %q, want %q", instance.ID, got, want)
		}
		if got := managed.BoundSecrets; len(got) != 1 || got[0].BindingName != "bot_token" || got[0].Value != "secret-value" {
			t.Fatalf("ResolveBridgeRuntime().ManagedInstance(%q).BoundSecrets = %#v, want resolved bot_token binding", instance.ID, got)
		}
		if len(resolver.calls) != 1 || resolver.calls[0].BindingName != "bot_token" {
			t.Fatalf("ResolveBridgeSecret() calls = %#v, want bot_token binding", resolver.calls)
		}

		updated, err := runtime.GetInstance(testutil.Context(t), instance.ID)
		if err != nil {
			t.Fatalf("GetInstance() error = %v", err)
		}
		if got, want := updated.Status, bridgepkg.BridgeStatusStarting; got != want {
			t.Fatalf("instance status after launch = %q, want %q", got, want)
		}
	})

	t.Run("ShouldRequireSecretResolverWhenBindingsExist", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 12, 35, 0, 0, time.UTC)
		runtime := newBridgeRuntime(db, discardLogger(), func() time.Time { return now }, nil)
		instance := mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
			ID:            "brg-secret-missing",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-secret-missing",
			DisplayName:   "Secret Missing",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusReady,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})
		if err := db.PutBridgeSecretBinding(testutil.Context(t), bridgepkg.BridgeSecretBinding{
			BridgeInstanceID: instance.ID,
			BindingName:      "bot_token",
			VaultRef:         "vault://bridges/ext-secret-missing/bot-token",
			Kind:             "bot_token",
			CreatedAt:        now,
			UpdatedAt:        now,
		}); err != nil {
			t.Fatalf("PutBridgeSecretBinding() error = %v", err)
		}

		_, err := runtime.ResolveBridgeRuntime(testutil.Context(t), instance.ExtensionName)
		if !errors.Is(err, errBridgeSecretResolverRequired) {
			t.Fatalf("ResolveBridgeRuntime() error = %v, want missing secret resolver sentinel", err)
		}
	})

	t.Run("ShouldNotPersistStartingWhenSecretResolutionFails", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 12, 36, 0, 0, time.UTC)
		resolverErr := errors.New("vault boom")
		resolver := &recordingBridgeSecretResolver{err: resolverErr}
		runtime := newBridgeRuntime(db, discardLogger(), func() time.Time { return now }, resolver)

		instance := mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
			ID:            "brg-secret-fail",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-secret-fail",
			DisplayName:   "Secret Failure",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusReady,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})
		if err := db.PutBridgeSecretBinding(testutil.Context(t), bridgepkg.BridgeSecretBinding{
			BridgeInstanceID: instance.ID,
			BindingName:      "bot_token",
			VaultRef:         "vault://bridges/ext-secret-fail/bot-token",
			Kind:             "bot_token",
			CreatedAt:        now,
			UpdatedAt:        now,
		}); err != nil {
			t.Fatalf("PutBridgeSecretBinding() error = %v", err)
		}

		_, err := runtime.ResolveBridgeRuntime(testutil.Context(t), instance.ExtensionName)
		if !errors.Is(err, resolverErr) {
			t.Fatalf("ResolveBridgeRuntime() error = %v, want wrapped resolver error", err)
		}

		updated, getErr := runtime.GetInstance(testutil.Context(t), instance.ID)
		if getErr != nil {
			t.Fatalf("GetInstance() error = %v", getErr)
		}
		if got, want := updated.Status, bridgepkg.BridgeStatusReady; got != want {
			t.Fatalf("instance status after failed secret resolution = %q, want %q", got, want)
		}
	})

	t.Run("ShouldResolveMultipleEnabledInstancesForOneExtension", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 12, 37, 0, 0, time.UTC)
		runtime := newBridgeRuntime(db, discardLogger(), func() time.Time { return now }, nil)

		first := mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
			ID:            "brg-multi-a",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-multi",
			DisplayName:   "Multi A",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusReady,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})
		second := mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
			ID:            "brg-multi-b",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-multi",
			DisplayName:   "Multi B",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusDegraded,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})

		launch, err := runtime.ResolveBridgeRuntime(testutil.Context(t), "ext-multi")
		if err != nil {
			t.Fatalf("ResolveBridgeRuntime() error = %v", err)
		}
		if launch == nil {
			t.Fatal("ResolveBridgeRuntime() = nil, want non-nil")
		}
		if got, want := launch.ManagedBridgeInstanceIDs(), []string{first.ID, second.ID}; !slices.Equal(got, want) {
			t.Fatalf("ResolveBridgeRuntime().ManagedBridgeInstanceIDs() = %#v, want %#v", got, want)
		}

		for _, instanceID := range []string{first.ID, second.ID} {
			managed, ok := launch.ManagedInstance(instanceID)
			if !ok {
				t.Fatalf("ResolveBridgeRuntime() missing managed instance %q", instanceID)
			}
			if got, want := managed.Instance.Status.Normalize(), bridgepkg.BridgeStatusStarting; got != want {
				t.Fatalf("managed instance %q status = %q, want %q", instanceID, got, want)
			}
		}
	})

	t.Run("ShouldDeferWhenNoEnabledInstanceExistsForExtension", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		runtime := newBridgeRuntime(db, discardLogger(), nil, nil)

		_, err := runtime.ResolveBridgeRuntime(testutil.Context(t), "ext-missing")
		if !errors.Is(err, extensionpkg.ErrBridgeRuntimeDeferred) {
			t.Fatalf("ResolveBridgeRuntime() error = %v, want deferred sentinel", err)
		}
	})
}

func TestBridgeRuntimeSecretBindings(t *testing.T) {
	t.Run("ShouldNormalizeBindingKeysOnWrite", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		runtime := newBridgeRuntime(db, discardLogger(), nil, nil)
		instance := mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
			ID:            "brg-secret-binding",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-secret-binding",
			DisplayName:   "Secret Binding",
			Enabled:       false,
			Status:        bridgepkg.BridgeStatusDisabled,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})

		err := runtime.PutSecretBinding(testutil.Context(t), bridgepkg.BridgeSecretBinding{
			BridgeInstanceID: " " + instance.ID + " ",
			BindingName:      " bot_token ",
			VaultRef:         "vault://bridges/ext-secret-binding/bot-token",
			Kind:             "token",
		})
		if err != nil {
			t.Fatalf("PutSecretBinding() error = %v", err)
		}

		bindings, err := runtime.ListSecretBindings(testutil.Context(t), " "+instance.ID+" ")
		if err != nil {
			t.Fatalf("ListSecretBindings() error = %v", err)
		}
		if got, want := len(bindings), 1; got != want {
			t.Fatalf("len(bindings) = %d, want %d", got, want)
		}
		if got, want := bindings[0].BridgeInstanceID, instance.ID; got != want {
			t.Fatalf("bindings[0].BridgeInstanceID = %q, want %q", got, want)
		}
		if got, want := bindings[0].BindingName, "bot_token"; got != want {
			t.Fatalf("bindings[0].BindingName = %q, want %q", got, want)
		}

		if err := runtime.DeleteSecretBinding(testutil.Context(t), " "+instance.ID+" ", " bot_token "); err != nil {
			t.Fatalf("DeleteSecretBinding() error = %v", err)
		}

		bindings, err = runtime.ListSecretBindings(testutil.Context(t), instance.ID)
		if err != nil {
			t.Fatalf("ListSecretBindings(after delete) error = %v", err)
		}
		if got := len(bindings); got != 0 {
			t.Fatalf("len(bindings after delete) = %d, want 0", got)
		}
	})

	t.Run("ShouldWrapStoreErrorsWithDaemonContext", func(t *testing.T) {
		t.Parallel()

		listErr := errors.New("list failed")
		putErr := errors.New("put failed")
		deleteErr := errors.New("delete failed")
		runtime := newBridgeRuntime(bridgeRuntimeStoreStub{
			listBridgeSecretBindingsFn: func(context.Context, string) ([]bridgepkg.BridgeSecretBinding, error) {
				return nil, listErr
			},
			putBridgeSecretBindingFn: func(context.Context, bridgepkg.BridgeSecretBinding) error {
				return putErr
			},
			deleteBridgeSecretBindingFn: func(context.Context, string, string) error {
				return deleteErr
			},
		}, discardLogger(), nil, nil)

		_, err := runtime.ListSecretBindings(testutil.Context(t), "brg-1")
		if !errors.Is(err, listErr) || !strings.Contains(err.Error(), "daemon: list bridge secret bindings") {
			t.Fatalf("ListSecretBindings() error = %v, want wrapped list failure", err)
		}
		err = runtime.PutSecretBinding(testutil.Context(t), bridgepkg.BridgeSecretBinding{
			BridgeInstanceID: " brg-1 ",
			BindingName:      " token ",
		})
		if !errors.Is(err, putErr) || !strings.Contains(err.Error(), "daemon: put bridge secret binding") {
			t.Fatalf("PutSecretBinding() error = %v, want wrapped put failure", err)
		}
		err = runtime.DeleteSecretBinding(testutil.Context(t), " brg-1 ", " token ")
		if !errors.Is(err, deleteErr) || !strings.Contains(err.Error(), "daemon: delete bridge secret binding") {
			t.Fatalf("DeleteSecretBinding() error = %v, want wrapped delete failure", err)
		}
	})
}

func TestBridgeRuntimeStopInstance(t *testing.T) {
	t.Run("ShouldBlockIngressAndPreserveRoutes", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 12, 45, 0, 0, time.UTC)
		runtime := newBridgeRuntime(db, discardLogger(), func() time.Time { return now }, nil)
		extensions := &fakeExtensionRuntime{}
		runtime.setExtensionRuntime(extensions)

		instance := mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
			ID:            "brg-stop",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-stop",
			DisplayName:   "Stop Bridge",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusReady,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})
		extensions.reloadCount = 0
		route := mustUpsertDaemonBridgeRoute(t, runtime, bridgepkg.BridgeRoute{
			Scope:            bridgepkg.ScopeGlobal,
			BridgeInstanceID: instance.ID,
			PeerID:           "peer-stop",
			SessionID:        "sess-stop",
			AgentName:        "coder",
			LastActivityAt:   now,
		})

		updated, err := runtime.StopInstance(testutil.Context(t), instance.ID)
		if err != nil {
			t.Fatalf("StopInstance() error = %v", err)
		}
		if updated.Enabled {
			t.Fatal("StopInstance() left instance enabled, want disabled")
		}
		if got, want := updated.Status, bridgepkg.BridgeStatusDisabled; got != want {
			t.Fatalf("StopInstance().Status = %q, want %q", got, want)
		}

		routes, err := runtime.ListRoutes(testutil.Context(t), instance.ID)
		if err != nil {
			t.Fatalf("ListRoutes() error = %v", err)
		}
		if len(routes) != 1 || routes[0].RoutingKeyHash != route.RoutingKeyHash {
			t.Fatalf("ListRoutes() = %#v, want preserved route %#v", routes, route)
		}

		if _, err := runtime.ResolveRoute(testutil.Context(t), route.RoutingKey()); !errors.Is(err, bridgepkg.ErrBridgeInstanceUnavailable) {
			t.Fatalf("ResolveRoute(disabled) error = %v, want ErrBridgeInstanceUnavailable", err)
		}
	})
}

func TestBridgeRuntimeRestartInstance(t *testing.T) {
	t.Run("ShouldPreserveRoutesAndReloadExtensions", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 4, 11, 13, 0, 0, 0, time.UTC)
		runtime := newBridgeRuntime(db, discardLogger(), func() time.Time { return now }, nil)
		extensions := &fakeExtensionRuntime{}
		runtime.setExtensionRuntime(extensions)

		instance := mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
			ID:            "brg-restart",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-restart",
			DisplayName:   "Restart Bridge",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusReady,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})
		extensions.reloadCount = 0
		route := mustUpsertDaemonBridgeRoute(t, runtime, bridgepkg.BridgeRoute{
			Scope:            bridgepkg.ScopeGlobal,
			BridgeInstanceID: instance.ID,
			PeerID:           "peer-restart",
			SessionID:        "sess-restart",
			AgentName:        "coder",
			LastActivityAt:   now,
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
		if got, want := updated.Status, bridgepkg.BridgeStatusStarting; got != want {
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

func TestBridgeRuntimeTransition(t *testing.T) {
	t.Run("ShouldRestorePreviousStateWhenReloadFails", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 4, 11, 13, 5, 0, 0, time.UTC)

		testCases := []struct {
			name       string
			request    bridgepkg.CreateInstanceRequest
			transition func(*bridgeRuntime, context.Context, string) (*bridgepkg.BridgeInstance, error)
			wantState  bridgepkg.BridgeStatus
			wantEnable bool
		}{
			{
				name: "ShouldRollbackStart",
				request: bridgepkg.CreateInstanceRequest{
					ID:            "brg-start-rollback",
					Scope:         bridgepkg.ScopeGlobal,
					Platform:      "slack",
					ExtensionName: "ext-start-rollback",
					DisplayName:   "Start Rollback",
					Enabled:       false,
					Status:        bridgepkg.BridgeStatusDisabled,
					RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
				},
				transition: func(runtime *bridgeRuntime, ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
					return runtime.StartInstance(ctx, id)
				},
				wantState:  bridgepkg.BridgeStatusDisabled,
				wantEnable: false,
			},
			{
				name: "ShouldRollbackStop",
				request: bridgepkg.CreateInstanceRequest{
					ID:            "brg-stop-rollback",
					Scope:         bridgepkg.ScopeGlobal,
					Platform:      "slack",
					ExtensionName: "ext-stop-rollback",
					DisplayName:   "Stop Rollback",
					Enabled:       true,
					Status:        bridgepkg.BridgeStatusReady,
					RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
				},
				transition: func(runtime *bridgeRuntime, ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
					return runtime.StopInstance(ctx, id)
				},
				wantState:  bridgepkg.BridgeStatusReady,
				wantEnable: true,
			},
			{
				name: "ShouldRollbackRestart",
				request: bridgepkg.CreateInstanceRequest{
					ID:            "brg-restart-rollback",
					Scope:         bridgepkg.ScopeGlobal,
					Platform:      "slack",
					ExtensionName: "ext-restart-rollback",
					DisplayName:   "Restart Rollback",
					Enabled:       true,
					Status:        bridgepkg.BridgeStatusReady,
					RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
				},
				transition: func(runtime *bridgeRuntime, ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
					return runtime.RestartInstance(ctx, id)
				},
				wantState:  bridgepkg.BridgeStatusReady,
				wantEnable: true,
			},
		}

		for _, tt := range testCases {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				db := openDaemonTestGlobalDB(t)
				reloadErr := errors.New("reload boom")
				runtime := newBridgeRuntime(db, discardLogger(), func() time.Time { return now }, nil)

				instance := mustCreateDaemonBridgeInstance(t, runtime, tt.request)
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
		runtime := newBridgeRuntime(db, discardLogger(), func() time.Time { return now }, nil)

		instance := mustCreateDaemonBridgeInstance(t, runtime, bridgepkg.CreateInstanceRequest{
			ID:            "brg-race",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "slack",
			ExtensionName: "ext-race",
			DisplayName:   "Race Bridge",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusReady,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
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
		if got, want := updated.Status, bridgepkg.BridgeStatusDisabled; got != want {
			t.Fatalf("GetInstance().Status after concurrent restart/stop = %q, want %q", got, want)
		}
	})
}

func mustCreateDaemonBridgeInstance(
	t *testing.T,
	runtime *bridgeRuntime,
	req bridgepkg.CreateInstanceRequest,
) *bridgepkg.BridgeInstance {
	t.Helper()

	instance, err := runtime.CreateInstance(testutil.Context(t), req)
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}
	return instance
}

func mustUpsertDaemonBridgeRoute(
	t *testing.T,
	runtime *bridgeRuntime,
	route bridgepkg.BridgeRoute,
) *bridgepkg.BridgeRoute {
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

type daemonExtensionFixture struct {
	name                string
	description         string
	capabilities        []string
	bridgePlatform      string
	bridgeDisplayName   string
	bridgeSecretSlots   string
	bridgeConfigSchema  string
	bridgeConfigVersion string
	enabled             bool
}

func mustInstallDaemonExtension(
	t *testing.T,
	registry *extensionpkg.Registry,
	fixture daemonExtensionFixture,
) *extensionpkg.ExtensionInfo {
	t.Helper()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "extension.toml")
	if err := os.WriteFile(manifestPath, []byte(daemonExtensionManifest(fixture)), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%s) error = %v", manifestPath, err)
	}

	manifest, err := extensionpkg.LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest(%s) error = %v", dir, err)
	}
	checksum, err := extensionpkg.ComputeDirectoryChecksum(dir)
	if err != nil {
		t.Fatalf("ComputeDirectoryChecksum(%s) error = %v", dir, err)
	}
	if err := registry.Install(manifest, dir, checksum); err != nil {
		t.Fatalf("Install(%s) error = %v", fixture.name, err)
	}
	if !fixture.enabled {
		if err := registry.Disable(fixture.name); err != nil {
			t.Fatalf("Disable(%s) error = %v", fixture.name, err)
		}
	}

	info, err := registry.Get(fixture.name)
	if err != nil {
		t.Fatalf("Get(%s) error = %v", fixture.name, err)
	}
	return info
}

func daemonExtensionManifest(fixture daemonExtensionFixture) string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "[extension]\nname = %q\nversion = \"0.1.0\"\ndescription = %q\nmin_agh_version = \"0.5.0\"\n\n", fixture.name, fixture.description)
	if len(fixture.capabilities) > 0 {
		fmt.Fprintf(&builder, "[capabilities]\nprovides = [%s]\n\n", quotedStringList(fixture.capabilities))
	}
	if fixture.bridgePlatform != "" || fixture.bridgeDisplayName != "" {
		fmt.Fprintf(&builder, "[bridge]\nplatform = %q\ndisplay_name = %q\n", fixture.bridgePlatform, fixture.bridgeDisplayName)
		if fixture.bridgeSecretSlots != "" {
			builder.WriteString(fixture.bridgeSecretSlots)
		}
		if fixture.bridgeConfigSchema != "" || fixture.bridgeConfigVersion != "" {
			fmt.Fprintf(&builder, "\n[bridge.config_schema]\nschema = %q\nversion = %q\n", fixture.bridgeConfigSchema, fixture.bridgeConfigVersion)
		}
	}
	return builder.String()
}

func quotedStringList(values []string) string {
	if len(values) == 0 {
		return ""
	}

	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return strings.Join(quoted, ", ")
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
