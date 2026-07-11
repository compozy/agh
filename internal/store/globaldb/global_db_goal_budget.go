package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

const goalBudgetAuthorizationTTL = time.Second

var _ goal.BudgetGuard = (*GlobalDB)(nil)

// FlushAndCheck synchronously max-flushes one Goal task's cumulative usage and fences the next effect.
func (g *GlobalDB) FlushAndCheck(
	ctx context.Context,
	snapshot goal.BudgetBoundarySnapshot,
) (goal.BudgetDecision, error) {
	if err := g.checkReady(ctx, "flush and check Goal budget"); err != nil {
		return goal.BudgetDecision{}, err
	}
	if err := validateGoalBudgetSnapshot(snapshot); err != nil {
		return goal.BudgetDecision{}, err
	}
	var decision goal.BudgetDecision
	err := g.withTaskImmediateTransaction(ctx, "flush and check Goal budget", func(exec taskSQLExecutor) error {
		var err error
		decision, err = flushAndCheckGoalBudgetWithExecutor(ctx, exec, snapshot)
		return err
	})
	if err != nil {
		return goal.BudgetDecision{}, err
	}
	return decision, nil
}

func flushAndCheckGoalBudgetWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	snapshot goal.BudgetBoundarySnapshot,
) (goal.BudgetDecision, error) {
	if err := validateGoalRunWorkspace(ctx, exec, snapshot.Key); err != nil {
		return goal.BudgetDecision{}, err
	}
	now, err := goalBudgetDatabaseNow(ctx, exec)
	if err != nil {
		return goal.BudgetDecision{}, err
	}
	if err := flushGoalTaskUsage(ctx, exec, snapshot); err != nil {
		return goal.BudgetDecision{}, err
	}
	tokensUsed, err := refreshLoopTokensUsedWithExecutor(ctx, exec, string(snapshot.Key.LoopRunID))
	if err != nil {
		return goal.BudgetDecision{}, err
	}
	budgetTokens, budgetWallSec, onExceeded, startedAt, budgetVersion, err := advanceGoalBudgetVersion(
		ctx,
		exec,
		snapshot.Key.LoopRunID,
	)
	if err != nil {
		return goal.BudgetDecision{}, err
	}
	exceeded := goalBudgetExceeded(tokensUsed, budgetTokens, now, startedAt, budgetWallSec)
	grant, grantAllowed, err := goalBudgetGrantForBoundary(ctx, exec, snapshot, exceeded)
	if err != nil {
		return goal.BudgetDecision{}, err
	}
	decision := goal.BudgetDecision{
		Allowed:       !exceeded || grantAllowed,
		BudgetVersion: budgetVersion,
		ValidUntil:    goalBudgetValidUntil(now, startedAt, budgetWallSec, grantAllowed),
	}
	if grantAllowed {
		decision.Cause = grant.Cause
		decision.GrantID = grant.ID
		decision.GrantTurn = grant.Turn
		decision.GrantScope = grant.Scope
	}
	if decision.Allowed {
		return decision, nil
	}
	decision.Cause = loop.ReasonCodeGoalBudgetFenced
	decision.Disposition = loop.ActionDispositionExhausted
	if onExceeded == dsl.BudgetExceededEscalate {
		decision.Disposition = loop.ActionDispositionNeedsApproval
	}
	return decision, nil
}

func goalBudgetDatabaseNow(ctx context.Context, exec taskSQLExecutor) (time.Time, error) {
	var raw string
	if err := exec.QueryRowContext(
		ctx,
		`SELECT strftime('%Y-%m-%dT%H:%M:%fZ', 'now')`,
	).Scan(&raw); err != nil {
		return time.Time{}, fmt.Errorf("store: read Goal budget database time: %w", err)
	}
	now, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("store: parse Goal budget database time: %w", err)
	}
	return now.UTC(), nil
}

func flushGoalTaskUsage(
	ctx context.Context,
	exec taskSQLExecutor,
	snapshot goal.BudgetBoundarySnapshot,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE task_runs
		 SET tokens_used = CASE
		   WHEN ? = 1 AND ? > tokens_used THEN ?
		   ELSE tokens_used
		 END
		 WHERE id = ? AND loop_run_id = ?
		   AND EXISTS (
		     SELECT 1 FROM loop_goal_checkpoints AS checkpoint
		     WHERE checkpoint.loop_run_id = ? AND checkpoint.generation = ?
		       AND checkpoint.node_id = ? AND checkpoint.item_index = ?
		       AND checkpoint.control_epoch = ? AND COALESCE(checkpoint.binding_epoch, 0) = ?
		       AND checkpoint.phase = ? AND COALESCE(checkpoint.task_run_id, '') = ?
		       AND COALESCE(checkpoint.queue_entry_id, '') = ?
		       AND COALESCE(checkpoint.prompt_id, '') = ?
		   )`,
		boolToInt(snapshot.TokensReported),
		snapshot.LiveTokensUsed,
		snapshot.LiveTokensUsed,
		strings.TrimSpace(snapshot.TaskRunID),
		string(snapshot.Key.LoopRunID),
		string(snapshot.Key.LoopRunID),
		snapshot.Key.Generation,
		string(snapshot.Key.NodeID),
		snapshot.Key.ItemIndex,
		snapshot.ExpectedControlEpoch,
		snapshot.ExpectedBindingEpoch,
		strings.TrimSpace(snapshot.ExpectedPhase),
		strings.TrimSpace(snapshot.TaskRunID),
		strings.TrimSpace(snapshot.ExpectedQueueEntryID),
		strings.TrimSpace(snapshot.ExpectedPromptID),
	)
	if err != nil {
		return fmt.Errorf("store: flush Goal task usage: %w", err)
	}
	return requireGoalRowsAffected(result, "flush Goal task usage")
}

func advanceGoalBudgetVersion(
	ctx context.Context,
	exec taskSQLExecutor,
	runID loop.RunID,
) (int, int, dsl.BudgetExceeded, time.Time, int64, error) {
	var budgetTokens, budgetWallSec int
	var onExceeded, startedAtRaw string
	var budgetVersion int64
	err := exec.QueryRowContext(
		ctx,
		`UPDATE loop_runs
		 SET budget_version = budget_version + 1
		 WHERE id = ?
		 RETURNING budget_tokens, budget_wall_sec, budget_on_exceeded, started_at, budget_version`,
		string(runID),
	).Scan(&budgetTokens, &budgetWallSec, &onExceeded, &startedAtRaw, &budgetVersion)
	if err != nil {
		return 0, 0, "", time.Time{}, 0, fmt.Errorf("store: advance Goal budget version: %w", err)
	}
	startedAt, err := store.ParseTimestamp(startedAtRaw)
	if err != nil {
		return 0, 0, "", time.Time{}, 0, fmt.Errorf("store: parse Goal budget started_at: %w", err)
	}
	return budgetTokens,
		budgetWallSec,
		dsl.BudgetExceeded(strings.TrimSpace(onExceeded)),
		startedAt,
		budgetVersion,
		nil
}

func validateGoalBudgetSnapshot(snapshot goal.BudgetBoundarySnapshot) error {
	if err := snapshot.Key.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(snapshot.TaskRunID) == "" || snapshot.ExpectedControlEpoch < 1 ||
		snapshot.ExpectedBindingEpoch < 0 || !goalCheckpointPhaseValid(strings.TrimSpace(snapshot.ExpectedPhase)) ||
		strings.TrimSpace(string(snapshot.Boundary)) == "" ||
		strings.TrimSpace(snapshot.Phase) == "" || snapshot.Turn < 0 ||
		strings.TrimSpace(snapshot.OperationID) == "" || snapshot.OperationBaseTokens < 0 ||
		snapshot.LiveTokensUsed < 0 || snapshot.LiveTokensUsed < snapshot.OperationBaseTokens {
		return fmt.Errorf("%w: Goal budget boundary snapshot is invalid", loop.ErrValidation)
	}
	return nil
}

func goalBudgetExceeded(
	tokensUsed int64,
	budgetTokens int,
	now time.Time,
	startedAt time.Time,
	budgetWallSec int,
) bool {
	if budgetTokens > 0 && tokensUsed >= int64(budgetTokens) {
		return true
	}
	return budgetWallSec > 0 && !now.Before(startedAt.Add(time.Duration(budgetWallSec)*time.Second))
}

func goalBudgetValidUntil(now time.Time, startedAt time.Time, wallSeconds int, approved bool) time.Time {
	validUntil := now.Add(goalBudgetAuthorizationTTL)
	if approved || wallSeconds <= 0 {
		return validUntil
	}
	wallDeadline := startedAt.Add(time.Duration(wallSeconds) * time.Second)
	if wallDeadline.Before(validUntil) {
		return wallDeadline
	}
	return validUntil
}

func goalBudgetGrantForBoundary(
	ctx context.Context,
	exec taskSQLExecutor,
	snapshot goal.BudgetBoundarySnapshot,
	exceeded bool,
) (goal.ControlGrant, bool, error) {
	if !exceeded {
		return goal.ControlGrant{}, false, nil
	}
	checkpoint, err := loadGoalCheckpointWithExecutor(ctx, exec, snapshot.Key)
	if err != nil {
		return goal.ControlGrant{}, false, err
	}
	if checkpoint.ControlGrant == nil || checkpoint.ControlGrant.Consumed ||
		checkpoint.ControlGrant.Kind != goal.ControlGrantBudget ||
		checkpoint.ControlGrant.Turn != int(snapshot.Turn) {
		return goal.ControlGrant{}, false, nil
	}
	grant := *checkpoint.ControlGrant
	allowed := false
	switch snapshot.Boundary {
	case goal.BudgetBeforeWork:
		allowed = grant.Scope == goal.ControlGrantScopeWorkAndSettle
	case goal.BudgetAfterWork, goal.BudgetBeforeCompact, goal.BudgetAfterCompact,
		goal.BudgetBeforeJudge, goal.BudgetAfterJudge:
		allowed = grant.Scope == goal.ControlGrantScopeSettleCurrent ||
			grant.Scope == goal.ControlGrantScopeWorkAndSettle
	}
	return grant, allowed, nil
}

func validateGoalBudgetDecision(
	ctx context.Context,
	exec taskSQLExecutor,
	runID loop.RunID,
	decision goal.BudgetDecision,
) error {
	if !decision.Allowed || decision.BudgetVersion < 0 || decision.ValidUntil.IsZero() {
		return goalPromptFencedError("budget decision does not authorize an effect")
	}
	var valid int
	err := exec.QueryRowContext(
		ctx,
		`SELECT CASE
			WHEN budget_version = ? AND julianday('now') <= julianday(?) THEN 1
			ELSE 0
		 END
		 FROM loop_runs WHERE id = ?`,
		decision.BudgetVersion,
		store.FormatTimestamp(decision.ValidUntil),
		string(runID),
	).Scan(&valid)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %s", loop.ErrRunNotFound, runID)
	}
	if err != nil {
		return fmt.Errorf("store: validate goal budget decision: %w", err)
	}
	if valid != 1 {
		return goalPromptFencedError("budget decision version or validity window is stale")
	}
	return nil
}
