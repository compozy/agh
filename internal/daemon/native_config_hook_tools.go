package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	core "github.com/pedronauck/agh/internal/api/core"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/store"
	toolspkg "github.com/pedronauck/agh/internal/tools"
)

const hookActionDisabled = "disabled"

func (n *daemonNativeTools) configShow(
	_ context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input configReadInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	cfg, workspaceRoot, err := n.loadNativeConfig(input.WorkspaceRoot)
	if err != nil {
		return toolspkg.ToolResult{}, nativeConfigValidationError(req.ToolID, err)
	}
	configMap := aghconfig.RedactedConfigMap(&cfg)
	return structuredResult(map[string]any{
		"scope":          nativeScopeForWorkspace(workspaceRoot),
		"workspace_root": workspaceRoot,
		"redacted":       true,
		"config":         configMap,
	}, "config")
}

func (n *daemonNativeTools) configList(
	_ context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input configReadInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	cfg, workspaceRoot, err := n.loadNativeConfig(input.WorkspaceRoot)
	if err != nil {
		return toolspkg.ToolResult{}, nativeConfigValidationError(req.ToolID, err)
	}
	entries := aghconfig.FlattenConfigEntries(aghconfig.RedactedConfigMap(&cfg))
	return structuredResult(map[string]any{
		"scope":          nativeScopeForWorkspace(workspaceRoot),
		"workspace_root": workspaceRoot,
		"redacted":       true,
		"entries":        entries,
	}, fmt.Sprintf("%d config entries", len(entries)))
}

func (n *daemonNativeTools) configGet(
	_ context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input configGetInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	cfg, workspaceRoot, err := n.loadNativeConfig(input.WorkspaceRoot)
	if err != nil {
		return toolspkg.ToolResult{}, nativeConfigValidationError(req.ToolID, err)
	}
	entries := aghconfig.FlattenConfigEntries(aghconfig.RedactedConfigMap(&cfg))
	entry, ok := aghconfig.EntryByPath(entries, input.Path)
	if !ok {
		return toolspkg.ToolResult{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeNotFound,
			req.ToolID,
			fmt.Sprintf("config path %q not found", strings.TrimSpace(input.Path)),
			toolspkg.ErrToolNotFound,
			toolspkg.ReasonToolUnknown,
		)
	}
	return structuredResult(map[string]any{
		"scope":          nativeScopeForWorkspace(workspaceRoot),
		"workspace_root": workspaceRoot,
		"entry":          entry,
	}, entry.Path)
}

func (n *daemonNativeTools) configSet(
	_ context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input configSetInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	policy, err := nativeConfigPathPolicy(req.ToolID, input.Path)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	value, err := aghconfig.NormalizeToolConfigValue(policy.Kind, input.Value)
	if err != nil {
		return toolspkg.ToolResult{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			req.ToolID,
			"config value is invalid",
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonConfigValidationFailed,
		)
	}
	target, workspaceRoot, err := n.nativeConfigWriteTarget(req.ToolID, input.Scope, input.WorkspaceRoot)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	if _, err := aghconfig.EditConfigOverlay(
		n.deps.HomePaths,
		workspaceRoot,
		target,
		func(editor *aghconfig.OverlayEditor) error {
			return editor.SetValue(policy.Segments, value)
		},
	); err != nil {
		return toolspkg.ToolResult{}, nativeConfigValidationError(req.ToolID, err)
	}
	outputValue := value
	if policy.Redacted {
		outputValue = aghconfig.RedactedValue()
	}
	path := strings.Join(policy.Segments, ".")
	return structuredResult(map[string]any{
		"path":     path,
		"value":    outputValue,
		"scope":    string(target.Scope()),
		"target":   target.Path(),
		"redacted": policy.Redacted,
	}, path)
}

func (n *daemonNativeTools) configUnset(
	_ context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input configUnsetInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	policy, err := nativeConfigPathPolicy(req.ToolID, input.Path)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	target, workspaceRoot, err := n.nativeConfigWriteTarget(req.ToolID, input.Scope, input.WorkspaceRoot)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	deleted := false
	if _, err := aghconfig.EditConfigOverlay(
		n.deps.HomePaths,
		workspaceRoot,
		target,
		func(editor *aghconfig.OverlayEditor) error {
			deleted = editor.HasPath(policy.Segments)
			return editor.Delete(policy.Segments)
		},
	); err != nil {
		return toolspkg.ToolResult{}, nativeConfigValidationError(req.ToolID, err)
	}
	path := strings.Join(policy.Segments, ".")
	return structuredResult(map[string]any{
		"path":    path,
		"deleted": deleted,
		"scope":   string(target.Scope()),
		"target":  target.Path(),
	}, path)
}

func (n *daemonNativeTools) configDiff(
	_ context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input configReadInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	workspaceRoot, err := nativeOptionalWorkspaceRoot(input.WorkspaceRoot)
	if err != nil {
		return toolspkg.ToolResult{}, nativeConfigScopeError(req.ToolID, err)
	}
	var beforeCfg aghconfig.Config
	if workspaceRoot == "" {
		beforeCfg = aghconfig.DefaultWithHome(n.deps.HomePaths)
	} else {
		beforeCfg, err = aghconfig.LoadForHome(n.deps.HomePaths)
		if err != nil {
			return toolspkg.ToolResult{}, nativeConfigValidationError(req.ToolID, err)
		}
	}
	loadOptions := []aghconfig.LoadOption{}
	if workspaceRoot != "" {
		loadOptions = append(loadOptions, aghconfig.WithWorkspaceRoot(workspaceRoot))
	}
	afterCfg, err := aghconfig.LoadForHome(n.deps.HomePaths, loadOptions...)
	if err != nil {
		return toolspkg.ToolResult{}, nativeConfigValidationError(req.ToolID, err)
	}
	before := aghconfig.FlattenConfigEntries(aghconfig.RedactedConfigMap(&beforeCfg))
	after := aghconfig.FlattenConfigEntries(aghconfig.RedactedConfigMap(&afterCfg))
	diff := aghconfig.DiffConfigEntries(before, after)
	return structuredResult(map[string]any{
		"scope":          nativeScopeForWorkspace(workspaceRoot),
		"workspace_root": workspaceRoot,
		"redacted":       true,
		"diff":           diff,
	}, fmt.Sprintf("%d config differences", len(diff)))
}

func (n *daemonNativeTools) configPath(
	_ context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input configPathInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	scope, err := nativeWriteScope(input.Scope)
	if err != nil {
		return toolspkg.ToolResult{}, nativeConfigScopeError(req.ToolID, err)
	}
	globalConfig, err := aghconfig.ResolveConfigWriteTarget(n.deps.HomePaths, "", aghconfig.WriteScopeGlobal)
	if err != nil {
		return toolspkg.ToolResult{}, nativeConfigScopeError(req.ToolID, err)
	}
	globalMCP, err := aghconfig.ResolveMCPSidecarWriteTarget(n.deps.HomePaths, "", aghconfig.WriteScopeGlobal)
	if err != nil {
		return toolspkg.ToolResult{}, nativeConfigScopeError(req.ToolID, err)
	}
	selected := globalConfig
	record := map[string]any{
		"home_dir":               n.deps.HomePaths.HomeDir,
		"global_config":          globalConfig.Path(),
		"global_mcp_json":        globalMCP.Path(),
		"scope":                  string(scope),
		"selected_config_target": selected.Path(),
	}
	if scope == aghconfig.WriteScopeWorkspace || strings.TrimSpace(input.WorkspaceRoot) != "" {
		workspaceRoot, err := nativeRequiredWorkspaceRoot(input.WorkspaceRoot)
		if err != nil {
			return toolspkg.ToolResult{}, nativeConfigScopeError(req.ToolID, err)
		}
		workspaceConfig, err := aghconfig.ResolveConfigWriteTarget(
			n.deps.HomePaths,
			workspaceRoot,
			aghconfig.WriteScopeWorkspace,
		)
		if err != nil {
			return toolspkg.ToolResult{}, nativeConfigScopeError(req.ToolID, err)
		}
		workspaceMCP, err := aghconfig.ResolveMCPSidecarWriteTarget(
			n.deps.HomePaths,
			workspaceRoot,
			aghconfig.WriteScopeWorkspace,
		)
		if err != nil {
			return toolspkg.ToolResult{}, nativeConfigScopeError(req.ToolID, err)
		}
		record["workspace_root"] = workspaceRoot
		record["workspace_config"] = workspaceConfig.Path()
		record["workspace_mcp_json"] = workspaceMCP.Path()
		if scope == aghconfig.WriteScopeWorkspace {
			selected = workspaceConfig
			record["selected_config_target"] = selected.Path()
		}
	}
	return structuredResult(record, fmt.Sprint(record["selected_config_target"]))
}

func (n *daemonNativeTools) hooksList(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input hooksListInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	filter, err := hookCatalogFilter(input, scope)
	if err != nil {
		return toolspkg.ToolResult{}, nativeHookValidationError(req.ToolID, err)
	}
	entries, err := n.deps.Observer.QueryHookCatalog(ctx, filter)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := core.HookCatalogPayloadsFromEntries(entries)
	return structuredResult(map[string]any{"hooks": payload}, fmt.Sprintf("%d hooks", len(payload)))
}

func (n *daemonNativeTools) hooksInfo(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input hooksInfoInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	filter, err := hookCatalogFilter(input.hooksListInput, scope)
	if err != nil {
		return toolspkg.ToolResult{}, nativeHookValidationError(req.ToolID, err)
	}
	entries, err := n.deps.Observer.QueryHookCatalog(ctx, filter)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	name := strings.TrimSpace(input.Name)
	for _, entry := range core.HookCatalogPayloadsFromEntries(entries) {
		if entry.Name == name {
			return structuredResult(map[string]any{"hook": entry}, entry.Name)
		}
	}
	return toolspkg.ToolResult{}, toolspkg.NewToolError(
		toolspkg.ErrorCodeNotFound,
		req.ToolID,
		fmt.Sprintf("hook %q not found", name),
		toolspkg.ErrToolNotFound,
		toolspkg.ReasonToolUnknown,
	)
}

func (n *daemonNativeTools) hooksEvents(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input hooksEventsInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	filter := hookspkg.EventFilter{SyncOnly: input.SyncOnly}
	if family := strings.TrimSpace(input.Family); family != "" {
		filter.Family = hookspkg.HookEventFamily(family)
		if err := filter.Family.Validate(); err != nil {
			return toolspkg.ToolResult{}, nativeHookValidationError(req.ToolID, err)
		}
	}
	events, err := n.deps.Observer.QueryHookEvents(ctx, filter)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := core.HookEventPayloadsFromDescriptors(events)
	return structuredResult(map[string]any{"events": payload}, fmt.Sprintf("%d hook events", len(payload)))
}

func (n *daemonNativeTools) hooksRuns(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input hooksRunsInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	query, err := input.query()
	if err != nil {
		return toolspkg.ToolResult{}, nativeHookValidationError(req.ToolID, err)
	}
	runs, err := n.deps.Observer.QueryHookRuns(ctx, query)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := core.HookRunPayloadsFromRecords(runs)
	return structuredResult(map[string]any{"runs": payload}, fmt.Sprintf("%d hook runs", len(payload)))
}

func (n *daemonNativeTools) hooksCreate(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input hookMutationInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	if err := nativeHookMutableSource(req.ToolID, input.Source); err != nil {
		return toolspkg.ToolResult{}, err
	}
	if hookEnvContainsSecret(input.Env) {
		return toolspkg.ToolResult{}, nativeHookSecretError(req.ToolID)
	}
	target, workspaceRoot, err := n.nativeConfigWriteTarget(req.ToolID, input.Scope, input.WorkspaceRoot)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	if existing, err := aghconfig.OverlayHookDeclarations(target); err != nil {
		return toolspkg.ToolResult{}, nativeHookValidationError(req.ToolID, err)
	} else if _, ok := findHookDecl(existing, input.Name); ok {
		return toolspkg.ToolResult{}, nativeHookValidationError(
			req.ToolID,
			fmt.Errorf("hook %q already exists in %s", strings.TrimSpace(input.Name), target.Kind()),
		)
	}
	if err := n.rejectImmutableHookIfPresent(ctx, req.ToolID, input.Name); err != nil {
		return toolspkg.ToolResult{}, err
	}
	decl, err := input.newDecl()
	if err != nil {
		return toolspkg.ToolResult{}, nativeHookValidationError(req.ToolID, err)
	}
	if hookEnvMapContainsSecret(decl.Env) {
		return toolspkg.ToolResult{}, nativeHookSecretError(req.ToolID)
	}
	if _, err := hookspkg.CanonicalizeHookDecl(decl); err != nil {
		return toolspkg.ToolResult{}, nativeHookValidationError(req.ToolID, err)
	}
	if _, err := aghconfig.EditConfigOverlay(
		n.deps.HomePaths,
		workspaceRoot,
		target,
		func(editor *aghconfig.OverlayEditor) error {
			return editor.UpsertArrayTableItem(
				[]string{"hooks", "declarations"},
				"name",
				decl.Name,
				aghconfig.HookDeclarationOverlayValues(decl),
			)
		},
	); err != nil {
		return toolspkg.ToolResult{}, nativeHookValidationError(req.ToolID, err)
	}
	if err := n.syncHookBindings(ctx, req.ToolID); err != nil {
		return toolspkg.ToolResult{}, err
	}
	return nativeHookMutationResult(decl, "created", target)
}

func (n *daemonNativeTools) hooksUpdate(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input hookMutationInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	if err := nativeHookMutableSource(req.ToolID, input.Source); err != nil {
		return toolspkg.ToolResult{}, err
	}
	if hookEnvContainsSecret(input.Env) {
		return toolspkg.ToolResult{}, nativeHookSecretError(req.ToolID)
	}
	target, workspaceRoot, err := n.nativeConfigWriteTarget(req.ToolID, input.Scope, input.WorkspaceRoot)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	decls, err := aghconfig.OverlayHookDeclarations(target)
	if err != nil {
		return toolspkg.ToolResult{}, nativeHookValidationError(req.ToolID, err)
	}
	current, ok := findHookDecl(decls, input.Name)
	if !ok {
		if err := n.rejectImmutableHookIfPresent(ctx, req.ToolID, input.Name); err != nil {
			return toolspkg.ToolResult{}, err
		}
		return toolspkg.ToolResult{}, nativeHookNotFoundError(req.ToolID, input.Name)
	}
	decl, err := input.apply(current)
	if err != nil {
		return toolspkg.ToolResult{}, nativeHookValidationError(req.ToolID, err)
	}
	if hookEnvMapContainsSecret(decl.Env) {
		return toolspkg.ToolResult{}, nativeHookSecretError(req.ToolID)
	}
	if _, err := hookspkg.CanonicalizeHookDecl(decl); err != nil {
		return toolspkg.ToolResult{}, nativeHookValidationError(req.ToolID, err)
	}
	if _, err := aghconfig.EditConfigOverlay(
		n.deps.HomePaths,
		workspaceRoot,
		target,
		func(editor *aghconfig.OverlayEditor) error {
			return editor.UpsertArrayTableItem(
				[]string{"hooks", "declarations"},
				"name",
				decl.Name,
				aghconfig.HookDeclarationOverlayValues(decl),
			)
		},
	); err != nil {
		return toolspkg.ToolResult{}, nativeHookValidationError(req.ToolID, err)
	}
	if err := n.syncHookBindings(ctx, req.ToolID); err != nil {
		return toolspkg.ToolResult{}, err
	}
	return nativeHookMutationResult(decl, "updated", target)
}

func (n *daemonNativeTools) hooksDelete(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input hookNameMutationInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	target, workspaceRoot, err := n.nativeConfigWriteTarget(req.ToolID, input.Scope, input.WorkspaceRoot)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	if deleted, err := n.deleteHookDeclaration(ctx, req.ToolID, target, workspaceRoot, input.Name); err != nil {
		return toolspkg.ToolResult{}, err
	} else if !deleted {
		return toolspkg.ToolResult{}, nativeHookNotFoundError(req.ToolID, input.Name)
	}
	return structuredResult(map[string]any{
		"name":   strings.TrimSpace(input.Name),
		"action": "deleted",
		"scope":  string(target.Scope()),
		"target": target.Path(),
	}, strings.TrimSpace(input.Name))
}

func (n *daemonNativeTools) hooksEnable(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	return n.setHookEnabled(ctx, req, true)
}

func (n *daemonNativeTools) hooksDisable(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	return n.setHookEnabled(ctx, req, false)
}

func (n *daemonNativeTools) loadNativeConfig(workspaceRootRaw string) (aghconfig.Config, string, error) {
	workspaceRoot, err := nativeOptionalWorkspaceRoot(workspaceRootRaw)
	if err != nil {
		return aghconfig.Config{}, "", err
	}
	loadOptions := []aghconfig.LoadOption{}
	if workspaceRoot != "" {
		loadOptions = append(loadOptions, aghconfig.WithWorkspaceRoot(workspaceRoot))
	}
	cfg, err := aghconfig.LoadForHome(n.deps.HomePaths, loadOptions...)
	if err != nil {
		return aghconfig.Config{}, "", err
	}
	return cfg, workspaceRoot, nil
}

func (n *daemonNativeTools) nativeConfigWriteTarget(
	id toolspkg.ToolID,
	scopeRaw string,
	workspaceRootRaw string,
) (aghconfig.WriteTarget, string, error) {
	scope, err := nativeWriteScope(scopeRaw)
	if err != nil {
		return aghconfig.WriteTarget{}, "", nativeConfigScopeError(id, err)
	}
	workspaceRoot := ""
	if scope == aghconfig.WriteScopeWorkspace {
		workspaceRoot, err = nativeRequiredWorkspaceRoot(workspaceRootRaw)
		if err != nil {
			return aghconfig.WriteTarget{}, "", nativeConfigScopeError(id, err)
		}
	}
	target, err := aghconfig.ResolveConfigWriteTarget(n.deps.HomePaths, workspaceRoot, scope)
	if err != nil {
		return aghconfig.WriteTarget{}, "", nativeConfigScopeError(id, err)
	}
	return target, workspaceRoot, nil
}

func nativeConfigPathPolicy(id toolspkg.ToolID, raw string) (aghconfig.PathPolicy, error) {
	path, err := aghconfig.ParseDottedConfigPath(raw)
	if err != nil {
		return aghconfig.PathPolicy{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			id,
			"config path is invalid",
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonConfigPathForbidden,
		)
	}
	policy, err := aghconfig.ClassifyToolConfigPath(path)
	if err != nil {
		return aghconfig.PathPolicy{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			id,
			"config path is invalid",
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonConfigPathForbidden,
		)
	}
	if policy.Denial == aghconfig.ConfigPathAllowed {
		return policy, nil
	}
	return aghconfig.PathPolicy{}, toolspkg.NewToolError(
		toolspkg.ErrorCodeDenied,
		id,
		fmt.Sprintf("config path %q is not mutable by tools", strings.TrimSpace(raw)),
		toolspkg.ErrToolDenied,
		nativeConfigDenialReason(policy.Denial),
	)
}

func nativeConfigDenialReason(denial aghconfig.PathDenial) toolspkg.ReasonCode {
	switch denial {
	case aghconfig.ConfigPathSecretForbidden:
		return toolspkg.ReasonConfigSecretPathForbidden
	case aghconfig.ConfigPathTrustForbidden:
		return toolspkg.ReasonConfigTrustRootForbidden
	default:
		return toolspkg.ReasonConfigPathForbidden
	}
}

func nativeConfigValidationError(id toolspkg.ToolID, err error) error {
	return toolspkg.NewToolError(
		toolspkg.ErrorCodeInvalidInput,
		id,
		"config write validation failed",
		fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
		toolspkg.ReasonConfigValidationFailed,
	)
}

func nativeConfigScopeError(id toolspkg.ToolID, err error) error {
	return toolspkg.NewToolError(
		toolspkg.ErrorCodeDenied,
		id,
		"config scope is not allowed",
		fmt.Errorf("%w: %w", toolspkg.ErrToolDenied, err),
		toolspkg.ReasonConfigScopeNotAllowed,
	)
}

func nativeWriteScope(raw string) (aghconfig.WriteScope, error) {
	scope := aghconfig.WriteScope(strings.ToLower(strings.TrimSpace(raw)))
	if scope == "" {
		scope = aghconfig.WriteScopeGlobal
	}
	if err := scope.Validate(); err != nil {
		return "", err
	}
	return scope, nil
}

func nativeOptionalWorkspaceRoot(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	return aghconfig.ResolvePath(raw)
}

func nativeRequiredWorkspaceRoot(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", errors.New("workspace_root is required for workspace scope")
	}
	return aghconfig.ResolvePath(raw)
}

func nativeScopeForWorkspace(workspaceRoot string) string {
	if strings.TrimSpace(workspaceRoot) == "" {
		return string(aghconfig.WriteScopeGlobal)
	}
	return string(aghconfig.WriteScopeWorkspace)
}

func hookCatalogFilter(input hooksListInput, scope toolspkg.Scope) (hookspkg.CatalogFilter, error) {
	filter := hookspkg.CatalogFilter{
		WorkspaceID:   firstNonEmpty(input.WorkspaceID, scope.WorkspaceID),
		WorkspaceRoot: strings.TrimSpace(input.WorkspaceRoot),
		AgentName:     strings.TrimSpace(input.Agent),
	}
	if event := strings.TrimSpace(input.Event); event != "" {
		parsed := hookspkg.HookEvent(event)
		if err := parsed.Validate(); err != nil {
			return hookspkg.CatalogFilter{}, err
		}
		filter.Event = parsed
	}
	if source := strings.TrimSpace(input.Source); source != "" {
		var parsed hookspkg.HookSource
		if err := parsed.UnmarshalText([]byte(source)); err != nil {
			return hookspkg.CatalogFilter{}, err
		}
		filter.Source = &parsed
	}
	if mode := strings.TrimSpace(input.Mode); mode != "" {
		parsed := hookspkg.HookMode(mode)
		if err := parsed.Validate(); err != nil {
			return hookspkg.CatalogFilter{}, err
		}
		filter.Mode = parsed
	}
	return filter, nil
}

func (n *daemonNativeTools) rejectImmutableHookIfPresent(
	ctx context.Context,
	id toolspkg.ToolID,
	name string,
) error {
	if n.deps.Observer == nil {
		return nil
	}
	entries, err := n.deps.Observer.QueryHookCatalog(ctx, hookspkg.CatalogFilter{})
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(name)
	for _, entry := range entries {
		if entry.Name != trimmed {
			continue
		}
		if entry.Source != hookspkg.HookSourceConfig {
			return toolspkg.NewToolError(
				toolspkg.ErrorCodeDenied,
				id,
				fmt.Sprintf("hook %q source %q is immutable by tools", trimmed, entry.Source.String()),
				toolspkg.ErrToolDenied,
				toolspkg.ReasonHookSourceImmutable,
			)
		}
		return nativeHookValidationError(id, fmt.Errorf("hook %q already exists", trimmed))
	}
	return nil
}

func (n *daemonNativeTools) deleteHookDeclaration(
	ctx context.Context,
	id toolspkg.ToolID,
	target aghconfig.WriteTarget,
	workspaceRoot string,
	name string,
) (bool, error) {
	deleted := false
	if _, err := aghconfig.EditConfigOverlay(
		n.deps.HomePaths,
		workspaceRoot,
		target,
		func(editor *aghconfig.OverlayEditor) error {
			var err error
			deleted, err = editor.DeleteArrayTableItem([]string{"hooks", "declarations"}, "name", name)
			return err
		},
	); err != nil {
		return false, nativeHookValidationError(id, err)
	}
	if !deleted {
		if err := n.rejectImmutableHookIfPresent(ctx, id, name); err != nil {
			return false, err
		}
		return false, nil
	}
	if err := n.syncHookBindings(ctx, id); err != nil {
		return false, err
	}
	return true, nil
}

func (n *daemonNativeTools) setHookEnabled(
	ctx context.Context,
	req toolspkg.CallRequest,
	enabled bool,
) (toolspkg.ToolResult, error) {
	var input hookNameMutationInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	target, workspaceRoot, err := n.nativeConfigWriteTarget(req.ToolID, input.Scope, input.WorkspaceRoot)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	decls, err := aghconfig.OverlayHookDeclarations(target)
	if err != nil {
		return toolspkg.ToolResult{}, nativeHookValidationError(req.ToolID, err)
	}
	decl, ok := findHookDecl(decls, input.Name)
	if !ok {
		if err := n.rejectImmutableHookIfPresent(ctx, req.ToolID, input.Name); err != nil {
			return toolspkg.ToolResult{}, err
		}
		return toolspkg.ToolResult{}, nativeHookNotFoundError(req.ToolID, input.Name)
	}
	decl.Enabled = boolPtr(enabled)
	if hookEnvMapContainsSecret(decl.Env) {
		return toolspkg.ToolResult{}, nativeHookSecretError(req.ToolID)
	}
	if _, err := hookspkg.CanonicalizeHookDecl(decl); err != nil {
		return toolspkg.ToolResult{}, nativeHookValidationError(req.ToolID, err)
	}
	if _, err := aghconfig.EditConfigOverlay(
		n.deps.HomePaths,
		workspaceRoot,
		target,
		func(editor *aghconfig.OverlayEditor) error {
			return editor.UpsertArrayTableItem(
				[]string{"hooks", "declarations"},
				"name",
				decl.Name,
				aghconfig.HookDeclarationOverlayValues(decl),
			)
		},
	); err != nil {
		return toolspkg.ToolResult{}, nativeHookValidationError(req.ToolID, err)
	}
	if err := n.syncHookBindings(ctx, req.ToolID); err != nil {
		return toolspkg.ToolResult{}, err
	}
	action := hookActionDisabled
	if enabled {
		action = "enabled"
	}
	return nativeHookMutationResult(decl, action, target)
}

func (n *daemonNativeTools) syncHookBindings(ctx context.Context, id toolspkg.ToolID) error {
	if n.deps.HookBindings == nil {
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeUnavailable,
			id,
			"hook binding publisher is unavailable",
			toolspkg.ErrToolUnavailable,
			toolspkg.ReasonDependencyMissing,
		)
	}
	if err := n.deps.HookBindings.Sync(ctx); err != nil {
		return nativeHookValidationError(id, err)
	}
	return nil
}

func nativeHookMutableSource(id toolspkg.ToolID, source *string) error {
	if source == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*source)
	if trimmed == "" || trimmed == hookspkg.HookSourceConfig.String() {
		return nil
	}
	return toolspkg.NewToolError(
		toolspkg.ErrorCodeDenied,
		id,
		fmt.Sprintf("hook source %q is immutable by tools", trimmed),
		toolspkg.ErrToolDenied,
		toolspkg.ReasonHookSourceImmutable,
	)
}

func nativeHookSecretError(id toolspkg.ToolID) error {
	return toolspkg.NewToolError(
		toolspkg.ErrorCodeDenied,
		id,
		"hook executor env contains a forbidden secret-bearing input",
		toolspkg.ErrToolDenied,
		toolspkg.ReasonHookSecretInputForbidden,
	)
}

func nativeHookValidationError(id toolspkg.ToolID, err error) error {
	return toolspkg.NewToolError(
		toolspkg.ErrorCodeInvalidInput,
		id,
		"hook declaration validation failed",
		fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
		toolspkg.ReasonHookValidationFailed,
	)
}

func nativeHookNotFoundError(id toolspkg.ToolID, name string) error {
	trimmed := strings.TrimSpace(name)
	return toolspkg.NewToolError(
		toolspkg.ErrorCodeNotFound,
		id,
		fmt.Sprintf("hook %q not found in target config overlay", trimmed),
		toolspkg.ErrToolNotFound,
		toolspkg.ReasonToolUnknown,
	)
}

func nativeHookMutationResult(
	decl hookspkg.HookDecl,
	action string,
	target aghconfig.WriteTarget,
) (toolspkg.ToolResult, error) {
	return structuredResult(map[string]any{
		"name":   decl.Name,
		"action": action,
		"scope":  string(target.Scope()),
		"target": target.Path(),
		"hook":   nativeHookDeclPayload(decl),
	}, decl.Name)
}

func nativeHookDeclPayload(decl hookspkg.HookDecl) map[string]any {
	payload := map[string]any{
		"name":          decl.Name,
		"event":         decl.Event.String(),
		"source":        decl.Source.String(),
		"mode":          string(decl.Mode),
		"required":      decl.Required,
		"priority":      decl.Priority,
		"executor_kind": string(decl.ExecutorKind),
		"command":       decl.Command,
		"args":          append([]string(nil), decl.Args...),
		"env":           cloneStringMap(decl.Env),
		"matcher":       decl.Matcher,
	}
	if decl.Enabled != nil {
		payload["enabled"] = *decl.Enabled
	} else {
		payload["enabled"] = true
	}
	if decl.Timeout > 0 {
		payload["timeout_ms"] = decl.Timeout.Milliseconds()
	}
	return payload
}

func findHookDecl(decls []hookspkg.HookDecl, name string) (hookspkg.HookDecl, bool) {
	trimmed := strings.TrimSpace(name)
	for _, decl := range decls {
		if strings.EqualFold(strings.TrimSpace(decl.Name), trimmed) {
			return decl, true
		}
	}
	return hookspkg.HookDecl{}, false
}

func hookEnvContainsSecret(env *map[string]string) bool {
	if env == nil {
		return false
	}
	return hookEnvMapContainsSecret(*env)
}

func hookEnvMapContainsSecret(env map[string]string) bool {
	for key, value := range env {
		if secretLikeText(key) || secretLikeText(value) {
			return true
		}
	}
	return false
}

func secretLikeText(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false
	}
	needles := []string{"secret", "token", "password", "api_key", "apikey", "authorization", "bearer"}
	for _, needle := range needles {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return strings.HasPrefix(lower, "sk-")
}

func cloneHookMatcher(src hookspkg.HookMatcher) hookspkg.HookMatcher {
	cloned := src
	if src.ToolReadOnly != nil {
		value := *src.ToolReadOnly
		cloned.ToolReadOnly = &value
	}
	if src.Autonomy != nil {
		value := *src.Autonomy
		cloned.Autonomy = &value
	}
	return cloned
}

func boolPtr(value bool) *bool {
	return &value
}

type configReadInput struct {
	WorkspaceRoot string `json:"workspace_root,omitempty"`
}

type configGetInput struct {
	Path          string `json:"path"`
	WorkspaceRoot string `json:"workspace_root,omitempty"`
}

type configSetInput struct {
	Path          string `json:"path"`
	Value         any    `json:"value"`
	Scope         string `json:"scope,omitempty"`
	WorkspaceRoot string `json:"workspace_root,omitempty"`
}

type configUnsetInput struct {
	Path          string `json:"path"`
	Scope         string `json:"scope,omitempty"`
	WorkspaceRoot string `json:"workspace_root,omitempty"`
}

type configPathInput struct {
	Scope         string `json:"scope,omitempty"`
	WorkspaceRoot string `json:"workspace_root,omitempty"`
}

type hooksListInput struct {
	WorkspaceRoot string `json:"workspace_root,omitempty"`
	WorkspaceID   string `json:"workspace_id,omitempty"`
	Agent         string `json:"agent,omitempty"`
	Event         string `json:"event,omitempty"`
	Source        string `json:"source,omitempty"`
	Mode          string `json:"mode,omitempty"`
}

type hooksInfoInput struct {
	hooksListInput
	Name string `json:"name"`
}

type hooksEventsInput struct {
	Family   string `json:"family,omitempty"`
	SyncOnly bool   `json:"sync_only,omitempty"`
}

type hooksRunsInput struct {
	SessionID string `json:"session_id,omitempty"`
	Event     string `json:"event,omitempty"`
	Outcome   string `json:"outcome,omitempty"`
	Since     string `json:"since,omitempty"`
	Last      int    `json:"last,omitempty"`
}

func (i hooksRunsInput) query() (store.HookRunQuery, error) {
	query := store.HookRunQuery{
		SessionID: strings.TrimSpace(i.SessionID),
		Event:     strings.TrimSpace(i.Event),
		Limit:     i.Last,
	}
	if strings.TrimSpace(i.Since) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(i.Since))
		if err != nil {
			return store.HookRunQuery{}, err
		}
		query.Since = parsed
	}
	if outcome := strings.TrimSpace(i.Outcome); outcome != "" {
		query.Outcome = hookspkg.HookRunOutcome(outcome)
		if err := query.Outcome.Validate(); err != nil {
			return store.HookRunQuery{}, err
		}
	}
	if query.Event != "" {
		if err := hookspkg.HookEvent(query.Event).Validate(); err != nil {
			return store.HookRunQuery{}, err
		}
	}
	if err := query.Validate(); err != nil {
		return store.HookRunQuery{}, err
	}
	return query, nil
}

type hookMutationInput struct {
	Name          string                `json:"name"`
	Scope         string                `json:"scope,omitempty"`
	WorkspaceRoot string                `json:"workspace_root,omitempty"`
	Event         *string               `json:"event,omitempty"`
	Mode          *string               `json:"mode,omitempty"`
	Required      *bool                 `json:"required,omitempty"`
	Priority      *int                  `json:"priority,omitempty"`
	Timeout       *string               `json:"timeout,omitempty"`
	Matcher       *hookspkg.HookMatcher `json:"matcher,omitempty"`
	Command       *string               `json:"command,omitempty"`
	Args          *[]string             `json:"args,omitempty"`
	Env           *map[string]string    `json:"env,omitempty"`
	Enabled       *bool                 `json:"enabled,omitempty"`
	Source        *string               `json:"source,omitempty"`
}

func (i hookMutationInput) newDecl() (hookspkg.HookDecl, error) {
	decl := hookspkg.HookDecl{
		Name:   strings.TrimSpace(i.Name),
		Source: hookspkg.HookSourceConfig,
	}
	return i.apply(decl)
}

func (i hookMutationInput) apply(decl hookspkg.HookDecl) (hookspkg.HookDecl, error) {
	decl.Name = strings.TrimSpace(firstNonEmpty(i.Name, decl.Name))
	decl.Source = hookspkg.HookSourceConfig
	if i.Event != nil {
		decl.Event = hookspkg.HookEvent(strings.TrimSpace(*i.Event))
	}
	if i.Mode != nil {
		decl.Mode = hookspkg.HookMode(strings.TrimSpace(*i.Mode))
	}
	if i.Required != nil {
		decl.Required = *i.Required
	}
	if i.Priority != nil {
		decl.Priority = *i.Priority
		decl.PrioritySet = true
	}
	if i.Timeout != nil {
		timeout, err := time.ParseDuration(strings.TrimSpace(*i.Timeout))
		if err != nil {
			return hookspkg.HookDecl{}, err
		}
		decl.Timeout = timeout
	}
	if i.Matcher != nil {
		matcher := cloneHookMatcher(*i.Matcher)
		if hookMatcherHasUnsupportedConfigFields(matcher) {
			return hookspkg.HookDecl{}, errors.New("hook matcher contains fields not supported by config overlays")
		}
		decl.Matcher = matcher
	}
	if i.Command != nil {
		decl.Command = strings.TrimSpace(*i.Command)
	}
	if i.Args != nil {
		decl.Args = append([]string(nil), (*i.Args)...)
	}
	if i.Env != nil {
		decl.Env = cloneStringMap(*i.Env)
	}
	if i.Enabled != nil {
		decl.Enabled = boolPtr(*i.Enabled)
	}
	return decl, nil
}

func hookMatcherHasUnsupportedConfigFields(matcher hookspkg.HookMatcher) bool {
	return strings.TrimSpace(matcher.SandboxID) != "" ||
		strings.TrimSpace(matcher.SandboxBackend) != "" ||
		strings.TrimSpace(matcher.SandboxProfile) != "" ||
		strings.TrimSpace(matcher.SyncDirection) != "" ||
		matcher.Autonomy != nil
}

type hookNameMutationInput struct {
	Name          string `json:"name"`
	Scope         string `json:"scope,omitempty"`
	WorkspaceRoot string `json:"workspace_root,omitempty"`
}
