package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/network/participation"
	workspacepkg "github.com/compozy/agh/internal/workspace"
	"github.com/gin-gonic/gin"
)

// GetNetworkCoordination returns the workspace coordination setting and invitation state.
func (h *BaseHandlers) GetNetworkCoordination(c *gin.Context) {
	workspaceID, ok := h.requireRouteWorkspaceID(c)
	if !ok {
		return
	}
	payload, err := h.networkCoordinationPayload(c.Request.Context(), workspaceID, c.Query("task_id"))
	if err != nil {
		h.respondError(c, statusForNetworkCoordinationError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.NetworkCoordinationResponse{Coordination: payload})
}

// PutNetworkCoordination enables or disables workspace coordination conversations.
func (h *BaseHandlers) PutNetworkCoordination(c *gin.Context) {
	workspaceID, ok := h.requireRouteWorkspaceID(c)
	if !ok {
		return
	}
	if h.CoordinationSettings == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("api: coordination settings are unavailable"))
		return
	}
	var req contract.PutNetworkCoordinationRequest
	if err := decodeStrictJSONBody(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode coordination request: %w", h.transportName(), err),
		)
		return
	}
	actor := coordinationActor(c)
	setting, err := h.CoordinationSettings.Set(c.Request.Context(), workspaceID, req.Enabled, actor)
	if err != nil {
		h.respondError(c, statusForNetworkCoordinationError(err), err)
		return
	}
	invitation, inviteErr := h.loadInvitationPayload(c.Request.Context(), workspaceID, c.Query("task_id"))
	if inviteErr != nil {
		h.respondError(c, statusForNetworkCoordinationError(inviteErr), inviteErr)
		return
	}
	payload := coordinationPayloadFromSetting(setting, invitation)
	c.JSON(http.StatusOK, contract.NetworkCoordinationResponse{Coordination: payload})
}

// PutNetworkCoordinationInvitation persists or resets invitation dismissal for one scope.
func (h *BaseHandlers) PutNetworkCoordinationInvitation(c *gin.Context) {
	workspaceID, ok := h.requireRouteWorkspaceID(c)
	if !ok {
		return
	}
	if h.CoordinationInvitations == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("api: coordination invitations are unavailable"))
		return
	}
	var req contract.PutNetworkCoordinationInvitationRequest
	if err := decodeStrictJSONBody(c, &req); err != nil {
		h.respondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("%s: decode invitation request: %w", h.transportName(), err),
		)
		return
	}
	scopeKind, scopeID, err := workspacepkg.NormalizeInvitationScope(req.Scope, workspaceID, req.TaskID)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	actor := coordinationActor(c)
	var invitation contract.NetworkCoordinationInvitationPayload
	if req.Dismissed {
		row, dismissErr := h.CoordinationInvitations.DismissInvitation(
			c.Request.Context(),
			scopeKind,
			scopeID,
			actor,
		)
		if dismissErr != nil {
			h.respondError(c, statusForNetworkCoordinationError(dismissErr), dismissErr)
			return
		}
		invitation = invitationPayloadFromRow(row)
	} else {
		if resetErr := h.CoordinationInvitations.ResetInvitation(
			c.Request.Context(),
			scopeKind,
			scopeID,
		); resetErr != nil {
			h.respondError(c, statusForNetworkCoordinationError(resetErr), resetErr)
			return
		}
		invitation = contract.NetworkCoordinationInvitationPayload{
			Scope:     scopeKind,
			TaskID:    invitationTaskID(scopeKind, scopeID),
			Dismissed: false,
		}
	}
	if h.CoordinationSettings == nil {
		h.respondError(c, http.StatusServiceUnavailable, errors.New("api: coordination settings are unavailable"))
		return
	}
	setting, settingErr := h.CoordinationSettings.Get(c.Request.Context(), workspaceID)
	if settingErr != nil {
		h.respondError(c, statusForNetworkCoordinationError(settingErr), settingErr)
		return
	}
	payload := coordinationPayloadFromSetting(setting, &invitation)
	c.JSON(http.StatusOK, contract.NetworkCoordinationResponse{Coordination: payload})
}

func (h *BaseHandlers) requireRouteWorkspaceID(c *gin.Context) (string, bool) {
	workspaceRef := strings.TrimSpace(c.Param("workspace_id"))
	if workspaceRef == "" {
		h.respondError(c, http.StatusBadRequest, errors.New("api: workspace_id is required"))
		return "", false
	}
	if h.Workspaces != nil {
		workspace, err := h.Workspaces.Get(c.Request.Context(), workspaceRef)
		if err != nil {
			h.respondError(c, StatusForWorkspaceError(err), err)
			return "", false
		}
		workspaceID := strings.TrimSpace(workspace.ID)
		if workspaceID == "" {
			h.respondError(c, http.StatusInternalServerError, errors.New("api: resolved workspace id is required"))
			return "", false
		}
		return workspaceID, true
	}
	return workspaceRef, true
}

func (h *BaseHandlers) networkCoordinationPayload(
	ctx context.Context,
	workspaceID string,
	taskID string,
) (contract.NetworkCoordinationPayload, error) {
	if h.CoordinationSettings == nil {
		return contract.NetworkCoordinationPayload{}, errors.New("api: coordination settings are unavailable")
	}
	setting, err := h.CoordinationSettings.Get(ctx, workspaceID)
	if err != nil {
		return contract.NetworkCoordinationPayload{}, err
	}
	invitation, inviteErr := h.loadInvitationPayload(ctx, workspaceID, taskID)
	if inviteErr != nil {
		return contract.NetworkCoordinationPayload{}, inviteErr
	}
	return coordinationPayloadFromSetting(setting, invitation), nil
}

func (h *BaseHandlers) loadInvitationPayload(
	ctx context.Context,
	workspaceID string,
	taskID string,
) (*contract.NetworkCoordinationInvitationPayload, error) {
	if h.CoordinationInvitations == nil {
		return nil, nil
	}
	scope := workspacepkg.InvitationScopeWorkspace
	if strings.TrimSpace(taskID) != "" {
		scope = workspacepkg.InvitationScopeTask
	}
	scopeKind, scopeID, err := workspacepkg.NormalizeInvitationScope(scope, workspaceID, taskID)
	if err != nil {
		return nil, err
	}
	row, err := h.CoordinationInvitations.GetInvitation(ctx, scopeKind, scopeID)
	if err != nil {
		return nil, err
	}
	payload := invitationPayloadFromRow(row)
	return &payload, nil
}

func coordinationActor(c *gin.Context) string {
	actor := strings.TrimSpace(c.GetHeader("X-AGH-Actor"))
	if actor == "" {
		return "operator"
	}
	return actor
}

func coordinationPayloadFromSetting(
	setting workspacepkg.CoordinationSetting,
	invitation *contract.NetworkCoordinationInvitationPayload,
) contract.NetworkCoordinationPayload {
	return contract.NetworkCoordinationPayload{
		WorkspaceID: setting.WorkspaceID,
		Enabled:     setting.Enabled,
		Revision:    setting.Revision,
		UpdatedAt:   setting.UpdatedAt,
		UpdatedBy:   setting.UpdatedBy,
		Invitation:  invitation,
	}
}

func invitationPayloadFromRow(row workspacepkg.CoordinationInvitation) contract.NetworkCoordinationInvitationPayload {
	payload := contract.NetworkCoordinationInvitationPayload{
		Scope:       row.ScopeKind,
		TaskID:      invitationTaskID(row.ScopeKind, row.ScopeID),
		Dismissed:   row.Dismissed,
		DismissedBy: row.DismissedBy,
	}
	if row.Dismissed && !row.DismissedAt.IsZero() {
		dismissedAt := row.DismissedAt.UTC()
		payload.DismissedAt = &dismissedAt
	}
	return payload
}

func invitationTaskID(scopeKind string, scopeID string) string {
	if scopeKind == workspacepkg.InvitationScopeTask {
		return scopeID
	}
	return ""
}

func statusForNetworkCoordinationError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, participation.ErrUnavailable):
		return http.StatusConflict
	case errors.Is(err, workspacepkg.ErrWorkspaceNotFound):
		return http.StatusNotFound
	default:
		msg := err.Error()
		if strings.Contains(msg, "invitation scope") ||
			strings.Contains(msg, "task_id") ||
			strings.Contains(msg, "workspace_id is required") {
			return http.StatusBadRequest
		}
		return http.StatusInternalServerError
	}
}
