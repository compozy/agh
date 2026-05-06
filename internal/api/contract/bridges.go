package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

// BridgeProviderConfigPayload carries provider-owned runtime configuration
// without constraining provider-specific keys in the transport contract.
type BridgeProviderConfigPayload json.RawMessage

// MarshalJSON preserves the compact raw JSON representation of provider config.
func (p BridgeProviderConfigPayload) MarshalJSON() ([]byte, error) {
	return marshalBridgeJSONPayload(json.RawMessage(p), "bridge provider config")
}

// UnmarshalJSON validates that provider config is an object-shaped JSON payload
// or null before storing the compact raw representation.
func (p *BridgeProviderConfigPayload) UnmarshalJSON(data []byte) error {
	normalized, err := normalizeBridgeJSONPayload(data, "bridge provider config", validateBridgeProviderConfigPayload)
	if err != nil {
		return err
	}
	*p = BridgeProviderConfigPayload(normalized)
	return nil
}

// BridgeDeliveryDefaultsPayload carries only typed delivery-target defaults.
type BridgeDeliveryDefaultsPayload json.RawMessage

// MarshalJSON preserves the compact raw JSON representation of delivery defaults.
func (p BridgeDeliveryDefaultsPayload) MarshalJSON() ([]byte, error) {
	return marshalBridgeJSONPayload(json.RawMessage(p), "bridge delivery defaults")
}

// UnmarshalJSON validates that delivery defaults remain scoped to the approved
// delivery-target fields or null.
func (p *BridgeDeliveryDefaultsPayload) UnmarshalJSON(data []byte) error {
	normalized, err := normalizeBridgeJSONPayload(
		data,
		"bridge delivery defaults",
		validateBridgeDeliveryDefaultsPayload,
	)
	if err != nil {
		return err
	}
	*p = BridgeDeliveryDefaultsPayload(normalized)
	return nil
}

// CreateBridgeRequest is the shared bridge-instance creation payload.
type CreateBridgeRequest struct {
	Scope            bridgepkg.Scope               `json:"scope"`
	WorkspaceID      string                        `json:"workspace_id,omitempty"`
	Platform         string                        `json:"platform"`
	ExtensionName    string                        `json:"extension_name"`
	DisplayName      string                        `json:"display_name"`
	Enabled          bool                          `json:"enabled"`
	Status           bridgepkg.BridgeStatus        `json:"status"`
	DMPolicy         bridgepkg.BridgeDMPolicy      `json:"dm_policy,omitempty"`
	RoutingPolicy    bridgepkg.RoutingPolicy       `json:"routing_policy"`
	ProviderConfig   BridgeProviderConfigPayload   `json:"provider_config,omitempty"`
	DeliveryDefaults BridgeDeliveryDefaultsPayload `json:"delivery_defaults,omitempty"`
	Degradation      *bridgepkg.BridgeDegradation  `json:"degradation,omitempty"`
}

// ToCreateInstanceRequest validates and converts the transport payload into the
// daemon-owned bridge create request.
func (r CreateBridgeRequest) ToCreateInstanceRequest() (bridgepkg.CreateInstanceRequest, error) {
	providerConfig, err := normalizeBridgeJSONPayload(
		json.RawMessage(r.ProviderConfig),
		"bridge provider config",
		validateBridgeProviderConfigPayload,
	)
	if err != nil {
		return bridgepkg.CreateInstanceRequest{}, err
	}
	deliveryDefaults, err := normalizeBridgeJSONPayload(
		json.RawMessage(r.DeliveryDefaults),
		"bridge delivery defaults",
		validateBridgeDeliveryDefaultsPayload,
	)
	if err != nil {
		return bridgepkg.CreateInstanceRequest{}, err
	}

	req := bridgepkg.CreateInstanceRequest{
		Scope:            r.Scope,
		WorkspaceID:      strings.TrimSpace(r.WorkspaceID),
		Platform:         strings.TrimSpace(r.Platform),
		ExtensionName:    strings.TrimSpace(r.ExtensionName),
		DisplayName:      strings.TrimSpace(r.DisplayName),
		Enabled:          r.Enabled,
		Status:           r.Status,
		DMPolicy:         r.DMPolicy,
		RoutingPolicy:    r.RoutingPolicy,
		ProviderConfig:   providerConfig,
		DeliveryDefaults: deliveryDefaults,
		Degradation:      cloneBridgeDegradation(r.Degradation),
	}
	if err := req.Validate(); err != nil {
		return bridgepkg.CreateInstanceRequest{}, err
	}
	return req, nil
}

// UpdateBridgeRequest is the shared mutable bridge-instance patch payload.
type UpdateBridgeRequest struct {
	DisplayName      *string                        `json:"display_name,omitempty"`
	DMPolicy         *bridgepkg.BridgeDMPolicy      `json:"dm_policy,omitempty"`
	RoutingPolicy    *bridgepkg.RoutingPolicy       `json:"routing_policy,omitempty"`
	ProviderConfig   *BridgeProviderConfigPayload   `json:"provider_config,omitempty"`
	DeliveryDefaults *BridgeDeliveryDefaultsPayload `json:"delivery_defaults,omitempty"`
	Degradation      *bridgepkg.BridgeDegradation   `json:"degradation,omitempty"`
	ClearDegradation bool                           `json:"clear_degradation,omitempty"`
}

// PutBridgeSecretBindingRequest is the shared bridge secret binding upsert payload.
type PutBridgeSecretBindingRequest struct {
	// SecretRef identifies the daemon-owned encrypted secret reference.
	SecretRef string `json:"secret_ref"`
	// Kind identifies the materialized secret kind passed to the provider runtime.
	Kind string `json:"kind"`
	// SecretValue is write-only plaintext stored into the vault when provided.
	SecretValue *string `json:"secret_value,omitempty"`
}

// ToUpdateInstanceRequest validates and converts the transport patch payload
// into the daemon-owned bridge update request for the supplied instance id.
func (r UpdateBridgeRequest) ToUpdateInstanceRequest(id string) (bridgepkg.UpdateInstanceRequest, error) {
	req := bridgepkg.UpdateInstanceRequest{
		ID:               strings.TrimSpace(id),
		ClearDegradation: r.ClearDegradation,
	}
	if r.DisplayName != nil {
		value := strings.TrimSpace(*r.DisplayName)
		req.DisplayName = &value
	}
	if r.DMPolicy != nil {
		value := *r.DMPolicy
		req.DMPolicy = &value
	}
	if r.RoutingPolicy != nil {
		value := *r.RoutingPolicy
		req.RoutingPolicy = &value
	}
	if r.ProviderConfig != nil {
		value, err := normalizeBridgeJSONPayload(
			json.RawMessage(*r.ProviderConfig),
			"bridge provider config",
			validateBridgeProviderConfigPayload,
		)
		if err != nil {
			return bridgepkg.UpdateInstanceRequest{}, err
		}
		req.ProviderConfig = &value
	}
	if r.DeliveryDefaults != nil {
		value, err := normalizeBridgeJSONPayload(
			json.RawMessage(*r.DeliveryDefaults),
			"bridge delivery defaults",
			validateBridgeDeliveryDefaultsPayload,
		)
		if err != nil {
			return bridgepkg.UpdateInstanceRequest{}, err
		}
		req.DeliveryDefaults = &value
	}
	if r.Degradation != nil {
		req.Degradation = cloneBridgeDegradation(r.Degradation)
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

// CreateTaskBridgeNotificationSubscriptionRequest captures one task-scoped
// terminal bridge notification target.
type CreateTaskBridgeNotificationSubscriptionRequest struct {
	SubscriptionID   string                 `json:"subscription_id,omitempty"`
	BridgeInstanceID string                 `json:"bridge_instance_id"`
	Scope            bridgepkg.Scope        `json:"scope"`
	WorkspaceID      string                 `json:"workspace_id,omitempty"`
	PeerID           string                 `json:"peer_id,omitempty"`
	ThreadID         string                 `json:"thread_id,omitempty"`
	GroupID          string                 `json:"group_id,omitempty"`
	DeliveryMode     bridgepkg.DeliveryMode `json:"delivery_mode"`
}

// ToResolveDeliveryTargetRequest validates and converts the transport payload
// into the daemon-owned delivery-target resolution request for the supplied
// bridge instance id.
func (r BridgeTestDeliveryRequest) ToResolveDeliveryTargetRequest(
	bridgeInstanceID string,
) (bridgepkg.ResolveDeliveryTargetRequest, error) {
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
	Bridges      []BridgePayload                `json:"bridges"`
	BridgeHealth map[string]BridgeHealthPayload `json:"bridge_health,omitempty"`
}

// BridgeHealthStreamPayload wraps one bridge-health SSE snapshot payload.
type BridgeHealthStreamPayload struct {
	GeneratedAt  time.Time                      `json:"generated_at"`
	BridgeHealth map[string]BridgeHealthPayload `json:"bridge_health"`
}

// BridgeProvidersResponse wraps the shared installed provider catalog.
type BridgeProvidersResponse struct {
	Providers []BridgeProviderPayload `json:"providers"`
}

// BridgeResponse wraps one shared bridge payload.
type BridgeResponse struct {
	Bridge BridgePayload       `json:"bridge"`
	Health BridgeHealthPayload `json:"health"`
}

// BridgeRoutesResponse wraps one bridge's route set.
type BridgeRoutesResponse struct {
	Routes []bridgepkg.BridgeRoute `json:"routes"`
}

// TaskBridgeNotificationCursorPayload exposes the durable cursor identity and
// latest diagnostic state for one bridge terminal notification subscription.
type TaskBridgeNotificationCursorPayload struct {
	ConsumerID      string     `json:"consumer_id"`
	StreamName      string     `json:"stream_name"`
	SubjectID       string     `json:"subject_id"`
	LastSequence    int64      `json:"last_sequence"`
	LastDeliveryID  string     `json:"last_delivery_id,omitempty"`
	LastDeliveredAt *time.Time `json:"last_delivered_at,omitempty"`
	LastError       string     `json:"last_error,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

// TaskBridgeNotificationSubscriptionPayload exposes one bridge terminal
// notification subscription plus its durable cursor diagnostics.
type TaskBridgeNotificationSubscriptionPayload struct {
	SubscriptionID   string                              `json:"subscription_id"`
	TaskID           string                              `json:"task_id"`
	BridgeInstanceID string                              `json:"bridge_instance_id"`
	Scope            bridgepkg.Scope                     `json:"scope"`
	WorkspaceID      string                              `json:"workspace_id,omitempty"`
	PeerID           string                              `json:"peer_id,omitempty"`
	ThreadID         string                              `json:"thread_id,omitempty"`
	GroupID          string                              `json:"group_id,omitempty"`
	DeliveryMode     bridgepkg.DeliveryMode              `json:"delivery_mode"`
	Cursor           TaskBridgeNotificationCursorPayload `json:"cursor"`
	CreatedBy        taskpkg.ActorIdentity               `json:"created_by"`
	CreatedAt        time.Time                           `json:"created_at"`
	UpdatedAt        time.Time                           `json:"updated_at"`
}

// TaskBridgeNotificationSubscriptionResponse wraps one task bridge
// notification subscription payload.
type TaskBridgeNotificationSubscriptionResponse struct {
	Subscription TaskBridgeNotificationSubscriptionPayload `json:"subscription"`
}

// TaskBridgeNotificationSubscriptionsResponse wraps a task bridge notification
// subscription list.
type TaskBridgeNotificationSubscriptionsResponse struct {
	Subscriptions []TaskBridgeNotificationSubscriptionPayload `json:"subscriptions"`
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
	BridgeInstanceID        string                       `json:"bridge_instance_id"`
	Status                  bridgepkg.BridgeStatus       `json:"status"`
	RouteCount              int                          `json:"route_count"`
	DeliveryBacklog         int                          `json:"delivery_backlog"`
	DeliveryDroppedTotal    int                          `json:"delivery_dropped_total"`
	DeliveryDroppedByReason map[string]int               `json:"delivery_dropped_by_reason,omitempty"`
	DeliveryFailuresTotal   int                          `json:"delivery_failures_total"`
	AuthFailuresTotal       int                          `json:"auth_failures_total"`
	LastSuccessAt           *time.Time                   `json:"last_success_at,omitempty"`
	LastError               string                       `json:"last_error,omitempty"`
	LastErrorAt             *time.Time                   `json:"last_error_at,omitempty"`
	Degradation             *bridgepkg.BridgeDegradation `json:"degradation,omitempty"`
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

// BridgeProviderPayload captures provider metadata exposed by bridge-management APIs.
type BridgeProviderPayload struct {
	Platform      string                                `json:"platform"`
	ExtensionName string                                `json:"extension_name"`
	DisplayName   string                                `json:"display_name"`
	Description   string                                `json:"description,omitempty"`
	SecretSlots   []bridgepkg.BridgeSecretSlot          `json:"secret_slots,omitempty"`
	ConfigSchema  *bridgepkg.BridgeProviderConfigSchema `json:"config_schema,omitempty"`
	Enabled       bool                                  `json:"enabled"`
	State         string                                `json:"state"`
	Health        string                                `json:"health"`
	HealthMessage string                                `json:"health_message,omitempty"`
}

// BridgePayload captures the shared bridge-management contract returned by HTTP/UDS.
type BridgePayload struct {
	ID               string                         `json:"id"`
	Scope            bridgepkg.Scope                `json:"scope"`
	WorkspaceID      string                         `json:"workspace_id,omitempty"`
	Platform         string                         `json:"platform"`
	ExtensionName    string                         `json:"extension_name"`
	DisplayName      string                         `json:"display_name"`
	Source           bridgepkg.BridgeInstanceSource `json:"source,omitempty"`
	Enabled          bool                           `json:"enabled"`
	Status           bridgepkg.BridgeStatus         `json:"status"`
	DMPolicy         bridgepkg.BridgeDMPolicy       `json:"dm_policy,omitempty"`
	RoutingPolicy    bridgepkg.RoutingPolicy        `json:"routing_policy"`
	ProviderConfig   BridgeProviderConfigPayload    `json:"provider_config,omitempty"`
	DeliveryDefaults BridgeDeliveryDefaultsPayload  `json:"delivery_defaults,omitempty"`
	Degradation      *bridgepkg.BridgeDegradation   `json:"degradation,omitempty"`
	CreatedAt        time.Time                      `json:"created_at"`
	UpdatedAt        time.Time                      `json:"updated_at"`
}

// ToBridgeSecretBinding validates and converts the transport payload into the daemon-owned binding request.
func (r PutBridgeSecretBindingRequest) ToBridgeSecretBinding(
	bridgeInstanceID string,
	bindingName string,
) (bridgepkg.BridgeSecretBinding, error) {
	binding := bridgepkg.BridgeSecretBinding{
		BridgeInstanceID: strings.TrimSpace(bridgeInstanceID),
		BindingName:      strings.TrimSpace(bindingName),
		SecretRef:        strings.TrimSpace(r.SecretRef),
		Kind:             strings.TrimSpace(r.Kind),
	}
	if err := binding.Validate(); err != nil {
		return bridgepkg.BridgeSecretBinding{}, err
	}
	return binding, nil
}

// BridgeSecretBindingsResponse wraps one bridge-instance secret binding list.
type BridgeSecretBindingsResponse struct {
	Bindings []bridgepkg.BridgeSecretBinding `json:"bindings"`
}

// BridgeSecretBindingResponse wraps one bridge secret binding payload.
type BridgeSecretBindingResponse struct {
	Binding bridgepkg.BridgeSecretBinding `json:"binding"`
}

func cloneBridgeDegradation(value *bridgepkg.BridgeDegradation) *bridgepkg.BridgeDegradation {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func marshalBridgeJSONPayload(value json.RawMessage, label string) ([]byte, error) {
	normalized, err := normalizeBridgeJSONPayload(value, label, nil)
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 {
		return []byte("null"), nil
	}
	return normalized, nil
}

func normalizeBridgeJSONPayload(
	value json.RawMessage,
	label string,
	validate func(json.RawMessage) error,
) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 {
		return nil, nil
	}
	if !json.Valid(trimmed) {
		return nil, fmt.Errorf("%s must be valid JSON", label)
	}

	var compacted bytes.Buffer
	if err := json.Compact(&compacted, trimmed); err != nil {
		return nil, fmt.Errorf("compact %s: %w", label, err)
	}
	normalized := compacted.Bytes()
	if validate != nil {
		if err := validate(normalized); err != nil {
			return nil, err
		}
	}
	return normalized, nil
}

func validateBridgeProviderConfigPayload(value json.RawMessage) error {
	if isJSONNull(value) {
		return nil
	}

	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return fmt.Errorf("bridge provider config must decode as JSON object or null: %w", err)
	}
	if _, ok := decoded.(map[string]any); !ok {
		return fmt.Errorf("bridge provider config must be a JSON object or null")
	}
	return nil
}

func validateBridgeDeliveryDefaultsPayload(value json.RawMessage) error {
	if isJSONNull(value) {
		return nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(value, &fields); err != nil {
		return fmt.Errorf("bridge delivery defaults must be a JSON object or null: %w", err)
	}

	var (
		peerID  string
		groupID string
		thread  string
	)

	for key, raw := range fields {
		switch key {
		case "peer_id":
			text, err := requireJSONStringField(raw, key)
			if err != nil {
				return err
			}
			peerID = strings.TrimSpace(text)
		case "thread_id":
			text, err := requireJSONStringField(raw, key)
			if err != nil {
				return err
			}
			thread = strings.TrimSpace(text)
		case "group_id":
			text, err := requireJSONStringField(raw, key)
			if err != nil {
				return err
			}
			groupID = strings.TrimSpace(text)
		case "mode":
			text, err := requireJSONStringField(raw, key)
			if err != nil {
				return err
			}
			if err := bridgepkg.DeliveryMode(text).Validate(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("bridge delivery defaults field %q is not supported", key)
		}
	}

	if thread != "" && peerID == "" && groupID == "" {
		return fmt.Errorf("bridge delivery defaults thread_id requires peer_id or group_id")
	}

	return nil
}

func requireJSONStringField(raw json.RawMessage, field string) (string, error) {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", fmt.Errorf("bridge delivery defaults field %q must be valid JSON: %w", field, err)
	}
	text, ok := decoded.(string)
	if !ok {
		return "", fmt.Errorf("bridge delivery defaults field %q must be a string", field)
	}
	return text, nil
}

func isJSONNull(value json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(value), []byte("null"))
}
