package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
)

func normalizeOriginBindingRequest(
	req goal.GetOrCreateBindingRequest,
	now time.Time,
) (goal.GetOrCreateBindingRequest, error) {
	if err := req.Key.Validate(); err != nil {
		return goal.GetOrCreateBindingRequest{}, err
	}
	if err := req.CheckpointKey.Validate(); err != nil {
		return goal.GetOrCreateBindingRequest{}, err
	}
	req = normalizeOriginBindingStrings(req)
	if err := validateOriginBindingRequest(req); err != nil {
		return goal.GetOrCreateBindingRequest{}, err
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = now
	}
	return req, nil
}

func normalizeOriginBindingStrings(req goal.GetOrCreateBindingRequest) goal.GetOrCreateBindingRequest {
	req.BindingAttemptID = strings.TrimSpace(req.BindingAttemptID)
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.CreationProfileRef = strings.TrimSpace(req.CreationProfileRef)
	req.PolicySpecDigest = strings.TrimSpace(req.PolicySpecDigest)
	req.CreationDigest = strings.TrimSpace(req.CreationDigest)
	req.ExpectedCheckpointPhase = strings.TrimSpace(req.ExpectedCheckpointPhase)
	req.ExpectedTaskRunID = strings.TrimSpace(req.ExpectedTaskRunID)
	req.ExpectedQueueEntryID = strings.TrimSpace(req.ExpectedQueueEntryID)
	req.ExpectedPromptID = strings.TrimSpace(req.ExpectedPromptID)
	req.ExpectedCheckpointSessionID = strings.TrimSpace(req.ExpectedCheckpointSessionID)
	req.ExpectedCheckpointHandle = strings.TrimSpace(req.ExpectedCheckpointHandle)
	return req
}

func validateOriginBindingRequest(req goal.GetOrCreateBindingRequest) error {
	if req.Key.WorkspaceID != req.CheckpointKey.WorkspaceID ||
		req.Key.LoopRunID != req.CheckpointKey.LoopRunID {
		return fmt.Errorf(
			"%w: origin binding checkpoint does not own the binding Run",
			loop.ErrValidation,
		)
	}
	checkpointBindingEmpty := req.ExpectedCheckpointBindingEpoch == 0 &&
		req.ExpectedCheckpointSessionID == "" && req.ExpectedCheckpointHandle == ""
	checkpointBindingComplete := req.ExpectedCheckpointBindingEpoch > 0 &&
		req.ExpectedCheckpointSessionID != "" && req.ExpectedCheckpointHandle != ""
	if req.BindingAttemptID == "" || req.SessionID == "" || req.CreationProfileRef == "" ||
		req.PolicySpecDigest == "" || req.CreationDigest == "" || req.ExpectedControlEpoch < 1 ||
		req.CheckpointKey.Generation < 1 || !goalCheckpointPhaseValid(req.ExpectedCheckpointPhase) ||
		req.ExpectedCheckpointPhase == goalCheckpointPhaseTerminal ||
		req.ExpectedTaskRunID == "" || (!checkpointBindingEmpty && !checkpointBindingComplete) {
		return fmt.Errorf(
			"%w: origin binding identity fields are required",
			loop.ErrValidation,
		)
	}
	if req.Ownership != goal.BindingOwnershipOriginBorrowed {
		return fmt.Errorf(
			"%w: get-or-create requires origin-borrowed ownership",
			loop.ErrValidation,
		)
	}
	return nil
}

func normalizePrepareBindingRequest(
	req goal.PrepareBindingAttemptRequest,
	now time.Time,
) (goal.PrepareBindingAttemptRequest, error) {
	if err := req.Key.Validate(); err != nil {
		return goal.PrepareBindingAttemptRequest{}, err
	}
	req.BindingAttemptID = strings.TrimSpace(req.BindingAttemptID)
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.CreationProfileRef = strings.TrimSpace(req.CreationProfileRef)
	req.PolicySpecDigest = strings.TrimSpace(req.PolicySpecDigest)
	req.CreationDigest = strings.TrimSpace(req.CreationDigest)
	if req.BindingEpoch < 1 || req.BindingAttemptID == "" || req.SessionID == "" ||
		req.CreationProfileRef == "" || req.PolicySpecDigest == "" || req.CreationDigest == "" {
		return goal.PrepareBindingAttemptRequest{}, fmt.Errorf(
			"%w: complete run-owned binding identity is required",
			loop.ErrValidation,
		)
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = now
	}
	return req, nil
}

func bindingOriginIdentityMatches(
	binding goal.SessionBinding,
	req goal.GetOrCreateBindingRequest,
) bool {
	return binding.BindingEpoch == 1 &&
		binding.Ownership == goal.BindingOwnershipOriginBorrowed &&
		binding.SessionID == req.SessionID &&
		binding.CreationProfileRef == req.CreationProfileRef &&
		binding.PolicySpecDigest == req.PolicySpecDigest &&
		binding.CreationDigest == req.CreationDigest
}

func bindingMatchesPrepareRequest(
	binding goal.SessionBinding,
	req goal.PrepareBindingAttemptRequest,
) bool {
	return binding.State == goal.BindingStateCreating &&
		binding.Ownership == goal.BindingOwnershipRunOwned &&
		binding.BindingAttemptID == req.BindingAttemptID &&
		binding.SessionID == req.SessionID &&
		binding.CreationProfileRef == req.CreationProfileRef &&
		binding.PolicySpecDigest == req.PolicySpecDigest &&
		binding.CreationDigest == req.CreationDigest
}

func validateBindingSessionIdentity(
	ctx context.Context,
	exec taskSQLExecutor,
	workspaceID loop.WorkspaceID,
	sessionID string,
	profileRef string,
	policyDigest string,
	creationDigest string,
) error {
	storedWorkspace, identity, found, err := readSessionCreationIdentity(ctx, exec, sessionID)
	if err != nil {
		return err
	}
	if !found || strings.TrimSpace(storedWorkspace) != strings.TrimSpace(string(workspaceID)) ||
		identity.CreationProfileRef != strings.TrimSpace(profileRef) ||
		identity.PolicySpecDigest != strings.TrimSpace(policyDigest) ||
		identity.CreationDigest != strings.TrimSpace(creationDigest) {
		return sessionCreationIdentityMismatch(sessionID)
	}
	return nil
}

func getActiveSessionBindingWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	key goal.BindingKey,
) (goal.SessionBinding, bool, error) {
	binding, err := scanGoalSessionBinding(exec.QueryRowContext(
		ctx,
		`SELECT `+goalBindingSelectColumns+`
		 FROM loop_session_bindings
		 WHERE loop_run_id = ? AND workspace_id = ? AND handle = ? AND state = 'active'`,
		string(key.LoopRunID),
		string(key.WorkspaceID),
		key.Handle,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return goal.SessionBinding{}, false, nil
	}
	if err != nil {
		return goal.SessionBinding{}, false, err
	}
	return binding, true, nil
}

func findSessionBindingAttemptWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	key goal.BindingKey,
	epoch int64,
) (goal.SessionBinding, bool, error) {
	binding, err := scanGoalSessionBinding(exec.QueryRowContext(
		ctx,
		`SELECT `+goalBindingSelectColumns+`
		 FROM loop_session_bindings
		 WHERE loop_run_id = ? AND workspace_id = ? AND handle = ? AND binding_epoch = ?`,
		string(key.LoopRunID),
		string(key.WorkspaceID),
		key.Handle,
		epoch,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return goal.SessionBinding{}, false, nil
	}
	if err != nil {
		return goal.SessionBinding{}, false, err
	}
	return binding, true, nil
}

func getSessionBindingAttemptWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	key goal.BindingKey,
	epoch int64,
) (goal.SessionBinding, error) {
	binding, found, err := findSessionBindingAttemptWithExecutor(ctx, exec, key, epoch)
	if err != nil {
		return goal.SessionBinding{}, err
	}
	if !found {
		return goal.SessionBinding{}, fmt.Errorf("%w: %s epoch %d", goal.ErrBindingNotFound, key.Handle, epoch)
	}
	return binding, nil
}

func scanGoalSessionBinding(scanner rowScanner) (goal.SessionBinding, error) {
	var (
		binding         goal.SessionBinding
		workspaceID     string
		failureCode     sql.NullString
		createdAtRaw    any
		activatedAtRaw  any
		failedAtRaw     any
		closedAtRaw     any
		adoptionAttempt sql.NullString
	)
	if err := scanner.Scan(
		&binding.Key.LoopRunID,
		&binding.Key.Handle,
		&binding.BindingEpoch,
		&binding.BindingAttemptID,
		&binding.SessionID,
		&workspaceID,
		&binding.CreationProfileRef,
		&binding.PolicySpecDigest,
		&binding.CreationDigest,
		&binding.Ownership,
		&binding.State,
		&binding.AdoptedGeneration,
		&adoptionAttempt,
		&failureCode,
		&createdAtRaw,
		&activatedAtRaw,
		&failedAtRaw,
		&closedAtRaw,
	); err != nil {
		return goal.SessionBinding{}, err
	}
	binding.Key.WorkspaceID = loop.WorkspaceID(workspaceID)
	binding.FailureCode = strings.TrimSpace(failureCode.String)
	binding.AdoptionAttemptID = strings.TrimSpace(adoptionAttempt.String)
	createdAt, err := parseGoalTimestampValue(createdAtRaw, "binding created_at")
	if err != nil {
		return goal.SessionBinding{}, err
	}
	binding.CreatedAt = createdAt
	if binding.ActivatedAt, err = parseOptionalGoalTimestampValue(activatedAtRaw, "binding activated_at"); err != nil {
		return goal.SessionBinding{}, err
	}
	if binding.FailedAt, err = parseOptionalGoalTimestampValue(failedAtRaw, "binding failed_at"); err != nil {
		return goal.SessionBinding{}, err
	}
	if binding.ClosedAt, err = parseOptionalGoalTimestampValue(closedAtRaw, "binding closed_at"); err != nil {
		return goal.SessionBinding{}, err
	}
	return binding, nil
}

func requireGoalRowsAffected(result sql.Result, action string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for %s: %w", action, err)
	}
	if affected != 1 {
		return goalControlStaleError(action + " lost compare-and-swap")
	}
	return nil
}
