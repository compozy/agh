package config

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/resources"
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
	if got, want := cfg.Skills.AllowedMarketplaceMCP, []string{
		"@registry/mcp-a",
		"@registry/mcp-b",
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("ApplyConfigOverlayFile() Skills.AllowedMarketplaceMCP = %#v, want %#v", got, want)
	}
	if got, want := cfg.Skills.AllowedMarketplaceHooks, []string{
		"@registry/hook-a",
		"@registry/hook-b",
	}; !slices.Equal(
		got,
		want,
	) {
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

	t.Run("ShouldApplyNetworkOverlay", func(t *testing.T) {
		homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}

		cfg := DefaultWithHome(homePaths)

		overlayPath := filepath.Join(t.TempDir(), "overlay.toml")
		writeFile(t, overlayPath, `
[network]
enabled = true
default_channel = "builders"
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
		if got, want := cfg.Network.DefaultChannel, "builders"; got != want {
			t.Fatalf("ApplyConfigOverlayFile() Network.DefaultChannel = %q, want %q", got, want)
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
	})
}

func TestApplyConfigOverlayFileMergesSandboxProfiles(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg := DefaultWithHome(homePaths)
	cfg.Sandboxes["daytona-dev"] = SandboxProfile{
		Backend:     "daytona",
		SyncMode:    "session-bidirectional",
		Persistence: "reuse",
		RuntimeRoot: "/workspace",
		Env: map[string]string{
			"GLOBAL_ONLY": "true",
			"SHARED":      "global",
		},
		Network: NetworkProfile{
			AllowOutbound: true,
			AllowList:     []string{"api.example.test"},
		},
		Daytona: DaytonaProfile{
			APIURL: "https://app.daytona.io/api",
			Target: "team-default",
			Image:  "ubuntu:24.04",
			Class:  "cpu-2",
		},
	}

	overlayPath := filepath.Join(t.TempDir(), "overlay.toml")
	writeFile(t, overlayPath, `
[defaults]
sandbox = "daytona-dev"

[sandboxes.daytona-dev]
sync_mode = "none"

[sandboxes.daytona-dev.env]
SHARED = "workspace"
WORKSPACE_ONLY = "true"

[sandboxes.daytona-dev.daytona]
snapshot = "snap-workspace"
`)

	if err := ApplyConfigOverlayFile(overlayPath, &cfg); err != nil {
		t.Fatalf("ApplyConfigOverlayFile() error = %v", err)
	}

	profile := cfg.Sandboxes["daytona-dev"]
	if got, want := cfg.Defaults.Sandbox, "daytona-dev"; got != want {
		t.Fatalf("Defaults.Sandbox = %q, want %q", got, want)
	}
	if profile.Backend != "daytona" || profile.Persistence != "reuse" || profile.RuntimeRoot != "/workspace" {
		t.Fatalf("sandbox profile base fields not preserved: %#v", profile)
	}
	if profile.SyncMode != "none" {
		t.Fatalf("sandbox SyncMode = %q, want none", profile.SyncMode)
	}
	if profile.Daytona.APIURL != "https://app.daytona.io/api" ||
		profile.Daytona.Image != "ubuntu:24.04" ||
		profile.Daytona.Snapshot != "snap-workspace" {
		t.Fatalf("Daytona overlay did not preserve provider fields: %#v", profile.Daytona)
	}
	if got, want := profile.Env["GLOBAL_ONLY"], "true"; got != want {
		t.Fatalf("Env[GLOBAL_ONLY] = %q, want %q", got, want)
	}
	if got, want := profile.Env["SHARED"], "workspace"; got != want {
		t.Fatalf("Env[SHARED] = %q, want %q", got, want)
	}
	if got, want := profile.Env["WORKSPACE_ONLY"], "true"; got != want {
		t.Fatalf("Env[WORKSPACE_ONLY] = %q, want %q", got, want)
	}
}

func TestApplyConfigOverlayFileAppliesExtensionsResourceOverlay(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg := DefaultWithHome(homePaths)
	cfg.Extensions.Resources.OperatorWriteRateLimit = ExtensionsResourceRateLimitConfig{
		Requests: 12,
		Window:   time.Minute,
		Queue:    0,
	}

	overlayPath := filepath.Join(t.TempDir(), "overlay.toml")
	writeFile(t, overlayPath, `
[extensions.resources]
allowed_kinds = ["tool"]
max_scope = "workspace"

[extensions.resources.snapshot_rate_limit]
requests = 2
window = "8s"
queue = 1
`)

	if err := ApplyConfigOverlayFile(overlayPath, &cfg); err != nil {
		t.Fatalf("ApplyConfigOverlayFile() error = %v", err)
	}
	if got, want := cfg.Extensions.Resources.AllowedKinds, []resources.ResourceKind{
		resources.ResourceKind("tool"),
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("ApplyConfigOverlayFile() AllowedKinds = %#v, want %#v", got, want)
	}
	if got, want := cfg.Extensions.Resources.MaxScope, resources.ResourceScopeKindWorkspace; got != want {
		t.Fatalf("ApplyConfigOverlayFile() MaxScope = %q, want %q", got, want)
	}
	if got, want := cfg.Extensions.Resources.SnapshotRateLimit.Requests, 2; got != want {
		t.Fatalf("ApplyConfigOverlayFile() SnapshotRateLimit.Requests = %d, want %d", got, want)
	}
	if got, want := cfg.Extensions.Resources.SnapshotRateLimit.Window, 8*time.Second; got != want {
		t.Fatalf("ApplyConfigOverlayFile() SnapshotRateLimit.Window = %s, want %s", got, want)
	}
	if got, want := cfg.Extensions.Resources.OperatorWriteRateLimit.Requests, 12; got != want {
		t.Fatalf("ApplyConfigOverlayFile() OperatorWriteRateLimit.Requests = %d, want %d", got, want)
	}
	if got, want := cfg.Extensions.Resources.OperatorWriteRateLimit.Window, time.Minute; got != want {
		t.Fatalf("ApplyConfigOverlayFile() OperatorWriteRateLimit.Window = %s, want %s", got, want)
	}
}

func TestValidateWrapsNetworkErrorsWithConfigContext(t *testing.T) {
	t.Parallel()

	t.Run("ShouldWrapNetworkValidationErrorsWithConfigContext", func(t *testing.T) {
		homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}

		cfg := DefaultWithHome(homePaths)
		cfg.Network.Port = 0

		err = cfg.Validate()
		if err == nil {
			t.Fatal("Validate() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "validate network config") || !strings.Contains(err.Error(), "network.port") {
			t.Fatalf("Validate() error = %q, want config and network context", err)
		}
	})
}

func TestValidateRejectsOverflowingNetworkDurations(t *testing.T) {
	t.Parallel()

	if strconv.IntSize < 64 {
		t.Skip("overflow validation requires 64-bit int")
	}

	tests := []struct {
		name   string
		mutate func(*Config)
		field  string
	}{
		{
			name: "ShouldRejectOverflowingGreetInterval",
			mutate: func(cfg *Config) {
				cfg.Network.GreetInterval = int(maxNetworkDurationSeconds + 1)
			},
			field: "network.greet_interval",
		},
		{
			name: "ShouldRejectOverflowingReplayAge",
			mutate: func(cfg *Config) {
				cfg.Network.MaxReplayAge = int(maxNetworkDurationSeconds + 1)
			},
			field: "network.max_replay_age",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
			if err != nil {
				t.Fatalf("ResolveHomePathsFrom() error = %v", err)
			}

			cfg := DefaultWithHome(homePaths)
			tc.mutate(&cfg)

			err = cfg.Validate()
			if err == nil {
				t.Fatalf("Validate() error = nil, want non-nil for %s", tc.field)
			}
			if !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("Validate() error = %q, want field %q", err, tc.field)
			}
		})
	}
}
