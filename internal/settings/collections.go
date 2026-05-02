package settings

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/vault"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func (s *service) ListCollection(ctx context.Context, req CollectionRequest) (CollectionEnvelope, error) {
	scope, workspaceID, err := s.normalizeReadScope(req.Scope, req.WorkspaceID)
	if err != nil {
		return CollectionEnvelope{}, fmt.Errorf("settings: list collection %q: %w", req.Collection, err)
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
	scope, workspaceID, err := s.normalizeReadScope(req.Scope, req.WorkspaceID)
	if err != nil {
		return MutationResult{}, fmt.Errorf("settings: put collection item %q: %w", req.Collection, err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return MutationResult{}, validationError(errors.New("settings: collection item name is required"))
	}

	switch req.Collection {
	case CollectionProviders:
		if scope != ScopeGlobal {
			return MutationResult{}, conflictError(errors.New("settings: providers do not support workspace scope"))
		}
		if req.Provider == nil {
			return MutationResult{}, validationError(errors.New("settings: provider payload is required"))
		}
		return s.putProvider(ctx, name, *req.Provider, req.ProviderSecrets)
	case CollectionMCPServers:
		if req.MCPServer == nil {
			return MutationResult{}, validationError(errors.New("settings: MCP server payload is required"))
		}
		return s.putMCPServer(ctx, scope, workspaceID, name, req.Target, *req.MCPServer, req.MCPSecrets)
	case CollectionSandboxes:
		if scope != ScopeGlobal {
			return MutationResult{}, conflictError(
				errors.New("settings: sandboxes do not support workspace scope"),
			)
		}
		if req.Sandbox == nil {
			return MutationResult{}, validationError(errors.New("settings: sandbox payload is required"))
		}
		return s.putSandbox(name, *req.Sandbox)
	case CollectionHooks:
		if scope != ScopeGlobal {
			return MutationResult{}, conflictError(errors.New("settings: hooks do not support workspace scope"))
		}
		if req.Hook == nil {
			return MutationResult{}, validationError(errors.New("settings: hook payload is required"))
		}
		return s.putHook(name, *req.Hook)
	default:
		return MutationResult{}, notFoundError(fmt.Errorf("settings: unknown collection %q", req.Collection))
	}
}

func (s *service) DeleteCollectionItem(ctx context.Context, req CollectionItemDeleteRequest) (MutationResult, error) {
	scope, workspaceID, err := s.normalizeReadScope(req.Scope, req.WorkspaceID)
	if err != nil {
		return MutationResult{}, fmt.Errorf("settings: delete collection item %q: %w", req.Collection, err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return MutationResult{}, validationError(errors.New("settings: collection item name is required"))
	}

	switch req.Collection {
	case CollectionProviders:
		if scope != ScopeGlobal {
			return MutationResult{}, conflictError(errors.New("settings: providers do not support workspace scope"))
		}
		return s.deleteProvider(name)
	case CollectionMCPServers:
		return s.deleteMCPServer(ctx, scope, workspaceID, name, req.Target)
	case CollectionSandboxes:
		if scope != ScopeGlobal {
			return MutationResult{}, conflictError(
				errors.New("settings: sandboxes do not support workspace scope"),
			)
		}
		return s.deleteSandbox(name)
	case CollectionHooks:
		if scope != ScopeGlobal {
			return MutationResult{}, conflictError(errors.New("settings: hooks do not support workspace scope"))
		}
		return s.deleteHook(name)
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

		settings := providerSettingsFromConfig(name, resolved)
		credentials, err := s.providerCredentialStatuses(ctx, resolved)
		if err != nil {
			return nil, fmt.Errorf("settings: provider %q credential status: %w", name, err)
		}
		item := ProviderItem{
			Name:             name,
			Settings:         settings,
			Default:          strings.TrimSpace(cfg.Defaults.Provider) == name,
			CommandAvailable: s.commandAvailable(resolved.Command),
			Credentials:      credentials,
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

		items = append(items, cloneProviderItem(item))
	}
	return items, nil
}

func providerSettingsFromConfig(name string, provider aghconfig.ProviderConfig) ProviderSettings {
	return ProviderSettings{
		Command:         provider.Command,
		DisplayName:     provider.DisplayName,
		DefaultModel:    provider.DefaultModel,
		Harness:         provider.EffectiveHarness(),
		RuntimeProvider: provider.RuntimeProviderName(name),
		Transport:       strings.TrimSpace(provider.Transport),
		BaseURL:         strings.TrimSpace(provider.BaseURL),
		CredentialSlots: provider.EffectiveCredentialSlots(),
	}
}

func providerFallbackFromBuiltin(name string, builtin aghconfig.ProviderConfig) *ProviderFallback {
	return &ProviderFallback{
		Source:   builtinProviderSource(),
		Settings: providerSettingsFromConfig(name, builtin),
	}
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
		status.Source = "env"
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
		shadowed := make([]SourceRef, 0, len(entries)-1)
		for idx := len(entries) - 2; idx >= 0; idx-- {
			shadowed = append(shadowed, entries[idx].Source)
		}
		item := MCPServerItem{
			Name:        effective.Server.Name,
			Transport:   effective.Server.EffectiveTransport(),
			Command:     effective.Server.Command,
			Args:        append([]string(nil), effective.Server.Args...),
			Env:         aghconfig.RedactStringMap(effective.Server.Env),
			SecretEnv:   aghconfig.RedactStringMap(effective.Server.SecretEnv),
			URL:         strings.TrimSpace(effective.Server.URL),
			Auth:        effective.Server.Auth,
			Scope:       scope,
			WorkspaceID: workspaceID,
			SourceMetadata: SourceMetadata{
				EffectiveSource:  effective.Source,
				ShadowedSources:  shadowed,
				AvailableTargets: availableTargetsForScope(scope),
			},
		}
		if s.mcpAuth != nil && !effective.Server.Auth.IsZero() {
			status, statusErr := s.mcpAuth.MCPAuthStatus(ctx, effective.Server)
			if statusErr != nil {
				return nil, fmt.Errorf("settings: load MCP auth status for %q: %w", name, statusErr)
			}
			item.AuthStatus = &status
		}
		items = append(items, cloneMCPServerItem(item))
	}
	return items, nil
}

func (s *service) putProvider(
	ctx context.Context,
	name string,
	settings ProviderSettings,
	secrets []ProviderSecretWrite,
) (MutationResult, error) {
	values := providerSettingsMap(settings)
	if len(values) == 0 && len(secrets) == 0 {
		return MutationResult{}, validationError(errors.New("settings: provider overlay requires at least one field"))
	}
	if err := s.putProviderSecrets(ctx, name, secrets); err != nil {
		return MutationResult{}, err
	}
	if len(values) == 0 {
		return mutationResultForCollection(CollectionProviders, ScopeGlobal, "", WriteTargetGlobalConfig), nil
	}

	target, err := aghconfig.ResolveConfigWriteTarget(s.homePaths, "", aghconfig.WriteScopeGlobal)
	if err != nil {
		return MutationResult{}, err
	}

	if _, err := aghconfig.EditConfigOverlay(s.homePaths, "", target, func(editor *aghconfig.OverlayEditor) error {
		return editor.SetTable([]string{"providers", name}, values)
	}); err != nil {
		return MutationResult{}, fmt.Errorf("settings: write provider %q: %w", name, err)
	}

	return mutationResultForCollection(CollectionProviders, ScopeGlobal, "", target.Kind()), nil
}

func (s *service) putProviderSecrets(ctx context.Context, providerName string, secrets []ProviderSecretWrite) error {
	if len(secrets) == 0 {
		return nil
	}
	if s.providerSecrets == nil {
		return validationError(errors.New("settings: secret store is not available"))
	}
	prefix, err := vaultSecretOwnerPrefix("providers", providerName)
	if err != nil {
		return validationError(err)
	}
	for _, secret := range secrets {
		ref := vault.NormalizeRef(secret.SecretRef)
		if ref == "" {
			return validationError(errors.New("settings: provider secret ref is required"))
		}
		if err := vault.ValidateSecretRefNamespace(ref, "providers"); err != nil {
			return validationError(
				fmt.Errorf("%w: provider secret refs must use vault:providers/<provider>/<slot>", err),
			)
		}
		if !strings.HasPrefix(ref, prefix) {
			return validationError(fmt.Errorf(
				"settings: provider secret ref %q must be scoped under %s",
				ref,
				strings.TrimSuffix(prefix, "/"),
			))
		}
		if strings.TrimSpace(secret.Value) == "" {
			return validationError(errors.New("settings: provider secret value is required"))
		}
		if _, err := s.providerSecrets.PutSecret(ctx, ref, strings.TrimSpace(secret.Kind), secret.Value); err != nil {
			return fmt.Errorf("settings: store provider secret %q: %w", strings.TrimSpace(secret.Name), err)
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

func (s *service) putMCPServer(
	ctx context.Context,
	scope ScopeKind,
	workspaceID string,
	name string,
	selector TargetSelector,
	server aghconfig.MCPServer,
	secrets MCPSecretValues,
) (MutationResult, error) {
	root, sources, err := s.resolveMCPTargetContext(ctx, scope, workspaceID)
	if err != nil {
		return MutationResult{}, err
	}
	target, err := s.resolveMCPPutTarget(scope, root, name, selector, sources)
	if err != nil {
		return MutationResult{}, err
	}

	normalized := server
	normalized.Name = strings.TrimSpace(normalized.Name)
	if normalized.Name == "" {
		normalized.Name = name
	}
	if normalized.Name != name {
		return MutationResult{}, validationError(fmt.Errorf(
			"settings: MCP server payload name %q does not match request name %q",
			normalized.Name,
			name,
		))
	}
	if err := s.putMCPSecrets(ctx, name, normalized, secrets); err != nil {
		return MutationResult{}, err
	}

	if target.Kind() == WriteTargetGlobalMCPSidecar || target.Kind() == WriteTargetWorkspaceMCPSidecar {
		if _, err := aghconfig.PutMCPSidecarServer(s.homePaths, root, target, normalized); err != nil {
			return MutationResult{}, fmt.Errorf("settings: write MCP server %q: %w", name, err)
		}
	} else {
		if _, err := aghconfig.EditConfigOverlay(
			s.homePaths,
			root,
			target,
			func(editor *aghconfig.OverlayEditor) error {
				return editor.UpsertArrayTableItem([]string{"mcp_servers"}, "name", name, mcpServerMap(normalized))
			},
		); err != nil {
			return MutationResult{}, fmt.Errorf("settings: write MCP server %q: %w", name, err)
		}
	}

	return mutationResultForCollection(CollectionMCPServers, scope, workspaceID, target.Kind()), nil
}

func (s *service) putMCPSecrets(
	ctx context.Context,
	serverName string,
	server aghconfig.MCPServer,
	secrets MCPSecretValues,
) error {
	if secrets.Empty() {
		return nil
	}
	if s.providerSecrets == nil {
		return validationError(errors.New("settings: secret store is not available"))
	}
	prefix, err := vaultSecretOwnerPrefix("mcp", serverName)
	if err != nil {
		return validationError(err)
	}
	if err := s.putMCPSecretEnvValues(ctx, prefix, server, secrets.SecretEnv); err != nil {
		return err
	}
	if err := s.putMCPAuthClientSecretValue(ctx, prefix, server, secrets.OAuthClientSecret); err != nil {
		return err
	}
	return nil
}

func (s *service) putMCPSecretEnvValues(
	ctx context.Context,
	prefix string,
	server aghconfig.MCPServer,
	values map[string]string,
) error {
	if len(values) == 0 {
		return nil
	}
	if server.EffectiveTransport() != aghconfig.MCPServerTransportStdio {
		return validationError(errors.New("settings: MCP secret_env values require stdio transport"))
	}
	for key, value := range values {
		envName := strings.TrimSpace(key)
		if !vault.EnvNamePattern.MatchString(envName) {
			return validationError(fmt.Errorf("settings: MCP secret_env key %q is invalid", envName))
		}
		ref, ok := declaredSecretEnvRef(server.SecretEnv, envName)
		if !ok {
			return validationError(
				fmt.Errorf(
					"settings: MCP secret_env value %q has no matching server.secret_env ref",
					envName,
				),
			)
		}
		expectedRef := prefix + "env/" + envName
		if ref != expectedRef {
			return validationError(fmt.Errorf(
				"settings: MCP secret_env ref %q must be scoped under %s",
				ref,
				expectedRef,
			))
		}
		if strings.TrimSpace(value) == "" {
			return validationError(fmt.Errorf("settings: MCP secret_env value %q is required", envName))
		}
		if _, err := s.providerSecrets.PutSecret(ctx, ref, "mcp_env", value); err != nil {
			return fmt.Errorf("settings: store MCP secret_env %q: %w", envName, err)
		}
	}
	return nil
}

func (s *service) putMCPAuthClientSecretValue(
	ctx context.Context,
	prefix string,
	server aghconfig.MCPServer,
	value *string,
) error {
	if value == nil {
		return nil
	}
	ref := vault.NormalizeRef(server.Auth.ClientSecretRef)
	expectedRef := prefix + "oauth/client-secret"
	if ref == "" {
		return validationError(errors.New("settings: MCP OAuth client_secret_ref is required for oauth_client_secret"))
	}
	if ref != expectedRef {
		return validationError(fmt.Errorf(
			"settings: MCP OAuth client_secret_ref %q must be %s",
			ref,
			expectedRef,
		))
	}
	if err := vault.ValidateSecretRefNamespace(ref, "mcp"); err != nil {
		return validationError(fmt.Errorf("settings: MCP OAuth client_secret_ref is invalid: %w", err))
	}
	if strings.TrimSpace(*value) == "" {
		return validationError(errors.New("settings: MCP OAuth client secret value is required"))
	}
	if _, err := s.providerSecrets.PutSecret(ctx, ref, "mcp_oauth_client_secret", *value); err != nil {
		return fmt.Errorf("settings: store MCP OAuth client secret: %w", err)
	}
	return nil
}

func declaredSecretEnvRef(secretEnv map[string]string, envName string) (string, bool) {
	for key, ref := range secretEnv {
		if strings.TrimSpace(key) == envName {
			return vault.NormalizeRef(ref), true
		}
	}
	return "", false
}

func (s *service) deleteMCPServer(
	ctx context.Context,
	scope ScopeKind,
	workspaceID string,
	name string,
	selector TargetSelector,
) (MutationResult, error) {
	root, sources, err := s.resolveMCPTargetContext(ctx, scope, workspaceID)
	if err != nil {
		return MutationResult{}, err
	}
	target, err := s.resolveMCPDeleteTarget(scope, root, name, selector, sources)
	if err != nil {
		return MutationResult{}, err
	}

	if target.Kind() == WriteTargetGlobalMCPSidecar || target.Kind() == WriteTargetWorkspaceMCPSidecar {
		_, deleted, deleteErr := aghconfig.DeleteMCPSidecarServer(s.homePaths, root, target, name)
		if deleteErr != nil {
			return MutationResult{}, fmt.Errorf("settings: delete MCP server %q: %w", name, deleteErr)
		}
		if !deleted {
			return MutationResult{}, notFoundError(
				fmt.Errorf("settings: MCP server %q not found in %q", name, target.Kind()),
			)
		}
	} else {
		if _, err := aghconfig.EditConfigOverlay(
			s.homePaths,
			root,
			target,
			func(editor *aghconfig.OverlayEditor) error {
				deleted, deleteErr := editor.DeleteArrayTableItem([]string{"mcp_servers"}, "name", name)
				if deleteErr != nil {
					return deleteErr
				}
				if !deleted {
					return notFoundError(
						fmt.Errorf("settings: MCP server %q not found in %q", name, target.Kind()),
					)
				}
				return nil
			},
		); err != nil {
			return MutationResult{}, fmt.Errorf("settings: delete MCP server %q: %w", name, err)
		}
	}

	return mutationResultForCollection(CollectionMCPServers, scope, workspaceID, target.Kind()), nil
}

type mcpSourceEntry struct {
	Source SourceRef
	Target WriteTargetKind
	Server aghconfig.MCPServer
}

func (s *service) resolveMCPTargetContext(
	ctx context.Context,
	scope ScopeKind,
	workspaceID string,
) (string, map[string][]mcpSourceEntry, error) {
	resolved, err := s.resolveWorkspace(ctx, scope, workspaceID)
	if err != nil {
		return "", nil, err
	}
	root := ""
	if resolved != nil {
		root = resolved.RootDir
	}

	sources, err := s.loadMCPSources(workspaceID, root, scope)
	if err != nil {
		return "", nil, err
	}
	return root, sources, nil
}

func (s *service) loadMCPSources(
	workspaceID string,
	workspaceRoot string,
	scope ScopeKind,
) (map[string][]mcpSourceEntry, error) {
	sources := make(map[string][]mcpSourceEntry)

	appendServers := func(kind WriteTargetKind, serverList []aghconfig.MCPServer) {
		for _, server := range serverList {
			name := strings.TrimSpace(server.Name)
			if name == "" {
				continue
			}
			sources[name] = append(sources[name], mcpSourceEntry{
				Source: sourceRefForWriteTarget(kind, workspaceID),
				Target: kind,
				Server: server,
			})
		}
	}

	globalConfigServers, err := loadMCPServersFromConfigFile(s.homePaths.ConfigFile, s.homePaths)
	if err != nil {
		return nil, fmt.Errorf("settings: load global config MCP servers: %w", err)
	}
	appendServers(WriteTargetGlobalConfig, globalConfigServers)

	globalSidecarServers, err := aghconfig.LoadMCPServersJSONFile(globalMCPSidecarPath(s.homePaths))
	if err != nil {
		return nil, fmt.Errorf("settings: load global MCP sidecar: %w", err)
	}
	appendServers(WriteTargetGlobalMCPSidecar, globalSidecarServers)

	if scope == ScopeWorkspace {
		workspaceConfigServers, loadErr := loadMCPServersFromConfigFile(workspaceConfigPath(workspaceRoot), s.homePaths)
		if loadErr != nil {
			return nil, fmt.Errorf("settings: load workspace config MCP servers: %w", loadErr)
		}
		appendServers(WriteTargetWorkspaceConfig, workspaceConfigServers)

		workspaceSidecarServers, loadErr := aghconfig.LoadMCPServersJSONFile(workspaceMCPSidecarPath(workspaceRoot))
		if loadErr != nil {
			return nil, fmt.Errorf("settings: load workspace MCP sidecar: %w", loadErr)
		}
		appendServers(WriteTargetWorkspaceMCPSidecar, workspaceSidecarServers)
	}

	return sources, nil
}

func loadMCPServersFromConfigFile(path string, homePaths aghconfig.HomePaths) ([]aghconfig.MCPServer, error) {
	cfg := aghconfig.DefaultWithHome(homePaths)
	if err := aghconfig.ApplyConfigOverlayFile(path, &cfg); err != nil {
		return nil, err
	}
	return append([]aghconfig.MCPServer(nil), cfg.MCPServers...), nil
}

func (s *service) resolveMCPPutTarget(
	scope ScopeKind,
	workspaceRoot string,
	name string,
	selector TargetSelector,
	sources map[string][]mcpSourceEntry,
) (aghconfig.WriteTarget, error) {
	normalized := normalizeTargetSelector(selector)
	if normalized == TargetConfig {
		return aghconfig.ResolveConfigWriteTarget(s.homePaths, workspaceRoot, scope)
	}
	if normalized == TargetSidecar {
		return aghconfig.ResolveMCPSidecarWriteTarget(s.homePaths, workspaceRoot, scope)
	}

	targetKind := preferredMCPPutTarget(scope, name, sources)
	switch targetKind {
	case WriteTargetGlobalConfig, WriteTargetWorkspaceConfig:
		return aghconfig.ResolveConfigWriteTarget(s.homePaths, workspaceRoot, scope)
	case WriteTargetGlobalMCPSidecar, WriteTargetWorkspaceMCPSidecar:
		return aghconfig.ResolveMCPSidecarWriteTarget(s.homePaths, workspaceRoot, scope)
	default:
		return aghconfig.WriteTarget{}, conflictError(
			fmt.Errorf("settings: unsupported MCP write target %q for %q", targetKind, name),
		)
	}
}

func (s *service) resolveMCPDeleteTarget(
	scope ScopeKind,
	workspaceRoot string,
	name string,
	selector TargetSelector,
	sources map[string][]mcpSourceEntry,
) (aghconfig.WriteTarget, error) {
	normalized := normalizeTargetSelector(selector)
	if normalized == TargetConfig {
		return aghconfig.ResolveConfigWriteTarget(s.homePaths, workspaceRoot, scope)
	}
	if normalized == TargetSidecar {
		return aghconfig.ResolveMCPSidecarWriteTarget(s.homePaths, workspaceRoot, scope)
	}

	targetKind, ok := preferredMCPDeleteTarget(scope, name, sources)
	if !ok {
		return aghconfig.WriteTarget{}, notFoundError(
			fmt.Errorf("settings: MCP server %q has no definition in %s scope", name, scope),
		)
	}
	switch targetKind {
	case WriteTargetGlobalConfig, WriteTargetWorkspaceConfig:
		return aghconfig.ResolveConfigWriteTarget(s.homePaths, workspaceRoot, scope)
	case WriteTargetGlobalMCPSidecar, WriteTargetWorkspaceMCPSidecar:
		return aghconfig.ResolveMCPSidecarWriteTarget(s.homePaths, workspaceRoot, scope)
	default:
		return aghconfig.WriteTarget{}, conflictError(
			fmt.Errorf("settings: unsupported MCP write target %q for %q", targetKind, name),
		)
	}
}

func preferredMCPPutTarget(scope ScopeKind, name string, sources map[string][]mcpSourceEntry) WriteTargetKind {
	if targetKind, ok := preferredMCPDeleteTarget(scope, name, sources); ok {
		return targetKind
	}
	if scope == ScopeWorkspace {
		return WriteTargetWorkspaceMCPSidecar
	}
	return WriteTargetGlobalMCPSidecar
}

func preferredMCPDeleteTarget(
	scope ScopeKind,
	name string,
	sources map[string][]mcpSourceEntry,
) (WriteTargetKind, bool) {
	entries := sources[strings.TrimSpace(name)]
	if len(entries) == 0 {
		return "", false
	}

	switch scope {
	case ScopeWorkspace:
		for idx := len(entries) - 1; idx >= 0; idx-- {
			switch entries[idx].Target {
			case WriteTargetWorkspaceMCPSidecar, WriteTargetWorkspaceConfig:
				return entries[idx].Target, true
			}
		}
	default:
		for idx := len(entries) - 1; idx >= 0; idx-- {
			switch entries[idx].Target {
			case WriteTargetGlobalMCPSidecar, WriteTargetGlobalConfig:
				return entries[idx].Target, true
			}
		}
	}

	return "", false
}

func normalizeTargetSelector(selector TargetSelector) TargetSelector {
	trimmed := TargetSelector(strings.TrimSpace(string(selector)))
	if trimmed == "" {
		return TargetAuto
	}
	return trimmed
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
	if strings.TrimSpace(settings.DefaultModel) != "" {
		values["default_model"] = strings.TrimSpace(settings.DefaultModel)
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
	if len(settings.CredentialSlots) > 0 {
		values["credential_slots"] = providerCredentialSlotMaps(settings.CredentialSlots)
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
		values["env"] = cloneStringMap(profile.Env)
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

func hookDeclarationMap(declaration hookspkg.HookDecl) map[string]any {
	values := map[string]any{
		"event": string(declaration.Event),
	}
	if declaration.Mode != "" {
		values["mode"] = string(declaration.Mode)
	}
	if declaration.Required {
		values["required"] = declaration.Required
	}
	if declaration.PrioritySet {
		values["priority"] = declaration.Priority
	}
	if declaration.Timeout > 0 {
		values["timeout"] = declaration.Timeout.String()
	}
	if matcher := hookMatcherMap(declaration); len(matcher) > 0 {
		values["matcher"] = matcher
	}
	if executor := hookExecutorMap(declaration); len(executor) > 0 {
		values["executor"] = executor
	} else {
		if strings.TrimSpace(declaration.Command) != "" {
			values["command"] = declaration.Command
		}
		if len(declaration.Args) > 0 {
			values["args"] = append([]string(nil), declaration.Args...)
		}
		if len(declaration.Env) > 0 {
			values["env"] = cloneStringMap(declaration.Env)
		}
		if len(declaration.SecretEnv) > 0 {
			values["secret_env"] = cloneStringMap(declaration.SecretEnv)
		}
	}
	return values
}

func hookMatcherMap(declaration hookspkg.HookDecl) map[string]any {
	matcher := map[string]any{}
	if strings.TrimSpace(declaration.Matcher.AgentName) != "" {
		matcher["agent_name"] = declaration.Matcher.AgentName
	}
	if strings.TrimSpace(declaration.Matcher.AgentType) != "" {
		matcher["agent_type"] = declaration.Matcher.AgentType
	}
	if strings.TrimSpace(declaration.Matcher.WorkspaceID) != "" {
		matcher["workspace_id"] = declaration.Matcher.WorkspaceID
	}
	if strings.TrimSpace(declaration.Matcher.WorkspaceRoot) != "" {
		matcher["workspace_root"] = declaration.Matcher.WorkspaceRoot
	}
	if strings.TrimSpace(declaration.Matcher.SessionType) != "" {
		matcher["session_type"] = declaration.Matcher.SessionType
	}
	if strings.TrimSpace(declaration.Matcher.InputClass) != "" {
		matcher["input_class"] = declaration.Matcher.InputClass
	}
	if strings.TrimSpace(declaration.Matcher.ACPEventType) != "" {
		matcher["acp_event_type"] = declaration.Matcher.ACPEventType
	}
	if strings.TrimSpace(declaration.Matcher.TurnID) != "" {
		matcher["turn_id"] = declaration.Matcher.TurnID
	}
	if strings.TrimSpace(declaration.Matcher.ToolID) != "" {
		matcher["tool_id"] = declaration.Matcher.ToolID
	}
	if strings.TrimSpace(declaration.Matcher.ToolName) != "" {
		matcher["tool_name"] = declaration.Matcher.ToolName
	}
	if declaration.Matcher.ToolReadOnly != nil {
		matcher["tool_read_only"] = *declaration.Matcher.ToolReadOnly
	}
	if strings.TrimSpace(declaration.Matcher.DecisionClass) != "" {
		matcher["decision_class"] = declaration.Matcher.DecisionClass
	}
	if strings.TrimSpace(declaration.Matcher.MessageRole) != "" {
		matcher["message_role"] = declaration.Matcher.MessageRole
	}
	if strings.TrimSpace(declaration.Matcher.MessageDeltaType) != "" {
		matcher["message_delta_type"] = declaration.Matcher.MessageDeltaType
	}
	if strings.TrimSpace(declaration.Matcher.CompactionReason) != "" {
		matcher["compaction_reason"] = declaration.Matcher.CompactionReason
	}
	if strings.TrimSpace(declaration.Matcher.CompactionStrategy) != "" {
		matcher["compaction_strategy"] = declaration.Matcher.CompactionStrategy
	}
	return matcher
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
		values["env"] = cloneStringMap(declaration.Env)
	}
	if len(declaration.SecretEnv) > 0 {
		values["secret_env"] = cloneStringMap(declaration.SecretEnv)
	}
	return values
}

func mcpServerMap(server aghconfig.MCPServer) map[string]any {
	values := map[string]any{}
	if server.Transport != "" {
		values["transport"] = string(server.Transport)
	}
	if strings.TrimSpace(server.Command) != "" {
		values["command"] = strings.TrimSpace(server.Command)
	}
	if len(server.Args) > 0 {
		values["args"] = append([]string(nil), server.Args...)
	}
	if len(server.Env) > 0 {
		values["env"] = cloneStringMap(server.Env)
	}
	if len(server.SecretEnv) > 0 {
		values["secret_env"] = cloneStringMap(server.SecretEnv)
	}
	if strings.TrimSpace(server.URL) != "" {
		values["url"] = strings.TrimSpace(server.URL)
	}
	if !server.Auth.IsZero() {
		values["auth"] = mcpAuthMap(server.Auth)
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
