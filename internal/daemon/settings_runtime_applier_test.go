package daemon

import (
	"os/exec"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/providers"
)

func TestDaemonSettingsRuntimeApplier(t *testing.T) {
	t.Parallel()

	t.Run("Should invalidate provider prestart cache after active config apply", func(t *testing.T) {
		t.Parallel()

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
		_ = providers.PreStart(t.Context(), provider, env)
		_ = providers.PreStart(t.Context(), provider, env)
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

		_ = providers.PreStart(t.Context(), provider, env)
		if calls != 2 {
			t.Fatalf("PreStart LookPath calls after apply = %d, want 2", calls)
		}
	})
}
