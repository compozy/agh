package globaldb

import (
	"context"
	"fmt"
	"strings"
	"time"

	loop "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

const (
	globalDBTaskClaimStatusKey = "status"
)

const (
	globalDBTaskClaimHandoffKey = "handoff"
)

type taskRunLeaseSnapshot struct {
	status         taskpkg.RunStatus
	sessionID      string
	leaseUntil     time.Time
	claimTokenHash string
}

const taskPriorityValueSQL = `CASE t.priority
	WHEN 'urgent' THEN 40
	WHEN 'high' THEN 30
	WHEN 'low' THEN 10
	ELSE 20
END`

// ClaimNextRun atomically selects and claims the next eligible queued task run.
func (g *GlobalDB) ClaimNextRun(
	ctx context.Context,
	criteria taskpkg.ClaimCriteria,
) (taskpkg.ClaimResult, error) {
	if err := g.checkReady(ctx, "claim next task run"); err != nil {
		return taskpkg.ClaimResult{}, err
	}
	normalized, err := criteria.Normalize(g.now())
	if err != nil {
		return taskpkg.ClaimResult{}, err
	}

	var result taskpkg.ClaimResult
	if err := g.withTaskImmediateTransaction(ctx, "claim next task run", func(exec taskSQLExecutor) error {
		if err := g.ensureClaimerHasNoActiveLease(ctx, exec, normalized); err != nil {
			return err
		}
		runID, err := g.selectClaimableRunID(ctx, exec, normalized)
		if err != nil {
			return err
		}
		if runID == "" {
			return taskpkg.ErrNoClaimableRun
		}

		claimToken, err := taskpkg.NewClaimToken()
		if err != nil {
			return err
		}
		claimHash, err := taskpkg.ClaimTokenHash(claimToken)
		if err != nil {
			return err
		}
		leaseUntil := normalized.Now.Add(normalized.LeaseDuration).UTC()
		if err := claimRunWithExecutor(ctx, exec, runID, normalized, claimToken, claimHash, leaseUntil); err != nil {
			return err
		}
		if err := setTaskCurrentRunProjectionForRun(ctx, exec, runID); err != nil {
			return err
		}

		run, err := g.getTaskRunWithExecutor(ctx, exec, runID)
		if err != nil {
			return err
		}
		if err := appendLoopNodeRunningEventWithExecutor(ctx, exec, run, normalized.Now); err != nil {
			return err
		}
		taskRecord, err := g.getTaskWithExecutor(ctx, exec, run.TaskID)
		if err != nil {
			return err
		}
		channel, err := g.coordinationChannelMetadata(ctx, exec, taskRecord, run)
		if err != nil {
			return err
		}
		result = taskpkg.ClaimResult{
			Task:                taskRecord,
			Run:                 run,
			ClaimToken:          claimToken,
			LeaseUntil:          leaseUntil,
			CoordinationChannel: channel,
		}
		return nil
	}); err != nil {
		return taskpkg.ClaimResult{}, err
	}

	return taskpkg.ClaimResult{
		Task:                result.Task,
		Run:                 result.Run,
		ClaimToken:          result.ClaimToken,
		LeaseUntil:          result.LeaseUntil,
		CoordinationChannel: result.CoordinationChannel,
	}, nil
}

// HeartbeatRunLease extends one active task-run lease after token verification.
func (g *GlobalDB) HeartbeatRunLease(
	ctx context.Context,
	heartbeat taskpkg.LeaseHeartbeat,
) (taskpkg.Run, error) {
	if err := g.checkReady(ctx, "heartbeat task run lease"); err != nil {
		return taskpkg.Run{}, err
	}
	normalized, err := heartbeat.Normalize(g.now())
	if err != nil {
		return taskpkg.Run{}, err
	}

	var updated taskpkg.Run
	if err := g.withTaskImmediateTransaction(ctx, "heartbeat task run lease", func(exec taskSQLExecutor) error {
		current, err := g.getTaskRunWithExecutor(ctx, exec, normalized.RunID)
		if err != nil {
			return err
		}
		if err := requireCurrentRunLease(current, normalized.ClaimToken, normalized.Now); err != nil {
			return err
		}
		leaseUntil := normalized.Now.Add(normalized.LeaseDuration).UTC()
		result, err := exec.ExecContext(
			ctx,
			`UPDATE task_runs
				 SET lease_until = ?, heartbeat_at = ?, claim_token = ?,
				     tokens_used = CASE
				       WHEN ? > tokens_used THEN ?
				       ELSE tokens_used
				     END
				 WHERE id = ? AND claim_token_hash = ? AND status IN (?, ?, ?)`,
			store.FormatTimestamp(leaseUntil),
			store.FormatTimestamp(normalized.Now),
			normalized.ClaimToken,
			normalized.TokensUsed,
			normalized.TokensUsed,
			normalized.RunID,
			current.ClaimTokenHash,
			taskpkg.TaskRunStatusClaimed.String(),
			taskpkg.TaskRunStatusStarting.String(),
			taskpkg.TaskRunStatusRunning.String(),
		)
		if err != nil {
			return fmt.Errorf("store: heartbeat task run lease %q: %w", normalized.RunID, err)
		}
		if err := requireRowsAffected(
			result,
			taskpkg.ErrTaskRunNotFound,
			normalized.RunID,
			"task run lease",
		); err != nil {
			return err
		}
		updated, err = g.getTaskRunWithExecutor(ctx, exec, normalized.RunID)
		if err != nil {
			return err
		}
		return g.appendLoopTokenTickForHeartbeat(ctx, exec, updated, normalized.TokensUsed)
	}); err != nil {
		return taskpkg.Run{}, err
	}
	return updated, nil
}

func (g *GlobalDB) appendLoopTokenTickForHeartbeat(
	ctx context.Context,
	exec taskSQLExecutor,
	run taskpkg.Run,
	tokensUsed int64,
) error {
	loopRunID := strings.TrimSpace(run.LoopRunID)
	if loopRunID == "" || tokensUsed <= 0 {
		return nil
	}
	tokensTotal, err := refreshLoopTokensUsedWithExecutor(ctx, exec, loopRunID)
	if err != nil {
		return err
	}
	if tokensTotal <= 0 {
		return nil
	}
	var workspaceID string
	if err := exec.QueryRowContext(
		ctx,
		`SELECT workspace_id FROM loop_runs WHERE id = ?`,
		loopRunID,
	).Scan(&workspaceID); err != nil {
		return fmt.Errorf("store: load loop run %q workspace for token tick: %w", loopRunID, err)
	}
	return appendLoopTokenTickEventWithExecutor(
		ctx,
		exec,
		loop.RunID(loopRunID),
		loop.WorkspaceID(workspaceID),
		run.ID,
		tokensTotal,
		false,
		run.HeartbeatAt,
	)
}

// ReleaseRunLease clears an active task-run lease after token verification and requeues the run.
func (g *GlobalDB) ReleaseRunLease(
	ctx context.Context,
	release taskpkg.LeaseRelease,
) (taskpkg.Run, error) {
	if err := g.checkReady(ctx, "release task run lease"); err != nil {
		return taskpkg.Run{}, err
	}
	normalized, err := release.Normalize(g.now())
	if err != nil {
		return taskpkg.Run{}, err
	}

	var updated taskpkg.Run
	if err := g.withTaskImmediateTransaction(ctx, "release task run lease", func(exec taskSQLExecutor) error {
		current, err := g.getTaskRunWithExecutor(ctx, exec, normalized.RunID)
		if err != nil {
			return err
		}
		if err := requireCurrentRunLease(current, normalized.ClaimToken, normalized.Now); err != nil {
			return err
		}
		if err := requeueLeasedRun(ctx, exec, current.ID); err != nil {
			return err
		}
		if err := clearTaskCurrentRunProjection(ctx, exec, current.TaskID, current.ID); err != nil {
			return err
		}
		updated, err = g.getTaskRunWithExecutor(ctx, exec, current.ID)
		return err
	}); err != nil {
		return taskpkg.Run{}, err
	}
	return updated, nil
}

// CompleteRunLease marks one claimed run complete after token verification.
