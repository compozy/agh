// Package store provides SQLite-backed persistence for AGH session and global state.
package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const (
	// SessionDatabaseName is the filename for per-session event storage.
	SessionDatabaseName = "events.db"
	// GlobalDatabaseName is the filename for the global AGH index database.
	GlobalDatabaseName = "agh.db"
	// SessionMetaName is the filename for quick session metadata lookups.
	SessionMetaName = "meta.json"

	sqliteDriverName       = "sqlite"
	defaultBusyTimeoutMS   = 5000
	defaultMaxOpenConns    = 8
	defaultMaxIdleConns    = 8
	defaultWriteBufferSize = 256
	defaultDrainTimeout    = 5 * time.Second
)

const timestampLayout = "2006-01-02T15:04:05.000000000Z"
const defaultSessionType = "user"

var (
	// ErrClosed reports that a session database no longer accepts writes.
	ErrClosed = errors.New("store: session database closed")
	// ErrDrainTimeout reports that shutdown timed out before queued writes drained.
	ErrDrainTimeout = errors.New("store: writer drain timeout")
)

// EventRecorder captures session events and token usage in the per-session database.
type EventRecorder interface {
	Record(ctx context.Context, event SessionEvent) error
	RecordTokenUsage(ctx context.Context, usage TokenUsage) error
	Query(ctx context.Context, query EventQuery) ([]SessionEvent, error)
	History(ctx context.Context, query EventQuery) ([]TurnHistory, error)
	Close(ctx context.Context) error
}

// SessionRegistry manages global session index records and observability metadata.
type SessionRegistry interface {
	RegisterSession(ctx context.Context, session SessionInfo) error
	UpdateSessionState(ctx context.Context, update SessionStateUpdate) error
	ListSessions(ctx context.Context, query SessionListQuery) ([]SessionInfo, error)
	ReconcileSessions(ctx context.Context, sessions []SessionInfo) (ReconcileResult, error)
	WriteEventSummary(ctx context.Context, summary EventSummary) error
	ListEventSummaries(ctx context.Context, query EventSummaryQuery) ([]EventSummary, error)
	UpdateTokenStats(ctx context.Context, update TokenStatsUpdate) error
	ListTokenStats(ctx context.Context, query TokenStatsQuery) ([]TokenStats, error)
	WritePermissionLog(ctx context.Context, entry PermissionLogEntry) error
	ListPermissionLog(ctx context.Context, query PermissionLogQuery) ([]PermissionLogEntry, error)
	Close(ctx context.Context) error
}

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
	Workspace    string
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
	case strings.TrimSpace(s.Workspace) == "":
		return errors.New("store: session workspace is required")
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
	Workspace    string    `json:"workspace"`
	SessionType  string    `json:"session_type,omitempty"`
	State        string    `json:"state"`
	ACPSessionID *string   `json:"acp_session_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Validate ensures the metadata file remains aligned with the session index schema.
func (m SessionMeta) Validate() error {
	info := SessionInfo(m)
	info.SessionType = normalizeSessionType(info.SessionType)
	return info.Validate()
}

// SessionDBFile returns the canonical events database path for a session directory.
func SessionDBFile(sessionDir string) string {
	return filepath.Join(sessionDir, SessionDatabaseName)
}

// SessionMetaFile returns the canonical metadata file path for a session directory.
func SessionMetaFile(sessionDir string) string {
	return filepath.Join(sessionDir, SessionMetaName)
}

type rowScanner interface {
	Scan(dest ...any) error
}

type clause struct {
	sql string
	arg any
	ok  bool
}

func stringClause(column string, value string) clause {
	value = strings.TrimSpace(value)
	if value == "" {
		return clause{}
	}

	return clause{
		sql: fmt.Sprintf("%s = ?", column),
		arg: value,
		ok:  true,
	}
}

func timeClause(column string, op string, value time.Time) clause {
	if value.IsZero() {
		return clause{}
	}

	return clause{
		sql: fmt.Sprintf("%s %s ?", column, op),
		arg: formatTimestamp(value),
		ok:  true,
	}
}

func int64Clause(column string, op string, value int64) clause {
	if value <= 0 {
		return clause{}
	}

	return clause{
		sql: fmt.Sprintf("%s %s ?", column, op),
		arg: value,
		ok:  true,
	}
}

func normalizeSessionType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultSessionType
	}
	return value
}

func buildClauses(input ...clause) ([]string, []any) {
	where := make([]string, 0, len(input))
	args := make([]any, 0, len(input))

	for _, item := range input {
		if !item.ok {
			continue
		}
		where = append(where, item.sql)
		args = append(args, item.arg)
	}

	return where, args
}

func appendWhere(query string, where []string) string {
	if len(where) == 0 {
		return query
	}
	return query + " WHERE " + strings.Join(where, " AND ")
}

func appendLimit(query string, args []any, limit int) (string, []any) {
	if limit <= 0 {
		return query, args
	}
	return query + " LIMIT ?", append(args, limit)
}

func normalizeTime(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	return value.UTC()
}

func formatTimestamp(value time.Time) string {
	return normalizeTime(value).Format(timestampLayout)
}

func parseTimestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(timestampLayout, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("store: parse timestamp %q: %w", value, err)
	}
	return parsed.UTC(), nil
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableStringPointer(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return strings.TrimSpace(*value)
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableFloat64(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	trimmed := strings.TrimSpace(value.String)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func nullInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}

func nullFloat64(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	v := value.Float64
	return &v
}

func newID(prefix string) string {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		now := time.Now().UTC().UnixNano()
		if strings.TrimSpace(prefix) == "" {
			return fmt.Sprintf("%d", now)
		}
		return fmt.Sprintf("%s-%d", prefix, now)
	}

	if strings.TrimSpace(prefix) == "" {
		return hex.EncodeToString(random[:])
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(random[:]))
}
