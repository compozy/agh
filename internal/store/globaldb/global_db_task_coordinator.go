package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
	aghworkspace "github.com/compozy/agh/internal/workspace"
)

const (
	loopCoordinatorActorRef = "loop"
	taskRunResultKindKey    = "kind"
)

// CompleteCoordinatorAndEnqueueNext applies a coordinator plan in one BEGIN IMMEDIATE transaction.
func (g *GlobalDB) CompleteCoordinatorAndEnqueueNext(
	ctx context.Context,
	completion taskpkg.CoordinatorCompletion,
	finalizer taskpkg.GenerationStateFinalizer,
) (taskpkg.CoordinatorCompletionResult, error) {
	if err := g.checkReady(ctx, "complete coordinator and enqueue next"); err != nil {
		return taskpkg.CoordinatorCompletionResult{}, err
	}
	if finalizer == nil {
		return taskpkg.CoordinatorCompletionResult{}, fmt.Errorf(
			"%w: generation state finalizer is required",
			taskpkg.ErrValidation,
		)
	}
	normalized, err := completion.Normalize(g.now())
	if err != nil {
		return taskpkg.CoordinatorCompletionResult{}, err
	}

	var result taskpkg.CoordinatorCompletionResult
	if err := g.withTaskImmediateTransaction(
		ctx,
		"complete coordinator and enqueue next",
		func(exec taskSQLExecutor) error {
			applied, applyErr := g.completeCoordinatorAndEnqueueNextWithExecutor(
				ctx,
				exec,
				normalized,
				finalizer,
			)
			if applyErr != nil {
				return applyErr
			}
			result = applied
			return nil
		},
	); err != nil {
		return taskpkg.CoordinatorCompletionResult{}, err
	}
	return result, nil
}

func (g *GlobalDB) completeCoordinatorAndEnqueueNextWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	completion taskpkg.CoordinatorCompletion,
	finalizer taskpkg.GenerationStateFinalizer,
) (taskpkg.CoordinatorCompletionResult, error) {
	current, loopRunID, err := g.prepareCoordinatorCompletionWithExecutor(ctx, exec, completion)
	if err != nil {
		return taskpkg.CoordinatorCompletionResult{}, err
	}
	if err := g.ensureCoordinatorPlanTasksWithExecutor(
		ctx,
		exec,
		completion.Plan,
		current,
		completion.Actor.Origin,
		completion.Now,
	); err != nil {
		return taskpkg.CoordinatorCompletionResult{}, err
	}
	if err := completeCoordinatorRunWithExecutor(ctx, exec, current, completion.Now); err != nil {
		return taskpkg.CoordinatorCompletionResult{}, err
	}
	if err := clearTaskCurrentRunProjection(ctx, exec, current.TaskID, current.ID); err != nil {
		return taskpkg.CoordinatorCompletionResult{}, err
	}
	if err := resetTaskBlockRecurrencesWithExecutor(ctx, exec, current.TaskID); err != nil {
		return taskpkg.CoordinatorCompletionResult{}, err
	}

	snapshot := completion.Plan.Snapshot
	if strings.TrimSpace(snapshot.LoopRunID) == "" {
		snapshot.LoopRunID = loopRunID
	}
	postReserveSnapshot := normalizePostReserveSnapshot(completion.Plan.PostReserveSnapshot, snapshot, loopRunID)
	if err := finalizer.WriteGenerationSnapshot(ctx, exec, snapshot); err != nil {
		return taskpkg.CoordinatorCompletionResult{}, err
	}
	if err := applyCoordinatorRunStopsWithExecutor(ctx, exec, completion.Plan.RunStops, completion.Now); err != nil {
		return taskpkg.CoordinatorCompletionResult{}, err
	}

	tokensUsed, err := refreshLoopTokensUsedWithExecutor(ctx, exec, loopRunID)
	if err != nil {
		return taskpkg.CoordinatorCompletionResult{}, err
	}
	loopRun, err := getLoopRunByIDWithExecutor(ctx, exec, loop.RunID(loopRunID))
	if err != nil {
		return taskpkg.CoordinatorCompletionResult{}, err
	}

	result, err := g.applyCoordinatorBoundaryWithExecutor(
		ctx,
		exec,
		&completion,
		&current,
		snapshot,
		postReserveSnapshot,
		finalizer,
		loopRun,
		tokensUsed,
	)
	if err != nil {
		return taskpkg.CoordinatorCompletionResult{}, err
	}

	updated, err := g.getTaskRunWithExecutor(ctx, exec, current.ID)
	if err != nil {
		return taskpkg.CoordinatorCompletionResult{}, err
	}
	result.Run = updated
	return result, nil
}

func normalizePostReserveSnapshot(
	postReserveSnapshot *taskpkg.GenerationSnapshot,
	base taskpkg.GenerationSnapshot,
	loopRunID string,
) *taskpkg.GenerationSnapshot {
	if postReserveSnapshot == nil {
		return nil
	}
	snapshot := *postReserveSnapshot
	if strings.TrimSpace(snapshot.LoopRunID) == "" {
		snapshot.LoopRunID = loopRunID
	}
	if snapshot.Generation <= 0 {
		snapshot.Generation = base.Generation
	}
	return &snapshot
}

func (g *GlobalDB) prepareCoordinatorCompletionWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	completion taskpkg.CoordinatorCompletion,
) (taskpkg.Run, string, error) {
	current, err := g.getTaskRunWithExecutor(ctx, exec, completion.RunID)
	if err != nil {
		return taskpkg.Run{}, "", err
	}
	if current.RunKind.Normalize() != taskpkg.RunKindCoordinator {
		return taskpkg.Run{}, "", fmt.Errorf(
			"%w: task run %q is %q, not %q",
			taskpkg.ErrInvalidStatusTransition,
			current.ID,
			current.RunKind.Normalize(),
			taskpkg.RunKindCoordinator,
		)
	}
	loopRunID := strings.TrimSpace(current.LoopRunID)
	if loopRunID == "" {
		return taskpkg.Run{}, "", fmt.Errorf(
			"%w: coordinator run %q has no loop_run_id",
			taskpkg.ErrValidation,
			current.ID,
		)
	}
	if err := requireCurrentRunLease(current, completion.ClaimToken, completion.Now); err != nil {
		return taskpkg.Run{}, "", err
	}
	if err := requireLeaseTerminalTransition(current, taskpkg.TaskRunStatusCompleted); err != nil {
		return taskpkg.Run{}, "", err
	}
	return current, loopRunID, nil
}

func (g *GlobalDB) applyCoordinatorBoundaryWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	completion *taskpkg.CoordinatorCompletion,
	current *taskpkg.Run,
	snapshot taskpkg.GenerationSnapshot,
	postReserveSnapshot *taskpkg.GenerationSnapshot,
	finalizer taskpkg.GenerationStateFinalizer,
	loopRun loop.Run,
	tokensUsed int64,
) (taskpkg.CoordinatorCompletionResult, error) {
	loopRunID := strings.TrimSpace(snapshot.LoopRunID)
	contextPayload, err := coordinatorResultContext(loopRun, loopRun.Generation, loopRun.Status)
	if err != nil {
		return taskpkg.CoordinatorCompletionResult{}, err
	}
	result := taskpkg.CoordinatorCompletionResult{
		LoopRunID:  loopRunID,
		Context:    contextPayload,
		TokensUsed: tokensUsed,
	}
	budgetExceeded := loopBudgetExceeded(loopRun, tokensUsed, completion.Now)
	switch {
	case loopRun.Status == loop.StatusWatching && completion.Plan.Terminal != nil:
		err := applyCoordinatorTerminalBoundary(ctx, exec, completion, snapshot, loopRun, &result)
		return result, err
	case loopRun.Status == loop.StatusWatching && coordinatorPlanHasContinuation(completion.Plan):
		err := g.applyCoordinatorWatchReadyBoundaryWithExecutor(
			ctx,
			exec,
			completion,
			current,
			snapshot,
			postReserveSnapshot,
			finalizer,
			loopRun,
			&result,
		)
		return result, err
	case loopRun.Status != loop.StatusRunning:
		err := applyCoordinatorYieldBoundary(ctx, exec, snapshot, &result)
		return result, err
	case shouldDeferCoordinatorBoundary(completion.Plan, loopRun, budgetExceeded):
		err := applyCoordinatorYieldBoundary(ctx, exec, snapshot, &result)
		return result, err
	case !completion.Plan.Yield &&
		completion.Plan.Terminal == nil &&
		budgetExceeded:
		err := applyCoordinatorBudgetBoundary(ctx, exec, completion, snapshot, loopRun, &result)
		return result, err
	case completion.Plan.Terminal != nil:
		err := applyCoordinatorTerminalBoundary(ctx, exec, completion, snapshot, loopRun, &result)
		return result, err
	case completion.Plan.Yield:
		err := applyCoordinatorYieldBoundary(ctx, exec, snapshot, &result)
		return result, err
	case loopRun.PauseRequested && loopRun.Status == loop.StatusRunning:
		err := applyCoordinatorPauseBoundary(ctx, exec, completion, snapshot, loopRun, &result)
		return result, err
	default:
		err := g.applyCoordinatorContinueBoundaryWithExecutor(
			ctx,
			exec,
			completion,
			current,
			snapshot,
			postReserveSnapshot,
			finalizer,
			&result,
		)
		return result, err
	}
}

func coordinatorPlanHasContinuation(plan taskpkg.CoordinatorCompletionPlan) bool {
	return len(plan.NodeRuns) > 0 || plan.NextCoordinator != nil
}

func shouldDeferCoordinatorBoundary(
	plan taskpkg.CoordinatorCompletionPlan,
	loopRun loop.Run,
	budgetExceeded bool,
) bool {
	if !plan.GenerationInFlight {
		return false
	}
	return plan.Terminal != nil || budgetExceeded || loopRun.PauseRequested
}

type coordinatorResultContextPayload struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	Name        string `json:"loop_name,omitempty"`
	ParentRunID string `json:"parent_loop_run_id,omitempty"`
	Generation  int    `json:"generation,omitempty"`
	Status      string `json:"status,omitempty"`
}

func coordinatorResultContext(
	run loop.Run,
	generation int,
	status loop.Status,
) (json.RawMessage, error) {
	payload := coordinatorResultContextPayload{
		WorkspaceID: string(run.WorkspaceID),
		Name:        strings.TrimSpace(run.LoopName),
		ParentRunID: string(run.ParentLoopRunID),
		Generation:  generation,
		Status:      string(status),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("store: marshal coordinator context: %w", err)
	}
	return data, nil
}

func updateCoordinatorResultContext(
	result *taskpkg.CoordinatorCompletionResult,
	generation int,
	status loop.Status,
) error {
	var payload coordinatorResultContextPayload
	if len(result.Context) > 0 {
		if err := json.Unmarshal(result.Context, &payload); err != nil {
			return fmt.Errorf("store: decode coordinator context: %w", err)
		}
	}
	if generation > 0 {
		payload.Generation = generation
	}
	if strings.TrimSpace(string(status)) != "" {
		payload.Status = string(status)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("store: marshal coordinator context: %w", err)
	}
	result.Context = data
	return nil
}

func applyCoordinatorBudgetBoundary(
	ctx context.Context,
	exec taskSQLExecutor,
	completion *taskpkg.CoordinatorCompletion,
	snapshot taskpkg.GenerationSnapshot,
	loopRun loop.Run,
	result *taskpkg.CoordinatorCompletionResult,
) error {
	budgetStatus := loop.StatusExhausted
	if string(loopRun.BudgetOnExceeded) == "escalate" {
		budgetStatus = loop.StatusNeedsApproval
	}
	if err := updateLoopBoundaryStatusWithExecutor(
		ctx,
		exec,
		loopRun,
		budgetStatus,
		loop.TransitionCauseBudget,
		completion.Now,
		snapshot.Generation,
	); err != nil {
		return err
	}
	if budgetStatus.Terminal() {
		if err := sweepOrphanedLoopOutputBlobsWithExecutor(ctx, exec); err != nil {
			return err
		}
	}
	if err := updateCoordinatorResultContext(result, snapshot.Generation, budgetStatus); err != nil {
		return err
	}
	result.Terminal = loopStatusIsTerminalOrApproval(budgetStatus)
	return nil
}

func applyCoordinatorTerminalBoundary(
	ctx context.Context,
	exec taskSQLExecutor,
	completion *taskpkg.CoordinatorCompletion,
	snapshot taskpkg.GenerationSnapshot,
	loopRun loop.Run,
	result *taskpkg.CoordinatorCompletionResult,
) error {
	terminalStatus := loop.Status(completion.Plan.Terminal.Status)
	if !terminalStatus.Valid() {
		return fmt.Errorf(
			"%w: coordinator terminal status is invalid: %q",
			taskpkg.ErrValidation,
			completion.Plan.Terminal.Status,
		)
	}
	cause := loop.TransitionCause(strings.TrimSpace(completion.Plan.Terminal.Cause))
	if strings.TrimSpace(string(cause)) == "" {
		cause = loop.TransitionCauseContract
	}
	if err := updateLoopBoundaryStatusWithExecutor(
		ctx,
		exec,
		loopRun,
		terminalStatus,
		cause,
		completion.Now,
		snapshot.Generation,
	); err != nil {
		return err
	}
	if terminalStatus.Terminal() {
		if err := sweepOrphanedLoopOutputBlobsWithExecutor(ctx, exec); err != nil {
			return err
		}
	}
	if err := updateCoordinatorResultContext(result, snapshot.Generation, terminalStatus); err != nil {
		return err
	}
	result.Terminal = loopStatusIsTerminalOrApproval(terminalStatus)
	return nil
}

func applyCoordinatorYieldBoundary(
	ctx context.Context,
	exec taskSQLExecutor,
	snapshot taskpkg.GenerationSnapshot,
	result *taskpkg.CoordinatorCompletionResult,
) error {
	if snapshot.Generation > 0 {
		if err := updateLoopGenerationWithExecutor(ctx, exec, result.LoopRunID, snapshot.Generation); err != nil {
			return err
		}
		if err := updateCoordinatorResultContext(result, snapshot.Generation, ""); err != nil {
			return err
		}
	}
	return nil
}

func applyCoordinatorPauseBoundary(
	ctx context.Context,
	exec taskSQLExecutor,
	completion *taskpkg.CoordinatorCompletion,
	snapshot taskpkg.GenerationSnapshot,
	loopRun loop.Run,
	result *taskpkg.CoordinatorCompletionResult,
) error {
	if err := updateLoopBoundaryStatusWithExecutor(
		ctx,
		exec,
		loopRun,
		loop.StatusPaused,
		loop.TransitionCausePauseBoundary,
		completion.Now,
		snapshot.Generation,
	); err != nil {
		return err
	}
	if err := updateCoordinatorResultContext(result, snapshot.Generation, loop.StatusPaused); err != nil {
		return err
	}
	result.Paused = true
	return nil
}

func (g *GlobalDB) applyCoordinatorContinueBoundaryWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	completion *taskpkg.CoordinatorCompletion,
	current *taskpkg.Run,
	snapshot taskpkg.GenerationSnapshot,
	postReserveSnapshot *taskpkg.GenerationSnapshot,
	finalizer taskpkg.GenerationStateFinalizer,
	result *taskpkg.CoordinatorCompletionResult,
) error {
	enqueued, err := g.reserveCoordinatorPlanRunsWithExecutor(
		ctx,
		exec,
		completion.Plan,
		*current,
		completion.Actor.Origin,
		completion.Now,
	)
	if err != nil {
		return err
	}
	if postReserveSnapshot != nil {
		if err := finalizer.WriteGenerationSnapshot(ctx, exec, *postReserveSnapshot); err != nil {
			return err
		}
		snapshot = *postReserveSnapshot
	}
	result.EnqueuedRuns = enqueued
	return applyCoordinatorYieldBoundary(ctx, exec, snapshot, result)
}

func (g *GlobalDB) applyCoordinatorWatchReadyBoundaryWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	completion *taskpkg.CoordinatorCompletion,
	current *taskpkg.Run,
	snapshot taskpkg.GenerationSnapshot,
	postReserveSnapshot *taskpkg.GenerationSnapshot,
	finalizer taskpkg.GenerationStateFinalizer,
	loopRun loop.Run,
	result *taskpkg.CoordinatorCompletionResult,
) error {
	if err := updateLoopBoundaryStatusWithExecutor(
		ctx,
		exec,
		loopRun,
		loop.StatusRunning,
		loop.TransitionCauseWatchPoll,
		completion.Now,
		snapshot.Generation,
	); err != nil {
		return err
	}
	if err := updateCoordinatorResultContext(result, snapshot.Generation, loop.StatusRunning); err != nil {
		return err
	}
	return g.applyCoordinatorContinueBoundaryWithExecutor(
		ctx,
		exec,
		completion,
		current,
		snapshot,
		postReserveSnapshot,
		finalizer,
		result,
	)
}

func loopBudgetExceeded(run loop.Run, tokensUsed int64, now time.Time) bool {
	if run.BudgetTokens > 0 && tokensUsed >= int64(run.BudgetTokens) {
		return true
	}
	if run.BudgetWallSec <= 0 || run.CreatedAt.IsZero() {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return !now.UTC().Before(run.CreatedAt.UTC().Add(time.Duration(run.BudgetWallSec) * time.Second))
}

func applyCoordinatorRunStopsWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	specs []taskpkg.CoordinatorStopSpec,
	now time.Time,
) error {
	stoppedAny := false
	for _, spec := range specs {
		normalized := spec.Normalize()
		child, err := getLoopRunByIDWithExecutor(ctx, exec, loop.RunID(normalized.LoopRunID))
		if err != nil {
			return err
		}
		if child.Status.Terminal() {
			continue
		}
		if err := updateLoopBoundaryStatusWithExecutor(
			ctx,
			exec,
			child,
			loop.StatusFailed,
			loop.TransitionCauseOperatorStop,
			now,
			child.Generation,
		); err != nil {
			return err
		}
		stoppedAny = true
	}
	if stoppedAny {
		if err := sweepOrphanedLoopOutputBlobsWithExecutor(ctx, exec); err != nil {
			return err
		}
	}
	return nil
}

func (g *GlobalDB) ensureCoordinatorPlanTasksWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	plan taskpkg.CoordinatorCompletionPlan,
	current taskpkg.Run,
	origin taskpkg.Origin,
	now time.Time,
) error {
	if len(plan.NodeTasks) == 0 && len(plan.Dependencies) == 0 {
		return nil
	}
	parentTask, err := g.getTaskWithExecutor(ctx, exec, current.TaskID)
	if err != nil {
		return err
	}
	for _, spec := range plan.NodeTasks {
		if err := g.createCoordinatorTaskIfMissingWithExecutor(ctx, exec, spec, parentTask, origin, now); err != nil {
			return err
		}
	}
	for _, spec := range plan.Dependencies {
		if err := g.createCoordinatorDependencyIfMissingWithExecutor(ctx, exec, spec, now); err != nil {
			return err
		}
	}
	return nil
}

func (g *GlobalDB) createCoordinatorTaskIfMissingWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	spec taskpkg.CoordinatorTaskSpec,
	parent taskpkg.Task,
	origin taskpkg.Origin,
	now time.Time,
) error {
	normalized := spec.Normalize()
	if _, err := g.getTaskWithExecutor(ctx, exec, normalized.TaskID); err == nil {
		return nil
	} else if !errors.Is(err, taskpkg.ErrTaskNotFound) {
		return err
	}
	taskRecord, err := g.normalizeTaskForCreate(
		coordinatorTaskRecord(normalized, parent, origin, now),
	)
	if err != nil {
		return err
	}
	if err := g.ensureCoordinatorTaskCreateReferencesWithExecutor(ctx, exec, taskRecord); err != nil {
		return err
	}
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO tasks (
			id, identifier, scope, workspace_id, parent_task_id, network_channel, title, description,
			priority, max_attempts, auto_enqueue_on_ready, status, approval_policy, approval_state,
			owner_kind, owner_ref, created_by_kind, created_by_ref, origin_kind, origin_ref,
			created_at, updated_at, closed_at, paused, paused_by, paused_at, paused_reason, wake_creator, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		taskRecord.ID,
		store.NullableString(taskRecord.Identifier),
		string(taskRecord.Scope),
		store.NullableString(taskRecord.WorkspaceID),
		store.NullableString(taskRecord.ParentTaskID),
		store.NullableString(taskRecord.NetworkChannel),
		taskRecord.Title,
		store.NullableString(taskRecord.Description),
		string(taskRecord.Priority),
		taskRecord.MaxAttempts,
		taskBoolToInt(taskRecord.AutoEnqueueOnReady),
		string(taskRecord.Status),
		string(taskRecord.ApprovalPolicy),
		string(taskRecord.ApprovalState),
		taskOwnerKindValue(taskRecord.Owner),
		taskOwnerRefValue(taskRecord.Owner),
		string(taskRecord.CreatedBy.Kind),
		taskRecord.CreatedBy.Ref,
		string(taskRecord.Origin.Kind),
		taskRecord.Origin.Ref,
		store.FormatTimestamp(taskRecord.CreatedAt),
		store.FormatTimestamp(taskRecord.UpdatedAt),
		nullableTaskTimestamp(taskRecord.ClosedAt),
		taskBoolToInt(taskRecord.Paused),
		taskRecord.PausedBy,
		nullableTaskTimestamp(taskRecord.PausedAt),
		taskRecord.PausedReason,
		taskBoolToInt(taskRecord.WakeCreator),
		nullableTaskJSON(taskRecord.Metadata),
	); err != nil {
		return fmt.Errorf("store: create coordinator node task %q: %w", taskRecord.ID, err)
	}
	return nil
}

func coordinatorTaskRecord(
	spec taskpkg.CoordinatorTaskSpec,
	parent taskpkg.Task,
	origin taskpkg.Origin,
	now time.Time,
) taskpkg.Task {
	return taskpkg.Task{
		ID:                 spec.TaskID,
		Scope:              parent.Scope,
		WorkspaceID:        parent.WorkspaceID,
		ParentTaskID:       parent.ID,
		NetworkChannel:     parent.NetworkChannel,
		Title:              spec.Title,
		Description:        spec.Description,
		Priority:           parent.Priority,
		MaxAttempts:        parent.MaxAttempts,
		Status:             taskpkg.TaskStatusReady,
		ApprovalPolicy:     taskpkg.ApprovalPolicyNone,
		ApprovalState:      taskpkg.ApprovalStateNotRequired,
		AutoEnqueueOnReady: false,
		CreatedBy: taskpkg.ActorIdentity{
			Kind: taskpkg.ActorKindDaemon,
			Ref:  loopCoordinatorActorRef,
		},
		Origin:    origin,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  spec.Metadata,
	}
}

func (g *GlobalDB) ensureCoordinatorTaskCreateReferencesWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	record taskpkg.Task,
) error {
	if err := taskpkg.ValidateScopeBinding(record.Scope, record.WorkspaceID, "task", "workspace_id"); err != nil {
		return err
	}
	if record.Scope == taskpkg.ScopeWorkspace {
		if err := g.ensureWorkspaceExistsWithExecutor(ctx, exec, record.WorkspaceID); err != nil {
			return err
		}
	}
	if strings.TrimSpace(record.ParentTaskID) != "" {
		if err := g.ensureTaskExistsWithExecutor(ctx, exec, record.ParentTaskID); err != nil {
			return err
		}
	}
	return nil
}

func (g *GlobalDB) ensureWorkspaceExistsWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	workspaceID string,
) error {
	trimmedID := strings.TrimSpace(workspaceID)
	if trimmedID == "" {
		return nil
	}
	var exists int
	if err := exec.QueryRowContext(ctx, `SELECT 1 FROM workspaces WHERE id = ?`, trimmedID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return aghworkspace.ErrWorkspaceNotFound
		}
		return fmt.Errorf("store: check workspace %q exists: %w", trimmedID, err)
	}
	return nil
}

func (g *GlobalDB) createCoordinatorDependencyIfMissingWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	spec taskpkg.CoordinatorDependencySpec,
	now time.Time,
) error {
	normalizedSpec := spec.Normalize()
	dependency, err := g.normalizeTaskDependencyForCreate(taskpkg.Dependency{
		TaskID:          normalizedSpec.TaskID,
		DependsOnTaskID: normalizedSpec.DependsOnTaskID,
		Kind:            normalizedSpec.Kind,
		CreatedAt:       now,
	})
	if err != nil {
		return err
	}
	if err := g.ensureTaskExistsWithExecutor(ctx, exec, dependency.TaskID); err != nil {
		return err
	}
	if err := g.ensureTaskExistsWithExecutor(ctx, exec, dependency.DependsOnTaskID); err != nil {
		return err
	}
	exists, err := g.taskDependencyExists(ctx, exec, dependency)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	count, err := g.countDependenciesWithExecutor(ctx, exec, dependency.TaskID)
	if err != nil {
		return err
	}
	if err := taskpkg.ValidateDependencyCount(count + 1); err != nil {
		return err
	}
	hasPath, err := g.hasDependencyPathWithExecutor(
		ctx,
		exec,
		dependency.DependsOnTaskID,
		dependency.TaskID,
	)
	if err != nil {
		return err
	}
	if hasPath {
		return fmt.Errorf(
			"%w: adding dependency %q -> %q would create a cycle",
			taskpkg.ErrCycleDetected,
			dependency.TaskID,
			dependency.DependsOnTaskID,
		)
	}
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO task_dependencies (task_id, depends_on_task_id, kind, created_at)
		 VALUES (?, ?, ?, ?)`,
		dependency.TaskID,
		dependency.DependsOnTaskID,
		string(dependency.Kind),
		store.FormatTimestamp(dependency.CreatedAt),
	); err != nil {
		return fmt.Errorf(
			"store: create coordinator dependency %q -> %q: %w",
			dependency.TaskID,
			dependency.DependsOnTaskID,
			err,
		)
	}
	return nil
}

func completeCoordinatorRunWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	run taskpkg.Run,
	now time.Time,
) error {
	resultJSON, err := json.Marshal(
		map[string]string{taskRunResultKindKey: taskpkg.RunKindCoordinator.String()},
	)
	if err != nil {
		return fmt.Errorf("store: marshal coordinator result: %w", err)
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE task_runs
		 SET status = ?, lease_until = NULL, heartbeat_at = NULL, claim_token = NULL,
		     ended_at = ?, error = NULL, result_json = ?
		 WHERE id = ? AND claim_token_hash = ?`,
		taskpkg.TaskRunStatusCompleted.String(),
		store.FormatTimestamp(now),
		string(resultJSON),
		run.ID,
		run.ClaimTokenHash,
	)
	if err != nil {
		return fmt.Errorf("store: complete coordinator run %q: %w", run.ID, err)
	}
	return requireRowsAffected(result, taskpkg.ErrTaskRunNotFound, run.ID, "coordinator run lease")
}

func refreshLoopTokensUsedWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	loopRunID string,
) (int64, error) {
	var tokensUsed int64
	if err := exec.QueryRowContext(
		ctx,
		`SELECT COALESCE(SUM(tokens_used), 0) FROM task_runs WHERE loop_run_id = ?`,
		loopRunID,
	).Scan(&tokensUsed); err != nil {
		return 0, fmt.Errorf("store: sum loop run %q task tokens: %w", loopRunID, err)
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_runs SET tokens_used = ? WHERE id = ?`,
		tokensUsed,
		loopRunID,
	)
	if err != nil {
		return 0, fmt.Errorf("store: refresh loop run %q tokens_used: %w", loopRunID, err)
	}
	if err := requireRowsAffected(result, loop.ErrRunNotFound, loopRunID, "loop run"); err != nil {
		return 0, err
	}
	return tokensUsed, nil
}

func updateLoopBoundaryStatusWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	current loop.Run,
	to loop.Status,
	cause loop.TransitionCause,
	at time.Time,
	generation int,
) error {
	if current.Status == to {
		return updateLoopGenerationWithExecutor(ctx, exec, string(current.ID), generation)
	}
	if !to.Valid() {
		return fmt.Errorf("%w: loop status is invalid: %q", loop.ErrValidation, to)
	}
	if strings.TrimSpace(string(cause)) == "" {
		return fmt.Errorf("%w: transition cause is required", loop.ErrValidation)
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_runs
		 SET status = ?,
		     pause_requested = 0,
		     generation = CASE WHEN ? > generation THEN ? ELSE generation END
		 WHERE id = ? AND status = ?`,
		string(to),
		generation,
		generation,
		string(current.ID),
		string(current.Status),
	)
	if err != nil {
		return fmt.Errorf(
			"store: transition loop run %q at coordinator boundary: %w",
			current.ID,
			err,
		)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"store: rows affected for loop run %q coordinator boundary: %w",
			current.ID,
			err,
		)
	}
	if affected == 0 {
		return fmt.Errorf(
			"%w: run_id=%s from=%s to=%s",
			loop.ErrTransitionConflict,
			current.ID,
			current.Status,
			to,
		)
	}
	return appendLoopRunStatusEvent(
		ctx,
		exec,
		current.ID,
		current.WorkspaceID,
		current.Status,
		to,
		cause,
		at,
	)
}

func updateLoopGenerationWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	loopRunID string,
	generation int,
) error {
	if generation <= 0 {
		return nil
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_runs
		 SET generation = CASE WHEN ? > generation THEN ? ELSE generation END
		 WHERE id = ?`,
		generation,
		generation,
		loopRunID,
	)
	if err != nil {
		return fmt.Errorf("store: update loop run %q generation: %w", loopRunID, err)
	}
	return requireRowsAffected(result, loop.ErrRunNotFound, loopRunID, "loop run")
}

func loopStatusIsTerminalOrApproval(status loop.Status) bool {
	switch status {
	case loop.StatusDone,
		loop.StatusNoOp,
		loop.StatusBlocked,
		loop.StatusFailed,
		loop.StatusExhausted,
		loop.StatusStalled,
		loop.StatusNeedsApproval:
		return true
	default:
		return false
	}
}

func (g *GlobalDB) reserveCoordinatorPlanRunsWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	plan taskpkg.CoordinatorCompletionPlan,
	current taskpkg.Run,
	origin taskpkg.Origin,
	queuedAt time.Time,
) ([]taskpkg.Run, error) {
	specs := make([]taskpkg.EnqueueSpec, 0, len(plan.NodeRuns)+1)
	specs = append(specs, plan.NodeRuns...)
	if plan.NextCoordinator != nil {
		next := *plan.NextCoordinator
		if next.RunKind.Normalize() == taskpkg.RunKindUnknown {
			next.RunKind = taskpkg.RunKindCoordinator
		}
		if strings.TrimSpace(next.LoopRunID) == "" {
			next.LoopRunID = current.LoopRunID
		}
		specs = append(specs, next)
	}

	enqueued := make([]taskpkg.Run, 0, len(specs))
	for _, spec := range specs {
		reservation := coordinatorPlanRunReservation(spec, current, origin, queuedAt)
		_, run, existing, err := g.reserveQueuedRunWithExecutor(ctx, exec, reservation)
		if err != nil {
			return nil, err
		}
		if existing {
			continue
		}
		enqueued = append(enqueued, run)
	}
	return enqueued, nil
}

func coordinatorPlanRunReservation(
	spec taskpkg.EnqueueSpec,
	current taskpkg.Run,
	origin taskpkg.Origin,
	queuedAt time.Time,
) queuedRunReservationInput {
	normalized := spec.Normalize()
	runID := strings.TrimSpace(normalized.RunID)
	if runID == "" {
		runID = store.NewID("run")
	}
	if strings.TrimSpace(normalized.LoopRunID) == "" {
		normalized.LoopRunID = current.LoopRunID
	}
	return queuedRunReservationInput{
		taskID:             normalized.TaskID,
		runID:              runID,
		runKind:            normalized.RunKind,
		loopRunID:          normalized.LoopRunID,
		idempotencyKey:     normalized.IdempotencyKey,
		origin:             origin,
		requestedChannel:   normalized.NetworkChannel,
		designationGroupID: normalized.DesignationGroupID,
		metadata:           normalizeTaskJSON(normalized.Metadata),
		queuedAt:           queuedAt,
	}
}
