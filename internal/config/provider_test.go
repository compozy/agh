package config

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
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
	if got, want := len(merged[0].Args), 1; got != want {
		t.Fatalf("MergeMCPServers() Args len = %d, want %d (%#v)", got, want, merged[0].Args)
	}
	if got, want := merged[0].Args[0], "-y"; got != want {
		t.Fatalf("MergeMCPServers() Args = %#v", merged[0].Args)
	}
	if merged[0].Env["TOKEN"] != "base" || merged[0].Env["OTHER"] != "1" {
		t.Fatalf("MergeMCPServers() Env = %#v", merged[0].Env)
	}
}

func TestMCPServerValidateSupportsRemoteOAuthPKCE(t *testing.T) {
	t.Parallel()

	server := MCPServer{
		Name:      "linear",
		Transport: MCPServerTransportSSE,
		URL:       "https://mcp.example/sse",
		Auth: MCPAuthConfig{
			Type:             MCPAuthTypeOAuth2PKCE,
			AuthorizationURL: "https://auth.example/authorize",
			TokenURL:         "https://auth.example/token",
			ClientID:         "client-1",
			Scopes:           []string{"read", "write"},
		},
	}
	if err := server.Validate("mcp_servers[0]"); err != nil {
		t.Fatalf("Validate(remote OAuth) error = %v", err)
	}

	server.Auth.TokenURL = ""
	if err := server.Validate("mcp_servers[0]"); err == nil {
		t.Fatal("Validate(missing token metadata) error = nil, want validation failure")
	}
}

func TestRedactedMCPServerDoesNotExposeEnvSecretValues(t *testing.T) {
	t.Parallel()

	server := MCPServer{
		Name:    "github",
		Command: "npx",
		Env: map[string]string{
			"GITHUB_TOKEN": "secret-token",
		},
	}
	redacted := RedactedMCPServer(server)
	if got := redacted.Env["GITHUB_TOKEN"]; got != RedactedValue() {
		t.Fatalf("redacted env = %q, want placeholder", got)
	}
	if server.Env["GITHUB_TOKEN"] != "secret-token" {
		t.Fatalf("source env mutated = %#v", server.Env)
	}
}

func TestMergeMCPServersTrimmedNamesCollide(t *testing.T) {
	t.Parallel()

	merged := MergeMCPServers(
		[]MCPServer{{Name: "  github  ", Command: "npx"}},
		[]MCPServer{{Name: "github", Args: []string{"-y"}}},
	)

	if len(merged) != 1 {
		t.Fatalf("MergeMCPServers() len = %d, want 1", len(merged))
	}
	if got, want := merged[0].Command, "npx"; got != want {
		t.Fatalf("MergeMCPServers() Command = %q, want %q", got, want)
	}
	if got, want := len(merged[0].Args), 1; got != want {
		t.Fatalf("MergeMCPServers() Args len = %d, want %d (%#v)", got, want, merged[0].Args)
	}
	if got, want := merged[0].Args[0], "-y"; got != want {
		t.Fatalf("MergeMCPServers() Args[0] = %q, want %q", got, want)
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

func TestOverrideMCPServersTrimmedNamesCollide(t *testing.T) {
	t.Parallel()

	merged := OverrideMCPServers(
		[]MCPServer{{Name: "  github  ", Command: "npx", Args: []string{"-y"}}},
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

	cfg := Config{}
	if _, err := cfg.ResolveProvider("unknown"); err == nil {
		t.Fatal("ResolveProvider() error = nil, want non-nil")
	} else if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("ResolveProvider() error = %v, want ErrProviderUnavailable", err)
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
	if len(resolved.Tools) != 0 {
		t.Fatalf("ResolveAgent() Tools = %#v, want empty default", resolved.Tools)
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

func TestResolveSessionAgent(t *testing.T) {
	t.Parallel()

	t.Run("Should match ResolveAgent when provider override is empty", func(t *testing.T) {
		t.Parallel()

		homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}

		cfg := DefaultWithHome(homePaths)
		cfg.MCPServers = []MCPServer{
			{Name: "global", Command: "global-command"},
		}

		agent := AgentDef{
			Name:        "coder",
			Provider:    "claude",
			Command:     "agent-command",
			Model:       "agent-model",
			Permissions: string(PermissionModeApproveReads),
			Prompt:      "prompt",
			Tools:       []string{"agh__skill_view"},
			Toolsets:    []string{"agh__catalog"},
			DenyTools:   []string{"agh__task_*"},
			MCPServers: []MCPServer{
				{Name: "agent", Command: "agent-command"},
			},
		}

		got, err := cfg.ResolveSessionAgent(agent, "")
		if err != nil {
			t.Fatalf("ResolveSessionAgent() error = %v", err)
		}

		want, err := cfg.ResolveAgent(agent)
		if err != nil {
			t.Fatalf("ResolveAgent() error = %v", err)
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ResolveSessionAgent() = %#v, want %#v", got, want)
		}
	})

	t.Run("Should preserve agent runtime fields when override matches agent provider", func(t *testing.T) {
		t.Parallel()

		homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}

		cfg := DefaultWithHome(homePaths)
		cfg.Providers["claude"] = ProviderConfig{
			Command:      "provider-claude-command",
			DefaultModel: "provider-claude-model",
		}

		agent := AgentDef{
			Name:     "coder",
			Provider: "claude",
			Command:  "agent-command",
			Model:    "agent-model",
			Prompt:   "prompt",
		}

		resolved, err := cfg.ResolveSessionAgent(agent, "claude")
		if err != nil {
			t.Fatalf("ResolveSessionAgent() error = %v", err)
		}
		if got, want := resolved.Command, "agent-command"; got != want {
			t.Fatalf("ResolveSessionAgent() Command = %q, want %q", got, want)
		}
		if got, want := resolved.Model, "agent-model"; got != want {
			t.Fatalf("ResolveSessionAgent() Model = %q, want %q", got, want)
		}
	})

	t.Run("Should preserve agent runtime fields when override matches default provider", func(t *testing.T) {
		t.Parallel()

		homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}

		cfg := DefaultWithHome(homePaths)
		cfg.Defaults.Provider = "claude"
		cfg.Providers["claude"] = ProviderConfig{
			Command:      "provider-claude-command",
			DefaultModel: "provider-claude-model",
		}

		agent := AgentDef{
			Name:    "coder",
			Command: "agent-command",
			Model:   "agent-model",
			Prompt:  "prompt",
		}

		resolved, err := cfg.ResolveSessionAgent(agent, "claude")
		if err != nil {
			t.Fatalf("ResolveSessionAgent() error = %v", err)
		}
		if got, want := resolved.Command, "agent-command"; got != want {
			t.Fatalf("ResolveSessionAgent() Command = %q, want %q", got, want)
		}
		if got, want := resolved.Model, "agent-model"; got != want {
			t.Fatalf("ResolveSessionAgent() Model = %q, want %q", got, want)
		}
	})

	t.Run("Should use workspace-merged runtime fields from the override provider", func(t *testing.T) {
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
			Command:      "workspace-claude-command",
			DefaultModel: "workspace-claude-model",
			MCPServers: []MCPServer{
				{Name: "provider-claude", Command: "provider-claude-command"},
			},
		}
		cfg.Providers["codex"] = ProviderConfig{
			Command:      "workspace-codex-command",
			DefaultModel: "workspace-codex-model",
			MCPServers: []MCPServer{
				{Name: "provider-codex", Command: "provider-codex-command"},
				{Name: "shared-provider", Command: "shared-provider-codex", Args: []string{"--codex"}},
			},
		}

		agent := AgentDef{
			Name:     "coder",
			Provider: "claude",
			Command:  "agent-command",
			Model:    "agent-model",
			Prompt:   "prompt",
			MCPServers: []MCPServer{
				{Name: "agent", Command: "agent-command"},
			},
		}

		resolved, err := cfg.ResolveSessionAgent(agent, "codex")
		if err != nil {
			t.Fatalf("ResolveSessionAgent() error = %v", err)
		}

		if got, want := resolved.Provider, "codex"; got != want {
			t.Fatalf("ResolveSessionAgent() Provider = %q, want %q", got, want)
		}
		if got, want := resolved.Command, "workspace-codex-command"; got != want {
			t.Fatalf("ResolveSessionAgent() Command = %q, want %q", got, want)
		}
		if got, want := resolved.Model, "workspace-codex-model"; got != want {
			t.Fatalf("ResolveSessionAgent() Model = %q, want %q", got, want)
		}
		if resolved.Command == agent.Command {
			t.Fatalf(
				"ResolveSessionAgent() Command = %q, want provider-owned command instead of agent override",
				resolved.Command,
			)
		}
		if resolved.Model == agent.Model {
			t.Fatalf(
				"ResolveSessionAgent() Model = %q, want provider-owned default instead of agent override",
				resolved.Model,
			)
		}
		if got, want := resolved.APIKeyEnv, "OPENAI_API_KEY"; got != want {
			t.Fatalf("ResolveSessionAgent() APIKeyEnv = %q, want %q", got, want)
		}

		if got, want := len(resolved.MCPServers), 4; got != want {
			t.Fatalf("ResolveSessionAgent() MCPServers len = %d, want %d (%#v)", got, want, resolved.MCPServers)
		}
		if got, want := resolved.MCPServers[0].Name, "global"; got != want {
			t.Fatalf("ResolveSessionAgent() MCPServers[0].Name = %q, want %q", got, want)
		}
		if got, want := mcpServerByName(
			t,
			resolved.MCPServers,
			"provider-codex",
		).Command, "provider-codex-command"; got != want {
			t.Fatalf("ResolveSessionAgent() provider-codex Command = %q, want %q", got, want)
		}
		if got, want := mcpServerByName(
			t,
			resolved.MCPServers,
			"shared-provider",
		).Command, "shared-provider-codex"; got != want {
			t.Fatalf("ResolveSessionAgent() shared-provider Command = %q, want %q", got, want)
		}
		if hasMCPServer(resolved.MCPServers, "provider-claude") {
			t.Fatalf(
				"ResolveSessionAgent() MCPServers = %#v, want provider-owned layer from selected provider only",
				resolved.MCPServers,
			)
		}
		if !hasMCPServer(resolved.MCPServers, "agent") {
			t.Fatalf("ResolveSessionAgent() MCPServers = %#v, want agent-local layer preserved", resolved.MCPServers)
		}
	})

	t.Run("Should reject an unknown override provider with the wrapped provider error", func(t *testing.T) {
		t.Parallel()

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

		_, err = cfg.ResolveSessionAgent(agent, "missing")
		if err == nil {
			t.Fatal("ResolveSessionAgent() error = nil, want unknown provider failure")
		}
		if !errors.Is(err, ErrProviderUnavailable) {
			t.Fatalf("ResolveSessionAgent() error = %v, want ErrProviderUnavailable", err)
		}
		if !strings.Contains(err.Error(), `resolve session agent with provider "missing"`) {
			t.Fatalf("ResolveSessionAgent() error = %q, want session override context", err.Error())
		}
		if !strings.Contains(err.Error(), `unknown provider "missing"`) {
			t.Fatalf("ResolveSessionAgent() error = %q, want unknown provider detail", err.Error())
		}
	})
}

func TestMCPServerValidateRejectsMissingFields(t *testing.T) {
	t.Parallel()

	if err := (MCPServer{}).Validate("mcp"); err == nil {
		t.Fatal("MCPServer.Validate() error = nil, want non-nil")
	}
}

func mcpServerByName(t *testing.T, servers []MCPServer, name string) MCPServer {
	t.Helper()

	for _, server := range servers {
		if server.Name == name {
			return server
		}
	}

	t.Fatalf("MCP server %q not found in %#v", name, servers)

	return MCPServer{}
}

func hasMCPServer(servers []MCPServer, name string) bool {
	for _, server := range servers {
		if server.Name == name {
			return true
		}
	}

	return false
}
