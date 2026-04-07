package sessiondb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

type SessionEvent = store.SessionEvent
type TokenUsage = store.TokenUsage
type EventQuery = store.EventQuery

const SessionDatabaseName = store.SessionDatabaseName

func TestOpenSessionDBCreatesSchemaAndEnablesWAL(t *testing.T) {
	t.Parallel()

	sessionDB := openTestSessionDB(t, "sess-open")

	assertTablesPresent(t, sessionDB.db, "events", "token_usage")
	assertJournalModeWAL(t, sessionDB.db)
	assertSynchronousNormal(t, sessionDB.db)
}

func TestSessionDBRecordAutoIncrementSequence(t *testing.T) {
	t.Parallel()

	sessionDB := openTestSessionDB(t, "sess-seq")
	base := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	callCount := 0
	sessionDB.now = func() time.Time {
		callCount++
		return base.Add(time.Duration(callCount) * time.Second)
	}

	ctx := testutil.Context(t)
	if err := sessionDB.Record(ctx, SessionEvent{TurnID: "turn-1", Type: "agent_message", AgentName: "coder", Content: `{"text":"one"}`}); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if err := sessionDB.Record(ctx, SessionEvent{TurnID: "turn-1", Type: "tool_call", AgentName: "coder", Content: `{"tool":"ls"}`}); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	events, err := sessionDB.Query(ctx, EventQuery{})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if got, want := len(events), 2; got != want {
		t.Fatalf("len(events) = %d, want %d", got, want)
	}
	if events[0].Sequence != 1 || events[1].Sequence != 2 {
		t.Fatalf("event sequences = [%d %d], want [1 2]", events[0].Sequence, events[1].Sequence)
	}
	if events[0].SessionID != "sess-seq" || events[1].SessionID != "sess-seq" {
		t.Fatalf("session ids = [%q %q], want sess-seq", events[0].SessionID, events[1].SessionID)
	}
}

func TestSessionDBRecordTokenUsageStoresNullableFieldsAsNULL(t *testing.T) {
	t.Parallel()

	sessionDB := openTestSessionDB(t, "sess-usage")
	outputTokens := int64(12)
	usage := TokenUsage{
		TurnID:       "turn-usage",
		OutputTokens: &outputTokens,
	}

	if err := sessionDB.RecordTokenUsage(testutil.Context(t), usage); err != nil {
		t.Fatalf("RecordTokenUsage() error = %v", err)
	}

	var (
		inputTokens sql.NullInt64
		output      sql.NullInt64
		totalTokens sql.NullInt64
		currency    sql.NullString
	)
	if err := sessionDB.db.QueryRowContext(
		testutil.Context(t),
		`SELECT input_tokens, output_tokens, total_tokens, cost_currency FROM token_usage WHERE turn_id = ?`,
		"turn-usage",
	).Scan(&inputTokens, &output, &totalTokens, &currency); err != nil {
		t.Fatalf("QueryRowContext() error = %v", err)
	}

	if inputTokens.Valid {
		t.Fatalf("input_tokens.Valid = true, want false")
	}
	if !output.Valid || output.Int64 != 12 {
		t.Fatalf("output_tokens = %#v, want valid 12", output)
	}
	if totalTokens.Valid {
		t.Fatalf("total_tokens.Valid = true, want false")
	}
	if currency.Valid {
		t.Fatalf("cost_currency.Valid = true, want false")
	}
}

func TestSessionDBQueryFilters(t *testing.T) {
	t.Parallel()

	sessionDB := openTestSessionDB(t, "sess-query")
	base := time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)
	callCount := 0
	sessionDB.now = func() time.Time {
		callCount++
		return base.Add(time.Duration(callCount) * time.Minute)
	}

	events := []SessionEvent{
		{TurnID: "turn-1", Type: "agent_message", AgentName: "coder", Content: `{"text":"one"}`},
		{TurnID: "turn-1", Type: "tool_call", AgentName: "coder", Content: `{"tool":"ls"}`},
		{TurnID: "turn-2", Type: "agent_message", AgentName: "reviewer", Content: `{"text":"two"}`},
		{TurnID: "turn-3", Type: "error", AgentName: "coder", Content: `{"error":"boom"}`},
	}
	for _, event := range events {
		if err := sessionDB.Record(testutil.Context(t), event); err != nil {
			t.Fatalf("Record(%q) error = %v", event.Type, err)
		}
	}

	tests := []struct {
		name      string
		query     EventQuery
		wantSeqs  []int64
		wantTypes []string
	}{
		{
			name:      "type filter",
			query:     EventQuery{Type: "agent_message"},
			wantSeqs:  []int64{1, 3},
			wantTypes: []string{"agent_message", "agent_message"},
		},
		{
			name:      "since filter",
			query:     EventQuery{Since: base.Add(2 * time.Minute)},
			wantSeqs:  []int64{2, 3, 4},
			wantTypes: []string{"tool_call", "agent_message", "error"},
		},
		{
			name:      "limit returns most recent in ascending order",
			query:     EventQuery{Limit: 2},
			wantSeqs:  []int64{3, 4},
			wantTypes: []string{"agent_message", "error"},
		},
		{
			name:      "agent filter",
			query:     EventQuery{AgentName: "coder"},
			wantSeqs:  []int64{1, 2, 4},
			wantTypes: []string{"agent_message", "tool_call", "error"},
		},
		{
			name:      "follow compatible after sequence filter",
			query:     EventQuery{AfterSequence: 2},
			wantSeqs:  []int64{3, 4},
			wantTypes: []string{"agent_message", "error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sessionDB.Query(testutil.Context(t), tt.query)
			if err != nil {
				t.Fatalf("Query() error = %v", err)
			}
			if gotSeqs := eventSequences(got); !equalInt64Slices(gotSeqs, tt.wantSeqs) {
				t.Fatalf("eventSequences() = %#v, want %#v", gotSeqs, tt.wantSeqs)
			}
			if gotTypes := eventTypes(got); !testutil.EqualStringSlices(gotTypes, tt.wantTypes) {
				t.Fatalf("eventTypes() = %#v, want %#v", gotTypes, tt.wantTypes)
			}
		})
	}
}

func TestSessionDBQueryOrderedBySequence(t *testing.T) {
	t.Parallel()

	sessionDB := openTestSessionDB(t, "sess-order")
	base := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	customTimes := []time.Time{
		base.Add(3 * time.Minute),
		base.Add(1 * time.Minute),
		base.Add(2 * time.Minute),
	}

	for index, ts := range customTimes {
		if err := sessionDB.Record(testutil.Context(t), SessionEvent{
			TurnID:    fmt.Sprintf("turn-%d", index+1),
			Type:      "agent_message",
			AgentName: "coder",
			Content:   fmt.Sprintf(`{"index":%d}`, index+1),
			Timestamp: ts,
		}); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}

	events, err := sessionDB.Query(testutil.Context(t), EventQuery{})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if gotSeqs := eventSequences(events); !equalInt64Slices(gotSeqs, []int64{1, 2, 3}) {
		t.Fatalf("eventSequences() = %#v, want %#v", gotSeqs, []int64{1, 2, 3})
	}
}

func TestSessionDBHistoryGroupsByTurn(t *testing.T) {
	t.Parallel()

	sessionDB := openTestSessionDB(t, "sess-history")
	input := []SessionEvent{
		{TurnID: "turn-a", Type: "agent_message", AgentName: "coder", Content: `{"text":"one"}`},
		{TurnID: "turn-a", Type: "tool_result", AgentName: "coder", Content: `{"tool":"ls"}`},
		{TurnID: "turn-b", Type: "agent_message", AgentName: "coder", Content: `{"text":"two"}`},
	}
	for _, event := range input {
		if err := sessionDB.Record(testutil.Context(t), event); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}

	history, err := sessionDB.History(testutil.Context(t), EventQuery{})
	if err != nil {
		t.Fatalf("History() error = %v", err)
	}
	if got, want := len(history), 2; got != want {
		t.Fatalf("len(history) = %d, want %d", got, want)
	}
	if history[0].TurnID != "turn-a" || history[1].TurnID != "turn-b" {
		t.Fatalf("turn ids = [%q %q], want [turn-a turn-b]", history[0].TurnID, history[1].TurnID)
	}
	if gotSeqs := eventSequences(history[0].Events); !equalInt64Slices(gotSeqs, []int64{1, 2}) {
		t.Fatalf("turn-a sequences = %#v, want %#v", gotSeqs, []int64{1, 2})
	}
	if gotSeqs := eventSequences(history[1].Events); !equalInt64Slices(gotSeqs, []int64{3}) {
		t.Fatalf("turn-b sequences = %#v, want %#v", gotSeqs, []int64{3})
	}
}

func TestSessionDBRecoversFromCorruption(t *testing.T) {
	t.Parallel()

	sessionDir := t.TempDir()
	path := filepath.Join(sessionDir, SessionDatabaseName)
	if err := os.WriteFile(path, []byte("not a sqlite database"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	sessionDB, err := OpenSessionDB(testutil.Context(t), "sess-corrupt", path)
	if err != nil {
		t.Fatalf("OpenSessionDB() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := sessionDB.Close(testutil.Context(t)); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	assertTablesPresent(t, sessionDB.db, "events", "token_usage")

	matches, err := filepath.Glob(path + ".corrupt.*")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if got, want := len(matches), 1; got != want {
		t.Fatalf("len(corrupt files) = %d, want %d (%v)", got, want, matches)
	}
}

func TestSessionDBWriteFailureReturnsError(t *testing.T) {
	t.Parallel()

	sessionDB := openTestSessionDB(t, "sess-full")

	var pageCount int
	if err := sessionDB.db.QueryRowContext(testutil.Context(t), "PRAGMA page_count").Scan(&pageCount); err != nil {
		t.Fatalf("QueryRowContext(page_count) error = %v", err)
	}
	if _, err := sessionDB.db.ExecContext(testutil.Context(t), fmt.Sprintf("PRAGMA max_page_count = %d", pageCount)); err != nil {
		t.Fatalf("ExecContext(max_page_count) error = %v", err)
	}

	err := sessionDB.Record(testutil.Context(t), SessionEvent{
		TurnID:    "turn-disk-full",
		Type:      "agent_message",
		AgentName: "coder",
		Content:   strings.Repeat("x", 1<<20),
	})
	if err == nil {
		t.Fatal("Record() error = nil, want non-nil")
	}

	events, queryErr := sessionDB.Query(testutil.Context(t), EventQuery{})
	if queryErr != nil {
		t.Fatalf("Query() error = %v", queryErr)
	}
	if got := len(events); got != 0 {
		t.Fatalf("len(events) = %d, want 0", got)
	}
}

func openTestSessionDB(t *testing.T, sessionID string) *SessionDB {
	t.Helper()

	sessionDB, err := OpenSessionDB(testutil.Context(t), sessionID, filepath.Join(t.TempDir(), SessionDatabaseName))
	if err != nil {
		t.Fatalf("OpenSessionDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := sessionDB.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	return sessionDB
}

func assertTablesPresent(t *testing.T, db *sql.DB, want ...string) {
	t.Helper()

	rows, err := db.QueryContext(testutil.Context(t), `SELECT name FROM sqlite_master WHERE type = 'table'`)
	if err != nil {
		t.Fatalf("QueryContext(sqlite_master) error = %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	have := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("Scan() error = %v", err)
		}
		have[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err() error = %v", err)
	}

	for _, table := range want {
		if _, ok := have[table]; !ok {
			keys := make([]string, 0, len(have))
			for key := range have {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			t.Fatalf("missing table %q, have %v", table, keys)
		}
	}
}

func assertJournalModeWAL(t *testing.T, db *sql.DB) {
	t.Helper()

	var mode string
	if err := db.QueryRowContext(testutil.Context(t), "PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("QueryRowContext(journal_mode) error = %v", err)
	}
	if !strings.EqualFold(mode, "wal") {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
}

func assertSynchronousNormal(t *testing.T, db *sql.DB) {
	t.Helper()

	var synchronous int
	if err := db.QueryRowContext(testutil.Context(t), "PRAGMA synchronous").Scan(&synchronous); err != nil {
		t.Fatalf("QueryRowContext(synchronous) error = %v", err)
	}
	if synchronous != 1 {
		t.Fatalf("synchronous = %d, want 1 (NORMAL)", synchronous)
	}
}

func eventSequences(events []SessionEvent) []int64 {
	out := make([]int64, 0, len(events))
	for _, event := range events {
		out = append(out, event.Sequence)
	}
	return out
}

func eventTypes(events []SessionEvent) []string {
	out := make([]string, 0, len(events))
	for _, event := range events {
		out = append(out, event.Type)
	}
	return out
}

func equalInt64Slices(left []int64, right []int64) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
