package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
`)
	writeFile(t, filepath.Join(workspaceRoot, DirName, ConfigName), `
[http]
port = 4242

[providers.claude]
default_model = "workspace-model"
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
}

func TestDefaultUsesResolvedHomePaths(t *testing.T) {
	t.Setenv("AGH_HOME", "")

	cfg, err := Default()
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}
	if cfg.HTTP.Port != 2123 || cfg.Defaults.Agent != "coder" {
		t.Fatalf("Default() = %#v", cfg)
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
