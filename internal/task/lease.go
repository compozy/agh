package task

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	// DefaultRunLeaseDuration is the conservative lease duration used when a caller omits one.
	DefaultRunLeaseDuration = 5 * time.Minute
	// MaxRunLeaseDuration is the MVP guardrail for a single task-run lease extension.
	MaxRunLeaseDuration = 24 * time.Hour

	claimTokenRandomBytes = 32
	claimTokenHashPrefix  = "sha256:"
)

var defaultCoordinationMessageKinds = []string{
	"status",
	"request",
	"reply",
	"blocker",
	"handoff",
	"result",
	"review_request",
}

var rawClaimTokenPattern = regexp.MustCompile(`agh_claim_[A-Za-z0-9_-]+`)

// ClaimCriteria captures the atomic next-work filters for one claiming session.
type ClaimCriteria struct {
	Scope                 Scope                `json:"scope,omitempty"`
	WorkspaceID           string               `json:"workspace_id,omitempty"`
	ClaimerSessionID      string               `json:"claimer_session_id"`
	ClaimedBy             *ActorIdentity       `json:"claimed_by,omitempty"`
	AgentName             string               `json:"agent_name,omitempty"`
	RequiredCapabilities  []string             `json:"required_capabilities,omitempty"`
	PriorityMin           int                  `json:"priority_min,omitempty"`
	CoordinationChannelID string               `json:"coordination_channel_id,omitempty"`
	Soul                  *SoulClaimProvenance `json:"soul,omitempty"`
	LeaseDuration         time.Duration        `json:"lease_duration"`
	Now                   time.Time            `json:"now"`
}

// SoulClaimProvenance captures pre-resolved session Soul data at claim time.
type SoulClaimProvenance struct {
	SnapshotID string    `json:"snapshot_id,omitempty"`
	Digest     string    `json:"digest,omitempty"`
	AgentName  string    `json:"agent_name,omitempty"`
	CapturedAt time.Time `json:"captured_at"`
}

// CoordinationChannelMetadata is the safe channel display metadata returned with a claim.
type CoordinationChannelMetadata struct {
	ID                  string    `json:"id"`
	Channel             string    `json:"channel,omitempty"`
	DisplayName         string    `json:"display_name"`
	Purpose             string    `json:"purpose,omitempty"`
	WorkspaceID         string    `json:"workspace_id,omitempty"`
	TaskID              string    `json:"task_id,omitempty"`
	RunID               string    `json:"run_id,omitempty"`
	WorkflowID          string    `json:"workflow_id,omitempty"`
	AllowedMessageKinds []string  `json:"allowed_message_kinds,omitempty"`
	LastActivityAt      time.Time `json:"last_activity_at"`
}

// ClaimResult is the successful synchronous claim result. ClaimToken is raw and must not cross public surfaces.
type ClaimResult struct {
	Task                Task                         `json:"task"`
	Run                 Run                          `json:"run"`
	ClaimToken          string                       `json:"claim_token"`
	LeaseUntil          time.Time                    `json:"lease_until"`
	CoordinationChannel *CoordinationChannelMetadata `json:"coordination_channel,omitempty"`
}

// LeaseHeartbeat captures a token-fenced lease extension request.
type LeaseHeartbeat struct {
	RunID         string        `json:"run_id"`
	ClaimToken    string        `json:"claim_token"`
	LeaseDuration time.Duration `json:"lease_duration"`
	Now           time.Time     `json:"now"`
}

// LeaseRelease captures a token-fenced release request.
type LeaseRelease struct {
	RunID      string    `json:"run_id"`
	ClaimToken string    `json:"claim_token"`
	Reason     string    `json:"reason,omitempty"`
	Now        time.Time `json:"now"`
}

// LeaseCompletion captures a token-fenced successful terminal transition.
type LeaseCompletion struct {
	RunID      string    `json:"run_id"`
	ClaimToken string    `json:"claim_token"`
	Result     RunResult `json:"result"`
	Now        time.Time `json:"now"`
}

// LeaseFailure captures a token-fenced failed terminal transition.
type LeaseFailure struct {
	RunID      string     `json:"run_id"`
	ClaimToken string     `json:"claim_token"`
	Failure    RunFailure `json:"failure"`
	Now        time.Time  `json:"now"`
}

// ExpiredLeaseRecovery captures deterministic recovery of stale task-run leases.
type ExpiredLeaseRecovery struct {
	Now    time.Time `json:"now"`
	Reason string    `json:"reason,omitempty"`
	Limit  int       `json:"limit,omitempty"`
}

// ExpiredLeaseRecoveryResult records one recovered lease and its previous owner state.
type ExpiredLeaseRecoveryResult struct {
	Run                    Run       `json:"run"`
	PreviousRunStatus      RunStatus `json:"previous_run_status"`
	PreviousSessionID      string    `json:"previous_session_id,omitempty"`
	PreviousLeaseUntil     time.Time `json:"previous_lease_until"`
	PreviousClaimTokenHash string    `json:"previous_claim_token_hash,omitempty"`
	Reason                 string    `json:"reason,omitempty"`
}

// SessionLeaseRelease captures a daemon-owned structural release for all active
// leases bound to one runtime session.
type SessionLeaseRelease struct {
	SessionID string    `json:"session_id"`
	Reason    string    `json:"reason,omitempty"`
	Now       time.Time `json:"now"`
}

// SessionLeaseReleaseResult records one structurally released session lease.
type SessionLeaseReleaseResult struct {
	Run                    Run       `json:"run"`
	PreviousRunStatus      RunStatus `json:"previous_run_status"`
	PreviousSessionID      string    `json:"previous_session_id,omitempty"`
	PreviousLeaseUntil     time.Time `json:"previous_lease_until"`
	PreviousClaimTokenHash string    `json:"previous_claim_token_hash,omitempty"`
	Reason                 string    `json:"reason,omitempty"`
}

// NewClaimToken generates one raw bearer token for a successful claim response.
func NewClaimToken() (string, error) {
	random := make([]byte, claimTokenRandomBytes)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("task: generate claim token: %w", err)
	}
	return "agh_claim_" + base64.RawURLEncoding.EncodeToString(random), nil
}

// RedactClaimTokens replaces raw claim bearer tokens in free-form strings.
func RedactClaimTokens(value string) string {
	if value == "" {
		return ""
	}
	return rawClaimTokenPattern.ReplaceAllString(value, "agh_claim_[REDACTED]")
}

// ClaimTokenHash returns the canonical hash persisted for one raw claim token.
func ClaimTokenHash(rawToken string) (string, error) {
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return "", fmt.Errorf("%w: claim_token is required", ErrValidation)
	}
	sum := sha256.Sum256([]byte(token))
	return claimTokenHashPrefix + hex.EncodeToString(sum[:]), nil
}

// VerifyClaimToken reports whether rawToken hashes to the persisted canonical hash.
func VerifyClaimToken(rawToken string, persistedHash string) bool {
	token := strings.TrimSpace(rawToken)
	hash := canonicalClaimTokenHash(persistedHash)
	if token == "" || hash == "" {
		return false
	}
	sum := sha256.Sum256([]byte(token))
	computed := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) == 1
}

// Normalize returns a validated claim criteria with default scope, time, and lease duration applied.
func (c ClaimCriteria) Normalize(defaultNow time.Time) (ClaimCriteria, error) {
	normalized := c
	normalized.Scope = normalized.Scope.Normalize()
	normalized.WorkspaceID = strings.TrimSpace(normalized.WorkspaceID)
	if normalized.Scope == "" {
		if normalized.WorkspaceID != "" {
			normalized.Scope = ScopeWorkspace
		} else {
			normalized.Scope = ScopeGlobal
		}
	}
	normalized.ClaimerSessionID = strings.TrimSpace(normalized.ClaimerSessionID)
	if normalized.ClaimedBy != nil {
		claimedBy := *normalized.ClaimedBy
		claimedBy.Kind = claimedBy.Kind.Normalize()
		claimedBy.Ref = strings.TrimSpace(claimedBy.Ref)
		normalized.ClaimedBy = &claimedBy
	}
	if normalized.ClaimedBy == nil && normalized.ClaimerSessionID != "" {
		normalized.ClaimedBy = &ActorIdentity{Kind: ActorKindAgentSession, Ref: normalized.ClaimerSessionID}
	}
	normalized.AgentName = strings.TrimSpace(normalized.AgentName)
	normalized.CoordinationChannelID = strings.TrimSpace(normalized.CoordinationChannelID)
	normalized.RequiredCapabilities = normalizeCapabilityCriteria(normalized.RequiredCapabilities)
	if normalized.LeaseDuration == 0 {
		normalized.LeaseDuration = DefaultRunLeaseDuration
	}
	if normalized.Now.IsZero() {
		normalized.Now = defaultNow.UTC()
	} else {
		normalized.Now = normalized.Now.UTC()
	}
	if normalized.Soul != nil {
		soulProvenance := *normalized.Soul
		soulProvenance.SnapshotID = strings.TrimSpace(soulProvenance.SnapshotID)
		soulProvenance.Digest = strings.TrimSpace(soulProvenance.Digest)
		soulProvenance.AgentName = strings.TrimSpace(soulProvenance.AgentName)
		if soulProvenance.CapturedAt.IsZero() {
			soulProvenance.CapturedAt = normalized.Now
		} else {
			soulProvenance.CapturedAt = soulProvenance.CapturedAt.UTC()
		}
		normalized.Soul = &soulProvenance
	}
	if err := normalized.Validate("claim_criteria"); err != nil {
		return ClaimCriteria{}, err
	}
	return normalized, nil
}

// Validate reports whether the claim criteria is safe to execute transactionally.
func (c ClaimCriteria) Validate(path string) error {
	if err := ValidateScopeBinding(c.Scope, c.WorkspaceID, path, "workspace_id"); err != nil {
		return err
	}
	if strings.TrimSpace(c.ClaimerSessionID) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "claimer_session_id"))
	}
	if c.ClaimedBy != nil {
		if err := c.ClaimedBy.Validate(nestedPath(path, "claimed_by")); err != nil {
			return err
		}
	}
	if err := ValidateCapabilityIDs(c.RequiredCapabilities, nestedPath(path, "required_capabilities")); err != nil {
		return err
	}
	if c.PriorityMin < 0 {
		return fmt.Errorf(
			"%w: %s must be zero or positive: %d",
			ErrValidation,
			nestedPath(path, "priority_min"),
			c.PriorityMin,
		)
	}
	if err := validateLeaseDuration(c.LeaseDuration, nestedPath(path, "lease_duration")); err != nil {
		return err
	}
	if c.Soul != nil {
		if err := c.Soul.Validate(nestedPath(path, "soul")); err != nil {
			return err
		}
	}
	if c.Now.IsZero() {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "now"))
	}
	return nil
}

// Validate reports whether pre-resolved Soul claim provenance is internally consistent.
func (p SoulClaimProvenance) Validate(path string) error {
	hasSnapshotID := strings.TrimSpace(p.SnapshotID) != ""
	hasDigest := strings.TrimSpace(p.Digest) != ""
	if hasSnapshotID && !hasDigest {
		return fmt.Errorf("%w: %s.digest is required when snapshot_id is set", ErrValidation, path)
	}
	if hasDigest && strings.TrimSpace(p.AgentName) == "" {
		return fmt.Errorf("%w: %s.agent_name is required when digest is set", ErrValidation, path)
	}
	if !p.CapturedAt.IsZero() && p.CapturedAt.Location() != time.UTC {
		return fmt.Errorf("%w: %s.captured_at must be UTC", ErrValidation, path)
	}
	return nil
}

// Normalize returns a validated heartbeat request with default time and lease duration applied.
func (h LeaseHeartbeat) Normalize(defaultNow time.Time) (LeaseHeartbeat, error) {
	normalized := h
	normalized.RunID = strings.TrimSpace(normalized.RunID)
	normalized.ClaimToken = strings.TrimSpace(normalized.ClaimToken)
	if normalized.LeaseDuration == 0 {
		normalized.LeaseDuration = DefaultRunLeaseDuration
	}
	normalized.Now = normalizeLeaseNow(normalized.Now, defaultNow)
	if err := normalized.Validate("lease_heartbeat"); err != nil {
		return LeaseHeartbeat{}, err
	}
	return normalized, nil
}

// Validate reports whether the heartbeat request is internally consistent.
func (h LeaseHeartbeat) Validate(path string) error {
	if err := validateLeaseRunToken(h.RunID, h.ClaimToken, path); err != nil {
		return err
	}
	if err := validateLeaseDuration(h.LeaseDuration, nestedPath(path, "lease_duration")); err != nil {
		return err
	}
	if h.Now.IsZero() {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "now"))
	}
	return nil
}

// Normalize returns a validated release request with default time applied.
func (r LeaseRelease) Normalize(defaultNow time.Time) (LeaseRelease, error) {
	normalized := r
	normalized.RunID = strings.TrimSpace(normalized.RunID)
	normalized.ClaimToken = strings.TrimSpace(normalized.ClaimToken)
	normalized.Reason = strings.TrimSpace(normalized.Reason)
	normalized.Now = normalizeLeaseNow(normalized.Now, defaultNow)
	if err := normalized.Validate("lease_release"); err != nil {
		return LeaseRelease{}, err
	}
	return normalized, nil
}

// Validate reports whether the release request is internally consistent.
func (r LeaseRelease) Validate(path string) error {
	return validateLeaseRunToken(r.RunID, r.ClaimToken, path)
}

// Normalize returns a validated structural session lease release request.
func (r SessionLeaseRelease) Normalize(defaultNow time.Time) (SessionLeaseRelease, error) {
	normalized := r
	normalized.SessionID = strings.TrimSpace(normalized.SessionID)
	normalized.Reason = strings.TrimSpace(normalized.Reason)
	normalized.Now = normalizeLeaseNow(normalized.Now, defaultNow)
	if err := normalized.Validate("session_lease_release"); err != nil {
		return SessionLeaseRelease{}, err
	}
	return normalized, nil
}

// Validate reports whether the structural session lease release is internally consistent.
func (r SessionLeaseRelease) Validate(path string) error {
	if strings.TrimSpace(r.SessionID) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "session_id"))
	}
	if r.Now.IsZero() {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "now"))
	}
	return nil
}

// Normalize returns a validated completion request with default time applied.
func (c LeaseCompletion) Normalize(defaultNow time.Time) (LeaseCompletion, error) {
	normalized := c
	normalized.RunID = strings.TrimSpace(normalized.RunID)
	normalized.ClaimToken = strings.TrimSpace(normalized.ClaimToken)
	normalized.Now = normalizeLeaseNow(normalized.Now, defaultNow)
	if err := normalized.Validate("lease_completion"); err != nil {
		return LeaseCompletion{}, err
	}
	return normalized, nil
}

// Validate reports whether the completion request is internally consistent.
func (c LeaseCompletion) Validate(path string) error {
	if err := validateLeaseRunToken(c.RunID, c.ClaimToken, path); err != nil {
		return err
	}
	return c.Result.Validate(nestedPath(path, "result"))
}

// Normalize returns a validated failure request with default time applied.
func (f LeaseFailure) Normalize(defaultNow time.Time) (LeaseFailure, error) {
	normalized := f
	normalized.RunID = strings.TrimSpace(normalized.RunID)
	normalized.ClaimToken = strings.TrimSpace(normalized.ClaimToken)
	normalized.Now = normalizeLeaseNow(normalized.Now, defaultNow)
	if err := normalized.Validate("lease_failure"); err != nil {
		return LeaseFailure{}, err
	}
	return normalized, nil
}

// Validate reports whether the failure request is internally consistent.
func (f LeaseFailure) Validate(path string) error {
	if err := validateLeaseRunToken(f.RunID, f.ClaimToken, path); err != nil {
		return err
	}
	return f.Failure.Validate(nestedPath(path, "failure"))
}

// Normalize returns a validated expired-lease recovery request.
func (r ExpiredLeaseRecovery) Normalize(defaultNow time.Time) (ExpiredLeaseRecovery, error) {
	normalized := r
	normalized.Now = normalizeLeaseNow(normalized.Now, defaultNow)
	normalized.Reason = strings.TrimSpace(normalized.Reason)
	if err := normalized.Validate("expired_lease_recovery"); err != nil {
		return ExpiredLeaseRecovery{}, err
	}
	return normalized, nil
}

// Validate reports whether the expired-lease recovery request is internally consistent.
func (r ExpiredLeaseRecovery) Validate(path string) error {
	if r.Now.IsZero() {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "now"))
	}
	if r.Limit < 0 {
		return fmt.Errorf("%w: %s must be zero or positive: %d", ErrValidation, nestedPath(path, "limit"), r.Limit)
	}
	return nil
}

func normalizeCapabilityCriteria(values []string) []string {
	return normalizeStringSet(values)
}

func normalizeStringSet(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			normalized = append(normalized, trimmed)
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func validateLeaseRunToken(runID string, claimToken string, path string) error {
	if strings.TrimSpace(runID) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "run_id"))
	}
	if strings.TrimSpace(claimToken) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "claim_token"))
	}
	return nil
}

func validateLeaseDuration(duration time.Duration, path string) error {
	if duration <= 0 {
		return fmt.Errorf("%w: %s must be positive", ErrValidation, path)
	}
	if duration > MaxRunLeaseDuration {
		return fmt.Errorf("%w: %s must be <= %s", ErrValidation, path, MaxRunLeaseDuration)
	}
	return nil
}

func normalizeLeaseNow(value time.Time, defaultNow time.Time) time.Time {
	if value.IsZero() {
		return defaultNow.UTC()
	}
	return value.UTC()
}

func canonicalClaimTokenHash(value string) string {
	hash := strings.TrimSpace(value)
	hash = strings.TrimPrefix(hash, claimTokenHashPrefix)
	if !isCanonicalClaimTokenHash(hash) {
		return ""
	}
	return hash
}

func sanitizedCoordinationChannelMetadata(metadata *CoordinationChannelMetadata) *CoordinationChannelMetadata {
	if metadata == nil {
		return nil
	}
	cloned := *metadata
	cloned.ID = strings.TrimSpace(cloned.ID)
	cloned.Channel = strings.TrimSpace(cloned.Channel)
	cloned.DisplayName = strings.TrimSpace(cloned.DisplayName)
	cloned.Purpose = strings.TrimSpace(cloned.Purpose)
	cloned.WorkspaceID = strings.TrimSpace(cloned.WorkspaceID)
	cloned.TaskID = strings.TrimSpace(cloned.TaskID)
	cloned.RunID = strings.TrimSpace(cloned.RunID)
	cloned.WorkflowID = strings.TrimSpace(cloned.WorkflowID)
	cloned.AllowedMessageKinds = normalizeStringSet(cloned.AllowedMessageKinds)
	if cloned.DisplayName == "" {
		if cloned.Channel != "" {
			cloned.DisplayName = cloned.Channel
		} else {
			cloned.DisplayName = cloned.ID
		}
	}
	if cloned.AllowedMessageKinds == nil {
		cloned.AllowedMessageKinds = append([]string(nil), defaultCoordinationMessageKinds...)
	}
	return &cloned
}

func claimResultWithoutRawTokenInMetadata(result *ClaimResult) {
	if result == nil {
		return
	}
	result.CoordinationChannel = sanitizedCoordinationChannelMetadata(result.CoordinationChannel)
	result.Task.Metadata = removeRawClaimTokenFields(result.Task.Metadata)
	result.Run.Metadata = removeRawClaimTokenFields(result.Run.Metadata)
	result.Run.Result = removeRawClaimTokenFields(result.Run.Result)
}

func removeRawClaimTokenFields(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || !hasRawClaimTokenField(raw) {
		return normalizeRawJSON(raw)
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}
	cleaned := removeRawClaimTokenFieldValue(decoded)
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return nil
	}
	return normalizeRawJSON(encoded)
}

func removeRawClaimTokenFieldValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cleaned := make(map[string]any, len(typed))
		for key, nested := range typed {
			if strings.EqualFold(strings.TrimSpace(key), "claim_token") {
				continue
			}
			cleaned[key] = removeRawClaimTokenFieldValue(nested)
		}
		return cleaned
	case []any:
		for idx, nested := range typed {
			typed[idx] = removeRawClaimTokenFieldValue(nested)
		}
		return typed
	default:
		return value
	}
}
