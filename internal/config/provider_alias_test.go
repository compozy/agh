package config

import "testing"

func TestProviderAliasResolution(t *testing.T) {
	t.Parallel()

	t.Run("Should resolve explicit provider aliases to canonical provider ids", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			want string
		}{
			{name: providerClaudeCodeAlias, want: providerClaudeKey},
			{name: "CLAUDE", want: "claude"},
			{name: "AI-Gateway", want: "vercel-ai-gateway"},
			{name: "aigateway", want: "vercel-ai-gateway"},
			{name: "open-code", want: "opencode"},
			{name: providerXaiDotAlias, want: providerXaiKey},
		}

		for _, tc := range tests {
			t.Run("Should resolve "+tc.name, func(t *testing.T) {
				t.Parallel()

				if got := CanonicalProviderName(tc.name); got != tc.want {
					t.Fatalf("CanonicalProviderName(%q) = %q, want %q", tc.name, got, tc.want)
				}
			})
		}
	})

	t.Run("Should resolve explicit model aliases inside the selected provider", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			provider string
			model    string
			want     string
		}{
			{provider: providerClaudeKey, model: modelClaudeSonnetAlias, want: providerClaudeSonnet46Value},
			{provider: providerClaudeCodeAlias, model: modelClaudeOpusAlias, want: modelClaudeOpus47ID},
			{provider: providerCodexKey, model: modelGPT5CompactAlias, want: modelGPT54ID},
			{provider: providerCodexKey, model: modelMiniAlias, want: modelGPT54MiniID},
			{provider: "vercel", model: modelClaudeOpusAlias, want: providerAnthropicClaudeOpus47Path},
			{provider: providerKimiAlias, model: providerKimiAlias, want: providerKimiK2ThinkingValue},
			{provider: providerXaiDotAlias, model: providerGrokAlias, want: providerGrok4FastNonReasoningValue},
			{provider: "unknown", model: "custom-model", want: "custom-model"},
		}

		for _, tc := range tests {
			t.Run("Should resolve "+tc.provider+" "+tc.model, func(t *testing.T) {
				t.Parallel()

				if got := CanonicalProviderModelName(tc.provider, tc.model); got != tc.want {
					t.Fatalf(
						"CanonicalProviderModelName(%q, %q) = %q, want %q",
						tc.provider,
						tc.model,
						got,
						tc.want,
					)
				}
			})
		}
	})

	t.Run("Should expose canonical provider and model values after resolving an agent", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultWithHome(HomePaths{})
		cfg.Defaults.Provider = "ai-gateway"
		agent := AgentDef{Name: "coder", Model: "opus", Prompt: "Help with the release."}

		resolved, err := cfg.ResolveAgent(agent)
		if err != nil {
			t.Fatalf("ResolveAgent() error = %v", err)
		}
		if got, want := resolved.Provider, providerVercelAIGatewayValue; got != want {
			t.Fatalf("ResolveAgent() Provider = %q, want %q", got, want)
		}
		if got, want := resolved.Model, providerAnthropicClaudeOpus47Path; got != want {
			t.Fatalf("ResolveAgent() Model = %q, want %q", got, want)
		}
	})

	t.Run("Should canonicalize configured default model aliases when resolving providers", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultWithHome(HomePaths{})
		cfg.Providers["claude"] = ProviderConfig{
			Models: ProviderModelsConfig{Default: "sonnet"},
		}

		resolved, err := cfg.ResolveProvider("claude-code")
		if err != nil {
			t.Fatalf("ResolveProvider() error = %v", err)
		}
		if got, want := resolved.Models.Default, providerClaudeSonnet46Value; got != want {
			t.Fatalf("ResolveProvider() Models.Default = %q, want %q", got, want)
		}
	})

	t.Run("Should preserve explicit curated ids before applying model aliases", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultWithHome(HomePaths{})
		cfg.Providers[providerCodexKey] = ProviderConfig{
			Models: ProviderModelsConfig{
				Default: modelGPT5Alias,
				Curated: []ProviderModelConfig{
					{ID: modelGPT5Alias, DisplayName: "GPT-5"},
				},
			},
		}

		resolved, err := cfg.ResolveProvider(providerCodexKey)
		if err != nil {
			t.Fatalf("ResolveProvider() error = %v", err)
		}
		if got, want := resolved.Models.Default, modelGPT5Alias; got != want {
			t.Fatalf("ResolveProvider() Models.Default = %q, want %q", got, want)
		}
	})
}
