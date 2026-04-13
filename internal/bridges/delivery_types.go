package bridges

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrDeliveryNotFound reports that no active or retained delivery matched the lookup.
	ErrDeliveryNotFound = errors.New("bridges: delivery not found")
	// ErrDeliveryQueueSaturated reports that a bounded delivery queue could not accept more work.
	ErrDeliveryQueueSaturated = errors.New("bridges: delivery queue saturated")
	// ErrDeliveryTransportUnavailable reports that the broker has no usable extension delivery transport.
	ErrDeliveryTransportUnavailable = errors.New("bridges: delivery transport unavailable")
)

const (
	// DeliveryEventTypeStart starts one progressive outbound delivery for a prompt turn.
	DeliveryEventTypeStart = "start"
	// DeliveryEventTypeDelta updates one progressive outbound delivery with newer full text.
	DeliveryEventTypeDelta = "delta"
	// DeliveryEventTypeFinal reports the terminal successful state for one delivery.
	DeliveryEventTypeFinal = "final"
	// DeliveryEventTypeError reports the terminal failed state for one delivery.
	DeliveryEventTypeError = "error"
	// DeliveryEventTypeResume rehydrates the latest delivery snapshot after adapter recovery.
	DeliveryEventTypeResume = "resume"
)

const (
	defaultDeliveryQueueCapacity  = 4
	defaultDeliveryRetryDelay     = 25 * time.Millisecond
	defaultDeliveryRequestTimeout = 5 * time.Second
)

// DeliveryTransport delivers negotiated daemon->extension bridge requests.
// The extension name remains explicit because the broker owns routing semantics,
// while the extension manager owns the subprocess runtime.
type DeliveryTransport interface {
	DeliverBridge(ctx context.Context, extensionName string, req DeliveryRequest) (DeliveryAck, error)
}

// DeliveryBroker is the daemon-owned outbound delivery surface used by the
// bridge runtime. Prompt projection registers live deliveries separately and
// then enqueues projected events through Deliver.
type DeliveryBroker interface {
	Deliver(ctx context.Context, evt DeliveryEvent) error
	Snapshot(ctx context.Context, deliveryID string) (*DeliverySnapshot, error)
}

// DeliveryProjectionEvent is the reduced session-event shape the broker needs
// to project prompt output into delivery-oriented bridge events. It remains
// ACP-agnostic so `internal/bridges` does not depend on runtime transport packages.
type DeliveryProjectionEvent struct {
	Type        string    `json:"type"`
	TurnID      string    `json:"turn_id"`
	Timestamp   time.Time `json:"timestamp,omitempty"`
	Text        string    `json:"text,omitempty"`
	Error       string    `json:"error,omitempty"`
	Fingerprint string    `json:"fingerprint,omitempty"`
}

// DeliveryRequest is the negotiated daemon->extension request payload for
// `bridges/deliver`. Regular streaming requests carry only Event. Recovery
// requests also carry Snapshot and use EventTypeResume.
type DeliveryRequest struct {
	Event    DeliveryEvent     `json:"event"`
	Snapshot *DeliverySnapshot `json:"snapshot,omitempty"`
}

// Validate reports whether the negotiated request is internally consistent.
func (r DeliveryRequest) Validate() error {
	if err := r.Event.Validate(); err != nil {
		return err
	}
	if r.Snapshot == nil {
		if normalizeDeliveryEventType(r.Event.EventType) == DeliveryEventTypeResume {
			return errors.New("bridges: resume delivery request requires a snapshot")
		}
		return nil
	}

	if err := r.Snapshot.Validate(); err != nil {
		return err
	}
	if r.Snapshot.DeliveryID != r.Event.DeliveryID {
		return errors.New("bridges: delivery request snapshot must match event delivery id")
	}
	if r.Snapshot.BridgeInstanceID != r.Event.BridgeInstanceID {
		return errors.New("bridges: delivery request snapshot must match event bridge instance id")
	}
	return nil
}

// DeliveryAck is the negotiated extension->daemon acknowledgement payload for
// one `bridges/deliver` request.
type DeliveryAck struct {
	DeliveryID             string `json:"delivery_id,omitempty"`
	Seq                    int64  `json:"seq,omitempty"`
	RemoteMessageID        string `json:"remote_message_id,omitempty"`
	ReplaceRemoteMessageID string `json:"replace_remote_message_id,omitempty"`
}

// ValidateFor reports whether the acknowledgement still belongs to the request
// that triggered it.
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

// DeliverySnapshot captures the current progressive state for one active
// delivery so the broker can resume it after adapter recovery.
type DeliverySnapshot struct {
	DeliveryID             string         `json:"delivery_id"`
	SessionID              string         `json:"session_id"`
	TurnID                 string         `json:"turn_id"`
	BridgeInstanceID       string         `json:"bridge_instance_id"`
	RoutingKey             RoutingKey     `json:"routing_key"`
	DeliveryTarget         DeliveryTarget `json:"delivery_target"`
	LatestSeq              int64          `json:"latest_seq"`
	LatestEventType        string         `json:"latest_event_type"`
	CurrentContent         MessageContent `json:"current_content,omitempty"`
	LastSentSeq            int64          `json:"last_sent_seq,omitempty"`
	LastAckedSeq           int64          `json:"last_acked_seq,omitempty"`
	RemoteMessageID        string         `json:"remote_message_id,omitempty"`
	ReplaceRemoteMessageID string         `json:"replace_remote_message_id,omitempty"`
	Final                  bool           `json:"final"`
	Error                  string         `json:"error,omitempty"`
	UpdatedAt              time.Time      `json:"updated_at"`
}

// Validate reports whether the snapshot contains the state needed to resume a
// negotiated bridge delivery.
func (s DeliverySnapshot) Validate() error {
	normalized := s.normalize()
	if err := requireField(normalized.DeliveryID, "delivery snapshot id"); err != nil {
		return err
	}
	if err := requireField(normalized.SessionID, "delivery snapshot session id"); err != nil {
		return err
	}
	if err := requireField(normalized.TurnID, "delivery snapshot turn id"); err != nil {
		return err
	}
	if err := requireField(normalized.BridgeInstanceID, "delivery snapshot bridge instance id"); err != nil {
		return err
	}
	if err := normalized.RoutingKey.Validate(); err != nil {
		return err
	}
	if normalized.RoutingKey.BridgeInstanceID != normalized.BridgeInstanceID {
		return errors.New("bridges: delivery snapshot bridge instance id must match routing key")
	}
	if err := normalized.DeliveryTarget.Validate(); err != nil {
		return err
	}
	if normalized.DeliveryTarget.BridgeInstanceID != normalized.BridgeInstanceID {
		return errors.New("bridges: delivery snapshot bridge instance id must match delivery target")
	}
	if normalized.LatestSeq < 0 {
		return fmt.Errorf("bridges: invalid delivery snapshot latest sequence %d", normalized.LatestSeq)
	}
	if normalized.LastSentSeq < 0 {
		return fmt.Errorf("bridges: invalid delivery snapshot last sent sequence %d", normalized.LastSentSeq)
	}
	if normalized.LastAckedSeq < 0 {
		return fmt.Errorf("bridges: invalid delivery snapshot last acked sequence %d", normalized.LastAckedSeq)
	}
	if normalized.LastAckedSeq > normalized.LastSentSeq {
		return errors.New("bridges: delivery snapshot last acked sequence cannot exceed last sent sequence")
	}
	if normalized.LastSentSeq > normalized.LatestSeq {
		return errors.New("bridges: delivery snapshot last sent sequence cannot exceed latest sequence")
	}
	if normalized.UpdatedAt.IsZero() {
		return errors.New("bridges: delivery snapshot updated at is required")
	}
	if err := validateDeliveryEventType(normalized.LatestEventType, normalized.Final); err != nil {
		return err
	}
	return nil
}

// PromptDeliveryRegistration binds one session prompt turn to a routed bridge
// delivery stream before or shortly after the prompt begins emitting events.
type PromptDeliveryRegistration struct {
	SessionID      string                    `json:"session_id"`
	TurnID         string                    `json:"turn_id"`
	ExtensionName  string                    `json:"extension_name"`
	DeliveryID     string                    `json:"delivery_id,omitempty"`
	RoutingKey     RoutingKey                `json:"routing_key"`
	DeliveryTarget DeliveryTarget            `json:"delivery_target"`
	SeedEvents     []DeliveryProjectionEvent `json:"seed_events,omitempty"`
}

// Validate reports whether the registration contains enough routed context to
// project session output into a negotiated delivery stream.
func (r PromptDeliveryRegistration) Validate() error {
	normalized := r.normalize()
	if err := requireField(normalized.SessionID, "prompt delivery registration session id"); err != nil {
		return err
	}
	if err := requireField(normalized.TurnID, "prompt delivery registration turn id"); err != nil {
		return err
	}
	if err := requireField(normalized.ExtensionName, "prompt delivery registration extension name"); err != nil {
		return err
	}
	if err := normalized.RoutingKey.Validate(); err != nil {
		return err
	}
	if err := normalized.DeliveryTarget.Validate(); err != nil {
		return err
	}
	if normalized.DeliveryTarget.BridgeInstanceID != normalized.RoutingKey.BridgeInstanceID {
		return errors.New("bridges: prompt delivery registration target must match routing key bridge instance")
	}
	return nil
}

// DeliveryBrokerOption customizes delivery-broker construction.
type DeliveryBrokerOption func(*Broker)

// WithDeliveryBrokerNow overrides the broker clock, mainly for tests.
func WithDeliveryBrokerNow(now func() time.Time) DeliveryBrokerOption {
	return func(b *Broker) {
		if now != nil {
			b.now = now
		}
	}
}

// WithDeliveryBrokerQueueCapacity overrides the bounded queue length per routed
// delivery worker. Values below 2 are raised to 2 so `start` and one terminal
// event can coexist under pressure.
func WithDeliveryBrokerQueueCapacity(capacity int) DeliveryBrokerOption {
	return func(b *Broker) {
		if capacity < 2 {
			capacity = 2
		}
		b.queueCapacity = capacity
	}
}

// WithDeliveryBrokerRetryDelay overrides the backoff between retry attempts
// after a delivery-transport failure.
func WithDeliveryBrokerRetryDelay(delay time.Duration) DeliveryBrokerOption {
	return func(b *Broker) {
		if delay > 0 {
			b.retryDelay = delay
		}
	}
}

// WithDeliveryBrokerRequestTimeout overrides the timeout applied to one
// negotiated `bridges/deliver` call.
func WithDeliveryBrokerRequestTimeout(timeout time.Duration) DeliveryBrokerOption {
	return func(b *Broker) {
		if timeout > 0 {
			b.requestTimeout = timeout
		}
	}
}

// WithDeliveryBrokerLifecycleContext injects the broker-owned lifecycle context
// used by background route workers.
func WithDeliveryBrokerLifecycleContext(ctx context.Context) DeliveryBrokerOption {
	return func(b *Broker) {
		if ctx != nil {
			b.lifecycleCtx = ctx
		}
	}
}

func normalizeDeliveryEventType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validateDeliveryEventType(value string, final bool) error {
	switch normalizeDeliveryEventType(value) {
	case DeliveryEventTypeStart:
		if final {
			return errors.New("bridges: delivery start event cannot be final")
		}
		return nil
	case DeliveryEventTypeDelta:
		if final {
			return errors.New("bridges: delivery delta event cannot be final")
		}
		return nil
	case DeliveryEventTypeFinal:
		if !final {
			return errors.New("bridges: delivery final event must set final=true")
		}
		return nil
	case DeliveryEventTypeError:
		if !final {
			return errors.New("bridges: delivery error event must set final=true")
		}
		return nil
	case DeliveryEventTypeResume:
		return nil
	case "":
		return errors.New("bridges: delivery event type is required")
	default:
		return fmt.Errorf("bridges: unsupported delivery event type %q", strings.TrimSpace(value))
	}
}

func (a DeliveryAck) normalize() DeliveryAck {
	normalized := a
	normalized.DeliveryID = strings.TrimSpace(normalized.DeliveryID)
	normalized.RemoteMessageID = strings.TrimSpace(normalized.RemoteMessageID)
	normalized.ReplaceRemoteMessageID = strings.TrimSpace(normalized.ReplaceRemoteMessageID)
	return normalized
}

func (s DeliverySnapshot) normalize() DeliverySnapshot {
	normalized := s
	normalized.DeliveryID = strings.TrimSpace(normalized.DeliveryID)
	normalized.SessionID = strings.TrimSpace(normalized.SessionID)
	normalized.TurnID = strings.TrimSpace(normalized.TurnID)
	normalized.BridgeInstanceID = strings.TrimSpace(normalized.BridgeInstanceID)
	normalized.RoutingKey = normalized.RoutingKey.normalize()
	normalized.DeliveryTarget = normalized.DeliveryTarget.normalize()
	normalized.LatestEventType = normalizeDeliveryEventType(normalized.LatestEventType)
	normalized.RemoteMessageID = strings.TrimSpace(normalized.RemoteMessageID)
	normalized.ReplaceRemoteMessageID = strings.TrimSpace(normalized.ReplaceRemoteMessageID)
	normalized.Error = strings.TrimSpace(normalized.Error)
	return normalized
}

func (r PromptDeliveryRegistration) normalize() PromptDeliveryRegistration {
	normalized := r
	normalized.SessionID = strings.TrimSpace(normalized.SessionID)
	normalized.TurnID = strings.TrimSpace(normalized.TurnID)
	normalized.ExtensionName = strings.TrimSpace(normalized.ExtensionName)
	normalized.DeliveryID = strings.TrimSpace(normalized.DeliveryID)
	normalized.RoutingKey = normalized.RoutingKey.normalize()
	normalized.DeliveryTarget = normalized.DeliveryTarget.normalize()
	if len(normalized.SeedEvents) > 0 {
		normalized.SeedEvents = append([]DeliveryProjectionEvent(nil), normalized.SeedEvents...)
		for idx := range normalized.SeedEvents {
			normalized.SeedEvents[idx] = normalized.SeedEvents[idx].normalize()
		}
	}
	return normalized
}

func (e DeliveryProjectionEvent) normalize() DeliveryProjectionEvent {
	normalized := e
	normalized.Type = strings.TrimSpace(normalized.Type)
	normalized.TurnID = strings.TrimSpace(normalized.TurnID)
	normalized.Error = strings.TrimSpace(normalized.Error)
	normalized.Fingerprint = strings.TrimSpace(normalized.Fingerprint)
	return normalized
}

func cloneDeliveryEvent(event DeliveryEvent) DeliveryEvent {
	cloned := event.normalize()
	cloned.Metadata = cloneRawJSON(cloned.Metadata)
	return cloned
}

func cloneDeliverySnapshot(snapshot DeliverySnapshot) DeliverySnapshot {
	cloned := snapshot.normalize()
	return cloned
}

func cloneDeliveryRequest(req DeliveryRequest) DeliveryRequest {
	cloned := DeliveryRequest{
		Event: cloneDeliveryEvent(req.Event),
	}
	if req.Snapshot != nil {
		snapshot := cloneDeliverySnapshot(*req.Snapshot)
		cloned.Snapshot = &snapshot
	}
	return cloned
}

func deliveryMetadataJSON(payload any) json.RawMessage {
	if payload == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return json.RawMessage(data)
}

func cloneRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}
