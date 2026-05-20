package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const (
	globalDBTaskReviewReviewIDKey = "review_id"
	globalDBTaskReviewSourceKey   = "source"
)

var _ taskpkg.RunReviewStore = (*GlobalDB)(nil)

const runReviewSelectColumnsSQL = `review_id, task_id, run_id, parent_review_id,
	policy, review_round, attempt, status, outcome, confidence, reason, delivery_id,
	missing_work_json, next_round_guidance, review_text, reviewer_session_id,
	reviewer_agent_name, reviewer_peer_id, reviewer_channel_id, reviewed_by_kind,
	reviewed_by_ref, requested_at, routed_at, started_at, reviewed_at, deadline_at,
	created_at, updated_at`

// RequestRunReview persists one review request or returns the existing idempotent request.
func (g *GlobalDB) RequestRunReview(
	ctx context.Context,
	review *taskpkg.RunReview,
) (stored taskpkg.RunReview, created bool, err error) {
	if err := g.checkReady(ctx, "request task run review"); err != nil {
		return taskpkg.RunReview{}, false, err
	}
	normalized, err := review.Normalize(g.now().UTC())
	if err != nil {
		return taskpkg.RunReview{}, false, err
	}

	if err := g.withTaskImmediateTransaction(ctx, "request task run review", func(exec taskSQLExecutor) error {
		run, err := g.getTaskRunWithExecutor(ctx, exec, normalized.RunID)
		if err != nil {
			return err
		}
		if strings.TrimSpace(run.TaskID) != normalized.TaskID {
			return fmt.Errorf(
				"%w: run %q belongs to task %q, not task %q",
				taskpkg.ErrValidation,
				run.ID,
				run.TaskID,
				normalized.TaskID,
			)
		}
		if !taskpkg.IsTerminalRunStatus(run.Status) {
			return fmt.Errorf(
				"%w: run %q is %q and cannot be reviewed until terminal",
				taskpkg.ErrInvalidStatusTransition,
				run.ID,
				run.Status.Normalize(),
			)
		}
		if !normalized.Policy.MatchesRunStatus(run.Status) {
			return fmt.Errorf(
				"%w: review policy %q does not apply to run status %q",
				taskpkg.ErrInvalidStatusTransition,
				normalized.Policy,
				run.Status.Normalize(),
			)
		}
		created, err = insertRunReviewRequest(ctx, exec, normalized)
		if err != nil {
			return err
		}
		stored, err = getRunReviewByRunRoundAttempt(
			ctx,
			exec,
			normalized.RunID,
			normalized.ReviewRound,
			normalized.Attempt,
		)
		if err != nil {
			return err
		}
		return linkTaskRunReviewRequest(ctx, exec, stored)
	}); err != nil {
		return taskpkg.RunReview{}, false, err
	}
	return stored, created, nil
}

// GetRunReview returns one persisted run review by id.
func (g *GlobalDB) GetRunReview(ctx context.Context, reviewID string) (taskpkg.RunReview, error) {
	if err := g.checkReady(ctx, "get task run review"); err != nil {
		return taskpkg.RunReview{}, err
	}
	trimmedID, err := requireTaskValue(reviewID, "task run review id")
	if err != nil {
		return taskpkg.RunReview{}, err
	}
	return getRunReviewByID(ctx, g.db, trimmedID)
}

// BindRunReviewSession binds an active review request to a reviewer session.
func (g *GlobalDB) BindRunReviewSession(
	ctx context.Context,
	req taskpkg.BindRunReviewSessionRequest,
	boundAt time.Time,
) (stored taskpkg.RunReview, err error) {
	if err := g.checkReady(ctx, "bind task run review session"); err != nil {
		return taskpkg.RunReview{}, err
	}
	normalized := req.Normalize()
	if err := normalized.Validate("run_review_binding"); err != nil {
		return taskpkg.RunReview{}, err
	}
	if boundAt.IsZero() {
		boundAt = g.now().UTC()
	} else {
		boundAt = boundAt.UTC()
	}

	if err := g.withTaskImmediateTransaction(ctx, "bind task run review session", func(exec taskSQLExecutor) error {
		current, err := getRunReviewByID(ctx, exec, normalized.ReviewID)
		if err != nil {
			return err
		}
		if err := validateRunReviewBindingTransition(current, normalized.SessionID); err != nil {
			return err
		}
		if err := updateRunReviewBinding(ctx, exec, normalized, boundAt); err != nil {
			return err
		}
		stored, err = getRunReviewByID(ctx, exec, normalized.ReviewID)
		return err
	}); err != nil {
		return taskpkg.RunReview{}, err
	}
	return stored, nil
}

// LookupRunReviewBySession returns the active run review bound to one reviewer session.
func (g *GlobalDB) LookupRunReviewBySession(ctx context.Context, sessionID string) (taskpkg.RunReview, error) {
	if err := g.checkReady(ctx, "lookup task run review by session"); err != nil {
		return taskpkg.RunReview{}, err
	}
	trimmedID, err := requireTaskValue(sessionID, "task run review session id")
	if err != nil {
		return taskpkg.RunReview{}, err
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT `+runReviewSelectColumnsSQL+`
		 FROM task_run_reviews
		 WHERE reviewer_session_id = ?
		   AND status IN (?, ?)
		 ORDER BY updated_at DESC, review_id DESC
		 LIMIT 1`,
		trimmedID,
		string(taskpkg.RunReviewStatusRouted),
		string(taskpkg.RunReviewStatusInReview),
	)
	review, err := scanRunReview(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.RunReview{}, taskpkg.ErrRunReviewNotFound
		}
		return taskpkg.RunReview{}, err
	}
	return review, nil
}

// ListRunReviews returns persisted run reviews that match the supplied filters.
func (g *GlobalDB) ListRunReviews(
	ctx context.Context,
	query taskpkg.RunReviewQuery,
) ([]taskpkg.RunReview, error) {
	if err := g.checkReady(ctx, "list task run reviews"); err != nil {
		return nil, err
	}
	normalized := normalizeRunReviewQuery(query)
	if err := normalized.Validate("run_review_query"); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT ` + runReviewSelectColumnsSQL + ` FROM task_run_reviews`
	where, args := store.BuildClauses(
		store.StringClause("task_id", normalized.TaskID),
		store.StringClause("run_id", normalized.RunID),
		store.StringClause("status", string(normalized.Status)),
		store.StringClause("reviewer_session_id", normalized.ReviewerSessionID),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY updated_at DESC, review_id DESC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, normalized.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query task run reviews: %w", err)
	}
	reviews := make([]taskpkg.RunReview, 0)
	for rows.Next() {
		review, scanErr := scanRunReview(rows)
		if scanErr != nil {
			return nil, joinRowsCloseError(rows, scanErr, "task run review query")
		}
		reviews = append(reviews, review)
	}
	if err := rows.Err(); err != nil {
		return nil, joinRowsCloseError(
			rows,
			fmt.Errorf("store: iterate task run reviews: %w", err),
			"task run review query",
		)
	}
	if err := joinRowsCloseError(rows, nil, "task run review query"); err != nil {
		return nil, err
	}
	return reviews, nil
}

// RecordRunReview persists one authoritative review verdict and optional continuation run.
func (g *GlobalDB) RecordRunReview(
	ctx context.Context,
	req taskpkg.RecordRunReviewRequest,
	actor taskpkg.ActorContext,
	recordedAt time.Time,
	continuationRunID string,
) (result taskpkg.RunReviewResult, err error) {
	if err := g.checkReady(ctx, "record task run review"); err != nil {
		return taskpkg.RunReviewResult{}, err
	}
	normalized := req.Normalize()
	if err := normalized.Validate("record_run_review"); err != nil {
		return taskpkg.RunReviewResult{}, err
	}
	if err := actor.Validate(); err != nil {
		return taskpkg.RunReviewResult{}, err
	}
	if recordedAt.IsZero() {
		recordedAt = g.now().UTC()
	} else {
		recordedAt = recordedAt.UTC()
	}
	if normalized.Verdict.Outcome == taskpkg.RunReviewOutcomeRejected {
		if _, err := requireTaskValue(continuationRunID, "continuation task run id"); err != nil {
			return taskpkg.RunReviewResult{}, err
		}
	}

	if err := g.withTaskImmediateTransaction(ctx, "record task run review", func(exec taskSQLExecutor) error {
		stored, txErr := g.recordRunReviewWithExecutor(ctx, exec, normalized, actor, recordedAt, continuationRunID)
		if txErr != nil {
			return txErr
		}
		result = stored
		return nil
	}); err != nil {
		return taskpkg.RunReviewResult{}, err
	}
	return result, nil
}

func insertRunReviewRequest(
	ctx context.Context,
	exec taskSQLExecutor,
	review taskpkg.RunReview,
) (bool, error) {
	result, err := exec.ExecContext(
		ctx,
		`INSERT INTO task_run_reviews (
			review_id, task_id, run_id, parent_review_id, policy, review_round,
			attempt, status, outcome, confidence, reason, delivery_id,
			missing_work_json, next_round_guidance, review_text, reviewer_session_id,
			reviewer_agent_name, reviewer_peer_id, reviewer_channel_id,
			reviewed_by_kind, reviewed_by_ref, requested_at, routed_at, started_at,
			reviewed_at, deadline_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(run_id, review_round, attempt) DO NOTHING`,
		review.ReviewID,
		review.TaskID,
		review.RunID,
		store.NullableString(review.ParentReviewID),
		string(review.Policy),
		review.ReviewRound,
		review.Attempt,
		string(review.Status),
		runReviewOutcomeValue(review.Outcome),
		runReviewConfidenceValue(review.Confidence),
		review.Reason,
		store.NullableString(review.DeliveryID),
		string(review.MissingWork),
		review.NextRoundGuidance,
		review.ReviewText,
		store.NullableString(review.ReviewerSessionID),
		review.ReviewerAgentName,
		review.ReviewerPeerID,
		review.ReviewerChannelID,
		runReviewActorKindValue(review.ReviewedBy),
		runReviewActorRefValue(review.ReviewedBy),
		store.FormatTimestamp(review.RequestedAt),
		nullableTaskTimestamp(review.RoutedAt),
		nullableTaskTimestamp(review.StartedAt),
		nullableTaskTimestamp(review.ReviewedAt),
		nullableTaskTimestamp(review.DeadlineAt),
		store.FormatTimestamp(review.CreatedAt),
		store.FormatTimestamp(review.UpdatedAt),
	)
	if err != nil {
		return false, fmt.Errorf("store: create task run review %q: %w", review.ReviewID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("store: rows affected for task run review %q: %w", review.ReviewID, err)
	}
	return affected > 0, nil
}

func getRunReviewByID(ctx context.Context, exec taskSQLExecutor, reviewID string) (taskpkg.RunReview, error) {
	row := exec.QueryRowContext(
		ctx,
		`SELECT `+runReviewSelectColumnsSQL+`
		 FROM task_run_reviews
		 WHERE review_id = ?`,
		reviewID,
	)
	review, err := scanRunReview(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.RunReview{}, taskpkg.ErrRunReviewNotFound
		}
		return taskpkg.RunReview{}, err
	}
	return review, nil
}

func getRunReviewByRunRoundAttempt(
	ctx context.Context,
	exec taskSQLExecutor,
	runID string,
	round int,
	attempt int,
) (taskpkg.RunReview, error) {
	row := exec.QueryRowContext(
		ctx,
		`SELECT `+runReviewSelectColumnsSQL+`
		 FROM task_run_reviews
		 WHERE run_id = ? AND review_round = ? AND attempt = ?`,
		runID,
		round,
		attempt,
	)
	review, err := scanRunReview(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.RunReview{}, taskpkg.ErrRunReviewNotFound
		}
		return taskpkg.RunReview{}, err
	}
	return review, nil
}

func linkTaskRunReviewRequest(ctx context.Context, exec taskSQLExecutor, review taskpkg.RunReview) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE task_runs
		 SET review_required = 0,
		     review_request_round = ?,
		     review_policy_snapshot = ?,
		     review_request_id = ?
		 WHERE id = ?
		   AND (review_request_id IS NULL OR review_request_id = ?)`,
		review.ReviewRound,
		string(review.Policy),
		review.ReviewID,
		review.RunID,
		review.ReviewID,
	)
	if err != nil {
		return fmt.Errorf("store: link task run %q to review %q: %w", review.RunID, review.ReviewID, err)
	}
	return requireRowsAffected(result, taskpkg.ErrTaskRunNotFound, review.RunID, "task run")
}

func (g *GlobalDB) recordRunReviewWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	req taskpkg.RecordRunReviewRequest,
	actor taskpkg.ActorContext,
	recordedAt time.Time,
	continuationRunID string,
) (taskpkg.RunReviewResult, error) {
	current, err := getRunReviewByID(ctx, exec, req.ReviewID)
	if err != nil {
		return taskpkg.RunReviewResult{}, err
	}
	if strings.TrimSpace(current.RunID) != strings.TrimSpace(req.RunID) {
		return taskpkg.RunReviewResult{}, fmt.Errorf(
			"%w: review %q belongs to run %q, not run %q",
			taskpkg.ErrValidation,
			current.ReviewID,
			current.RunID,
			req.RunID,
		)
	}
	run, err := g.getTaskRunWithExecutor(ctx, exec, current.RunID)
	if err != nil {
		return taskpkg.RunReviewResult{}, err
	}
	if !taskpkg.IsTerminalRunStatus(run.Status) {
		return taskpkg.RunReviewResult{}, fmt.Errorf(
			"%w: run %q is %q and cannot record review verdict until terminal",
			taskpkg.ErrInvalidStatusTransition,
			run.ID,
			run.Status.Normalize(),
		)
	}
	if current.Status.Normalize() == taskpkg.RunReviewStatusRecorded {
		return replayRecordedRunReview(ctx, exec, current, req, actor.Actor)
	}
	if err := validateRunReviewRecordTransition(current, actor.Actor); err != nil {
		return taskpkg.RunReviewResult{}, err
	}

	recorded := applyRunReviewVerdict(current, req.Verdict, actor.Actor, recordedAt)
	if err := updateRunReviewVerdict(ctx, exec, recorded); err != nil {
		return taskpkg.RunReviewResult{}, err
	}
	if err := updateTaskReviewRollup(ctx, exec, recorded, recordedAt); err != nil {
		return taskpkg.RunReviewResult{}, err
	}
	result := taskpkg.RunReviewResult{Review: cloneRunReviewForStore(recorded)}
	if recorded.Outcome.Normalize() != taskpkg.RunReviewOutcomeRejected {
		return result, nil
	}
	taskRecord, err := g.getTaskWithExecutor(ctx, exec, recorded.TaskID)
	if err != nil {
		return taskpkg.RunReviewResult{}, err
	}
	continuation, err := g.createReviewContinuationRun(
		ctx,
		exec,
		taskRecord,
		run,
		recorded,
		actor,
		continuationRunID,
		recordedAt,
	)
	if err != nil {
		return taskpkg.RunReviewResult{}, err
	}
	result.ContinuationRun = &continuation
	return result, nil
}

func replayRecordedRunReview(
	ctx context.Context,
	exec taskSQLExecutor,
	current taskpkg.RunReview,
	req taskpkg.RecordRunReviewRequest,
	actor taskpkg.ActorIdentity,
) (taskpkg.RunReviewResult, error) {
	if !matchesRecordedRunReviewReplay(current, req.Verdict, actor) {
		return taskpkg.RunReviewResult{}, fmt.Errorf(
			"%w: review %q is already recorded with a different verdict",
			taskpkg.ErrConflict,
			current.ReviewID,
		)
	}
	result := taskpkg.RunReviewResult{Review: cloneRunReviewForStore(current)}
	if current.Outcome.Normalize() != taskpkg.RunReviewOutcomeRejected {
		return result, nil
	}
	continuation, err := getTaskRunByReviewID(ctx, exec, current.ReviewID)
	if err != nil {
		return taskpkg.RunReviewResult{}, err
	}
	result.ContinuationRun = &continuation
	return result, nil
}

func validateRunReviewRecordTransition(review taskpkg.RunReview, actor taskpkg.ActorIdentity) error {
	switch review.Status.Normalize() {
	case taskpkg.RunReviewStatusRequested,
		taskpkg.RunReviewStatusRouted,
		taskpkg.RunReviewStatusInReview:
	default:
		return fmt.Errorf(
			"%w: task run review %q with status %q cannot record a verdict",
			taskpkg.ErrInvalidStatusTransition,
			review.ReviewID,
			review.Status.Normalize(),
		)
	}
	if actor.Kind.Normalize() != taskpkg.ActorKindAgentSession {
		return nil
	}
	if strings.TrimSpace(review.ReviewerSessionID) == "" {
		return nil
	}
	if strings.TrimSpace(actor.Ref) == strings.TrimSpace(review.ReviewerSessionID) {
		return nil
	}
	return fmt.Errorf(
		"%w: reviewer session %q cannot record review %q bound to session %q",
		taskpkg.ErrPermissionDenied,
		actor.Ref,
		review.ReviewID,
		review.ReviewerSessionID,
	)
}

func applyRunReviewVerdict(
	review taskpkg.RunReview,
	verdict taskpkg.RunReviewVerdict,
	actor taskpkg.ActorIdentity,
	recordedAt time.Time,
) taskpkg.RunReview {
	recorded := cloneRunReviewForStore(review)
	recorded.Status = taskpkg.RunReviewStatusRecorded
	recorded.Outcome = verdict.Outcome.Normalize()
	recorded.Confidence = verdict.Confidence
	recorded.Reason = verdict.Reason
	recorded.DeliveryID = verdict.DeliveryID
	recorded.MissingWork = cloneTaskRawJSON(verdict.MissingWork)
	recorded.NextRoundGuidance = verdict.NextRoundGuidance
	recorded.ReviewText = verdict.ReviewText
	recorded.ReviewedBy = &taskpkg.ActorIdentity{Kind: actor.Kind.Normalize(), Ref: strings.TrimSpace(actor.Ref)}
	recorded.ReviewedAt = recordedAt
	recorded.UpdatedAt = recordedAt
	return recorded
}

func updateRunReviewVerdict(ctx context.Context, exec taskSQLExecutor, review taskpkg.RunReview) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE task_run_reviews
		 SET status = ?,
		     outcome = ?,
		     confidence = ?,
		     reason = ?,
		     delivery_id = ?,
		     missing_work_json = ?,
		     next_round_guidance = ?,
		     review_text = ?,
		     reviewed_by_kind = ?,
		     reviewed_by_ref = ?,
		     reviewed_at = ?,
		     updated_at = ?
		 WHERE review_id = ?`,
		string(taskpkg.RunReviewStatusRecorded),
		string(review.Outcome),
		runReviewConfidenceValue(review.Confidence),
		review.Reason,
		review.DeliveryID,
		string(review.MissingWork),
		review.NextRoundGuidance,
		review.ReviewText,
		runReviewActorKindValue(review.ReviewedBy),
		runReviewActorRefValue(review.ReviewedBy),
		store.FormatTimestamp(review.ReviewedAt),
		store.FormatTimestamp(review.UpdatedAt),
		review.ReviewID,
	)
	if err != nil {
		return mapRunReviewRecordError(review.ReviewID, err)
	}
	return requireRowsAffected(result, taskpkg.ErrRunReviewNotFound, review.ReviewID, "task run review")
}

func updateTaskReviewRollup(
	ctx context.Context,
	exec taskSQLExecutor,
	review taskpkg.RunReview,
	recordedAt time.Time,
) error {
	nextRound := review.ReviewRound
	circuitOpenedAt := any(nil)
	circuitReason := ""
	if review.Outcome.Normalize() == taskpkg.RunReviewOutcomeRejected {
		nextRound = review.ReviewRound + 1
	}
	if review.Outcome.Normalize() == taskpkg.RunReviewOutcomeBlocked {
		circuitOpenedAt = store.FormatTimestamp(recordedAt)
		circuitReason = review.Reason
	}

	result, err := exec.ExecContext(
		ctx,
		`UPDATE tasks
		 SET review_round = ?,
		     last_review_id = ?,
		     last_review_outcome = ?,
		     review_circuit_opened_at = ?,
		     review_circuit_reason = ?,
		     updated_at = ?
		 WHERE id = ?`,
		nextRound,
		review.ReviewID,
		string(review.Outcome),
		circuitOpenedAt,
		circuitReason,
		store.FormatTimestamp(recordedAt),
		review.TaskID,
	)
	if err != nil {
		return fmt.Errorf("store: update task %q review rollup: %w", review.TaskID, err)
	}
	return requireRowsAffected(result, taskpkg.ErrTaskNotFound, review.TaskID, "task")
}

func (g *GlobalDB) createReviewContinuationRun(
	ctx context.Context,
	exec taskSQLExecutor,
	taskRecord taskpkg.Task,
	parentRun taskpkg.Run,
	review taskpkg.RunReview,
	actor taskpkg.ActorContext,
	runID string,
	queuedAt time.Time,
) (taskpkg.Run, error) {
	openRunID, err := g.findOpenRunIDForQueuedRunReservation(ctx, exec, taskRecord.ID)
	if err != nil {
		return taskpkg.Run{}, err
	}
	if openRunID != "" {
		return taskpkg.Run{}, fmt.Errorf(
			"%w: task %q has open run %q; finish or cancel it before enqueueing review continuation",
			taskpkg.ErrInvalidStatusTransition,
			taskRecord.ID,
			openRunID,
		)
	}
	nextAttempt, err := nextTaskRunAttemptWithExecutor(ctx, exec, taskRecord)
	if err != nil {
		return taskpkg.Run{}, err
	}
	metadata, err := reviewContinuationMetadata(review, &parentRun)
	if err != nil {
		return taskpkg.Run{}, err
	}
	networkChannel := resolveStoredRunChannel(parentRun.NetworkChannel, taskRecord.NetworkChannel)
	coordinationChannelID := coordinationChannelIDForQueuedRun(taskRecord, networkChannel, runID)
	if err := ensureQueuedRunCoordinationChannel(
		ctx,
		exec,
		taskRecord,
		coordinationChannelID,
		actor.Origin,
		queuedAt,
	); err != nil {
		return taskpkg.Run{}, err
	}
	run := taskpkg.Run{
		ID:                    strings.TrimSpace(runID),
		TaskID:                taskRecord.ID,
		Status:                taskpkg.TaskRunStatusQueued,
		Attempt:               nextAttempt,
		Origin:                actor.Origin,
		NetworkChannel:        networkChannel,
		CoordinationChannelID: coordinationChannelID,
		Review: &taskpkg.RunReviewLineage{
			ParentRunID:        parentRun.ID,
			ReviewID:           review.ReviewID,
			ReviewRound:        review.ReviewRound + 1,
			ContinuationReason: review.Reason,
			MissingWork:        cloneTaskRawJSON(review.MissingWork),
			NextRoundGuidance:  review.NextRoundGuidance,
		},
		Metadata: metadata,
		QueuedAt: queuedAt,
	}
	normalized, err := g.normalizeTaskRunForCreate(run)
	if err != nil {
		return taskpkg.Run{}, err
	}
	if err := insertQueuedTaskRun(ctx, exec, normalized); err != nil {
		return taskpkg.Run{}, mapReviewContinuationInsertError(review.ReviewID, err)
	}
	return g.getTaskRunWithExecutor(ctx, exec, normalized.ID)
}

func reviewContinuationMetadata(review taskpkg.RunReview, parentRun *taskpkg.Run) (json.RawMessage, error) {
	payload := map[string]string{
		globalDBTaskReviewSourceKey:   "task_run_review",
		globalDBTaskReviewReviewIDKey: review.ReviewID,
		globalDBOutcomeKey:            string(review.Outcome),
	}
	if parentRun != nil {
		payload["parent_run_id"] = parentRun.ID
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("store: encode review continuation metadata: %w", err)
	}
	return raw, nil
}

func getTaskRunByReviewID(
	ctx context.Context,
	exec taskSQLExecutor,
	reviewID string,
) (taskpkg.Run, error) {
	row := exec.QueryRowContext(
		ctx,
		`SELECT `+taskRunSelectColumnsSQL+`
		 FROM task_runs
		 WHERE review_id = ?`,
		reviewID,
	)
	run, err := scanTaskRunRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.Run{}, taskpkg.ErrTaskRunNotFound
		}
		return taskpkg.Run{}, err
	}
	return run, nil
}

func matchesRecordedRunReviewReplay(
	current taskpkg.RunReview,
	verdict taskpkg.RunReviewVerdict,
	actor taskpkg.ActorIdentity,
) bool {
	return current.DeliveryID == verdict.DeliveryID &&
		current.Outcome.Normalize() == verdict.Outcome.Normalize() &&
		actorsEqual(current.ReviewedBy, actor) &&
		current.Reason == verdict.Reason &&
		string(current.MissingWork) == string(verdict.MissingWork) &&
		current.NextRoundGuidance == verdict.NextRoundGuidance &&
		current.ReviewText == verdict.ReviewText
}

func actorsEqual(left *taskpkg.ActorIdentity, right taskpkg.ActorIdentity) bool {
	if left == nil {
		return false
	}
	return left.Kind.Normalize() == right.Kind.Normalize() &&
		strings.TrimSpace(left.Ref) == strings.TrimSpace(right.Ref)
}

func mapRunReviewRecordError(reviewID string, err error) error {
	if err == nil {
		return nil
	}
	if isTaskRunReviewSQLiteUniqueConstraint(err) {
		return fmt.Errorf("%w: delivery id is already recorded for review %q", taskpkg.ErrConflict, reviewID)
	}
	return fmt.Errorf("store: record task run review %q: %w", reviewID, err)
}

func mapReviewContinuationInsertError(reviewID string, err error) error {
	if err == nil {
		return nil
	}
	if isTaskRunReviewSQLiteUniqueConstraint(err) {
		return fmt.Errorf("%w: continuation run already exists for review %q", taskpkg.ErrConflict, reviewID)
	}
	return err
}

func validateRunReviewBindingTransition(review taskpkg.RunReview, sessionID string) error {
	currentSessionID := strings.TrimSpace(review.ReviewerSessionID)
	nextSessionID := strings.TrimSpace(sessionID)
	if currentSessionID != "" &&
		currentSessionID != nextSessionID &&
		review.Status.Normalize() != taskpkg.RunReviewStatusInReview {
		return fmt.Errorf(
			"%w: task run review %q is already bound to session %q",
			taskpkg.ErrInvalidStatusTransition,
			review.ReviewID,
			review.ReviewerSessionID,
		)
	}
	switch review.Status.Normalize() {
	case taskpkg.RunReviewStatusRequested,
		taskpkg.RunReviewStatusRouted,
		taskpkg.RunReviewStatusInReview:
		return nil
	default:
		return fmt.Errorf(
			"%w: task run review %q with status %q cannot bind a reviewer session",
			taskpkg.ErrInvalidStatusTransition,
			review.ReviewID,
			review.Status.Normalize(),
		)
	}
}

func updateRunReviewBinding(
	ctx context.Context,
	exec taskSQLExecutor,
	req taskpkg.BindRunReviewSessionRequest,
	boundAt time.Time,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE task_run_reviews
		 SET status = ?,
		     reviewer_session_id = ?,
		     reviewer_agent_name = ?,
		     reviewer_peer_id = ?,
		     reviewer_channel_id = ?,
		     started_at = COALESCE(started_at, ?),
		     updated_at = ?
		 WHERE review_id = ?`,
		string(taskpkg.RunReviewStatusInReview),
		req.SessionID,
		req.ReviewerAgentName,
		req.ReviewerPeerID,
		req.ReviewerChannelID,
		store.FormatTimestamp(boundAt),
		store.FormatTimestamp(boundAt),
		req.ReviewID,
	)
	if err != nil {
		return mapRunReviewBindError(req.ReviewID, req.SessionID, err)
	}
	return requireRowsAffected(result, taskpkg.ErrRunReviewNotFound, req.ReviewID, "task run review")
}

func mapRunReviewBindError(reviewID string, sessionID string, err error) error {
	if err == nil {
		return nil
	}
	if isTaskRunReviewSQLiteUniqueConstraint(err) {
		return fmt.Errorf(
			"%w: reviewer session %q already has an active task run review",
			taskpkg.ErrInvalidStatusTransition,
			sessionID,
		)
	}
	return fmt.Errorf("store: bind task run review %q to session %q: %w", reviewID, sessionID, err)
}

func isTaskRunReviewSQLiteUniqueConstraint(err error) bool {
	var sqliteErr *sqlite.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE
}

func scanRunReview(scanner rowScanner) (taskpkg.RunReview, error) {
	var review taskpkg.RunReview
	var fields runReviewScanFields
	if err := scanner.Scan(
		&review.ReviewID,
		&review.TaskID,
		&review.RunID,
		&fields.parentReviewID,
		&fields.policy,
		&review.ReviewRound,
		&review.Attempt,
		&fields.status,
		&fields.outcome,
		&fields.confidence,
		&review.Reason,
		&fields.deliveryID,
		&fields.missingWork,
		&review.NextRoundGuidance,
		&review.ReviewText,
		&fields.reviewerSessionID,
		&review.ReviewerAgentName,
		&review.ReviewerPeerID,
		&review.ReviewerChannelID,
		&fields.reviewedByKind,
		&fields.reviewedByRef,
		&fields.requestedAtRaw,
		&fields.routedAtRaw,
		&fields.startedAtRaw,
		&fields.reviewedAtRaw,
		&fields.deadlineAtRaw,
		&fields.createdAtRaw,
		&fields.updatedAtRaw,
	); err != nil {
		return taskpkg.RunReview{}, fmt.Errorf("store: scan task run review: %w", err)
	}
	return fields.record(review)
}

type runReviewScanFields struct {
	parentReviewID    sql.NullString
	policy            string
	status            string
	outcome           sql.NullString
	confidence        sql.NullFloat64
	deliveryID        sql.NullString
	missingWork       string
	reviewerSessionID sql.NullString
	reviewedByKind    sql.NullString
	reviewedByRef     sql.NullString
	requestedAtRaw    string
	routedAtRaw       sql.NullString
	startedAtRaw      sql.NullString
	reviewedAtRaw     sql.NullString
	deadlineAtRaw     sql.NullString
	createdAtRaw      string
	updatedAtRaw      string
}

func (fields runReviewScanFields) record(review taskpkg.RunReview) (taskpkg.RunReview, error) {
	review.ParentReviewID = taskNullStringValue(fields.parentReviewID)
	review.Policy = taskpkg.ReviewPolicy(strings.TrimSpace(fields.policy))
	review.Status = taskpkg.RunReviewStatus(strings.TrimSpace(fields.status))
	review.Outcome = taskpkg.RunReviewOutcome(taskNullStringValue(fields.outcome))
	if fields.confidence.Valid {
		confidence := fields.confidence.Float64
		review.Confidence = &confidence
	}
	review.DeliveryID = taskNullStringValue(fields.deliveryID)
	review.MissingWork = []byte(strings.TrimSpace(fields.missingWork))
	review.ReviewerSessionID = taskNullStringValue(fields.reviewerSessionID)
	if strings.TrimSpace(fields.reviewedByKind.String) != "" || strings.TrimSpace(fields.reviewedByRef.String) != "" {
		review.ReviewedBy = &taskpkg.ActorIdentity{
			Kind: taskpkg.ActorKind(strings.TrimSpace(fields.reviewedByKind.String)),
			Ref:  strings.TrimSpace(fields.reviewedByRef.String),
		}
	}
	if err := assignRunReviewTimestamps(&review, fields); err != nil {
		return taskpkg.RunReview{}, err
	}
	normalized, err := (&review).Normalize(review.UpdatedAt)
	if err != nil {
		return taskpkg.RunReview{}, err
	}
	return normalized, nil
}

func assignRunReviewTimestamps(review *taskpkg.RunReview, fields runReviewScanFields) error {
	requestedAt, err := store.ParseTimestamp(fields.requestedAtRaw)
	if err != nil {
		return fmt.Errorf("store: parse task run review requested_at: %w", err)
	}
	createdAt, err := store.ParseTimestamp(fields.createdAtRaw)
	if err != nil {
		return fmt.Errorf("store: parse task run review created_at: %w", err)
	}
	updatedAt, err := store.ParseTimestamp(fields.updatedAtRaw)
	if err != nil {
		return fmt.Errorf("store: parse task run review updated_at: %w", err)
	}
	review.RequestedAt = requestedAt
	review.CreatedAt = createdAt
	review.UpdatedAt = updatedAt
	if err := assignNullableTaskTimestamp(&review.RoutedAt, fields.routedAtRaw); err != nil {
		return fmt.Errorf("store: parse task run review routed_at: %w", err)
	}
	if err := assignNullableTaskTimestamp(&review.StartedAt, fields.startedAtRaw); err != nil {
		return fmt.Errorf("store: parse task run review started_at: %w", err)
	}
	if err := assignNullableTaskTimestamp(&review.ReviewedAt, fields.reviewedAtRaw); err != nil {
		return fmt.Errorf("store: parse task run review reviewed_at: %w", err)
	}
	if err := assignNullableTaskTimestamp(&review.DeadlineAt, fields.deadlineAtRaw); err != nil {
		return fmt.Errorf("store: parse task run review deadline_at: %w", err)
	}
	return nil
}

func runReviewOutcomeValue(outcome taskpkg.RunReviewOutcome) any {
	normalized := outcome.Normalize()
	if normalized == "" {
		return nil
	}
	return string(normalized)
}

func runReviewConfidenceValue(confidence *float64) any {
	if confidence == nil {
		return nil
	}
	return *confidence
}

func runReviewActorKindValue(actor *taskpkg.ActorIdentity) string {
	if actor == nil {
		return ""
	}
	return string(actor.Kind)
}

func runReviewActorRefValue(actor *taskpkg.ActorIdentity) string {
	if actor == nil {
		return ""
	}
	return actor.Ref
}

func normalizeRunReviewQuery(query taskpkg.RunReviewQuery) taskpkg.RunReviewQuery {
	normalized := query
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.RunID = strings.TrimSpace(normalized.RunID)
	normalized.Status = normalized.Status.Normalize()
	normalized.ReviewerSessionID = strings.TrimSpace(normalized.ReviewerSessionID)
	return normalized
}

func cloneRunReviewForStore(review taskpkg.RunReview) taskpkg.RunReview {
	cloned := review
	cloned.MissingWork = cloneTaskRawJSON(review.MissingWork)
	if review.ReviewedBy != nil {
		reviewedBy := *review.ReviewedBy
		cloned.ReviewedBy = &reviewedBy
	}
	if review.Confidence != nil {
		confidence := *review.Confidence
		cloned.Confidence = &confidence
	}
	return cloned
}

func cloneTaskRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}
