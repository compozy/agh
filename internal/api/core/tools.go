package core

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	toolspkg "github.com/pedronauck/agh/internal/tools"
)

const (
	toolInvokeStatusCompleted = "completed"
	toolsetStatusExpanded     = "expanded"
	toolsetStatusDegraded     = "degraded"
)

// ListTools returns the operator-visible registry projection.
func (h *BaseHandlers) ListTools(c *gin.Context) {
	if h.Tools == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("tool registry is not configured"))
		return
	}
	scope := h.operatorToolScope(c)
	views, err := h.Tools.List(c.Request.Context(), scope)
	if err != nil {
		h.respondToolError(c, err)
		return
	}
	c.JSON(http.StatusOK, contract.ToolsResponse{Tools: ToolPayloadsFromViews(views)})
}

// SearchTools searches the operator-visible registry projection.
func (h *BaseHandlers) SearchTools(c *gin.Context) {
	req, ok := h.bindToolSearch(c)
	if !ok {
		return
	}
	if h.Tools == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("tool registry is not configured"))
		return
	}
	scope := toolScopeFromSearch(h.operatorToolScope(c), req)
	views, err := h.Tools.Search(c.Request.Context(), scope, toolspkg.SearchQuery{
		Query: req.Query,
		Limit: req.Limit,
	})
	if err != nil {
		h.respondToolError(c, err)
		return
	}
	c.JSON(http.StatusOK, contract.ToolsResponse{Tools: ToolPayloadsFromViews(views)})
}

// GetTool returns one operator-visible tool projection.
func (h *BaseHandlers) GetTool(c *gin.Context) {
	if h.Tools == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("tool registry is not configured"))
		return
	}
	id, ok := h.toolIDParam(c)
	if !ok {
		return
	}
	view, err := h.Tools.Get(c.Request.Context(), h.operatorToolScope(c), id)
	if err != nil {
		h.respondToolError(c, err)
		return
	}
	c.JSON(http.StatusOK, contract.ToolResponse{Tool: ToolPayloadFromView(&view)})
}

// CreateToolApproval mints one daemon-memory approval reference for a concrete invocation.
func (h *BaseHandlers) CreateToolApproval(c *gin.Context) {
	if h.Tools == nil || h.ToolApprovals == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("tool approval service is not configured"))
		return
	}
	id, ok := h.toolIDParam(c)
	if !ok {
		return
	}
	var req contract.ToolApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondToolError(c, toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			id,
			fmt.Sprintf("%s: decode tool approval request: %v", h.transportName(), err),
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonSchemaInvalid,
		))
		return
	}
	scope := h.operatorToolScope(c)
	scope.SessionID = firstNonEmpty(req.SessionID, scope.SessionID)
	scope.WorkspaceID = firstNonEmpty(req.WorkspaceID, scope.WorkspaceID)
	scope.AgentName = firstNonEmpty(req.AgentName, scope.AgentName)
	if _, err := h.Tools.Get(c.Request.Context(), scope, id); err != nil {
		h.respondToolError(c, err)
		return
	}
	grant, err := h.ToolApprovals.CreateToolApproval(c.Request.Context(), scope, toolspkg.ApprovalRequest{
		ToolID:      id,
		SessionID:   scope.SessionID,
		WorkspaceID: scope.WorkspaceID,
		AgentName:   scope.AgentName,
		Input:       cloneRawMessage(req.Input),
		InputDigest: req.InputDigest,
	})
	if err != nil {
		h.respondToolError(c, err)
		return
	}
	c.JSON(http.StatusCreated, contract.ToolApprovalResponse{Approval: contract.ToolApprovalPayload{
		ApprovalToken: grant.ApprovalToken,
		ExpiresAt:     grant.ExpiresAt,
		ToolID:        grant.ToolID,
		InputDigest:   grant.InputDigest,
	}})
}

// InvokeTool dispatches a concrete tool invocation through the registry.
func (h *BaseHandlers) InvokeTool(c *gin.Context) {
	if h.Tools == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("tool registry is not configured"))
		return
	}
	id, ok := h.toolIDParam(c)
	if !ok {
		return
	}
	var req contract.ToolInvokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondToolError(c, toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			id,
			fmt.Sprintf("%s: decode tool invoke request: %v", h.transportName(), err),
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonSchemaInvalid,
		))
		return
	}
	scope := h.operatorToolScope(c)
	scope.SessionID = firstNonEmpty(req.SessionID, scope.SessionID)
	scope.WorkspaceID = firstNonEmpty(req.WorkspaceID, scope.WorkspaceID)
	scope.AgentName = firstNonEmpty(req.AgentName, scope.AgentName)
	result, err := h.Tools.Call(c.Request.Context(), scope, toolspkg.CallRequest{
		ToolID:               id,
		ToolCallID:           req.ToolCallID,
		TurnID:               req.TurnID,
		SessionID:            scope.SessionID,
		WorkspaceID:          scope.WorkspaceID,
		AgentName:            scope.AgentName,
		CorrelationID:        req.CorrelationID,
		Input:                cloneRawMessage(req.Input),
		SensitiveInputFields: append([]string(nil), req.SensitiveInputFields...),
		ApprovalToken:        req.ApprovalToken,
	})
	if err != nil {
		h.respondToolError(c, err)
		return
	}
	c.JSON(http.StatusOK, contract.ToolInvokeResponse{
		ToolID:     id,
		Status:     toolInvokeStatusCompleted,
		Result:     result,
		Truncated:  result.Truncated,
		DurationMS: result.DurationMS,
		Events:     []contract.ToolCallEventPayload{},
	})
}

// ListSessionTools returns the session/model-callable projection.
func (h *BaseHandlers) ListSessionTools(c *gin.Context) {
	if h.Tools == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("tool registry is not configured"))
		return
	}
	scope := h.sessionToolScope(c)
	views, err := h.Tools.List(c.Request.Context(), scope)
	if err != nil {
		h.respondToolError(c, err)
		return
	}
	c.JSON(http.StatusOK, contract.ToolsResponse{Tools: ToolPayloadsFromViews(views)})
}

// SearchSessionTools searches only within the session/model-callable projection.
func (h *BaseHandlers) SearchSessionTools(c *gin.Context) {
	req, ok := h.bindToolSearch(c)
	if !ok {
		return
	}
	if h.Tools == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("tool registry is not configured"))
		return
	}
	scope := h.sessionToolScope(c)
	scope.WorkspaceID = firstNonEmpty(req.WorkspaceID, scope.WorkspaceID)
	scope.AgentName = firstNonEmpty(req.AgentName, scope.AgentName)
	views, err := h.Tools.Search(c.Request.Context(), scope, toolspkg.SearchQuery{
		Query: req.Query,
		Limit: req.Limit,
	})
	if err != nil {
		h.respondToolError(c, err)
		return
	}
	c.JSON(http.StatusOK, contract.ToolsResponse{Tools: ToolPayloadsFromViews(views)})
}

// ListToolsets returns named toolsets with expansion diagnostics.
func (h *BaseHandlers) ListToolsets(c *gin.Context) {
	if h.Toolsets == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("toolset registry is not configured"))
		return
	}
	views, err := h.Toolsets.ListToolsets(c.Request.Context(), h.operatorToolScope(c))
	if err != nil {
		h.respondToolError(c, err)
		return
	}
	c.JSON(http.StatusOK, contract.ToolsetsResponse{Toolsets: ToolsetPayloadsFromViews(views)})
}

// GetToolset returns one named toolset with expansion diagnostics.
func (h *BaseHandlers) GetToolset(c *gin.Context) {
	if h.Toolsets == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("toolset registry is not configured"))
		return
	}
	id := toolspkg.ToolsetID(strings.TrimSpace(c.Param("id")))
	if err := id.Validate(); err != nil {
		h.respondToolError(c, err)
		return
	}
	view, err := h.Toolsets.GetToolset(c.Request.Context(), h.operatorToolScope(c), id)
	if err != nil {
		h.respondToolError(c, err)
		return
	}
	c.JSON(http.StatusOK, contract.ToolsetResponse{Toolset: ToolsetPayloadFromView(view)})
}

// ToolPayloadsFromViews converts registry views into public DTOs.
func ToolPayloadsFromViews(views []toolspkg.ToolView) []contract.ToolPayload {
	payloads := make([]contract.ToolPayload, 0, len(views))
	for i := range views {
		payloads = append(payloads, ToolPayloadFromView(&views[i]))
	}
	return payloads
}

// ToolPayloadFromView converts one registry view into a public DTO.
func ToolPayloadFromView(view *toolspkg.ToolView) contract.ToolPayload {
	return contract.ToolPayload{
		Descriptor:   toolDescriptorPayload(view.Descriptor),
		Availability: toolAvailabilityPayload(view.Availability),
		Decision:     toolDecisionPayload(view.Decision),
	}
}

// ToolsetPayloadsFromViews converts toolset projections into public DTOs.
func ToolsetPayloadsFromViews(views []toolspkg.ToolsetView) []contract.ToolsetPayload {
	payloads := make([]contract.ToolsetPayload, 0, len(views))
	for i := range views {
		payloads = append(payloads, ToolsetPayloadFromView(views[i]))
	}
	return payloads
}

// ToolsetPayloadFromView converts one toolset projection into a public DTO.
func ToolsetPayloadFromView(view toolspkg.ToolsetView) contract.ToolsetPayload {
	status := toolsetStatusExpanded
	if len(view.ReasonCodes) > 0 {
		status = toolsetStatusDegraded
	}
	return contract.ToolsetPayload{
		ID:            view.Toolset.ID,
		Tools:         append([]string(nil), view.Toolset.Tools...),
		Toolsets:      append([]toolspkg.ToolsetID(nil), view.Toolset.Toolsets...),
		ExpandedTools: append([]toolspkg.ToolID(nil), view.ExpandedTools...),
		Status:        status,
		ReasonCodes:   append([]toolspkg.ReasonCode(nil), view.ReasonCodes...),
	}
}

// toolDescriptorPayload detaches registry-owned descriptor data from transport DTOs.
func toolDescriptorPayload(d toolspkg.Descriptor) contract.ToolDescriptorPayload {
	return contract.ToolDescriptorPayload{
		ToolID:              d.ID,
		Backend:             toolBackendPayload(d.Backend),
		DisplayTitle:        d.DisplayTitle,
		Description:         d.Description,
		InputSchema:         cloneRawMessage(d.InputSchema),
		OutputSchema:        cloneRawMessage(d.OutputSchema),
		Source:              toolSourcePayload(d.Source),
		Visibility:          d.Visibility,
		Risk:                d.Risk,
		ReadOnly:            d.ReadOnly,
		Destructive:         d.Destructive,
		OpenWorld:           d.OpenWorld,
		RequiresInteraction: d.RequiresInteraction,
		ConcurrencySafe:     d.ConcurrencySafe,
		MaxResultBytes:      d.MaxResultBytes,
		Toolsets:            append([]toolspkg.ToolsetID(nil), d.Toolsets...),
		Tags:                append([]string(nil), d.Tags...),
		SearchHints:         append([]string(nil), d.SearchHints...),
	}
}

// toolBackendPayload preserves backend routing metadata without exposing registry internals.
func toolBackendPayload(backend toolspkg.BackendRef) contract.ToolBackendRefPayload {
	return contract.ToolBackendRefPayload{
		Kind:                 backend.Kind,
		ExtensionID:          backend.ExtensionID,
		Handler:              backend.Handler,
		MCPServer:            backend.MCPServer,
		MCPTool:              backend.MCPTool,
		NativeName:           backend.NativeName,
		RequiresCapabilities: append([]string(nil), backend.RequiresCapabilities...),
	}
}

// toolSourcePayload carries provenance fields needed for operator audits.
func toolSourcePayload(source toolspkg.SourceRef) contract.ToolSourceRefPayload {
	return contract.ToolSourceRefPayload{
		Kind:            source.Kind,
		Owner:           source.Owner,
		RawServerName:   source.RawServerName,
		RawToolName:     source.RawToolName,
		ResourceID:      source.ResourceID,
		ResourceVersion: source.ResourceVersion,
		WorkspaceID:     source.WorkspaceID,
		Scope:           source.Scope,
	}
}

// toolAvailabilityPayload keeps availability reasons stable across transports.
func toolAvailabilityPayload(availability toolspkg.Availability) contract.ToolAvailabilityPayload {
	return contract.ToolAvailabilityPayload{
		Registered:  availability.Registered,
		Enabled:     availability.Enabled,
		Available:   availability.Available,
		Authorized:  availability.Authorized,
		Executable:  availability.Executable,
		Conflicted:  availability.Conflicted,
		ReasonCodes: append([]toolspkg.ReasonCode(nil), availability.ReasonCodes...),
	}
}

// toolDecisionPayload exposes policy outcomes without sharing mutable registry state.
func toolDecisionPayload(decision toolspkg.EffectiveToolDecision) contract.ToolPolicyDecisionPayload {
	return contract.ToolPolicyDecisionPayload{
		VisibleToOperator:    decision.VisibleToOperator,
		VisibleToSession:     decision.VisibleToSession,
		Callable:             decision.Callable,
		ApprovalRequired:     decision.ApprovalRequired,
		SystemPermissionMode: decision.SystemPermissionMode,
		SessionPolicyResult:  decision.SessionPolicyResult,
		AgentPolicyResult:    decision.AgentPolicyResult,
		RegistryPolicyResult: decision.RegistryPolicyResult,
		SourcePolicyResult:   decision.SourcePolicyResult,
		AvailabilityResult:   decision.AvailabilityResult,
		HookResult:           decision.HookResult,
		ReasonCodes:          append([]toolspkg.ReasonCode(nil), decision.ReasonCodes...),
	}
}

// bindToolSearch normalizes malformed search input into the shared tool error contract.
func (h *BaseHandlers) bindToolSearch(c *gin.Context) (contract.ToolSearchRequest, bool) {
	var req contract.ToolSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondToolError(c, toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			"",
			fmt.Sprintf("%s: decode tool search request: %v", h.transportName(), err),
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonSchemaInvalid,
		))
		return contract.ToolSearchRequest{}, false
	}
	return req, true
}

// toolIDParam validates route IDs before they reach registry lookups.
func (h *BaseHandlers) toolIDParam(c *gin.Context) (toolspkg.ToolID, bool) {
	id := toolspkg.ToolID(strings.TrimSpace(c.Param("id")))
	if err := id.Validate(); err != nil {
		h.respondToolError(c, err)
		return "", false
	}
	return id, true
}

// operatorToolScope builds privileged projections from query parameters only.
func (h *BaseHandlers) operatorToolScope(c *gin.Context) toolspkg.Scope {
	return toolspkg.Scope{
		WorkspaceID: strings.TrimSpace(firstNonEmpty(c.Query("workspace_id"), c.Query("workspace"))),
		SessionID:   strings.TrimSpace(c.Query("session_id")),
		AgentName:   strings.TrimSpace(c.Query("agent_name")),
		Operator:    true,
	}
}

// sessionToolScope anchors session projections to the route session ID.
func (h *BaseHandlers) sessionToolScope(c *gin.Context) toolspkg.Scope {
	return toolspkg.Scope{
		WorkspaceID: strings.TrimSpace(firstNonEmpty(c.Query("workspace_id"), c.Query("workspace"))),
		SessionID:   strings.TrimSpace(c.Param("id")),
		AgentName:   strings.TrimSpace(c.Query("agent_name")),
	}
}

// toolScopeFromSearch lets request bodies narrow default scope without losing route context.
func toolScopeFromSearch(scope toolspkg.Scope, req contract.ToolSearchRequest) toolspkg.Scope {
	scope.WorkspaceID = firstNonEmpty(req.WorkspaceID, scope.WorkspaceID)
	scope.SessionID = firstNonEmpty(req.SessionID, scope.SessionID)
	scope.AgentName = firstNonEmpty(req.AgentName, scope.AgentName)
	return scope
}

// respondToolError serializes stable tool errors without backend error text.
func (h *BaseHandlers) respondToolError(c *gin.Context, err error) {
	status := StatusForToolError(err)
	var toolErr *toolspkg.ToolError
	payload := contract.ToolErrorPayload{
		Code:    toolspkg.ErrorCodeBackendFailed,
		Message: http.StatusText(status),
	}
	switch {
	case errors.As(err, &toolErr):
		payload.Code = toolErr.Code
		payload.Message = safeToolErrorMessage(status, toolErr.Code)
		payload.ToolID = toolErr.ToolID
		payload.ReasonCodes = append([]toolspkg.ReasonCode(nil), toolErr.ReasonCodes...)
		payload.Layer = toolErrorLayer(toolErr.ReasonCodes)
	case err != nil:
		payload.Code = toolErrorCodeForStatus(status)
		payload.Message = safeToolErrorMessage(status, payload.Code)
		if reason, ok := toolspkg.ReasonOf(err); ok {
			payload.ReasonCodes = []toolspkg.ReasonCode{reason}
			payload.Layer = toolErrorLayer(payload.ReasonCodes)
		}
	default:
		payload.Code = toolErrorCodeForStatus(status)
		payload.Message = http.StatusText(status)
	}
	if h.MaskInternalErrors && status >= http.StatusInternalServerError {
		payload.Message = http.StatusText(status)
	}
	if strings.TrimSpace(payload.Message) == "" {
		payload.Message = http.StatusText(status)
	}
	c.JSON(status, contract.ToolErrorResponse{Error: payload})
}

// safeToolErrorMessage maps internal failures to client-safe contract messages.
func safeToolErrorMessage(status int, code toolspkg.ErrorCode) string {
	switch code {
	case toolspkg.ErrorCodeNotFound:
		return "tool not found"
	case toolspkg.ErrorCodeConflict:
		return "tool conflict"
	case toolspkg.ErrorCodeUnavailable:
		return "tool unavailable"
	case toolspkg.ErrorCodeDenied:
		return "tool invocation denied"
	case toolspkg.ErrorCodeApprovalRequired:
		return "tool approval required"
	case toolspkg.ErrorCodeInvalidInput:
		return "invalid tool request"
	case toolspkg.ErrorCodeResultTooLarge:
		return "tool result too large"
	case toolspkg.ErrorCodeCanceled:
		return "tool call canceled"
	case toolspkg.ErrorCodeTimedOut:
		return "tool call timed out"
	case toolspkg.ErrorCodeBackendFailed:
		return "tool backend failed"
	default:
		message := http.StatusText(status)
		if strings.TrimSpace(message) == "" {
			return "tool request failed"
		}
		return message
	}
}

// toolErrorCodeForStatus keeps non-tool errors compatible with tool error payloads.
func toolErrorCodeForStatus(status int) toolspkg.ErrorCode {
	switch status {
	case http.StatusBadRequest:
		return toolspkg.ErrorCodeInvalidInput
	case http.StatusNotFound:
		return toolspkg.ErrorCodeNotFound
	case http.StatusForbidden:
		return toolspkg.ErrorCodeDenied
	case http.StatusAccepted:
		return toolspkg.ErrorCodeApprovalRequired
	case http.StatusConflict:
		return toolspkg.ErrorCodeConflict
	case http.StatusUnprocessableEntity:
		return toolspkg.ErrorCodeUnavailable
	default:
		return toolspkg.ErrorCodeBackendFailed
	}
}

// toolErrorLayer groups registry reasons into stable transport-facing layers.
func toolErrorLayer(reasons []toolspkg.ReasonCode) string {
	for _, reason := range reasons {
		switch reason {
		case toolspkg.ReasonSourceDisabled,
			toolspkg.ReasonMCPAuthUnconfigured,
			toolspkg.ReasonMCPAuthRequired,
			toolspkg.ReasonMCPAuthExpired,
			toolspkg.ReasonMCPAuthInvalid,
			toolspkg.ReasonMCPAuthRefreshFailed:
			return "source_policy"
		case toolspkg.ReasonSessionDenied:
			return "session_lineage"
		case toolspkg.ReasonPolicyDenied, toolspkg.ReasonVisibilityDenied:
			return "registry_policy"
		case toolspkg.ReasonHookDenied:
			return "hook"
		case toolspkg.ReasonApprovalRequired,
			toolspkg.ReasonApprovalUnreachable,
			toolspkg.ReasonApprovalTimedOut,
			toolspkg.ReasonApprovalCanceled,
			toolspkg.ReasonApprovalTokenMissing,
			toolspkg.ReasonApprovalTokenExpired,
			toolspkg.ReasonApprovalTokenMismatch,
			toolspkg.ReasonApprovalTokenReplayed:
			return "approval"
		case toolspkg.ReasonBackendUnhealthy,
			toolspkg.ReasonBackendNotExecutable,
			toolspkg.ReasonConflictedID,
			toolspkg.ReasonConflictedSanitizedName:
			return "availability"
		}
	}
	return ""
}
