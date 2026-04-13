package bridges

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrBridgeInstanceNotFound reports that no persisted bridge instance matched the lookup.
	ErrBridgeInstanceNotFound = errors.New("bridges: bridge instance not found")
	// ErrBridgeInstanceUnavailable reports that the instance exists but cannot currently accept routing work.
	ErrBridgeInstanceUnavailable = errors.New("bridges: bridge instance unavailable")
	// ErrBridgeSecretBindingNotFound reports that no persisted secret binding matched the lookup.
	ErrBridgeSecretBindingNotFound = errors.New("bridges: bridge secret binding not found")
	// ErrBridgeRouteNotFound reports that no persisted route matched the lookup.
	ErrBridgeRouteNotFound = errors.New("bridges: bridge route not found")
	// ErrIngestDedupRecordNotFound reports that no active ingest dedup record matched the lookup.
	ErrIngestDedupRecordNotFound = errors.New("bridges: ingest dedup record not found")
	// ErrInvalidBridgeStateTransition reports that the requested instance lifecycle transition is not allowed.
	ErrInvalidBridgeStateTransition = errors.New("bridges: invalid bridge state transition")
)

// Scope identifies whether a bridge resource is daemon-global or workspace-owned.
type Scope string

const (
	// ScopeGlobal identifies a daemon-global bridge resource.
	ScopeGlobal Scope = "global"
	// ScopeWorkspace identifies a workspace-owned bridge resource.
	ScopeWorkspace Scope = "workspace"
)

// Normalize returns the normalized representation of the scope.
func (s Scope) Normalize() Scope {
	return Scope(strings.ToLower(strings.TrimSpace(string(s))))
}

// Validate reports whether the scope is supported.
func (s Scope) Validate() error {
	switch s.Normalize() {
	case ScopeGlobal, ScopeWorkspace:
		return nil
	case "":
		return errors.New("bridges: scope is required")
	default:
		return fmt.Errorf("bridges: unsupported scope %q", s)
	}
}

// ValidateScopeWorkspaceID enforces the canonical scope and workspace invariant.
func ValidateScopeWorkspaceID(scope Scope, workspaceID string) error {
	normalizedScope := scope.Normalize()
	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	if err := normalizedScope.Validate(); err != nil {
		return err
	}

	switch normalizedScope {
	case ScopeGlobal:
		if trimmedWorkspaceID != "" {
			return errors.New("bridges: global scope cannot include workspace id")
		}
	case ScopeWorkspace:
		if trimmedWorkspaceID == "" {
			return errors.New("bridges: workspace scope requires workspace id")
		}
	}

	return nil
}

// BridgeStatus reports the operator-visible lifecycle state of a bridge instance.
type BridgeStatus string

const (
	// BridgeStatusDisabled reports an instance that is intentionally disabled.
	BridgeStatusDisabled BridgeStatus = "disabled"
	// BridgeStatusStarting reports an instance that is launching or reconnecting.
	BridgeStatusStarting BridgeStatus = "starting"
	// BridgeStatusReady reports an instance that is healthy and ready to ingest/deliver.
	BridgeStatusReady BridgeStatus = "ready"
	// BridgeStatusDegraded reports an instance that is partially working with known issues.
	BridgeStatusDegraded BridgeStatus = "degraded"
	// BridgeStatusAuthRequired reports an instance that cannot operate until authentication is refreshed.
	BridgeStatusAuthRequired BridgeStatus = "auth_required"
	// BridgeStatusError reports an instance that is unhealthy due to a terminal or repeated fault.
	BridgeStatusError BridgeStatus = "error"
)

// Normalize returns the normalized representation of the status.
func (s BridgeStatus) Normalize() BridgeStatus {
	return BridgeStatus(strings.ToLower(strings.TrimSpace(string(s))))
}

// Validate reports whether the status belongs to the closed bridge status set.
func (s BridgeStatus) Validate() error {
	switch s.Normalize() {
	case BridgeStatusDisabled,
		BridgeStatusStarting,
		BridgeStatusReady,
		BridgeStatusDegraded,
		BridgeStatusAuthRequired,
		BridgeStatusError:
		return nil
	case "":
		return errors.New("bridges: bridge status is required")
	default:
		return fmt.Errorf("bridges: unsupported bridge status %q", s)
	}
}

// RoutingPolicy controls which platform identity dimensions participate in routing.
type RoutingPolicy struct {
	IncludePeer   bool `json:"include_peer"`
	IncludeThread bool `json:"include_thread"`
	IncludeGroup  bool `json:"include_group"`
}

// Validate reports whether the routing policy is internally consistent.
func (p RoutingPolicy) Validate() error {
	if p.IncludeThread && !p.IncludePeer && !p.IncludeGroup {
		return errors.New("bridges: routing policy cannot include thread without peer or group")
	}
	return nil
}

// BridgeInstance is the authoritative persisted configuration for one bridge adapter instance.
type BridgeInstance struct {
	ID               string          `json:"id"`
	Scope            Scope           `json:"scope"`
	WorkspaceID      string          `json:"workspace_id,omitempty"`
	Platform         string          `json:"platform"`
	ExtensionName    string          `json:"extension_name"`
	DisplayName      string          `json:"display_name"`
	Enabled          bool            `json:"enabled"`
	Status           BridgeStatus    `json:"status"`
	RoutingPolicy    RoutingPolicy   `json:"routing_policy"`
	DeliveryDefaults json.RawMessage `json:"delivery_defaults,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// Validate reports whether the persisted bridge instance shape is complete and valid.
func (i BridgeInstance) Validate() error {
	normalized := i.normalize()
	if err := requireField(normalized.ID, "bridge instance id"); err != nil {
		return err
	}
	if err := ValidateScopeWorkspaceID(normalized.Scope, normalized.WorkspaceID); err != nil {
		return err
	}
	if err := requireField(normalized.Platform, "bridge instance platform"); err != nil {
		return err
	}
	if err := requireField(normalized.ExtensionName, "bridge instance extension name"); err != nil {
		return err
	}
	if err := requireField(normalized.DisplayName, "bridge instance display name"); err != nil {
		return err
	}
	if err := normalized.Status.Validate(); err != nil {
		return err
	}
	if err := validateInstanceLifecycle(normalized.Enabled, normalized.Status); err != nil {
		return err
	}
	if err := normalized.RoutingPolicy.Validate(); err != nil {
		return err
	}
	if _, err := normalizeRawJSON(normalized.DeliveryDefaults, "bridge instance delivery defaults"); err != nil {
		return err
	}
	return nil
}

// BridgeSecretBinding binds one named bridge secret slot to a daemon-managed vault reference.
type BridgeSecretBinding struct {
	BridgeInstanceID string    `json:"bridge_instance_id"`
	BindingName      string    `json:"binding_name"`
	VaultRef         string    `json:"vault_ref"`
	Kind             string    `json:"kind"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Validate reports whether the persisted secret binding is complete and valid.
func (b BridgeSecretBinding) Validate() error {
	normalized := b.normalize()
	if err := requireField(normalized.BridgeInstanceID, "bridge secret binding bridge instance id"); err != nil {
		return err
	}
	if err := requireField(normalized.BindingName, "bridge secret binding name"); err != nil {
		return err
	}
	if err := requireField(normalized.VaultRef, "bridge secret binding vault ref"); err != nil {
		return err
	}
	if err := requireField(normalized.Kind, "bridge secret binding kind"); err != nil {
		return err
	}
	return nil
}

// DeliveryTarget identifies an outbound delivery destination within one bridge instance.
type DeliveryTarget struct {
	BridgeInstanceID string       `json:"bridge_instance_id"`
	PeerID           string       `json:"peer_id,omitempty"`
	ThreadID         string       `json:"thread_id,omitempty"`
	GroupID          string       `json:"group_id,omitempty"`
	Mode             DeliveryMode `json:"mode,omitempty"`
}

// Validate reports whether the delivery target contains a supported mode and
// the identity fields required by that mode.
func (t DeliveryTarget) Validate() error {
	normalized := t.normalize()
	if err := requireField(normalized.BridgeInstanceID, "delivery target bridge instance id"); err != nil {
		return err
	}
	if err := normalized.Mode.Validate(); err != nil {
		return err
	}
	if normalized.ThreadID != "" && normalized.PeerID == "" && normalized.GroupID == "" {
		return fmt.Errorf(
			"bridges: delivery target thread id requires peer id or group id for mode %q",
			normalized.Mode,
		)
	}

	switch normalized.Mode {
	case DeliveryModeDirectSend, DeliveryModeReply:
		if normalized.PeerID == "" && normalized.GroupID == "" {
			return fmt.Errorf(
				"bridges: delivery target mode %q requires peer id or group id",
				normalized.Mode,
			)
		}
	}

	return nil
}

// IsZero reports whether the target carries any values.
func (t DeliveryTarget) IsZero() bool {
	normalized := t.normalize()
	return normalized.BridgeInstanceID == "" &&
		normalized.PeerID == "" &&
		normalized.ThreadID == "" &&
		normalized.GroupID == "" &&
		normalized.Mode == ""
}

// MessageSender identifies the upstream actor that produced an inbound message.
type MessageSender struct {
	ID          string `json:"id,omitempty"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

// MessageContent carries normalized text content shared by inbound and outbound bridge models.
type MessageContent struct {
	Text string `json:"text,omitempty"`
}

// MessageAttachment captures normalized attachment metadata shared by ingest and delivery flows.
type MessageAttachment struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	MIMEType string `json:"mime_type,omitempty"`
	URL      string `json:"url,omitempty"`
}

// InboundMessageEnvelope is the normalized bridge ingest payload delivered by adapters.
type InboundMessageEnvelope struct {
	BridgeInstanceID  string              `json:"bridge_instance_id"`
	Scope             Scope               `json:"scope"`
	WorkspaceID       string              `json:"workspace_id,omitempty"`
	PeerID            string              `json:"peer_id,omitempty"`
	ThreadID          string              `json:"thread_id,omitempty"`
	GroupID           string              `json:"group_id,omitempty"`
	PlatformMessageID string              `json:"platform_message_id"`
	ReceivedAt        time.Time           `json:"received_at"`
	Sender            MessageSender       `json:"sender"`
	Content           MessageContent      `json:"content"`
	Attachments       []MessageAttachment `json:"attachments,omitempty"`
	IdempotencyKey    string              `json:"idempotency_key"`
}

// Validate reports whether the inbound envelope contains the required identifying fields.
func (e InboundMessageEnvelope) Validate() error {
	normalized := e.normalize()
	if err := requireField(normalized.BridgeInstanceID, "inbound message bridge instance id"); err != nil {
		return err
	}
	if err := ValidateScopeWorkspaceID(normalized.Scope, normalized.WorkspaceID); err != nil {
		return err
	}
	if err := requireField(normalized.PlatformMessageID, "inbound message platform message id"); err != nil {
		return err
	}
	if normalized.ReceivedAt.IsZero() {
		return errors.New("bridges: inbound message received at is required")
	}
	if err := requireField(normalized.IdempotencyKey, "inbound message idempotency key"); err != nil {
		return err
	}
	return nil
}

// DeliveryEvent is the daemon-owned outbound projection sent to a bridge adapter.
type DeliveryEvent struct {
	DeliveryID       string          `json:"delivery_id"`
	BridgeInstanceID string          `json:"bridge_instance_id"`
	RoutingKey       RoutingKey      `json:"routing_key"`
	DeliveryTarget   DeliveryTarget  `json:"delivery_target"`
	Seq              int64           `json:"seq"`
	EventType        string          `json:"event_type"`
	Content          MessageContent  `json:"content"`
	Final            bool            `json:"final"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
}

// Validate reports whether the delivery event contains the required identifiers.
func (e DeliveryEvent) Validate() error {
	normalized := e.normalize()
	if err := requireField(normalized.DeliveryID, "delivery event id"); err != nil {
		return err
	}
	if err := requireField(normalized.BridgeInstanceID, "delivery event bridge instance id"); err != nil {
		return err
	}
	if err := normalized.RoutingKey.Validate(); err != nil {
		return err
	}
	if normalized.RoutingKey.BridgeInstanceID != normalized.BridgeInstanceID {
		return errors.New("bridges: delivery event bridge instance id must match routing key")
	}
	if !normalized.DeliveryTarget.IsZero() {
		if err := normalized.DeliveryTarget.Validate(); err != nil {
			return err
		}
		if normalized.DeliveryTarget.BridgeInstanceID != normalized.BridgeInstanceID {
			return errors.New("bridges: delivery target bridge instance id must match delivery event")
		}
	}
	if normalized.Seq < 0 {
		return fmt.Errorf("bridges: invalid delivery event sequence %d", normalized.Seq)
	}
	if err := validateDeliveryEventType(normalized.EventType, normalized.Final); err != nil {
		return err
	}
	if _, err := normalizeRawJSON(normalized.Metadata, "delivery event metadata"); err != nil {
		return err
	}
	return nil
}

// IngestDedupRecord tracks inbound idempotency keys with an explicit TTL.
type IngestDedupRecord struct {
	IdempotencyKey   string    `json:"idempotency_key"`
	BridgeInstanceID string    `json:"bridge_instance_id"`
	ReceivedAt       time.Time `json:"received_at"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// Validate reports whether the dedup record is complete and time-consistent.
func (r IngestDedupRecord) Validate() error {
	normalized := r.normalize()
	if err := requireField(normalized.IdempotencyKey, "ingest dedup idempotency key"); err != nil {
		return err
	}
	if err := requireField(normalized.BridgeInstanceID, "ingest dedup bridge instance id"); err != nil {
		return err
	}
	if normalized.ReceivedAt.IsZero() {
		return errors.New("bridges: ingest dedup received at is required")
	}
	if normalized.ExpiresAt.IsZero() {
		return errors.New("bridges: ingest dedup expires at is required")
	}
	if !normalized.ExpiresAt.After(normalized.ReceivedAt) {
		return errors.New("bridges: ingest dedup expires at must be after received at")
	}
	return nil
}

func (i BridgeInstance) normalize() BridgeInstance {
	normalized := i
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.Scope = normalized.Scope.Normalize()
	normalized.WorkspaceID = strings.TrimSpace(normalized.WorkspaceID)
	normalized.Platform = strings.TrimSpace(normalized.Platform)
	normalized.ExtensionName = strings.TrimSpace(normalized.ExtensionName)
	normalized.DisplayName = strings.TrimSpace(normalized.DisplayName)
	normalized.Status = normalized.Status.Normalize()
	normalized.DeliveryDefaults = bytes.TrimSpace(normalized.DeliveryDefaults)
	return normalized
}

func (b BridgeSecretBinding) normalize() BridgeSecretBinding {
	normalized := b
	normalized.BridgeInstanceID = strings.TrimSpace(normalized.BridgeInstanceID)
	normalized.BindingName = strings.TrimSpace(normalized.BindingName)
	normalized.VaultRef = strings.TrimSpace(normalized.VaultRef)
	normalized.Kind = strings.TrimSpace(normalized.Kind)
	return normalized
}

func (t DeliveryTarget) normalize() DeliveryTarget {
	normalized := t
	normalized.BridgeInstanceID = strings.TrimSpace(normalized.BridgeInstanceID)
	normalized.PeerID = strings.TrimSpace(normalized.PeerID)
	normalized.ThreadID = strings.TrimSpace(normalized.ThreadID)
	normalized.GroupID = strings.TrimSpace(normalized.GroupID)
	normalized.Mode = normalized.Mode.Normalize()
	return normalized
}

func (e InboundMessageEnvelope) normalize() InboundMessageEnvelope {
	normalized := e
	if len(e.Attachments) > 0 {
		normalized.Attachments = append([]MessageAttachment(nil), e.Attachments...)
	}
	normalized.BridgeInstanceID = strings.TrimSpace(normalized.BridgeInstanceID)
	normalized.Scope = normalized.Scope.Normalize()
	normalized.WorkspaceID = strings.TrimSpace(normalized.WorkspaceID)
	normalized.PeerID = strings.TrimSpace(normalized.PeerID)
	normalized.ThreadID = strings.TrimSpace(normalized.ThreadID)
	normalized.GroupID = strings.TrimSpace(normalized.GroupID)
	normalized.PlatformMessageID = strings.TrimSpace(normalized.PlatformMessageID)
	normalized.Sender = MessageSender{
		ID:          strings.TrimSpace(normalized.Sender.ID),
		Username:    strings.TrimSpace(normalized.Sender.Username),
		DisplayName: strings.TrimSpace(normalized.Sender.DisplayName),
	}
	normalized.Content = MessageContent{Text: strings.TrimSpace(normalized.Content.Text)}
	normalized.IdempotencyKey = strings.TrimSpace(normalized.IdempotencyKey)
	for idx := range normalized.Attachments {
		normalized.Attachments[idx] = MessageAttachment{
			ID:       strings.TrimSpace(normalized.Attachments[idx].ID),
			Name:     strings.TrimSpace(normalized.Attachments[idx].Name),
			MIMEType: strings.TrimSpace(normalized.Attachments[idx].MIMEType),
			URL:      strings.TrimSpace(normalized.Attachments[idx].URL),
		}
	}
	return normalized
}

func (e DeliveryEvent) normalize() DeliveryEvent {
	normalized := e
	normalized.DeliveryID = strings.TrimSpace(normalized.DeliveryID)
	normalized.BridgeInstanceID = strings.TrimSpace(normalized.BridgeInstanceID)
	normalized.RoutingKey = normalized.RoutingKey.normalize()
	normalized.DeliveryTarget = normalized.DeliveryTarget.normalize()
	normalized.EventType = normalizeDeliveryEventType(normalized.EventType)
	normalized.Content = MessageContent{Text: strings.TrimSpace(normalized.Content.Text)}
	normalized.Metadata = bytes.TrimSpace(normalized.Metadata)
	return normalized
}

func (r IngestDedupRecord) normalize() IngestDedupRecord {
	normalized := r
	normalized.IdempotencyKey = strings.TrimSpace(normalized.IdempotencyKey)
	normalized.BridgeInstanceID = strings.TrimSpace(normalized.BridgeInstanceID)
	return normalized
}

func requireField(value string, label string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("bridges: %s is required", label)
	}
	return nil
}

func normalizeRawJSON(value json.RawMessage, label string) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 {
		return nil, nil
	}
	if !json.Valid(trimmed) {
		return nil, fmt.Errorf("bridges: %s must be valid JSON", label)
	}

	var compacted bytes.Buffer
	if err := json.Compact(&compacted, trimmed); err != nil {
		return nil, fmt.Errorf("bridges: compact %s: %w", label, err)
	}

	return compacted.Bytes(), nil
}
