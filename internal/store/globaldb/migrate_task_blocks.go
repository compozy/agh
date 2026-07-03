package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const taskColumnsBeforeBlocks = `id, identifier, scope, workspace_id, parent_task_id, network_channel, ` +
	`title, description, priority, max_attempts, status, approval_policy, approval_state, ` +
	`owner_kind, owner_ref, created_by_kind, created_by_ref, origin_kind, origin_ref, ` +
	`created_at, updated_at, closed_at, metadata_json, current_run_id, paused, paused_by, ` +
	`paused_at, paused_reason, max_runtime_seconds, spawn_failure_count, last_spawn_error, ` +
	`review_policy, review_max_rounds, review_round, last_review_id, last_review_outcome, ` +
	`review_circuit_opened_at, review_circuit_reason, auto_enqueue_on_ready`

// migrateTaskBlockingFoundation rebuilds tasks once to drop the status CHECK while
// adding task-block prerequisite columns. This must be an UpConn migration because
// SQLite requires PRAGMA foreign_keys = OFF across the create-copy-swap, and every
// existing tasks index is recaptured from sqlite_master so later migrations survive.
func migrateTaskBlockingFoundation(ctx context.Context, conn *sql.Conn) (err error) {
	exists, err := tableExists(ctx, conn, "tasks")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	cleanupCtx := context.WithoutCancel(ctx)
	foreignKeysDisabled := false
	defer func() {
		if foreignKeysDisabled {
			joinCleanupError(&err, restoreForeignKeys(cleanupCtx, conn))
		}
	}()
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("store: disable foreign keys for tasks blocking foundation migration: %w", err)
	}
	foreignKeysDisabled = true

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("store: begin tasks blocking foundation migration: %w", err)
	}
	finished := false
	defer func() {
		if !finished {
			joinCleanupError(&err, rollbackImmediate(cleanupCtx, conn, "tasks blocking foundation migration"))
		}
	}()

	indexDDL, err := captureTasksIndexDDL(ctx, conn)
	if err != nil {
		return err
	}
	statements := []string{
		taskWithoutStatusCheckCreateSQL,
		`INSERT INTO tasks_new (` + taskColumnsBeforeBlocks + `) SELECT ` + taskColumnsBeforeBlocks + ` FROM tasks;`,
		`DROP TABLE tasks;`,
		`ALTER TABLE tasks_new RENAME TO tasks;`,
	}
	statements = append(statements, indexDDL...)
	statements = append(
		statements,
		`CREATE INDEX IF NOT EXISTS idx_tasks_created_by ON tasks(created_by_kind, created_by_ref);`,
	)
	for _, stmt := range statements {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: rebuild tasks for blocking foundation: %w", err)
		}
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("store: commit tasks blocking foundation migration: %w", err)
	}
	finished = true
	return nil
}

// migrateTaskBlockTables runs after v48 in strict migration order so the block
// table foreign keys target the rebuilt tasks table on fresh installs and upgrades.
func migrateTaskBlockTables(ctx context.Context, conn *sql.Conn) (err error) {
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("store: begin task block tables migration: %w", err)
	}
	cleanupCtx := context.WithoutCancel(ctx)
	finished := false
	defer func() {
		if !finished {
			joinCleanupError(&err, rollbackImmediate(cleanupCtx, conn, "task block tables migration"))
		}
	}()

	for _, stmt := range taskBlockTableStatements() {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: create task block tables: %w", err)
		}
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("store: commit task block tables migration: %w", err)
	}
	finished = true
	return nil
}

func captureTasksIndexDDL(ctx context.Context, exec sqlQueryExecutor) (ddl []string, err error) {
	rows, err := exec.QueryContext(
		ctx,
		`SELECT sql FROM sqlite_master
			 WHERE type = 'index' AND tbl_name = 'tasks' AND sql IS NOT NULL
			 ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: capture tasks index ddl: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close tasks index ddl rows: %w", closeErr)
			if err == nil {
				err = closeErr
				return
			}
			err = errors.Join(err, closeErr)
		}
	}()
	for rows.Next() {
		var stmt string
		if err := rows.Scan(&stmt); err != nil {
			return nil, fmt.Errorf("store: scan tasks index ddl: %w", err)
		}
		ddl = append(ddl, stmt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate tasks index ddl: %w", err)
	}
	return ddl, nil
}

const taskWithoutStatusCheckCreateSQL = `CREATE TABLE tasks_new (
		id              TEXT PRIMARY KEY,
		identifier      TEXT,
		scope           TEXT NOT NULL CHECK (scope IN ('global', 'workspace')),
		workspace_id    TEXT REFERENCES workspaces(id) ON DELETE CASCADE,
		parent_task_id  TEXT REFERENCES tasks(id),
		network_channel TEXT,
		title           TEXT NOT NULL,
		description     TEXT,
		priority        TEXT NOT NULL DEFAULT 'medium' CHECK (
			priority IN ('low', 'medium', 'high', 'urgent')
		),
		max_attempts    INTEGER NOT NULL DEFAULT 3 CHECK (max_attempts > 0 AND max_attempts <= 10),
		status          TEXT NOT NULL,
		approval_policy TEXT NOT NULL DEFAULT 'none' CHECK (
			approval_policy IN ('none', 'manual')
		),
		approval_state  TEXT NOT NULL DEFAULT 'not_required' CHECK (
			approval_state IN ('not_required', 'pending', 'approved', 'rejected')
		),
		owner_kind      TEXT CHECK (
			owner_kind IS NULL OR owner_kind IN (
				'human', 'agent_session', 'automation', 'extension', 'network_peer', 'pool'
			)
		),
		owner_ref       TEXT,
		created_by_kind TEXT NOT NULL CHECK (
			created_by_kind IN (
				'human', 'agent_session', 'automation', 'extension', 'network_peer', 'daemon'
			)
		),
		created_by_ref  TEXT NOT NULL,
		origin_kind     TEXT NOT NULL CHECK (
			origin_kind IN (
				'cli', 'web', 'uds', 'http', 'automation', 'extension', 'network', 'agent_session', 'daemon'
			)
		),
		origin_ref      TEXT NOT NULL,
		created_at      TEXT NOT NULL,
		updated_at      TEXT NOT NULL,
		closed_at       TEXT,
		metadata_json   TEXT,
		current_run_id  TEXT REFERENCES task_runs(id) ON DELETE SET NULL,
		paused          INTEGER NOT NULL DEFAULT 0 CHECK (paused IN (0, 1)),
		paused_by       TEXT NOT NULL DEFAULT '',
		paused_at       TEXT,
		paused_reason   TEXT NOT NULL DEFAULT '',
		max_runtime_seconds INTEGER NOT NULL DEFAULT 0 CHECK (max_runtime_seconds >= 0),
		spawn_failure_count INTEGER NOT NULL DEFAULT 0 CHECK (spawn_failure_count >= 0),
		last_spawn_error TEXT NOT NULL DEFAULT '',
		review_policy TEXT NOT NULL DEFAULT 'none' CHECK (
			review_policy IN ('none', 'on_success', 'on_failure', 'always')
		),
		review_max_rounds INTEGER NOT NULL DEFAULT 3 CHECK (review_max_rounds >= 0),
		review_round INTEGER NOT NULL DEFAULT 0 CHECK (review_round >= 0),
		last_review_id TEXT,
		last_review_outcome TEXT CHECK (
			last_review_outcome IS NULL OR last_review_outcome IN (
				'approved', 'rejected', 'blocked', 'error', 'timeout', 'invalid_output'
			)
		),
		review_circuit_opened_at TEXT,
		review_circuit_reason TEXT,
		auto_enqueue_on_ready INTEGER NOT NULL DEFAULT 0 CHECK (auto_enqueue_on_ready IN (0, 1)),
		needs_attention_reason  TEXT,
		needs_attention_at      TEXT,
		needs_attention_by_kind TEXT,
		needs_attention_by_ref  TEXT,
		wake_creator            INTEGER NOT NULL DEFAULT 1,
		CHECK (
			(scope = 'global' AND workspace_id IS NULL) OR
			(scope = 'workspace' AND workspace_id IS NOT NULL)
		),
		CHECK (
			(owner_kind IS NULL AND owner_ref IS NULL) OR
			(owner_kind IS NOT NULL AND owner_ref IS NOT NULL)
		),
		CHECK (parent_task_id IS NULL OR parent_task_id <> id),
		CHECK (
			(approval_policy = 'none' AND approval_state = 'not_required') OR
			(approval_policy = 'manual' AND approval_state IN ('pending', 'approved', 'rejected'))
		)
		);`

func taskBlockTableStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS task_blocks (
			id              TEXT PRIMARY KEY,
			workspace_id    TEXT,
			task_id         TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			kind            TEXT NOT NULL CHECK (kind IN ('needs_input','capability','transient')),
			reason          TEXT NOT NULL CHECK (length(reason) > 0),
			details_json    TEXT,
			created_by_kind TEXT NOT NULL,
			created_by_ref  TEXT NOT NULL,
			created_at      TEXT NOT NULL,
			expires_at      TEXT,
			cleared_at      TEXT,
			cleared_by_kind TEXT,
			cleared_by_ref  TEXT,
			clear_note      TEXT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_task_blocks_open
			ON task_blocks(task_id)
			WHERE cleared_at IS NULL;`,
		`CREATE INDEX IF NOT EXISTS idx_task_blocks_expiry
			ON task_blocks(expires_at)
			WHERE cleared_at IS NULL AND expires_at IS NOT NULL;`,
		`CREATE TABLE IF NOT EXISTS task_block_recurrences (
			task_id    TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			kind       TEXT NOT NULL,
			count      INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (task_id, kind)
		);`,
	}
}
