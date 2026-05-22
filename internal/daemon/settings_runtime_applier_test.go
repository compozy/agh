package daemon

import (
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
