package globaldb

import (
	"context"
	"fmt"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	taskpkg "github.com/compozy/agh/internal/task"
)

func (g *GlobalDB) applyCoordinatorBudgetExceededBoundaryWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	completion *taskpkg.CoordinatorCompletion,
	current *taskpkg.Run,
	snapshot taskpkg.GenerationSnapshot,
	postReserveSnapshot *taskpkg.GenerationSnapshot,
	finalizer taskpkg.GenerationStateFinalizer,
	loopRun looppkg.Run,
	result *taskpkg.CoordinatorCompletionResult,
) error {
	if loopRun.BudgetApprovalSeq > 0 {
		consumed, err := consumeLoopBudgetApprovalWithExecutor(ctx, exec, loopRun.ID)
		if err != nil {
			return err
		}
		if consumed {
			return g.applyCoordinatorContinueBoundaryWithExecutor(
				ctx,
				exec,
				completion,
				current,
				snapshot,
				postReserveSnapshot,
				finalizer,
				result,
			)
		}
	}
	return applyCoordinatorBudgetBoundary(ctx, exec, completion, snapshot, loopRun, result)
}

func consumeLoopBudgetApprovalWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	runID looppkg.RunID,
) (bool, error) {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_runs
		    SET budget_approval_seq = budget_approval_seq - 1
		  WHERE id = ? AND budget_approval_seq > 0`,
		string(runID),
	)
	if err != nil {
		return false, fmt.Errorf("store: consume loop run %q budget approval: %w", runID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("store: rows affected consuming loop run %q budget approval: %w", runID, err)
	}
	return affected > 0, nil
}

func loopBudgetExceeded(run looppkg.Run, tokensUsed int64, now time.Time) bool {
	if run.BudgetTokens > 0 && tokensUsed >= int64(run.BudgetTokens) {
		return true
	}
	if run.BudgetWallSec <= 0 || run.StartedAt.IsZero() {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return !now.UTC().Before(run.StartedAt.UTC().Add(time.Duration(run.BudgetWallSec) * time.Second))
}
