package core

import (
	"context"
	"errors"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/heartbeat"
)

func (h *BaseHandlers) availableHeartbeatStatusForHealth(
	ctx context.Context,
	health contract.SessionHealthPayload,
	includeHealth bool,
) (*contract.HeartbeatStatusResponse, error) {
	status, err := h.heartbeatStatusForHealth(ctx, health, includeHealth)
	if errors.Is(err, heartbeat.ErrAuthoringAgentNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &status, nil
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
