package settings

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/config/lifecycle"
	hookspkg "github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/providerauth"
	authproviders "github.com/compozy/agh/internal/providers"
	"github.com/compozy/agh/internal/vault"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

const settingsCredentialSourceEnv = "env"

func (s *service) ListCollection(ctx context.Context, req CollectionRequest) (CollectionEnvelope, error) {
	scope, workspaceID, err := s.normalizeReadScope(req.Scope, req.WorkspaceID)
	if err != nil {
		return CollectionEnvelope{}, fmt.Errorf("settings: list collection %q: %w", req.Collection, err)
	}
	if scope == ScopeAgent {
		return CollectionEnvelope{}, conflictError(
			fmt.Errorf("settings: collection %q does not support agent scope", req.Collection),
		)
	}
	if req.Collection != CollectionMCPServers && scope == ScopeWorkspace {
		return CollectionEnvelope{}, conflictError(
			fmt.Errorf("settings: collection %q does not support workspace scope", req.Collection),
		)
	}

	cfg, resolved, err := s.loadConfig(ctx, scope, workspaceID)
	if err != nil {
		return CollectionEnvelope{}, fmt.Errorf("settings: load collection %q config: %w", req.Collection, err)
	}

	envelope := CollectionEnvelope{
		Collection:  req.Collection,
		Scope:       scope,
		WorkspaceID: workspaceID,
	}

	switch req.Collection {
	case CollectionProviders:
		envelope.AvailableScopes = []ScopeKind{ScopeGlobal}
		items, buildErr := s.buildProviderItems(ctx, &cfg)
		if buildErr != nil {
			return CollectionEnvelope{}, buildErr
		}
		envelope.Providers = items
	case CollectionMCPServers:
		envelope.AvailableScopes = []ScopeKind{ScopeGlobal, ScopeWorkspace}
		items, buildErr := s.buildMCPServerItems(ctx, scope, workspaceID, resolved)
		if buildErr != nil {
			return CollectionEnvelope{}, buildErr
		}
		envelope.MCPServers = items
	case CollectionSandboxes:
		envelope.AvailableScopes = []ScopeKind{ScopeGlobal}
		items, buildErr := s.buildSandboxItems(ctx, &cfg)
		if buildErr != nil {
			return CollectionEnvelope{}, buildErr
		}
		envelope.Sandboxes = items
	case CollectionHooks:
		envelope.AvailableScopes = []ScopeKind{ScopeGlobal}
		envelope.Hooks = buildHookItems(cfg.Hooks.Declarations)
	default:
		return CollectionEnvelope{}, notFoundError(fmt.Errorf("settings: unknown collection %q", req.Collection))
	}

	return envelope, nil
}

func (s *service) PutCollectionItem(ctx context.Context, req CollectionItemPutRequest) (MutationResult, error) {
	finalize := func(result MutationResult, err error) (MutationResult, error) {
		return s.finalizeCommittedCollectionMutation(ctx, result, err, "replace", true)
	}

	scope, workspaceID, err := s.normalizeReadScope(req.Scope, req.WorkspaceID)
	if err != nil {
		return MutationResult{}, fmt.Errorf("settings: put collection item %q: %w", req.Collection, err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return MutationResult{}, validationError(errors.New("settings: collection item name is required"))
	}
	if scope == ScopeAgent {
		return MutationResult{}, conflictError(
			fmt.Errorf("settings: collection %q does not support agent scope", req.Collection),
		)
	}

	switch req.Collection {
	case CollectionProviders:
		if scope != ScopeGlobal {
			return MutationResult{}, conflictError(errors.New("settings: providers do not support workspace scope"))
		}
		if req.Provider == nil {
			return MutationResult{}, validationError(errors.New("settings: provider payload is required"))
		}
		return finalize(s.putProvider(
			ctx,
			name,
			*req.Provider,
			req.ProviderModelCuration,
			req.ProviderSecrets,
		))
	case CollectionMCPServers:
		return finalize(s.putMCPCollectionItem(ctx, scope, workspaceID, name, req))
	case CollectionSandboxes:
		if scope != ScopeGlobal {
			return MutationResult{}, conflictError(
				errors.New("settings: sandboxes do not support workspace scope"),
			)
		}
		if req.Sandbox == nil {
			return MutationResult{}, validationError(errors.New("settings: sandbox payload is required"))
		}
		return finalize(s.putSandbox(name, *req.Sandbox))
	case CollectionHooks:
		if scope != ScopeGlobal {
			return MutationResult{}, conflictError(errors.New("settings: hooks do not support workspace scope"))
		}
		if req.Hook == nil {
			return MutationResult{}, validationError(errors.New("settings: hook payload is required"))
		}
		return finalize(s.putHook(name, *req.Hook))
	default:
		return MutationResult{}, notFoundError(fmt.Errorf("settings: unknown collection %q", req.Collection))
	}
}

func (s *service) DeleteCollectionItem(ctx context.Context, req CollectionItemDeleteRequest) (MutationResult, error) {
	finalize := func(result MutationResult, err error) (MutationResult, error) {
		return s.finalizeCommittedCollectionMutation(ctx, result, err, "delete", false)
	}

	scope, workspaceID, err := s.normalizeReadScope(req.Scope, req.WorkspaceID)
	if err != nil {
		return MutationResult{}, fmt.Errorf("settings: delete collection item %q: %w", req.Collection, err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return MutationResult{}, validationError(errors.New("settings: collection item name is required"))
	}
	if scope == ScopeAgent {
		return MutationResult{}, conflictError(
			fmt.Errorf("settings: collection %q does not support agent scope", req.Collection),
		)
	}

	switch req.Collection {
	case CollectionProviders:
		if scope != ScopeGlobal {
			return MutationResult{}, conflictError(errors.New("settings: providers do not support workspace scope"))
		}
		return finalize(s.deleteProvider(name))
	case CollectionMCPServers:
		return finalize(s.deleteMCPServer(ctx, scope, workspaceID, name, req.Target))
	case CollectionSandboxes:
		if scope != ScopeGlobal {
			return MutationResult{}, conflictError(
				errors.New("settings: sandboxes do not support workspace scope"),
			)
		}
		return finalize(s.deleteSandbox(name))
	case CollectionHooks:
		if scope != ScopeGlobal {
			return MutationResult{}, conflictError(errors.New("settings: hooks do not support workspace scope"))
		}
		return finalize(s.deleteHook(name))
	default:
		return MutationResult{}, notFoundError(fmt.Errorf("settings: unknown collection %q", req.Collection))
	}
}

func (s *service) buildProviderItems(ctx context.Context, cfg *aghconfig.Config) ([]ProviderItem, error) {
	builtins := aghconfig.BuiltinProviders()
	names := make([]string, 0, len(builtins)+len(cfg.Providers))
	seen := make(map[string]struct{}, len(builtins)+len(cfg.Providers))
	for name := range builtins {
		names = append(names, name)
		seen[name] = struct{}{}
	}
	for name := range cfg.Providers {
		if _, ok := seen[name]; ok {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]ProviderItem, 0, len(names))
	for _, name := range names {
		resolved, err := cfg.ResolveProvider(name)
		if err != nil {
			return nil, fmt.Errorf("settings: resolve provider %q: %w", name, err)
		}

		settings, err := s.providerSettingsFromConfig(ctx, name, resolved)
		if err != nil {
			return nil, err
		}
		credentials, err := s.providerCredentialStatuses(ctx, resolved)
		if err != nil {
			return nil, fmt.Errorf("settings: provider %q credential status: %w", name, err)
		}
		authStatus, err := providerAuthStatus(s.homePaths, name, resolved, credentials, s.commandLookPath)
		if err != nil {
			return nil, fmt.Errorf("settings: provider %q auth status: %w", name, err)
		}
		item := ProviderItem{
			Name:             name,
			Settings:         settings,
			Default:          strings.TrimSpace(cfg.Defaults.Provider) == name,
			CommandAvailable: s.commandAvailable(resolved.Command),
			Credentials:      credentials,
			AuthStatus:       authStatus,
		}

		if overlay, ok := cfg.Providers[name]; ok {
			item.SourceMetadata = SourceMetadata{
				EffectiveSource:  sourceRefForWriteTarget(WriteTargetGlobalConfig, ""),
				AvailableTargets: []WriteTargetKind{WriteTargetGlobalConfig},
			}
			if builtin, builtinOK := builtins[name]; builtinOK {
				item.SourceMetadata.ShadowedSources = []SourceRef{builtinProviderSource()}
				item.Fallback = providerFallbackFromBuiltin(name, builtin)
			}
			if strings.TrimSpace(overlay.Command) == "" && item.Settings.Command == "" {
				item.CommandAvailable = false
			}
		} else {
			item.SourceMetadata = SourceMetadata{
				EffectiveSource:  builtinProviderSource(),
				AvailableTargets: []WriteTargetKind{WriteTargetGlobalConfig},
			}
		}

		items = append(items, cloneProviderItem(&item))
	}
	return items, nil
}

func providerAuthStatus(
	homePaths aghconfig.HomePaths,
	providerName string,
	provider aghconfig.ProviderConfig,
	credentials []ProviderCredentialStatus,
	lookPath func(string) (string, error),
) (ProviderAuthStatus, error) {
	status := ProviderAuthStatus{
		Mode:       provider.EffectiveAuthMode(),
		EnvPolicy:  provider.EffectiveEnvPolicy(),
		HomePolicy: provider.EffectiveHomePolicy(),
		StatusCmd:  strings.TrimSpace(provider.AuthStatusCmd),
		LoginCmd:   strings.TrimSpace(provider.AuthLoginCmd),
	}
	classification, err := authproviders.ClassifyDeclared(context.Background(), provider, providerAuthStatusProbeEnv(
		providerName,
		credentials,
		lookPath,
	))
	if err != nil {
		return ProviderAuthStatus{}, err
	}
	status.State = string(classification.State)
	status.Code = classification.Code
	status.Message = classification.Message
	switch status.Mode {
	case aghconfig.ProviderAuthModeBoundSecret:
		return status, nil
	case aghconfig.ProviderAuthModeNone:
		return status, nil
	default:
		nativeCLI, err := providerauth.NativeCLIStatusForProvider(provider, lookPath)
		if err != nil {
			return ProviderAuthStatus{}, err
		}
		status.NativeCLI = nativeCLI
		loginEnv, err := providerauth.NativeCLILoginEnv(homePaths, providerName, provider, os.Environ())
		if err != nil {
			return ProviderAuthStatus{}, err
		}
		status.LoginEnv = loginEnv
	}
	return status, nil
}

func providerAuthStatusProbeEnv(
	providerName string,
	credentials []ProviderCredentialStatus,
	lookPath func(string) (string, error),
) *authproviders.ProbeEnv {
	return &authproviders.ProbeEnv{
		ProviderName: strings.TrimSpace(providerName),
		LookPath:     lookPath,
		LookupEnv: func(key string) (string, bool) {
			for _, credential := range credentials {
				if credential.Source != settingsCredentialSourceEnv || !credential.Present {
					continue
				}
				envName, err := vault.EnvNameFromRef(credential.SecretRef)
				if err == nil && envName == key {
					return "present", true
				}
			}
			return "", false
		},
		Vault: providerAuthStatusCredentialVault(credentials),
	}
}

type providerAuthStatusCredentialVault []ProviderCredentialStatus

func (v providerAuthStatusCredentialVault) GetMetadata(_ context.Context, ref string) (vault.Metadata, error) {
	normalized := vault.NormalizeRef(ref)
	for _, credential := range v {
		if vault.NormalizeRef(credential.SecretRef) != normalized {
			continue
		}
		if !credential.Present {
			return vault.Metadata{}, vault.ErrSecretNotFound
		}
		return vault.Metadata{Ref: normalized, Present: true, Kind: strings.TrimSpace(credential.Kind)}, nil
	}
	return vault.Metadata{}, vault.ErrSecretNotFound
}

func (s *service) providerCredentialStatuses(
	ctx context.Context,
	provider aghconfig.ProviderConfig,
) ([]ProviderCredentialStatus, error) {
	slots := provider.EffectiveCredentialSlots()
	if len(slots) == 0 {
		return nil, nil
	}
	statuses := make([]ProviderCredentialStatus, 0, len(slots))
	for _, slot := range slots {
		status, err := s.providerCredentialStatus(ctx, slot)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (s *service) providerCredentialStatus(
	ctx context.Context,
	slot aghconfig.ProviderCredentialSlot,
) (ProviderCredentialStatus, error) {
	secretRef := vault.NormalizeRef(slot.SecretRef)
	status := ProviderCredentialStatus{
		Name:      strings.TrimSpace(slot.Name),
		TargetEnv: strings.TrimSpace(slot.TargetEnv),
		SecretRef: secretRef,
		Kind:      strings.TrimSpace(slot.Kind),
		Required:  slot.Required,
	}
	switch {
	case vault.IsEnvRef(secretRef):
		status.Source = settingsCredentialSourceEnv
		status.Present = s.envPresent(strings.TrimSpace(strings.TrimPrefix(secretRef, "env:")))
		return status, nil
	case vault.IsSecretRef(secretRef):
		status.Source = "vault"
		if s.providerSecrets == nil {
			return status, nil
		}
		metadata, err := s.providerSecrets.GetMetadata(ctx, secretRef)
		if err != nil {
			if errors.Is(err, vault.ErrSecretNotFound) {
				return status, nil
			}
			return ProviderCredentialStatus{}, err
		}
		status.Present = metadata.Present
		return status, nil
	default:
		status.Source = "unsupported"
		return status, nil
	}
}

func (s *service) buildSandboxItems(
	ctx context.Context,
	cfg *aghconfig.Config,
) ([]SandboxItem, error) {
	usage := make(map[string]int)
	if s.workspaceResolver != nil {
		workspaces, err := s.workspaceResolver.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("settings: list workspaces for sandbox usage: %w", err)
		}
		for _, workspace := range workspaces {
			ref := strings.TrimSpace(workspace.SandboxRef)
			if ref == "" {
				continue
			}
			usage[ref]++
		}
	}

	names := make([]string, 0, len(cfg.Sandboxes))
	for name := range cfg.Sandboxes {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]SandboxItem, 0, len(names))
	for _, name := range names {
		item := SandboxItem{
			Name:                name,
			Profile:             cfg.Sandboxes[name],
			WorkspaceUsageCount: usage[name],
			SourceMetadata:      globalConfigSourceMetadata(),
		}
		items = append(items, cloneSandboxItem(item))
	}
	return items, nil
}

func buildHookItems(declarations []hookspkg.HookDecl) []HookItem {
	items := make([]HookItem, 0, len(declarations))
	for _, decl := range declarations {
		item := HookItem{
			Name:           strings.TrimSpace(decl.Name),
			Declaration:    decl,
			SourceMetadata: globalConfigSourceMetadata(),
		}
		items = append(items, cloneHookItem(&item))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items
}

func (s *service) buildMCPServerItems(
	ctx context.Context,
	scope ScopeKind,
	workspaceID string,
	resolved *workspacepkg.ResolvedWorkspace,
) ([]MCPServerItem, error) {
	root := ""
	if resolved != nil {
		root = resolved.RootDir
	}

	sources, err := s.loadMCPSources(workspaceID, root, scope)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]MCPServerItem, 0, len(names))
	for _, name := range names {
		entries := sources[name]
		if len(entries) == 0 {
			continue
		}
		effective := entries[len(entries)-1]
		item := baseMCPServerItem(effective, entries, scope, workspaceID)
		if err := s.populateMCPServerStatuses(ctx, &item, effective); err != nil {
			return nil, err
		}
		items = append(items, cloneMCPServerItem(item))
	}
	return items, nil
}

func (s *service) putProvider(
	ctx context.Context,
	name string,
	settings ProviderSettings,
	modelCuration *ProviderModelCurationRequest,
	secrets []ProviderSecretWrite,
) (MutationResult, error) {
	reconciledSettings, err := s.reconcileProviderCuratedWrite(ctx, name, settings, modelCuration)
	if err != nil {
		return MutationResult{}, err
	}
	settings = reconciledSettings
	values := providerSettingsMap(settings)
	if len(values) == 0 && len(secrets) == 0 {
		return MutationResult{}, validationError(errors.New("settings: provider overlay requires at least one field"))
	}
	secretWrites, err := s.prepareProviderSecretWrites(name, secrets)
	if err != nil {
		return MutationResult{}, err
	}
	var target aghconfig.WriteTarget
	classification := providerWriteClassification{}
	if len(values) != 0 {
		target, err = aghconfig.ResolveConfigWriteTarget(s.homePaths, "", aghconfig.WriteScopeGlobal)
		if err != nil {
			return MutationResult{}, err
		}
		classification, err = s.classifyProviderWrite(ctx, name, settings)
		if err != nil {
			return MutationResult{}, fmt.Errorf("settings: write provider %q: %w", name, err)
		}
	}
	if classification.noOp && len(secretWrites) == 0 && modelCuration == nil {
		result := mutationResultForProvider(target.Kind(), true)
		result.Warnings = []string{sectionsNoChangesValue}
		return result, nil
	}
	if err := s.storePreparedSecrets(ctx, secretWrites); err != nil {
		return MutationResult{}, err
	}
	if len(values) == 0 {
		return mutationResultForCollection(CollectionProviders, ScopeGlobal, "", WriteTargetGlobalConfig), nil
	}

	if _, err := aghconfig.EditConfigOverlay(s.homePaths, "", target, func(editor *aghconfig.OverlayEditor) error {
		path := []string{string(CollectionProviders), name}
		if err := editor.Delete(path); err != nil {
			return err
		}
		return editor.SetTable(path, values)
	}); err != nil {
		return MutationResult{}, fmt.Errorf("settings: write provider %q: %w", name, err)
	}

	return mutationResultForProvider(target.Kind(), classification.modelOnly && len(secretWrites) == 0), nil
}

type preparedSecretWrite struct {
	description string
	ref         string
	kind        string
	value       string
}

func (s *service) prepareProviderSecretWrites(
	providerName string,
	secrets []ProviderSecretWrite,
) ([]preparedSecretWrite, error) {
	if len(secrets) == 0 {
		return nil, nil
	}
	if s.providerSecrets == nil {
		return nil, validationError(errors.New("settings: secret store is not available"))
	}
	prefix, err := vaultSecretOwnerPrefix("providers", providerName)
	if err != nil {
		return nil, validationError(err)
	}
	writes := make([]preparedSecretWrite, 0, len(secrets))
	for _, secret := range secrets {
		ref := vault.NormalizeRef(secret.SecretRef)
		if ref == "" {
			return nil, validationError(errors.New("settings: provider secret ref is required"))
		}
		if err := vault.ValidateSecretRefNamespace(ref, "providers"); err != nil {
			return nil, validationError(
				fmt.Errorf("%w: provider secret refs must use vault:providers/<provider>/<slot>", err),
			)
		}
		if !strings.HasPrefix(ref, prefix) {
			return nil, validationError(fmt.Errorf(
				"settings: provider secret ref %q must be scoped under %s",
				ref,
				strings.TrimSuffix(prefix, "/"),
			))
		}
		if strings.TrimSpace(secret.Value) == "" {
			return nil, validationError(errors.New("settings: provider secret value is required"))
		}
		writes = append(writes, preparedSecretWrite{
			description: fmt.Sprintf("provider secret %q", strings.TrimSpace(secret.Name)),
			ref:         ref,
			kind:        strings.TrimSpace(secret.Kind),
			value:       secret.Value,
		})
	}
	return writes, nil
}

func providerConfigFromSettings(settings ProviderSettings) aghconfig.ProviderConfig {
	return aghconfig.ProviderConfig{
		Command:         strings.TrimSpace(settings.Command),
		DisplayName:     strings.TrimSpace(settings.DisplayName),
		Models:          providerModelsConfigFromSettings(settings.Models),
		Harness:         settings.Harness,
		RuntimeProvider: strings.TrimSpace(settings.RuntimeProvider),
		Transport:       strings.TrimSpace(settings.Transport),
		BaseURL:         strings.TrimSpace(settings.BaseURL),
		AuthMode:        settings.AuthMode,
		EnvPolicy:       settings.EnvPolicy,
		HomePolicy:      settings.HomePolicy,
		AuthStatusCmd:   strings.TrimSpace(settings.AuthStatusCmd),
		AuthLoginCmd:    strings.TrimSpace(settings.AuthLoginCmd),
		CredentialSlots: providerCredentialSlotsFromSettings(settings.CredentialSlots),
	}
}

func providerModelsConfigFromSettings(models aghconfig.ProviderModelsConfig) aghconfig.ProviderModelsConfig {
	return aghconfig.ProviderModelsConfig{
		Default:   strings.TrimSpace(models.Default),
		Curated:   providerModelConfigsFromSettings(models.Curated),
		Discovery: providerModelsDiscoveryConfigFromSettings(models.Discovery),
		Reasoning: aghconfig.ProviderReasoningConfig{Apply: models.Reasoning.Apply},
	}
}

func providerModelConfigsFromSettings(
	models []aghconfig.ProviderModelConfig,
) []aghconfig.ProviderModelConfig {
	if models == nil {
		return nil
	}
	values := make([]aghconfig.ProviderModelConfig, 0, len(models))
	for _, model := range models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		values = append(values, aghconfig.ProviderModelConfig{
			ID:                     id,
			DisplayName:            strings.TrimSpace(model.DisplayName),
			ContextWindow:          cloneInt64Ptr(model.ContextWindow),
			MaxInputTokens:         cloneInt64Ptr(model.MaxInputTokens),
			MaxOutputTokens:        cloneInt64Ptr(model.MaxOutputTokens),
			SupportsTools:          cloneBoolPtr(model.SupportsTools),
			SupportsReasoning:      cloneBoolPtr(model.SupportsReasoning),
			ReasoningEfforts:       cloneStringSlicePreserveNil(model.ReasoningEfforts),
			DefaultReasoningEffort: strings.TrimSpace(model.DefaultReasoningEffort),
			CostInputPerMillion:    cloneFloat64Ptr(model.CostInputPerMillion),
			CostOutputPerMillion:   cloneFloat64Ptr(model.CostOutputPerMillion),
			Deprecated:             cloneBoolPtr(model.Deprecated),
			Hidden:                 cloneBoolPtr(model.Hidden),
			Featured:               cloneBoolPtr(model.Featured),
			ReleaseDate:            strings.TrimSpace(model.ReleaseDate),
		})
	}
	return values
}

func providerModelsDiscoveryConfigFromSettings(
	discovery aghconfig.ProviderModelsDiscoveryConfig,
) aghconfig.ProviderModelsDiscoveryConfig {
	return aghconfig.ProviderModelsDiscoveryConfig{
		Enabled:  cloneBoolPtr(discovery.Enabled),
		Command:  strings.TrimSpace(discovery.Command),
		Endpoint: strings.TrimSpace(discovery.Endpoint),
		Timeout:  strings.TrimSpace(discovery.Timeout),
	}
}

func providerCredentialSlotsFromSettings(
	slots []aghconfig.ProviderCredentialSlot,
) []aghconfig.ProviderCredentialSlot {
	values := make([]aghconfig.ProviderCredentialSlot, 0, len(slots))
	for _, slot := range slots {
		normalized := aghconfig.ProviderCredentialSlot{
			Name:      strings.TrimSpace(slot.Name),
			TargetEnv: strings.TrimSpace(slot.TargetEnv),
			SecretRef: strings.TrimSpace(slot.SecretRef),
			Kind:      strings.TrimSpace(slot.Kind),
			Required:  slot.Required,
		}
		if normalized.Name == "" && normalized.TargetEnv == "" && normalized.SecretRef == "" && normalized.Kind == "" {
			continue
		}
		values = append(values, normalized)
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func (s *service) storePreparedSecrets(ctx context.Context, writes []preparedSecretWrite) error {
	if len(writes) == 0 {
		return nil
	}
	if s.providerSecrets == nil {
		return validationError(errors.New("settings: secret store is not available"))
	}
	for _, write := range writes {
		if _, err := s.providerSecrets.PutSecret(ctx, write.ref, write.kind, write.value); err != nil {
			return fmt.Errorf("settings: store %s: %w", write.description, err)
		}
	}
	return nil
}

func vaultSecretOwnerPrefix(namespace string, owner string) (string, error) {
	normalizedNamespace := strings.Trim(strings.TrimSpace(namespace), "/")
	normalizedOwner := strings.Trim(strings.TrimSpace(owner), "/")
	if normalizedNamespace == "" {
		return "", errors.New("settings: secret namespace is required")
	}
	if normalizedOwner == "" {
		return "", errors.New("settings: secret owner is required")
	}
	prefix := "vault:" + normalizedNamespace + "/" + normalizedOwner + "/"
	if err := vault.ValidateSecretRef(prefix + "value"); err != nil {
		return "", fmt.Errorf("settings: invalid secret owner %q: %w", normalizedOwner, err)
	}
	return prefix, nil
}

func (s *service) deleteProvider(name string) (MutationResult, error) {
	target, err := aghconfig.ResolveConfigWriteTarget(s.homePaths, "", aghconfig.WriteScopeGlobal)
	if err != nil {
		return MutationResult{}, err
	}

	if _, err := aghconfig.EditConfigOverlay(s.homePaths, "", target, func(editor *aghconfig.OverlayEditor) error {
		path := []string{"providers", name}
		if !editor.HasPath(path) {
			return notFoundError(fmt.Errorf("settings: provider %q overlay not found", name))
		}
		return editor.Delete(path)
	}); err != nil {
		return MutationResult{}, fmt.Errorf("settings: delete provider %q: %w", name, err)
	}

	return mutationResultForCollection(CollectionProviders, ScopeGlobal, "", target.Kind()), nil
}

func (s *service) putSandbox(name string, profile aghconfig.SandboxProfile) (MutationResult, error) {
	values := sandboxProfileMap(profile)
	target, err := aghconfig.ResolveConfigWriteTarget(s.homePaths, "", aghconfig.WriteScopeGlobal)
	if err != nil {
		return MutationResult{}, err
	}

	if _, err := aghconfig.EditConfigOverlay(s.homePaths, "", target, func(editor *aghconfig.OverlayEditor) error {
		return editor.SetTable([]string{"sandboxes", name}, values)
	}); err != nil {
		return MutationResult{}, fmt.Errorf("settings: write sandbox %q: %w", name, err)
	}

	return mutationResultForCollection(CollectionSandboxes, ScopeGlobal, "", target.Kind()), nil
}

func (s *service) deleteSandbox(name string) (MutationResult, error) {
	target, err := aghconfig.ResolveConfigWriteTarget(s.homePaths, "", aghconfig.WriteScopeGlobal)
	if err != nil {
		return MutationResult{}, err
	}

	if _, err := aghconfig.EditConfigOverlay(s.homePaths, "", target, func(editor *aghconfig.OverlayEditor) error {
		path := []string{"sandboxes", name}
		if !editor.HasPath(path) {
			return notFoundError(fmt.Errorf("settings: sandbox %q not found", name))
		}
		return editor.Delete(path)
	}); err != nil {
		return MutationResult{}, fmt.Errorf("settings: delete sandbox %q: %w", name, err)
	}

	return mutationResultForCollection(CollectionSandboxes, ScopeGlobal, "", target.Kind()), nil
}

func (s *service) putHook(name string, declaration hookspkg.HookDecl) (MutationResult, error) {
	normalized, err := normalizeHookDeclaration(name, declaration)
	if err != nil {
		return MutationResult{}, err
	}

	target, err := aghconfig.ResolveConfigWriteTarget(s.homePaths, "", aghconfig.WriteScopeGlobal)
	if err != nil {
		return MutationResult{}, err
	}

	if _, err := aghconfig.EditConfigOverlay(s.homePaths, "", target, func(editor *aghconfig.OverlayEditor) error {
		return editor.UpsertArrayTableItem(
			[]string{"hooks", "declarations"},
			"name",
			name,
			hookDeclarationMap(normalized),
		)
	}); err != nil {
		return MutationResult{}, fmt.Errorf("settings: write hook %q: %w", name, err)
	}

	return mutationResultForCollection(CollectionHooks, ScopeGlobal, "", target.Kind()), nil
}

func (s *service) deleteHook(name string) (MutationResult, error) {
	target, err := aghconfig.ResolveConfigWriteTarget(s.homePaths, "", aghconfig.WriteScopeGlobal)
	if err != nil {
		return MutationResult{}, err
	}

	if _, err := aghconfig.EditConfigOverlay(s.homePaths, "", target, func(editor *aghconfig.OverlayEditor) error {
		deleted, deleteErr := editor.DeleteArrayTableItem([]string{"hooks", "declarations"}, "name", name)
		if deleteErr != nil {
			return deleteErr
		}
		if !deleted {
			return notFoundError(fmt.Errorf("settings: hook %q not found", name))
		}
		return nil
	}); err != nil {
		return MutationResult{}, fmt.Errorf("settings: delete hook %q: %w", name, err)
	}

	return mutationResultForCollection(CollectionHooks, ScopeGlobal, "", target.Kind()), nil
}

func mutationResultForCollection(
	collection CollectionName,
	scope ScopeKind,
	workspaceID string,
	target WriteTargetKind,
) MutationResult {
	classification := restartRequiredClassification()
	return MutationResult{
		Section:         SectionName(collection),
		Scope:           scope,
		WriteTarget:     target,
		WorkspaceID:     workspaceID,
		Behavior:        classification.Behavior,
		Applied:         classification.Applied,
		RestartRequired: classification.RestartRequired,
		RestartScope:    classification.RestartScope,
		Lifecycle:       lifecycle.RestartRequired,
		DiffClass:       lifecycle.DiffClassForRoot(string(collection)),
	}
}

func providerSettingsMap(settings ProviderSettings) map[string]any {
	values := make(map[string]any)
	if strings.TrimSpace(settings.Command) != "" {
		values["command"] = strings.TrimSpace(settings.Command)
	}
	if strings.TrimSpace(settings.DisplayName) != "" {
		values["display_name"] = strings.TrimSpace(settings.DisplayName)
	}
	if models := providerModelsSettingsMap(settings.Models); len(models) > 0 {
		values["models"] = models
	}
	if settings.Harness != "" {
		values["harness"] = string(settings.Harness)
	}
	if strings.TrimSpace(settings.RuntimeProvider) != "" {
		values["runtime_provider"] = strings.TrimSpace(settings.RuntimeProvider)
	}
	if strings.TrimSpace(settings.Transport) != "" {
		values["transport"] = strings.TrimSpace(settings.Transport)
	}
	if strings.TrimSpace(settings.BaseURL) != "" {
		values["base_url"] = strings.TrimSpace(settings.BaseURL)
	}
	if settings.AuthMode != "" {
		values["auth_mode"] = string(settings.AuthMode)
	}
	if settings.EnvPolicy != "" {
		values["env_policy"] = string(settings.EnvPolicy)
	}
	if settings.HomePolicy != "" {
		values["home_policy"] = string(settings.HomePolicy)
	}
	if strings.TrimSpace(settings.AuthStatusCmd) != "" {
		values["auth_status_command"] = strings.TrimSpace(settings.AuthStatusCmd)
	}
	if strings.TrimSpace(settings.AuthLoginCmd) != "" {
		values["auth_login_command"] = strings.TrimSpace(settings.AuthLoginCmd)
	}
	if len(settings.CredentialSlots) > 0 {
		values["credential_slots"] = providerCredentialSlotMaps(settings.CredentialSlots)
	}
	return values
}

func providerModelsSettingsMap(models aghconfig.ProviderModelsConfig) map[string]any {
	values := make(map[string]any)
	if strings.TrimSpace(models.Default) != "" {
		values["default"] = strings.TrimSpace(models.Default)
	}
	if models.Curated != nil {
		values["curated"] = providerModelConfigMaps(models.Curated)
	}
	if discovery := providerModelsDiscoveryMap(models.Discovery); len(discovery) > 0 {
		values["discovery"] = discovery
	}
	if models.Reasoning.Apply != "" {
		values["reasoning"] = map[string]any{"apply": string(models.Reasoning.Apply)}
	}
	return values
}

func providerModelConfigMaps(models []aghconfig.ProviderModelConfig) []map[string]any {
	values := make([]map[string]any, 0, len(models))
	for _, model := range models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		entry := make(map[string]any)
		entry["id"] = id
		if strings.TrimSpace(model.DisplayName) != "" {
			entry["display_name"] = strings.TrimSpace(model.DisplayName)
		}
		if model.ContextWindow != nil {
			entry["context_window"] = *model.ContextWindow
		}
		if model.MaxInputTokens != nil {
			entry["max_input_tokens"] = *model.MaxInputTokens
		}
		if model.MaxOutputTokens != nil {
			entry["max_output_tokens"] = *model.MaxOutputTokens
		}
		if model.SupportsTools != nil {
			entry["supports_tools"] = *model.SupportsTools
		}
		if model.SupportsReasoning != nil {
			entry["supports_reasoning"] = *model.SupportsReasoning
		}
		if model.ReasoningEfforts != nil {
			entry["reasoning_efforts"] = cloneStringSlicePreserveNil(model.ReasoningEfforts)
		}
		if strings.TrimSpace(model.DefaultReasoningEffort) != "" {
			entry["default_reasoning_effort"] = strings.TrimSpace(model.DefaultReasoningEffort)
		}
		if model.CostInputPerMillion != nil {
			entry["cost_input_per_million"] = *model.CostInputPerMillion
		}
		if model.CostOutputPerMillion != nil {
			entry["cost_output_per_million"] = *model.CostOutputPerMillion
		}
		if model.Deprecated != nil {
			entry["deprecated"] = *model.Deprecated
		}
		if model.Hidden != nil {
			entry["hidden"] = *model.Hidden
		}
		if model.Featured != nil {
			entry["featured"] = *model.Featured
		}
		if strings.TrimSpace(model.ReleaseDate) != "" {
			entry["release_date"] = strings.TrimSpace(model.ReleaseDate)
		}
		values = append(values, entry)
	}
	return values
}

func providerModelsDiscoveryMap(discovery aghconfig.ProviderModelsDiscoveryConfig) map[string]any {
	values := make(map[string]any)
	if discovery.Enabled != nil {
		values["enabled"] = *discovery.Enabled
	}
	if strings.TrimSpace(discovery.Command) != "" {
		values["command"] = strings.TrimSpace(discovery.Command)
	}
	if strings.TrimSpace(discovery.Endpoint) != "" {
		values["endpoint"] = strings.TrimSpace(discovery.Endpoint)
	}
	if strings.TrimSpace(discovery.Timeout) != "" {
		values["timeout"] = strings.TrimSpace(discovery.Timeout)
	}
	return values
}

func providerCredentialSlotMaps(slots []aghconfig.ProviderCredentialSlot) []map[string]any {
	values := make([]map[string]any, 0, len(slots))
	for _, slot := range slots {
		value := make(map[string]any)
		if strings.TrimSpace(slot.Name) != "" {
			value["name"] = strings.TrimSpace(slot.Name)
		}
		if strings.TrimSpace(slot.TargetEnv) != "" {
			value["target_env"] = strings.TrimSpace(slot.TargetEnv)
		}
		if strings.TrimSpace(slot.SecretRef) != "" {
			value["secret_ref"] = strings.TrimSpace(slot.SecretRef)
		}
		if strings.TrimSpace(slot.Kind) != "" {
			value["kind"] = strings.TrimSpace(slot.Kind)
		}
		value["required"] = slot.Required
		if len(value) > 1 {
			values = append(values, value)
		}
	}
	return values
}

func sandboxProfileMap(profile aghconfig.SandboxProfile) map[string]any {
	values := map[string]any{
		"backend": profile.Backend,
	}
	if strings.TrimSpace(profile.SyncMode) != "" {
		values["sync_mode"] = profile.SyncMode
	}
	if strings.TrimSpace(profile.Persistence) != "" {
		values["persistence"] = profile.Persistence
	}
	if strings.TrimSpace(profile.RuntimeRoot) != "" {
		values["runtime_root"] = profile.RuntimeRoot
	}
	if len(profile.Env) > 0 {
		values[settingsCredentialSourceEnv] = cloneStringMap(profile.Env)
	}
	if len(profile.SecretEnv) > 0 {
		values["secret_env"] = cloneStringMap(profile.SecretEnv)
	}
	if network := networkProfileMap(profile.Network); len(network) > 0 {
		values["network"] = network
	}
	if daytona := daytonaProfileMap(profile.Daytona); len(daytona) > 0 {
		values["daytona"] = daytona
	}
	return values
}

func networkProfileMap(profile aghconfig.NetworkProfile) map[string]any {
	if !profile.AllowPublicIngress &&
		!profile.AllowOutbound &&
		!profile.Required &&
		len(profile.AllowList) == 0 &&
		len(profile.DenyList) == 0 {
		return nil
	}

	network := map[string]any{
		"allow_public_ingress": profile.AllowPublicIngress,
		"allow_outbound":       profile.AllowOutbound,
		"required":             profile.Required,
	}
	if len(profile.AllowList) > 0 {
		network["allow_list"] = append([]string(nil), profile.AllowList...)
	}
	if len(profile.DenyList) > 0 {
		network["deny_list"] = append([]string(nil), profile.DenyList...)
	}
	return network
}

func daytonaProfileMap(profile aghconfig.DaytonaProfile) map[string]any {
	values := map[string]any{}
	if strings.TrimSpace(profile.APIURL) != "" {
		values["api_url"] = profile.APIURL
	}
	if strings.TrimSpace(profile.Target) != "" {
		values["target"] = profile.Target
	}
	if strings.TrimSpace(profile.Image) != "" {
		values["image"] = profile.Image
	}
	if strings.TrimSpace(profile.Snapshot) != "" {
		values["snapshot"] = profile.Snapshot
	}
	if strings.TrimSpace(profile.Class) != "" {
		values["class"] = profile.Class
	}
	if strings.TrimSpace(profile.AutoStop) != "" {
		values["auto_stop"] = profile.AutoStop
	}
	if strings.TrimSpace(profile.AutoArchive) != "" {
		values["auto_archive"] = profile.AutoArchive
	}
	return values
}

func normalizeHookDeclaration(name string, declaration hookspkg.HookDecl) (hookspkg.HookDecl, error) {
	normalized := cloneHookDecl(declaration)
	normalized.Name = strings.TrimSpace(normalized.Name)
	if normalized.Name == "" {
		normalized.Name = name
	}
	if normalized.Name != name {
		return hookspkg.HookDecl{}, validationError(fmt.Errorf(
			"settings: hook payload name %q does not match request name %q",
			normalized.Name,
			name,
		))
	}
	if err := hookspkg.ValidateHookDecl(normalized); err != nil {
		return hookspkg.HookDecl{}, validationError(fmt.Errorf("settings: validate hook %q: %w", name, err))
	}
	return normalized, nil
}

func hookMatcherMap(declaration hookspkg.HookDecl) map[string]any {
	matcher := map[string]any{}
	hookMatcherString(matcher, "agent_name", declaration.Matcher.AgentName)
	hookMatcherString(matcher, "agent_type", declaration.Matcher.AgentType)
	hookMatcherString(matcher, "workspace_id", declaration.Matcher.WorkspaceID)
	hookMatcherString(matcher, "workspace_root", declaration.Matcher.WorkspaceRoot)
	hookMatcherString(matcher, "session_type", declaration.Matcher.SessionType)
	hookMatcherString(matcher, "input_class", declaration.Matcher.InputClass)
	hookMatcherString(matcher, "acp_event_type", declaration.Matcher.ACPEventType)
	hookMatcherString(matcher, "turn_id", declaration.Matcher.TurnID)
	hookMatcherString(matcher, "tool_id", declaration.Matcher.ToolID)
	hookMatcherString(matcher, "tool_name", declaration.Matcher.ToolName)
	if declaration.Matcher.ToolReadOnly != nil {
		matcher["tool_read_only"] = *declaration.Matcher.ToolReadOnly
	}
	hookMatcherString(matcher, "decision_class", declaration.Matcher.DecisionClass)
	hookMatcherString(matcher, "message_role", declaration.Matcher.MessageRole)
	hookMatcherString(matcher, "message_delta_type", declaration.Matcher.MessageDeltaType)
	hookNetworkMatcherMap(matcher, declaration.Matcher.NetworkMatcher)
	hookCompactionMatcherMap(matcher, declaration.Matcher.CompactionMatcher)
	return matcher
}

func hookMatcherString(matcher map[string]any, key string, value string) {
	if strings.TrimSpace(value) != "" {
		matcher[key] = value
	}
}

func hookNetworkMatcherMap(matcher map[string]any, network *hookspkg.NetworkMatcher) {
	if network == nil {
		return
	}
	hookMatcherString(matcher, "channel", network.Channel)
	hookMatcherString(matcher, "surface", network.Surface)
	hookMatcherString(matcher, "kind", network.Kind)
	hookMatcherString(matcher, "direction", network.Direction)
	hookMatcherString(matcher, "work_state", network.WorkState)
}

func hookCompactionMatcherMap(matcher map[string]any, compaction *hookspkg.CompactionMatcher) {
	if compaction == nil {
		return
	}
	hookMatcherString(matcher, "compaction_reason", compaction.Reason)
	hookMatcherString(matcher, "compaction_strategy", compaction.Strategy)
}

func hookExecutorMap(declaration hookspkg.HookDecl) map[string]any {
	values := map[string]any{}
	if declaration.ExecutorKind != "" {
		values["kind"] = string(declaration.ExecutorKind)
	}
	if strings.TrimSpace(declaration.Command) != "" {
		values["command"] = declaration.Command
	}
	if len(declaration.Args) > 0 {
		values["args"] = append([]string(nil), declaration.Args...)
	}
	if len(declaration.Env) > 0 {
		values[settingsCredentialSourceEnv] = cloneStringMap(declaration.Env)
	}
	if len(declaration.SecretEnv) > 0 {
		values["secret_env"] = cloneStringMap(declaration.SecretEnv)
	}
	return values
}

func mcpAuthMap(auth aghconfig.MCPAuthConfig) map[string]any {
	values := map[string]any{}
	if auth.Type != "" {
		values["type"] = string(auth.Type)
	}
	if strings.TrimSpace(auth.IssuerURL) != "" {
		values["issuer_url"] = strings.TrimSpace(auth.IssuerURL)
	}
	if strings.TrimSpace(auth.MetadataURL) != "" {
		values["metadata_url"] = strings.TrimSpace(auth.MetadataURL)
	}
	if strings.TrimSpace(auth.AuthorizationURL) != "" {
		values["authorization_url"] = strings.TrimSpace(auth.AuthorizationURL)
	}
	if strings.TrimSpace(auth.TokenURL) != "" {
		values["token_url"] = strings.TrimSpace(auth.TokenURL)
	}
	if strings.TrimSpace(auth.RevocationURL) != "" {
		values["revocation_url"] = strings.TrimSpace(auth.RevocationURL)
	}
	if strings.TrimSpace(auth.ClientID) != "" {
		values["client_id"] = strings.TrimSpace(auth.ClientID)
	}
	if strings.TrimSpace(auth.ClientSecretRef) != "" {
		values["client_secret_ref"] = strings.TrimSpace(auth.ClientSecretRef)
	}
	if len(auth.Scopes) > 0 {
		values["scopes"] = append([]string(nil), auth.Scopes...)
	}
	return values
}

func (s *service) commandAvailable(command string) bool {
	binary := firstCommandToken(command)
	if binary == "" {
		return false
	}
	_, err := s.commandLookPath(binary)
	return err == nil
}

func firstCommandToken(command string) string {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func (s *service) envPresent(name string) bool {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return false
	}
	value, ok := s.lookupEnv(trimmed)
	return ok && strings.TrimSpace(value) != ""
}

func workspaceMCPSidecarPath(root string) string {
	return filepath.Join(strings.TrimSpace(root), aghconfig.DirName, aghconfig.MCPJSONName)
}
