package store

import (
	"fmt"
	"strings"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
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
	ID           string
	Name         string
	AgentName    string
	WorkspaceID  string
	Channel      string
	SessionType  string
	State        string
	ACPSessionID *string
	StopReason   StopReason
	StopDetail   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
	return nil
}

// SessionListQuery filters global session index queries.
type SessionListQuery struct {
	State     string
	AgentName string
	Limit     int
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
	UpdatedAt     time.Time
}

// Validate ensures the update contains the required fields.
func (u SessionStateUpdate) Validate() error {
	if err := requireField(u.ID, "session update id"); err != nil {
		return err
	}
	if err := requireField(u.State, "session update state"); err != nil {
		return err
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
	Summary   string
	Timestamp time.Time
}

// Validate ensures the summary contains the required identifying fields.
func (s EventSummary) Validate() error {
	if err := requireField(s.SessionID, "event summary session id"); err != nil {
		return err
	}
	if err := requireField(s.Type, "event summary type"); err != nil {
		return err
	}
	if err := requireField(s.AgentName, "event summary agent name"); err != nil {
		return err
	}
	return nil
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
	if err := requireField(e.MessageID, "network audit message id"); err != nil {
		return err
	}
	if e.Size < 0 {
		return fmt.Errorf("store: network audit size must be zero or positive: %d", e.Size)
	}
	if direction == "rejected" && strings.TrimSpace(e.Reason) == "" {
		return fmt.Errorf("store: network audit reason is required when direction is %q", e.Direction)
	}
	return nil
}

// NetworkAuditQuery filters network audit lookups.
type NetworkAuditQuery struct {
	SessionID string
	Direction string
	Kind      string
	Channel   string
	MessageID string
	Since     time.Time
	Limit     int
}

// Validate ensures the query uses sane bounds.
func (q NetworkAuditQuery) Validate() error {
	return requirePositiveLimit(q.Limit, "network audit limit")
}

// NetworkMessageEntry is one persisted network timeline message.
type NetworkMessageEntry struct {
	MessageID string
	SessionID string
	Channel   string
	PeerFrom  string
	Kind      string
	Intent    string
	Text      string
	Timestamp time.Time
}

// Validate ensures the persisted network message is complete and internally consistent.
func (e NetworkMessageEntry) Validate() error {
	if err := requireField(e.MessageID, "network message id"); err != nil {
		return err
	}
	if err := requireField(e.Channel, "network message channel"); err != nil {
		return err
	}
	if err := requireField(e.PeerFrom, "network message peer_from"); err != nil {
		return err
	}
	if err := requireField(e.Kind, "network message kind"); err != nil {
		return err
	}
	if err := requireField(e.Text, "network message text"); err != nil {
		return err
	}
	return nil
}

// NetworkMessageQuery filters persisted network timeline lookups.
type NetworkMessageQuery struct {
	SessionID string
	Channel   string
	PeerFrom  string
	MessageID string
	Since     time.Time
	Limit     int
}

// Validate ensures the query uses sane bounds.
func (q NetworkMessageQuery) Validate() error {
	return requirePositiveLimit(q.Limit, "network message limit")
}

// ReconcileResult reports which sessions were indexed or marked orphaned.
type ReconcileResult struct {
	Indexed  []string
	Orphaned []string
}

// SessionMeta is the atomically-written session metadata document.
type SessionMeta struct {
	ID           string      `json:"id"`
	Name         string      `json:"name,omitempty"`
	AgentName    string      `json:"agent_name"`
	WorkspaceID  string      `json:"workspace_id,omitempty"`
	Channel      string      `json:"channel,omitempty"`
	SessionType  string      `json:"session_type,omitempty"`
	State        string      `json:"state"`
	StopReason   *StopReason `json:"stop_reason,omitempty"`
	StopDetail   string      `json:"stop_detail,omitempty"`
	ACPSessionID *string     `json:"acp_session_id,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
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
	return nil
}
