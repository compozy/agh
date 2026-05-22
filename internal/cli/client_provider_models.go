package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
)

// ProviderModelListQuery captures provider model catalog list filters.
type ProviderModelListQuery struct {
	ProviderID   string
	SourceID     string
	Refresh      bool
	IncludeStale bool
}

// ProviderModelListRecord is the native provider model catalog list response.
type ProviderModelListRecord = contract.ProviderModelListResponse

// ProviderModelRecord is one native provider model catalog projection.
type ProviderModelRecord = contract.ProviderModelPayload

// ProviderModelRefreshRequest captures one provider model catalog refresh request.
type ProviderModelRefreshRequest = contract.ProviderModelRefreshRequest

// ProviderModelRefreshRecord is the native provider model catalog refresh response.
type ProviderModelRefreshRecord = contract.ProviderModelRefreshResponse

// ProviderModelStatusRecord is the native provider model catalog source status response.
type ProviderModelStatusRecord = contract.ProviderModelStatusResponse

// ProviderModelSourceStatusRecord is one provider-scoped source status row.
type ProviderModelSourceStatusRecord = contract.ModelCatalogSourceStatusPayload

func (c *unixSocketClient) ListProviderModels(
	ctx context.Context,
	query ProviderModelListQuery,
) (ProviderModelListRecord, error) {
	path := providerModelsPath(query.ProviderID, "")
	var response ProviderModelListRecord
	values := providerModelListValues(ProviderModelListQuery{
		SourceID:     query.SourceID,
		Refresh:      query.Refresh,
		IncludeStale: query.IncludeStale,
	})
	if err := c.doJSON(ctx, http.MethodGet, path, values, nil, &response); err != nil {
		return ProviderModelListRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) RefreshProviderModels(
	ctx context.Context,
	providerID string,
	request ProviderModelRefreshRequest,
) (ProviderModelRefreshRecord, error) {
	trimmedProvider := strings.TrimSpace(providerID)
	if trimmedProvider == "" {
		return ProviderModelRefreshRecord{}, fmt.Errorf("provider model provider_id is required")
	}
	path := providerModelsPath(trimmedProvider, "refresh")
	var response ProviderModelRefreshRecord
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return ProviderModelRefreshRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ProviderModelStatus(
	ctx context.Context,
	providerID string,
) (ProviderModelStatusRecord, error) {
	trimmedProvider := strings.TrimSpace(providerID)
	if trimmedProvider == "" {
		return ProviderModelStatusRecord{}, fmt.Errorf("provider model provider_id is required")
	}
	path := providerModelsPath(trimmedProvider, "status")
	var response ProviderModelStatusRecord
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return ProviderModelStatusRecord{}, err
	}
	return response, nil
}

func providerModelListValues(query ProviderModelListQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.ProviderID); trimmed != "" {
		values.Set("provider_id", trimmed)
	}
	if trimmed := strings.TrimSpace(query.SourceID); trimmed != "" {
		values.Set("source_id", trimmed)
	}
	if query.Refresh {
		values.Set("refresh", "true")
	}
	if query.IncludeStale {
		values.Set("include_stale", "true")
	}
	return values
}

func providerModelsPath(providerID string, action string) string {
	trimmedProvider := strings.TrimSpace(providerID)
	trimmedAction := strings.Trim(strings.TrimSpace(action), "/")
	if trimmedProvider == "" {
		if trimmedAction == "" {
			return "/api/model-catalog/models"
		}
		if trimmedAction == lifecycleStatusKey {
			return "/api/model-catalog/sources/status"
		}
		return "/api/model-catalog/models/" + url.PathEscape(trimmedAction)
	}
	path := "/api/model-catalog/providers/" + url.PathEscape(trimmedProvider) + "/models"
	if trimmedAction != "" {
		path += "/" + url.PathEscape(trimmedAction)
	}
	return path
}
