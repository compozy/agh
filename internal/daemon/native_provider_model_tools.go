package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	core "github.com/compozy/agh/internal/api/core"
	"github.com/compozy/agh/internal/modelcatalog"
	toolspkg "github.com/compozy/agh/internal/tools"
)

type providerModelsListInput struct {
	ProviderID   string `json:"provider_id,omitempty"`
	SourceID     string `json:"source_id,omitempty"`
	IncludeStale bool   `json:"include_stale,omitempty"`
}

type providerModelsRefreshInput struct {
	ProviderID string `json:"provider_id,omitempty"`
	SourceID   string `json:"source_id,omitempty"`
	Force      bool   `json:"force,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
}

type providerModelsStatusInput struct {
	ProviderID string `json:"provider_id,omitempty"`
}

func (n *daemonNativeTools) providerModelToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDProviderModelsList: {
			call:         n.providerModelsList,
			availability: availability,
		},
		toolspkg.ToolIDProviderModelsRefresh: {
			call:         n.providerModelsRefresh,
			availability: availability,
		},
		toolspkg.ToolIDProviderModelsStatus: {
			call:         n.providerModelsStatus,
			availability: availability,
		},
	}
}

func (n *daemonNativeTools) providerModelsList(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input providerModelsListInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	providerID, err := nativeProviderModelProviderID(req.ToolID, input.ProviderID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	sourceID, err := nativeProviderModelSourceID(req.ToolID, input.SourceID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	models, err := n.deps.ModelCatalog.ListModels(ctx, modelcatalog.ListOptions{
		ProviderID:   providerID,
		SourceID:     sourceID,
		IncludeStale: input.IncludeStale,
		Now:          time.Now().UTC(),
	})
	if err != nil {
		return toolspkg.ToolResult{}, nativeProviderModelToolError(req.ToolID, err)
	}
	payload := core.ProviderModelListPayloadFromModels(models)
	return structuredResult(payload, fmt.Sprintf("%d provider models", len(payload.Models)))
}

func (n *daemonNativeTools) providerModelsRefresh(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input providerModelsRefreshInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	providerID, err := nativeProviderModelProviderID(req.ToolID, input.ProviderID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	sourceID, err := nativeProviderModelSourceID(req.ToolID, input.SourceID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	statuses, err := n.deps.ModelCatalog.Refresh(ctx, modelcatalog.RefreshOptions{
		ProviderID: providerID,
		SourceID:   sourceID,
		Force:      input.Force,
		RequestID:  strings.TrimSpace(input.RequestID),
		Now:        time.Now().UTC(),
	})
	payload := contract.ProviderModelRefreshResponse{
		Sources: core.SourceStatusPayloadsFromStatuses(statuses),
	}
	if err != nil {
		if len(payload.Sources) == 0 {
			return toolspkg.ToolResult{}, nativeProviderModelToolError(req.ToolID, err)
		}
		payload.Error = modelcatalog.RedactString(err.Error())
	}
	return structuredResult(payload, fmt.Sprintf("%d provider model sources", len(payload.Sources)))
}

func (n *daemonNativeTools) providerModelsStatus(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input providerModelsStatusInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	providerID, err := nativeProviderModelProviderID(req.ToolID, input.ProviderID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	statuses, err := n.deps.ModelCatalog.ListSourceStatus(ctx, providerID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeProviderModelToolError(req.ToolID, err)
	}
	payload := contract.ProviderModelStatusResponse{
		Sources: core.SourceStatusPayloadsFromStatuses(statuses),
	}
	return structuredResult(payload, fmt.Sprintf("%d provider model sources", len(payload.Sources)))
}

func nativeProviderModelProviderID(id toolspkg.ToolID, providerID string) (string, error) {
	trimmed := strings.TrimSpace(providerID)
	if trimmed == "" {
		return "", nil
	}
	for idx, ch := range trimmed {
		valid := ch >= 'a' && ch <= 'z' ||
			ch >= '0' && ch <= '9' ||
			(idx > 0 && (ch == '-' || ch == '_'))
		if !valid {
			err := core.NewModelCatalogValidationError(
				fmt.Errorf("provider_id %q must match ^[a-z0-9][a-z0-9_-]*$", providerID),
			)
			return "", nativeProviderModelToolError(id, err)
		}
	}
	return trimmed, nil
}

func nativeProviderModelSourceID(id toolspkg.ToolID, sourceID string) (string, error) {
	trimmed := strings.TrimSpace(sourceID)
	if trimmed == "" {
		return "", nil
	}
	if err := modelcatalog.ValidateSourceID(trimmed); err != nil {
		return "", nativeProviderModelToolError(id, core.NewModelCatalogValidationError(err))
	}
	return trimmed, nil
}

func nativeProviderModelToolError(id toolspkg.ToolID, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, core.ErrModelCatalogValidation),
		errors.Is(err, modelcatalog.ErrSourceNotRegistered):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			id,
			modelcatalog.RedactString(err.Error()),
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonSchemaInvalid,
		)
	case errors.Is(err, core.ErrModelCatalogUnavailable),
		errors.Is(err, modelcatalog.ErrAllSourcesFailed):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeUnavailable,
			id,
			modelcatalog.RedactString(err.Error()),
			fmt.Errorf("%w: %w", toolspkg.ErrToolUnavailable, err),
			toolspkg.ReasonBackendUnhealthy,
		)
	default:
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeBackendFailed,
			id,
			modelcatalog.RedactString(err.Error()),
			err,
			toolspkg.ReasonBackendUnhealthy,
		)
	}
}
