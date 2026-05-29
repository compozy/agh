package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// taskRunColumns lists every task_runs column in storage order so the rebuild copy is
// explicit (never relies on SELECT * ordering). Keep in sync with the task_runs schema.
const taskRunColumns = `id, task_id, status, attempt, previous_run_id, failure_kind, ` +
	`claimed_by_kind, claimed_by_ref, session_id, origin_kind, origin_ref, idempotency_key, ` +
	`network_channel, queued_at, claimed_at, started_at, ended_at, error, metadata_json, ` +
	`result_json, summary, claimed_agent_name, claimed_peer_id, terminalized_by_session_id, ` +
	`terminalized_by_agent_name, terminalized_by_peer_id, terminalized_by_actor_kind, ` +
	`terminalized_by_actor_ref, review_required, review_request_round, review_policy_snapshot, ` +
	`review_request_id, parent_run_id, review_id, review_round, continuation_reason, ` +
	`missing_work_json, next_round_guidance, claim_token, claim_token_hash, lease_until, ` +
	`heartbeat_at, coordination_channel_id`

// migrateDropTaskRunStatusCheck rebuilds task_runs without the status enum CHECK so the
// run-status enum's single source of truth becomes RunStatus.Validate (Go), letting new
// statuses like needs_attention exist. Every other constraint, FK, and index is preserved:
// the existing index DDL is captured from sqlite_master and replayed after the rebuild, so
// indexes added by later migrations are never dropped. SQLite cannot drop a CHECK in place,
// and the rebuild's foreign-key handling needs PRAGMA foreign_keys = OFF, a no-op inside a
// transaction — hence a UpConn migration.
func migrateDropTaskRunStatusCheck(ctx context.Context, conn *sql.Conn) (err error) {
	exists, err := tableExists(ctx, conn, "task_runs")
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
		return fmt.Errorf("store: disable foreign keys for task_runs status migration: %w", err)
	}
	foreignKeysDisabled = true

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin task_runs status migration: %w", err)
	}
	defer func() {
		joinCleanupError(&err, rollbackTx(tx, "task_runs status migration"))
	}()

	indexDDL, err := captureTaskRunIndexDDL(ctx, tx)
	if err != nil {
		return err
	}

	statements := []string{
		taskRunsWithoutStatusCheckCreateStatement(),
		`INSERT INTO task_runs_new (` + taskRunColumns + `) SELECT ` + taskRunColumns + ` FROM task_runs;`,
		`DROP TABLE task_runs;`,
		`ALTER TABLE task_runs_new RENAME TO task_runs;`,
	}
	statements = append(statements, indexDDL...)
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: rebuild task_runs without status check: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit task_runs status migration: %w", err)
	}
	return nil
}

// captureTaskRunIndexDDL returns the CREATE statements for every explicit task_runs index
// (auto-created PRIMARY KEY indexes have NULL sql and are excluded) so they survive the
// table rebuild regardless of which migration introduced them.
func captureTaskRunIndexDDL(ctx context.Context, tx *sql.Tx) (ddl []string, err error) {
	rows, err := tx.QueryContext(
		ctx,
		`SELECT sql FROM sqlite_master
			 WHERE type = 'index' AND tbl_name = 'task_runs' AND sql IS NOT NULL
			 ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: capture task_runs index ddl: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close task_runs index ddl rows: %w", closeErr)
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
			return nil, fmt.Errorf("store: scan task_runs index ddl: %w", err)
		}
		ddl = append(ddl, stmt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate task_runs index ddl: %w", err)
	}
	return ddl, nil
}

func taskRunsWithoutStatusCheckCreateStatement() string {
	return `CREATE TABLE task_runs_new (
		id              TEXT PRIMARY KEY,
		task_id         TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		status          TEXT NOT NULL,
		attempt         INTEGER NOT NULL CHECK (attempt > 0),
		previous_run_id TEXT,
		failure_kind    TEXT NOT NULL DEFAULT '' CHECK (
			failure_kind = '' OR failure_kind IN ('operator_forced')
		),
		claimed_by_kind TEXT CHECK (
			claimed_by_kind IS NULL OR claimed_by_kind IN (
				'human', 'agent_session', 'automation', 'extension', 'network_peer', 'daemon'
			)
		),
		claimed_by_ref  TEXT,
		session_id      TEXT,
		origin_kind     TEXT NOT NULL CHECK (
			origin_kind IN (
				'cli', 'web', 'uds', 'http', 'automation', 'extension', 'network', 'agent_session', 'daemon'
			)
		),
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
		summary         TEXT NOT NULL DEFAULT '',
		claimed_agent_name TEXT NOT NULL DEFAULT '',
		claimed_peer_id TEXT NOT NULL DEFAULT '',
		terminalized_by_session_id TEXT NOT NULL DEFAULT '',
		terminalized_by_agent_name TEXT NOT NULL DEFAULT '',
		terminalized_by_peer_id TEXT NOT NULL DEFAULT '',
		terminalized_by_actor_kind TEXT NOT NULL DEFAULT '',
		terminalized_by_actor_ref TEXT NOT NULL DEFAULT '',
		review_required BOOLEAN NOT NULL DEFAULT 0 CHECK (review_required IN (0, 1)),
		review_request_round INTEGER NOT NULL DEFAULT 0 CHECK (review_request_round >= 0),
		review_policy_snapshot TEXT NOT NULL DEFAULT '' CHECK (
			review_policy_snapshot = '' OR
			review_policy_snapshot IN ('none', 'on_success', 'on_failure', 'always')
		),
		review_request_id TEXT REFERENCES task_run_reviews(review_id),
		parent_run_id TEXT REFERENCES task_runs(id),
		review_id TEXT REFERENCES task_run_reviews(review_id),
		review_round INTEGER NOT NULL DEFAULT 0 CHECK (review_round >= 0),
		continuation_reason TEXT NOT NULL DEFAULT '',
		missing_work_json TEXT NOT NULL DEFAULT '[]',
		next_round_guidance TEXT NOT NULL DEFAULT '',
		claim_token TEXT,
		claim_token_hash TEXT,
		lease_until TEXT,
		heartbeat_at TEXT,
		coordination_channel_id TEXT,
		CHECK (
			(claimed_by_kind IS NULL AND claimed_by_ref IS NULL) OR
			(claimed_by_kind IS NOT NULL AND claimed_by_ref IS NOT NULL)
		),
		CHECK (status <> 'queued' OR session_id IS NULL)
	);`
}
