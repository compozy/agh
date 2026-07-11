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
	clearControlState := loopStatusClearsControlState(to)
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_runs
		 SET status = ?,
		     pause_requested = CASE WHEN ? THEN 0 ELSE pause_requested END,
		     control_actor_kind = CASE WHEN ? THEN NULL ELSE control_actor_kind END,
		     control_actor_id = CASE WHEN ? THEN NULL ELSE control_actor_id END,
		     control_requested_at = CASE WHEN ? THEN NULL ELSE control_requested_at END,
		     active_gate_id = CASE WHEN ? != ? THEN '' ELSE active_gate_id END,
		     active_human_criteria_json = CASE WHEN ? != ? THEN '[]' ELSE active_human_criteria_json END
		 WHERE id = ? AND status = ?`,
		string(to),
		clearControlState,
		clearControlState,
		clearControlState,
		clearControlState,
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

func loopStatusClearsControlState(status looppkg.Status) bool {
	switch status {
	case looppkg.StatusRunning,
		looppkg.StatusPaused,
		looppkg.StatusDone,
		looppkg.StatusNoOp,
		looppkg.StatusBlocked,
		looppkg.StatusFailed,
		looppkg.StatusExhausted,
		looppkg.StatusStalled:
		return true
	default:
		return false
	}
}
