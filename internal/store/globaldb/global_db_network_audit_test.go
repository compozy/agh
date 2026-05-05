package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestOpenGlobalDBCreatesNetworkAuditLogSchema(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	assertTablesPresent(t, globalDB.db, "network_audit_log")
	assertTableColumns(t, globalDB.db, "network_audit_log", []string{
		"id",
		"session_id",
		"direction",
		"kind",
		"channel",
		"surface",
		"thread_id",
		"direct_id",
		"work_id",
		"peer_from",
		"peer_to",
		"message_id",
		"reason",
		"size",
		"timestamp",
	})
	assertTableHasNoForeignKeys(t, globalDB.db, "network_audit_log")
}

func TestGlobalDBWriteAndListNetworkAudit(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-network-audit")

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	globalDB.now = func() time.Time { return now }

	if err := globalDB.WriteNetworkAudit(testutil.Context(t), store.NetworkAuditEntry{
		SessionID: "sess-network-audit",
		Direction: "sent",
		Kind:      "say",
		Channel:   "builders",
		Surface:   store.NetworkSurfaceDirect,
		DirectID:  "direct_0123456789abcdef0123456789abcdef",
		WorkID:    "work_patch_42",
		PeerFrom:  "coder.sess-network-audit",
		PeerTo:    "reviewer.sess-xyz",
		MessageID: "msg_direct_01",
		Size:      128,
	}); err != nil {
		t.Fatalf("WriteNetworkAudit(sent) error = %v", err)
	}

	if err := globalDB.WriteNetworkAudit(testutil.Context(t), store.NetworkAuditEntry{
		SessionID: "sess-network-audit",
		Direction: "rejected",
		Kind:      "receipt",
		Channel:   "builders",
		PeerFrom:  "reviewer.sess-xyz",
		MessageID: "msg_receipt_01",
		Reason:    "not_found",
		Size:      64,
		Timestamp: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("WriteNetworkAudit(rejected) error = %v", err)
	}

	entries, err := globalDB.ListNetworkAudit(testutil.Context(t), store.NetworkAuditQuery{
		SessionID: "sess-network-audit",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListNetworkAudit() error = %v", err)
	}
	if got, want := len(entries), 2; got != want {
		t.Fatalf("len(entries) = %d, want %d", got, want)
	}

	if got, want := entries[0].Direction, "sent"; got != want {
		t.Fatalf("entries[0].Direction = %q, want %q", got, want)
	}
	if got, want := entries[0].Timestamp, now; !got.Equal(want) {
		t.Fatalf("entries[0].Timestamp = %s, want %s", got, want)
	}
	if got, want := entries[0].PeerTo, "reviewer.sess-xyz"; got != want {
		t.Fatalf("entries[0].PeerTo = %q, want %q", got, want)
	}
	if got, want := entries[0].Surface, store.NetworkSurfaceDirect; got != want {
		t.Fatalf("entries[0].Surface = %q, want %q", got, want)
	}
	if got, want := entries[0].DirectID, "direct_0123456789abcdef0123456789abcdef"; got != want {
		t.Fatalf("entries[0].DirectID = %q, want %q", got, want)
	}
	if got, want := entries[0].WorkID, "work_patch_42"; got != want {
		t.Fatalf("entries[0].WorkID = %q, want %q", got, want)
	}

	if got, want := entries[1].Direction, "rejected"; got != want {
		t.Fatalf("entries[1].Direction = %q, want %q", got, want)
	}
	if got, want := entries[1].Reason, "not_found"; got != want {
		t.Fatalf("entries[1].Reason = %q, want %q", got, want)
	}
}

func TestGlobalDBWriteNetworkAuditAllowsUnknownSessionID(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	if err := globalDB.WriteNetworkAudit(testutil.Context(t), store.NetworkAuditEntry{
		SessionID: "sess-network-unknown",
		Direction: "sent",
		Kind:      "greet",
		Channel:   "builders",
		PeerFrom:  "coder.sess-network-unknown",
		MessageID: "msg_greet_01",
		Size:      32,
	}); err != nil {
		t.Fatalf("WriteNetworkAudit(unknown session) error = %v", err)
	}

	entries, err := globalDB.ListNetworkAudit(testutil.Context(t), store.NetworkAuditQuery{
		SessionID: "sess-network-unknown",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListNetworkAudit(unknown session) error = %v", err)
	}
	if got, want := len(entries), 1; got != want {
		t.Fatalf("len(entries) = %d, want %d", got, want)
	}
	if got, want := entries[0].MessageID, "msg_greet_01"; got != want {
		t.Fatalf("entries[0].MessageID = %q, want %q", got, want)
	}
}

func TestGlobalDBNetworkAuditGuardClauses(t *testing.T) {
	t.Parallel()

	var nilDB *GlobalDB
	if err := nilDB.WriteNetworkAudit(testutil.Context(t), store.NetworkAuditEntry{}); err == nil {
		t.Fatal("WriteNetworkAudit(nil receiver) error = nil, want non-nil")
	}
	if _, err := nilDB.ListNetworkAudit(testutil.Context(t), store.NetworkAuditQuery{}); err == nil {
		t.Fatal("ListNetworkAudit(nil receiver) error = nil, want non-nil")
	}

	globalDB := openTestGlobalDB(t)
	if err := globalDB.WriteNetworkAudit(nilGlobalContext(), store.NetworkAuditEntry{}); err == nil {
		t.Fatal("WriteNetworkAudit(nil ctx) error = nil, want non-nil")
	}
	if _, err := globalDB.ListNetworkAudit(nilGlobalContext(), store.NetworkAuditQuery{}); err == nil {
		t.Fatal("ListNetworkAudit(nil ctx) error = nil, want non-nil")
	}
	if err := globalDB.Close(testutil.Context(t)); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := globalDB.WriteNetworkAudit(
		testutil.Context(t),
		store.NetworkAuditEntry{},
	); !errors.Is(
		err,
		store.ErrClosed,
	) {
		t.Fatalf("WriteNetworkAudit(after close) error = %v, want ErrClosed", err)
	}
}

func TestGlobalDBWriteNetworkAuditRejectsWhitechannelPaddedDirection(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-network-direction")

	err := globalDB.WriteNetworkAudit(testutil.Context(t), store.NetworkAuditEntry{
		SessionID: "sess-network-direction",
		Direction: " sent ",
		Kind:      "direct",
		Channel:   "builders",
		PeerFrom:  "coder.sess-network-direction",
		MessageID: "msg_direction_01",
		Size:      12,
	})
	if err == nil {
		t.Fatal("WriteNetworkAudit(whitespace direction) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "must not contain surrounding whitespace") {
		t.Fatalf("WriteNetworkAudit(whitespace direction) error = %v, want whitespace validation context", err)
	}
}

func TestGlobalDBListNetworkAuditWrapsTimestampParseFailures(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-network-bad-timestamp")

	if _, err := globalDB.db.ExecContext(
		testutil.Context(t),
		`INSERT INTO network_audit_log (
			id, session_id, direction, kind, channel, surface, thread_id, direct_id, work_id,
			peer_from, peer_to, message_id, reason, size, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"naud_bad_timestamp",
		"sess-network-bad-timestamp",
		"sent",
		"say",
		"builders",
		store.NetworkSurfaceThread,
		"thread_bad_timestamp",
		nil,
		nil,
		"coder.sess-network-bad-timestamp",
		nil,
		"msg_bad_timestamp_01",
		nil,
		1,
		"not-a-timestamp",
	); err != nil {
		t.Fatalf("ExecContext(insert invalid audit row) error = %v", err)
	}

	_, err := globalDB.ListNetworkAudit(testutil.Context(t), store.NetworkAuditQuery{
		SessionID: "sess-network-bad-timestamp",
		Limit:     10,
	})
	if err == nil {
		t.Fatal("ListNetworkAudit(invalid timestamp) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "parse network audit timestamp") {
		t.Fatalf("ListNetworkAudit(invalid timestamp) error = %v, want wrapped timestamp parse context", err)
	}
}

func TestOpenGlobalDBMigratesNetworkAuditSchemaWithoutSessionForeignKey(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), GlobalDatabaseName)
	seedLegacyNetworkAuditSchema(t, dbPath)

	globalDB, err := OpenGlobalDB(testutil.Context(t), dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(migrate network audit schema) error = %v", err)
	}
	t.Cleanup(func() {
		if err := globalDB.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	assertTableHasNoForeignKeys(t, globalDB.db, "network_audit_log")

	entries, err := globalDB.ListNetworkAudit(testutil.Context(t), store.NetworkAuditQuery{
		SessionID: "sess-network-legacy",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListNetworkAudit(legacy session) error = %v", err)
	}
	if got, want := len(entries), 1; got != want {
		t.Fatalf("len(entries) = %d, want %d", got, want)
	}
	if got, want := entries[0].MessageID, "msg_legacy_01"; got != want {
		t.Fatalf("entries[0].MessageID = %q, want %q", got, want)
	}

	if err := globalDB.WriteNetworkAudit(testutil.Context(t), store.NetworkAuditEntry{
		SessionID: "sess-network-after-migration",
		Direction: "received",
		Kind:      "greet",
		Channel:   "builders",
		PeerFrom:  "coder.sess-network-after-migration",
		MessageID: "msg_after_migration_01",
		Size:      64,
	}); err != nil {
		t.Fatalf("WriteNetworkAudit(after migration unknown session) error = %v", err)
	}
}

func seedLegacyNetworkAuditSchema(t *testing.T, path string) {
	t.Helper()

	db, err := store.OpenSQLiteDatabase(testutil.Context(t), path, func(ctx context.Context, db *sql.DB) error {
		statements := []string{
			`CREATE TABLE workspaces (
				id            TEXT PRIMARY KEY,
				root_dir      TEXT NOT NULL UNIQUE,
				add_dirs      TEXT NOT NULL DEFAULT '[]',
				name          TEXT NOT NULL UNIQUE,
				default_agent TEXT DEFAULT '',
				created_at    TEXT NOT NULL,
				updated_at    TEXT NOT NULL
			);`,
			`CREATE TABLE sessions (
				id             TEXT PRIMARY KEY,
				name           TEXT,
				agent_name     TEXT NOT NULL,
				workspace_id   TEXT NOT NULL REFERENCES workspaces(id),
				session_type   TEXT NOT NULL DEFAULT 'user',
				channel          TEXT NOT NULL DEFAULT '',
				state          TEXT NOT NULL,
				acp_session_id TEXT,
				stop_reason    TEXT,
				stop_detail    TEXT,
				created_at     TEXT NOT NULL,
				updated_at     TEXT NOT NULL
			);`,
			`CREATE TABLE network_audit_log (
				id         TEXT PRIMARY KEY,
				session_id TEXT NOT NULL REFERENCES sessions(id),
				direction  TEXT NOT NULL,
				kind       TEXT NOT NULL,
				channel      TEXT NOT NULL,
				peer_from  TEXT NOT NULL,
				peer_to    TEXT,
				message_id TEXT NOT NULL,
				reason     TEXT,
				size       INTEGER NOT NULL,
				timestamp  TEXT NOT NULL
			);`,
			`CREATE INDEX idx_net_audit_ts ON network_audit_log(timestamp);`,
			`CREATE INDEX idx_net_audit_session ON network_audit_log(session_id);`,
		}
		for _, stmt := range statements {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return err
			}
		}

		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO workspaces (id, root_dir, add_dirs, name, default_agent, created_at, updated_at)
			 VALUES (?, ?, '[]', ?, '', ?, ?)`,
			"ws-network-legacy",
			filepath.Join(t.TempDir(), "legacy-workspace"),
			"network-legacy",
			"2026-04-11T12:00:00.000000000Z",
			"2026-04-11T12:00:00.000000000Z",
		); err != nil {
			return err
		}
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO sessions (
				id, name, agent_name, workspace_id, session_type, channel, state, acp_session_id, stop_reason, stop_detail, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"sess-network-legacy",
			nil,
			"coder",
			"ws-network-legacy",
			"user",
			"builders",
			"active",
			nil,
			nil,
			nil,
			"2026-04-11T12:01:00.000000000Z",
			"2026-04-11T12:01:00.000000000Z",
		); err != nil {
			return err
		}
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO network_audit_log (
				id, session_id, direction, kind, channel, peer_from, peer_to, message_id, reason, size, timestamp
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"naud_legacy_01",
			"sess-network-legacy",
			"sent",
			"greet",
			"builders",
			"coder.sess-network-legacy",
			nil,
			"msg_legacy_01",
			nil,
			44,
			"2026-04-11T12:02:00.000000000Z",
		); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		t.Fatalf("seedLegacyNetworkAuditSchema() error = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("seedLegacyNetworkAuditSchema close error = %v", err)
	}
}

func assertTableHasNoForeignKeys(t *testing.T, db *sql.DB, table string) {
	t.Helper()

	rows, err := db.QueryContext(testutil.Context(t), `PRAGMA foreign_key_list(`+table+`)`)
	if err != nil {
		t.Fatalf("PRAGMA foreign_key_list(%s) error = %v", table, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if rows.Next() {
		var (
			id       int
			seq      int
			refTable string
			from     string
			to       string
			onUpdate string
			onDelete string
			match    string
		)
		if err := rows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			t.Fatalf("Scan(foreign_key_list %s) error = %v", table, err)
		}
		t.Fatalf("foreign_key_list(%s) unexpectedly references %q via %q -> %q", table, refTable, from, to)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(foreign_key_list %s) error = %v", table, err)
	}
}
