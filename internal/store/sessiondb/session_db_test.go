package sessiondb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
)

type SessionEvent = store.SessionEvent
type TokenUsage = store.TokenUsage
type EventQuery = store.EventQuery

const SessionDatabaseName = store.SessionDatabaseName

func TestOpenSessionDBCreatesSchemaAndEnablesWAL(t *testing.T) {
	t.Parallel()

	t.Run("Should create schema and enable WAL", func(t *testing.T) {
		t.Parallel()

		sessionDB := openTestSessionDB(t, "sess-open")

		assertTablesPresent(t, sessionDB.db, "schema_migrations", "events", "token_usage")
		assertUniqueIndex(t, sessionDB.db, "events", "idx_events_sequence")
		assertJournalModeWAL(t, sessionDB.db)
		assertSynchronousNormal(t, sessionDB.db)
	})
}

func TestOpenSessionDBDisablesAutomaticWALCheckpoints(t *testing.T) {
	t.Parallel()

	t.Run("Should defer WAL checkpoints until explicit close", func(t *testing.T) {
		t.Parallel()

		sessionDB := openTestSessionDB(t, "sess-wal-checkpoint")

		assertWALAutoCheckpoint(t, sessionDB.db, 0)
	})
}

func TestOpenSessionDBRecordsSchemaMigrationAndRepeatedBootIsIdempotent(t *testing.T) {
	t.Parallel()

	t.Run("Should record schema migrations and keep repeated boot idempotent", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), SessionDatabaseName)
		first, err := OpenSessionDB(ctx, "sess-idempotent", path)
		if err != nil {
			t.Fatalf("OpenSessionDB(first) error = %v", err)
		}
		firstRecords, err := store.AppliedMigrations(ctx, first.db)
		if err != nil {
			t.Fatalf("AppliedMigrations(first) error = %v", err)
		}
		assertSessionSchemaMigrations(t, firstRecords)
		assertUniqueIndex(t, first.db, "events", "idx_events_sequence")
		if err := first.Close(ctx); err != nil {
			t.Fatalf("Close(first) error = %v", err)
		}

		second, err := OpenSessionDB(ctx, "sess-idempotent", path)
		if err != nil {
			t.Fatalf("OpenSessionDB(second) error = %v", err)
		}
		t.Cleanup(func() {
			if err := second.Close(testutil.Context(t)); err != nil {
				t.Fatalf("Close(second) error = %v", err)
			}
		})
		secondRecords, err := store.AppliedMigrations(ctx, second.db)
		if err != nil {
			t.Fatalf("AppliedMigrations(second) error = %v", err)
		}
		assertSessionSchemaMigrations(t, secondRecords)
		assertUniqueIndex(t, second.db, "events", "idx_events_sequence")
		for index, firstRecord := range firstRecords {
			if !secondRecords[index].AppliedAt.Equal(firstRecord.AppliedAt) {
				t.Fatalf(
					"second v%d applied_at = %s, want unchanged %s",
					firstRecord.Version,
					secondRecords[index].AppliedAt,
					firstRecord.AppliedAt,
				)
			}
		}
	})
}

func TestOpenSessionDBStripsCanonicalRawPayloadsAndVacuumsOldRows(t *testing.T) {
	t.Run("Should strip canonical raw payloads and vacuum old session rows", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), SessionDatabaseName)
		legacyRaw := strings.Repeat("search result line\n", 350000)
		legacyContent := fmt.Sprintf(
			`{"schema":%q,"type":"tool_call","turn_id":"turn-1","tool_call_id":"call-1","timestamp":"2026-04-25T22:00:00Z","raw":{"sessionUpdate":"tool_call_update","toolCallId":"call-1","content":[{"type":"content","content":{"type":"text","text":%q}}]}}`,
			canonicalEventSchema,
			legacyRaw,
		)

		legacyDB, err := store.OpenSQLiteDatabase(
			ctx,
			path,
			func(ctx context.Context, db *sql.DB) error {
				if err := store.RunMigrations(ctx, db, sessionSchemaMigrations[:1]); err != nil {
					return err
				}
				_, err := db.ExecContext(
					ctx,
					`INSERT INTO events (id, sequence, turn_id, type, agent_name, content, timestamp)
					 VALUES (?, ?, ?, ?, ?, ?, ?)`,
					"ev-legacy",
					1,
					"turn-1",
					"tool_call",
					"ceo",
					legacyContent,
					"2026-04-25T22:00:00Z",
				)
				return err
			},
		)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase(legacy) error = %v", err)
		}
		if err := store.Checkpoint(ctx, legacyDB); err != nil {
			t.Fatalf("Checkpoint(legacy) error = %v", err)
		}
		if err := legacyDB.Close(); err != nil {
			t.Fatalf("legacyDB.Close() error = %v", err)
		}

		before, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat(before) error = %v", err)
		}

		sessionDB, err := OpenSessionDB(ctx, "sess-legacy-raw", path)
		if err != nil {
			t.Fatalf("OpenSessionDB(migrated) error = %v", err)
		}

		var migratedContent string
		if err := sessionDB.db.QueryRowContext(
			ctx,
			`SELECT content FROM events WHERE id = ?`,
			"ev-legacy",
		).Scan(&migratedContent); err != nil {
			t.Fatalf("QueryRowContext(content) error = %v", err)
		}
		if strings.Contains(migratedContent, `"raw"`) {
			t.Fatalf(
				"migrated content still contains raw payload: %q",
				migratedContent[:smallerInt(len(migratedContent), 200)],
			)
		}

		if err := store.Checkpoint(ctx, sessionDB.db); err != nil {
			t.Fatalf("Checkpoint(migrated) error = %v", err)
		}
		if err := sessionDB.Close(ctx); err != nil {
			t.Fatalf("Close(migrated) error = %v", err)
		}

		after, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat(after) error = %v", err)
		}
		if after.Size() >= before.Size() {
			t.Fatalf(
				"events.db size after migrate = %d, want smaller than %d",
				after.Size(),
				before.Size(),
			)
		}
	})
}

func TestOpenSessionSQLiteDoesNotFailWhenVacuumFails(t *testing.T) {
	t.Run("Should keep opening the session database when vacuuming fails", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), SessionDatabaseName)
		sentinel := errors.New("vacuum unavailable")

		db, err := openSessionSQLiteWithVacuum(ctx, path, func(context.Context, *sql.DB) error {
			return sentinel
		})
		if err != nil {
			t.Fatalf("openSessionSQLiteWithVacuum() error = %v, want nil", err)
		}
		t.Cleanup(func() {
			if err := db.Close(); err != nil {
				t.Fatalf("db.Close() error = %v", err)
			}
		})

		assertTablesPresent(t, db, "schema_migrations", "events", "token_usage")
		assertJournalModeWAL(t, db)
	})
}

func smallerInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestSessionDBRecordAutoIncrementSequence(t *testing.T) {
	t.Parallel()

	t.Run("Should assign strict sequences for a single handle", func(t *testing.T) {
		t.Parallel()

		sessionDB := openTestSessionDB(t, "sess-seq")
		base := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
		callCount := 0
		sessionDB.now = func() time.Time {
			callCount++
			return base.Add(time.Duration(callCount) * time.Second)
		}

		ctx := testutil.Context(t)
		if err := sessionDB.Record(
			ctx,
			SessionEvent{TurnID: "turn-1", Type: "agent_message", AgentName: "coder", Content: `{"text":"one"}`},
		); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
		if err := sessionDB.Record(
			ctx,
			SessionEvent{TurnID: "turn-1", Type: "tool_call", AgentName: "coder", Content: `{"tool":"ls"}`},
		); err != nil {
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
	})

	t.Run("Should assign strict sequences for concurrent handles", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), SessionDatabaseName)
		first, err := OpenSessionDB(ctx, "sess-seq-shared", path)
		if err != nil {
			t.Fatalf("OpenSessionDB(first) error = %v", err)
		}
		t.Cleanup(func() {
			if err := first.Close(testutil.Context(t)); err != nil {
				t.Fatalf("Close(first) error = %v", err)
			}
		})
		second, err := OpenSessionDB(ctx, "sess-seq-shared", path)
		if err != nil {
			t.Fatalf("OpenSessionDB(second) error = %v", err)
		}
		t.Cleanup(func() {
			if err := second.Close(testutil.Context(t)); err != nil {
				t.Fatalf("Close(second) error = %v", err)
			}
		})

		if err := first.Record(ctx, SessionEvent{
			TurnID:    "turn-1",
			Type:      "agent_message",
			AgentName: "coder",
			Content:   `{"text":"one"}`,
		}); err != nil {
			t.Fatalf("Record(first) error = %v", err)
		}
		if err := second.Record(ctx, SessionEvent{
			TurnID:    "turn-2",
			Type:      "tool_result",
			AgentName: "coder",
			Content:   `{"text":"two"}`,
		}); err != nil {
			t.Fatalf("Record(second) error = %v", err)
		}

		events, err := first.Query(ctx, EventQuery{})
		if err != nil {
			t.Fatalf("Query() error = %v", err)
		}
		if gotSeqs := eventSequences(events); !equalInt64Slices(gotSeqs, []int64{1, 2}) {
			t.Fatalf("eventSequences() = %#v, want %#v", gotSeqs, []int64{1, 2})
		}
		afterFirst, err := first.Query(ctx, EventQuery{AfterSequence: 1})
		if err != nil {
			t.Fatalf("Query(AfterSequence: 1) error = %v", err)
		}
		if gotSeqs := eventSequences(afterFirst); !equalInt64Slices(gotSeqs, []int64{2}) {
			t.Fatalf("after first event sequences = %#v, want %#v", gotSeqs, []int64{2})
		}
	})
}

func TestSessionDBRecordTokenUsageStoresNullableFieldsAsNULL(t *testing.T) {
	t.Parallel()

	t.Run("Should store nullable token usage fields as NULL", func(t *testing.T) {
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
	})
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

	t.Run("Should order events by sequence", func(t *testing.T) {
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
	})
}

func TestSessionDBHistoryGroupsByTurn(t *testing.T) {
	t.Parallel()

	t.Run("Should group history by turn", func(t *testing.T) {
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
	})
}

func TestSessionDBRecoversFromCorruption(t *testing.T) {
	t.Parallel()

	t.Run("Should recover corrupt database files", func(t *testing.T) {
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

		assertTablesPresent(t, sessionDB.db, "schema_migrations", "events", "token_usage")

		matches, err := filepath.Glob(path + ".corrupt.*")
		if err != nil {
			t.Fatalf("Glob() error = %v", err)
		}
		if got, want := len(matches), 1; got != want {
			t.Fatalf("len(corrupt files) = %d, want %d (%v)", got, want, matches)
		}
	})
}

func TestSessionDBWriteFailureReturnsError(t *testing.T) {
	t.Parallel()

	t.Run("Should return errors when writes fail", func(t *testing.T) {
		t.Parallel()

		sessionDB := openTestSessionDB(t, "sess-full")

		var pageCount int
		if err := sessionDB.db.QueryRowContext(testutil.Context(t), "PRAGMA page_count").Scan(&pageCount); err != nil {
			t.Fatalf("QueryRowContext(page_count) error = %v", err)
		}
		if _, err := sessionDB.db.ExecContext(
			testutil.Context(t),
			fmt.Sprintf("PRAGMA max_page_count = %d", pageCount),
		); err != nil {
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
	})
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
		if err := rows.Close(); err != nil {
			t.Fatalf("rows.Close() error = %v", err)
		}
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

func assertSessionSchemaMigrations(t *testing.T, records []store.MigrationRecord) {
	t.Helper()

	want := []struct {
		version int
		name    string
	}{
		{version: 1, name: "create_session_schema"},
		{version: 2, name: "strip_canonical_event_raw_payloads"},
		{version: 3, name: "enforce_unique_event_sequences"},
	}
	if got, wantLen := len(records), len(want); got != wantLen {
		t.Fatalf("len(records) = %d, want %d", got, wantLen)
	}
	for index, wantRecord := range want {
		if records[index].Version != wantRecord.version || records[index].Name != wantRecord.name {
			t.Fatalf(
				"records[%d] = %#v, want version %d name %q",
				index,
				records[index],
				wantRecord.version,
				wantRecord.name,
			)
		}
	}
}

func assertUniqueIndex(t *testing.T, db *sql.DB, table string, indexName string) {
	t.Helper()

	rows, err := db.QueryContext(testutil.Context(t), "PRAGMA index_list("+table+")")
	if err != nil {
		t.Fatalf("QueryContext(index_list %s) error = %v", table, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Fatalf("rows.Close() error = %v", err)
		}
	}()

	for rows.Next() {
		var (
			seq     int
			name    string
			unique  int
			origin  string
			partial int
		)
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("Scan(index_list %s) error = %v", table, err)
		}
		if name == indexName {
			if unique != 1 {
				t.Fatalf("index %s unique = %d, want 1", indexName, unique)
			}
			return
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(index_list %s) error = %v", table, err)
	}
	t.Fatalf("index %s missing on table %s", indexName, table)
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

func assertWALAutoCheckpoint(t *testing.T, db *sql.DB, want int) {
	t.Helper()

	var pages int
	if err := db.QueryRowContext(testutil.Context(t), "PRAGMA wal_autocheckpoint").Scan(&pages); err != nil {
		t.Fatalf("QueryRowContext(wal_autocheckpoint) error = %v", err)
	}
	if pages != want {
		t.Fatalf("wal_autocheckpoint = %d, want %d", pages, want)
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
