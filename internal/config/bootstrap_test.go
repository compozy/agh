package config

import (
	"os"
	"path/filepath"
	"slices"
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
	auth_mode = "bound_secret"
	# keep credential slot comment
	[[providers.claude.credential_slots]]
	name = "api_key"
	target_env = "ANTHROPIC_KEY"
	secret_ref = "env:ANTHROPIC_KEY"
	kind = "api_key"
	required = true
	`)

	cfg, err := SaveBootstrapConfig(homePaths, "claude", "claude-sonnet-4-6")
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
	if cfg.Memory.Dream.Agent != DefaultMemoryDreamAgentName {
		t.Fatalf(
			"SaveBootstrapConfig() Memory.Dream.Agent = %q, want %q",
			cfg.Memory.Dream.Agent,
			DefaultMemoryDreamAgentName,
		)
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
	slots := reloaded.Providers["claude"].EffectiveCredentialSlots()
	if len(slots) != 1 || slots[0].TargetEnv != "ANTHROPIC_KEY" || slots[0].SecretRef != "env:ANTHROPIC_KEY" {
		t.Fatalf(
			"LoadGlobalConfig() Providers[claude].CredentialSlots = %#v, want ANTHROPIC_KEY slot",
			slots,
		)
	}
	if reloaded.Providers["claude"].Models.Default != "claude-sonnet-4-6" {
		t.Fatalf(
			"LoadGlobalConfig() Providers[claude].Models.Default = %q, want %q",
			reloaded.Providers["claude"].Models.Default,
			"claude-sonnet-4-6",
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
		`# keep credential slot comment`,
		`agent = "general"`,
		`provider = "claude"`,
		`mode = "approve-all"`,
		`default = "claude-sonnet-4-6"`,
		`port = 3030`,
		`secret_ref = "env:ANTHROPIC_KEY"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("config contents missing %q\n%s", want, text)
		}
	}
}

func TestSaveBootstrapConfigAllowsProviderManagedModel(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	cfg, err := SaveBootstrapConfig(homePaths, "blackbox", "")
	if err != nil {
		t.Fatalf("SaveBootstrapConfig() error = %v", err)
	}
	if cfg.Defaults.Provider != "blackbox" {
		t.Fatalf("SaveBootstrapConfig() Defaults.Provider = %q, want blackbox", cfg.Defaults.Provider)
	}
	if got := cfg.Providers["blackbox"].Models.Default; got != "" {
		t.Fatalf("SaveBootstrapConfig() Providers[blackbox].Models.Default = %q, want empty", got)
	}

	contents, err := os.ReadFile(homePaths.ConfigFile)
	if err != nil {
		t.Fatalf("ReadFile(config) error = %v", err)
	}
	text := string(contents)
	if strings.Contains(text, `default_model =`) {
		t.Fatalf("config contents wrote provider-managed default model:\n%s", text)
	}
}

func TestSaveBootstrapConfigMigratesPriorBootstrapDreamAgent(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	writeFile(t, homePaths.ConfigFile, `
[memory.dream]
agent = "general"
`)

	cfg, err := SaveBootstrapConfig(homePaths, "claude", "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("SaveBootstrapConfig() error = %v", err)
	}
	if got := cfg.Memory.Dream.Agent; got != DefaultMemoryDreamAgentName {
		t.Fatalf("SaveBootstrapConfig() Memory.Dream.Agent = %q, want %q", got, DefaultMemoryDreamAgentName)
	}

	reloaded, err := LoadGlobalConfig(homePaths)
	if err != nil {
		t.Fatalf("LoadGlobalConfig() error = %v", err)
	}
	if got := reloaded.Memory.Dream.Agent; got != DefaultMemoryDreamAgentName {
		t.Fatalf("LoadGlobalConfig() Memory.Dream.Agent = %q, want %q", got, DefaultMemoryDreamAgentName)
	}
}

func TestSaveBootstrapConfigRequiresModelForPiProviders(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	_, err = SaveBootstrapConfig(homePaths, "openrouter", "")
	if err == nil {
		t.Fatal("SaveBootstrapConfig() error = nil, want model required error")
	}
	wantErr := `bootstrap model is required for provider "openrouter"`
	if err.Error() != wantErr {
		t.Fatalf("SaveBootstrapConfig() error = %q, want %q", err.Error(), wantErr)
	}
}

func TestSaveBootstrapConfigNetworkBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		seed               string
		wantEnabled        bool
		wantDefaultChannel string
		wantNetworkSection bool
	}{
		{
			name:               "ShouldKeepNetworkEnabledByDefaultOnFirstRun",
			wantDefaultChannel: "default",
			wantEnabled:        true,
			wantNetworkSection: false,
		},
		{
			name: "ShouldPreserveExplicitNetworkDisable",
			seed: `
[network]
enabled = false
default_channel = "legacy"
`,
			wantEnabled:        false,
			wantDefaultChannel: "legacy",
			wantNetworkSection: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
			if err != nil {
				t.Fatalf("ResolveHomePathsFrom() error = %v", err)
			}

			if strings.TrimSpace(tt.seed) != "" {
				writeFile(t, homePaths.ConfigFile, tt.seed)
			}

			cfg, err := SaveBootstrapConfig(homePaths, "claude", "claude-sonnet-4-6")
			if err != nil {
				t.Fatalf("SaveBootstrapConfig() error = %v", err)
			}
			if got := cfg.Network.Enabled; got != tt.wantEnabled {
				t.Fatalf("SaveBootstrapConfig() Network.Enabled = %t, want %t", got, tt.wantEnabled)
			}
			if got := cfg.Network.DefaultChannel; got != tt.wantDefaultChannel {
				t.Fatalf("SaveBootstrapConfig() Network.DefaultChannel = %q, want %q", got, tt.wantDefaultChannel)
			}

			reloaded, err := LoadGlobalConfig(homePaths)
			if err != nil {
				t.Fatalf("LoadGlobalConfig() error = %v", err)
			}
			if got := reloaded.Network.Enabled; got != tt.wantEnabled {
				t.Fatalf("LoadGlobalConfig() Network.Enabled = %t, want %t", got, tt.wantEnabled)
			}
			if got := reloaded.Network.DefaultChannel; got != tt.wantDefaultChannel {
				t.Fatalf("LoadGlobalConfig() Network.DefaultChannel = %q, want %q", got, tt.wantDefaultChannel)
			}

			contents, err := os.ReadFile(homePaths.ConfigFile)
			if err != nil {
				t.Fatalf("ReadFile(config) error = %v", err)
			}
			text := string(contents)
			if !tt.wantNetworkSection {
				if strings.Contains(text, "[network]") {
					t.Fatalf("bootstrap config wrote an unexpected network section:\n%s", text)
				}
				return
			}

			for _, want := range []string{
				"[network]",
				`enabled = false`,
				`default_channel = "legacy"`,
			} {
				if !strings.Contains(text, want) {
					t.Fatalf("config contents missing %q\n%s", want, text)
				}
			}
		})
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

func TestEnsureOnboardingAgentCreatesValidProvisioningAgent(t *testing.T) {
	t.Parallel()

	t.Run("Should create a valid provisioning agent", func(t *testing.T) {
		t.Parallel()

		homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}

		path, created, err := EnsureOnboardingAgent(homePaths)
		if err != nil {
			t.Fatalf("EnsureOnboardingAgent() error = %v", err)
		}
		if !created {
			t.Fatal("EnsureOnboardingAgent() created = false, want true")
		}

		agent, err := LoadAgentDef(OnboardingAgentName, homePaths)
		if err != nil {
			t.Fatalf("LoadAgentDef(onboarding) error = %v", err)
		}
		if agent.Name != OnboardingAgentName {
			t.Fatalf("agent name = %q, want %q", agent.Name, OnboardingAgentName)
		}
		wantTools := []string{
			"agh__workspace_list",
			"agh__workspace_describe",
			"agh__network_channels",
			"agh__network_channel_create",
			"agh__agent_create",
		}
		if !slices.Equal(agent.Tools, wantTools) {
			t.Fatalf("onboarding agent tools = %#v, want %#v", agent.Tools, wantTools)
		}
		if len(agent.Toolsets) != 0 {
			t.Fatalf("onboarding agent toolsets = %#v, want none", agent.Toolsets)
		}
		if agent.Permissions != string(PermissionModeApproveAll) {
			t.Fatalf("onboarding agent permissions = %q, want %q", agent.Permissions, PermissionModeApproveAll)
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(onboarding agent) error = %v", err)
		}
		text := string(contents)
		for _, denied := range []string{"toolsets:", "- agh__coordination", "- agh__workspace\n"} {
			if strings.Contains(text, denied) {
				t.Fatalf("onboarding agent contents included broad grant %q:\n%s", denied, text)
			}
		}

		_, createdAgain, err := EnsureOnboardingAgent(homePaths)
		if err != nil {
			t.Fatalf("EnsureOnboardingAgent() second error = %v", err)
		}
		if createdAgain {
			t.Fatal("EnsureOnboardingAgent() second created = true, want false")
		}
	})
}

func TestValidatePublicAgentNameRejectsInternalManagedAgent(t *testing.T) {
	t.Parallel()

	t.Run("Should reject the reserved onboarding agent name", func(t *testing.T) {
		t.Parallel()

		err := ValidatePublicAgentName(OnboardingAgentName)
		if err == nil {
			t.Fatal("ValidatePublicAgentName(onboarding) error = nil, want reserved-name error")
		}
		if !strings.Contains(err.Error(), "reserved for internal AGH use") {
			t.Fatalf("ValidatePublicAgentName(onboarding) error = %v, want reserved-name message", err)
		}
	})

	t.Run("Should reject reserved internal agent names case insensitively", func(t *testing.T) {
		t.Parallel()

		for _, name := range []string{"Onboarding", "ONBOARDING", " onboarding "} {
			t.Run("Should reject "+name, func(t *testing.T) {
				t.Parallel()

				err := ValidatePublicAgentName(name)
				if err == nil {
					t.Fatalf("ValidatePublicAgentName(%q) error = nil, want reserved-name error", name)
				}
			})
		}
	})

	t.Run("Should allow normal public agent names", func(t *testing.T) {
		t.Parallel()

		if err := ValidatePublicAgentName(DefaultAgentName); err != nil {
			t.Fatalf("ValidatePublicAgentName(general) error = %v", err)
		}
	})
}
