package config

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultRolesConfigPreservesRoleBehavior(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve coordinator defaults", func(t *testing.T) {
		t.Parallel()

		got := DefaultRolesConfig().Coordinator
		if got.Enabled {
			t.Fatal("DefaultRolesConfig().Coordinator.Enabled = true, want false")
		}
		if got.Agent != "" {
			t.Fatalf("DefaultRolesConfig().Coordinator.Agent = %q, want empty", got.Agent)
		}
		if got.TTL != 2*time.Hour {
			t.Fatalf("DefaultRolesConfig().Coordinator.TTL = %s, want 2h", got.TTL)
		}
		if got.MaxChildren != 5 {
			t.Fatalf("DefaultRolesConfig().Coordinator.MaxChildren = %d, want 5", got.MaxChildren)
		}
		if got.MaxActiveSessionsPerWorkspace != 5 {
			t.Fatalf(
				"DefaultRolesConfig().Coordinator.MaxActiveSessionsPerWorkspace = %d, want 5",
				got.MaxActiveSessionsPerWorkspace,
			)
		}
	})

	t.Run("Should preserve session-backed role defaults", func(t *testing.T) {
		t.Parallel()

		got := DefaultRolesConfig()
		for name, role := range map[RoleName]RoleConfig{
			RoleDream:             got.Dream,
			RoleCheckpointSummary: got.CheckpointSummary,
			RoleMemoryExtractor:   got.MemoryExtractor,
			RoleAutoTitle:         got.AutoTitle,
		} {
			if !role.Enabled {
				t.Errorf("DefaultRolesConfig().%s.Enabled = false, want true", name)
			}
			if role.Agent != "" {
				t.Errorf("DefaultRolesConfig().%s.Agent = %q, want empty", name, role.Agent)
			}
		}
	})

	t.Run("Should preserve memory controller defaults", func(t *testing.T) {
		t.Parallel()

		got := DefaultRolesConfig().MemoryController
		if !got.Enabled {
			t.Fatal("DefaultRolesConfig().MemoryController.Enabled = false, want true")
		}
		if got.Model != "anthropic/claude-haiku-4" {
			t.Fatalf("DefaultRolesConfig().MemoryController.Model = %q, want anthropic/claude-haiku-4", got.Model)
		}
		if got.Timeout != 250*time.Millisecond {
			t.Fatalf("DefaultRolesConfig().MemoryController.Timeout = %s, want 250ms", got.Timeout)
		}
		if got.TopK != 5 || got.PromptVersion != "v1" || got.MaxTokensOut != 256 {
			t.Fatalf(
				"DefaultRolesConfig().MemoryController = %#v, want top_k=5 prompt_version=v1 max_tokens_out=256",
				got,
			)
		}
	})
}

func TestRolesConfigValidateEnforcesBoundsAndRoutes(t *testing.T) {
	t.Parallel()

	t.Run("Should reject a coordinator TTL below the floor", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultRolesConfig()
		cfg.Coordinator.TTL = 30 * time.Second
		err := cfg.Validate("roles", &Config{})
		if err == nil || !strings.Contains(err.Error(), "roles.coordinator.ttl") ||
			!strings.Contains(err.Error(), "1m0s") || !strings.Contains(err.Error(), "24h0m0s") {
			t.Fatalf("Validate() error = %v, want coordinator TTL path and bounds", err)
		}
	})

	t.Run("Should accept both coordinator TTL boundaries", func(t *testing.T) {
		t.Parallel()

		for _, ttl := range []time.Duration{time.Minute, 24 * time.Hour} {
			cfg := DefaultRolesConfig()
			cfg.Coordinator.TTL = ttl
			if err := cfg.Validate("roles", &Config{}); err != nil {
				t.Fatalf("Validate(ttl=%s) error = %v", ttl, err)
			}
		}
	})

	t.Run("Should reject a coordinator TTL above the ceiling", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultRolesConfig()
		cfg.Coordinator.TTL = 24*time.Hour + time.Second
		err := cfg.Validate("roles", &Config{})
		if err == nil || !strings.Contains(err.Error(), "roles.coordinator.ttl") {
			t.Fatalf("Validate() error = %v, want coordinator TTL error", err)
		}
	})

	t.Run("Should reject coordinator child counts outside the safe range", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			value   int
			message string
		}{
			{value: 0, message: "must be positive"},
			{value: 6, message: "must be <= 5"},
		} {
			cfg := DefaultRolesConfig()
			cfg.Coordinator.MaxChildren = tc.value
			err := cfg.Validate("roles", &Config{})
			if err == nil || !strings.Contains(err.Error(), "roles.coordinator.max_children") ||
				!strings.Contains(err.Error(), tc.message) {
				t.Errorf("Validate(max_children=%d) error = %v, want %q", tc.value, err, tc.message)
			}
		}
	})

	t.Run("Should require a provider for every fallback", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultRolesConfig()
		cfg.Dream.FallbackChain = []RoleFallback{{Model: "model-a"}}
		err := cfg.Validate("roles", &Config{})
		if err == nil || !strings.Contains(err.Error(), "roles.dream.fallback_chain[0].provider is required") {
			t.Fatalf("Validate() error = %v, want fallback provider path", err)
		}
	})

	t.Run("Should reject an unknown fallback provider", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultRolesConfig()
		cfg.Dream.FallbackChain = []RoleFallback{{Provider: "missing", Model: "model-a"}}
		err := cfg.Validate("roles", &Config{})
		if err == nil || !strings.Contains(err.Error(), "roles.dream.fallback_chain[0].provider") ||
			!strings.Contains(err.Error(), "missing") {
			t.Fatalf("Validate() error = %v, want wrapped provider-resolution failure", err)
		}
	})

	t.Run("Should reject an invalid reasoning effort", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultRolesConfig()
		cfg.MemoryExtractor.ReasoningEffort = "extreme"
		err := cfg.Validate("roles", &Config{})
		if err == nil || !strings.Contains(err.Error(), "roles.memory_extractor.reasoning_effort") ||
			!strings.Contains(err.Error(), "extreme") {
			t.Fatalf("Validate() error = %v, want reasoning enum error", err)
		}
	})

	t.Run("Should require a positive enabled controller timeout", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultRolesConfig()
		cfg.MemoryController.Timeout = 0
		err := cfg.Validate("roles", &Config{})
		if err == nil || !strings.Contains(err.Error(), "roles.memory_controller.timeout must be positive") {
			t.Fatalf("Validate() error = %v, want controller timeout error", err)
		}
	})
}

func TestRolesCoordinatorOverlayDecodesEmbeddedFields(t *testing.T) {
	t.Parallel()

	t.Run("Should decode shared fields and coordinator extras from one table", func(t *testing.T) {
		t.Parallel()

		overlay, err := loadConfigOverlayBytes([]byte(`
[roles.coordinator]
enabled = true
model = "model-a"
ttl = "1h"
`), "roles.toml")
		if err != nil {
			t.Fatalf("loadConfigOverlayBytes() error = %v", err)
		}
		cfg := DefaultWithHome(HomePaths{})
		if err := overlay.Apply(&cfg); err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		if !cfg.Roles.Coordinator.Enabled || cfg.Roles.Coordinator.Model != "model-a" ||
			cfg.Roles.Coordinator.TTL != time.Hour {
			t.Fatalf("Config.Roles.Coordinator = %#v, want enabled/model/ttl overlay", cfg.Roles.Coordinator)
		}
	})
}
