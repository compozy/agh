package contract

import (
	"encoding/json"
	"strings"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
)

// CreateBridgeRequest is the shared bridge-instance creation payload.
type CreateBridgeRequest struct {
	Scope            bridgepkg.Scope         `json:"scope"`
	WorkspaceID      string                  `json:"workspace_id,omitempty"`
	Platform         string                  `json:"platform"`
	ExtensionName    string                  `json:"extension_name"`
	DisplayName      string                  `json:"display_name"`
	Enabled          bool                    `json:"enabled"`
	Status           bridgepkg.BridgeStatus  `json:"status"`
	RoutingPolicy    bridgepkg.RoutingPolicy `json:"routing_policy"`
	DeliveryDefaults json.RawMessage         `json:"delivery_defaults,omitempty"`
}

// ToCreateInstanceRequest validates and converts the transport payload into the
// daemon-owned bridge create request.
func (r CreateBridgeRequest) ToCreateInstanceRequest() (bridgepkg.CreateInstanceRequest, error) {
	req := bridgepkg.CreateInstanceRequest{
		Scope:            r.Scope,
		WorkspaceID:      strings.TrimSpace(r.WorkspaceID),
		Platform:         strings.TrimSpace(r.Platform),
		ExtensionName:    strings.TrimSpace(r.ExtensionName),
		DisplayName:      strings.TrimSpace(r.DisplayName),
		Enabled:          r.Enabled,
		Status:           r.Status,
		RoutingPolicy:    r.RoutingPolicy,
		DeliveryDefaults: cloneRawMessage(r.DeliveryDefaults),
	}
	if err := req.Validate(); err != nil {
		return bridgepkg.CreateInstanceRequest{}, err
	}
	return req, nil
}

// UpdateBridgeRequest is the shared mutable bridge-instance patch payload.
type UpdateBridgeRequest struct {
	DisplayName      *string                  `json:"display_name,omitempty"`
	RoutingPolicy    *bridgepkg.RoutingPolicy `json:"routing_policy,omitempty"`
	DeliveryDefaults *json.RawMessage         `json:"delivery_defaults,omitempty"`
}

// ToUpdateInstanceRequest validates and converts the transport patch payload
// into the daemon-owned bridge update request for the supplied instance id.
func (r UpdateBridgeRequest) ToUpdateInstanceRequest(id string) (bridgepkg.UpdateInstanceRequest, error) {
	req := bridgepkg.UpdateInstanceRequest{ID: strings.TrimSpace(id)}
	if r.DisplayName != nil {
		value := strings.TrimSpace(*r.DisplayName)
		req.DisplayName = &value
	}
	if r.RoutingPolicy != nil {
		value := *r.RoutingPolicy
		req.RoutingPolicy = &value
	}
	if r.DeliveryDefaults != nil {
		value := cloneRawMessage(*r.DeliveryDefaults)
		req.DeliveryDefaults = &value
	}
	if err := req.Validate(); err != nil {
		return bridgepkg.UpdateInstanceRequest{}, err
	}
	return req, nil
}

// BridgeDeliveryTargetInput is the shared typed delivery-target override payload.
type BridgeDeliveryTargetInput struct {
	BridgeInstanceID string                 `json:"bridge_instance_id,omitempty"`
	PeerID           string                 `json:"peer_id,omitempty"`
	ThreadID         string                 `json:"thread_id,omitempty"`
	GroupID          string                 `json:"group_id,omitempty"`
	Mode             bridgepkg.DeliveryMode `json:"mode,omitempty"`
}

// BridgeTestDeliveryRequest is the shared typed dry-run delivery payload.
type BridgeTestDeliveryRequest struct {
	Message string                    `json:"message,omitempty"`
	Target  BridgeDeliveryTargetInput `json:"target"`
}

// ToResolveDeliveryTargetRequest validates and converts the transport payload
// into the daemon-owned delivery-target resolution request for the supplied
// bridge instance id.
func (r BridgeTestDeliveryRequest) ToResolveDeliveryTargetRequest(bridgeInstanceID string) (bridgepkg.ResolveDeliveryTargetRequest, error) {
	req := bridgepkg.ResolveDeliveryTargetRequest{
		BridgeInstanceID: strings.TrimSpace(r.Target.BridgeInstanceID),
		PeerID:           strings.TrimSpace(r.Target.PeerID),
		ThreadID:         strings.TrimSpace(r.Target.ThreadID),
		GroupID:          strings.TrimSpace(r.Target.GroupID),
		Mode:             r.Target.Mode.Normalize(),
	}
	req.BridgeInstanceID = strings.TrimSpace(req.BridgeInstanceID)
	trimmedID := strings.TrimSpace(bridgeInstanceID)
	if req.BridgeInstanceID == "" {
		req.BridgeInstanceID = trimmedID
	}
	if req.BridgeInstanceID != trimmedID {
		return bridgepkg.ResolveDeliveryTargetRequest{}, ErrBridgeInstanceMismatch
	}
	if err := req.Validate(); err != nil {
		return bridgepkg.ResolveDeliveryTargetRequest{}, err
	}
	return req, nil
}

// ErrBridgeInstanceMismatch reports a body/path bridge-instance mismatch for
// typed delivery-target requests.
var ErrBridgeInstanceMismatch = bridgeContractError("bridge instance id must match request path")

type bridgeContractError string

func (e bridgeContractError) Error() string {
	return string(e)
}

// BridgesResponse wraps the shared bridge list payload.
type BridgesResponse struct {
	Bridges      []bridgepkg.BridgeInstance     `json:"bridges"`
	BridgeHealth map[string]BridgeHealthPayload `json:"bridge_health,omitempty"`
}

// BridgeResponse wraps one shared bridge payload.
type BridgeResponse struct {
	Bridge bridgepkg.BridgeInstance `json:"bridge"`
	Health BridgeHealthPayload      `json:"health"`
}

// BridgeRoutesResponse wraps one bridge's route set.
type BridgeRoutesResponse struct {
	Routes []bridgepkg.BridgeRoute `json:"routes"`
}

// BridgeTestDeliveryResponse wraps the dry-run delivery-target resolution payload.
type BridgeTestDeliveryResponse struct {
	Status         string                   `json:"status"`
	Message        string                   `json:"message,omitempty"`
	DeliveryTarget bridgepkg.DeliveryTarget `json:"delivery_target"`
}

// BridgeHealthPayload captures the additive per-instance observability fields
// exposed through bridge APIs.
type BridgeHealthPayload struct {
	BridgeInstanceID        string                 `json:"bridge_instance_id"`
	Status                  bridgepkg.BridgeStatus `json:"status"`
	RouteCount              int                    `json:"route_count"`
	DeliveryBacklog         int                    `json:"delivery_backlog"`
	DeliveryDroppedTotal    int                    `json:"delivery_dropped_total"`
	DeliveryDroppedByReason map[string]int         `json:"delivery_dropped_by_reason,omitempty"`
	DeliveryFailuresTotal   int                    `json:"delivery_failures_total"`
	AuthFailuresTotal       int                    `json:"auth_failures_total"`
	LastError               string                 `json:"last_error,omitempty"`
	LastErrorAt             *time.Time             `json:"last_error_at,omitempty"`
}

// BridgeStatusCountsPayload captures aggregate per-status counts for bridge health.
type BridgeStatusCountsPayload struct {
	Disabled     int `json:"disabled"`
	Starting     int `json:"starting"`
	Ready        int `json:"ready"`
	Degraded     int `json:"degraded"`
	AuthRequired int `json:"auth_required"`
	Error        int `json:"error"`
}

// BridgeAggregateHealthPayload captures the additive bridge summary nested
// under the daemon health response.
type BridgeAggregateHealthPayload struct {
	TotalInstances        int                       `json:"total_instances"`
	RouteCount            int                       `json:"route_count"`
	DeliveryBacklog       int                       `json:"delivery_backlog"`
	DeliveryDroppedTotal  int                       `json:"delivery_dropped_total"`
	DeliveryFailuresTotal int                       `json:"delivery_failures_total"`
	AuthFailuresTotal     int                       `json:"auth_failures_total"`
	StatusCounts          BridgeStatusCountsPayload `json:"status_counts"`
}

func cloneRawMessage(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}
