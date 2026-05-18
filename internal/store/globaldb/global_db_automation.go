package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	automation "github.com/pedronauck/agh/internal/automation/model"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

// CreateJob stores a new automation job definition.
func (g *GlobalDB) CreateJob(ctx context.Context, job automation.Job) (automation.Job, error) {
	if err := g.checkReady(ctx, "create automation job"); err != nil {
		return automation.Job{}, err
	}

	normalized, err := g.normalizeJobForCreate(job)
	if err != nil {
		return automation.Job{}, err
	}
	if err := g.insertJob(ctx, g.db, normalized); err != nil {
		return automation.Job{}, fmt.Errorf("store: create automation job %q: %w", normalized.ID, err)
	}

	return normalized, nil
}

// UpdateJob replaces the mutable fields of a persisted automation job definition.
func (g *GlobalDB) UpdateJob(ctx context.Context, job automation.Job) (automation.Job, error) {
	if err := g.checkReady(ctx, "update automation job"); err != nil {
		return automation.Job{}, err
	}

	normalized, err := g.normalizeJobForUpdate(job)
	if err != nil {
		return automation.Job{}, err
	}

	scheduleJSON, taskJSON, retryJSON, fireLimitJSON, err := encodeJobRecord(normalized)
	if err != nil {
		return automation.Job{}, err
	}

	result, err := g.db.ExecContext(
		ctx,
		`UPDATE automation_jobs
		 SET scope = ?, name = ?, agent_name = ?, workspace_id = ?, prompt = ?,
		     schedule = ?, task = ?, enabled = ?, retry = ?, fire_limit = ?,
		     source = ?, updated_at = ?
		 WHERE id = ?`,
		normalized.Scope,
		normalized.Name,
		normalized.AgentName,
		store.NullableString(normalized.WorkspaceID),
		normalized.Prompt,
		scheduleJSON,
		taskJSON,
		normalized.Enabled,
		retryJSON,
		fireLimitJSON,
		normalized.Source,
		store.FormatTimestamp(normalized.UpdatedAt),
		normalized.ID,
	)
	if err != nil {
		return automation.Job{}, fmt.Errorf(
			"store: update automation job %q: %w",
			normalized.ID,
			mapAutomationJobConstraintError(err),
		)
	}

	if err := requireRowsAffected(result, automation.ErrJobNotFound, normalized.ID, "automation job"); err != nil {
		return automation.Job{}, err
	}

	return g.GetJob(ctx, normalized.ID)
}

// DeleteJob removes an automation job definition.
func (g *GlobalDB) DeleteJob(ctx context.Context, id string) error {
	if err := g.checkReady(ctx, "delete automation job"); err != nil {
		return err
	}

	trimmedID, err := requireAutomationID(id, "automation job id")
	if err != nil {
		return err
	}

	result, err := g.db.ExecContext(ctx, `DELETE FROM automation_jobs WHERE id = ?`, trimmedID)
	if err != nil {
		return fmt.Errorf("store: delete automation job %q: %w", trimmedID, mapAutomationJobConstraintError(err))
	}

	return requireRowsAffected(result, automation.ErrJobNotFound, trimmedID, "automation job")
}

// GetJob loads one persisted automation job definition by primary key.
func (g *GlobalDB) GetJob(ctx context.Context, id string) (automation.Job, error) {
	if err := g.checkReady(ctx, "get automation job"); err != nil {
		return automation.Job{}, err
	}

	trimmedID, err := requireAutomationID(id, "automation job id")
	if err != nil {
		return automation.Job{}, err
	}

	return g.getJobByQuery(
		ctx,
		`SELECT
			id, scope, name, agent_name, workspace_id, prompt, schedule, task,
			enabled, retry, fire_limit, source, created_at, updated_at
		 FROM automation_jobs
		 WHERE id = ?`,
		trimmedID,
	)
}

// ListJobs returns persisted automation jobs using the supplied filters.
func (g *GlobalDB) ListJobs(ctx context.Context, query automation.JobListQuery) ([]automation.Job, error) {
	if err := g.checkReady(ctx, "list automation jobs"); err != nil {
		return nil, err
	}
	if err := validateAutomationJobListQuery(query); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT
		id, scope, name, agent_name, workspace_id, prompt, schedule, task,
		enabled, retry, fire_limit, source, created_at, updated_at
		FROM automation_jobs`
	where, args := store.BuildClauses(
		store.StringClause("scope", string(query.Scope)),
		store.StringClause("workspace_id", query.WorkspaceID),
		store.StringClause("source", string(query.Source)),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY name ASC, id ASC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query automation jobs: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	jobs := make([]automation.Job, 0)
	for rows.Next() {
		job, scanErr := scanAutomationJob(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate automation jobs: %w", err)
	}

	return jobs, nil
}

// CreateTrigger stores a new automation trigger definition.
func (g *GlobalDB) CreateTrigger(ctx context.Context, trigger automation.Trigger) (automation.Trigger, error) {
	if err := g.checkReady(ctx, "create automation trigger"); err != nil {
		return automation.Trigger{}, err
	}

	normalized, err := g.normalizeTriggerForCreate(trigger)
	if err != nil {
		return automation.Trigger{}, err
	}
	if err := g.insertTrigger(ctx, g.db, normalized); err != nil {
		return automation.Trigger{}, fmt.Errorf("store: create automation trigger %q: %w", normalized.ID, err)
	}

	return normalized, nil
}

// UpdateTrigger replaces the mutable fields of a persisted automation trigger definition.
func (g *GlobalDB) UpdateTrigger(ctx context.Context, trigger automation.Trigger) (automation.Trigger, error) {
	if err := g.checkReady(ctx, "update automation trigger"); err != nil {
		return automation.Trigger{}, err
	}

	normalized, err := g.normalizeTriggerForUpdate(trigger)
	if err != nil {
		return automation.Trigger{}, err
	}

	filterJSON, retryJSON, fireLimitJSON, err := encodeTriggerRecord(normalized)
	if err != nil {
		return automation.Trigger{}, err
	}

	result, err := g.db.ExecContext(
		ctx,
		`UPDATE automation_triggers
		 SET scope = ?, name = ?, agent_name = ?, workspace_id = ?, prompt = ?,
		     event = ?, filter = ?, enabled = ?, retry = ?, fire_limit = ?,
		     source = ?, webhook_id = ?, endpoint_slug = ?, webhook_secret_ref = ?, updated_at = ?
		 WHERE id = ?`,
		normalized.Scope,
		normalized.Name,
		normalized.AgentName,
		store.NullableString(normalized.WorkspaceID),
		normalized.Prompt,
		normalized.Event,
		filterJSON,
		normalized.Enabled,
		retryJSON,
		fireLimitJSON,
		normalized.Source,
		store.NullableString(normalized.WebhookID),
		store.NullableString(normalized.EndpointSlug),
		store.NullableString(normalized.WebhookSecretRef),
		store.FormatTimestamp(normalized.UpdatedAt),
		normalized.ID,
	)
	if err != nil {
		return automation.Trigger{}, fmt.Errorf(
			"store: update automation trigger %q: %w",
			normalized.ID,
			mapAutomationTriggerConstraintError(err),
		)
	}

	if err := requireRowsAffected(
		result,
		automation.ErrTriggerNotFound,
		normalized.ID,
		"automation trigger",
	); err != nil {
		return automation.Trigger{}, err
	}

	return g.GetTrigger(ctx, normalized.ID)
}

// DeleteTrigger removes an automation trigger definition.
func (g *GlobalDB) DeleteTrigger(ctx context.Context, id string) error {
	if err := g.checkReady(ctx, "delete automation trigger"); err != nil {
		return err
	}

	trimmedID, err := requireAutomationID(id, "automation trigger id")
	if err != nil {
		return err
	}

	result, err := g.db.ExecContext(ctx, `DELETE FROM automation_triggers WHERE id = ?`, trimmedID)
	if err != nil {
		return fmt.Errorf(
			"store: delete automation trigger %q: %w",
			trimmedID,
			mapAutomationTriggerConstraintError(err),
		)
	}

	return requireRowsAffected(result, automation.ErrTriggerNotFound, trimmedID, "automation trigger")
}

// GetTrigger loads one persisted automation trigger definition by primary key.
func (g *GlobalDB) GetTrigger(ctx context.Context, id string) (automation.Trigger, error) {
	if err := g.checkReady(ctx, "get automation trigger"); err != nil {
		return automation.Trigger{}, err
	}

	trimmedID, err := requireAutomationID(id, "automation trigger id")
	if err != nil {
		return automation.Trigger{}, err
	}

	return g.getTriggerByQuery(
		ctx,
		`SELECT
				id, scope, name, agent_name, workspace_id, prompt, event, filter,
				enabled, retry, fire_limit, source, webhook_id, endpoint_slug,
				webhook_secret_ref, created_at, updated_at
			 FROM automation_triggers
			 WHERE id = ?`,
		trimmedID,
	)
}

// GetTriggerByWebhookID loads a webhook trigger using its stable webhook identifier.
func (g *GlobalDB) GetTriggerByWebhookID(ctx context.Context, webhookID string) (automation.Trigger, error) {
	if err := g.checkReady(ctx, "get automation trigger by webhook id"); err != nil {
		return automation.Trigger{}, err
	}

	trimmedWebhookID, err := requireAutomationID(webhookID, "automation trigger webhook id")
	if err != nil {
		return automation.Trigger{}, err
	}

	return g.getTriggerByQuery(
		ctx,
		`SELECT
				id, scope, name, agent_name, workspace_id, prompt, event, filter,
				enabled, retry, fire_limit, source, webhook_id, endpoint_slug,
				webhook_secret_ref, created_at, updated_at
			 FROM automation_triggers
			 WHERE webhook_id = ?`,
		trimmedWebhookID,
	)
}

// ListTriggers returns persisted automation triggers using the supplied filters.
func (g *GlobalDB) ListTriggers(ctx context.Context, query automation.TriggerListQuery) ([]automation.Trigger, error) {
	if err := g.checkReady(ctx, "list automation triggers"); err != nil {
		return nil, err
	}
	if err := validateAutomationTriggerListQuery(query); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT id, scope, name, agent_name, workspace_id, prompt, event, filter, enabled, retry, fire_limit, source, webhook_id, endpoint_slug, webhook_secret_ref, created_at, updated_at FROM automation_triggers`
	where, args := store.BuildClauses(
		store.StringClause("scope", string(query.Scope)),
		store.StringClause("workspace_id", query.WorkspaceID),
		store.StringClause("event", query.Event),
		store.StringClause("source", string(query.Source)),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY name ASC, id ASC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query automation triggers: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	triggers := make([]automation.Trigger, 0)
	for rows.Next() {
		trigger, scanErr := scanAutomationTrigger(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		triggers = append(triggers, trigger)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate automation triggers: %w", err)
	}

	return triggers, nil
}

// CreateRun stores a new automation run history row.
func (g *GlobalDB) CreateRun(ctx context.Context, run automation.Run) (automation.Run, error) {
	if err := g.checkReady(ctx, "create automation run"); err != nil {
		return automation.Run{}, err
	}

	normalized, err := g.normalizeRunForCreate(run)
	if err != nil {
		return automation.Run{}, err
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO automation_runs (
			id, job_id, trigger_id, session_id, task_id, task_run_id, fire_id,
			status, attempt, scheduled_at, started_at, ended_at, error,
			delivery_error, delivery_error_at
		)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		normalized.ID,
		store.NullableString(normalized.JobID),
		store.NullableString(normalized.TriggerID),
		store.NullableString(normalized.SessionID),
		store.NullableString(normalized.TaskID),
		store.NullableString(normalized.TaskRunID),
		store.NullableString(normalized.FireID),
		normalized.Status,
		normalized.Attempt,
		nullableAutomationTimestamp(normalized.ScheduledAt),
		nullableAutomationTimestamp(normalized.StartedAt),
		nullableAutomationTimestamp(normalized.EndedAt),
		store.NullableString(normalized.Error),
		store.NullableString(normalized.DeliveryError),
		nullableAutomationTimestamp(normalized.DeliveryErrorAt),
	); err != nil {
		return automation.Run{}, fmt.Errorf(
			"store: create automation run %q: %w",
			normalized.ID,
			mapAutomationRunConstraintError(err),
		)
	}

	return normalized, nil
}

// UpdateRun replaces the mutable fields of a persisted automation run.
func (g *GlobalDB) UpdateRun(ctx context.Context, run automation.Run) (automation.Run, error) {
	if err := g.checkReady(ctx, "update automation run"); err != nil {
		return automation.Run{}, err
	}

	normalized, err := g.normalizeRunForUpdate(run)
	if err != nil {
		return automation.Run{}, err
	}

	result, err := g.db.ExecContext(
		ctx,
		`UPDATE automation_runs
		 SET job_id = ?, trigger_id = ?, session_id = ?, task_id = ?,
		     task_run_id = ?, fire_id = ?, status = ?, attempt = ?,
		     scheduled_at = ?, started_at = ?, ended_at = ?, error = ?,
		     delivery_error = ?, delivery_error_at = ?
		 WHERE id = ?`,
		store.NullableString(normalized.JobID),
		store.NullableString(normalized.TriggerID),
		store.NullableString(normalized.SessionID),
		store.NullableString(normalized.TaskID),
		store.NullableString(normalized.TaskRunID),
		store.NullableString(normalized.FireID),
		normalized.Status,
		normalized.Attempt,
		nullableAutomationTimestamp(normalized.ScheduledAt),
		nullableAutomationTimestamp(normalized.StartedAt),
		nullableAutomationTimestamp(normalized.EndedAt),
		store.NullableString(normalized.Error),
		store.NullableString(normalized.DeliveryError),
		nullableAutomationTimestamp(normalized.DeliveryErrorAt),
		normalized.ID,
	)
	if err != nil {
		return automation.Run{}, fmt.Errorf(
			"store: update automation run %q: %w",
			normalized.ID,
			mapAutomationRunConstraintError(err),
		)
	}

	if err := requireRowsAffected(result, automation.ErrRunNotFound, normalized.ID, "automation run"); err != nil {
		return automation.Run{}, err
	}

	return g.GetRun(ctx, normalized.ID)
}

// DeleteRun removes an automation run history row.
func (g *GlobalDB) DeleteRun(ctx context.Context, id string) error {
	if err := g.checkReady(ctx, "delete automation run"); err != nil {
		return err
	}

	trimmedID, err := requireAutomationID(id, "automation run id")
	if err != nil {
		return err
	}

	result, err := g.db.ExecContext(ctx, `DELETE FROM automation_runs WHERE id = ?`, trimmedID)
	if err != nil {
		return fmt.Errorf("store: delete automation run %q: %w", trimmedID, err)
	}

	return requireRowsAffected(result, automation.ErrRunNotFound, trimmedID, "automation run")
}

// GetRun loads one persisted automation run by primary key.
func (g *GlobalDB) GetRun(ctx context.Context, id string) (automation.Run, error) {
	if err := g.checkReady(ctx, "get automation run"); err != nil {
		return automation.Run{}, err
	}

	trimmedID, err := requireAutomationID(id, "automation run id")
	if err != nil {
		return automation.Run{}, err
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT
			id, job_id, trigger_id, session_id, task_id, task_run_id, fire_id,
			status, attempt, scheduled_at, started_at, ended_at, error,
			delivery_error, delivery_error_at
		 FROM automation_runs
		 WHERE id = ?`,
		trimmedID,
	)
	run, err := scanAutomationRun(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return automation.Run{}, automation.ErrRunNotFound
		}
		return automation.Run{}, err
	}

	return run, nil
}

// ListRuns returns filtered automation run history rows.
func (g *GlobalDB) ListRuns(ctx context.Context, query automation.RunQuery) ([]automation.Run, error) {
	if err := g.checkReady(ctx, "list automation runs"); err != nil {
		return nil, err
	}
	if err := validateAutomationRunQuery(query); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT
		id, job_id, trigger_id, session_id, task_id, task_run_id, fire_id,
		status, attempt, scheduled_at, started_at, ended_at, error,
		delivery_error, delivery_error_at
		FROM automation_runs`
	where, args := buildAutomationRunClauses(query)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY started_at DESC, id DESC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query automation runs: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	runs := make([]automation.Run, 0)
	for rows.Next() {
		run, scanErr := scanAutomationRun(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate automation runs: %w", err)
	}

	return runs, nil
}

// CountRuns returns the number of automation runs matching the supplied filters.
func (g *GlobalDB) CountRuns(ctx context.Context, query automation.RunQuery) (int64, error) {
	if err := g.checkReady(ctx, "count automation runs"); err != nil {
		return 0, err
	}
	if err := validateAutomationRunQuery(query); err != nil {
		return 0, err
	}

	sqlQuery := `SELECT COUNT(*) FROM automation_runs`
	where, args := buildAutomationRunClauses(query)
	sqlQuery = store.AppendWhere(sqlQuery, where)

	var count int64
	if err := g.db.QueryRowContext(ctx, sqlQuery, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: count automation runs: %w", err)
	}

	return count, nil
}

// SetJobEnabledOverlay upserts the runtime enabled override for a config-backed job.
func (g *GlobalDB) SetJobEnabledOverlay(
	ctx context.Context,
	overlay automation.JobEnabledOverlay,
) (automation.JobEnabledOverlay, error) {
	if err := g.checkReady(ctx, "set automation job overlay"); err != nil {
		return automation.JobEnabledOverlay{}, err
	}

	normalized, err := normalizeJobOverlay(overlay, g.now())
	if err != nil {
		return automation.JobEnabledOverlay{}, err
	}
	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO automation_job_overlays (job_id, enabled_override, updated_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(job_id) DO UPDATE SET
			enabled_override = excluded.enabled_override,
			updated_at = excluded.updated_at`,
		normalized.JobID,
		normalized.EnabledOverride,
		store.FormatTimestamp(normalized.UpdatedAt),
	); err != nil {
		return automation.JobEnabledOverlay{}, fmt.Errorf(
			"store: set automation job overlay %q: %w",
			normalized.JobID,
			err,
		)
	}

	return normalized, nil
}

// GetJobEnabledOverlay loads one persisted job enabled overlay by job id.
func (g *GlobalDB) GetJobEnabledOverlay(ctx context.Context, jobID string) (automation.JobEnabledOverlay, error) {
	if err := g.checkReady(ctx, "get automation job overlay"); err != nil {
		return automation.JobEnabledOverlay{}, err
	}

	trimmedID, err := requireAutomationID(jobID, "automation job overlay id")
	if err != nil {
		return automation.JobEnabledOverlay{}, err
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT job_id, enabled_override, updated_at
		 FROM automation_job_overlays
		 WHERE job_id = ?`,
		trimmedID,
	)
	overlay, err := scanJobEnabledOverlay(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return automation.JobEnabledOverlay{}, automation.ErrJobOverlayNotFound
		}
		return automation.JobEnabledOverlay{}, err
	}

	return overlay, nil
}

// ListJobEnabledOverlays returns all persisted job enabled overlays.
func (g *GlobalDB) ListJobEnabledOverlays(ctx context.Context) ([]automation.JobEnabledOverlay, error) {
	if err := g.checkReady(ctx, "list automation job overlays"); err != nil {
		return nil, err
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT job_id, enabled_override, updated_at
		 FROM automation_job_overlays
		 ORDER BY job_id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query automation job overlays: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	overlays := make([]automation.JobEnabledOverlay, 0)
	for rows.Next() {
		overlay, scanErr := scanJobEnabledOverlay(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		overlays = append(overlays, overlay)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate automation job overlays: %w", err)
	}

	return overlays, nil
}

// DeleteJobEnabledOverlay clears a persisted job enabled overlay if it exists.
func (g *GlobalDB) DeleteJobEnabledOverlay(ctx context.Context, jobID string) error {
	if err := g.checkReady(ctx, "delete automation job overlay"); err != nil {
		return err
	}

	trimmedID, err := requireAutomationID(jobID, "automation job overlay id")
	if err != nil {
		return err
	}

	if _, err := g.db.ExecContext(ctx, `DELETE FROM automation_job_overlays WHERE job_id = ?`, trimmedID); err != nil {
		return fmt.Errorf("store: delete automation job overlay %q: %w", trimmedID, err)
	}

	return nil
}

// SetTriggerEnabledOverlay upserts the runtime enabled override for a config-backed trigger.
func (g *GlobalDB) SetTriggerEnabledOverlay(
	ctx context.Context,
	overlay automation.TriggerEnabledOverlay,
) (automation.TriggerEnabledOverlay, error) {
	if err := g.checkReady(ctx, "set automation trigger overlay"); err != nil {
		return automation.TriggerEnabledOverlay{}, err
	}

	normalized, err := normalizeTriggerOverlay(overlay, g.now())
	if err != nil {
		return automation.TriggerEnabledOverlay{}, err
	}
	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO automation_trigger_overlays (trigger_id, enabled_override, updated_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(trigger_id) DO UPDATE SET
			enabled_override = excluded.enabled_override,
			updated_at = excluded.updated_at`,
		normalized.TriggerID,
		normalized.EnabledOverride,
		store.FormatTimestamp(normalized.UpdatedAt),
	); err != nil {
		return automation.TriggerEnabledOverlay{}, fmt.Errorf(
			"store: set automation trigger overlay %q: %w",
			normalized.TriggerID,
			err,
		)
	}

	return normalized, nil
}

// GetTriggerEnabledOverlay loads one persisted trigger enabled overlay by trigger id.
func (g *GlobalDB) GetTriggerEnabledOverlay(
	ctx context.Context,
	triggerID string,
) (automation.TriggerEnabledOverlay, error) {
	if err := g.checkReady(ctx, "get automation trigger overlay"); err != nil {
		return automation.TriggerEnabledOverlay{}, err
	}

	trimmedID, err := requireAutomationID(triggerID, "automation trigger overlay id")
	if err != nil {
		return automation.TriggerEnabledOverlay{}, err
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT trigger_id, enabled_override, updated_at
		 FROM automation_trigger_overlays
		 WHERE trigger_id = ?`,
		trimmedID,
	)
	overlay, err := scanTriggerEnabledOverlay(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return automation.TriggerEnabledOverlay{}, automation.ErrTriggerOverlayNotFound
		}
		return automation.TriggerEnabledOverlay{}, err
	}

	return overlay, nil
}

// ListTriggerEnabledOverlays returns all persisted trigger enabled overlays.
func (g *GlobalDB) ListTriggerEnabledOverlays(ctx context.Context) ([]automation.TriggerEnabledOverlay, error) {
	if err := g.checkReady(ctx, "list automation trigger overlays"); err != nil {
		return nil, err
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT trigger_id, enabled_override, updated_at
		 FROM automation_trigger_overlays
		 ORDER BY trigger_id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query automation trigger overlays: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	overlays := make([]automation.TriggerEnabledOverlay, 0)
	for rows.Next() {
		overlay, scanErr := scanTriggerEnabledOverlay(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		overlays = append(overlays, overlay)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate automation trigger overlays: %w", err)
	}

	return overlays, nil
}

// DeleteTriggerEnabledOverlay clears a persisted trigger enabled overlay if it exists.
func (g *GlobalDB) DeleteTriggerEnabledOverlay(ctx context.Context, triggerID string) error {
	if err := g.checkReady(ctx, "delete automation trigger overlay"); err != nil {
		return err
	}

	trimmedID, err := requireAutomationID(triggerID, "automation trigger overlay id")
	if err != nil {
		return err
	}

	if _, err := g.db.ExecContext(
		ctx,
		`DELETE FROM automation_trigger_overlays WHERE trigger_id = ?`,
		trimmedID,
	); err != nil {
		return fmt.Errorf("store: delete automation trigger overlay %q: %w", trimmedID, err)
	}

	return nil
}

func (g *GlobalDB) insertJob(ctx context.Context, exec sqlExecutor, job automation.Job) error {
	scheduleJSON, taskJSON, retryJSON, fireLimitJSON, err := encodeJobRecord(job)
	if err != nil {
		return err
	}

	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO automation_jobs (
			id, scope, name, agent_name, workspace_id, prompt, schedule, task,
			enabled, retry, fire_limit, source, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID,
		job.Scope,
		job.Name,
		job.AgentName,
		store.NullableString(job.WorkspaceID),
		job.Prompt,
		scheduleJSON,
		taskJSON,
		job.Enabled,
		retryJSON,
		fireLimitJSON,
		job.Source,
		store.FormatTimestamp(job.CreatedAt),
		store.FormatTimestamp(job.UpdatedAt),
	); err != nil {
		return mapAutomationJobConstraintError(err)
	}

	return nil
}

func (g *GlobalDB) insertTrigger(ctx context.Context, exec sqlExecutor, trigger automation.Trigger) error {
	filterJSON, retryJSON, fireLimitJSON, err := encodeTriggerRecord(trigger)
	if err != nil {
		return err
	}

	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO automation_triggers (
			id, scope, name, agent_name, workspace_id, prompt, event, filter,
			enabled, retry, fire_limit, source, webhook_id, endpoint_slug, webhook_secret_ref,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		trigger.ID,
		trigger.Scope,
		trigger.Name,
		trigger.AgentName,
		store.NullableString(trigger.WorkspaceID),
		trigger.Prompt,
		trigger.Event,
		filterJSON,
		trigger.Enabled,
		retryJSON,
		fireLimitJSON,
		trigger.Source,
		store.NullableString(trigger.WebhookID),
		store.NullableString(trigger.EndpointSlug),
		store.NullableString(trigger.WebhookSecretRef),
		store.FormatTimestamp(trigger.CreatedAt),
		store.FormatTimestamp(trigger.UpdatedAt),
	); err != nil {
		return mapAutomationTriggerConstraintError(err)
	}

	return nil
}

func (g *GlobalDB) getJobByQuery(ctx context.Context, query string, args ...any) (automation.Job, error) {
	row := g.db.QueryRowContext(ctx, query, args...)
	job, err := scanAutomationJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return automation.Job{}, automation.ErrJobNotFound
		}
		return automation.Job{}, err
	}
	return job, nil
}

func (g *GlobalDB) getTriggerByQuery(ctx context.Context, query string, args ...any) (automation.Trigger, error) {
	row := g.db.QueryRowContext(ctx, query, args...)
	trigger, err := scanAutomationTrigger(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return automation.Trigger{}, automation.ErrTriggerNotFound
		}
		return automation.Trigger{}, err
	}
	return trigger, nil
}

func (g *GlobalDB) normalizeJobForCreate(job automation.Job) (automation.Job, error) {
	normalized := normalizeAutomationJob(job)
	if normalized.Source == "" {
		normalized.Source = automation.JobSourceDynamic
	}
	if strings.TrimSpace(normalized.ID) == "" {
		normalized.ID = store.NewID("job")
	}
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.CreatedAt
	}
	if err := normalized.Validate("job"); err != nil {
		return automation.Job{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) normalizeJobForUpdate(job automation.Job) (automation.Job, error) {
	normalized := normalizeAutomationJob(job)
	if strings.TrimSpace(normalized.ID) == "" {
		return automation.Job{}, errors.New("store: automation job id is required")
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = g.now()
	}
	if err := normalized.Validate("job"); err != nil {
		return automation.Job{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) normalizeTriggerForCreate(trigger automation.Trigger) (automation.Trigger, error) {
	normalized := normalizeAutomationTrigger(trigger)
	if normalized.Source == "" {
		normalized.Source = automation.JobSourceDynamic
	}
	if strings.TrimSpace(normalized.ID) == "" {
		normalized.ID = store.NewID("trg")
	}
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.CreatedAt
	}
	if err := normalized.Validate("trigger"); err != nil {
		return automation.Trigger{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) normalizeTriggerForUpdate(trigger automation.Trigger) (automation.Trigger, error) {
	normalized := normalizeAutomationTrigger(trigger)
	if strings.TrimSpace(normalized.ID) == "" {
		return automation.Trigger{}, errors.New("store: automation trigger id is required")
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = g.now()
	}
	if err := normalized.Validate("trigger"); err != nil {
		return automation.Trigger{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) normalizeRunForCreate(run automation.Run) (automation.Run, error) {
	normalized := normalizeAutomationRun(run)
	if strings.TrimSpace(normalized.ID) == "" {
		normalized.ID = store.NewID("run")
	}
	if normalized.Attempt == 0 {
		normalized.Attempt = 1
	}
	if err := validateAutomationRunRecord(normalized); err != nil {
		return automation.Run{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) normalizeRunForUpdate(run automation.Run) (automation.Run, error) {
	normalized := normalizeAutomationRun(run)
	if strings.TrimSpace(normalized.ID) == "" {
		return automation.Run{}, errors.New("store: automation run id is required")
	}
	if normalized.Attempt <= 0 {
		return automation.Run{}, fmt.Errorf("store: automation run %q attempt must be positive", normalized.ID)
	}
	if err := validateAutomationRunRecord(normalized); err != nil {
		return automation.Run{}, err
	}
	return normalized, nil
}

func scanAutomationJob(scanner rowScanner) (automation.Job, error) {
	var (
		job         automation.Job
		scope       string
		workspaceID sql.NullString
		scheduleRaw sql.NullString
		taskRaw     sql.NullString
		retryRaw    string
		fireLimit   string
		source      string
		createdAt   string
		updatedAt   string
	)
	if err := scanner.Scan(
		&job.ID,
		&scope,
		&job.Name,
		&job.AgentName,
		&workspaceID,
		&job.Prompt,
		&scheduleRaw,
		&taskRaw,
		&job.Enabled,
		&retryRaw,
		&fireLimit,
		&source,
		&createdAt,
		&updatedAt,
	); err != nil {
		return automation.Job{}, fmt.Errorf("store: scan automation job: %w", err)
	}

	job.Scope = automation.Scope(strings.TrimSpace(scope))
	job.WorkspaceID = automationNullStringValue(workspaceID)
	job.Source = automation.JobSource(strings.TrimSpace(source))

	if err := decodeAutomationSchedule(scheduleRaw, &job.Schedule); err != nil {
		return automation.Job{}, err
	}
	if err := decodeAutomationTaskConfig(taskRaw, &job.Task); err != nil {
		return automation.Job{}, err
	}
	if err := decodeAutomationJSON(retryRaw, &job.Retry, "job.retry"); err != nil {
		return automation.Job{}, err
	}
	if err := decodeAutomationJSON(fireLimit, &job.FireLimit, "job.fire_limit"); err != nil {
		return automation.Job{}, err
	}

	parsedCreatedAt, err := store.ParseTimestamp(createdAt)
	if err != nil {
		return automation.Job{}, err
	}
	parsedUpdatedAt, err := store.ParseTimestamp(updatedAt)
	if err != nil {
		return automation.Job{}, err
	}
	job.CreatedAt = parsedCreatedAt
	job.UpdatedAt = parsedUpdatedAt

	return job, nil
}

func scanAutomationTrigger(scanner rowScanner) (automation.Trigger, error) {
	var (
		trigger          automation.Trigger
		scope            string
		workspaceID      sql.NullString
		filterRaw        sql.NullString
		retryRaw         string
		fireLimitRaw     string
		source           string
		webhookID        sql.NullString
		endpointSlug     sql.NullString
		webhookSecretRef sql.NullString
		createdAt        string
		updatedAt        string
	)
	if err := scanner.Scan(
		&trigger.ID,
		&scope,
		&trigger.Name,
		&trigger.AgentName,
		&workspaceID,
		&trigger.Prompt,
		&trigger.Event,
		&filterRaw,
		&trigger.Enabled,
		&retryRaw,
		&fireLimitRaw,
		&source,
		&webhookID,
		&endpointSlug,
		&webhookSecretRef,
		&createdAt,
		&updatedAt,
	); err != nil {
		return automation.Trigger{}, fmt.Errorf("store: scan automation trigger: %w", err)
	}

	trigger.Scope = automation.Scope(strings.TrimSpace(scope))
	trigger.WorkspaceID = automationNullStringValue(workspaceID)
	trigger.Source = automation.JobSource(strings.TrimSpace(source))
	trigger.WebhookID = automationNullStringValue(webhookID)
	trigger.EndpointSlug = automationNullStringValue(endpointSlug)
	trigger.WebhookSecretRef = automationNullStringValue(webhookSecretRef)

	if err := decodeAutomationFilter(filterRaw, &trigger.Filter); err != nil {
		return automation.Trigger{}, err
	}
	if err := decodeAutomationJSON(retryRaw, &trigger.Retry, "trigger.retry"); err != nil {
		return automation.Trigger{}, err
	}
	if err := decodeAutomationJSON(fireLimitRaw, &trigger.FireLimit, "trigger.fire_limit"); err != nil {
		return automation.Trigger{}, err
	}

	parsedCreatedAt, err := store.ParseTimestamp(createdAt)
	if err != nil {
		return automation.Trigger{}, err
	}
	parsedUpdatedAt, err := store.ParseTimestamp(updatedAt)
	if err != nil {
		return automation.Trigger{}, err
	}
	trigger.CreatedAt = parsedCreatedAt
	trigger.UpdatedAt = parsedUpdatedAt

	return trigger, nil
}

func scanAutomationRun(scanner rowScanner) (automation.Run, error) {
	var (
		run           automation.Run
		jobID         sql.NullString
		triggerID     sql.NullString
		sessionID     sql.NullString
		taskID        sql.NullString
		taskRunID     sql.NullString
		fireID        sql.NullString
		status        string
		scheduledAt   sql.NullString
		startedAt     sql.NullString
		endedAt       sql.NullString
		runErr        sql.NullString
		deliveryErr   sql.NullString
		deliveryErrAt sql.NullString
	)
	if err := scanner.Scan(
		&run.ID,
		&jobID,
		&triggerID,
		&sessionID,
		&taskID,
		&taskRunID,
		&fireID,
		&status,
		&run.Attempt,
		&scheduledAt,
		&startedAt,
		&endedAt,
		&runErr,
		&deliveryErr,
		&deliveryErrAt,
	); err != nil {
		return automation.Run{}, fmt.Errorf("store: scan automation run: %w", err)
	}

	run.JobID = automationNullStringValue(jobID)
	run.TriggerID = automationNullStringValue(triggerID)
	run.SessionID = automationNullStringValue(sessionID)
	run.TaskID = automationNullStringValue(taskID)
	run.TaskRunID = automationNullStringValue(taskRunID)
	run.FireID = automationNullStringValue(fireID)
	run.Status = automation.RunStatus(strings.TrimSpace(status))
	if scheduledAt.Valid {
		value, err := store.ParseTimestamp(scheduledAt.String)
		if err != nil {
			return automation.Run{}, err
		}
		run.ScheduledAt = &value
	}
	if startedAt.Valid {
		value, err := store.ParseTimestamp(startedAt.String)
		if err != nil {
			return automation.Run{}, err
		}
		run.StartedAt = &value
	}
	if endedAt.Valid {
		value, err := store.ParseTimestamp(endedAt.String)
		if err != nil {
			return automation.Run{}, err
		}
		run.EndedAt = &value
	}
	if runErr.Valid {
		run.Error = runErr.String
	}
	if deliveryErr.Valid {
		run.DeliveryError = deliveryErr.String
	}
	if deliveryErrAt.Valid {
		value, err := store.ParseTimestamp(deliveryErrAt.String)
		if err != nil {
			return automation.Run{}, err
		}
		run.DeliveryErrorAt = &value
	}

	return run, nil
}

func scanJobEnabledOverlay(scanner rowScanner) (automation.JobEnabledOverlay, error) {
	var (
		overlay      automation.JobEnabledOverlay
		updatedAtRaw string
	)
	if err := scanner.Scan(&overlay.JobID, &overlay.EnabledOverride, &updatedAtRaw); err != nil {
		return automation.JobEnabledOverlay{}, fmt.Errorf("store: scan automation job overlay: %w", err)
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return automation.JobEnabledOverlay{}, err
	}
	overlay.UpdatedAt = updatedAt
	return overlay, nil
}

func scanTriggerEnabledOverlay(scanner rowScanner) (automation.TriggerEnabledOverlay, error) {
	var (
		overlay      automation.TriggerEnabledOverlay
		updatedAtRaw string
	)
	if err := scanner.Scan(&overlay.TriggerID, &overlay.EnabledOverride, &updatedAtRaw); err != nil {
		return automation.TriggerEnabledOverlay{}, fmt.Errorf("store: scan automation trigger overlay: %w", err)
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return automation.TriggerEnabledOverlay{}, err
	}
	overlay.UpdatedAt = updatedAt
	return overlay, nil
}

func encodeJobRecord(job automation.Job) (string, any, string, string, error) {
	scheduleJSON, err := encodeAutomationJSON(job.Schedule, "job.schedule")
	if err != nil {
		return "", nil, "", "", err
	}
	taskJSON, err := encodeOptionalAutomationJSON(job.Task, job.Task == nil, "job.task")
	if err != nil {
		return "", nil, "", "", err
	}
	retryJSON, err := encodeAutomationJSON(job.Retry, "job.retry")
	if err != nil {
		return "", nil, "", "", err
	}
	fireLimitJSON, err := encodeAutomationJSON(job.FireLimit, "job.fire_limit")
	if err != nil {
		return "", nil, "", "", err
	}

	return scheduleJSON, taskJSON, retryJSON, fireLimitJSON, nil
}

func encodeTriggerRecord(trigger automation.Trigger) (any, string, string, error) {
	filterJSON, err := encodeOptionalAutomationJSON(trigger.Filter, len(trigger.Filter) == 0, "trigger.filter")
	if err != nil {
		return nil, "", "", err
	}
	retryJSON, err := encodeAutomationJSON(trigger.Retry, "trigger.retry")
	if err != nil {
		return nil, "", "", err
	}
	fireLimitJSON, err := encodeAutomationJSON(trigger.FireLimit, "trigger.fire_limit")
	if err != nil {
		return nil, "", "", err
	}

	return filterJSON, retryJSON, fireLimitJSON, nil
}

func validateAutomationJobListQuery(query automation.JobListQuery) error {
	if query.Limit < 0 {
		return fmt.Errorf("store: invalid automation job limit %d", query.Limit)
	}
	if query.Scope != "" {
		if err := query.Scope.Validate("job_query.scope"); err != nil {
			return err
		}
	}
	if query.Source != "" {
		if err := query.Source.Validate("job_query.source"); err != nil {
			return err
		}
	}
	if query.Scope == automation.AutomationScopeGlobal && strings.TrimSpace(query.WorkspaceID) != "" {
		return errors.New("store: automation job workspace_id filter must be empty when scope is global")
	}
	return nil
}

func validateAutomationTriggerListQuery(query automation.TriggerListQuery) error {
	if query.Limit < 0 {
		return fmt.Errorf("store: invalid automation trigger limit %d", query.Limit)
	}
	if query.Scope != "" {
		if err := query.Scope.Validate("trigger_query.scope"); err != nil {
			return err
		}
	}
	if query.Source != "" {
		if err := query.Source.Validate("trigger_query.source"); err != nil {
			return err
		}
	}
	if query.Scope == automation.AutomationScopeGlobal && strings.TrimSpace(query.WorkspaceID) != "" {
		return errors.New("store: automation trigger workspace_id filter must be empty when scope is global")
	}
	return nil
}

func validateAutomationRunQuery(query automation.RunQuery) error {
	if query.Limit < 0 {
		return fmt.Errorf("store: invalid automation run limit %d", query.Limit)
	}
	if query.Status != "" {
		if err := query.Status.Validate("run_query.status"); err != nil {
			return err
		}
	}
	if !query.Until.IsZero() && !query.Since.IsZero() && query.Until.Before(query.Since) {
		return errors.New("store: automation run query until must not be before since")
	}
	return nil
}

func validateAutomationRunRecord(run automation.Run) error {
	if err := run.Validate("run"); err != nil {
		return err
	}
	jobID := strings.TrimSpace(run.JobID)
	triggerID := strings.TrimSpace(run.TriggerID)
	taskID := strings.TrimSpace(run.TaskID)
	taskRunID := strings.TrimSpace(run.TaskRunID)
	switch {
	case jobID == "" && triggerID == "":
		return errors.New("store: automation run job_id or trigger_id is required")
	case jobID != "" && triggerID != "":
		return errors.New("store: automation run must reference either a job or a trigger, not both")
	case taskRunID != "" && taskID == "":
		return errors.New("store: automation run task_id is required when task_run_id is set")
	case run.Status == automation.RunDelegated && taskID == "":
		return errors.New("store: automation run task_id is required when status is delegated")
	case run.Status == automation.RunDelegated && taskRunID == "":
		return errors.New("store: automation run task_run_id is required when status is delegated")
	default:
		return nil
	}
}

func buildAutomationRunClauses(query automation.RunQuery) ([]string, []any) {
	where, args := store.BuildClauses(
		store.StringClause("job_id", query.JobID),
		store.StringClause("trigger_id", query.TriggerID),
		store.StringClause("status", string(query.Status)),
		store.NotStringClause("id", query.ExcludeID),
		store.TimeClause("started_at", ">=", query.Since),
		store.TimeClause("started_at", "<=", query.Until),
	)
	return where, args
}

func requireAutomationID(value string, label string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("store: %s is required", label)
	}
	return trimmed, nil
}

func requireRowsAffected(result sql.Result, notFound error, id string, label string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for %s %q: %w", label, id, err)
	}
	if affected == 0 {
		return fmt.Errorf("store: %s %q: %w", label, id, notFound)
	}
	return nil
}

func mapAutomationJobConstraintError(err error) error {
	if err == nil {
		return nil
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unique constraint failed: automation_jobs.name"),
		strings.Contains(message, "unique constraint failed: automation_jobs.workspace_id, automation_jobs.name"):
		return automation.ErrJobNameTaken
	case strings.Contains(message, "foreign key constraint failed"):
		return aghworkspace.ErrWorkspaceNotFound
	default:
		return err
	}
}

func mapAutomationTriggerConstraintError(err error) error {
	if err == nil {
		return nil
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unique constraint failed: automation_triggers.name"),
		strings.Contains(message, "unique constraint failed: automation_triggers.workspace_id, automation_triggers.name"):
		return automation.ErrTriggerNameTaken
	case strings.Contains(message, "unique constraint failed: automation_triggers.webhook_id"):
		return automation.ErrTriggerWebhookIDTaken
	case strings.Contains(message, "foreign key constraint failed"):
		return aghworkspace.ErrWorkspaceNotFound
	default:
		return err
	}
}

func mapAutomationRunConstraintError(err error) error {
	if err == nil {
		return nil
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unique constraint failed: automation_runs.id"),
		strings.Contains(message, "constraint failed: automation_runs.id"),
		strings.Contains(message, "unique constraint failed: automation_runs.fire_id"),
		strings.Contains(message, "constraint failed: automation_runs.fire_id"),
		strings.Contains(message, "uq_automation_runs_fire_id"):
		return automation.ErrRunAlreadyExists
	default:
		return err
	}
}

func normalizeAutomationJob(job automation.Job) automation.Job {
	job.ID = strings.TrimSpace(job.ID)
	job.Scope = automation.Scope(strings.TrimSpace(string(job.Scope)))
	job.Name = strings.TrimSpace(job.Name)
	job.AgentName = strings.TrimSpace(job.AgentName)
	job.WorkspaceID = strings.TrimSpace(job.WorkspaceID)
	job.Source = automation.JobSource(strings.TrimSpace(string(job.Source)))
	job.Retry.BaseDelay = strings.TrimSpace(job.Retry.BaseDelay)
	job.Retry.Strategy = automation.RetryStrategy(strings.TrimSpace(string(job.Retry.Strategy)))
	job.FireLimit.Window = strings.TrimSpace(job.FireLimit.Window)
	if job.Schedule != nil {
		schedule := *job.Schedule
		schedule.Mode = automation.ScheduleMode(strings.TrimSpace(string(schedule.Mode)))
		schedule.Expr = strings.TrimSpace(schedule.Expr)
		schedule.Interval = strings.TrimSpace(schedule.Interval)
		schedule.Time = strings.TrimSpace(schedule.Time)
		job.Schedule = &schedule
	}
	if job.Task != nil {
		taskConfig := *job.Task
		taskConfig.Title = strings.TrimSpace(taskConfig.Title)
		taskConfig.Description = strings.TrimSpace(taskConfig.Description)
		taskConfig.NetworkChannel = strings.TrimSpace(taskConfig.NetworkChannel)
		if taskConfig.Owner != nil {
			owner := *taskConfig.Owner
			owner.Kind = taskpkg.OwnerKind(strings.TrimSpace(string(owner.Kind)))
			owner.Ref = strings.TrimSpace(owner.Ref)
			taskConfig.Owner = &owner
		}
		job.Task = &taskConfig
	}
	return job
}

func normalizeAutomationTrigger(trigger automation.Trigger) automation.Trigger {
	trigger.ID = strings.TrimSpace(trigger.ID)
	trigger.Scope = automation.Scope(strings.TrimSpace(string(trigger.Scope)))
	trigger.Name = strings.TrimSpace(trigger.Name)
	trigger.AgentName = strings.TrimSpace(trigger.AgentName)
	trigger.WorkspaceID = strings.TrimSpace(trigger.WorkspaceID)
	trigger.Event = strings.TrimSpace(trigger.Event)
	trigger.Source = automation.JobSource(strings.TrimSpace(string(trigger.Source)))
	trigger.WebhookID = strings.TrimSpace(trigger.WebhookID)
	trigger.EndpointSlug = strings.TrimSpace(trigger.EndpointSlug)
	trigger.WebhookSecretRef = strings.TrimSpace(trigger.WebhookSecretRef)
	trigger.Retry.BaseDelay = strings.TrimSpace(trigger.Retry.BaseDelay)
	trigger.Retry.Strategy = automation.RetryStrategy(strings.TrimSpace(string(trigger.Retry.Strategy)))
	trigger.FireLimit.Window = strings.TrimSpace(trigger.FireLimit.Window)
	if len(trigger.Filter) > 0 {
		normalized := make(map[string]string, len(trigger.Filter))
		for rawKey, rawValue := range trigger.Filter {
			normalized[strings.TrimSpace(rawKey)] = strings.TrimSpace(rawValue)
		}
		trigger.Filter = normalized
	}
	return trigger
}

func normalizeAutomationRun(run automation.Run) automation.Run {
	run.ID = strings.TrimSpace(run.ID)
	run.JobID = strings.TrimSpace(run.JobID)
	run.TriggerID = strings.TrimSpace(run.TriggerID)
	run.SessionID = strings.TrimSpace(run.SessionID)
	run.TaskID = strings.TrimSpace(run.TaskID)
	run.TaskRunID = strings.TrimSpace(run.TaskRunID)
	run.FireID = strings.TrimSpace(run.FireID)
	run.Status = automation.RunStatus(strings.TrimSpace(string(run.Status)))
	run.Error = strings.TrimSpace(run.Error)
	run.DeliveryError = strings.TrimSpace(run.DeliveryError)
	return run
}

func normalizeJobOverlay(overlay automation.JobEnabledOverlay, now time.Time) (automation.JobEnabledOverlay, error) {
	overlay.JobID = strings.TrimSpace(overlay.JobID)
	if overlay.JobID == "" {
		return automation.JobEnabledOverlay{}, errors.New("store: automation job overlay id is required")
	}
	if overlay.UpdatedAt.IsZero() {
		overlay.UpdatedAt = now
	}
	return overlay, nil
}

func normalizeTriggerOverlay(
	overlay automation.TriggerEnabledOverlay,
	now time.Time,
) (automation.TriggerEnabledOverlay, error) {
	overlay.TriggerID = strings.TrimSpace(overlay.TriggerID)
	if overlay.TriggerID == "" {
		return automation.TriggerEnabledOverlay{}, errors.New("store: automation trigger overlay id is required")
	}
	if overlay.UpdatedAt.IsZero() {
		overlay.UpdatedAt = now
	}
	return overlay, nil
}

func encodeAutomationJSON(value any, label string) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("store: encode %s: %w", label, err)
	}
	return string(data), nil
}

func encodeOptionalAutomationJSON(value any, empty bool, label string) (any, error) {
	if empty {
		return nil, nil
	}
	encoded, err := encodeAutomationJSON(value, label)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

func decodeAutomationJSON[T any](raw string, target *T, label string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("store: %s is required", label)
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return fmt.Errorf("store: decode %s: %w", label, err)
	}
	return nil
}

func decodeAutomationSchedule(raw sql.NullString, target **automation.ScheduleSpec) error {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		*target = nil
		return nil
	}

	var schedule automation.ScheduleSpec
	if err := json.Unmarshal([]byte(raw.String), &schedule); err != nil {
		return fmt.Errorf("store: decode job.schedule: %w", err)
	}
	*target = &schedule
	return nil
}

func decodeAutomationTaskConfig(raw sql.NullString, target **automation.JobTaskConfig) error {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		*target = nil
		return nil
	}

	var taskConfig automation.JobTaskConfig
	if err := json.Unmarshal([]byte(raw.String), &taskConfig); err != nil {
		return fmt.Errorf("store: decode job.task: %w", err)
	}
	*target = &taskConfig
	return nil
}

func decodeAutomationFilter(raw sql.NullString, target *map[string]string) error {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		*target = nil
		return nil
	}
	var filter map[string]string
	if err := json.Unmarshal([]byte(raw.String), &filter); err != nil {
		return fmt.Errorf("store: decode trigger.filter: %w", err)
	}
	*target = filter
	return nil
}

func nullableAutomationTimestamp(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return store.FormatTimestamp(*value)
}

func automationNullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return strings.TrimSpace(value.String)
}
