package globaldb

import (
	"database/sql"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
)

const (
	networkConversationMigrationVersion = 21
	networkConversationTestWorkspaceID  = "ws-network-conversation"
)

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
			"idx_net_timeline_thread_sequence",
			"idx_net_timeline_direct_sequence",
			"idx_net_timeline_work_sequence",
			"idx_net_timeline_presence_sequence",
			"idx_net_timeline_kind_sequence",
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
		assertUniqueIndexColumns(
			t,
			globalDB.db,
			"network_direct_rooms",
			[]string{"workspace_id", "channel", "peer_a", "peer_b"},
		)
		assertForeignKeysEnabled(t, globalDB.db)
	})
}

func TestNetworkConversationMigrationRebuildsLegacyTimeline(t *testing.T) {
	t.Parallel()

	t.Run("Should remove legacy network timeline rows during the workspace-qualified hard cut", func(t *testing.T) {
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
			"idx_net_timeline_thread_ts",
			"idx_net_timeline_direct_ts",
			"idx_net_timeline_work_ts",
			"idx_net_timeline_presence_ts",
			"idx_net_timeline_kind_ts",
		)
		assertIndexesPresent(t, db, "network_timeline_log", "idx_net_timeline_presence_sequence")

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
		if got, want := strings.Join(gotIDs, ","), ""; got != want {
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

func TestNetworkTimelineSequenceMigration(t *testing.T) {
	t.Parallel()

	t.Run("Should create causal sequence schema on a fresh database", func(t *testing.T) {
		t.Parallel()

		globalDB := openFreshTestGlobalDB(t)
		assertNetworkTimelineSequenceSchema(t, globalDB.db)
		assertAppliedMigrationVersion(t, globalDB.db, networkTimelineSequenceMigration.Version)
	})

	t.Run("Should preserve v70 rows and projections across upgrade and reopen", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		db, err := store.OpenSQLiteDatabase(ctx, path, nil)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase(v70) error = %v", err)
		}
		t.Cleanup(func() {
			if db != nil {
				if closeErr := db.Close(); closeErr != nil {
					t.Errorf("db.Close(cleanup) error = %v", closeErr)
				}
			}
		})

		preSequence := globalSchemaMigrations[:globalMigrationIndex(t, networkTimelineSequenceMigration.Version)]
		if err := store.RunMigrations(ctx, db, preSequence); err != nil {
			t.Fatalf("RunMigrations(v70) error = %v", err)
		}
		rowIDs := seedNetworkTimelineSequenceV70(t, db)

		if err := store.RunMigrations(ctx, db, globalSchemaMigrations); err != nil {
			t.Fatalf("RunMigrations(v71) error = %v", err)
		}
		assertNetworkTimelineSequenceSchema(t, db)
		assertNetworkTimelineSequenceRows(t, db, rowIDs)
		assertNetworkTimelineWorkspaceScopedUniqueness(t, db)
		assertNetworkSequenceProjectionBackfill(t, db)
		assertNetworkTimelineSequenceContinues(t, db)

		firstRecords, err := store.AppliedMigrations(ctx, db)
		if err != nil {
			t.Fatalf("AppliedMigrations(first) error = %v", err)
		}
		if err := store.RunMigrations(ctx, db, globalSchemaMigrations); err != nil {
			t.Fatalf("RunMigrations(idempotent) error = %v", err)
		}
		secondRecords, err := store.AppliedMigrations(ctx, db)
		if err != nil {
			t.Fatalf("AppliedMigrations(second) error = %v", err)
		}
		if got, want := len(secondRecords), len(firstRecords); got != want {
			t.Fatalf("len(secondRecords) = %d, want %d", got, want)
		}
		if !secondRecords[len(secondRecords)-1].AppliedAt.Equal(firstRecords[len(firstRecords)-1].AppliedAt) {
			t.Fatal("v71 applied_at changed on idempotent migration run")
		}

		if err := db.Close(); err != nil {
			t.Fatalf("db.Close(v71) error = %v", err)
		}
		db = nil
		reopened, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(reopen v71) error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := reopened.Close(testutil.Context(t)); closeErr != nil {
				t.Errorf("Close(reopened v71) error = %v", closeErr)
			}
		})
		assertNetworkTimelineSequenceSchema(t, reopened.db)
		assertNetworkTimelineSequenceRows(t, reopened.db, rowIDs)
		assertNetworkSequenceProjectionBackfill(t, reopened.db)
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
			networkConversationTestWorkspaceID,
			"builders",
			"direct_0123456789abcdef0123456789abcdef",
			"coder.sess-abc",
			"reviewer.sess-xyz",
		)
		_, err := globalDB.db.ExecContext(
			ctx,
			`INSERT INTO network_direct_rooms (
				workspace_id, channel, direct_id, peer_a, peer_b, opened_at, last_activity_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			networkConversationTestWorkspaceID,
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
				workspace_id, channel, direct_id, peer_a, peer_b, opened_at, last_activity_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			networkConversationTestWorkspaceID,
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
				work_id, workspace_id, channel, surface, thread_id, opened_by_peer_id, state, opened_at, last_activity_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"work_missing_thread",
			networkConversationTestWorkspaceID,
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
			`DELETE FROM network_threads WHERE workspace_id = ? AND channel = ? AND thread_id = ?`,
			networkConversationTestWorkspaceID,
			"builders",
			"thread_restrict",
		)
		requireSQLiteConstraintError(t, err)

		insertDirectRoom(
			t,
			globalDB.db,
			networkConversationTestWorkspaceID,
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
			`DELETE FROM network_direct_rooms WHERE workspace_id = ? AND channel = ? AND direct_id = ?`,
			networkConversationTestWorkspaceID,
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
				workspace_id, channel, thread_id, peer_id, first_message_id, first_seen_at, last_seen_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			networkConversationTestWorkspaceID,
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
			`DELETE FROM network_threads WHERE workspace_id = ? AND channel = ? AND thread_id = ?`,
			networkConversationTestWorkspaceID,
			"builders",
			"thread_cascade",
		); err != nil {
			t.Fatalf("delete unreferenced thread error = %v", err)
		}

		var count int
		if err := globalDB.db.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM network_thread_participants WHERE workspace_id = ? AND channel = ? AND thread_id = ?`,
			networkConversationTestWorkspaceID,
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

func seedNetworkTimelineSequenceV70(t *testing.T, db *sql.DB) map[string]int64 {
	t.Helper()

	ctx := testutil.Context(t)
	messages := []struct {
		messageID string
		surface   any
		threadID  any
		directID  any
		peerTo    any
		kind      string
		preview   string
		timestamp string
	}{
		{
			messageID: "msg-sequence-thread-root",
			surface:   store.NetworkSurfaceThread,
			threadID:  "thread_sequence",
			kind:      store.NetworkKindSay,
			preview:   "thread timestamp latest",
			timestamp: "2026-07-11T12:00:10Z",
		},
		{
			messageID: "msg-sequence-direct-root",
			surface:   store.NetworkSurfaceDirect,
			directID:  "direct_0123456789abcdef0123456789abcdef",
			peerTo:    "reviewer.peer",
			kind:      store.NetworkKindSay,
			preview:   "direct timestamp latest",
			timestamp: "2026-07-11T12:00:11Z",
		},
		{
			messageID: "msg-sequence-presence",
			kind:      store.NetworkKindGreet,
			preview:   "presence",
			timestamp: "2026-07-11T12:00:20Z",
		},
		{
			messageID: "msg-sequence-thread-latest",
			surface:   store.NetworkSurfaceThread,
			threadID:  "thread_sequence",
			kind:      store.NetworkKindSay,
			preview:   "thread causal latest",
			timestamp: "2026-07-11T12:00:01Z",
		},
		{
			messageID: "msg-sequence-direct-latest",
			surface:   store.NetworkSurfaceDirect,
			directID:  "direct_0123456789abcdef0123456789abcdef",
			peerTo:    "reviewer.peer",
			kind:      store.NetworkKindSay,
			preview:   "direct causal latest",
			timestamp: "2026-07-11T12:00:02Z",
		},
	}
	for _, message := range messages {
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO network_timeline_log (
				message_id, session_id, workspace_id, channel, surface, thread_id, direct_id,
				direction, peer_from, peer_to, kind, text, preview_text, body_json, timestamp
			) VALUES (?, 'sess-sequence', 'ws-sequence', 'launch', ?, ?, ?, 'received',
				'coder.peer', ?, ?, ?, ?, '{}', ?)`,
			message.messageID,
			message.surface,
			message.threadID,
			message.directID,
			message.peerTo,
			message.kind,
			message.preview,
			message.preview,
			message.timestamp,
		); err != nil {
			t.Fatalf("insert v70 timeline message %q error = %v", message.messageID, err)
		}
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO network_threads (
			workspace_id, channel, thread_id, root_message_id, opened_at, last_activity_at,
			message_count, last_message_preview
		) VALUES (
			'ws-sequence', 'launch', 'thread_sequence', 'msg-sequence-thread-root',
			'2026-07-11T12:00:10Z', '2026-07-11T12:00:10Z', 2, 'thread timestamp latest'
		)`,
	); err != nil {
		t.Fatalf("insert v70 thread projection error = %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO network_direct_rooms (
			workspace_id, channel, direct_id, peer_a, peer_b, opened_at, last_activity_at,
			message_count, last_message_preview
		) VALUES (
			'ws-sequence', 'launch', 'direct_0123456789abcdef0123456789abcdef',
			'coder.peer', 'reviewer.peer', '2026-07-11T12:00:11Z', '2026-07-11T12:00:11Z',
			2, 'direct timestamp latest'
		)`,
	); err != nil {
		t.Fatalf("insert v70 direct projection error = %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO network_channel_stats (
			workspace_id, channel, message_count, presence_count, historical_participant_count,
			last_activity_at, last_presence_at, last_message_id, last_message_preview
		) VALUES (
			'ws-sequence', 'launch', 2, 1, 2, '2026-07-11T12:00:10Z',
			'2026-07-11T12:00:20Z', 'msg-sequence-thread-root', 'thread timestamp latest'
		)`,
	); err != nil {
		t.Fatalf("insert v70 channel projection error = %v", err)
	}

	rows, err := db.QueryContext(ctx, `SELECT message_id, rowid FROM network_timeline_log ORDER BY rowid`)
	if err != nil {
		t.Fatalf("query v70 rowids error = %v", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			t.Errorf("rows.Close(v70 rowids) error = %v", closeErr)
		}
	}()
	rowIDs := make(map[string]int64, len(messages))
	for rows.Next() {
		var messageID string
		var rowID int64
		if err := rows.Scan(&messageID, &rowID); err != nil {
			t.Fatalf("scan v70 rowid error = %v", err)
		}
		rowIDs[messageID] = rowID
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate v70 rowids error = %v", err)
	}
	return rowIDs
}

func assertNetworkTimelineSequenceSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	projectionColumns := []struct {
		table  string
		column string
	}{
		{table: "network_threads", column: "opened_sequence"},
		{table: "network_threads", column: "last_activity_sequence"},
		{table: "network_direct_rooms", column: "opened_sequence"},
		{table: "network_direct_rooms", column: "last_activity_sequence"},
		{table: "network_channel_stats", column: "last_activity_sequence"},
		{table: "network_channel_stats", column: "last_presence_sequence"},
		{table: "network_channel_stats", column: "last_message_sequence"},
	}
	for _, column := range projectionColumns {
		var notNull int
		var defaultValue sql.NullString
		if err := db.QueryRowContext(
			testutil.Context(t),
			`SELECT "notnull", dflt_value FROM pragma_table_info('`+column.table+`') WHERE name = ?`,
			column.column,
		).Scan(&notNull, &defaultValue); err != nil {
			t.Fatalf("read %s.%s definition error = %v", column.table, column.column, err)
		}
		if notNull != 1 || !defaultValue.Valid || defaultValue.String != "0" {
			t.Fatalf(
				"%s.%s definition = notnull %d/default %#v, want 1/0",
				column.table,
				column.column,
				notNull,
				defaultValue,
			)
		}
	}
	assertUniqueIndexColumns(t, db, "network_timeline_log", []string{"workspace_id", "message_id"})

	var columnType string
	var primaryKey int
	if err := db.QueryRowContext(
		testutil.Context(t),
		`SELECT type, pk FROM pragma_table_info('network_timeline_log') WHERE name = 'sequence'`,
	).Scan(&columnType, &primaryKey); err != nil {
		t.Fatalf("read network timeline sequence column error = %v", err)
	}
	if columnType != "INTEGER" || primaryKey != 1 {
		t.Fatalf("network timeline sequence definition = %s/pk=%d, want INTEGER/pk=1", columnType, primaryKey)
	}

	indexes := []struct {
		name string
		want []string
	}{
		{
			name: "idx_net_timeline_thread_sequence",
			want: []string{"workspace_id", "channel", "thread_id", "sequence"},
		},
		{
			name: "idx_net_timeline_direct_sequence",
			want: []string{"workspace_id", "channel", "direct_id", "sequence"},
		},
		{name: "idx_net_timeline_work_sequence", want: []string{"workspace_id", "work_id", "sequence"}},
		{name: "idx_net_timeline_presence_sequence", want: []string{"workspace_id", "channel", "sequence"}},
		{name: "idx_net_timeline_kind_sequence", want: []string{"workspace_id", "kind", "sequence"}},
		{
			name: "idx_network_threads_activity",
			want: []string{"workspace_id", "channel", "last_activity_sequence", "thread_id"},
		},
		{
			name: "idx_network_threads_created",
			want: []string{"workspace_id", "channel", "opened_sequence", "thread_id"},
		},
		{
			name: "idx_network_threads_open_work",
			want: []string{
				"workspace_id",
				"channel",
				"open_work_count",
				"last_activity_sequence",
				"thread_id",
			},
		},
		{
			name: "idx_network_direct_rooms_activity",
			want: []string{"workspace_id", "channel", "last_activity_sequence", "direct_id"},
		},
		{
			name: "idx_network_direct_rooms_peer_a",
			want: []string{"workspace_id", "channel", "peer_a", "last_activity_sequence"},
		},
		{
			name: "idx_network_direct_rooms_peer_b",
			want: []string{"workspace_id", "channel", "peer_b", "last_activity_sequence"},
		},
		{
			name: "idx_network_direct_rooms_created",
			want: []string{"workspace_id", "channel", "opened_at", "direct_id"},
		},
		{
			name: "idx_network_direct_rooms_open_work",
			want: []string{
				"workspace_id",
				"channel",
				"open_work_count",
				"last_activity_sequence",
				"direct_id",
			},
		},
		{
			name: "idx_network_channel_stats_activity",
			want: []string{"workspace_id", "last_activity_sequence", "channel"},
		},
	}
	for _, index := range indexes {
		if got := indexColumns(t, db, index.name); !slices.Equal(got, index.want) {
			t.Fatalf("index %s columns = %#v, want %#v", index.name, got, index.want)
		}
	}
}

func assertNetworkTimelineSequenceRows(t *testing.T, db *sql.DB, rowIDs map[string]int64) {
	t.Helper()

	ctx := testutil.Context(t)
	for messageID, rowID := range rowIDs {
		var sequence int64
		var body string
		if err := db.QueryRowContext(
			ctx,
			`SELECT sequence, body_json FROM network_timeline_log
			 WHERE workspace_id = 'ws-sequence' AND message_id = ?`,
			messageID,
		).Scan(&sequence, &body); err != nil {
			t.Fatalf("read migrated timeline message %q error = %v", messageID, err)
		}
		if sequence != rowID || body != "{}" {
			t.Fatalf("migrated timeline message %q = sequence %d/body %q, want %d/{}", messageID, sequence, body, rowID)
		}
	}
}

func assertNetworkTimelineWorkspaceScopedUniqueness(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx := testutil.Context(t)
	_, err := db.ExecContext(
		ctx,
		`INSERT INTO network_timeline_log (
			message_id, workspace_id, channel, direction, peer_from, kind, body_json, timestamp
		) VALUES (
			'msg-sequence-thread-root', 'ws-sequence', 'launch', 'received',
			'coder.peer', 'greet', '{}', '2026-07-11T12:00:30Z'
		)`,
	)
	requireSQLiteConstraintError(t, err)
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO network_timeline_log (
			message_id, workspace_id, channel, direction, peer_from, kind, body_json, timestamp
		) VALUES (
			'msg-sequence-thread-root', 'ws-sequence-other', 'launch', 'received',
			'coder.peer', 'greet', '{}', '2026-07-11T12:00:30Z'
		)`,
	); err != nil {
		t.Fatalf("insert same message id in another workspace error = %v", err)
	}
	var count int
	if err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM network_timeline_log WHERE message_id = 'msg-sequence-thread-root'`,
	).Scan(&count); err != nil {
		t.Fatalf("count workspace-scoped duplicate message IDs error = %v", err)
	}
	if count != 2 {
		t.Fatalf("workspace-scoped duplicate message count = %d, want 2", count)
	}
}

func assertNetworkSequenceProjectionBackfill(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx := testutil.Context(t)
	var openedSequence, activitySequence int64
	var activityAt, preview string
	if err := db.QueryRowContext(
		ctx,
		`SELECT opened_sequence, last_activity_sequence, last_activity_at, last_message_preview
		 FROM network_threads
		 WHERE workspace_id = 'ws-sequence' AND channel = 'launch' AND thread_id = 'thread_sequence'`,
	).Scan(&openedSequence, &activitySequence, &activityAt, &preview); err != nil {
		t.Fatalf("read migrated thread projection error = %v", err)
	}
	if openedSequence != 1 || activitySequence != 4 || activityAt != "2026-07-11T12:00:01Z" ||
		preview != "thread causal latest" {
		t.Fatalf(
			"thread projection = %d/%d/%q/%q, want 1/4/causal latest",
			openedSequence,
			activitySequence,
			activityAt,
			preview,
		)
	}

	if err := db.QueryRowContext(
		ctx,
		`SELECT opened_sequence, last_activity_sequence, last_activity_at, last_message_preview
		 FROM network_direct_rooms
		 WHERE workspace_id = 'ws-sequence' AND channel = 'launch'
			AND direct_id = 'direct_0123456789abcdef0123456789abcdef'`,
	).Scan(&openedSequence, &activitySequence, &activityAt, &preview); err != nil {
		t.Fatalf("read migrated direct projection error = %v", err)
	}
	if openedSequence != 2 || activitySequence != 5 || activityAt != "2026-07-11T12:00:02Z" ||
		preview != "direct causal latest" {
		t.Fatalf(
			"direct projection = %d/%d/%q/%q, want 2/5/causal latest",
			openedSequence,
			activitySequence,
			activityAt,
			preview,
		)
	}

	var presenceSequence, messageSequence int64
	var presenceAt, messageID string
	if err := db.QueryRowContext(
		ctx,
		`SELECT last_activity_sequence, last_presence_sequence, last_message_sequence,
			last_activity_at, last_presence_at, last_message_id, last_message_preview
		 FROM network_channel_stats
		 WHERE workspace_id = 'ws-sequence' AND channel = 'launch'`,
	).Scan(
		&activitySequence,
		&presenceSequence,
		&messageSequence,
		&activityAt,
		&presenceAt,
		&messageID,
		&preview,
	); err != nil {
		t.Fatalf("read migrated channel projection error = %v", err)
	}
	if activitySequence != 4 || presenceSequence != 3 || messageSequence != 4 ||
		activityAt != "2026-07-11T12:00:01Z" || presenceAt != "2026-07-11T12:00:20Z" ||
		messageID != "msg-sequence-thread-latest" || preview != "thread causal latest" {
		t.Fatalf(
			"channel projection = activity %d/%q presence %d/%q message %d/%q/%q",
			activitySequence,
			activityAt,
			presenceSequence,
			presenceAt,
			messageSequence,
			messageID,
			preview,
		)
	}
}

func assertNetworkTimelineSequenceContinues(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx := testutil.Context(t)
	var maxBefore int64
	if err := db.QueryRowContext(ctx, `SELECT MAX(sequence) FROM network_timeline_log`).Scan(&maxBefore); err != nil {
		t.Fatalf("read maximum migrated sequence error = %v", err)
	}
	assertSQLiteSequence := func(want int64) {
		t.Helper()
		var got int64
		if err := db.QueryRowContext(
			ctx,
			`SELECT seq FROM sqlite_sequence WHERE name = 'network_timeline_log'`,
		).Scan(&got); err != nil {
			t.Fatalf("read sqlite_sequence for network timeline error = %v", err)
		}
		if got != want {
			t.Fatalf("network timeline sqlite_sequence = %d, want %d", got, want)
		}
	}
	assertSQLiteSequence(maxBefore)
	insert := func(messageID string) int64 {
		t.Helper()
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO network_timeline_log (
				message_id, workspace_id, channel, direction, peer_from, kind, body_json, timestamp
			) VALUES (?, 'ws-sequence', 'launch', 'received', 'coder.peer', 'greet', '{}',
				'2026-07-11T12:00:30Z')`,
			messageID,
		); err != nil {
			t.Fatalf("insert post-migration timeline message %q error = %v", messageID, err)
		}
		var sequence int64
		if err := db.QueryRowContext(
			ctx,
			`SELECT sequence FROM network_timeline_log WHERE workspace_id = 'ws-sequence' AND message_id = ?`,
			messageID,
		).Scan(&sequence); err != nil {
			t.Fatalf("read post-migration timeline sequence %q error = %v", messageID, err)
		}
		return sequence
	}
	first := insert("msg-sequence-after-migration")
	if first <= maxBefore {
		t.Fatalf("first new sequence = %d, want greater than migrated max %d", first, maxBefore)
	}
	assertSQLiteSequence(first)
	if _, err := db.ExecContext(
		ctx,
		`DELETE FROM network_timeline_log
		 WHERE workspace_id = 'ws-sequence' AND message_id = 'msg-sequence-after-migration'`,
	); err != nil {
		t.Fatalf("delete highest timeline sequence error = %v", err)
	}
	second := insert("msg-sequence-after-delete")
	if second <= first {
		t.Fatalf("sequence after deleting maximum = %d, want greater than %d", second, first)
	}
	assertSQLiteSequence(second)
}

func insertThread(t *testing.T, db *sql.DB, channel string, threadID string, rootMessageID string) {
	t.Helper()

	if _, err := db.ExecContext(
		testutil.Context(t),
		`INSERT INTO network_threads (
			workspace_id, channel, thread_id, root_message_id, opened_by_peer_id, opened_at, last_activity_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		networkConversationTestWorkspaceID,
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

func insertDirectRoom(
	t *testing.T,
	db *sql.DB,
	workspaceID string,
	channel string,
	directID string,
	peerA string,
	peerB string,
) {
	t.Helper()

	if _, err := db.ExecContext(
		testutil.Context(t),
		`INSERT INTO network_direct_rooms (
			workspace_id, channel, direct_id, peer_a, peer_b, opened_at, last_activity_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		workspaceID,
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
			work_id, workspace_id, channel, surface, thread_id, opened_by_peer_id, state, opened_at, last_activity_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		workID,
		networkConversationTestWorkspaceID,
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
			work_id, workspace_id, channel, surface, direct_id, opened_by_peer_id, state, opened_at, last_activity_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		workID,
		networkConversationTestWorkspaceID,
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
