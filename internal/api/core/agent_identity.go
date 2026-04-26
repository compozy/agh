package core

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/agentidentity"
	"github.com/pedronauck/agh/internal/api/contract"
)

const agentActionMe = "agent.me"

// StatusForAgentIdentityError maps agent identity failures to transport statuses.
func StatusForAgentIdentityError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
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
		return agentidentity.Caller{}, errors.New("api: session service is not configured")
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
			Lineage:   SessionLineagePayloadFromStore(caller.Session.Lineage),
			CreatedAt: caller.Session.CreatedAt,
			UpdatedAt: caller.Session.UpdatedAt,
		},
	}
	return contract.NormalizeAgentMePayload(payload)
}
