package config

import (
	"path/filepath"
	"testing"
)

func TestBuiltinProvidersContainExpectedCommands(t *testing.T) {
	t.Parallel()

	providers := BuiltinProviders()
	tests := map[string]string{
		"claude":   "npx -y @agentclientprotocol/claude-agent-acp@0.24.2",
		"codex":    "npx @zed-industries/codex-acp@0.10.0",
		"gemini":   "gemini --acp",
		"opencode": "npx -y opencode-ai acp",
		"copilot":  "copilot --acp --stdio",
		"cursor":   "cursor-agent acp",
		"kiro":     "kiro-cli-chat acp",
		"pi":       "npx pi-acp@0.0.22",
	}

	for name, want := range tests {
		got, ok := providers[name]
		if !ok {
			t.Fatalf("BuiltinProviders() missing %q", name)
		}
		if got.Command != want {
			t.Fatalf("BuiltinProviders()[%q].Command = %q, want %q", name, got.Command, want)
		}
	}
}

func TestProviderConfigOverrideMergesWithBuiltins(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Providers: map[string]ProviderConfig{
			"claude": {
				DefaultModel: "claude-opus-override",
			},
		},
	}

	provider, err := cfg.ResolveProvider("claude")
	if err != nil {
		t.Fatalf("ResolveProvider() error = %v", err)
	}
	if provider.Command == "" {
		t.Fatal("ResolveProvider() Command = empty, want builtin command")
	}
	if provider.DefaultModel != "claude-opus-override" {
		t.Fatalf("ResolveProvider() DefaultModel = %q, want %q", provider.DefaultModel, "claude-opus-override")
	}
	if provider.APIKeyEnv != "ANTHROPIC_API_KEY" {
		t.Fatalf("ResolveProvider() APIKeyEnv = %q, want %q", provider.APIKeyEnv, "ANTHROPIC_API_KEY")
	}
}

func TestResolveAgentModelOverridesProviderDefault(t *testing.T) {
	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg := DefaultWithHome(homePaths)
	agent := AgentDef{
		Name:     "coder",
		Provider: "claude",
		Model:    "agent-model",
		Prompt:   "prompt",
	}

	resolved, err := cfg.ResolveAgent(agent)
	if err != nil {
		t.Fatalf("ResolveAgent() error = %v", err)
	}
	if resolved.Model != "agent-model" {
		t.Fatalf("ResolveAgent() Model = %q, want %q", resolved.Model, "agent-model")
	}
}

func TestMCPServersMergeAgentAndProviderLevels(t *testing.T) {
	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg := DefaultWithHome(homePaths)
	cfg.Providers["claude"] = ProviderConfig{
		MCPServers: []MCPServer{
			{Name: "github", Command: "npx"},
		},
	}

	agent := AgentDef{
		Name:     "coder",
		Provider: "claude",
		Prompt:   "prompt",
		MCPServers: []MCPServer{
			{Name: "memory", Command: "memory-server"},
		},
	}

	resolved, err := cfg.ResolveAgent(agent)
	if err != nil {
		t.Fatalf("ResolveAgent() error = %v", err)
	}

	if len(resolved.MCPServers) != 2 {
		t.Fatalf("ResolveAgent() MCPServers = %#v, want 2 entries", resolved.MCPServers)
	}
	if resolved.MCPServers[0].Name != "github" || resolved.MCPServers[1].Name != "memory" {
		t.Fatalf("ResolveAgent() MCPServers = %#v", resolved.MCPServers)
	}
}

func TestMergeMCPServersSameNameOverlaysFields(t *testing.T) {
	t.Parallel()

	merged := MergeMCPServers(
		[]MCPServer{{Name: "github", Command: "npx", Env: map[string]string{"TOKEN": "base"}}},
		[]MCPServer{{Name: "github", Args: []string{"-y"}, Env: map[string]string{"OTHER": "1"}}},
	)

	if len(merged) != 1 {
		t.Fatalf("MergeMCPServers() len = %d, want 1", len(merged))
	}
	if merged[0].Command != "npx" {
		t.Fatalf("MergeMCPServers() Command = %q, want %q", merged[0].Command, "npx")
	}
	if len(merged[0].Args) != 1 || merged[0].Args[0] != "-y" {
		t.Fatalf("MergeMCPServers() Args = %#v", merged[0].Args)
	}
	if merged[0].Env["TOKEN"] != "base" || merged[0].Env["OTHER"] != "1" {
		t.Fatalf("MergeMCPServers() Env = %#v", merged[0].Env)
	}
}

func TestOverrideMCPServersSameNameReplacesObject(t *testing.T) {
	t.Parallel()

	merged := OverrideMCPServers(
		[]MCPServer{{Name: "github", Command: "npx", Args: []string{"-y"}, Env: map[string]string{"TOKEN": "base"}}},
		[]MCPServer{{Name: "github", Command: "node"}},
	)

	if len(merged) != 1 {
		t.Fatalf("OverrideMCPServers() len = %d, want 1", len(merged))
	}
	if got, want := merged[0].Command, "node"; got != want {
		t.Fatalf("OverrideMCPServers() Command = %q, want %q", got, want)
	}
	if got := len(merged[0].Args); got != 0 {
		t.Fatalf("OverrideMCPServers() Args = %#v, want replacement semantics", merged[0].Args)
	}
	if got := len(merged[0].Env); got != 0 {
		t.Fatalf("OverrideMCPServers() Env = %#v, want replacement semantics", merged[0].Env)
	}
}

func TestResolveAgentMergesTopLevelProviderAndAgentMCPServers(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg := DefaultWithHome(homePaths)
	cfg.MCPServers = []MCPServer{
		{Name: "global", Command: "global-command"},
	}
	cfg.Providers["claude"] = ProviderConfig{
		MCPServers: []MCPServer{
			{Name: "provider", Command: "provider-command"},
		},
	}

	agent := AgentDef{
		Name:     "coder",
		Provider: "claude",
		Prompt:   "prompt",
		MCPServers: []MCPServer{
			{Name: "agent", Command: "agent-command"},
		},
	}

	resolved, err := cfg.ResolveAgent(agent)
	if err != nil {
		t.Fatalf("ResolveAgent() error = %v", err)
	}

	if got, want := len(resolved.MCPServers), 3; got != want {
		t.Fatalf("ResolveAgent() MCPServers len = %d, want %d (%#v)", got, want, resolved.MCPServers)
	}
	if got, want := resolved.MCPServers[0].Name, "global"; got != want {
		t.Fatalf("ResolveAgent() MCPServers[0].Name = %q, want %q", got, want)
	}
	if got, want := resolved.MCPServers[1].Name, "provider"; got != want {
		t.Fatalf("ResolveAgent() MCPServers[1].Name = %q, want %q", got, want)
	}
	if got, want := resolved.MCPServers[2].Name, "agent"; got != want {
		t.Fatalf("ResolveAgent() MCPServers[2].Name = %q, want %q", got, want)
	}
}

func TestResolveProviderRejectsUnknownProvider(t *testing.T) {
	t.Parallel()

	if _, err := (Config{}).ResolveProvider("unknown"); err == nil {
		t.Fatal("ResolveProvider() error = nil, want non-nil")
	}
}

func TestResolveAgentDefaultsToolsAndPermissions(t *testing.T) {
	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg := DefaultWithHome(homePaths)
	agent := AgentDef{
		Name:     "coder",
		Provider: "claude",
		Prompt:   "prompt",
	}

	resolved, err := cfg.ResolveAgent(agent)
	if err != nil {
		t.Fatalf("ResolveAgent() error = %v", err)
	}
	if len(resolved.Tools) != 1 || resolved.Tools[0] != "*" {
		t.Fatalf("ResolveAgent() Tools = %#v", resolved.Tools)
	}
	if resolved.Permissions != string(PermissionModeApproveAll) {
		t.Fatalf("ResolveAgent() Permissions = %q, want %q", resolved.Permissions, PermissionModeApproveAll)
	}
}

func TestResolveAgentFallsBackToDefaultsProvider(t *testing.T) {
	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg := DefaultWithHome(homePaths)
	cfg.Defaults.Provider = "claude"
	agent := AgentDef{
		Name:   DefaultAgentName,
		Prompt: "prompt",
	}

	resolved, err := cfg.ResolveAgent(agent)
	if err != nil {
		t.Fatalf("ResolveAgent() error = %v", err)
	}
	if resolved.Provider != "claude" {
		t.Fatalf("ResolveAgent() Provider = %q, want %q", resolved.Provider, "claude")
	}
}

func TestMCPServerValidateRejectsMissingFields(t *testing.T) {
	t.Parallel()

	if err := (MCPServer{}).Validate("mcp"); err == nil {
		t.Fatal("MCPServer.Validate() error = nil, want non-nil")
	}
}
