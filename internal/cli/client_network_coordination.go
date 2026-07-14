package cli

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
)

// NetworkCoordinationRecord is the shared coordination payload.
type NetworkCoordinationRecord = contract.NetworkCoordinationPayload

// NetworkUsageRecord is the shared usage response payload.
type NetworkUsageRecord = contract.NetworkUsageResponse

// PutNetworkCoordinationRequest is the shared coordination mutation payload.
type PutNetworkCoordinationRequest = contract.PutNetworkCoordinationRequest

// PutNetworkCoordinationInvitationRequest is the shared invitation mutation payload.
type PutNetworkCoordinationInvitationRequest = contract.PutNetworkCoordinationInvitationRequest

func (c *unixSocketClient) GetNetworkCoordination(
	ctx context.Context,
	workspaceRef string,
	taskID string,
) (NetworkCoordinationRecord, error) {
	path, err := networkCoordinationPath(workspaceRef)
	if err != nil {
		return NetworkCoordinationRecord{}, err
	}
	values := url.Values{}
	if trimmed := strings.TrimSpace(taskID); trimmed != "" {
		values.Set("task_id", trimmed)
	}
	var response contract.NetworkCoordinationResponse
	if err := c.doJSON(ctx, http.MethodGet, path, values, nil, &response); err != nil {
		return NetworkCoordinationRecord{}, err
	}
	return response.Coordination, nil
}

func (c *unixSocketClient) PutNetworkCoordination(
	ctx context.Context,
	workspaceRef string,
	request PutNetworkCoordinationRequest,
	taskID string,
) (NetworkCoordinationRecord, error) {
	path, err := networkCoordinationPath(workspaceRef)
	if err != nil {
		return NetworkCoordinationRecord{}, err
	}
	values := url.Values{}
	if trimmed := strings.TrimSpace(taskID); trimmed != "" {
		values.Set("task_id", trimmed)
	}
	var response contract.NetworkCoordinationResponse
	if err := c.doJSON(ctx, http.MethodPut, path, values, request, &response); err != nil {
		return NetworkCoordinationRecord{}, err
	}
	return response.Coordination, nil
}

func (c *unixSocketClient) PutNetworkCoordinationInvitation(
	ctx context.Context,
	workspaceRef string,
	request PutNetworkCoordinationInvitationRequest,
) (NetworkCoordinationRecord, error) {
	path, err := networkCoordinationPath(workspaceRef)
	if err != nil {
		return NetworkCoordinationRecord{}, err
	}
	var response contract.NetworkCoordinationResponse
	if err := c.doJSON(ctx, http.MethodPut, path+"/invitation", nil, request, &response); err != nil {
		return NetworkCoordinationRecord{}, err
	}
	return response.Coordination, nil
}

func (c *unixSocketClient) GetNetworkUsage(ctx context.Context, workspaceRef string) (NetworkUsageRecord, error) {
	base, err := networkBasePath(workspaceRef)
	if err != nil {
		return NetworkUsageRecord{}, err
	}
	var response NetworkUsageRecord
	if err := c.doJSON(ctx, http.MethodGet, base+"/usage", nil, nil, &response); err != nil {
		return NetworkUsageRecord{}, err
	}
	return response, nil
}

func networkCoordinationPath(workspaceRef string) (string, error) {
	workspaceRef, err := requireNetworkPathValue("workspace_id", workspaceRef)
	if err != nil {
		return "", err
	}
	return "/api/workspaces/" + url.PathEscape(workspaceRef) + "/network-coordination", nil
}
