package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/environment"
)

func TestSeedConfigPreservesLiveProviderAndAgentValidation(t *testing.T) {
	t.Parallel()

	homePaths := NewHomePaths(t)
	SeedConfig(t, homePaths, ConfigSeedOptions{
		DefaultAgent: "coder",
		Providers: map[string]aghconfig.ProviderConfig{
			"fake": {
				Command:      "fake-agent --stdio",
				DefaultModel: "fake-model",
			},
		},
		AgentDefs: []AgentSeed{{
			Name:        "coder",
			Provider:    "fake",
			Permissions: string(aghconfig.PermissionModeApproveReads),
			Prompt:      "You are a deterministic test agent.",
		}},
	})

	loaded, err := aghconfig.LoadForHome(homePaths)
	if err != nil {
		t.Fatalf("LoadForHome() error = %v", err)
	}

	agent, err := aghconfig.LoadAgentDef("coder", homePaths)
	if err != nil {
		t.Fatalf("LoadAgentDef(coder) error = %v", err)
	}

	resolved, err := loaded.ResolveAgent(agent)
	if err != nil {
		t.Fatalf("ResolveAgent(coder) error = %v", err)
	}
	if got, want := resolved.Provider, "fake"; got != want {
		t.Fatalf("resolved.Provider = %q, want %q", got, want)
	}
	if got, want := resolved.Command, "fake-agent --stdio"; got != want {
		t.Fatalf("resolved.Command = %q, want %q", got, want)
	}
	if got, want := resolved.Model, "fake-model"; got != want {
		t.Fatalf("resolved.Model = %q, want %q", got, want)
	}
	if got, want := resolved.Permissions, string(aghconfig.PermissionModeApproveReads); got != want {
		t.Fatalf("resolved.Permissions = %q, want %q", got, want)
	}
}

func TestSeedConfigPersistsNetworkOverlay(t *testing.T) {
	t.Parallel()

	homePaths := NewHomePaths(t)
	SeedConfig(t, homePaths, ConfigSeedOptions{
		Mutate: func(cfg *aghconfig.Config) {
			cfg.Network.Enabled = true
			cfg.Network.DefaultChannel = "builders"
		},
	})

	loaded, err := aghconfig.LoadForHome(homePaths)
	if err != nil {
		t.Fatalf("LoadForHome() error = %v", err)
	}
	if !loaded.Network.Enabled {
		t.Fatal("loaded.Network.Enabled = false, want true")
	}
	if got, want := loaded.Network.DefaultChannel, "builders"; got != want {
		t.Fatalf("loaded.Network.DefaultChannel = %q, want %q", got, want)
	}
}

func TestSeedConfigPersistsEnvironmentProfilesAndDefault(t *testing.T) {
	t.Parallel()

	homePaths := NewHomePaths(t)
	SeedConfig(t, homePaths, ConfigSeedOptions{
		DefaultEnvironment: "local-sandbox",
		Environments: map[string]aghconfig.EnvironmentProfile{
			"local-sandbox": {
				Backend:     "local",
				Persistence: "reuse",
				RuntimeRoot: "/workspace/runtime",
				Env: map[string]string{
					"APP_MODE": "test",
				},
			},
		},
	})

	loaded, err := aghconfig.LoadForHome(homePaths)
	if err != nil {
		t.Fatalf("LoadForHome() error = %v", err)
	}
	if got, want := loaded.Defaults.Environment, "local-sandbox"; got != want {
		t.Fatalf("loaded.Defaults.Environment = %q, want %q", got, want)
	}

	resolved, err := loaded.ResolveEnvironment("local-sandbox")
	if err != nil {
		t.Fatalf("ResolveEnvironment(local-sandbox) error = %v", err)
	}
	if got, want := resolved.Backend, environment.BackendLocal; got != want {
		t.Fatalf("resolved.Backend = %q, want %q", got, want)
	}
	if got, want := resolved.Profile, "local-sandbox"; got != want {
		t.Fatalf("resolved.Profile = %q, want %q", got, want)
	}
	if got, want := resolved.RuntimeRootDir, "/workspace/runtime"; got != want {
		t.Fatalf("resolved.RuntimeRootDir = %q, want %q", got, want)
	}
	if resolved.DestroyOnStop {
		t.Fatal("resolved.DestroyOnStop = true, want false for reuse persistence")
	}
	if got, want := resolved.Env["APP_MODE"], "test"; got != want {
		t.Fatalf("resolved.Env[APP_MODE] = %q, want %q", got, want)
	}
}

func TestWriteSeedConfigFileRewritesOverlayWithPermissions(t *testing.T) {
	t.Parallel()

	homePaths := NewHomePaths(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 24242
	cfg.Permissions.Mode = aghconfig.PermissionModeApproveAll

	if err := writeSeedConfigFile(homePaths, &cfg); err != nil {
		t.Fatalf("writeSeedConfigFile() error = %v", err)
	}

	firstContents, err := os.ReadFile(homePaths.ConfigFile)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", homePaths.ConfigFile, err)
	}
	if !strings.Contains(string(firstContents), "port = 24242") {
		t.Fatalf("config contents = %s, want initial port", string(firstContents))
	}
	if !strings.Contains(string(firstContents), "[permissions]") {
		t.Fatalf("config contents = %s, want permissions section", string(firstContents))
	}

	cfg.HTTP.Port = 25252
	if err := writeSeedConfigFile(homePaths, &cfg); err != nil {
		t.Fatalf("writeSeedConfigFile(rewrite) error = %v", err)
	}

	reloaded, err := aghconfig.LoadForHome(homePaths)
	if err != nil {
		t.Fatalf("LoadForHome() error = %v", err)
	}
	if got, want := reloaded.HTTP.Port, 25252; got != want {
		t.Fatalf("reloaded.HTTP.Port = %d, want %d", got, want)
	}
	if got, want := reloaded.Permissions.Mode, aghconfig.PermissionModeApproveAll; got != want {
		t.Fatalf("reloaded.Permissions.Mode = %q, want %q", got, want)
	}
}

func TestPrepareRuntimeLayoutEnvironmentSeedDoesNotLeakBetweenRuns(t *testing.T) {
	t.Parallel()

	first := prepareRuntimeLayout(t, RuntimeHarnessOptions{
		ConfigSeed: ConfigSeedOptions{
			DefaultEnvironment: "local-sandbox",
			Environments: map[string]aghconfig.EnvironmentProfile{
				"local-sandbox": {
					Backend:     "local",
					Persistence: "reuse",
				},
			},
		},
	})
	second := prepareRuntimeLayout(t, RuntimeHarnessOptions{})

	firstLoaded, err := aghconfig.LoadForHome(first.HomePaths)
	if err != nil {
		t.Fatalf("LoadForHome(first) error = %v", err)
	}
	secondLoaded, err := aghconfig.LoadForHome(second.HomePaths)
	if err != nil {
		t.Fatalf("LoadForHome(second) error = %v", err)
	}

	if got, want := firstLoaded.Defaults.Environment, "local-sandbox"; got != want {
		t.Fatalf("firstLoaded.Defaults.Environment = %q, want %q", got, want)
	}
	if _, err := firstLoaded.ResolveEnvironment("local-sandbox"); err != nil {
		t.Fatalf("firstLoaded.ResolveEnvironment(local-sandbox) error = %v", err)
	}

	if got := secondLoaded.Defaults.Environment; got != "" {
		t.Fatalf("secondLoaded.Defaults.Environment = %q, want empty default environment", got)
	}
	if _, err := secondLoaded.ResolveEnvironment("local-sandbox"); err == nil {
		t.Fatal("secondLoaded.ResolveEnvironment(local-sandbox) error = nil, want profile isolation")
	}
}

func TestWriteAgentDefPersistsOptionalSections(t *testing.T) {
	t.Parallel()

	homePaths := NewHomePaths(t)
	WriteAgentDef(t, homePaths, AgentSeed{
		Name:        "builder",
		Provider:    "fake",
		Command:     "fake-agent --stdio",
		Model:       "model-1",
		Permissions: string(aghconfig.PermissionModeApproveAll),
		Tools:       []string{"read", "write"},
		MCPServers: []aghconfig.MCPServer{{
			Name:    "filesystem",
			Command: "mcp-fs",
			Args:    []string{"--root", "/workspace"},
			Env: map[string]string{
				"TOKEN": "secret",
			},
		}},
		Prompt: "You are a builder.",
	})

	agent, err := aghconfig.LoadAgentDef("builder", homePaths)
	if err != nil {
		t.Fatalf("LoadAgentDef(builder) error = %v", err)
	}
	if got, want := agent.Command, "fake-agent --stdio"; got != want {
		t.Fatalf("agent.Command = %q, want %q", got, want)
	}
	if got, want := len(agent.Tools), 2; got != want {
		t.Fatalf("len(agent.Tools) = %d, want %d", got, want)
	}
	if got, want := len(agent.MCPServers), 1; got != want {
		t.Fatalf("len(agent.MCPServers) = %d, want %d", got, want)
	}
}

func TestSeedWorkspaceTargetPathRejectsEscapes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	if _, err := seedWorkspaceTargetPath(root, "../outside.txt"); err == nil ||
		!strings.Contains(err.Error(), "escapes root") {
		t.Fatalf("seedWorkspaceTargetPath(escape) error = %v, want escape validation", err)
	}
	if _, err := seedWorkspaceTargetPath(root, ""); err == nil ||
		!strings.Contains(err.Error(), "must reference a file") {
		t.Fatalf("seedWorkspaceTargetPath(blank) error = %v, want file validation", err)
	}
}

func TestWriteAgentDefEscapesYAMLSensitiveValues(t *testing.T) {
	t.Parallel()

	homePaths := NewHomePaths(t)
	WriteAgentDef(t, homePaths, AgentSeed{
		Name:        "builder",
		Provider:    "fake:provider",
		Command:     "fake-agent --prompt \"review:all #now\"",
		Model:       "model:1",
		Permissions: string(aghconfig.PermissionModeApproveAll),
		Tools:       []string{"read:all", "write #notes"},
		MCPServers: []aghconfig.MCPServer{{
			Name:    "filesystem",
			Command: "mcp-fs --mode=read:write",
			Args:    []string{"--root", "/workspace/#demo", "--label=ops:review"},
			Env: map[string]string{
				"TOKEN":   "secret:value #1",
				"PROMPT":  "line one\nline two",
				"CHANNEL": "ops:review",
			},
		}},
		Prompt: "You are a builder.\nRespect review:all #notes.",
	})

	agent, err := aghconfig.LoadAgentDef("builder", homePaths)
	if err != nil {
		t.Fatalf("LoadAgentDef(builder) error = %v", err)
	}
	if got, want := agent.Provider, "fake:provider"; got != want {
		t.Fatalf("agent.Provider = %q, want %q", got, want)
	}
	if got, want := agent.Command, "fake-agent --prompt \"review:all #now\""; got != want {
		t.Fatalf("agent.Command = %q, want %q", got, want)
	}
	if got, want := agent.Model, "model:1"; got != want {
		t.Fatalf("agent.Model = %q, want %q", got, want)
	}
	if got, want := agent.Tools, []string{"read:all", "write #notes"}; len(got) != len(want) ||
		got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("agent.Tools = %#v, want %#v", got, want)
	}
	if got, want := len(agent.MCPServers), 1; got != want {
		t.Fatalf("len(agent.MCPServers) = %d, want %d", got, want)
	}
	if got, want := agent.MCPServers[0].Command, "mcp-fs --mode=read:write"; got != want {
		t.Fatalf("agent.MCPServers[0].Command = %q, want %q", got, want)
	}
	if got, want := agent.MCPServers[0].Args[1], "/workspace/#demo"; got != want {
		t.Fatalf("agent.MCPServers[0].Args[1] = %q, want %q", got, want)
	}
	if got, want := agent.MCPServers[0].Env["TOKEN"], "secret:value #1"; got != want {
		t.Fatalf("agent.MCPServers[0].Env[TOKEN] = %q, want %q", got, want)
	}
	if got, want := agent.MCPServers[0].Env["PROMPT"], "line one\nline two"; got != want {
		t.Fatalf("agent.MCPServers[0].Env[PROMPT] = %q, want %q", got, want)
	}
	if got, want := agent.Prompt, "You are a builder.\nRespect review:all #notes."; got != want {
		t.Fatalf("agent.Prompt = %q, want %q", got, want)
	}
}

func TestShortSocketPathUsesTempDirAndAllowsEarlyRemoval(t *testing.T) {
	t.Parallel()

	path := shortSocketPath(t)
	if got, want := filepath.Clean(filepath.Dir(path)), filepath.Clean(os.TempDir()); got != want {
		t.Fatalf("filepath.Dir(shortSocketPath()) = %q, want %q", got, want)
	}
	if err := os.WriteFile(path, []byte("socket"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("os.Remove(%q) error = %v", path, err)
	}
}
