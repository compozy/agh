package daemon

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	aghconfig "github.com/compozy/agh/internal/config"
	diagcontract "github.com/compozy/agh/internal/diagnosticcontract"
	"github.com/compozy/agh/internal/providers"
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
