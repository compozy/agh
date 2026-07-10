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

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
	"github.com/compozy/agh/internal/transcript"
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

	t.Run("Should disable sqlite autocheckpoint for writer-owned WAL policy", func(t *testing.T) {
		t.Parallel()

		sessionDB := openTestSessionDB(t, "sess-wal-checkpoint")

		assertWALAutoCheckpoint(t, sessionDB.db, 0)
	})
}

func TestSessionDBPassiveCheckpoint(t *testing.T) {
	t.Parallel()

	t.Run("Should keep live read-only openers complete after passive checkpoints", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), SessionDatabaseName)
		sessionDB, err := OpenSessionDB(ctx, "sess-passive-checkpoint", path)
		if err != nil {
			t.Fatalf("OpenSessionDB() error = %v", err)
		}
		t.Cleanup(func() {
			if err := sessionDB.Close(testutil.Context(t)); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		})

		for idx := range sessionPassiveCheckpointEvery + 5 {
			if err := sessionDB.Record(ctx, SessionEvent{
				TurnID:    "turn-checkpoint",
				Type:      "agent_message",
				AgentName: "coder",
				Content:   fmt.Sprintf(`{"text":"chunk-%d"}`, idx),
			}); err != nil {
				t.Fatalf("Record(%d) error = %v", idx, err)
			}
		}

		readOnly, err := OpenSessionDBReadOnly(ctx, "sess-passive-checkpoint", path)
		if err != nil {
			t.Fatalf("OpenSessionDBReadOnly() error = %v", err)
		}
		t.Cleanup(func() {
			if err := readOnly.Close(testutil.Context(t)); err != nil {
				t.Fatalf("Close(readOnly) error = %v", err)
			}
		})
		events, err := readOnly.Query(ctx, EventQuery{})
		if err != nil {
			t.Fatalf("Query(readOnly) error = %v", err)
		}
		if got, want := len(events), sessionPassiveCheckpointEvery+5; got != want {
			t.Fatalf("len(events) = %d, want %d", got, want)
		}
	})
}

func TestSessionDBRecordPersistedBatchCoalescesPromptChunks(t *testing.T) {
	t.Parallel()

	t.Run("Should persist contiguous same-turn text chunks as one event row", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		sessionDB := openTestSessionDB(t, "sess-coalesce-contiguous")
		now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)

		persisted, err := sessionDB.RecordPersistedBatch(ctx, []SessionEvent{
			canonicalStoreEvent(t, acp.AgentEvent{
				Type:      acp.EventTypeAgentMessage,
				SessionID: "acp-coalesce",
				TurnID:    "turn-coalesce",
				Timestamp: now,
				Text:      "hello ",
			}, "coder"),
			canonicalStoreEvent(t, acp.AgentEvent{
				Type:      acp.EventTypeAgentMessage,
				SessionID: "acp-coalesce",
				TurnID:    "turn-coalesce",
				Timestamp: now.Add(time.Millisecond),
				Text:      "world",
			}, "coder"),
			canonicalStoreEvent(t, acp.AgentEvent{
				Type:      acp.EventTypeDone,
				SessionID: "acp-coalesce",
				TurnID:    "turn-coalesce",
				Timestamp: now.Add(2 * time.Millisecond),
			}, "coder"),
		})
		if err != nil {
			t.Fatalf("RecordPersistedBatch() error = %v", err)
		}
		if got, want := len(persisted), 2; got != want {
			t.Fatalf("len(persisted) = %d, want %d", got, want)
		}
		assertEventSequences(t, persisted, []int64{1, 2})
		if got, want := storedAgentText(t, persisted[0]), "hello world"; got != want {
			t.Fatalf("coalesced text = %q, want %q", got, want)
		}

		stored, err := sessionDB.Query(ctx, EventQuery{})
		if err != nil {
			t.Fatalf("Query() error = %v", err)
		}
		if got, want := len(stored), 2; got != want {
			t.Fatalf("len(stored) = %d, want %d", got, want)
		}
		assertEventSequences(t, stored, []int64{1, 2})
	})

	t.Run("Should flush coalesced text when a non chunk boundary appears", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		sessionDB := openTestSessionDB(t, "sess-coalesce-boundary")
		now := time.Date(2026, 7, 7, 12, 1, 0, 0, time.UTC)
		persisted, err := sessionDB.RecordPersistedBatch(ctx, []SessionEvent{
			canonicalStoreEvent(t, acp.AgentEvent{
				Type:      acp.EventTypeAgentMessage,
				SessionID: "acp-boundary",
				TurnID:    "turn-boundary",
				Timestamp: now,
				Text:      "a",
			}, "coder"),
			canonicalStoreEvent(t, acp.AgentEvent{
				Type:      acp.EventTypeAgentMessage,
				SessionID: "acp-boundary",
				TurnID:    "turn-boundary",
				Timestamp: now.Add(time.Millisecond),
				Text:      "b",
			}, "coder"),
			canonicalStoreEvent(t, acp.AgentEvent{
				Type:       acp.EventTypeToolCall,
				SessionID:  "acp-boundary",
				TurnID:     "turn-boundary",
				Timestamp:  now.Add(2 * time.Millisecond),
				ToolCallID: "call-1",
				Title:      "Read",
			}, "coder"),
			canonicalStoreEvent(t, acp.AgentEvent{
				Type:      acp.EventTypeAgentMessage,
				SessionID: "acp-boundary",
				TurnID:    "turn-boundary",
				Timestamp: now.Add(3 * time.Millisecond),
				Text:      "c",
			}, "coder"),
		})
		if err != nil {
			t.Fatalf("RecordPersistedBatch() error = %v", err)
		}
		if got, want := len(persisted), 3; got != want {
			t.Fatalf("len(persisted) = %d, want %d", got, want)
		}
		assertEventSequences(t, persisted, []int64{1, 2, 3})
		if got, want := storedAgentText(t, persisted[0]), "ab"; got != want {
			t.Fatalf("first coalesced text = %q, want %q", got, want)
		}
		if got, want := persisted[1].Type, acp.EventTypeToolCall; got != want {
			t.Fatalf("boundary type = %q, want %q", got, want)
		}
		if got, want := storedAgentText(t, persisted[2]), "c"; got != want {
			t.Fatalf("second text = %q, want %q", got, want)
		}
	})

	t.Run("Should keep transcript output equal to uncoalesced chunks", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		sessionDB := openTestSessionDB(t, "sess-coalesce-transcript")
		now := time.Date(2026, 7, 7, 12, 2, 0, 0, time.UTC)
		uncoalesced := []SessionEvent{
			canonicalStoreEvent(t, acp.AgentEvent{
				Type:      acp.EventTypeAgentMessage,
				SessionID: "acp-transcript",
				TurnID:    "turn-transcript",
				Timestamp: now,
				Text:      "same ",
			}, "coder"),
			canonicalStoreEvent(t, acp.AgentEvent{
				Type:      acp.EventTypeAgentMessage,
				SessionID: "acp-transcript",
				TurnID:    "turn-transcript",
				Timestamp: now.Add(time.Millisecond),
				Text:      "answer",
			}, "coder"),
		}
		for idx := range uncoalesced {
			uncoalesced[idx].Sequence = int64(idx + 1)
		}

		persisted, err := sessionDB.RecordPersistedBatch(ctx, uncoalesced)
		if err != nil {
			t.Fatalf("RecordPersistedBatch() error = %v", err)
		}
		uncoalescedEntries, err := transcript.ToUIEntries(uncoalesced)
		if err != nil {
			t.Fatalf("ToUIEntries(uncoalesced) error = %v", err)
		}
		coalescedEntries, err := transcript.ToUIEntries(persisted)
		if err != nil {
			t.Fatalf("ToUIEntries(coalesced) error = %v", err)
		}
		if got, want := lastEntryText(t, coalescedEntries), lastEntryText(t, uncoalescedEntries); got != want {
			t.Fatalf("coalesced transcript text = %q, want %q", got, want)
		}
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

func TestOpenSessionDBReadOnly(t *testing.T) {
	t.Parallel()

	t.Run("Should fail without creating a missing database", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), SessionDatabaseName)
		_, err := OpenSessionDBReadOnly(ctx, "sess-read-only-missing", path)
		if err == nil {
			t.Fatal("OpenSessionDBReadOnly(missing) error = nil, want non-nil")
		}
		if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("Stat(read-only missing path) error = %v, want os.ErrNotExist", statErr)
		}
	})

	t.Run("Should query an existing database without accepting writes", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), SessionDatabaseName)
		writer, err := OpenSessionDB(ctx, "sess-read-only-existing", path)
		if err != nil {
			t.Fatalf("OpenSessionDB() error = %v", err)
		}
		if err := writer.Record(ctx, SessionEvent{
			TurnID:    "turn-read-only",
			Type:      "agent_message",
			AgentName: "coder",
			Content:   "{\"text\":\"persisted\"}",
		}); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
		if err := writer.Close(ctx); err != nil {
			t.Fatalf("Close(writer) error = %v", err)
		}

		reader, err := OpenSessionDBReadOnly(ctx, "sess-read-only-existing", path)
		if err != nil {
			t.Fatalf("OpenSessionDBReadOnly(existing) error = %v", err)
		}
		defer func() {
			if closeErr := reader.Close(testutil.Context(t)); closeErr != nil {
				t.Fatalf("Close(reader) error = %v", closeErr)
			}
		}()

		events, err := reader.Query(ctx, EventQuery{})
		if err != nil {
			t.Fatalf("Query(read-only) error = %v", err)
		}
		if got, want := len(events), 1; got != want {
			t.Fatalf("len(events) = %d, want %d", got, want)
		}
		if events[0].SessionID != "sess-read-only-existing" || events[0].TurnID != "turn-read-only" {
			t.Fatalf("events[0] = %#v, want session/turn ids set", events[0])
		}

		err = reader.Record(ctx, SessionEvent{
			TurnID:    "turn-forbidden",
			Type:      "agent_message",
			AgentName: "coder",
			Content:   "{\"text\":\"forbidden\"}",
		})
		if !errors.Is(err, ErrReadOnlyRecordEvents) {
			t.Fatalf("Record(read-only) error = %v, want read-only write rejection", err)
		}

		err = reader.RecordTokenUsage(ctx, TokenUsage{TurnID: "turn-forbidden"})
		if !errors.Is(err, ErrReadOnlyRecordTokenUsage) {
			t.Fatalf("RecordTokenUsage(read-only) error = %v, want read-only usage rejection", err)
		}
	})

	t.Run("Should apply identical cursor bounds to full and metadata projections", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), SessionDatabaseName)
		writer, err := OpenSessionDB(ctx, "sess-read-only-projections", path)
		if err != nil {
			t.Fatalf("OpenSessionDB() error = %v", err)
		}
		for _, turnID := range []string{"turn-1", "turn-2", "turn-3"} {
			if err := writer.Record(ctx, SessionEvent{
				TurnID:    turnID,
				Type:      "agent_message",
				AgentName: "coder",
				Content:   `{"text":"persisted"}`,
			}); err != nil {
				t.Fatalf("Record(%s) error = %v", turnID, err)
			}
		}
		if err := writer.Close(ctx); err != nil {
			t.Fatalf("Close(writer) error = %v", err)
		}

		reader, err := OpenSessionDBReadOnly(ctx, "sess-read-only-projections", path)
		if err != nil {
			t.Fatalf("OpenSessionDBReadOnly() error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := reader.Close(testutil.Context(t)); closeErr != nil {
				t.Fatalf("Close(reader) error = %v", closeErr)
			}
		})

		query := EventQuery{BeforeSequence: 3, Limit: 1}
		fullEvents, err := reader.Query(ctx, query)
		if err != nil {
			t.Fatalf("Query() error = %v", err)
		}
		metadata, err := reader.QueryEventMetadata(ctx, query)
		if err != nil {
			t.Fatalf("QueryEventMetadata() error = %v", err)
		}
		if got, want := eventSequences(fullEvents), []int64{2}; !equalInt64Slices(got, want) {
			t.Fatalf("Query() sequences = %#v, want %#v", got, want)
		}
		if got, want := metadataEventSequences(metadata), []int64{2}; !equalInt64Slices(got, want) {
			t.Fatalf("QueryEventMetadata() sequences = %#v, want %#v", got, want)
		}
	})

	t.Run("Should retry transient SQLite locks while opening", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), SessionDatabaseName)
		transientLock := errors.New("database is locked")
		calls := 0

		reader, err := openSessionDBReadOnlyWithRetry(
			ctx,
			"sess-read-only-locked",
			path,
			func(context.Context, string, string) (*ReadOnlySessionDB, error) {
				calls++
				if calls < 3 {
					return nil, transientLock
				}
				return &ReadOnlySessionDB{sessionID: "sess-read-only-locked"}, nil
			},
			func(err error) bool {
				return errors.Is(err, transientLock)
			},
			newReadOnlyOpenConfig([]ReadOnlyOpenOption{
				WithReadOnlyOpenRetry(3, time.Nanosecond, time.Nanosecond),
			}),
		)
		if err != nil {
			t.Fatalf("openSessionDBReadOnlyWithRetry() error = %v", err)
		}
		if reader == nil || reader.sessionID != "sess-read-only-locked" {
			t.Fatalf("openSessionDBReadOnlyWithRetry() reader = %#v, want session id", reader)
		}
		if got, want := calls, 3; got != want {
			t.Fatalf("openSessionDBReadOnlyWithRetry() calls = %d, want %d", got, want)
		}
	})
}

func TestSessionDBClear(t *testing.T) {
	t.Parallel()

	t.Run("Should serialize through the writer queue", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		sessionDB := &SessionDB{
			writeCh: make(chan sessionWriteRequest, 1),
		}
		sessionDB.state.Store(sessionStateOpen)

		received := make(chan sessionWriteKind, 1)
		done := make(chan struct{})
		go func() {
			defer close(done)
			req, ok := <-sessionDB.writeCh
			if !ok {
				return
			}
			received <- req.kind
			req.result <- sessionWriteResult{}
		}()
		t.Cleanup(func() {
			close(sessionDB.writeCh)
			<-done
		})

		if err := sessionDB.Clear(ctx); err != nil {
			t.Fatalf("Clear() error = %v", err)
		}
		select {
		case got := <-received:
			if got != sessionWriteClear {
				t.Fatalf("Clear() queued kind = %d, want %d", got, sessionWriteClear)
			}
		case <-ctx.Done():
			t.Fatalf("Clear() writer request not observed: %v", ctx.Err())
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

	t.Run("Should not split a turn when after_sequence falls inside it", func(t *testing.T) {
		t.Parallel()

		sessionDB := openTestSessionDB(t, "sess-history-after")
		input := []SessionEvent{
			{TurnID: "turn-a", Type: "agent_message", AgentName: "coder", Content: `{"text":"one"}`},
			{TurnID: "turn-a", Type: "tool_result", AgentName: "coder", Content: `{"tool":"ls"}`},
			{TurnID: "turn-b", Type: "agent_message", AgentName: "coder", Content: `{"text":"two"}`},
			{TurnID: "turn-b", Type: "tool_result", AgentName: "coder", Content: `{"tool":"pwd"}`},
			{TurnID: "turn-c", Type: "agent_message", AgentName: "coder", Content: `{"text":"three"}`},
		}
		for _, event := range input {
			if err := sessionDB.Record(testutil.Context(t), event); err != nil {
				t.Fatalf("Record() error = %v", err)
			}
		}

		history, err := sessionDB.History(testutil.Context(t), EventQuery{AfterSequence: 3})
		if err != nil {
			t.Fatalf("History() error = %v", err)
		}
		if got, want := len(history), 1; got != want {
			t.Fatalf("len(history) = %d, want %d", got, want)
		}
		if history[0].TurnID != "turn-c" {
			t.Fatalf("history[0].TurnID = %q, want turn-c", history[0].TurnID)
		}
		if gotSeqs := eventSequences(history[0].Events); !equalInt64Slices(gotSeqs, []int64{5}) {
			t.Fatalf("turn-c sequences = %#v, want %#v", gotSeqs, []int64{5})
		}
	})

	t.Run("Should read bounded history in chunks without splitting returned turns", func(t *testing.T) {
		t.Parallel()

		events := make([]store.SessionEvent, 0, 302)
		for sequence := int64(1); sequence <= 300; sequence++ {
			events = append(events, store.SessionEvent{
				ID:       fmt.Sprintf("event-%d", sequence),
				Sequence: sequence,
				TurnID:   "turn-a",
				Type:     "agent_message",
			})
		}
		events = append(events,
			store.SessionEvent{ID: "event-301", Sequence: 301, TurnID: "turn-b", Type: "agent_message"},
			store.SessionEvent{ID: "event-302", Sequence: 302, TurnID: "turn-c", Type: "agent_message"},
		)

		calls := make([]store.EventQuery, 0)
		queryEvents := func(_ context.Context, query store.EventQuery) ([]store.SessionEvent, error) {
			calls = append(calls, query)
			filtered := make([]store.SessionEvent, 0, len(events))
			for _, event := range events {
				if query.BeforeSequence > 0 && event.Sequence >= query.BeforeSequence {
					continue
				}
				filtered = append(filtered, event)
			}
			if query.Limit > 0 && len(filtered) > query.Limit {
				filtered = filtered[len(filtered)-query.Limit:]
			}
			return append([]store.SessionEvent(nil), filtered...), nil
		}

		history, err := queryTurnHistory(testutil.Context(t), store.EventQuery{Limit: 3}, queryEvents)
		if err != nil {
			t.Fatalf("queryTurnHistory() error = %v", err)
		}
		if got, want := len(calls), 2; got != want {
			t.Fatalf("query calls = %d, want %d", got, want)
		}
		if calls[0].Limit != historyEventChunkSize {
			t.Fatalf("first query limit = %d, want %d", calls[0].Limit, historyEventChunkSize)
		}
		if calls[0].AfterSequence != 0 {
			t.Fatalf("first query after_sequence = %d, want 0", calls[0].AfterSequence)
		}
		if calls[1].BeforeSequence != 47 {
			t.Fatalf("second query before_sequence = %d, want 47", calls[1].BeforeSequence)
		}
		if got, want := len(history), 3; got != want {
			t.Fatalf("len(history) = %d, want %d", got, want)
		}
		if history[0].TurnID != "turn-a" || history[1].TurnID != "turn-b" ||
			history[2].TurnID != "turn-c" {
			t.Fatalf(
				"turn ids = [%q %q %q], want [turn-a turn-b turn-c]",
				history[0].TurnID,
				history[1].TurnID,
				history[2].TurnID,
			)
		}
		if got, want := len(history[0].Events), 300; got != want {
			t.Fatalf("turn-a events = %d, want %d", got, want)
		}
		if gotSeqs := eventSequences(history[0].Events); !equalInt64Slices(gotSeqs[:2], []int64{1, 2}) {
			t.Fatalf("turn-a first sequences = %#v, want %#v", gotSeqs[:2], []int64{1, 2})
		}
	})

	t.Run("Should not split a read-only turn when after_sequence falls inside it", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), SessionDatabaseName)
		writer, err := OpenSessionDB(ctx, "sess-history-read-only-after", path)
		if err != nil {
			t.Fatalf("OpenSessionDB() error = %v", err)
		}
		input := []SessionEvent{
			{TurnID: "turn-a", Type: "agent_message", AgentName: "coder", Content: `{"text":"one"}`},
			{TurnID: "turn-a", Type: "tool_result", AgentName: "coder", Content: `{"tool":"ls"}`},
			{TurnID: "turn-b", Type: "agent_message", AgentName: "coder", Content: `{"text":"two"}`},
			{TurnID: "turn-b", Type: "tool_result", AgentName: "coder", Content: `{"tool":"pwd"}`},
			{TurnID: "turn-c", Type: "agent_message", AgentName: "coder", Content: `{"text":"three"}`},
		}
		for _, event := range input {
			if err := writer.Record(ctx, event); err != nil {
				t.Fatalf("Record() error = %v", err)
			}
		}
		if err := writer.Close(ctx); err != nil {
			t.Fatalf("Close(writer) error = %v", err)
		}

		reader, err := OpenSessionDBReadOnly(ctx, "sess-history-read-only-after", path)
		if err != nil {
			t.Fatalf("OpenSessionDBReadOnly() error = %v", err)
		}
		defer func() {
			if closeErr := reader.Close(testutil.Context(t)); closeErr != nil {
				t.Fatalf("Close(reader) error = %v", closeErr)
			}
		}()

		history, err := reader.History(ctx, EventQuery{AfterSequence: 3})
		if err != nil {
			t.Fatalf("History(read-only) error = %v", err)
		}
		if got, want := len(history), 1; got != want {
			t.Fatalf("len(history) = %d, want %d", got, want)
		}
		if history[0].TurnID != "turn-c" {
			t.Fatalf("history[0].TurnID = %q, want turn-c", history[0].TurnID)
		}
		if gotSeqs := eventSequences(history[0].Events); !equalInt64Slices(gotSeqs, []int64{5}) {
			t.Fatalf("turn-c sequences = %#v, want %#v", gotSeqs, []int64{5})
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

func canonicalStoreEvent(t *testing.T, event acp.AgentEvent, agentName string) SessionEvent {
	t.Helper()

	payload, err := transcript.MarshalAgentEvent(event)
	if err != nil {
		t.Fatalf("MarshalAgentEvent(%s) error = %v", event.Type, err)
	}
	return SessionEvent{
		TurnID:    event.TurnID,
		Type:      event.Type,
		AgentName: agentName,
		Content:   payload,
		Timestamp: event.Timestamp,
	}
}

func assertEventSequences(t *testing.T, events []SessionEvent, want []int64) {
	t.Helper()

	if len(events) != len(want) {
		t.Fatalf("len(events) = %d, want %d", len(events), len(want))
	}
	for idx, event := range events {
		if event.Sequence != want[idx] {
			t.Fatalf("events[%d].Sequence = %d, want %d", idx, event.Sequence, want[idx])
		}
	}
}

func storedAgentText(t *testing.T, event SessionEvent) string {
	t.Helper()

	agentEvent, err := transcript.UnmarshalAgentEvent(event.Content)
	if err != nil {
		t.Fatalf("UnmarshalAgentEvent() error = %v", err)
	}
	return agentEvent.Text
}

func lastEntryText(t *testing.T, entries []transcript.Entry) string {
	t.Helper()

	if len(entries) == 0 {
		t.Fatal("entries is empty, want transcript content")
	}
	return transcript.UIMessageText(entries[len(entries)-1].Message)
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

func metadataEventSequences(events []EventMetadata) []int64 {
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
