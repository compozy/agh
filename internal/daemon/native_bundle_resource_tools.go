package daemon

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	bundlepkg "github.com/pedronauck/agh/internal/bundles"
	"github.com/pedronauck/agh/internal/resources"
	toolspkg "github.com/pedronauck/agh/internal/tools"
)

type bundleInfoInput struct {
	ID string `json:"id"`
}

type bundleActivateToolInput struct {
	ExtensionName               string `json:"extension_name"`
	BundleName                  string `json:"bundle_name"`
	ProfileName                 string `json:"profile_name"`
	Scope                       string `json:"scope"`
	Workspace                   string `json:"workspace"`
	BindPrimaryChannelAsDefault bool   `json:"bind_primary_channel_as_default"`
}

type resourceFilterInput struct {
	Kind       string `json:"kind"`
	Limit      int    `json:"limit"`
	ScopeKind  string `json:"scope_kind"`
	ScopeID    string `json:"scope_id"`
	OwnerKind  string `json:"owner_kind"`
	OwnerID    string `json:"owner_id"`
	SourceKind string `json:"source_kind"`
	SourceID   string `json:"source_id"`
}

type resourceInfoInput struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func (n *daemonNativeTools) bundleToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDBundlesList:       {call: n.bundlesList, availability: availability},
		toolspkg.ToolIDBundlesInfo:       {call: n.bundlesInfo, availability: availability},
		toolspkg.ToolIDBundlesActivate:   {call: n.bundlesActivate, availability: availability},
		toolspkg.ToolIDBundlesDeactivate: {call: n.bundlesDeactivate, availability: availability},
		toolspkg.ToolIDBundlesStatus:     {call: n.bundlesStatus, availability: availability},
	}
}

func (n *daemonNativeTools) resourceToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDResourcesList:     {call: n.resourcesList, availability: availability},
		toolspkg.ToolIDResourcesInfo:     {call: n.resourcesInfo, availability: availability},
		toolspkg.ToolIDResourcesSnapshot: {call: n.resourcesSnapshot, availability: availability},
	}
}

func (n *daemonNativeTools) bundlesList(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	catalog, err := n.deps.BundleService.Catalog(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, nativeBundleToolError(req.ToolID, err)
	}
	activations, err := n.deps.BundleService.ListActivations(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, nativeBundleToolError(req.ToolID, err)
	}
	payload := map[string]any{
		"bundles":     core.BundleCatalogPayloads(catalog),
		"activations": bundleActivationPayloads(activations),
	}
	return structuredResult(payload, fmt.Sprintf("%d bundles", len(catalog)))
}

func (n *daemonNativeTools) bundlesInfo(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input bundleInfoInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	id, err := requiredNativeString(req.ToolID, "id", input.ID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	activation, err := n.deps.BundleService.GetActivation(ctx, id)
	if err != nil {
		return toolspkg.ToolResult{}, nativeBundleToolError(req.ToolID, err)
	}
	return structuredResult(map[string]any{"activation": core.BundleActivationPayload(activation)}, id)
}

func (n *daemonNativeTools) bundlesActivate(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input bundleActivateToolInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	activation, err := n.deps.BundleService.Activate(ctx, bundlepkg.ActivateRequest{
		ExtensionName:               strings.TrimSpace(input.ExtensionName),
		BundleName:                  strings.TrimSpace(input.BundleName),
		ProfileName:                 strings.TrimSpace(input.ProfileName),
		Scope:                       bundlepkg.Scope(strings.TrimSpace(input.Scope)).Normalize(),
		Workspace:                   strings.TrimSpace(input.Workspace),
		BindPrimaryChannelAsDefault: input.BindPrimaryChannelAsDefault,
	})
	if err != nil {
		return toolspkg.ToolResult{}, nativeBundleToolError(req.ToolID, err)
	}
	return structuredResult(
		map[string]any{"activation": core.BundleActivationPayload(activation)},
		activation.Activation.ID,
	)
}

func (n *daemonNativeTools) bundlesDeactivate(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input bundleInfoInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	id, err := requiredNativeString(req.ToolID, "id", input.ID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	if err := n.deps.BundleService.Deactivate(ctx, id); err != nil {
		return toolspkg.ToolResult{}, nativeBundleToolError(req.ToolID, err)
	}
	return structuredResult(map[string]any{"id": id, "deactivated": true}, id)
}

func (n *daemonNativeTools) bundlesStatus(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	catalog, err := n.deps.BundleService.Catalog(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, nativeBundleToolError(req.ToolID, err)
	}
	activations, err := n.deps.BundleService.ListActivations(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, nativeBundleToolError(req.ToolID, err)
	}
	network, err := n.deps.BundleService.NetworkSettings(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, nativeBundleToolError(req.ToolID, err)
	}
	payload := map[string]any{
		"bundle_count":     len(catalog),
		"activation_count": len(activations),
		nativeToolsNetworkKey: contract.BundleNetworkSettingsPayload{
			ConfiguredDefaultChannel: strings.TrimSpace(network.ConfiguredDefaultChannel),
			EffectiveDefaultChannel:  strings.TrimSpace(network.EffectiveDefaultChannel),
			EffectiveDefaultSource:   strings.TrimSpace(network.EffectiveDefaultSource),
			DeclaredChannels:         core.DeclaredNetworkChannelPayloads(network.DeclaredChannels),
		},
	}
	return structuredResult(payload, fmt.Sprintf("%d active bundles", len(activations)))
}

func (n *daemonNativeTools) resourcesList(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	filter, err := decodeResourceFilterInput(req)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	records, err := n.deps.Resources.List(ctx, filter)
	if err != nil {
		return toolspkg.ToolResult{}, nativeResourceToolError(req.ToolID, err)
	}
	return structuredResult(
		map[string]any{"records": core.ResourceRecordPayloadsFromRaw(records)},
		fmt.Sprintf("%d resources", len(records)),
	)
}

func (n *daemonNativeTools) resourcesInfo(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	kind, id, err := decodeResourceInfoInput(req)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	record, err := n.deps.Resources.Get(ctx, kind, id)
	if err != nil {
		return toolspkg.ToolResult{}, nativeResourceToolError(req.ToolID, err)
	}
	return structuredResult(map[string]any{"record": core.ResourceRecordPayloadFromRaw(record)}, id)
}

func (n *daemonNativeTools) resourcesSnapshot(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	filter, err := decodeResourceFilterInput(req)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	records, err := n.deps.Resources.List(ctx, filter)
	if err != nil {
		return toolspkg.ToolResult{}, nativeResourceToolError(req.ToolID, err)
	}
	payload := map[string]any{
		"count":   len(records),
		"records": core.ResourceRecordPayloadsFromRaw(records),
	}
	return structuredResult(payload, fmt.Sprintf("%d resources", len(records)))
}

func bundleActivationPayloads(items []bundlepkg.ActivationPreview) []contract.BundleActivationPayload {
	if len(items) == 0 {
		return []contract.BundleActivationPayload{}
	}
	payload := make([]contract.BundleActivationPayload, 0, len(items))
	for _, item := range items {
		payload = append(payload, core.BundleActivationPayload(item))
	}
	return payload
}

func decodeResourceFilterInput(req toolspkg.CallRequest) (resources.ResourceFilter, error) {
	var input resourceFilterInput
	if err := decodeNativeInput(req, &input); err != nil {
		return resources.ResourceFilter{}, err
	}
	filter := resources.ResourceFilter{Limit: input.Limit}
	if kind := strings.TrimSpace(input.Kind); kind != "" {
		filter.Kind = resources.ResourceKind(kind)
		if err := filter.Kind.Validate("kind"); err != nil {
			return resources.ResourceFilter{}, nativeResourceToolError(req.ToolID, err)
		}
	}
	if scope, ok, err := resourceScopeFromInput(input.ScopeKind, input.ScopeID, "scope"); err != nil {
		return resources.ResourceFilter{}, nativeResourceToolError(req.ToolID, err)
	} else if ok {
		filter.Scope = &scope
	}
	if owner, ok, err := resourceOwnerFromInput(input.OwnerKind, input.OwnerID, "owner"); err != nil {
		return resources.ResourceFilter{}, nativeResourceToolError(req.ToolID, err)
	} else if ok {
		filter.Owner = &owner
	}
	if source, ok, err := resourceSourceFromInput(input.SourceKind, input.SourceID, "source"); err != nil {
		return resources.ResourceFilter{}, nativeResourceToolError(req.ToolID, err)
	} else if ok {
		filter.Source = &source
	}
	return filter, nil
}

func decodeResourceInfoInput(req toolspkg.CallRequest) (resources.ResourceKind, string, error) {
	var input resourceInfoInput
	if err := decodeNativeInput(req, &input); err != nil {
		return "", "", err
	}
	kindRaw, err := requiredNativeString(req.ToolID, "kind", input.Kind)
	if err != nil {
		return "", "", err
	}
	id, err := requiredNativeString(req.ToolID, "id", input.ID)
	if err != nil {
		return "", "", err
	}
	kind := resources.ResourceKind(kindRaw)
	if err := kind.Validate("kind"); err != nil {
		return "", "", nativeResourceToolError(req.ToolID, err)
	}
	return kind, id, nil
}

func resourceScopeFromInput(rawKind string, rawID string, path string) (resources.ResourceScope, bool, error) {
	if strings.TrimSpace(rawKind) == "" && strings.TrimSpace(rawID) == "" {
		return resources.ResourceScope{}, false, nil
	}
	scope := resources.ResourceScope{Kind: resources.ResourceScopeKind(rawKind), ID: rawID}.Normalize()
	if err := scope.Validate(path); err != nil {
		return resources.ResourceScope{}, false, err
	}
	return scope, true, nil
}

func resourceOwnerFromInput(rawKind string, rawID string, path string) (resources.ResourceOwner, bool, error) {
	if strings.TrimSpace(rawKind) == "" && strings.TrimSpace(rawID) == "" {
		return resources.ResourceOwner{}, false, nil
	}
	owner := resources.ResourceOwner{Kind: resources.ResourceOwnerKind(rawKind), ID: rawID}.Normalize()
	if err := owner.Validate(path); err != nil {
		return resources.ResourceOwner{}, false, err
	}
	return owner, true, nil
}

func resourceSourceFromInput(rawKind string, rawID string, path string) (resources.ResourceSource, bool, error) {
	if strings.TrimSpace(rawKind) == "" && strings.TrimSpace(rawID) == "" {
		return resources.ResourceSource{}, false, nil
	}
	source := resources.ResourceSource{Kind: resources.ResourceSourceKind(rawKind), ID: rawID}.Normalize()
	if err := source.Validate(path); err != nil {
		return resources.ResourceSource{}, false, err
	}
	return source, true, nil
}

func nativeBundleToolError(id toolspkg.ToolID, err error) error {
	return nativeHTTPStatusToolError(id, err, core.StatusForBundleError(err))
}

func nativeResourceToolError(id toolspkg.ToolID, err error) error {
	return nativeHTTPStatusToolError(id, err, core.StatusForResourceError(err))
}

func nativeHTTPStatusToolError(id toolspkg.ToolID, err error, status int) error {
	code := toolspkg.ErrorCodeBackendFailed
	cause := toolspkg.ErrToolBackendFailed
	reason := toolspkg.ReasonBackendUnhealthy
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusRequestEntityTooLarge:
		code = toolspkg.ErrorCodeInvalidInput
		cause = toolspkg.ErrToolInvalidInput
		reason = toolspkg.ReasonConfigValidationFailed
	case http.StatusForbidden:
		code = toolspkg.ErrorCodeDenied
		cause = toolspkg.ErrToolDenied
		reason = toolspkg.ReasonPolicyDenied
	case http.StatusNotFound:
		code = toolspkg.ErrorCodeNotFound
		cause = toolspkg.ErrToolNotFound
		reason = toolspkg.ReasonToolUnknown
	case http.StatusConflict:
		code = toolspkg.ErrorCodeConflict
		cause = toolspkg.ErrToolConflict
		reason = toolspkg.ReasonConflictedID
	case http.StatusServiceUnavailable:
		code = toolspkg.ErrorCodeUnavailable
		cause = toolspkg.ErrToolUnavailable
		reason = toolspkg.ReasonDependencyMissing
	}
	return toolspkg.NewToolError(
		code,
		id,
		err.Error(),
		fmt.Errorf("%w: %w", cause, err),
		reason,
	)
}
