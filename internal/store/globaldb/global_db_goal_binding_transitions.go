package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

// FinalizeSessionBindingCreation atomically activates a created session or exposes Stop cleanup.
func (g *GlobalDB) FinalizeSessionBindingCreation(
	ctx context.Context,
	req goal.ActivateBindingRequest,
) (goal.SessionBinding, bool, error) {
	if err := g.checkReady(ctx, "finalize Goal session binding creation"); err != nil {
		return goal.SessionBinding{}, false, err
	}
	if err := req.Key.Validate(); err != nil {
		return goal.SessionBinding{}, false, err
	}
	if req.ExpectedBindingEpoch < 1 {
		return goal.SessionBinding{}, false, fmt.Errorf("store: goal binding epoch must be at least 1")
	}
	if req.ActivatedAt.IsZero() {
		req.ActivatedAt = g.now()
	}
	var binding goal.SessionBinding
	stopped := false
	err := g.withTaskImmediateTransaction(
		ctx,
		"finalize Goal session binding creation",
		func(exec taskSQLExecutor) error {
			current, err := getSessionBindingAttemptWithExecutor(ctx, exec, req.Key, req.ExpectedBindingEpoch)
			if err != nil {
				return err
			}
			if current.State == goal.BindingStateFailed {
				if current.FailureCode != goalBindingFailureStopCreationUnsettled &&
					current.FailureCode != goalBindingFailureStopCreationSettled {
					return goalControlStaleError("created Goal binding failed outside Stop settlement")
				}
				if current.FailureCode == goalBindingFailureStopCreationUnsettled {
					result, updateErr := exec.ExecContext(
						ctx,
						`UPDATE loop_session_bindings SET failure_code = ?
					 WHERE loop_run_id = ? AND handle = ? AND binding_epoch = ?
					   AND state = 'failed' AND failure_code = ?`,
						goalBindingFailureStopCreationSettled,
						string(req.Key.LoopRunID),
						req.Key.Handle,
						req.ExpectedBindingEpoch,
						goalBindingFailureStopCreationUnsettled,
					)
					if updateErr != nil {
						return fmt.Errorf("store: settle stopped Goal creation before activation: %w", updateErr)
					}
					if err := requireGoalRowsAffected(
						result,
						"settle stopped Goal creation before activation",
					); err != nil {
						return err
					}
					current.FailureCode = goalBindingFailureStopCreationSettled
				}
				binding = current
				stopped = true
				return nil
			}
			binding, err = activateSessionBindingWithExecutor(ctx, exec, req)
			return err
		},
	)
	return binding, stopped, err
}

// ActivateSessionBinding swaps one successfully created run-owned attempt into active state.
func (g *GlobalDB) ActivateSessionBinding(
	ctx context.Context,
	req goal.ActivateBindingRequest,
) (goal.SessionBinding, error) {
	if err := g.checkReady(ctx, "activate goal session binding"); err != nil {
		return goal.SessionBinding{}, err
	}
	if err := req.Key.Validate(); err != nil {
		return goal.SessionBinding{}, err
	}
	if req.ExpectedBindingEpoch < 1 {
		return goal.SessionBinding{}, fmt.Errorf("store: goal binding epoch must be at least 1")
	}
	if req.ActivatedAt.IsZero() {
		req.ActivatedAt = g.now()
	}
	var activated goal.SessionBinding
	err := g.withTaskImmediateTransaction(ctx, "activate goal session binding", func(exec taskSQLExecutor) error {
		var err error
		activated, err = activateSessionBindingWithExecutor(ctx, exec, req)
		return err
	})
	if err != nil {
		return goal.SessionBinding{}, err
	}
	return activated, nil
}

func activateSessionBindingWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.ActivateBindingRequest,
) (goal.SessionBinding, error) {
	if err := validateBindingRunWorkspace(ctx, exec, req.Key); err != nil {
		return goal.SessionBinding{}, err
	}
	target, err := getSessionBindingAttemptWithExecutor(ctx, exec, req.Key, req.ExpectedBindingEpoch)
	if err != nil {
		return goal.SessionBinding{}, err
	}
	if target.State == goal.BindingStateActive {
		return target, nil
	}
	if target.State != goal.BindingStateCreating || target.Ownership != goal.BindingOwnershipRunOwned {
		return goal.SessionBinding{}, goalControlStaleError(
			"target binding is not a run-owned creating attempt",
		)
	}
	if err := validateBindingSessionIdentity(
		ctx,
		exec,
		req.Key.WorkspaceID,
		target.SessionID,
		target.CreationProfileRef,
		target.PolicySpecDigest,
		target.CreationDigest,
	); err != nil {
		return goal.SessionBinding{}, err
	}
	current, found, err := getActiveSessionBindingWithExecutor(ctx, exec, req.Key)
	if err != nil {
		return goal.SessionBinding{}, err
	}
	if err := validateReseedGrantForActivation(ctx, exec, req, current, found); err != nil {
		return goal.SessionBinding{}, err
	}
	if found && current.BindingEpoch != target.BindingEpoch {
		nextState := goal.BindingStateReseeded
		if current.Ownership == goal.BindingOwnershipOriginBorrowed {
			nextState = goal.BindingStateClosed
		}
		if err := retireGoalBinding(ctx, exec, req, current, nextState); err != nil {
			return goal.SessionBinding{}, err
		}
	}
	if err := activateGoalBindingRow(ctx, exec, req); err != nil {
		return goal.SessionBinding{}, err
	}
	if err := resetGoalRecoveryAfterBindingActivation(ctx, exec, req); err != nil {
		return goal.SessionBinding{}, err
	}
	if req.GrantID > 0 {
		if err := consumeGoalReseedGrant(ctx, exec, req); err != nil {
			return goal.SessionBinding{}, err
		}
	}
	activated, err := getSessionBindingAttemptWithExecutor(ctx, exec, req.Key, req.ExpectedBindingEpoch)
	if err != nil {
		return goal.SessionBinding{}, err
	}
	if err := enqueueGoalReseedOutboxIfSessionOrigin(ctx, exec, req.Key, activated, req.ActivatedAt); err != nil {
		return goal.SessionBinding{}, err
	}
	return activated, nil
}

func resetGoalRecoveryAfterBindingActivation(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.ActivateBindingRequest,
) error {
	if req.CheckpointKey == nil {
		return nil
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_checkpoints
		 SET recovery_streak = 0, compaction_baseline_used = NULL,
		     compaction_recovery_required = 0, updated_at = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND control_epoch = ?`,
		store.FormatTimestamp(req.ActivatedAt),
		string(req.Key.LoopRunID),
		req.CheckpointKey.Generation,
		string(req.CheckpointKey.NodeID),
		req.CheckpointKey.ItemIndex,
		req.ExpectedControlEpoch,
	)
	if err != nil {
		return fmt.Errorf("store: reset Goal recovery after binding activation: %w", err)
	}
	return requireGoalRowsAffected(result, "reset Goal recovery after binding activation")
}

func retireGoalBinding(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.ActivateBindingRequest,
	current goal.SessionBinding,
	nextState goal.BindingState,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_session_bindings
		 SET state = ?, closed_at = ?
		 WHERE loop_run_id = ? AND handle = ? AND binding_epoch = ? AND state = 'active'`,
		string(nextState),
		store.FormatTimestamp(req.ActivatedAt),
		string(req.Key.LoopRunID),
		req.Key.Handle,
		current.BindingEpoch,
	)
	if err != nil {
		return fmt.Errorf("store: retire active goal binding: %w", err)
	}
	if err := requireGoalRowsAffected(result, "retire active goal binding"); err != nil {
		return err
	}
	if current.Ownership == goal.BindingOwnershipRunOwned {
		return enqueueGoalSessionCleanupWithExecutor(
			ctx,
			exec,
			current,
			goal.SessionCleanupCauseReseed,
			req.ActivatedAt,
		)
	}
	return nil
}

func activateGoalBindingRow(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.ActivateBindingRequest,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_session_bindings
		 SET state = 'active', activated_at = ?
		 WHERE loop_run_id = ? AND handle = ? AND binding_epoch = ? AND state = 'creating'`,
		store.FormatTimestamp(req.ActivatedAt),
		string(req.Key.LoopRunID),
		req.Key.Handle,
		req.ExpectedBindingEpoch,
	)
	if err != nil {
		return fmt.Errorf("store: activate goal binding: %w", err)
	}
	return requireGoalRowsAffected(result, "activate goal binding")
}

func consumeGoalReseedGrant(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.ActivateBindingRequest,
) error {
	query := `UPDATE loop_goal_checkpoints
		 SET control_grant_consumed = 1, updated_at = ?
		 WHERE loop_run_id = ? AND control_epoch = ? AND control_grant_id = ?
		   AND control_grant_kind = 'reseed' AND control_grant_scope = 'rotate-binding'
		   AND control_grant_consumed = 0`
	args := []any{
		store.FormatTimestamp(req.ActivatedAt),
		string(req.Key.LoopRunID),
		req.ExpectedControlEpoch,
		req.GrantID,
	}
	if req.CheckpointKey != nil {
		query += goalBindingCheckpointPredicate
		args = append(
			args,
			req.CheckpointKey.Generation,
			string(req.CheckpointKey.NodeID),
			req.CheckpointKey.ItemIndex,
		)
	}
	result, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("store: consume goal reseed grant: %w", err)
	}
	return requireGoalRowsAffected(result, "consume goal reseed grant")
}

// CloseSessionBinding closes one exact active epoch idempotently.
func (g *GlobalDB) CloseSessionBinding(ctx context.Context, req goal.CloseBindingRequest) error {
	if err := g.checkReady(ctx, "close goal session binding"); err != nil {
		return err
	}
	if err := req.Key.Validate(); err != nil {
		return err
	}
	if req.ExpectedBindingEpoch < 1 {
		return fmt.Errorf("store: goal binding epoch must be at least 1")
	}
	if req.ClosedAt.IsZero() {
		req.ClosedAt = g.now()
	}
	return g.withTaskImmediateTransaction(ctx, "close goal session binding", func(exec taskSQLExecutor) error {
		if err := validateBindingRunWorkspace(ctx, exec, req.Key); err != nil {
			return err
		}
		binding, err := getSessionBindingAttemptWithExecutor(ctx, exec, req.Key, req.ExpectedBindingEpoch)
		if err != nil {
			return err
		}
		if binding.State == goal.BindingStateClosed || binding.State == goal.BindingStateReseeded {
			return nil
		}
		if binding.State != goal.BindingStateActive {
			return goalControlStaleError("binding is not active")
		}
		return closeGoalBindingWithCleanup(
			ctx,
			exec,
			req.Key,
			req.ExpectedBindingEpoch,
			binding.SessionID,
			goal.SessionCleanupCauseTerminal,
			req.ClosedAt,
		)
	})
}

// GetActiveSessionBinding returns the exact active epoch for one workspace-owned handle.
func (g *GlobalDB) GetActiveSessionBinding(
	ctx context.Context,
	key goal.BindingKey,
) (goal.SessionBinding, error) {
	if err := g.checkReady(ctx, "get active goal session binding"); err != nil {
		return goal.SessionBinding{}, err
	}
	if err := validateBindingRunWorkspace(ctx, g.db, key); err != nil {
		return goal.SessionBinding{}, err
	}
	binding, found, err := getActiveSessionBindingWithExecutor(ctx, g.db, key)
	if err != nil {
		return goal.SessionBinding{}, err
	}
	if !found {
		return goal.SessionBinding{}, fmt.Errorf("%w: %s", goal.ErrBindingNotFound, key.Handle)
	}
	return binding, nil
}

// GetSessionBindingAttempt returns one immutable binding attempt by epoch.
func (g *GlobalDB) GetSessionBindingAttempt(
	ctx context.Context,
	key goal.BindingKey,
	epoch int64,
) (goal.SessionBinding, error) {
	if err := g.checkReady(ctx, "get goal session binding attempt"); err != nil {
		return goal.SessionBinding{}, err
	}
	if err := validateBindingRunWorkspace(ctx, g.db, key); err != nil {
		return goal.SessionBinding{}, err
	}
	return getSessionBindingAttemptWithExecutor(ctx, g.db, key, epoch)
}

func validateReseedGrantForActivation(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.ActivateBindingRequest,
	current goal.SessionBinding,
	currentFound bool,
) error {
	if req.GrantID == 0 && req.ExpectedControlEpoch == 0 {
		return nil
	}
	if req.ExpectedControlEpoch < 1 {
		return goalControlStaleError("binding activation control identity is incomplete")
	}
	if err := validateBindingActivationControlEpoch(ctx, exec, req); err != nil {
		return err
	}
	if req.GrantID == 0 {
		if currentFound && current.Ownership == goal.BindingOwnershipOriginBorrowed &&
			current.BindingEpoch != req.ExpectedBindingEpoch {
			return goalControlStaleError("borrowed-origin reseed requires an explicit grant")
		}
		return nil
	}
	var exists int
	query := `SELECT 1 FROM loop_goal_checkpoints
		 WHERE loop_run_id = ? AND control_epoch = ? AND control_grant_id = ?
		   AND control_grant_kind = 'reseed' AND control_grant_scope = 'rotate-binding'
		   AND control_grant_consumed = 0`
	args := []any{string(req.Key.LoopRunID), req.ExpectedControlEpoch, req.GrantID}
	if req.CheckpointKey != nil {
		query += goalBindingCheckpointPredicate
		args = append(
			args,
			req.CheckpointKey.Generation,
			string(req.CheckpointKey.NodeID),
			req.CheckpointKey.ItemIndex,
		)
	}
	err := exec.QueryRowContext(ctx, query, args...).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return goalControlStaleError("reseed grant is stale or consumed")
	}
	if err != nil {
		return fmt.Errorf("store: validate reseed activation grant: %w", err)
	}
	return nil
}

func validateBindingActivationControlEpoch(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.ActivateBindingRequest,
) error {
	query := `SELECT 1 FROM loop_goal_checkpoints WHERE loop_run_id = ? AND control_epoch = ?`
	args := []any{string(req.Key.LoopRunID), req.ExpectedControlEpoch}
	if req.CheckpointKey != nil {
		if err := req.CheckpointKey.Validate(); err != nil {
			return err
		}
		query += goalBindingCheckpointPredicate
		args = append(
			args,
			req.CheckpointKey.Generation,
			string(req.CheckpointKey.NodeID),
			req.CheckpointKey.ItemIndex,
		)
	}
	var exists int
	err := exec.QueryRowContext(ctx, query, args...).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return goalControlStaleError("binding activation control epoch is stale")
	}
	if err != nil {
		return fmt.Errorf("store: validate binding activation control epoch: %w", err)
	}
	return nil
}
