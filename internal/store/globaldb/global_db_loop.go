package globaldb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/store"
)

var _ looppkg.Store = (*GlobalDB)(nil)

const loopRunSelectColumnsSQL = `
	id, workspace_id, loop_name, status, generation, reattempt_strategy, created_at, started_at,
	last_progress_at, definition_version, definition_digest, active_gate_id,
	active_human_criteria_json, budget_approval_seq, start_metadata_json,
	consecutive_failures, budget_tokens, budget_wall_sec,
	budget_on_exceeded, tokens_used, parent_loop_run_id, pause_requested,
	control_actor_kind, control_actor_id, control_requested_at, inputs_json, iteration_cap,
	started_by_kind, started_by_ref, started_origin_kind, started_origin_ref,
	goal_context_nudge_ratio, origin_kind, origin_session_id,
	origin_creation_profile_ref, origin_policy_spec_digest, origin_creation_digest`

// CreateLoopRunForStart atomically applies the loop concurrency policy and persists a new run.
func (g *GlobalDB) CreateLoopRunForStart(
	ctx context.Context,
	run looppkg.Run,
	policy dsl.ConcurrencyPolicy,
) (looppkg.Run, error) {
	if err := g.checkReady(ctx, "create loop run for start"); err != nil {
		return looppkg.Run{}, err
	}
	normalized, err := normalizeLoopRunForCreate(run)
	if err != nil {
		return looppkg.Run{}, err
	}
	if normalized.Origin.Kind != looppkg.RunOriginCatalog {
		return looppkg.Run{}, fmt.Errorf("%w: catalog start requires catalog Run origin", looppkg.ErrValidation)
	}
	inputsJSON, startMetadataJSON, err := marshalLoopRunCreatePayload(normalized)
	if err != nil {
		return looppkg.Run{}, err
	}
	if policy == "" {
		policy = dsl.ConcurrencyForbid
	}
	created := normalized
	err = g.withTaskImmediateTransaction(ctx, "create loop run for start", func(exec taskSQLExecutor) error {
		var decisionErr error
		created, decisionErr = applyLoopStartConcurrencyPolicy(ctx, exec, normalized, policy)
		if decisionErr != nil {
			return decisionErr
		}
		return g.persistStartedLoopRunWithExecutor(ctx, exec, created, inputsJSON, startMetadataJSON)
	})
	if err != nil {
		return looppkg.Run{}, err
	}
	return created, nil
}

func (g *GlobalDB) persistStartedLoopRunWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	run looppkg.Run,
	inputsJSON []byte,
	startMetadataJSON []byte,
) error {
	if err := upsertLoopDefinitionSnapshot(ctx, exec, run, g.now()); err != nil {
		return err
	}
	if err := insertLoopRun(ctx, exec, run, inputsJSON, startMetadataJSON); err != nil {
		return err
	}
	if run.Status != looppkg.StatusRunning {
		return nil
	}
	_, _, err := g.reserveLoopCoordinatorRunWithExecutor(
		ctx,
		exec,
		run,
		loopCoordinatorStartOrigin(),
		run.CreatedAt,
		loopCoordinatorRunID(run.ID, run.Generation+1),
		loopCoordinatorIdempotencyKey(run.ID, run.Generation+1),
	)
	return err
}

func applyLoopStartConcurrencyPolicy(
	ctx context.Context,
	exec taskSQLExecutor,
	run looppkg.Run,
	policy dsl.ConcurrencyPolicy,
) (looppkg.Run, error) {
	active, err := findActiveLoopRunWithExecutor(ctx, exec, run.WorkspaceID, run.LoopName)
	if err != nil {
		return looppkg.Run{}, err
	}
	created := run
	switch policy {
	case dsl.ConcurrencyForbid:
		if active != nil {
			return looppkg.Run{}, loopConcurrencyConflict(active)
		}
		created.Status = looppkg.StatusRunning
	case dsl.ConcurrencyQueue:
		if active != nil {
			created.Status = looppkg.StatusQueued
		} else {
			created.Status = looppkg.StatusRunning
		}
	case dsl.ConcurrencyAllow:
		created.Status = looppkg.StatusRunning
	default:
		return looppkg.Run{}, fmt.Errorf("%w: concurrency policy is invalid: %q", looppkg.ErrValidation, policy)
	}
	return created, nil
}

func insertLoopRun(
	ctx context.Context,
	exec taskSQLExecutor,
	run looppkg.Run,
	inputsJSON []byte,
	startMetadataJSON []byte,
) error {
	_, err := exec.ExecContext(
		ctx,
		`INSERT INTO loop_runs (
			id, workspace_id, loop_name, status, generation, reattempt_strategy, created_at, started_at,
			last_progress_at, definition_version, definition_digest, active_gate_id,
			active_human_criteria_json, budget_approval_seq, start_metadata_json,
			consecutive_failures, iteration_cap, budget_tokens, budget_wall_sec,
			budget_on_exceeded, tokens_used, parent_loop_run_id, pause_requested, inputs_json,
			started_by_kind, started_by_ref, started_origin_kind, started_origin_ref,
			goal_context_nudge_ratio, origin_kind, origin_session_id,
			origin_creation_profile_ref, origin_policy_spec_digest, origin_creation_digest
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(run.ID),
		string(run.WorkspaceID),
		run.LoopName,
		string(run.Status),
		run.Generation,
		string(run.ReattemptStrategy),
		store.FormatTimestamp(run.CreatedAt),
		store.FormatTimestamp(run.StartedAt),
		store.FormatTimestamp(run.LastProgressAt),
		run.DefinitionVersion,
		run.DefinitionDigest,
		string(run.ActiveGateID),
		string(run.ActiveHumanCriteria),
		run.BudgetApprovalSeq,
		string(startMetadataJSON),
		run.ConsecutiveFailures,
		run.IterationCap,
		run.BudgetTokens,
		run.BudgetWallSec,
		string(run.BudgetOnExceeded),
		run.TokensUsed,
		nullString(string(run.ParentLoopRunID)),
		boolToInt(run.PauseRequested),
		string(inputsJSON),
		string(run.StartedBy.Kind),
		run.StartedBy.Ref,
		string(run.StartedOrigin.Kind),
		run.StartedOrigin.Ref,
		run.GoalContextNudgeRatio,
		string(run.Origin.Kind),
		nullString(run.Origin.SessionID),
		nullString(run.Origin.CreationProfileRef),
		nullString(run.Origin.PolicySpecDigest),
		nullString(run.Origin.CreationDigest),
	)
	if err != nil {
		return fmt.Errorf("store: insert loop run %q: %w", run.ID, err)
	}
	return appendLoopRunStatusEvent(
		ctx,
		exec,
		run.ID,
		run.WorkspaceID,
		"",
		run.Status,
		looppkg.TransitionCauseStart,
		run.CreatedAt,
	)
}

func marshalLoopRunCreatePayload(run looppkg.Run) ([]byte, []byte, error) {
	inputsJSON, err := json.Marshal(run.Inputs)
	if err != nil {
		return nil, nil, fmt.Errorf("store: marshal loop run inputs: %w", err)
	}
	startMetadataJSON, err := json.Marshal(run.StartMetadata)
	if err != nil {
		return nil, nil, fmt.Errorf("store: marshal loop run start metadata: %w", err)
	}
	return inputsJSON, startMetadataJSON, nil
}

// GetLoopRun loads one workspace-scoped loop_run.
func (g *GlobalDB) GetLoopRun(
	ctx context.Context,
	ws looppkg.WorkspaceID,
	runID looppkg.RunID,
) (looppkg.Run, error) {
	if err := g.checkReady(ctx, "get loop run"); err != nil {
		return looppkg.Run{}, err
	}
	row := g.db.QueryRowContext(
		ctx,
		`SELECT `+loopRunSelectColumnsSQL+` FROM loop_runs WHERE workspace_id = ? AND id = ?`,
		string(ws),
		string(runID),
	)
	return scanLoopRun(row)
}

// GetLoopRunByID loads one loop_run without a workspace filter for ancestry/transition internals.
func (g *GlobalDB) GetLoopRunByID(ctx context.Context, runID looppkg.RunID) (looppkg.Run, error) {
	if err := g.checkReady(ctx, "get loop run by id"); err != nil {
		return looppkg.Run{}, err
	}
	row := g.db.QueryRowContext(
		ctx,
		`SELECT `+loopRunSelectColumnsSQL+` FROM loop_runs WHERE id = ?`,
		string(runID),
	)
	return scanLoopRun(row)
}

// FindActiveLoopRun returns the oldest live run for a workspace loop.
func (g *GlobalDB) FindActiveLoopRun(
	ctx context.Context,
	ws looppkg.WorkspaceID,
	loopName string,
) (*looppkg.Run, error) {
	if err := g.checkReady(ctx, "find active loop run"); err != nil {
		return nil, err
	}
	return findActiveLoopRunWithExecutor(ctx, g.db, ws, loopName)
}

func findActiveLoopRunWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	ws looppkg.WorkspaceID,
	loopName string,
) (*looppkg.Run, error) {
	row := exec.QueryRowContext(
		ctx,
		`SELECT `+loopRunSelectColumnsSQL+`
		 FROM loop_runs
		 WHERE workspace_id = ?
		   AND loop_name = ?
		   AND status IN (?, ?, ?, ?, ?)
		 ORDER BY created_at ASC, id ASC
		 LIMIT 1`,
		string(ws),
		strings.TrimSpace(loopName),
		string(looppkg.StatusQueued),
		string(looppkg.StatusRunning),
		string(looppkg.StatusWatching),
		string(looppkg.StatusNeedsApproval),
		string(looppkg.StatusPaused),
	)
	run, err := scanLoopRun(row)
	if err != nil {
		if errors.Is(err, looppkg.ErrRunNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &run, nil
}

func loopConcurrencyConflict(active *looppkg.Run) error {
	if active == nil {
		return looppkg.ErrConcurrencyConflict
	}
	return &looppkg.ReasonError{
		Code: looppkg.ReasonCodeActiveRunExists,
		Err:  looppkg.ErrConcurrencyConflict,
		Meta: map[string]string{
			"active_run_id":            string(active.ID),
			columnLoopName:             active.LoopName,
			globalDBTaskClaimStatusKey: string(active.Status),
		},
	}
}

// CompareAndSwapLoopRunStatus updates status only from the expected durable value.
func (g *GlobalDB) CompareAndSwapLoopRunStatus(
	ctx context.Context,
	runID looppkg.RunID,
	from looppkg.Status,
	to looppkg.Status,
	cause looppkg.TransitionCause,
	at time.Time,
) error {
	if err := g.checkReady(ctx, "transition loop run status"); err != nil {
		return err
	}
	if strings.TrimSpace(string(runID)) == "" {
		return fmt.Errorf("%w: run id is required", looppkg.ErrValidation)
	}
	if !from.Valid() || !to.Valid() {
		return fmt.Errorf("%w: loop status is invalid", looppkg.ErrValidation)
	}
	if strings.TrimSpace(string(cause)) == "" {
		return fmt.Errorf("%w: transition cause is required", looppkg.ErrValidation)
	}
	if from == to {
		return nil
	}
	if at.IsZero() {
		at = g.now()
	}
	return g.withTaskImmediateTransaction(
		ctx,
		"transition loop run status",
		func(exec taskSQLExecutor) error {
			return compareAndSwapLoopRunStatusWithExecutor(ctx, exec, runID, from, to, cause, at)
		},
	)
}

// UpsertLoopConfig persists a no-fork per-loop config override.
func (g *GlobalDB) UpsertLoopConfig(
	ctx context.Context,
	ws looppkg.WorkspaceID,
	loopName string,
	cfg looppkg.LoopConfig,
) error {
	if err := g.checkReady(ctx, "upsert loop config"); err != nil {
		return err
	}
	workspaceID := strings.TrimSpace(string(ws))
	trimmedLoopName := strings.TrimSpace(loopName)
	if workspaceID == "" {
		return fmt.Errorf("%w: workspace_id is required", looppkg.ErrValidation)
	}
	if trimmedLoopName == "" {
		return fmt.Errorf("%w: loop_name is required", looppkg.ErrValidation)
	}
	normalized, err := normalizeLoopConfigForStore(cfg)
	if err != nil {
		return err
	}
	patch := loopConfigPatchFlagsForStore(cfg, normalized)
	_, err = g.db.ExecContext(
		ctx,
		`INSERT INTO loop_config (
			workspace_id, loop_name, human_gate_enabled, reattempt_strategy, enabled_checks_json,
			iteration_cap, budget_tokens, budget_wall_sec, budget_on_exceeded,
			no_progress_window, fan_out_width, gate_max_revisions,
			model_default_worker, model_default_judge
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(workspace_id, loop_name) DO UPDATE SET
				human_gate_enabled = CASE WHEN ? THEN excluded.human_gate_enabled ELSE human_gate_enabled END,
				reattempt_strategy = CASE WHEN ? THEN excluded.reattempt_strategy ELSE reattempt_strategy END,
				enabled_checks_json = CASE WHEN ? THEN excluded.enabled_checks_json ELSE enabled_checks_json END,
				iteration_cap = CASE WHEN ? THEN excluded.iteration_cap ELSE iteration_cap END,
				budget_tokens = CASE WHEN ? THEN excluded.budget_tokens ELSE budget_tokens END,
				budget_wall_sec = CASE WHEN ? THEN excluded.budget_wall_sec ELSE budget_wall_sec END,
				budget_on_exceeded = CASE WHEN ? THEN excluded.budget_on_exceeded ELSE budget_on_exceeded END,
				no_progress_window = CASE WHEN ? THEN excluded.no_progress_window ELSE no_progress_window END,
				fan_out_width = CASE WHEN ? THEN excluded.fan_out_width ELSE fan_out_width END,
				gate_max_revisions = CASE WHEN ? THEN excluded.gate_max_revisions ELSE gate_max_revisions END,
				model_default_worker = CASE WHEN ? THEN excluded.model_default_worker ELSE model_default_worker END,
				model_default_judge = CASE WHEN ? THEN excluded.model_default_judge ELSE model_default_judge END`,
		workspaceID,
		trimmedLoopName,
		boolPtrToInt(normalized.HumanGateEnabled),
		nullStringPtr(normalized.ReattemptStrategy),
		enabledChecksForStore(normalized.EnabledChecks),
		nullIntPtr(normalized.IterationCap),
		nullIntPtr(normalized.BudgetTokens),
		nullIntPtr(normalized.BudgetWallSec),
		nullStringPtr(normalized.BudgetOnExceeded),
		nullIntPtr(normalized.NoProgressWindow),
		nullIntPtr(normalized.FanOutWidth),
		nullIntPtr(normalized.GateMaxRevisions),
		modelDefaultNullString(normalized.ModelDefaults, true),
		modelDefaultNullString(normalized.ModelDefaults, false),
		patch.HumanGate,
		patch.Reattempt,
		patch.EnabledChecks,
		patch.IterationCap,
		patch.BudgetTokens,
		patch.BudgetWallSec,
		patch.BudgetOnExceeded,
		patch.NoProgressWindow,
		patch.FanOutWidth,
		patch.GateMaxRevisions,
		patch.ModelWorker,
		patch.ModelJudge,
	)
	if err != nil {
		return fmt.Errorf("store: upsert loop config %q/%q: %w", workspaceID, trimmedLoopName, err)
	}
	return nil
}

// GetLoopConfig loads a no-fork per-loop config override.
func (g *GlobalDB) GetLoopConfig(
	ctx context.Context,
	ws looppkg.WorkspaceID,
	loopName string,
) (*looppkg.LoopConfig, error) {
	if err := g.checkReady(ctx, "get loop config"); err != nil {
		return nil, err
	}
	row := g.db.QueryRowContext(
		ctx,
		`SELECT human_gate_enabled, reattempt_strategy, enabled_checks_json,
		        iteration_cap, budget_tokens, budget_wall_sec, budget_on_exceeded,
		        no_progress_window, fan_out_width, gate_max_revisions,
		        model_default_worker, model_default_judge
		 FROM loop_config
		 WHERE workspace_id = ? AND loop_name = ?`,
		string(ws),
		strings.TrimSpace(loopName),
	)
	cfg, err := scanLoopConfig(row)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func getLoopRunByIDWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	runID looppkg.RunID,
) (looppkg.Run, error) {
	row := exec.QueryRowContext(
		ctx,
		`SELECT `+loopRunSelectColumnsSQL+` FROM loop_runs WHERE id = ?`,
		string(runID),
	)
	return scanLoopRun(row)
}
