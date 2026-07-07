package globaldb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

func (g *GlobalDB) FailRunLease(
	ctx context.Context,
	failure taskpkg.LeaseFailure,
) (taskpkg.Run, error) {
	if err := g.checkReady(ctx, "fail task run lease"); err != nil {
		return taskpkg.Run{}, err
	}
	normalized, err := failure.Normalize(g.now())
	if err != nil {
		return taskpkg.Run{}, err
	}

	var updated taskpkg.Run
	if err := g.withTaskImmediateTransaction(ctx, "fail task run lease", func(exec taskSQLExecutor) error {
		current, err := g.getTaskRunWithExecutor(ctx, exec, normalized.RunID)
		if err != nil {
			return err
		}
		if err := requireCurrentRunLease(current, normalized.ClaimToken, normalized.Now); err != nil {
			return err
		}
		if err := requireLeaseTerminalTransition(current, taskpkg.TaskRunStatusFailed); err != nil {
			return err
		}
		result, err := exec.ExecContext(
			ctx,
			`UPDATE task_runs
			 SET status = ?, lease_until = NULL, heartbeat_at = NULL, claim_token = NULL,
			     ended_at = ?, tokens_used = ?, error = ?, result_json = NULL
			 WHERE id = ? AND claim_token_hash = ?`,
			taskpkg.TaskRunStatusFailed.String(),
			store.FormatTimestamp(normalized.Now),
			normalized.TokensUsed,
			normalized.Failure.Error,
			current.ID,
			current.ClaimTokenHash,
		)
		if err != nil {
			return fmt.Errorf("store: fail task run lease %q: %w", current.ID, err)
		}
		if err := requireRowsAffected(result, taskpkg.ErrTaskRunNotFound, current.ID, "task run lease"); err != nil {
			return err
		}
		if err := recordLoopNodeTerminalWithExecutor(
			ctx,
			exec,
			current,
			"failure",
			loopFailureReasonCode(normalized.Failure),
			nil,
			normalized.Now,
		); err != nil {
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

// ListAutonomyLeaseHandles returns internal-only lease handles for one session.
// Public task-run read projections keep claim_token masked.
func (g *GlobalDB) ListAutonomyLeaseHandles(
	ctx context.Context,
	sessionID string,
) (handles []taskpkg.AutonomyLeaseHandle, err error) {
	if err := g.checkReady(ctx, "list autonomy lease handles"); err != nil {
		return nil, err
	}
	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return nil, fmt.Errorf("%w: session_id is required", taskpkg.ErrValidation)
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT tr.id, tr.task_id, COALESCE(t.workspace_id, ''), tr.status,
		        COALESCE(tr.session_id, ''), tr.claimed_by_kind, tr.claimed_by_ref,
		        COALESCE(tr.claim_token, ''), COALESCE(tr.claim_token_hash, ''),
		        tr.lease_until, tr.heartbeat_at
		   FROM task_runs tr
		   JOIN tasks t ON t.id = tr.task_id
		  WHERE tr.session_id = ?
		    AND COALESCE(tr.claim_token_hash, '') <> ''
		  ORDER BY COALESCE(tr.lease_until, '') DESC, tr.id ASC`,
		trimmedSessionID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"store: list autonomy lease handles for session %q: %w",
			trimmedSessionID,
			err,
		)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close autonomy lease handle rows: %w", closeErr)
		}
	}()

	handles = make([]taskpkg.AutonomyLeaseHandle, 0)
	for rows.Next() {
		handle, scanErr := scanAutonomyLeaseHandle(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		handles = append(handles, handle)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate autonomy lease handles: %w", err)
	}
	return handles, nil
}

func scanAutonomyLeaseHandle(rows *sql.Rows) (taskpkg.AutonomyLeaseHandle, error) {
	var handle taskpkg.AutonomyLeaseHandle
	var status string
	var claimedByKind sql.NullString
	var claimedByRef sql.NullString
	var leaseUntilRaw sql.NullString
	var heartbeatAtRaw sql.NullString
	if err := rows.Scan(
		&handle.RunID,
		&handle.TaskID,
		&handle.WorkspaceID,
		&status,
		&handle.SessionID,
		&claimedByKind,
		&claimedByRef,
		&handle.ClaimToken,
		&handle.ClaimTokenHash,
		&leaseUntilRaw,
		&heartbeatAtRaw,
	); err != nil {
		return taskpkg.AutonomyLeaseHandle{}, fmt.Errorf(
			"store: scan autonomy lease handle: %w",
			err,
		)
	}
	handle.Status = taskpkg.ParseRunStatus(status).Normalize()
	if claimedByKind.Valid || claimedByRef.Valid {
		handle.ClaimedBy = &taskpkg.ActorIdentity{
			Kind: taskpkg.ActorKind(strings.TrimSpace(claimedByKind.String)),
			Ref:  strings.TrimSpace(claimedByRef.String),
		}
	}
	if err := setAutonomyLeaseHandleTimestamps(&handle, leaseUntilRaw, heartbeatAtRaw); err != nil {
		return taskpkg.AutonomyLeaseHandle{}, err
	}
	return handle, nil
}

func setAutonomyLeaseHandleTimestamps(
	handle *taskpkg.AutonomyLeaseHandle,
	leaseUntilRaw sql.NullString,
	heartbeatAtRaw sql.NullString,
) error {
	if leaseUntilRaw.Valid && strings.TrimSpace(leaseUntilRaw.String) != "" {
		leaseUntil, err := store.ParseTimestamp(leaseUntilRaw.String)
		if err != nil {
			return fmt.Errorf("store: parse autonomy lease_until for run %q: %w", handle.RunID, err)
		}
		handle.LeaseUntil = leaseUntil
	}
	if heartbeatAtRaw.Valid && strings.TrimSpace(heartbeatAtRaw.String) != "" {
		heartbeatAt, err := store.ParseTimestamp(heartbeatAtRaw.String)
		if err != nil {
			return fmt.Errorf(
				"store: parse autonomy heartbeat_at for run %q: %w",
				handle.RunID,
				err,
			)
		}
		handle.HeartbeatAt = heartbeatAt
	}
	return nil
}

// RecoverExpiredRunLeases requeues stale active leases without issuing new ownership.
func (g *GlobalDB) RecoverExpiredRunLeases(
	ctx context.Context,
	recovery taskpkg.ExpiredLeaseRecovery,
) ([]taskpkg.ExpiredLeaseRecoveryResult, error) {
	if err := g.checkReady(ctx, "recover expired task run leases"); err != nil {
		return nil, err
	}
	normalized, err := recovery.Normalize(g.now())
	if err != nil {
		return nil, err
	}

	recovered := make([]taskpkg.ExpiredLeaseRecoveryResult, 0)
	if err := g.withTaskImmediateTransaction(ctx, "recover expired task run leases", func(exec taskSQLExecutor) error {
		runIDs, err := expiredLeaseRunIDs(ctx, exec, normalized)
		if err != nil {
			return err
		}
		for _, runID := range runIDs {
			current, err := g.getTaskRunWithExecutor(ctx, exec, runID)
			if err != nil {
				return err
			}
			if current.LeaseUntil.IsZero() || current.LeaseUntil.After(normalized.Now) {
				continue
			}
			snapshot := taskRunLeaseSnapshot{
				status:         current.Status,
				sessionID:      current.SessionID,
				leaseUntil:     current.LeaseUntil,
				claimTokenHash: current.ClaimTokenHash,
			}
			if err := requeueExpiredLease(ctx, exec, current, snapshot); err != nil {
				return err
			}
			if err := clearTaskCurrentRunProjection(ctx, exec, current.TaskID, current.ID); err != nil {
				return err
			}
			updated, err := g.getTaskRunWithExecutor(ctx, exec, current.ID)
			if err != nil {
				return err
			}
			recovered = append(recovered, taskpkg.ExpiredLeaseRecoveryResult{
				Run:                    updated,
				PreviousRunStatus:      snapshot.status,
				PreviousSessionID:      snapshot.sessionID,
				PreviousLeaseUntil:     snapshot.leaseUntil,
				PreviousClaimTokenHash: snapshot.claimTokenHash,
				Reason:                 normalized.Reason,
			})
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return recovered, nil
}
