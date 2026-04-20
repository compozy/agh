// Package network defines the AGH Network v0 protocol surface shared by the
// transport, router, and delivery layers.
package network

import (
	"encoding/json"
	"fmt"
)

// ProtocolV0 is the RFC v0 wire protocol identifier.
const ProtocolV0 = "agh-network/v0"

// Kind identifies one normative AGH Network message kind.
type Kind string

const (
	KindGreet      Kind = "greet"
	KindWhois      Kind = "whois"
	KindSay        Kind = "say"
	KindDirect     Kind = "direct"
	KindCapability Kind = "capability"
	KindReceipt    Kind = "receipt"
	KindTrace      Kind = "trace"
)

var validKinds = map[Kind]struct{}{
	KindGreet:      {},
	KindWhois:      {},
	KindSay:        {},
	KindDirect:     {},
	KindCapability: {},
	KindReceipt:    {},
	KindTrace:      {},
}

// Validate reports whether the kind is one of the documented RFC values.
func (k Kind) Validate() error {
	if _, ok := validKinds[k]; !ok {
		return fmt.Errorf("%w: kind=%q", ErrInvalidKind, string(k))
	}
	return nil
}

// ReceiptStatus identifies one receipt admission status.
type ReceiptStatus string

const (
	ReceiptStatusAccepted    ReceiptStatus = "accepted"
	ReceiptStatusRejected    ReceiptStatus = "rejected"
	ReceiptStatusDuplicate   ReceiptStatus = "duplicate"
	ReceiptStatusExpired     ReceiptStatus = "expired"
	ReceiptStatusUnsupported ReceiptStatus = "unsupported"
	ReceiptStatusCanceled    ReceiptStatus = "canceled"
)

var validReceiptStatuses = map[ReceiptStatus]struct{}{
	ReceiptStatusAccepted:    {},
	ReceiptStatusRejected:    {},
	ReceiptStatusDuplicate:   {},
	ReceiptStatusExpired:     {},
	ReceiptStatusUnsupported: {},
	ReceiptStatusCanceled:    {},
}

// Validate reports whether the receipt status is documented by the RFC.
func (s ReceiptStatus) Validate() error {
	if _, ok := validReceiptStatuses[s]; !ok {
		return fmt.Errorf("%w: receipt status=%q", ErrInvalidField, string(s))
	}
	return nil
}

// WhoisType identifies the request or response shape for `whois`.
type WhoisType string

const (
	WhoisTypeRequest  WhoisType = "request"
	WhoisTypeResponse WhoisType = "response"
)

var validWhoisTypes = map[WhoisType]struct{}{
	WhoisTypeRequest:  {},
	WhoisTypeResponse: {},
}

// Validate reports whether the whois type is a documented value.
func (t WhoisType) Validate() error {
	if _, ok := validWhoisTypes[t]; !ok {
		return fmt.Errorf("%w: whois type=%q", ErrInvalidField, string(t))
	}
	return nil
}

// ReasonCode identifies one registered protocol rejection reason.
type ReasonCode string

const (
	ReasonCodeMalformed          ReasonCode = "malformed"
	ReasonCodeExpired            ReasonCode = "expired"
	ReasonCodeDuplicate          ReasonCode = "duplicate"
	ReasonCodeUnsupportedKind    ReasonCode = "unsupported_kind"
	ReasonCodeUnsupportedProfile ReasonCode = "unsupported_profile"
	ReasonCodeVerificationFailed ReasonCode = "verification_failed"
	ReasonCodeNotTarget          ReasonCode = "not_target"
	ReasonCodeNotFound           ReasonCode = "not_found"
	ReasonCodeBusy               ReasonCode = "busy"
	ReasonCodeInternal           ReasonCode = "internal"
	ReasonCodeInteractionClosed  ReasonCode = "interaction_closed"
)

var validReasonCodes = map[ReasonCode]struct{}{
	ReasonCodeMalformed:          {},
	ReasonCodeExpired:            {},
	ReasonCodeDuplicate:          {},
	ReasonCodeUnsupportedKind:    {},
	ReasonCodeUnsupportedProfile: {},
	ReasonCodeVerificationFailed: {},
	ReasonCodeNotTarget:          {},
	ReasonCodeNotFound:           {},
	ReasonCodeBusy:               {},
	ReasonCodeInternal:           {},
	ReasonCodeInteractionClosed:  {},
}

// Validate reports whether the reason code belongs to the v0 registry.
func (r ReasonCode) Validate() error {
	if _, ok := validReasonCodes[r]; !ok {
		return fmt.Errorf("%w: reason_code=%q", ErrInvalidField, string(r))
	}
	return nil
}

// InteractionState identifies one RFC interaction lifecycle state.
type InteractionState string

const (
	StateSubmitted  InteractionState = "submitted"
	StateWorking    InteractionState = "working"
	StateNeedsInput InteractionState = "needs_input"
	StateCompleted  InteractionState = "completed"
	StateFailed     InteractionState = "failed"
	StateCanceled   InteractionState = "canceled"
)

var validInteractionStates = map[InteractionState]struct{}{
	StateSubmitted:  {},
	StateWorking:    {},
	StateNeedsInput: {},
	StateCompleted:  {},
	StateFailed:     {},
	StateCanceled:   {},
}

// Validate reports whether the interaction state is documented by the RFC.
func (s InteractionState) Validate() error {
	if _, ok := validInteractionStates[s]; !ok {
		return fmt.Errorf("%w: interaction state=%q", ErrInvalidField, string(s))
	}
	return nil
}

// Proof preserves the opaque v0 proof payload for forward compatibility.
type Proof map[string]json.RawMessage

// ExtensionMap preserves opaque extension payloads without interpreting them.
type ExtensionMap map[string]json.RawMessage

// Envelope is the shared AGH Network v0 wire envelope.
type Envelope struct {
	Protocol      string          `json:"protocol"`
	ID            string          `json:"id"`
	Kind          Kind            `json:"kind"`
	Channel       string          `json:"channel"`
	From          string          `json:"from"`
	To            *string         `json:"to"`
	InteractionID *string         `json:"interaction_id,omitempty"`
	ReplyTo       *string         `json:"reply_to,omitempty"`
	TraceID       *string         `json:"trace_id,omitempty"`
	CausationID   *string         `json:"causation_id,omitempty"`
	TS            int64           `json:"ts"`
	ExpiresAt     *int64          `json:"expires_at,omitempty"`
	Body          json.RawMessage `json:"body"`
	Proof         *Proof          `json:"proof"`
	Ext           ExtensionMap    `json:"ext,omitempty"`
}

// IsDirected reports whether the envelope targets a specific peer.
func (e Envelope) IsDirected() bool {
	return e.To != nil
}

// IsBroadcast reports whether the envelope is channel-broadcast.
func (e Envelope) IsBroadcast() bool {
	return e.To == nil
}

// DecodeBody parses and validates the envelope body using the envelope kind.
func (e Envelope) DecodeBody() (Body, error) {
	return DecodeBody(e.Kind, e.Body)
}

// Body is the typed representation of one envelope body.
type Body interface {
	Kind() Kind
}

var (
	_ Body = GreetBody{}
	_ Body = WhoisBody{}
	_ Body = SayBody{}
	_ Body = DirectBody{}
	_ Body = CapabilityBody{}
	_ Body = ReceiptBody{}
	_ Body = TraceBody{}
)

// GreetBody advertises peer presence and capabilities in a channel.
type GreetBody struct {
	PeerCard PeerCard `json:"peer_card"`
	Summary  string   `json:"summary,omitempty"`
}

// Kind returns the wire kind for the body.
func (GreetBody) Kind() Kind { return KindGreet }

// PeerCard advertises one peer's identity and capabilities.
type PeerCard struct {
	PeerID              string       `json:"peer_id"`
	DisplayName         *string      `json:"display_name,omitempty"`
	ProfilesSupported   []string     `json:"profiles_supported"`
	Capabilities        []string     `json:"capabilities"`
	ArtifactsSupported  []string     `json:"artifacts_supported"`
	TrustModesSupported []string     `json:"trust_modes_supported"`
	Ext                 ExtensionMap `json:"ext,omitempty"`
}

// WhoisBody requests or returns peer card information.
type WhoisBody struct {
	Type     WhoisType `json:"type"`
	Query    string    `json:"query,omitempty"`
	PeerCard *PeerCard `json:"peer_card,omitempty"`
}

// Kind returns the wire kind for the body.
func (WhoisBody) Kind() Kind { return KindWhois }

// SayBody carries broadcast chat-first communication.
type SayBody struct {
	Text      string            `json:"text"`
	Artifacts []json.RawMessage `json:"artifacts,omitempty"`
	Intent    string            `json:"intent,omitempty"`
}

// Kind returns the wire kind for the body.
func (SayBody) Kind() Kind { return KindSay }

// DirectBody carries targeted interaction content.
type DirectBody struct {
	Text      string            `json:"text"`
	Intent    string            `json:"intent,omitempty"`
	Artifacts []json.RawMessage `json:"artifacts,omitempty"`
}

// Kind returns the wire kind for the body.
func (DirectBody) Kind() Kind { return KindDirect }

// CapabilityBody carries or advertises one transferable capability artifact.
type CapabilityBody struct {
	Capability CapabilityEnvelopePayload `json:"capability"`
}

// Kind returns the wire kind for the body.
func (CapabilityBody) Kind() Kind { return KindCapability }

// CapabilityEnvelopePayload is the transferable unified capability document.
type CapabilityEnvelopePayload struct {
	ID                string   `json:"id"`
	Summary           string   `json:"summary"`
	Outcome           string   `json:"outcome"`
	Version           string   `json:"version,omitempty"`
	Digest            string   `json:"digest"`
	ContextNeeded     []string `json:"context_needed,omitempty"`
	ArtifactsExpected []string `json:"artifacts_expected,omitempty"`
	ExecutionOutline  []string `json:"execution_outline,omitempty"`
	Constraints       []string `json:"constraints,omitempty"`
	Examples          []string `json:"examples,omitempty"`
	Requirements      []string `json:"requirements,omitempty"`
}

// ReceiptBody acknowledges or rejects protocol-level admission.
type ReceiptBody struct {
	ForID      string        `json:"for_id"`
	Status     ReceiptStatus `json:"status"`
	ReasonCode *ReasonCode   `json:"reason_code,omitempty"`
	Detail     *string       `json:"detail,omitempty"`
}

// Kind returns the wire kind for the body.
func (ReceiptBody) Kind() Kind { return KindReceipt }

// TraceBody reports progress or terminal outcome for an interaction.
type TraceBody struct {
	State        InteractionState  `json:"state"`
	Message      string            `json:"message,omitempty"`
	Result       json.RawMessage   `json:"result,omitempty"`
	ArtifactRefs []json.RawMessage `json:"artifact_refs,omitempty"`
}

// Kind returns the wire kind for the body.
func (TraceBody) Kind() Kind { return KindTrace }
