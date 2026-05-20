package core

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	automationmodel "github.com/pedronauck/agh/internal/automation/model"
	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/resources"
	settingspkg "github.com/pedronauck/agh/internal/settings"
)

const (
	settingsErrorKey = "error"
)

const (
	settingsRestartStatusPathPrefix  = "/api/settings/actions/restart/"
	settingsObservabilityLogTailPath = "/api/settings/observability/log-tail"
)

var (
	errSettingsServiceUnavailable = errors.New("settings service is not configured")
	errSettingsRestartUnavailable = errors.New("settings restart controller is not configured")
	errSettingsUpdateUnavailable  = errors.New("settings update controller is not configured")
)

// SettingsLogTailEventPayload is the shared SSE payload for daemon log tailing.
type SettingsLogTailEventPayload struct {
	Line string `json:"line"`
}

// GetSettingsGeneral returns the general settings section.
func (h *BaseHandlers) GetSettingsGeneral(c *gin.Context) {
	h.getSettingsSection(c, settingspkg.SectionGeneral)
}

// UpdateSettingsGeneral persists the general settings section.
func (h *BaseHandlers) UpdateSettingsGeneral(c *gin.Context) {
	req, err := parseUpdateSettingsGeneralRequest(c)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	h.updateSettingsSection(c, req)
}

// GetSettingsMemory returns the memory settings section.
func (h *BaseHandlers) GetSettingsMemory(c *gin.Context) {
	h.getSettingsSection(c, settingspkg.SectionMemory)
}

// UpdateSettingsMemory persists the memory settings section.
func (h *BaseHandlers) UpdateSettingsMemory(c *gin.Context) {
	req, err := parseUpdateSettingsMemoryRequest(c)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	if err := h.validateSettingsMemoryProvider(c.Request.Context(), req); err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	h.updateSettingsSection(c, req)
}

// GetSettingsSkills returns the skills settings section.
func (h *BaseHandlers) GetSettingsSkills(c *gin.Context) {
	h.getSettingsSection(c, settingspkg.SectionSkills)
}

// UpdateSettingsSkills persists the skills settings section.
func (h *BaseHandlers) UpdateSettingsSkills(c *gin.Context) {
	req, err := parseUpdateSettingsSkillsRequest(c)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	h.updateSettingsSection(c, req)
}

// GetSettingsAutomation returns the automation settings section.
func (h *BaseHandlers) GetSettingsAutomation(c *gin.Context) {
	h.getSettingsSection(c, settingspkg.SectionAutomation)
}

// UpdateSettingsAutomation persists the automation settings section.
func (h *BaseHandlers) UpdateSettingsAutomation(c *gin.Context) {
	req, err := parseUpdateSettingsAutomationRequest(c)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	h.updateSettingsSection(c, req)
}

// GetSettingsNetwork returns the network settings section.
func (h *BaseHandlers) GetSettingsNetwork(c *gin.Context) {
	h.getSettingsSection(c, settingspkg.SectionNetwork)
}

// UpdateSettingsNetwork persists the network settings section.
func (h *BaseHandlers) UpdateSettingsNetwork(c *gin.Context) {
	req, err := parseUpdateSettingsNetworkRequest(c)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	h.updateSettingsSection(c, req)
}

// GetSettingsObservability returns the observability settings section.
func (h *BaseHandlers) GetSettingsObservability(c *gin.Context) {
	h.getSettingsSection(c, settingspkg.SectionObservability)
}

// UpdateSettingsObservability persists the observability settings section.
func (h *BaseHandlers) UpdateSettingsObservability(c *gin.Context) {
	req, err := parseUpdateSettingsObservabilityRequest(c)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	h.updateSettingsSection(c, req)
}

// GetSettingsHooksExtensions returns the hooks and extensions settings section.
func (h *BaseHandlers) GetSettingsHooksExtensions(c *gin.Context) {
	h.getSettingsSection(c, settingspkg.SectionHooksExtensions)
}

// GetSettingsUpdate returns the current software update status snapshot.
func (h *BaseHandlers) GetSettingsUpdate(c *gin.Context) {
	if h.SettingsUpdate == nil {
		h.respondError(c, http.StatusServiceUnavailable, errSettingsUpdateUnavailable)
		return
	}

	status, err := h.SettingsUpdate.GetUpdate(c.Request.Context())
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}

	c.JSON(http.StatusOK, SettingsUpdateResponseFromStatus(status))
}

// UpdateSettingsHooksExtensions persists the hooks and extensions settings section.
func (h *BaseHandlers) UpdateSettingsHooksExtensions(c *gin.Context) {
	req, err := parseUpdateSettingsHooksExtensionsRequest(c)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	h.updateSettingsSection(c, req)
}

// ListSettingsProviders returns the provider settings collection.
func (h *BaseHandlers) ListSettingsProviders(c *gin.Context) {
	h.listSettingsCollection(c, settingspkg.CollectionProviders)
}

// GetSettingsProvider returns one provider settings item.
func (h *BaseHandlers) GetSettingsProvider(c *gin.Context) {
	h.getSettingsCollectionItem(c, settingspkg.CollectionProviders)
}

// PutSettingsProvider upserts one provider settings item.
func (h *BaseHandlers) PutSettingsProvider(c *gin.Context) {
	req, err := parsePutSettingsProviderRequest(c)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	h.putSettingsCollectionItem(c, req)
}

// DeleteSettingsProvider deletes one provider settings item.
func (h *BaseHandlers) DeleteSettingsProvider(c *gin.Context) {
	req, err := parseDeleteSettingsCollectionRequest(c, settingspkg.CollectionProviders)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	h.deleteSettingsCollectionItem(c, req)
}

// ListSettingsMCPServers returns the MCP server settings collection.
func (h *BaseHandlers) ListSettingsMCPServers(c *gin.Context) {
	h.listSettingsCollection(c, settingspkg.CollectionMCPServers)
}

// PutSettingsMCPServer upserts one MCP server settings item.
func (h *BaseHandlers) PutSettingsMCPServer(c *gin.Context) {
	req, err := parsePutSettingsMCPServerRequest(c)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	h.putSettingsCollectionItem(c, req)
}

// DeleteSettingsMCPServer deletes one MCP server settings item.
func (h *BaseHandlers) DeleteSettingsMCPServer(c *gin.Context) {
	req, err := parseDeleteSettingsCollectionRequest(c, settingspkg.CollectionMCPServers)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	h.deleteSettingsCollectionItem(c, req)
}

// ListSettingsSandboxes returns the sandbox settings collection.
func (h *BaseHandlers) ListSettingsSandboxes(c *gin.Context) {
	h.listSettingsCollection(c, settingspkg.CollectionSandboxes)
}

// GetSettingsSandbox returns one sandbox settings item.
func (h *BaseHandlers) GetSettingsSandbox(c *gin.Context) {
	h.getSettingsCollectionItem(c, settingspkg.CollectionSandboxes)
}

// PutSettingsSandbox upserts one sandbox settings item.
func (h *BaseHandlers) PutSettingsSandbox(c *gin.Context) {
	req, err := parsePutSettingsSandboxRequest(c)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	h.putSettingsCollectionItem(c, req)
}

// DeleteSettingsSandbox deletes one sandbox settings item.
func (h *BaseHandlers) DeleteSettingsSandbox(c *gin.Context) {
	req, err := parseDeleteSettingsCollectionRequest(c, settingspkg.CollectionSandboxes)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	h.deleteSettingsCollectionItem(c, req)
}

// ListSettingsHooks returns the hook settings collection.
func (h *BaseHandlers) ListSettingsHooks(c *gin.Context) {
	h.listSettingsCollection(c, settingspkg.CollectionHooks)
}

// PutSettingsHook upserts one hook settings item.
func (h *BaseHandlers) PutSettingsHook(c *gin.Context) {
	req, err := parsePutSettingsHookRequest(c)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	h.putSettingsCollectionItem(c, req)
}

// DeleteSettingsHook deletes one hook settings item.
func (h *BaseHandlers) DeleteSettingsHook(c *gin.Context) {
	req, err := parseDeleteSettingsCollectionRequest(c, settingspkg.CollectionHooks)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	h.deleteSettingsCollectionItem(c, req)
}

// TriggerSettingsRestart starts the asynchronous daemon restart flow.
func (h *BaseHandlers) TriggerSettingsRestart(c *gin.Context) {
	if h.SettingsRestart == nil {
		h.respondError(c, http.StatusServiceUnavailable, errSettingsRestartUnavailable)
		return
	}

	operation, err := h.SettingsRestart.RequestRestart(c.Request.Context())
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}

	c.JSON(http.StatusAccepted, SettingsRestartActionResponseFromOperation(operation))
}

// GetSettingsRestartStatus returns the persisted restart operation payload.
func (h *BaseHandlers) GetSettingsRestartStatus(c *gin.Context) {
	if h.SettingsRestart == nil {
		h.respondError(c, http.StatusServiceUnavailable, errSettingsRestartUnavailable)
		return
	}

	operationID, err := requiredSettingsPathValue(c.Param("operation_id"), "operation_id")
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}

	operation, err := h.SettingsRestart.GetRestartOperation(c.Request.Context(), operationID)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}

	c.JSON(http.StatusOK, SettingsRestartActionStatusFromOperation(operation))
}

// StreamSettingsObservabilityLogTail streams daemon log lines over SSE.
func (h *BaseHandlers) StreamSettingsObservabilityLogTail(c *gin.Context) {
	logPath := strings.TrimSpace(h.HomePaths.LogFile)
	if logPath == "" {
		h.respondError(c, http.StatusInternalServerError, errors.New("settings log tail file is not configured"))
		return
	}

	file, info, err := openSettingsLogTailFile(logPath)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	defer file.Close()

	writer, err := PrepareSSE(c)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	reader := bufio.NewReader(file)
	var partial string

	ticker := time.NewTicker(settingsLogTailPollInterval(h.PollInterval))
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.StreamDoneChannel():
			return
		case <-ticker.C:
			rotated, rotationErr := settingsLogTailRotated(logPath, info, file)
			if rotationErr != nil {
				h.writeSSEBestEffort(writer, SSEMessage{
					Name: settingsErrorKey,
					Data: ErrorPayloadForError(rotationErr),
				})
				return
			}
			if rotated {
				return
			}
			if drainErr := h.drainSettingsLogTail(writer, reader, &partial); drainErr != nil {
				h.writeSSEBestEffort(writer, SSEMessage{
					Name: settingsErrorKey,
					Data: ErrorPayloadForError(drainErr),
				})
				return
			}
		}
	}
}

func (h *BaseHandlers) getSettingsSection(c *gin.Context, section settingspkg.SectionName) {
	if h.Settings == nil {
		h.respondError(c, http.StatusServiceUnavailable, errSettingsServiceUnavailable)
		return
	}

	req, err := parseSettingsSectionRequest(c, section)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}

	envelope, err := h.Settings.GetSection(c.Request.Context(), req)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}

	payload, err := SettingsSectionResponseFromEnvelope(envelope)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, payload)
}

func (h *BaseHandlers) updateSettingsSection(c *gin.Context, req settingspkg.SectionUpdateRequest) {
	if h.Settings == nil {
		h.respondError(c, http.StatusServiceUnavailable, errSettingsServiceUnavailable)
		return
	}

	result, err := h.Settings.UpdateSection(
		settingspkg.WithMutationSource(c.Request.Context(), h.TransportName),
		req,
	)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}

	payload, err := SettingsSectionMutationResultPayloadFromResult(result)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, payload)
}

func (h *BaseHandlers) listSettingsCollection(c *gin.Context, collection settingspkg.CollectionName) {
	if h.Settings == nil {
		h.respondError(c, http.StatusServiceUnavailable, errSettingsServiceUnavailable)
		return
	}

	req, err := parseSettingsCollectionRequest(c, collection)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}

	envelope, err := h.Settings.ListCollection(c.Request.Context(), req)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}

	payload, err := SettingsCollectionResponseFromEnvelope(envelope)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, payload)
}

func (h *BaseHandlers) getSettingsCollectionItem(c *gin.Context, collection settingspkg.CollectionName) {
	if h.Settings == nil {
		h.respondError(c, http.StatusServiceUnavailable, errSettingsServiceUnavailable)
		return
	}

	req, err := parseSettingsCollectionRequest(c, collection)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}
	name, err := requiredSettingsPathValue(c.Param("name"), "name")
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}

	envelope, err := h.Settings.ListCollection(c.Request.Context(), req)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}

	switch collection {
	case settingspkg.CollectionProviders:
		item, found := findSettingsProvider(envelope.Providers, name)
		if !found {
			notFound := NewSettingsNotFoundError(fmt.Errorf("provider %q not found", name))
			h.respondError(c, StatusForSettingsError(notFound), notFound)
			return
		}
		c.JSON(http.StatusOK, contract.SettingsProviderResponse{Provider: settingsProviderItemPayload(&item)})
	case settingspkg.CollectionSandboxes:
		item, found := findSettingsSandbox(envelope.Sandboxes, name)
		if !found {
			notFound := NewSettingsNotFoundError(fmt.Errorf("sandbox %q not found", name))
			h.respondError(c, StatusForSettingsError(notFound), notFound)
			return
		}
		c.JSON(http.StatusOK, contract.SettingsSandboxResponse{
			Sandbox: contract.SettingsSandboxItemPayload{
				Name:                strings.TrimSpace(item.Name),
				Profile:             settingsSandboxProfilePayload(item.Profile),
				WorkspaceUsageCount: item.WorkspaceUsageCount,
				SourceMetadata:      settingsSourceMetadataPayload(item.SourceMetadata),
			},
		})
	default:
		h.respondError(
			c,
			http.StatusInternalServerError,
			fmt.Errorf("settings item lookup unsupported for %q", collection),
		)
	}
}

func (h *BaseHandlers) putSettingsCollectionItem(c *gin.Context, req settingspkg.CollectionItemPutRequest) {
	if h.Settings == nil {
		h.respondError(c, http.StatusServiceUnavailable, errSettingsServiceUnavailable)
		return
	}

	result, err := h.Settings.PutCollectionItem(
		settingspkg.WithMutationSource(c.Request.Context(), h.TransportName),
		req,
	)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}

	payload, err := SettingsCollectionMutationResultPayloadFromResult(result)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, payload)
}

func (h *BaseHandlers) deleteSettingsCollectionItem(c *gin.Context, req settingspkg.CollectionItemDeleteRequest) {
	if h.Settings == nil {
		h.respondError(c, http.StatusServiceUnavailable, errSettingsServiceUnavailable)
		return
	}

	result, err := h.Settings.DeleteCollectionItem(
		settingspkg.WithMutationSource(c.Request.Context(), h.TransportName),
		req,
	)
	if err != nil {
		h.respondError(c, StatusForSettingsError(err), err)
		return
	}

	payload, err := SettingsCollectionMutationResultPayloadFromResult(result)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, payload)
}

func parseSettingsSectionRequest(
	c *gin.Context,
	section settingspkg.SectionName,
) (settingspkg.SectionRequest, error) {
	scope, workspaceID, err := parseSettingsScope(c.Query("scope"), c.Query("workspace_id"))
	if err != nil {
		return settingspkg.SectionRequest{}, err
	}
	agentName := strings.TrimSpace(c.Query("agent_name"))
	if agentName != "" {
		if section != settingspkg.SectionSkills {
			return settingspkg.SectionRequest{}, NewSettingsValidationError(
				errors.New("agent_name is only supported for skills"),
			)
		}
		if err := aghconfig.ValidateAgentName(agentName); err != nil {
			return settingspkg.SectionRequest{}, NewSettingsValidationError(err)
		}
	}
	return settingspkg.SectionRequest{
		Section:     section,
		Scope:       scope,
		WorkspaceID: workspaceID,
		AgentName:   agentName,
	}, nil
}

func parseSettingsCollectionRequest(
	c *gin.Context,
	collection settingspkg.CollectionName,
) (settingspkg.CollectionRequest, error) {
	scope, workspaceID, err := parseSettingsScope(c.Query("scope"), c.Query("workspace_id"))
	if err != nil {
		return settingspkg.CollectionRequest{}, err
	}
	return settingspkg.CollectionRequest{
		Collection:  collection,
		Scope:       scope,
		WorkspaceID: workspaceID,
	}, nil
}

func parseDeleteSettingsCollectionRequest(
	c *gin.Context,
	collection settingspkg.CollectionName,
) (settingspkg.CollectionItemDeleteRequest, error) {
	req, err := parseSettingsCollectionRequest(c, collection)
	if err != nil {
		return settingspkg.CollectionItemDeleteRequest{}, err
	}
	name, err := requiredSettingsPathValue(c.Param("name"), "name")
	if err != nil {
		return settingspkg.CollectionItemDeleteRequest{}, err
	}
	target, err := parseSettingsTarget(c.Query("target"))
	if err != nil {
		return settingspkg.CollectionItemDeleteRequest{}, err
	}
	return settingspkg.CollectionItemDeleteRequest{
		CollectionRequest: req,
		Name:              name,
		Target:            target,
	}, nil
}

func parseUpdateSettingsGeneralRequest(c *gin.Context) (settingspkg.SectionUpdateRequest, error) {
	var body struct {
		Config *contract.SettingsGeneralConfigPayload `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		return settingspkg.SectionUpdateRequest{}, NewSettingsValidationError(
			fmt.Errorf("decode general settings request: %w", err),
		)
	}
	if body.Config == nil {
		return settingspkg.SectionUpdateRequest{}, NewSettingsValidationError(errors.New("general.config is required"))
	}
	req, err := parseSettingsSectionRequest(c, settingspkg.SectionGeneral)
	if err != nil {
		return settingspkg.SectionUpdateRequest{}, err
	}
	config, err := generalSettingsFromPayload(*body.Config)
	if err != nil {
		return settingspkg.SectionUpdateRequest{}, err
	}
	return settingspkg.SectionUpdateRequest{SectionRequest: req, General: &config}, nil
}

func parseUpdateSettingsMemoryRequest(c *gin.Context) (settingspkg.SectionUpdateRequest, error) {
	var body struct {
		Config *contract.SettingsMemoryConfigPayload `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		return settingspkg.SectionUpdateRequest{}, NewSettingsValidationError(
			fmt.Errorf("decode memory settings request: %w", err),
		)
	}
	if body.Config == nil {
		return settingspkg.SectionUpdateRequest{}, NewSettingsValidationError(errors.New("memory.config is required"))
	}
	req, err := parseSettingsSectionRequest(c, settingspkg.SectionMemory)
	if err != nil {
		return settingspkg.SectionUpdateRequest{}, err
	}
	config, err := memoryConfigFromPayload(body.Config)
	if err != nil {
		return settingspkg.SectionUpdateRequest{}, err
	}
	return settingspkg.SectionUpdateRequest{SectionRequest: req, Memory: &config}, nil
}

func (h *BaseHandlers) validateSettingsMemoryProvider(
	ctx context.Context,
	req settingspkg.SectionUpdateRequest,
) error {
	if req.Memory == nil {
		return nil
	}
	name := strings.TrimSpace(req.Memory.Provider.Name)
	if name == "" || name == memoryLocalProviderName {
		return nil
	}
	if h.MemoryProviders == nil {
		return NewSettingsValidationError(
			fmt.Errorf("memory.config.provider.name %q is not available", name),
		)
	}
	if _, err := h.MemoryProviders.Get(ctx, req.WorkspaceID, name); err != nil {
		if errors.Is(err, extensionpkg.ErrMemoryProviderNotFound) {
			return NewSettingsValidationError(
				fmt.Errorf("memory.config.provider.name %q is not available: %w", name, err),
			)
		}
		return fmt.Errorf("memory.config.provider.name %q lookup failed: %w", name, err)
	}
	return nil
}

func parseUpdateSettingsSkillsRequest(c *gin.Context) (settingspkg.SectionUpdateRequest, error) {
	var body struct {
		Config *contract.SettingsSkillsConfigPayload `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		return settingspkg.SectionUpdateRequest{}, NewSettingsValidationError(
			fmt.Errorf("decode skills settings request: %w", err),
		)
	}
	if body.Config == nil {
		return settingspkg.SectionUpdateRequest{}, NewSettingsValidationError(errors.New("skills.config is required"))
	}
	req, err := parseSettingsSectionRequest(c, settingspkg.SectionSkills)
	if err != nil {
		return settingspkg.SectionUpdateRequest{}, err
	}
	config, err := skillsConfigFromPayload(*body.Config)
	if err != nil {
		return settingspkg.SectionUpdateRequest{}, err
	}
	return settingspkg.SectionUpdateRequest{SectionRequest: req, Skills: &config}, nil
}

func parseUpdateSettingsAutomationRequest(c *gin.Context) (settingspkg.SectionUpdateRequest, error) {
	var body struct {
		Config *contract.SettingsAutomationConfigPayload `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		return settingspkg.SectionUpdateRequest{}, NewSettingsValidationError(
			fmt.Errorf("decode automation settings request: %w", err),
		)
	}
	if body.Config == nil {
		return settingspkg.SectionUpdateRequest{}, NewSettingsValidationError(
			errors.New("automation.config is required"),
		)
	}
	req, err := parseSettingsSectionRequest(c, settingspkg.SectionAutomation)
	if err != nil {
		return settingspkg.SectionUpdateRequest{}, err
	}
	config, err := automationSettingsFromPayload(*body.Config)
	if err != nil {
		return settingspkg.SectionUpdateRequest{}, err
	}
	return settingspkg.SectionUpdateRequest{SectionRequest: req, Automation: &config}, nil
}

func parseUpdateSettingsNetworkRequest(c *gin.Context) (settingspkg.SectionUpdateRequest, error) {
	var body struct {
		Config *contract.SettingsNetworkConfigPayload `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		return settingspkg.SectionUpdateRequest{}, NewSettingsValidationError(
			fmt.Errorf("decode network settings request: %w", err),
		)
	}
	if body.Config == nil {
		return settingspkg.SectionUpdateRequest{}, NewSettingsValidationError(errors.New("network.config is required"))
	}
	req, err := parseSettingsSectionRequest(c, settingspkg.SectionNetwork)
	if err != nil {
		return settingspkg.SectionUpdateRequest{}, err
	}
	config, err := networkConfigFromPayload(*body.Config)
	if err != nil {
		return settingspkg.SectionUpdateRequest{}, err
	}
	return settingspkg.SectionUpdateRequest{SectionRequest: req, Network: &config}, nil
}

func parseUpdateSettingsObservabilityRequest(c *gin.Context) (settingspkg.SectionUpdateRequest, error) {
	var body struct {
		Config *contract.SettingsObservabilityConfigPayload `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		return settingspkg.SectionUpdateRequest{}, NewSettingsValidationError(
			fmt.Errorf("decode observability settings request: %w", err),
		)
	}
	if body.Config == nil {
		return settingspkg.SectionUpdateRequest{}, NewSettingsValidationError(
			errors.New("observability.config is required"),
		)
	}
	req, err := parseSettingsSectionRequest(c, settingspkg.SectionObservability)
	if err != nil {
		return settingspkg.SectionUpdateRequest{}, err
	}
	config, err := observabilityConfigFromPayload(*body.Config)
	if err != nil {
		return settingspkg.SectionUpdateRequest{}, err
	}
	return settingspkg.SectionUpdateRequest{SectionRequest: req, Observability: &config}, nil
}

func parseUpdateSettingsHooksExtensionsRequest(c *gin.Context) (settingspkg.SectionUpdateRequest, error) {
	var body struct {
		Config *contract.SettingsExtensionsConfigPayload `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		return settingspkg.SectionUpdateRequest{}, NewSettingsValidationError(
			fmt.Errorf("decode hooks-extensions settings request: %w", err),
		)
	}
	if body.Config == nil {
		return settingspkg.SectionUpdateRequest{}, NewSettingsValidationError(
			errors.New("hooks-extensions.config is required"),
		)
	}
	req, err := parseSettingsSectionRequest(c, settingspkg.SectionHooksExtensions)
	if err != nil {
		return settingspkg.SectionUpdateRequest{}, err
	}
	config, err := extensionsConfigFromPayload(*body.Config)
	if err != nil {
		return settingspkg.SectionUpdateRequest{}, err
	}
	return settingspkg.SectionUpdateRequest{SectionRequest: req, HooksExtensions: &config}, nil
}

func parsePutSettingsProviderRequest(c *gin.Context) (settingspkg.CollectionItemPutRequest, error) {
	var body contract.PutSettingsProviderRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		return settingspkg.CollectionItemPutRequest{}, NewSettingsValidationError(
			fmt.Errorf("decode provider settings request: %w", err),
		)
	}
	if providerSettingsPayloadEmpty(body.Settings) && len(body.Secrets) == 0 {
		return settingspkg.CollectionItemPutRequest{}, NewSettingsValidationError(
			errors.New("provider.settings is required"),
		)
	}
	req, err := parseSettingsCollectionRequest(c, settingspkg.CollectionProviders)
	if err != nil {
		return settingspkg.CollectionItemPutRequest{}, err
	}
	name, err := requiredSettingsPathValue(c.Param("name"), "name")
	if err != nil {
		return settingspkg.CollectionItemPutRequest{}, err
	}
	settings := settingspkg.ProviderSettings{
		Command:         strings.TrimSpace(body.Settings.Command),
		DisplayName:     strings.TrimSpace(body.Settings.DisplayName),
		Models:          providerModelsFromPayload(body.Settings.Models),
		ModelsSet:       body.Settings.Models != nil,
		Harness:         aghconfig.ProviderHarness(strings.TrimSpace(body.Settings.Harness)),
		RuntimeProvider: strings.TrimSpace(body.Settings.RuntimeProvider),
		Transport:       strings.TrimSpace(body.Settings.Transport),
		BaseURL:         strings.TrimSpace(body.Settings.BaseURL),
		AuthMode:        aghconfig.ProviderAuthMode(strings.TrimSpace(body.Settings.AuthMode)),
		EnvPolicy:       aghconfig.ProviderEnvPolicy(strings.TrimSpace(body.Settings.EnvPolicy)),
		HomePolicy:      aghconfig.ProviderHomePolicy(strings.TrimSpace(body.Settings.HomePolicy)),
		AuthStatusCmd:   strings.TrimSpace(body.Settings.AuthStatusCmd),
		AuthLoginCmd:    strings.TrimSpace(body.Settings.AuthLoginCmd),
		CredentialSlots: providerCredentialSlotsFromPayload(body.Settings.CredentialSlots),
	}
	return settingspkg.CollectionItemPutRequest{
		CollectionRequest: req,
		Name:              name,
		Provider:          &settings,
		ProviderSecrets:   providerSecretWritesFromPayload(body.Secrets),
	}, nil
}

func providerSettingsPayloadEmpty(payload contract.SettingsProviderSettingsPayload) bool {
	return strings.TrimSpace(payload.Command) == "" &&
		strings.TrimSpace(payload.DisplayName) == "" &&
		payload.Models == nil &&
		strings.TrimSpace(payload.Harness) == "" &&
		strings.TrimSpace(payload.RuntimeProvider) == "" &&
		strings.TrimSpace(payload.Transport) == "" &&
		strings.TrimSpace(payload.BaseURL) == "" &&
		strings.TrimSpace(payload.AuthMode) == "" &&
		strings.TrimSpace(payload.EnvPolicy) == "" &&
		strings.TrimSpace(payload.HomePolicy) == "" &&
		strings.TrimSpace(payload.AuthStatusCmd) == "" &&
		strings.TrimSpace(payload.AuthLoginCmd) == "" &&
		len(payload.CredentialSlots) == 0
}

func providerCredentialSlotsFromPayload(
	payloads []contract.SettingsProviderCredentialSlotPayload,
) []aghconfig.ProviderCredentialSlot {
	if len(payloads) == 0 {
		return nil
	}
	slots := make([]aghconfig.ProviderCredentialSlot, 0, len(payloads))
	for _, payload := range payloads {
		slots = append(slots, aghconfig.ProviderCredentialSlot{
			Name:      strings.TrimSpace(payload.Name),
			TargetEnv: strings.TrimSpace(payload.TargetEnv),
			SecretRef: strings.TrimSpace(payload.SecretRef),
			Kind:      strings.TrimSpace(payload.Kind),
			Required:  payload.Required,
		})
	}
	return slots
}

func providerModelsFromPayload(payload *contract.SettingsProviderModelsPayload) aghconfig.ProviderModelsConfig {
	if payload == nil {
		return aghconfig.ProviderModelsConfig{}
	}
	return aghconfig.ProviderModelsConfig{
		Default:   strings.TrimSpace(payload.Default),
		Curated:   providerModelConfigsFromPayload(payload.Curated),
		Discovery: providerModelsDiscoveryFromPayload(payload.Discovery),
	}
}

func providerModelsDiscoveryFromPayload(
	payload *contract.SettingsProviderModelsDiscoveryPayload,
) aghconfig.ProviderModelsDiscoveryConfig {
	if payload == nil {
		return aghconfig.ProviderModelsDiscoveryConfig{}
	}
	return aghconfig.ProviderModelsDiscoveryConfig{
		Enabled:  cloneBoolPtr(payload.Enabled),
		Command:  strings.TrimSpace(payload.Command),
		Endpoint: strings.TrimSpace(payload.Endpoint),
		Timeout:  strings.TrimSpace(payload.Timeout),
	}
}

func providerModelConfigsFromPayload(
	payloads []contract.SettingsProviderModelPayload,
) []aghconfig.ProviderModelConfig {
	if payloads == nil {
		return nil
	}
	models := make([]aghconfig.ProviderModelConfig, 0, len(payloads))
	for _, payload := range payloads {
		models = append(models, aghconfig.ProviderModelConfig{
			ID:                     strings.TrimSpace(payload.ID),
			DisplayName:            strings.TrimSpace(payload.DisplayName),
			ContextWindow:          cloneInt64Ptr(payload.ContextWindow),
			MaxInputTokens:         cloneInt64Ptr(payload.MaxInputTokens),
			MaxOutputTokens:        cloneInt64Ptr(payload.MaxOutputTokens),
			SupportsTools:          cloneBoolPtr(payload.SupportsTools),
			SupportsReasoning:      cloneBoolPtr(payload.SupportsReasoning),
			ReasoningEfforts:       trimStringSliceInternal(payload.ReasoningEfforts),
			DefaultReasoningEffort: strings.TrimSpace(payload.DefaultReasoningEffort),
			CostInputPerMillion:    cloneFloat64Ptr(payload.CostInputPerMillion),
			CostOutputPerMillion:   cloneFloat64Ptr(payload.CostOutputPerMillion),
		})
	}
	return models
}

func cloneFloat64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func providerSecretWritesFromPayload(
	payloads []contract.SettingsProviderSecretWritePayload,
) []settingspkg.ProviderSecretWrite {
	if len(payloads) == 0 {
		return nil
	}
	secrets := make([]settingspkg.ProviderSecretWrite, 0, len(payloads))
	for _, payload := range payloads {
		secrets = append(secrets, settingspkg.ProviderSecretWrite{
			Name:      strings.TrimSpace(payload.Name),
			SecretRef: strings.TrimSpace(payload.SecretRef),
			Kind:      strings.TrimSpace(payload.Kind),
			Value:     payload.Value,
		})
	}
	return secrets
}

func parsePutSettingsMCPServerRequest(c *gin.Context) (settingspkg.CollectionItemPutRequest, error) {
	var body struct {
		Server       *contract.SettingsMCPServerPayload       `json:"server"`
		SecretValues *contract.SettingsMCPSecretValuesPayload `json:"secret_values,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		return settingspkg.CollectionItemPutRequest{}, NewSettingsValidationError(
			fmt.Errorf("decode MCP server settings request: %w", err),
		)
	}
	if body.Server == nil {
		return settingspkg.CollectionItemPutRequest{}, NewSettingsValidationError(
			errors.New("mcp-servers.server is required"),
		)
	}
	req, err := parseSettingsCollectionRequest(c, settingspkg.CollectionMCPServers)
	if err != nil {
		return settingspkg.CollectionItemPutRequest{}, err
	}
	name, err := requiredSettingsPathValue(c.Param("name"), "name")
	if err != nil {
		return settingspkg.CollectionItemPutRequest{}, err
	}
	bodyName := strings.TrimSpace(body.Server.Name)
	if bodyName != "" && bodyName != name {
		return settingspkg.CollectionItemPutRequest{}, NewSettingsValidationError(
			fmt.Errorf("mcp-servers.server.name must match path name %q", name),
		)
	}
	target, err := parseSettingsTarget(c.Query("target"))
	if err != nil {
		return settingspkg.CollectionItemPutRequest{}, err
	}
	server := aghconfig.MCPServer{
		Name:      name,
		Transport: aghconfig.MCPServerTransport(strings.TrimSpace(body.Server.Transport)),
		Command:   strings.TrimSpace(body.Server.Command),
		Args:      cloneStrings(body.Server.Args),
		Env:       cloneStringMap(body.Server.Env),
		SecretEnv: cloneStringMap(body.Server.SecretEnv),
		URL:       strings.TrimSpace(body.Server.URL),
	}
	if body.Server.Auth != nil {
		server.Auth = aghconfig.MCPAuthConfig{
			Type:             aghconfig.MCPAuthType(strings.TrimSpace(body.Server.Auth.Type)),
			IssuerURL:        strings.TrimSpace(body.Server.Auth.IssuerURL),
			MetadataURL:      strings.TrimSpace(body.Server.Auth.MetadataURL),
			AuthorizationURL: strings.TrimSpace(body.Server.Auth.AuthorizationURL),
			TokenURL:         strings.TrimSpace(body.Server.Auth.TokenURL),
			RevocationURL:    strings.TrimSpace(body.Server.Auth.RevocationURL),
			ClientID:         strings.TrimSpace(body.Server.Auth.ClientID),
			ClientSecretRef:  strings.TrimSpace(body.Server.Auth.ClientSecretRef),
			Scopes:           cloneStrings(body.Server.Auth.Scopes),
		}
	}
	if err := server.Validate("server"); err != nil {
		return settingspkg.CollectionItemPutRequest{}, NewSettingsValidationError(err)
	}
	return settingspkg.CollectionItemPutRequest{
		CollectionRequest: req,
		Name:              name,
		Target:            target,
		MCPServer:         &server,
		MCPSecrets:        mcpSecretValuesFromPayload(body.SecretValues),
	}, nil
}

func mcpSecretValuesFromPayload(payload *contract.SettingsMCPSecretValuesPayload) settingspkg.MCPSecretValues {
	if payload == nil {
		return settingspkg.MCPSecretValues{}
	}
	var oauthClientSecret *string
	if payload.OAuthClientSecret != nil {
		value := *payload.OAuthClientSecret
		oauthClientSecret = &value
	}
	return settingspkg.MCPSecretValues{
		SecretEnv:         cloneStringMap(payload.SecretEnv),
		OAuthClientSecret: oauthClientSecret,
	}
}

func parsePutSettingsSandboxRequest(c *gin.Context) (settingspkg.CollectionItemPutRequest, error) {
	var body struct {
		Profile *contract.SettingsSandboxProfilePayload `json:"profile"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		return settingspkg.CollectionItemPutRequest{}, NewSettingsValidationError(
			fmt.Errorf("decode sandbox settings request: %w", err),
		)
	}
	if body.Profile == nil {
		return settingspkg.CollectionItemPutRequest{}, NewSettingsValidationError(
			errors.New("sandboxes.profile is required"),
		)
	}
	req, err := parseSettingsCollectionRequest(c, settingspkg.CollectionSandboxes)
	if err != nil {
		return settingspkg.CollectionItemPutRequest{}, err
	}
	name, err := requiredSettingsPathValue(c.Param("name"), "name")
	if err != nil {
		return settingspkg.CollectionItemPutRequest{}, err
	}
	profile, err := sandboxProfileFromPayload(*body.Profile)
	if err != nil {
		return settingspkg.CollectionItemPutRequest{}, err
	}
	return settingspkg.CollectionItemPutRequest{
		CollectionRequest: req,
		Name:              name,
		Sandbox:           &profile,
	}, nil
}

func parsePutSettingsHookRequest(c *gin.Context) (settingspkg.CollectionItemPutRequest, error) {
	var body struct {
		Declaration *contract.SettingsHookDeclarationPayload `json:"declaration"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		return settingspkg.CollectionItemPutRequest{}, NewSettingsValidationError(
			fmt.Errorf("decode hook settings request: %w", err),
		)
	}
	if body.Declaration == nil {
		return settingspkg.CollectionItemPutRequest{}, NewSettingsValidationError(
			errors.New("hooks.declaration is required"),
		)
	}
	req, err := parseSettingsCollectionRequest(c, settingspkg.CollectionHooks)
	if err != nil {
		return settingspkg.CollectionItemPutRequest{}, err
	}
	name, err := requiredSettingsPathValue(c.Param("name"), "name")
	if err != nil {
		return settingspkg.CollectionItemPutRequest{}, err
	}
	bodyName := strings.TrimSpace(body.Declaration.Name)
	if bodyName != "" && bodyName != name {
		return settingspkg.CollectionItemPutRequest{}, NewSettingsValidationError(
			fmt.Errorf("hooks.declaration.name must match path name %q", name),
		)
	}
	declaration := *body.Declaration
	declaration.Name = name
	decl, err := hookDeclarationFromPayload(declaration)
	if err != nil {
		return settingspkg.CollectionItemPutRequest{}, err
	}
	return settingspkg.CollectionItemPutRequest{
		CollectionRequest: req,
		Name:              name,
		Hook:              &decl,
	}, nil
}

func parseSettingsScope(rawScope string, rawWorkspaceID string) (settingspkg.ScopeKind, string, error) {
	scope := settingspkg.ScopeKind(strings.TrimSpace(rawScope))
	if scope == "" {
		scope = settingspkg.ScopeGlobal
	}
	if err := scope.Validate(); err != nil {
		return "", "", NewSettingsValidationError(err)
	}
	return scope, strings.TrimSpace(rawWorkspaceID), nil
}

func parseSettingsTarget(raw string) (settingspkg.TargetSelector, error) {
	target := settingspkg.TargetSelector(strings.TrimSpace(raw))
	if target == "" {
		target = settingspkg.TargetAuto
	}
	switch target {
	case settingspkg.TargetAuto, settingspkg.TargetConfig, settingspkg.TargetSidecar:
		return target, nil
	default:
		return "", NewSettingsValidationError(
			fmt.Errorf("settings.target must be one of %q, %q, %q", "auto", "config", "sidecar"),
		)
	}
}

func requiredSettingsPathValue(raw string, field string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", NewSettingsValidationError(fmt.Errorf("%s is required", field))
	}
	return value, nil
}

func generalSettingsFromPayload(payload contract.SettingsGeneralConfigPayload) (settingspkg.GeneralSettings, error) {
	sessionTimeout, err := time.ParseDuration(strings.TrimSpace(payload.SessionTimeout))
	if err != nil {
		return settingspkg.GeneralSettings{}, NewSettingsValidationError(
			fmt.Errorf("general.config.session_timeout: %w", err),
		)
	}

	value := settingspkg.GeneralSettings{
		Defaults: aghconfig.DefaultsConfig{
			Agent:    strings.TrimSpace(payload.Defaults.Agent),
			Provider: strings.TrimSpace(payload.Defaults.Provider),
			Sandbox:  strings.TrimSpace(payload.Defaults.Sandbox),
		},
		Limits: aghconfig.LimitsConfig{
			MaxConcurrentAgents: payload.Limits.MaxConcurrentAgents,
		},
		Permissions: aghconfig.PermissionsConfig{
			Mode: aghconfig.PermissionMode(payload.Permissions.Mode),
		},
		SessionTimeout: sessionTimeout,
		HTTP: aghconfig.HTTPConfig{
			Host: strings.TrimSpace(payload.HTTP.Host),
			Port: payload.HTTP.Port,
		},
		Daemon: aghconfig.DaemonConfig{
			Socket: strings.TrimSpace(payload.Daemon.Socket),
		},
	}

	if err := value.Defaults.Validate(); err != nil {
		return settingspkg.GeneralSettings{}, NewSettingsValidationError(err)
	}
	if err := value.Limits.Validate(); err != nil {
		return settingspkg.GeneralSettings{}, NewSettingsValidationError(err)
	}
	if err := value.Permissions.Validate(); err != nil {
		return settingspkg.GeneralSettings{}, NewSettingsValidationError(err)
	}
	if err := (aghconfig.SessionConfig{
		Limits:      aghconfig.SessionLimitsConfig{Timeout: value.SessionTimeout},
		Supervision: aghconfig.DefaultSessionSupervisionConfig(),
	}).Validate(); err != nil {
		return settingspkg.GeneralSettings{}, NewSettingsValidationError(err)
	}
	if err := value.HTTP.Validate(); err != nil {
		return settingspkg.GeneralSettings{}, NewSettingsValidationError(err)
	}
	if err := value.Daemon.Validate(); err != nil {
		return settingspkg.GeneralSettings{}, NewSettingsValidationError(err)
	}

	return value, nil
}

func memoryConfigFromPayload(payload *contract.SettingsMemoryConfigPayload) (aghconfig.MemoryConfig, error) {
	if payload == nil {
		return aghconfig.MemoryConfig{}, NewSettingsValidationError(errors.New("memory.config is required"))
	}
	controller, err := memoryControllerConfigFromPayload(payload.Controller)
	if err != nil {
		return aghconfig.MemoryConfig{}, err
	}
	extractor, err := memoryExtractorConfigFromPayload(payload.Extractor)
	if err != nil {
		return aghconfig.MemoryConfig{}, err
	}
	dream, err := memoryDreamConfigFromPayload(payload.Dream)
	if err != nil {
		return aghconfig.MemoryConfig{}, err
	}
	session, err := memorySessionConfigFromPayload(payload.Session)
	if err != nil {
		return aghconfig.MemoryConfig{}, err
	}
	provider, err := memoryProviderConfigFromPayload(payload.Provider)
	if err != nil {
		return aghconfig.MemoryConfig{}, err
	}

	value := aghconfig.MemoryConfig{
		Enabled:    payload.Enabled,
		GlobalDir:  strings.TrimSpace(payload.GlobalDir),
		Controller: controller,
		Recall:     memoryRecallConfigFromPayload(payload.Recall),
		Decisions:  memoryDecisionsConfigFromPayload(payload.Decisions),
		Extractor:  extractor,
		Dream:      dream,
		Session:    session,
		Daily:      memoryDailyConfigFromPayload(payload.Daily),
		File: aghconfig.MemoryFileConfig{
			MaxLines: payload.File.MaxLines,
			MaxBytes: payload.File.MaxBytes,
		},
		Provider: provider,
		Workspace: aghconfig.MemoryWorkspaceConfig{
			TOMLPath:   strings.TrimSpace(payload.Workspace.TOMLPath),
			AutoCreate: payload.Workspace.AutoCreate,
		},
	}
	if err := value.Validate(); err != nil {
		return aghconfig.MemoryConfig{}, NewSettingsValidationError(err)
	}
	return value, nil
}

func memoryControllerConfigFromPayload(
	payload contract.SettingsMemoryControllerPayload,
) (aghconfig.MemoryControllerConfig, error) {
	maxLatency, err := parseSettingsDuration("memory.config.controller.max_latency", payload.MaxLatency)
	if err != nil {
		return aghconfig.MemoryControllerConfig{}, err
	}
	timeout, err := parseSettingsDuration("memory.config.controller.llm.timeout", payload.LLM.Timeout)
	if err != nil {
		return aghconfig.MemoryControllerConfig{}, err
	}
	return aghconfig.MemoryControllerConfig{
		Mode:            strings.TrimSpace(payload.Mode),
		MaxLatency:      maxLatency,
		DefaultOpOnFail: strings.TrimSpace(payload.DefaultOpOnFail),
		LLM: aghconfig.MemoryControllerLLMConfig{
			Enabled:       payload.LLM.Enabled,
			Model:         strings.TrimSpace(payload.LLM.Model),
			TopK:          payload.LLM.TopK,
			PromptVersion: strings.TrimSpace(payload.LLM.PromptVersion),
			Timeout:       timeout,
			MaxTokensOut:  payload.LLM.MaxTokensOut,
		},
		Policy: aghconfig.MemoryControllerPolicyConfig{
			MaxContentChars: payload.Policy.MaxContentChars,
			MaxWritesPerMin: payload.Policy.MaxWritesPerMin,
			AllowOrigins:    cloneStrings(payload.Policy.AllowOrigins),
		},
	}, nil
}

func memoryRecallConfigFromPayload(payload contract.SettingsMemoryRecallPayload) aghconfig.MemoryRecallConfig {
	return aghconfig.MemoryRecallConfig{
		TopK:                   payload.TopK,
		RawCandidates:          payload.RawCandidates,
		Fusion:                 strings.TrimSpace(payload.Fusion),
		IncludeAlreadySurfaced: payload.IncludeAlreadySurfaced,
		IncludeSystem:          payload.IncludeSystem,
		Weights: aghconfig.MemoryRecallWeightsConfig{
			BM25Unicode:  payload.Weights.BM25Unicode,
			BM25Trigram:  payload.Weights.BM25Trigram,
			Recency:      payload.Weights.Recency,
			RecallSignal: payload.Weights.RecallSignal,
		},
		Freshness: aghconfig.MemoryRecallFreshnessConfig{
			BannerAfterDays: payload.Freshness.BannerAfterDays,
		},
		Signals: aghconfig.MemoryRecallSignalsConfig{
			QueueCapacity:  payload.Signals.QueueCapacity,
			WorkerRetryMax: payload.Signals.WorkerRetryMax,
			MetricsEnabled: payload.Signals.MetricsEnabled,
		},
	}
}

func memoryDecisionsConfigFromPayload(payload contract.SettingsMemoryDecisionsPayload) aghconfig.MemoryDecisionsConfig {
	return aghconfig.MemoryDecisionsConfig{
		PruneAfterAppliedDays: payload.PruneAfterAppliedDays,
		KeepAuditSummary:      payload.KeepAuditSummary,
		MaxPostContentBytes:   payload.MaxPostContentBytes,
	}
}

func memoryExtractorConfigFromPayload(
	payload contract.SettingsMemoryExtractorPayload,
) (aghconfig.MemoryExtractorConfig, error) {
	deadline, err := parseSettingsDuration("memory.config.extractor.deadline", payload.Deadline)
	if err != nil {
		return aghconfig.MemoryExtractorConfig{}, err
	}
	return aghconfig.MemoryExtractorConfig{
		Enabled:          payload.Enabled,
		Mode:             strings.TrimSpace(payload.Mode),
		ThrottleTurns:    payload.ThrottleTurns,
		Deadline:         deadline,
		SandboxInboxOnly: payload.SandboxInboxOnly,
		InboxPath:        strings.TrimSpace(payload.InboxPath),
		DLQPath:          strings.TrimSpace(payload.DLQPath),
		Model:            strings.TrimSpace(payload.Model),
		Queue: aghconfig.MemoryExtractorQueueConfig{
			Capacity:    payload.Queue.Capacity,
			CoalesceMax: payload.Queue.CoalesceMax,
		},
	}, nil
}

func memoryDreamConfigFromPayload(payload contract.SettingsMemoryDreamPayload) (aghconfig.DreamConfig, error) {
	debounce, err := parseSettingsDuration("memory.config.dream.debounce", payload.Debounce)
	if err != nil {
		return aghconfig.DreamConfig{}, err
	}
	checkInterval, err := parseSettingsDuration("memory.config.dream.check_interval", payload.CheckInterval)
	if err != nil {
		return aghconfig.DreamConfig{}, err
	}
	return aghconfig.DreamConfig{
		Enabled:       payload.Enabled,
		Agent:         strings.TrimSpace(payload.Agent),
		MinHours:      payload.MinHours,
		MinSessions:   payload.MinSessions,
		Debounce:      debounce,
		PromptVersion: strings.TrimSpace(payload.PromptVersion),
		CheckInterval: checkInterval,
		Gates: aghconfig.MemoryDreamGatesConfig{
			MinUnpromoted:  payload.Gates.MinUnpromoted,
			MinRecallCount: payload.Gates.MinRecallCount,
			MinScore:       payload.Gates.MinScore,
		},
		Scoring: memoryDreamScoringConfigFromPayload(payload.Scoring),
	}, nil
}

func memoryDreamScoringConfigFromPayload(
	payload contract.SettingsMemoryDreamScoringPayload,
) aghconfig.MemoryDreamScoringConfig {
	return aghconfig.MemoryDreamScoringConfig{
		RecencyHalfLifeDays: payload.RecencyHalfLifeDays,
		Weights: aghconfig.MemoryDreamScoringWeightsConfig{
			Frequency: payload.Weights.Frequency,
			Relevance: payload.Weights.Relevance,
			Recency:   payload.Weights.Recency,
			Freshness: payload.Weights.Freshness,
		},
	}
}

func memorySessionConfigFromPayload(
	payload contract.SettingsMemorySessionPayload,
) (aghconfig.MemorySessionConfig, error) {
	eventsPurgeGrace, err := parseSettingsDuration(
		"memory.config.session.events_purge_grace",
		payload.EventsPurgeGrace,
	)
	if err != nil {
		return aghconfig.MemorySessionConfig{}, err
	}
	return aghconfig.MemorySessionConfig{
		LedgerFormat:     strings.TrimSpace(payload.LedgerFormat),
		LedgerRoot:       strings.TrimSpace(payload.LedgerRoot),
		EventsPurgeGrace: eventsPurgeGrace,
		ColdArchiveDays:  payload.ColdArchiveDays,
		HardDeleteDays:   payload.HardDeleteDays,
		MaxArchiveBytes:  payload.MaxArchiveBytes,
		UnboundPartition: strings.TrimSpace(payload.UnboundPartition),
	}, nil
}

func memoryDailyConfigFromPayload(payload contract.SettingsMemoryDailyPayload) aghconfig.MemoryDailyConfig {
	return aghconfig.MemoryDailyConfig{
		MaxBytes:        payload.MaxBytes,
		MaxLines:        payload.MaxLines,
		RotateFormat:    strings.TrimSpace(payload.RotateFormat),
		DreamingWindow:  payload.DreamingWindow,
		ColdArchiveDays: payload.ColdArchiveDays,
		HardDeleteDays:  payload.HardDeleteDays,
		MaxArchiveBytes: payload.MaxArchiveBytes,
		SweepHour:       payload.SweepHour,
		ArchivePath:     strings.TrimSpace(payload.ArchivePath),
	}
}

func memoryProviderConfigFromPayload(
	payload contract.SettingsMemoryProviderPayload,
) (aghconfig.MemoryProviderConfig, error) {
	timeout, err := parseSettingsDuration("memory.config.provider.timeout", payload.Timeout)
	if err != nil {
		return aghconfig.MemoryProviderConfig{}, err
	}
	cooldown, err := parseSettingsDuration("memory.config.provider.cooldown", payload.Cooldown)
	if err != nil {
		return aghconfig.MemoryProviderConfig{}, err
	}
	return aghconfig.MemoryProviderConfig{
		Name:             strings.TrimSpace(payload.Name),
		Timeout:          timeout,
		FailureThreshold: payload.FailureThreshold,
		Cooldown:         cooldown,
	}, nil
}

func parseSettingsDuration(path string, value string) (time.Duration, error) {
	duration, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return 0, NewSettingsValidationError(fmt.Errorf("%s: %w", path, err))
	}
	return duration, nil
}

func skillsConfigFromPayload(payload contract.SettingsSkillsConfigPayload) (aghconfig.SkillsConfig, error) {
	pollInterval, err := time.ParseDuration(strings.TrimSpace(payload.PollInterval))
	if err != nil {
		return aghconfig.SkillsConfig{}, NewSettingsValidationError(
			fmt.Errorf("skills.config.poll_interval: %w", err),
		)
	}

	value := aghconfig.SkillsConfig{
		Enabled:                 payload.Enabled,
		DisabledSkills:          cloneStrings(payload.DisabledSkills),
		PollInterval:            pollInterval,
		AllowedMarketplaceMCP:   cloneStrings(payload.AllowedMarketplaceMCP),
		AllowedMarketplaceHooks: cloneStrings(payload.AllowedMarketplaceHooks),
		Marketplace: aghconfig.MarketplaceConfig{
			Registry: strings.TrimSpace(payload.Marketplace.Registry),
			BaseURL:  strings.TrimSpace(payload.Marketplace.BaseURL),
		},
	}
	if err := value.Validate(); err != nil {
		return aghconfig.SkillsConfig{}, NewSettingsValidationError(err)
	}
	return value, nil
}

func automationSettingsFromPayload(
	payload contract.SettingsAutomationConfigPayload,
) (settingspkg.AutomationSettings, error) {
	config := aghconfig.AutomationConfig{
		Enabled:           payload.Enabled,
		Timezone:          strings.TrimSpace(payload.Timezone),
		MaxConcurrentJobs: payload.MaxConcurrentJobs,
		DefaultFireLimit:  payload.DefaultFireLimit,
	}
	if err := config.Validate(); err != nil {
		return settingspkg.AutomationSettings{}, NewSettingsValidationError(err)
	}
	return settingspkg.AutomationSettings{
		Enabled:           config.Enabled,
		Timezone:          config.Timezone,
		MaxConcurrentJobs: config.MaxConcurrentJobs,
		DefaultFireLimit:  config.DefaultFireLimit,
	}, nil
}

func networkConfigFromPayload(payload contract.SettingsNetworkConfigPayload) (aghconfig.NetworkConfig, error) {
	value := aghconfig.NetworkConfig{
		Enabled:        payload.Enabled,
		DefaultChannel: strings.TrimSpace(payload.DefaultChannel),
		Port:           payload.Port,
		MaxPayload:     payload.MaxPayload,
		GreetInterval:  payload.GreetInterval,
		MaxReplayAge:   payload.MaxReplayAge,
		MaxQueueDepth:  payload.MaxQueueDepth,
	}
	if err := value.Validate(); err != nil {
		return aghconfig.NetworkConfig{}, NewSettingsValidationError(err)
	}
	return value, nil
}

func observabilityConfigFromPayload(
	payload contract.SettingsObservabilityConfigPayload,
) (aghconfig.ObservabilityConfig, error) {
	value := aghconfig.ObservabilityConfig{
		Enabled:        payload.Enabled,
		RetentionDays:  payload.RetentionDays,
		MaxGlobalBytes: payload.MaxGlobalBytes,
		Transcripts: aghconfig.ObservabilityTranscriptConfig{
			Enabled:            payload.Transcripts.Enabled,
			SegmentBytes:       payload.Transcripts.SegmentBytes,
			MaxBytesPerSession: payload.Transcripts.MaxBytesPerSession,
		},
	}
	if err := value.Validate(); err != nil {
		return aghconfig.ObservabilityConfig{}, NewSettingsValidationError(err)
	}
	return value, nil
}

func extensionsConfigFromPayload(
	payload contract.SettingsExtensionsConfigPayload,
) (aghconfig.ExtensionsConfig, error) {
	snapshotRateLimit, err := extensionRateLimitConfigFromPayload(
		payload.Resources.SnapshotRateLimit,
		"hooks-extensions.config.resources.snapshot_rate_limit",
	)
	if err != nil {
		return aghconfig.ExtensionsConfig{}, err
	}
	operatorWriteRateLimit, err := extensionRateLimitConfigFromPayload(
		payload.Resources.OperatorWriteRateLimit,
		"hooks-extensions.config.resources.operator_write_rate_limit",
	)
	if err != nil {
		return aghconfig.ExtensionsConfig{}, err
	}

	allowedKinds := make([]resources.ResourceKind, 0, len(payload.Resources.AllowedKinds))
	for _, value := range payload.Resources.AllowedKinds {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			allowedKinds = append(allowedKinds, resources.ResourceKind(trimmed))
		}
	}

	value := aghconfig.ExtensionsConfig{
		Marketplace: aghconfig.ExtensionsMarketplaceConfig{
			Registry: strings.TrimSpace(payload.Marketplace.Registry),
			BaseURL:  strings.TrimSpace(payload.Marketplace.BaseURL),
		},
		Resources: aghconfig.ExtensionsResourcesConfig{
			AllowedKinds:           allowedKinds,
			MaxScope:               payload.Resources.MaxScope,
			SnapshotRateLimit:      snapshotRateLimit,
			OperatorWriteRateLimit: operatorWriteRateLimit,
		},
	}
	if err := value.Validate(); err != nil {
		return aghconfig.ExtensionsConfig{}, NewSettingsValidationError(err)
	}
	return value, nil
}

func extensionRateLimitConfigFromPayload(
	payload contract.SettingsExtensionRateLimitPayload,
	path string,
) (aghconfig.ExtensionsResourceRateLimitConfig, error) {
	window, err := time.ParseDuration(strings.TrimSpace(payload.Window))
	if err != nil && strings.TrimSpace(payload.Window) != "" {
		return aghconfig.ExtensionsResourceRateLimitConfig{}, NewSettingsValidationError(
			fmt.Errorf("%s.window: %w", path, err),
		)
	}
	return aghconfig.ExtensionsResourceRateLimitConfig{
		Requests: payload.Requests,
		Window:   window,
		Queue:    payload.Queue,
	}, nil
}

func sandboxProfileFromPayload(
	payload contract.SettingsSandboxProfilePayload,
) (aghconfig.SandboxProfile, error) {
	value := aghconfig.SandboxProfile{
		Backend:     strings.TrimSpace(payload.Backend),
		SyncMode:    strings.TrimSpace(payload.SyncMode),
		Persistence: strings.TrimSpace(payload.Persistence),
		RuntimeRoot: strings.TrimSpace(payload.RuntimeRoot),
		Env:         cloneStringMap(payload.Env),
		SecretEnv:   cloneStringMap(payload.SecretEnv),
	}
	if payload.Network != nil {
		value.Network = aghconfig.NetworkProfile{
			AllowPublicIngress: payload.Network.AllowPublicIngress,
			AllowOutbound:      payload.Network.AllowOutbound,
			AllowList:          cloneStrings(payload.Network.AllowList),
			DenyList:           cloneStrings(payload.Network.DenyList),
			Required:           payload.Network.Required,
		}
	}
	if payload.Daytona != nil {
		value.Daytona = aghconfig.DaytonaProfile{
			APIURL:      strings.TrimSpace(payload.Daytona.APIURL),
			Target:      strings.TrimSpace(payload.Daytona.Target),
			Image:       strings.TrimSpace(payload.Daytona.Image),
			Snapshot:    strings.TrimSpace(payload.Daytona.Snapshot),
			Class:       strings.TrimSpace(payload.Daytona.Class),
			AutoStop:    strings.TrimSpace(payload.Daytona.AutoStop),
			AutoArchive: strings.TrimSpace(payload.Daytona.AutoArchive),
		}
	}
	if err := value.Validate("sandbox.profile"); err != nil {
		return aghconfig.SandboxProfile{}, NewSettingsValidationError(err)
	}
	return value, nil
}

func hookDeclarationFromPayload(
	payload contract.SettingsHookDeclarationPayload,
) (hookspkg.HookDecl, error) {
	timeout, err := parseOptionalDuration(payload.Timeout, "hooks.declaration.timeout")
	if err != nil {
		return hookspkg.HookDecl{}, err
	}
	priority, err := hookspkg.PriorityFromInt(payload.Priority)
	if err != nil {
		return hookspkg.HookDecl{}, err
	}

	value := hookspkg.HookDecl{
		Name:         strings.TrimSpace(payload.Name),
		Event:        payload.Event,
		Source:       hookspkg.HookSourceConfig,
		Mode:         payload.Mode,
		Required:     payload.Required,
		Priority:     priority,
		PrioritySet:  payload.Priority != 0,
		Timeout:      timeout,
		Matcher:      payload.Matcher,
		ExecutorKind: payload.ExecutorKind,
		Command:      strings.TrimSpace(payload.Command),
		Args:         cloneStrings(payload.Args),
		Env:          cloneStringMap(payload.Env),
		SecretEnv:    cloneStringMap(payload.SecretEnv),
		Metadata:     cloneStringMap(payload.Metadata),
	}
	if err := hookspkg.ValidateHookDecl(value); err != nil {
		return hookspkg.HookDecl{}, NewSettingsValidationError(err)
	}
	return value, nil
}

func parseOptionalDuration(raw string, path string) (time.Duration, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, NewSettingsValidationError(fmt.Errorf("%s: %w", path, err))
	}
	return duration, nil
}

func settingsRestartStatusURL(operationID string) string {
	return settingsRestartStatusPathPrefix + strings.TrimSpace(operationID)
}

func openSettingsLogTailFile(path string) (*os.File, os.FileInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open settings log tail file %q: %w", path, err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("stat settings log tail file %q: %w", path, err)
	}
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("seek settings log tail file %q: %w", path, err)
	}
	return file, info, nil
}

func settingsLogTailPollInterval(interval time.Duration) time.Duration {
	if interval > 0 {
		return interval
	}
	return defaultPollInterval
}

func settingsLogTailRotated(path string, initial os.FileInfo, file *os.File) (bool, error) {
	current, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, fmt.Errorf("stat settings log tail file %q: %w", path, err)
	}
	if initial != nil && !os.SameFile(initial, current) {
		return true, nil
	}
	if file != nil {
		offset, seekErr := file.Seek(0, io.SeekCurrent)
		if seekErr != nil {
			return false, fmt.Errorf("read settings log tail offset: %w", seekErr)
		}
		if current.Size() < offset {
			return true, nil
		}
	}
	return false, nil
}

func (h *BaseHandlers) drainSettingsLogTail(
	writer FlushWriter,
	reader *bufio.Reader,
	partial *string,
) error {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				if partial != nil {
					*partial += line
				}
				return nil
			}
			return fmt.Errorf("read settings log tail: %w", err)
		}
		if partial != nil {
			line = *partial + line
			*partial = ""
		}
		h.writeSSEBestEffort(writer, SSEMessage{
			Name: "log",
			Data: SettingsLogTailEventPayload{Line: strings.TrimRight(line, "\r\n")},
		})
	}
}

func findSettingsProvider(values []settingspkg.ProviderItem, name string) (settingspkg.ProviderItem, bool) {
	for idx := range values {
		value := &values[idx]
		if strings.TrimSpace(value.Name) == name {
			return *value, true
		}
	}
	return settingspkg.ProviderItem{}, false
}

func findSettingsSandbox(values []settingspkg.SandboxItem, name string) (settingspkg.SandboxItem, bool) {
	for _, value := range values {
		if strings.TrimSpace(value.Name) == name {
			return value, true
		}
	}
	return settingspkg.SandboxItem{}, false
}

func automationFireLimitFromPayload(
	payload automationmodel.FireLimitConfig,
) automationmodel.FireLimitConfig {
	return payload
}
