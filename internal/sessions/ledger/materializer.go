// Package ledger materializes read-only forensic session ledgers from events.db.
package ledger

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/fileutil"
	"github.com/pedronauck/agh/internal/store"
)

const (
	DefaultUnboundPartition = "_unbound"
	ledgerFileName          = "ledger.jsonl"
	ledgerVersion           = 1
	closeTimeout            = 5 * time.Second
)

var (
	// ErrLedgerExists reports that a materialized ledger already exists with a different checksum.
	ErrLedgerExists = errors.New("sessions/ledger: ledger already exists with different content")
	// ErrInvalidRecord reports a session record that cannot produce a safe forensic path.
	ErrInvalidRecord = errors.New("sessions/ledger: invalid session ledger record")
)

// EventStoreOpener opens the live session event database for read-only projection.
type EventStoreOpener func(ctx context.Context, sessionID string, path string) (store.EventRecorder, error)

// Config controls forensic ledger materialization.
type Config struct {
	RootDir          string
	UnboundPartition string
	OpenEventStore   EventStoreOpener
}

// Materializer projects session events into ledger.jsonl after a session ends.
type Materializer struct {
	rootDir          string
	unboundPartition string
	openEventStore   EventStoreOpener
}

// Result describes one materialization attempt.
type Result struct {
	Path     string
	Checksum string
	Events   int
	Written  bool
}

type ledgerMetaLine struct {
	Type          string `json:"type"`
	Version       int    `json:"version"`
	SessionID     string `json:"session_id"`
	WorkspaceID   string `json:"workspace_id"`
	SpawnParentID string `json:"spawn_parent_id,omitempty"`
	RootSessionID string `json:"root_session_id,omitempty"`
	SpawnDepth    int    `json:"spawn_depth,omitempty"`
	AgentName     string `json:"agent_name,omitempty"`
	SessionType   string `json:"session_type,omitempty"`
	StartedAt     string `json:"started_at,omitempty"`
	EndedAt       string `json:"ended_at,omitempty"`
}

type ledgerEventLine struct {
	Type      string          `json:"type"`
	Version   int             `json:"version"`
	SessionID string          `json:"session_id"`
	Sequence  int64           `json:"sequence"`
	EventID   string          `json:"event_id"`
	TurnID    string          `json:"turn_id,omitempty"`
	EventType string          `json:"event_type"`
	AgentName string          `json:"agent_name,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	Timestamp string          `json:"timestamp,omitempty"`
}

type ledgerTarget struct {
	path        string
	workspaceID string
}

// NewMaterializer creates a forensic ledger materializer rooted at Config.RootDir.
func NewMaterializer(config Config) (*Materializer, error) {
	root := strings.TrimSpace(config.RootDir)
	if root == "" {
		return nil, fmt.Errorf("%w: root dir is required", ErrInvalidRecord)
	}
	unbound := strings.TrimSpace(config.UnboundPartition)
	if unbound == "" {
		unbound = DefaultUnboundPartition
	}
	opener := config.OpenEventStore
	if opener == nil {
		opener = openReadOnlyEventStore
	}
	return &Materializer{
		rootDir:          root,
		unboundPartition: unbound,
		openEventStore:   opener,
	}, nil
}

// MaterializeSessionLedger implements session.LedgerMaterializer.
func (m *Materializer) MaterializeSessionLedger(ctx context.Context, record store.SessionLedgerRecord) error {
	_, err := m.Materialize(ctx, record)
	return err
}

// Materialize writes ledger.jsonl from existing durable session evidence.
func (m *Materializer) Materialize(ctx context.Context, record store.SessionLedgerRecord) (result Result, err error) {
	if ctx == nil {
		return Result{}, errors.New("sessions/ledger: materialize context is required")
	}
	if m == nil {
		return Result{}, errors.New("sessions/ledger: materializer is required")
	}

	target, err := m.target(record)
	if err != nil {
		return Result{}, err
	}
	eventsDBPath := strings.TrimSpace(record.EventsDBPath)
	if eventsDBPath == "" {
		return Result{}, fmt.Errorf("%w: events db path is required", ErrInvalidRecord)
	}
	recorder, err := m.openEventStore(ctx, strings.TrimSpace(record.SessionID), eventsDBPath)
	if err != nil {
		return Result{}, fmt.Errorf("sessions/ledger: open event store for %q: %w", record.SessionID, err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), closeTimeout)
		defer cancel()
		if closeErr := recorder.Close(closeCtx); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	events, err := recorder.Query(ctx, store.EventQuery{})
	if err != nil {
		return Result{}, fmt.Errorf("sessions/ledger: query events for %q: %w", record.SessionID, err)
	}
	rendered, err := renderLedger(record, target.workspaceID, events)
	if err != nil {
		return Result{}, err
	}
	checksum := checksumLedger(rendered)
	result = Result{
		Path:     target.path,
		Checksum: checksum,
		Events:   len(events),
	}

	existing, err := os.ReadFile(target.path)
	switch {
	case err == nil && bytes.Equal(existing, rendered):
		return result, nil
	case err == nil:
		return Result{}, fmt.Errorf("%w: %s", ErrLedgerExists, target.path)
	case errors.Is(err, os.ErrNotExist):
	default:
		return Result{}, fmt.Errorf("sessions/ledger: read existing ledger %q: %w", target.path, err)
	}

	if err := os.MkdirAll(filepath.Dir(target.path), 0o755); err != nil {
		return Result{}, fmt.Errorf("sessions/ledger: create ledger directory for %q: %w", target.path, err)
	}
	if err := fileutil.AtomicWriteFile(target.path, rendered, 0o644); err != nil {
		return Result{}, fmt.Errorf("sessions/ledger: write ledger %q: %w", target.path, err)
	}
	result.Written = true
	return result, nil
}

// Path returns the deterministic ledger.jsonl path for a session.
func (m *Materializer) Path(record store.SessionLedgerRecord) (string, error) {
	target, err := m.target(record)
	if err != nil {
		return "", err
	}
	return target.path, nil
}

func (m *Materializer) target(record store.SessionLedgerRecord) (ledgerTarget, error) {
	sessionID, err := safeSegment(record.SessionID, "session id")
	if err != nil {
		return ledgerTarget{}, err
	}
	partitionValue := strings.TrimSpace(record.WorkspaceID)
	if partitionValue == "" {
		partitionValue = m.unboundPartition
	}
	partition, err := safeSegment(partitionValue, "workspace id")
	if err != nil {
		return ledgerTarget{}, err
	}
	return ledgerTarget{
		path:        filepath.Join(m.rootDir, partition, sessionID, ledgerFileName),
		workspaceID: partition,
	}, nil
}

type readOnlyEventStore struct {
	db        *sql.DB
	sessionID string
}

var _ store.EventRecorder = (*readOnlyEventStore)(nil)

type ledgerRowScanner interface {
	Scan(dest ...any) error
}

func openReadOnlyEventStore(ctx context.Context, sessionID string, path string) (store.EventRecorder, error) {
	if ctx == nil {
		return nil, errors.New("sessions/ledger: open read-only event store context is required")
	}
	cleanSessionID := strings.TrimSpace(sessionID)
	if cleanSessionID == "" {
		return nil, errors.New("sessions/ledger: read-only event store session id is required")
	}
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, fmt.Errorf("%w: events db path is required", ErrInvalidRecord)
	}
	db, err := sql.Open("sqlite", readOnlySQLiteDSN(cleanPath))
	if err != nil {
		return nil, fmt.Errorf("sessions/ledger: open read-only event store %q: %w", cleanPath, err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.PingContext(ctx); err != nil {
		return nil, closeReadOnlyAfterOpenError(
			db,
			fmt.Errorf("sessions/ledger: ping read-only event store %q: %w", cleanPath, err),
		)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA query_only = ON"); err != nil {
		return nil, closeReadOnlyAfterOpenError(
			db,
			fmt.Errorf("sessions/ledger: enable read-only event store guard %q: %w", cleanPath, err),
		)
	}
	return &readOnlyEventStore{db: db, sessionID: cleanSessionID}, nil
}

func closeReadOnlyAfterOpenError(db *sql.DB, openErr error) error {
	if db == nil {
		return openErr
	}
	if closeErr := db.Close(); closeErr != nil {
		return errors.Join(
			openErr,
			fmt.Errorf("sessions/ledger: close read-only event store after open failure: %w", closeErr),
		)
	}
	return openErr
}

func readOnlySQLiteDSN(path string) string {
	u := url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(path),
	}
	query := u.Query()
	query.Set("mode", "ro")
	u.RawQuery = query.Encode()
	return u.String()
}

func (s *readOnlyEventStore) Record(context.Context, store.SessionEvent) error {
	return errors.New("sessions/ledger: read-only event store cannot record events")
}

func (s *readOnlyEventStore) RecordTokenUsage(context.Context, store.TokenUsage) error {
	return errors.New("sessions/ledger: read-only event store cannot record token usage")
}

func (s *readOnlyEventStore) Query(
	ctx context.Context,
	query store.EventQuery,
) (events []store.SessionEvent, err error) {
	if s == nil || s.db == nil {
		return nil, errors.New("sessions/ledger: read-only event store is required")
	}
	if ctx == nil {
		return nil, errors.New("sessions/ledger: query read-only event store context is required")
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}
	baseQuery := `SELECT id, sequence, turn_id, type, agent_name, content, timestamp FROM events`
	where, args := store.BuildClauses(
		store.StringClause("type", query.Type),
		store.StringClause("agent_name", query.AgentName),
		store.StringClause("turn_id", query.TurnID),
		store.TimeClause("timestamp", ">=", query.Since),
		store.Int64Clause("sequence", ">", query.AfterSequence),
	)
	baseQuery = store.AppendWhere(baseQuery, where)

	sqlQuery := baseQuery
	if query.Limit > 0 {
		sqlQuery = `SELECT id, sequence, turn_id, type, agent_name, content, timestamp
			FROM (` + baseQuery + ` ORDER BY sequence DESC LIMIT ?) AS recent_events
			ORDER BY sequence ASC`
		args = append(args, query.Limit)
	} else {
		sqlQuery += " ORDER BY sequence ASC"
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("sessions/ledger: query read-only events: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("sessions/ledger: close read-only event rows: %w", closeErr)
		}
	}()

	events = make([]store.SessionEvent, 0)
	for rows.Next() {
		event, scanErr := s.scanEvent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sessions/ledger: iterate read-only events: %w", err)
	}
	return events, nil
}

func (s *readOnlyEventStore) History(ctx context.Context, query store.EventQuery) ([]store.TurnHistory, error) {
	events, err := s.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	turns := make([]store.TurnHistory, 0)
	indexByTurnID := make(map[string]int, len(events))
	for _, event := range events {
		if idx, ok := indexByTurnID[event.TurnID]; ok {
			turns[idx].Events = append(turns[idx].Events, event)
			continue
		}
		indexByTurnID[event.TurnID] = len(turns)
		turns = append(turns, store.TurnHistory{
			TurnID: event.TurnID,
			Events: []store.SessionEvent{event},
		})
	}
	return turns, nil
}

func (s *readOnlyEventStore) Close(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("sessions/ledger: close read-only event store context is required")
	}
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("sessions/ledger: close read-only event store: %w", err)
	}
	s.db = nil
	return nil
}

func (s *readOnlyEventStore) scanEvent(row ledgerRowScanner) (store.SessionEvent, error) {
	var event store.SessionEvent
	var timestamp string
	if err := row.Scan(
		&event.ID,
		&event.Sequence,
		&event.TurnID,
		&event.Type,
		&event.AgentName,
		&event.Content,
		&timestamp,
	); err != nil {
		return store.SessionEvent{}, fmt.Errorf("sessions/ledger: scan read-only event: %w", err)
	}
	parsed, err := store.ParseTimestamp(timestamp)
	if err != nil {
		return store.SessionEvent{}, err
	}
	event.SessionID = s.sessionID
	event.Timestamp = parsed
	return event, nil
}

func renderLedger(record store.SessionLedgerRecord, workspaceID string, events []store.SessionEvent) ([]byte, error) {
	var buf bytes.Buffer
	meta := ledgerMetaLine{
		Type:        "ledger_meta",
		Version:     ledgerVersion,
		SessionID:   strings.TrimSpace(record.SessionID),
		WorkspaceID: workspaceID,
		AgentName:   strings.TrimSpace(record.AgentName),
		SessionType: strings.TrimSpace(record.SessionType),
		StartedAt:   formatLedgerTime(record.StartedAt),
		EndedAt:     formatLedgerTime(record.EndedAt),
	}
	if lineage := store.NormalizeSessionLineage(record.SessionID, record.Lineage); lineage != nil {
		meta.SpawnParentID = strings.TrimSpace(lineage.ParentSessionID)
		meta.RootSessionID = strings.TrimSpace(lineage.RootSessionID)
		meta.SpawnDepth = lineage.SpawnDepth
	}
	if err := appendJSONL(&buf, meta); err != nil {
		return nil, err
	}
	for _, event := range orderedEvents(events) {
		line := ledgerEventLine{
			Type:      "session_event",
			Version:   ledgerVersion,
			SessionID: strings.TrimSpace(firstNonEmpty(event.SessionID, record.SessionID)),
			Sequence:  event.Sequence,
			EventID:   strings.TrimSpace(event.ID),
			TurnID:    strings.TrimSpace(event.TurnID),
			EventType: strings.TrimSpace(event.Type),
			AgentName: strings.TrimSpace(event.AgentName),
			Content:   contentJSON(event.Content),
			Timestamp: formatLedgerTime(event.Timestamp),
		}
		if err := appendJSONL(&buf, line); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func orderedEvents(events []store.SessionEvent) []store.SessionEvent {
	ordered := append([]store.SessionEvent(nil), events...)
	slices.SortStableFunc(ordered, func(a store.SessionEvent, b store.SessionEvent) int {
		switch {
		case a.Sequence < b.Sequence:
			return -1
		case a.Sequence > b.Sequence:
			return 1
		default:
			return strings.Compare(a.ID, b.ID)
		}
	})
	return ordered
}

func appendJSONL(buf *bytes.Buffer, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("sessions/ledger: encode ledger line: %w", err)
	}
	if _, err := buf.Write(raw); err != nil {
		return fmt.Errorf("sessions/ledger: write ledger line: %w", err)
	}
	if err := buf.WriteByte('\n'); err != nil {
		return fmt.Errorf("sessions/ledger: terminate ledger line: %w", err)
	}
	return nil
}

func contentJSON(content string) json.RawMessage {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}
	if json.Valid([]byte(trimmed)) {
		return json.RawMessage(trimmed)
	}
	raw, err := json.Marshal(content)
	if err != nil {
		return nil
	}
	return json.RawMessage(raw)
}

func checksumLedger(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func safeSegment(value string, field string) (string, error) {
	segment := strings.TrimSpace(value)
	if segment == "" {
		return "", fmt.Errorf("%w: %s is required", ErrInvalidRecord, field)
	}
	if filepath.IsAbs(segment) || segment == "." || segment == ".." {
		return "", fmt.Errorf("%w: unsafe %s %q", ErrInvalidRecord, field, value)
	}
	if strings.Contains(segment, "/") || strings.Contains(segment, `\`) || strings.ContainsRune(segment, 0) {
		return "", fmt.Errorf("%w: unsafe %s %q", ErrInvalidRecord, field, value)
	}
	if filepath.Clean(segment) != segment {
		return "", fmt.Errorf("%w: unsafe %s %q", ErrInvalidRecord, field, value)
	}
	return segment, nil
}

func formatLedgerTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
