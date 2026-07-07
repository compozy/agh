package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateAutomationLoopTargets(ctx context.Context, tx *sql.Tx) error {
	jobsExist, err := tableExists(ctx, tx, "automation_jobs")
	if err != nil {
		return err
	}
	if jobsExist {
		if err := addMissingMigrationColumns(
			ctx,
			tx,
			"automation_jobs",
			automationJobLoopTargetColumnSpecs(),
		); err != nil {
			return err
		}
	}
	triggersExist, err := tableExists(ctx, tx, "automation_triggers")
	if err != nil {
		return err
	}
	if triggersExist {
		if err := addMissingMigrationColumns(
			ctx,
			tx,
			"automation_triggers",
			automationTriggerLoopTargetColumnSpecs(),
		); err != nil {
			return err
		}
	}
	runsExist, err := tableExists(ctx, tx, "automation_runs")
	if err != nil {
		return err
	}
	if runsExist {
		if err := addMissingMigrationColumns(
			ctx,
			tx,
			"automation_runs",
			automationRunLoopTargetColumnSpecs(),
		); err != nil {
			return err
		}
	}
	for _, statement := range automationLoopTargetIndexStatements(jobsExist, triggersExist, runsExist) {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: apply automation loop target index: %w", err)
		}
	}
	return nil
}

func automationJobLoopTargetColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{
			name: columnTargetKind,
			sql: `ALTER TABLE automation_jobs ADD COLUMN target_kind TEXT NOT NULL DEFAULT 'agent' ` +
				`CHECK (target_kind IN ('agent', 'loop'))`,
		},
		{
			name: columnLoopWorkspaceID,
			sql:  `ALTER TABLE automation_jobs ADD COLUMN loop_workspace_id TEXT REFERENCES workspaces(id) ON DELETE CASCADE`,
		},
		{
			name: columnLoopName,
			sql:  `ALTER TABLE automation_jobs ADD COLUMN loop_name TEXT`,
		},
		{
			name: columnLoopInputs,
			sql:  `ALTER TABLE automation_jobs ADD COLUMN loop_inputs TEXT`,
		},
		{
			name: columnLoopInputMapping,
			sql:  `ALTER TABLE automation_jobs ADD COLUMN loop_input_mapping TEXT`,
		},
	}
}

func automationTriggerLoopTargetColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{
			name: columnTargetKind,
			sql: `ALTER TABLE automation_triggers ADD COLUMN target_kind TEXT NOT NULL DEFAULT 'agent' ` +
				`CHECK (target_kind IN ('agent', 'loop'))`,
		},
		{
			name: columnLoopWorkspaceID,
			sql: `ALTER TABLE automation_triggers ADD COLUMN loop_workspace_id TEXT ` +
				`REFERENCES workspaces(id) ON DELETE CASCADE`,
		},
		{
			name: columnLoopName,
			sql:  `ALTER TABLE automation_triggers ADD COLUMN loop_name TEXT`,
		},
		{
			name: columnLoopInputs,
			sql:  `ALTER TABLE automation_triggers ADD COLUMN loop_inputs TEXT`,
		},
		{
			name: columnLoopInputMapping,
			sql:  `ALTER TABLE automation_triggers ADD COLUMN loop_input_mapping TEXT`,
		},
	}
}

func automationRunLoopTargetColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{
			name: columnLoopRunID,
			sql:  `ALTER TABLE automation_runs ADD COLUMN loop_run_id TEXT REFERENCES loop_runs(id) ON DELETE SET NULL`,
		},
	}
}

func automationLoopTargetIndexStatements(jobsExist bool, triggersExist bool, runsExist bool) []string {
	statements := []string{}
	if jobsExist {
		statements = append(statements, `CREATE INDEX IF NOT EXISTS idx_automation_jobs_loop_target
			ON automation_jobs(loop_name, loop_workspace_id) WHERE target_kind = 'loop'`)
	}
	if triggersExist {
		statements = append(statements, `CREATE INDEX IF NOT EXISTS idx_automation_triggers_loop_target
			ON automation_triggers(loop_name, loop_workspace_id) WHERE target_kind = 'loop'`)
	}
	if runsExist {
		statements = append(
			statements,
			`CREATE INDEX IF NOT EXISTS idx_automation_runs_loop_run ON automation_runs(loop_run_id)`,
		)
	}
	return statements
}
