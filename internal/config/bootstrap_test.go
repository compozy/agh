package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveAgentNameFallsBackToDefaults(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Defaults: DefaultsConfig{Agent: DefaultAgentName},
	}

	resolved, err := ResolveAgentName("", cfg.Defaults)
	if err != nil {
		t.Fatalf("ResolveAgentName() error = %v", err)
	}
	if resolved != DefaultAgentName {
		t.Fatalf("ResolveAgentName() = %q, want %q", resolved, DefaultAgentName)
	}
}

func TestSaveBootstrapConfigWritesManagedDefaults(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	writeFile(t, homePaths.ConfigFile, `
# bootstrap comment
[http]
port = 3030

[providers.claude]
# keep api key comment
api_key_env = "ANTHROPIC_KEY"
`)

	cfg, err := SaveBootstrapConfig(homePaths, "claude", "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("SaveBootstrapConfig() error = %v", err)
	}
	if cfg.Defaults.Agent != DefaultAgentName {
		t.Fatalf("SaveBootstrapConfig() Defaults.Agent = %q, want %q", cfg.Defaults.Agent, DefaultAgentName)
	}
	if cfg.Defaults.Provider != "claude" {
		t.Fatalf("SaveBootstrapConfig() Defaults.Provider = %q, want %q", cfg.Defaults.Provider, "claude")
	}
	if cfg.Permissions.Mode != PermissionModeApproveAll {
		t.Fatalf("SaveBootstrapConfig() Permissions.Mode = %q, want %q", cfg.Permissions.Mode, PermissionModeApproveAll)
	}
	if cfg.Memory.Dream.Agent != DefaultAgentName {
		t.Fatalf("SaveBootstrapConfig() Memory.Dream.Agent = %q, want %q", cfg.Memory.Dream.Agent, DefaultAgentName)
	}
	if !cfg.Network.Enabled {
		t.Fatal("SaveBootstrapConfig() Network.Enabled = false, want inherited enabled default")
	}

	reloaded, err := LoadGlobalConfig(homePaths)
	if err != nil {
		t.Fatalf("LoadGlobalConfig() error = %v", err)
	}
	if reloaded.HTTP.Port != 3030 {
		t.Fatalf("LoadGlobalConfig() HTTP.Port = %d, want 3030", reloaded.HTTP.Port)
	}
	if reloaded.Providers["claude"].APIKeyEnv != "ANTHROPIC_KEY" {
		t.Fatalf(
			"LoadGlobalConfig() Providers[claude].APIKeyEnv = %q, want %q",
			reloaded.Providers["claude"].APIKeyEnv,
			"ANTHROPIC_KEY",
		)
	}
	if reloaded.Providers["claude"].DefaultModel != "claude-sonnet-4-20250514" {
		t.Fatalf(
			"LoadGlobalConfig() Providers[claude].DefaultModel = %q, want %q",
			reloaded.Providers["claude"].DefaultModel,
			"claude-sonnet-4-20250514",
		)
	}
	if !reloaded.Network.Enabled {
		t.Fatal("LoadGlobalConfig() Network.Enabled = false, want inherited enabled default")
	}

	contents, err := os.ReadFile(homePaths.ConfigFile)
	if err != nil {
		t.Fatalf("ReadFile(config) error = %v", err)
	}
	text := string(contents)
	for _, want := range []string{
		`# bootstrap comment`,
		`# keep api key comment`,
		`agent = "general"`,
		`provider = "claude"`,
		`mode = "approve-all"`,
		`default_model = "claude-sonnet-4-20250514"`,
		`port = 3030`,
		`api_key_env = "ANTHROPIC_KEY"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("config contents missing %q\n%s", want, text)
		}
	}
}

func TestSaveBootstrapConfigFirstRunKeepsNetworkEnabledByDefault(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg, err := SaveBootstrapConfig(homePaths, "claude", "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("SaveBootstrapConfig() error = %v", err)
	}
	if !cfg.Network.Enabled {
		t.Fatal("SaveBootstrapConfig() Network.Enabled = false, want true on first run")
	}

	reloaded, err := LoadGlobalConfig(homePaths)
	if err != nil {
		t.Fatalf("LoadGlobalConfig() error = %v", err)
	}
	if !reloaded.Network.Enabled {
		t.Fatal("LoadGlobalConfig() Network.Enabled = false, want true on first run")
	}

	contents, err := os.ReadFile(homePaths.ConfigFile)
	if err != nil {
		t.Fatalf("ReadFile(config) error = %v", err)
	}
	if strings.Contains(string(contents), "[network]") {
		t.Fatalf("bootstrap config wrote an unexpected network section:\n%s", string(contents))
	}
}

func TestSaveBootstrapConfigPreservesExplicitNetworkDisable(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	writeFile(t, homePaths.ConfigFile, `
[network]
enabled = false
default_channel = "legacy"
`)

	cfg, err := SaveBootstrapConfig(homePaths, "claude", "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("SaveBootstrapConfig() error = %v", err)
	}
	if cfg.Network.Enabled {
		t.Fatal("SaveBootstrapConfig() Network.Enabled = true, want preserved explicit false")
	}
	if got, want := cfg.Network.DefaultChannel, "legacy"; got != want {
		t.Fatalf("SaveBootstrapConfig() Network.DefaultChannel = %q, want %q", got, want)
	}

	reloaded, err := LoadGlobalConfig(homePaths)
	if err != nil {
		t.Fatalf("LoadGlobalConfig() error = %v", err)
	}
	if reloaded.Network.Enabled {
		t.Fatal("LoadGlobalConfig() Network.Enabled = true, want preserved explicit false")
	}
	if got, want := reloaded.Network.DefaultChannel, "legacy"; got != want {
		t.Fatalf("LoadGlobalConfig() Network.DefaultChannel = %q, want %q", got, want)
	}

	contents, err := os.ReadFile(homePaths.ConfigFile)
	if err != nil {
		t.Fatalf("ReadFile(config) error = %v", err)
	}
	for _, want := range []string{
		"[network]",
		`enabled = false`,
		`default_channel = "legacy"`,
	} {
		if !strings.Contains(string(contents), want) {
			t.Fatalf("config contents missing %q\n%s", want, string(contents))
		}
	}
}

func TestEnsureBootstrapAgentCreatesAndPreservesManagedAgent(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	path, created, err := EnsureBootstrapAgent(homePaths)
	if err != nil {
		t.Fatalf("EnsureBootstrapAgent() error = %v", err)
	}
	if !created {
		t.Fatal("EnsureBootstrapAgent() created = false, want true")
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(agent) error = %v", err)
	}
	if !strings.Contains(string(contents), "name: "+DefaultAgentName) {
		t.Fatalf("agent contents = %q, want default agent name", string(contents))
	}
	if strings.Contains(string(contents), "provider:") {
		t.Fatalf("agent contents = %q, want provider omitted", string(contents))
	}

	if err := os.WriteFile(path, []byte("custom"), 0o644); err != nil {
		t.Fatalf("WriteFile(agent) error = %v", err)
	}
	againPath, createdAgain, err := EnsureBootstrapAgent(homePaths)
	if err != nil {
		t.Fatalf("EnsureBootstrapAgent() second error = %v", err)
	}
	if createdAgain {
		t.Fatal("EnsureBootstrapAgent() second created = true, want false")
	}
	if againPath != path {
		t.Fatalf("EnsureBootstrapAgent() path = %q, want %q", againPath, path)
	}

	preserved, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(agent second) error = %v", err)
	}
	if string(preserved) != "custom" {
		t.Fatalf("agent contents after second ensure = %q, want custom", string(preserved))
	}
}
