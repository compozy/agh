package session

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/diagnostics"
	"github.com/pedronauck/agh/internal/testutil"
	"github.com/pedronauck/agh/internal/vault"
)

type fakeProviderSecretResolver struct {
	values map[string]string
	errs   map[string]error
}

func (r fakeProviderSecretResolver) ResolveRef(ctx context.Context, ref string) (string, error) {
	if ctx == nil {
		return "", errors.New("test resolver context is required")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err, ok := r.errs[ref]; ok {
		return "", err
	}
	if value, ok := r.values[ref]; ok {
		return value, nil
	}
	return "", vault.ErrSecretNotFound
}

func TestPrepareProviderForStartInjectsSecretsAndMaterializesPiRuntime(t *testing.T) {
	t.Parallel()

	t.Run("Should inject resolved provider secrets and redact dynamic values", func(t *testing.T) {
		t.Parallel()

		secret := "sk-provider-runtime-secret-123456"
		manager := &Manager{
			providerSecrets: fakeProviderSecretResolver{
				values: map[string]string{
					"vault:providers/openrouter/api-key":   secret,
					"vault:providers/openrouter/secondary": "secondary-provider-runtime-secret",
				},
			},
		}
		session := &Session{sessionDir: t.TempDir()}
		t.Cleanup(session.clearProviderSecretRedactions)

		resolved := aghconfig.ResolvedAgent{
			Provider:        "openrouter",
			Model:           "openai/gpt-5.4",
			Harness:         aghconfig.ProviderHarnessPiACP,
			RuntimeProvider: "openrouter",
			BaseURL:         "https://openrouter.ai/api/v1",
			Transport:       "openai",
			CredentialSlots: []aghconfig.ProviderCredentialSlot{
				{
					Name:      "secondary",
					TargetEnv: "OPENROUTER_SECONDARY_TOKEN",
					SecretRef: "vault:providers/openrouter/secondary",
					Kind:      "oauth",
					Required:  false,
				},
				{
					Name:      "api_key",
					TargetEnv: "OPENROUTER_API_KEY",
					SecretRef: "vault:providers/openrouter/api-key",
					Kind:      "api_key",
					Required:  true,
				},
			},
		}

		opts, err := manager.prepareProviderForStart(testutil.Context(t), session, resolved, acp.StartOpts{
			Env: []string{"OPENROUTER_API_KEY=old", "KEEP=1"},
		})
		if err != nil {
			t.Fatalf("prepareProviderForStart() error = %v", err)
		}

		if got := envValue(opts.Env, "OPENROUTER_API_KEY"); got != secret {
			t.Fatalf("OPENROUTER_API_KEY = %q, want injected secret", got)
		}
		runtimeDir := envValue(opts.Env, "PI_CODING_AGENT_DIR")
		if runtimeDir == "" {
			t.Fatal("PI_CODING_AGENT_DIR = empty, want materialized pi runtime directory")
		}
		settings := readProviderJSON[piSettingsFile](t, filepath.Join(runtimeDir, "settings.json"))
		if settings.DefaultProvider != "openrouter" || settings.DefaultModel != "openai/gpt-5.4" {
			t.Fatalf("settings.json = %#v, want openrouter defaults", settings)
		}
		assertProviderRuntimeFileMode(t, filepath.Join(runtimeDir, "settings.json"), 0o600)
		models := readProviderJSON[piModelsFile](t, filepath.Join(runtimeDir, "models.json"))
		provider := models.Providers["openrouter"]
		if provider.APIKey != "OPENROUTER_API_KEY" {
			t.Fatalf("models.json apiKey = %q, want injected env name", provider.APIKey)
		}
		if provider.BaseURL != "https://openrouter.ai/api/v1" || provider.API != "openai" {
			t.Fatalf("models.json provider = %#v, want base URL and API transport", provider)
		}
		assertProviderRuntimeFileMode(t, filepath.Join(runtimeDir, "models.json"), 0o600)
		assertPiRuntimeEnvContract(t, opts, "openrouter", secret)
		payload, err := os.ReadFile(filepath.Join(runtimeDir, "models.json"))
		if err != nil {
			t.Fatalf("ReadFile(models.json) error = %v", err)
		}
		if strings.Contains(string(payload), secret) {
			t.Fatalf("models.json leaked secret payload: %s", payload)
		}
		redacted := diagnostics.RedactAndBound("stderr leaked "+secret, 256)
		if strings.Contains(redacted, secret) {
			t.Fatalf("RedactAndBound(dynamic provider secret) = %q leaked secret", redacted)
		}
	})

	t.Run("Should omit pi apiKey when optional credential is missing", func(t *testing.T) {
		t.Parallel()

		manager := &Manager{
			providerSecrets: fakeProviderSecretResolver{
				errs: map[string]error{
					"vault:providers/openrouter/api-key": vault.ErrMissingSecret,
				},
			},
		}
		session := &Session{sessionDir: t.TempDir()}
		resolved := aghconfig.ResolvedAgent{
			Provider:        "openrouter",
			Model:           "openai/gpt-5.4",
			Harness:         aghconfig.ProviderHarnessPiACP,
			RuntimeProvider: "openrouter",
			CredentialSlots: []aghconfig.ProviderCredentialSlot{
				{
					Name:      "api_key",
					TargetEnv: "OPENROUTER_API_KEY",
					SecretRef: "vault:providers/openrouter/api-key",
					Kind:      "api_key",
					Required:  false,
				},
			},
		}

		opts, err := manager.prepareProviderForStart(testutil.Context(t), session, resolved, acp.StartOpts{
			Env: []string{"OPENROUTER_API_KEY=parent-shell-secret"},
		})
		if err != nil {
			t.Fatalf("prepareProviderForStart(optional missing) error = %v", err)
		}
		if got := envValue(opts.Env, "OPENROUTER_API_KEY"); got != "" {
			t.Fatalf("OPENROUTER_API_KEY = %q, want scrubbed env for missing vault secret", got)
		}
		runtimeDir := envValue(opts.Env, "PI_CODING_AGENT_DIR")
		if runtimeDir == "" {
			t.Fatal("PI_CODING_AGENT_DIR = empty, want materialized pi runtime")
		}
		models := readProviderJSON[piModelsFile](t, filepath.Join(runtimeDir, "models.json"))
		if got := models.Providers["openrouter"].APIKey; got != "" {
			t.Fatalf("models.json apiKey = %q, want omitted apiKey for missing optional secret", got)
		}
		payload, err := os.ReadFile(filepath.Join(runtimeDir, "models.json"))
		if err != nil {
			t.Fatalf("ReadFile(models.json) error = %v", err)
		}
		if strings.Contains(string(payload), `"apiKey"`) {
			t.Fatalf("models.json contains apiKey for missing optional secret: %s", payload)
		}
	})

	t.Run("Should pass Pi runtime contract through session start options", func(t *testing.T) {
		t.Parallel()

		secret := "sk-provider-runtime-secret-from-create"
		h := newHarness(t, WithProviderSecretResolver(fakeProviderSecretResolver{
			values: map[string]string{
				"env:OPENROUTER_API_KEY": secret,
			},
		}))
		resolved, err := h.resolver.Resolve(testutil.Context(t), h.workspaceID)
		if err != nil {
			t.Fatalf("Resolve(workspace) error = %v", err)
		}
		resolved.Agents = []aghconfig.AgentDef{
			{
				Name:     "coder",
				Provider: "openrouter",
				Model:    "openai/gpt-5.4",
				Prompt:   "You are a coding assistant.",
			},
		}
		h.resolver.upsert(&resolved)
		h.driver.startHook = func(opts acp.StartOpts, _ int) (*fakeProcess, error) {
			if !strings.Contains(opts.Command, "pi-acp@latest") {
				t.Fatalf("StartOpts.Command = %q, want pi-acp command", opts.Command)
			}
			assertPiRuntimeEnvContract(t, opts, "openrouter", secret)
			return newFakeProcess(opts.AgentName, opts.Command, opts.Cwd, "acp-pi-runtime"), nil
		}

		session, err := h.manager.Create(testutil.Context(t), CreateOpts{
			AgentName: "coder",
			Name:      "pi-runtime-contract",
			Workspace: h.workspaceID,
		})
		if err != nil {
			t.Fatalf("Create(pi runtime contract) error = %v", err)
		}
		t.Cleanup(func() {
			if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
				t.Fatalf("Stop(pi runtime contract cleanup) error = %v", err)
			}
		})
	})
}

func TestShouldSkipMissingProviderSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resolved aghconfig.ResolvedAgent
		ref      string
		slot     aghconfig.ProviderCredentialSlot
		err      error
		want     bool
	}{
		{
			name:     "Should skip optional missing pi vault secret",
			resolved: aghconfig.ResolvedAgent{Harness: aghconfig.ProviderHarnessPiACP},
			ref:      "vault:providers/openrouter/api-key",
			slot:     aghconfig.ProviderCredentialSlot{Required: false},
			err:      vault.ErrMissingSecret,
			want:     true,
		},
		{
			name:     "Should require missing pi vault secret when slot is required",
			resolved: aghconfig.ResolvedAgent{Harness: aghconfig.ProviderHarnessPiACP},
			ref:      "vault:providers/openrouter/api-key",
			slot:     aghconfig.ProviderCredentialSlot{Required: true},
			err:      vault.ErrMissingSecret,
			want:     false,
		},
		{
			name:     "Should require missing env refs for direct acp providers when slot is required",
			resolved: aghconfig.ResolvedAgent{Harness: aghconfig.ProviderHarnessACP},
			ref:      "env:ANTHROPIC_API_KEY",
			slot:     aghconfig.ProviderCredentialSlot{Required: true},
			err:      vault.ErrMissingSecret,
			want:     false,
		},
		{
			name:     "Should skip optional missing env refs for direct acp providers",
			resolved: aghconfig.ResolvedAgent{Harness: aghconfig.ProviderHarnessACP},
			ref:      "env:ANTHROPIC_API_KEY",
			slot:     aghconfig.ProviderCredentialSlot{Required: false},
			err:      vault.ErrMissingSecret,
			want:     true,
		},
		{
			name:     "Should skip optional missing secret refs for direct acp providers",
			resolved: aghconfig.ResolvedAgent{Harness: aghconfig.ProviderHarnessACP},
			ref:      "vault:providers/custom/api-key",
			slot:     aghconfig.ProviderCredentialSlot{Required: false},
			err:      vault.ErrSecretNotFound,
			want:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := shouldSkipMissingProviderSecret(tc.resolved, tc.ref, tc.slot, tc.err); got != tc.want {
				t.Fatalf("shouldSkipMissingProviderSecret() = %t, want %t", got, tc.want)
			}
		})
	}
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if value, ok := strings.CutPrefix(entry, prefix); ok {
			return value
		}
	}
	return ""
}

func assertProviderRuntimeFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode(%q) = %#o, want %#o", path, got, want)
	}
}

func assertPiRuntimeEnvContract(t *testing.T, opts acp.StartOpts, providerName string, wantSecret string) {
	t.Helper()

	runtimeDir := envValue(opts.Env, "PI_CODING_AGENT_DIR")
	if runtimeDir == "" {
		t.Fatal("PI_CODING_AGENT_DIR = empty, want materialized pi runtime directory")
	}
	models := readProviderJSON[piModelsFile](t, filepath.Join(runtimeDir, "models.json"))
	provider, ok := models.Providers[providerName]
	if !ok {
		t.Fatalf("models.json providers = %#v, want provider %q", models.Providers, providerName)
	}
	if provider.APIKey == "" {
		t.Fatalf("models.json providers[%q].apiKey = empty, want env var reference", providerName)
	}
	if got := envValue(opts.Env, provider.APIKey); got != wantSecret {
		t.Fatalf("child env %s = %q, want resolved provider secret", provider.APIKey, got)
	}
}

func readProviderJSON[T any](t *testing.T, path string) T {
	t.Helper()

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	var value T
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", path, err)
	}
	return value
}
