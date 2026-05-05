package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/agentidentity"
	"github.com/pedronauck/agh/internal/api/contract"
)

const (
	agentActionMe                = "agent.me"
	agentActionCoordinatorConfig = "agent.coordinator.config"
)

var (
	errAgentIdentityUnavailable = errors.New("api: session service is not configured")
	errCoordinatorConfigMissing = errors.New("api: coordinator config service is not configured")
)

// StatusForAgentIdentityError maps agent identity failures to transport statuses.
func StatusForAgentIdentityError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, errAgentIdentityUnavailable),
		errors.Is(err, agentidentity.ErrIdentityLookupUnavailable):
		return http.StatusServiceUnavailable
	case errors.Is(err, agentidentity.ErrIdentityUnauthorized):
		return http.StatusForbidden
	case errors.Is(err, agentidentity.ErrIdentityRequired),
		errors.Is(err, agentidentity.ErrIdentityMismatch),
		errors.Is(err, agentidentity.ErrIdentityStale):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// AgentMe returns the daemon-validated caller identity for agent-facing UDS operations.
func (h *BaseHandlers) AgentMe(c *gin.Context) {
	caller, ok := h.requireAgentCaller(c, agentActionMe)
	if !ok {
		return
	}
	payload := agentMePayloadFromCaller(caller)
	h.enrichAgentMePayload(c.Request.Context(), caller, &payload)
	c.JSON(http.StatusOK, contract.AgentMeResponse{Me: contract.NormalizeAgentMePayload(payload)})
}

// AgentCoordinatorConfig returns the resolved coordinator policy for the caller workspace.
func (h *BaseHandlers) AgentCoordinatorConfig(c *gin.Context) {
	caller, ok := h.requireAgentCaller(c, agentActionCoordinatorConfig)
	if !ok {
		return
	}
	payload, err := h.agentCoordinatorConfigPayload(c.Request.Context(), caller.Session.WorkspaceID)
	if err != nil {
		h.respondError(c, statusForCoordinatorConfigError(err), err)
		return
	}
	c.JSON(http.StatusOK, contract.AgentCoordinatorConfigResponse{Coordinator: payload})
}

func (h *BaseHandlers) requireAgentCaller(
	c *gin.Context,
	action string,
) (agentidentity.Caller, bool) {
	caller, err := h.resolveAgentCaller(
		c.Request.Context(),
		agentCallerCredentialsFromRequest(c),
		action,
	)
	if err != nil {
		h.respondError(c, StatusForAgentIdentityError(err), err)
		return agentidentity.Caller{}, false
	}
	return caller, true
}

func (h *BaseHandlers) resolveAgentCaller(
	ctx context.Context,
	credentials agentidentity.Credentials,
	action string,
) (agentidentity.Caller, error) {
	if h == nil || h.Sessions == nil {
		return agentidentity.Caller{}, errAgentIdentityUnavailable
	}
	return agentidentity.Resolve(ctx, agentidentity.ResolveOptions{
		Credentials: credentials,
		Lookup:      h.agentSessionLookup,
		OriginKind:  taskOriginKindForTransport(h.transportName()),
		OriginRef:   strings.TrimSpace(action),
	})
}

func (h *BaseHandlers) agentSessionLookup(
	ctx context.Context,
	sessionID string,
) (agentidentity.SessionSnapshot, error) {
	info, err := h.Sessions.Status(ctx, sessionID)
	if err != nil {
		return agentidentity.SessionSnapshot{}, err
	}
	return agentidentity.SessionSnapshotFromInfo(info), nil
}

func agentCallerCredentialsFromRequest(c *gin.Context) agentidentity.Credentials {
	if c == nil || c.Request == nil {
		return agentidentity.Credentials{}
	}
	return agentidentity.Credentials{
		SessionID:   c.GetHeader(agentidentity.HeaderSessionID),
		AgentName:   c.GetHeader(agentidentity.HeaderAgent),
		WorkspaceID: c.GetHeader(agentidentity.HeaderWorkspaceID),
	}
}

func agentMePayloadFromCaller(caller agentidentity.Caller) contract.AgentMePayload {
	payload := contract.AgentMePayload{
		Self: contract.AgentIdentityPayload{
			SessionID: caller.Session.ID,
			AgentName: caller.Session.AgentName,
			Provider:  caller.Session.Provider,
			Model:     caller.Session.Model,
		},
		Workspace: contract.AgentWorkspacePayload{
			ID:      caller.Session.WorkspaceID,
			RootDir: caller.Session.WorkspacePath,
		},
		Session: contract.AgentSessionPayload{
			ID:        caller.Session.ID,
			Name:      caller.Session.Name,
			Type:      caller.Session.Type,
			State:     caller.Session.State,
			Channel:   caller.Session.Channel,
			Lineage:   contract.SessionLineagePayloadFromStore(caller.Session.Lineage),
			CreatedAt: caller.Session.CreatedAt,
			UpdatedAt: caller.Session.UpdatedAt,
		},
	}
	return contract.NormalizeAgentMePayload(payload)
}

func (h *BaseHandlers) agentCoordinatorConfigPayload(
	ctx context.Context,
	workspaceID string,
) (contract.CoordinatorConfigPayload, error) {
	if h == nil || h.CoordinatorConfig == nil {
		return contract.CoordinatorConfigPayload{}, errCoordinatorConfigMissing
	}
	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	cfg, err := h.CoordinatorConfig.ResolveCoordinatorConfig(ctx, trimmedWorkspaceID)
	if err != nil {
		return contract.CoordinatorConfigPayload{}, fmt.Errorf("resolve coordinator config: %w", err)
	}
	source := contract.CoordinatorConfigSourceGlobal
	if trimmedWorkspaceID != "" {
		source = contract.CoordinatorConfigSourceWorkspace
	}
	return CoordinatorConfigPayloadFromConfig(cfg, source, trimmedWorkspaceID), nil
}

func statusForCoordinatorConfigError(err error) int {
	if errors.Is(err, errCoordinatorConfigMissing) {
		return http.StatusServiceUnavailable
	}
	return http.StatusInternalServerError
}
