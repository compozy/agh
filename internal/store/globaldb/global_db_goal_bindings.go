package globaldb

import (
	"context"
	"fmt"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

const goalBindingSelectColumns = `
	loop_run_id, handle, binding_epoch, binding_attempt_id, session_id, workspace_id,
	creation_profile_ref, policy_spec_digest, creation_digest, ownership, state,
	adopted_generation, adoption_attempt_id, failure_code, created_at, activated_at, failed_at, closed_at`

var (
	_ goal.BindingStore = (*GlobalDB)(nil)
)

// GetOrCreateSessionBinding atomically adopts an origin session or returns the compatible active epoch.
func (g *GlobalDB) GetOrCreateSessionBinding(
	ctx context.Context,
	req goal.GetOrCreateBindingRequest,
) (goal.SessionBinding, error) {
	if err := g.checkReady(ctx, "get or create goal session binding"); err != nil {
		return goal.SessionBinding{}, err
	}
	normalized, err := normalizeOriginBindingRequest(req, g.now())
	if err != nil {
		return goal.SessionBinding{}, err
	}
	var binding goal.SessionBinding
	err = g.withTaskImmediateTransaction(ctx, "get or create goal session binding", func(exec taskSQLExecutor) error {
		if err := validateBindingRunWorkspace(ctx, exec, normalized.Key); err != nil {
			return err
		}
		if err := validateOriginBindingAdoptionOwner(ctx, exec, normalized); err != nil {
			return err
		}
		active, found, err := getActiveSessionBindingWithExecutor(ctx, exec, normalized.Key)
		if err != nil {
			return err
		}
		if found {
			if !bindingOriginIdentityMatches(active, normalized) ||
				(active.AdoptedGeneration == normalized.CheckpointKey.Generation &&
					active.AdoptionAttemptID != normalized.BindingAttemptID) {
				return goalBindingMismatchError("active binding policy/profile or origin identity differs")
			}
			binding, err = advanceActiveOriginBindingAdoption(ctx, exec, active, normalized)
			return err
		}
		existing, found, err := findSessionBindingAttemptWithExecutor(ctx, exec, normalized.Key, 1)
		if err != nil {
			return err
		}
		if found {
			binding, err = reactivateClosedOriginBinding(ctx, exec, existing, normalized)
			return err
		}
		if err := validateBindingSessionIdentity(
			ctx,
			exec,
			normalized.Key.WorkspaceID,
			normalized.SessionID,
			normalized.CreationProfileRef,
			normalized.PolicySpecDigest,
			normalized.CreationDigest,
		); err != nil {
			return err
		}
		_, err = exec.ExecContext(
			ctx,
			`INSERT INTO loop_session_bindings (
				loop_run_id, handle, binding_epoch, binding_attempt_id, session_id, workspace_id,
				creation_profile_ref, policy_spec_digest, creation_digest, ownership, state,
				created_at, activated_at, adopted_generation, adoption_attempt_id
			) VALUES (?, ?, 1, ?, ?, ?, ?, ?, ?, 'origin-borrowed', 'active', ?, ?, ?, ?)`,
			string(normalized.Key.LoopRunID),
			normalized.Key.Handle,
			normalized.BindingAttemptID,
			normalized.SessionID,
			string(normalized.Key.WorkspaceID),
			normalized.CreationProfileRef,
			normalized.PolicySpecDigest,
			normalized.CreationDigest,
			store.FormatTimestamp(normalized.CreatedAt),
			store.FormatTimestamp(normalized.CreatedAt),
			normalized.CheckpointKey.Generation,
			normalized.BindingAttemptID,
		)
		if err != nil {
			return fmt.Errorf("store: insert origin goal binding: %w", err)
		}
		binding, err = getSessionBindingAttemptWithExecutor(ctx, exec, normalized.Key, 1)
		return err
	})
	if err != nil {
		return goal.SessionBinding{}, err
	}
	return binding, nil
}

func validateOriginBindingAdoptionOwner(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.GetOrCreateBindingRequest,
) error {
	checkpoint, err := loadGoalCheckpointWithExecutor(ctx, exec, req.CheckpointKey)
	if err != nil {
		return err
	}
	if checkpoint.ControlEpoch != req.ExpectedControlEpoch ||
		checkpoint.Phase != req.ExpectedCheckpointPhase ||
		checkpoint.TaskRunID != req.ExpectedTaskRunID ||
		checkpoint.QueueEntryID != req.ExpectedQueueEntryID ||
		checkpoint.PromptID != req.ExpectedPromptID ||
		checkpoint.BindingEpoch != req.ExpectedCheckpointBindingEpoch ||
		checkpoint.SessionID != req.ExpectedCheckpointSessionID ||
		checkpoint.BindingHandle != req.ExpectedCheckpointHandle {
		return goalControlStaleError("origin binding checkpoint owner changed")
	}
	var status string
	var generation int
	if err := exec.QueryRowContext(
		ctx,
		`SELECT status, generation FROM loop_runs WHERE id = ? AND workspace_id = ?`,
		string(req.Key.LoopRunID),
		string(req.Key.WorkspaceID),
	).Scan(&status, &generation); err != nil {
		return fmt.Errorf("store: load origin binding Run owner: %w", err)
	}
	if status != string(looppkg.StatusRunning) || generation != req.CheckpointKey.Generation {
		return goalControlStaleError("origin binding Run is not live at the requested generation")
	}
	return nil
}

func advanceActiveOriginBindingAdoption(
	ctx context.Context,
	exec taskSQLExecutor,
	existing goal.SessionBinding,
	req goal.GetOrCreateBindingRequest,
) (goal.SessionBinding, error) {
	generation := req.CheckpointKey.Generation
	if existing.AdoptedGeneration > generation {
		return goal.SessionBinding{}, goalControlStaleError("origin binding adoption generation advanced")
	}
	if existing.AdoptedGeneration == generation {
		if existing.AdoptionAttemptID != req.BindingAttemptID {
			return goal.SessionBinding{}, goalControlStaleError("origin binding adoption attempt changed")
		}
		return existing, nil
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_session_bindings SET adopted_generation = ?, adoption_attempt_id = ?
		 WHERE loop_run_id = ? AND handle = ? AND binding_epoch = 1
		   AND ownership = 'origin-borrowed' AND state = 'active' AND adopted_generation = ?`,
		generation,
		req.BindingAttemptID,
		string(req.Key.LoopRunID),
		req.Key.Handle,
		existing.AdoptedGeneration,
	)
	if err != nil {
		return goal.SessionBinding{}, fmt.Errorf("store: advance origin binding adoption: %w", err)
	}
	if err := requireGoalRowsAffected(result, "advance origin binding adoption"); err != nil {
		return goal.SessionBinding{}, err
	}
	return getSessionBindingAttemptWithExecutor(ctx, exec, req.Key, 1)
}

func reactivateClosedOriginBinding(
	ctx context.Context,
	exec taskSQLExecutor,
	existing goal.SessionBinding,
	req goal.GetOrCreateBindingRequest,
) (goal.SessionBinding, error) {
	if !bindingOriginIdentityMatches(existing, req) {
		return goal.SessionBinding{}, goalBindingMismatchError(
			"origin binding policy/profile or identity differs",
		)
	}
	if existing.State != goal.BindingStateClosed {
		return goal.SessionBinding{}, goalControlStaleError("origin binding is not reusable")
	}
	if existing.AdoptedGeneration >= req.CheckpointKey.Generation {
		return goal.SessionBinding{}, goalControlStaleError("closed origin binding requires a later generation")
	}
	if err := validateBindingSessionIdentity(
		ctx,
		exec,
		req.Key.WorkspaceID,
		req.SessionID,
		req.CreationProfileRef,
		req.PolicySpecDigest,
		req.CreationDigest,
	); err != nil {
		return goal.SessionBinding{}, err
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_session_bindings
		 SET state = 'active', closed_at = NULL, adopted_generation = ?, adoption_attempt_id = ?
		 WHERE loop_run_id = ? AND handle = ? AND binding_epoch = 1
		   AND ownership = 'origin-borrowed' AND state = 'closed' AND adopted_generation = ?`,
		req.CheckpointKey.Generation,
		req.BindingAttemptID,
		string(req.Key.LoopRunID),
		req.Key.Handle,
		existing.AdoptedGeneration,
	)
	if err != nil {
		return goal.SessionBinding{}, fmt.Errorf("store: reactivate origin goal binding: %w", err)
	}
	if err := requireGoalRowsAffected(result, "reactivate origin goal binding"); err != nil {
		return goal.SessionBinding{}, err
	}
	return getSessionBindingAttemptWithExecutor(ctx, exec, req.Key, 1)
}

// PrepareSessionBindingAttempt persists one run-owned creating epoch before session creation.
func (g *GlobalDB) PrepareSessionBindingAttempt(
	ctx context.Context,
	req goal.PrepareBindingAttemptRequest,
) (goal.SessionBinding, error) {
	if err := g.checkReady(ctx, "prepare goal session binding attempt"); err != nil {
		return goal.SessionBinding{}, err
	}
	normalized, err := normalizePrepareBindingRequest(req, g.now())
	if err != nil {
		return goal.SessionBinding{}, err
	}
	var prepared goal.SessionBinding
	err = g.withTaskImmediateTransaction(ctx, "prepare goal session binding attempt", func(exec taskSQLExecutor) error {
		var err error
		prepared, err = prepareSessionBindingAttemptWithExecutor(ctx, exec, normalized)
		return err
	})
	if err != nil {
		return goal.SessionBinding{}, err
	}
	return prepared, nil
}

func prepareSessionBindingAttemptWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.PrepareBindingAttemptRequest,
) (goal.SessionBinding, error) {
	if err := validateBindingRunWorkspace(ctx, exec, req.Key); err != nil {
		return goal.SessionBinding{}, err
	}
	active, found, err := getActiveSessionBindingWithExecutor(ctx, exec, req.Key)
	if err != nil {
		return goal.SessionBinding{}, err
	}
	if found && (active.PolicySpecDigest != req.PolicySpecDigest ||
		active.CreationProfileRef != req.CreationProfileRef) {
		return goal.SessionBinding{}, goalBindingMismatchError(
			"successor policy/profile differs from active binding",
		)
	}
	existing, found, err := findSessionBindingAttemptWithExecutor(ctx, exec, req.Key, req.BindingEpoch)
	if err != nil {
		return goal.SessionBinding{}, err
	}
	if found {
		if !bindingMatchesPrepareRequest(existing, req) {
			return goal.SessionBinding{}, goalBindingMismatchError(
				"binding epoch already has different creation identity",
			)
		}
		return existing, nil
	}
	if err := validateNextBindingAttemptEpoch(ctx, exec, req); err != nil {
		return goal.SessionBinding{}, err
	}
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO loop_session_bindings (
			loop_run_id, handle, binding_epoch, binding_attempt_id, session_id, workspace_id,
			creation_profile_ref, policy_spec_digest, creation_digest, ownership, state, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'run-owned', 'creating', ?)`,
		string(req.Key.LoopRunID),
		req.Key.Handle,
		req.BindingEpoch,
		req.BindingAttemptID,
		req.SessionID,
		string(req.Key.WorkspaceID),
		req.CreationProfileRef,
		req.PolicySpecDigest,
		req.CreationDigest,
		store.FormatTimestamp(req.CreatedAt),
	); err != nil {
		return goal.SessionBinding{}, fmt.Errorf("store: insert creating goal binding: %w", err)
	}
	return getSessionBindingAttemptWithExecutor(ctx, exec, req.Key, req.BindingEpoch)
}

func validateNextBindingAttemptEpoch(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.PrepareBindingAttemptRequest,
) error {
	var maximumEpoch int64
	if err := exec.QueryRowContext(
		ctx,
		`SELECT COALESCE(MAX(binding_epoch), 0)
		 FROM loop_session_bindings WHERE loop_run_id = ? AND handle = ?`,
		string(req.Key.LoopRunID),
		req.Key.Handle,
	).Scan(&maximumEpoch); err != nil {
		return fmt.Errorf("store: load maximum goal binding epoch: %w", err)
	}
	if req.BindingEpoch != maximumEpoch+1 {
		return goalControlStaleError("binding attempt must use the next consecutive epoch")
	}
	var creatingCount int
	if err := exec.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM loop_session_bindings
		 WHERE loop_run_id = ? AND handle = ? AND state = 'creating'`,
		string(req.Key.LoopRunID),
		req.Key.Handle,
	).Scan(&creatingCount); err != nil {
		return fmt.Errorf("store: count creating goal bindings: %w", err)
	}
	if creatingCount != 0 {
		return goalControlStaleError("another binding creation attempt is already pending")
	}
	return nil
}
