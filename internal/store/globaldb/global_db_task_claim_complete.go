package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

func (g *GlobalDB) CompleteRunLease(
	ctx context.Context,
	completion taskpkg.LeaseCompletion,
) (taskpkg.Run, error) {
	if err := g.checkReady(ctx, "complete task run lease"); err != nil {
		return taskpkg.Run{}, err
	}
	normalized, err := completion.Normalize(g.now())
	if err != nil {
		return taskpkg.Run{}, err
	}

	var updated taskpkg.Run
	if err := g.withTaskImmediateTransaction(ctx, "complete task run lease", func(exec taskSQLExecutor) error {
		var err error
		updated, err = g.completeRunLeaseWithExecutor(ctx, exec, normalized)
		return err
	}); err != nil {
		return taskpkg.Run{}, err
	}
	return updated, nil
}

func (g *GlobalDB) completeRunLeaseWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	normalized taskpkg.LeaseCompletion,
) (taskpkg.Run, error) {
	current, err := g.getTaskRunWithExecutor(ctx, exec, normalized.RunID)
	if err != nil {
		return taskpkg.Run{}, err
	}
	if err := requireCurrentRunLease(current, normalized.ClaimToken, normalized.Now); err != nil {
		return taskpkg.Run{}, err
	}
	if err := requireLeaseTerminalTransition(current, taskpkg.TaskRunStatusCompleted); err != nil {
		return taskpkg.Run{}, err
	}
	if err := g.verifyCompletionCreatedTaskClaims(ctx, exec, current, normalized.CreatedTaskIDs); err != nil {
		return taskpkg.Run{}, err
	}
	resultPayload, outputRef, err := g.storeLoopResultPayloadByRefIfLarge(ctx, exec, current, normalized)
	if err != nil {
		return taskpkg.Run{}, err
	}
	if err := completeRunLeaseRowWithExecutor(ctx, exec, current, normalized, resultPayload); err != nil {
		return taskpkg.Run{}, err
	}
	if err := recordLoopNodeTerminalWithExecutor(
		ctx,
		exec,
		current,
		"success",
		loopNodeTerminalOutputRef(outputRef, resultPayload),
		normalized.Now,
	); err != nil {
		return taskpkg.Run{}, err
	}
	if err := clearTaskCurrentRunProjection(ctx, exec, current.TaskID, current.ID); err != nil {
		return taskpkg.Run{}, err
	}
	if err := resetTaskBlockRecurrencesWithExecutor(ctx, exec, current.TaskID); err != nil {
		return taskpkg.Run{}, err
	}
	updated, err := g.getTaskRunWithExecutor(ctx, exec, current.ID)
	if err != nil {
		return taskpkg.Run{}, err
	}
	return updated, nil
}

func (g *GlobalDB) storeLoopResultPayloadByRefIfLarge(
	ctx context.Context,
	exec taskSQLExecutor,
	run taskpkg.Run,
	completion taskpkg.LeaseCompletion,
) (json.RawMessage, string, error) {
	resultPayload := normalizeTaskJSON(completion.Result.Value)
	if !loopNodeRunShouldExternalizeResult(run, resultPayload) {
		return resultPayload, "", nil
	}
	outputRef := looppkg.OutputRefForPayload(resultPayload)
	if err := upsertLoopOutputBlobWithExecutor(ctx, exec, outputRef, resultPayload, completion.Now); err != nil {
		return nil, "", err
	}
	return nil, outputRef, nil
}

func completeRunLeaseRowWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	current taskpkg.Run,
	normalized taskpkg.LeaseCompletion,
	resultPayload json.RawMessage,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE task_runs
		 SET status = ?, lease_until = NULL, heartbeat_at = NULL, claim_token = NULL,
		     ended_at = ?, tokens_used = ?, error = NULL, result_json = ?
		 WHERE id = ? AND claim_token_hash = ?`,
		taskpkg.TaskRunStatusCompleted.String(),
		store.FormatTimestamp(normalized.Now),
		normalized.TokensUsed,
		nullableTaskJSON(resultPayload),
		current.ID,
		current.ClaimTokenHash,
	)
	if err != nil {
		return fmt.Errorf("store: complete task run lease %q: %w", current.ID, err)
	}
	return requireRowsAffected(result, taskpkg.ErrTaskRunNotFound, current.ID, "task run lease")
}

func loopNodeRunShouldExternalizeResult(run taskpkg.Run, payload json.RawMessage) bool {
	if strings.TrimSpace(run.LoopRunID) == "" {
		return false
	}
	if run.RunKind.Normalize() == taskpkg.RunKindCoordinator {
		return false
	}
	return looppkg.OutputPayloadRequiresRef(payload)
}

func (g *GlobalDB) verifyCompletionCreatedTaskClaims(
	ctx context.Context,
	exec taskSQLExecutor,
	run taskpkg.Run,
	claimedTaskIDs []string,
) error {
	if len(claimedTaskIDs) == 0 {
		return nil
	}
	taskRecord, err := g.getTaskWithExecutor(ctx, exec, run.TaskID)
	if err != nil {
		return err
	}
	sessionID := strings.TrimSpace(run.SessionID)
	invalidTaskIDs := make([]string, 0)
	for _, claimedTaskID := range claimedTaskIDs {
		ok := false
		if sessionID != "" {
			var lookupErr error
			ok, lookupErr = completionCreatedTaskClaimMatches(
				ctx,
				exec,
				claimedTaskID,
				taskRecord.WorkspaceID,
				sessionID,
			)
			if lookupErr != nil {
				return lookupErr
			}
		}
		if !ok {
			invalidTaskIDs = append(invalidTaskIDs, claimedTaskID)
		}
	}
	if len(invalidTaskIDs) > 0 {
		return taskpkg.NewHallucinatedTaskRefsError(run, claimedTaskIDs, invalidTaskIDs)
	}
	return nil
}

func completionCreatedTaskClaimMatches(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
	workspaceID string,
	sessionID string,
) (bool, error) {
	var found int
	if err := exec.QueryRowContext(
		ctx,
		`SELECT 1
		   FROM tasks
		  WHERE id = ?
		    AND COALESCE(workspace_id, '') = ?
		    AND created_by_kind = ?
		    AND created_by_ref = ?`,
		strings.TrimSpace(taskID),
		strings.TrimSpace(workspaceID),
		string(taskpkg.ActorKindAgentSession),
		strings.TrimSpace(sessionID),
	).Scan(&found); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("store: verify completion created task claim %q: %w", taskID, err)
	}
	return found == 1, nil
}

type loopNodeRunMetadata struct {
	Generation int    `json:"generation"`
	NodeID     string `json:"node_id"`
	ItemIndex  int    `json:"item_index"`
}

func recordLoopNodeTerminalWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	run taskpkg.Run,
	outcome string,
	outputRef string,
	terminalAt time.Time,
) error {
	const (
		loopNodeOutcomeFailure  = "failure"
		loopNodeOutputFailed    = "failed"
		loopNodeOutputSucceeded = "succeeded"
	)

	loopRunID := strings.TrimSpace(run.LoopRunID)
	if loopRunID == "" || run.RunKind.Normalize() == taskpkg.RunKindCoordinator {
		return nil
	}
	status := loopNodeOutputSucceeded
	failureDelta := 0
	if outcome == loopNodeOutcomeFailure {
		status = loopNodeOutputFailed
		failureDelta = 1
	}
	if err := updateLoopNodeOutputTerminalWithExecutor(ctx, exec, run, loopRunID, status, outputRef); err != nil {
		return err
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_runs
		 SET last_progress_at = CASE WHEN last_progress_at < ? THEN ? ELSE last_progress_at END,
		     consecutive_failures = CASE WHEN ? = 1 THEN consecutive_failures + 1 ELSE 0 END
		 WHERE id = ?`,
		store.FormatTimestamp(terminalAt),
		store.FormatTimestamp(terminalAt),
		failureDelta,
		loopRunID,
	)
	if err != nil {
		return fmt.Errorf("store: record loop run %q node terminal progress: %w", loopRunID, err)
	}
	return requireRowsAffected(result, looppkg.ErrRunNotFound, loopRunID, "loop run")
}

func updateLoopNodeOutputTerminalWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	run taskpkg.Run,
	loopRunID string,
	status string,
	outputRef string,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_generation_outputs
		 SET status = ?,
		     output_ref = CASE WHEN ? = '' THEN output_ref ELSE ? END,
		     task_run_id = ?
		 WHERE loop_run_id = ?
		   AND task_run_id = ?`,
		status,
		strings.TrimSpace(outputRef),
		strings.TrimSpace(outputRef),
		run.ID,
		loopRunID,
		run.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"store: update loop run %q node output for task run %q: %w",
			loopRunID,
			run.ID,
			err,
		)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"store: rows affected updating loop run %q node output for task run %q: %w",
			loopRunID,
			run.ID,
			err,
		)
	}
	if affected > 0 {
		return nil
	}
	metadata, ok, err := loopNodeMetadataFromTaskRun(run.Metadata)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	_, err = exec.ExecContext(
		ctx,
		`INSERT INTO loop_generation_outputs (
			loop_run_id, generation, node_id, item_index, status, output_ref, task_run_id
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(loop_run_id, generation, node_id, item_index) DO UPDATE SET
			status = excluded.status,
			output_ref = excluded.output_ref,
			task_run_id = excluded.task_run_id`,
		loopRunID,
		metadata.Generation,
		metadata.NodeID,
		metadata.ItemIndex,
		status,
		nullString(outputRef),
		run.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"store: insert loop run %q node output for task run %q: %w",
			loopRunID,
			run.ID,
			err,
		)
	}
	return nil
}

func loopNodeMetadataFromTaskRun(raw json.RawMessage) (loopNodeRunMetadata, bool, error) {
	if len(raw) == 0 {
		return loopNodeRunMetadata{}, false, nil
	}
	var metadata loopNodeRunMetadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return loopNodeRunMetadata{}, false, fmt.Errorf(
			"store: decode loop node task run metadata: %w",
			err,
		)
	}
	metadata.NodeID = strings.TrimSpace(metadata.NodeID)
	if metadata.Generation <= 0 || metadata.NodeID == "" || metadata.ItemIndex < 0 {
		return loopNodeRunMetadata{}, false, nil
	}
	return metadata, true, nil
}

func loopFailureReasonCode(failure taskpkg.RunFailure) string {
	type reasonEnvelope struct {
		ReasonCode string `json:"reason_code"`
		Code       string `json:"code"`
	}
	var envelope reasonEnvelope
	if len(failure.Metadata) > 0 {
		if err := json.Unmarshal(failure.Metadata, &envelope); err == nil {
			if strings.TrimSpace(envelope.ReasonCode) != "" {
				return strings.TrimSpace(envelope.ReasonCode)
			}
			if strings.TrimSpace(envelope.Code) != "" {
				return strings.TrimSpace(envelope.Code)
			}
		}
	}
	for _, code := range []string{"dependency_missing", "credential_missing", "resource_unreachable"} {
		if strings.Contains(failure.Error, code) {
			return code
		}
	}
	return ""
}

// FailRunLease marks one claimed run failed after token verification.
