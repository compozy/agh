package bridges

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// DeliveryMessageReference identifies one previously delivered message.
type DeliveryMessageReference struct {
	DeliveryID      string `json:"delivery_id,omitempty"`
	RemoteMessageID string `json:"remote_message_id,omitempty"`
}

// Validate reports whether the reference identifies at least one prior message handle.
func (r DeliveryMessageReference) Validate() error {
	normalized := r.normalize()
	if normalized.DeliveryID == "" && normalized.RemoteMessageID == "" {
		return errors.New("bridges: delivery reference requires delivery id or remote message id")
	}
	return nil
}

// DeliveryErrorDetail captures one typed delivery failure payload.
type DeliveryErrorDetail struct {
	Message string `json:"message"`
}

// Validate reports whether the error detail carries a message.
func (d DeliveryErrorDetail) Validate() error {
	return requireField(strings.TrimSpace(d.Message), "delivery error message")
}

// DeliveryResumeState captures the typed resumable delivery phase.
type DeliveryResumeState struct {
	LatestEventType DeliveryEventType `json:"latest_event_type"`
}

// Validate reports whether the resume state references a supported prior event type.
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

// DeliveryEvent is the daemon-owned outbound projection sent to a bridge adapter.
type DeliveryEvent struct {
	DeliveryID       string                    `json:"delivery_id"`
	BridgeInstanceID string                    `json:"bridge_instance_id"`
	RoutingKey       RoutingKey                `json:"routing_key"`
	DeliveryTarget   DeliveryTarget            `json:"delivery_target"`
	Seq              int64                     `json:"seq"`
	EventType        DeliveryEventType         `json:"event_type"`
	Content          MessageContent            `json:"content"`
	Final            bool                      `json:"final"`
	Operation        DeliveryOperation         `json:"operation,omitempty"`
	Reference        *DeliveryMessageReference `json:"reference,omitempty"`
	Error            *DeliveryErrorDetail      `json:"error,omitempty"`
	Resume           *DeliveryResumeState      `json:"resume,omitempty"`
	Progress         *ToolProgress             `json:"progress,omitempty"`
	ProviderMetadata json.RawMessage           `json:"provider_metadata,omitempty"`
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
	if err := normalized.Operation.Validate(); err != nil {
		return err
	}
	if err := validateDeliveryEventType(normalized.EventType, normalized.Final); err != nil {
		return err
	}
	if _, err := normalizeRawJSON(normalized.ProviderMetadata, "delivery event provider metadata"); err != nil {
		return err
	}
	if err := normalized.validateOperation(); err != nil {
		return err
	}
	return normalized.validateTypedFields()
}

func (e DeliveryEvent) normalize() DeliveryEvent {
	normalized := e
	normalized.DeliveryID = strings.TrimSpace(normalized.DeliveryID)
	normalized.BridgeInstanceID = strings.TrimSpace(normalized.BridgeInstanceID)
	normalized.RoutingKey = normalized.RoutingKey.normalize()
	normalized.DeliveryTarget = normalized.DeliveryTarget.normalize()
	normalized.EventType = normalizeDeliveryEventType(normalized.EventType)
	normalized.Content = MessageContent{Text: strings.TrimSpace(normalized.Content.Text)}
	normalized.Operation = normalized.Operation.Normalize()
	if normalized.Operation == "" {
		normalized.Operation = DeliveryOperationPost
	}
	normalized.ProviderMetadata = bytes.TrimSpace(normalized.ProviderMetadata)
	if normalized.Reference != nil {
		reference := normalized.Reference.normalize()
		normalized.Reference = &reference
	}
	if normalized.Error != nil {
		errorDetail := normalized.Error.normalize()
		normalized.Error = &errorDetail
	}
	if normalized.Resume != nil {
		resume := normalized.Resume.normalize()
		normalized.Resume = &resume
	}
	normalized.Progress = cloneToolProgress(normalized.Progress)
	return normalized
}

// IsLifecycleOnlyFinal reports whether the event closes a progress-only POST
// without materializing or replacing provider message content.
func (e DeliveryEvent) IsLifecycleOnlyFinal() bool {
	operation := e.Operation.Normalize()
	if operation == "" {
		operation = DeliveryOperationPost
	}
	return normalizeDeliveryEventType(e.EventType) == DeliveryEventTypeFinal &&
		operation == DeliveryOperationPost &&
		strings.TrimSpace(e.Content.Text) == ""
}

func (r DeliveryMessageReference) normalize() DeliveryMessageReference {
	return DeliveryMessageReference{
		DeliveryID:      strings.TrimSpace(r.DeliveryID),
		RemoteMessageID: strings.TrimSpace(r.RemoteMessageID),
	}
}

func (d DeliveryErrorDetail) normalize() DeliveryErrorDetail {
	return DeliveryErrorDetail{Message: strings.TrimSpace(d.Message)}
}

func (s DeliveryResumeState) normalize() DeliveryResumeState {
	return DeliveryResumeState{LatestEventType: normalizeDeliveryEventType(s.LatestEventType)}
}

func (e DeliveryEvent) validateOperation() error {
	switch e.Operation {
	case DeliveryOperationPost:
		if e.Reference != nil {
			return errors.New("bridges: delivery post operation cannot include a reference")
		}
	case DeliveryOperationEdit, DeliveryOperationDelete:
		if e.Reference == nil {
			return fmt.Errorf("bridges: delivery %s operation requires a reference", e.Operation)
		}
		if err := e.Reference.Validate(); err != nil {
			return err
		}
	}
	if e.EventType == DeliveryEventTypeDelete && e.Operation != DeliveryOperationDelete {
		return errors.New("bridges: delete delivery events must use delete operation")
	}
	if e.EventType != DeliveryEventTypeDelete && e.Operation == DeliveryOperationDelete {
		return errors.New("bridges: delete operation requires delete event type")
	}
	if e.EventType == DeliveryEventTypeProgress && e.Operation != DeliveryOperationPost {
		return errors.New("bridges: progress delivery events must use post operation")
	}
	return nil
}

func (e DeliveryEvent) validateTypedFields() error {
	switch e.EventType {
	case DeliveryEventTypeError:
		return e.validateErrorFields()
	case DeliveryEventTypeResume:
		return e.validateResumeFields()
	case DeliveryEventTypeDelete:
		return e.validateDeleteFields()
	case DeliveryEventTypeProgress:
		return e.validateProgressFields()
	default:
		return e.validateUntypedFields()
	}
}

func (e DeliveryEvent) validateErrorFields() error {
	if e.Error == nil {
		return errors.New("bridges: delivery error events require an error payload")
	}
	if err := e.Error.Validate(); err != nil {
		return err
	}
	if e.Resume != nil || e.Progress != nil {
		return errors.New("bridges: delivery error events cannot include resume or progress payloads")
	}
	return nil
}

func (e DeliveryEvent) validateResumeFields() error {
	if e.Resume == nil {
		return errors.New("bridges: delivery resume events require a resume payload")
	}
	if err := e.Resume.Validate(); err != nil {
		return err
	}
	if e.Error != nil || e.Progress != nil {
		return errors.New("bridges: delivery resume events cannot include error or progress payloads")
	}
	return nil
}

func (e DeliveryEvent) validateDeleteFields() error {
	if strings.TrimSpace(e.Content.Text) != "" {
		return errors.New("bridges: delivery delete events cannot include message content")
	}
	if e.Error != nil || e.Resume != nil || e.Progress != nil {
		return errors.New("bridges: delivery delete events cannot include typed payloads")
	}
	return nil
}

func (e DeliveryEvent) validateProgressFields() error {
	if e.Progress == nil {
		return errors.New("bridges: delivery progress events require a progress payload")
	}
	if err := e.Progress.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(e.Content.Text) != "" {
		return errors.New("bridges: delivery progress events cannot include message content")
	}
	if e.Error != nil || e.Resume != nil {
		return errors.New("bridges: delivery progress events cannot include error or resume payloads")
	}
	return nil
}

func (e DeliveryEvent) validateUntypedFields() error {
	if e.Error != nil {
		return errors.New("bridges: only delivery error events may include error payload")
	}
	if e.Resume != nil {
		return errors.New("bridges: only delivery resume events may include resume payload")
	}
	if e.Progress != nil {
		return errors.New("bridges: only delivery progress events may include progress payload")
	}
	return nil
}
