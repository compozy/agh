package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestDefaultWithHomeIncludesAutonomyCoordinatorDefaults(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg := DefaultWithHome(homePaths)
	coordinator := cfg.Autonomy.Coordinator
	if coordinator.Enabled {
		t.Fatal("DefaultWithHome() Autonomy.Coordinator.Enabled = true, want false")
	}
	if got, want := coordinator.AgentName, DefaultCoordinatorAgentName; got != want {
		t.Fatalf("DefaultWithHome() coordinator AgentName = %q, want %q", got, want)
	}
	if coordinator.Provider != "" || coordinator.Model != "" {
		t.Fatalf(
			"DefaultWithHome() coordinator provider/model = %q/%q, want bundled fallback",
			coordinator.Provider,
			coordinator.Model,
		)
	}
	if got, want := coordinator.DefaultTTL, DefaultCoordinatorTTL; got != want {
		t.Fatalf("DefaultWithHome() coordinator DefaultTTL = %s, want %s", got, want)
	}
	if got, want := coordinator.MaxChildren, DefaultCoordinatorMaxChildren; got != want {
		t.Fatalf("DefaultWithHome() coordinator MaxChildren = %d, want %d", got, want)
	}
	if got, want := coordinator.MaxActivePerWorkspace, DefaultCoordinatorMaxActivePerWorkspace; got != want {
		t.Fatalf("DefaultWithHome() coordinator MaxActivePerWorkspace = %d, want %d", got, want)
	}
}

func TestSchedulerConfigValidateMonotonic(t *testing.T) {
	t.Parallel()

	t.Run("Should accept the monotonic defaults", func(t *testing.T) {
		t.Parallel()
		if err := DefaultSchedulerConfig().Validate("autonomy.scheduler"); err != nil {
			t.Fatalf("DefaultSchedulerConfig().Validate() error = %v, want nil", err)
		}
	})

	t.Run("Should reject non-positive and non-monotonic thresholds", func(t *testing.T) {
		t.Parallel()
		base := DefaultSchedulerConfig()
		cases := []struct {
			name            string
			wantErrContains string
			mutate          func(*SchedulerConfig)
		}{
			{
				"Should reject non-positive fan_out",
				"fan_out_after must be positive",
				func(c *SchedulerConfig) { c.FanOutAfter = 0 },
			},
			{
				"Should reject spawn before fan_out",
				"spawn_after must be >= fan_out_after",
				func(c *SchedulerConfig) { c.SpawnAfter = c.FanOutAfter - 1 },
			},
			{
				"Should reject event before spawn",
				"event_after must be >= spawn_after",
				func(c *SchedulerConfig) { c.EventAfter = c.SpawnAfter - 1 },
			},
			{
				"Should reject needs_attention before event",
				"needs_attention_after must be >= event_after",
				func(c *SchedulerConfig) { c.NeedsAttentionAfter = c.EventAfter - 1 },
			},
			{
				"Should reject non-positive min_queued_age",
				"min_queued_age must be positive",
				func(c *SchedulerConfig) { c.MinQueuedAge = 0 },
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				cfg := base
				tc.mutate(&cfg)
				err := cfg.Validate("autonomy.scheduler")
				if err == nil {
					t.Fatalf("Validate(%s) error = nil, want rejection", tc.name)
				}
				if !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Fatalf("Validate(%s) error = %v, want substring %q", tc.name, err, tc.wantErrContains)
				}
			})
		}
	})
}

func TestLoadWorkspaceOverridesAutonomyCoordinatorValues(t *testing.T) {
	workspaceRoot, homePaths := prepareAutonomyConfigTestEnv(t)

	writeFile(t, homePaths.ConfigFile, `
[autonomy.coordinator]
enabled = true
agent_name = "global-coordinator"
provider = "claude"
model = "global-model"
default_ttl = "2h"
max_children = 5
max_active_per_workspace = 1
`)
	writeFile(t, filepath.Join(workspaceRoot, DirName, ConfigName), `
[autonomy.coordinator]
enabled = false
agent_name = "workspace-coordinator"
provider = "codex"
model = "workspace-model"
default_ttl = "3h"
max_children = 2
max_active_per_workspace = 1
`)

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	coordinator := cfg.Autonomy.Coordinator
	if coordinator.Enabled {
		t.Fatal("Load() coordinator Enabled = true, want workspace false")
	}
	if got, want := coordinator.AgentName, "workspace-coordinator"; got != want {
		t.Fatalf("Load() coordinator AgentName = %q, want %q", got, want)
	}
	if got, want := coordinator.Provider, "codex"; got != want {
		t.Fatalf("Load() coordinator Provider = %q, want %q", got, want)
	}
	if got, want := coordinator.Model, "workspace-model"; got != want {
		t.Fatalf("Load() coordinator Model = %q, want %q", got, want)
	}
	if got, want := coordinator.DefaultTTL, 3*time.Hour; got != want {
		t.Fatalf("Load() coordinator DefaultTTL = %s, want %s", got, want)
	}
	if got, want := coordinator.MaxChildren, 2; got != want {
		t.Fatalf("Load() coordinator MaxChildren = %d, want %d", got, want)
	}
}

func TestLoadRejectsInvalidAutonomyCoordinatorConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name: "empty agent name",
			config: `
[autonomy.coordinator]
agent_name = ""
`,
			wantErr: "autonomy.coordinator.agent_name",
		},
		{
			name: "unknown provider",
			config: `
[autonomy.coordinator]
provider = "missing-provider"
model = "model"
`,
			wantErr: "autonomy.coordinator.provider",
		},
		{
			name: "pi provider without default model",
			config: `
[providers.custom-pi]
command = "npx -y pi-acp@latest"
harness = "pi_acp"
[[providers.custom-pi.credential_slots]]
name = "api_key"
target_env = "CUSTOM_API_KEY"
secret_ref = "env:CUSTOM_API_KEY"
kind = "api_key"
required = true

[autonomy.coordinator]
provider = "custom-pi"
`,
			wantErr: "autonomy.coordinator.model",
		},
		{
			name: "ttl too short",
			config: `
[autonomy.coordinator]
default_ttl = "30s"
`,
			wantErr: "autonomy.coordinator.default_ttl",
		},
		{
			name: "ttl too long",
			config: `
[autonomy.coordinator]
default_ttl = "25h"
`,
			wantErr: "autonomy.coordinator.default_ttl",
		},
		{
			name: "negative max children",
			config: `
[autonomy.coordinator]
max_children = -1
`,
			wantErr: "autonomy.coordinator.max_children",
		},
		{
			name: "excess max children",
			config: `
[autonomy.coordinator]
max_children = 6
`,
			wantErr: "autonomy.coordinator.max_children",
		},
		{
			name: "coordinator uniqueness limit",
			config: `
[autonomy.coordinator]
max_active_per_workspace = 2
`,
			wantErr: "autonomy.coordinator.max_active_per_workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspaceRoot, homePaths := prepareAutonomyConfigTestEnv(t)
			writeFile(t, homePaths.ConfigFile, tt.config)

			_, err := Load(WithWorkspaceRoot(workspaceRoot))
			if err == nil {
				t.Fatal("Load() error = nil, want validation failure")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Load() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadAllowsDirectACPAutonomyProviderWithoutModelDefault(t *testing.T) {
	t.Run("Should accept provider-managed model for direct ACP provider", func(t *testing.T) {
		workspaceRoot, homePaths := prepareAutonomyConfigTestEnv(t)
		writeFile(t, homePaths.ConfigFile, `
[autonomy.coordinator]
provider = "opencode"
`)

		cfg, err := Load(WithWorkspaceRoot(workspaceRoot))
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if got, want := cfg.Autonomy.Coordinator.Provider, "opencode"; got != want {
			t.Fatalf("Load() coordinator Provider = %q, want %q", got, want)
		}
		if got := cfg.Autonomy.Coordinator.Model; got != "" {
			t.Fatalf("Load() coordinator Model = %q, want empty", got)
		}
	})
}

func TestLoadRejectsUnknownAutonomyConfigKeys(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name: "unknown autonomy key",
			config: `
[autonomy]
unknown = true
`,
			wantErr: "autonomy.unknown",
		},
		{
			name: "unknown coordinator key",
			config: `
[autonomy.coordinator]
unknown = true
`,
			wantErr: "autonomy.coordinator.unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspaceRoot, homePaths := prepareAutonomyConfigTestEnv(t)
			writeFile(t, homePaths.ConfigFile, tt.config)

			_, err := Load(WithWorkspaceRoot(workspaceRoot))
			if err == nil {
				t.Fatal("Load() error = nil, want unknown-key failure")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Load() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestResolveCoordinatorConfigUsesProviderModelPrecedence(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	cfg := DefaultWithHome(homePaths)
	cfg.Defaults.Provider = "codex"
	fallback := AgentDef{
		Name:     DefaultCoordinatorAgentName,
		Provider: "claude",
		Model:    "fallback-model",
		Prompt:   "fallback prompt",
	}

	resolved, err := cfg.ResolveCoordinatorConfig(fallback)
	if err != nil {
		t.Fatalf("ResolveCoordinatorConfig() error = %v", err)
	}
	if got, want := resolved.Provider, "claude"; got != want {
		t.Fatalf("ResolveCoordinatorConfig() Provider = %q, want fallback %q", got, want)
	}
	if got, want := resolved.Model, "fallback-model"; got != want {
		t.Fatalf("ResolveCoordinatorConfig() Model = %q, want fallback %q", got, want)
	}

	cfg.Autonomy.Coordinator.Provider = "codex"
	cfg.Autonomy.Coordinator.Model = "config-model"
	resolved, err = cfg.ResolveCoordinatorConfig(fallback)
	if err != nil {
		t.Fatalf("ResolveCoordinatorConfig(config override) error = %v", err)
	}
	if got, want := resolved.Provider, "codex"; got != want {
		t.Fatalf("ResolveCoordinatorConfig() Provider = %q, want config %q", got, want)
	}
	if got, want := resolved.Model, "config-model"; got != want {
		t.Fatalf("ResolveCoordinatorConfig() Model = %q, want config %q", got, want)
	}
}

func TestResolveCoordinatorConfigAllowsDirectACPProviderManagedModel(t *testing.T) {
	t.Parallel()

	t.Run("Should resolve without model for direct ACP provider", func(t *testing.T) {
		t.Parallel()

		homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}
		cfg := DefaultWithHome(homePaths)
		cfg.Defaults.Provider = "opencode"

		resolved, err := cfg.ResolveCoordinatorConfig(AgentDef{Name: DefaultCoordinatorAgentName})
		if err != nil {
			t.Fatalf("ResolveCoordinatorConfig() error = %v", err)
		}
		if got, want := resolved.Provider, "opencode"; got != want {
			t.Fatalf("ResolveCoordinatorConfig() Provider = %q, want %q", got, want)
		}
		if got := resolved.Model; got != "" {
			t.Fatalf("ResolveCoordinatorConfig() Model = %q, want empty", got)
		}
	})
}

func TestLoadAutonomyOverlayPreservesOtherConfigSections(t *testing.T) {
	workspaceRoot, homePaths := prepareAutonomyConfigTestEnv(t)

	writeFile(t, homePaths.ConfigFile, `
	[providers.claude]
	auth_mode = "bound_secret"
	[providers.claude.models]
	default = "global-model"
	[[providers.claude.credential_slots]]
	name = "api_key"
	target_env = "GLOBAL_KEY"
	secret_ref = "env:GLOBAL_KEY"
	kind = "api_key"
	required = true

[[hooks.declarations]]
name = "shared"
event = "tool.pre_call"
mode = "sync"
timeout = "5s"

[hooks.declarations.executor]
command = "/bin/global"

[network]
enabled = true
default_channel = "builders"
port = 4333
max_payload = 65536
greet_interval = 45
max_replay_age = 600
max_queue_depth = 250

[memory]
enabled = true

[memory.dream]
enabled = true
agent = "general"
min_hours = 36
min_sessions = 4
check_interval = "20m"

[skills]
enabled = true
disabled_skills = ["global-skill"]
poll_interval = "4s"
allowed_marketplace_mcp = ["@global/mcp"]
allowed_marketplace_hooks = ["@global/hook"]

[skills.marketplace]
registry = "clawhub"
base_url = "https://global.example.test/api/v1"

[autonomy.coordinator]
enabled = true
provider = "claude"
model = "global-coordinator"
default_ttl = "2h"
max_children = 5
max_active_per_workspace = 1
	`)
	writeFile(t, filepath.Join(workspaceRoot, DirName, ConfigName), `
	[providers.claude]
	auth_mode = "bound_secret"
	[[providers.claude.credential_slots]]
	name = "api_key"
	target_env = "WORKSPACE_KEY"
	secret_ref = "env:WORKSPACE_KEY"
	kind = "api_key"
	required = true

	[[hooks.declarations]]
name = "workspace-only"
event = "tool.pre_call"
mode = "sync"

[hooks.declarations.executor]
command = "/bin/workspace"

[network]
max_queue_depth = 400

[memory.dream]
min_sessions = 6

[skills]
poll_interval = "9s"

[autonomy.coordinator]
model = "workspace-coordinator"
default_ttl = "3h"
max_children = 2
`)

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	claude, err := cfg.ResolveProvider("claude")
	if err != nil {
		t.Fatalf("ResolveProvider(claude) error = %v", err)
	}
	if claude.Models.Default != "global-model" {
		t.Fatalf("ResolveProvider(claude) = %#v, want merged provider fields", claude)
	}
	if slots := claude.EffectiveCredentialSlots(); len(slots) != 1 ||
		slots[0].TargetEnv != "WORKSPACE_KEY" ||
		slots[0].SecretRef != "env:WORKSPACE_KEY" {
		t.Fatalf("ResolveProvider(claude) CredentialSlots = %#v, want workspace slot", slots)
	}
	decls, err := HookDeclarations(cfg.Hooks, nil)
	if err != nil {
		t.Fatalf("HookDeclarations() error = %v", err)
	}
	if got, want := len(decls), 2; got != want {
		t.Fatalf("len(HookDeclarations()) = %d, want %d", got, want)
	}
	if cfg.Network.DefaultChannel != "builders" || cfg.Network.MaxQueueDepth != 400 {
		t.Fatalf("Load() Network = %#v, want preserved channel and workspace queue depth", cfg.Network)
	}
	if cfg.Memory.Dream.MinHours != 36 || cfg.Memory.Dream.MinSessions != 6 {
		t.Fatalf("Load() Memory.Dream = %#v, want merged dream config", cfg.Memory.Dream)
	}
	if got, want := cfg.Skills.DisabledSkills, []string{"global-skill"}; !slices.Equal(got, want) {
		t.Fatalf("Load() Skills.DisabledSkills = %#v, want %#v", got, want)
	}
	if got, want := cfg.Skills.PollInterval, 9*time.Second; got != want {
		t.Fatalf("Load() Skills.PollInterval = %s, want %s", got, want)
	}
	if got, want := cfg.Autonomy.Coordinator.Model, "workspace-coordinator"; got != want {
		t.Fatalf("Load() coordinator Model = %q, want %q", got, want)
	}
	if got, want := cfg.Autonomy.Coordinator.DefaultTTL, 3*time.Hour; got != want {
		t.Fatalf("Load() coordinator DefaultTTL = %s, want %s", got, want)
	}
	if got, want := cfg.Autonomy.Coordinator.MaxChildren, 2; got != want {
		t.Fatalf("Load() coordinator MaxChildren = %d, want %d", got, want)
	}
}

func TestLoadAutonomyDoesNotUseAmbientWorkspaceOrMutateEnv(t *testing.T) {
	workspaceRoot, homePaths := prepareAutonomyConfigTestEnv(t)
	ambientWorkspace := t.TempDir()
	writeFile(t, homePaths.ConfigFile, `
[autonomy.coordinator]
provider = "claude"
model = "global-coordinator"
`)
	writeFile(t, filepath.Join(workspaceRoot, DirName, ConfigName), `
[autonomy.coordinator]
model = "target-workspace"
`)
	writeFile(t, filepath.Join(ambientWorkspace, DirName, ConfigName), `
[autonomy.coordinator]
model = "ambient-workspace"
`)

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(ambientWorkspace); err != nil {
		t.Fatalf("Chdir(%q) error = %v", ambientWorkspace, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previousWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
	beforeHome := os.Getenv("AGH_HOME")

	cfg, err := LoadForHome(homePaths, WithWorkspaceRoot(workspaceRoot))
	if err != nil {
		t.Fatalf("LoadForHome(target workspace) error = %v", err)
	}
	if got, want := cfg.Autonomy.Coordinator.Model, "target-workspace"; got != want {
		t.Fatalf("LoadForHome() coordinator Model = %q, want explicit workspace %q", got, want)
	}
	if got := os.Getenv("AGH_HOME"); got != beforeHome {
		t.Fatalf("LoadForHome() mutated AGH_HOME = %q, want %q", got, beforeHome)
	}

	globalOnly, err := LoadForHome(homePaths)
	if err != nil {
		t.Fatalf("LoadForHome(global only) error = %v", err)
	}
	if got, want := globalOnly.Autonomy.Coordinator.Model, "global-coordinator"; got != want {
		t.Fatalf("LoadForHome(global only) coordinator Model = %q, want %q", got, want)
	}
}

func prepareAutonomyConfigTestEnv(t *testing.T) (string, HomePaths) {
	t.Helper()

	workspaceRoot := t.TempDir()
	homeRoot := filepath.Join(t.TempDir(), "home")
	t.Setenv("AGH_HOME", homeRoot)

	homePaths, err := ResolveHomePaths()
	if err != nil {
		t.Fatalf("ResolveHomePaths() error = %v", err)
	}
	if err := EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	return workspaceRoot, homePaths
}
