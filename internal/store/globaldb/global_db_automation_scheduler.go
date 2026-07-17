package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	automation "github.com/compozy/agh/internal/automation/model"
	"github.com/compozy/agh/internal/network/participation"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb/sqlcgen"
)

// GetSchedulerState loads one durable automation scheduler cursor by job id.
func (g *AutomationRepo) GetSchedulerState(ctx context.Context, jobID string) (automation.SchedulerState, error) {
	if err := g.checkReady(ctx, "get automation scheduler state"); err != nil {
		return automation.SchedulerState{}, err
	}

	trimmedID, err := requireAutomationID(jobID, "automation scheduler job id")
	if err != nil {
		return automation.SchedulerState{}, err
	}

	row, err := g.queries.GetAutomationSchedulerState(ctx, trimmedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return automation.SchedulerState{}, automation.ErrSchedulerStateNotFound
		}
		return automation.SchedulerState{}, err
	}
	return automationSchedulerFromGenerated(row)
}

// ListSchedulerStates returns every durable automation scheduler cursor.
func (g *AutomationRepo) ListSchedulerStates(ctx context.Context) ([]automation.SchedulerState, error) {
	if err := g.checkReady(ctx, "list automation scheduler states"); err != nil {
		return nil, err
	}

	rows, err := g.queries.ListAutomationSchedulerStates(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: query automation scheduler states: %w", err)
	}
	states := make([]automation.SchedulerState, 0, len(rows))
	for _, row := range rows {
		state, scanErr := automationSchedulerFromGenerated(row)
		if scanErr != nil {
			return nil, scanErr
		}
		states = append(states, state)
	}
	return states, nil
}

// SaveSchedulerState upserts one durable automation scheduler cursor.
func (g *AutomationRepo) SaveSchedulerState(
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
	if err := g.queries.UpsertAutomationSchedulerState(ctx, automationSchedulerParams(normalized)); err != nil {
		return automation.SchedulerState{}, fmt.Errorf(
			"store: save automation scheduler state %q: %w",
			normalized.JobID,
			err,
		)
	}
	return g.GetSchedulerState(ctx, normalized.JobID)
}

// DeleteSchedulerState removes a durable scheduler cursor if it exists.
func (g *AutomationRepo) DeleteSchedulerState(ctx context.Context, jobID string) error {
	if err := g.checkReady(ctx, "delete automation scheduler state"); err != nil {
		return err
	}

	trimmedID, err := requireAutomationID(jobID, "automation scheduler job id")
	if err != nil {
		return err
	}
	if err := g.queries.DeleteAutomationSchedulerState(ctx, trimmedID); err != nil {
		return fmt.Errorf("store: delete automation scheduler state %q: %w", trimmedID, err)
	}
	return nil
}

// ClaimScheduledRun advances one durable cursor and creates a run reservation
// before scheduler dispatch begins.
func (g *AutomationRepo) ClaimScheduledRun(
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
		UpdatedAt:                 normalized.ClaimedAt,
	}
	if normalized.CatchUp {
		nextState.LastMisfireAt = cloneTimePointer(existing.LastMisfireAt)
		nextState.MisfireCount = existing.MisfireCount
	}
	if nextState.MisfireGraceSeconds < 0 {
		nextState.MisfireGraceSeconds = 0
	}
	if err := upsertSchedulerStateTx(ctx, tx, nextState); err != nil {
		return automation.SchedulerClaimResult{}, err
	}

	run := automation.Run{
		ID:                   normalized.RunID,
		JobID:                normalized.JobID,
		FireID:               normalized.FireID,
		Status:               automation.RunScheduled,
		Attempt:              1,
		ScheduledAt:          automationTimePointer(normalized.ScheduledAt),
		StartedAt:            automationTimePointer(normalized.ClaimedAt),
		NetworkParticipation: participation.CloneRequest(normalized.NetworkParticipation),
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
func (g *AutomationRepo) RecordRunDeliveryError(
	ctx context.Context,
	runID string,
	runErr error,
) (automation.Run, error) {
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

	affected, err := g.queries.RecordAutomationRunDeliveryError(
		ctx,
		sqlcgen.RecordAutomationRunDeliveryErrorParams{
			DeliveryError:   nullableAutomationString(runErr.Error()),
			DeliveryErrorAt: nullableAutomationString(store.FormatTimestamp(g.now().UTC())),
			ID:              trimmedID,
		},
	)
	if err != nil {
		return automation.Run{}, fmt.Errorf("store: record automation run delivery error %q: %w", trimmedID, err)
	}
	if err := requireAutomationAffected(affected, automation.ErrRunNotFound, trimmedID, "automation run"); err != nil {
		return automation.Run{}, err
	}
	return g.GetRun(ctx, trimmedID)
}

func (g *AutomationRepo) normalizeSchedulerState(state automation.SchedulerState) (automation.SchedulerState, error) {
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

func (g *AutomationRepo) normalizeSchedulerClaim(claim automation.SchedulerClaim) (automation.SchedulerClaim, error) {
	claim.JobID = strings.TrimSpace(claim.JobID)
	claim.RunID = strings.TrimSpace(claim.RunID)
	claim.FireID = strings.TrimSpace(claim.FireID)
	claim.ScheduleHash = strings.TrimSpace(claim.ScheduleHash)
	if claim.ClaimedAt.IsZero() {
		claim.ClaimedAt = g.now().UTC()
	}
	claim.NetworkParticipation = normalizeAutomationParticipationIntent(claim.NetworkParticipation)
	if err := claim.Validate("scheduler_claim"); err != nil {
		return automation.SchedulerClaim{}, err
	}
	return claim, nil
}

func getSchedulerStateTx(
	ctx context.Context,
	tx *sql.Tx,
	jobID string,
) (automation.SchedulerState, error) {
	row, err := sqlcgen.New(tx).GetAutomationSchedulerState(ctx, jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return automation.SchedulerState{}, automation.ErrSchedulerStateNotFound
		}
		return automation.SchedulerState{}, err
	}
	return automationSchedulerFromGenerated(row)
}

func upsertSchedulerStateTx(ctx context.Context, tx *sql.Tx, state automation.SchedulerState) error {
	if err := state.Validate("scheduler_state"); err != nil {
		return err
	}
	if err := sqlcgen.New(tx).UpsertAutomationSchedulerState(ctx, automationSchedulerParams(state)); err != nil {
		return fmt.Errorf("store: upsert automation scheduler state %q: %w", state.JobID, err)
	}
	return nil
}

func insertAutomationRunTx(ctx context.Context, tx *sql.Tx, run automation.Run) error {
	if err := validateAutomationRunRecord(run); err != nil {
		return err
	}
	metadataJSON, err := encodeAutomationRunMetadata(run.Metadata)
	if err != nil {
		return err
	}
	networkParticipation, err := encodeOptionalAutomationParticipation(
		run.NetworkParticipation,
		run.NetworkParticipation == nil,
		"run.network_participation",
	)
	if err != nil {
		return err
	}
	if err := sqlcgen.New(tx).InsertAutomationRun(
		ctx,
		automationRunParams(run, networkParticipation, metadataJSON),
	); err != nil {
		if isSQLiteUniqueConstraint(err) {
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
		return automation.SchedulerCatchUpPolicySkip
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
