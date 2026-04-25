package config

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/environment"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/resources"
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

[session.limits]
timeout = "30m"

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

[extensions.marketplace]
registry = "github"
base_url = "https://api.github.example.test"

[memory]
enabled = true
global_dir = "~/agh-memory-test"

[memory.dream]
enabled = true
agent = "claude"
min_hours = 48
min_sessions = 5
check_interval = "45m"

[network]
enabled = true
default_channel = "builders"
port = 4333
max_payload = 65536
greet_interval = 45
max_replay_age = 600
max_queue_depth = 250
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
	if got, want := cfg.Session.Limits.Timeout, 30*time.Minute; got != want {
		t.Fatalf("Load() Session.Limits.Timeout = %s, want %s", got, want)
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
	if got, want := cfg.Skills.AllowedMarketplaceMCP, []string{
		"@registry/skill-a",
		"@registry/skill-b",
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("Load() Skills.AllowedMarketplaceMCP = %#v, want %#v", got, want)
	}
	if got, want := cfg.Skills.AllowedMarketplaceHooks, []string{
		"@registry/hook-a",
		"@registry/hook-b",
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("Load() Skills.AllowedMarketplaceHooks = %#v, want %#v", got, want)
	}
	if got, want := cfg.Skills.Marketplace.Registry, "clawhub"; got != want {
		t.Fatalf("Load() Skills.Marketplace.Registry = %q, want %q", got, want)
	}
	if got, want := cfg.Skills.Marketplace.BaseURL, "https://registry.example.test/api/v1"; got != want {
		t.Fatalf("Load() Skills.Marketplace.BaseURL = %q, want %q", got, want)
	}
	if got, want := cfg.Extensions.Marketplace.Registry, "github"; got != want {
		t.Fatalf("Load() Extensions.Marketplace.Registry = %q, want %q", got, want)
	}
	if got, want := cfg.Extensions.Marketplace.BaseURL, "https://api.github.example.test"; got != want {
		t.Fatalf("Load() Extensions.Marketplace.BaseURL = %q, want %q", got, want)
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
	if !cfg.Network.Enabled {
		t.Fatal("Load() Network.Enabled = false, want true")
	}
	if got, want := cfg.Network.DefaultChannel, "builders"; got != want {
		t.Fatalf("Load() Network.DefaultChannel = %q, want %q", got, want)
	}
	if got, want := cfg.Network.Port, 4333; got != want {
		t.Fatalf("Load() Network.Port = %d, want %d", got, want)
	}
	if got, want := cfg.Network.MaxPayload, 65536; got != want {
		t.Fatalf("Load() Network.MaxPayload = %d, want %d", got, want)
	}
	if got, want := cfg.Network.GreetInterval, 45; got != want {
		t.Fatalf("Load() Network.GreetInterval = %d, want %d", got, want)
	}
	if got, want := cfg.Network.MaxReplayAge, 600; got != want {
		t.Fatalf("Load() Network.MaxReplayAge = %d, want %d", got, want)
	}
	if got, want := cfg.Network.MaxQueueDepth, 250; got != want {
		t.Fatalf("Load() Network.MaxQueueDepth = %d, want %d", got, want)
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

func TestLoadEnvironmentProfilesFromTOML(t *testing.T) {
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
[defaults]
environment = "daytona-dev"

[environments.local]
backend = "local"

[environments.daytona-dev]
backend = "daytona"
sync_mode = "session-bidirectional"
persistence = "reuse"
runtime_root = "/home/daytona/workspace"

[environments.daytona-dev.env]
NODE_ENV = "development"
AGH_PROFILE = "daytona"

[environments.daytona-dev.network]
allow_public_ingress = false
allow_outbound = true
allow_list = ["api.example.test"]
deny_list = ["metadata.google.internal"]

[environments.daytona-dev.daytona]
api_url = "https://app.daytona.io/api"
target = "team-default"
image = "ubuntu:24.04"
snapshot = "snap-agent-base"
class = "cpu-2"
auto_stop = "30m"
auto_archive = "24h"
`)

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := cfg.Defaults.Environment, "daytona-dev"; got != want {
		t.Fatalf("Defaults.Environment = %q, want %q", got, want)
	}
	profile := cfg.Environments["daytona-dev"]
	if profile.Backend != "daytona" || profile.Daytona.Snapshot != "snap-agent-base" {
		t.Fatalf("daytona profile = %#v, want parsed profile", profile)
	}
	if got, want := profile.Env["NODE_ENV"], "development"; got != want {
		t.Fatalf("profile Env[NODE_ENV] = %q, want %q", got, want)
	}

	resolved, err := cfg.ResolveEnvironment(cfg.Defaults.Environment)
	if err != nil {
		t.Fatalf("ResolveEnvironment() error = %v", err)
	}
	if resolved.Backend != environment.BackendDaytona {
		t.Fatalf("resolved.Backend = %q, want %q", resolved.Backend, environment.BackendDaytona)
	}
	if resolved.SyncMode != environment.SyncModeSessionBidirectional ||
		resolved.Persistence != environment.PersistenceReuse {
		t.Fatalf("resolved sync/persistence = %q/%q", resolved.SyncMode, resolved.Persistence)
	}
	if resolved.Daytona == nil {
		t.Fatal("resolved.Daytona = nil, want profile")
	}
	if got, want := resolved.Daytona.StartupSource, environment.DaytonaStartupSourceSnapshot; got != want {
		t.Fatalf("resolved Daytona startup source = %q, want %q", got, want)
	}
	if got, want := resolved.Daytona.StartupRef, "snap-agent-base"; got != want {
		t.Fatalf("resolved Daytona startup ref = %q, want %q", got, want)
	}
}

func TestDaytonaSnapshotWinsOverImageInResolvedProfile(t *testing.T) {
	t.Parallel()

	resolved, err := (EnvironmentProfile{
		Backend: "daytona",
		Daytona: DaytonaProfile{
			Image:    "ubuntu:24.04",
			Snapshot: "snap-prebuilt",
		},
	}).Resolve("daytona-dev")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Daytona == nil {
		t.Fatal("resolved.Daytona = nil, want profile")
	}
	if got, want := resolved.Daytona.StartupSource, environment.DaytonaStartupSourceSnapshot; got != want {
		t.Fatalf("StartupSource = %q, want %q", got, want)
	}
	if got, want := resolved.Daytona.StartupRef, "snap-prebuilt"; got != want {
		t.Fatalf("StartupRef = %q, want %q", got, want)
	}
	if got, want := resolved.Daytona.Image, "ubuntu:24.04"; got != want {
		t.Fatalf("Image = %q, want preserved fallback %q", got, want)
	}
}

func TestEnvironmentProfileValidationRejectsInvalidBackend(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	cfg := DefaultWithHome(homePaths)
	cfg.Environments["bad"] = EnvironmentProfile{Backend: "docker"}

	err = cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want invalid backend")
	}
	if !strings.Contains(err.Error(), "environments.bad.backend") {
		t.Fatalf("Validate() error = %v, want environments.bad.backend", err)
	}
}

func TestEnvironmentProfileValidationRejectsInvalidSyncMode(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	cfg := DefaultWithHome(homePaths)
	cfg.Environments["bad"] = EnvironmentProfile{
		Backend:  "daytona",
		SyncMode: "continuous",
	}

	err = cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want invalid sync_mode")
	}
	if !strings.Contains(err.Error(), "environments.bad.sync_mode") {
		t.Fatalf("Validate() error = %v, want environments.bad.sync_mode", err)
	}
}

func TestEnvironmentProfileValidationRejectsInvalidPersistence(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	cfg := DefaultWithHome(homePaths)
	cfg.Environments["bad"] = EnvironmentProfile{
		Backend:     "daytona",
		Persistence: "forever",
	}

	err = cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want invalid persistence")
	}
	if !strings.Contains(err.Error(), "environments.bad.persistence") {
		t.Fatalf("Validate() error = %v, want environments.bad.persistence", err)
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

[session.limits]
timeout = "20m"

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

[session.limits]
timeout = "45m"

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
	if got, want := cfg.Session.Limits.Timeout, 45*time.Minute; got != want {
		t.Fatalf("Load() Session.Limits.Timeout = %s, want %s", got, want)
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

func TestLoadMergesTopLevelMCPServersAcrossConfigAndJSONSidecars(t *testing.T) {
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
[[mcp_servers]]
name = "partial"
command = "global-command"

[[mcp_servers]]
name = "sidecar"
command = "global-inline"
args = ["--inline"]
`)
	writeFile(t, filepath.Join(homePaths.HomeDir, MCPJSONName), `{
  "mcpServers": {
    "sidecar": {
      "command": "global-sidecar"
    }
  }
}`)
	writeFile(t, filepath.Join(workspaceRoot, DirName, ConfigName), `
[[mcp_servers]]
name = "partial"
args = ["--workspace"]
env = { WORKSPACE = "1" }

[[mcp_servers]]
name = "workspace-only"
command = "workspace-command"

[[mcp_servers]]
name = "replace-me"
command = "workspace-inline"
`)
	writeFile(t, filepath.Join(workspaceRoot, DirName, MCPJSONName), `{
  "mcp_servers": {
    "replace-me": {
      "command": "workspace-sidecar"
    },
    "workspace-json": {
      "command": "workspace-json-command"
    }
  }
}`)

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := len(cfg.MCPServers), 5; got != want {
		t.Fatalf("Load() MCPServers len = %d, want %d (%#v)", got, want, cfg.MCPServers)
	}
	if got, want := cfg.MCPServers[0].Name, "partial"; got != want {
		t.Fatalf("Load() MCPServers[0].Name = %q, want %q", got, want)
	}
	if got, want := cfg.MCPServers[0].Command, "global-command"; got != want {
		t.Fatalf("Load() partial.Command = %q, want %q", got, want)
	}
	if got, want := cfg.MCPServers[0].Args[0], "--workspace"; got != want {
		t.Fatalf("Load() partial.Args = %#v, want workspace field merge", cfg.MCPServers[0].Args)
	}
	if got, want := cfg.MCPServers[0].Env["WORKSPACE"], "1"; got != want {
		t.Fatalf("Load() partial.Env[WORKSPACE] = %q, want %q", got, want)
	}
	if got, want := cfg.MCPServers[1].Command, "global-sidecar"; got != want {
		t.Fatalf("Load() sidecar.Command = %q, want %q", got, want)
	}
	if got := len(cfg.MCPServers[1].Args); got != 0 {
		t.Fatalf("Load() sidecar.Args = %#v, want same-scope sidecar replacement", cfg.MCPServers[1].Args)
	}
	if got, want := cfg.MCPServers[2].Command, "workspace-command"; got != want {
		t.Fatalf("Load() workspace-only.Command = %q, want %q", got, want)
	}
	if got, want := cfg.MCPServers[3].Command, "workspace-sidecar"; got != want {
		t.Fatalf("Load() replace-me.Command = %q, want %q", got, want)
	}
	if got, want := cfg.MCPServers[4].Command, "workspace-json-command"; got != want {
		t.Fatalf("Load() workspace-json.Command = %q, want %q", got, want)
	}
}

func TestLoadSupportsRemoteMCPAuthFieldsInTOML(t *testing.T) {
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
[[mcp_servers]]
name = "linear"
transport = "sse"
url = "https://mcp.example/sse"

[mcp_servers.auth]
type = "oauth2_pkce"
authorization_url = "https://auth.example/authorize"
token_url = "https://auth.example/token"
client_id = "client-id"
client_secret_env = "LINEAR_CLIENT_SECRET"
scopes = ["read"]

[[providers.codex.mcp_servers]]
name = "remote-provider"
transport = "http"
url = "https://provider.example/mcp"

[providers.codex.mcp_servers.auth]
type = "oauth2_pkce"
issuer_url = "https://issuer.example"
client_id = "provider-client"
scopes = ["tools"]
`)

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot))
	if err != nil {
		t.Fatalf("Load(remote MCP TOML) error = %v", err)
	}

	if got, want := len(cfg.MCPServers), 1; got != want {
		t.Fatalf("Load() MCPServers len = %d, want %d (%#v)", got, want, cfg.MCPServers)
	}
	linear := cfg.MCPServers[0]
	if got, want := linear.Transport, MCPServerTransportSSE; got != want {
		t.Fatalf("Load() linear.Transport = %q, want %q", got, want)
	}
	if got, want := linear.URL, "https://mcp.example/sse"; got != want {
		t.Fatalf("Load() linear.URL = %q, want %q", got, want)
	}
	if got, want := linear.Auth.Type, MCPAuthTypeOAuth2PKCE; got != want {
		t.Fatalf("Load() linear.Auth.Type = %q, want %q", got, want)
	}
	if got, want := linear.Auth.AuthorizationURL, "https://auth.example/authorize"; got != want {
		t.Fatalf("Load() linear.Auth.AuthorizationURL = %q, want %q", got, want)
	}
	if got, want := linear.Auth.TokenURL, "https://auth.example/token"; got != want {
		t.Fatalf("Load() linear.Auth.TokenURL = %q, want %q", got, want)
	}
	if got, want := linear.Auth.ClientSecretEnv, "LINEAR_CLIENT_SECRET"; got != want {
		t.Fatalf("Load() linear.Auth.ClientSecretEnv = %q, want %q", got, want)
	}
	if got, want := linear.Auth.Scopes, []string{"read"}; !slices.Equal(got, want) {
		t.Fatalf("Load() linear.Auth.Scopes = %#v, want %#v", got, want)
	}

	codex, err := cfg.ResolveProvider("codex")
	if err != nil {
		t.Fatalf("ResolveProvider(codex) error = %v", err)
	}
	if got, want := len(codex.MCPServers), 1; got != want {
		t.Fatalf("ResolveProvider(codex) MCPServers len = %d, want %d (%#v)", got, want, codex.MCPServers)
	}
	providerRemote := codex.MCPServers[0]
	if got, want := providerRemote.Transport, MCPServerTransportHTTP; got != want {
		t.Fatalf("provider remote Transport = %q, want %q", got, want)
	}
	if got, want := providerRemote.Auth.IssuerURL, "https://issuer.example"; got != want {
		t.Fatalf("provider remote IssuerURL = %q, want %q", got, want)
	}
	if got, want := providerRemote.Auth.ClientID, "provider-client"; got != want {
		t.Fatalf("provider remote ClientID = %q, want %q", got, want)
	}
}

func TestSessionLimitsConfigValidateRejectsNegativeTimeout(t *testing.T) {
	t.Run("Should reject negative timeout", func(t *testing.T) {
		t.Parallel()

		cfg := SessionLimitsConfig{Timeout: -time.Second}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("SessionLimitsConfig.Validate() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "session.limits.timeout") {
			t.Fatalf("SessionLimitsConfig.Validate() error = %v, want session.limits.timeout context", err)
		}
	})
}

func TestSessionSupervisionConfigValidateRejectsWarningAfterTimeout(t *testing.T) {
	t.Run("Should reject warning threshold after timeout", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultSessionSupervisionConfig()
		cfg.InactivityWarningAfter = 2 * time.Minute
		cfg.InactivityTimeout = time.Minute

		err := cfg.Validate()
		if err == nil {
			t.Fatal("SessionSupervisionConfig.Validate() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "session.supervision.inactivity_warning_after") ||
			!strings.Contains(err.Error(), "session.supervision.inactivity_timeout") {
			t.Fatalf("SessionSupervisionConfig.Validate() error = %v, want threshold context", err)
		}
	})
}

func TestObservabilityConfigValidateRetentionDays(t *testing.T) {
	t.Run("Should allow zero as keep history", func(t *testing.T) {
		t.Parallel()

		cfg := validObservabilityConfigForTest()
		cfg.RetentionDays = 0

		if err := cfg.Validate(); err != nil {
			t.Fatalf("ObservabilityConfig.Validate() error = %v, want nil for keep-history retention", err)
		}
	})

	t.Run("Should reject negative retention days", func(t *testing.T) {
		t.Parallel()

		cfg := validObservabilityConfigForTest()
		cfg.RetentionDays = -1

		err := cfg.Validate()
		if err == nil {
			t.Fatal("ObservabilityConfig.Validate() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "observability.retention_days must be zero or positive") {
			t.Fatalf("ObservabilityConfig.Validate() error = %v, want retention_days context", err)
		}
	})
}

func validObservabilityConfigForTest() ObservabilityConfig {
	return ObservabilityConfig{
		Enabled:        true,
		MaxGlobalBytes: 1,
		Transcripts: ObservabilityTranscriptConfig{
			Enabled:            true,
			SegmentBytes:       1,
			MaxBytesPerSession: 1,
		},
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

	if cfg.Observability.Enabled != true || cfg.Observability.RetentionDays != 7 ||
		cfg.Observability.MaxGlobalBytes != 1000 {
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
		t.Fatalf(
			"DefaultWithHome() Skills.AllowedMarketplaceMCP = %#v, want nil/empty",
			cfg.Skills.AllowedMarketplaceMCP,
		)
	}
	if cfg.Skills.AllowedMarketplaceHooks != nil {
		t.Fatalf(
			"DefaultWithHome() Skills.AllowedMarketplaceHooks = %#v, want nil/empty",
			cfg.Skills.AllowedMarketplaceHooks,
		)
	}
	if cfg.Skills.Marketplace != (MarketplaceConfig{}) {
		t.Fatalf("DefaultWithHome() Skills.Marketplace = %#v, want zero value", cfg.Skills.Marketplace)
	}
	if cfg.Extensions.Marketplace != (ExtensionsMarketplaceConfig{}) {
		t.Fatalf("DefaultWithHome() Extensions.Marketplace = %#v, want zero value", cfg.Extensions.Marketplace)
	}
	if len(cfg.Extensions.Resources.AllowedKinds) != 0 ||
		cfg.Extensions.Resources.MaxScope != "" ||
		cfg.Extensions.Resources.SnapshotRateLimit != (ExtensionsResourceRateLimitConfig{}) ||
		cfg.Extensions.Resources.OperatorWriteRateLimit != (ExtensionsResourceRateLimitConfig{}) {
		t.Fatalf("DefaultWithHome() Extensions.Resources = %#v, want zero value", cfg.Extensions.Resources)
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

func TestExtensionsConfigValidateMarketplaceConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfg         ExtensionsConfig
		wantErrPath string
	}{
		{
			name: "ShouldAcceptValidMarketplaceConfig",
			cfg: ExtensionsConfig{
				Marketplace: ExtensionsMarketplaceConfig{
					Registry: "github",
					BaseURL:  "https://api.github.example.test",
				},
			},
		},
		{
			name: "ShouldAcceptEmptyMarketplaceConfig",
			cfg:  ExtensionsConfig{},
		},
		{
			name: "ShouldRejectMarketplaceBaseURLWithoutHost",
			cfg: ExtensionsConfig{
				Marketplace: ExtensionsMarketplaceConfig{
					Registry: "github",
					BaseURL:  "https://",
				},
			},
			wantErrPath: "extensions.marketplace.base_url",
		},
		{
			name: "ShouldRejectUnknownMarketplaceRegistry",
			cfg: ExtensionsConfig{
				Marketplace: ExtensionsMarketplaceConfig{
					Registry: "unknown",
					BaseURL:  "https://api.github.example.test",
				},
			},
			wantErrPath: "extensions.marketplace.registry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErrPath == "" {
				if err != nil {
					t.Fatalf("ExtensionsConfig.Validate() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ExtensionsConfig.Validate() error = nil, want marketplace validation failure")
			}
			if !strings.Contains(err.Error(), tt.wantErrPath) {
				t.Fatalf("ExtensionsConfig.Validate() error = %v, want %s context", err, tt.wantErrPath)
			}
		})
	}

	t.Run("ShouldWarnForHTTPBaseURL", func(t *testing.T) {
		// This subtest swaps slog.Default(), so the parent test must remain
		// non-parallel to avoid cross-test interference.
		var logs bytes.Buffer
		original := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn})))
		defer slog.SetDefault(original)

		cfg := ExtensionsConfig{
			Marketplace: ExtensionsMarketplaceConfig{
				Registry: "github",
				BaseURL:  "http://api.github.example.test",
			},
		}

		if err := cfg.Validate(); err != nil {
			t.Fatalf("ExtensionsConfig.Validate(http) error = %v", err)
		}
		if !strings.Contains(logs.String(), "insecure http scheme") {
			t.Fatalf("ExtensionsConfig.Validate(http) logs = %q, want insecure http scheme warning", logs.String())
		}
	})
}

func TestExtensionsConfigValidateResourcesConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cfg         ExtensionsConfig
		wantErrPath string
	}{
		{
			name: "ShouldAcceptValidResourcePolicy",
			cfg: ExtensionsConfig{
				Resources: ExtensionsResourcesConfig{
					AllowedKinds: []resources.ResourceKind{
						resources.ResourceKind("tool"),
						resources.ResourceKind("mcp_server"),
					},
					MaxScope: resources.ResourceScopeKindWorkspace,
					SnapshotRateLimit: ExtensionsResourceRateLimitConfig{
						Requests: 2,
						Window:   5 * time.Second,
						Queue:    1,
					},
					OperatorWriteRateLimit: ExtensionsResourceRateLimitConfig{
						Requests: 10,
						Window:   time.Minute,
						Queue:    0,
					},
				},
			},
		},
		{
			name: "ShouldRejectDaemonOnlyAllowedKind",
			cfg: ExtensionsConfig{
				Resources: ExtensionsResourcesConfig{
					AllowedKinds: []resources.ResourceKind{resources.ResourceKind("bridge.instance")},
				},
			},
			wantErrPath: "extensions.resources.allowed_kinds",
		},
		{
			name: "ShouldRejectInvalidResourceMaxScope",
			cfg: ExtensionsConfig{
				Resources: ExtensionsResourcesConfig{
					MaxScope: resources.ResourceScopeKind("session"),
				},
			},
			wantErrPath: "extensions.resources.max_scope",
		},
		{
			name: "ShouldRejectInvalidSnapshotRateLimit",
			cfg: ExtensionsConfig{
				Resources: ExtensionsResourcesConfig{
					SnapshotRateLimit: ExtensionsResourceRateLimitConfig{
						Requests: 0,
						Window:   time.Second,
					},
				},
			},
			wantErrPath: "extensions.resources.snapshot_rate_limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErrPath == "" {
				if err != nil {
					t.Fatalf("ExtensionsConfig.Validate() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ExtensionsConfig.Validate() error = nil, want validation failure")
			}
			if !strings.Contains(err.Error(), tt.wantErrPath) {
				t.Fatalf("ExtensionsConfig.Validate() error = %v, want %s context", err, tt.wantErrPath)
			}
		})
	}
}

func TestLoadRoundTripsExtensionsResourcePolicy(t *testing.T) {
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
[extensions.resources]
allowed_kinds = ["tool", "mcp_server"]
max_scope = "global"

[extensions.resources.snapshot_rate_limit]
requests = 3
window = "15s"
queue = 1

[extensions.resources.operator_write_rate_limit]
requests = 12
window = "1m"
queue = 0
`)

	writeFile(t, filepath.Join(workspaceRoot, DirName, ConfigName), `
[extensions.resources]
allowed_kinds = ["tool"]
max_scope = "workspace"

[extensions.resources.snapshot_rate_limit]
requests = 1
window = "5s"
queue = 1
`)

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := cfg.Extensions.Resources.AllowedKinds, []resources.ResourceKind{
		resources.ResourceKind("tool"),
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("Load() AllowedKinds = %#v, want %#v", got, want)
	}
	if got, want := cfg.Extensions.Resources.MaxScope, resources.ResourceScopeKindWorkspace; got != want {
		t.Fatalf("Load() MaxScope = %q, want %q", got, want)
	}
	if got, want := cfg.Extensions.Resources.SnapshotRateLimit.Requests, 1; got != want {
		t.Fatalf("Load() SnapshotRateLimit.Requests = %d, want %d", got, want)
	}
	if got, want := cfg.Extensions.Resources.SnapshotRateLimit.Window, 5*time.Second; got != want {
		t.Fatalf("Load() SnapshotRateLimit.Window = %s, want %s", got, want)
	}
	if got, want := cfg.Extensions.Resources.OperatorWriteRateLimit.Requests, 12; got != want {
		t.Fatalf("Load() OperatorWriteRateLimit.Requests = %d, want %d", got, want)
	}
	if got, want := cfg.Extensions.Resources.OperatorWriteRateLimit.Window, time.Minute; got != want {
		t.Fatalf("Load() OperatorWriteRateLimit.Window = %s, want %s", got, want)
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

func TestLoadUsesDotEnvForAGHHomeWithoutMutatingProcessEnv(t *testing.T) {
	workspaceRoot := t.TempDir()
	homeRoot := filepath.Join(t.TempDir(), "dotenv-home")

	unsetEnvForTest(t, "AGH_HOME")
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
	if _, ok := os.LookupEnv("AGH_HOME"); ok {
		t.Fatal("Load() mutated process AGH_HOME, want workspace dotenv scoped to the current load only")
	}
}

func TestLoadForHomeDoesNotLeakDotEnvSecretsAcrossWorkspaceLoads(t *testing.T) {
	const secretEnv = "AGH_CONFIG_TASK09_WEBHOOK_SECRET"

	unsetEnvForTest(t, secretEnv)

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	writeFile(t, homePaths.ConfigFile, `
[automation]
timezone = "UTC"
max_concurrent_jobs = 1
default_fire_limit = { max = 1, window = "5m" }

[[automation.triggers]]
scope = "global"
name = "deploy"
event = "webhook"
endpoint_slug = "deploy-review"
agent = "summarizer"
prompt = "Review {{ index .Data \"payload\" }}"
webhook_secret_env = "`+secretEnv+`"
`)

	workspaceWithEnv := t.TempDir()
	writeFile(t, filepath.Join(workspaceWithEnv, ".env"), secretEnv+"=workspace-only-secret\n")

	if _, err := LoadForHome(homePaths, WithWorkspaceRoot(workspaceWithEnv)); err != nil {
		t.Fatalf("LoadForHome(workspaceWithEnv) error = %v", err)
	}

	workspaceWithoutEnv := t.TempDir()
	_, err = LoadForHome(homePaths, WithWorkspaceRoot(workspaceWithoutEnv))
	if err == nil {
		t.Fatal(
			"LoadForHome(workspaceWithoutEnv) error = nil, want missing webhook secret env after isolated dotenv load",
		)
	}
	if got := err.Error(); !strings.Contains(got, "webhook_secret_env") {
		t.Fatalf("LoadForHome(workspaceWithoutEnv) error = %q, want webhook_secret_env detail", got)
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

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot), withoutDotEnv())
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

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot), withoutValidation())
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

	if cfg.HTTP != want.HTTP || cfg.Defaults != want.Defaults || cfg.Limits != want.Limits ||
		cfg.Permissions != want.Permissions {
		t.Fatalf("Load() = %#v, want defaults %#v", cfg, want)
	}
	if cfg.Daemon.Socket != want.Daemon.Socket {
		t.Fatalf("Load() Daemon.Socket = %q, want %q", cfg.Daemon.Socket, want.Daemon.Socket)
	}
	if cfg.Memory != want.Memory {
		t.Fatalf("Load() Memory = %#v, want %#v", cfg.Memory, want.Memory)
	}
	if cfg.Skills.Enabled != want.Skills.Enabled || cfg.Skills.PollInterval != want.Skills.PollInterval ||
		!slices.Equal(cfg.Skills.DisabledSkills, want.Skills.DisabledSkills) {
		t.Fatalf("Load() Skills = %#v, want %#v", cfg.Skills, want.Skills)
	}
	if cfg.Network != want.Network {
		t.Fatalf("Load() Network = %#v, want %#v", cfg.Network, want.Network)
	}
	if !cfg.Network.Enabled {
		t.Fatal("Load() Network.Enabled = false, want true by default")
	}
}

func TestDefaultConfigUsesResolvedHomePaths(t *testing.T) {
	t.Setenv("AGH_HOME", "")

	cfg, err := defaultConfig()
	if err != nil {
		t.Fatalf("defaultConfig() error = %v", err)
	}
	if cfg.HTTP.Port != 2123 || cfg.Defaults.Agent != DefaultAgentName {
		t.Fatalf("defaultConfig() = %#v", cfg)
	}
	if cfg.Permissions.Mode != PermissionModeApproveAll {
		t.Fatalf("defaultConfig() Permissions.Mode = %q, want %q", cfg.Permissions.Mode, PermissionModeApproveAll)
	}
	if cfg.Memory.Dream.Agent != DefaultAgentName {
		t.Fatalf("defaultConfig() Memory.Dream.Agent = %q, want %q", cfg.Memory.Dream.Agent, DefaultAgentName)
	}
	if !cfg.Skills.Enabled {
		t.Fatal("defaultConfig() Skills.Enabled = false, want true")
	}
	if got, want := cfg.Skills.PollInterval, 3*time.Second; got != want {
		t.Fatalf("defaultConfig() Skills.PollInterval = %s, want %s", got, want)
	}
	if !cfg.Network.Enabled {
		t.Fatal("defaultConfig() Network.Enabled = false, want true")
	}
	if got, want := cfg.Network.DefaultChannel, "default"; got != want {
		t.Fatalf("defaultConfig() Network.DefaultChannel = %q, want %q", got, want)
	}
	if got, want := cfg.Network.Port, -1; got != want {
		t.Fatalf("defaultConfig() Network.Port = %d, want %d", got, want)
	}
	if got, want := cfg.Network.MaxPayload, 1<<20; got != want {
		t.Fatalf("defaultConfig() Network.MaxPayload = %d, want %d", got, want)
	}
}

func TestLoadRespectsExplicitNetworkDisable(t *testing.T) {
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
[network]
enabled = false
default_channel = "operators"
`)

	cfg, err := Load(WithWorkspaceRoot(workspaceRoot))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Network.Enabled {
		t.Fatal("Load() Network.Enabled = true, want explicit false override to win")
	}
	if got, want := cfg.Network.DefaultChannel, "operators"; got != want {
		t.Fatalf("Load() Network.DefaultChannel = %q, want %q", got, want)
	}
}

func TestNetworkConfigValidateRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name: "invalid port",
			mutate: func(cfg *Config) {
				cfg.Network.Port = 0
			},
			wantErr: "network.port",
		},
		{
			name: "invalid payload",
			mutate: func(cfg *Config) {
				cfg.Network.MaxPayload = 0
			},
			wantErr: "network.max_payload",
		},
		{
			name: "payload over int32",
			mutate: func(cfg *Config) {
				cfg.Network.MaxPayload = 1 << 31
			},
			wantErr: "network.max_payload",
		},
		{
			name: "invalid greet interval",
			mutate: func(cfg *Config) {
				cfg.Network.GreetInterval = 0
			},
			wantErr: "network.greet_interval",
		},
		{
			name: "invalid replay age",
			mutate: func(cfg *Config) {
				cfg.Network.MaxReplayAge = 0
			},
			wantErr: "network.max_replay_age",
		},
		{
			name: "invalid queue depth",
			mutate: func(cfg *Config) {
				cfg.Network.MaxQueueDepth = 0
			},
			wantErr: "network.max_queue_depth",
		},
		{
			name: "ShouldRejectInvalidDefaultChannel",
			mutate: func(cfg *Config) {
				cfg.Network.DefaultChannel = "Bad Channel"
			},
			wantErr: "network.default_channel",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := DefaultWithHome(homePaths)
			tc.mutate(&cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatalf("Validate() error = nil for %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() error = %q, want substring %q", err, tc.wantErr)
			}
		})
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

func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()

	value, hadValue := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv(%q) error = %v", key, err)
	}

	t.Cleanup(func() {
		var err error
		if hadValue {
			err = os.Setenv(key, value)
		} else {
			err = os.Unsetenv(key)
		}
		if err != nil {
			t.Fatalf("restore env %q error = %v", key, err)
		}
	})
}
