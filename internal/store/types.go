package store

import (
	"errors"
	"fmt"
	"strings"
	"time"
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
	switch {
	case strings.TrimSpace(e.TurnID) == "":
		return errors.New("store: event turn id is required")
	case strings.TrimSpace(e.Type) == "":
		return errors.New("store: event type is required")
	case strings.TrimSpace(e.AgentName) == "":
		return errors.New("store: event agent name is required")
	default:
		return nil
	}
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
	if q.Limit < 0 {
		return fmt.Errorf("store: invalid event limit %d", q.Limit)
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
	if strings.TrimSpace(u.TurnID) == "" {
		return errors.New("store: token usage turn id is required")
	}
	return nil
}

// SessionInfo is the canonical session index row stored in the global database.
type SessionInfo struct {
	ID           string
	Name         string
	AgentName    string
	WorkspaceID  string
	SessionType  string
	State        string
	ACPSessionID *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Validate ensures the session record contains the required fields.
func (s SessionInfo) Validate() error {
	switch {
	case strings.TrimSpace(s.ID) == "":
		return errors.New("store: session id is required")
	case strings.TrimSpace(s.AgentName) == "":
		return errors.New("store: session agent name is required")
	case strings.TrimSpace(s.WorkspaceID) == "":
		return errors.New("store: session workspace id is required")
	case strings.TrimSpace(s.State) == "":
		return errors.New("store: session state is required")
	default:
		return nil
	}
}

// SessionListQuery filters global session index queries.
type SessionListQuery struct {
	State     string
	AgentName string
	Limit     int
}

// Validate ensures the query uses sane bounds.
func (q SessionListQuery) Validate() error {
	if q.Limit < 0 {
		return fmt.Errorf("store: invalid session limit %d", q.Limit)
	}
	return nil
}

// SessionStateUpdate updates only the stateful fields of an indexed session.
type SessionStateUpdate struct {
	ID           string
	State        string
	ACPSessionID *string
	UpdatedAt    time.Time
}

// Validate ensures the update contains the required fields.
func (u SessionStateUpdate) Validate() error {
	switch {
	case strings.TrimSpace(u.ID) == "":
		return errors.New("store: session update id is required")
	case strings.TrimSpace(u.State) == "":
		return errors.New("store: session update state is required")
	default:
		return nil
	}
}

// EventSummary is the global, cross-session observability record for one event.
type EventSummary struct {
	ID        string
	SessionID string
	Type      string
	AgentName string
	Summary   string
	Timestamp time.Time
}

// Validate ensures the summary contains the required identifying fields.
func (s EventSummary) Validate() error {
	switch {
	case strings.TrimSpace(s.SessionID) == "":
		return errors.New("store: event summary session id is required")
	case strings.TrimSpace(s.Type) == "":
		return errors.New("store: event summary type is required")
	case strings.TrimSpace(s.AgentName) == "":
		return errors.New("store: event summary agent name is required")
	default:
		return nil
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
	if q.Limit < 0 {
		return fmt.Errorf("store: invalid event summary limit %d", q.Limit)
	}
	return nil
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
	switch {
	case strings.TrimSpace(u.SessionID) == "":
		return errors.New("store: token stats session id is required")
	case strings.TrimSpace(u.AgentName) == "":
		return errors.New("store: token stats agent name is required")
	default:
		return nil
	}
}

// TokenStatsQuery filters token aggregation lookups.
type TokenStatsQuery struct {
	SessionID string
	AgentName string
	Limit     int
}

// Validate ensures the query uses sane bounds.
func (q TokenStatsQuery) Validate() error {
	if q.Limit < 0 {
		return fmt.Errorf("store: invalid token stats limit %d", q.Limit)
	}
	return nil
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
	switch {
	case strings.TrimSpace(e.SessionID) == "":
		return errors.New("store: permission log session id is required")
	case strings.TrimSpace(e.AgentName) == "":
		return errors.New("store: permission log agent name is required")
	case strings.TrimSpace(e.Action) == "":
		return errors.New("store: permission log action is required")
	case strings.TrimSpace(e.Resource) == "":
		return errors.New("store: permission log resource is required")
	case strings.TrimSpace(e.Decision) == "":
		return errors.New("store: permission log decision is required")
	case strings.TrimSpace(e.PolicyUsed) == "":
		return errors.New("store: permission log policy is required")
	default:
		return nil
	}
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
	if q.Limit < 0 {
		return fmt.Errorf("store: invalid permission log limit %d", q.Limit)
	}
	return nil
}

// ReconcileResult reports which sessions were indexed or marked orphaned.
type ReconcileResult struct {
	Indexed  []string
	Orphaned []string
}

// SessionMeta is the atomically-written session metadata document.
type SessionMeta struct {
	ID           string    `json:"id"`
	Name         string    `json:"name,omitempty"`
	AgentName    string    `json:"agent_name"`
	WorkspaceID  string    `json:"workspace_id,omitempty"`
	SessionType  string    `json:"session_type,omitempty"`
	State        string    `json:"state"`
	ACPSessionID *string   `json:"acp_session_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Validate ensures the metadata file remains aligned with the session index schema.
func (m SessionMeta) Validate() error {
	switch {
	case strings.TrimSpace(m.ID) == "":
		return errors.New("store: session id is required")
	case strings.TrimSpace(m.AgentName) == "":
		return errors.New("store: session agent name is required")
	case strings.TrimSpace(m.WorkspaceID) == "":
		return errors.New("store: session workspace id is required")
	case strings.TrimSpace(m.State) == "":
		return errors.New("store: session state is required")
	default:
		return nil
	}
}
