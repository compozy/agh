package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	registrypkg "github.com/pedronauck/agh/internal/registry"
	registrygithub "github.com/pedronauck/agh/internal/registry/github"
	toolspkg "github.com/pedronauck/agh/internal/tools"
)

const (
	defaultExtensionToolSearchLimit = 20
	extensionToolSourceLocal        = "local"
	extensionToolSourceMarketplace  = "marketplace"
	extensionRegistryGitHub         = "github"
)

type extensionMarketplaceSourceLoader func(
	context.Context,
	aghconfig.ExtensionsMarketplaceConfig,
) ([]registrypkg.Source, error)

type extensionSearchInput struct {
	Query  string `json:"query"`
	Source string `json:"source"`
	Limit  int    `json:"limit"`
}

type extensionNameInput struct {
	Name string `json:"name"`
}

type extensionInstallInput struct {
	Source   string `json:"source"`
	Path     string `json:"path"`
	Checksum string `json:"checksum"`
	Slug     string `json:"slug"`
	Registry string `json:"registry"`
	Version  string `json:"version"`
	Asset    string `json:"asset"`
}

type extensionUpdateInput struct {
	Name      string `json:"name"`
	All       bool   `json:"all"`
	CheckOnly bool   `json:"check_only"`
}

func (n *daemonNativeTools) extensionToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDExtensionsSearch: {
			call:         n.extensionSearch,
			availability: availability,
		},
		toolspkg.ToolIDExtensionsList: {
			call:         n.extensionList,
			availability: availability,
		},
		toolspkg.ToolIDExtensionsInfo: {
			call:         n.extensionInfo,
			availability: availability,
		},
		toolspkg.ToolIDExtensionsInstall: {
			call:         n.extensionInstall,
			availability: availability,
		},
		toolspkg.ToolIDExtensionsUpdate: {
			call:         n.extensionUpdate,
			availability: availability,
		},
		toolspkg.ToolIDExtensionsRemove: {
			call:         n.extensionRemove,
			availability: availability,
		},
		toolspkg.ToolIDExtensionsEnable: {
			call:         n.extensionEnable,
			availability: availability,
		},
		toolspkg.ToolIDExtensionsDisable: {
			call:         n.extensionDisable,
			availability: availability,
		},
	}
}

func (n *daemonNativeTools) extensionSearch(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input extensionSearchInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	query, err := requiredNativeString(req.ToolID, "query", input.Query)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	limit := input.Limit
	if limit == 0 {
		limit = defaultExtensionToolSearchLimit
	}
	listings, err := extensionpkg.SearchMarketplaceExtensions(
		ctx,
		n.extensionMarketplaceLoader(),
		query,
		input.Source,
		limit,
	)
	if err != nil {
		return toolspkg.ToolResult{}, nativeExtensionToolError(req.ToolID, err)
	}
	return structuredResult(map[string]any{"extensions": listings}, fmt.Sprintf("%d extension listings", len(listings)))
}

func (n *daemonNativeTools) extensionList(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input struct{}
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	service := n.extensionService()
	items, err := service.List(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, nativeExtensionToolError(req.ToolID, err)
	}
	return structuredResult(map[string]any{"extensions": items}, fmt.Sprintf("%d installed extensions", len(items)))
}

func (n *daemonNativeTools) extensionInfo(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input extensionNameInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	name, err := requiredNativeString(req.ToolID, "name", input.Name)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	item, err := n.extensionService().Status(ctx, name)
	if err != nil {
		return toolspkg.ToolResult{}, nativeExtensionToolError(req.ToolID, err)
	}
	return structuredResult(map[string]any{"extension": item}, item.Name)
}

func (n *daemonNativeTools) extensionInstall(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input extensionInstallInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}

	source, err := input.installSource()
	if err != nil {
		return toolspkg.ToolResult{}, nativeExtensionValidationError(req.ToolID, err)
	}

	switch source {
	case extensionToolSourceLocal:
		item, err := n.extensionInstallLocal(ctx, req.ToolID, input)
		if err != nil {
			return toolspkg.ToolResult{}, nativeExtensionToolError(req.ToolID, err)
		}
		return structuredResult(map[string]any{"extension": item}, item.Name)
	case extensionToolSourceMarketplace:
		item, err := n.extensionInstallMarketplace(ctx, req.ToolID, input)
		if err != nil {
			return toolspkg.ToolResult{}, nativeExtensionToolError(req.ToolID, err)
		}
		return structuredResult(map[string]any{"extension": item}, item.Name)
	default:
		return toolspkg.ToolResult{}, nativeExtensionSourceError(
			req.ToolID,
			fmt.Errorf("unsupported extension install source %q", source),
		)
	}
}

func (n *daemonNativeTools) extensionUpdate(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input extensionUpdateInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	names := []string{}
	if strings.TrimSpace(input.Name) != "" {
		names = append(names, input.Name)
	}
	if input.All && len(names) > 0 {
		return toolspkg.ToolResult{}, nativeExtensionValidationError(
			req.ToolID,
			errors.New("extension update accepts name or all, not both"),
		)
	}

	items, err := extensionpkg.UpdateMarketplaceManaged(
		ctx,
		n.deps.HomePaths,
		n.deps.ExtensionRegistry,
		n.extensionMarketplaceLoader(),
		extensionpkg.MarketplaceUpdateRequest{
			Names:     names,
			All:       input.All,
			CheckOnly: input.CheckOnly,
		},
		n.extensionService().reload,
	)
	if err != nil {
		return toolspkg.ToolResult{}, nativeExtensionToolError(req.ToolID, err)
	}
	return structuredResult(map[string]any{"updates": items}, fmt.Sprintf("%d extension updates", len(items)))
}

func (n *daemonNativeTools) extensionRemove(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input extensionNameInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	name, err := requiredNativeString(req.ToolID, "name", input.Name)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}

	item, err := extensionpkg.RemoveManagedExtension(ctx, n.deps.ExtensionRegistry, name, n.extensionService().reload)
	if err != nil {
		return toolspkg.ToolResult{}, nativeExtensionToolError(req.ToolID, err)
	}
	return structuredResult(map[string]any{"extension": item}, item.Name)
}

func (n *daemonNativeTools) extensionEnable(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input extensionNameInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	name, err := requiredNativeString(req.ToolID, "name", input.Name)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	item, err := n.extensionService().Enable(ctx, name)
	if err != nil {
		return toolspkg.ToolResult{}, nativeExtensionToolError(req.ToolID, err)
	}
	return structuredResult(map[string]any{"extension": item}, item.Name)
}

func (n *daemonNativeTools) extensionDisable(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input extensionNameInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	name, err := requiredNativeString(req.ToolID, "name", input.Name)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	item, err := n.extensionService().Disable(ctx, name)
	if err != nil {
		return toolspkg.ToolResult{}, nativeExtensionToolError(req.ToolID, err)
	}
	return structuredResult(map[string]any{"extension": item}, item.Name)
}

func (n *daemonNativeTools) extensionInstallLocal(
	ctx context.Context,
	toolID toolspkg.ToolID,
	input extensionInstallInput,
) (contract.ExtensionPayload, error) {
	if err := input.validateLocal(); err != nil {
		return contract.ExtensionPayload{}, nativeExtensionValidationError(toolID, err)
	}
	path := strings.TrimSpace(input.Path)
	checksum := strings.TrimSpace(input.Checksum)
	if checksum == "" {
		computed, err := extensionpkg.ComputeDirectoryChecksum(path)
		if err != nil {
			return contract.ExtensionPayload{}, nativeExtensionValidationError(toolID, err)
		}
		checksum = computed
	}
	return n.extensionService().Install(ctx, contract.InstallExtensionRequest{
		Path:     path,
		Checksum: checksum,
	})
}

func (n *daemonNativeTools) extensionInstallMarketplace(
	ctx context.Context,
	toolID toolspkg.ToolID,
	input extensionInstallInput,
) (contract.ExtensionPayload, error) {
	if err := input.validateMarketplace(); err != nil {
		return contract.ExtensionPayload{}, nativeExtensionValidationError(toolID, err)
	}
	service := n.extensionService()
	info, err := extensionpkg.InstallMarketplaceManaged(
		ctx,
		n.deps.HomePaths,
		n.deps.ExtensionRegistry,
		n.extensionMarketplaceLoader(),
		extensionpkg.MarketplaceInstallRequest{
			Slug:         input.Slug,
			SourceFilter: input.Registry,
			Version:      input.Version,
			Asset:        input.Asset,
		},
	)
	if err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := service.reload(ctx); err != nil {
		return contract.ExtensionPayload{}, service.rollbackFailedInstall(ctx, info.Name, err)
	}
	return service.Status(ctx, info.Name)
}

func (n *daemonNativeTools) extensionService() *daemonExtensionService {
	runtime := extensionRuntime(nil)
	if n.deps.ExtensionRuntime != nil {
		runtime = n.deps.ExtensionRuntime()
	}
	service, ok := newDaemonExtensionService(
		n.deps.ExtensionRegistry,
		runtime,
		n.deps.HookBindings,
		n.deps.AgentSkills,
		n.deps.ToolMCP,
		n.deps.Bundles,
		n.deps.HomePaths,
		nil,
		nil,
	).(*daemonExtensionService)
	if !ok {
		return nil
	}
	return service
}

func (n *daemonNativeTools) extensionMarketplaceLoader() extensionpkg.MarketplaceSourceLoader {
	return func(ctx context.Context) ([]registrypkg.Source, error) {
		loader := n.deps.ExtensionSources
		if loader == nil {
			loader = defaultDaemonExtensionMarketplaceSourceLoader
		}
		return loader(ctx, n.deps.ExtensionMarket)
	}
}

func defaultDaemonExtensionMarketplaceSourceLoader(
	_ context.Context,
	cfg aghconfig.ExtensionsMarketplaceConfig,
) ([]registrypkg.Source, error) {
	registryName := strings.ToLower(strings.TrimSpace(cfg.Registry))
	if registryName == "" && strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("daemon: extensions marketplace is not configured")
	}

	switch registryName {
	case extensionRegistryGitHub:
		return []registrypkg.Source{registrygithub.NewClient(cfg.BaseURL)}, nil
	default:
		return nil, fmt.Errorf("daemon: unsupported extension registry %q", cfg.Registry)
	}
}

func (i extensionInstallInput) installSource() (string, error) {
	source := strings.ToLower(strings.TrimSpace(i.Source))
	switch {
	case source != "":
		return source, nil
	case strings.TrimSpace(i.Path) != "":
		return extensionToolSourceLocal, nil
	case strings.TrimSpace(i.Slug) != "":
		return extensionToolSourceMarketplace, nil
	default:
		return "", errors.New("extension install requires either path or slug")
	}
}

func (i extensionInstallInput) validateLocal() error {
	if strings.TrimSpace(i.Path) == "" {
		return errors.New("local extension install requires path")
	}
	if strings.TrimSpace(i.Slug) != "" ||
		strings.TrimSpace(i.Registry) != "" ||
		strings.TrimSpace(i.Version) != "" ||
		strings.TrimSpace(i.Asset) != "" {
		return errors.New("local extension install cannot include marketplace slug, registry, version, or asset")
	}
	return nil
}

func (i extensionInstallInput) validateMarketplace() error {
	if strings.TrimSpace(i.Slug) == "" {
		return errors.New("marketplace extension install requires slug")
	}
	if strings.TrimSpace(i.Path) != "" || strings.TrimSpace(i.Checksum) != "" {
		return errors.New("marketplace extension install cannot include local path or checksum")
	}
	return nil
}

func nativeExtensionToolError(id toolspkg.ToolID, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, extensionpkg.ErrExtensionNotFound):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeNotFound,
			id,
			err.Error(),
			fmt.Errorf("%w: %w", toolspkg.ErrToolNotFound, err),
			toolspkg.ReasonExtensionNotInstalled,
		)
	case errors.Is(err, extensionpkg.ErrExtensionExists),
		errors.Is(err, extensionpkg.ErrExtensionChecksumMismatch):
		return nativeExtensionValidationError(id, err)
	case errors.Is(err, extensionpkg.ErrExtensionHasActiveBundles):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeConflict,
			id,
			err.Error(),
			fmt.Errorf("%w: %w", toolspkg.ErrToolConflict, err),
			toolspkg.ReasonExtensionValidationFailed,
		)
	case isExtensionSourceError(err):
		return nativeExtensionSourceError(id, err)
	default:
		return err
	}
}

func nativeExtensionValidationError(id toolspkg.ToolID, err error) error {
	return toolspkg.NewToolError(
		toolspkg.ErrorCodeInvalidInput,
		id,
		"extension validation failed",
		fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
		toolspkg.ReasonExtensionValidationFailed,
	)
}

func nativeExtensionSourceError(id toolspkg.ToolID, err error) error {
	return toolspkg.NewToolError(
		toolspkg.ErrorCodeDenied,
		id,
		"extension source is not allowed",
		fmt.Errorf("%w: %w", toolspkg.ErrToolDenied, err),
		toolspkg.ReasonExtensionSourceForbidden,
	)
}

func isExtensionSourceError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "marketplace") ||
		strings.Contains(message, "registry source") ||
		strings.Contains(message, "registry is not configured") ||
		strings.Contains(message, "unsupported extension registry")
}
