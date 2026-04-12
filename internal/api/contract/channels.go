package contract

import (
	"encoding/json"
	"strings"
	"time"

	channelspkg "github.com/pedronauck/agh/internal/channels"
)

// CreateChannelRequest is the shared channel-instance creation payload.
type CreateChannelRequest struct {
	Scope            channelspkg.Scope         `json:"scope"`
	WorkspaceID      string                    `json:"workspace_id,omitempty"`
	Platform         string                    `json:"platform"`
	ExtensionName    string                    `json:"extension_name"`
	DisplayName      string                    `json:"display_name"`
	Enabled          bool                      `json:"enabled"`
	Status           channelspkg.ChannelStatus `json:"status"`
	RoutingPolicy    channelspkg.RoutingPolicy `json:"routing_policy"`
	DeliveryDefaults json.RawMessage           `json:"delivery_defaults,omitempty"`
}

// ToCreateInstanceRequest validates and converts the transport payload into the
// daemon-owned channel create request.
func (r CreateChannelRequest) ToCreateInstanceRequest() (channelspkg.CreateInstanceRequest, error) {
	req := channelspkg.CreateInstanceRequest{
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
		return channelspkg.CreateInstanceRequest{}, err
	}
	return req, nil
}

// UpdateChannelRequest is the shared mutable channel-instance patch payload.
type UpdateChannelRequest struct {
	DisplayName      *string                    `json:"display_name,omitempty"`
	RoutingPolicy    *channelspkg.RoutingPolicy `json:"routing_policy,omitempty"`
	DeliveryDefaults *json.RawMessage           `json:"delivery_defaults,omitempty"`
}

// ToUpdateInstanceRequest validates and converts the transport patch payload
// into the daemon-owned channel update request for the supplied instance id.
func (r UpdateChannelRequest) ToUpdateInstanceRequest(id string) (channelspkg.UpdateInstanceRequest, error) {
	req := channelspkg.UpdateInstanceRequest{ID: strings.TrimSpace(id)}
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
		return channelspkg.UpdateInstanceRequest{}, err
	}
	return req, nil
}

// ChannelDeliveryTargetInput is the shared typed delivery-target override payload.
type ChannelDeliveryTargetInput struct {
	ChannelInstanceID string                   `json:"channel_instance_id,omitempty"`
	PeerID            string                   `json:"peer_id,omitempty"`
	ThreadID          string                   `json:"thread_id,omitempty"`
	GroupID           string                   `json:"group_id,omitempty"`
	Mode              channelspkg.DeliveryMode `json:"mode,omitempty"`
}

// ChannelTestDeliveryRequest is the shared typed dry-run delivery payload.
type ChannelTestDeliveryRequest struct {
	Message string                     `json:"message,omitempty"`
	Target  ChannelDeliveryTargetInput `json:"target"`
}

// ToResolveDeliveryTargetRequest validates and converts the transport payload
// into the daemon-owned delivery-target resolution request for the supplied
// channel instance id.
func (r ChannelTestDeliveryRequest) ToResolveDeliveryTargetRequest(channelInstanceID string) (channelspkg.ResolveDeliveryTargetRequest, error) {
	req := channelspkg.ResolveDeliveryTargetRequest{
		ChannelInstanceID: strings.TrimSpace(r.Target.ChannelInstanceID),
		PeerID:            strings.TrimSpace(r.Target.PeerID),
		ThreadID:          strings.TrimSpace(r.Target.ThreadID),
		GroupID:           strings.TrimSpace(r.Target.GroupID),
		Mode:              r.Target.Mode.Normalize(),
	}
	req.ChannelInstanceID = strings.TrimSpace(req.ChannelInstanceID)
	trimmedID := strings.TrimSpace(channelInstanceID)
	if req.ChannelInstanceID == "" {
		req.ChannelInstanceID = trimmedID
	}
	if req.ChannelInstanceID != trimmedID {
		return channelspkg.ResolveDeliveryTargetRequest{}, ErrChannelInstanceMismatch
	}
	if err := req.Validate(); err != nil {
		return channelspkg.ResolveDeliveryTargetRequest{}, err
	}
	return req, nil
}

// ErrChannelInstanceMismatch reports a body/path channel-instance mismatch for
// typed delivery-target requests.
var ErrChannelInstanceMismatch = channelContractError("channel instance id must match request path")

type channelContractError string

func (e channelContractError) Error() string {
	return string(e)
}

// ChannelsResponse wraps the shared channel list payload.
type ChannelsResponse struct {
	Channels      []channelspkg.ChannelInstance   `json:"channels"`
	ChannelHealth map[string]ChannelHealthPayload `json:"channel_health,omitempty"`
}

// ChannelResponse wraps one shared channel payload.
type ChannelResponse struct {
	Channel channelspkg.ChannelInstance `json:"channel"`
	Health  ChannelHealthPayload        `json:"health"`
}

// ChannelRoutesResponse wraps one channel's route set.
type ChannelRoutesResponse struct {
	Routes []channelspkg.ChannelRoute `json:"routes"`
}

// ChannelTestDeliveryResponse wraps the dry-run delivery-target resolution payload.
type ChannelTestDeliveryResponse struct {
	Status         string                     `json:"status"`
	Message        string                     `json:"message,omitempty"`
	DeliveryTarget channelspkg.DeliveryTarget `json:"delivery_target"`
}

// ChannelHealthPayload captures the additive per-instance observability fields
// exposed through channel APIs.
type ChannelHealthPayload struct {
	ChannelInstanceID       string                    `json:"channel_instance_id"`
	Status                  channelspkg.ChannelStatus `json:"status"`
	RouteCount              int                       `json:"route_count"`
	DeliveryBacklog         int                       `json:"delivery_backlog"`
	DeliveryDroppedTotal    int                       `json:"delivery_dropped_total"`
	DeliveryDroppedByReason map[string]int            `json:"delivery_dropped_by_reason,omitempty"`
	DeliveryFailuresTotal   int                       `json:"delivery_failures_total"`
	AuthFailuresTotal       int                       `json:"auth_failures_total"`
	LastError               string                    `json:"last_error,omitempty"`
	LastErrorAt             *time.Time                `json:"last_error_at,omitempty"`
}

// ChannelStatusCountsPayload captures aggregate per-status counts for channel health.
type ChannelStatusCountsPayload struct {
	Disabled     int `json:"disabled"`
	Starting     int `json:"starting"`
	Ready        int `json:"ready"`
	Degraded     int `json:"degraded"`
	AuthRequired int `json:"auth_required"`
	Error        int `json:"error"`
}

// ChannelAggregateHealthPayload captures the additive channel summary nested
// under the daemon health response.
type ChannelAggregateHealthPayload struct {
	TotalInstances        int                        `json:"total_instances"`
	RouteCount            int                        `json:"route_count"`
	DeliveryBacklog       int                        `json:"delivery_backlog"`
	DeliveryDroppedTotal  int                        `json:"delivery_dropped_total"`
	DeliveryFailuresTotal int                        `json:"delivery_failures_total"`
	AuthFailuresTotal     int                        `json:"auth_failures_total"`
	StatusCounts          ChannelStatusCountsPayload `json:"status_counts"`
}

func cloneRawMessage(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}
