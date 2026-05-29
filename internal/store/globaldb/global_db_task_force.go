package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

type retryTaskRunArgs struct {
	sourceRunID string
	newRunID    string
	origin      taskpkg.Origin
	metadata    json.RawMessage
	queuedAt    time.Time
	reason      string
}

// ForceReleaseTaskRun requeues one claimed run with snapshot fencing.
func (g *GlobalDB) ForceReleaseTaskRun(
	ctx context.Context,
	release taskpkg.ForceReleaseRunMutation,
) (taskpkg.ForceRunMutationResult, error) {
	if err := g.checkReady(ctx, "force release task run"); err != nil {
		return taskpkg.ForceRunMutationResult{}, err
	}
	runID, err := requireTaskValue(release.RunID, "task run id")
	if err != nil {
		return taskpkg.ForceRunMutationResult{}, err
	}

	var result taskpkg.ForceRunMutationResult
	if err := g.withTaskImmediateTransaction(ctx, "force release task run", func(exec taskSQLExecutor) error {
		previous, err := g.getTaskRunWithExecutor(ctx, exec, runID)
		if err != nil {
			return err
		}
		if previous.Status.Normalize() != taskpkg.TaskRunStatusClaimed {
			return fmt.Errorf(
				"%w: task run %q is %s; only claimed runs can be force released",
				taskpkg.ErrInvalidStatusTransition,
				previous.ID,
				previous.Status.Normalize(),
			)
		}
		next := forceReleasedTaskRun(previous)
		if err := updateTaskRunRecordWithSnapshotCAS(ctx, exec, previous, next); err != nil {
			return err
		}
		if err := updateTaskCurrentRunProjectionForRunUpdate(ctx, exec, previous, next); err != nil {
			return err
		}
		result = taskpkg.ForceRunMutationResult{Previous: previous, Run: next}
		return nil
	}); err != nil {
		return taskpkg.ForceRunMutationResult{}, err
	}
	return result, nil
}

// ForceFailTaskRun marks one queued or claimed run as operator-forced failed with snapshot fencing.
func (g *GlobalDB) ForceFailTaskRun(
	ctx context.Context,
	failure taskpkg.ForceFailRunMutation,
) (taskpkg.ForceRunMutationResult, error) {
	if err := g.checkReady(ctx, "force fail task run"); err != nil {
		return taskpkg.ForceRunMutationResult{}, err
	}
	runID, err := requireTaskValue(failure.RunID, "task run id")
	if err != nil {
		return taskpkg.ForceRunMutationResult{}, err
	}
	reason := strings.TrimSpace(failure.Reason)
	if reason == "" {
		return taskpkg.ForceRunMutationResult{}, fmt.Errorf("%w: force fail reason is required", taskpkg.ErrValidation)
	}
	now := normalizedForceRunTime(failure.Now, g.now)

	var result taskpkg.ForceRunMutationResult
	if err := g.withTaskImmediateTransaction(ctx, "force fail task run", func(exec taskSQLExecutor) error {
		previous, err := g.getTaskRunWithExecutor(ctx, exec, runID)
		if err != nil {
			return err
		}
		if !forceFailTaskRunStatusAllowed(previous.Status) {
			return fmt.Errorf(
				"%w: task run %q is %s; only queued or claimed runs can be force failed",
				taskpkg.ErrInvalidStatusTransition,
				previous.ID,
				previous.Status.Normalize(),
			)
		}
		next := forceFailedTaskRun(previous, reason, now)
		if err := updateTaskRunRecordWithSnapshotCAS(ctx, exec, previous, next); err != nil {
			return err
		}
		if err := updateTaskCurrentRunProjectionForRunUpdate(ctx, exec, previous, next); err != nil {
			return err
		}
		result = taskpkg.ForceRunMutationResult{Previous: previous, Run: next}
		return nil
	}); err != nil {
		return taskpkg.ForceRunMutationResult{}, err
	}
	return result, nil
}

// RetryTaskRun creates one queued retry run linked to a failed source run.
func (g *GlobalDB) RetryTaskRun(
	ctx context.Context,
	retry taskpkg.RetryRunMutation,
) (taskpkg.RetryRunResult, error) {
	if err := g.checkReady(ctx, "retry task run"); err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	args, err := normalizeRetryTaskRunArgs(retry, g.now)
	if err != nil {
		return taskpkg.RetryRunResult{}, err
	}

	var result taskpkg.RetryRunResult
	if err := g.withTaskImmediateTransaction(ctx, "retry task run", func(exec taskSQLExecutor) error {
		created, err := g.retryTaskRunWithExecutor(ctx, exec, args)
		if err == nil {
			result = created
		}
		return err
	}); err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	return result, nil
}

func normalizeRetryTaskRunArgs(
	retry taskpkg.RetryRunMutation,
	now func() time.Time,
) (retryTaskRunArgs, error) {
	sourceRunID, err := requireTaskValue(retry.SourceRunID, "source task run id")
	if err != nil {
		return retryTaskRunArgs{}, err
	}
	newRunID, err := requireTaskValue(retry.NewRunID, "new task run id")
	if err != nil {
		return retryTaskRunArgs{}, err
	}
	origin := taskpkg.Origin{Kind: retry.Origin.Kind.Normalize(), Ref: strings.TrimSpace(retry.Origin.Ref)}
	if err := origin.Validate("retry_run.origin"); err != nil {
		return retryTaskRunArgs{}, err
	}
	metadata := normalizeTaskJSON(retry.Metadata)
	if err := taskpkg.ValidateMetadataSize(metadata, "retry_run.metadata"); err != nil {
		return retryTaskRunArgs{}, err
	}
	return retryTaskRunArgs{
		sourceRunID: sourceRunID,
		newRunID:    newRunID,
		origin:      origin,
		metadata:    metadata,
		queuedAt:    normalizedForceRunTime(retry.QueuedAt, now),
	}, nil
}

func (g *GlobalDB) retryTaskRunWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	args retryTaskRunArgs,
) (taskpkg.RetryRunResult, error) {
	source, err := g.retryTaskRunSource(ctx, exec, args.sourceRunID)
	if err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	taskRecord, err := g.retryTaskRunTask(ctx, exec, source.TaskID)
	if err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	nextAttempt, err := nextTaskRunAttemptWithExecutor(ctx, exec, taskRecord)
	if err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	return g.insertRetryTaskRun(ctx, exec, args, source, taskRecord, nextAttempt)
}

func (g *GlobalDB) retryTaskRunSource(
	ctx context.Context,
	exec taskSQLExecutor,
	sourceRunID string,
) (taskpkg.Run, error) {
	source, err := g.getTaskRunWithExecutor(ctx, exec, sourceRunID)
	if err != nil {
		return taskpkg.Run{}, err
	}
	if source.Status.Normalize() != taskpkg.TaskRunStatusFailed {
		return taskpkg.Run{}, fmt.Errorf(
			"%w: task run %q is %s; only failed runs can be retried",
			taskpkg.ErrInvalidStatusTransition,
			source.ID,
			source.Status.Normalize(),
		)
	}
	if err := requireRetryDepthWithExecutor(ctx, exec, source); err != nil {
		return taskpkg.Run{}, err
	}
	if err := requireNoRetryChildWithExecutor(ctx, exec, source.ID); err != nil {
		return taskpkg.Run{}, err
	}
	return source, nil
}

func (g *GlobalDB) retryTaskRunTask(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
) (taskpkg.Task, error) {
	taskRecord, err := g.getTaskWithExecutor(ctx, exec, taskID)
	if err != nil {
		return taskpkg.Task{}, err
	}
	if err := validateTaskForQueuedRunReservation(taskRecord); err != nil {
		return taskpkg.Task{}, err
	}
	openRunID, err := g.findOpenRunIDForQueuedRunReservation(ctx, exec, taskRecord.ID)
	if err != nil {
		return taskpkg.Task{}, err
	}
	if openRunID != "" {
		return taskpkg.Task{}, fmt.Errorf(
			"%w: task %q has open run %q; finish or cancel it before retrying another run",
			taskpkg.ErrInvalidStatusTransition,
			taskRecord.ID,
			openRunID,
		)
	}
	return taskRecord, nil
}

func (g *GlobalDB) insertRetryTaskRun(
	ctx context.Context,
	exec taskSQLExecutor,
	args retryTaskRunArgs,
	source taskpkg.Run,
	taskRecord taskpkg.Task,
	nextAttempt int,
) (taskpkg.RetryRunResult, error) {
	networkChannel := resolveStoredRunChannel(source.NetworkChannel, taskRecord.NetworkChannel)
	coordinationChannelID := coordinationChannelIDForQueuedRun(taskRecord, networkChannel, args.newRunID)
	if err := ensureQueuedRunCoordinationChannel(
		ctx,
		exec,
		taskRecord,
		coordinationChannelID,
		args.origin,
		args.queuedAt,
	); err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	run := taskpkg.Run{
		ID:                    args.newRunID,
		TaskID:                taskRecord.ID,
		Status:                taskpkg.TaskRunStatusQueued,
		Attempt:               nextAttempt,
		PreviousRunID:         source.ID,
		Origin:                args.origin,
		NetworkChannel:        networkChannel,
		CoordinationChannelID: coordinationChannelID,
		Metadata:              args.metadata,
		QueuedAt:              args.queuedAt,
	}
	normalizedRun, err := g.normalizeTaskRunForCreate(run)
	if err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	if err := insertQueuedTaskRun(ctx, exec, normalizedRun); err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	return taskpkg.RetryRunResult{PreviousRun: source, Run: normalizedRun}, nil
}

// RecoverTaskRun terminalizes a needs_attention run as failed and queues one fresh child in
// the same transaction, so the source leaves the open-run set before the requeue reservation.
func (g *GlobalDB) RecoverTaskRun(
	ctx context.Context,
	mutation taskpkg.RecoverRunMutation,
) (taskpkg.RetryRunResult, error) {
	if err := g.checkReady(ctx, "recover task run"); err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	args, err := normalizeRecoverTaskRunArgs(mutation, g.now)
	if err != nil {
		return taskpkg.RetryRunResult{}, err
	}

	var result taskpkg.RetryRunResult
	if err := g.withTaskImmediateTransaction(ctx, "recover task run", func(exec taskSQLExecutor) error {
		created, err := g.recoverTaskRunWithExecutor(ctx, exec, args)
		if err == nil {
			result = created
		}
		return err
	}); err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	return result, nil
}

func normalizeRecoverTaskRunArgs(
	mutation taskpkg.RecoverRunMutation,
	now func() time.Time,
) (retryTaskRunArgs, error) {
	sourceRunID, err := requireTaskValue(mutation.SourceRunID, "source task run id")
	if err != nil {
		return retryTaskRunArgs{}, err
	}
	newRunID, err := requireTaskValue(mutation.NewRunID, "new task run id")
	if err != nil {
		return retryTaskRunArgs{}, err
	}
	origin := taskpkg.Origin{Kind: mutation.Origin.Kind.Normalize(), Ref: strings.TrimSpace(mutation.Origin.Ref)}
	if err := origin.Validate("recover_run.origin"); err != nil {
		return retryTaskRunArgs{}, err
	}
	metadata := normalizeTaskJSON(mutation.Metadata)
	if err := taskpkg.ValidateMetadataSize(metadata, "recover_run.metadata"); err != nil {
		return retryTaskRunArgs{}, err
	}
	return retryTaskRunArgs{
		sourceRunID: sourceRunID,
		newRunID:    newRunID,
		origin:      origin,
		metadata:    metadata,
		queuedAt:    normalizedForceRunTime(mutation.QueuedAt, now),
		reason:      strings.TrimSpace(mutation.Reason),
	}, nil
}

func (g *GlobalDB) recoverTaskRunWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	args retryTaskRunArgs,
) (taskpkg.RetryRunResult, error) {
	source, err := g.getTaskRunWithExecutor(ctx, exec, args.sourceRunID)
	if err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	if source.Status.Normalize() != taskpkg.TaskRunStatusNeedsAttention {
		return taskpkg.RetryRunResult{}, fmt.Errorf(
			"%w: task run %q is %s; only needs_attention runs can be recovered",
			taskpkg.ErrInvalidStatusTransition,
			source.ID,
			source.Status.Normalize(),
		)
	}
	if err := requireRetryDepthWithExecutor(ctx, exec, source); err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	if err := requireNoRetryChildWithExecutor(ctx, exec, source.ID); err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	failed := forceFailedTaskRun(source, args.reason, args.queuedAt)
	if err := updateTaskRunRecordWithSnapshotCAS(ctx, exec, source, failed); err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	taskRecord, err := g.retryTaskRunTask(ctx, exec, failed.TaskID)
	if err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	nextAttempt, err := nextTaskRunAttemptWithExecutor(ctx, exec, taskRecord)
	if err != nil {
		return taskpkg.RetryRunResult{}, err
	}
	return g.insertRetryTaskRun(ctx, exec, args, failed, taskRecord, nextAttempt)
}

// MarkTaskRunNeedsAttention transitions one queued run to needs_attention via a status CAS.
func (g *GlobalDB) MarkTaskRunNeedsAttention(
	ctx context.Context,
	runID string,
	diagnostic string,
) (taskpkg.Run, error) {
	if err := g.checkReady(ctx, "mark task run needs attention"); err != nil {
		return taskpkg.Run{}, err
	}
	id, err := requireTaskValue(runID, "task run id")
	if err != nil {
		return taskpkg.Run{}, err
	}
	var run taskpkg.Run
	if err := g.withTaskImmediateTransaction(
		ctx,
		"mark task run needs attention",
		func(exec taskSQLExecutor) error {
			result, err := exec.ExecContext(
				ctx,
				`UPDATE task_runs SET status = 'needs_attention', error = ? WHERE id = ? AND status = 'queued'`,
				strings.TrimSpace(diagnostic),
				id,
			)
			if err != nil {
				return fmt.Errorf("store: mark task run needs attention: %w", err)
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("store: mark task run needs attention rows: %w", err)
			}
			if affected == 0 {
				return fmt.Errorf("%w: task run %q is not queued", taskpkg.ErrInvalidStatusTransition, id)
			}
			updated, err := g.getTaskRunWithExecutor(ctx, exec, id)
			if err != nil {
				return err
			}
			run = updated
			return nil
		},
	); err != nil {
		return taskpkg.Run{}, err
	}
	return run, nil
}

func forceReleasedTaskRun(previous taskpkg.Run) taskpkg.Run {
	next := previous
	next.Status = taskpkg.TaskRunStatusQueued
	next.ClaimedBy = nil
	next.ClaimedAt = time.Time{}
	next.SessionID = ""
	next.ClaimToken = ""
	next.ClaimTokenHash = ""
	next.LeaseUntil = time.Time{}
	next.HeartbeatAt = time.Time{}
	next.StartedAt = time.Time{}
	next.EndedAt = time.Time{}
	next.Error = ""
	next.FailureKind = ""
	next.Result = nil
	return next
}

func forceFailedTaskRun(previous taskpkg.Run, reason string, now time.Time) taskpkg.Run {
	next := previous
	next.Status = taskpkg.TaskRunStatusFailed
	next.Error = strings.TrimSpace(reason)
	next.FailureKind = taskpkg.FailureKindOperatorForced
	next.Result = nil
	next.ClaimToken = ""
	next.ClaimTokenHash = ""
	next.LeaseUntil = time.Time{}
	next.HeartbeatAt = time.Time{}
	next.EndedAt = now
	return next
}

func forceFailTaskRunStatusAllowed(status taskpkg.RunStatus) bool {
	switch status.Normalize() {
	case taskpkg.TaskRunStatusQueued, taskpkg.TaskRunStatusClaimed:
		return true
	default:
		return false
	}
}

func updateTaskRunRecordWithSnapshotCAS(
	ctx context.Context,
	exec taskSQLExecutor,
	previous taskpkg.Run,
	next taskpkg.Run,
) error {
	lineage := taskRunReviewLineage(next)
	result, err := exec.ExecContext(
		ctx,
		`UPDATE task_runs
		 SET task_id = ?, status = ?, attempt = ?, previous_run_id = ?, failure_kind = ?, claimed_by_kind = ?,
		     claimed_by_ref = ?, session_id = ?, origin_kind = ?,
		     origin_ref = ?, idempotency_key = ?, network_channel = ?,
		     claim_token = ?, claim_token_hash = ?, lease_until = ?,
		     heartbeat_at = ?, coordination_channel_id = ?, queued_at = ?,
		     claimed_at = ?, started_at = ?, ended_at = ?, error = ?,
		     metadata_json = ?, result_json = ?, review_required = ?,
		     review_request_round = ?, review_policy_snapshot = ?, review_request_id = ?,
		     parent_run_id = ?, review_id = ?, review_round = ?, continuation_reason = ?,
		     missing_work_json = ?, next_round_guidance = ?
		 WHERE id = ?
		   AND status = ?
		   AND COALESCE(session_id, '') = ?
		   AND COALESCE(claim_token_hash, '') = ?
		   AND COALESCE(lease_until, '') = ?`,
		next.TaskID,
		string(next.Status),
		next.Attempt,
		store.NullableString(next.PreviousRunID),
		strings.TrimSpace(next.FailureKind),
		taskActorKindValue(next.ClaimedBy),
		taskActorRefValue(next.ClaimedBy),
		store.NullableString(next.SessionID),
		string(next.Origin.Kind),
		next.Origin.Ref,
		store.NullableString(next.IdempotencyKey),
		store.NullableString(next.NetworkChannel),
		nil,
		store.NullableString(next.ClaimTokenHash),
		nullableTaskTimestamp(next.LeaseUntil),
		nullableTaskTimestamp(next.HeartbeatAt),
		store.NullableString(next.CoordinationChannelID),
		store.FormatTimestamp(next.QueuedAt),
		nullableTaskTimestamp(next.ClaimedAt),
		nullableTaskTimestamp(next.StartedAt),
		nullableTaskTimestamp(next.EndedAt),
		store.NullableString(next.Error),
		nullableTaskJSON(next.Metadata),
		nullableTaskJSON(next.Result),
		lineage.Required,
		lineage.RequestRound,
		string(lineage.PolicySnapshot),
		store.NullableString(lineage.RequestID),
		store.NullableString(lineage.ParentRunID),
		store.NullableString(lineage.ReviewID),
		lineage.ReviewRound,
		lineage.ContinuationReason,
		string(lineage.MissingWork),
		lineage.NextRoundGuidance,
		next.ID,
		string(previous.Status.Normalize()),
		strings.TrimSpace(previous.SessionID),
		strings.TrimSpace(previous.ClaimTokenHash),
		forceRunCASTimestamp(previous.LeaseUntil),
	)
	if err != nil {
		return fmt.Errorf("store: force update task run %q: %w", next.ID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for force update task run %q: %w", next.ID, err)
	}
	if rowsAffected > 0 {
		return nil
	}
	return forceRunCASMiss(ctx, exec, next.ID)
}

func forceRunCASMiss(ctx context.Context, exec taskSQLExecutor, runID string) error {
	var existing string
	if err := exec.QueryRowContext(ctx, `SELECT id FROM task_runs WHERE id = ?`, runID).Scan(&existing); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.ErrTaskRunNotFound
		}
		return fmt.Errorf("store: inspect force update CAS miss for task run %q: %w", runID, err)
	}
	return fmt.Errorf(
		"%w: task run %q changed before force operation applied",
		taskpkg.ErrInvalidStatusTransition,
		runID,
	)
}

func requireRetryDepthWithExecutor(ctx context.Context, exec taskSQLExecutor, source taskpkg.Run) error {
	byID, err := taskRunsByIDForTaskWithExecutor(ctx, exec, source.TaskID)
	if err != nil {
		return err
	}
	depth := 0
	for current := source; strings.TrimSpace(current.PreviousRunID) != ""; {
		depth++
		if depth >= taskpkg.MaxRetryRunChainDepth {
			return taskpkg.ErrRetryChainTooDeep
		}
		previous, ok := byID[strings.TrimSpace(current.PreviousRunID)]
		if !ok {
			break
		}
		current = previous
	}
	return nil
}

func taskRunsByIDForTaskWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
) (runs map[string]taskpkg.Run, err error) {
	rows, err := exec.QueryContext(
		ctx,
		`SELECT `+taskRunSelectColumnsSQL+`
		   FROM task_runs
		  WHERE task_id = ?`,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list retry lineage runs for task %q: %w", taskID, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close retry lineage rows for task %q: %w", taskID, closeErr)
		}
	}()
	byID := make(map[string]taskpkg.Run)
	for rows.Next() {
		run, err := scanTaskRunRecord(rows)
		if err != nil {
			return nil, err
		}
		byID[run.ID] = run
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate retry lineage runs for task %q: %w", taskID, err)
	}
	return byID, nil
}

func requireNoRetryChildWithExecutor(ctx context.Context, exec taskSQLExecutor, sourceRunID string) error {
	var existing string
	if err := exec.QueryRowContext(
		ctx,
		`SELECT id FROM task_runs WHERE previous_run_id = ? ORDER BY queued_at DESC, id DESC LIMIT 1`,
		sourceRunID,
	).Scan(&existing); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("store: lookup retry child for task run %q: %w", sourceRunID, err)
	}
	return fmt.Errorf(
		"%w: task run %q already has retry run %q",
		taskpkg.ErrInvalidStatusTransition,
		sourceRunID,
		existing,
	)
}

func forceRunCASTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return store.FormatTimestamp(value.UTC())
}

func normalizedForceRunTime(value time.Time, fallback func() time.Time) time.Time {
	if value.IsZero() {
		return fallback().UTC()
	}
	return value.UTC()
}
