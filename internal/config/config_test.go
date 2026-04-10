package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

func TestLoadValidTOMLConfigWithAllSections(t *testing.T) {
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

	writeFile(t, homePaths.ConfigFile, `
[daemon]
socket = "~/.agh/custom.sock"

[http]
host = "127.0.0.1"
port = 3030

[defaults]
agent = "researcher"
provider = "claude"

[limits]
max_sessions = 11
max_concurrent_agents = 22

[permissions]
mode = "approve-all"

[providers.claude]
default_model = "claude-opus"
api_key_env = "ANTHROPIC_KEY"
[[providers.claude.mcp_servers]]
name = "github"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-github"]
env = { GITHUB_TOKEN = "x" }

[observability]
enabled = true
retention_days = 14
max_global_bytes = 2048

[observability.transcripts]
enabled = true
segment_bytes = 512
max_bytes_per_session = 4096

[log]
level = "debug"

[skills]
enabled = false
disabled_skills = ["code-review", "agh-session-guide"]
poll_interval = "5s"
allowed_marketplace_mcp = ["@registry/skill-a", "@registry/skill-b"]
allowed_marketplace_hooks = ["@registry/hook-a", "@registry/hook-b"]

[skills.marketplace]
registry = "clawhub"
base_url = "https://registry.example.test/api/v1"

[memory]
enabled = true
global_dir = "~/agh-memory-test"

[memory.dream]
enabled = true
agent = "claude"
min_hours = 48
min_sessions = 5
check_interval = "45m"
`)

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Daemon.Socket == "~/.agh/custom.sock" {
		t.Fatalf("Load() did not normalize daemon socket path: %q", cfg.Daemon.Socket)
	}
	if cfg.HTTP.Host != "127.0.0.1" || cfg.HTTP.Port != 3030 {
		t.Fatalf("Load() HTTP = %#v", cfg.HTTP)
	}
	if cfg.Defaults.Agent != "researcher" {
		t.Fatalf("Load() Defaults.Agent = %q, want %q", cfg.Defaults.Agent, "researcher")
	}
	if cfg.Defaults.Provider != "claude" {
		t.Fatalf("Load() Defaults.Provider = %q, want %q", cfg.Defaults.Provider, "claude")
	}
	if cfg.Limits.MaxSessions != 11 || cfg.Limits.MaxConcurrentAgents != 22 {
		t.Fatalf("Load() Limits = %#v", cfg.Limits)
	}
	if cfg.Permissions.Mode != PermissionModeApproveAll {
		t.Fatalf("Load() Permissions.Mode = %q, want %q", cfg.Permissions.Mode, PermissionModeApproveAll)
	}
	if cfg.Observability.RetentionDays != 14 || cfg.Observability.MaxGlobalBytes != 2048 {
		t.Fatalf("Load() Observability = %#v", cfg.Observability)
	}
	if cfg.Observability.Transcripts.SegmentBytes != 512 || cfg.Observability.Transcripts.MaxBytesPerSession != 4096 {
		t.Fatalf("Load() Transcript config = %#v", cfg.Observability.Transcripts)
	}
	if cfg.Log.Level != "debug" {
		t.Fatalf("Load() Log.Level = %q, want %q", cfg.Log.Level, "debug")
	}
	if cfg.Skills.Enabled {
		t.Fatal("Load() Skills.Enabled = true, want false")
	}
	if got, want := cfg.Skills.PollInterval, 5*time.Second; got != want {
		t.Fatalf("Load() Skills.PollInterval = %s, want %s", got, want)
	}
	if got, want := cfg.Skills.DisabledSkills, []string{"code-review", "agh-session-guide"}; !slices.Equal(got, want) {
		t.Fatalf("Load() Skills.DisabledSkills = %#v, want %#v", got, want)
	}
	if got, want := cfg.Skills.AllowedMarketplaceMCP, []string{"@registry/skill-a", "@registry/skill-b"}; !slices.Equal(got, want) {
		t.Fatalf("Load() Skills.AllowedMarketplaceMCP = %#v, want %#v", got, want)
	}
	if got, want := cfg.Skills.AllowedMarketplaceHooks, []string{"@registry/hook-a", "@registry/hook-b"}; !slices.Equal(got, want) {
		t.Fatalf("Load() Skills.AllowedMarketplaceHooks = %#v, want %#v", got, want)
	}
	if got, want := cfg.Skills.Marketplace.Registry, "clawhub"; got != want {
		t.Fatalf("Load() Skills.Marketplace.Registry = %q, want %q", got, want)
	}
	if got, want := cfg.Skills.Marketplace.BaseURL, "https://registry.example.test/api/v1"; got != want {
		t.Fatalf("Load() Skills.Marketplace.BaseURL = %q, want %q", got, want)
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}
	if !cfg.Memory.Enabled {
		t.Fatal("Load() Memory.Enabled = false, want true")
	}
	if got, want := cfg.Memory.GlobalDir, filepath.Join(userHome, "agh-memory-test"); got != want {
		t.Fatalf("Load() Memory.GlobalDir = %q, want %q", got, want)
	}
	if got, want := cfg.Memory.Dream.Agent, "claude"; got != want {
		t.Fatalf("Load() Memory.Dream.Agent = %q, want %q", got, want)
	}
	if got, want := cfg.Memory.Dream.MinHours, 48.0; got != want {
		t.Fatalf("Load() Memory.Dream.MinHours = %v, want %v", got, want)
	}
	if got, want := cfg.Memory.Dream.MinSessions, 5; got != want {
		t.Fatalf("Load() Memory.Dream.MinSessions = %d, want %d", got, want)
	}
	if got, want := cfg.Memory.Dream.CheckInterval, 45*time.Minute; got != want {
		t.Fatalf("Load() Memory.Dream.CheckInterval = %s, want %s", got, want)
	}

	claude, err := cfg.ResolveProvider("claude")
	if err != nil {
		t.Fatalf("ResolveProvider() error = %v", err)
	}
	if claude.Command == "" || claude.DefaultModel != "claude-opus" || claude.APIKeyEnv != "ANTHROPIC_KEY" {
		t.Fatalf("ResolveProvider() = %#v", claude)
	}
	if len(claude.MCPServers) != 1 || claude.MCPServers[0].Name != "github" {
		t.Fatalf("ResolveProvider() MCPServers = %#v", claude.MCPServers)
	}
}

func TestLoadWorkspaceOverridesGlobalValues(t *testing.T) {
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

	writeFile(t, homePaths.ConfigFile, `
[http]
host = "localhost"
port = 2123

[providers.claude]
default_model = "global-model"
api_key_env = "GLOBAL_KEY"

[skills]
enabled = true
disabled_skills = ["global-skill"]
poll_interval = "3s"
allowed_marketplace_mcp = ["@global/skill"]
allowed_marketplace_hooks = ["@global/hook"]

[skills.marketplace]
registry = "clawhub"
base_url = "https://global.example.test/api/v1"
`)
	writeFile(t, filepath.Join(workspaceRoot, DirName, ConfigName), `
[http]
port = 4242

[providers.claude]
default_model = "workspace-model"

[skills]
enabled = false
disabled_skills = ["workspace-skill"]
poll_interval = "9s"
allowed_marketplace_mcp = ["@workspace/skill"]
allowed_marketplace_hooks = ["@workspace/hook"]

[skills.marketplace]
registry = "clawhub"
base_url = "https://workspace.example.test/api/v1"
`)

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTP.Host != "localhost" || cfg.HTTP.Port != 4242 {
		t.Fatalf("Load() HTTP = %#v", cfg.HTTP)
	}

	claude, err := cfg.ResolveProvider("claude")
	if err != nil {
		t.Fatalf("ResolveProvider() error = %v", err)
	}
	if claude.DefaultModel != "workspace-model" {
		t.Fatalf("ResolveProvider() DefaultModel = %q, want %q", claude.DefaultModel, "workspace-model")
	}
	if claude.APIKeyEnv != "GLOBAL_KEY" {
		t.Fatalf("ResolveProvider() APIKeyEnv = %q, want %q", claude.APIKeyEnv, "GLOBAL_KEY")
	}
	if cfg.Skills.Enabled {
		t.Fatal("Load() Skills.Enabled = true, want false")
	}
	if got, want := cfg.Skills.AllowedMarketplaceMCP, []string{"@workspace/skill"}; !slices.Equal(got, want) {
		t.Fatalf("Load() Skills.AllowedMarketplaceMCP = %#v, want %#v", got, want)
	}
	if got, want := cfg.Skills.AllowedMarketplaceHooks, []string{"@workspace/hook"}; !slices.Equal(got, want) {
		t.Fatalf("Load() Skills.AllowedMarketplaceHooks = %#v, want %#v", got, want)
	}
	if got, want := cfg.Skills.Marketplace.Registry, "clawhub"; got != want {
		t.Fatalf("Load() Skills.Marketplace.Registry = %q, want %q", got, want)
	}
	if got, want := cfg.Skills.Marketplace.BaseURL, "https://workspace.example.test/api/v1"; got != want {
		t.Fatalf("Load() Skills.Marketplace.BaseURL = %q, want %q", got, want)
	}
	if got, want := cfg.Skills.PollInterval, 9*time.Second; got != want {
		t.Fatalf("Load() Skills.PollInterval = %s, want %s", got, want)
	}
	if got, want := cfg.Skills.DisabledSkills, []string{"workspace-skill"}; !slices.Equal(got, want) {
		t.Fatalf("Load() Skills.DisabledSkills = %#v, want %#v", got, want)
	}
}

func TestLoadWorkspaceAddsValuesWithoutClobberingGlobal(t *testing.T) {
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

	writeFile(t, homePaths.ConfigFile, `
[observability]
enabled = true
retention_days = 7
max_global_bytes = 1000

[observability.transcripts]
enabled = true
segment_bytes = 128
max_bytes_per_session = 2048

[providers.claude]
default_model = "global-model"
`)
	writeFile(t, filepath.Join(workspaceRoot, DirName, ConfigName), `
[observability.transcripts]
segment_bytes = 256

[providers.claude]
api_key_env = "WORKSPACE_KEY"
`)

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Observability.Enabled != true || cfg.Observability.RetentionDays != 7 || cfg.Observability.MaxGlobalBytes != 1000 {
		t.Fatalf("Load() Observability = %#v", cfg.Observability)
	}
	if cfg.Observability.Transcripts.SegmentBytes != 256 || cfg.Observability.Transcripts.MaxBytesPerSession != 2048 {
		t.Fatalf("Load() Transcripts = %#v", cfg.Observability.Transcripts)
	}

	claude, err := cfg.ResolveProvider("claude")
	if err != nil {
		t.Fatalf("ResolveProvider() error = %v", err)
	}
	if claude.DefaultModel != "global-model" || claude.APIKeyEnv != "WORKSPACE_KEY" {
		t.Fatalf("ResolveProvider() = %#v", claude)
	}
}

func TestLoadWithoutWorkspaceRootIgnoresCurrentDirectoryWorkspaceFiles(t *testing.T) {
	// Intentionally not parallel: this test mutates process-global cwd via os.Chdir.
	homeRoot := filepath.Join(t.TempDir(), "home")
	dotenvHome := filepath.Join(t.TempDir(), "dotenv-home")
	cwd := t.TempDir()

	t.Setenv("AGH_HOME", homeRoot)

	homePaths, err := ResolveHomePaths()
	if err != nil {
		t.Fatalf("ResolveHomePaths() error = %v", err)
	}
	if err := EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	writeFile(t, homePaths.ConfigFile, `
[http]
port = 3030
`)

	dotenvPaths, err := ResolveHomePathsFrom(dotenvHome)
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := EnsureHomeLayout(dotenvPaths); err != nil {
		t.Fatalf("EnsureHomeLayout(dotenv) error = %v", err)
	}
	writeFile(t, dotenvPaths.ConfigFile, `
[http]
port = 9090
`)

	writeFile(t, filepath.Join(cwd, ".env"), "AGH_HOME="+dotenvHome+"\n")
	writeFile(t, filepath.Join(cwd, DirName, ConfigName), `
[http]
port = 4242
`)

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("Chdir(%q) error = %v", cwd, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previousWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := cfg.HTTP.Port, 3030; got != want {
		t.Fatalf("Load() HTTP.Port = %d, want %d", got, want)
	}
	if got, want := cfg.Daemon.Socket, homePaths.DaemonSocket; got != want {
		t.Fatalf("Load() Daemon.Socket = %q, want %q", got, want)
	}
}

func TestLoadWithWorkspaceRootUsesExplicitRootOnly(t *testing.T) {
	workspaceRoot := t.TempDir()
	otherWorkspace := t.TempDir()
	homeRoot := filepath.Join(t.TempDir(), "home")

	t.Setenv("AGH_HOME", homeRoot)

	homePaths, err := ResolveHomePaths()
	if err != nil {
		t.Fatalf("ResolveHomePaths() error = %v", err)
	}
	if err := EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	writeFile(t, homePaths.ConfigFile, `
[http]
host = "localhost"
port = 2123
`)
	writeFile(t, filepath.Join(workspaceRoot, DirName, ConfigName), `
[http]
port = 4242
`)
	writeFile(t, filepath.Join(otherWorkspace, DirName, ConfigName), `
[http]
port = 9999
`)

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(otherWorkspace); err != nil {
		t.Fatalf("Chdir(%q) error = %v", otherWorkspace, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previousWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := cfg.HTTP.Port, 4242; got != want {
		t.Fatalf("Load() HTTP.Port = %d, want %d", got, want)
	}
}

func TestLoadRejectsUnknownConfigKeys(t *testing.T) {
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

	writeFile(t, homePaths.ConfigFile, `
[http]
port = 2123
unknown = true
`)

	if _, err := Load(WithWorkspaceRoot(workspaceRoot)); err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadRejectsUnknownSkillsConfigKeys(t *testing.T) {
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

	writeFile(t, homePaths.ConfigFile, `
[skills]
poll_interval = "3s"
unknown = true
`)

	_, err = Load(WithWorkspaceRoot(workspaceRoot))
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "skills.unknown") {
		t.Fatalf("Load() error = %v, want skills.unknown in message", err)
	}
}

func TestDefaultWithHomeLeavesMarketplaceConfigEmpty(t *testing.T) {
	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg := DefaultWithHome(homePaths)
	if cfg.Skills.AllowedMarketplaceMCP != nil {
		t.Fatalf("DefaultWithHome() Skills.AllowedMarketplaceMCP = %#v, want nil/empty", cfg.Skills.AllowedMarketplaceMCP)
	}
	if cfg.Skills.AllowedMarketplaceHooks != nil {
		t.Fatalf("DefaultWithHome() Skills.AllowedMarketplaceHooks = %#v, want nil/empty", cfg.Skills.AllowedMarketplaceHooks)
	}
	if cfg.Skills.Marketplace != (MarketplaceConfig{}) {
		t.Fatalf("DefaultWithHome() Skills.Marketplace = %#v, want zero value", cfg.Skills.Marketplace)
	}
}

func TestSkillsConfigValidateMarketplaceConfig(t *testing.T) {
	t.Parallel()

	base := SkillsConfig{
		Enabled:      true,
		PollInterval: time.Second,
	}

	t.Run("ShouldAcceptValidMarketplaceConfig", func(t *testing.T) {
		cfg := base
		cfg.Marketplace = MarketplaceConfig{
			Registry: "clawhub",
			BaseURL:  "https://registry.example.test/api/v1",
		}

		if err := cfg.Validate(); err != nil {
			t.Fatalf("SkillsConfig.Validate() error = %v", err)
		}
	})

	t.Run("ShouldRejectEmptyRegistryWhenMarketplaceConfigured", func(t *testing.T) {
		cfg := base
		cfg.Marketplace = MarketplaceConfig{
			BaseURL: "https://registry.example.test/api/v1",
		}

		err := cfg.Validate()
		if err == nil {
			t.Fatal("SkillsConfig.Validate() error = nil, want registry validation failure")
		}
		if !strings.Contains(err.Error(), "skills.marketplace.registry") {
			t.Fatalf("SkillsConfig.Validate() error = %v, want marketplace registry context", err)
		}
	})

	t.Run("ShouldRejectInvalidMarketplaceBaseURL", func(t *testing.T) {
		cfg := base
		cfg.Marketplace = MarketplaceConfig{
			Registry: "clawhub",
			BaseURL:  "ftp://registry.example.test/api/v1",
		}

		err := cfg.Validate()
		if err == nil {
			t.Fatal("SkillsConfig.Validate() error = nil, want marketplace base_url validation failure")
		}
		if !strings.Contains(err.Error(), "skills.marketplace.base_url") {
			t.Fatalf("SkillsConfig.Validate() error = %v, want marketplace base_url context", err)
		}
	})
}

func TestValidateRejectsInvalidPorts(t *testing.T) {
	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	tests := []struct {
		name string
		port int
	}{
		{name: "zero", port: 0},
		{name: "too high", port: 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultWithHome(homePaths)
			cfg.HTTP.Port = tt.port

			if err := cfg.Validate(); err == nil {
				t.Fatalf("Validate() error = nil for port %d", tt.port)
			}
		})
	}
}

func TestValidateRejectsUnknownPermissionMode(t *testing.T) {
	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg := DefaultWithHome(homePaths)
	cfg.Permissions.Mode = PermissionMode("maybe")
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
}

func TestValidateWrapsHooksConfigErrors(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg := DefaultWithHome(homePaths)
	cfg.Hooks.Declarations = []hookspkg.HookDecl{{
		Name:   "broken-hook",
		Event:  "bad.event",
		Source: hookspkg.HookSourceConfig,
	}}

	err = cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "validate hooks config") {
		t.Fatalf("Validate() error = %q, want hooks config context", err)
	}
}

func TestDreamConfigValidateRejectsNonPositiveThresholds(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		patch func(*DreamConfig)
	}{
		{
			name: "min hours",
			patch: func(cfg *DreamConfig) {
				cfg.MinHours = 0
			},
		},
		{
			name: "min sessions",
			patch: func(cfg *DreamConfig) {
				cfg.MinSessions = 0
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := DreamConfig{
				Enabled:       true,
				Agent:         "claude",
				MinHours:      24,
				MinSessions:   3,
				CheckInterval: 30 * time.Minute,
			}
			tc.patch(&cfg)

			if err := cfg.Validate(); err == nil {
				t.Fatalf("Validate() error = nil for %s", tc.name)
			}
		})
	}
}

func TestLoadRejectsNonPositiveSkillsPollInterval(t *testing.T) {
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

	writeFile(t, homePaths.ConfigFile, `
[skills]
poll_interval = "0s"
`)

	_, err = Load(WithWorkspaceRoot(workspaceRoot))
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "skills.poll_interval") {
		t.Fatalf("Load() error = %v, want skills.poll_interval in message", err)
	}
}

func TestLoadUsesDotEnvForAGHHome(t *testing.T) {
	workspaceRoot := t.TempDir()
	homeRoot := filepath.Join(t.TempDir(), "dotenv-home")

	writeFile(t, filepath.Join(workspaceRoot, ".env"), "AGH_HOME="+homeRoot+"\n")

	homePaths, err := ResolveHomePathsFrom(homeRoot)
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	writeFile(t, homePaths.ConfigFile, `
[defaults]
agent = "dotenv-agent"
`)

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Defaults.Agent != "dotenv-agent" {
		t.Fatalf("Load() Defaults.Agent = %q, want %q", cfg.Defaults.Agent, "dotenv-agent")
	}
}

func TestLoadWithoutDotEnvOptionIgnoresDotEnv(t *testing.T) {
	workspaceRoot := t.TempDir()
	envHome := filepath.Join(t.TempDir(), "dotenv-home")
	overrideHome := filepath.Join(t.TempDir(), "override-home")

	t.Setenv("AGH_HOME", overrideHome)
	writeFile(t, filepath.Join(workspaceRoot, ".env"), "AGH_HOME="+envHome+"\n")

	overridePaths, err := ResolveHomePathsFrom(overrideHome)
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot), WithoutDotEnv())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Daemon.Socket != overridePaths.DaemonSocket {
		t.Fatalf("Load() Daemon.Socket = %q, want %q", cfg.Daemon.Socket, overridePaths.DaemonSocket)
	}
}

func TestLoadWithoutValidationReturnsMergedConfig(t *testing.T) {
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

	writeFile(t, homePaths.ConfigFile, `
[http]
host = "localhost"
port = 0
`)

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot), WithoutValidation())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.Port != 0 {
		t.Fatalf("Load() HTTP.Port = %d, want 0", cfg.HTTP.Port)
	}
}

func TestLoadMissingConfigReturnsDefaults(t *testing.T) {
	workspaceRoot := t.TempDir()
	homeRoot := filepath.Join(t.TempDir(), "home")
	t.Setenv("AGH_HOME", homeRoot)

	homePaths, err := ResolveHomePaths()
	if err != nil {
		t.Fatalf("ResolveHomePaths() error = %v", err)
	}
	want := DefaultWithHome(homePaths)

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTP != want.HTTP || cfg.Defaults != want.Defaults || cfg.Limits != want.Limits || cfg.Permissions != want.Permissions {
		t.Fatalf("Load() = %#v, want defaults %#v", cfg, want)
	}
	if cfg.Daemon.Socket != want.Daemon.Socket {
		t.Fatalf("Load() Daemon.Socket = %q, want %q", cfg.Daemon.Socket, want.Daemon.Socket)
	}
	if cfg.Memory != want.Memory {
		t.Fatalf("Load() Memory = %#v, want %#v", cfg.Memory, want.Memory)
	}
	if cfg.Skills.Enabled != want.Skills.Enabled || cfg.Skills.PollInterval != want.Skills.PollInterval || !slices.Equal(cfg.Skills.DisabledSkills, want.Skills.DisabledSkills) {
		t.Fatalf("Load() Skills = %#v, want %#v", cfg.Skills, want.Skills)
	}
}

func TestDefaultUsesResolvedHomePaths(t *testing.T) {
	t.Setenv("AGH_HOME", "")

	cfg, err := Default()
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}
	if cfg.HTTP.Port != 2123 || cfg.Defaults.Agent != DefaultAgentName {
		t.Fatalf("Default() = %#v", cfg)
	}
	if cfg.Permissions.Mode != PermissionModeApproveAll {
		t.Fatalf("Default() Permissions.Mode = %q, want %q", cfg.Permissions.Mode, PermissionModeApproveAll)
	}
	if cfg.Memory.Dream.Agent != DefaultAgentName {
		t.Fatalf("Default() Memory.Dream.Agent = %q, want %q", cfg.Memory.Dream.Agent, DefaultAgentName)
	}
	if !cfg.Skills.Enabled {
		t.Fatal("Default() Skills.Enabled = false, want true")
	}
	if got, want := cfg.Skills.PollInterval, 3*time.Second; got != want {
		t.Fatalf("Default() Skills.PollInterval = %s, want %s", got, want)
	}
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimLeft(contents, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
