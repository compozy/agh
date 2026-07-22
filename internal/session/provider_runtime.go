package session

import (
	"context"

	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/compozy/agh/internal/acp"
	aghconfig "github.com/compozy/agh/internal/config"

	"github.com/compozy/agh/internal/fileutil"
	"github.com/compozy/agh/internal/providerenv"
	authproviders "github.com/compozy/agh/internal/providers"

	"github.com/compozy/agh/internal/vault"
)

const (
	providerRuntimeAPIKeyKey = "api_key"
	codexAuthFileName        = "auth.json"
	codexHomeEnvKey          = "CODEX_HOME"
	providerCodexHomeEnvKey  = "PROVIDER_CODEX_HOME"
)

type envProviderSecretResolver struct {
	lookupEnv func(string) (string, bool)
}

const (
	runtimeProviderAnthropic = "anthropic"
	runtimeProviderClaude    = "claude"
	runtimeProviderCodex     = "codex"
)

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
	opts.Env = setProviderStartEnv(opts.Env, resolved)

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
	if shouldUseManagedOnboardingCodexHome(session, resolved) {
		opts.Env, err = m.applyManagedOnboardingCodexHome(ctx, session, opts.Env)
		if err != nil {
			return acp.StartOpts{}, fmt.Errorf("session: prepare onboarding codex home: %w", err)
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
	if resolved.Harness == aghconfig.ProviderHarnessPiACP &&
		resolved.AuthMode == aghconfig.ProviderAuthModeBoundSecret {
		runtimeDir, err := m.materializePiRuntime(
			session,
			resolved,
			secretBindings.injectedTargetEnvs,
		)
		if err != nil {
			return acp.StartOpts{}, err
		}
		opts.Env = setSessionStartEnvValue(opts.Env, "PI_CODING_AGENT_DIR", runtimeDir)
	}
	opts.ProviderName = strings.TrimSpace(resolved.Provider)
	providerConfig := providerConfigFromResolvedAgent(resolved)
	opts.ProviderConfig = &providerConfig
	probeEnv := providerProbeEnvForStart(m, resolved, opts.Env)
	opts.ProviderAuthEnv = &probeEnv
	return opts, nil
}

func setProviderStartEnv(env []string, resolved aghconfig.ResolvedAgent) []string {
	env = setSessionStartEnvValue(env, "AGH_PROVIDER", strings.TrimSpace(resolved.Provider))
	env = setSessionStartEnvValue(env, "AGH_PROVIDER_HARNESS", string(resolved.Harness))
	env = setSessionStartEnvValue(env, "AGH_PROVIDER_AUTH_MODE", string(resolved.AuthMode))
	env = setSessionStartEnvValue(env, "AGH_PROVIDER_ENV_POLICY", string(resolved.EnvPolicy))
	env = setSessionStartEnvValue(env, "AGH_PROVIDER_HOME_POLICY", string(resolved.HomePolicy))
	env = setSessionStartEnvValue(env, "AGH_MODEL", strings.TrimSpace(resolved.Model))
	return setProviderModelEnv(env, resolved)
}

func shouldUseManagedOnboardingCodexHome(session *Session, resolved aghconfig.ResolvedAgent) bool {
	if session == nil {
		return false
	}
	return sessionUsesManagedOnboardingAgent(session, resolved) &&
		effectiveRuntimeProvider(resolved) == runtimeProviderCodex &&
		resolved.AuthMode == aghconfig.ProviderAuthModeNativeCLI &&
		resolved.HomePolicy == aghconfig.ProviderHomePolicyOperator
}

func effectiveRuntimeProvider(resolved aghconfig.ResolvedAgent) string {
	runtimeProvider := strings.TrimSpace(resolved.RuntimeProvider)
	if runtimeProvider != "" {
		return runtimeProvider
	}
	return strings.TrimSpace(resolved.Provider)
}

func sessionUsesManagedOnboardingAgent(session *Session, resolved aghconfig.ResolvedAgent) bool {
	return strings.TrimSpace(session.AgentName) == aghconfig.OnboardingAgentName ||
		strings.TrimSpace(resolved.Name) == aghconfig.OnboardingAgentName
}

func (m *Manager) applyManagedOnboardingCodexHome(
	ctx context.Context,
	session *Session,
	env []string,
) ([]string, error) {
	if ctx == nil {
		return nil, errors.New("session: onboarding codex home context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if session == nil {
		return nil, errors.New("session: onboarding codex home requires a session")
	}
	workspaceID := strings.TrimSpace(session.WorkspaceID)
	if !providerenv.SafeProviderHomeSegment(workspaceID) {
		return nil, fmt.Errorf("workspace %q cannot use managed onboarding codex home", workspaceID)
	}
	if strings.TrimSpace(m.homePaths.HomeDir) == "" {
		return nil, errors.New("AGH home is required for managed onboarding codex home")
	}

	managedRoot := filepath.Clean(m.homePaths.HomeDir)
	codexHome := filepath.Join(
		managedRoot,
		"providers",
		runtimeProviderCodex,
		"onboarding",
		workspaceID,
		runtimeProviderCodex,
	)
	if err := providerenv.EnsurePrivateDirUnder(managedRoot, codexHome); err != nil {
		return nil, err
	}
	if err := materializeOnboardingCodexAuth(env, codexHome); err != nil {
		return nil, err
	}
	env = setSessionStartEnvValue(env, codexHomeEnvKey, codexHome)
	env = setSessionStartEnvValue(env, providerCodexHomeEnvKey, codexHome)
	return env, nil
}

func materializeOnboardingCodexAuth(env []string, codexHome string) error {
	sourceHome := operatorCodexHome(env)
	if sourceHome == "" {
		return nil
	}
	sourceAuth := filepath.Join(sourceHome, codexAuthFileName)
	targetAuth := filepath.Join(codexHome, codexAuthFileName)
	if sourceAuth == targetAuth {
		return nil
	}
	payload, err := os.ReadFile(sourceAuth)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read operator codex auth %q: %w", sourceAuth, err)
	}
	if err := fileutil.AtomicWriteFile(targetAuth, payload, 0o600); err != nil {
		return fmt.Errorf("write onboarding codex auth %q: %w", targetAuth, err)
	}
	if err := os.Chmod(targetAuth, 0o600); err != nil {
		return fmt.Errorf("protect onboarding codex auth %q: %w", targetAuth, err)
	}
	return nil
}

func operatorCodexHome(env []string) string {
	if value := providerEnvValue(env, codexHomeEnvKey); value != "" {
		return filepath.Clean(value)
	}
	if value := providerEnvValue(env, providerCodexHomeEnvKey); value != "" {
		return filepath.Clean(value)
	}
	if home := providerEnvValue(env, "HOME"); home != "" {
		return filepath.Join(home, ".codex")
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, ".codex")
	}
	return ""
}

func providerEnvValue(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if value, ok := strings.CutPrefix(entry, prefix); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func providerProbeEnvForStart(
	m *Manager,
	resolved aghconfig.ResolvedAgent,
	env []string,
) authproviders.ProbeEnv {
	return authproviders.ProbeEnv{
		ProviderName: strings.TrimSpace(resolved.Provider),
		HomePaths:    m.homePaths,
		LookupEnv:    providerLookupEnv(env),
		Vault:        providerSecretMetadataResolver{resolver: m.providerSecrets},
		CommandEnv:   append([]string(nil), env...),
	}
}

func providerLookupEnv(env []string) func(string) (string, bool) {
	values := make(map[string]string, len(env))
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	return func(key string) (string, bool) {
		if value, ok := values[key]; ok {
			return value, true
		}
		return os.LookupEnv(key)
	}
}
