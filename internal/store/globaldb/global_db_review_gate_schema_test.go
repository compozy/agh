package globaldb

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestGlobalDBReviewGateSchemaMigration(t *testing.T) {
	t.Parallel()

	t.Run("Should create review gate schema on fresh DB", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)

		assertReviewGateSchema(t, globalDB.db)
	})

	t.Run("Should migrate previous review gate schema and preserve rows", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		dbPath := filepath.Join(t.TempDir(), GlobalDatabaseName)
		legacyDB := openPreviousReviewGateSchemaDB(t, dbPath)
		insertMigrationRecordsThroughVersion(t, legacyDB, 17)
		insertPreviousReviewGateRows(t, legacyDB)
		if err := legacyDB.Close(); err != nil {
			t.Fatalf("legacyDB.Close() error = %v", err)
		}

		globalDB, err := OpenGlobalDB(ctx, dbPath)
		if err != nil {
			t.Fatalf("OpenGlobalDB() error = %v", err)
		}
		t.Cleanup(func() {
			if err := globalDB.Close(ctx); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		})

		assertReviewGateSchema(t, globalDB.db)
		assertPreviousReviewGateRowsPreserved(t, globalDB.db)
	})
}

func TestReviewGateSchemaStatements(t *testing.T) {
	t.Parallel()

	t.Run("Should use shared review gate DDL in fresh global schema", func(t *testing.T) {
		t.Parallel()

		for _, statement := range taskRunReviewTableSchemaStatements() {
			if !schemaStatementsContain(globalSchemaStatements, statement) {
				t.Fatalf("globalSchemaStatements missing review table statement %q", statement)
			}
		}
		for _, statement := range taskReviewGateIndexStatements() {
			if !schemaStatementsContain(globalSchemaStatements, statement) {
				t.Fatalf("globalSchemaStatements missing review index statement %q", statement)
			}
		}
	})
}

func openPreviousReviewGateSchemaDB(t *testing.T, dbPath string) *sql.DB {
	t.Helper()

	ctx := testutil.Context(t)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	for _, statement := range []string{
		`CREATE TABLE tasks (
			id              TEXT PRIMARY KEY,
			identifier      TEXT,
			scope           TEXT NOT NULL CHECK (scope IN ('global', 'workspace')),
			workspace_id    TEXT,
			parent_task_id  TEXT,
			network_channel TEXT,
			title           TEXT NOT NULL,
			description     TEXT,
			priority        TEXT NOT NULL DEFAULT 'medium',
			max_attempts    INTEGER NOT NULL DEFAULT 3,
			status          TEXT NOT NULL,
			approval_policy TEXT NOT NULL DEFAULT 'none',
			approval_state  TEXT NOT NULL DEFAULT 'not_required',
			owner_kind      TEXT,
			owner_ref       TEXT,
			created_by_kind TEXT NOT NULL,
			created_by_ref  TEXT NOT NULL,
			origin_kind     TEXT NOT NULL,
			origin_ref      TEXT NOT NULL,
			created_at      TEXT NOT NULL,
			updated_at      TEXT NOT NULL,
			closed_at       TEXT,
			metadata_json   TEXT,
			current_run_id  TEXT REFERENCES task_runs(id) ON DELETE SET NULL,
			max_runtime_seconds INTEGER NOT NULL DEFAULT 0 CHECK (max_runtime_seconds >= 0),
			spawn_failure_count INTEGER NOT NULL DEFAULT 0 CHECK (spawn_failure_count >= 0),
			last_spawn_error TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE task_runs (
			id              TEXT PRIMARY KEY,
			task_id         TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			status          TEXT NOT NULL,
			attempt         INTEGER NOT NULL,
			claimed_by_kind TEXT,
			claimed_by_ref  TEXT,
			session_id      TEXT,
			origin_kind     TEXT NOT NULL,
			origin_ref      TEXT NOT NULL,
			idempotency_key TEXT,
			network_channel TEXT,
			queued_at       TEXT NOT NULL,
			claimed_at      TEXT,
			started_at      TEXT,
			ended_at        TEXT,
			error           TEXT,
			metadata_json   TEXT,
			result_json     TEXT,
			claim_token     TEXT,
			claim_token_hash TEXT,
			lease_until     TEXT,
			heartbeat_at    TEXT,
			coordination_channel_id TEXT,
			summary         TEXT NOT NULL DEFAULT '',
			claimed_agent_name TEXT NOT NULL DEFAULT '',
			claimed_peer_id TEXT NOT NULL DEFAULT '',
			terminalized_by_session_id TEXT NOT NULL DEFAULT '',
			terminalized_by_agent_name TEXT NOT NULL DEFAULT '',
			terminalized_by_peer_id TEXT NOT NULL DEFAULT '',
			terminalized_by_actor_kind TEXT NOT NULL DEFAULT '',
			terminalized_by_actor_ref TEXT NOT NULL DEFAULT ''
		);`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("ExecContext(previous review schema) error = %v", err)
		}
	}
	for _, statement := range taskOrchestrationProfileSchemaStatements() {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("ExecContext(profile schema) error = %v", err)
		}
	}
	if err := store.RunMigrations(ctx, db, nil); err != nil {
		t.Fatalf("RunMigrations(empty) error = %v", err)
	}
	return db
}

func insertPreviousReviewGateRows(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx := testutil.Context(t)
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO tasks (
			id, identifier, scope, workspace_id, parent_task_id, network_channel, title, description,
			priority, max_attempts, status, approval_policy, approval_state, owner_kind, owner_ref,
			created_by_kind, created_by_ref, origin_kind, origin_ref, created_at, updated_at,
			closed_at, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"task-review-migration",
		"review-migration",
		string(taskpkg.ScopeGlobal),
		nil,
		nil,
		nil,
		"Review migration task",
		"Preserve this task",
		string(taskpkg.PriorityMedium),
		3,
		string(taskpkg.TaskStatusReady),
		string(taskpkg.ApprovalPolicyNone),
		string(taskpkg.ApprovalStateNotRequired),
		nil,
		nil,
		string(taskpkg.ActorKindHuman),
		"user:alice",
		string(taskpkg.OriginKindCLI),
		"cli",
		"2026-05-05T12:00:00.000000000Z",
		"2026-05-05T12:00:00.000000000Z",
		nil,
		`{"preserved":true}`,
	); err != nil {
		t.Fatalf("insert previous review task error = %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO task_runs (
			id, task_id, status, attempt, claimed_by_kind, claimed_by_ref, session_id,
			origin_kind, origin_ref, idempotency_key, network_channel, queued_at,
			claimed_at, started_at, ended_at, error, metadata_json, result_json,
			claim_token, claim_token_hash, lease_until, heartbeat_at, coordination_channel_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"run-review-migration",
		"task-review-migration",
		string(taskpkg.TaskRunStatusQueued),
		1,
		nil,
		nil,
		nil,
		string(taskpkg.OriginKindCLI),
		"cli",
		nil,
		nil,
		"2026-05-05T12:00:00.000000000Z",
		nil,
		nil,
		nil,
		nil,
		`{"run":true}`,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	); err != nil {
		t.Fatalf("insert previous review task run error = %v", err)
	}
}

func assertReviewGateSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	assertTablesPresent(t, db, "task_run_reviews")
	assertTableHasColumns(t, db, "tasks", []string{
		"review_policy",
		"review_max_rounds",
		"review_round",
		"last_review_id",
		"last_review_outcome",
		"review_circuit_opened_at",
		"review_circuit_reason",
	})
	assertTableHasColumns(t, db, "task_runs", []string{
		"review_required",
		"review_request_round",
		"review_policy_snapshot",
		"review_request_id",
		"parent_run_id",
		"review_id",
		"review_round",
		"continuation_reason",
		"missing_work_json",
		"next_round_guidance",
	})
	assertTableColumns(t, db, "task_run_reviews", []string{
		"review_id",
		"task_id",
		"run_id",
		"parent_review_id",
		"policy",
		"review_round",
		"attempt",
		"status",
		"outcome",
		"confidence",
		"reason",
		"delivery_id",
		"missing_work_json",
		"next_round_guidance",
		"review_text",
		"reviewer_session_id",
		"reviewer_agent_name",
		"reviewer_peer_id",
		"reviewer_channel_id",
		"reviewed_by_kind",
		"reviewed_by_ref",
		"requested_at",
		"routed_at",
		"started_at",
		"reviewed_at",
		"deadline_at",
		"created_at",
		"updated_at",
	})
	assertIndexesPresent(
		t,
		db,
		"task_run_reviews",
		"idx_task_run_reviews_task_round_attempt",
		"uq_task_run_reviews_run_round_attempt",
		"idx_task_run_reviews_run_status",
		"idx_task_run_reviews_deadline",
		"idx_task_run_reviews_reviewer_session",
		"idx_task_run_reviews_reviewer_agent",
		"idx_task_run_reviews_reviewer_peer",
		"idx_task_run_reviews_reviewer_channel",
		"uq_task_run_reviews_reviewer_session_active",
		"uq_task_run_reviews_delivery",
	)
	assertIndexesPresent(t, db, "tasks", "idx_tasks_review_policy", "idx_tasks_review_round")
	assertIndexesPresent(
		t,
		db,
		"task_runs",
		"idx_task_runs_parent_run",
		"idx_task_runs_review_request",
		"uq_task_runs_review_id",
		"idx_task_runs_task_review_round",
	)
}

func assertTableHasColumns(t *testing.T, db *sql.DB, table string, want []string) {
	t.Helper()

	columns := queryTableColumns(t, db, table)
	for _, column := range want {
		if _, ok := columns[column]; !ok {
			t.Fatalf("column %s.%s missing from columns %#v", table, column, columns)
		}
	}
}

func queryTableColumns(t *testing.T, db *sql.DB, table string) map[string]struct{} {
	t.Helper()

	rows, err := db.QueryContext(testutil.Context(t), "PRAGMA table_info("+table+")")
	if err != nil {
		t.Fatalf("QueryContext(table_info %q) error = %v", table, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Fatalf("rows.Close(table_info %q) error = %v", table, err)
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
			t.Fatalf("Scan(table_info %q) error = %v", table, err)
		}
		columns[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(table_info %q) error = %v", table, err)
	}
	return columns
}

func assertPreviousReviewGateRowsPreserved(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx := testutil.Context(t)
	var (
		reviewPolicy          string
		reviewMaxRounds       int
		reviewRound           int
		lastReviewID          sql.NullString
		lastReviewOutcome     sql.NullString
		reviewCircuitOpenedAt sql.NullString
		reviewCircuitReason   sql.NullString
	)
	if err := db.QueryRowContext(
		ctx,
		`SELECT review_policy, review_max_rounds, review_round, last_review_id,
		        last_review_outcome, review_circuit_opened_at, review_circuit_reason
		 FROM tasks
		 WHERE id = ?`,
		"task-review-migration",
	).Scan(
		&reviewPolicy,
		&reviewMaxRounds,
		&reviewRound,
		&lastReviewID,
		&lastReviewOutcome,
		&reviewCircuitOpenedAt,
		&reviewCircuitReason,
	); err != nil {
		t.Fatalf("query migrated review task defaults error = %v", err)
	}
	if reviewPolicy != "none" || reviewMaxRounds != 3 || reviewRound != 0 {
		t.Fatalf(
			"review task defaults = policy:%q max_rounds:%d round:%d, want none/3/0",
			reviewPolicy,
			reviewMaxRounds,
			reviewRound,
		)
	}
	if lastReviewID.Valid || lastReviewOutcome.Valid || reviewCircuitOpenedAt.Valid || reviewCircuitReason.Valid {
		t.Fatalf(
			"review task nullable defaults = %#v/%#v/%#v/%#v, want NULL",
			lastReviewID,
			lastReviewOutcome,
			reviewCircuitOpenedAt,
			reviewCircuitReason,
		)
	}

	var (
		reviewRequired     int
		reviewRequestRound int
		reviewPolicySnap   string
		reviewRequestID    sql.NullString
		parentRunID        sql.NullString
		reviewID           sql.NullString
		runReviewRound     int
		continuationReason string
		missingWorkJSON    string
		nextRoundGuidance  string
	)
	if err := db.QueryRowContext(
		ctx,
		`SELECT review_required, review_request_round, review_policy_snapshot,
		        review_request_id, parent_run_id, review_id, review_round, continuation_reason,
		        missing_work_json, next_round_guidance
		 FROM task_runs
		 WHERE id = ?`,
		"run-review-migration",
	).Scan(
		&reviewRequired,
		&reviewRequestRound,
		&reviewPolicySnap,
		&reviewRequestID,
		&parentRunID,
		&reviewID,
		&runReviewRound,
		&continuationReason,
		&missingWorkJSON,
		&nextRoundGuidance,
	); err != nil {
		t.Fatalf("query migrated review run defaults error = %v", err)
	}
	if reviewRequired != 0 || reviewRequestRound != 0 || reviewPolicySnap != "" || runReviewRound != 0 ||
		continuationReason != "" || missingWorkJSON != "[]" || nextRoundGuidance != "" {
		t.Fatalf(
			"review run defaults = %d/%d/%q/%d/%q/%q/%q, want 0/0/empty/0/empty/[]/empty",
			reviewRequired,
			reviewRequestRound,
			reviewPolicySnap,
			runReviewRound,
			continuationReason,
			missingWorkJSON,
			nextRoundGuidance,
		)
	}
	if reviewRequestID.Valid || parentRunID.Valid || reviewID.Valid {
		t.Fatalf("review run nullable defaults = %#v/%#v/%#v, want NULL", reviewRequestID, parentRunID, reviewID)
	}
}
