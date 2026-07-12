package bridges

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// NetworkConversationSurface identifies one explicit AGH network conversation container.
type NetworkConversationSurface string

const (
	// NetworkConversationSurfaceThread maps bridge ingress into a public AGH thread.
	NetworkConversationSurfaceThread NetworkConversationSurface = "thread"
	// NetworkConversationSurfaceDirect maps bridge ingress into a resolved AGH direct room.
	NetworkConversationSurfaceDirect NetworkConversationSurface = "direct"
)

// Normalize returns the canonical bridge conversation surface.
func (s NetworkConversationSurface) Normalize() NetworkConversationSurface {
	return NetworkConversationSurface(strings.ToLower(strings.TrimSpace(string(s))))
}

// NetworkConversationRef carries an explicit bridge-to-AGH conversation mapping.
type NetworkConversationRef struct {
	Channel     string                     `json:"channel"`
	Surface     NetworkConversationSurface `json:"surface"`
	ThreadID    string                     `json:"thread_id,omitempty"`
	DirectID    string                     `json:"direct_id,omitempty"`
	WorkID      string                     `json:"work_id,omitempty"`
	ReplyTo     string                     `json:"reply_to,omitempty"`
	TraceID     string                     `json:"trace_id,omitempty"`
	CausationID string                     `json:"causation_id,omitempty"`
}

// Validate reports whether the explicit bridge mapping selects one AGH conversation container.
func (r NetworkConversationRef) Validate() error {
	normalized := r.normalize()
	if err := requireField(normalized.Channel, "network conversation channel"); err != nil {
		return err
	}
	switch normalized.Surface {
	case NetworkConversationSurfaceThread:
		if err := validateBridgeNetworkConversationID(normalized.ThreadID, "thread_id"); err != nil {
			return err
		}
		if normalized.DirectID != "" {
			return errors.New("bridges: network conversation direct_id must be empty for thread surface")
		}
	case NetworkConversationSurfaceDirect:
		if err := validateBridgeNetworkConversationID(normalized.DirectID, "direct_id"); err != nil {
			return err
		}
		if normalized.ThreadID != "" {
			return errors.New("bridges: network conversation thread_id must be empty for direct surface")
		}
	default:
		return fmt.Errorf(
			"bridges: network conversation surface must be one of %q or %q",
			NetworkConversationSurfaceThread,
			NetworkConversationSurfaceDirect,
		)
	}
	if normalized.WorkID != "" {
		if err := validateBridgeNetworkConversationID(normalized.WorkID, "work_id"); err != nil {
			return err
		}
	}
	return nil
}

func (r NetworkConversationRef) normalize() NetworkConversationRef {
	return NetworkConversationRef{
		Channel:     strings.TrimSpace(r.Channel),
		Surface:     r.Surface.Normalize(),
		ThreadID:    strings.TrimSpace(r.ThreadID),
		DirectID:    strings.TrimSpace(r.DirectID),
		WorkID:      strings.TrimSpace(r.WorkID),
		ReplyTo:     strings.TrimSpace(r.ReplyTo),
		TraceID:     strings.TrimSpace(r.TraceID),
		CausationID: strings.TrimSpace(r.CausationID),
	}
}

// InboundMessageEnvelope is the normalized bridge ingest payload delivered by adapters.
type InboundMessageEnvelope struct {
	BridgeInstanceID  string                  `json:"bridge_instance_id"`
	Scope             Scope                   `json:"scope"`
	WorkspaceID       string                  `json:"workspace_id,omitempty"`
	PeerID            string                  `json:"peer_id,omitempty"`
	ThreadID          string                  `json:"thread_id,omitempty"`
	GroupID           string                  `json:"group_id,omitempty"`
	PlatformMessageID string                  `json:"platform_message_id,omitempty"`
	ReceivedAt        time.Time               `json:"received_at"`
	Sender            MessageSender           `json:"sender"`
	Content           MessageContent          `json:"content,omitzero"`
	Attachments       []MessageAttachment     `json:"attachments,omitempty"`
	EventFamily       InboundEventFamily      `json:"event_family"`
	Command           *InboundCommand         `json:"command,omitempty"`
	Action            *InboundAction          `json:"action,omitempty"`
	Reaction          *InboundReaction        `json:"reaction,omitempty"`
	Edit              *InboundEdit            `json:"edit,omitempty"`
	ReplyToText       string                  `json:"reply_to_text,omitempty"`
	ReplyToAuthorID   string                  `json:"reply_to_author_id,omitempty"`
	ReplyToAuthorName string                  `json:"reply_to_author_name,omitempty"`
	Conversation      *NetworkConversationRef `json:"conversation,omitempty"`
	ProviderMetadata  json.RawMessage         `json:"provider_metadata,omitempty"`
	IdempotencyKey    string                  `json:"idempotency_key"`
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
	if normalized.ReceivedAt.IsZero() {
		return errors.New("bridges: inbound message received at is required")
	}
	if err := normalized.EventFamily.Validate(); err != nil {
		return err
	}
	if _, err := normalizeRawJSON(normalized.ProviderMetadata, "inbound provider metadata"); err != nil {
		return err
	}
	if err := normalized.validateNetworkConversation(); err != nil {
		return err
	}
	if err := requireField(normalized.IdempotencyKey, "inbound message idempotency key"); err != nil {
		return err
	}
	return normalized.validatePayload()
}

// NetworkConversationRef returns only the explicit AGH conversation mapping.
func (e InboundMessageEnvelope) NetworkConversationRef() (NetworkConversationRef, bool, error) {
	normalized := e.normalize()
	if normalized.Conversation == nil {
		return NetworkConversationRef{}, false, nil
	}
	ref := normalized.Conversation.normalize()
	if err := ref.Validate(); err != nil {
		return NetworkConversationRef{}, false, err
	}
	return ref, true, nil
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
	normalized.EventFamily = normalized.EventFamily.Normalize()
	if normalized.EventFamily == "" && normalized.Command == nil && normalized.Action == nil &&
		normalized.Reaction == nil && normalized.Edit == nil {
		normalized.EventFamily = InboundEventFamilyMessage
	}
	normalized.ReplyToText = strings.TrimSpace(normalized.ReplyToText)
	normalized.ReplyToAuthorID = strings.TrimSpace(normalized.ReplyToAuthorID)
	normalized.ReplyToAuthorName = strings.TrimSpace(normalized.ReplyToAuthorName)
	normalized.IdempotencyKey = strings.TrimSpace(normalized.IdempotencyKey)
	normalized.ProviderMetadata = bytes.TrimSpace(normalized.ProviderMetadata)
	for idx := range normalized.Attachments {
		normalized.Attachments[idx] = MessageAttachment{
			ID:       strings.TrimSpace(normalized.Attachments[idx].ID),
			Name:     strings.TrimSpace(normalized.Attachments[idx].Name),
			MIMEType: strings.TrimSpace(normalized.Attachments[idx].MIMEType),
			URL:      strings.TrimSpace(normalized.Attachments[idx].URL),
		}
	}
	if normalized.Command != nil {
		command := normalized.Command.normalize()
		normalized.Command = &command
	}
	if normalized.Action != nil {
		action := normalized.Action.normalize()
		normalized.Action = &action
	}
	if normalized.Reaction != nil {
		reaction := normalized.Reaction.normalize()
		normalized.Reaction = &reaction
	}
	if normalized.Edit != nil {
		edit := normalized.Edit.normalize()
		normalized.Edit = &edit
	}
	if normalized.Conversation != nil {
		conversation := normalized.Conversation.normalize()
		normalized.Conversation = &conversation
	}
	return normalized
}

func (e InboundMessageEnvelope) validatePayload() error {
	if err := e.validateReplyContext(); err != nil {
		return err
	}
	switch e.EventFamily {
	case InboundEventFamilyMessage:
		return e.validateMessagePayload()
	case InboundEventFamilyCommand:
		return e.validateCommandPayload()
	case InboundEventFamilyAction:
		return e.validateActionPayload()
	case InboundEventFamilyReaction:
		return e.validateReactionPayload()
	case InboundEventFamilyEdit:
		return e.validateEditPayload()
	default:
		return errors.New("bridges: inbound event family is required")
	}
}

func (e InboundMessageEnvelope) validateReplyContext() error {
	if e.ReplyToText == "" && e.ReplyToAuthorID == "" && e.ReplyToAuthorName == "" {
		return nil
	}
	if e.EventFamily != InboundEventFamilyMessage && e.EventFamily != InboundEventFamilyEdit {
		return fmt.Errorf("bridges: inbound %s family cannot include reply context", e.EventFamily)
	}
	return nil
}

func (e InboundMessageEnvelope) validateMessagePayload() error {
	if e.Command != nil || e.Action != nil || e.Reaction != nil || e.Edit != nil {
		return errors.New("bridges: inbound message family cannot include typed interaction payloads")
	}
	return requireField(e.PlatformMessageID, "inbound message platform message id")
}

func (e InboundMessageEnvelope) validateCommandPayload() error {
	if e.Command == nil {
		return errors.New("bridges: inbound command family requires command payload")
	}
	if e.Action != nil || e.Reaction != nil || e.Edit != nil {
		return errors.New("bridges: inbound command family cannot include action, reaction, or edit payloads")
	}
	if err := e.validateMessageFieldsAbsent("command"); err != nil {
		return err
	}
	return e.Command.Validate()
}

func (e InboundMessageEnvelope) validateActionPayload() error {
	if e.Action == nil {
		return errors.New("bridges: inbound action family requires action payload")
	}
	if e.Command != nil || e.Reaction != nil || e.Edit != nil {
		return errors.New("bridges: inbound action family cannot include command, reaction, or edit payloads")
	}
	if err := e.validateMessageFieldsAbsent("action"); err != nil {
		return err
	}
	return e.Action.Validate()
}

func (e InboundMessageEnvelope) validateReactionPayload() error {
	if e.Reaction == nil {
		return errors.New("bridges: inbound reaction family requires reaction payload")
	}
	if e.Command != nil || e.Action != nil || e.Edit != nil {
		return errors.New("bridges: inbound reaction family cannot include command, action, or edit payloads")
	}
	if err := e.validateMessageFieldsAbsent("reaction"); err != nil {
		return err
	}
	return e.Reaction.Validate()
}

func (e InboundMessageEnvelope) validateEditPayload() error {
	if e.Edit == nil {
		return errors.New("bridges: inbound edit family requires edit payload")
	}
	if e.Command != nil || e.Action != nil || e.Reaction != nil {
		return errors.New("bridges: inbound edit family cannot include command, action, or reaction payloads")
	}
	if err := e.validateMessageFieldsAbsent("edit"); err != nil {
		return err
	}
	return e.Edit.Validate()
}

func (e InboundMessageEnvelope) validateMessageFieldsAbsent(family string) error {
	if e.PlatformMessageID != "" || strings.TrimSpace(e.Content.Text) != "" || len(e.Attachments) > 0 {
		return fmt.Errorf("bridges: inbound %s family cannot include message payload fields", family)
	}
	return nil
}

func (e InboundMessageEnvelope) validateNetworkConversation() error {
	if e.Conversation == nil {
		return nil
	}
	return e.Conversation.Validate()
}

func validateBridgeNetworkConversationID(value string, field string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("bridges: network conversation %s is required", field)
	}
	if len(trimmed) > 128 || strings.ContainsAny(trimmed, `/\`) || containsControlCharacter(trimmed) {
		return fmt.Errorf("bridges: invalid network conversation %s %q", field, value)
	}
	switch field {
	case "thread_id":
		if !strings.HasPrefix(trimmed, "thread_") {
			return fmt.Errorf("bridges: invalid network conversation thread_id %q", value)
		}
	case "direct_id":
		if !strings.HasPrefix(trimmed, "direct_") || len(trimmed) != len("direct_")+32 {
			return fmt.Errorf("bridges: invalid network conversation direct_id %q", value)
		}
	}
	return nil
}

func containsControlCharacter(value string) bool {
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return true
		}
	}
	return false
}
