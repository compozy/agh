package extensionpkg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	apicontract "github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	"github.com/pedronauck/agh/internal/heartbeat"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/soul"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const defaultHostAPIWakeEventLimit = 10

var errHostAPIAuthoredValidation = errors.New("extension: authored context validation")

type hostAPISoulAuthoringService interface {
	soul.AuthoringService
}

type hostAPISoulRefresher interface {
	RefreshSoulWithExpectedDigest(context.Context, string, string) (session.SoulRefreshResult, error)
}

type hostAPIHeartbeatAuthoringService interface {
	heartbeat.AuthoringService
}

type hostAPIHeartbeatStatusService interface {
	heartbeat.StatusService
}

type hostAPIHeartbeatWakeService interface {
	Wake(context.Context, heartbeat.WakeRequest) (heartbeat.WakeDecision, error)
}

type hostAPISessionHealthReader interface {
	GetSessionHealth(context.Context, string) (heartbeat.SessionHealth, error)
}

type hostAPIHeartbeatWakeEventReader interface {
	ListHeartbeatWakeEvents(context.Context, heartbeat.WakeEventListQuery) ([]heartbeat.WakeEvent, error)
}

type hostAPIAgentSoulGetParams = extensioncontract.AgentSoulGetParams
type hostAPIAgentSoulValidateParams = extensioncontract.AgentSoulValidateParams
type hostAPIAgentSoulPutParams = extensioncontract.AgentSoulPutParams
type hostAPIAgentSoulDeleteParams = extensioncontract.AgentSoulDeleteParams
type hostAPIAgentSoulHistoryParams = extensioncontract.AgentSoulHistoryParams
type hostAPIAgentSoulRollbackParams = extensioncontract.AgentSoulRollbackParams
type hostAPIAgentHeartbeatGetParams = extensioncontract.AgentHeartbeatGetParams
type hostAPIAgentHeartbeatValidateParams = extensioncontract.AgentHeartbeatValidateParams
type hostAPIAgentHeartbeatPutParams = extensioncontract.AgentHeartbeatPutParams
type hostAPIAgentHeartbeatDeleteParams = extensioncontract.AgentHeartbeatDeleteParams
type hostAPIAgentHeartbeatHistoryParams = extensioncontract.AgentHeartbeatHistoryParams
type hostAPIAgentHeartbeatRollbackParams = extensioncontract.AgentHeartbeatRollbackParams
type hostAPIAgentHeartbeatStatusParams = extensioncontract.AgentHeartbeatStatusParams
type hostAPIAgentHeartbeatWakeParams = extensioncontract.AgentHeartbeatWakeParams
type hostAPISessionSoulRefreshParams = extensioncontract.SessionSoulRefreshParams
type hostAPISessionHealthGetParams = extensioncontract.SessionHealthGetParams
type hostAPISessionStatusGetParams = extensioncontract.SessionStatusGetParams

type hostAPIAuthoredAgentTarget struct {
	workspaceID     string
	workspaceRoot   string
	agentName       string
	agentPath       string
	soulConfig      aghconfig.SoulConfig
	heartbeatConfig aghconfig.HeartbeatConfig
}

func (h *HostAPIHandler) handleAgentsSoulGet(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.soulAuthoring == nil {
		return nil, unavailableRPCError(errors.New("extension: soul authoring service is not configured"))
	}
	var params hostAPIAgentSoulGetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	target, err := h.resolveHostAPIAuthoredAgentTarget(ctx, params.WorkspaceID, params.AgentName)
	if err != nil {
		return nil, mapHostAPISoulRPCError(err)
	}
	result, err := h.soulAuthoring.Validate(ctx, soul.ValidateRequest{Target: target.soulAuthoringTarget()})
	return hostAPISoulPayload(target, &result.Soul, "", err)
}

func (h *HostAPIHandler) handleAgentsSoulValidate(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.soulAuthoring == nil {
		return nil, unavailableRPCError(errors.New("extension: soul authoring service is not configured"))
	}
	var params hostAPIAgentSoulValidateParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	target, err := h.resolveHostAPIAuthoredAgentTarget(ctx, params.WorkspaceID, params.AgentName)
	if err != nil {
		return nil, mapHostAPISoulRPCError(err)
	}
	var body *string
	if strings.TrimSpace(params.Body) != "" {
		body = &params.Body
	}
	result, err := h.soulAuthoring.Validate(ctx, soul.ValidateRequest{
		Target: target.soulAuthoringTarget(),
		Body:   body,
	})
	return hostAPISoulPayload(target, &result.Soul, "", err)
}

func (h *HostAPIHandler) handleAgentsSoulPut(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.soulAuthoring == nil {
		return nil, unavailableRPCError(errors.New("extension: soul authoring service is not configured"))
	}
	var params hostAPIAgentSoulPutParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	target, err := h.resolveHostAPIAuthoredAgentTarget(ctx, params.WorkspaceID, params.AgentName)
	if err != nil {
		return nil, mapHostAPISoulRPCError(err)
	}
	result, err := h.soulAuthoring.Put(ctx, soul.PutRequest{
		Target:         target.soulAuthoringTarget(),
		Body:           params.Body,
		ExpectedDigest: strings.TrimSpace(params.ExpectedDigest),
		Actor:          hostAPISoulActor(ctx),
		Origin:         hostAPISoulOrigin(extensioncontract.HostAPIMethodAgentsSoulPut),
	})
	if err != nil {
		return nil, mapHostAPISoulRPCError(err)
	}
	return hostAPISoulMutationResponse(target, &result)
}

func (h *HostAPIHandler) handleAgentsSoulDelete(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.soulAuthoring == nil {
		return nil, unavailableRPCError(errors.New("extension: soul authoring service is not configured"))
	}
	var params hostAPIAgentSoulDeleteParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	target, err := h.resolveHostAPIAuthoredAgentTarget(ctx, params.WorkspaceID, params.AgentName)
	if err != nil {
		return nil, mapHostAPISoulRPCError(err)
	}
	result, err := h.soulAuthoring.Delete(ctx, soul.DeleteRequest{
		Target:         target.soulAuthoringTarget(),
		ExpectedDigest: strings.TrimSpace(params.ExpectedDigest),
		Actor:          hostAPISoulActor(ctx),
		Origin:         hostAPISoulOrigin(extensioncontract.HostAPIMethodAgentsSoulDelete),
	})
	if err != nil {
		return nil, mapHostAPISoulRPCError(err)
	}
	return hostAPISoulMutationResponse(target, &result)
}

func (h *HostAPIHandler) handleAgentsSoulHistory(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.soulAuthoring == nil {
		return nil, unavailableRPCError(errors.New("extension: soul authoring service is not configured"))
	}
	var params hostAPIAgentSoulHistoryParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	target, err := h.resolveHostAPIAuthoredAgentTarget(ctx, params.WorkspaceID, params.AgentName)
	if err != nil {
		return nil, mapHostAPISoulRPCError(err)
	}
	result, err := h.soulAuthoring.History(ctx, soul.HistoryRequest{
		Target: target.soulAuthoringTarget(),
		Limit:  params.Limit,
	})
	if err != nil {
		return nil, mapHostAPISoulRPCError(err)
	}
	return hostAPISoulHistoryResponse(result)
}

func (h *HostAPIHandler) handleAgentsSoulRollback(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.soulAuthoring == nil {
		return nil, unavailableRPCError(errors.New("extension: soul authoring service is not configured"))
	}
	var params hostAPIAgentSoulRollbackParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	target, err := h.resolveHostAPIAuthoredAgentTarget(ctx, params.WorkspaceID, params.AgentName)
	if err != nil {
		return nil, mapHostAPISoulRPCError(err)
	}
	result, err := h.soulAuthoring.Rollback(ctx, soul.RollbackRequest{
		Target:         target.soulAuthoringTarget(),
		RevisionID:     strings.TrimSpace(params.RevisionID),
		ExpectedDigest: strings.TrimSpace(params.ExpectedDigest),
		Actor:          hostAPISoulActor(ctx),
		Origin:         hostAPISoulOrigin(extensioncontract.HostAPIMethodAgentsSoulRollback),
	})
	if err != nil {
		return nil, mapHostAPISoulRPCError(err)
	}
	return hostAPISoulMutationResponse(target, &result)
}

func (h *HostAPIHandler) handleSessionsSoulRefresh(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.soulRefresher == nil {
		return nil, unavailableRPCError(errors.New("extension: soul refresh service is not configured"))
	}
	var params hostAPISessionSoulRefreshParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	sessionID := strings.TrimSpace(params.SessionID)
	if sessionID == "" {
		return nil, invalidParamsRPCError(errors.New("session_id is required"))
	}
	result, err := h.soulRefresher.RefreshSoulWithExpectedDigest(
		ctx,
		sessionID,
		strings.TrimSpace(params.ExpectedDigest),
	)
	if err != nil {
		return nil, mapHostAPISoulRPCError(err)
	}
	return hostAPIAgentSoulPayloadFromRefreshResult(result)
}

func (h *HostAPIHandler) handleAgentsHeartbeatGet(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.heartbeatStatus == nil {
		return nil, unavailableRPCError(errors.New("extension: heartbeat status service is not configured"))
	}
	var params hostAPIAgentHeartbeatGetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	target, err := h.resolveHostAPIAuthoredAgentTarget(ctx, params.WorkspaceID, params.AgentName)
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	result, err := h.heartbeatStatus.Inspect(ctx, heartbeat.InspectRequest{Target: target.heartbeatAuthoringTarget()})
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	snapshotID := ""
	if result.Snapshot != nil {
		snapshotID = strings.TrimSpace(result.Snapshot.ID)
	}
	return apicontract.HeartbeatPolicyPayloadFromResolved(result.AgentName, &result.Policy, snapshotID)
}

func (h *HostAPIHandler) handleAgentsHeartbeatValidate(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.heartbeatAuthor == nil {
		return nil, unavailableRPCError(errors.New("extension: heartbeat authoring service is not configured"))
	}
	var params hostAPIAgentHeartbeatValidateParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	target, err := h.resolveHostAPIAuthoredAgentTarget(ctx, params.WorkspaceID, params.AgentName)
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	body := params.Body
	result, err := h.heartbeatAuthor.Validate(ctx, heartbeat.ValidateRequest{
		Target: target.heartbeatAuthoringTarget(),
		Body:   &body,
	})
	return hostAPIHeartbeatPayload(target, &result.Policy, "", err)
}

func (h *HostAPIHandler) handleAgentsHeartbeatPut(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.heartbeatAuthor == nil {
		return nil, unavailableRPCError(errors.New("extension: heartbeat authoring service is not configured"))
	}
	var params hostAPIAgentHeartbeatPutParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	target, err := h.resolveHostAPIAuthoredAgentTarget(ctx, params.WorkspaceID, params.AgentName)
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	result, err := h.heartbeatAuthor.Put(ctx, heartbeat.PutRequest{
		Target:         target.heartbeatAuthoringTarget(),
		Body:           params.Body,
		ExpectedDigest: strings.TrimSpace(params.ExpectedDigest),
		Actor:          hostAPIHeartbeatActor(ctx),
	})
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	return hostAPIHeartbeatMutationResponse(&result)
}

func (h *HostAPIHandler) handleAgentsHeartbeatDelete(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.heartbeatAuthor == nil {
		return nil, unavailableRPCError(errors.New("extension: heartbeat authoring service is not configured"))
	}
	var params hostAPIAgentHeartbeatDeleteParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	target, err := h.resolveHostAPIAuthoredAgentTarget(ctx, params.WorkspaceID, params.AgentName)
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	result, err := h.heartbeatAuthor.Delete(ctx, heartbeat.DeleteRequest{
		Target:         target.heartbeatAuthoringTarget(),
		ExpectedDigest: strings.TrimSpace(params.ExpectedDigest),
		Actor:          hostAPIHeartbeatActor(ctx),
	})
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	return hostAPIHeartbeatMutationResponse(&result)
}

func (h *HostAPIHandler) handleAgentsHeartbeatHistory(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.heartbeatAuthor == nil {
		return nil, unavailableRPCError(errors.New("extension: heartbeat authoring service is not configured"))
	}
	var params hostAPIAgentHeartbeatHistoryParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	target, err := h.resolveHostAPIAuthoredAgentTarget(ctx, params.WorkspaceID, params.AgentName)
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	result, err := h.heartbeatAuthor.History(ctx, heartbeat.HistoryRequest{
		Target: target.heartbeatAuthoringTarget(),
		Limit:  params.Limit,
	})
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	return hostAPIHeartbeatHistoryResponse(result)
}

func (h *HostAPIHandler) handleAgentsHeartbeatRollback(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.heartbeatAuthor == nil {
		return nil, unavailableRPCError(errors.New("extension: heartbeat authoring service is not configured"))
	}
	var params hostAPIAgentHeartbeatRollbackParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	target, err := h.resolveHostAPIAuthoredAgentTarget(ctx, params.WorkspaceID, params.AgentName)
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	result, err := h.heartbeatAuthor.Rollback(ctx, heartbeat.RollbackRequest{
		Target:         target.heartbeatAuthoringTarget(),
		RevisionID:     strings.TrimSpace(params.RevisionID),
		TargetDigest:   strings.TrimSpace(params.TargetDigest),
		ExpectedDigest: strings.TrimSpace(params.ExpectedDigest),
		Actor:          hostAPIHeartbeatActor(ctx),
	})
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	return hostAPIHeartbeatMutationResponse(&result)
}

func (h *HostAPIHandler) handleAgentsHeartbeatStatus(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.heartbeatStatus == nil {
		return nil, unavailableRPCError(errors.New("extension: heartbeat status service is not configured"))
	}
	var params hostAPIAgentHeartbeatStatusParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	target, err := h.resolveHostAPIAuthoredAgentTarget(ctx, params.WorkspaceID, params.AgentName)
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	result, err := h.heartbeatStatus.Status(ctx, heartbeat.StatusRequest{
		Target:               target.heartbeatAuthoringTarget(),
		SessionID:            strings.TrimSpace(params.SessionID),
		IncludeSessionHealth: params.IncludeSessionHealth,
	})
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	response, err := apicontract.HeartbeatStatusResponseFromResult(&result)
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	if params.IncludeRecentWakeEvents {
		events, err := h.hostAPIWakeEvents(ctx, target, params.SessionID)
		if err != nil {
			return nil, mapHostAPIHeartbeatRPCError(err)
		}
		response.WakeEvents = events
	}
	return response, nil
}

func (h *HostAPIHandler) handleAgentsHeartbeatWake(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.heartbeatWake == nil {
		return nil, unavailableRPCError(errors.New("extension: heartbeat wake service is not configured"))
	}
	var params hostAPIAgentHeartbeatWakeParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	target, err := h.resolveHostAPIAuthoredAgentTarget(ctx, params.WorkspaceID, params.AgentName)
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	source := heartbeat.WakeSource(params.Source)
	if source == "" {
		source = heartbeat.WakeSourceManual
	}
	decision, err := h.heartbeatWake.Wake(ctx, heartbeat.WakeRequest{
		WorkspaceID: target.workspaceID,
		AgentName:   target.agentName,
		SessionID:   strings.TrimSpace(params.SessionID),
		Source:      source,
		DryRun:      params.DryRun,
	})
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	payload, err := apicontract.HeartbeatWakeDecisionPayloadFromDomain(decision)
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	return apicontract.HeartbeatWakeResponse{Decision: payload}, nil
}

func (h *HostAPIHandler) handleSessionsHealthGet(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPISessionHealthGetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	health, err := h.hostAPISessionHealthPayload(ctx, params.SessionID)
	if err != nil {
		return nil, err
	}
	return apicontract.SessionHealthResponse{Health: health}, nil
}

func (h *HostAPIHandler) handleSessionsStatusGet(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPISessionStatusGetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	health, err := h.hostAPISessionHealthPayload(ctx, params.SessionID)
	if err != nil {
		return nil, err
	}
	response := apicontract.SessionStatusResponse{
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
	if h.heartbeatStatus == nil {
		return response, nil
	}
	target, err := h.resolveHostAPIAuthoredAgentTarget(ctx, health.WorkspaceID, health.AgentName)
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	status, err := h.heartbeatStatus.Status(ctx, heartbeat.StatusRequest{
		Target:    target.heartbeatAuthoringTarget(),
		SessionID: health.SessionID,
	})
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	converted, err := apicontract.HeartbeatStatusResponseFromResult(&status)
	if err != nil {
		return nil, mapHostAPIHeartbeatRPCError(err)
	}
	response.WakeState = converted.WakeState
	return response, nil
}

func (h *HostAPIHandler) resolveHostAPIAuthoredAgentTarget(
	ctx context.Context,
	workspaceRef string,
	agentName string,
) (hostAPIAuthoredAgentTarget, error) {
	name := strings.TrimSpace(agentName)
	if name == "" {
		return hostAPIAuthoredAgentTarget{}, fmt.Errorf("%w: agent_name is required", errHostAPIAuthoredValidation)
	}
	ref := strings.TrimSpace(workspaceRef)
	if ref == "" {
		return hostAPIAuthoredAgentTarget{}, fmt.Errorf("%w: workspace_id is required", errHostAPIAuthoredValidation)
	}
	if h.workspaces == nil {
		return hostAPIAuthoredAgentTarget{}, workspacepkg.ErrWorkspaceResolverUnavailable
	}
	resolved, err := h.workspaces.Resolve(ctx, ref)
	if err != nil {
		return hostAPIAuthoredAgentTarget{}, err
	}
	root := strings.TrimSpace(resolved.RootDir)
	if root == "" {
		return hostAPIAuthoredAgentTarget{}, workspacepkg.ErrWorkspaceRootMissing
	}
	return hostAPIAuthoredAgentTarget{
		workspaceID:     strings.TrimSpace(resolved.ID),
		workspaceRoot:   root,
		agentName:       name,
		agentPath:       hostAPIAuthoredAgentPath(&resolved, name),
		soulConfig:      resolved.Config.Agents.Soul,
		heartbeatConfig: resolved.Config.Agents.Heartbeat,
	}, nil
}

func (t hostAPIAuthoredAgentTarget) soulAuthoringTarget() soul.AuthoringTarget {
	return soul.AuthoringTarget{
		WorkspaceID:   t.workspaceID,
		WorkspaceRoot: t.workspaceRoot,
		AgentName:     t.agentName,
		AgentPath:     t.agentPath,
		Config:        t.soulConfig,
		ConfigSource:  "workspace",
	}
}

func (t hostAPIAuthoredAgentTarget) heartbeatAuthoringTarget() heartbeat.AuthoringTarget {
	return heartbeat.AuthoringTarget{
		WorkspaceID:   t.workspaceID,
		WorkspaceRoot: t.workspaceRoot,
		AgentName:     t.agentName,
		AgentPath:     t.agentPath,
		Config:        t.heartbeatConfig,
	}
}

func (t hostAPIAuthoredAgentTarget) soulConfigProvenance() soul.ConfigProvenance {
	provenance, err := soul.NewConfigProvenance(t.soulConfig, "workspace")
	if err != nil {
		return soul.ConfigProvenance{}
	}
	return provenance
}

func hostAPIAuthoredAgentPath(workspace *workspacepkg.ResolvedWorkspace, agentName string) string {
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

func hostAPISoulPayload(
	target hostAPIAuthoredAgentTarget,
	resolved *soul.ResolvedSoul,
	snapshotID string,
	err error,
) (apicontract.AgentSoulPayload, error) {
	if err != nil && !errors.Is(err, soul.ErrInvalid) {
		return apicontract.AgentSoulPayload{}, mapHostAPISoulRPCError(err)
	}
	payload := apicontract.AgentSoulPayloadFromResolved(
		target.agentName,
		resolved,
		strings.TrimSpace(snapshotID),
		target.soulConfigProvenance(),
	)
	if err := apicontract.ValidateAuthoredContextRedacted(payload); err != nil {
		return apicontract.AgentSoulPayload{}, mapHostAPISoulRPCError(err)
	}
	return payload, nil
}

func hostAPIHeartbeatPayload(
	target hostAPIAuthoredAgentTarget,
	policy *heartbeat.ResolvedPolicy,
	snapshotID string,
	err error,
) (apicontract.HeartbeatPolicyPayload, error) {
	if err != nil && !errors.Is(err, heartbeat.ErrInvalid) {
		return apicontract.HeartbeatPolicyPayload{}, mapHostAPIHeartbeatRPCError(err)
	}
	payload, err := apicontract.HeartbeatPolicyPayloadFromResolved(
		target.agentName,
		policy,
		strings.TrimSpace(snapshotID),
	)
	if err != nil {
		return apicontract.HeartbeatPolicyPayload{}, mapHostAPIHeartbeatRPCError(err)
	}
	if err := apicontract.ValidateAuthoredContextRedacted(payload); err != nil {
		return apicontract.HeartbeatPolicyPayload{}, mapHostAPIHeartbeatRPCError(err)
	}
	return payload, nil
}

func hostAPISoulMutationResponse(
	target hostAPIAuthoredAgentTarget,
	result *soul.MutationResult,
) (apicontract.AgentSoulMutationResponse, error) {
	if result == nil {
		return apicontract.AgentSoulMutationResponse{}, mapHostAPISoulRPCError(errors.New("soul result is required"))
	}
	payload := apicontract.AgentSoulPayloadFromResolved(
		target.agentName,
		&result.Soul,
		result.Snapshot.ID,
		target.soulConfigProvenance(),
	)
	payload.RevisionID = result.Revision.ID
	revision, err := apicontract.AgentSoulRevisionPayloadFromDomain(result.Revision)
	if err != nil {
		return apicontract.AgentSoulMutationResponse{}, mapHostAPISoulRPCError(err)
	}
	response := apicontract.AgentSoulMutationResponse{Soul: payload, Revision: revision}
	if err := apicontract.ValidateAuthoredContextRedacted(response); err != nil {
		return apicontract.AgentSoulMutationResponse{}, mapHostAPISoulRPCError(err)
	}
	return response, nil
}

func hostAPISoulHistoryResponse(result soul.HistoryResult) (apicontract.AgentSoulHistoryResponse, error) {
	revisions := make([]apicontract.AgentSoulRevisionPayload, 0, len(result.Revisions))
	for _, revision := range result.Revisions {
		converted, err := apicontract.AgentSoulRevisionPayloadFromDomain(revision)
		if err != nil {
			return apicontract.AgentSoulHistoryResponse{}, mapHostAPISoulRPCError(err)
		}
		revisions = append(revisions, converted)
	}
	response := apicontract.AgentSoulHistoryResponse{Revisions: revisions}
	if err := apicontract.ValidateAuthoredContextRedacted(response); err != nil {
		return apicontract.AgentSoulHistoryResponse{}, mapHostAPISoulRPCError(err)
	}
	return response, nil
}

func hostAPIHeartbeatMutationResponse(
	result *heartbeat.MutationResult,
) (apicontract.HeartbeatMutationResponse, error) {
	if result == nil {
		return apicontract.HeartbeatMutationResponse{}, mapHostAPIHeartbeatRPCError(
			errors.New("heartbeat result is required"),
		)
	}
	policy, err := apicontract.HeartbeatPolicyPayloadFromResolved(
		result.Revision.AgentName,
		&result.Policy,
		result.Snapshot.ID,
	)
	if err != nil {
		return apicontract.HeartbeatMutationResponse{}, mapHostAPIHeartbeatRPCError(err)
	}
	revision, err := apicontract.HeartbeatRevisionPayloadFromDomain(result.Revision)
	if err != nil {
		return apicontract.HeartbeatMutationResponse{}, mapHostAPIHeartbeatRPCError(err)
	}
	response := apicontract.HeartbeatMutationResponse{Heartbeat: policy, Revision: revision}
	if err := apicontract.ValidateAuthoredContextRedacted(response); err != nil {
		return apicontract.HeartbeatMutationResponse{}, mapHostAPIHeartbeatRPCError(err)
	}
	return response, nil
}

func hostAPIHeartbeatHistoryResponse(result heartbeat.HistoryResult) (apicontract.HeartbeatHistoryResponse, error) {
	revisions := make([]apicontract.HeartbeatRevisionPayload, 0, len(result.Revisions))
	for _, revision := range result.Revisions {
		converted, err := apicontract.HeartbeatRevisionPayloadFromDomain(revision)
		if err != nil {
			return apicontract.HeartbeatHistoryResponse{}, mapHostAPIHeartbeatRPCError(err)
		}
		revisions = append(revisions, converted)
	}
	response := apicontract.HeartbeatHistoryResponse{Revisions: revisions}
	if err := apicontract.ValidateAuthoredContextRedacted(response); err != nil {
		return apicontract.HeartbeatHistoryResponse{}, mapHostAPIHeartbeatRPCError(err)
	}
	return response, nil
}

func (h *HostAPIHandler) hostAPISessionHealthPayload(
	ctx context.Context,
	sessionID string,
) (apicontract.SessionHealthPayload, error) {
	if h.sessionHealth == nil {
		return apicontract.SessionHealthPayload{}, unavailableRPCError(
			errors.New("extension: session health service is not configured"),
		)
	}
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return apicontract.SessionHealthPayload{}, invalidParamsRPCError(errors.New("session_id is required"))
	}
	health, err := h.sessionHealth.GetSessionHealth(ctx, id)
	if err != nil {
		return apicontract.SessionHealthPayload{}, mapHostAPIHeartbeatRPCError(err)
	}
	payload, err := apicontract.SessionHealthPayloadFromDomain(health)
	if err != nil {
		return apicontract.SessionHealthPayload{}, mapHostAPIHeartbeatRPCError(err)
	}
	if err := apicontract.ValidateAuthoredContextRedacted(payload); err != nil {
		return apicontract.SessionHealthPayload{}, mapHostAPIHeartbeatRPCError(err)
	}
	return payload, nil
}

func (h *HostAPIHandler) hostAPIWakeEvents(
	ctx context.Context,
	target hostAPIAuthoredAgentTarget,
	sessionID string,
) ([]apicontract.HeartbeatWakeEventPayload, error) {
	if h.wakeEvents == nil {
		return nil, nil
	}
	events, err := h.wakeEvents.ListHeartbeatWakeEvents(ctx, heartbeat.WakeEventListQuery{
		WorkspaceID: target.workspaceID,
		AgentName:   target.agentName,
		SessionID:   strings.TrimSpace(sessionID),
		Limit:       defaultHostAPIWakeEventLimit,
	})
	if err != nil {
		return nil, err
	}
	payload := make([]apicontract.HeartbeatWakeEventPayload, 0, len(events))
	for _, event := range events {
		converted, err := apicontract.HeartbeatWakeEventPayloadFromDomain(event)
		if err != nil {
			return nil, err
		}
		payload = append(payload, converted)
	}
	return payload, apicontract.ValidateAuthoredContextRedacted(payload)
}

func hostAPIAgentSoulPayloadFromRefreshResult(
	result session.SoulRefreshResult,
) (apicontract.AgentSoulPayload, error) {
	if result.Soul != nil {
		snapshotID := ""
		provenance := result.ConfigProvenance
		if result.Snapshot != nil {
			snapshotID = strings.TrimSpace(result.Snapshot.ID)
			envelope, err := result.Snapshot.ProfileEnvelope()
			if err == nil && strings.TrimSpace(provenance.Digest) == "" {
				provenance = envelope.ConfigProvenance
			}
		}
		payload := apicontract.AgentSoulPayloadFromResolved(result.AgentName, result.Soul, snapshotID, provenance)
		return payload, apicontract.ValidateAuthoredContextRedacted(payload)
	}
	if result.Snapshot == nil {
		return apicontract.AgentSoulPayload{}, nil
	}
	payload, err := hostAPIAgentSoulPayloadFromSnapshot(*result.Snapshot)
	if err != nil {
		return apicontract.AgentSoulPayload{}, err
	}
	return payload, apicontract.ValidateAuthoredContextRedacted(payload)
}

func hostAPIAgentSoulPayloadFromSnapshot(snapshot soul.Snapshot) (apicontract.AgentSoulPayload, error) {
	envelope, err := snapshot.ProfileEnvelope()
	if err != nil {
		return apicontract.AgentSoulPayload{}, mapHostAPISoulRPCError(err)
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
	return apicontract.AgentSoulPayloadFromResolved(
		snapshot.AgentName,
		&resolved,
		snapshot.ID,
		envelope.ConfigProvenance,
	), nil
}

func hostAPISoulActor(ctx context.Context) soul.AuthoringIdentity {
	return soul.AuthoringIdentity{
		Kind: "extension",
		Ref:  hostAPIExtensionNameFromContext(ctx),
	}
}

func hostAPISoulOrigin(method extensioncontract.HostAPIMethod) soul.AuthoringIdentity {
	return soul.AuthoringIdentity{Kind: "host_api", Ref: strings.TrimSpace(string(method))}
}

func hostAPIHeartbeatActor(ctx context.Context) heartbeat.AuthoringIdentity {
	return heartbeat.AuthoringIdentity{
		Kind: string(heartbeat.ActorKindExtension),
		Ref:  hostAPIExtensionNameFromContext(ctx),
	}
}

func mapHostAPISoulRPCError(err error) error {
	if err == nil {
		return nil
	}
	status := hostAPISoulHTTPStatus(err)
	return hostAPIStatusRPCError(status, http.StatusText(status), map[string]string{"error": err.Error()})
}

func mapHostAPIHeartbeatRPCError(err error) error {
	if err == nil {
		return nil
	}
	status := hostAPIHeartbeatHTTPStatus(err)
	return hostAPIStatusRPCError(status, http.StatusText(status), map[string]string{"error": err.Error()})
}

func hostAPISoulHTTPStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, workspacepkg.ErrWorkspaceResolverUnavailable):
		return http.StatusServiceUnavailable
	case errors.Is(err, errHostAPIAuthoredValidation):
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
		errors.Is(err, apicontract.ErrInvalidAuthoredContextEnum):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

func hostAPIHeartbeatHTTPStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, workspacepkg.ErrWorkspaceResolverUnavailable):
		return http.StatusServiceUnavailable
	case errors.Is(err, errHostAPIAuthoredValidation):
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
		errors.Is(err, apicontract.ErrInvalidAuthoredContextEnum):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}
