package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	automation "github.com/compozy/agh/internal/automation/model"
	"github.com/compozy/agh/internal/store"
	aghworkspace "github.com/compozy/agh/internal/workspace"
)

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
				webhook_secret_ref, target_kind, loop_workspace_id, loop_name,
				loop_inputs, loop_input_mapping, created_at, updated_at
			 FROM automation_triggers
			 WHERE webhook_id = ?`,
		trimmedWebhookID,
	)
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
	metadataJSON, err := encodeAutomationRunMetadata(normalized.Metadata)
	if err != nil {
		return automation.Run{}, err
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO automation_runs (
			id, job_id, trigger_id, session_id, task_id, task_run_id, fire_id,
			status, attempt, scheduled_at, started_at, ended_at, error,
			delivery_error, delivery_error_at, loop_run_id, metadata_json
		)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
		store.NullableString(normalized.LoopRunID),
		metadataJSON,
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
	metadataJSON, err := encodeAutomationRunMetadata(normalized.Metadata)
	if err != nil {
		return automation.Run{}, err
	}

	result, err := g.db.ExecContext(ctx, automationRunUpdateSQL, automationRunUpdateArgs(normalized, metadataJSON)...)
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
			delivery_error, delivery_error_at, loop_run_id, metadata_json
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
func (g *GlobalDB) ListRuns(
	ctx context.Context,
	query automation.RunQuery,
) (runs []automation.Run, err error) {
	if err := g.checkReady(ctx, "list automation runs"); err != nil {
		return nil, err
	}
	if err := validateAutomationRunQuery(query); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT
		id, job_id, trigger_id, session_id, task_id, task_run_id, fire_id,
		status, attempt, scheduled_at, started_at, ended_at, error,
		delivery_error, delivery_error_at, loop_run_id, metadata_json
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
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("store: close automation run rows: %w", closeErr))
		}
	}()

	runs = make([]automation.Run, 0)
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
func (g *GlobalDB) ListJobEnabledOverlays(
	ctx context.Context,
) (overlays []automation.JobEnabledOverlay, err error) {
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
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("store: close automation job overlay rows: %w", closeErr))
		}
	}()

	overlays = make([]automation.JobEnabledOverlay, 0)
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
func (g *GlobalDB) ListTriggerEnabledOverlays(
	ctx context.Context,
) (overlays []automation.TriggerEnabledOverlay, err error) {
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
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("store: close automation trigger overlay rows: %w", closeErr))
		}
	}()

	overlays = make([]automation.TriggerEnabledOverlay, 0)
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
	scheduleJSON, taskJSON, retryJSON, fireLimitJSON, loopTarget, err := encodeJobRecord(job)
	if err != nil {
		return err
	}

	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO automation_jobs (
			id, scope, name, agent_name, workspace_id, prompt, schedule, task,
			enabled, retry, fire_limit, source, target_kind, loop_workspace_id,
			loop_name, loop_inputs, loop_input_mapping, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
		job.TargetKind,
		store.NullableString(loopTarget.workspaceID),
		store.NullableString(loopTarget.loopName),
		loopTarget.inputsJSON,
		loopTarget.inputMappingJSON,
		store.FormatTimestamp(job.CreatedAt),
		store.FormatTimestamp(job.UpdatedAt),
	); err != nil {
		return mapAutomationJobConstraintError(err)
	}

	return nil
}

func (g *GlobalDB) insertTrigger(ctx context.Context, exec sqlExecutor, trigger automation.Trigger) error {
	filterJSON, retryJSON, fireLimitJSON, loopTarget, err := encodeTriggerRecord(trigger)
	if err != nil {
		return err
	}

	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO automation_triggers (
			id, scope, name, agent_name, workspace_id, prompt, event, filter,
			enabled, retry, fire_limit, source, webhook_id, endpoint_slug, webhook_secret_ref,
			target_kind, loop_workspace_id, loop_name, loop_inputs, loop_input_mapping,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
		trigger.TargetKind,
		store.NullableString(loopTarget.workspaceID),
		store.NullableString(loopTarget.loopName),
		loopTarget.inputsJSON,
		loopTarget.inputMappingJSON,
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
		targetKind  string
		loopTarget  automationLoopTargetRecord
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
		&targetKind,
		&loopTarget.workspaceID,
		&loopTarget.loopName,
		&loopTarget.inputsRaw,
		&loopTarget.inputMappingRaw,
		&createdAt,
		&updatedAt,
	); err != nil {
		return automation.Job{}, fmt.Errorf("store: scan automation job: %w", err)
	}

	job.Scope = automation.Scope(strings.TrimSpace(scope))
	job.WorkspaceID = automationNullStringValue(workspaceID)
	job.Source = automation.JobSource(strings.TrimSpace(source))
	job.TargetKind = automation.TargetKind(strings.TrimSpace(targetKind))

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
	if err := decodeAutomationLoopTarget(loopTarget, job.TargetKind, &job.LoopTarget, "job.loop_target"); err != nil {
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
		targetKind       string
		loopTarget       automationLoopTargetRecord
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
		&targetKind,
		&loopTarget.workspaceID,
		&loopTarget.loopName,
		&loopTarget.inputsRaw,
		&loopTarget.inputMappingRaw,
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
	trigger.TargetKind = automation.TargetKind(strings.TrimSpace(targetKind))

	if err := decodeAutomationFilter(filterRaw, &trigger.Filter); err != nil {
		return automation.Trigger{}, err
	}
	if err := decodeAutomationJSON(retryRaw, &trigger.Retry, "trigger.retry"); err != nil {
		return automation.Trigger{}, err
	}
	if err := decodeAutomationJSON(fireLimitRaw, &trigger.FireLimit, "trigger.fire_limit"); err != nil {
		return automation.Trigger{}, err
	}
	if err := decodeAutomationLoopTarget(
		loopTarget,
		trigger.TargetKind,
		&trigger.LoopTarget,
		"trigger.loop_target",
	); err != nil {
		return automation.Trigger{}, err
	}

	if err := assignAutomationTriggerTimestamps(&trigger, createdAt, updatedAt); err != nil {
		return automation.Trigger{}, err
	}

	return trigger, nil
}

func assignAutomationTriggerTimestamps(trigger *automation.Trigger, createdAt string, updatedAt string) error {
	parsedCreatedAt, err := store.ParseTimestamp(createdAt)
	if err != nil {
		return err
	}
	parsedUpdatedAt, err := store.ParseTimestamp(updatedAt)
	if err != nil {
		return err
	}
	trigger.CreatedAt = parsedCreatedAt
	trigger.UpdatedAt = parsedUpdatedAt
	return nil
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
		loopRunID     sql.NullString
		metadataRaw   string
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
		&loopRunID,
		&metadataRaw,
	); err != nil {
		return automation.Run{}, fmt.Errorf("store: scan automation run: %w", err)
	}

	run.JobID = automationNullStringValue(jobID)
	run.TriggerID = automationNullStringValue(triggerID)
	run.SessionID = automationNullStringValue(sessionID)
	run.TaskID = automationNullStringValue(taskID)
	run.TaskRunID = automationNullStringValue(taskRunID)
	run.FireID = automationNullStringValue(fireID)
	run.LoopRunID = automationNullStringValue(loopRunID)
	run.Status = automation.RunStatus(strings.TrimSpace(status))
	if err := assignAutomationRunTimestamps(&run, scheduledAt, startedAt, endedAt, deliveryErrAt); err != nil {
		return automation.Run{}, err
	}
	assignAutomationRunErrors(&run, runErr, deliveryErr)
	if err := decodeAutomationRunMetadata(metadataRaw, &run.Metadata); err != nil {
		return automation.Run{}, err
	}

	return run, nil
}

func assignAutomationRunTimestamps(
	run *automation.Run,
	scheduledAt sql.NullString,
	startedAt sql.NullString,
	endedAt sql.NullString,
	deliveryErrAt sql.NullString,
) error {
	if scheduledAt.Valid {
		value, err := store.ParseTimestamp(scheduledAt.String)
		if err != nil {
			return err
		}
		run.ScheduledAt = &value
	}
	if startedAt.Valid {
		value, err := store.ParseTimestamp(startedAt.String)
		if err != nil {
			return err
		}
		run.StartedAt = &value
	}
	if endedAt.Valid {
		value, err := store.ParseTimestamp(endedAt.String)
		if err != nil {
			return err
		}
		run.EndedAt = &value
	}
	if deliveryErrAt.Valid {
		value, err := store.ParseTimestamp(deliveryErrAt.String)
		if err != nil {
			return err
		}
		run.DeliveryErrorAt = &value
	}
	return nil
}

func assignAutomationRunErrors(run *automation.Run, runErr sql.NullString, deliveryErr sql.NullString) {
	if runErr.Valid {
		run.Error = runErr.String
	}
	if deliveryErr.Valid {
		run.DeliveryError = deliveryErr.String
	}
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

func encodeJobRecord(job automation.Job) (string, any, string, string, automationLoopTargetEncoded, error) {
	scheduleJSON, err := encodeAutomationJSON(job.Schedule, "job.schedule")
	if err != nil {
		return "", nil, "", "", automationLoopTargetEncoded{}, err
	}
	taskJSON, err := encodeOptionalAutomationJSON(job.Task, job.Task == nil, "job.task")
	if err != nil {
		return "", nil, "", "", automationLoopTargetEncoded{}, err
	}
	retryJSON, err := encodeAutomationJSON(job.Retry, "job.retry")
	if err != nil {
		return "", nil, "", "", automationLoopTargetEncoded{}, err
	}
	fireLimitJSON, err := encodeAutomationJSON(job.FireLimit, "job.fire_limit")
	if err != nil {
		return "", nil, "", "", automationLoopTargetEncoded{}, err
	}
	loopTarget, err := encodeAutomationLoopTarget(job.LoopTarget, "job.loop_target")
	if err != nil {
		return "", nil, "", "", automationLoopTargetEncoded{}, err
	}

	return scheduleJSON, taskJSON, retryJSON, fireLimitJSON, loopTarget, nil
}

func encodeTriggerRecord(trigger automation.Trigger) (any, string, string, automationLoopTargetEncoded, error) {
	filterJSON, err := encodeOptionalAutomationJSON(trigger.Filter, len(trigger.Filter) == 0, "trigger.filter")
	if err != nil {
		return nil, "", "", automationLoopTargetEncoded{}, err
	}
	retryJSON, err := encodeAutomationJSON(trigger.Retry, "trigger.retry")
	if err != nil {
		return nil, "", "", automationLoopTargetEncoded{}, err
	}
	fireLimitJSON, err := encodeAutomationJSON(trigger.FireLimit, "trigger.fire_limit")
	if err != nil {
		return nil, "", "", automationLoopTargetEncoded{}, err
	}
	loopTarget, err := encodeAutomationLoopTarget(trigger.LoopTarget, "trigger.loop_target")
	if err != nil {
		return nil, "", "", automationLoopTargetEncoded{}, err
	}

	return filterJSON, retryJSON, fireLimitJSON, loopTarget, nil
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
	loopRunID := strings.TrimSpace(run.LoopRunID)
	taskDelegated := taskID != "" && taskRunID != ""
	loopDelegated := loopRunID != ""
	switch {
	case jobID == "" && triggerID == "":
		return errors.New("store: automation run job_id or trigger_id is required")
	case jobID != "" && triggerID != "":
		return errors.New("store: automation run must reference either a job or a trigger, not both")
	case taskRunID != "" && taskID == "":
		return errors.New("store: automation run task_id is required when task_run_id is set")
	case taskID != "" && taskRunID == "":
		return errors.New("store: automation run task_run_id is required when task_id is set")
	case loopRunID != "" && run.Status != automation.RunDelegated:
		return errors.New("store: automation run loop_run_id requires delegated status")
	case run.Status == automation.RunDelegated && taskDelegated == loopDelegated:
		return errors.New("store: delegated automation run requires exactly one task run or loop run target")
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

func encodeAutomationJSON(value any, label string) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("store: encode %s: %w", label, err)
	}
	return string(data), nil
}

func encodeAutomationRunMetadata(metadata map[string]any) (string, error) {
	if metadata == nil {
		metadata = map[string]any{}
	}
	return encodeAutomationJSON(metadata, "run.metadata")
}

func decodeAutomationRunMetadata(raw string, target *map[string]any) error {
	if strings.TrimSpace(raw) == "" {
		*target = map[string]any{}
		return nil
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return fmt.Errorf("store: decode run.metadata: %w", err)
	}
	if *target == nil {
		*target = map[string]any{}
	}
	return nil
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
