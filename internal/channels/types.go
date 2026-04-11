package channels

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrChannelInstanceNotFound reports that no persisted channel instance matched the lookup.
	ErrChannelInstanceNotFound = errors.New("channels: channel instance not found")
	// ErrChannelInstanceUnavailable reports that the instance exists but cannot currently accept routing work.
	ErrChannelInstanceUnavailable = errors.New("channels: channel instance unavailable")
	// ErrChannelSecretBindingNotFound reports that no persisted secret binding matched the lookup.
	ErrChannelSecretBindingNotFound = errors.New("channels: channel secret binding not found")
	// ErrChannelRouteNotFound reports that no persisted route matched the lookup.
	ErrChannelRouteNotFound = errors.New("channels: channel route not found")
	// ErrIngestDedupRecordNotFound reports that no active ingest dedup record matched the lookup.
	ErrIngestDedupRecordNotFound = errors.New("channels: ingest dedup record not found")
	// ErrInvalidChannelStateTransition reports that the requested instance lifecycle transition is not allowed.
	ErrInvalidChannelStateTransition = errors.New("channels: invalid channel state transition")
)

// Scope identifies whether a channel resource is daemon-global or workspace-owned.
type Scope string

const (
	// ScopeGlobal identifies a daemon-global channel resource.
	ScopeGlobal Scope = "global"
	// ScopeWorkspace identifies a workspace-owned channel resource.
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
		return errors.New("channels: scope is required")
	default:
		return fmt.Errorf("channels: unsupported scope %q", s)
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
			return errors.New("channels: global scope cannot include workspace id")
		}
	case ScopeWorkspace:
		if trimmedWorkspaceID == "" {
			return errors.New("channels: workspace scope requires workspace id")
		}
	}

	return nil
}

// ChannelStatus reports the operator-visible lifecycle state of a channel instance.
type ChannelStatus string

const (
	// ChannelStatusDisabled reports an instance that is intentionally disabled.
	ChannelStatusDisabled ChannelStatus = "disabled"
	// ChannelStatusStarting reports an instance that is launching or reconnecting.
	ChannelStatusStarting ChannelStatus = "starting"
	// ChannelStatusReady reports an instance that is healthy and ready to ingest/deliver.
	ChannelStatusReady ChannelStatus = "ready"
	// ChannelStatusDegraded reports an instance that is partially working with known issues.
	ChannelStatusDegraded ChannelStatus = "degraded"
	// ChannelStatusAuthRequired reports an instance that cannot operate until authentication is refreshed.
	ChannelStatusAuthRequired ChannelStatus = "auth_required"
	// ChannelStatusError reports an instance that is unhealthy due to a terminal or repeated fault.
	ChannelStatusError ChannelStatus = "error"
)

// Normalize returns the normalized representation of the status.
func (s ChannelStatus) Normalize() ChannelStatus {
	return ChannelStatus(strings.ToLower(strings.TrimSpace(string(s))))
}

// Validate reports whether the status belongs to the closed channel status set.
func (s ChannelStatus) Validate() error {
	switch s.Normalize() {
	case ChannelStatusDisabled,
		ChannelStatusStarting,
		ChannelStatusReady,
		ChannelStatusDegraded,
		ChannelStatusAuthRequired,
		ChannelStatusError:
		return nil
	case "":
		return errors.New("channels: channel status is required")
	default:
		return fmt.Errorf("channels: unsupported channel status %q", s)
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
		return errors.New("channels: routing policy cannot include thread without peer or group")
	}
	return nil
}

// ChannelInstance is the authoritative persisted configuration for one channel adapter instance.
type ChannelInstance struct {
	ID               string          `json:"id"`
	Scope            Scope           `json:"scope"`
	WorkspaceID      string          `json:"workspace_id,omitempty"`
	Platform         string          `json:"platform"`
	ExtensionName    string          `json:"extension_name"`
	DisplayName      string          `json:"display_name"`
	Enabled          bool            `json:"enabled"`
	Status           ChannelStatus   `json:"status"`
	RoutingPolicy    RoutingPolicy   `json:"routing_policy"`
	DeliveryDefaults json.RawMessage `json:"delivery_defaults,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// Validate reports whether the persisted channel instance shape is complete and valid.
func (i ChannelInstance) Validate() error {
	normalized := i.normalize()
	if err := requireField(normalized.ID, "channel instance id"); err != nil {
		return err
	}
	if err := ValidateScopeWorkspaceID(normalized.Scope, normalized.WorkspaceID); err != nil {
		return err
	}
	if err := requireField(normalized.Platform, "channel instance platform"); err != nil {
		return err
	}
	if err := requireField(normalized.ExtensionName, "channel instance extension name"); err != nil {
		return err
	}
	if err := requireField(normalized.DisplayName, "channel instance display name"); err != nil {
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
	if _, err := normalizeRawJSON(normalized.DeliveryDefaults, "channel instance delivery defaults"); err != nil {
		return err
	}
	return nil
}

// ChannelSecretBinding binds one named channel secret slot to a daemon-managed vault reference.
type ChannelSecretBinding struct {
	ChannelInstanceID string    `json:"channel_instance_id"`
	BindingName       string    `json:"binding_name"`
	VaultRef          string    `json:"vault_ref"`
	Kind              string    `json:"kind"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Validate reports whether the persisted secret binding is complete and valid.
func (b ChannelSecretBinding) Validate() error {
	normalized := b.normalize()
	if err := requireField(normalized.ChannelInstanceID, "channel secret binding channel instance id"); err != nil {
		return err
	}
	if err := requireField(normalized.BindingName, "channel secret binding name"); err != nil {
		return err
	}
	if err := requireField(normalized.VaultRef, "channel secret binding vault ref"); err != nil {
		return err
	}
	if err := requireField(normalized.Kind, "channel secret binding kind"); err != nil {
		return err
	}
	return nil
}

// DeliveryTarget identifies an outbound delivery destination within one channel instance.
type DeliveryTarget struct {
	ChannelInstanceID string       `json:"channel_instance_id"`
	PeerID            string       `json:"peer_id,omitempty"`
	ThreadID          string       `json:"thread_id,omitempty"`
	GroupID           string       `json:"group_id,omitempty"`
	Mode              DeliveryMode `json:"mode,omitempty"`
}

// Validate reports whether the delivery target contains a supported mode and
// the identity fields required by that mode.
func (t DeliveryTarget) Validate() error {
	normalized := t.normalize()
	if err := requireField(normalized.ChannelInstanceID, "delivery target channel instance id"); err != nil {
		return err
	}
	if err := normalized.Mode.Validate(); err != nil {
		return err
	}
	if normalized.ThreadID != "" && normalized.PeerID == "" && normalized.GroupID == "" {
		return fmt.Errorf(
			"channels: delivery target thread id requires peer id or group id for mode %q",
			normalized.Mode,
		)
	}

	switch normalized.Mode {
	case DeliveryModeDirectSend, DeliveryModeReply:
		if normalized.PeerID == "" && normalized.GroupID == "" {
			return fmt.Errorf(
				"channels: delivery target mode %q requires peer id or group id",
				normalized.Mode,
			)
		}
	}

	return nil
}

// IsZero reports whether the target carries any values.
func (t DeliveryTarget) IsZero() bool {
	normalized := t.normalize()
	return normalized.ChannelInstanceID == "" &&
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

// MessageContent carries normalized text content shared by inbound and outbound channel models.
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

// InboundMessageEnvelope is the normalized channel ingest payload delivered by adapters.
type InboundMessageEnvelope struct {
	ChannelInstanceID string              `json:"channel_instance_id"`
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
	if err := requireField(normalized.ChannelInstanceID, "inbound message channel instance id"); err != nil {
		return err
	}
	if err := ValidateScopeWorkspaceID(normalized.Scope, normalized.WorkspaceID); err != nil {
		return err
	}
	if err := requireField(normalized.PlatformMessageID, "inbound message platform message id"); err != nil {
		return err
	}
	if normalized.ReceivedAt.IsZero() {
		return errors.New("channels: inbound message received at is required")
	}
	if err := requireField(normalized.IdempotencyKey, "inbound message idempotency key"); err != nil {
		return err
	}
	return nil
}

// DeliveryEvent is the daemon-owned outbound projection sent to a channel adapter.
type DeliveryEvent struct {
	DeliveryID        string          `json:"delivery_id"`
	ChannelInstanceID string          `json:"channel_instance_id"`
	RoutingKey        RoutingKey      `json:"routing_key"`
	DeliveryTarget    DeliveryTarget  `json:"delivery_target"`
	Seq               int64           `json:"seq"`
	EventType         string          `json:"event_type"`
	Content           MessageContent  `json:"content"`
	Final             bool            `json:"final"`
	Metadata          json.RawMessage `json:"metadata,omitempty"`
}

// Validate reports whether the delivery event contains the required identifiers.
func (e DeliveryEvent) Validate() error {
	normalized := e.normalize()
	if err := requireField(normalized.DeliveryID, "delivery event id"); err != nil {
		return err
	}
	if err := requireField(normalized.ChannelInstanceID, "delivery event channel instance id"); err != nil {
		return err
	}
	if err := normalized.RoutingKey.Validate(); err != nil {
		return err
	}
	if normalized.RoutingKey.ChannelInstanceID != normalized.ChannelInstanceID {
		return errors.New("channels: delivery event channel instance id must match routing key")
	}
	if !normalized.DeliveryTarget.IsZero() {
		if err := normalized.DeliveryTarget.Validate(); err != nil {
			return err
		}
		if normalized.DeliveryTarget.ChannelInstanceID != normalized.ChannelInstanceID {
			return errors.New("channels: delivery target channel instance id must match delivery event")
		}
	}
	if normalized.Seq < 0 {
		return fmt.Errorf("channels: invalid delivery event sequence %d", normalized.Seq)
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
	IdempotencyKey    string    `json:"idempotency_key"`
	ChannelInstanceID string    `json:"channel_instance_id"`
	ReceivedAt        time.Time `json:"received_at"`
	ExpiresAt         time.Time `json:"expires_at"`
}

// Validate reports whether the dedup record is complete and time-consistent.
func (r IngestDedupRecord) Validate() error {
	normalized := r.normalize()
	if err := requireField(normalized.IdempotencyKey, "ingest dedup idempotency key"); err != nil {
		return err
	}
	if err := requireField(normalized.ChannelInstanceID, "ingest dedup channel instance id"); err != nil {
		return err
	}
	if normalized.ReceivedAt.IsZero() {
		return errors.New("channels: ingest dedup received at is required")
	}
	if normalized.ExpiresAt.IsZero() {
		return errors.New("channels: ingest dedup expires at is required")
	}
	if !normalized.ExpiresAt.After(normalized.ReceivedAt) {
		return errors.New("channels: ingest dedup expires at must be after received at")
	}
	return nil
}

func (i ChannelInstance) normalize() ChannelInstance {
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

func (b ChannelSecretBinding) normalize() ChannelSecretBinding {
	normalized := b
	normalized.ChannelInstanceID = strings.TrimSpace(normalized.ChannelInstanceID)
	normalized.BindingName = strings.TrimSpace(normalized.BindingName)
	normalized.VaultRef = strings.TrimSpace(normalized.VaultRef)
	normalized.Kind = strings.TrimSpace(normalized.Kind)
	return normalized
}

func (t DeliveryTarget) normalize() DeliveryTarget {
	normalized := t
	normalized.ChannelInstanceID = strings.TrimSpace(normalized.ChannelInstanceID)
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
	normalized.ChannelInstanceID = strings.TrimSpace(normalized.ChannelInstanceID)
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
	normalized.ChannelInstanceID = strings.TrimSpace(normalized.ChannelInstanceID)
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
	normalized.ChannelInstanceID = strings.TrimSpace(normalized.ChannelInstanceID)
	return normalized
}

func requireField(value string, label string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("channels: %s is required", label)
	}
	return nil
}

func normalizeRawJSON(value json.RawMessage, label string) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 {
		return nil, nil
	}
	if !json.Valid(trimmed) {
		return nil, fmt.Errorf("channels: %s must be valid JSON", label)
	}

	var compacted bytes.Buffer
	if err := json.Compact(&compacted, trimmed); err != nil {
		return nil, fmt.Errorf("channels: compact %s: %w", label, err)
	}

	return compacted.Bytes(), nil
}
