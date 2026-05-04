package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/diagnostics"
	"github.com/pedronauck/agh/internal/providerenv"
	"github.com/pedronauck/agh/internal/vault"
)

type envProviderSecretResolver struct {
	lookupEnv func(string) (string, bool)
}

func (r envProviderSecretResolver) ResolveRef(ctx context.Context, ref string) (string, error) {
	if ctx == nil {
		return "", errors.New("session: provider secret context is required")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	normalized := vault.NormalizeRef(ref)
	if !vault.IsEnvRef(normalized) {
		return "", fmt.Errorf("%w: %s", vault.ErrUnsupportedSecretRef, normalized)
	}
	if r.lookupEnv == nil {
		return "", errors.New("session: provider env lookup is not configured")
	}
	envName := strings.TrimSpace(strings.TrimPrefix(normalized, "env:"))
	value, ok := r.lookupEnv(envName)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%w: env:%s", vault.ErrMissingSecret, envName)
	}
	return value, nil
}

func (m *Manager) prepareProviderForStart(
	ctx context.Context,
	session *Session,
	resolved aghconfig.ResolvedAgent,
	opts acp.StartOpts,
) (acp.StartOpts, error) {
	opts.Env = setSessionStartEnvValue(opts.Env, "AGH_PROVIDER", strings.TrimSpace(resolved.Provider))
	opts.Env = setSessionStartEnvValue(opts.Env, "AGH_PROVIDER_HARNESS", string(resolved.Harness))
	opts.Env = setSessionStartEnvValue(opts.Env, "AGH_PROVIDER_AUTH_MODE", string(resolved.AuthMode))
	opts.Env = setSessionStartEnvValue(opts.Env, "AGH_PROVIDER_ENV_POLICY", string(resolved.EnvPolicy))
	opts.Env = setSessionStartEnvValue(opts.Env, "AGH_PROVIDER_HOME_POLICY", string(resolved.HomePolicy))
	opts.Env = setSessionStartEnvValue(opts.Env, "AGH_MODEL", strings.TrimSpace(resolved.Model))

	var err error
	if resolved.HomePolicy == aghconfig.ProviderHomePolicyIsolated {
		opts.Env, err = providerenv.ApplyHomePolicy(
			m.homePaths,
			strings.TrimSpace(resolved.Provider),
			resolved.HomePolicy,
			opts.Env,
		)
		if err != nil {
			return acp.StartOpts{}, fmt.Errorf("session: apply provider home policy: %w", err)
		}
	}
	if resolved.Harness == aghconfig.ProviderHarnessPiACP &&
		resolved.AuthMode == aghconfig.ProviderAuthModeNativeCLI {
		opts.Env, err = providerenv.ApplyPiAgentDirPolicy(
			m.homePaths,
			strings.TrimSpace(resolved.Provider),
			resolved.HomePolicy,
			opts.Env,
		)
		if err != nil {
			return acp.StartOpts{}, fmt.Errorf("session: apply pi auth directory policy: %w", err)
		}
	}

	secretBindings, err := m.injectProviderSecrets(ctx, resolved, opts.Env)
	if err != nil {
		return acp.StartOpts{}, err
	}
	opts.Env = secretBindings.env
	if session != nil {
		session.addProviderSecretRedactions(secretBindings.redactionCleanups)
	}
	if resolved.Harness != aghconfig.ProviderHarnessPiACP {
		return opts, nil
	}
	if resolved.AuthMode != aghconfig.ProviderAuthModeBoundSecret {
		return opts, nil
	}
	runtimeDir, err := m.materializePiRuntime(session, resolved, secretBindings.injectedTargetEnvs)
	if err != nil {
		return acp.StartOpts{}, err
	}
	opts.Env = setSessionStartEnvValue(opts.Env, "PI_CODING_AGENT_DIR", runtimeDir)
	return opts, nil
}

type providerSecretBindings struct {
	env                []string
	injectedTargetEnvs map[string]struct{}
	redactionCleanups  []func()
}

func (m *Manager) injectProviderSecrets(
	ctx context.Context,
	resolved aghconfig.ResolvedAgent,
	env []string,
) (providerSecretBindings, error) {
	bindings := providerSecretBindings{
		env:                env,
		injectedTargetEnvs: make(map[string]struct{}),
	}
	if len(resolved.CredentialSlots) == 0 {
		return bindings, nil
	}
	for _, slot := range resolved.CredentialSlots {
		updated, targetEnv, cleanup, err := m.injectProviderSecret(ctx, resolved, slot, bindings.env)
		if err != nil {
			runProviderSecretRedactions(bindings.redactionCleanups)
			return providerSecretBindings{}, err
		}
		bindings.env = updated
		if targetEnv == "" {
			continue
		}
		bindings.injectedTargetEnvs[targetEnv] = struct{}{}
		if cleanup != nil {
			bindings.redactionCleanups = append(bindings.redactionCleanups, cleanup)
		}
	}
	return bindings, nil
}

func (m *Manager) injectProviderSecret(
	ctx context.Context,
	resolved aghconfig.ResolvedAgent,
	slot aghconfig.ProviderCredentialSlot,
	env []string,
) ([]string, string, func(), error) {
	secretRef := vault.NormalizeRef(slot.SecretRef)
	targetEnv := strings.TrimSpace(slot.TargetEnv)
	if secretRef == "" || targetEnv == "" {
		return env, "", nil, nil
	}
	if vault.IsSecretRef(secretRef) {
		env = unsetSessionStartEnvKeys(env, targetEnv)
	}
	value, err := m.providerSecrets.ResolveRef(ctx, secretRef)
	if err != nil {
		if shouldSkipMissingProviderSecret(resolved, secretRef, slot, err) {
			return env, "", nil, nil
		}
		return nil, "", nil, fmt.Errorf(
			"session: resolve provider credential %q for %q: %w",
			slot.Name,
			resolved.Provider,
			err,
		)
	}
	return setSessionStartEnvValue(env, targetEnv, value), targetEnv, diagnostics.RegisterDynamicSecret(value), nil
}

func shouldSkipMissingProviderSecret(
	resolved aghconfig.ResolvedAgent,
	secretRef string,
	slot aghconfig.ProviderCredentialSlot,
	err error,
) bool {
	if resolved.Harness == aghconfig.ProviderHarnessPiACP {
		return !slot.Required && (errors.Is(err, vault.ErrMissingSecret) || errors.Is(err, vault.ErrSecretNotFound))
	}
	if vault.IsSecretRef(secretRef) {
		return !slot.Required && errors.Is(err, vault.ErrSecretNotFound)
	}
	if vault.IsEnvRef(secretRef) {
		return !slot.Required && errors.Is(err, vault.ErrMissingSecret)
	}
	return false
}

type piSettingsFile struct {
	DefaultProvider string   `json:"defaultProvider"`
	DefaultModel    string   `json:"defaultModel"`
	EnabledModels   []string `json:"enabledModels,omitempty"`
}

type piModelsFile struct {
	Providers map[string]piModelsProvider `json:"providers"`
}

type piModelsProvider struct {
	BaseURL string         `json:"baseUrl,omitempty"`
	API     string         `json:"api,omitempty"`
	APIKey  string         `json:"apiKey,omitempty"`
	Models  []piModelEntry `json:"models,omitempty"`
}

type piModelEntry struct {
	ID string `json:"id"`
}

func (m *Manager) materializePiRuntime(
	session *Session,
	resolved aghconfig.ResolvedAgent,
	injectedTargetEnvs map[string]struct{},
) (string, error) {
	if session == nil {
		return "", errors.New("session: pi runtime requires a session")
	}
	runtimeProvider := strings.TrimSpace(resolved.RuntimeProvider)
	if runtimeProvider == "" {
		runtimeProvider = strings.TrimSpace(resolved.Provider)
	}
	model := strings.TrimSpace(resolved.Model)
	if runtimeProvider == "" {
		return "", errors.New("session: pi runtime provider is required")
	}
	if model == "" {
		return "", errors.New("session: pi model is required")
	}

	runtimeDir := filepath.Join(session.sessionDir, "provider-runtime", "pi")
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		return "", fmt.Errorf("session: create pi runtime directory %q: %w", runtimeDir, err)
	}
	settings := piSettingsFile{
		DefaultProvider: runtimeProvider,
		DefaultModel:    model,
		EnabledModels:   []string{model},
	}
	if err := writeProviderJSON(filepath.Join(runtimeDir, "settings.json"), settings); err != nil {
		return "", err
	}

	models := piModelsFile{
		Providers: map[string]piModelsProvider{
			runtimeProvider: {
				BaseURL: strings.TrimSpace(resolved.BaseURL),
				API:     strings.TrimSpace(resolved.Transport),
				APIKey:  piCredentialEnv(resolved.CredentialSlots, injectedTargetEnvs),
				Models:  []piModelEntry{{ID: model}},
			},
		},
	}
	if err := writeProviderJSON(filepath.Join(runtimeDir, "models.json"), models); err != nil {
		return "", err
	}
	return runtimeDir, nil
}

func piCredentialEnv(slots []aghconfig.ProviderCredentialSlot, injectedTargetEnvs map[string]struct{}) string {
	for _, slot := range slots {
		targetEnv := strings.TrimSpace(slot.TargetEnv)
		if strings.TrimSpace(slot.Kind) == "api_key" && injectedProviderTargetEnv(targetEnv, injectedTargetEnvs) {
			return targetEnv
		}
	}
	for _, slot := range slots {
		targetEnv := strings.TrimSpace(slot.TargetEnv)
		if injectedProviderTargetEnv(targetEnv, injectedTargetEnvs) {
			return targetEnv
		}
	}
	return ""
}

func injectedProviderTargetEnv(targetEnv string, injectedTargetEnvs map[string]struct{}) bool {
	targetEnv = strings.TrimSpace(targetEnv)
	if targetEnv == "" || len(injectedTargetEnvs) == 0 {
		return false
	}
	_, ok := injectedTargetEnvs[targetEnv]
	return ok
}

func runProviderSecretRedactions(cleanups []func()) {
	for _, cleanup := range cleanups {
		if cleanup != nil {
			cleanup()
		}
	}
}

func writeProviderJSON(path string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("session: marshal provider runtime file %q: %w", path, err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return fmt.Errorf("session: write provider runtime file %q: %w", path, err)
	}
	return nil
}
