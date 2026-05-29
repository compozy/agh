package globaldb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	eventspkg "github.com/compozy/agh/internal/events"
	presetspkg "github.com/compozy/agh/internal/notifications/presets"
)

const eventSummaryProviderColumn = "provider"

const eventSummaryProviderBackfillSQL = "UPDATE event_summaries SET provider = COALESCE(" +
	"(SELECT provider FROM sessions WHERE sessions.id = event_summaries.session_id), '') " +
	"WHERE trim(provider) = '' AND trim(session_id) <> ''"

const eventSummaryOutcomeBackfillSQL = "UPDATE event_summaries SET outcome = COALESCE(" +
	"(SELECT outcome FROM event_summary_outcome_backfill " +
	"WHERE event_summary_outcome_backfill.type = event_summaries.type), ?) WHERE outcome = ?"

func migrateConfigApplyRecords(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS config_apply_records (
			id                  TEXT PRIMARY KEY,
			desired_config_hash TEXT NOT NULL,
			active_config_hash  TEXT NOT NULL,
			generation          INTEGER NOT NULL CHECK (generation >= 0),
			actor               TEXT NOT NULL,
			diff_class          TEXT NOT NULL,
			status              TEXT NOT NULL CHECK (status IN ('pending_apply', 'applied', 'blocked', 'failed')),
			diagnostic_json     TEXT NOT NULL DEFAULT '',
			created_at          TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			applied_at          TEXT,
			updated_at          TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_config_apply_records_desired
			ON config_apply_records(desired_config_hash, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_config_apply_records_active
			ON config_apply_records(active_config_hash, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_config_apply_records_generation
			ON config_apply_records(generation DESC, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_config_apply_records_actor
			ON config_apply_records(actor, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_config_apply_records_status
			ON config_apply_records(status, updated_at DESC);`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: apply config apply records schema: %w", err)
		}
	}
	return nil
}

func migrateEventSummaryProjections(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "event_summaries")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if err := addMissingMigrationColumns(ctx, tx, "event_summaries", []migrationColumnSpec{
		{
			name: eventSummaryProviderColumn,
			sql:  `ALTER TABLE event_summaries ADD COLUMN provider TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: globalDBOutcomeKey,
			sql:  `ALTER TABLE event_summaries ADD COLUMN outcome TEXT NOT NULL DEFAULT 'info'`,
		},
	}); err != nil {
		return err
	}

	for _, statement := range []string{
		idxSummaryProviderTimestampSQL,
		idxSummaryOutcomeTimestampSQL,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: apply event summary projection index: %w", err)
		}
	}

	sessionsExists, err := tableExists(ctx, tx, "sessions")
	if err != nil {
		return err
	}
	if sessionsExists {
		if _, err := tx.ExecContext(
			ctx,
			eventSummaryProviderBackfillSQL,
		); err != nil {
			return fmt.Errorf("store: backfill event summary provider: %w", err)
		}
	}

	if err := backfillEventSummaryOutcomes(ctx, tx); err != nil {
		return err
	}

	return nil
}

func migrateSchedulerPauseState(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS scheduler_pause (
			id         INTEGER PRIMARY KEY CHECK (id = 1),
			paused     INTEGER NOT NULL DEFAULT 0 CHECK (paused IN (0, 1)),
			paused_by  TEXT NOT NULL DEFAULT '',
			paused_at  TEXT,
			reason     TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`INSERT OR IGNORE INTO scheduler_pause (id, paused, paused_by, reason) VALUES (1, 0, '', '');`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: apply scheduler pause schema: %w", err)
		}
	}
	return nil
}

// healSchedulerPauseUpdatedAtSQL rewrites a non-canonical scheduler_pause.updated_at (the
// SQLite CURRENT_TIMESTAMP form the v29 seed produced) into the canonical RFC3339Nano layout.
// The NOT LIKE '%T%' guard makes it idempotent: canonical values already contain 'T'.
const healSchedulerPauseUpdatedAtSQL = `UPDATE scheduler_pause
	SET updated_at = strftime('%Y-%m-%dT%H:%M:%S', updated_at) || '.000000000Z'
	WHERE id = 1
	  AND updated_at NOT LIKE '%T%'
	  AND strftime('%Y-%m-%dT%H:%M:%S', updated_at) IS NOT NULL`

func migrateHealSchedulerPauseUpdatedAt(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, healSchedulerPauseUpdatedAtSQL); err != nil {
		return fmt.Errorf("store: heal scheduler pause updated_at: %w", err)
	}
	return nil
}

func migrateTaskRunForceOps(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "task_runs")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if err := addMissingMigrationColumns(ctx, tx, "task_runs", []migrationColumnSpec{
		{
			name: "previous_run_id",
			sql:  `ALTER TABLE task_runs ADD COLUMN previous_run_id TEXT`,
		},
		{
			name: migrateWorkspaceFailureKindKey,
			sql: `ALTER TABLE task_runs ADD COLUMN failure_kind TEXT NOT NULL DEFAULT '' CHECK (
				failure_kind = '' OR failure_kind IN ('operator_forced')
			)`,
		},
	}); err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`CREATE INDEX IF NOT EXISTS idx_task_runs_previous ON task_runs(previous_run_id);`,
	); err != nil {
		return fmt.Errorf("store: create task run previous index: %w", err)
	}
	return nil
}

func migratePauseState(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "tasks")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if err := addMissingMigrationColumns(ctx, tx, "tasks", []migrationColumnSpec{
		{
			name: "paused",
			sql:  `ALTER TABLE tasks ADD COLUMN paused INTEGER NOT NULL DEFAULT 0 CHECK (paused IN (0, 1))`,
		},
		{
			name: "paused_by",
			sql:  `ALTER TABLE tasks ADD COLUMN paused_by TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: "paused_at",
			sql:  `ALTER TABLE tasks ADD COLUMN paused_at TEXT`,
		},
		{
			name: "paused_reason",
			sql:  `ALTER TABLE tasks ADD COLUMN paused_reason TEXT NOT NULL DEFAULT ''`,
		},
	}); err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`CREATE INDEX IF NOT EXISTS idx_tasks_paused ON tasks(paused, updated_at DESC);`,
	); err != nil {
		return fmt.Errorf("store: create task paused index: %w", err)
	}
	return nil
}

func migrateExtensionProvenance(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "extensions")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	columns := []migrationColumnSpec{
		{
			name: globalDBExtensionProvenanceJSONKey,
			sql:  "ALTER TABLE extensions ADD COLUMN " + globalDBExtensionProvenanceJSONKey + " TEXT NOT NULL DEFAULT '{}'",
		},
	}
	if addErr := addMissingMigrationColumns(ctx, tx, "extensions", columns); addErr != nil {
		return addErr
	}
	return nil
}

func migrateBridgeTargetDirectory(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS bridge_target_directory (
			bridge_id       TEXT NOT NULL REFERENCES bridge_instances(id) ON DELETE CASCADE,
			canonical_route TEXT NOT NULL,
			display_name    TEXT NOT NULL,
			normalized      TEXT NOT NULL,
			target_type     TEXT NOT NULL CHECK (target_type IN ('channel','user','room','thread','group')),
			qualifier       TEXT NOT NULL DEFAULT '',
			capabilities    TEXT NOT NULL DEFAULT '',
			updated_at      TEXT NOT NULL,
			last_seen_at    TEXT,
			PRIMARY KEY (bridge_id, canonical_route)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_btd_bridge_norm
			ON bridge_target_directory(bridge_id, normalized);`,
		`CREATE INDEX IF NOT EXISTS idx_btd_bridge_qualifier
			ON bridge_target_directory(bridge_id, qualifier);`,
		`CREATE TABLE IF NOT EXISTS bridge_target_directory_refresh (
			bridge_id                  TEXT PRIMARY KEY REFERENCES bridge_instances(id) ON DELETE CASCADE,
			last_successful_refresh_at TEXT NOT NULL
		);`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: apply bridge target directory schema: %w", err)
		}
	}
	return nil
}

func migrateNotificationPresets(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "bridge_instances")
	if err != nil {
		return err
	}
	if exists {
		if err := addMissingMigrationColumns(ctx, tx, "bridge_instances", []migrationColumnSpec{
			{
				name: globalDBBridgeNotificationSuppressColumn,
				sql:  `ALTER TABLE bridge_instances ADD COLUMN notification_suppress BOOLEAN NOT NULL DEFAULT 0`,
			},
		}); err != nil {
			return err
		}
	}
	for _, statement := range notificationPresetSchemaStatements() {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: create notification preset schema: %w", err)
		}
	}
	for _, defaultPreset := range presetspkg.BuiltInPresets(time.Now().UTC()) {
		if err := seedNotificationPresetDefault(ctx, tx, defaultPreset, time.Now().UTC()); err != nil {
			return err
		}
	}
	return nil
}

func backfillEventSummaryOutcomes(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TEMP TABLE IF NOT EXISTS event_summary_outcome_backfill (
			type    TEXT PRIMARY KEY,
			outcome TEXT NOT NULL
		);`,
		`DELETE FROM event_summary_outcome_backfill;`,
	}
	for _, statement := range statements {
		_, execErr := tx.ExecContext(ctx, statement)
		if execErr != nil {
			return fmt.Errorf("store: prepare event summary outcome backfill: %w", execErr)
		}
	}

	for _, meta := range eventspkg.All() {
		_, execErr := tx.ExecContext(
			ctx,
			`INSERT INTO event_summary_outcome_backfill (type, outcome) VALUES (?, ?)`,
			meta.Name,
			string(meta.Outcome),
		)
		if execErr != nil {
			return fmt.Errorf("store: stage event summary outcome for %s: %w", meta.Name, execErr)
		}
	}

	_, execErr := tx.ExecContext(
		ctx,
		eventSummaryOutcomeBackfillSQL,
		string(eventspkg.OutcomeInfo),
		string(eventspkg.OutcomeInfo),
	)
	if execErr != nil {
		return fmt.Errorf("store: backfill event summary outcomes: %w", execErr)
	}

	_, execErr = tx.ExecContext(ctx, `DROP TABLE event_summary_outcome_backfill`)
	if execErr != nil {
		return fmt.Errorf("store: drop event summary outcome backfill table: %w", execErr)
	}
	return nil
}
