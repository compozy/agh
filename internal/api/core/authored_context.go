package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/heartbeat"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/soul"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const defaultWakeEventInspectLimit = 10

var (
	errAuthoredContextValidation = errors.New("authored context validation error")
	errSoulAuthoringUnavailable  = errors.New("api: soul authoring service is not configured")
	errSoulRefreshUnavailable    = errors.New("api: soul refresh service is not configured")
	errHeartbeatAuthoringMissing = errors.New("api: heartbeat authoring service is not configured")
	errHeartbeatStatusMissing    = errors.New("api: heartbeat status service is not configured")
	errHeartbeatWakeMissing      = errors.New("api: heartbeat wake service is not configured")
	errSessionHealthMissing      = errors.New("api: session health service is not configured")
)

type authoredAgentTarget struct {
	workspaceID         string
	sessionWorkspaceID  string
	workspaceRoot       string
	agentName           string
	agentPath           string
	soulConfig          aghconfig.SoulConfig
	heartbeatConfig     aghconfig.HeartbeatConfig
	packageOwned        bool
	soulSourcePath      string
	soulBody            string
	heartbeatSourcePath string
	heartbeatBody       string
}

// StatusForSoulError maps Soul/session authoring failures to deterministic transport statuses.
func StatusForSoulError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, errSoulAuthoringUnavailable),
		errors.Is(err, errSoulRefreshUnavailable):
		return http.StatusServiceUnavailable
	case errors.Is(err, errAuthoredContextValidation):
		return http.StatusBadRequest
	case errors.Is(err, soul.ErrAuthoringConflict),
		errors.Is(err, session.ErrSoulRefreshConflict),
		errors.Is(err, session.ErrSoulRefreshDigestConflict),
		errors.Is(err, session.ErrSessionNotActive):
		return http.StatusConflict
	case errors.Is(err, soul.ErrAuthoringAgentNotFound),
		errors.Is(err, soul.ErrAuthoringMissing),
		errors.Is(err, soul.ErrRevisionNotFound),
		errors.Is(err, soul.ErrSnapshotNotFound),
		errors.Is(err, session.ErrSessionNotFound),
		errors.Is(err, workspacepkg.ErrWorkspaceNotFound),
		errors.Is(err, os.ErrNotExist):
		return http.StatusNotFound
	case errors.Is(err, workspacepkg.ErrWorkspaceRootMissing):
		return http.StatusGone
	case errors.Is(err, soul.ErrInvalid),
		errors.Is(err, soul.ErrInvalidSnapshot),
		errors.Is(err, soul.ErrInvalidRevision),
		errors.Is(err, soul.ErrAuthoringPathRejected),
		errors.Is(err, contract.ErrInvalidAuthoredContextEnum):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// StatusForHeartbeatError maps Heartbeat authoring/status failures to deterministic transport statuses.
func StatusForHeartbeatError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, errHeartbeatAuthoringMissing),
		errors.Is(err, errHeartbeatStatusMissing),
		errors.Is(err, errHeartbeatWakeMissing),
		errors.Is(err, errSessionHealthMissing):
		return http.StatusServiceUnavailable
	case errors.Is(err, errAuthoredContextValidation):
		return http.StatusBadRequest
	case errors.Is(err, heartbeat.ErrAuthoringConflict):
		return http.StatusConflict
	case errors.Is(err, heartbeat.ErrAuthoringAgentNotFound),
		errors.Is(err, heartbeat.ErrAuthoringNoPolicy),
		errors.Is(err, heartbeat.ErrRevisionNotFound),
		errors.Is(err, heartbeat.ErrSnapshotNotFound),
		errors.Is(err, heartbeat.ErrSessionHealthNotFound),
		errors.Is(err, session.ErrSessionNotFound),
		errors.Is(err, workspacepkg.ErrWorkspaceNotFound),
		errors.Is(err, os.ErrNotExist):
		return http.StatusNotFound
	case errors.Is(err, workspacepkg.ErrWorkspaceRootMissing):
		return http.StatusGone
	case errors.Is(err, heartbeat.ErrInvalid),
		errors.Is(err, heartbeat.ErrInvalidSnapshot),
		errors.Is(err, heartbeat.ErrInvalidRevision),
		errors.Is(err, heartbeat.ErrInvalidSessionHealth),
		errors.Is(err, heartbeat.ErrInvalidWakeEvent),
		errors.Is(err, heartbeat.ErrInvalidWakeState),
		errors.Is(err, heartbeat.ErrAuthoringPathRejected),
		errors.Is(err, contract.ErrInvalidAuthoredContextEnum):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// AgentSoul returns the caller's resolved full Soul read model.
func (h *BaseHandlers) AgentSoul(c *gin.Context) {
	caller, ok := h.requireAgentCaller(c, "agent.soul.inspect")
	if !ok {
		return
	}
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		caller.Session.WorkspaceID,
		caller.Session.AgentName,
	)
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	h.inspectSoulTarget(c, target)
}

// GetAgentSoul returns the resolved full Soul read model for one workspace-visible agent.
func (h *BaseHandlers) GetAgentSoul(c *gin.Context) {
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		authoredWorkspaceRefFromQuery(c),
		pathAgentName(c),
	)
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	h.inspectSoulTarget(c, target)
}

// ValidateAgentSoulDefinition validates a proposed SOUL.md body for one workspace-visible agent.
func (h *BaseHandlers) ValidateAgentSoulDefinition(c *gin.Context) {
	var req contract.AgentSoulValidateByPathRequest
	if err := decodeAuthoredJSONBody(c, &req, false); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	agentName, err := authoredRouteAgentName(pathAgentName(c))
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		firstNonEmpty(req.WorkspaceID, authoredWorkspaceRefFromQuery(c)),
		agentName,
	)
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	h.validateSoulTarget(c, target, req.Body)
}

// ValidateAgentSoul validates a proposed SOUL.md body for the caller's agent.
func (h *BaseHandlers) ValidateAgentSoul(c *gin.Context) {
	caller, ok := h.requireAgentCaller(c, "agent.soul.validate")
	if !ok {
		return
	}
	var req contract.AgentSoulValidateRequest
	if err := decodeAuthoredJSONBody(c, &req, true); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		firstNonEmpty(req.WorkspaceID, caller.Session.WorkspaceID),
		firstNonEmpty(req.AgentName, caller.Session.AgentName),
	)
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	h.validateSoulTarget(c, target, req.Body)
}

// PutAgentSoul creates or replaces SOUL.md through the managed authoring service.
func (h *BaseHandlers) PutAgentSoul(c *gin.Context) {
	if h.SoulAuthoring == nil {
		h.respondError(c, StatusForSoulError(errSoulAuthoringUnavailable), errSoulAuthoringUnavailable)
		return
	}
	if h.rejectSoulIfMatch(c) {
		return
	}
	var req contract.AgentSoulPutByPathRequest
	if err := decodeAuthoredJSONBody(c, &req, false); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	agentName, err := authoredRouteAgentName(pathAgentName(c))
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		firstNonEmpty(req.WorkspaceID, authoredWorkspaceRefFromQuery(c)),
		agentName,
	)
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	if err := target.rejectPackageOwnedSoulMutation(); err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	result, err := h.SoulAuthoring.Put(c.Request.Context(), soul.PutRequest{
		Target:         target.soulAuthoringTarget(),
		Body:           req.Body,
		ExpectedDigest: strings.TrimSpace(req.ExpectedDigest),
		Actor:          h.soulActorForRequest(),
		Origin:         h.soulOriginForRequest(c),
	})
	h.respondSoulMutation(c, target, &result, err)
}

// DeleteAgentSoul removes SOUL.md through the managed authoring service.
func (h *BaseHandlers) DeleteAgentSoul(c *gin.Context) {
	if h.SoulAuthoring == nil {
		h.respondError(c, StatusForSoulError(errSoulAuthoringUnavailable), errSoulAuthoringUnavailable)
		return
	}
	if h.rejectSoulIfMatch(c) {
		return
	}
	var req contract.AgentSoulDeleteByPathRequest
	if err := decodeAuthoredJSONBody(c, &req, false); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	agentName, err := authoredRouteAgentName(pathAgentName(c))
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		firstNonEmpty(req.WorkspaceID, authoredWorkspaceRefFromQuery(c)),
		agentName,
	)
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	if err := target.rejectPackageOwnedSoulMutation(); err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	result, err := h.SoulAuthoring.Delete(c.Request.Context(), soul.DeleteRequest{
		Target:         target.soulAuthoringTarget(),
		ExpectedDigest: strings.TrimSpace(req.ExpectedDigest),
		Actor:          h.soulActorForRequest(),
		Origin:         h.soulOriginForRequest(c),
	})
	h.respondSoulMutation(c, target, &result, err)
}

// ListAgentSoulHistory lists SOUL.md managed authoring revisions.
func (h *BaseHandlers) ListAgentSoulHistory(c *gin.Context) {
	if h.SoulAuthoring == nil {
		h.respondError(c, StatusForSoulError(errSoulAuthoringUnavailable), errSoulAuthoringUnavailable)
		return
	}
	limit, err := parsePositiveIntQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		authoredWorkspaceRefFromQuery(c),
		pathAgentName(c),
	)
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	if target.packageOwned {
		response := contract.AgentSoulHistoryResponse{
			Revisions: []contract.AgentSoulRevisionPayload{},
		}
		h.respondAuthoredJSON(
			c,
			http.StatusOK,
			response,
		)
		return
	}
	result, err := h.SoulAuthoring.History(c.Request.Context(), soul.HistoryRequest{
		Target: target.soulAuthoringTarget(),
		Limit:  limit,
	})
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	response, err := soulHistoryResponse(result)
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	h.respondAuthoredJSON(c, http.StatusOK, response)
}

// RollbackAgentSoul rolls SOUL.md back to a prior managed revision.
func (h *BaseHandlers) RollbackAgentSoul(c *gin.Context) {
	if h.SoulAuthoring == nil {
		h.respondError(c, StatusForSoulError(errSoulAuthoringUnavailable), errSoulAuthoringUnavailable)
		return
	}
	if h.rejectSoulIfMatch(c) {
		return
	}
	var req contract.AgentSoulRollbackByPathRequest
	if err := decodeAuthoredJSONBody(c, &req, false); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	agentName, err := authoredRouteAgentName(pathAgentName(c))
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		firstNonEmpty(req.WorkspaceID, authoredWorkspaceRefFromQuery(c)),
		agentName,
	)
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	if err := target.rejectPackageOwnedSoulMutation(); err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	result, err := h.SoulAuthoring.Rollback(c.Request.Context(), soul.RollbackRequest{
		Target:         target.soulAuthoringTarget(),
		RevisionID:     strings.TrimSpace(req.RevisionID),
		ExpectedDigest: strings.TrimSpace(req.ExpectedDigest),
		Actor:          h.soulActorForRequest(),
		Origin:         h.soulOriginForRequest(c),
	})
	h.respondSoulMutation(c, target, &result, err)
}

// RefreshSessionSoul refreshes an idle session's Soul snapshot through body-level CAS.
func (h *BaseHandlers) RefreshSessionSoul(c *gin.Context) {
	if h.SoulRefresher == nil {
		h.respondError(c, StatusForSoulError(errSoulRefreshUnavailable), errSoulRefreshUnavailable)
		return
	}
	if h.rejectSoulIfMatch(c) {
		return
	}
	var req contract.SessionSoulRefreshRequest
	if err := decodeAuthoredJSONBody(c, &req, true); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	_, sessionID, _, ok := h.routeSessionInWorkspace(c)
	if !ok {
		return
	}
	result, err := h.SoulRefresher.RefreshSoulWithExpectedDigest(
		c.Request.Context(),
		sessionID,
		req.ExpectedDigest,
	)
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	payload, err := agentSoulPayloadFromRefreshResult(result)
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	h.respondAuthoredJSON(c, http.StatusOK, payload)
}

// GetAgentHeartbeat returns the resolved Heartbeat policy for one workspace-visible agent.
func (h *BaseHandlers) GetAgentHeartbeat(c *gin.Context) {
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		authoredWorkspaceRefFromQuery(c),
		pathAgentName(c),
	)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	h.inspectHeartbeatTarget(c, target)
}

// ValidateAgentHeartbeat validates a proposed HEARTBEAT.md body.
func (h *BaseHandlers) ValidateAgentHeartbeat(c *gin.Context) {
	if h.HeartbeatAuthoring == nil {
		h.respondError(c, StatusForHeartbeatError(errHeartbeatAuthoringMissing), errHeartbeatAuthoringMissing)
		return
	}
	var req contract.HeartbeatValidateByPathRequest
	if err := decodeAuthoredJSONBody(c, &req, false); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	agentName, err := authoredRouteAgentName(pathAgentName(c))
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		firstNonEmpty(req.WorkspaceID, authoredWorkspaceRefFromQuery(c)),
		agentName,
	)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	if target.packageOwned {
		body := req.Body
		policy, policyErr := target.resolvePackageOwnedHeartbeat(c.Request.Context(), &body)
		h.respondHeartbeatPolicy(c, target.agentName, &policy, "", policyErr)
		return
	}
	body := req.Body
	result, err := h.HeartbeatAuthoring.Validate(c.Request.Context(), heartbeat.ValidateRequest{
		Target: target.heartbeatAuthoringTarget(),
		Body:   &body,
	})
	h.respondHeartbeatPolicy(c, target.agentName, &result.Policy, "", err)
}

// PutAgentHeartbeat creates or replaces HEARTBEAT.md through the managed authoring service.
func (h *BaseHandlers) PutAgentHeartbeat(c *gin.Context) {
	if h.rejectHeartbeatIfMatch(c) {
		return
	}
	if h.HeartbeatAuthoring == nil {
		h.respondError(c, StatusForHeartbeatError(errHeartbeatAuthoringMissing), errHeartbeatAuthoringMissing)
		return
	}
	var req contract.HeartbeatPutByPathRequest
	if err := decodeAuthoredJSONBody(c, &req, false); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	agentName, err := authoredRouteAgentName(pathAgentName(c))
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		firstNonEmpty(req.WorkspaceID, authoredWorkspaceRefFromQuery(c)),
		agentName,
	)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	if err := target.rejectPackageOwnedHeartbeatMutation(); err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	result, err := h.HeartbeatAuthoring.Put(c.Request.Context(), heartbeat.PutRequest{
		Target:         target.heartbeatAuthoringTarget(),
		Body:           req.Body,
		ExpectedDigest: strings.TrimSpace(req.ExpectedDigest),
		Actor:          h.heartbeatActorForRequest(),
	})
	h.respondHeartbeatMutation(c, target, &result, err)
}

// DeleteAgentHeartbeat removes HEARTBEAT.md through the managed authoring service.
func (h *BaseHandlers) DeleteAgentHeartbeat(c *gin.Context) {
	if h.rejectHeartbeatIfMatch(c) {
		return
	}
	if h.HeartbeatAuthoring == nil {
		h.respondError(c, StatusForHeartbeatError(errHeartbeatAuthoringMissing), errHeartbeatAuthoringMissing)
		return
	}
	var req contract.HeartbeatDeleteByPathRequest
	if err := decodeAuthoredJSONBody(c, &req, false); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	agentName, err := authoredRouteAgentName(pathAgentName(c))
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		firstNonEmpty(req.WorkspaceID, authoredWorkspaceRefFromQuery(c)),
		agentName,
	)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	if err := target.rejectPackageOwnedHeartbeatMutation(); err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	result, err := h.HeartbeatAuthoring.Delete(c.Request.Context(), heartbeat.DeleteRequest{
		Target:         target.heartbeatAuthoringTarget(),
		ExpectedDigest: strings.TrimSpace(req.ExpectedDigest),
		Actor:          h.heartbeatActorForRequest(),
	})
	h.respondHeartbeatMutation(c, target, &result, err)
}

// ListAgentHeartbeatHistory lists HEARTBEAT.md managed authoring revisions.
func (h *BaseHandlers) ListAgentHeartbeatHistory(c *gin.Context) {
	if h.HeartbeatAuthoring == nil {
		h.respondError(c, StatusForHeartbeatError(errHeartbeatAuthoringMissing), errHeartbeatAuthoringMissing)
		return
	}
	limit, err := parsePositiveIntQuery(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		authoredWorkspaceRefFromQuery(c),
		pathAgentName(c),
	)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	if target.packageOwned {
		response := contract.HeartbeatHistoryResponse{
			Revisions: []contract.HeartbeatRevisionPayload{},
		}
		h.respondAuthoredJSON(
			c,
			http.StatusOK,
			response,
		)
		return
	}
	result, err := h.HeartbeatAuthoring.History(c.Request.Context(), heartbeat.HistoryRequest{
		Target: target.heartbeatAuthoringTarget(),
		Limit:  limit,
	})
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	response, err := heartbeatHistoryResponse(result)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	h.respondAuthoredJSON(c, http.StatusOK, response)
}

// RollbackAgentHeartbeat rolls HEARTBEAT.md back to a prior revision or snapshot digest.
func (h *BaseHandlers) RollbackAgentHeartbeat(c *gin.Context) {
	if h.rejectHeartbeatIfMatch(c) {
		return
	}
	if h.HeartbeatAuthoring == nil {
		h.respondError(c, StatusForHeartbeatError(errHeartbeatAuthoringMissing), errHeartbeatAuthoringMissing)
		return
	}
	var req contract.HeartbeatRollbackByPathRequest
	if err := decodeAuthoredJSONBody(c, &req, false); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	agentName, err := authoredRouteAgentName(pathAgentName(c))
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		firstNonEmpty(req.WorkspaceID, authoredWorkspaceRefFromQuery(c)),
		agentName,
	)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	if err := target.rejectPackageOwnedHeartbeatMutation(); err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	result, err := h.HeartbeatAuthoring.Rollback(c.Request.Context(), heartbeat.RollbackRequest{
		Target:         target.heartbeatAuthoringTarget(),
		RevisionID:     strings.TrimSpace(req.RevisionID),
		TargetDigest:   strings.TrimSpace(req.TargetDigest),
		ExpectedDigest: strings.TrimSpace(req.ExpectedDigest),
		Actor:          h.heartbeatActorForRequest(),
	})
	h.respondHeartbeatMutation(c, target, &result, err)
}

// GetAgentHeartbeatStatus returns policy, wake state, and optional session health.
func (h *BaseHandlers) GetAgentHeartbeatStatus(c *gin.Context) {
	if h.HeartbeatStatus == nil {
		h.respondError(c, StatusForHeartbeatError(errHeartbeatStatusMissing), errHeartbeatStatusMissing)
		return
	}
	includeHealth, err := parseBoolQuery(c, "include_session_health")
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	includeEvents, err := parseBoolQuery(c, "include_recent_wake_events")
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		authoredWorkspaceRefFromQuery(c),
		pathAgentName(c),
	)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	sessionID := strings.TrimSpace(c.Query("session_id"))
	if sessionID != "" {
		if _, err := h.requireSessionInWorkspace(
			c.Request.Context(),
			target.storageWorkspaceID(),
			sessionID,
		); err != nil {
			h.respondError(c, statusForWorkspaceScopedResourceError(err), err)
			return
		}
	}
	result, err := h.HeartbeatStatus.Status(c.Request.Context(), heartbeat.StatusRequest{
		Target:               target.heartbeatAuthoringTarget(),
		SessionID:            sessionID,
		IncludeSessionHealth: includeHealth,
	})
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	response, err := contract.HeartbeatStatusResponseFromResult(&result)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	if includeEvents {
		events, eventsErr := h.heartbeatWakeEvents(c.Request.Context(), target, sessionID)
		if eventsErr != nil {
			h.respondError(c, StatusForHeartbeatError(eventsErr), eventsErr)
			return
		}
		response.WakeEvents = events
	}
	h.respondAuthoredJSON(c, http.StatusOK, response)
}

// WakeAgentHeartbeat requests one manual advisory Heartbeat wake decision.
func (h *BaseHandlers) WakeAgentHeartbeat(c *gin.Context) {
	if h.HeartbeatWake == nil {
		h.respondError(c, StatusForHeartbeatError(errHeartbeatWakeMissing), errHeartbeatWakeMissing)
		return
	}
	var req contract.HeartbeatWakeByPathRequest
	if err := decodeAuthoredJSONBody(c, &req, false); err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	agentName, err := authoredRouteAgentName(pathAgentName(c))
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	target, err := h.resolveAuthoredAgentTarget(
		c.Request.Context(),
		firstNonEmpty(req.WorkspaceID, authoredWorkspaceRefFromQuery(c)),
		agentName,
	)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	source := heartbeat.WakeSource(req.Source)
	if source == "" {
		source = heartbeat.WakeSourceManual
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID != "" {
		if _, err := h.requireSessionInWorkspace(
			c.Request.Context(),
			target.storageWorkspaceID(),
			sessionID,
		); err != nil {
			h.respondError(c, statusForWorkspaceScopedResourceError(err), err)
			return
		}
	}
	decision, err := h.HeartbeatWake.Wake(c.Request.Context(), heartbeat.WakeRequest{
		WorkspaceID: target.storageWorkspaceID(),
		AgentName:   target.agentName,
		SessionID:   sessionID,
		Source:      source,
		DryRun:      req.DryRun,
	})
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	payload, err := contract.HeartbeatWakeDecisionPayloadFromDomain(decision)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	status := statusForHeartbeatWakeDecision(decision)
	h.respondAuthoredJSON(c, status, contract.HeartbeatWakeResponse{Decision: payload})
}

// GetSessionHealth returns metadata-only session health and wake eligibility.
func (h *BaseHandlers) GetSessionHealth(c *gin.Context) {
	health, ok := h.sessionHealthPayloadForRoute(c)
	if !ok {
		return
	}
	h.respondAuthoredJSON(c, http.StatusOK, contract.SessionHealthResponse{Health: health})
}

// GetSessionStatus returns compact session health with wake state when available.
func (h *BaseHandlers) GetSessionStatus(c *gin.Context) {
	health, ok := h.sessionHealthPayloadForRoute(c)
	if !ok {
		return
	}
	response := contract.SessionStatusResponse{
		SessionID:           health.SessionID,
		WorkspaceID:         health.WorkspaceID,
		AgentName:           health.AgentName,
		State:               health.State,
		Health:              health.Health,
		ActivePrompt:        health.ActivePrompt,
		Attachable:          health.Attachable,
		EligibleForWake:     health.EligibleForWake,
		IneligibilityReason: health.IneligibilityReason,
		UpdatedAt:           health.UpdatedAt,
	}
	if h.HeartbeatStatus != nil {
		status, err := h.heartbeatStatusForHealth(c.Request.Context(), health, false)
		if err != nil {
			h.respondError(c, StatusForHeartbeatError(err), err)
			return
		}
		response.WakeState = status.WakeState
	}
	h.respondAuthoredJSON(c, http.StatusOK, response)
}

// InspectSession returns detailed health, wake audit, and policy correlation metadata.
func (h *BaseHandlers) InspectSession(c *gin.Context) {
	includeEvents, err := parseBoolQuery(c, "include_recent_wake_events")
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err)
		return
	}
	health, ok := h.sessionHealthPayloadForRoute(c)
	if !ok {
		return
	}
	response := contract.SessionInspectResponse{
		SessionID: health.SessionID,
		Health:    health,
	}
	if h.HeartbeatStatus != nil {
		status, statusErr := h.heartbeatStatusForHealth(c.Request.Context(), health, true)
		if statusErr != nil {
			h.respondError(c, StatusForHeartbeatError(statusErr), statusErr)
			return
		}
		response.WakeState = status.WakeState
		response.PolicyDigest = status.Digest
		response.ConfigDigest = status.ConfigDigest
		response.Diagnostics = status.Diagnostics
	}
	if includeEvents {
		target, targetErr := h.resolveAuthoredAgentTarget(c.Request.Context(), health.WorkspaceID, health.AgentName)
		if targetErr != nil {
			h.respondError(c, StatusForHeartbeatError(targetErr), targetErr)
			return
		}
		events, eventsErr := h.heartbeatWakeEvents(c.Request.Context(), target, health.SessionID)
		if eventsErr != nil {
			h.respondError(c, StatusForHeartbeatError(eventsErr), eventsErr)
			return
		}
		response.WakeEvents = events
	}
	h.respondAuthoredJSON(c, http.StatusOK, response)
}

func (h *BaseHandlers) inspectSoulTarget(c *gin.Context, target authoredAgentTarget) {
	if target.packageOwned {
		resolved, err := target.resolvePackageOwnedSoul(c.Request.Context(), nil)
		h.respondSoulPayload(c, target.agentName, &resolved, target.soulConfigProvenance(), err)
		return
	}
	if h.SoulAuthoring == nil {
		h.respondError(c, StatusForSoulError(errSoulAuthoringUnavailable), errSoulAuthoringUnavailable)
		return
	}
	result, err := h.SoulAuthoring.Validate(c.Request.Context(), soul.ValidateRequest{
		Target: target.soulAuthoringTarget(),
	})
	h.respondSoulPayload(c, target.agentName, &result.Soul, target.soulConfigProvenance(), err)
}

func (h *BaseHandlers) validateSoulTarget(c *gin.Context, target authoredAgentTarget, body string) {
	if target.packageOwned {
		var bodyPtr *string
		if strings.TrimSpace(body) != "" {
			bodyPtr = &body
		}
		resolved, err := target.resolvePackageOwnedSoul(c.Request.Context(), bodyPtr)
		h.respondSoulPayload(c, target.agentName, &resolved, target.soulConfigProvenance(), err)
		return
	}
	if h.SoulAuthoring == nil {
		h.respondError(c, StatusForSoulError(errSoulAuthoringUnavailable), errSoulAuthoringUnavailable)
		return
	}
	var bodyPtr *string
	if strings.TrimSpace(body) != "" {
		bodyPtr = &body
	}
	result, err := h.SoulAuthoring.Validate(c.Request.Context(), soul.ValidateRequest{
		Target: target.soulAuthoringTarget(),
		Body:   bodyPtr,
	})
	h.respondSoulPayload(c, target.agentName, &result.Soul, target.soulConfigProvenance(), err)
}

func (h *BaseHandlers) inspectHeartbeatTarget(c *gin.Context, target authoredAgentTarget) {
	if target.packageOwned && h.HeartbeatStatus == nil {
		policy, err := target.resolvePackageOwnedHeartbeat(c.Request.Context(), nil)
		h.respondHeartbeatPolicy(c, target.agentName, &policy, "", err)
		return
	}
	if h.HeartbeatStatus == nil {
		h.respondError(c, StatusForHeartbeatError(errHeartbeatStatusMissing), errHeartbeatStatusMissing)
		return
	}
	result, err := h.HeartbeatStatus.Inspect(c.Request.Context(), heartbeat.InspectRequest{
		Target: target.heartbeatAuthoringTarget(),
	})
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	snapshotID := ""
	if result.Snapshot != nil {
		snapshotID = result.Snapshot.ID
	}
	h.respondHeartbeatPolicy(c, result.AgentName, &result.Policy, snapshotID, nil)
}

func (h *BaseHandlers) respondSoulPayload(
	c *gin.Context,
	agentName string,
	resolved *soul.ResolvedSoul,
	provenance soul.ConfigProvenance,
	err error,
) {
	if err != nil && !errors.Is(err, soul.ErrInvalid) {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	payload := contract.AgentSoulPayloadFromResolved(agentName, resolved, "", provenance)
	status := http.StatusOK
	if err != nil || resolved == nil || !resolved.Valid {
		status = http.StatusUnprocessableEntity
	}
	h.respondAuthoredJSON(c, status, payload)
}

func (h *BaseHandlers) respondSoulMutation(
	c *gin.Context,
	target authoredAgentTarget,
	result *soul.MutationResult,
	err error,
) {
	if err != nil {
		if errors.Is(err, soul.ErrInvalid) {
			if result != nil {
				h.respondSoulPayload(c, target.agentName, &result.Soul, target.soulConfigProvenance(), err)
				return
			}
		}
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	response, err := soulMutationResponse(target, result)
	if err != nil {
		h.respondError(c, StatusForSoulError(err), err)
		return
	}
	h.respondAuthoredJSON(c, http.StatusOK, response)
}

func (h *BaseHandlers) respondHeartbeatPolicy(
	c *gin.Context,
	agentName string,
	policy *heartbeat.ResolvedPolicy,
	snapshotID string,
	err error,
) {
	if err != nil && !errors.Is(err, heartbeat.ErrInvalid) {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	payload, convertErr := contract.HeartbeatPolicyPayloadFromResolved(agentName, policy, snapshotID)
	if convertErr != nil {
		h.respondError(c, StatusForHeartbeatError(convertErr), convertErr)
		return
	}
	status := http.StatusOK
	if err != nil || policy == nil || !policy.Valid {
		status = http.StatusUnprocessableEntity
	}
	h.respondAuthoredJSON(c, status, payload)
}

func (h *BaseHandlers) respondHeartbeatMutation(
	c *gin.Context,
	target authoredAgentTarget,
	result *heartbeat.MutationResult,
	err error,
) {
	if err != nil {
		if errors.Is(err, heartbeat.ErrInvalid) {
			if result != nil {
				h.respondHeartbeatPolicy(c, target.agentName, &result.Policy, "", err)
				return
			}
		}
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	response, err := heartbeatMutationResponse(result)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return
	}
	h.respondAuthoredJSON(c, http.StatusOK, response)
}

func (h *BaseHandlers) respondAuthoredJSON(c *gin.Context, status int, payload any) {
	if err := contract.ValidateAuthoredContextRedacted(payload); err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(status, payload)
}

func (h *BaseHandlers) resolveAuthoredAgentTarget(
	ctx context.Context,
	workspaceRef string,
	agentName string,
) (authoredAgentTarget, error) {
	name := strings.TrimSpace(agentName)
	if name == "" {
		return authoredAgentTarget{}, newAuthoredValidationError("agent_name is required")
	}
	ref := strings.TrimSpace(workspaceRef)
	if ref == "" {
		return authoredAgentTarget{}, newAuthoredValidationError("workspace_id is required")
	}
	if h.Workspaces == nil {
		return authoredAgentTarget{}, workspacepkg.ErrWorkspaceResolverUnavailable
	}
	resolved, err := h.Workspaces.Resolve(ctx, ref)
	if err != nil {
		return authoredAgentTarget{}, err
	}
	root := strings.TrimSpace(resolved.RootDir)
	if root == "" {
		return authoredAgentTarget{}, workspacepkg.ErrWorkspaceRootMissing
	}
	return authoredAgentTarget{
		workspaceID:        strings.TrimSpace(resolved.WorkspaceID),
		sessionWorkspaceID: strings.TrimSpace(resolved.ID),
		workspaceRoot:      root,
		agentName:          name,
		agentPath:          authoredAgentPath(&resolved, name),
		soulConfig:         resolved.Config.Agents.Soul,
		heartbeatConfig:    resolved.Config.Agents.Heartbeat,
	}.withAgentArtifacts(h.AgentCatalog, name, &resolved), nil
}

func (t authoredAgentTarget) withAgentArtifacts(
	catalog AgentCatalog,
	agentName string,
	resolved *workspacepkg.ResolvedWorkspace,
) authoredAgentTarget {
	if catalog == nil {
		return t
	}
	resolver, ok := catalog.(session.AgentArtifactResolver)
	if !ok {
		return t
	}
	artifacts, err := resolver.ResolveAgentArtifacts(agentName, resolved)
	if err != nil {
		return t
	}
	if sourcePath := strings.TrimSpace(artifacts.Agent.SourcePath); sourcePath != "" {
		t.agentPath = sourcePath
	}
	t.packageOwned = artifacts.PackageOwned
	t.soulSourcePath = strings.TrimSpace(artifacts.SoulSourcePath)
	t.soulBody = artifacts.SoulBody
	t.heartbeatSourcePath = strings.TrimSpace(artifacts.HeartbeatSourcePath)
	t.heartbeatBody = artifacts.HeartbeatBody
	return t
}

func (t authoredAgentTarget) soulAuthoringTarget() soul.AuthoringTarget {
	return soul.AuthoringTarget{
		WorkspaceID:   t.storageWorkspaceID(),
		WorkspaceRoot: authoredContextSourceRoot(t.workspaceRoot, t.agentPath),
		AgentName:     t.agentName,
		AgentPath:     t.agentPath,
		Config:        t.soulConfig,
		ConfigSource:  "workspace",
	}
}

func (t authoredAgentTarget) heartbeatAuthoringTarget() heartbeat.AuthoringTarget {
	return heartbeat.AuthoringTarget{
		WorkspaceID:   t.storageWorkspaceID(),
		WorkspaceRoot: authoredContextSourceRoot(t.workspaceRoot, t.agentPath),
		AgentName:     t.agentName,
		AgentPath:     t.agentPath,
		Config:        t.heartbeatConfig,
	}
}

func (t authoredAgentTarget) storageWorkspaceID() string {
	if id := strings.TrimSpace(t.sessionWorkspaceID); id != "" {
		return id
	}
	return strings.TrimSpace(t.workspaceID)
}

func authoredContextSourceRoot(workspaceRoot string, agentPath string) string {
	root := strings.TrimSpace(workspaceRoot)
	source := strings.TrimSpace(agentPath)
	if source == "" || !filepath.IsAbs(source) || pathWithinRoot(root, source) {
		return root
	}
	if derived := trustedRootFromAgentSourcePath(source); derived != "" {
		return derived
	}
	return root
}

func trustedRootFromAgentSourcePath(agentPath string) string {
	cleaned := filepath.Clean(strings.TrimSpace(agentPath))
	if !strings.EqualFold(filepath.Base(cleaned), "AGENT.md") {
		return ""
	}
	agentDir := filepath.Dir(cleaned)
	agentsDir := filepath.Dir(agentDir)
	if filepath.Base(agentsDir) != aghconfig.AgentsDirName {
		return ""
	}
	root := filepath.Dir(agentsDir)
	if filepath.Base(root) == aghconfig.DirName {
		return filepath.Dir(root)
	}
	return root
}

func pathWithinRoot(root string, sourcePath string) bool {
	trimmedRoot := strings.TrimSpace(root)
	trimmedSource := strings.TrimSpace(sourcePath)
	if trimmedRoot == "" || trimmedSource == "" {
		return false
	}
	absRoot, err := filepath.Abs(filepath.Clean(trimmedRoot))
	if err != nil {
		return false
	}
	sourceForRoot := filepath.Clean(trimmedSource)
	if !filepath.IsAbs(sourceForRoot) {
		sourceForRoot = filepath.Join(absRoot, sourceForRoot)
	}
	absSource, err := filepath.Abs(sourceForRoot)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absSource)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func (t authoredAgentTarget) soulConfigProvenance() soul.ConfigProvenance {
	provenance, err := soul.NewConfigProvenance(t.soulConfig, "workspace")
	if err != nil {
		return soul.ConfigProvenance{}
	}
	return provenance
}

func (t authoredAgentTarget) resolvePackageOwnedSoul(
	ctx context.Context,
	body *string,
) (soul.ResolvedSoul, error) {
	sourcePath := t.packageOwnedSoulSourcePath()
	if body != nil {
		resolved, err := soul.Parse(ctx, soul.ParseRequest{
			SourcePath:    sourcePath,
			WorkspaceRoot: t.workspaceRoot,
			Content:       []byte(*body),
			Config:        t.soulConfig,
		})
		if err != nil {
			return resolved, err
		}
		return resolved, nil
	}
	if strings.TrimSpace(t.soulBody) == "" {
		return soul.Empty(t.soulConfig, sourcePath)
	}
	resolved, err := soul.Parse(ctx, soul.ParseRequest{
		SourcePath:    sourcePath,
		WorkspaceRoot: t.workspaceRoot,
		Content:       []byte(t.soulBody),
		Config:        t.soulConfig,
	})
	if err != nil {
		return resolved, err
	}
	return resolved, nil
}

func (t authoredAgentTarget) resolvePackageOwnedHeartbeat(
	ctx context.Context,
	body *string,
) (heartbeat.ResolvedPolicy, error) {
	sourcePath := t.packageOwnedHeartbeatSourcePath()
	if body != nil {
		policy, err := heartbeat.Parse(ctx, heartbeat.ParseRequest{
			SourcePath:    sourcePath,
			WorkspaceRoot: t.workspaceRoot,
			Content:       []byte(*body),
			Config:        t.heartbeatConfig,
		})
		if err != nil {
			return policy, err
		}
		return policy, nil
	}
	if strings.TrimSpace(t.heartbeatBody) == "" {
		return heartbeat.Empty(t.heartbeatConfig, sourcePath)
	}
	policy, err := heartbeat.Parse(ctx, heartbeat.ParseRequest{
		SourcePath:    sourcePath,
		WorkspaceRoot: t.workspaceRoot,
		Content:       []byte(t.heartbeatBody),
		Config:        t.heartbeatConfig,
	})
	if err != nil {
		return policy, err
	}
	return policy, nil
}

func (t authoredAgentTarget) rejectPackageOwnedSoulMutation() error {
	if !t.packageOwned {
		return nil
	}
	return fmt.Errorf("%w: package-owned SOUL.md is read-only", soul.ErrAuthoringConflict)
}

func (t authoredAgentTarget) rejectPackageOwnedHeartbeatMutation() error {
	if !t.packageOwned {
		return nil
	}
	return fmt.Errorf("%w: package-owned HEARTBEAT.md is read-only", heartbeat.ErrAuthoringConflict)
}

func (t authoredAgentTarget) packageOwnedSoulSourcePath() string {
	if sourcePath := strings.TrimSpace(t.soulSourcePath); sourcePath != "" {
		return sourcePath
	}
	return authoredSidecarPath(t.agentPath, soul.FileName)
}

func (t authoredAgentTarget) packageOwnedHeartbeatSourcePath() string {
	if sourcePath := strings.TrimSpace(t.heartbeatSourcePath); sourcePath != "" {
		return sourcePath
	}
	return authoredSidecarPath(t.agentPath, heartbeat.FileName)
}

func (h *BaseHandlers) soulActorForRequest() soul.AuthoringIdentity {
	return soul.AuthoringIdentity{
		Kind: "user",
		Ref:  h.transportName(),
	}
}

func (h *BaseHandlers) soulOriginForRequest(c *gin.Context) soul.AuthoringIdentity {
	action := ""
	if c != nil && c.Request != nil {
		action = c.Request.Method + " " + c.FullPath()
	}
	return soul.AuthoringIdentity{
		Kind: h.transportName(),
		Ref:  strings.TrimSpace(action),
	}
}

func (h *BaseHandlers) heartbeatActorForRequest() heartbeat.AuthoringIdentity {
	return heartbeat.AuthoringIdentity{
		Kind: string(heartbeat.ActorKindUser),
		Ref:  h.transportName(),
	}
}

func (h *BaseHandlers) rejectHeartbeatIfMatch(c *gin.Context) bool {
	return h.rejectExpectedDigestHeader(c, "heartbeat_if_match_header_unsupported", StatusForHeartbeatError)
}

func (h *BaseHandlers) rejectSoulIfMatch(c *gin.Context) bool {
	return h.rejectExpectedDigestHeader(c, "soul_if_match_header_unsupported", StatusForSoulError)
}

func (h *BaseHandlers) rejectExpectedDigestHeader(
	c *gin.Context,
	code string,
	statusFor func(error) int,
) bool {
	if strings.TrimSpace(c.GetHeader("If-Match")) == "" {
		return false
	}
	err := newAuthoredValidationError(code + ": use expected_digest in request body")
	h.respondError(c, statusFor(err), err)
	return true
}

func (h *BaseHandlers) sessionHealthPayloadForRoute(c *gin.Context) (contract.SessionHealthPayload, bool) {
	if h.SessionHealth == nil {
		h.respondError(c, StatusForHeartbeatError(errSessionHealthMissing), errSessionHealthMissing)
		return contract.SessionHealthPayload{}, false
	}
	scope, sessionID, _, ok := h.routeSessionInWorkspace(c)
	if !ok {
		return contract.SessionHealthPayload{}, false
	}
	health, err := h.SessionHealth.GetSessionHealth(c.Request.Context(), sessionID)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return contract.SessionHealthPayload{}, false
	}
	if strings.TrimSpace(health.WorkspaceID) != scope.SessionWorkspaceID() {
		h.respondError(c, http.StatusNotFound, errWorkspaceScopedResourceNotFound)
		return contract.SessionHealthPayload{}, false
	}
	payload, err := contract.SessionHealthPayloadFromDomain(health)
	if err != nil {
		h.respondError(c, StatusForHeartbeatError(err), err)
		return contract.SessionHealthPayload{}, false
	}
	return payload, true
}

func (h *BaseHandlers) sessionPayloadsWithOptionalHealth(
	ctx context.Context,
	infos []*session.Info,
	includeHealth bool,
) ([]contract.SessionPayload, error) {
	payloads := make([]contract.SessionPayload, 0, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		payload, err := h.sessionPayloadWithOptionalHealth(ctx, info, includeHealth)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}
	return payloads, nil
}

func (h *BaseHandlers) sessionPayloadWithOptionalHealth(
	ctx context.Context,
	info *session.Info,
	includeHealth bool,
) (contract.SessionPayload, error) {
	payload := SessionPayloadFromInfo(info)
	if !includeHealth {
		return payload, nil
	}
	if h.SessionHealth == nil {
		return contract.SessionPayload{}, errSessionHealthMissing
	}
	health, err := h.SessionHealth.GetSessionHealth(ctx, payload.ID)
	if err != nil {
		return contract.SessionPayload{}, err
	}
	payload.Badge = session.BadgeForHealth(info, health)
	converted, err := contract.SessionHealthPayloadFromDomain(health)
	if err != nil {
		return contract.SessionPayload{}, err
	}
	payload.Health = &converted
	return payload, nil
}

func (h *BaseHandlers) heartbeatStatusForHealth(
	ctx context.Context,
	health contract.SessionHealthPayload,
	includeHealth bool,
) (contract.HeartbeatStatusResponse, error) {
	target, err := h.resolveAuthoredAgentTarget(ctx, health.WorkspaceID, health.AgentName)
	if err != nil {
		return contract.HeartbeatStatusResponse{}, err
	}
	result, err := h.HeartbeatStatus.Status(ctx, heartbeat.StatusRequest{
		Target:               target.heartbeatAuthoringTarget(),
		SessionID:            health.SessionID,
		IncludeSessionHealth: includeHealth,
	})
	if err != nil {
		return contract.HeartbeatStatusResponse{}, err
	}
	return contract.HeartbeatStatusResponseFromResult(&result)
}

func (h *BaseHandlers) heartbeatWakeEvents(
	ctx context.Context,
	target authoredAgentTarget,
	sessionID string,
) ([]contract.HeartbeatWakeEventPayload, error) {
	if h.HeartbeatWakeEvents == nil {
		return nil, nil
	}
	events, err := h.HeartbeatWakeEvents.ListHeartbeatWakeEvents(ctx, heartbeat.WakeEventListQuery{
		WorkspaceID: target.storageWorkspaceID(),
		AgentName:   target.agentName,
		SessionID:   strings.TrimSpace(sessionID),
		Limit:       defaultWakeEventInspectLimit,
	})
	if err != nil {
		return nil, err
	}
	payload := make([]contract.HeartbeatWakeEventPayload, 0, len(events))
	for _, event := range events {
		converted, convertErr := contract.HeartbeatWakeEventPayloadFromDomain(event)
		if convertErr != nil {
			return nil, convertErr
		}
		payload = append(payload, converted)
	}
	return payload, nil
}

func soulMutationResponse(
	target authoredAgentTarget,
	result *soul.MutationResult,
) (contract.AgentSoulMutationResponse, error) {
	if result == nil {
		return contract.AgentSoulMutationResponse{}, errors.New("soul mutation result is required")
	}
	payload := contract.AgentSoulPayloadFromResolved(
		target.agentName,
		&result.Soul,
		result.Snapshot.ID,
		target.soulConfigProvenance(),
	)
	payload.RevisionID = result.Revision.ID
	revision, err := contract.AgentSoulRevisionPayloadFromDomain(result.Revision)
	if err != nil {
		return contract.AgentSoulMutationResponse{}, err
	}
	return contract.AgentSoulMutationResponse{Soul: payload, Revision: revision}, nil
}

func soulHistoryResponse(result soul.HistoryResult) (contract.AgentSoulHistoryResponse, error) {
	revisions := make([]contract.AgentSoulRevisionPayload, 0, len(result.Revisions))
	for _, revision := range result.Revisions {
		converted, err := contract.AgentSoulRevisionPayloadFromDomain(revision)
		if err != nil {
			return contract.AgentSoulHistoryResponse{}, err
		}
		revisions = append(revisions, converted)
	}
	return contract.AgentSoulHistoryResponse{Revisions: revisions}, nil
}

func heartbeatMutationResponse(result *heartbeat.MutationResult) (contract.HeartbeatMutationResponse, error) {
	if result == nil {
		return contract.HeartbeatMutationResponse{}, errors.New("heartbeat mutation result is required")
	}
	policy, err := contract.HeartbeatPolicyPayloadFromResolved(
		result.Revision.AgentName,
		&result.Policy,
		result.Snapshot.ID,
	)
	if err != nil {
		return contract.HeartbeatMutationResponse{}, err
	}
	revision, err := contract.HeartbeatRevisionPayloadFromDomain(result.Revision)
	if err != nil {
		return contract.HeartbeatMutationResponse{}, err
	}
	return contract.HeartbeatMutationResponse{Heartbeat: policy, Revision: revision}, nil
}

func heartbeatHistoryResponse(result heartbeat.HistoryResult) (contract.HeartbeatHistoryResponse, error) {
	revisions := make([]contract.HeartbeatRevisionPayload, 0, len(result.Revisions))
	for _, revision := range result.Revisions {
		converted, err := contract.HeartbeatRevisionPayloadFromDomain(revision)
		if err != nil {
			return contract.HeartbeatHistoryResponse{}, err
		}
		revisions = append(revisions, converted)
	}
	return contract.HeartbeatHistoryResponse{Revisions: revisions}, nil
}

func agentSoulPayloadFromRefreshResult(result session.SoulRefreshResult) (contract.AgentSoulPayload, error) {
	if result.Soul != nil {
		snapshotID := ""
		provenance := result.ConfigProvenance
		if result.Snapshot != nil {
			snapshotID = result.Snapshot.ID
			envelope, err := result.Snapshot.ProfileEnvelope()
			if err == nil && strings.TrimSpace(provenance.Digest) == "" {
				provenance = envelope.ConfigProvenance
			}
		}
		return contract.AgentSoulPayloadFromResolved(result.AgentName, result.Soul, snapshotID, provenance), nil
	}
	if result.Snapshot == nil {
		return contract.AgentSoulPayload{}, nil
	}
	return agentSoulPayloadFromSnapshot(*result.Snapshot)
}

func agentSoulPayloadFromSnapshot(snapshot soul.Snapshot) (contract.AgentSoulPayload, error) {
	envelope, err := snapshot.ProfileEnvelope()
	if err != nil {
		return contract.AgentSoulPayload{}, err
	}
	resolved := soul.ResolvedSoul{
		Enabled:     envelope.ReadModel.Enabled,
		Present:     envelope.Present,
		Active:      envelope.Active,
		Valid:       envelope.Valid,
		SourcePath:  snapshot.SourcePath,
		Digest:      snapshot.Digest,
		Profile:     envelope.Profile,
		Compact:     envelope.Compact,
		ReadModel:   envelope.ReadModel,
		Diagnostics: envelope.Diagnostics,
	}
	return contract.AgentSoulPayloadFromResolved(
		snapshot.AgentName,
		&resolved,
		snapshot.ID,
		envelope.ConfigProvenance,
	), nil
}

func statusForHeartbeatWakeDecision(decision heartbeat.WakeDecision) int {
	switch decision.Result {
	case heartbeat.WakeResultSent:
		return http.StatusOK
	case heartbeat.WakeResultFailed:
		return http.StatusInternalServerError
	default:
		return http.StatusConflict
	}
}

func decodeAuthoredJSONBody(c *gin.Context, dst any, allowEmpty bool) error {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		if allowEmpty {
			return nil
		}
		return errors.New("request body is required")
	}
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) && allowEmpty {
			return nil
		}
		return fmt.Errorf("decode request body: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode request body: multiple JSON values are not allowed")
		}
		return fmt.Errorf("decode request body: %w", err)
	}
	return nil
}

func authoredWorkspaceRefFromQuery(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return firstNonEmpty(c.Query("workspace_id"), c.Query("workspace"))
}

func pathAgentName(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return firstNonEmpty(c.Param("name"), c.Param("agent_name"))
}

func authoredRouteAgentName(pathName string) (string, error) {
	path := strings.TrimSpace(pathName)
	if path != "" {
		return path, nil
	}
	return "", newAuthoredValidationError("agent_name is required")
}

func authoredAgentPath(workspace *workspacepkg.ResolvedWorkspace, agentName string) string {
	name := strings.TrimSpace(agentName)
	if workspace == nil {
		return ""
	}
	for _, agent := range workspace.Agents {
		if strings.TrimSpace(agent.Name) == name && strings.TrimSpace(agent.SourcePath) != "" {
			return strings.TrimSpace(agent.SourcePath)
		}
	}
	if root := strings.TrimSpace(workspace.RootDir); root != "" && name != "" {
		return filepath.Join(root, aghconfig.DirName, aghconfig.AgentsDirName, name, "AGENT.md")
	}
	return ""
}

func authoredSidecarPath(agentPath string, fileName string) string {
	trimmedAgentPath := strings.TrimSpace(agentPath)
	trimmedFileName := strings.TrimSpace(fileName)
	if trimmedAgentPath == "" {
		return trimmedFileName
	}
	return filepath.ToSlash(filepath.Join(filepath.Dir(trimmedAgentPath), trimmedFileName))
}

func newAuthoredValidationError(message string) error {
	return fmt.Errorf("%w: %s", errAuthoredContextValidation, strings.TrimSpace(message))
}
