package heartbeat

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const snapshotSchemaVersion = 1

var (
	// ErrSnapshotNotFound reports a missing persisted Heartbeat snapshot.
	ErrSnapshotNotFound = errors.New("heartbeat: snapshot not found")
	// ErrRevisionNotFound reports a missing persisted Heartbeat authoring revision.
	ErrRevisionNotFound = errors.New("heartbeat: revision not found")
	// ErrSessionHealthNotFound reports a missing persisted session health row.
	ErrSessionHealthNotFound = errors.New("heartbeat: session health not found")
	// ErrWakeStateNotFound reports a missing persisted Heartbeat wake state row.
	ErrWakeStateNotFound = errors.New("heartbeat: wake state not found")
	// ErrWakeEventNotFound reports a missing persisted Heartbeat wake event row.
	ErrWakeEventNotFound = errors.New("heartbeat: wake event not found")
	// ErrInvalidSnapshot reports a malformed persisted Heartbeat snapshot.
	ErrInvalidSnapshot = errors.New("heartbeat: invalid snapshot")
	// ErrInvalidRevision reports a malformed Heartbeat authoring revision.
	ErrInvalidRevision = errors.New("heartbeat: invalid revision")
	// ErrInvalidSessionHealth reports a malformed persisted session health row.
	ErrInvalidSessionHealth = errors.New("heartbeat: invalid session health")
	// ErrInvalidWakeState reports a malformed persisted Heartbeat wake state row.
	ErrInvalidWakeState = errors.New("heartbeat: invalid wake state")
	// ErrInvalidWakeEvent reports a malformed persisted Heartbeat wake event row.
	ErrInvalidWakeEvent = errors.New("heartbeat: invalid wake event")
)

// RevisionOperation describes a managed HEARTBEAT.md authoring mutation.
type RevisionOperation string

const (
	// RevisionOperationWrite records a create or update mutation.
	RevisionOperationWrite RevisionOperation = "write"
	// RevisionOperationDelete records a managed delete mutation.
	RevisionOperationDelete RevisionOperation = "delete"
	// RevisionOperationRollback records a managed rollback mutation.
	RevisionOperationRollback RevisionOperation = "rollback"
)

// ActorKind describes the redacted authoring actor class.
type ActorKind string

const (
	ActorKindUser      ActorKind = "user"
	ActorKindAgent     ActorKind = "agent"
	ActorKindExtension ActorKind = "extension"
	ActorKindSystem    ActorKind = "system"
)

// SessionHealthState describes daemon-owned session runtime state for wake eligibility.
type SessionHealthState string

const (
	SessionHealthStateIdle      SessionHealthState = "idle"
	SessionHealthStatePrompting SessionHealthState = "prompting"
	SessionHealthStateStopped   SessionHealthState = "stopped"
	SessionHealthStateDetached  SessionHealthState = "detached"
)

// SessionHealthStatus classifies metadata-only session health.
type SessionHealthStatus string

const (
	SessionHealthHealthy  SessionHealthStatus = "healthy"
	SessionHealthDegraded SessionHealthStatus = "degraded"
	SessionHealthStale    SessionHealthStatus = "stale"
	SessionHealthDead     SessionHealthStatus = "dead"
	SessionHealthUnknown  SessionHealthStatus = "unknown"
)

// SessionHealthIneligibilityReason is a closed reason for wake-ineligible health rows.
type SessionHealthIneligibilityReason string

const (
	SessionHealthReasonPromptActive  SessionHealthIneligibilityReason = "session_prompt_active"
	SessionHealthReasonNotAttachable SessionHealthIneligibilityReason = "session_not_attachable"
	SessionHealthReasonUnhealthy     SessionHealthIneligibilityReason = "session_unhealthy"
	SessionHealthReasonStale         SessionHealthIneligibilityReason = "session_health_stale"
	SessionHealthReasonHung          SessionHealthIneligibilityReason = "session_health_hung"
	SessionHealthReasonDead          SessionHealthIneligibilityReason = "session_health_dead"
	SessionHealthReasonUnknown       SessionHealthIneligibilityReason = "session_health_unknown"
)

// WakeSource classifies who requested an advisory Heartbeat wake.
type WakeSource string

const (
	WakeSourceScheduler      WakeSource = "scheduler"
	WakeSourceManual         WakeSource = "manual"
	WakeSourceHarnessReentry WakeSource = "harness_reentry"
)

// WakeResult classifies the outcome of one advisory Heartbeat wake attempt.
type WakeResult string

const (
	WakeResultSent        WakeResult = "sent"
	WakeResultSkipped     WakeResult = "skipped"
	WakeResultCoalesced   WakeResult = "coalesced"
	WakeResultRateLimited WakeResult = "rate_limited"
	WakeResultFailed      WakeResult = "failed"
)

// WakeReason is a closed deterministic reason for Heartbeat wake state and audit.
type WakeReason string

const (
	WakeReasonSent                  WakeReason = "wake_sent"
	WakeReasonHeartbeatDisabled     WakeReason = "heartbeat_disabled"
	WakeReasonHeartbeatInvalid      WakeReason = "heartbeat_invalid"
	WakeReasonHeartbeatNoPolicy     WakeReason = "heartbeat_no_policy"
	WakeReasonHeartbeatRateLimited  WakeReason = "heartbeat_rate_limited"
	WakeReasonHeartbeatNoEligible   WakeReason = "heartbeat_no_eligible_session"
	WakeReasonCooldownActive        WakeReason = "cooldown_active"
	WakeReasonQuietWindow           WakeReason = "quiet_window"
	WakeReasonSessionNotFound       WakeReason = "session_not_found"
	WakeReasonSessionUnhealthy      WakeReason = "session_unhealthy"
	WakeReasonSessionNotAttachable  WakeReason = "session_not_attachable"
	WakeReasonSessionPromptActive   WakeReason = "session_prompt_active"
	WakeReasonSessionPromptRace     WakeReason = "session_prompt_active_race"
	WakeReasonSyntheticPromptFailed WakeReason = "synthetic_prompt_failed"
	WakeReasonCoalesced             WakeReason = "wake_coalesced"
)

// Snapshot is the immutable storage row for a resolved HEARTBEAT.md policy.
type Snapshot struct {
	ID              string
	WorkspaceID     string
	AgentName       string
	SourcePath      string
	SchemaVersion   int
	Digest          string
	ConfigDigest    string
	Body            string
	FrontmatterJSON json.RawMessage
	ResolvedJSON    json.RawMessage
	DiagnosticsJSON json.RawMessage
	CreatedAt       time.Time
}

// SnapshotEnvelope is the structured JSON envelope stored in Snapshot.ResolvedJSON.
type SnapshotEnvelope struct {
	SchemaVersion    int                `json:"schema_version"`
	Present          bool               `json:"present"`
	Active           bool               `json:"active"`
	Valid            bool               `json:"valid"`
	Summary          string             `json:"summary,omitempty"`
	GuidanceMarkdown string             `json:"guidance_markdown,omitempty"`
	Preferences      Preferences        `json:"preferences"`
	ConfigProvenance ConfigProvenance   `json:"config_provenance"`
	Prompt           PromptContribution `json:"prompt"`
	Status           StatusData         `json:"status"`
	Diagnostics      []Diagnostic       `json:"diagnostics,omitempty"`
}

// SnapshotListQuery filters persisted Heartbeat snapshot rows.
type SnapshotListQuery struct {
	WorkspaceID string
	AgentName   string
	Digest      string
	Limit       int
}

// Revision is one append-only managed HEARTBEAT.md authoring history row.
type Revision struct {
	ID             string
	WorkspaceID    string
	AgentName      string
	SourcePath     string
	Operation      RevisionOperation
	PreviousDigest string
	NewDigest      string
	NewSnapshotID  string
	Body           string
	ActorKind      ActorKind
	ActorID        string
	CreatedAt      time.Time
}

// RevisionListQuery filters managed Heartbeat authoring revision history.
type RevisionListQuery struct {
	WorkspaceID string
	AgentName   string
	Operation   RevisionOperation
	Limit       int
}

// RollbackLookup selects the prior revision body used by managed rollback.
type RollbackLookup struct {
	WorkspaceID string
	AgentName   string
	RevisionID  string
}

// SessionHealth is the metadata-only runtime health row for one session.
type SessionHealth struct {
	SessionID           string
	WorkspaceID         string
	AgentName           string
	State               SessionHealthState
	Health              SessionHealthStatus
	ActivePrompt        bool
	Attachable          bool
	EligibleForWake     bool
	IneligibilityReason string
	LastActivityAt      time.Time
	LastPresenceAt      time.Time
	LastError           string
	UpdatedAt           time.Time
}

// SessionHealthListQuery filters persisted session health rows.
type SessionHealthListQuery struct {
	WorkspaceID     string
	AgentName       string
	SessionID       string
	State           SessionHealthState
	Health          SessionHealthStatus
	EligibleForWake *bool
	Limit           int
}

// WakeState is the per-session cooldown/coalescing summary for Heartbeat wakes.
type WakeState struct {
	WorkspaceID      string
	AgentName        string
	SessionID        string
	PolicySnapshotID string
	LastWakeAt       time.Time
	NextAllowedAt    time.Time
	CoalescedCount   int
	LastResult       WakeResult
	LastReason       WakeReason
	UpdatedAt        time.Time
}

// WakeStateListQuery filters Heartbeat wake state rows.
type WakeStateListQuery struct {
	WorkspaceID string
	AgentName   string
	SessionID   string
	Limit       int
}

// WakeEvent is one retained Heartbeat wake audit row.
type WakeEvent struct {
	ID                string
	WorkspaceID       string
	AgentName         string
	SessionID         string
	PolicySnapshotID  string
	Source            WakeSource
	Result            WakeResult
	Reason            WakeReason
	SyntheticPromptID string
	CreatedAt         time.Time
	ExpiresAt         time.Time
}

// WakeEventListQuery filters retained Heartbeat wake audit rows.
type WakeEventListQuery struct {
	WorkspaceID string
	AgentName   string
	SessionID   string
	Source      WakeSource
	Result      WakeResult
	Reason      WakeReason
	Limit       int
}

// SnapshotFromResolved creates a persistence row from a resolved Heartbeat policy.
func SnapshotFromResolved(
	id string,
	workspaceID string,
	agentName string,
	resolved *ResolvedPolicy,
	createdAt time.Time,
) (Snapshot, error) {
	if resolved == nil {
		return Snapshot{}, fmt.Errorf("%w: resolved policy is required", ErrInvalidSnapshot)
	}
	frontmatter, err := json.Marshal(resolved.Frontmatter)
	if err != nil {
		return Snapshot{}, fmt.Errorf("heartbeat: marshal snapshot frontmatter: %w", err)
	}
	diagnostics, err := DiagnosticsJSON(resolved.Diagnostics)
	if err != nil {
		return Snapshot{}, err
	}
	envelope := SnapshotEnvelope{
		SchemaVersion:    firstPositive(resolved.SchemaVersion, snapshotSchemaVersion),
		Present:          resolved.Present,
		Active:           resolved.Active,
		Valid:            resolved.Valid,
		Summary:          resolved.Summary,
		GuidanceMarkdown: resolved.GuidanceMarkdown,
		Preferences:      resolved.Preferences,
		ConfigProvenance: resolved.ConfigProvenance,
		Prompt:           resolved.Prompt,
		Status:           resolved.Status,
		Diagnostics:      cloneDiagnostics(resolved.Diagnostics),
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return Snapshot{}, fmt.Errorf("heartbeat: marshal snapshot envelope: %w", err)
	}
	snapshot := Snapshot{
		ID:              id,
		WorkspaceID:     workspaceID,
		AgentName:       agentName,
		SourcePath:      resolved.SourcePath,
		SchemaVersion:   envelope.SchemaVersion,
		Digest:          resolved.Digest,
		ConfigDigest:    firstNonEmpty(resolved.ConfigDigest, resolved.ConfigProvenance.Digest),
		Body:            resolved.GuidanceMarkdown,
		FrontmatterJSON: frontmatter,
		ResolvedJSON:    encoded,
		DiagnosticsJSON: diagnostics,
		CreatedAt:       createdAt,
	}
	if err := snapshot.Validate(); err != nil {
		return Snapshot{}, err
	}
	return snapshot.Normalize(), nil
}

// ResolvedEnvelope decodes the structured JSON envelope stored with the snapshot.
func (s Snapshot) ResolvedEnvelope() (SnapshotEnvelope, error) {
	normalized := s.Normalize()
	var envelope SnapshotEnvelope
	if err := json.Unmarshal(normalized.ResolvedJSON, &envelope); err != nil {
		return SnapshotEnvelope{}, fmt.Errorf("%w: decode resolved_json: %w", ErrInvalidSnapshot, err)
	}
	if envelope.SchemaVersion != snapshotSchemaVersion {
		return SnapshotEnvelope{}, fmt.Errorf(
			"%w: unsupported resolved schema version %d",
			ErrInvalidSnapshot,
			envelope.SchemaVersion,
		)
	}
	return envelope, nil
}

// DiagnosticsJSON encodes redacted validation diagnostics for snapshot and revision storage.
func DiagnosticsJSON(diagnostics []Diagnostic) (json.RawMessage, error) {
	encoded, err := json.Marshal(cloneDiagnostics(diagnostics))
	if err != nil {
		return nil, fmt.Errorf("heartbeat: marshal diagnostics: %w", err)
	}
	return encoded, nil
}

// Normalize trims metadata fields and applies JSON defaults.
func (s Snapshot) Normalize() Snapshot {
	s.ID = strings.TrimSpace(s.ID)
	s.WorkspaceID = strings.TrimSpace(s.WorkspaceID)
	s.AgentName = strings.TrimSpace(s.AgentName)
	s.SourcePath = strings.TrimSpace(s.SourcePath)
	s.Digest = strings.TrimSpace(s.Digest)
	s.ConfigDigest = strings.TrimSpace(s.ConfigDigest)
	if s.SchemaVersion == 0 {
		s.SchemaVersion = snapshotSchemaVersion
	}
	if len(s.FrontmatterJSON) == 0 {
		s.FrontmatterJSON = json.RawMessage(`{}`)
	}
	if len(s.ResolvedJSON) == 0 {
		s.ResolvedJSON = json.RawMessage(`{}`)
	}
	if len(s.DiagnosticsJSON) == 0 {
		s.DiagnosticsJSON = json.RawMessage(`[]`)
	}
	return s
}

// Validate ensures the snapshot can be stored as immutable provenance.
func (s Snapshot) Validate() error {
	normalized := s.Normalize()
	switch {
	case normalized.ID == "":
		return fmt.Errorf("%w: id is required", ErrInvalidSnapshot)
	case normalized.WorkspaceID == "":
		return fmt.Errorf("%w: workspace id is required", ErrInvalidSnapshot)
	case normalized.AgentName == "":
		return fmt.Errorf("%w: agent name is required", ErrInvalidSnapshot)
	case normalized.SourcePath == "":
		return fmt.Errorf("%w: source path is required", ErrInvalidSnapshot)
	case normalized.SchemaVersion <= 0:
		return fmt.Errorf("%w: schema version is required", ErrInvalidSnapshot)
	case normalized.Digest == "":
		return fmt.Errorf("%w: digest is required", ErrInvalidSnapshot)
	case normalized.ConfigDigest == "":
		return fmt.Errorf("%w: config digest is required", ErrInvalidSnapshot)
	case !json.Valid(normalized.FrontmatterJSON):
		return fmt.Errorf("%w: frontmatter_json must be valid JSON", ErrInvalidSnapshot)
	case !json.Valid(normalized.ResolvedJSON):
		return fmt.Errorf("%w: resolved_json must be valid JSON", ErrInvalidSnapshot)
	case !json.Valid(normalized.DiagnosticsJSON):
		return fmt.Errorf("%w: diagnostics_json must be valid JSON", ErrInvalidSnapshot)
	case normalized.CreatedAt.IsZero():
		return fmt.Errorf("%w: created_at is required", ErrInvalidSnapshot)
	}
	return nil
}

// Validate ensures the snapshot query uses sane bounds.
func (q SnapshotListQuery) Validate() error {
	if q.Limit < 0 {
		return fmt.Errorf("%w: invalid snapshot limit %d", ErrInvalidSnapshot, q.Limit)
	}
	return nil
}

// Normalize trims revision metadata fields.
func (r Revision) Normalize() Revision {
	r.ID = strings.TrimSpace(r.ID)
	r.WorkspaceID = strings.TrimSpace(r.WorkspaceID)
	r.AgentName = strings.TrimSpace(r.AgentName)
	r.SourcePath = strings.TrimSpace(r.SourcePath)
	r.Operation = RevisionOperation(strings.TrimSpace(string(r.Operation)))
	r.PreviousDigest = strings.TrimSpace(r.PreviousDigest)
	r.NewDigest = strings.TrimSpace(r.NewDigest)
	r.NewSnapshotID = strings.TrimSpace(r.NewSnapshotID)
	r.ActorKind = ActorKind(strings.TrimSpace(string(r.ActorKind)))
	r.ActorID = strings.TrimSpace(r.ActorID)
	return r
}

// Validate ensures the revision is append-only authoring history.
func (r Revision) Validate() error {
	normalized := r.Normalize()
	switch {
	case normalized.ID == "":
		return fmt.Errorf("%w: id is required", ErrInvalidRevision)
	case normalized.WorkspaceID == "":
		return fmt.Errorf("%w: workspace id is required", ErrInvalidRevision)
	case normalized.AgentName == "":
		return fmt.Errorf("%w: agent name is required", ErrInvalidRevision)
	case normalized.SourcePath == "":
		return fmt.Errorf("%w: source path is required", ErrInvalidRevision)
	case !ValidRevisionOperation(normalized.Operation):
		return fmt.Errorf("%w: invalid operation %q", ErrInvalidRevision, normalized.Operation)
	case normalized.Operation == RevisionOperationDelete && normalized.NewDigest != "":
		return fmt.Errorf("%w: delete revision must not set new digest", ErrInvalidRevision)
	case normalized.Operation != RevisionOperationDelete && normalized.NewDigest == "":
		return fmt.Errorf("%w: new digest is required", ErrInvalidRevision)
	case !ValidActorKind(normalized.ActorKind):
		return fmt.Errorf("%w: invalid actor kind %q", ErrInvalidRevision, normalized.ActorKind)
	case normalized.ActorID == "":
		return fmt.Errorf("%w: actor id is required", ErrInvalidRevision)
	case normalized.CreatedAt.IsZero():
		return fmt.Errorf("%w: created_at is required", ErrInvalidRevision)
	}
	return nil
}

// Validate ensures the revision query uses sane bounds and operation filters.
func (q RevisionListQuery) Validate() error {
	if q.Limit < 0 {
		return fmt.Errorf("%w: invalid revision limit %d", ErrInvalidRevision, q.Limit)
	}
	if q.Operation != "" && !ValidRevisionOperation(q.Operation) {
		return fmt.Errorf("%w: invalid operation %q", ErrInvalidRevision, q.Operation)
	}
	return nil
}

// Validate ensures rollback lookup identifiers are complete.
func (q RollbackLookup) Validate() error {
	switch {
	case strings.TrimSpace(q.WorkspaceID) == "":
		return fmt.Errorf("%w: workspace id is required", ErrInvalidRevision)
	case strings.TrimSpace(q.AgentName) == "":
		return fmt.Errorf("%w: agent name is required", ErrInvalidRevision)
	case strings.TrimSpace(q.RevisionID) == "":
		return fmt.Errorf("%w: revision id is required", ErrInvalidRevision)
	default:
		return nil
	}
}

// Normalize trims session health metadata fields.
func (h SessionHealth) Normalize() SessionHealth {
	h.SessionID = strings.TrimSpace(h.SessionID)
	h.WorkspaceID = strings.TrimSpace(h.WorkspaceID)
	h.AgentName = strings.TrimSpace(h.AgentName)
	h.State = SessionHealthState(strings.TrimSpace(string(h.State)))
	h.Health = SessionHealthStatus(strings.TrimSpace(string(h.Health)))
	h.IneligibilityReason = strings.TrimSpace(h.IneligibilityReason)
	h.LastError = strings.TrimSpace(h.LastError)
	return h
}

// Validate ensures session health remains metadata-only runtime state.
func (h SessionHealth) Validate() error {
	normalized := h.Normalize()
	switch {
	case normalized.SessionID == "":
		return fmt.Errorf("%w: session id is required", ErrInvalidSessionHealth)
	case normalized.WorkspaceID == "":
		return fmt.Errorf("%w: workspace id is required", ErrInvalidSessionHealth)
	case normalized.AgentName == "":
		return fmt.Errorf("%w: agent name is required", ErrInvalidSessionHealth)
	case !ValidSessionHealthState(normalized.State):
		return fmt.Errorf("%w: invalid state %q", ErrInvalidSessionHealth, normalized.State)
	case !ValidSessionHealthStatus(normalized.Health):
		return fmt.Errorf("%w: invalid health %q", ErrInvalidSessionHealth, normalized.Health)
	case normalized.IneligibilityReason != "" &&
		!ValidSessionHealthIneligibilityReason(normalized.IneligibilityReason):
		return fmt.Errorf(
			"%w: invalid ineligibility reason %q",
			ErrInvalidSessionHealth,
			normalized.IneligibilityReason,
		)
	case normalized.UpdatedAt.IsZero():
		return fmt.Errorf("%w: updated_at is required", ErrInvalidSessionHealth)
	}
	return nil
}

// Validate ensures the health query uses sane bounds and enum filters.
func (q SessionHealthListQuery) Validate() error {
	if q.Limit < 0 {
		return fmt.Errorf("%w: invalid session health limit %d", ErrInvalidSessionHealth, q.Limit)
	}
	if q.State != "" && !ValidSessionHealthState(q.State) {
		return fmt.Errorf("%w: invalid state %q", ErrInvalidSessionHealth, q.State)
	}
	if q.Health != "" && !ValidSessionHealthStatus(q.Health) {
		return fmt.Errorf("%w: invalid health %q", ErrInvalidSessionHealth, q.Health)
	}
	return nil
}

// Normalize trims wake state metadata fields.
func (s WakeState) Normalize() WakeState {
	s.WorkspaceID = strings.TrimSpace(s.WorkspaceID)
	s.AgentName = strings.TrimSpace(s.AgentName)
	s.SessionID = strings.TrimSpace(s.SessionID)
	s.PolicySnapshotID = strings.TrimSpace(s.PolicySnapshotID)
	s.LastResult = WakeResult(strings.TrimSpace(string(s.LastResult)))
	s.LastReason = WakeReason(strings.TrimSpace(string(s.LastReason)))
	return s
}

// Validate ensures wake state cannot become claimable work.
func (s WakeState) Validate() error {
	normalized := s.Normalize()
	switch {
	case normalized.WorkspaceID == "":
		return fmt.Errorf("%w: workspace id is required", ErrInvalidWakeState)
	case normalized.AgentName == "":
		return fmt.Errorf("%w: agent name is required", ErrInvalidWakeState)
	case normalized.SessionID == "":
		return fmt.Errorf("%w: session id is required", ErrInvalidWakeState)
	case normalized.CoalescedCount < 0:
		return fmt.Errorf("%w: coalesced count must be non-negative", ErrInvalidWakeState)
	case !ValidWakeResult(normalized.LastResult):
		return fmt.Errorf("%w: invalid result %q", ErrInvalidWakeState, normalized.LastResult)
	case normalized.LastReason != "" && !ValidWakeReason(normalized.LastReason):
		return fmt.Errorf("%w: invalid reason %q", ErrInvalidWakeState, normalized.LastReason)
	case normalized.UpdatedAt.IsZero():
		return fmt.Errorf("%w: updated_at is required", ErrInvalidWakeState)
	}
	return nil
}

// Validate ensures the wake state query uses sane bounds.
func (q WakeStateListQuery) Validate() error {
	if q.Limit < 0 {
		return fmt.Errorf("%w: invalid wake state limit %d", ErrInvalidWakeState, q.Limit)
	}
	return nil
}

// Normalize trims wake event metadata fields.
func (e WakeEvent) Normalize() WakeEvent {
	e.ID = strings.TrimSpace(e.ID)
	e.WorkspaceID = strings.TrimSpace(e.WorkspaceID)
	e.AgentName = strings.TrimSpace(e.AgentName)
	e.SessionID = strings.TrimSpace(e.SessionID)
	e.PolicySnapshotID = strings.TrimSpace(e.PolicySnapshotID)
	e.Source = WakeSource(strings.TrimSpace(string(e.Source)))
	e.Result = WakeResult(strings.TrimSpace(string(e.Result)))
	e.Reason = WakeReason(strings.TrimSpace(string(e.Reason)))
	e.SyntheticPromptID = strings.TrimSpace(e.SyntheticPromptID)
	return e
}

// Validate ensures the wake event is retained audit state, not claimable work.
func (e WakeEvent) Validate() error {
	normalized := e.Normalize()
	switch {
	case normalized.ID == "":
		return fmt.Errorf("%w: id is required", ErrInvalidWakeEvent)
	case normalized.WorkspaceID == "":
		return fmt.Errorf("%w: workspace id is required", ErrInvalidWakeEvent)
	case normalized.AgentName == "":
		return fmt.Errorf("%w: agent name is required", ErrInvalidWakeEvent)
	case !ValidWakeSource(normalized.Source):
		return fmt.Errorf("%w: invalid source %q", ErrInvalidWakeEvent, normalized.Source)
	case !ValidWakeResult(normalized.Result):
		return fmt.Errorf("%w: invalid result %q", ErrInvalidWakeEvent, normalized.Result)
	case !ValidWakeReason(normalized.Reason):
		return fmt.Errorf("%w: invalid reason %q", ErrInvalidWakeEvent, normalized.Reason)
	case normalized.CreatedAt.IsZero():
		return fmt.Errorf("%w: created_at is required", ErrInvalidWakeEvent)
	case normalized.ExpiresAt.IsZero():
		return fmt.Errorf("%w: expires_at is required", ErrInvalidWakeEvent)
	}
	return nil
}

// Validate ensures the wake event query uses sane bounds and enum filters.
func (q WakeEventListQuery) Validate() error {
	if q.Limit < 0 {
		return fmt.Errorf("%w: invalid wake event limit %d", ErrInvalidWakeEvent, q.Limit)
	}
	if q.Source != "" && !ValidWakeSource(q.Source) {
		return fmt.Errorf("%w: invalid source %q", ErrInvalidWakeEvent, q.Source)
	}
	if q.Result != "" && !ValidWakeResult(q.Result) {
		return fmt.Errorf("%w: invalid result %q", ErrInvalidWakeEvent, q.Result)
	}
	if q.Reason != "" && !ValidWakeReason(q.Reason) {
		return fmt.Errorf("%w: invalid reason %q", ErrInvalidWakeEvent, q.Reason)
	}
	return nil
}

// ValidRevisionOperation reports whether operation is a supported revision enum member.
func ValidRevisionOperation(operation RevisionOperation) bool {
	switch operation {
	case RevisionOperationWrite, RevisionOperationDelete, RevisionOperationRollback:
		return true
	default:
		return false
	}
}

// ValidActorKind reports whether kind is a supported authoring actor enum member.
func ValidActorKind(kind ActorKind) bool {
	switch kind {
	case ActorKindUser, ActorKindAgent, ActorKindExtension, ActorKindSystem:
		return true
	default:
		return false
	}
}

// ValidSessionHealthState reports whether state is a supported session health state.
func ValidSessionHealthState(state SessionHealthState) bool {
	switch state {
	case SessionHealthStateIdle,
		SessionHealthStatePrompting,
		SessionHealthStateStopped,
		SessionHealthStateDetached:
		return true
	default:
		return false
	}
}

// ValidSessionHealthStatus reports whether health is a supported session health enum member.
func ValidSessionHealthStatus(health SessionHealthStatus) bool {
	switch health {
	case SessionHealthHealthy,
		SessionHealthDegraded,
		SessionHealthStale,
		SessionHealthDead,
		SessionHealthUnknown:
		return true
	default:
		return false
	}
}

// ValidSessionHealthIneligibilityReason reports whether reason is a supported session-health reason.
func ValidSessionHealthIneligibilityReason(reason string) bool {
	switch SessionHealthIneligibilityReason(strings.TrimSpace(reason)) {
	case SessionHealthReasonPromptActive,
		SessionHealthReasonNotAttachable,
		SessionHealthReasonUnhealthy,
		SessionHealthReasonStale,
		SessionHealthReasonHung,
		SessionHealthReasonDead,
		SessionHealthReasonUnknown:
		return true
	default:
		return false
	}
}

// ValidWakeSource reports whether source is a supported wake source enum member.
func ValidWakeSource(source WakeSource) bool {
	switch source {
	case WakeSourceScheduler, WakeSourceManual, WakeSourceHarnessReentry:
		return true
	default:
		return false
	}
}

// ValidWakeResult reports whether result is a supported wake result enum member.
func ValidWakeResult(result WakeResult) bool {
	switch result {
	case WakeResultSent,
		WakeResultSkipped,
		WakeResultCoalesced,
		WakeResultRateLimited,
		WakeResultFailed:
		return true
	default:
		return false
	}
}

// ValidWakeReason reports whether reason is a supported wake reason enum member.
func ValidWakeReason(reason WakeReason) bool {
	switch reason {
	case WakeReasonSent,
		WakeReasonHeartbeatDisabled,
		WakeReasonHeartbeatInvalid,
		WakeReasonHeartbeatNoPolicy,
		WakeReasonHeartbeatRateLimited,
		WakeReasonHeartbeatNoEligible,
		WakeReasonCooldownActive,
		WakeReasonQuietWindow,
		WakeReasonSessionNotFound,
		WakeReasonSessionUnhealthy,
		WakeReasonSessionNotAttachable,
		WakeReasonSessionPromptActive,
		WakeReasonSessionPromptRace,
		WakeReasonSyntheticPromptFailed,
		WakeReasonCoalesced:
		return true
	default:
		return false
	}
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
