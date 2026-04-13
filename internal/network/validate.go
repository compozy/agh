package network

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
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
	// ErrExpired reports an envelope that is already expired.
	ErrExpired = errors.New("network: expired")
	// ErrReplayTooOld reports an envelope outside the receiver replay window.
	ErrReplayTooOld = errors.New("network: replay window exceeded")
)

var (
	channelPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	peerIDPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
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

	normalized := Envelope{
		Protocol:      strings.TrimSpace(env.Protocol),
		ID:            strings.TrimSpace(env.ID),
		Kind:          Kind(strings.TrimSpace(string(env.Kind))),
		Channel:       strings.TrimSpace(env.Channel),
		From:          strings.TrimSpace(env.From),
		TS:            env.TS,
		Body:          cloneRawMessage(env.Body),
		Proof:         cloneProof(env.Proof),
		Ext:           cloneExtensionMap(env.Ext),
		InteractionID: normalizeOptionalIdentifier(env.InteractionID),
		ReplyTo:       normalizeOptionalIdentifier(env.ReplyTo),
		TraceID:       normalizeOptionalIdentifier(env.TraceID),
		CausationID:   normalizeOptionalIdentifier(env.CausationID),
		To:            normalizeOptionalIdentifier(env.To),
		ExpiresAt:     cloneInt64Ptr(env.ExpiresAt),
	}

	if normalized.Protocol == "" {
		return Envelope{}, fmt.Errorf("%w: protocol is required", ErrMissingField)
	}
	if normalized.Protocol != ProtocolV0 {
		return Envelope{}, fmt.Errorf("%w: protocol=%q", ErrInvalidField, normalized.Protocol)
	}
	if normalized.ID == "" {
		return Envelope{}, fmt.Errorf("%w: id is required", ErrMissingField)
	}
	if err := normalized.Kind.Validate(); err != nil {
		return Envelope{}, err
	}
	if normalized.Channel == "" {
		return Envelope{}, fmt.Errorf("%w: channel is required", ErrMissingField)
	}
	if err := ValidateChannel(normalized.Channel); err != nil {
		return Envelope{}, err
	}
	if normalized.From == "" {
		return Envelope{}, fmt.Errorf("%w: from is required", ErrMissingField)
	}
	if err := ValidatePeerID(normalized.From); err != nil {
		return Envelope{}, fmt.Errorf("%w: from", err)
	}
	if normalized.To != nil {
		if err := ValidatePeerID(*normalized.To); err != nil {
			return Envelope{}, fmt.Errorf("%w: to", err)
		}
	}
	if normalized.InteractionID != nil && *normalized.InteractionID == "" {
		return Envelope{}, fmt.Errorf("%w: interaction_id", ErrInvalidField)
	}
	if normalized.ReplyTo != nil && *normalized.ReplyTo == "" {
		return Envelope{}, fmt.Errorf("%w: reply_to", ErrInvalidField)
	}
	if normalized.TraceID != nil && *normalized.TraceID == "" {
		return Envelope{}, fmt.Errorf("%w: trace_id", ErrInvalidField)
	}
	if normalized.CausationID != nil && *normalized.CausationID == "" {
		return Envelope{}, fmt.Errorf("%w: causation_id", ErrInvalidField)
	}
	if normalized.TS <= 0 {
		return Envelope{}, fmt.Errorf("%w: ts is required", ErrMissingField)
	}
	if _, err := validateJSONObject("body", normalized.Body); err != nil {
		return Envelope{}, err
	}
	if _, err := normalized.DecodeBody(); err != nil {
		return Envelope{}, err
	}
	if err := validateKindEnvelopeRules(normalized); err != nil {
		return Envelope{}, err
	}
	if err := validateEnvelopeFreshness(normalized, opts); err != nil {
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
	if !channelPattern.MatchString(channel) {
		return fmt.Errorf("%w: channel=%q", ErrInvalidField, channel)
	}
	return nil
}

// ValidatePeerID reports whether the peer identifier matches the RFC grammar.
func ValidatePeerID(peerID string) error {
	if !peerIDPattern.MatchString(peerID) {
		return fmt.Errorf("%w: peer_id=%q", ErrInvalidField, peerID)
	}
	return nil
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

	switch kind {
	case KindGreet:
		var body GreetBody
		if err := decodeJSON(raw, &body); err != nil {
			return nil, fmt.Errorf("%w: greet body: %w", ErrInvalidBody, err)
		}
		if err := normalizeAndValidateGreetBody(&body); err != nil {
			return nil, err
		}
		return body, nil
	case KindWhois:
		var body WhoisBody
		if err := decodeJSON(raw, &body); err != nil {
			return nil, fmt.Errorf("%w: whois body: %w", ErrInvalidBody, err)
		}
		if err := normalizeAndValidateWhoisBody(&body); err != nil {
			return nil, err
		}
		return body, nil
	case KindSay:
		var body SayBody
		if err := decodeJSON(raw, &body); err != nil {
			return nil, fmt.Errorf("%w: say body: %w", ErrInvalidBody, err)
		}
		if err := normalizeAndValidateSayBody(&body); err != nil {
			return nil, err
		}
		return body, nil
	case KindDirect:
		var body DirectBody
		if err := decodeJSON(raw, &body); err != nil {
			return nil, fmt.Errorf("%w: direct body: %w", ErrInvalidBody, err)
		}
		if err := normalizeAndValidateDirectBody(&body); err != nil {
			return nil, err
		}
		return body, nil
	case KindRecipe:
		object, err := validateJSONObject("body", raw)
		if err != nil {
			return nil, err
		}
		if _, ok := object["recipe"]; !ok {
			return nil, fmt.Errorf("%w: recipe body must wrap artifact fields inside \"recipe\", e.g. {\"recipe\":{...}}", ErrInvalidBody)
		}
		var body RecipeBody
		if err := decodeJSON(raw, &body); err != nil {
			return nil, fmt.Errorf("%w: recipe body: %w", ErrInvalidBody, err)
		}
		if err := normalizeAndValidateRecipeBody(&body); err != nil {
			return nil, err
		}
		return body, nil
	case KindReceipt:
		var body ReceiptBody
		if err := decodeJSON(raw, &body); err != nil {
			return nil, fmt.Errorf("%w: receipt body: %w", ErrInvalidBody, err)
		}
		if err := normalizeAndValidateReceiptBody(&body); err != nil {
			return nil, err
		}
		return body, nil
	case KindTrace:
		var body TraceBody
		if err := decodeJSON(raw, &body); err != nil {
			return nil, fmt.Errorf("%w: trace body: %w", ErrInvalidBody, err)
		}
		if err := normalizeAndValidateTraceBody(&body); err != nil {
			return nil, err
		}
		return body, nil
	default:
		return nil, fmt.Errorf("%w: kind=%q", ErrInvalidKind, string(kind))
	}
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
		}
	case DirectBody:
		if env.To == nil {
			return fmt.Errorf("%w: direct to is required", ErrMissingField)
		}
		if env.InteractionID == nil {
			return fmt.Errorf("%w: direct interaction_id is required", ErrMissingField)
		}
	case ReceiptBody:
		if env.InteractionID == nil {
			return fmt.Errorf("%w: receipt interaction_id is required", ErrMissingField)
		}
	case TraceBody:
		if env.InteractionID == nil {
			return fmt.Errorf("%w: trace interaction_id is required", ErrMissingField)
		}
	}

	return nil
}

func validateEnvelopeFreshness(env Envelope, opts ValidateOptions) error {
	nowUnix := opts.Now.Unix()

	if env.ExpiresAt != nil {
		if *env.ExpiresAt <= nowUnix {
			return fmt.Errorf("%w: expires_at=%d", ErrExpired, *env.ExpiresAt)
		}
		return nil
	}

	maxAge := int64(opts.MaxReplayAge / time.Second)
	if maxAge > 0 && nowUnix-env.TS > maxAge {
		return fmt.Errorf("%w: ts=%d max_replay_age=%s", ErrReplayTooOld, env.TS, opts.MaxReplayAge)
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

func normalizeAndValidateDirectBody(body *DirectBody) error {
	if strings.TrimSpace(body.Text) == "" {
		return fmt.Errorf("%w: direct text is required", ErrInvalidBody)
	}
	body.Intent = strings.TrimSpace(body.Intent)
	return nil
}

func normalizeAndValidateRecipeBody(body *RecipeBody) error {
	body.Recipe.RecipeID = strings.TrimSpace(body.Recipe.RecipeID)
	body.Recipe.Version = strings.TrimSpace(body.Recipe.Version)
	body.Recipe.ContentType = strings.TrimSpace(body.Recipe.ContentType)
	body.Recipe.Digest = strings.TrimSpace(body.Recipe.Digest)
	body.Recipe.URI = strings.TrimSpace(body.Recipe.URI)
	body.Recipe.Inputs = normalizeStringList(body.Recipe.Inputs)
	body.Recipe.Outputs = normalizeStringList(body.Recipe.Outputs)
	body.Recipe.Requirements = normalizeStringList(body.Recipe.Requirements)

	if body.Recipe.RecipeID == "" {
		return fmt.Errorf("%w: recipe.recipe_id is required", ErrInvalidBody)
	}
	if body.Recipe.Version == "" {
		return fmt.Errorf("%w: recipe.version is required", ErrInvalidBody)
	}
	if body.Recipe.ContentType == "" {
		return fmt.Errorf("%w: recipe.content_type is required", ErrInvalidBody)
	}
	if body.Recipe.Digest == "" {
		return fmt.Errorf("%w: recipe.digest is required", ErrInvalidBody)
	}
	if strings.TrimSpace(body.Recipe.URI) == "" && strings.TrimSpace(body.Recipe.Inline) == "" {
		return fmt.Errorf("%w: recipe requires uri or inline", ErrInvalidBody)
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
	body.State = InteractionState(strings.TrimSpace(string(body.State)))
	if err := body.State.Validate(); err != nil {
		return err
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
