package core

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/modelcatalog"
	"github.com/gin-gonic/gin"
)

const (
	modelCatalogModelsSegment    = "models"
	modelCatalogProvidersSegment = "providers"
	modelCatalogRefreshSegment   = "refresh"
	modelCatalogSourcesSegment   = "sources"
	modelCatalogStatusSegment    = "status"
)

var errModelCatalogRouteNotFound = errors.New("model catalog route not found")

// ModelCatalogRoute dispatches the native provider model catalog route family.
func (h *BaseHandlers) ModelCatalogRoute(c *gin.Context) {
	if h == nil {
		RespondError(c, http.StatusServiceUnavailable, ErrModelCatalogUnavailable, false)
		return
	}
	parts := modelCatalogPathParts(c.Param("catalog_path"))
	switch c.Request.Method {
	case http.MethodGet:
		h.dispatchModelCatalogGET(c, parts)
	case http.MethodPost:
		h.dispatchModelCatalogPOST(c, parts)
	default:
		RespondError(c, http.StatusNotFound, errModelCatalogRouteNotFound, h.MaskInternalErrors)
	}
}

// OpenAIModels lists catalog models using the OpenAI-compatible shape.
func (h *BaseHandlers) OpenAIModels(c *gin.Context) {
	providerID, err := validateModelCatalogProviderID(c.Query("provider_id"))
	if err != nil {
		RespondOpenAIError(
			c,
			StatusForModelCatalogError(err),
			err,
			h != nil && h.MaskInternalErrors,
		)
		return
	}
	service, err := h.modelCatalogService()
	if err != nil {
		RespondOpenAIError(c, StatusForModelCatalogError(err), err, h != nil && h.MaskInternalErrors)
		return
	}
	models, err := service.ListModels(c.Request.Context(), modelcatalog.ListOptions{
		ProviderID:   providerID,
		IncludeStale: true,
		Now:          h.nowUTC(),
	})
	if err != nil {
		RespondOpenAIError(c, StatusForModelCatalogError(err), err, h.MaskInternalErrors)
		return
	}
	c.JSON(http.StatusOK, OpenAIModelListPayloadFromModels(models))
}

func (h *BaseHandlers) dispatchModelCatalogGET(c *gin.Context, parts []string) {
	switch {
	case len(parts) == 1 && parts[0] == modelCatalogModelsSegment:
		h.listProviderModels(c, "")
	case len(parts) == 2 && parts[0] == modelCatalogSourcesSegment && parts[1] == modelCatalogStatusSegment:
		h.providerModelStatus(c, "")
	case len(parts) == 3 &&
		parts[0] == modelCatalogProvidersSegment &&
		parts[2] == modelCatalogModelsSegment:
		h.listProviderModels(c, parts[1])
	case len(parts) == 4 &&
		parts[0] == modelCatalogProvidersSegment &&
		parts[2] == modelCatalogModelsSegment &&
		parts[3] == modelCatalogStatusSegment:
		h.providerModelStatus(c, parts[1])
	default:
		RespondError(c, http.StatusNotFound, errModelCatalogRouteNotFound, h.MaskInternalErrors)
	}
}

func (h *BaseHandlers) dispatchModelCatalogPOST(c *gin.Context, parts []string) {
	switch {
	case len(parts) == 2 && parts[0] == modelCatalogModelsSegment && parts[1] == modelCatalogRefreshSegment:
		h.refreshProviderModels(c, "")
	case len(parts) == 4 &&
		parts[0] == modelCatalogProvidersSegment &&
		parts[2] == modelCatalogModelsSegment &&
		parts[3] == modelCatalogRefreshSegment:
		h.refreshProviderModels(c, parts[1])
	default:
		RespondError(c, http.StatusNotFound, errModelCatalogRouteNotFound, h.MaskInternalErrors)
	}
}

func (h *BaseHandlers) listProviderModels(c *gin.Context, providerParam string) {
	opts, err := h.modelCatalogListOptions(c, providerParam)
	if err != nil {
		RespondError(c, StatusForModelCatalogError(err), err, h.MaskInternalErrors)
		return
	}
	service, err := h.modelCatalogService()
	if err != nil {
		RespondError(c, StatusForModelCatalogError(err), err, h.MaskInternalErrors)
		return
	}
	models, err := service.ListModels(c.Request.Context(), opts)
	if err != nil {
		RespondError(c, StatusForModelCatalogError(err), err, h.MaskInternalErrors)
		return
	}
	c.JSON(http.StatusOK, ProviderModelListPayloadFromModels(models))
}

func (h *BaseHandlers) refreshProviderModels(c *gin.Context, providerParam string) {
	opts, err := h.modelCatalogRefreshOptions(c, providerParam)
	if err != nil {
		RespondError(c, StatusForModelCatalogError(err), err, h.MaskInternalErrors)
		return
	}
	service, err := h.modelCatalogService()
	if err != nil {
		RespondError(c, StatusForModelCatalogError(err), err, h.MaskInternalErrors)
		return
	}
	statuses, err := service.Refresh(c.Request.Context(), opts)
	payload := contract.ProviderModelRefreshResponse{
		Sources: SourceStatusPayloadsFromStatuses(statuses),
	}
	if err != nil {
		status := StatusForModelCatalogError(err)
		if len(payload.Sources) > 0 {
			payload.Error = modelcatalog.RedactString(err.Error())
			c.JSON(status, payload)
			return
		}
		RespondError(c, status, err, h.MaskInternalErrors)
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *BaseHandlers) providerModelStatus(c *gin.Context, providerParam string) {
	providerID, err := validateModelCatalogProviderID(providerParam)
	if err != nil {
		RespondError(c, StatusForModelCatalogError(err), err, h.MaskInternalErrors)
		return
	}
	service, err := h.modelCatalogService()
	if err != nil {
		RespondError(c, StatusForModelCatalogError(err), err, h.MaskInternalErrors)
		return
	}
	statuses, err := service.ListSourceStatus(c.Request.Context(), providerID)
	if err != nil {
		RespondError(c, StatusForModelCatalogError(err), err, h.MaskInternalErrors)
		return
	}
	c.JSON(http.StatusOK, contract.ProviderModelStatusResponse{
		Sources: SourceStatusPayloadsFromStatuses(statuses),
	})
}

func (h *BaseHandlers) modelCatalogListOptions(
	c *gin.Context,
	providerParam string,
) (modelcatalog.ListOptions, error) {
	providerID := providerParam
	if providerID == "" {
		providerID = c.Query("provider_id")
	}
	trimmedProvider, err := validateModelCatalogProviderID(providerID)
	if err != nil {
		return modelcatalog.ListOptions{}, err
	}
	sourceID, err := validateOptionalModelCatalogSourceID(firstNonEmpty(c.Query("source_id"), c.Query("source")))
	if err != nil {
		return modelcatalog.ListOptions{}, err
	}
	refresh, err := parseBoolQuery(c, "refresh")
	if err != nil {
		return modelcatalog.ListOptions{}, NewModelCatalogValidationError(err)
	}
	includeStale, err := parseBoolQuery(c, "include_stale")
	if err != nil {
		return modelcatalog.ListOptions{}, NewModelCatalogValidationError(err)
	}
	return modelcatalog.ListOptions{
		ProviderID:   trimmedProvider,
		SourceID:     sourceID,
		Refresh:      refresh,
		IncludeStale: includeStale,
		Now:          h.nowUTC(),
	}, nil
}

func (h *BaseHandlers) modelCatalogRefreshOptions(
	c *gin.Context,
	providerParam string,
) (modelcatalog.RefreshOptions, error) {
	providerID, err := validateModelCatalogProviderID(providerParam)
	if err != nil {
		return modelcatalog.RefreshOptions{}, err
	}
	var request contract.ProviderModelRefreshRequest
	if err := bindOptionalModelCatalogRefreshRequest(c, &request); err != nil {
		return modelcatalog.RefreshOptions{}, err
	}
	sourceID, err := validateOptionalModelCatalogSourceID(
		firstNonEmpty(request.SourceID, c.Query("source_id"), c.Query("source")),
	)
	if err != nil {
		return modelcatalog.RefreshOptions{}, err
	}
	return modelcatalog.RefreshOptions{
		ProviderID: providerID,
		SourceID:   sourceID,
		Force:      request.Force,
		RequestID:  strings.TrimSpace(request.RequestID),
		Now:        h.nowUTC(),
	}, nil
}

func (h *BaseHandlers) modelCatalogService() (ModelCatalogService, error) {
	if h == nil || h.ModelCatalog == nil {
		return nil, ErrModelCatalogUnavailable
	}
	return h.ModelCatalog, nil
}

func bindOptionalModelCatalogRefreshRequest(
	c *gin.Context,
	request *contract.ProviderModelRefreshRequest,
) error {
	if c == nil || c.Request == nil || c.Request.Body == nil || c.Request.Body == http.NoBody {
		return nil
	}
	if err := c.ShouldBindJSON(request); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return NewModelCatalogValidationError(fmt.Errorf("invalid refresh request body: %w", err))
	}
	return nil
}

func validateOptionalModelCatalogSourceID(sourceID string) (string, error) {
	trimmed := strings.TrimSpace(sourceID)
	if trimmed == "" {
		return "", nil
	}
	if err := modelcatalog.ValidateSourceID(trimmed); err != nil {
		return "", NewModelCatalogValidationError(err)
	}
	return trimmed, nil
}

func validateModelCatalogProviderID(providerID string) (string, error) {
	trimmed := strings.TrimSpace(providerID)
	if trimmed == "" {
		return "", nil
	}
	for idx, ch := range trimmed {
		valid := ch >= 'a' && ch <= 'z' ||
			ch >= '0' && ch <= '9' ||
			(idx > 0 && (ch == '-' || ch == '_'))
		if !valid {
			return "", NewModelCatalogValidationError(
				fmt.Errorf("provider_id %q must match ^[a-z0-9][a-z0-9_-]*$", providerID),
			)
		}
	}
	return trimmed, nil
}

func modelCatalogPathParts(path string) []string {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	if trimmed == "" {
		return nil
	}
	rawParts := strings.Split(trimmed, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if trimmedPart := strings.TrimSpace(part); trimmedPart != "" {
			parts = append(parts, trimmedPart)
		}
	}
	return parts
}
