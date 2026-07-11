package globaldb

import (
	"context"
	"fmt"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
)

func compareAndSwapLoopRunStatusWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	runID looppkg.RunID,
	from looppkg.Status,
	to looppkg.Status,
	cause looppkg.TransitionCause,
	at time.Time,
) error {
	current, err := getLoopRunByIDWithExecutor(ctx, exec, runID)
	if err != nil {
		return err
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_runs
		 SET status = ?,
		     pause_requested = CASE WHEN ? IN (?, ?, ?, ?, ?, ?, ?, ?) THEN 0 ELSE pause_requested END,
		     active_gate_id = CASE WHEN ? != ? THEN '' ELSE active_gate_id END,
		     active_human_criteria_json = CASE WHEN ? != ? THEN '[]' ELSE active_human_criteria_json END
		 WHERE id = ? AND status = ?`,
		string(to),
		string(to),
		string(looppkg.StatusRunning),
		string(looppkg.StatusPaused),
		string(looppkg.StatusDone),
		string(looppkg.StatusNoOp),
		string(looppkg.StatusBlocked),
		string(looppkg.StatusFailed),
		string(looppkg.StatusExhausted),
		string(looppkg.StatusStalled),
		string(to),
		string(looppkg.StatusNeedsApproval),
		string(to),
		string(looppkg.StatusNeedsApproval),
		string(runID),
		string(from),
	)
	if err != nil {
		return fmt.Errorf("store: transition loop run %q: %w", runID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for loop run %q transition: %w", runID, err)
	}
	if affected == 0 {
		return fmt.Errorf("%w: run_id=%s from=%s to=%s", looppkg.ErrTransitionConflict, runID, from, to)
	}
	if err := appendLoopRunStatusEvent(ctx, exec, runID, current.WorkspaceID, from, to, cause, at); err != nil {
		return err
	}
	if to.Terminal() {
		if err := sweepOrphanedLoopOutputBlobsWithExecutor(ctx, exec); err != nil {
			return err
		}
	}
	return nil
}
