package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	automation "github.com/compozy/agh/internal/automation/model"
	"github.com/compozy/agh/internal/store"
)

const (
	automationJobCatalogTable              = "automation_job_catalog_entries"
	automationTriggerCatalogTable          = "automation_trigger_catalog_entries"
	automationTriggerCatalogFilterTable    = "automation_trigger_catalog_filter_terms"
	automationJobCatalogOrderIndex         = "idx_automation_job_catalog_order"
	automationJobCatalogWorkspaceIndex     = "idx_automation_job_catalog_workspace_order"
	automationTriggerCatalogOrderIndex     = "idx_automation_trigger_catalog_order"
	automationTriggerCatalogWorkspaceIndex = "idx_automation_trigger_catalog_workspace_order"
)

var automationCatalogProjectionMigration = store.Migration{
	Version:  70,
	Name:     "add_automation_catalog_projections",
	Up:       migrateAutomationCatalogProjections,
	Checksum: "2026-07-10-add-automation-catalog-projections",
}

func migrateAutomationCatalogProjections(ctx context.Context, tx *sql.Tx) error {
	if err := normalizeEmptyAutomationAgentLoopTargets(ctx, tx); err != nil {
		return err
	}
	for _, statement := range automationCatalogSchemaStatements() {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: create automation catalog projection schema: %w", err)
		}
	}
	jobs, err := readAutomationJobsForCatalogMigration(ctx, tx)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if err := upsertAutomationJobCatalog(ctx, tx, job); err != nil {
			return fmt.Errorf("store: backfill automation job catalog %q: %w", job.ID, err)
		}
	}
	triggers, err := readAutomationTriggersForCatalogMigration(ctx, tx)
	if err != nil {
		return err
	}
	for _, trigger := range triggers {
		if err := upsertAutomationTriggerCatalog(ctx, tx, trigger); err != nil {
			return fmt.Errorf("store: backfill automation trigger catalog %q: %w", trigger.ID, err)
		}
	}
	return nil
}

func normalizeEmptyAutomationAgentLoopTargets(ctx context.Context, tx *sql.Tx) error {
	const normalizationSQL = `
		SET loop_workspace_id = NULL,
			loop_name = NULL,
			loop_inputs = NULL,
			loop_input_mapping = NULL
		WHERE target_kind = 'agent'
			AND (loop_workspace_id IS NOT NULL OR loop_name IS NOT NULL
				OR loop_inputs IS NOT NULL OR loop_input_mapping IS NOT NULL)
			AND NULLIF(TRIM(loop_workspace_id), '') IS NULL
			AND NULLIF(TRIM(loop_name), '') IS NULL
			AND CASE
				WHEN loop_inputs IS NULL OR TRIM(loop_inputs) = '' THEN 1
				WHEN json_valid(loop_inputs) THEN json(loop_inputs) = '{}'
				ELSE 0
			END
			AND CASE
				WHEN loop_input_mapping IS NULL OR TRIM(loop_input_mapping) = '' THEN 1
				WHEN json_valid(loop_input_mapping) THEN json(loop_input_mapping) = '{}'
				ELSE 0
			END`
	statements := []struct {
		label string
		sql   string
	}{
		{
			label: "job",
			sql:   `UPDATE automation_jobs` + normalizationSQL,
		},
		{
			label: "trigger",
			sql:   `UPDATE automation_triggers` + normalizationSQL,
		},
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement.sql); err != nil {
			return fmt.Errorf("store: normalize empty automation %s agent loop target: %w", statement.label, err)
		}
	}
	return nil
}

func automationCatalogSchemaStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS automation_job_catalog_entries (
			job_id                   TEXT PRIMARY KEY REFERENCES automation_jobs(id) ON DELETE CASCADE,
			scope                    TEXT NOT NULL,
			workspace_id             TEXT NOT NULL DEFAULT '',
			source                   TEXT NOT NULL,
			source_rank              INTEGER NOT NULL,
			name                     TEXT NOT NULL,
			loop_name                TEXT NOT NULL DEFAULT '',
			enabled                  BOOLEAN NOT NULL,
			search_name              TEXT NOT NULL,
			search_agent_name        TEXT NOT NULL,
			search_prompt            TEXT NOT NULL,
			search_scope             TEXT NOT NULL,
			search_source            TEXT NOT NULL,
			search_schedule_mode     TEXT NOT NULL,
			search_schedule_expr     TEXT NOT NULL,
			search_schedule_interval TEXT NOT NULL,
			search_schedule_time     TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_automation_job_catalog_order
			ON automation_job_catalog_entries(source_rank, name, job_id)`,
		`CREATE INDEX IF NOT EXISTS idx_automation_job_catalog_workspace_order
			ON automation_job_catalog_entries(workspace_id, source_rank, name, job_id)`,
		`CREATE TABLE IF NOT EXISTS automation_trigger_catalog_entries (
			trigger_id           TEXT PRIMARY KEY REFERENCES automation_triggers(id) ON DELETE CASCADE,
			scope                TEXT NOT NULL,
			workspace_id         TEXT NOT NULL DEFAULT '',
			event                TEXT NOT NULL,
			source               TEXT NOT NULL,
			source_rank          INTEGER NOT NULL,
			name                 TEXT NOT NULL,
			loop_name            TEXT NOT NULL DEFAULT '',
			enabled              BOOLEAN NOT NULL,
			search_name          TEXT NOT NULL,
			search_agent_name    TEXT NOT NULL,
			search_prompt        TEXT NOT NULL,
			search_scope         TEXT NOT NULL,
			search_source        TEXT NOT NULL,
			search_event         TEXT NOT NULL,
			search_endpoint_slug TEXT NOT NULL,
			search_webhook_id    TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_automation_trigger_catalog_order
			ON automation_trigger_catalog_entries(source_rank, name, trigger_id)`,
		`CREATE INDEX IF NOT EXISTS idx_automation_trigger_catalog_workspace_order
			ON automation_trigger_catalog_entries(workspace_id, source_rank, name, trigger_id)`,
		`CREATE TABLE IF NOT EXISTS automation_trigger_catalog_filter_terms (
			trigger_id TEXT NOT NULL REFERENCES automation_triggers(id) ON DELETE CASCADE,
			value      TEXT NOT NULL,
			PRIMARY KEY (trigger_id, value)
		)`,
	}
}

func readAutomationJobsForCatalogMigration(
	ctx context.Context,
	tx *sql.Tx,
) (jobs []automation.Job, err error) {
	rows, err := tx.QueryContext(ctx, automationJobRichSelectSQL)
	if err != nil {
		return nil, fmt.Errorf("store: query automation jobs for catalog backfill: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("store: close automation job catalog backfill rows: %w", closeErr))
		}
	}()
	for rows.Next() {
		job, scanErr := scanAutomationJob(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate automation job catalog backfill rows: %w", err)
	}
	return jobs, nil
}

func readAutomationTriggersForCatalogMigration(
	ctx context.Context,
	tx *sql.Tx,
) (triggers []automation.Trigger, err error) {
	rows, err := tx.QueryContext(ctx, automationTriggerRichSelectSQL)
	if err != nil {
		return nil, fmt.Errorf("store: query automation triggers for catalog backfill: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("store: close automation trigger catalog backfill rows: %w", closeErr))
		}
	}()
	for rows.Next() {
		trigger, scanErr := scanAutomationTrigger(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		triggers = append(triggers, trigger)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate automation trigger catalog backfill rows: %w", err)
	}
	return triggers, nil
}
