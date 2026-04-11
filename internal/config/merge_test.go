package config

import (
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestApplyConfigOverlayFileAppliesSkillsOverlay(t *testing.T) {
	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg := DefaultWithHome(homePaths)
	cfg.Skills.DisabledSkills = []string{"global-skill"}

	overlayPath := filepath.Join(t.TempDir(), "overlay.toml")
	writeFile(t, overlayPath, `
[skills]
enabled = false
disabled_skills = ["workspace-skill", "code-review"]
poll_interval = "9s"
allowed_marketplace_mcp = ["@registry/mcp-a", "@registry/mcp-b"]
allowed_marketplace_hooks = ["@registry/hook-a", "@registry/hook-b"]

[skills.marketplace]
registry = "clawhub"
base_url = "https://registry.example.test/api/v1"
`)

	if err := ApplyConfigOverlayFile(overlayPath, &cfg); err != nil {
		t.Fatalf("ApplyConfigOverlayFile() error = %v", err)
	}

	if cfg.Skills.Enabled {
		t.Fatal("ApplyConfigOverlayFile() Skills.Enabled = true, want false")
	}
	if got, want := cfg.Skills.PollInterval, 9*time.Second; got != want {
		t.Fatalf("ApplyConfigOverlayFile() Skills.PollInterval = %s, want %s", got, want)
	}
	if got, want := cfg.Skills.DisabledSkills, []string{"workspace-skill", "code-review"}; !slices.Equal(got, want) {
		t.Fatalf("ApplyConfigOverlayFile() Skills.DisabledSkills = %#v, want %#v", got, want)
	}
	if got, want := cfg.Skills.AllowedMarketplaceMCP, []string{"@registry/mcp-a", "@registry/mcp-b"}; !slices.Equal(got, want) {
		t.Fatalf("ApplyConfigOverlayFile() Skills.AllowedMarketplaceMCP = %#v, want %#v", got, want)
	}
	if got, want := cfg.Skills.AllowedMarketplaceHooks, []string{"@registry/hook-a", "@registry/hook-b"}; !slices.Equal(got, want) {
		t.Fatalf("ApplyConfigOverlayFile() Skills.AllowedMarketplaceHooks = %#v, want %#v", got, want)
	}
	if got, want := cfg.Skills.Marketplace.Registry, "clawhub"; got != want {
		t.Fatalf("ApplyConfigOverlayFile() Skills.Marketplace.Registry = %q, want %q", got, want)
	}
	if got, want := cfg.Skills.Marketplace.BaseURL, "https://registry.example.test/api/v1"; got != want {
		t.Fatalf("ApplyConfigOverlayFile() Skills.Marketplace.BaseURL = %q, want %q", got, want)
	}
}

func TestApplyConfigOverlayFileLeavesMarketplaceDefaultsWhenOverlayOmitsFields(t *testing.T) {
	t.Run("ShouldLeaveMarketplaceDefaultsWhenOverlayOmitsFields", func(t *testing.T) {
		homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}

		cfg := DefaultWithHome(homePaths)
		cfg.Skills.Marketplace = MarketplaceConfig{
			Registry: "clawhub",
			BaseURL:  "https://global.example.test/api/v1",
		}

		overlayPath := filepath.Join(t.TempDir(), "overlay.toml")
		writeFile(t, overlayPath, `
[skills]
enabled = true
`)

		if err := ApplyConfigOverlayFile(overlayPath, &cfg); err != nil {
			t.Fatalf("ApplyConfigOverlayFile() error = %v", err)
		}

		if got, want := cfg.Skills.Marketplace.Registry, "clawhub"; got != want {
			t.Fatalf("ApplyConfigOverlayFile() Skills.Marketplace.Registry = %q, want %q", got, want)
		}
		if got, want := cfg.Skills.Marketplace.BaseURL, "https://global.example.test/api/v1"; got != want {
			t.Fatalf("ApplyConfigOverlayFile() Skills.Marketplace.BaseURL = %q, want %q", got, want)
		}
	})
}

func TestApplyConfigOverlayFileAppliesNetworkOverlay(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg := DefaultWithHome(homePaths)

	overlayPath := filepath.Join(t.TempDir(), "overlay.toml")
	writeFile(t, overlayPath, `
[network]
enabled = true
default_space = "builders"
port = 4555
max_payload = 12345
greet_interval = 15
max_replay_age = 90
max_queue_depth = 12
`)

	if err := ApplyConfigOverlayFile(overlayPath, &cfg); err != nil {
		t.Fatalf("ApplyConfigOverlayFile() error = %v", err)
	}

	if !cfg.Network.Enabled {
		t.Fatal("ApplyConfigOverlayFile() Network.Enabled = false, want true")
	}
	if got, want := cfg.Network.DefaultSpace, "builders"; got != want {
		t.Fatalf("ApplyConfigOverlayFile() Network.DefaultSpace = %q, want %q", got, want)
	}
	if got, want := cfg.Network.Port, 4555; got != want {
		t.Fatalf("ApplyConfigOverlayFile() Network.Port = %d, want %d", got, want)
	}
	if got, want := cfg.Network.MaxPayload, 12345; got != want {
		t.Fatalf("ApplyConfigOverlayFile() Network.MaxPayload = %d, want %d", got, want)
	}
	if got, want := cfg.Network.GreetInterval, 15; got != want {
		t.Fatalf("ApplyConfigOverlayFile() Network.GreetInterval = %d, want %d", got, want)
	}
	if got, want := cfg.Network.MaxReplayAge, 90; got != want {
		t.Fatalf("ApplyConfigOverlayFile() Network.MaxReplayAge = %d, want %d", got, want)
	}
	if got, want := cfg.Network.MaxQueueDepth, 12; got != want {
		t.Fatalf("ApplyConfigOverlayFile() Network.MaxQueueDepth = %d, want %d", got, want)
	}
}
