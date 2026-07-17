package daemon

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"testing"

	aghconfig "github.com/compozy/agh/internal/config"
	diagcontract "github.com/compozy/agh/internal/diagnosticcontract"
	"github.com/compozy/agh/internal/marketplace"
	"github.com/compozy/agh/internal/providers"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb"
)

func TestDaemonSettingsRuntimeApplier(t *testing.T) {
	// not parallel: this test mutates the provider pre-start process-wide cache.
	t.Run("Should invalidate provider prestart cache after active config apply", func(t *testing.T) {
		providers.InvalidatePreStartCache()
		t.Cleanup(providers.InvalidatePreStartCache)

		calls := 0
		env := &providers.ProbeEnv{
			ProviderName: "config-apply-cache",
			LookPath: func(string) (string, error) {
				calls++
				return "", exec.ErrNotFound
			},
		}
		provider := aghconfig.ProviderConfig{
			Command:  "config-apply-cache acp",
			AuthMode: aghconfig.ProviderAuthModeNativeCLI,
		}
		assertMissingCLIReport(t, "first", providers.PreStart(t.Context(), provider, env))
		assertMissingCLIReport(t, "cached", providers.PreStart(t.Context(), provider, env))
		if calls != 1 {
			t.Fatalf("PreStart LookPath calls before apply = %d, want 1", calls)
		}

		cfg := aghconfig.Config{}
		failures := daemonSettingsRuntimeApplier{
			daemon: &Daemon{},
			state:  &bootState{cfg: cfg},
		}.ApplyActiveConfig(t.Context(), &cfg)
		if len(failures) != 0 {
			t.Fatalf("ApplyActiveConfig() failures = %#v, want none", failures)
		}

		assertMissingCLIReport(t, "after apply", providers.PreStart(t.Context(), provider, env))
		if calls != 2 {
			t.Fatalf("PreStart LookPath calls after apply = %d, want 2", calls)
		}
	})

	t.Run("Should rollback MCP after runtime apply failure", func(t *testing.T) {
		t.Parallel()

		previous := aghconfig.Config{
			Providers: map[string]aghconfig.ProviderConfig{
				"codex": {Command: "codex acp", AuthMode: aghconfig.ProviderAuthModeNativeCLI},
			},
		}
		next := aghconfig.Config{
			Providers: map[string]aghconfig.ProviderConfig{
				"codex": {Command: "codex acp --next", AuthMode: aghconfig.ProviderAuthModeNativeCLI},
			},
		}
		syncCalls := 0
		daemonInstance := &Daemon{}
		failures := daemonSettingsRuntimeApplier{
			daemon: daemonInstance,
			state: &bootState{
				cfg: previous,
				toolMCPResources: toolMCPPublisherFunc(func(context.Context) error {
					syncCalls++
					if syncCalls == 1 {
						return errors.New("mcp sync boom")
					}
					return nil
				}),
			},
		}.ApplyActiveConfig(t.Context(), &next)
		if syncCalls != 2 {
			t.Fatalf("MCP Sync calls = %d, want 2 (apply + rollback)", syncCalls)
		}
		if len(failures) != 1 {
			t.Fatalf("ApplyActiveConfig() failures = %#v, want one mcp failure", failures)
		}
		if failures[0].Subsystem != "mcp" {
			t.Fatalf("failure subsystem = %q, want mcp", failures[0].Subsystem)
		}
		if got := daemonInstance.config.Providers["codex"].Command; got != "codex acp" {
			t.Fatalf("restored daemon config command = %q, want previous", got)
		}
	})

	t.Run("Should apply and rollback extension side-load policy live", func(t *testing.T) {
		t.Parallel()

		nativeDeps, registry, _, _ := newNativeExtensionToolDeps(t)
		extensionService, ok := newDaemonExtensionService(
			registry,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nativeDeps.HomePaths,
			nil,
			nil,
		).(*daemonExtensionService)
		if !ok {
			t.Fatal("newDaemonExtensionService() did not return daemon service")
		}
		previous := aghconfig.Config{}
		next := previous
		next.Extensions.Marketplace.AllowUnverified = true
		syncCalls := 0
		state := &bootState{
			cfg:  previous,
			deps: RuntimeDeps{Extensions: extensionService},
			toolMCPResources: toolMCPPublisherFunc(func(context.Context) error {
				syncCalls++
				return errors.New("mcp sync boom")
			}),
		}
		failures := daemonSettingsRuntimeApplier{daemon: &Daemon{}, state: state}.ApplyActiveConfig(t.Context(), &next)
		if len(failures) != 2 {
			t.Fatalf("ApplyActiveConfig() failures = %#v, want mcp plus rollback", failures)
		}
		if extensionService.marketplaceConfig().AllowUnverified {
			t.Fatal("extension side-load policy = true after rollback, want false")
		}
	})

	t.Run("Should record mcp_rollback when MCP rollback sync fails", func(t *testing.T) {
		t.Parallel()

		previous := aghconfig.Config{}
		next := aghconfig.Config{
			Providers: map[string]aghconfig.ProviderConfig{
				"codex": {Command: "codex acp", AuthMode: aghconfig.ProviderAuthModeNativeCLI},
			},
		}
		syncCalls := 0
		failures := daemonSettingsRuntimeApplier{
			daemon: &Daemon{},
			state: &bootState{
				cfg: previous,
				toolMCPResources: toolMCPPublisherFunc(func(context.Context) error {
					syncCalls++
					return errors.New("mcp sync boom")
				}),
			},
		}.ApplyActiveConfig(t.Context(), &next)
		if syncCalls != 2 {
			t.Fatalf("MCP Sync calls = %d, want 2 (apply + rollback)", syncCalls)
		}
		if len(failures) != 2 {
			t.Fatalf("ApplyActiveConfig() failures = %#v, want mcp + mcp_rollback", failures)
		}
		if failures[0].Subsystem != "mcp" {
			t.Fatalf("first failure subsystem = %q, want mcp", failures[0].Subsystem)
		}
		if failures[1].Subsystem != "mcp_rollback" {
			t.Fatalf("second failure subsystem = %q, want mcp_rollback", failures[1].Subsystem)
		}
	})

	t.Run("Should restore marketplace sources when another live dependency fails", func(t *testing.T) {
		t.Parallel()

		firstServer := newMarketplaceFeedServer(t, "rollback-first")
		secondServer := newMarketplaceFeedServer(t, "rollback-second")
		homePaths := testHomePaths(t)
		previous := aghconfig.DefaultWithHome(homePaths)
		previous.Marketplace.Catalog.BaseURL = firstServer.URL
		previous.Marketplace.Catalog.Timeout = "1s"
		next := previous
		next.Marketplace.Catalog.BaseURL = secondServer.URL

		registry, err := globaldb.OpenGlobalDB(
			t.Context(),
			filepath.Join(t.TempDir(), store.GlobalDatabaseName),
		)
		if err != nil {
			t.Fatalf("OpenGlobalDB() error = %v", err)
		}
		t.Cleanup(func() {
			if err := registry.Close(context.Background()); err != nil {
				t.Errorf("Close() error = %v", err)
			}
		})
		marketplaceStore, err := marketplace.NewSQLiteStore(registry)
		if err != nil {
			t.Fatalf("NewSQLiteStore() error = %v", err)
		}
		runtime, err := newMarketplaceRuntime(marketplaceStore, nil, previous.Marketplace.Catalog, nil)
		if err != nil {
			t.Fatalf("newMarketplaceRuntime() error = %v", err)
		}
		if _, err := runtime.Refresh(t.Context(), marketplace.KindSkill); err != nil {
			t.Fatalf("Refresh(seed) error = %v", err)
		}

		syncCalls := 0
		failures := daemonSettingsRuntimeApplier{
			daemon: &Daemon{},
			state: &bootState{
				cfg:         previous,
				marketplace: runtime,
				toolMCPResources: toolMCPPublisherFunc(func(context.Context) error {
					syncCalls++
					if syncCalls == 1 {
						return errors.New("mcp sync boom")
					}
					return nil
				}),
			},
		}.ApplyActiveConfig(t.Context(), &next)
		if len(failures) != 1 || failures[0].Subsystem != "mcp" {
			t.Fatalf("ApplyActiveConfig() failures = %#v, want one mcp failure", failures)
		}
		if _, err := runtime.Refresh(t.Context(), marketplace.KindSkill); err != nil {
			t.Fatalf("Refresh(after rollback) error = %v", err)
		}
		assertMarketplaceRuntimeEntry(t, runtime, "rollback-first")
	})
}

func assertMissingCLIReport(t *testing.T, label string, report providers.PreStartReport) {
	t.Helper()

	if report.Item == nil {
		t.Fatalf("PreStart(%s).Item = nil, want diagnostic", label)
	}
	if report.Item.Code != diagcontract.CodeProviderCLIMissing {
		t.Fatalf("PreStart(%s).Code = %q, want %q", label, report.Item.Code, diagcontract.CodeProviderCLIMissing)
	}
}
