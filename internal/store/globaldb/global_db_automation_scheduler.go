package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	automation "github.com/pedronauck/agh/internal/automation/model"
	"github.com/pedronauck/agh/internal/store"
)

// GetSchedulerState loads one durable automation scheduler cursor by job id.
func (g *GlobalDB) GetSchedulerState(ctx context.Context, jobID string) (automation.SchedulerState, error) {
	if err := g.checkReady(ctx, "get automation scheduler state"); err != nil {
		return automation.SchedulerState{}, err
	}

	trimmedID, err := requireAutomationID(jobID, "automation scheduler job id")
	if err != nil {
		return automation.SchedulerState{}, err
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT
			job_id, next_run_at, last_run_at, last_scheduled_at, last_fire_id,
			schedule_hash, catch_up_policy, misfire_grace_seconds,
			consecutive_resume_failures, last_misfire_at, misfire_count, updated_at
		 FROM automation_scheduler_state
		 WHERE job_id = ?`,
		trimmedID,
	)
	state, err := scanAutomationSchedulerState(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return automation.SchedulerState{}, automation.ErrSchedulerStateNotFound
		}
		return automation.SchedulerState{}, err
	}
	return state, nil
}

// ListSchedulerStates returns every durable automation scheduler cursor.
func (g *GlobalDB) ListSchedulerStates(ctx context.Context) ([]automation.SchedulerState, error) {
	if err := g.checkReady(ctx, "list automation scheduler states"); err != nil {
		return nil, err
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT
			job_id, next_run_at, last_run_at, last_scheduled_at, last_fire_id,
			schedule_hash, catch_up_policy, misfire_grace_seconds,
			consecutive_resume_failures, last_misfire_at, misfire_count, updated_at
		 FROM automation_scheduler_state
		 ORDER BY job_id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query automation scheduler states: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	states := make([]automation.SchedulerState, 0)
	for rows.Next() {
		state, scanErr := scanAutomationSchedulerState(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate automation scheduler states: %w", err)
	}
	return states, nil
}

// SaveSchedulerState upserts one durable automation scheduler cursor.
func (g *GlobalDB) SaveSchedulerState(
	ctx context.Context,
	state automation.SchedulerState,
) (automation.SchedulerState, error) {
	if err := g.checkReady(ctx, "save automation scheduler state"); err != nil {
		return automation.SchedulerState{}, err
	}

	normalized, err := g.normalizeSchedulerState(state)
	if err != nil {
		return automation.SchedulerState{}, err
	}
	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO automation_scheduler_state (
			job_id, next_run_at, last_run_at, last_scheduled_at, last_fire_id,
			schedule_hash, catch_up_policy, misfire_grace_seconds,
			consecutive_resume_failures, last_misfire_at, misfire_count, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id) DO UPDATE SET
			next_run_at = excluded.next_run_at,
			last_run_at = excluded.last_run_at,
			last_scheduled_at = excluded.last_scheduled_at,
			last_fire_id = excluded.last_fire_id,
			schedule_hash = excluded.schedule_hash,
			catch_up_policy = excluded.catch_up_policy,
			misfire_grace_seconds = excluded.misfire_grace_seconds,
			consecutive_resume_failures = excluded.consecutive_resume_failures,
			last_misfire_at = excluded.last_misfire_at,
			misfire_count = excluded.misfire_count,
			updated_at = excluded.updated_at`,
		normalized.JobID,
		nullableAutomationTimestamp(normalized.NextRunAt),
		nullableAutomationTimestamp(normalized.LastRunAt),
		nullableAutomationTimestamp(normalized.LastScheduledAt),
		normalized.LastFireID,
		normalized.ScheduleHash,
		normalized.CatchUpPolicy,
		normalized.MisfireGraceSeconds,
		normalized.ConsecutiveResumeFailures,
		nullableAutomationTimestamp(normalized.LastMisfireAt),
		normalized.MisfireCount,
		store.FormatTimestamp(normalized.UpdatedAt),
	); err != nil {
		return automation.SchedulerState{}, fmt.Errorf(
			"store: save automation scheduler state %q: %w",
			normalized.JobID,
			err,
		)
	}
	return g.GetSchedulerState(ctx, normalized.JobID)
}

// DeleteSchedulerState removes a durable scheduler cursor if it exists.
func (g *GlobalDB) DeleteSchedulerState(ctx context.Context, jobID string) error {
	if err := g.checkReady(ctx, "delete automation scheduler state"); err != nil {
		return err
	}

	trimmedID, err := requireAutomationID(jobID, "automation scheduler job id")
	if err != nil {
		return err
	}
	if _, err := g.db.ExecContext(
		ctx,
		`DELETE FROM automation_scheduler_state WHERE job_id = ?`,
		trimmedID,
	); err != nil {
		return fmt.Errorf("store: delete automation scheduler state %q: %w", trimmedID, err)
	}
	return nil
}

// ClaimScheduledRun advances one durable cursor and creates a run reservation
// before scheduler dispatch begins.
func (g *GlobalDB) ClaimScheduledRun(
	ctx context.Context,
	claim automation.SchedulerClaim,
) (result automation.SchedulerClaimResult, err error) {
	if err := g.checkReady(ctx, "claim automation scheduled run"); err != nil {
		return automation.SchedulerClaimResult{}, err
	}

	normalized, err := g.normalizeSchedulerClaim(claim)
	if err != nil {
		return automation.SchedulerClaimResult{}, err
	}

	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return automation.SchedulerClaimResult{}, fmt.Errorf("store: begin automation scheduled run claim: %w", err)
	}
	defer func() {
		joinCleanupError(&err, rollbackTx(tx, "automation scheduled run claim"))
	}()

	existing, err := getSchedulerStateTx(ctx, tx, normalized.JobID)
	if err != nil && !errors.Is(err, automation.ErrSchedulerStateNotFound) {
		return automation.SchedulerClaimResult{}, err
	}
	if strings.TrimSpace(existing.LastFireID) == normalized.FireID {
		return automation.SchedulerClaimResult{}, fmt.Errorf(
			"store: automation scheduled fire %q: %w",
			normalized.FireID,
			automation.ErrScheduledFireAlreadyClaimed,
		)
	}

	nextState := automation.SchedulerState{
		JobID:                     normalized.JobID,
		NextRunAt:                 cloneTimePointer(normalized.NextRunAt),
		LastRunAt:                 automationTimePointer(normalized.ClaimedAt),
		LastScheduledAt:           automationTimePointer(normalized.ScheduledAt),
		LastFireID:                normalized.FireID,
		ScheduleHash:              normalized.ScheduleHash,
		CatchUpPolicy:             schedulerCatchUpPolicyOrDefault(existing.CatchUpPolicy),
		MisfireGraceSeconds:       existing.MisfireGraceSeconds,
		ConsecutiveResumeFailures: 0,
		LastMisfireAt:             cloneTimePointer(existing.LastMisfireAt),
		MisfireCount:              existing.MisfireCount,
		UpdatedAt:                 normalized.ClaimedAt,
	}
	if nextState.MisfireGraceSeconds < 0 {
		nextState.MisfireGraceSeconds = 0
	}
	if err := upsertSchedulerStateTx(ctx, tx, nextState); err != nil {
		return automation.SchedulerClaimResult{}, err
	}

	run := automation.Run{
		ID:          normalized.RunID,
		JobID:       normalized.JobID,
		FireID:      normalized.FireID,
		Status:      automation.RunScheduled,
		Attempt:     1,
		ScheduledAt: automationTimePointer(normalized.ScheduledAt),
		StartedAt:   automationTimePointer(normalized.ClaimedAt),
	}
	if err := insertAutomationRunTx(ctx, tx, run); err != nil {
		return automation.SchedulerClaimResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return automation.SchedulerClaimResult{}, fmt.Errorf("store: commit automation scheduled run claim: %w", err)
	}

	result.State = nextState
	result.Run = run
	return result, nil
}

// RecordRunDeliveryError stores delivery diagnostics separately from normal
// execution errors on an existing automation run.
func (g *GlobalDB) RecordRunDeliveryError(ctx context.Context, runID string, runErr error) (automation.Run, error) {
	if err := g.checkReady(ctx, "record automation run delivery error"); err != nil {
		return automation.Run{}, err
	}

	trimmedID, err := requireAutomationID(runID, "automation run id")
	if err != nil {
		return automation.Run{}, err
	}
	if runErr == nil {
		return g.GetRun(ctx, trimmedID)
	}

	result, err := g.db.ExecContext(
		ctx,
		`UPDATE automation_runs
		 SET delivery_error = ?, delivery_error_at = ?
		 WHERE id = ?`,
		strings.TrimSpace(runErr.Error()),
		store.FormatTimestamp(g.now().UTC()),
		trimmedID,
	)
	if err != nil {
		return automation.Run{}, fmt.Errorf("store: record automation run delivery error %q: %w", trimmedID, err)
	}
	if err := requireRowsAffected(result, automation.ErrRunNotFound, trimmedID, "automation run"); err != nil {
		return automation.Run{}, err
	}
	return g.GetRun(ctx, trimmedID)
}

func (g *GlobalDB) normalizeSchedulerState(state automation.SchedulerState) (automation.SchedulerState, error) {
	state.JobID = strings.TrimSpace(state.JobID)
	state.LastFireID = strings.TrimSpace(state.LastFireID)
	state.ScheduleHash = strings.TrimSpace(state.ScheduleHash)
	state.CatchUpPolicy = schedulerCatchUpPolicyOrDefault(state.CatchUpPolicy)
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = g.now().UTC()
	}
	if err := state.Validate("scheduler_state"); err != nil {
		return automation.SchedulerState{}, err
	}
	return state, nil
}

func (g *GlobalDB) normalizeSchedulerClaim(claim automation.SchedulerClaim) (automation.SchedulerClaim, error) {
	claim.JobID = strings.TrimSpace(claim.JobID)
	claim.RunID = strings.TrimSpace(claim.RunID)
	claim.FireID = strings.TrimSpace(claim.FireID)
	claim.ScheduleHash = strings.TrimSpace(claim.ScheduleHash)
	if claim.ClaimedAt.IsZero() {
		claim.ClaimedAt = g.now().UTC()
	}
	if err := claim.Validate("scheduler_claim"); err != nil {
		return automation.SchedulerClaim{}, err
	}
	return claim, nil
}

func scanAutomationSchedulerState(scanner rowScanner) (automation.SchedulerState, error) {
	var (
		state           automation.SchedulerState
		nextRunAt       sql.NullString
		lastRunAt       sql.NullString
		lastScheduledAt sql.NullString
		lastFireID      string
		scheduleHash    string
		catchUpPolicy   string
		lastMisfireAt   sql.NullString
		updatedAtRaw    string
	)
	if err := scanner.Scan(
		&state.JobID,
		&nextRunAt,
		&lastRunAt,
		&lastScheduledAt,
		&lastFireID,
		&scheduleHash,
		&catchUpPolicy,
		&state.MisfireGraceSeconds,
		&state.ConsecutiveResumeFailures,
		&lastMisfireAt,
		&state.MisfireCount,
		&updatedAtRaw,
	); err != nil {
		return automation.SchedulerState{}, fmt.Errorf("store: scan automation scheduler state: %w", err)
	}
	var err error
	if state.NextRunAt, err = parseNullableAutomationTime(nextRunAt); err != nil {
		return automation.SchedulerState{}, err
	}
	if state.LastRunAt, err = parseNullableAutomationTime(lastRunAt); err != nil {
		return automation.SchedulerState{}, err
	}
	if state.LastScheduledAt, err = parseNullableAutomationTime(lastScheduledAt); err != nil {
		return automation.SchedulerState{}, err
	}
	if state.LastMisfireAt, err = parseNullableAutomationTime(lastMisfireAt); err != nil {
		return automation.SchedulerState{}, err
	}
	state.LastFireID = strings.TrimSpace(lastFireID)
	state.ScheduleHash = strings.TrimSpace(scheduleHash)
	state.CatchUpPolicy = automation.SchedulerCatchUpPolicy(strings.TrimSpace(catchUpPolicy))
	state.UpdatedAt, err = store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return automation.SchedulerState{}, err
	}
	if err := state.Validate("scheduler_state"); err != nil {
		return automation.SchedulerState{}, err
	}
	return state, nil
}

func getSchedulerStateTx(
	ctx context.Context,
	tx *sql.Tx,
	jobID string,
) (automation.SchedulerState, error) {
	row := tx.QueryRowContext(
		ctx,
		`SELECT
			job_id, next_run_at, last_run_at, last_scheduled_at, last_fire_id,
			schedule_hash, catch_up_policy, misfire_grace_seconds,
			consecutive_resume_failures, last_misfire_at, misfire_count, updated_at
		 FROM automation_scheduler_state
		 WHERE job_id = ?`,
		jobID,
	)
	state, err := scanAutomationSchedulerState(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return automation.SchedulerState{}, automation.ErrSchedulerStateNotFound
		}
		return automation.SchedulerState{}, err
	}
	return state, nil
}

func upsertSchedulerStateTx(ctx context.Context, tx *sql.Tx, state automation.SchedulerState) error {
	if err := state.Validate("scheduler_state"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO automation_scheduler_state (
			job_id, next_run_at, last_run_at, last_scheduled_at, last_fire_id,
			schedule_hash, catch_up_policy, misfire_grace_seconds,
			consecutive_resume_failures, last_misfire_at, misfire_count, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id) DO UPDATE SET
			next_run_at = excluded.next_run_at,
			last_run_at = excluded.last_run_at,
			last_scheduled_at = excluded.last_scheduled_at,
			last_fire_id = excluded.last_fire_id,
			schedule_hash = excluded.schedule_hash,
			catch_up_policy = excluded.catch_up_policy,
			misfire_grace_seconds = excluded.misfire_grace_seconds,
			consecutive_resume_failures = excluded.consecutive_resume_failures,
			last_misfire_at = excluded.last_misfire_at,
			misfire_count = excluded.misfire_count,
			updated_at = excluded.updated_at`,
		state.JobID,
		nullableAutomationTimestamp(state.NextRunAt),
		nullableAutomationTimestamp(state.LastRunAt),
		nullableAutomationTimestamp(state.LastScheduledAt),
		state.LastFireID,
		state.ScheduleHash,
		state.CatchUpPolicy,
		state.MisfireGraceSeconds,
		state.ConsecutiveResumeFailures,
		nullableAutomationTimestamp(state.LastMisfireAt),
		state.MisfireCount,
		store.FormatTimestamp(state.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: upsert automation scheduler state %q: %w", state.JobID, err)
	}
	return nil
}

func insertAutomationRunTx(ctx context.Context, tx *sql.Tx, run automation.Run) error {
	if err := validateAutomationRunRecord(run); err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO automation_runs (
			id, job_id, trigger_id, session_id, task_id, task_run_id, fire_id,
			status, attempt, scheduled_at, started_at, ended_at, error,
			delivery_error, delivery_error_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID,
		store.NullableString(run.JobID),
		store.NullableString(run.TriggerID),
		store.NullableString(run.SessionID),
		store.NullableString(run.TaskID),
		store.NullableString(run.TaskRunID),
		store.NullableString(run.FireID),
		run.Status,
		run.Attempt,
		nullableAutomationTimestamp(run.ScheduledAt),
		nullableAutomationTimestamp(run.StartedAt),
		nullableAutomationTimestamp(run.EndedAt),
		store.NullableString(run.Error),
		store.NullableString(run.DeliveryError),
		nullableAutomationTimestamp(run.DeliveryErrorAt),
	); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique constraint failed: automation_runs.fire_id") {
			return fmt.Errorf(
				"store: automation scheduled fire %q: %w",
				run.FireID,
				automation.ErrScheduledFireAlreadyClaimed,
			)
		}
		return fmt.Errorf("store: create automation run %q: %w", run.ID, err)
	}
	return nil
}

func schedulerCatchUpPolicyOrDefault(policy automation.SchedulerCatchUpPolicy) automation.SchedulerCatchUpPolicy {
	if policy == "" {
		return automation.SchedulerCatchUpPolicySkipMissed
	}
	return policy
}

func parseNullableAutomationTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil, nil
	}
	parsed, err := store.ParseTimestamp(value.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	clone := *value
	return &clone
}

func automationTimePointer(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}
