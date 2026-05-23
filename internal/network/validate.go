package network

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/diagnostics"
	"github.com/compozy/agh/internal/network/rules"
	taskpkg "github.com/compozy/agh/internal/task"
)

const (
	validateDirectIDKey = "direct_id"
	validateThreadIDKey = "thread_id"
)

var (
	// ErrInvalidEnvelope reports a structurally invalid envelope.
	ErrInvalidEnvelope = errors.New("network: invalid envelope")
	// ErrMissingField reports a required protocol field is absent.
	ErrMissingField = errors.New("network: missing field")
	// ErrInvalidField reports a present field violates protocol rules.
	ErrInvalidField = errors.New("network: invalid field")
	// ErrInvalidKind reports an unknown or unsupported message kind.
	ErrInvalidKind = errors.New("network: invalid kind")
	// ErrInvalidBody reports a malformed or invalid kind-specific body.
	ErrInvalidBody = errors.New("network: invalid body")
	// ErrEnvelopeTooLarge reports an envelope exceeding the protocol size limit.
	ErrEnvelopeTooLarge = errors.New("network: envelope too large")
	// ErrExpired reports an envelope that is already expired.
	ErrExpired = errors.New("network: expired")
	// ErrReplayTooOld reports an envelope outside the receiver replay window.
	ErrReplayTooOld = errors.New("network: replay window exceeded")
	// ErrVerificationFailed reports a syntactically valid envelope whose
	// integrity checks failed.
	ErrVerificationFailed = errors.New("network: verification failed")
	// ErrLegacyFieldRejected reports an obsolete hard-cut wire field.
	ErrLegacyFieldRejected = errors.New("network: legacy field rejected")
	// ErrDirectRoomCollision reports that a direct_id is bound to a different
	// channel peer pair than its deterministic identity permits.
	ErrDirectRoomCollision = errors.New("network: direct room collision")
)

var (
	peerIDPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
	threadIDPattern = regexp.MustCompile(`^thread_[a-z0-9][a-z0-9_-]{2,95}$`)
	directIDPattern = regexp.MustCompile(`^direct_[a-f0-9]{32}$`)
)

// DefaultMaxReplayAge is the RFC-recommended maximum receiver replay age when
// `expires_at` is not present.
const DefaultMaxReplayAge = 5 * time.Minute

// ValidateOptions configures envelope validation and normalization.
type ValidateOptions struct {
	Now          time.Time
	MaxReplayAge time.Duration
}

// ParseEnvelope decodes, validates, and normalizes one raw envelope.
func ParseEnvelope(data []byte, opts ValidateOptions) (Envelope, error) {
	if err := rejectLegacyEnvelopeFields(data); err != nil {
		return Envelope{}, err
	}
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return Envelope{}, fmt.Errorf("%w: decode envelope: %w", ErrInvalidEnvelope, err)
	}
	return NormalizeEnvelope(env, opts)
}

// NormalizeEnvelope trims identifier fields, validates the envelope, and
// returns a safe cloned copy for downstream use.
func NormalizeEnvelope(env Envelope, opts ValidateOptions) (Envelope, error) {
	opts = opts.withDefaults()

	normalized := normalizeEnvelopeCopy(env)
	if err := validateEnvelopeHeader(normalized); err != nil {
		return Envelope{}, err
	}
	if err := validateEnvelopeParticipants(normalized); err != nil {
		return Envelope{}, err
	}
	if err := validateEnvelopeReferences(normalized); err != nil {
		return Envelope{}, err
	}
	if err := validateEnvelopeBodyAndFreshness(normalized, opts); err != nil {
		return Envelope{}, err
	}
	return normalized, nil
}

// ValidateEnvelope validates one envelope without returning a normalized copy.
func ValidateEnvelope(env Envelope, opts ValidateOptions) error {
	_, err := NormalizeEnvelope(env, opts)
	return err
}

// ValidateChannel reports whether the channel matches the RFC grammar.
func ValidateChannel(channel string) error {
	if !rules.ValidChannel(channel) {
		return fmt.Errorf("%w: channel=%q", ErrInvalidField, channel)
	}
	return nil
}

// ValidateWorkspaceID reports whether the workspace identity can safely occupy
// one NATS subject token and one protocol envelope field.
func ValidateWorkspaceID(workspaceID string) error {
	trimmed := strings.TrimSpace(workspaceID)
	if trimmed == "" {
		return fmt.Errorf("%w: workspace_id is required", ErrMissingField)
	}
	if strings.ContainsAny(trimmed, ". *>") || containsControlCharacter(trimmed) {
		return fmt.Errorf("%w: workspace_id=%q", ErrInvalidField, workspaceID)
	}
	return nil
}

// ValidateSurface reports whether the surface matches the RFC conversation values.
func ValidateSurface(surface Surface) error {
	return Surface(strings.TrimSpace(string(surface))).Validate()
}

// ValidatePeerID reports whether the peer identifier matches the RFC grammar.
func ValidatePeerID(peerID string) error {
	if !peerIDPattern.MatchString(peerID) {
		return fmt.Errorf("%w: peer_id=%q", ErrInvalidField, peerID)
	}
	return nil
}

// ValidateConversationID reports whether a container identifier matches its field grammar.
func ValidateConversationID(id string, field string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return fmt.Errorf("%w: %s is required", ErrMissingField, field)
	}
	switch field {
	case validateThreadIDKey:
		if !threadIDPattern.MatchString(trimmed) {
			return fmt.Errorf("%w: thread_id=%q", ErrInvalidField, id)
		}
	case validateDirectIDKey:
		if !directIDPattern.MatchString(trimmed) {
			return fmt.Errorf("%w: direct_id=%q", ErrInvalidField, id)
		}
	default:
		if strings.ContainsAny(trimmed, `/\`) || containsControlCharacter(trimmed) || len(trimmed) > 128 {
			return fmt.Errorf("%w: %s=%q", ErrInvalidField, field, id)
		}
	}
	return nil
}

// ValidateWorkID reports whether a work id can safely cross the network boundary.
func ValidateWorkID(id string) error {
	return ValidateConversationID(id, "work_id")
}

// ValidateWorkState reports whether the state is a known work lifecycle state.
func ValidateWorkState(state WorkState) error {
	return WorkState(strings.TrimSpace(string(state))).Validate()
}

// ValidateWorkTransition reports whether a trace may advance work from one state to another.
func ValidateWorkTransition(from WorkState, to WorkState) error {
	current := WorkState(strings.TrimSpace(string(from)))
	next := WorkState(strings.TrimSpace(string(to)))
	if err := ValidateWorkState(current); err != nil {
		return err
	}
	if err := ValidateWorkState(next); err != nil {
		return err
	}
	if !canApplyTrace(current, next) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidStateTransition, current, next)
	}
	return nil
}

// NormalizeDirectRoomPeers validates, rejects same-peer rooms, and returns
// peers in their stable direct-room storage order.
func NormalizeDirectRoomPeers(localPeer string, remotePeer string) (string, string, error) {
	peerA := strings.TrimSpace(localPeer)
	peerB := strings.TrimSpace(remotePeer)
	if err := ValidatePeerID(peerA); err != nil {
		return "", "", err
	}
	if err := ValidatePeerID(peerB); err != nil {
		return "", "", err
	}
	if peerA == peerB {
		return "", "", fmt.Errorf("%w: direct room peers must differ", ErrInvalidField)
	}
	if peerB < peerA {
		peerA, peerB = peerB, peerA
	}
	return peerA, peerB, nil
}

// ValidateDirectRoomPeers reports whether two peer IDs form a valid two-party direct room.
func ValidateDirectRoomPeers(peerA string, peerB string) error {
	_, _, err := NormalizeDirectRoomPeers(peerA, peerB)
	return err
}

// DirectRoomIdentity derives the stable two-party direct room identity scoped to one workspace channel.
func DirectRoomIdentity(
	workspaceID string,
	channel string,
	localPeer string,
	remotePeer string,
) (string, string, string, error) {
	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	if err := ValidateWorkspaceID(trimmedWorkspaceID); err != nil {
		return "", "", "", err
	}
	trimmedChannel := strings.TrimSpace(channel)
	if err := ValidateChannel(trimmedChannel); err != nil {
		return "", "", "", err
	}
	peerA, peerB, err := NormalizeDirectRoomPeers(localPeer, remotePeer)
	if err != nil {
		return "", "", "", err
	}

	sum := sha256.Sum256([]byte(
		"agh-network/direct-room/v0\x00" + trimmedWorkspaceID + "\x00" + trimmedChannel + "\x00" + peerA + "\x00" + peerB,
	))
	return "direct_" + hex.EncodeToString(sum[:])[:32], peerA, peerB, nil
}

// ValidateDirectRoomBinding proves that an existing direct room row matches
// the deterministic identity for its workspace/channel-scoped peer pair.
func ValidateDirectRoomBinding(workspaceID string, channel string, directID string, peerA string, peerB string) error {
	trimmedDirectID := strings.TrimSpace(directID)
	if err := ValidateConversationID(trimmedDirectID, validateDirectIDKey); err != nil {
		return err
	}
	expectedID, _, _, err := DirectRoomIdentity(workspaceID, channel, peerA, peerB)
	if err != nil {
		return err
	}
	if trimmedDirectID != expectedID {
		return fmt.Errorf(
			"%w: direct_id=%q expected=%q",
			ErrDirectRoomCollision,
			trimmedDirectID,
			expectedID,
		)
	}
	return nil
}

// ValidateConversationRef reports whether a conversation reference identifies exactly one container.
func ValidateConversationRef(ref ConversationRef) error {
	return ref.Validate()
}

// ValidateEnvelopeConversation enforces kind-specific container and work fields.
func ValidateEnvelopeConversation(env Envelope) error {
	if env.WorkID != nil {
		if err := ValidateWorkID(*env.WorkID); err != nil {
			return err
		}
	}
	if isDiscoveryKind(env.Kind) {
		return validateDiscoveryEnvelopeOmitsConversation(env)
	}
	if !isConversationKind(env.Kind) {
		return nil
	}
	if _, err := ConversationRefFromEnvelope(env); err != nil {
		return err
	}
	switch env.Kind {
	case KindCapability:
		if env.WorkID == nil {
			return fmt.Errorf("%w: capability work_id is required", ErrMissingField)
		}
	case KindReceipt:
		if env.WorkID == nil {
			return fmt.Errorf("%w: receipt work_id is required", ErrMissingField)
		}
	case KindTrace:
		if env.WorkID == nil {
			return fmt.Errorf("%w: trace work_id is required", ErrMissingField)
		}
	}
	return nil
}

// ConversationRefFromEnvelope returns the validated container reference for a conversation envelope.
func ConversationRefFromEnvelope(env Envelope) (ConversationRef, error) {
	if env.Surface == nil {
		if env.ThreadID != nil {
			return ConversationRef{}, fmt.Errorf("%w: thread_id requires surface", ErrInvalidField)
		}
		if env.DirectID != nil {
			return ConversationRef{}, fmt.Errorf("%w: direct_id requires surface", ErrInvalidField)
		}
		return ConversationRef{}, fmt.Errorf("%w: surface is required", ErrMissingField)
	}

	ref := ConversationRef{
		WorkspaceID: env.WorkspaceID,
		Channel:     env.Channel,
		Surface:     *env.Surface,
	}
	if env.ThreadID != nil {
		ref.ThreadID = *env.ThreadID
	}
	if env.DirectID != nil {
		ref.DirectID = *env.DirectID
	}
	if err := ref.Validate(); err != nil {
		return ConversationRef{}, err
	}
	return ref, nil
}

func validateDiscoveryEnvelopeOmitsConversation(env Envelope) error {
	if env.Surface != nil {
		return fmt.Errorf("%w: %s must not include surface", ErrInvalidField, env.Kind)
	}
	if env.ThreadID != nil {
		return fmt.Errorf("%w: %s must not include thread_id", ErrInvalidField, env.Kind)
	}
	if env.DirectID != nil {
		return fmt.Errorf("%w: %s must not include direct_id", ErrInvalidField, env.Kind)
	}
	if env.WorkID != nil {
		return fmt.Errorf("%w: %s must not include work_id", ErrInvalidField, env.Kind)
	}
	return nil
}

func isDiscoveryKind(kind Kind) bool {
	return kind == KindGreet || kind == KindWhois
}

func isConversationKind(kind Kind) bool {
	switch kind {
	case KindSay, KindCapability, KindReceipt, KindTrace:
		return true
	default:
		return false
	}
}

// RouteToken derives the deterministic NATS route token for one peer.
func RouteToken(peerID string) (string, error) {
	peerID = strings.TrimSpace(peerID)
	if err := ValidatePeerID(peerID); err != nil {
		return "", err
	}

	sum := sha256.Sum256([]byte(peerID))
	return hex.EncodeToString(sum[:16]), nil
}

// DecodeBody parses and validates one kind-specific envelope body.
func DecodeBody(kind Kind, raw json.RawMessage) (Body, error) {
	if _, err := validateJSONObject("body", raw); err != nil {
		return nil, err
	}

	decoder, err := bodyDecoderForKind(kind)
	if err != nil {
		return nil, err
	}
	return decoder(raw)
}

type bodyDecoder func(json.RawMessage) (Body, error)

func normalizeEnvelopeCopy(env Envelope) Envelope {
	return Envelope{
		Protocol:    strings.TrimSpace(env.Protocol),
		ID:          strings.TrimSpace(env.ID),
		WorkspaceID: strings.TrimSpace(env.WorkspaceID),
		Kind:        Kind(strings.TrimSpace(string(env.Kind))),
		Channel:     strings.TrimSpace(env.Channel),
		Surface:     normalizeOptionalSurface(env.Surface),
		ThreadID:    normalizeOptionalIdentifier(env.ThreadID),
		DirectID:    normalizeOptionalIdentifier(env.DirectID),
		From:        strings.TrimSpace(env.From),
		TS:          env.TS,
		Body:        cloneRawMessage(env.Body),
		Proof:       cloneProof(env.Proof),
		Ext:         cloneExtensionMap(env.Ext),
		WorkID:      normalizeOptionalIdentifier(env.WorkID),
		ReplyTo:     normalizeOptionalIdentifier(env.ReplyTo),
		TraceID:     normalizeOptionalIdentifier(env.TraceID),
		CausationID: normalizeOptionalIdentifier(env.CausationID),
		To:          normalizeOptionalIdentifier(env.To),
		ExpiresAt:   cloneInt64Ptr(env.ExpiresAt),
	}
}

func validateEnvelopeHeader(env Envelope) error {
	if env.Protocol == "" {
		return fmt.Errorf("%w: protocol is required", ErrMissingField)
	}
	if env.Protocol != ProtocolV0 {
		return fmt.Errorf("%w: protocol=%q", ErrInvalidField, env.Protocol)
	}
	if env.ID == "" {
		return fmt.Errorf("%w: id is required", ErrMissingField)
	}
	if err := ValidateWorkspaceID(env.WorkspaceID); err != nil {
		return err
	}
	if err := env.Kind.Validate(); err != nil {
		return err
	}
	if env.Channel == "" {
		return fmt.Errorf("%w: channel is required", ErrMissingField)
	}
	return ValidateChannel(env.Channel)
}

func validateEnvelopeParticipants(env Envelope) error {
	if env.From == "" {
		return fmt.Errorf("%w: from is required", ErrMissingField)
	}
	if err := ValidatePeerID(env.From); err != nil {
		return fmt.Errorf("%w: from", err)
	}
	if env.To != nil {
		if err := ValidatePeerID(*env.To); err != nil {
			return fmt.Errorf("%w: to", err)
		}
	}
	return nil
}

func validateEnvelopeReferences(env Envelope) error {
	if err := validateOptionalIdentifierField(env.ReplyTo, "reply_to"); err != nil {
		return err
	}
	if err := validateOptionalIdentifierField(env.TraceID, "trace_id"); err != nil {
		return err
	}
	if err := validateOptionalIdentifierField(env.CausationID, "causation_id"); err != nil {
		return err
	}
	if env.TS <= 0 {
		return fmt.Errorf("%w: ts is required", ErrMissingField)
	}
	return ValidateEnvelopeConversation(env)
}

func validateOptionalIdentifierField(value *string, field string) error {
	if value != nil && *value == "" {
		return fmt.Errorf("%w: %s", ErrInvalidField, field)
	}
	return nil
}

func validateEnvelopeBodyAndFreshness(env Envelope, opts ValidateOptions) error {
	if _, err := validateJSONObject("body", env.Body); err != nil {
		return err
	}
	if _, err := env.DecodeBody(); err != nil {
		return err
	}
	if err := validateKindEnvelopeRules(env); err != nil {
		return err
	}
	if err := validateEnvelopeContainsNoRawSecrets(env); err != nil {
		return err
	}
	return validateEnvelopeFreshness(env, opts)
}

func bodyDecoderForKind(kind Kind) (bodyDecoder, error) {
	switch kind {
	case KindGreet:
		return func(raw json.RawMessage) (Body, error) {
			return decodeNormalizedBody(raw, "greet", normalizeAndValidateGreetBody)
		}, nil
	case KindWhois:
		return func(raw json.RawMessage) (Body, error) {
			return decodeNormalizedBody(raw, "whois", normalizeAndValidateWhoisBody)
		}, nil
	case KindSay:
		return func(raw json.RawMessage) (Body, error) {
			return decodeNormalizedBody(raw, "say", normalizeAndValidateSayBody)
		}, nil
	case KindCapability:
		return decodeCapabilityEnvelopeBody, nil
	case KindReceipt:
		return func(raw json.RawMessage) (Body, error) {
			return decodeNormalizedBody(raw, "receipt", normalizeAndValidateReceiptBody)
		}, nil
	case KindTrace:
		return func(raw json.RawMessage) (Body, error) {
			return decodeNormalizedBody(raw, "trace", normalizeAndValidateTraceBody)
		}, nil
	default:
		return nil, fmt.Errorf("%w: kind=%q", ErrInvalidKind, string(kind))
	}
}

func decodeCapabilityEnvelopeBody(raw json.RawMessage) (Body, error) {
	object, err := validateJSONObject("body", raw)
	if err != nil {
		return nil, err
	}
	if _, ok := object["capability"]; !ok {
		return nil, fmt.Errorf(
			"%w: capability body must wrap artifact fields inside \"capability\", e.g. {\"capability\":{...}}",
			ErrInvalidBody,
		)
	}
	return decodeNormalizedBody(raw, "capability", normalizeAndValidateCapabilityBody)
}

func decodeNormalizedBody[T Body](raw json.RawMessage, label string, normalize func(*T) error) (Body, error) {
	var body T
	if err := decodeJSON(raw, &body); err != nil {
		return nil, fmt.Errorf("%w: %s body: %w", ErrInvalidBody, label, err)
	}
	if err := normalize(&body); err != nil {
		return nil, err
	}
	return body, nil
}

func validateKindEnvelopeRules(env Envelope) error {
	body, err := env.DecodeBody()
	if err != nil {
		return err
	}

	switch typed := body.(type) {
	case GreetBody:
		if typed.PeerCard.PeerID != env.From {
			return fmt.Errorf("%w: greet peer_card.peer_id must match from", ErrInvalidBody)
		}
		if err := validatePeerCardPrivilegedCapabilities(typed.PeerCard, env.Proof); err != nil {
			return err
		}
	case WhoisBody:
		if typed.Type == WhoisTypeResponse {
			if env.ReplyTo == nil {
				return fmt.Errorf("%w: whois response reply_to is required", ErrMissingField)
			}
			if typed.PeerCard == nil {
				return fmt.Errorf("%w: whois response peer_card is required", ErrInvalidBody)
			}
			if typed.PeerCard.PeerID != env.From {
				return fmt.Errorf("%w: whois response peer_card.peer_id must match from", ErrInvalidBody)
			}
			if err := validatePeerCardPrivilegedCapabilities(*typed.PeerCard, env.Proof); err != nil {
				return err
			}
		}
	case ReceiptBody:
		if env.WorkID == nil {
			return fmt.Errorf("%w: receipt work_id is required", ErrMissingField)
		}
	case TraceBody:
		if env.WorkID == nil {
			return fmt.Errorf("%w: trace work_id is required", ErrMissingField)
		}
	}

	return nil
}

func validateEnvelopeFreshness(env Envelope, opts ValidateOptions) error {
	nowUnix := opts.Now.Unix()
	maxAge := int64(opts.MaxReplayAge / time.Second)
	if maxAge > 0 && env.TS-nowUnix > maxAge {
		return fmt.Errorf("%w: ts=%d max_replay_age=%s", ErrReplayTooOld, env.TS, opts.MaxReplayAge)
	}

	if env.ExpiresAt != nil {
		if *env.ExpiresAt <= nowUnix {
			return fmt.Errorf("%w: expires_at=%d", ErrExpired, *env.ExpiresAt)
		}
		return nil
	}

	if maxAge > 0 && nowUnix-env.TS > maxAge {
		return fmt.Errorf("%w: ts=%d max_replay_age=%s", ErrReplayTooOld, env.TS, opts.MaxReplayAge)
	}

	return nil
}

func validatePeerCardPrivilegedCapabilities(card PeerCard, proof *Proof) error {
	if !containsString(card.Capabilities, networkTaskWriteCapability) {
		return nil
	}
	if proof == nil || len(*proof) == 0 {
		return fmt.Errorf(
			"%w: peer_card capability %q requires proof",
			ErrVerificationFailed,
			networkTaskWriteCapability,
		)
	}
	return nil
}

func validateEnvelopeContainsNoRawSecrets(env Envelope) error {
	if envelopeRawValueContainsSecret(env.Body) {
		return fmt.Errorf("%w: raw secret material is not allowed in network body", ErrInvalidBody)
	}
	if env.Proof != nil {
		for _, raw := range *env.Proof {
			if envelopeRawValueContainsSecret(raw) {
				return fmt.Errorf("%w: raw secret material is not allowed in network proof", ErrInvalidBody)
			}
		}
	}
	for _, raw := range env.Ext {
		if envelopeRawValueContainsSecret(raw) {
			return fmt.Errorf("%w: raw secret material is not allowed in network ext", ErrInvalidBody)
		}
	}
	return nil
}

func envelopeRawValueContainsSecret(raw json.RawMessage) bool {
	if len(bytes.TrimSpace(raw)) == 0 {
		return false
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return envelopeStringContainsSecret(string(raw))
	}
	return envelopeValueContainsSecret("", value)
}

func envelopeValueContainsSecret(key string, value any) bool {
	if envelopeStringContainsSecret(key) || (envelopeKeyCarriesRawSecret(key) && envelopeValueIsNonEmpty(value)) {
		return true
	}

	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return envelopeStringContainsSecret(typed)
	case []any:
		for _, item := range typed {
			if envelopeValueContainsSecret("", item) {
				return true
			}
		}
		return false
	case map[string]any:
		for nestedKey, nestedValue := range typed {
			if envelopeValueContainsSecret(nestedKey, nestedValue) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func envelopeStringContainsSecret(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	return taskpkg.RedactClaimTokens(value) != value || diagnostics.Redact(value) != value
}

func envelopeValueIsNonEmpty(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return true
	}
}

func envelopeKeyCarriesRawSecret(key string) bool {
	normalized := strings.NewReplacer("_", "", "-", "", ".", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	if normalized == "" || strings.Contains(normalized, "hash") {
		return false
	}
	switch normalized {
	case "apikey",
		"accesstoken",
		"refreshtoken",
		"mcpauthtoken",
		"oauthcode",
		"authorizationcode",
		"codeverifier",
		"pkceverifier",
		"secretbinding",
		"clientsecret",
		"authorization",
		"password",
		"secret",
		"token",
		"claimtoken":
		return true
	default:
		return false
	}
}

func rejectLegacyEnvelopeFields(data []byte) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return nil
	}
	if _, ok := object["interaction_id"]; ok {
		return fmt.Errorf("%w: interaction_id", ErrLegacyFieldRejected)
	}
	return nil
}

func normalizeAndValidateGreetBody(body *GreetBody) error {
	if err := normalizeAndValidatePeerCard(&body.PeerCard); err != nil {
		return err
	}
	return nil
}

func normalizeAndValidatePeerCard(card *PeerCard) error {
	card.PeerID = strings.TrimSpace(card.PeerID)
	if card.PeerID == "" {
		return fmt.Errorf("%w: peer_card.peer_id is required", ErrInvalidBody)
	}
	if err := ValidatePeerID(card.PeerID); err != nil {
		return fmt.Errorf("%w: peer_card.peer_id", err)
	}

	card.DisplayName = normalizeOptionalText(card.DisplayName)
	card.ProfilesSupported = normalizeStringList(card.ProfilesSupported)
	card.Capabilities = normalizeStringList(card.Capabilities)
	card.ArtifactsSupported = normalizeStringList(card.ArtifactsSupported)
	card.TrustModesSupported = normalizeStringList(card.TrustModesSupported)
	card.Ext = cloneExtensionMap(card.Ext)

	if card.ProfilesSupported == nil {
		return fmt.Errorf("%w: peer_card.profiles_supported is required", ErrInvalidBody)
	}
	if card.Capabilities == nil {
		return fmt.Errorf("%w: peer_card.capabilities is required", ErrInvalidBody)
	}
	if card.ArtifactsSupported == nil {
		return fmt.Errorf("%w: peer_card.artifacts_supported is required", ErrInvalidBody)
	}
	if card.TrustModesSupported == nil {
		return fmt.Errorf("%w: peer_card.trust_modes_supported is required", ErrInvalidBody)
	}

	return nil
}

func normalizeAndValidateWhoisBody(body *WhoisBody) error {
	body.Type = WhoisType(strings.TrimSpace(string(body.Type)))
	if err := body.Type.Validate(); err != nil {
		return err
	}

	if body.Type == WhoisTypeRequest {
		if body.PeerCard != nil {
			return fmt.Errorf("%w: whois request must not include peer_card", ErrInvalidBody)
		}
		return nil
	}

	if body.PeerCard == nil {
		return fmt.Errorf("%w: whois response peer_card is required", ErrInvalidBody)
	}
	return normalizeAndValidatePeerCard(body.PeerCard)
}

func normalizeAndValidateSayBody(body *SayBody) error {
	if strings.TrimSpace(body.Text) == "" {
		return fmt.Errorf("%w: say text is required", ErrInvalidBody)
	}
	body.Intent = strings.TrimSpace(body.Intent)
	return nil
}

func normalizeAndValidateCapabilityBody(body *CapabilityBody) error {
	body.Capability.ID = strings.TrimSpace(body.Capability.ID)
	body.Capability.Summary = strings.TrimSpace(body.Capability.Summary)
	body.Capability.Outcome = strings.TrimSpace(body.Capability.Outcome)
	body.Capability.Version = strings.TrimSpace(body.Capability.Version)
	body.Capability.Digest = strings.TrimSpace(body.Capability.Digest)
	body.Capability.ContextNeeded = normalizeStringList(body.Capability.ContextNeeded)
	body.Capability.ArtifactsExpected = normalizeStringList(body.Capability.ArtifactsExpected)
	body.Capability.ExecutionOutline = normalizeStringList(body.Capability.ExecutionOutline)
	body.Capability.Constraints = normalizeStringList(body.Capability.Constraints)
	body.Capability.Examples = normalizeStringList(body.Capability.Examples)
	body.Capability.Requirements = normalizeCapabilityRequirementList(body.Capability.Requirements)

	switch {
	case body.Capability.ID == "":
		return fmt.Errorf("%w: capability.id is required", ErrInvalidBody)
	case body.Capability.Summary == "":
		return fmt.Errorf("%w: capability.summary is required", ErrInvalidBody)
	case body.Capability.Outcome == "":
		return fmt.Errorf("%w: capability.outcome is required", ErrInvalidBody)
	case body.Capability.Digest == "":
		return fmt.Errorf("%w: capability.digest is required", ErrInvalidBody)
	}
	if err := validateCapabilityRequirements(body.Capability.Requirements, "capability.requirements"); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidBody, err)
	}

	expectedDigest, err := aghconfig.CanonicalCapabilityDigest(aghconfig.CapabilityDef{
		ID:                body.Capability.ID,
		Summary:           body.Capability.Summary,
		Outcome:           body.Capability.Outcome,
		Version:           body.Capability.Version,
		ContextNeeded:     slices.Clone(body.Capability.ContextNeeded),
		ArtifactsExpected: slices.Clone(body.Capability.ArtifactsExpected),
		ExecutionOutline:  slices.Clone(body.Capability.ExecutionOutline),
		Constraints:       slices.Clone(body.Capability.Constraints),
		Examples:          slices.Clone(body.Capability.Examples),
		Requirements:      slices.Clone(body.Capability.Requirements),
	})
	if err != nil {
		return fmt.Errorf("%w: compute capability digest: %w", ErrInvalidBody, err)
	}
	if body.Capability.Digest != expectedDigest {
		return fmt.Errorf(
			"%w: capability.digest=%q does not match canonical digest %q",
			ErrVerificationFailed,
			body.Capability.Digest,
			expectedDigest,
		)
	}

	return nil
}

func normalizeAndValidateReceiptBody(body *ReceiptBody) error {
	body.ForID = strings.TrimSpace(body.ForID)
	if body.ForID == "" {
		return fmt.Errorf("%w: receipt for_id is required", ErrInvalidBody)
	}

	body.Status = ReceiptStatus(strings.TrimSpace(string(body.Status)))
	if err := body.Status.Validate(); err != nil {
		return err
	}

	if body.ReasonCode != nil {
		normalized := ReasonCode(strings.TrimSpace(string(*body.ReasonCode)))
		body.ReasonCode = &normalized
		if err := body.ReasonCode.Validate(); err != nil {
			return err
		}
	}
	body.Detail = normalizeOptionalText(body.Detail)

	switch body.Status {
	case ReceiptStatusAccepted:
		if body.ReasonCode != nil {
			return fmt.Errorf("%w: accepted receipt must not include reason_code", ErrInvalidBody)
		}
	case ReceiptStatusRejected, ReceiptStatusDuplicate, ReceiptStatusExpired, ReceiptStatusUnsupported:
		if body.ReasonCode == nil {
			return fmt.Errorf("%w: receipt status %q requires reason_code", ErrInvalidBody, body.Status)
		}
	}

	return nil
}

func normalizeAndValidateTraceBody(body *TraceBody) error {
	body.State = WorkState(strings.TrimSpace(string(body.State)))
	if err := body.State.Validate(); err != nil {
		return err
	}
	return nil
}

func normalizeCapabilityRequirementList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, len(values))
	for idx, value := range values {
		normalized[idx] = strings.TrimSpace(value)
	}
	return normalized
}

func validateCapabilityRequirements(requirements []string, fieldPrefix string) error {
	if len(requirements) == 0 {
		return nil
	}

	seen := make(map[string]int, len(requirements))
	for idx, requirement := range requirements {
		if requirement == "" {
			return fmt.Errorf("%s[%d] is required", fieldPrefix, idx)
		}
		if priorIdx, ok := seen[requirement]; ok {
			return fmt.Errorf(
				"%s duplicate value %q after normalization at indexes %d and %d",
				fieldPrefix,
				requirement,
				priorIdx,
				idx,
			)
		}
		seen[requirement] = idx
	}

	return nil
}

func validateJSONObject(field string, raw json.RawMessage) (map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("%w: %s is required", ErrMissingField, field)
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &object); err != nil {
		return nil, fmt.Errorf("%w: %s must be a JSON object: %w", ErrInvalidField, field, err)
	}

	return object, nil
}

func decodeJSON(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func cloneExtensionMap(ext ExtensionMap) ExtensionMap {
	if ext == nil {
		return nil
	}

	cloned := make(ExtensionMap, len(ext))
	for key, value := range ext {
		cloned[key] = cloneRawMessage(value)
	}
	return cloned
}

func cloneProof(proof *Proof) *Proof {
	if proof == nil {
		return nil
	}

	cloned := make(Proof, len(*proof))
	for key, value := range *proof {
		cloned[key] = cloneRawMessage(value)
	}
	return &cloned
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneSurfacePtr(value *Surface) *Surface {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func normalizeOptionalSurface(value *Surface) *Surface {
	if value == nil {
		return nil
	}
	normalized := Surface(strings.TrimSpace(string(*value)))
	if normalized == "" {
		return nil
	}
	return &normalized
}

func normalizeOptionalIdentifier(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func containsControlCharacter(value string) bool {
	return strings.ContainsFunc(value, func(r rune) bool {
		return r < 0x20 || r == 0x7f
	})
}

func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}

	if strings.TrimSpace(*value) == "" {
		return nil
	}

	text := *value
	return &text
}

func normalizeStringList(values []string) []string {
	if values == nil {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func (opts ValidateOptions) withDefaults() ValidateOptions {
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if opts.MaxReplayAge <= 0 {
		opts.MaxReplayAge = DefaultMaxReplayAge
	}
	return opts
}
