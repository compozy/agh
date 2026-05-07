package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

const (
	// NetworkSurfaceThread stores a public thread conversation container.
	NetworkSurfaceThread = "thread"
	// NetworkSurfaceDirect stores a two-party direct-room conversation container.
	NetworkSurfaceDirect = "direct"

	// NetworkKindGreet stores a presence announcement.
	NetworkKindGreet = "greet"
	// NetworkKindWhois stores a peer identity request or response.
	NetworkKindWhois = "whois"
	// NetworkKindSay stores a text conversation message.
	NetworkKindSay = "say"
	// NetworkKindCapability stores a capability transfer message.
	NetworkKindCapability = "capability"
	// NetworkKindReceipt stores an admission receipt message.
	NetworkKindReceipt = "receipt"
	// NetworkKindTrace stores a work lifecycle trace message.
	NetworkKindTrace = "trace"

	// NetworkWorkStateSubmitted is the initial work state.
	NetworkWorkStateSubmitted = "submitted"
	// NetworkWorkStateWorking marks active work.
	NetworkWorkStateWorking = "working"
	// NetworkWorkStateNeedsInput marks blocked work awaiting input.
	NetworkWorkStateNeedsInput = "needs_input"
	// NetworkWorkStateCompleted marks successful terminal work.
	NetworkWorkStateCompleted = "completed"
	// NetworkWorkStateFailed marks failed terminal work.
	NetworkWorkStateFailed = "failed"
	// NetworkWorkStateCanceled marks canceled terminal work.
	NetworkWorkStateCanceled = "canceled"
)

var (
	networkThreadIDPattern = regexp.MustCompile(`^thread_[a-z0-9][a-z0-9_-]{2,95}$`)
	networkDirectIDPattern = regexp.MustCompile(`^direct_[a-f0-9]{32}$`)
	networkPeerIDPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
)

// SessionEvent is a persisted event row for a single AGH session.
type SessionEvent struct {
	ID        string
	SessionID string
	Sequence  int64
	TurnID    string
	Type      string
	AgentName string
	Content   string
	Timestamp time.Time
}

// SessionLedgerRecord carries the durable session evidence needed to materialize
// a forensic session ledger after the live event store has been closed.
type SessionLedgerRecord struct {
	SessionID    string
	WorkspaceID  string
	AgentName    string
	SessionType  string
	EventsDBPath string
	Lineage      *SessionLineage
	StartedAt    time.Time
	EndedAt      time.Time
}

// EventCorrelation carries the canonical cross-surface correlation keys for
// session and observability events.
type EventCorrelation struct {
	TaskID               string     `json:"task_id,omitempty"`
	RunID                string     `json:"run_id,omitempty"`
	WorkflowID           string     `json:"workflow_id,omitempty"`
	ClaimTokenHash       string     `json:"claim_token_hash,omitempty"`
	LeaseUntil           *time.Time `json:"lease_until,omitempty"`
	CoordinatorSessionID string     `json:"coordinator_session_id,omitempty"`
	SchedulerReason      string     `json:"scheduler_reason,omitempty"`
	HookEvent            string     `json:"hook_event,omitempty"`
	HookName             string     `json:"hook_name,omitempty"`
	ActorKind            string     `json:"actor_kind,omitempty"`
	ActorID              string     `json:"actor_id,omitempty"`
	ReleaseReason        string     `json:"release_reason,omitempty"`
}

// Normalize trims string fields and canonicalizes timestamps.
func (c EventCorrelation) Normalize() EventCorrelation {
	normalized := EventCorrelation{
		TaskID:               strings.TrimSpace(c.TaskID),
		RunID:                strings.TrimSpace(c.RunID),
		WorkflowID:           strings.TrimSpace(c.WorkflowID),
		ClaimTokenHash:       strings.TrimSpace(c.ClaimTokenHash),
		CoordinatorSessionID: strings.TrimSpace(c.CoordinatorSessionID),
		SchedulerReason:      strings.TrimSpace(c.SchedulerReason),
		HookEvent:            strings.TrimSpace(c.HookEvent),
		HookName:             strings.TrimSpace(c.HookName),
		ActorKind:            strings.TrimSpace(c.ActorKind),
		ActorID:              strings.TrimSpace(c.ActorID),
		ReleaseReason:        strings.TrimSpace(c.ReleaseReason),
	}
	normalized.LeaseUntil = cloneNormalizedTimestamp(c.LeaseUntil)
	return normalized
}

// IsZero reports whether the correlation payload carries any fields.
func (c EventCorrelation) IsZero() bool {
	normalized := c.Normalize()
	return normalized.TaskID == "" &&
		normalized.RunID == "" &&
		normalized.WorkflowID == "" &&
		normalized.ClaimTokenHash == "" &&
		normalized.LeaseUntil == nil &&
		normalized.CoordinatorSessionID == "" &&
		normalized.SchedulerReason == "" &&
		normalized.HookEvent == "" &&
		normalized.HookName == "" &&
		normalized.ActorKind == "" &&
		normalized.ActorID == "" &&
		normalized.ReleaseReason == ""
}

// Validate ensures the event has the required fields for persistence.
func (e SessionEvent) Validate() error {
	if err := requireField(e.TurnID, "event turn id"); err != nil {
		return err
	}
	if err := requireField(e.Type, "event type"); err != nil {
		return err
	}
	if err := requireField(e.AgentName, "event agent name"); err != nil {
		return err
	}
	return nil
}

// EventQuery filters per-session events while preserving follow-friendly ordering.
type EventQuery struct {
	Type          string
	AgentName     string
	TurnID        string
	Since         time.Time
	Limit         int
	AfterSequence int64
}

// Validate ensures the query is internally consistent.
func (q EventQuery) Validate() error {
	if err := requirePositiveLimit(q.Limit, "event limit"); err != nil {
		return err
	}
	if q.AfterSequence < 0 {
		return fmt.Errorf("store: invalid event after sequence %d", q.AfterSequence)
	}
	return nil
}

// TurnHistory groups ordered events by their turn identifier.
type TurnHistory struct {
	TurnID string
	Events []SessionEvent
}

// HookRunQuery filters persisted per-session hook run records.
type HookRunQuery struct {
	SessionID string
	Event     string
	Outcome   hookspkg.HookRunOutcome
	Since     time.Time
	Limit     int
}

// Validate ensures the query uses sane bounds.
func (q HookRunQuery) Validate() error {
	if q.Outcome != "" {
		if err := q.Outcome.Validate(); err != nil {
			return err
		}
	}
	return requirePositiveLimit(q.Limit, "hook run limit")
}

// TokenUsage captures per-turn usage data reported by an ACP provider.
type TokenUsage struct {
	TurnID           string
	InputTokens      *int64
	OutputTokens     *int64
	TotalTokens      *int64
	ThoughtTokens    *int64
	CacheReadTokens  *int64
	CacheWriteTokens *int64
	ContextUsed      *int64
	ContextSize      *int64
	CostAmount       *float64
	CostCurrency     *string
	Timestamp        time.Time
}

// Validate ensures the usage payload has the required fields.
func (u TokenUsage) Validate() error {
	return requireField(u.TurnID, "token usage turn id")
}

// StopReason classifies why a session ended.
type StopReason string

const (
	StopCompleted      StopReason = "completed"
	StopUserCanceled   StopReason = "user_canceled"
	StopMaxIterations  StopReason = "max_iterations"
	StopLoopDetected   StopReason = "loop_detected"
	StopTimeout        StopReason = "timeout"
	StopBudgetExceeded StopReason = "budget_exceeded"
	StopError          StopReason = "error"
	StopAgentCrashed   StopReason = "agent_crashed"
	StopHookStopped    StopReason = "hook_stopped"
	StopShutdown       StopReason = "shutdown"
)

// ValidStopReason reports whether r is a supported stop reason enum member.
func ValidStopReason(r StopReason) bool {
	switch r {
	case StopCompleted,
		StopUserCanceled,
		StopMaxIterations,
		StopLoopDetected,
		StopTimeout,
		StopBudgetExceeded,
		StopError,
		StopAgentCrashed,
		StopHookStopped,
		StopShutdown:
		return true
	default:
		return false
	}
}

// SessionInfo is the canonical session index row stored in the global database.
type SessionInfo struct {
	ID               string
	Name             string
	AgentName        string
	Provider         string
	WorkspaceID      string
	Channel          string
	SessionType      string
	Lineage          *SessionLineage
	State            string
	ACPSessionID     *string
	StopReason       StopReason
	StopDetail       string
	Failure          *SessionFailure
	Liveness         *SessionLivenessMeta
	Sandbox          *SessionSandboxMeta
	SoulSnapshotID   string
	SoulDigest       string
	ParentSoulDigest string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Validate ensures the session record contains the required fields.
func (s SessionInfo) Validate() error {
	if err := requireField(s.ID, "session id"); err != nil {
		return err
	}
	if err := requireField(s.AgentName, "session agent name"); err != nil {
		return err
	}
	if err := requireField(s.WorkspaceID, "session workspace id"); err != nil {
		return err
	}
	if err := requireField(s.State, "session state"); err != nil {
		return err
	}
	if err := ValidateSessionLineage(s.ID, s.Lineage); err != nil {
		return err
	}
	if err := s.Liveness.Validate(); err != nil {
		return err
	}
	if err := validateSessionSoulProvenance(s.SoulSnapshotID, s.SoulDigest); err != nil {
		return err
	}
	if s.Failure != nil {
		if err := s.Failure.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// SessionListQuery filters global session index queries.
type SessionListQuery struct {
	State           string
	AgentName       string
	SessionType     string
	ParentSessionID string
	RootSessionID   string
	SpawnRole       string
	Limit           int
}

// Validate ensures the query uses sane bounds.
func (q SessionListQuery) Validate() error {
	return requirePositiveLimit(q.Limit, "session limit")
}

// SessionStateUpdate updates only the stateful fields of an indexed session.
type SessionStateUpdate struct {
	ID            string
	State         string
	ACPSessionID  *string
	StopReasonSet bool
	StopReason    *string
	StopDetail    string
	FailureSet    bool
	Failure       *SessionFailure
	Liveness      *SessionLivenessMeta
	Sandbox       *SessionSandboxMeta
	UpdatedAt     time.Time
}

// SessionSoulSnapshotUpdate updates the Soul provenance attached to a session.
type SessionSoulSnapshotUpdate struct {
	ID               string
	SoulSnapshotID   string
	SoulDigest       string
	ParentSoulDigest string
	UpdatedAt        time.Time
}

// Validate ensures session Soul provenance is internally consistent.
func (u SessionSoulSnapshotUpdate) Validate() error {
	if err := requireField(u.ID, "session soul update id"); err != nil {
		return err
	}
	return validateSessionSoulProvenance(u.SoulSnapshotID, u.SoulDigest)
}

// Validate ensures the update contains the required fields.
func (u SessionStateUpdate) Validate() error {
	if err := requireField(u.ID, "session update id"); err != nil {
		return err
	}
	if err := requireField(u.State, "session update state"); err != nil {
		return err
	}
	if err := u.Liveness.Validate(); err != nil {
		return err
	}
	if u.Failure != nil {
		if err := u.Failure.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func validateSessionSoulProvenance(snapshotID string, digest string) error {
	hasSnapshotID := strings.TrimSpace(snapshotID) != ""
	hasDigest := strings.TrimSpace(digest) != ""
	if hasSnapshotID && !hasDigest {
		return errors.New("store: session soul digest is required when soul snapshot id is set")
	}
	return nil
}

// EventSummary is the global, cross-session observability record for one event.
type EventSummary struct {
	ID        string
	SessionID string
	Sequence  int64
	Type      string
	AgentName string
	Content   json.RawMessage
	EventCorrelation
	ParentSessionID string
	RootSessionID   string
	SpawnDepth      int
	Summary         string
	Timestamp       time.Time
}

// Validate ensures the summary contains the required identifying fields.
func (s EventSummary) Validate() error {
	eventType := strings.TrimSpace(s.Type)
	if err := requireField(eventType, "event summary type"); err != nil {
		return err
	}
	if eventSummaryAllowsGlobalScope(eventType) {
		return nil
	}
	if err := requireField(s.SessionID, "event summary session id"); err != nil {
		return err
	}
	if err := requireField(s.AgentName, "event summary agent name"); err != nil {
		return err
	}
	return nil
}

func cloneNormalizedTimestamp(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

func eventSummaryAllowsGlobalScope(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case "settings.changed",
		"skills.shadow",
		"skills.load_failed",
		"hook.dispatch.start",
		"hook.dispatch.complete",
		"memory.provider.collision":
		return true
	default:
		return false
	}
}

// EventSummaryQuery filters global event summary queries.
type EventSummaryQuery struct {
	SessionID string
	AgentName string
	Type      string
	Since     time.Time
	Limit     int
}

// Validate ensures the query uses sane bounds.
func (q EventSummaryQuery) Validate() error {
	return requirePositiveLimit(q.Limit, "event summary limit")
}

// ObservabilityRetentionSweepResult reports how many global observability rows
// were deleted by one retention sweep.
type ObservabilityRetentionSweepResult struct {
	CutoffAt              time.Time
	DeletedEventSummaries int64
	DeletedTokenStats     int64
	DeletedPermissionLogs int64
}

// TokenStats is the aggregated usage record for a session in the global database.
type TokenStats struct {
	ID           string
	SessionID    string
	AgentName    string
	InputTokens  *int64
	OutputTokens *int64
	TotalTokens  *int64
	TotalCost    *float64
	CostCurrency *string
	TurnCount    int64
	UpdatedAt    time.Time
}

// TokenStatsUpdate adds one or more turns of usage into a session aggregate.
type TokenStatsUpdate struct {
	SessionID    string
	AgentName    string
	InputTokens  *int64
	OutputTokens *int64
	TotalTokens  *int64
	CostAmount   *float64
	CostCurrency *string
	Turns        int64
	UpdatedAt    time.Time
}

// Validate ensures the aggregate update contains the required identifying fields.
func (u TokenStatsUpdate) Validate() error {
	if err := requireField(u.SessionID, "token stats session id"); err != nil {
		return err
	}
	if err := requireField(u.AgentName, "token stats agent name"); err != nil {
		return err
	}
	return nil
}

// TokenStatsQuery filters token aggregation lookups.
type TokenStatsQuery struct {
	SessionID string
	AgentName string
	Limit     int
}

// Validate ensures the query uses sane bounds.
func (q TokenStatsQuery) Validate() error {
	return requirePositiveLimit(q.Limit, "token stats limit")
}

// PermissionLogEntry is an audit log entry for a daemon permission decision.
type PermissionLogEntry struct {
	ID         string
	SessionID  string
	AgentName  string
	Action     string
	Resource   string
	Decision   string
	PolicyUsed string
	Timestamp  time.Time
}

// Validate ensures the permission audit entry is complete.
func (e PermissionLogEntry) Validate() error {
	if err := requireField(e.SessionID, "permission log session id"); err != nil {
		return err
	}
	if err := requireField(e.AgentName, "permission log agent name"); err != nil {
		return err
	}
	if err := requireField(e.Action, "permission log action"); err != nil {
		return err
	}
	if err := requireField(e.Resource, "permission log resource"); err != nil {
		return err
	}
	if err := requireField(e.Decision, "permission log decision"); err != nil {
		return err
	}
	if err := requireField(e.PolicyUsed, "permission log policy"); err != nil {
		return err
	}
	return nil
}

// PermissionLogQuery filters permission audit queries.
type PermissionLogQuery struct {
	SessionID string
	AgentName string
	Decision  string
	Since     time.Time
	Limit     int
}

// Validate ensures the query uses sane bounds.
func (q PermissionLogQuery) Validate() error {
	return requirePositiveLimit(q.Limit, "permission log limit")
}

// NetworkAuditEntry is an audit row for one network message event.
type NetworkAuditEntry struct {
	ID        string
	SessionID string
	Direction string
	Kind      string
	Channel   string
	Surface   string
	ThreadID  string
	DirectID  string
	WorkID    string
	PeerFrom  string
	PeerTo    string
	MessageID string
	Reason    string
	Size      int
	Timestamp time.Time
}

// Validate ensures the network audit entry is complete and internally consistent.
func (e NetworkAuditEntry) Validate() error {
	if err := requireField(e.SessionID, "network audit session id"); err != nil {
		return err
	}
	if err := requireField(e.Direction, "network audit direction"); err != nil {
		return err
	}
	direction := strings.TrimSpace(e.Direction)
	switch direction {
	case "sent", "received", "rejected", "delivered":
	default:
		return fmt.Errorf(
			"store: network audit direction must be one of %q, %q, %q, %q: %q",
			"sent",
			"received",
			"rejected",
			"delivered",
			e.Direction,
		)
	}
	if direction != e.Direction {
		return fmt.Errorf("store: network audit direction must not contain surrounding whitespace: %q", e.Direction)
	}
	if err := requireField(e.Kind, "network audit kind"); err != nil {
		return err
	}
	if err := requireField(e.Channel, "network audit channel"); err != nil {
		return err
	}
	if err := requireField(e.PeerFrom, "network audit peer_from"); err != nil {
		return err
	}
	if err := validateOptionalNetworkConversation(
		e.Surface,
		e.ThreadID,
		e.DirectID,
		"network audit conversation",
	); err != nil {
		return err
	}
	if strings.TrimSpace(e.WorkID) != "" {
		if err := validateNetworkConversationID(e.WorkID, "work_id"); err != nil {
			return err
		}
	}
	if err := requireField(e.MessageID, "network audit message id"); err != nil {
		return err
	}
	if e.Size < 0 {
		return fmt.Errorf("store: network audit size must be zero or positive: %d", e.Size)
	}
	if direction == "rejected" && strings.TrimSpace(e.Reason) == "" {
		return fmt.Errorf("store: network audit reason is required when direction is %q", e.Direction)
	}
	if networkAuditEntryContainsRawClaimToken(e) {
		return fmt.Errorf("store: network audit entry contains raw claim_token material")
	}
	return nil
}

// NetworkAuditQuery filters network audit lookups.
type NetworkAuditQuery struct {
	SessionID string
	Direction string
	Kind      string
	Channel   string
	Surface   string
	ThreadID  string
	DirectID  string
	WorkID    string
	MessageID string
	Since     time.Time
	Limit     int
}

// Validate ensures the query uses sane bounds.
func (q NetworkAuditQuery) Validate() error {
	return requirePositiveLimit(q.Limit, "network audit limit")
}

// NetworkChannelEntry stores durable channel metadata for the operator-facing
// network workspace.
type NetworkChannelEntry struct {
	Channel     string
	WorkspaceID string
	Purpose     string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate ensures the persisted channel metadata is complete.
func (e NetworkChannelEntry) Validate() error {
	if err := requireField(e.Channel, "network channel channel"); err != nil {
		return err
	}
	if err := requireField(e.WorkspaceID, "network channel workspace_id"); err != nil {
		return err
	}
	if err := requireField(e.Purpose, "network channel purpose"); err != nil {
		return err
	}
	return nil
}

// NetworkChannelQuery filters persisted network channel metadata lookups.
type NetworkChannelQuery struct {
	Channel     string
	WorkspaceID string
	Limit       int
}

// Validate ensures the query uses sane bounds.
func (q NetworkChannelQuery) Validate() error {
	return requirePositiveLimit(q.Limit, "network channel limit")
}

// NetworkConversationRef identifies one persisted network conversation container.
type NetworkConversationRef struct {
	Channel  string
	Surface  string
	ThreadID string
	DirectID string
}

// Validate ensures the reference identifies exactly one supported container.
func (r NetworkConversationRef) Validate() error {
	if err := requireField(r.Channel, "network conversation channel"); err != nil {
		return err
	}
	surface := strings.TrimSpace(r.Surface)
	switch surface {
	case NetworkSurfaceThread:
		if err := validateNetworkConversationID(r.ThreadID, "thread_id"); err != nil {
			return err
		}
		if strings.TrimSpace(r.DirectID) != "" {
			return fmt.Errorf("store: network conversation direct_id must be empty for surface %q", surface)
		}
	case NetworkSurfaceDirect:
		if err := validateNetworkConversationID(r.DirectID, "direct_id"); err != nil {
			return err
		}
		if strings.TrimSpace(r.ThreadID) != "" {
			return fmt.Errorf("store: network conversation thread_id must be empty for surface %q", surface)
		}
	default:
		return fmt.Errorf(
			"store: network conversation surface must be one of %q or %q: %q",
			NetworkSurfaceThread,
			NetworkSurfaceDirect,
			r.Surface,
		)
	}
	return nil
}

// ContainerID returns the thread or direct-room id selected by the surface.
func (r NetworkConversationRef) ContainerID() string {
	switch strings.TrimSpace(r.Surface) {
	case NetworkSurfaceThread:
		return strings.TrimSpace(r.ThreadID)
	case NetworkSurfaceDirect:
		return strings.TrimSpace(r.DirectID)
	default:
		return ""
	}
}

// NetworkThreadSummary is the list/detail projection for a public thread.
type NetworkThreadSummary struct {
	Channel            string
	ThreadID           string
	RootMessageID      string
	Title              string
	OpenedByPeerID     string
	OpenedSessionID    string
	OpenedAt           time.Time
	LastActivityAt     time.Time
	MessageCount       int
	ParticipantCount   int
	OpenWorkCount      int
	LastMessagePreview string
}

// Validate ensures the public-thread summary is internally consistent.
func (s NetworkThreadSummary) Validate() error {
	ref := NetworkConversationRef{
		Channel:  s.Channel,
		Surface:  NetworkSurfaceThread,
		ThreadID: s.ThreadID,
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if err := requireField(s.RootMessageID, "network thread root message id"); err != nil {
		return err
	}
	if err := requireField(s.OpenedByPeerID, "network thread opened_by_peer_id"); err != nil {
		return err
	}
	if s.OpenedAt.IsZero() {
		return fmt.Errorf("store: network thread opened_at is required")
	}
	if s.LastActivityAt.IsZero() {
		return fmt.Errorf("store: network thread last_activity_at is required")
	}
	if s.MessageCount < 0 {
		return fmt.Errorf("store: network thread message_count must be zero or positive: %d", s.MessageCount)
	}
	if s.ParticipantCount < 0 {
		return fmt.Errorf("store: network thread participant_count must be zero or positive: %d", s.ParticipantCount)
	}
	if s.OpenWorkCount < 0 {
		return fmt.Errorf("store: network thread open_work_count must be zero or positive: %d", s.OpenWorkCount)
	}
	return nil
}

// NetworkDirectRoomSummary is the list/detail projection for a direct room.
type NetworkDirectRoomSummary struct {
	Channel            string
	DirectID           string
	PeerA              string
	PeerB              string
	OpenedAt           time.Time
	LastActivityAt     time.Time
	MessageCount       int
	OpenWorkCount      int
	LastMessagePreview string
}

// Validate ensures the direct-room summary is internally consistent.
func (s NetworkDirectRoomSummary) Validate() error {
	if err := validateNetworkDirectRoom(s.Channel, s.DirectID, s.PeerA, s.PeerB); err != nil {
		return err
	}
	if s.OpenedAt.IsZero() {
		return fmt.Errorf("store: network direct room opened_at is required")
	}
	if s.LastActivityAt.IsZero() {
		return fmt.Errorf("store: network direct room last_activity_at is required")
	}
	if s.MessageCount < 0 {
		return fmt.Errorf("store: network direct room message_count must be zero or positive: %d", s.MessageCount)
	}
	if s.OpenWorkCount < 0 {
		return fmt.Errorf("store: network direct room open_work_count must be zero or positive: %d", s.OpenWorkCount)
	}
	return nil
}

// NetworkDirectRoomEntry is the write DTO for a direct-room row.
type NetworkDirectRoomEntry struct {
	Channel        string
	DirectID       string
	PeerA          string
	PeerB          string
	OpenedAt       time.Time
	LastActivityAt time.Time
}

// Validate ensures direct-room membership is stable and ordered.
func (e NetworkDirectRoomEntry) Validate() error {
	if err := validateNetworkDirectRoom(e.Channel, e.DirectID, e.PeerA, e.PeerB); err != nil {
		return err
	}
	if e.OpenedAt.IsZero() {
		return fmt.Errorf("store: network direct room opened_at is required")
	}
	if e.LastActivityAt.IsZero() {
		return fmt.Errorf("store: network direct room last_activity_at is required")
	}
	return nil
}

// NetworkWorkEntry stores lifecycle metadata for work inside one conversation.
type NetworkWorkEntry struct {
	WorkID          string
	Channel         string
	Surface         string
	ThreadID        string
	DirectID        string
	OpenedByPeerID  string
	OpenedSessionID string
	TargetPeerID    string
	State           string
	OpenedAt        time.Time
	LastActivityAt  time.Time
	TerminalAt      *time.Time
}

// Validate ensures a work row is bound to exactly one conversation container.
func (e NetworkWorkEntry) Validate() error {
	if err := validateNetworkConversationID(e.WorkID, "work_id"); err != nil {
		return err
	}
	ref := NetworkConversationRef{
		Channel:  e.Channel,
		Surface:  e.Surface,
		ThreadID: e.ThreadID,
		DirectID: e.DirectID,
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if err := requireField(e.OpenedByPeerID, "network work opened_by_peer_id"); err != nil {
		return err
	}
	if err := validateNetworkWorkState(e.State); err != nil {
		return err
	}
	if e.OpenedAt.IsZero() {
		return fmt.Errorf("store: network work opened_at is required")
	}
	if e.LastActivityAt.IsZero() {
		return fmt.Errorf("store: network work last_activity_at is required")
	}
	if e.TerminalAt != nil && !networkWorkStateIsTerminal(e.State) {
		return fmt.Errorf("store: network work terminal_at requires terminal state")
	}
	return nil
}

// NetworkConversationMessage is one persisted network conversation or presence message.
type NetworkConversationMessage struct {
	MessageID   string
	SessionID   string
	Channel     string
	Surface     string
	ThreadID    string
	DirectID    string
	Direction   string
	PeerFrom    string
	PeerTo      string
	Kind        string
	WorkID      string
	ReplyTo     string
	TraceID     string
	CausationID string
	Intent      string
	Text        string
	PreviewText string
	Body        json.RawMessage
	Timestamp   time.Time
}

// NetworkMessageEntry is the persisted network timeline row used by existing store interfaces.
type NetworkMessageEntry = NetworkConversationMessage

// Validate ensures the persisted network message is complete and internally consistent.
func (e NetworkConversationMessage) Validate() error {
	if err := requireField(e.MessageID, "network message id"); err != nil {
		return err
	}
	if err := requireField(e.Channel, "network message channel"); err != nil {
		return err
	}
	if err := requireField(e.Direction, "network message direction"); err != nil {
		return err
	}
	direction := strings.TrimSpace(e.Direction)
	if direction != e.Direction {
		return fmt.Errorf("store: unsupported network message direction %q", e.Direction)
	}
	switch direction {
	case "sent", "received":
	default:
		return fmt.Errorf("store: unsupported network message direction %q", e.Direction)
	}
	if err := requireField(e.PeerFrom, "network message peer_from"); err != nil {
		return err
	}
	if err := requireField(e.Kind, "network message kind"); err != nil {
		return err
	}
	if err := validateNetworkMessageKind(e.Kind); err != nil {
		return err
	}
	if err := e.validateConversationFields(); err != nil {
		return err
	}
	if len(e.Body) == 0 {
		return fmt.Errorf("store: network message body is required")
	}
	if !json.Valid(e.Body) {
		return fmt.Errorf("store: network message body must be valid JSON")
	}
	if networkRawJSONContainsClaimToken(e.Body) {
		return fmt.Errorf("store: network message body contains raw claim_token material")
	}
	if networkConversationMessageContainsRawClaimToken(e) {
		return fmt.Errorf("store: network message entry contains raw claim_token material")
	}
	return nil
}

func (e NetworkConversationMessage) validateConversationFields() error {
	switch strings.TrimSpace(e.Kind) {
	case NetworkKindGreet, NetworkKindWhois:
		if strings.TrimSpace(e.Surface) != "" ||
			strings.TrimSpace(e.ThreadID) != "" ||
			strings.TrimSpace(e.DirectID) != "" ||
			strings.TrimSpace(e.WorkID) != "" {
			return fmt.Errorf("store: network %s message cannot carry conversation or work fields", e.Kind)
		}
		return nil
	case NetworkKindSay:
		if err := (NetworkConversationRef{
			Channel:  e.Channel,
			Surface:  e.Surface,
			ThreadID: e.ThreadID,
			DirectID: e.DirectID,
		}).Validate(); err != nil {
			return err
		}
		if strings.TrimSpace(e.WorkID) != "" {
			return validateNetworkConversationID(e.WorkID, "work_id")
		}
		return nil
	case NetworkKindCapability, NetworkKindReceipt, NetworkKindTrace:
		if err := (NetworkConversationRef{
			Channel:  e.Channel,
			Surface:  e.Surface,
			ThreadID: e.ThreadID,
			DirectID: e.DirectID,
		}).Validate(); err != nil {
			return err
		}
		if err := validateNetworkConversationID(e.WorkID, "work_id"); err != nil {
			return err
		}
		return nil
	default:
		return validateNetworkMessageKind(e.Kind)
	}
}

// NetworkConversationWriteResult reports the transactional write outcome.
type NetworkConversationWriteResult struct {
	MessageID          string
	Duplicate          bool
	ConversationOpened bool
	WorkOpened         bool
	WorkTransitioned   bool
	WorkState          string
	LastActivityAt     time.Time
}

// NetworkMessageQuery filters persisted network timeline lookups.
type NetworkMessageQuery struct {
	SessionID       string
	Channel         string
	PeerID          string
	PeerFrom        string
	PeerTo          string
	Kind            string
	Direction       string
	MessageID       string
	BeforeMessageID string
	AfterMessageID  string
	DirectedOnly    bool
	IncludePresence bool
	Since           time.Time
	Limit           int
}

// NetworkThreadQuery filters public-thread summary lookups.
type NetworkThreadQuery struct {
	Limit int
	After string
}

// Validate ensures the query uses sane bounds.
func (q NetworkThreadQuery) Validate() error {
	return requirePositiveLimit(q.Limit, "network thread limit")
}

// NetworkDirectRoomQuery filters direct-room summary lookups.
type NetworkDirectRoomQuery struct {
	PeerID string
	Limit  int
	After  string
}

// Validate ensures the query uses sane bounds.
func (q NetworkDirectRoomQuery) Validate() error {
	return requirePositiveLimit(q.Limit, "network direct room limit")
}

// NetworkConversationMessageQuery filters conversation message lookups.
type NetworkConversationMessageQuery struct {
	BeforeMessageID string
	AfterMessageID  string
	Kind            string
	WorkID          string
	Limit           int
}

// Validate ensures the query uses sane bounds and one cursor direction.
func (q NetworkConversationMessageQuery) Validate() error {
	if strings.TrimSpace(q.BeforeMessageID) != "" && strings.TrimSpace(q.AfterMessageID) != "" {
		return fmt.Errorf("store: network conversation message query cannot specify both before and after cursors")
	}
	if strings.TrimSpace(q.Kind) != "" {
		if err := validateNetworkMessageKind(q.Kind); err != nil {
			return err
		}
	}
	if strings.TrimSpace(q.WorkID) != "" {
		if err := validateNetworkConversationID(q.WorkID, "work_id"); err != nil {
			return err
		}
	}
	return requirePositiveLimit(q.Limit, "network conversation message limit")
}

// NormalizeNetworkDirectRoomPeers validates and orders a two-party room pair.
func NormalizeNetworkDirectRoomPeers(peerA string, peerB string) (string, string, error) {
	first := strings.TrimSpace(peerA)
	second := strings.TrimSpace(peerB)
	if err := validateNetworkPeerID(first, "peer_a"); err != nil {
		return "", "", err
	}
	if err := validateNetworkPeerID(second, "peer_b"); err != nil {
		return "", "", err
	}
	if first == second {
		return "", "", fmt.Errorf("store: network direct room peers must differ")
	}
	if second < first {
		first, second = second, first
	}
	return first, second, nil
}

// NetworkDirectRoomIdentity derives the stable direct-room id for one ordered peer pair.
func NetworkDirectRoomIdentity(channel string, peerA string, peerB string) (string, string, string, error) {
	trimmedChannel := strings.TrimSpace(channel)
	if err := requireField(trimmedChannel, "network direct room channel"); err != nil {
		return "", "", "", err
	}
	normalizedA, normalizedB, err := NormalizeNetworkDirectRoomPeers(peerA, peerB)
	if err != nil {
		return "", "", "", err
	}
	sum := sha256.Sum256([]byte(
		"agh-network/direct-room/v1\x00" + trimmedChannel + "\x00" + normalizedA + "\x00" + normalizedB,
	))
	return "direct_" + hex.EncodeToString(sum[:])[:32], normalizedA, normalizedB, nil
}

func validateOptionalNetworkConversation(surface string, threadID string, directID string, label string) error {
	if strings.TrimSpace(surface) == "" && strings.TrimSpace(threadID) == "" && strings.TrimSpace(directID) == "" {
		return nil
	}
	if err := (NetworkConversationRef{
		Channel:  "audit",
		Surface:  surface,
		ThreadID: threadID,
		DirectID: directID,
	}).Validate(); err != nil {
		return fmt.Errorf("store: invalid %s: %w", label, err)
	}
	return nil
}

func validateNetworkDirectRoom(channel string, directID string, peerA string, peerB string) error {
	ref := NetworkConversationRef{
		Channel:  channel,
		Surface:  NetworkSurfaceDirect,
		DirectID: directID,
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	normalizedA, normalizedB, err := NormalizeNetworkDirectRoomPeers(peerA, peerB)
	if err != nil {
		return err
	}
	if normalizedA != strings.TrimSpace(peerA) || normalizedB != strings.TrimSpace(peerB) {
		return fmt.Errorf("store: network direct room peers must be stored in lexicographic order")
	}
	return nil
}

func validateNetworkMessageKind(kind string) error {
	switch strings.TrimSpace(kind) {
	case NetworkKindGreet,
		NetworkKindWhois,
		NetworkKindSay,
		NetworkKindCapability,
		NetworkKindReceipt,
		NetworkKindTrace:
		return nil
	default:
		return fmt.Errorf("store: unsupported network message kind %q", kind)
	}
}

func validateNetworkWorkState(state string) error {
	switch strings.TrimSpace(state) {
	case NetworkWorkStateSubmitted,
		NetworkWorkStateWorking,
		NetworkWorkStateNeedsInput,
		NetworkWorkStateCompleted,
		NetworkWorkStateFailed,
		NetworkWorkStateCanceled:
		return nil
	default:
		return fmt.Errorf("store: unsupported network work state %q", state)
	}
}

func networkWorkStateIsTerminal(state string) bool {
	switch strings.TrimSpace(state) {
	case NetworkWorkStateCompleted, NetworkWorkStateFailed, NetworkWorkStateCanceled:
		return true
	default:
		return false
	}
}

func validateNetworkConversationID(id string, field string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return fmt.Errorf("store: network %s is required", field)
	}
	switch field {
	case "thread_id":
		if !networkThreadIDPattern.MatchString(trimmed) {
			return fmt.Errorf("store: invalid network thread_id %q", id)
		}
	case "direct_id":
		if !networkDirectIDPattern.MatchString(trimmed) {
			return fmt.Errorf("store: invalid network direct_id %q", id)
		}
	default:
		if len(trimmed) > 128 || strings.ContainsAny(trimmed, `/\`) || containsControlCharacter(trimmed) {
			return fmt.Errorf("store: invalid network %s %q", field, id)
		}
	}
	return nil
}

func validateNetworkPeerID(peerID string, field string) error {
	trimmed := strings.TrimSpace(peerID)
	if !networkPeerIDPattern.MatchString(trimmed) {
		return fmt.Errorf("store: invalid network %s %q", field, peerID)
	}
	return nil
}

func containsControlCharacter(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

func networkAuditEntryContainsRawClaimToken(entry NetworkAuditEntry) bool {
	values := []string{
		entry.ID,
		entry.SessionID,
		entry.Direction,
		entry.Kind,
		entry.Channel,
		entry.Surface,
		entry.ThreadID,
		entry.DirectID,
		entry.WorkID,
		entry.PeerFrom,
		entry.PeerTo,
		entry.MessageID,
		entry.Reason,
	}
	return slices.ContainsFunc(values, networkStringContainsRawClaimToken)
}

func networkConversationMessageContainsRawClaimToken(entry NetworkConversationMessage) bool {
	values := []string{
		entry.MessageID,
		entry.SessionID,
		entry.Channel,
		entry.Surface,
		entry.ThreadID,
		entry.DirectID,
		entry.Direction,
		entry.PeerFrom,
		entry.PeerTo,
		entry.Kind,
		entry.WorkID,
		entry.ReplyTo,
		entry.TraceID,
		entry.CausationID,
		entry.Intent,
		entry.Text,
		entry.PreviewText,
	}
	return slices.ContainsFunc(values, networkStringContainsRawClaimToken)
}

func networkRawJSONContainsClaimToken(raw json.RawMessage) bool {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return networkStringContainsRawClaimToken(string(raw))
	}
	return networkValueContainsClaimToken("", value)
}

func networkValueContainsClaimToken(key string, value any) bool {
	if networkStringContainsRawClaimToken(key) || networkClaimTokenKeyHasValue(key, value) {
		return true
	}
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return networkStringContainsRawClaimToken(typed)
	case []any:
		for _, item := range typed {
			if networkValueContainsClaimToken("", item) {
				return true
			}
		}
	case map[string]any:
		for nestedKey, nestedValue := range typed {
			if networkValueContainsClaimToken(nestedKey, nestedValue) {
				return true
			}
		}
	}
	return false
}

func networkClaimTokenKeyHasValue(key string, value any) bool {
	normalized := strings.NewReplacer("_", "", "-", "", ".", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	if normalized != "claimtoken" {
		return false
	}
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

func networkStringContainsRawClaimToken(value string) bool {
	return strings.Contains(strings.TrimSpace(value), "agh_claim_")
}

// Validate ensures the query uses sane bounds.
func (q NetworkMessageQuery) Validate() error {
	if strings.TrimSpace(q.BeforeMessageID) != "" && strings.TrimSpace(q.AfterMessageID) != "" {
		return fmt.Errorf("store: network message query cannot specify both before and after cursors")
	}
	return requirePositiveLimit(q.Limit, "network message limit")
}

// ReconcileResult reports which sessions were indexed or marked orphaned.
type ReconcileResult struct {
	Indexed  []string
	Orphaned []string
}

// SessionSandboxMeta is the persisted runtime sandbox state for a session.
type SessionSandboxMeta struct {
	SandboxID             string          `json:"sandbox_id,omitempty"`
	Backend               string          `json:"backend"`
	Profile               string          `json:"profile,omitempty"`
	State                 string          `json:"state,omitempty"`
	InstanceID            string          `json:"instance_id,omitempty"`
	RuntimeRootDir        string          `json:"runtime_root_dir,omitempty"`
	RuntimeAdditionalDirs []string        `json:"runtime_additional_dirs,omitempty"`
	ProviderState         json.RawMessage `json:"provider_state,omitempty"`
	SSHAccessExpiresAt    *time.Time      `json:"ssh_access_expires_at,omitempty"`
	LastSyncAt            *time.Time      `json:"last_sync_at,omitempty"`
	LastSyncError         string          `json:"last_sync_error,omitempty"`
}

// SessionMeta is the atomically-written session metadata document.
type SessionMeta struct {
	ID               string               `json:"id"`
	Name             string               `json:"name,omitempty"`
	AgentName        string               `json:"agent_name"`
	Provider         string               `json:"provider,omitempty"`
	Model            string               `json:"model,omitempty"`
	ReasoningEffort  string               `json:"reasoning_effort,omitempty"`
	WorkspaceID      string               `json:"workspace_id,omitempty"`
	Channel          string               `json:"channel,omitempty"`
	SessionType      string               `json:"session_type,omitempty"`
	Lineage          *SessionLineage      `json:"lineage,omitempty"`
	State            string               `json:"state"`
	StopReason       *StopReason          `json:"stop_reason,omitempty"`
	StopDetail       string               `json:"stop_detail,omitempty"`
	Failure          *SessionFailure      `json:"failure,omitempty"`
	ACPSessionID     *string              `json:"acp_session_id,omitempty"`
	Liveness         *SessionLivenessMeta `json:"liveness,omitempty"`
	Sandbox          *SessionSandboxMeta  `json:"sandbox,omitempty"`
	SoulSnapshotID   string               `json:"soul_snapshot_id,omitempty"`
	SoulDigest       string               `json:"soul_digest,omitempty"`
	ParentSoulDigest string               `json:"parent_soul_digest,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

// Validate ensures the metadata file remains aligned with the session index schema.
func (m SessionMeta) Validate() error {
	if err := requireField(m.ID, "session id"); err != nil {
		return err
	}
	if err := requireField(m.AgentName, "session agent name"); err != nil {
		return err
	}
	if err := requireField(m.WorkspaceID, "session workspace id"); err != nil {
		return err
	}
	if err := requireField(m.State, "session state"); err != nil {
		return err
	}
	if m.StopReason != nil && !ValidStopReason(*m.StopReason) {
		return fmt.Errorf("store: invalid session stop reason %q", *m.StopReason)
	}
	if err := ValidateSessionLineage(m.ID, m.Lineage); err != nil {
		return err
	}
	if m.Failure != nil {
		if err := m.Failure.Validate(); err != nil {
			return err
		}
	}
	if err := m.Liveness.Validate(); err != nil {
		return err
	}
	if err := validateSessionSoulProvenance(m.SoulSnapshotID, m.SoulDigest); err != nil {
		return err
	}
	return nil
}
