package contract

import (
	"errors"
	"fmt"
	"strings"
)

// DeliveryMode identifies daemon-owned outbound delivery behavior.
type DeliveryMode string

const (
	DeliveryModeDirectSend DeliveryMode = "direct-send"
	DeliveryModeReply      DeliveryMode = "reply"
)

// Normalize returns the canonical delivery mode.
func (m DeliveryMode) Normalize() DeliveryMode {
	switch normalized := strings.ToLower(strings.TrimSpace(string(m))); normalized {
	case "direct", "direct-send", "direct_send", "send":
		return DeliveryModeDirectSend
	case "reply", "reply-send", "reply_send":
		return DeliveryModeReply
	default:
		return DeliveryMode(normalized)
	}
}

// Validate reports whether the delivery mode is supported.
func (m DeliveryMode) Validate() error {
	switch m.Normalize() {
	case DeliveryModeDirectSend, DeliveryModeReply:
		return nil
	case "":
		return errors.New("bridges: delivery target mode is required")
	default:
		return fmt.Errorf("bridges: unsupported delivery target mode %q", strings.TrimSpace(string(m)))
	}
}

// DeliveryTarget identifies an outbound destination within one bridge instance.
type DeliveryTarget struct {
	BridgeInstanceID string       `json:"bridge_instance_id"`
	PeerID           string       `json:"peer_id,omitempty"`
	ThreadID         string       `json:"thread_id,omitempty"`
	GroupID          string       `json:"group_id,omitempty"`
	Mode             DeliveryMode `json:"mode,omitempty"`
}

// NormalizeDeliveryTarget returns the canonical wire destination.
func NormalizeDeliveryTarget(target DeliveryTarget) DeliveryTarget {
	return target.normalize()
}

// Validate reports whether the target contains the required identity.
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
	if normalized.PeerID == "" && normalized.GroupID == "" {
		return fmt.Errorf(
			"bridges: delivery target mode %q requires peer id or group id",
			normalized.Mode,
		)
	}
	return nil
}

// IsZero reports whether the target carries any values.
func (t DeliveryTarget) IsZero() bool {
	normalized := t.normalize()
	return normalized.BridgeInstanceID == "" && normalized.PeerID == "" &&
		normalized.ThreadID == "" && normalized.GroupID == "" && normalized.Mode == ""
}

func (t DeliveryTarget) normalize() DeliveryTarget {
	t.BridgeInstanceID = strings.TrimSpace(t.BridgeInstanceID)
	t.PeerID = strings.TrimSpace(t.PeerID)
	t.ThreadID = strings.TrimSpace(t.ThreadID)
	t.GroupID = strings.TrimSpace(t.GroupID)
	t.Mode = t.Mode.Normalize()
	return t
}

// MessageSender identifies the upstream actor that produced a message.
type MessageSender struct {
	ID          string `json:"id,omitempty"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

// MessageContent carries normalized text content.
type MessageContent struct {
	Text string `json:"text,omitempty"`
}

// MessageAttachment captures normalized attachment metadata.
type MessageAttachment struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	MIMEType string `json:"mime_type,omitempty"`
	URL      string `json:"url,omitempty"`
}

// DeliveryOperation identifies post, edit, or delete behavior.
type DeliveryOperation string

const (
	DeliveryOperationPost   DeliveryOperation = "post"
	DeliveryOperationEdit   DeliveryOperation = "edit"
	DeliveryOperationDelete DeliveryOperation = "delete"
)

// Normalize returns the canonical delivery operation.
func (o DeliveryOperation) Normalize() DeliveryOperation {
	return DeliveryOperation(strings.ToLower(strings.TrimSpace(string(o))))
}

// Validate reports whether the delivery operation is supported.
func (o DeliveryOperation) Validate() error {
	switch o.Normalize() {
	case "", DeliveryOperationPost, DeliveryOperationEdit, DeliveryOperationDelete:
		return nil
	default:
		return fmt.Errorf("bridges: unsupported delivery operation %q", strings.TrimSpace(string(o)))
	}
}

// DeliveryEventType is one closed daemon-to-adapter lifecycle event.
type DeliveryEventType string

const (
	DeliveryEventTypeStart    DeliveryEventType = "start"
	DeliveryEventTypeDelta    DeliveryEventType = "delta"
	DeliveryEventTypeFinal    DeliveryEventType = "final"
	DeliveryEventTypeError    DeliveryEventType = "error"
	DeliveryEventTypeResume   DeliveryEventType = "resume"
	DeliveryEventTypeDelete   DeliveryEventType = "delete"
	DeliveryEventTypeProgress DeliveryEventType = "progress"
)

// DeliveryEventTypeValues returns the closed wire values in stable order.
func DeliveryEventTypeValues() []string {
	return []string{
		string(DeliveryEventTypeStart), string(DeliveryEventTypeDelta),
		string(DeliveryEventTypeFinal), string(DeliveryEventTypeError),
		string(DeliveryEventTypeResume), string(DeliveryEventTypeDelete),
		string(DeliveryEventTypeProgress),
	}
}

// NormalizeDeliveryEventType returns the canonical lifecycle event type.
func NormalizeDeliveryEventType(value DeliveryEventType) DeliveryEventType {
	return normalizeDeliveryEventType(value)
}

// IsTerminalDeliveryEventType reports whether a lifecycle event is terminal.
func IsTerminalDeliveryEventType(value DeliveryEventType) bool {
	return isTerminalDeliveryEventType(value)
}

func normalizeDeliveryEventType(value DeliveryEventType) DeliveryEventType {
	return DeliveryEventType(strings.ToLower(strings.TrimSpace(string(value))))
}

func isTerminalDeliveryEventType(value DeliveryEventType) bool {
	switch normalizeDeliveryEventType(value) {
	case DeliveryEventTypeFinal, DeliveryEventTypeError, DeliveryEventTypeDelete:
		return true
	default:
		return false
	}
}

func validateDeliveryEventType(value DeliveryEventType, final bool) error {
	switch normalizeDeliveryEventType(value) {
	case DeliveryEventTypeStart:
		if final {
			return errors.New("bridges: delivery start event cannot be final")
		}
	case DeliveryEventTypeDelta:
		if final {
			return errors.New("bridges: delivery delta event cannot be final")
		}
	case DeliveryEventTypeProgress:
		if final {
			return errors.New("bridges: delivery progress event cannot be final")
		}
	case DeliveryEventTypeFinal:
		if !final {
			return errors.New("bridges: delivery final event must set final=true")
		}
	case DeliveryEventTypeError:
		if !final {
			return errors.New("bridges: delivery error event must set final=true")
		}
	case DeliveryEventTypeResume:
		return nil
	case DeliveryEventTypeDelete:
		if !final {
			return errors.New("bridges: delivery delete event must set final=true")
		}
	case "":
		return errors.New("bridges: delivery event type is required")
	default:
		return fmt.Errorf("bridges: unsupported delivery event type %q", strings.TrimSpace(string(value)))
	}
	return nil
}

// DeliveryMessageReference identifies one previously delivered message.
type DeliveryMessageReference struct {
	DeliveryID      string `json:"delivery_id,omitempty"`
	RemoteMessageID string `json:"remote_message_id,omitempty"`
}

// NormalizeDeliveryMessageReference returns a canonical prior-message handle.
func NormalizeDeliveryMessageReference(reference DeliveryMessageReference) DeliveryMessageReference {
	return reference.normalize()
}

// Validate reports whether the reference carries a prior message handle.
func (r DeliveryMessageReference) Validate() error {
	normalized := r.normalize()
	if normalized.DeliveryID == "" && normalized.RemoteMessageID == "" {
		return errors.New("bridges: delivery reference requires delivery id or remote message id")
	}
	return nil
}

func (r DeliveryMessageReference) normalize() DeliveryMessageReference {
	return DeliveryMessageReference{
		DeliveryID: strings.TrimSpace(r.DeliveryID), RemoteMessageID: strings.TrimSpace(r.RemoteMessageID),
	}
}

// DeliveryErrorDetail captures a typed delivery failure.
type DeliveryErrorDetail struct {
	Message string `json:"message"`
}

// Validate reports whether the error carries a message.
func (d DeliveryErrorDetail) Validate() error {
	return requireField(strings.TrimSpace(d.Message), "delivery error message")
}

func (d DeliveryErrorDetail) normalize() DeliveryErrorDetail {
	return DeliveryErrorDetail{Message: strings.TrimSpace(d.Message)}
}

// DeliveryResumeState captures the resumable delivery phase.
type DeliveryResumeState struct {
	LatestEventType DeliveryEventType `json:"latest_event_type"`
}

// Validate reports whether the resume state references a supported event.
func (s DeliveryResumeState) Validate() error {
	normalized := s.normalize()
	if normalized.LatestEventType == "" {
		return errors.New("bridges: delivery resume latest event type is required")
	}
	if normalized.LatestEventType == DeliveryEventTypeResume {
		return errors.New("bridges: delivery resume latest event type cannot itself be resume")
	}
	return validateDeliveryEventType(
		normalized.LatestEventType,
		isTerminalDeliveryEventType(normalized.LatestEventType),
	)
}

func (s DeliveryResumeState) normalize() DeliveryResumeState {
	return DeliveryResumeState{LatestEventType: normalizeDeliveryEventType(s.LatestEventType)}
}

// DeliveryAck acknowledges one delivery request.
type DeliveryAck struct {
	DeliveryID             string `json:"delivery_id,omitempty"`
	Seq                    int64  `json:"seq,omitempty"`
	RemoteMessageID        string `json:"remote_message_id,omitempty"`
	ReplaceRemoteMessageID string `json:"replace_remote_message_id,omitempty"`
}

// NormalizeDeliveryAck returns a canonical acknowledgement.
func NormalizeDeliveryAck(ack DeliveryAck) DeliveryAck {
	return ack.normalize()
}

// ValidateFor reports whether the acknowledgement belongs to an event.
func (a DeliveryAck) ValidateFor(event DeliveryEvent) error {
	normalized := a.normalize()
	if normalized.DeliveryID != "" && normalized.DeliveryID != strings.TrimSpace(event.DeliveryID) {
		return fmt.Errorf(
			"bridges: delivery ack delivery id %q does not match event %q",
			normalized.DeliveryID,
			strings.TrimSpace(event.DeliveryID),
		)
	}
	if normalized.Seq != 0 && normalized.Seq != event.Seq {
		return fmt.Errorf("bridges: delivery ack sequence %d does not match event %d", normalized.Seq, event.Seq)
	}
	return nil
}

func (a DeliveryAck) normalize() DeliveryAck {
	a.DeliveryID = strings.TrimSpace(a.DeliveryID)
	a.RemoteMessageID = strings.TrimSpace(a.RemoteMessageID)
	a.ReplaceRemoteMessageID = strings.TrimSpace(a.ReplaceRemoteMessageID)
	return a
}
