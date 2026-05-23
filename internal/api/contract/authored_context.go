package contract

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/diagnostics"
	heartbeatpkg "github.com/compozy/agh/internal/heartbeat"
	soulpkg "github.com/compozy/agh/internal/soul"
)

const maxAuthoredContextLastErrorBytes = 2048

var (
	// ErrUnsafeAuthoredContextPayload reports raw credentials or disallowed prompt data in a public DTO.
	ErrUnsafeAuthoredContextPayload = errors.New(
		"contract: authored context payload contains unsafe token or prompt data",
	)
	// ErrInvalidAuthoredContextEnum reports a value outside a closed public contract enum.
	ErrInvalidAuthoredContextEnum = errors.New("contract: invalid authored context enum")
)

// AuthoredValidationStatus is the closed validation state shared by authored-context surfaces.
type AuthoredValidationStatus string

const (
	AuthoredValidationMissing  AuthoredValidationStatus = "missing"
	AuthoredValidationInactive AuthoredValidationStatus = "inactive"
	AuthoredValidationValid    AuthoredValidationStatus = "valid"
	AuthoredValidationInvalid  AuthoredValidationStatus = "invalid"
)

// AuthoredDiagnosticSeverity is the closed diagnostic severity shared by Soul and Heartbeat DTOs.
type AuthoredDiagnosticSeverity string

const (
	AuthoredDiagnosticInfo    AuthoredDiagnosticSeverity = "info"
	AuthoredDiagnosticWarning AuthoredDiagnosticSeverity = "warning"
	AuthoredDiagnosticError   AuthoredDiagnosticSeverity = "error"
)

// AgentSoulRevisionAction describes one managed SOUL.md authoring mutation.
type AgentSoulRevisionAction string

const (
	AgentSoulRevisionPut      AgentSoulRevisionAction = "put"
	AgentSoulRevisionDelete   AgentSoulRevisionAction = "delete"
	AgentSoulRevisionRollback AgentSoulRevisionAction = "rollback"
)

// HeartbeatRevisionOperation describes one managed HEARTBEAT.md authoring mutation.
type HeartbeatRevisionOperation string

const (
	HeartbeatRevisionWrite    HeartbeatRevisionOperation = "write"
	HeartbeatRevisionDelete   HeartbeatRevisionOperation = "delete"
	HeartbeatRevisionRollback HeartbeatRevisionOperation = "rollback"
)

// HeartbeatActorKind describes a redacted Heartbeat authoring actor class.
type HeartbeatActorKind string

const (
	HeartbeatActorUser      HeartbeatActorKind = "user"
	HeartbeatActorAgent     HeartbeatActorKind = "agent"
	HeartbeatActorExtension HeartbeatActorKind = "extension"
	HeartbeatActorSystem    HeartbeatActorKind = "system"
)

// SessionHealthState describes daemon-owned runtime state for wake eligibility.
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

// SessionHealthIneligibilityReason is the closed reason for wake-ineligible health rows.
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

// HeartbeatWakeSource classifies who requested an advisory Heartbeat wake.
type HeartbeatWakeSource string

const (
	HeartbeatWakeSourceScheduler      HeartbeatWakeSource = "scheduler"
	HeartbeatWakeSourceManual         HeartbeatWakeSource = "manual"
	HeartbeatWakeSourceHarnessReentry HeartbeatWakeSource = "harness_reentry"
)

// HeartbeatWakeResult classifies the outcome of an advisory Heartbeat wake attempt.
type HeartbeatWakeResult string

const (
	HeartbeatWakeResultSent        HeartbeatWakeResult = "sent"
	HeartbeatWakeResultSkipped     HeartbeatWakeResult = "skipped"
	HeartbeatWakeResultCoalesced   HeartbeatWakeResult = "coalesced"
	HeartbeatWakeResultRateLimited HeartbeatWakeResult = "rate_limited"
	HeartbeatWakeResultFailed      HeartbeatWakeResult = "failed"
)

// HeartbeatWakeReason is the closed deterministic reason for Heartbeat wake audit.
type HeartbeatWakeReason string

const (
	HeartbeatWakeReasonSent                  HeartbeatWakeReason = "wake_sent"
	HeartbeatWakeReasonDisabled              HeartbeatWakeReason = "heartbeat_disabled"
	HeartbeatWakeReasonInvalid               HeartbeatWakeReason = "heartbeat_invalid"
	HeartbeatWakeReasonNoPolicy              HeartbeatWakeReason = "heartbeat_no_policy"
	HeartbeatWakeReasonRateLimited           HeartbeatWakeReason = "heartbeat_rate_limited"
	HeartbeatWakeReasonNoEligibleSession     HeartbeatWakeReason = "heartbeat_no_eligible_session"
	HeartbeatWakeReasonCooldownActive        HeartbeatWakeReason = "cooldown_active"
	HeartbeatWakeReasonQuietWindow           HeartbeatWakeReason = "quiet_window"
	HeartbeatWakeReasonSessionNotFound       HeartbeatWakeReason = "session_not_found"
	HeartbeatWakeReasonSessionUnhealthy      HeartbeatWakeReason = "session_unhealthy"
	HeartbeatWakeReasonSessionNotAttachable  HeartbeatWakeReason = "session_not_attachable"
	HeartbeatWakeReasonSessionPromptActive   HeartbeatWakeReason = "session_prompt_active"
	HeartbeatWakeReasonSessionPromptRace     HeartbeatWakeReason = "session_prompt_active_race"
	HeartbeatWakeReasonSyntheticPromptFailed HeartbeatWakeReason = "synthetic_prompt_failed"
	HeartbeatWakeReasonCoalesced             HeartbeatWakeReason = "wake_coalesced"
)

// AuthoredContextDiagnosticPayload is a redacted, deterministic authored-context diagnostic.
type AuthoredContextDiagnosticPayload struct {
	Code         string                     `json:"code"`
	Severity     AuthoredDiagnosticSeverity `json:"severity"`
	Message      string                     `json:"message"`
	Field        string                     `json:"field,omitempty"`
	Section      string                     `json:"section,omitempty"`
	SourcePath   string                     `json:"source_path,omitempty"`
	Line         int                        `json:"line,omitempty"`
	Column       int                        `json:"column,omitempty"`
	OwnerSurface string                     `json:"owner_surface,omitempty"`
}

// AuthoredContextLimitsPayload reports stable parser/projection limits.
type AuthoredContextLimitsPayload struct {
	MaxBodyBytes           int64 `json:"max_body_bytes"`
	ContextProjectionBytes int64 `json:"context_projection_bytes,omitempty"`
	MaxBytes               int64 `json:"max_bytes,omitempty"`
}

// AuthoredContextActorPayload records redacted actor or origin identity.
type AuthoredContextActorPayload struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref,omitempty"`
}

// AgentSoulConfigProvenancePayload reports the Soul config subset that shaped a read model.
type AgentSoulConfigProvenancePayload struct {
	Digest                 string `json:"digest"`
	Source                 string `json:"source,omitempty"`
	Enabled                bool   `json:"enabled"`
	MaxBodyBytes           int64  `json:"max_body_bytes"`
	ContextProjectionBytes int64  `json:"context_projection_bytes"`
}

// AgentSoulFrontmatterPayload is the allowlisted strict SOUL.md metadata projection.
type AgentSoulFrontmatterPayload struct {
	Version       string   `json:"version,omitempty"`
	Role          string   `json:"role,omitempty"`
	Tone          []string `json:"tone,omitempty"`
	Principles    []string `json:"principles,omitempty"`
	Constraints   []string `json:"constraints,omitempty"`
	Collaboration []string `json:"collaboration,omitempty"`
	MemoryPolicy  []string `json:"memory_policy,omitempty"`
	Tags          []string `json:"tags,omitempty"`
}

// AgentSoulSectionPayload is the compact, context-safe Soul projection for `/agent/context`.
type AgentSoulSectionPayload struct {
	Enabled          bool                     `json:"enabled"`
	Present          bool                     `json:"present"`
	Active           bool                     `json:"active"`
	Valid            bool                     `json:"valid"`
	ValidationStatus AuthoredValidationStatus `json:"validation_status,omitempty"`
	SnapshotID       string                   `json:"snapshot_id,omitempty"`
	Digest           string                   `json:"digest,omitempty"`
	ConfigDigest     string                   `json:"config_digest,omitempty"`
	SourcePath       string                   `json:"source_path,omitempty"`
	Role             string                   `json:"role,omitempty"`
	Tone             []string                 `json:"tone"`
	Principles       []string                 `json:"principles"`
	Truncated        bool                     `json:"truncated,omitempty"`
	MaxBytes         int64                    `json:"max_bytes,omitempty"`
	MaxBodyBytes     int64                    `json:"max_body_bytes,omitempty"`
}

// AgentSoulPayload is the full resolved Soul read model for dedicated inspect/authoring surfaces.
type AgentSoulPayload struct {
	AgentName        string                             `json:"agent_name,omitempty"`
	Enabled          bool                               `json:"enabled"`
	Present          bool                               `json:"present"`
	Active           bool                               `json:"active"`
	Valid            bool                               `json:"valid"`
	ValidationStatus AuthoredValidationStatus           `json:"validation_status"`
	SnapshotID       string                             `json:"snapshot_id,omitempty"`
	RevisionID       string                             `json:"revision_id,omitempty"`
	Digest           string                             `json:"digest,omitempty"`
	SourcePath       string                             `json:"source_path,omitempty"`
	Frontmatter      AgentSoulFrontmatterPayload        `json:"frontmatter"`
	Body             string                             `json:"body,omitempty"`
	Truncated        bool                               `json:"truncated,omitempty"`
	Limits           AuthoredContextLimitsPayload       `json:"limits"`
	ConfigProvenance AgentSoulConfigProvenancePayload   `json:"config_provenance"`
	Diagnostics      []AuthoredContextDiagnosticPayload `json:"diagnostics,omitempty"`
	CreatedAt        *time.Time                         `json:"created_at,omitempty"`
}

// AgentSoulValidateRequest validates a proposed SOUL.md body or the current file.
type AgentSoulValidateRequest struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	AgentName   string `json:"agent_name,omitempty"`
	Body        string `json:"body,omitempty"`
}

// AgentSoulValidateByPathRequest validates SOUL.md for the route agent.
type AgentSoulValidateByPathRequest struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	Body        string `json:"body,omitempty"`
}

// AgentSoulPutRequest creates or replaces SOUL.md through the managed authoring service.
type AgentSoulPutRequest struct {
	WorkspaceID    string `json:"workspace_id,omitempty"`
	AgentName      string `json:"agent_name"`
	Body           string `json:"body"`
	ExpectedDigest string `json:"expected_digest"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// AgentSoulPutByPathRequest creates or replaces SOUL.md for the route agent.
type AgentSoulPutByPathRequest struct {
	WorkspaceID    string `json:"workspace_id,omitempty"`
	Body           string `json:"body"`
	ExpectedDigest string `json:"expected_digest"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// AgentSoulDeleteRequest removes SOUL.md through the managed authoring service.
type AgentSoulDeleteRequest struct {
	WorkspaceID    string `json:"workspace_id,omitempty"`
	AgentName      string `json:"agent_name"`
	ExpectedDigest string `json:"expected_digest"`
}

// AgentSoulDeleteByPathRequest removes SOUL.md for the route agent.
type AgentSoulDeleteByPathRequest struct {
	WorkspaceID    string `json:"workspace_id,omitempty"`
	ExpectedDigest string `json:"expected_digest"`
}

// AgentSoulRollbackRequest restores a prior SOUL.md revision through the managed authoring service.
type AgentSoulRollbackRequest struct {
	WorkspaceID    string `json:"workspace_id,omitempty"`
	AgentName      string `json:"agent_name"`
	RevisionID     string `json:"revision_id"`
	ExpectedDigest string `json:"expected_digest"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// AgentSoulRollbackByPathRequest restores a prior SOUL.md revision for the route agent.
type AgentSoulRollbackByPathRequest struct {
	WorkspaceID    string `json:"workspace_id,omitempty"`
	RevisionID     string `json:"revision_id"`
	ExpectedDigest string `json:"expected_digest"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// AgentSoulHistoryRequest lists managed SOUL.md authoring revisions.
type AgentSoulHistoryRequest struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	AgentName   string `json:"agent_name"`
	Limit       int    `json:"limit,omitempty"`
	Cursor      string `json:"cursor,omitempty"`
}

// AgentSoulRevisionPayload is a redacted SOUL.md authoring history row.
type AgentSoulRevisionPayload struct {
	ID             string                             `json:"id"`
	AgentName      string                             `json:"agent_name"`
	SourcePath     string                             `json:"source_path"`
	Action         AgentSoulRevisionAction            `json:"action"`
	PreviousDigest string                             `json:"previous_digest,omitempty"`
	NewDigest      string                             `json:"new_digest,omitempty"`
	Diagnostics    []AuthoredContextDiagnosticPayload `json:"diagnostics,omitempty"`
	Actor          AuthoredContextActorPayload        `json:"actor"`
	Origin         *AuthoredContextActorPayload       `json:"origin,omitempty"`
	CreatedAt      time.Time                          `json:"created_at"`
}

// AgentSoulHistoryResponse returns bounded SOUL.md authoring history.
type AgentSoulHistoryResponse struct {
	Revisions  []AgentSoulRevisionPayload `json:"revisions"`
	NextCursor string                     `json:"next_cursor,omitempty"`
}

// AgentSoulMutationResponse returns the post-mutation read model and audit row.
type AgentSoulMutationResponse struct {
	Soul     AgentSoulPayload         `json:"soul"`
	Revision AgentSoulRevisionPayload `json:"revision"`
}

// SessionSoulRefreshRequest refreshes an idle session Soul snapshot through body-level CAS.
type SessionSoulRefreshRequest struct {
	ExpectedDigest string `json:"expected_digest"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// HeartbeatTimeWindowPayload is one authored local wall-clock active or quiet window.
type HeartbeatTimeWindowPayload struct {
	Timezone string `json:"timezone"`
	Start    string `json:"start"`
	End      string `json:"end"`
}

// HeartbeatContextProjectionPayload captures authored context projection hints.
type HeartbeatContextProjectionPayload struct {
	Include []string `json:"include,omitempty"`
}

// HeartbeatFrontmatterPreferencesPayload captures authored preference hints before config bounds.
type HeartbeatFrontmatterPreferencesPayload struct {
	MinInterval  string                       `json:"min_interval,omitempty"`
	ActiveHours  []HeartbeatTimeWindowPayload `json:"active_hours,omitempty"`
	QuietWindows []HeartbeatTimeWindowPayload `json:"quiet_windows,omitempty"`
}

// HeartbeatFrontmatterPayload is the allowlisted strict HEARTBEAT.md metadata projection.
type HeartbeatFrontmatterPayload struct {
	Version     int                                    `json:"version"`
	Enabled     bool                                   `json:"enabled"`
	Summary     string                                 `json:"summary,omitempty"`
	Preferences HeartbeatFrontmatterPreferencesPayload `json:"preferences"`
	Context     HeartbeatContextProjectionPayload      `json:"context"`
}

// HeartbeatPreferencesPayload captures config-bound wake policy hints.
type HeartbeatPreferencesPayload struct {
	MinInterval  string                            `json:"min_interval"`
	ActiveHours  []HeartbeatTimeWindowPayload      `json:"active_hours,omitempty"`
	QuietWindows []HeartbeatTimeWindowPayload      `json:"quiet_windows,omitempty"`
	Context      HeartbeatContextProjectionPayload `json:"context"`
}

// HeartbeatConfigSubsetPayload is the canonical config authority subset for Heartbeat.
type HeartbeatConfigSubsetPayload struct {
	Enabled                      bool   `json:"enabled"`
	MaxBodyBytes                 int64  `json:"max_body_bytes"`
	ContextProjectionBytes       int64  `json:"context_projection_bytes"`
	MinInterval                  string `json:"min_interval"`
	DefaultInterval              string `json:"default_interval"`
	WakeCooldown                 string `json:"wake_cooldown"`
	MaxWakesPerCycle             int    `json:"max_wakes_per_cycle"`
	ActiveSessionOnly            bool   `json:"active_session_only"`
	AllowActiveHoursPreferences  bool   `json:"allow_active_hours_preferences"`
	WakeEventRetention           string `json:"wake_event_retention"`
	SessionHealthStaleAfter      string `json:"session_health_stale_after"`
	SessionHealthHookMinInterval string `json:"session_health_hook_min_interval"`
}

// HeartbeatConfigProvenancePayload reports the Heartbeat config subset that shaped a policy.
type HeartbeatConfigProvenancePayload struct {
	Digest string                       `json:"digest"`
	Subset HeartbeatConfigSubsetPayload `json:"subset"`
}

// HeartbeatPromptContributionPayload is the bounded synthetic-wake prompt contribution.
type HeartbeatPromptContributionPayload struct {
	Active           bool                               `json:"active"`
	Digest           string                             `json:"digest,omitempty"`
	ConfigDigest     string                             `json:"config_digest,omitempty"`
	SourcePath       string                             `json:"source_path,omitempty"`
	Summary          string                             `json:"summary,omitempty"`
	GuidanceMarkdown string                             `json:"guidance_markdown,omitempty"`
	Preferences      HeartbeatPreferencesPayload        `json:"preferences"`
	Truncated        bool                               `json:"truncated"`
	MaxBytes         int64                              `json:"max_bytes"`
	MaxBodyBytes     int64                              `json:"max_body_bytes"`
	Diagnostics      []AuthoredContextDiagnosticPayload `json:"diagnostics,omitempty"`
	Context          HeartbeatContextProjectionPayload  `json:"context"`
}

// HeartbeatPolicyPayload is the full resolved HEARTBEAT.md read model.
type HeartbeatPolicyPayload struct {
	AgentName        string                             `json:"agent_name,omitempty"`
	Enabled          bool                               `json:"enabled"`
	Present          bool                               `json:"present"`
	Active           bool                               `json:"active"`
	Valid            bool                               `json:"valid"`
	ValidationStatus AuthoredValidationStatus           `json:"validation_status"`
	SourcePath       string                             `json:"source_path,omitempty"`
	Digest           string                             `json:"digest,omitempty"`
	ConfigDigest     string                             `json:"config_digest,omitempty"`
	SnapshotID       string                             `json:"snapshot_id,omitempty"`
	SchemaVersion    int                                `json:"schema_version"`
	Summary          string                             `json:"summary,omitempty"`
	GuidanceMarkdown string                             `json:"guidance_markdown,omitempty"`
	Frontmatter      HeartbeatFrontmatterPayload        `json:"frontmatter"`
	Preferences      HeartbeatPreferencesPayload        `json:"preferences"`
	ConfigProvenance HeartbeatConfigProvenancePayload   `json:"config_provenance"`
	Prompt           HeartbeatPromptContributionPayload `json:"prompt"`
	Diagnostics      []AuthoredContextDiagnosticPayload `json:"diagnostics,omitempty"`
	Limits           AuthoredContextLimitsPayload       `json:"limits"`
	CreatedAt        *time.Time                         `json:"created_at,omitempty"`
}

// HeartbeatValidateRequest validates a proposed HEARTBEAT.md body.
type HeartbeatValidateRequest struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	AgentName   string `json:"agent_name,omitempty"`
	Body        string `json:"body"`
}

// HeartbeatValidateByPathRequest validates HEARTBEAT.md for the route agent.
type HeartbeatValidateByPathRequest struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	Body        string `json:"body"`
}

// HeartbeatPutRequest creates or replaces HEARTBEAT.md through the managed authoring service.
type HeartbeatPutRequest struct {
	WorkspaceID    string `json:"workspace_id,omitempty"`
	AgentName      string `json:"agent_name"`
	Body           string `json:"body"`
	ExpectedDigest string `json:"expected_digest"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// HeartbeatPutByPathRequest creates or replaces HEARTBEAT.md for the route agent.
type HeartbeatPutByPathRequest struct {
	WorkspaceID    string `json:"workspace_id,omitempty"`
	Body           string `json:"body"`
	ExpectedDigest string `json:"expected_digest"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// HeartbeatDeleteRequest removes HEARTBEAT.md through the managed authoring service.
type HeartbeatDeleteRequest struct {
	WorkspaceID    string `json:"workspace_id,omitempty"`
	AgentName      string `json:"agent_name"`
	ExpectedDigest string `json:"expected_digest"`
}

// HeartbeatDeleteByPathRequest removes HEARTBEAT.md for the route agent.
type HeartbeatDeleteByPathRequest struct {
	WorkspaceID    string `json:"workspace_id,omitempty"`
	ExpectedDigest string `json:"expected_digest"`
}

// HeartbeatRollbackRequest restores a prior HEARTBEAT.md revision or snapshot body.
type HeartbeatRollbackRequest struct {
	WorkspaceID    string `json:"workspace_id,omitempty"`
	AgentName      string `json:"agent_name"`
	RevisionID     string `json:"revision_id,omitempty"`
	TargetDigest   string `json:"target_digest,omitempty"`
	ExpectedDigest string `json:"expected_digest"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// HeartbeatRollbackByPathRequest restores a HEARTBEAT.md revision for the route agent.
type HeartbeatRollbackByPathRequest struct {
	WorkspaceID    string `json:"workspace_id,omitempty"`
	RevisionID     string `json:"revision_id,omitempty"`
	TargetDigest   string `json:"target_digest,omitempty"`
	ExpectedDigest string `json:"expected_digest"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// HeartbeatHistoryRequest lists managed HEARTBEAT.md authoring revisions.
type HeartbeatHistoryRequest struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	AgentName   string `json:"agent_name"`
	Limit       int    `json:"limit,omitempty"`
	Cursor      string `json:"cursor,omitempty"`
}

// HeartbeatStatusRequest composes policy, wake state, and optional session health.
type HeartbeatStatusRequest struct {
	WorkspaceID             string `json:"workspace_id,omitempty"`
	AgentName               string `json:"agent_name"`
	SessionID               string `json:"session_id,omitempty"`
	IncludeSessionHealth    bool   `json:"include_session_health,omitempty"`
	IncludeRecentWakeEvents bool   `json:"include_recent_wake_events,omitempty"`
}

// HeartbeatRevisionPayload is a redacted HEARTBEAT.md authoring history row.
type HeartbeatRevisionPayload struct {
	ID             string                     `json:"id"`
	AgentName      string                     `json:"agent_name"`
	SourcePath     string                     `json:"source_path"`
	Operation      HeartbeatRevisionOperation `json:"operation"`
	PreviousDigest string                     `json:"previous_digest,omitempty"`
	NewDigest      string                     `json:"new_digest,omitempty"`
	NewSnapshotID  string                     `json:"new_snapshot_id,omitempty"`
	Actor          HeartbeatActorPayload      `json:"actor"`
	CreatedAt      time.Time                  `json:"created_at"`
}

// HeartbeatActorPayload records a redacted Heartbeat authoring actor.
type HeartbeatActorPayload struct {
	Kind HeartbeatActorKind `json:"kind"`
	Ref  string             `json:"ref,omitempty"`
}

// HeartbeatHistoryResponse returns bounded HEARTBEAT.md authoring history.
type HeartbeatHistoryResponse struct {
	Revisions  []HeartbeatRevisionPayload `json:"revisions"`
	NextCursor string                     `json:"next_cursor,omitempty"`
}

// HeartbeatMutationResponse returns the post-mutation read model and audit row.
type HeartbeatMutationResponse struct {
	Heartbeat HeartbeatPolicyPayload   `json:"heartbeat"`
	Revision  HeartbeatRevisionPayload `json:"revision"`
}

// SessionHealthPayload is the metadata-only runtime health row for one session.
type SessionHealthPayload struct {
	SessionID           string                           `json:"session_id"`
	WorkspaceID         string                           `json:"workspace_id"`
	AgentName           string                           `json:"agent_name"`
	State               SessionHealthState               `json:"state"`
	Health              SessionHealthStatus              `json:"health"`
	ActivePrompt        bool                             `json:"active_prompt"`
	Attachable          bool                             `json:"attachable"`
	EligibleForWake     bool                             `json:"eligible_for_wake"`
	IneligibilityReason SessionHealthIneligibilityReason `json:"ineligibility_reason,omitempty"`
	LastActivityAt      *time.Time                       `json:"last_activity_at,omitempty"`
	LastPresenceAt      *time.Time                       `json:"last_presence_at,omitempty"`
	LastError           string                           `json:"last_error,omitempty"`
	UpdatedAt           time.Time                        `json:"updated_at"`
}

// SessionHealthResponse wraps one session health read model.
type SessionHealthResponse struct {
	Health SessionHealthPayload `json:"health"`
}

// SessionStatusResponse returns compact session status plus wake eligibility.
type SessionStatusResponse struct {
	SessionID           string                           `json:"session_id"`
	WorkspaceID         string                           `json:"workspace_id"`
	AgentName           string                           `json:"agent_name"`
	State               SessionHealthState               `json:"state"`
	Health              SessionHealthStatus              `json:"health"`
	ActivePrompt        bool                             `json:"active_prompt"`
	Attachable          bool                             `json:"attachable"`
	EligibleForWake     bool                             `json:"eligible_for_wake"`
	IneligibilityReason SessionHealthIneligibilityReason `json:"ineligibility_reason,omitempty"`
	WakeState           *HeartbeatWakeStatePayload       `json:"wake_state,omitempty"`
	UpdatedAt           time.Time                        `json:"updated_at"`
}

// SessionInspectResponse returns detailed health, wake audit, and policy diagnostics.
type SessionInspectResponse struct {
	SessionID    string                             `json:"session_id"`
	Health       SessionHealthPayload               `json:"health"`
	WakeState    *HeartbeatWakeStatePayload         `json:"wake_state,omitempty"`
	WakeEvents   []HeartbeatWakeEventPayload        `json:"wake_events,omitempty"`
	PolicyDigest string                             `json:"policy_digest,omitempty"`
	ConfigDigest string                             `json:"config_digest,omitempty"`
	Diagnostics  []AuthoredContextDiagnosticPayload `json:"diagnostics,omitempty"`
}

// HeartbeatWakeStatePayload is the per-session cooldown/coalescing summary.
type HeartbeatWakeStatePayload struct {
	WorkspaceID      string              `json:"workspace_id,omitempty"`
	AgentName        string              `json:"agent_name,omitempty"`
	SessionID        string              `json:"session_id"`
	PolicySnapshotID string              `json:"policy_snapshot_id,omitempty"`
	LastWakeAt       *time.Time          `json:"last_wake_at,omitempty"`
	NextAllowedAt    *time.Time          `json:"next_allowed_at,omitempty"`
	CoalescedCount   int                 `json:"coalesced_count"`
	LastResult       HeartbeatWakeResult `json:"last_result"`
	LastReason       HeartbeatWakeReason `json:"last_reason,omitempty"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

// HeartbeatWakeEventPayload is one retained Heartbeat wake audit row.
type HeartbeatWakeEventPayload struct {
	ID                string              `json:"id"`
	WorkspaceID       string              `json:"workspace_id,omitempty"`
	AgentName         string              `json:"agent_name,omitempty"`
	SessionID         string              `json:"session_id,omitempty"`
	PolicySnapshotID  string              `json:"policy_snapshot_id,omitempty"`
	Source            HeartbeatWakeSource `json:"source"`
	Result            HeartbeatWakeResult `json:"result"`
	Reason            HeartbeatWakeReason `json:"reason"`
	SyntheticPromptID string              `json:"synthetic_prompt_id,omitempty"`
	CreatedAt         time.Time           `json:"created_at"`
	ExpiresAt         time.Time           `json:"expires_at"`
}

// HeartbeatWakeDecisionPayload reports the auditable result of one wake decision.
type HeartbeatWakeDecisionPayload struct {
	WakeEventID       string                             `json:"wake_event_id,omitempty"`
	Result            HeartbeatWakeResult                `json:"result"`
	Reason            HeartbeatWakeReason                `json:"reason"`
	PolicySnapshotID  string                             `json:"policy_snapshot_id,omitempty"`
	PolicyDigest      string                             `json:"policy_digest,omitempty"`
	ConfigDigest      string                             `json:"config_digest,omitempty"`
	SyntheticPromptID string                             `json:"synthetic_prompt_id,omitempty"`
	Diagnostics       []AuthoredContextDiagnosticPayload `json:"diagnostics,omitempty"`
}

// HeartbeatWakeRequest asks for one manual advisory Heartbeat wake decision.
type HeartbeatWakeRequest struct {
	WorkspaceID    string              `json:"workspace_id,omitempty"`
	AgentName      string              `json:"agent_name"`
	SessionID      string              `json:"session_id"`
	Source         HeartbeatWakeSource `json:"source"`
	DryRun         bool                `json:"dry_run,omitempty"`
	IdempotencyKey string              `json:"idempotency_key,omitempty"`
}

// HeartbeatWakeByPathRequest asks for one wake decision for the route agent.
type HeartbeatWakeByPathRequest struct {
	WorkspaceID    string              `json:"workspace_id,omitempty"`
	SessionID      string              `json:"session_id"`
	Source         HeartbeatWakeSource `json:"source"`
	DryRun         bool                `json:"dry_run,omitempty"`
	IdempotencyKey string              `json:"idempotency_key,omitempty"`
}

// HeartbeatWakeResponse wraps one advisory wake decision.
type HeartbeatWakeResponse struct {
	Decision HeartbeatWakeDecisionPayload `json:"decision"`
}

// HeartbeatStatusResponse composes policy status, wake state, recent wake audit, and optional health.
type HeartbeatStatusResponse struct {
	AgentName        string                             `json:"agent_name"`
	SourcePath       string                             `json:"source_path,omitempty"`
	Enabled          bool                               `json:"enabled"`
	Present          bool                               `json:"present"`
	Active           bool                               `json:"active"`
	Valid            bool                               `json:"valid"`
	ValidationStatus AuthoredValidationStatus           `json:"validation_status"`
	Digest           string                             `json:"digest,omitempty"`
	ConfigDigest     string                             `json:"config_digest,omitempty"`
	SnapshotID       string                             `json:"snapshot_id,omitempty"`
	Summary          string                             `json:"summary,omitempty"`
	Preferences      HeartbeatPreferencesPayload        `json:"preferences"`
	Diagnostics      []AuthoredContextDiagnosticPayload `json:"diagnostics,omitempty"`
	WakeState        *HeartbeatWakeStatePayload         `json:"wake_state,omitempty"`
	WakeEvents       []HeartbeatWakeEventPayload        `json:"wake_events,omitempty"`
	SessionHealth    *SessionHealthPayload              `json:"session_health,omitempty"`
	RevisionCursor   string                             `json:"revision_cursor,omitempty"`
}

// SessionPayloadHealth attaches optional health to session list/read responses when requested.
type SessionPayloadHealth = SessionHealthPayload

// AuthoredValidationStatusValues returns the closed authored validation status enum values.
func AuthoredValidationStatusValues() []string {
	return []string{
		string(AuthoredValidationMissing),
		string(AuthoredValidationInactive),
		string(AuthoredValidationValid),
		string(AuthoredValidationInvalid),
	}
}

// AuthoredDiagnosticSeverityValues returns the closed diagnostic severity enum values.
func AuthoredDiagnosticSeverityValues() []string {
	return []string{
		string(AuthoredDiagnosticInfo),
		string(AuthoredDiagnosticWarning),
		string(AuthoredDiagnosticError),
	}
}

// AgentSoulRevisionActionValues returns the closed Soul revision action enum values.
func AgentSoulRevisionActionValues() []string {
	return []string{
		string(AgentSoulRevisionPut),
		string(AgentSoulRevisionDelete),
		string(AgentSoulRevisionRollback),
	}
}

// HeartbeatRevisionOperationValues returns the closed Heartbeat revision operation enum values.
func HeartbeatRevisionOperationValues() []string {
	return []string{
		string(HeartbeatRevisionWrite),
		string(HeartbeatRevisionDelete),
		string(HeartbeatRevisionRollback),
	}
}

// HeartbeatActorKindValues returns the closed Heartbeat actor kind enum values.
func HeartbeatActorKindValues() []string {
	return []string{
		string(HeartbeatActorUser),
		string(HeartbeatActorAgent),
		string(HeartbeatActorExtension),
		string(HeartbeatActorSystem),
	}
}

// SessionHealthStateValues returns the closed session health state enum values.
func SessionHealthStateValues() []string {
	return []string{
		string(SessionHealthStateIdle),
		string(SessionHealthStatePrompting),
		string(SessionHealthStateStopped),
		string(SessionHealthStateDetached),
	}
}

// SessionHealthStatusValues returns the closed session health status enum values.
func SessionHealthStatusValues() []string {
	return []string{
		string(SessionHealthHealthy),
		string(SessionHealthDegraded),
		string(SessionHealthStale),
		string(SessionHealthDead),
		string(SessionHealthUnknown),
	}
}

// SessionHealthIneligibilityReasonValues returns closed wake-ineligibility reason enum values.
func SessionHealthIneligibilityReasonValues() []string {
	return []string{
		string(SessionHealthReasonPromptActive),
		string(SessionHealthReasonNotAttachable),
		string(SessionHealthReasonUnhealthy),
		string(SessionHealthReasonStale),
		string(SessionHealthReasonHung),
		string(SessionHealthReasonDead),
		string(SessionHealthReasonUnknown),
	}
}

// HeartbeatWakeSourceValues returns the closed wake source enum values.
func HeartbeatWakeSourceValues() []string {
	return []string{
		string(HeartbeatWakeSourceScheduler),
		string(HeartbeatWakeSourceManual),
		string(HeartbeatWakeSourceHarnessReentry),
	}
}

// HeartbeatWakeResultValues returns the closed wake result enum values.
func HeartbeatWakeResultValues() []string {
	return []string{
		string(HeartbeatWakeResultSent),
		string(HeartbeatWakeResultSkipped),
		string(HeartbeatWakeResultCoalesced),
		string(HeartbeatWakeResultRateLimited),
		string(HeartbeatWakeResultFailed),
	}
}

// HeartbeatWakeReasonValues returns the closed wake reason enum values.
func HeartbeatWakeReasonValues() []string {
	return []string{
		string(HeartbeatWakeReasonSent),
		string(HeartbeatWakeReasonDisabled),
		string(HeartbeatWakeReasonInvalid),
		string(HeartbeatWakeReasonNoPolicy),
		string(HeartbeatWakeReasonRateLimited),
		string(HeartbeatWakeReasonNoEligibleSession),
		string(HeartbeatWakeReasonCooldownActive),
		string(HeartbeatWakeReasonQuietWindow),
		string(HeartbeatWakeReasonSessionNotFound),
		string(HeartbeatWakeReasonSessionUnhealthy),
		string(HeartbeatWakeReasonSessionNotAttachable),
		string(HeartbeatWakeReasonSessionPromptActive),
		string(HeartbeatWakeReasonSessionPromptRace),
		string(HeartbeatWakeReasonSyntheticPromptFailed),
		string(HeartbeatWakeReasonCoalesced),
	}
}

// Valid reports whether the validation status is a closed enum member.
func (s AuthoredValidationStatus) Valid() bool {
	switch s {
	case AuthoredValidationMissing,
		AuthoredValidationInactive,
		AuthoredValidationValid,
		AuthoredValidationInvalid:
		return true
	default:
		return false
	}
}

// Validate reports an error for values outside the closed validation status enum.
func (s AuthoredValidationStatus) Validate() error {
	if s.Valid() {
		return nil
	}
	return fmt.Errorf("%w: validation_status %q", ErrInvalidAuthoredContextEnum, s)
}

// Valid reports whether the diagnostic severity is a closed enum member.
func (s AuthoredDiagnosticSeverity) Valid() bool {
	switch s {
	case AuthoredDiagnosticInfo, AuthoredDiagnosticWarning, AuthoredDiagnosticError:
		return true
	default:
		return false
	}
}

// Validate reports an error for values outside the closed diagnostic severity enum.
func (s AuthoredDiagnosticSeverity) Validate() error {
	if s.Valid() {
		return nil
	}
	return fmt.Errorf("%w: diagnostic severity %q", ErrInvalidAuthoredContextEnum, s)
}

// Valid reports whether the wake reason is a closed enum member.
func (r HeartbeatWakeReason) Valid() bool {
	switch r {
	case HeartbeatWakeReasonSent,
		HeartbeatWakeReasonDisabled,
		HeartbeatWakeReasonInvalid,
		HeartbeatWakeReasonNoPolicy,
		HeartbeatWakeReasonRateLimited,
		HeartbeatWakeReasonNoEligibleSession,
		HeartbeatWakeReasonCooldownActive,
		HeartbeatWakeReasonQuietWindow,
		HeartbeatWakeReasonSessionNotFound,
		HeartbeatWakeReasonSessionUnhealthy,
		HeartbeatWakeReasonSessionNotAttachable,
		HeartbeatWakeReasonSessionPromptActive,
		HeartbeatWakeReasonSessionPromptRace,
		HeartbeatWakeReasonSyntheticPromptFailed,
		HeartbeatWakeReasonCoalesced:
		return true
	default:
		return false
	}
}

// Validate reports an error for values outside the closed wake reason enum.
func (r HeartbeatWakeReason) Validate() error {
	if r.Valid() {
		return nil
	}
	return fmt.Errorf("%w: wake reason %q", ErrInvalidAuthoredContextEnum, r)
}

// Valid reports whether the health state is a closed enum member.
func (s SessionHealthState) Valid() bool {
	switch s {
	case SessionHealthStateIdle,
		SessionHealthStatePrompting,
		SessionHealthStateStopped,
		SessionHealthStateDetached:
		return true
	default:
		return false
	}
}

// Validate reports an error for values outside the closed health state enum.
func (s SessionHealthState) Validate() error {
	if s.Valid() {
		return nil
	}
	return fmt.Errorf("%w: session health state %q", ErrInvalidAuthoredContextEnum, s)
}

// Valid reports whether the health status is a closed enum member.
func (s SessionHealthStatus) Valid() bool {
	switch s {
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

// Validate reports an error for values outside the closed health status enum.
func (s SessionHealthStatus) Validate() error {
	if s.Valid() {
		return nil
	}
	return fmt.Errorf("%w: session health status %q", ErrInvalidAuthoredContextEnum, s)
}

// ContainsUnsafeAuthoredContextPayload reports whether a payload includes raw credentials or prompt data.
func ContainsUnsafeAuthoredContextPayload(payload any) (bool, error) {
	content, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Errorf("marshal authored-context safety payload: %w", err)
	}
	return containsUnsafeAuthoredContextJSON(content), nil
}

// ValidateAuthoredContextRedacted rejects raw credentials or disallowed prompt data in public DTOs.
func ValidateAuthoredContextRedacted(payload any) error {
	found, err := ContainsUnsafeAuthoredContextPayload(payload)
	if err != nil {
		return err
	}
	if found {
		return ErrUnsafeAuthoredContextPayload
	}
	return nil
}

// AgentSoulSectionPayloadFromResolved converts a resolver result into the compact context projection.
func AgentSoulSectionPayloadFromResolved(
	resolved *soulpkg.ResolvedSoul,
	snapshotID string,
	configProvenance soulpkg.ConfigProvenance,
) AgentSoulSectionPayload {
	if resolved == nil {
		return AgentSoulSectionPayload{}
	}
	compact := resolved.Compact
	return AgentSoulSectionPayload{
		Enabled:          compact.Enabled,
		Present:          compact.Present,
		Active:           compact.Active,
		Valid:            resolved.Valid,
		ValidationStatus: validationStatus(resolved.Present, resolved.Active, resolved.Valid),
		SnapshotID:       strings.TrimSpace(snapshotID),
		Digest:           firstAuthoredNonEmpty(compact.Digest, resolved.Digest),
		ConfigDigest:     strings.TrimSpace(configProvenance.Digest),
		SourcePath:       firstAuthoredNonEmpty(compact.SourcePath, resolved.SourcePath),
		Role:             strings.TrimSpace(compact.Role),
		Tone:             normalizeAuthoredStrings(compact.Tone),
		Principles:       normalizeAuthoredStrings(compact.Principles),
		Truncated:        compact.Truncated,
		MaxBytes:         compact.MaxBytes,
		MaxBodyBytes:     compact.MaxBodyBytes,
	}
}

// AgentSoulPayloadFromResolved converts a resolver result into the full Soul read model.
func AgentSoulPayloadFromResolved(
	agentName string,
	resolved *soulpkg.ResolvedSoul,
	snapshotID string,
	configProvenance soulpkg.ConfigProvenance,
) AgentSoulPayload {
	if resolved == nil {
		return AgentSoulPayload{AgentName: strings.TrimSpace(agentName)}
	}
	readModel := resolved.ReadModel
	return AgentSoulPayload{
		AgentName:        strings.TrimSpace(agentName),
		Enabled:          resolved.Enabled,
		Present:          resolved.Present,
		Active:           resolved.Active,
		Valid:            resolved.Valid,
		ValidationStatus: validationStatus(resolved.Present, resolved.Active, resolved.Valid),
		SnapshotID:       strings.TrimSpace(snapshotID),
		Digest:           firstAuthoredNonEmpty(readModel.Digest, resolved.Digest),
		SourcePath:       firstAuthoredNonEmpty(readModel.SourcePath, resolved.SourcePath),
		Frontmatter:      agentSoulFrontmatterPayload(readModel.Frontmatter),
		Body:             readModel.Body,
		Truncated:        readModel.Truncated,
		Limits: AuthoredContextLimitsPayload{
			MaxBodyBytes:           readModel.MaxBodyBytes,
			ContextProjectionBytes: readModel.ContextProjectionBytes,
		},
		ConfigProvenance: AgentSoulConfigProvenancePayloadFromDomain(configProvenance),
		Diagnostics:      soulDiagnosticsPayload(resolved.Diagnostics),
	}
}

// AgentSoulConfigProvenancePayloadFromDomain converts Soul config provenance into the public DTO.
func AgentSoulConfigProvenancePayloadFromDomain(
	provenance soulpkg.ConfigProvenance,
) AgentSoulConfigProvenancePayload {
	return AgentSoulConfigProvenancePayload{
		Digest:                 strings.TrimSpace(provenance.Digest),
		Source:                 strings.TrimSpace(provenance.Source),
		Enabled:                provenance.Enabled,
		MaxBodyBytes:           provenance.MaxBodyBytes,
		ContextProjectionBytes: provenance.ContextProjectionBytes,
	}
}

// HeartbeatPolicyPayloadFromResolved converts a resolver result into the full Heartbeat read model.
func HeartbeatPolicyPayloadFromResolved(
	agentName string,
	policy *heartbeatpkg.ResolvedPolicy,
	snapshotID string,
) (HeartbeatPolicyPayload, error) {
	if policy == nil {
		return HeartbeatPolicyPayload{AgentName: strings.TrimSpace(agentName)}, nil
	}
	diagnostics, err := heartbeatDiagnosticsPayload(policy.Diagnostics)
	if err != nil {
		return HeartbeatPolicyPayload{}, err
	}
	promptDiagnostics, err := heartbeatDiagnosticsPayload(policy.Prompt.Diagnostics)
	if err != nil {
		return HeartbeatPolicyPayload{}, err
	}
	return HeartbeatPolicyPayload{
		AgentName:        strings.TrimSpace(agentName),
		Enabled:          policy.Enabled,
		Present:          policy.Present,
		Active:           policy.Active,
		Valid:            policy.Valid,
		ValidationStatus: validationStatus(policy.Present, policy.Active, policy.Valid),
		SourcePath:       strings.TrimSpace(policy.SourcePath),
		Digest:           strings.TrimSpace(policy.Digest),
		ConfigDigest:     strings.TrimSpace(policy.ConfigDigest),
		SnapshotID:       strings.TrimSpace(snapshotID),
		SchemaVersion:    policy.SchemaVersion,
		Summary:          strings.TrimSpace(policy.Summary),
		GuidanceMarkdown: policy.GuidanceMarkdown,
		Frontmatter:      heartbeatFrontmatterPayload(policy.Frontmatter),
		Preferences:      heartbeatPreferencesPayload(policy.Preferences),
		ConfigProvenance: heartbeatConfigProvenancePayload(policy.ConfigProvenance),
		Prompt:           heartbeatPromptContributionPayload(policy.Prompt, promptDiagnostics),
		Diagnostics:      diagnostics,
		Limits: AuthoredContextLimitsPayload{
			MaxBodyBytes:           policy.Status.MaxBodyBytes,
			ContextProjectionBytes: policy.Status.ContextProjectionBytes,
		},
	}, nil
}

// HeartbeatStatusResponseFromResult converts composed Heartbeat status into the public DTO.
func HeartbeatStatusResponseFromResult(result *heartbeatpkg.StatusResult) (HeartbeatStatusResponse, error) {
	if result == nil {
		return HeartbeatStatusResponse{}, nil
	}
	diagnostics, err := heartbeatDiagnosticsPayload(result.Diagnostics)
	if err != nil {
		return HeartbeatStatusResponse{}, err
	}
	var wakeState *HeartbeatWakeStatePayload
	if result.WakeState != nil {
		converted, convertErr := HeartbeatWakeStatePayloadFromDomain(*result.WakeState)
		if convertErr != nil {
			return HeartbeatStatusResponse{}, convertErr
		}
		wakeState = &converted
	}
	var sessionHealth *SessionHealthPayload
	if result.SessionHealth != nil {
		converted, convertErr := SessionHealthPayloadFromDomain(*result.SessionHealth)
		if convertErr != nil {
			return HeartbeatStatusResponse{}, convertErr
		}
		sessionHealth = &converted
	}
	return HeartbeatStatusResponse{
		AgentName:        strings.TrimSpace(result.AgentName),
		SourcePath:       strings.TrimSpace(result.SourcePath),
		Enabled:          result.Enabled,
		Present:          result.Present,
		Active:           result.Active,
		Valid:            result.Valid,
		ValidationStatus: validationStatus(result.Present, result.Active, result.Valid),
		Digest:           strings.TrimSpace(result.Digest),
		ConfigDigest:     strings.TrimSpace(result.ConfigDigest),
		SnapshotID:       strings.TrimSpace(result.SnapshotID),
		Summary:          strings.TrimSpace(result.Summary),
		Preferences:      heartbeatPreferencesPayload(result.Preferences),
		Diagnostics:      diagnostics,
		WakeState:        wakeState,
		SessionHealth:    sessionHealth,
	}, nil
}

// SessionHealthPayloadFromDomain converts daemon-owned session health into the public DTO.
func SessionHealthPayloadFromDomain(health heartbeatpkg.SessionHealth) (SessionHealthPayload, error) {
	normalized := health.Normalize()
	state := SessionHealthState(normalized.State)
	if err := state.Validate(); err != nil {
		return SessionHealthPayload{}, err
	}
	status := SessionHealthStatus(normalized.Health)
	if err := status.Validate(); err != nil {
		return SessionHealthPayload{}, err
	}
	reason := SessionHealthIneligibilityReason(strings.TrimSpace(normalized.IneligibilityReason))
	if reason != "" && !validSessionHealthIneligibilityReason(reason) {
		return SessionHealthPayload{}, fmt.Errorf("%w: session health ineligibility reason %q",
			ErrInvalidAuthoredContextEnum,
			reason,
		)
	}
	return SessionHealthPayload{
		SessionID:           normalized.SessionID,
		WorkspaceID:         normalized.WorkspaceID,
		AgentName:           normalized.AgentName,
		State:               state,
		Health:              status,
		ActivePrompt:        normalized.ActivePrompt,
		Attachable:          normalized.Attachable,
		EligibleForWake:     normalized.EligibleForWake,
		IneligibilityReason: reason,
		LastActivityAt:      authoredTimePtr(normalized.LastActivityAt),
		LastPresenceAt:      authoredTimePtr(normalized.LastPresenceAt),
		LastError:           sanitizeAuthoredContextLastError(normalized.LastError),
		UpdatedAt:           normalized.UpdatedAt.UTC(),
	}, nil
}

// HeartbeatWakeStatePayloadFromDomain converts wake coalescing state into the public DTO.
func HeartbeatWakeStatePayloadFromDomain(state heartbeatpkg.WakeState) (HeartbeatWakeStatePayload, error) {
	normalized := state.Normalize()
	result := HeartbeatWakeResult(normalized.LastResult)
	if !validHeartbeatWakeResult(result) {
		return HeartbeatWakeStatePayload{}, fmt.Errorf("%w: wake result %q", ErrInvalidAuthoredContextEnum, result)
	}
	reason := HeartbeatWakeReason(normalized.LastReason)
	if reason != "" {
		if err := reason.Validate(); err != nil {
			return HeartbeatWakeStatePayload{}, err
		}
	}
	return HeartbeatWakeStatePayload{
		WorkspaceID:      normalized.WorkspaceID,
		AgentName:        normalized.AgentName,
		SessionID:        normalized.SessionID,
		PolicySnapshotID: normalized.PolicySnapshotID,
		LastWakeAt:       authoredTimePtr(normalized.LastWakeAt),
		NextAllowedAt:    authoredTimePtr(normalized.NextAllowedAt),
		CoalescedCount:   normalized.CoalescedCount,
		LastResult:       result,
		LastReason:       reason,
		UpdatedAt:        normalized.UpdatedAt.UTC(),
	}, nil
}

// HeartbeatWakeEventPayloadFromDomain converts one wake audit row into the public DTO.
func HeartbeatWakeEventPayloadFromDomain(event heartbeatpkg.WakeEvent) (HeartbeatWakeEventPayload, error) {
	normalized := event.Normalize()
	source := HeartbeatWakeSource(normalized.Source)
	if !validHeartbeatWakeSource(source) {
		return HeartbeatWakeEventPayload{}, fmt.Errorf("%w: wake source %q", ErrInvalidAuthoredContextEnum, source)
	}
	result := HeartbeatWakeResult(normalized.Result)
	if !validHeartbeatWakeResult(result) {
		return HeartbeatWakeEventPayload{}, fmt.Errorf("%w: wake result %q", ErrInvalidAuthoredContextEnum, result)
	}
	reason := HeartbeatWakeReason(normalized.Reason)
	if err := reason.Validate(); err != nil {
		return HeartbeatWakeEventPayload{}, err
	}
	return HeartbeatWakeEventPayload{
		ID:                normalized.ID,
		WorkspaceID:       normalized.WorkspaceID,
		AgentName:         normalized.AgentName,
		SessionID:         normalized.SessionID,
		PolicySnapshotID:  normalized.PolicySnapshotID,
		Source:            source,
		Result:            result,
		Reason:            reason,
		SyntheticPromptID: normalized.SyntheticPromptID,
		CreatedAt:         normalized.CreatedAt.UTC(),
		ExpiresAt:         normalized.ExpiresAt.UTC(),
	}, nil
}

// HeartbeatWakeDecisionPayloadFromDomain converts one wake decision into the public DTO.
func HeartbeatWakeDecisionPayloadFromDomain(
	decision heartbeatpkg.WakeDecision,
) (HeartbeatWakeDecisionPayload, error) {
	result := HeartbeatWakeResult(decision.Result)
	if !validHeartbeatWakeResult(result) {
		return HeartbeatWakeDecisionPayload{}, fmt.Errorf("%w: wake result %q", ErrInvalidAuthoredContextEnum, result)
	}
	reason := HeartbeatWakeReason(decision.Reason)
	if err := reason.Validate(); err != nil {
		return HeartbeatWakeDecisionPayload{}, err
	}
	diagnostics, err := heartbeatDiagnosticsPayload(decision.Diagnostics)
	if err != nil {
		return HeartbeatWakeDecisionPayload{}, err
	}
	return HeartbeatWakeDecisionPayload{
		WakeEventID:       strings.TrimSpace(decision.WakeEventID),
		Result:            result,
		Reason:            reason,
		PolicySnapshotID:  strings.TrimSpace(decision.PolicySnapshotID),
		PolicyDigest:      strings.TrimSpace(decision.PolicyDigest),
		ConfigDigest:      strings.TrimSpace(decision.ConfigDigest),
		SyntheticPromptID: strings.TrimSpace(decision.SyntheticPromptID),
		Diagnostics:       diagnostics,
	}, nil
}

// AgentSoulRevisionPayloadFromDomain converts a Soul authoring revision into the public DTO.
func AgentSoulRevisionPayloadFromDomain(revision soulpkg.Revision) (AgentSoulRevisionPayload, error) {
	normalized := revision.Normalize()
	action := AgentSoulRevisionAction(normalized.Action)
	if !validAgentSoulRevisionAction(action) {
		return AgentSoulRevisionPayload{}, fmt.Errorf("%w: soul revision action %q",
			ErrInvalidAuthoredContextEnum,
			action,
		)
	}
	diagnostics, err := decodeSoulRevisionDiagnostics(normalized.DiagnosticsJSON)
	if err != nil {
		return AgentSoulRevisionPayload{}, err
	}
	origin := authoredActorPtr(normalized.OriginKind, normalized.OriginRef)
	return AgentSoulRevisionPayload{
		ID:             normalized.ID,
		AgentName:      normalized.AgentName,
		SourcePath:     normalized.SourcePath,
		Action:         action,
		PreviousDigest: normalized.PreviousDigest,
		NewDigest:      normalized.NewDigest,
		Diagnostics:    diagnostics,
		Actor: AuthoredContextActorPayload{
			Kind: normalized.ActorKind,
			Ref:  normalized.ActorID,
		},
		Origin:    origin,
		CreatedAt: normalized.CreatedAt.UTC(),
	}, nil
}

// HeartbeatRevisionPayloadFromDomain converts a Heartbeat authoring revision into the public DTO.
func HeartbeatRevisionPayloadFromDomain(revision heartbeatpkg.Revision) (HeartbeatRevisionPayload, error) {
	normalized := revision.Normalize()
	operation := HeartbeatRevisionOperation(normalized.Operation)
	if !validHeartbeatRevisionOperation(operation) {
		return HeartbeatRevisionPayload{}, fmt.Errorf("%w: heartbeat revision operation %q",
			ErrInvalidAuthoredContextEnum,
			operation,
		)
	}
	actorKind := HeartbeatActorKind(normalized.ActorKind)
	if !validHeartbeatActorKind(actorKind) {
		return HeartbeatRevisionPayload{}, fmt.Errorf("%w: heartbeat actor kind %q",
			ErrInvalidAuthoredContextEnum,
			actorKind,
		)
	}
	return HeartbeatRevisionPayload{
		ID:             normalized.ID,
		AgentName:      normalized.AgentName,
		SourcePath:     normalized.SourcePath,
		Operation:      operation,
		PreviousDigest: normalized.PreviousDigest,
		NewDigest:      normalized.NewDigest,
		NewSnapshotID:  normalized.NewSnapshotID,
		Actor: HeartbeatActorPayload{
			Kind: actorKind,
			Ref:  normalized.ActorID,
		},
		CreatedAt: normalized.CreatedAt.UTC(),
	}, nil
}

func containsUnsafeAuthoredContextJSON(data []byte) bool {
	return containsUnsafePublicContractJSON(data)
}

func isUnsafeAuthoredContextString(value string) bool {
	return isUnsafePublicContractString(value)
}

func sanitizeAuthoredContextLastError(value string) string {
	redacted := diagnostics.RedactAndBound(value, maxAuthoredContextLastErrorBytes)
	if isUnsafeAuthoredContextString(redacted) {
		return "[REDACTED]"
	}
	return redacted
}

func validationStatus(present bool, active bool, valid bool) AuthoredValidationStatus {
	switch {
	case !present:
		return AuthoredValidationMissing
	case !valid:
		return AuthoredValidationInvalid
	case !active:
		return AuthoredValidationInactive
	default:
		return AuthoredValidationValid
	}
}

func agentSoulFrontmatterPayload(front soulpkg.Frontmatter) AgentSoulFrontmatterPayload {
	return AgentSoulFrontmatterPayload{
		Version:       strings.TrimSpace(front.Version),
		Role:          strings.TrimSpace(front.Role),
		Tone:          normalizeAuthoredStrings(front.Tone),
		Principles:    normalizeAuthoredStrings(front.Principles),
		Constraints:   normalizeAuthoredStrings(front.Constraints),
		Collaboration: normalizeAuthoredStrings(front.Collaboration),
		MemoryPolicy:  normalizeAuthoredStrings(front.MemoryPolicy),
		Tags:          normalizeAuthoredStrings(front.Tags),
	}
}

func soulDiagnosticsPayload(items []soulpkg.Diagnostic) []AuthoredContextDiagnosticPayload {
	if len(items) == 0 {
		return nil
	}
	payload := make([]AuthoredContextDiagnosticPayload, 0, len(items))
	for _, item := range items {
		payload = append(payload, AuthoredContextDiagnosticPayload{
			Code:         strings.TrimSpace(item.Code),
			Severity:     AuthoredDiagnosticError,
			Message:      strings.TrimSpace(item.Message),
			Field:        strings.TrimSpace(item.Field),
			Section:      strings.TrimSpace(item.Section),
			SourcePath:   strings.TrimSpace(item.SourcePath),
			Line:         item.Line,
			Column:       item.Column,
			OwnerSurface: ownerSurfaceForAuthoredDiagnostic(item.Code, item.Field, item.Section),
		})
	}
	return payload
}

func heartbeatDiagnosticsPayload(
	items []heartbeatpkg.Diagnostic,
) ([]AuthoredContextDiagnosticPayload, error) {
	if len(items) == 0 {
		return nil, nil
	}
	payload := make([]AuthoredContextDiagnosticPayload, 0, len(items))
	for _, item := range items {
		severity := AuthoredDiagnosticSeverity(strings.TrimSpace(item.Severity))
		if severity == "" {
			severity = AuthoredDiagnosticError
		}
		if err := severity.Validate(); err != nil {
			return nil, err
		}
		payload = append(payload, AuthoredContextDiagnosticPayload{
			Code:         strings.TrimSpace(item.Code),
			Severity:     severity,
			Message:      strings.TrimSpace(item.Message),
			Field:        strings.TrimSpace(item.Field),
			Section:      strings.TrimSpace(item.Section),
			SourcePath:   strings.TrimSpace(item.SourcePath),
			Line:         item.Line,
			Column:       item.Column,
			OwnerSurface: ownerSurfaceForAuthoredDiagnostic(item.Code, item.Field, item.Section),
		})
	}
	return payload, nil
}

func decodeSoulRevisionDiagnostics(raw json.RawMessage) ([]AuthoredContextDiagnosticPayload, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var diagnostics []soulpkg.Diagnostic
	if err := json.Unmarshal(raw, &diagnostics); err != nil {
		return nil, fmt.Errorf("decode soul revision diagnostics: %w", err)
	}
	return soulDiagnosticsPayload(diagnostics), nil
}

func heartbeatFrontmatterPayload(front heartbeatpkg.Frontmatter) HeartbeatFrontmatterPayload {
	return HeartbeatFrontmatterPayload{
		Version: front.Version,
		Enabled: front.Enabled,
		Summary: strings.TrimSpace(front.Summary),
		Preferences: HeartbeatFrontmatterPreferencesPayload{
			MinInterval:  strings.TrimSpace(front.Preferences.MinInterval),
			ActiveHours:  heartbeatTimeWindowsPayload(front.Preferences.ActiveHours),
			QuietWindows: heartbeatTimeWindowsPayload(front.Preferences.QuietWindows),
		},
		Context: HeartbeatContextProjectionPayload{
			Include: normalizeAuthoredStrings(front.Context.Include),
		},
	}
}

func heartbeatPreferencesPayload(preferences heartbeatpkg.Preferences) HeartbeatPreferencesPayload {
	return HeartbeatPreferencesPayload{
		MinInterval:  preferences.MinInterval.String(),
		ActiveHours:  heartbeatTimeWindowsPayload(preferences.ActiveHours),
		QuietWindows: heartbeatTimeWindowsPayload(preferences.QuietWindows),
		Context: HeartbeatContextProjectionPayload{
			Include: normalizeAuthoredStrings(preferences.Context.Include),
		},
	}
}

func heartbeatConfigProvenancePayload(
	provenance heartbeatpkg.ConfigProvenance,
) HeartbeatConfigProvenancePayload {
	return HeartbeatConfigProvenancePayload{
		Digest: strings.TrimSpace(provenance.Digest),
		Subset: HeartbeatConfigSubsetPayload{
			Enabled:                      provenance.Subset.Enabled,
			MaxBodyBytes:                 provenance.Subset.MaxBodyBytes,
			ContextProjectionBytes:       provenance.Subset.ContextProjectionBytes,
			MinInterval:                  strings.TrimSpace(provenance.Subset.MinInterval),
			DefaultInterval:              strings.TrimSpace(provenance.Subset.DefaultInterval),
			WakeCooldown:                 strings.TrimSpace(provenance.Subset.WakeCooldown),
			MaxWakesPerCycle:             provenance.Subset.MaxWakesPerCycle,
			ActiveSessionOnly:            provenance.Subset.ActiveSessionOnly,
			AllowActiveHoursPreferences:  provenance.Subset.AllowActiveHoursPreferences,
			WakeEventRetention:           strings.TrimSpace(provenance.Subset.WakeEventRetention),
			SessionHealthStaleAfter:      strings.TrimSpace(provenance.Subset.SessionHealthStaleAfter),
			SessionHealthHookMinInterval: strings.TrimSpace(provenance.Subset.SessionHealthHookMinInterval),
		},
	}
}

func heartbeatPromptContributionPayload(
	prompt heartbeatpkg.PromptContribution,
	diagnostics []AuthoredContextDiagnosticPayload,
) HeartbeatPromptContributionPayload {
	return HeartbeatPromptContributionPayload{
		Active:           prompt.Active,
		Digest:           strings.TrimSpace(prompt.Digest),
		ConfigDigest:     strings.TrimSpace(prompt.ConfigDigest),
		SourcePath:       strings.TrimSpace(prompt.SourcePath),
		Summary:          strings.TrimSpace(prompt.Summary),
		GuidanceMarkdown: prompt.GuidanceMarkdown,
		Preferences:      heartbeatPreferencesPayload(prompt.Preferences),
		Truncated:        prompt.Truncated,
		MaxBytes:         prompt.MaxBytes,
		MaxBodyBytes:     prompt.MaxBodyBytes,
		Diagnostics:      diagnostics,
		Context: HeartbeatContextProjectionPayload{
			Include: normalizeAuthoredStrings(prompt.Context.Include),
		},
	}
}

func heartbeatTimeWindowsPayload(items []heartbeatpkg.TimeWindow) []HeartbeatTimeWindowPayload {
	if len(items) == 0 {
		return nil
	}
	payload := make([]HeartbeatTimeWindowPayload, 0, len(items))
	for _, item := range items {
		payload = append(payload, HeartbeatTimeWindowPayload{
			Timezone: strings.TrimSpace(item.Timezone),
			Start:    strings.TrimSpace(item.Start),
			End:      strings.TrimSpace(item.End),
		})
	}
	return payload
}

func ownerSurfaceForAuthoredDiagnostic(code string, field string, section string) string {
	key := strings.ToLower(strings.TrimSpace(firstAuthoredNonEmpty(field, section, code)))
	switch key {
	case "tools", "tool", "capabilities", "capability":
		return "capabilities.toml"
	case "provider", "model", "permission", "permissions":
		return "AGENT.md"
	case "task", "tasks", "lease", "claim", "claim_token", "heartbeat":
		return "task runtime"
	case "scheduler", "wake", "network":
		return "config"
	default:
		return ""
	}
}

func authoredActorPtr(kind string, ref string) *AuthoredContextActorPayload {
	trimmedKind := strings.TrimSpace(kind)
	trimmedRef := strings.TrimSpace(ref)
	if trimmedKind == "" && trimmedRef == "" {
		return nil
	}
	return &AuthoredContextActorPayload{
		Kind: trimmedKind,
		Ref:  trimmedRef,
	}
}

func validAgentSoulRevisionAction(action AgentSoulRevisionAction) bool {
	switch action {
	case AgentSoulRevisionPut, AgentSoulRevisionDelete, AgentSoulRevisionRollback:
		return true
	default:
		return false
	}
}

func validHeartbeatRevisionOperation(operation HeartbeatRevisionOperation) bool {
	switch operation {
	case HeartbeatRevisionWrite, HeartbeatRevisionDelete, HeartbeatRevisionRollback:
		return true
	default:
		return false
	}
}

func validHeartbeatActorKind(kind HeartbeatActorKind) bool {
	switch kind {
	case HeartbeatActorUser, HeartbeatActorAgent, HeartbeatActorExtension, HeartbeatActorSystem:
		return true
	default:
		return false
	}
}

func validSessionHealthIneligibilityReason(reason SessionHealthIneligibilityReason) bool {
	switch reason {
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

func validHeartbeatWakeSource(source HeartbeatWakeSource) bool {
	switch source {
	case HeartbeatWakeSourceScheduler, HeartbeatWakeSourceManual, HeartbeatWakeSourceHarnessReentry:
		return true
	default:
		return false
	}
}

func validHeartbeatWakeResult(result HeartbeatWakeResult) bool {
	switch result {
	case HeartbeatWakeResultSent,
		HeartbeatWakeResultSkipped,
		HeartbeatWakeResultCoalesced,
		HeartbeatWakeResultRateLimited,
		HeartbeatWakeResultFailed:
		return true
	default:
		return false
	}
}

func authoredTimePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func normalizeAuthoredStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		normalized = append(normalized, strings.TrimSpace(value))
	}
	return normalized
}

func firstAuthoredNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
