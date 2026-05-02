package task

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

// ClaimNextRun atomically claims the next eligible run for one session and returns the raw claim token once.
func (m *Service) ClaimNextRun(
	ctx context.Context,
	criteria ClaimCriteria,
	actor ActorContext,
) (*ClaimResult, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}
	normalized, err := m.normalizeClaimCriteriaForActor(criteria, actor)
	if err != nil {
		return nil, err
	}
	patched, err := m.dispatchTaskRunPreClaimCriteria(ctx, normalized, actor)
	if err != nil {
		return nil, err
	}

	result, err := m.store.ClaimNextRun(ctx, patched)
	if err != nil {
		return nil, err
	}
	claimResultWithoutRawTokenInMetadata(&result)

	reconciledTask, err := m.reconcileTaskCascade(ctx, result.Run.TaskID)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, result.Run.TaskID, result.Run.ID, taskEventRunClaimed, actor, runClaimedPayload{
		Status:         result.Run.Status,
		TaskStatus:     reconciledTask.Status,
		ClaimedBy:      ActorIdentity{Kind: actor.Actor.Kind, Ref: actor.Actor.Ref},
		ClaimTokenHash: result.Run.ClaimTokenHash,
		LeaseUntil:     result.Run.LeaseUntil,
	}); err != nil {
		return nil, err
	}
	if err := m.dispatchTaskRunPostClaim(ctx, result.Run, reconciledTask, actor); err != nil {
		return nil, err
	}
	result.Task = reconciledTask
	return &result, nil
}

// HeartbeatRunLease extends one active task-run lease after token verification.
func (m *Service) HeartbeatRunLease(
	ctx context.Context,
	heartbeat LeaseHeartbeat,
	actor ActorContext,
) (*Run, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}
	normalized, err := heartbeat.Normalize(m.now().UTC())
	if err != nil {
		return nil, err
	}
	run, err := m.store.HeartbeatRunLease(ctx, normalized)
	if err != nil {
		return nil, err
	}
	taskRecord, err := m.store.GetTask(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunLeaseExtended, actor, leaseExtendedPayload{
		Status:         run.Status,
		TaskStatus:     taskRecord.Status,
		LeaseUntil:     run.LeaseUntil,
		HeartbeatAt:    run.HeartbeatAt,
		ClaimTokenHash: run.ClaimTokenHash,
	}); err != nil {
		return nil, err
	}
	if err := m.dispatchTaskRunLeaseExtended(ctx, run, taskRecord, actor); err != nil {
		return nil, err
	}
	return &run, nil
}

// ReleaseRunLease releases one active task-run lease after token verification and requeues the run.
func (m *Service) ReleaseRunLease(
	ctx context.Context,
	release LeaseRelease,
	actor ActorContext,
) (*Run, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}
	normalized, err := release.Normalize(m.now().UTC())
	if err != nil {
		return nil, err
	}
	previous, taskRecord, err := m.loadRunWithTask(ctx, normalized.RunID)
	if err != nil {
		return nil, err
	}
	run, err := m.store.ReleaseRunLease(ctx, normalized)
	if err != nil {
		return nil, err
	}
	reconciledTask, err := m.reconcileTaskCascade(ctx, taskRecord.ID)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunReleased, actor, releasedRunPayload{
		PreviousStatus: previous.Status,
		Status:         run.Status,
		TaskStatus:     reconciledTask.Status,
		Reason:         normalized.Reason,
		SessionID:      previous.SessionID,
	}); err != nil {
		return nil, err
	}
	if err := m.dispatchTaskRunReleased(ctx, run, reconciledTask, actor, previous, normalized.Reason); err != nil {
		return nil, err
	}
	return &run, nil
}

// ReleaseSessionRunLeases structurally releases every active task-run lease
// bound to one session without requiring the raw claim token. This is reserved
// for daemon-owned runtime cleanup paths such as safe-spawn reaping.
func (m *Service) ReleaseSessionRunLeases(
	ctx context.Context,
	release SessionLeaseRelease,
	actor ActorContext,
) ([]SessionLeaseReleaseResult, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}
	normalized, err := release.Normalize(m.now().UTC())
	if err != nil {
		return nil, err
	}
	runs, err := m.activeSessionRunLeases(ctx, normalized.SessionID)
	if err != nil {
		return nil, err
	}

	results := make([]SessionLeaseReleaseResult, 0, len(runs))
	for _, previous := range runs {
		run := requeueSessionRunLease(previous)
		if err := m.store.UpdateTaskRun(ctx, run); err != nil {
			return nil, err
		}
		reconciledTask, err := m.reconcileTaskCascade(ctx, run.TaskID)
		if err != nil {
			return nil, err
		}
		if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunReleased, actor, releasedRunPayload{
			PreviousStatus: previous.Status,
			Status:         run.Status,
			TaskStatus:     reconciledTask.Status,
			Reason:         normalized.Reason,
			SessionID:      previous.SessionID,
		}); err != nil {
			return nil, err
		}
		if err := m.dispatchTaskRunReleased(ctx, run, reconciledTask, actor, previous, normalized.Reason); err != nil {
			return nil, err
		}
		results = append(results, SessionLeaseReleaseResult{
			Run:                    run,
			PreviousRunStatus:      previous.Status,
			PreviousSessionID:      previous.SessionID,
			PreviousLeaseUntil:     previous.LeaseUntil,
			PreviousClaimTokenHash: previous.ClaimTokenHash,
			Reason:                 normalized.Reason,
		})
	}
	return results, nil
}

// CompleteRunLease marks one active task-run lease complete after token verification.
func (m *Service) CompleteRunLease(
	ctx context.Context,
	completion LeaseCompletion,
	actor ActorContext,
) (*Run, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}
	normalized, err := completion.Normalize(m.now().UTC())
	if err != nil {
		return nil, err
	}
	run, err := m.store.CompleteRunLease(ctx, normalized)
	if err != nil {
		return nil, err
	}
	reconciledTask, err := m.reconcileTaskCascade(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunCompleted, actor, completedRunPayload{
		Status:         run.Status,
		TaskStatus:     reconciledTask.Status,
		Result:         cloneRawJSON(run.Result),
		ClaimTokenHash: run.ClaimTokenHash,
	}); err != nil {
		return nil, err
	}
	if err := m.dispatchTaskRunCompleted(ctx, run, reconciledTask, actor); err != nil {
		return nil, err
	}
	return &run, nil
}

// FailRunLease marks one active task-run lease failed after token verification.
func (m *Service) FailRunLease(
	ctx context.Context,
	failure LeaseFailure,
	actor ActorContext,
) (*Run, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}
	normalized, err := failure.Normalize(m.now().UTC())
	if err != nil {
		return nil, err
	}
	run, err := m.store.FailRunLease(ctx, normalized)
	if err != nil {
		return nil, err
	}
	reconciledTask, err := m.reconcileTaskCascade(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunFailed, actor, failedRunPayload{
		Status:         run.Status,
		TaskStatus:     reconciledTask.Status,
		Error:          run.Error,
		Metadata:       cloneRawJSON(normalized.Failure.Metadata),
		ClaimTokenHash: run.ClaimTokenHash,
	}); err != nil {
		return nil, err
	}
	if err := m.dispatchTaskRunFailed(ctx, run, reconciledTask, actor); err != nil {
		return nil, err
	}
	return &run, nil
}

// RecoverExpiredRunLeases requeues stale task-run leases and emits lease-expiration hooks once.
func (m *Service) RecoverExpiredRunLeases(
	ctx context.Context,
	recovery ExpiredLeaseRecovery,
	actor ActorContext,
) ([]ExpiredLeaseRecoveryResult, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}
	normalized, err := recovery.Normalize(m.now().UTC())
	if err != nil {
		return nil, err
	}
	results, err := m.store.RecoverExpiredRunLeases(ctx, normalized)
	if err != nil {
		return nil, err
	}
	for idx := range results {
		result := &results[idx]
		reconciledTask, err := m.reconcileTaskCascade(ctx, result.Run.TaskID)
		if err != nil {
			return nil, err
		}
		if err := m.recordTaskEvent(
			ctx,
			result.Run.TaskID,
			result.Run.ID,
			taskEventRunLeaseExpired,
			actor,
			expiredLeasePayload{
				PreviousStatus:      result.PreviousRunStatus,
				Status:              result.Run.Status,
				TaskStatus:          reconciledTask.Status,
				Reason:              result.Reason,
				SessionID:           result.PreviousSessionID,
				LeaseUntil:          result.PreviousLeaseUntil,
				PreviousTokenHash:   result.PreviousClaimTokenHash,
				CoordinationChannel: result.Run.CoordinationChannelID,
			},
		); err != nil {
			return nil, err
		}
		if err := m.dispatchTaskRunLeaseExpired(
			ctx,
			result.Run,
			reconciledTask,
			actor,
			result,
		); err != nil {
			return nil, err
		}
		if err := m.dispatchTaskRunLeaseRecoveredFromExpiration(
			ctx,
			result.Run,
			reconciledTask,
			actor,
			result,
		); err != nil {
			return nil, err
		}
	}
	return results, nil
}

func (m *Service) activeSessionRunLeases(ctx context.Context, sessionID string) ([]Run, error) {
	statuses := []RunStatus{TaskRunStatusClaimed, TaskRunStatusStarting, TaskRunStatusRunning}
	runs := make([]Run, 0, len(statuses))
	for _, status := range statuses {
		matches, err := m.store.ListTaskRuns(ctx, RunQuery{
			SessionID: sessionID,
			Status:    status,
		})
		if err != nil {
			return nil, err
		}
		for _, run := range matches {
			if strings.TrimSpace(run.SessionID) != strings.TrimSpace(sessionID) ||
				strings.TrimSpace(run.ClaimTokenHash) == "" {
				continue
			}
			runs = append(runs, run)
		}
	}
	return runs, nil
}

// LookupActiveRunForSession resolves the internal claim token for a session-owned
// run while preserving the existing token-fenced lease writers as the sole
// mutation authority.
func (m *Service) LookupActiveRunForSession(
	ctx context.Context,
	sessionID string,
	runID string,
) (AutonomyLeaseHandle, error) {
	normalizedSessionID, normalizedRunID, err := normalizeAutonomyLookupInput(sessionID, runID)
	if err != nil {
		return AutonomyLeaseHandle{}, err
	}
	store, ok := m.store.(AutonomyLeaseStore)
	if !ok {
		return AutonomyLeaseHandle{}, errors.New("task: autonomy lease lookup store is unavailable")
	}
	handles, err := store.ListAutonomyLeaseHandles(ctx, normalizedSessionID)
	if err != nil {
		return AutonomyLeaseHandle{}, err
	}
	return resolveAutonomyLeaseHandle(normalizedSessionID, normalizedRunID, handles, m.now().UTC())
}

func normalizeAutonomyLookupInput(sessionID string, runID string) (string, string, error) {
	normalizedSessionID := strings.TrimSpace(sessionID)
	if normalizedSessionID == "" {
		return "", "", autonomyError(
			AutonomySessionRequired,
			ErrPermissionDenied,
			"agent session identity is required",
		)
	}
	normalizedRunID := strings.TrimSpace(runID)
	if normalizedRunID == "" {
		return "", "", fmt.Errorf("%w: run_id is required", ErrValidation)
	}
	return normalizedSessionID, normalizedRunID, nil
}

func resolveAutonomyLeaseHandle(
	sessionID string,
	runID string,
	handles []AutonomyLeaseHandle,
	now time.Time,
) (AutonomyLeaseHandle, error) {
	target, hasTarget, activeHandles := autonomyLeaseCandidates(handles, sessionID, runID, now)
	if len(activeHandles) > 1 {
		return AutonomyLeaseHandle{}, autonomyError(
			AutonomyLeaseAlreadyHeld,
			ErrActiveRunLease,
			"session %q owns multiple active task-run leases",
			sessionID,
		)
	}
	if !hasTarget {
		return missingAutonomyLeaseError(sessionID, runID, activeHandles)
	}
	if !isActiveAutonomyLeaseHandle(target, sessionID, now) {
		return AutonomyLeaseHandle{}, autonomyError(
			AutonomyLeaseExpired,
			ErrLeaseExpired,
			"run %q is not an active lease for session %q",
			runID,
			sessionID,
		)
	}
	if len(activeHandles) == 1 && activeHandles[0].RunID != runID {
		return AutonomyLeaseHandle{}, autonomyError(
			AutonomyForeignRun,
			ErrPermissionDenied,
			"run %q is not owned by session %q",
			runID,
			sessionID,
		)
	}
	return target, nil
}

func autonomyLeaseCandidates(
	handles []AutonomyLeaseHandle,
	sessionID string,
	runID string,
	now time.Time,
) (AutonomyLeaseHandle, bool, []AutonomyLeaseHandle) {
	var target AutonomyLeaseHandle
	hasTarget := false
	activeHandles := make([]AutonomyLeaseHandle, 0, len(handles))
	for idx := range handles {
		handle := normalizeAutonomyLeaseHandle(handles[idx])
		if handle.RunID == runID {
			target = handle
			hasTarget = true
		}
		if isActiveAutonomyLeaseHandle(handle, sessionID, now) {
			activeHandles = append(activeHandles, handle)
		}
	}
	return target, hasTarget, activeHandles
}

func normalizeAutonomyLeaseHandle(handle AutonomyLeaseHandle) AutonomyLeaseHandle {
	handle.SessionID = strings.TrimSpace(handle.SessionID)
	handle.RunID = strings.TrimSpace(handle.RunID)
	handle.TaskID = strings.TrimSpace(handle.TaskID)
	handle.WorkspaceID = strings.TrimSpace(handle.WorkspaceID)
	handle.ClaimToken = strings.TrimSpace(handle.ClaimToken)
	handle.ClaimTokenHash = strings.TrimSpace(handle.ClaimTokenHash)
	return handle
}

func isActiveAutonomyLeaseHandle(handle AutonomyLeaseHandle, sessionID string, now time.Time) bool {
	return handle.SessionID == sessionID &&
		isAutonomyLeaseStatusActive(handle.Status) &&
		!handle.LeaseUntil.IsZero() &&
		handle.LeaseUntil.After(now) &&
		handle.ClaimToken != "" &&
		handle.ClaimTokenHash != ""
}

func missingAutonomyLeaseError(
	sessionID string,
	runID string,
	activeHandles []AutonomyLeaseHandle,
) (AutonomyLeaseHandle, error) {
	if len(activeHandles) == 1 {
		return AutonomyLeaseHandle{}, autonomyError(
			AutonomyForeignRun,
			ErrPermissionDenied,
			"run %q is not owned by session %q",
			runID,
			sessionID,
		)
	}
	return AutonomyLeaseHandle{}, autonomyError(
		AutonomyNoActiveLease,
		ErrInvalidClaimToken,
		"session %q has no active task-run lease",
		sessionID,
	)
}

func requeueSessionRunLease(run Run) Run {
	run.Status = TaskRunStatusQueued
	run.ClaimedBy = nil
	run.ClaimedAt = time.Time{}
	run.SessionID = ""
	run.ClaimToken = ""
	run.ClaimTokenHash = ""
	run.LeaseUntil = time.Time{}
	run.HeartbeatAt = time.Time{}
	run.StartedAt = time.Time{}
	run.EndedAt = time.Time{}
	run.Error = ""
	run.Result = nil
	return run
}

func (m *Service) normalizeClaimCriteriaForActor(
	criteria ClaimCriteria,
	actor ActorContext,
) (ClaimCriteria, error) {
	normalized := criteria
	if strings.TrimSpace(normalized.ClaimerSessionID) == "" && actor.Actor.Kind.Normalize() == ActorKindAgentSession {
		normalized.ClaimerSessionID = strings.TrimSpace(actor.Actor.Ref)
	}
	if normalized.ClaimedBy == nil {
		normalized.ClaimedBy = &ActorIdentity{
			Kind: actor.Actor.Kind.Normalize(),
			Ref:  strings.TrimSpace(actor.Actor.Ref),
		}
	}
	if strings.TrimSpace(normalized.AgentName) == "" && actor.Actor.Kind.Normalize() == ActorKindAgentSession {
		normalized.AgentName = strings.TrimSpace(actor.Actor.Ref)
	}
	return normalized.Normalize(m.now().UTC())
}

func (m *Service) dispatchTaskRunPreClaimCriteria(
	ctx context.Context,
	criteria ClaimCriteria,
	actor ActorContext,
) (ClaimCriteria, error) {
	taskContext := hookspkg.TaskRunContext{
		WorkspaceID:           strings.TrimSpace(criteria.WorkspaceID),
		CoordinationChannelID: strings.TrimSpace(criteria.CoordinationChannelID),
		AgentName:             strings.TrimSpace(criteria.AgentName),
		SessionID:             strings.TrimSpace(criteria.ClaimerSessionID),
		ActorKind:             string(actor.Actor.Kind.Normalize()),
		ActorRef:              strings.TrimSpace(actor.Actor.Ref),
	}
	if criteria.Soul != nil {
		taskContext.SoulSnapshotID = strings.TrimSpace(criteria.Soul.SnapshotID)
		taskContext.SoulDigest = strings.TrimSpace(criteria.Soul.Digest)
	}
	payload := hookspkg.TaskRunPreClaimPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskRunPreClaim,
			Timestamp: m.now().UTC(),
		},
		TaskRunContext: taskContext,
		Criteria: hookspkg.TaskRunClaimCriteria{
			WorkspaceID:           criteria.WorkspaceID,
			ClaimerSessionID:      criteria.ClaimerSessionID,
			AgentName:             criteria.AgentName,
			RequiredCapabilities:  append([]string(nil), criteria.RequiredCapabilities...),
			PriorityMin:           criteria.PriorityMin,
			CoordinationChannelID: criteria.CoordinationChannelID,
		},
	}
	result, err := m.taskHooks.DispatchTaskRunPreClaim(ctx, payload)
	if err != nil {
		return ClaimCriteria{}, err
	}
	if result.Denied {
		reason := strings.TrimSpace(result.DenyReason)
		if reason == "" {
			reason = "task run claim denied by hook"
		}
		return ClaimCriteria{}, fmt.Errorf("%w: %s", ErrPermissionDenied, reason)
	}
	patched := criteria
	patched.RequiredCapabilities = append([]string(nil), result.Criteria.RequiredCapabilities...)
	patched.PriorityMin = result.Criteria.PriorityMin
	if strings.TrimSpace(result.Criteria.CoordinationChannelID) != "" {
		patched.CoordinationChannelID = strings.TrimSpace(result.Criteria.CoordinationChannelID)
	}
	return patched.Normalize(m.now().UTC())
}
