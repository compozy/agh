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
allowed_marketplace_mcp = ["marketplace-a", "marketplace-b"]

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
	if got, want := cfg.Skills.AllowedMarketplaceMCP, []string{"marketplace-a", "marketplace-b"}; !slices.Equal(got, want) {
		t.Fatalf("ApplyConfigOverlayFile() Skills.AllowedMarketplaceMCP = %#v, want %#v", got, want)
	}
	if got, want := cfg.Skills.Marketplace.Registry, "clawhub"; got != want {
		t.Fatalf("ApplyConfigOverlayFile() Skills.Marketplace.Registry = %q, want %q", got, want)
	}
	if got, want := cfg.Skills.Marketplace.BaseURL, "https://registry.example.test/api/v1"; got != want {
		t.Fatalf("ApplyConfigOverlayFile() Skills.Marketplace.BaseURL = %q, want %q", got, want)
	}
}

func TestApplyConfigOverlayFileLeavesMarketplaceDefaultsWhenOverlayOmitsFields(t *testing.T) {
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
}
