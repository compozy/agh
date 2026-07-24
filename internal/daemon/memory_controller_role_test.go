package daemon

import (
	"testing"
	"time"

	aghconfig "github.com/compozy/agh/internal/config"
)

func TestMemoryControllerRoleCallOptions(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve every pinned default in the model call options", func(t *testing.T) {
		t.Parallel()

		cfg := roleResolverConfig()
		options, err := newRoleResolver(&cfg, nil, nil).resolveMemoryControllerCallOptions(t.Context(), "")
		if err != nil {
			t.Fatalf("resolveMemoryControllerCallOptions() error = %v", err)
		}
		if !options.Enabled || options.Model != "anthropic/claude-haiku-4" ||
			options.Timeout != 250*time.Millisecond || options.TopK != 5 ||
			options.PromptVersion != "v1" || options.MaxTokensOut != 256 {
			t.Fatalf("resolveMemoryControllerCallOptions() = %#v, want pinned defaults", options)
		}
	})

	t.Run("Should use the effective workspace role knobs", func(t *testing.T) {
		t.Parallel()

		global := roleResolverConfig()
		workspace := global
		workspace.Roles.MemoryController = aghconfig.MemoryControllerRoleConfig{
			Enabled:         true,
			Provider:        "gateway",
			Model:           "workspace-controller",
			ReasoningEffort: "high",
			Timeout:         time.Second,
			TopK:            8,
			PromptVersion:   "v2",
			MaxTokensOut:    512,
			FallbackChain: []aghconfig.RoleFallback{{
				Provider: "backup",
				Model:    "backup-controller",
			}},
		}
		resolver := newRoleResolver(&global, roleWorkspaceResolverStub{configs: map[string]aghconfig.Config{
			"ws-a": workspace,
		}}, nil)
		options, err := resolver.resolveMemoryControllerCallOptions(t.Context(), "ws-a")
		if err != nil {
			t.Fatalf("resolveMemoryControllerCallOptions() error = %v", err)
		}
		if options.Provider != "gateway" || options.Model != "workspace-controller" ||
			options.ReasoningEffort != "high" || options.Timeout != time.Second ||
			options.TopK != 8 || options.PromptVersion != "v2" || options.MaxTokensOut != 512 ||
			len(options.Fallbacks) != 1 {
			t.Fatalf("resolveMemoryControllerCallOptions() = %#v, want workspace options", options)
		}
	})
}
