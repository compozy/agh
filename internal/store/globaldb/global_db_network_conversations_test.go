package globaldb

import (
	"database/sql"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

const networkConversationMigrationVersion = 21

func TestOpenGlobalDBCreatesNetworkConversationSchema(t *testing.T) {
	t.Parallel()

	t.Run("Should create final conversation tables and indexes on fresh DB", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)

		assertTablesPresent(
			t,
			globalDB.db,
			"network_timeline_log",
			"network_audit_log",
			"network_threads",
			"network_thread_participants",
			"network_direct_rooms",
			"network_work",
		)
		assertIndexesPresent(
			t,
			globalDB.db,
			"network_timeline_log",
			"idx_net_timeline_thread_ts",
			"idx_net_timeline_direct_ts",
			"idx_net_timeline_work_ts",
			"idx_net_timeline_presence_ts",
			"idx_net_timeline_kind_ts",
		)
		assertIndexesPresent(
			t,
			globalDB.db,
			"network_audit_log",
			"idx_net_audit_conversation",
			"idx_net_audit_work",
		)
		assertIndexesPresent(
			t,
			globalDB.db,
			"network_direct_rooms",
			"idx_network_direct_rooms_activity",
			"idx_network_direct_rooms_peer_a",
			"idx_network_direct_rooms_peer_b",
		)
		assertIndexesPresent(
			t,
			globalDB.db,
			"network_work",
			"idx_network_work_conversation",
			"idx_network_work_state",
		)
		assertUniqueIndexColumns(t, globalDB.db, "network_direct_rooms", []string{"channel", "peer_a", "peer_b"})
		assertForeignKeysEnabled(t, globalDB.db)
	})
}

func TestNetworkConversationMigrationRebuildsLegacyTimeline(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve only legacy presence rows and remove flat timeline columns", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		seedLegacyNetworkConversationDatabase(t, path)

		db, err := store.OpenSQLiteDatabase(ctx, path, nil)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase() error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := db.Close(); closeErr != nil {
				t.Fatalf("db.Close() error = %v", closeErr)
			}
		})

		if err := store.RunMigrations(ctx, db, globalSchemaMigrations); err != nil {
			t.Fatalf("RunMigrations() error = %v", err)
		}

		assertTableLacksColumns(t, db, "network_timeline_log", "interaction_id")
		assertIndexesAbsent(
			t,
			db,
			"network_timeline_log",
			"idx_net_timeline_channel_ts",
			"idx_net_timeline_peer_from_ts",
			"idx_net_timeline_peer_to_ts",
		)
		assertIndexesPresent(t, db, "network_timeline_log", "idx_net_timeline_presence_ts")

		rows, err := db.QueryContext(
			ctx,
			`SELECT message_id, kind, surface, thread_id, direct_id, work_id
			FROM network_timeline_log
			ORDER BY message_id ASC`,
		)
		if err != nil {
			t.Fatalf("query migrated timeline error = %v", err)
		}
		defer func() {
			if closeErr := rows.Close(); closeErr != nil {
				t.Fatalf("rows.Close() error = %v", closeErr)
			}
		}()

		gotIDs := make([]string, 0)
		for rows.Next() {
			var (
				messageID string
				kind      string
				surface   sql.NullString
				threadID  sql.NullString
				directID  sql.NullString
				workID    sql.NullString
			)
			if err := rows.Scan(&messageID, &kind, &surface, &threadID, &directID, &workID); err != nil {
				t.Fatalf("rows.Scan() error = %v", err)
			}
			gotIDs = append(gotIDs, messageID)
			if kind != store.NetworkKindGreet && kind != store.NetworkKindWhois {
				t.Fatalf("kind = %q, want only greet/whois", kind)
			}
			if surface.Valid || threadID.Valid || directID.Valid || workID.Valid {
				t.Fatalf("presence row %q retained conversation fields", messageID)
			}
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows.Err() error = %v", err)
		}
		if got, want := strings.Join(gotIDs, ","), "msg_greet_01,msg_whois_01"; got != want {
			t.Fatalf("migrated timeline ids = %q, want %q", got, want)
		}

		assertAppliedMigrationVersion(t, db, networkConversationMigrationVersion)
	})
}

func TestNetworkConversationMigrationReopenAfterRestart(t *testing.T) {
	t.Parallel()

	t.Run(
		"Should upgrade observed task and bridge migration history by appending network migration",
		func(t *testing.T) {
			t.Parallel()

			ctx := testutil.Context(t)
			path := filepath.Join(t.TempDir(), GlobalDatabaseName)
			seedLegacyNetworkConversationDatabase(t, path)

			beforeDB, err := store.OpenSQLiteDatabase(ctx, path, nil)
			if err != nil {
				t.Fatalf("OpenSQLiteDatabase(before) error = %v", err)
			}
			beforeRecords, err := store.AppliedMigrations(ctx, beforeDB)
			if err != nil {
				t.Fatalf("AppliedMigrations(before) error = %v", err)
			}
			if err := beforeDB.Close(); err != nil {
				t.Fatalf("beforeDB.Close() error = %v", err)
			}
			assertAppliedGlobalMigrationPrefix(t, beforeRecords, networkConversationMigrationVersion-1)

			first, err := OpenGlobalDB(ctx, path)
			if err != nil {
				t.Fatalf("OpenGlobalDB(first) error = %v", err)
			}
			firstRecords, err := store.AppliedMigrations(ctx, first.db)
			if err != nil {
				t.Fatalf("AppliedMigrations(first) error = %v", err)
			}
			assertAppliedGlobalMigrationOrder(t, firstRecords)
			for index, before := range beforeRecords {
				if !firstRecords[index].AppliedAt.Equal(before.AppliedAt) {
					t.Fatalf(
						"migration %d applied_at = %s, want unchanged %s",
						before.Version,
						firstRecords[index].AppliedAt,
						before.AppliedAt,
					)
				}
			}
			assertTaskOrchestrationProfileSchema(t, first.db)
			assertReviewGateSchema(t, first.db)
			assertNotificationCursorSchema(t, first.db)
			assertBridgeTaskSubscriptionSchema(t, first.db)
			assertTableLacksColumns(t, first.db, "network_timeline_log", "interaction_id")
			assertTablesPresent(t, first.db, "network_threads", "network_direct_rooms", "network_work", "memory_events")
			if err := first.Close(ctx); err != nil {
				t.Fatalf("Close(first) error = %v", err)
			}

			second, err := OpenGlobalDB(ctx, path)
			if err != nil {
				t.Fatalf("OpenGlobalDB(second) error = %v", err)
			}
			t.Cleanup(func() {
				if closeErr := second.Close(ctx); closeErr != nil {
					t.Fatalf("Close(second) error = %v", closeErr)
				}
			})
			secondRecords, err := store.AppliedMigrations(ctx, second.db)
			if err != nil {
				t.Fatalf("AppliedMigrations(second) error = %v", err)
			}
			assertAppliedGlobalMigrationOrder(t, secondRecords)
			if got, want := len(secondRecords), len(firstRecords); got != want {
				t.Fatalf("len(secondRecords) = %d, want %d", got, want)
			}
			for index, firstRecord := range firstRecords {
				if !secondRecords[index].AppliedAt.Equal(firstRecord.AppliedAt) {
					t.Fatalf(
						"second migration %d applied_at = %s, want unchanged %s",
						firstRecord.Version,
						secondRecords[index].AppliedAt,
						firstRecord.AppliedAt,
					)
				}
			}
		},
	)

	t.Run("Should record migration version and keep schema stable after reopen", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		seedLegacyNetworkConversationDatabase(t, path)

		first, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(first) error = %v", err)
		}
		firstRecords, err := store.AppliedMigrations(ctx, first.db)
		if err != nil {
			t.Fatalf("AppliedMigrations(first) error = %v", err)
		}
		assertAppliedMigrationVersion(t, first.db, networkConversationMigrationVersion)
		if err := first.Close(ctx); err != nil {
			t.Fatalf("Close(first) error = %v", err)
		}

		second, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(second) error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := second.Close(ctx); closeErr != nil {
				t.Fatalf("Close(second) error = %v", closeErr)
			}
		})
		secondRecords, err := store.AppliedMigrations(ctx, second.db)
		if err != nil {
			t.Fatalf("AppliedMigrations(second) error = %v", err)
		}
		if got, want := len(secondRecords), len(firstRecords); got != want {
			t.Fatalf("len(secondRecords) = %d, want %d", got, want)
		}
		if !secondRecords[len(secondRecords)-1].AppliedAt.Equal(firstRecords[len(firstRecords)-1].AppliedAt) {
			t.Fatalf("migration v%d applied_at changed after reopen", networkConversationMigrationVersion)
		}
		assertTableLacksColumns(t, second.db, "network_timeline_log", "interaction_id")
		assertTablesPresent(t, second.db, "network_threads", "network_direct_rooms", "network_work")
	})
}

func TestNetworkConversationConstraints(t *testing.T) {
	t.Parallel()

	t.Run("Should enforce direct room uniqueness and ordered peers", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		assertForeignKeysEnabled(t, globalDB.db)

		insertDirectRoom(
			t,
			globalDB.db,
			"builders",
			"direct_0123456789abcdef0123456789abcdef",
			"coder.sess-abc",
			"reviewer.sess-xyz",
		)
		_, err := globalDB.db.ExecContext(
			ctx,
			`INSERT INTO network_direct_rooms (
				channel, direct_id, peer_a, peer_b, opened_at, last_activity_at
			) VALUES (?, ?, ?, ?, ?, ?)`,
			"builders",
			"direct_fedcba9876543210fedcba9876543210",
			"coder.sess-abc",
			"reviewer.sess-xyz",
			"2026-05-05T12:00:00Z",
			"2026-05-05T12:00:00Z",
		)
		requireSQLiteConstraintError(t, err)

		_, err = globalDB.db.ExecContext(
			ctx,
			`INSERT INTO network_direct_rooms (
				channel, direct_id, peer_a, peer_b, opened_at, last_activity_at
			) VALUES (?, ?, ?, ?, ?, ?)`,
			"builders",
			"direct_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"reviewer.sess-xyz",
			"coder.sess-abc",
			"2026-05-05T12:00:00Z",
			"2026-05-05T12:00:00Z",
		)
		requireSQLiteConstraintError(t, err)
	})

	t.Run("Should reject missing work containers and restrict referenced deletes", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		assertForeignKeysEnabled(t, globalDB.db)

		_, err := globalDB.db.ExecContext(
			ctx,
			`INSERT INTO network_work (
				work_id, channel, surface, thread_id, opened_by_peer_id, state, opened_at, last_activity_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			"work_missing_thread",
			"builders",
			store.NetworkSurfaceThread,
			"thread_missing",
			"coder.sess-abc",
			store.NetworkWorkStateSubmitted,
			"2026-05-05T12:00:00Z",
			"2026-05-05T12:00:00Z",
		)
		requireSQLiteConstraintError(t, err)

		insertThread(t, globalDB.db, "builders", "thread_restrict", "msg_root_restrict")
		insertWorkForThread(t, globalDB.db, "work_thread_restrict", "builders", "thread_restrict")
		_, err = globalDB.db.ExecContext(
			ctx,
			`DELETE FROM network_threads WHERE channel = ? AND thread_id = ?`,
			"builders",
			"thread_restrict",
		)
		requireSQLiteConstraintError(t, err)

		insertDirectRoom(
			t,
			globalDB.db,
			"builders",
			"direct_0123456789abcdef0123456789abcdef",
			"coder.sess-abc",
			"reviewer.sess-xyz",
		)
		insertWorkForDirect(
			t,
			globalDB.db,
			"work_direct_restrict",
			"builders",
			"direct_0123456789abcdef0123456789abcdef",
		)
		_, err = globalDB.db.ExecContext(
			ctx,
			`DELETE FROM network_direct_rooms WHERE channel = ? AND direct_id = ?`,
			"builders",
			"direct_0123456789abcdef0123456789abcdef",
		)
		requireSQLiteConstraintError(t, err)
	})

	t.Run("Should cascade thread participants when an unreferenced thread is deleted", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		assertForeignKeysEnabled(t, globalDB.db)

		insertThread(t, globalDB.db, "builders", "thread_cascade", "msg_root_cascade")
		if _, err := globalDB.db.ExecContext(
			ctx,
			`INSERT INTO network_thread_participants (
				channel, thread_id, peer_id, first_message_id, first_seen_at, last_seen_at
			) VALUES (?, ?, ?, ?, ?, ?)`,
			"builders",
			"thread_cascade",
			"coder.sess-abc",
			"msg_root_cascade",
			"2026-05-05T12:00:00Z",
			"2026-05-05T12:00:00Z",
		); err != nil {
			t.Fatalf("insert thread participant error = %v", err)
		}
		if _, err := globalDB.db.ExecContext(
			ctx,
			`DELETE FROM network_threads WHERE channel = ? AND thread_id = ?`,
			"builders",
			"thread_cascade",
		); err != nil {
			t.Fatalf("delete unreferenced thread error = %v", err)
		}

		var count int
		if err := globalDB.db.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM network_thread_participants WHERE channel = ? AND thread_id = ?`,
			"builders",
			"thread_cascade",
		).Scan(&count); err != nil {
			t.Fatalf("count thread participants error = %v", err)
		}
		if count != 0 {
			t.Fatalf("participant count after cascade = %d, want 0", count)
		}
	})
}

func seedLegacyNetworkConversationDatabase(t *testing.T, path string) {
	t.Helper()

	ctx := testutil.Context(t)
	db, err := store.OpenSQLiteDatabase(ctx, path, nil)
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase(legacy) error = %v", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("legacy db.Close() error = %v", closeErr)
		}
	}()

	if err := store.RunMigrations(ctx, db, globalSchemaMigrations[:networkConversationMigrationVersion-1]); err != nil {
		t.Fatalf("RunMigrations(legacy seed) error = %v", err)
	}

	statements := []string{
		`DROP TABLE IF EXISTS network_timeline_log;`,
		`DROP TABLE IF EXISTS network_audit_log;`,
		`CREATE TABLE network_timeline_log (
				message_id     TEXT PRIMARY KEY,
				session_id     TEXT,
				channel        TEXT NOT NULL,
				direction      TEXT NOT NULL,
				peer_from      TEXT NOT NULL,
				peer_to        TEXT,
				kind           TEXT NOT NULL,
				interaction_id TEXT,
				reply_to       TEXT,
				trace_id       TEXT,
				causation_id   TEXT,
				intent         TEXT,
				text           TEXT,
				preview_text   TEXT NOT NULL DEFAULT '',
				body_json      TEXT NOT NULL,
				timestamp      TEXT NOT NULL
			);`,
		`CREATE INDEX idx_net_timeline_channel_ts ON network_timeline_log(channel, timestamp, message_id);`,
		`CREATE INDEX idx_net_timeline_peer_from_ts ON network_timeline_log(peer_from, timestamp, message_id);`,
		`CREATE INDEX idx_net_timeline_peer_to_ts ON network_timeline_log(peer_to, timestamp, message_id);`,
		`CREATE INDEX idx_net_timeline_kind_ts ON network_timeline_log(kind, timestamp, message_id);`,
		`CREATE TABLE network_audit_log (
				id         TEXT PRIMARY KEY,
				session_id TEXT NOT NULL,
				direction  TEXT NOT NULL,
				kind       TEXT NOT NULL,
				channel    TEXT NOT NULL,
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
			t.Fatalf("apply legacy network statement error = %v", err)
		}
	}

	legacyMessages := []struct {
		id     string
		kind   string
		workID any
	}{
		{id: "msg_greet_01", kind: store.NetworkKindGreet},
		{id: "msg_whois_01", kind: store.NetworkKindWhois},
		{id: "msg_say_01", kind: store.NetworkKindSay, workID: "work_legacy_say"},
		{id: "msg_direct_01", kind: "direct", workID: "work_legacy_direct"},
		{id: "msg_receipt_01", kind: store.NetworkKindReceipt, workID: "work_legacy_receipt"},
	}
	for index, message := range legacyMessages {
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO network_timeline_log (
				message_id, session_id, channel, direction, peer_from, peer_to, kind, interaction_id,
				reply_to, trace_id, causation_id, intent, text, preview_text, body_json, timestamp
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			message.id,
			"sess-legacy",
			"builders",
			"received",
			"peer-a",
			nil,
			message.kind,
			message.workID,
			nil,
			nil,
			nil,
			nil,
			"legacy",
			"legacy",
			`{"text":"legacy"}`,
			"2026-05-05T12:00:0"+string(rune('0'+index))+"Z",
		); err != nil {
			t.Fatalf("insert legacy timeline %q error = %v", message.id, err)
		}
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO network_audit_log (
			id, session_id, direction, kind, channel, peer_from, peer_to, message_id, reason, size, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"naud_legacy_01",
		"sess-legacy",
		"received",
		store.NetworkKindGreet,
		"builders",
		"peer-a",
		nil,
		"msg_greet_01",
		nil,
		64,
		"2026-05-05T12:00:09Z",
	); err != nil {
		t.Fatalf("insert legacy audit error = %v", err)
	}
}

func insertThread(t *testing.T, db *sql.DB, channel string, threadID string, rootMessageID string) {
	t.Helper()

	if _, err := db.ExecContext(
		testutil.Context(t),
		`INSERT INTO network_threads (
			channel, thread_id, root_message_id, opened_by_peer_id, opened_at, last_activity_at
		) VALUES (?, ?, ?, ?, ?, ?)`,
		channel,
		threadID,
		rootMessageID,
		"coder.sess-abc",
		"2026-05-05T12:00:00Z",
		"2026-05-05T12:00:00Z",
	); err != nil {
		t.Fatalf("insert network thread %q error = %v", threadID, err)
	}
}

func insertDirectRoom(t *testing.T, db *sql.DB, channel string, directID string, peerA string, peerB string) {
	t.Helper()

	if _, err := db.ExecContext(
		testutil.Context(t),
		`INSERT INTO network_direct_rooms (
			channel, direct_id, peer_a, peer_b, opened_at, last_activity_at
		) VALUES (?, ?, ?, ?, ?, ?)`,
		channel,
		directID,
		peerA,
		peerB,
		"2026-05-05T12:00:00Z",
		"2026-05-05T12:00:00Z",
	); err != nil {
		t.Fatalf("insert network direct room %q error = %v", directID, err)
	}
}

func insertWorkForThread(t *testing.T, db *sql.DB, workID string, channel string, threadID string) {
	t.Helper()

	if _, err := db.ExecContext(
		testutil.Context(t),
		`INSERT INTO network_work (
			work_id, channel, surface, thread_id, opened_by_peer_id, state, opened_at, last_activity_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		workID,
		channel,
		store.NetworkSurfaceThread,
		threadID,
		"coder.sess-abc",
		store.NetworkWorkStateSubmitted,
		"2026-05-05T12:00:00Z",
		"2026-05-05T12:00:00Z",
	); err != nil {
		t.Fatalf("insert network thread work %q error = %v", workID, err)
	}
}

func insertWorkForDirect(t *testing.T, db *sql.DB, workID string, channel string, directID string) {
	t.Helper()

	if _, err := db.ExecContext(
		testutil.Context(t),
		`INSERT INTO network_work (
			work_id, channel, surface, direct_id, opened_by_peer_id, state, opened_at, last_activity_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		workID,
		channel,
		store.NetworkSurfaceDirect,
		directID,
		"coder.sess-abc",
		store.NetworkWorkStateSubmitted,
		"2026-05-05T12:00:00Z",
		"2026-05-05T12:00:00Z",
	); err != nil {
		t.Fatalf("insert network direct work %q error = %v", workID, err)
	}
}

func assertForeignKeysEnabled(t *testing.T, db *sql.DB) {
	t.Helper()

	var enabled int
	if err := db.QueryRowContext(testutil.Context(t), `PRAGMA foreign_keys`).Scan(&enabled); err != nil {
		t.Fatalf("PRAGMA foreign_keys error = %v", err)
	}
	if enabled != 1 {
		t.Fatalf("PRAGMA foreign_keys = %d, want 1", enabled)
	}
}

func assertTableLacksColumns(t *testing.T, db *sql.DB, table string, columns ...string) {
	t.Helper()

	got := tableColumnSet(t, db, table)
	for _, column := range columns {
		if _, ok := got[column]; ok {
			t.Fatalf("table %s unexpectedly has column %q", table, column)
		}
	}
}

func tableColumnSet(t *testing.T, db *sql.DB, table string) map[string]struct{} {
	t.Helper()

	rows, err := db.QueryContext(testutil.Context(t), "PRAGMA table_info("+table+")")
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s) error = %v", table, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			t.Fatalf("rows.Close(table_info %s) error = %v", table, closeErr)
		}
	}()

	columns := make(map[string]struct{})
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &primaryKey); err != nil {
			t.Fatalf("scan table_info(%s) error = %v", table, err)
		}
		columns[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(table_info %s) error = %v", table, err)
	}
	return columns
}

func assertIndexesAbsent(t *testing.T, db *sql.DB, table string, wantAbsent ...string) {
	t.Helper()

	indexes := tableIndexSet(t, db, table)
	for _, indexName := range wantAbsent {
		if _, ok := indexes[indexName]; ok {
			t.Fatalf("index %q unexpectedly present on %s", indexName, table)
		}
	}
}

func tableIndexSet(t *testing.T, db *sql.DB, table string) map[string]struct{} {
	t.Helper()

	rows, err := db.QueryContext(testutil.Context(t), "PRAGMA index_list("+table+")")
	if err != nil {
		t.Fatalf("PRAGMA index_list(%s) error = %v", table, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			t.Fatalf("rows.Close(index_list %s) error = %v", table, closeErr)
		}
	}()

	indexes := make(map[string]struct{})
	for rows.Next() {
		var (
			seq     int
			name    string
			unique  int
			origin  string
			partial int
		)
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("scan index_list(%s) error = %v", table, err)
		}
		indexes[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(index_list %s) error = %v", table, err)
	}
	return indexes
}

func assertUniqueIndexColumns(t *testing.T, db *sql.DB, table string, want []string) {
	t.Helper()

	rows, err := db.QueryContext(testutil.Context(t), "PRAGMA index_list("+table+")")
	if err != nil {
		t.Fatalf("PRAGMA index_list(%s) error = %v", table, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			t.Fatalf("rows.Close(index_list %s) error = %v", table, closeErr)
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
			t.Fatalf("scan index_list(%s) error = %v", table, err)
		}
		if unique != 1 {
			continue
		}
		if slices.Equal(indexColumns(t, db, name), want) {
			return
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(index_list %s) error = %v", table, err)
	}
	t.Fatalf("unique index on %s columns %#v not found", table, want)
}

func indexColumns(t *testing.T, db *sql.DB, indexName string) []string {
	t.Helper()

	rows, err := db.QueryContext(testutil.Context(t), "PRAGMA index_info("+indexName+")")
	if err != nil {
		t.Fatalf("PRAGMA index_info(%s) error = %v", indexName, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			t.Fatalf("rows.Close(index_info %s) error = %v", indexName, closeErr)
		}
	}()

	columns := make([]string, 0)
	for rows.Next() {
		var (
			seqno int
			cid   int
			name  string
		)
		if err := rows.Scan(&seqno, &cid, &name); err != nil {
			t.Fatalf("scan index_info(%s) error = %v", indexName, err)
		}
		columns = append(columns, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(index_info %s) error = %v", indexName, err)
	}
	return columns
}

func assertAppliedMigrationVersion(t *testing.T, db *sql.DB, version int) {
	t.Helper()

	records, err := store.AppliedMigrations(testutil.Context(t), db)
	if err != nil {
		t.Fatalf("AppliedMigrations() error = %v", err)
	}
	for _, record := range records {
		if record.Version == version {
			return
		}
	}
	t.Fatalf("schema migration version %d missing from %#v", version, records)
}

func requireSQLiteConstraintError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want sqlite constraint error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "constraint") {
		t.Fatalf("error = %v, want sqlite constraint failure", err)
	}
}
