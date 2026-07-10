package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

var taskEventsTypeSeqMigration = store.Migration{
	Version:  62,
	Name:     "add_task_events_type_seq_index",
	Up:       migrateTaskEventsTypeSeqIndex,
	Checksum: "2026-07-08-add-task-events-type-seq-index",
}

var watchEventReplayProjectionsMigration = store.Migration{
	Version:  63,
	Name:     "add_watch_event_replay_projections",
	Up:       migrateWatchEventReplayProjections,
	Checksum: "2026-07-09-add-watch-event-replay-projections",
}

const automationWatchEventsTableStatement = `CREATE TABLE automation_watch_events (
	seq          INTEGER PRIMARY KEY AUTOINCREMENT,
	run_id       TEXT NOT NULL,
	job_id       TEXT NOT NULL DEFAULT '',
	trigger_id   TEXT NOT NULL DEFAULT '',
	session_id   TEXT NOT NULL DEFAULT '',
	status       TEXT NOT NULL CHECK (status IN ('completed', 'failed')),
	attempt      INTEGER NOT NULL,
	started_at   TEXT,
	ended_at     TEXT,
	error        TEXT NOT NULL DEFAULT '',
	agent_name   TEXT NOT NULL DEFAULT '',
	workspace_id TEXT NOT NULL DEFAULT '',
	retry_json   TEXT NOT NULL DEFAULT ''
)`

const automationWatchEventsInsertTriggerStatement = `CREATE TRIGGER automation_watch_events_after_insert
AFTER INSERT ON automation_runs
WHEN NEW.status IN ('completed', 'failed')
BEGIN
	INSERT INTO automation_watch_events (
		run_id, job_id, trigger_id, session_id, status, attempt,
		started_at, ended_at, error, agent_name, workspace_id, retry_json
	) VALUES (
		NEW.id,
		COALESCE(NEW.job_id, ''),
		COALESCE(NEW.trigger_id, ''),
		COALESCE(NEW.session_id, ''),
		NEW.status,
		NEW.attempt,
		NEW.started_at,
		NEW.ended_at,
		COALESCE(NEW.error, ''),
		COALESCE(
			(SELECT agent_name FROM automation_jobs WHERE id = NEW.job_id),
			(SELECT agent_name FROM automation_triggers WHERE id = NEW.trigger_id),
			''
		),
		COALESCE(
			NULLIF((SELECT workspace_id FROM automation_jobs WHERE id = NEW.job_id), ''),
			NULLIF((SELECT workspace_id FROM automation_triggers WHERE id = NEW.trigger_id), ''),
			NULLIF((SELECT workspace_id FROM loop_runs WHERE id = NEW.loop_run_id), ''),
			NULLIF((SELECT loop_workspace_id FROM automation_jobs WHERE id = NEW.job_id), ''),
			NULLIF((SELECT loop_workspace_id FROM automation_triggers WHERE id = NEW.trigger_id), ''),
			''
		),
		COALESCE(
			(SELECT retry FROM automation_jobs WHERE id = NEW.job_id),
			(SELECT retry FROM automation_triggers WHERE id = NEW.trigger_id),
			''
		)
	);
END`

const automationWatchEventsUpdateTriggerStatement = `CREATE TRIGGER automation_watch_events_after_terminal_update
AFTER UPDATE OF status ON automation_runs
WHEN NEW.status IN ('completed', 'failed') AND OLD.status NOT IN ('completed', 'failed')
BEGIN
	INSERT INTO automation_watch_events (
		run_id, job_id, trigger_id, session_id, status, attempt,
		started_at, ended_at, error, agent_name, workspace_id, retry_json
	) VALUES (
		NEW.id,
		COALESCE(NEW.job_id, ''),
		COALESCE(NEW.trigger_id, ''),
		COALESCE(NEW.session_id, ''),
		NEW.status,
		NEW.attempt,
		NEW.started_at,
		NEW.ended_at,
		COALESCE(NEW.error, ''),
		COALESCE(
			(SELECT agent_name FROM automation_jobs WHERE id = NEW.job_id),
			(SELECT agent_name FROM automation_triggers WHERE id = NEW.trigger_id),
			''
		),
		COALESCE(
			NULLIF((SELECT workspace_id FROM automation_jobs WHERE id = NEW.job_id), ''),
			NULLIF((SELECT workspace_id FROM automation_triggers WHERE id = NEW.trigger_id), ''),
			NULLIF((SELECT workspace_id FROM loop_runs WHERE id = NEW.loop_run_id), ''),
			NULLIF((SELECT loop_workspace_id FROM automation_jobs WHERE id = NEW.job_id), ''),
			NULLIF((SELECT loop_workspace_id FROM automation_triggers WHERE id = NEW.trigger_id), ''),
			''
		),
		COALESCE(
			(SELECT retry FROM automation_jobs WHERE id = NEW.job_id),
			(SELECT retry FROM automation_triggers WHERE id = NEW.trigger_id),
			''
		)
	);
END`

type networkWorkProjectionUpdate struct {
	seq          int64
	opened       bool
	transitioned bool
	state        string
}

func migrateWatchEventReplayProjections(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		automationWatchEventsTableStatement,
		`CREATE INDEX idx_automation_watch_events_workspace_status_seq
			ON automation_watch_events(workspace_id, status, seq)`,
		`INSERT INTO automation_watch_events (
			run_id, job_id, trigger_id, session_id, status, attempt,
			started_at, ended_at, error, agent_name, workspace_id, retry_json
		)
		SELECT
			ar.id,
			COALESCE(ar.job_id, ''),
			COALESCE(ar.trigger_id, ''),
			COALESCE(ar.session_id, ''),
			ar.status,
			ar.attempt,
			ar.started_at,
			ar.ended_at,
			COALESCE(ar.error, ''),
			COALESCE(aj.agent_name, at.agent_name, ''),
			COALESCE(
				NULLIF(aj.workspace_id, ''),
				NULLIF(at.workspace_id, ''),
				NULLIF(lr.workspace_id, ''),
				NULLIF(aj.loop_workspace_id, ''),
				NULLIF(at.loop_workspace_id, ''),
				''
			),
			COALESCE(aj.retry, at.retry, '')
		FROM automation_runs ar
		LEFT JOIN automation_jobs aj ON aj.id = ar.job_id
		LEFT JOIN automation_triggers at ON at.id = ar.trigger_id
		LEFT JOIN loop_runs lr ON lr.id = ar.loop_run_id
		WHERE ar.status IN ('completed', 'failed')
		ORDER BY ar.rowid ASC`,
		automationWatchEventsInsertTriggerStatement,
		automationWatchEventsUpdateTriggerStatement,
		`ALTER TABLE network_timeline_log
			ADD COLUMN work_opened INTEGER NOT NULL DEFAULT 0 CHECK (work_opened IN (0, 1))`,
		`ALTER TABLE network_timeline_log
			ADD COLUMN work_transitioned INTEGER NOT NULL DEFAULT 0 CHECK (work_transitioned IN (0, 1))`,
		`ALTER TABLE network_timeline_log
			ADD COLUMN work_state TEXT NOT NULL DEFAULT '' CHECK (
				work_state IN ('', 'submitted', 'working', 'needs_input', 'completed', 'failed', 'canceled')
			)`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: apply watch-event replay projection migration: %w", err)
		}
	}
	updates, err := readNetworkWorkProjectionUpdates(ctx, tx)
	if err != nil {
		return err
	}
	for _, update := range updates {
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE network_timeline_log
			 SET work_opened = ?, work_transitioned = ?, work_state = ?
			 WHERE rowid = ?`,
			update.opened,
			update.transitioned,
			update.state,
			update.seq,
		); err != nil {
			return fmt.Errorf("store: backfill network work projection at row %d: %w", update.seq, err)
		}
	}
	return nil
}

func readNetworkWorkProjectionUpdates(
	ctx context.Context,
	tx *sql.Tx,
) ([]networkWorkProjectionUpdate, error) {
	rows, err := tx.QueryContext(
		ctx,
		`SELECT rowid, workspace_id, COALESCE(work_id, ''), COALESCE(peer_to, ''), kind, body_json
		 FROM network_timeline_log
		 WHERE TRIM(COALESCE(work_id, '')) <> ''
		 ORDER BY rowid ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: read network timeline for work projection backfill: %w", err)
	}

	states := make(map[string]string)
	updates := make([]networkWorkProjectionUpdate, 0)
	for rows.Next() {
		var (
			seq       int64
			workspace string
			workID    string
			peerTo    string
			kind      string
			bodyRaw   string
		)
		if err := rows.Scan(&seq, &workspace, &workID, &peerTo, &kind, &bodyRaw); err != nil {
			return nil, joinRowsCloseError(
				rows,
				fmt.Errorf("store: scan network work projection backfill row: %w", err),
				"network work projection backfill query",
			)
		}
		entry := store.NetworkConversationMessage{
			WorkspaceID: strings.TrimSpace(workspace),
			WorkID:      strings.TrimSpace(workID),
			PeerTo:      strings.TrimSpace(peerTo),
			Kind:        strings.TrimSpace(kind),
			Body:        json.RawMessage(bodyRaw),
		}
		key := entry.WorkspaceID + "\x00" + entry.WorkID
		projection, projectionErr := networkWorkProjectionForState(states[key], entry)
		if projectionErr != nil {
			return nil, joinRowsCloseError(
				rows,
				fmt.Errorf("store: project network work timeline row %d: %w", seq, projectionErr),
				"network work projection backfill query",
			)
		}
		if projection.state != "" {
			states[key] = projection.state
		}
		updates = append(updates, networkWorkProjectionUpdate{
			seq:          seq,
			opened:       projection.opened,
			transitioned: projection.transitioned,
			state:        projection.state,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, joinRowsCloseError(
			rows,
			fmt.Errorf("store: iterate network work projection backfill: %w", err),
			"network work projection backfill query",
		)
	}
	if err := joinRowsCloseError(rows, nil, "network work projection backfill query"); err != nil {
		return nil, err
	}
	return updates, nil
}
