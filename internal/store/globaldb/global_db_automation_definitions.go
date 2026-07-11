package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	automation "github.com/compozy/agh/internal/automation/model"
	"github.com/compozy/agh/internal/store"
)

// CreateJob stores a new automation job definition.
func (g *GlobalDB) CreateJob(ctx context.Context, job automation.Job) (created automation.Job, err error) {
	if err := g.checkReady(ctx, "create automation job"); err != nil {
		return automation.Job{}, err
	}
	normalized, err := g.normalizeJobForCreate(job)
	if err != nil {
		return automation.Job{}, err
	}
	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return automation.Job{}, fmt.Errorf("store: begin create automation job %q: %w", normalized.ID, err)
	}
	defer rollbackAutomationDefinitionTx(&err, tx, "create automation job")
	if err := g.insertJob(ctx, tx, normalized); err != nil {
		return automation.Job{}, fmt.Errorf("store: create automation job %q: %w", normalized.ID, err)
	}
	if err := upsertAutomationJobCatalog(ctx, tx, normalized); err != nil {
		return automation.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return automation.Job{}, fmt.Errorf("store: commit create automation job %q: %w", normalized.ID, err)
	}
	return normalized, nil
}

// UpdateJob replaces the mutable fields of a persisted automation job definition.
func (g *GlobalDB) UpdateJob(ctx context.Context, job automation.Job) (updated automation.Job, err error) {
	if err := g.checkReady(ctx, "update automation job"); err != nil {
		return automation.Job{}, err
	}
	normalized, err := g.normalizeJobForUpdate(job)
	if err != nil {
		return automation.Job{}, err
	}
	scheduleJSON, taskJSON, retryJSON, fireLimitJSON, loopTarget, err := encodeJobRecord(normalized)
	if err != nil {
		return automation.Job{}, err
	}
	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return automation.Job{}, fmt.Errorf("store: begin update automation job %q: %w", normalized.ID, err)
	}
	defer rollbackAutomationDefinitionTx(&err, tx, "update automation job")
	result, err := tx.ExecContext(
		ctx,
		`UPDATE automation_jobs
		 SET scope = ?, name = ?, agent_name = ?, workspace_id = ?, prompt = ?,
		     schedule = ?, task = ?, enabled = ?, retry = ?, fire_limit = ?,
		     source = ?, target_kind = ?, loop_workspace_id = ?, loop_name = ?,
		     loop_inputs = ?, loop_input_mapping = ?, updated_at = ?
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
		normalized.TargetKind,
		store.NullableString(loopTarget.workspaceID),
		store.NullableString(loopTarget.loopName),
		loopTarget.inputsJSON,
		loopTarget.inputMappingJSON,
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
	if err := upsertAutomationJobCatalog(ctx, tx, normalized); err != nil {
		return automation.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return automation.Job{}, fmt.Errorf("store: commit update automation job %q: %w", normalized.ID, err)
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
	return g.getJobByQuery(ctx, automationJobRichSelectSQL+` WHERE id = ?`, trimmedID)
}

// CreateTrigger stores a new automation trigger definition.
func (g *GlobalDB) CreateTrigger(
	ctx context.Context,
	trigger automation.Trigger,
) (created automation.Trigger, err error) {
	if err := g.checkReady(ctx, "create automation trigger"); err != nil {
		return automation.Trigger{}, err
	}
	normalized, err := g.normalizeTriggerForCreate(trigger)
	if err != nil {
		return automation.Trigger{}, err
	}
	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return automation.Trigger{}, fmt.Errorf(
			"store: begin create automation trigger %q: %w",
			normalized.ID,
			err,
		)
	}
	defer rollbackAutomationDefinitionTx(&err, tx, "create automation trigger")
	if err := g.insertTrigger(ctx, tx, normalized); err != nil {
		return automation.Trigger{}, fmt.Errorf("store: create automation trigger %q: %w", normalized.ID, err)
	}
	if err := upsertAutomationTriggerCatalog(ctx, tx, normalized); err != nil {
		return automation.Trigger{}, err
	}
	if err := tx.Commit(); err != nil {
		return automation.Trigger{}, fmt.Errorf(
			"store: commit create automation trigger %q: %w",
			normalized.ID,
			err,
		)
	}
	return normalized, nil
}

// UpdateTrigger replaces the mutable fields of a persisted automation trigger definition.
func (g *GlobalDB) UpdateTrigger(
	ctx context.Context,
	trigger automation.Trigger,
) (updated automation.Trigger, err error) {
	if err := g.checkReady(ctx, "update automation trigger"); err != nil {
		return automation.Trigger{}, err
	}
	normalized, err := g.normalizeTriggerForUpdate(trigger)
	if err != nil {
		return automation.Trigger{}, err
	}
	filterJSON, retryJSON, fireLimitJSON, loopTarget, err := encodeTriggerRecord(normalized)
	if err != nil {
		return automation.Trigger{}, err
	}
	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return automation.Trigger{}, fmt.Errorf(
			"store: begin update automation trigger %q: %w",
			normalized.ID,
			err,
		)
	}
	defer rollbackAutomationDefinitionTx(&err, tx, "update automation trigger")
	result, err := tx.ExecContext(
		ctx,
		`UPDATE automation_triggers
		 SET scope = ?, name = ?, agent_name = ?, workspace_id = ?, prompt = ?,
		     event = ?, filter = ?, enabled = ?, retry = ?, fire_limit = ?,
		     source = ?, webhook_id = ?, endpoint_slug = ?, webhook_secret_ref = ?,
		     target_kind = ?, loop_workspace_id = ?, loop_name = ?, loop_inputs = ?,
		     loop_input_mapping = ?, updated_at = ?
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
		normalized.TargetKind,
		store.NullableString(loopTarget.workspaceID),
		store.NullableString(loopTarget.loopName),
		loopTarget.inputsJSON,
		loopTarget.inputMappingJSON,
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
	if err := upsertAutomationTriggerCatalog(ctx, tx, normalized); err != nil {
		return automation.Trigger{}, err
	}
	if err := tx.Commit(); err != nil {
		return automation.Trigger{}, fmt.Errorf(
			"store: commit update automation trigger %q: %w",
			normalized.ID,
			err,
		)
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
	return g.getTriggerByQuery(ctx, automationTriggerRichSelectSQL+` WHERE id = ?`, trimmedID)
}

func rollbackAutomationDefinitionTx(target *error, tx *sql.Tx, action string) {
	if target == nil || tx == nil {
		return
	}
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		rollbackErr := fmt.Errorf("store: rollback %s: %w", action, err)
		*target = errors.Join(*target, rollbackErr)
	}
}
