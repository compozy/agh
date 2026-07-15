package task

import (
	"context"

	"errors"
	"fmt"

	"strings"

	"time"
)

// PublishTask transitions one durable draft into manager-owned runnable reconciliation,
// then enqueues the executable run for the explicit execution boundary.
func (m *Service) PublishTask(
	ctx context.Context,
	id string,
	req ExecutionRequest,
	actor ActorContext,
) (*Execution, error) {
	return m.executeTaskBoundary(ctx, id, req, ExecutionActionPublish, actor)
}

// StartTask enqueues one executable run for an already-created task.
func (m *Service) StartTask(
	ctx context.Context,
	id string,
	req ExecutionRequest,
	actor ActorContext,
) (*Execution, error) {
	return m.executeTaskBoundary(ctx, id, req, ExecutionActionStart, actor)
}

// ApproveTask records one approval decision for a manual-approval task that is
// currently awaiting a decision, then enqueues the approved run.
func (m *Service) ApproveTask(
	ctx context.Context,
	id string,
	req ExecutionRequest,
	actor ActorContext,
) (*Execution, error) {
	return m.executeTaskBoundary(ctx, id, req, ExecutionActionApproval, actor)
}

// RejectTask records one rejection decision for a manual-approval task that is
// currently awaiting a decision and reconciles the resulting task status.
func (m *Service) RejectTask(ctx context.Context, id string, actor ActorContext) (*Task, error) {
	return m.transitionTaskApproval(ctx, id, ApprovalStateRejected, taskEventRejected, actor)
}

func (m *Service) executeTaskBoundary(
	ctx context.Context,
	id string,
	req ExecutionRequest,
	action ExecutionAction,
	actor ActorContext,
) (*Execution, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: task id is required", ErrValidation)
	}
	normalizedReq, err := normalizeTaskExecutionRequest(req)
	if err != nil {
		return nil, err
	}
	if err := m.validateNetworkChannel("task_execution.network_channel", normalizedReq.NetworkChannel); err != nil {
		return nil, err
	}
	idempotencyKey := taskExecutionIdempotencyKey(trimmedID, action, normalizedReq.IdempotencyKey)
	if existing, ok, err := m.taskExecutionFromIdempotency(ctx, trimmedID, idempotencyKey, action, actor); err != nil {
		return nil, err
	} else if ok {
		return existing, nil
	}

	approvalTask, err := m.prepareTaskExecutionBoundary(ctx, trimmedID, action, actor)
	if err != nil {
		return nil, err
	}

	if execution, ok, err := m.approvalExistingRunExecution(
		ctx,
		approvalTask,
	); err != nil {
		return nil, err
	} else if ok {
		return execution, nil
	}

	if execution, ok, err := m.approvalAutoEnqueueExecution(
		ctx,
		approvalTask,
		normalizedReq,
		idempotencyKey,
		actor,
	); err != nil {
		return nil, err
	} else if ok {
		return execution, nil
	}

	run, err := m.EnqueueRun(ctx, EnqueueRun{
		TaskID:         trimmedID,
		IdempotencyKey: idempotencyKey,
		NetworkChannel: normalizedReq.NetworkChannel,
		Metadata:       normalizedReq.Metadata,
	}, actor)
	if err != nil {
		return nil, err
	}
	taskRecord, err := m.store.GetTask(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}
	return &Execution{
		Task:   taskRecord,
		Run:    *run,
		Action: action,
	}, nil
}

func (m *Service) approvalExistingRunExecution(
	ctx context.Context,
	approvalTask *Task,
) (*Execution, bool, error) {
	if approvalTask == nil {
		return nil, false, nil
	}
	runs, err := m.store.ListTaskRuns(ctx, RunQuery{TaskID: approvalTask.ID})
	if err != nil {
		return nil, false, fmt.Errorf(
			"task: list runs while approving task %q: %w",
			approvalTask.ID,
			err,
		)
	}
	var openRun *Run
	for idx := range runs {
		if isTerminalRunStatus(runs[idx].Status) {
			continue
		}
		if openRun != nil {
			return nil, false, fmt.Errorf(
				"%w: task %q has multiple open runs %q and %q",
				ErrConflict,
				approvalTask.ID,
				openRun.ID,
				runs[idx].ID,
			)
		}
		openRun = &runs[idx]
	}
	if openRun == nil {
		return nil, false, nil
	}
	if strings.TrimSpace(openRun.TaskID) != approvalTask.ID {
		return nil, false, fmt.Errorf(
			"%w: task %q current run %q belongs to task %q",
			ErrValidation,
			approvalTask.ID,
			openRun.ID,
			openRun.TaskID,
		)
	}
	taskRecord, err := m.store.GetTask(ctx, approvalTask.ID)
	if err != nil {
		return nil, false, err
	}
	return &Execution{
		Task:        taskRecord,
		Run:         *openRun,
		Action:      ExecutionActionApproval,
		ExistingRun: true,
	}, true, nil
}

func (m *Service) prepareTaskExecutionBoundary(
	ctx context.Context,
	taskID string,
	action ExecutionAction,
	actor ActorContext,
) (*Task, error) {
	switch action {
	case ExecutionActionPublish:
		_, err := m.publishTaskIntent(ctx, taskID, actor)
		return nil, err
	case ExecutionActionApproval:
		return m.transitionTaskApproval(
			ctx,
			taskID,
			ApprovalStateApproved,
			taskEventApproved,
			actor,
		)
	case ExecutionActionStart:
		taskRecord, err := m.store.GetTask(ctx, taskID)
		if err != nil {
			return nil, err
		}
		return nil, m.ensureTaskExecutable(ctx, taskRecord)
	default:
		return nil, fmt.Errorf("%w: unsupported task execution action %q", ErrValidation, action)
	}
}

func (m *Service) approvalAutoEnqueueExecution(
	ctx context.Context,
	approvalTask *Task,
	req ExecutionRequest,
	executionIdempotencyKey string,
	actor ActorContext,
) (*Execution, bool, error) {
	if approvalTask == nil ||
		!approvalTask.AutoEnqueueOnReady ||
		approvalTask.Status.Normalize() != TaskStatusReady {
		return nil, false, nil
	}
	trigger := autoEnqueueTrigger{
		Kind: autoEnqueueTriggerApprovalGranted,
		Ref:  approvalTask.ID,
	}
	run, err := m.EnqueueRun(ctx, EnqueueRun{
		TaskID:         approvalTask.ID,
		IdempotencyKey: trigger.idempotencyKey(approvalTask.ID),
		NetworkChannel: req.NetworkChannel,
		Metadata:       req.Metadata,
	}, actor)
	if err != nil {
		return nil, false, err
	}
	if err := m.saveApprovalIdempotencyAlias(
		ctx,
		approvalTask.ID,
		run.ID,
		executionIdempotencyKey,
		actor,
	); err != nil {
		return nil, false, err
	}
	taskRecord, err := m.store.GetTask(ctx, run.TaskID)
	if err != nil {
		return nil, false, err
	}
	m.recordAutoEnqueueTriggered(ctx, *approvalTask, run, trigger, actor)
	return &Execution{
		Task:   taskRecord,
		Run:    *run,
		Action: ExecutionActionApproval,
	}, true, nil
}

func (m *Service) saveApprovalIdempotencyAlias(
	ctx context.Context,
	taskID string,
	runID string,
	idempotencyKey string,
	actor ActorContext,
) error {
	if err := m.store.SaveTaskRunIdempotency(ctx, RunIdempotency{
		IdempotencyKey: idempotencyKey,
		RunID:          runID,
		Origin:         actor.Origin,
		CreatedAt:      m.now().UTC(),
	}); err != nil {
		return fmt.Errorf(
			"task: save approval idempotency alias for task %q run %q: %w",
			taskID,
			runID,
			err,
		)
	}
	return nil
}

func (m *Service) taskExecutionFromIdempotency(
	ctx context.Context,
	taskID string,
	idempotencyKey string,
	action ExecutionAction,
	actor ActorContext,
) (*Execution, bool, error) {
	run, err := m.store.GetTaskRunByIdempotencyKey(ctx, idempotencyKey, actor.Origin)
	switch {
	case errors.Is(err, ErrTaskRunIdempotencyNotFound):
		return nil, false, nil
	case err != nil:
		return nil, false, err
	}
	if strings.TrimSpace(run.TaskID) != taskID {
		return nil, false, fmt.Errorf(
			"%w: idempotency key %q is already bound to task %q",
			ErrValidation,
			idempotencyKey,
			run.TaskID,
		)
	}
	taskRecord, err := m.store.GetTask(ctx, run.TaskID)
	if err != nil {
		return nil, false, err
	}
	return &Execution{
		Task:        taskRecord,
		Run:         run,
		Action:      action,
		ExistingRun: true,
	}, true, nil
}

func (m *Service) publishTaskIntent(
	ctx context.Context,
	id string,
	actor ActorContext,
) (*Task, error) {
	record, err := m.store.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	if record.Status.Normalize() != TaskStatusDraft {
		return nil, fmt.Errorf(
			"%w: task %q cannot publish from %q",
			ErrInvalidStatusTransition,
			record.ID,
			record.Status,
		)
	}

	candidate := record
	candidate.Status = TaskStatusPending
	if err := m.ensureTaskExecutable(ctx, candidate); err != nil {
		return nil, err
	}

	previousStatus := record.Status
	record.Status = TaskStatusPending
	record.UpdatedAt = m.now().UTC()
	record.ClosedAt = time.Time{}
	if err := m.store.UpdateTask(ctx, record, actor); err != nil {
		return nil, err
	}
	m.dispatchTaskStatusChanged(ctx, record, previousStatus, record.Status, actor)

	reconciled, err := m.reconcileTaskCascade(ctx, record.ID, actor)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, reconciled.ID, "", taskEventPublished, actor, publishedTaskPayload{
		PreviousStatus: TaskStatusDraft,
		Status:         reconciled.Status,
		ApprovalState:  reconciled.ApprovalState,
	}); err != nil {
		return nil, err
	}

	return &reconciled, nil
}

func (m *Service) transitionTaskApproval(
	ctx context.Context,
	id string,
	target ApprovalState,
	eventType string,
	actor ActorContext,
) (*Task, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: task id is required", ErrValidation)
	}

	record, err := m.store.GetTask(ctx, trimmedID)
	if err != nil {
		return nil, err
	}

	previousApprovalState := normalizeApprovalStateOrDefault(
		record.ApprovalPolicy,
		record.ApprovalState,
	)
	if !taskApprovalDecisionAllowed(record, target) {
		return nil, fmt.Errorf(
			"%w: task %q cannot transition approval from %q to %q",
			ErrInvalidStatusTransition,
			record.ID,
			previousApprovalState,
			target.Normalize(),
		)
	}

	record.ApprovalState = target.Normalize()
	record.UpdatedAt = m.now().UTC()
	record.ClosedAt = time.Time{}
	if err := m.store.UpdateTask(ctx, record, actor); err != nil {
		return nil, err
	}

	reconciled, err := m.reconcileTaskCascade(ctx, record.ID, actor)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, reconciled.ID, "", eventType, actor, approvalDecisionTaskPayload{
		PreviousApprovalState: previousApprovalState,
		ApprovalState:         reconciled.ApprovalState,
		Status:                reconciled.Status,
	}); err != nil {
		return nil, err
	}

	return &reconciled, nil
}

func taskApprovalDecisionAllowed(record Task, target ApprovalState) bool {
	normalizedPolicy := normalizeApprovalPolicyOrDefault(record.ApprovalPolicy)
	if normalizedPolicy != ApprovalPolicyManual {
		return false
	}
	switch target.Normalize() {
	case ApprovalStateApproved, ApprovalStateRejected:
	default:
		return false
	}
	return normalizeApprovalStateOrDefault(
		normalizedPolicy,
		record.ApprovalState,
	) == ApprovalStatePending
}
