package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/store"
)

const (
	taskEventCreated           = "task.created"
	taskEventUpdated           = "task.updated"
	taskEventCancelled         = "task.cancelled"
	taskEventChildCreated      = "task.child_created"
	taskEventDependencyAdded   = "task.dependency_added"
	taskEventDependencyRemoved = "task.dependency_removed"
	taskEventRunEnqueued       = "task.run_enqueued"
	taskEventRunClaimed        = "task.run_claimed"
	taskEventRunStarting       = "task.run_starting"
	taskEventRunSessionBound   = "task.run_session_bound"
	taskEventRunStarted        = "task.run_started"
	taskEventRunCompleted      = "task.run_completed"
	taskEventRunFailed         = "task.run_failed"
	taskEventRunCancelled      = "task.run_cancelled"
	taskEventRunForceStopped   = "task.run_force_stopped"
	taskEventRunRecovered      = "task.run_recovered"
	taskEventRunRejected       = "task.run_rejected"
)

// Option customizes TaskManager construction.
type Option func(*managerOptions)

type managerOptions struct {
	store             Store
	sessions          SessionExecutor
	channelValidator  func(string) error
	now               func() time.Time
	newID             func(prefix string) string
	cancelGracePeriod time.Duration
}

// TaskManager centralizes canonical task-domain creation, mutation, read, and
// graph-management rules above the persistence layer.
type TaskManager struct {
	store             Store
	sessions          SessionExecutor
	channelValidator  func(string) error
	now               func() time.Time
	newID             func(prefix string) string
	cancelGracePeriod time.Duration
}

var _ Manager = (*TaskManager)(nil)

// WithStore injects the durable task-domain store consumed by the manager.
func WithStore(store Store) Option {
	return func(opts *managerOptions) {
		opts.store = store
	}
}

// WithSessionExecutor injects the runtime session bridge used by later
// task-run lifecycle operations.
func WithSessionExecutor(sessions SessionExecutor) Option {
	return func(opts *managerOptions) {
		opts.sessions = sessions
	}
}

// WithNetworkChannelValidator injects the active channel validator used to
// check task and run bindings without coupling the task package to the network
// runtime implementation.
func WithNetworkChannelValidator(validator func(string) error) Option {
	return func(opts *managerOptions) {
		opts.channelValidator = validator
	}
}

// WithManagerNow overrides the manager clock for deterministic tests.
func WithManagerNow(now func() time.Time) Option {
	return func(opts *managerOptions) {
		opts.now = now
	}
}

// WithIDGenerator overrides identifier generation for deterministic tests.
func WithIDGenerator(newID func(prefix string) string) Option {
	return func(opts *managerOptions) {
		opts.newID = newID
	}
}

// WithCancelGracePeriod overrides the cooperative-stop grace period used before
// requesting forced session termination during task-driven cancellation.
func WithCancelGracePeriod(timeout time.Duration) Option {
	return func(opts *managerOptions) {
		opts.cancelGracePeriod = timeout
	}
}

// NewManager constructs one task-domain manager with the supplied dependencies.
func NewManager(opts ...Option) (*TaskManager, error) {
	options := managerOptions{
		now: func() time.Time {
			return time.Now().UTC()
		},
		newID: store.NewID,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	if options.store == nil {
		return nil, fmt.Errorf("task: manager store is required")
	}
	if options.now == nil {
		return nil, fmt.Errorf("task: manager clock is required")
	}
	if options.newID == nil {
		return nil, fmt.Errorf("task: manager id generator is required")
	}
	if options.cancelGracePeriod < 0 {
		return nil, fmt.Errorf("task: manager cancel grace period must be zero or positive")
	}

	return &TaskManager{
		store:             options.store,
		sessions:          options.sessions,
		channelValidator:  options.channelValidator,
		now:               options.now,
		newID:             options.newID,
		cancelGracePeriod: options.cancelGracePeriod,
	}, nil
}

// CreateTask derives one canonical task record from trusted actor context and
// persists the corresponding immutable audit event.
func (m *TaskManager) CreateTask(ctx context.Context, spec CreateTask, actor ActorContext) (*Task, error) {
	if err := requireCreateAuthority(actor, spec.Scope); err != nil {
		return nil, err
	}

	normalizedSpec, err := normalizeCreateTaskSpec(spec)
	if err != nil {
		return nil, err
	}
	if err := m.validateParentConstraints(ctx, normalizedSpec); err != nil {
		return nil, err
	}
	if err := m.validateNetworkChannel("create_task.network_channel", normalizedSpec.NetworkChannel); err != nil {
		return nil, err
	}

	now := m.now().UTC()
	record := Task{
		ID:             normalizedSpec.ID,
		Identifier:     normalizedSpec.Identifier,
		Scope:          normalizedSpec.Scope,
		WorkspaceID:    normalizedSpec.WorkspaceID,
		ParentTaskID:   normalizedSpec.ParentTaskID,
		NetworkChannel: normalizedSpec.NetworkChannel,
		Title:          normalizedSpec.Title,
		Description:    normalizedSpec.Description,
		Status:         TaskStatusReady,
		Owner:          cloneOwnership(normalizedSpec.Owner),
		CreatedBy:      actor.Actor,
		Origin:         actor.Origin,
		CreatedAt:      now,
		UpdatedAt:      now,
		Metadata:       cloneRawJSON(normalizedSpec.Metadata),
	}
	if strings.TrimSpace(record.ID) == "" {
		record.ID = m.newID("task")
	}
	if err := record.Validate(); err != nil {
		return nil, err
	}
	if err := m.store.CreateTask(ctx, record); err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, record.ID, "", taskEventCreated, actor, createdTaskPayload{
		Scope:          record.Scope,
		WorkspaceID:    record.WorkspaceID,
		ParentTaskID:   record.ParentTaskID,
		Status:         record.Status,
		NetworkChannel: record.NetworkChannel,
		Owner:          cloneOwnership(record.Owner),
	}); err != nil {
		return nil, err
	}

	return &record, nil
}

// CreateChildTask creates one child task beneath the supplied parent and emits
// an additional parent-scoped audit event.
func (m *TaskManager) CreateChildTask(ctx context.Context, parentTaskID string, spec CreateTask, actor ActorContext) (*Task, error) {
	trimmedParentID := strings.TrimSpace(parentTaskID)
	if trimmedParentID == "" {
		return nil, fmt.Errorf("%w: child parent task id is required", ErrValidation)
	}
	if strings.TrimSpace(spec.ParentTaskID) != "" && strings.TrimSpace(spec.ParentTaskID) != trimmedParentID {
		return nil, fmt.Errorf("%w: create_task.parent_task_id must match child parent task id", ErrValidation)
	}

	spec.ParentTaskID = trimmedParentID
	child, err := m.CreateTask(ctx, spec, actor)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, trimmedParentID, "", taskEventChildCreated, actor, childCreatedTaskPayload{
		ChildTaskID:      child.ID,
		ChildScope:       child.Scope,
		ChildWorkspaceID: child.WorkspaceID,
	}); err != nil {
		return nil, err
	}
	return child, nil
}

// UpdateTask applies one mutable patch while preserving immutable identity and
// structural fields under manager control.
func (m *TaskManager) UpdateTask(ctx context.Context, id string, patch TaskPatch, actor ActorContext) (*Task, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: task id is required", ErrValidation)
	}
	normalizedPatch, err := normalizeTaskPatch(patch)
	if err != nil {
		return nil, err
	}
	if normalizedPatch.NetworkChannel != nil {
		if err := m.validateNetworkChannel("task_patch.network_channel", *normalizedPatch.NetworkChannel); err != nil {
			return nil, err
		}
	}

	current, err := m.store.GetTask(ctx, trimmedID)
	if err != nil {
		return nil, err
	}

	updated := current
	changedFields := make([]string, 0, len(MutableTaskFields()))

	if normalizedPatch.Title != nil && updated.Title != *normalizedPatch.Title {
		updated.Title = *normalizedPatch.Title
		changedFields = append(changedFields, TaskFieldTitle)
	}
	if normalizedPatch.Description != nil && updated.Description != *normalizedPatch.Description {
		updated.Description = *normalizedPatch.Description
		changedFields = append(changedFields, TaskFieldDescription)
	}
	if normalizedPatch.Metadata != nil && !sameRawJSON(updated.Metadata, *normalizedPatch.Metadata) {
		updated.Metadata = cloneRawJSON(*normalizedPatch.Metadata)
		changedFields = append(changedFields, TaskFieldMetadata)
	}
	if normalizedPatch.NetworkChannel != nil && updated.NetworkChannel != *normalizedPatch.NetworkChannel {
		updated.NetworkChannel = *normalizedPatch.NetworkChannel
		changedFields = append(changedFields, TaskFieldNetworkChannel)
	}
	if normalizedPatch.Owner != nil && !sameOwnership(updated.Owner, normalizedPatch.Owner) {
		updated.Owner = cloneOwnership(normalizedPatch.Owner)
		changedFields = append(changedFields, TaskFieldOwner)
	}
	if normalizedPatch.ClearOwner && updated.Owner != nil {
		updated.Owner = nil
		changedFields = append(changedFields, TaskFieldOwner)
	}
	if len(changedFields) == 0 {
		return &current, nil
	}

	dependencies, err := m.store.ListDependencies(ctx, trimmedID)
	if err != nil {
		return nil, err
	}
	runs, err := m.store.ListTaskRuns(ctx, TaskRunQuery{TaskID: trimmedID})
	if err != nil {
		return nil, err
	}

	canonicalStatus, err := m.canonicalTaskStatus(ctx, current, dependencies, runs)
	if err != nil {
		return nil, err
	}
	updated.Status = canonicalStatus
	updated.UpdatedAt = m.now().UTC()
	if err := m.store.UpdateTask(ctx, updated); err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, updated.ID, "", taskEventUpdated, actor, updatedTaskPayload{
		ChangedFields: append([]string(nil), changedFields...),
		Status:        updated.Status,
	}); err != nil {
		return nil, err
	}

	return &updated, nil
}

// CancelTask propagates manager-owned cancellation through the target task,
// affected runs, and all non-terminal descendants.
func (m *TaskManager) CancelTask(ctx context.Context, id string, req CancelTask, actor ActorContext) (*Task, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: task id is required", ErrValidation)
	}
	normalizedReq, err := normalizeCancelTask(req)
	if err != nil {
		return nil, err
	}

	tree, err := m.collectTaskTree(ctx, trimmedID)
	if err != nil {
		return nil, err
	}
	if len(tree) == 0 {
		return nil, ErrTaskNotFound
	}

	root := tree[0]
	rootRuns, err := m.store.ListTaskRuns(ctx, TaskRunQuery{TaskID: root.ID})
	if err != nil {
		return nil, err
	}
	rootDeps, err := m.store.ListDependencies(ctx, root.ID)
	if err != nil {
		return nil, err
	}
	rootStatus, err := m.canonicalTaskStatus(ctx, root, rootDeps, rootRuns)
	if err != nil {
		return nil, err
	}
	if isTerminalTaskStatus(rootStatus) && rootStatus != TaskStatusCancelled && !hasOpenRun(rootRuns) {
		return nil, fmt.Errorf("%w: task %q cannot transition from %q to %q", ErrInvalidStatusTransition, root.ID, rootStatus, TaskStatusCancelled)
	}

	cancelledRoot := root
	for idx, record := range tree {
		runs, err := m.store.ListTaskRuns(ctx, TaskRunQuery{TaskID: record.ID})
		if err != nil {
			return nil, err
		}
		dependencies, err := m.store.ListDependencies(ctx, record.ID)
		if err != nil {
			return nil, err
		}
		status, err := m.canonicalTaskStatus(ctx, record, dependencies, runs)
		if err != nil {
			return nil, err
		}
		record.Status = status

		if idx > 0 && isTerminalTaskStatus(status) {
			if record.ID == trimmedID {
				cancelledRoot = record
			}
			continue
		}

		propagatedFromTaskID := ""
		if idx > 0 {
			propagatedFromTaskID = trimmedID
		}

		cancelledRunIDs := make([]string, 0)
		for _, run := range runs {
			if isTerminalRunStatus(run.Status) {
				continue
			}
			cancelledRun, err := m.cancelRunRecord(ctx, record, run, CancelRun(normalizedReq), actor, cancelRunOptions{
				propagatedFromTaskID: propagatedFromTaskID,
				reconcileTask:        false,
			})
			if err != nil {
				return nil, err
			}
			cancelledRunIDs = append(cancelledRunIDs, cancelledRun.ID)
		}

		if status.Normalize() == TaskStatusCancelled && len(cancelledRunIDs) == 0 {
			if record.ID == trimmedID {
				cancelledRoot = record
			}
			continue
		}

		record.Status = TaskStatusCancelled
		record.UpdatedAt = m.now().UTC()
		record.ClosedAt = record.UpdatedAt
		if err := m.store.UpdateTask(ctx, record); err != nil {
			return nil, err
		}
		if err := m.recordTaskEvent(ctx, record.ID, "", taskEventCancelled, actor, cancelledTaskPayload{
			Reason:               normalizedReq.Reason,
			Metadata:             cloneRawJSON(normalizedReq.Metadata),
			Status:               record.Status,
			PropagatedFromTaskID: propagatedFromTaskID,
			CancelledRunIDs:      append([]string(nil), cancelledRunIDs...),
		}); err != nil {
			return nil, err
		}
		if err := m.reconcileDependentTasks(ctx, record.ID, map[string]struct{}{record.ID: {}}); err != nil {
			return nil, err
		}

		if record.ID == trimmedID {
			cancelledRoot = record
		}
	}

	return &cancelledRoot, nil
}

// GetTask returns one expanded task view after enforcing read authority.
func (m *TaskManager) GetTask(ctx context.Context, id string, actor ActorContext) (*TaskView, error) {
	if err := requireReadAuthority(actor); err != nil {
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

	children, err := m.store.ListTasks(ctx, TaskQuery{ParentTaskID: trimmedID})
	if err != nil {
		return nil, err
	}
	dependencies, err := m.store.ListDependencies(ctx, trimmedID)
	if err != nil {
		return nil, err
	}
	runs, err := m.store.ListTaskRuns(ctx, TaskRunQuery{TaskID: trimmedID})
	if err != nil {
		return nil, err
	}
	events, err := m.store.ListTaskEvents(ctx, TaskEventQuery{TaskID: trimmedID})
	if err != nil {
		return nil, err
	}

	view := &TaskView{
		Task:         record,
		Children:     children,
		Dependencies: dependencies,
		Runs:         runs,
		Events:       events,
	}
	view.Task.Status, err = m.canonicalTaskStatus(ctx, record, dependencies, runs)
	if err != nil {
		return nil, err
	}
	return view, nil
}

// ListTasks returns task summaries that satisfy the supplied query filters
// after enforcing read authority.
func (m *TaskManager) ListTasks(ctx context.Context, query TaskQuery, actor ActorContext) ([]TaskSummary, error) {
	if err := requireReadAuthority(actor); err != nil {
		return nil, err
	}
	return m.store.ListTasks(ctx, query)
}

// AddDependency adds one dependency edge through the manager, reconciles the
// task status, and records the canonical audit event.
func (m *TaskManager) AddDependency(ctx context.Context, spec AddDependency, actor ActorContext) error {
	if err := requireWriteAuthority(actor); err != nil {
		return err
	}

	normalizedSpec, err := normalizeAddDependencySpec(spec)
	if err != nil {
		return err
	}
	if err := m.store.CreateDependency(ctx, TaskDependency{
		TaskID:          normalizedSpec.TaskID,
		DependsOnTaskID: normalizedSpec.DependsOnTaskID,
		Kind:            normalizedSpec.Kind,
	}); err != nil {
		return err
	}

	record, err := m.reconcileTaskCascade(ctx, normalizedSpec.TaskID)
	if err != nil {
		return err
	}
	return m.recordTaskEvent(ctx, normalizedSpec.TaskID, "", taskEventDependencyAdded, actor, dependencyTaskPayload{
		DependsOnTaskID: normalizedSpec.DependsOnTaskID,
		Kind:            normalizedSpec.Kind,
		Status:          record.Status,
	})
}

// RemoveDependency deletes one dependency edge through the manager, reconciles
// the task status, and records the canonical audit event.
func (m *TaskManager) RemoveDependency(ctx context.Context, taskID string, dependsOnID string, actor ActorContext) error {
	if err := requireWriteAuthority(actor); err != nil {
		return err
	}

	trimmedTaskID := strings.TrimSpace(taskID)
	if trimmedTaskID == "" {
		return fmt.Errorf("%w: task id is required", ErrValidation)
	}
	trimmedDependsOnID := strings.TrimSpace(dependsOnID)
	if trimmedDependsOnID == "" {
		return fmt.Errorf("%w: depends_on_task_id is required", ErrValidation)
	}

	if err := m.store.DeleteDependency(ctx, trimmedTaskID, trimmedDependsOnID); err != nil {
		return err
	}

	record, err := m.reconcileTaskCascade(ctx, trimmedTaskID)
	if err != nil {
		return err
	}
	return m.recordTaskEvent(ctx, trimmedTaskID, "", taskEventDependencyRemoved, actor, dependencyTaskPayload{
		DependsOnTaskID: trimmedDependsOnID,
		Kind:            DependencyKindBlocks,
		Status:          record.Status,
	})
}

// EnqueueRun persists one new queue-first task run under manager authority.
func (m *TaskManager) EnqueueRun(ctx context.Context, spec EnqueueRun, actor ActorContext) (*TaskRun, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}

	normalizedSpec, err := normalizeEnqueueRunSpec(spec)
	if err != nil {
		return nil, err
	}
	if err := requireLifecycleIdempotency(actor, normalizedSpec.IdempotencyKey, "enqueue_run"); err != nil {
		return nil, err
	}
	if err := m.validateNetworkChannel("enqueue_run.network_channel", normalizedSpec.NetworkChannel); err != nil {
		return nil, err
	}

	taskRecord, err := m.store.GetTask(ctx, normalizedSpec.TaskID)
	if err != nil {
		return nil, err
	}
	if taskRecord.Status.Normalize() == TaskStatusCancelled {
		return nil, fmt.Errorf("%w: task %q is cancelled", ErrInvalidStatusTransition, taskRecord.ID)
	}

	if existing, err := m.lookupIdempotentRun(ctx, normalizedSpec.IdempotencyKey, actor.Origin, normalizedSpec.TaskID); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	existingRuns, err := m.store.ListTaskRuns(ctx, TaskRunQuery{TaskID: normalizedSpec.TaskID})
	if err != nil {
		return nil, err
	}
	run := TaskRun{
		ID:             m.newID("run"),
		TaskID:         normalizedSpec.TaskID,
		Status:         TaskRunStatusQueued,
		Attempt:        nextRunAttempt(existingRuns),
		Origin:         actor.Origin,
		IdempotencyKey: normalizedSpec.IdempotencyKey,
		NetworkChannel: resolvedRunChannel(normalizedSpec.NetworkChannel, taskRecord.NetworkChannel),
		QueuedAt:       m.now().UTC(),
	}
	if err := m.store.CreateTaskRun(ctx, run); err != nil {
		return nil, err
	}
	if err := m.saveRunIdempotency(ctx, run, actor.Origin); err != nil {
		return nil, err
	}

	reconciledTask, err := m.reconcileTaskCascade(ctx, normalizedSpec.TaskID)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunEnqueued, actor, runEnqueuedPayload{
		Attempt:        run.Attempt,
		Status:         run.Status,
		TaskStatus:     reconciledTask.Status,
		NetworkChannel: run.NetworkChannel,
		IdempotencyKey: run.IdempotencyKey,
	}); err != nil {
		return nil, err
	}

	return &run, nil
}

// ClaimRun transitions one queued run into the claimed state.
func (m *TaskManager) ClaimRun(ctx context.Context, runID string, claim ClaimRun, actor ActorContext) (*TaskRun, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}

	normalizedClaim, err := normalizeClaimRun(claim)
	if err != nil {
		return nil, err
	}
	if err := requireLifecycleIdempotency(actor, normalizedClaim.IdempotencyKey, "claim_run"); err != nil {
		return nil, err
	}

	run, taskRecord, err := m.loadRunWithTask(ctx, runID)
	if err != nil {
		return nil, err
	}
	if err := m.ensureTaskExecutable(ctx, taskRecord); err != nil {
		return nil, err
	}
	if err := requireRunTransition(run, TaskRunStatusClaimed); err != nil {
		return nil, err
	}

	run.Status = TaskRunStatusClaimed
	run.ClaimedBy = &ActorIdentity{Kind: actor.Actor.Kind, Ref: actor.Actor.Ref}
	run.ClaimedAt = m.now().UTC()
	if err := m.store.UpdateTaskRun(ctx, run); err != nil {
		return nil, err
	}

	reconciledTask, err := m.reconcileTaskCascade(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunClaimed, actor, runClaimedPayload{
		Status:     run.Status,
		TaskStatus: reconciledTask.Status,
		ClaimedBy:  *run.ClaimedBy,
	}); err != nil {
		return nil, err
	}

	return &run, nil
}

// StartRun transitions one claimed or starting run into active execution.
func (m *TaskManager) StartRun(ctx context.Context, runID string, req StartRun, actor ActorContext) (*TaskRun, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}

	normalizedReq, err := normalizeStartRun(req)
	if err != nil {
		return nil, err
	}
	if err := requireLifecycleIdempotency(actor, normalizedReq.IdempotencyKey, "start_run"); err != nil {
		return nil, err
	}

	run, taskRecord, err := m.loadRunWithTask(ctx, runID)
	if err != nil {
		return nil, err
	}
	if err := m.ensureTaskExecutable(ctx, taskRecord); err != nil {
		return nil, err
	}
	if err := m.validateRunChannelUsable(ctx, taskRecord, run, actor, "start"); err != nil {
		return nil, err
	}

	switch run.Status.Normalize() {
	case TaskRunStatusClaimed:
		if err := m.requireSessionExecutor("start run"); err != nil {
			return nil, err
		}

		run.Status = TaskRunStatusStarting
		if err := m.store.UpdateTaskRun(ctx, run); err != nil {
			return nil, err
		}

		startingTask, err := m.reconcileTaskCascade(ctx, run.TaskID)
		if err != nil {
			return nil, err
		}
		if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunStarting, actor, runTransitionPayload{
			Status:     run.Status,
			TaskStatus: startingTask.Status,
			SessionID:  run.SessionID,
		}); err != nil {
			return nil, err
		}

		sessionRef, err := m.sessions.StartTaskSession(ctx, StartTaskSession{
			Task:  startingTask,
			Run:   run,
			Actor: actor,
		})
		if err != nil {
			failedRun, failErr := m.failRunRecord(ctx, taskRecord, run, RunFailure{
				Error: fmt.Sprintf("start task session: %v", err),
			}, actor)
			if failErr != nil {
				return nil, errorsJoin(err, failErr)
			}
			return failedRun, fmt.Errorf("task: start task session for run %q: %w", run.ID, err)
		}
		if sessionRef == nil {
			failedRun, failErr := m.failRunRecord(ctx, taskRecord, run, RunFailure{
				Error: "start task session: nil session reference",
			}, actor)
			if failErr != nil {
				return nil, failErr
			}
			return failedRun, fmt.Errorf("%w: start_task_session returned nil session reference", ErrValidation)
		}
		if err := sessionRef.Validate(); err != nil {
			failedRun, failErr := m.failRunRecord(ctx, taskRecord, run, RunFailure{
				Error: fmt.Sprintf("start task session: %v", err),
			}, actor)
			if failErr != nil {
				return nil, errorsJoin(err, failErr)
			}
			return failedRun, err
		}
		run.SessionID = strings.TrimSpace(sessionRef.SessionID)
	case TaskRunStatusStarting:
		if strings.TrimSpace(run.SessionID) == "" {
			return nil, fmt.Errorf("%w: task run %q cannot transition from %q to %q without a session binding", ErrInvalidStatusTransition, run.ID, run.Status, TaskRunStatusRunning)
		}
	default:
		return nil, requireRunTransition(run, TaskRunStatusRunning)
	}

	run.Status = TaskRunStatusRunning
	run.StartedAt = m.now().UTC()
	if err := m.store.UpdateTaskRun(ctx, run); err != nil {
		return nil, err
	}

	reconciledTask, err := m.reconcileTaskCascade(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunStarted, actor, runTransitionPayload{
		Status:     run.Status,
		TaskStatus: reconciledTask.Status,
		SessionID:  run.SessionID,
	}); err != nil {
		return nil, err
	}

	return &run, nil
}

// AttachRunSession binds one existing session to a claimed or starting run.
func (m *TaskManager) AttachRunSession(ctx context.Context, runID string, sessionID string, actor ActorContext) (*TaskRun, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}
	if err := m.requireSessionExecutor("attach run session"); err != nil {
		return nil, err
	}

	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return nil, fmt.Errorf("%w: session id is required", ErrValidation)
	}

	run, taskRecord, err := m.loadRunWithTask(ctx, runID)
	if err != nil {
		return nil, err
	}
	if err := m.ensureTaskExecutable(ctx, taskRecord); err != nil {
		return nil, err
	}
	if err := m.validateRunChannelUsable(ctx, taskRecord, run, actor, "attach"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(run.SessionID) != "" {
		return nil, ErrSessionAlreadyBound
	}

	switch run.Status.Normalize() {
	case TaskRunStatusClaimed, TaskRunStatusStarting:
	default:
		return nil, ErrSessionAttachNotAllowed
	}

	activeBindings, err := m.store.CountActiveSessionBindings(ctx, trimmedSessionID)
	if err != nil {
		return nil, err
	}
	if activeBindings > 0 {
		return nil, ErrSessionAlreadyBound
	}

	sessionRef, err := m.sessions.AttachTaskSession(ctx, run.ID, trimmedSessionID)
	if err != nil {
		return nil, err
	}
	if sessionRef == nil {
		return nil, fmt.Errorf("%w: attach_task_session returned nil session reference", ErrValidation)
	}
	if err := sessionRef.Validate(); err != nil {
		return nil, err
	}

	run.SessionID = strings.TrimSpace(sessionRef.SessionID)
	if run.Status.Normalize() == TaskRunStatusClaimed {
		run.Status = TaskRunStatusStarting
	}
	if err := m.store.UpdateTaskRun(ctx, run); err != nil {
		return nil, err
	}

	reconciledTask, err := m.reconcileTaskCascade(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunSessionBound, actor, runTransitionPayload{
		Status:     run.Status,
		TaskStatus: reconciledTask.Status,
		SessionID:  run.SessionID,
	}); err != nil {
		return nil, err
	}

	return &run, nil
}

// CompleteRun marks one running task run as completed and reconciles task state.
func (m *TaskManager) CompleteRun(ctx context.Context, runID string, result RunResult, actor ActorContext) (*TaskRun, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}

	normalizedResult, err := normalizeRunResult(result)
	if err != nil {
		return nil, err
	}

	run, taskRecord, err := m.loadRunWithTask(ctx, runID)
	if err != nil {
		return nil, err
	}
	if err := requireRunTransition(run, TaskRunStatusCompleted); err != nil {
		return nil, err
	}

	run.Status = TaskRunStatusCompleted
	run.Result = cloneRawJSON(normalizedResult.Value)
	run.Error = ""
	run.EndedAt = m.now().UTC()
	if err := m.store.UpdateTaskRun(ctx, run); err != nil {
		return nil, err
	}

	reconciledTask, err := m.reconcileTaskCascade(ctx, taskRecord.ID)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunCompleted, actor, completedRunPayload{
		Status:     run.Status,
		TaskStatus: reconciledTask.Status,
		Result:     cloneRawJSON(run.Result),
	}); err != nil {
		return nil, err
	}

	return &run, nil
}

// FailRun marks one starting or running task run as failed and reconciles task state.
func (m *TaskManager) FailRun(ctx context.Context, runID string, failure RunFailure, actor ActorContext) (*TaskRun, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}

	normalizedFailure, err := normalizeRunFailure(failure)
	if err != nil {
		return nil, err
	}

	run, taskRecord, err := m.loadRunWithTask(ctx, runID)
	if err != nil {
		return nil, err
	}
	return m.failRunRecord(ctx, taskRecord, run, normalizedFailure, actor)
}

// CancelRun cancels one non-terminal task run under manager authority.
func (m *TaskManager) CancelRun(ctx context.Context, runID string, req CancelRun, actor ActorContext) (*TaskRun, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}

	normalizedReq, err := normalizeCancelRun(req)
	if err != nil {
		return nil, err
	}

	run, taskRecord, err := m.loadRunWithTask(ctx, runID)
	if err != nil {
		return nil, err
	}
	return m.cancelRunRecord(ctx, taskRecord, run, normalizedReq, actor, cancelRunOptions{
		reconcileTask: true,
	})
}

// RecoverRunOnBoot applies one daemon-owned recovery decision to a non-terminal
// run discovered during startup reconciliation.
func (m *TaskManager) RecoverRunOnBoot(ctx context.Context, runID string, recovery RunBootRecovery, actor ActorContext) (*TaskRun, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}

	normalizedRecovery, err := normalizeRunBootRecovery(recovery)
	if err != nil {
		return nil, err
	}

	run, taskRecord, err := m.loadRunWithTask(ctx, runID)
	if err != nil {
		return nil, err
	}

	previousStatus := run.Status.Normalize()
	previousSessionID := strings.TrimSpace(run.SessionID)
	switch normalizedRecovery.Action.Normalize() {
	case RunBootRecoveryRequeue:
		if previousStatus != TaskRunStatusClaimed {
			return nil, fmt.Errorf("%w: task run %q cannot recover from %q via %q", ErrInvalidStatusTransition, run.ID, previousStatus, normalizedRecovery.Action)
		}

		run.Status = TaskRunStatusQueued
		run.ClaimedBy = nil
		run.ClaimedAt = time.Time{}
		run.SessionID = ""
		run.StartedAt = time.Time{}
		run.EndedAt = time.Time{}
		run.Error = ""
		run.Result = nil
		if err := m.store.UpdateTaskRun(ctx, run); err != nil {
			return nil, err
		}

		reconciledTask, err := m.reconcileTaskCascade(ctx, taskRecord.ID)
		if err != nil {
			return nil, err
		}
		if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunRecovered, actor, recoveredRunPayload{
			Action:         normalizedRecovery.Action,
			PreviousStatus: previousStatus,
			Status:         run.Status,
			TaskStatus:     reconciledTask.Status,
			Reason:         normalizedRecovery.Reason,
			SessionID:      previousSessionID,
			SessionState:   normalizedRecovery.SessionState,
		}); err != nil {
			return nil, err
		}
		return &run, nil

	case RunBootRecoveryMarkRunning:
		switch previousStatus {
		case TaskRunStatusClaimed, TaskRunStatusStarting:
		case TaskRunStatusRunning:
			return &run, nil
		default:
			return nil, fmt.Errorf("%w: task run %q cannot recover from %q via %q", ErrInvalidStatusTransition, run.ID, previousStatus, normalizedRecovery.Action)
		}
		if previousSessionID == "" {
			return nil, fmt.Errorf("%w: task run %q cannot recover to running without a session binding", ErrInvalidStatusTransition, run.ID)
		}

		run.Status = TaskRunStatusRunning
		if run.StartedAt.IsZero() {
			run.StartedAt = m.now().UTC()
		}
		if err := m.store.UpdateTaskRun(ctx, run); err != nil {
			return nil, err
		}

		reconciledTask, err := m.reconcileTaskCascade(ctx, taskRecord.ID)
		if err != nil {
			return nil, err
		}
		if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunRecovered, actor, recoveredRunPayload{
			Action:         normalizedRecovery.Action,
			PreviousStatus: previousStatus,
			Status:         run.Status,
			TaskStatus:     reconciledTask.Status,
			Reason:         normalizedRecovery.Reason,
			SessionID:      previousSessionID,
			SessionState:   normalizedRecovery.SessionState,
		}); err != nil {
			return nil, err
		}
		return &run, nil

	case RunBootRecoveryFail:
		failedRun, err := m.failRunRecord(ctx, taskRecord, run, RunFailure{
			Error:    runBootRecoveryError(run, normalizedRecovery),
			Metadata: runBootRecoveryMetadata(run, normalizedRecovery),
		}, actor)
		if err != nil {
			return nil, err
		}

		reconciledTask, err := m.store.GetTask(ctx, taskRecord.ID)
		if err != nil {
			return nil, err
		}
		if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunRecovered, actor, recoveredRunPayload{
			Action:         normalizedRecovery.Action,
			PreviousStatus: previousStatus,
			Status:         failedRun.Status,
			TaskStatus:     reconciledTask.Status,
			Reason:         normalizedRecovery.Reason,
			SessionID:      previousSessionID,
			SessionState:   normalizedRecovery.SessionState,
		}); err != nil {
			return nil, err
		}
		return failedRun, nil

	default:
		return nil, fmt.Errorf("%w: run boot recovery action %q is not supported", ErrValidation, normalizedRecovery.Action)
	}
}

func requireReadAuthority(actor ActorContext) error {
	if err := actor.Validate(); err != nil {
		return err
	}
	if !actor.Authority.Read {
		return ErrPermissionDenied
	}
	return nil
}

func requireWriteAuthority(actor ActorContext) error {
	if err := actor.Validate(); err != nil {
		return err
	}
	if !actor.Authority.Write {
		return ErrPermissionDenied
	}
	return nil
}

func requireCreateAuthority(actor ActorContext, scope Scope) error {
	if err := requireWriteAuthority(actor); err != nil {
		return err
	}

	switch scope.Normalize() {
	case ScopeGlobal:
		if !actor.Authority.CreateGlobal {
			return ErrPermissionDenied
		}
	case ScopeWorkspace:
		if !actor.Authority.CreateWorkspace {
			return ErrPermissionDenied
		}
	default:
		return fmt.Errorf("%w: create_task.scope is required", ErrValidation)
	}
	return nil
}

func normalizeCreateTaskSpec(spec CreateTask) (CreateTask, error) {
	normalized := spec
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.Identifier = strings.TrimSpace(normalized.Identifier)
	normalized.Scope = normalized.Scope.Normalize()
	normalized.WorkspaceID = strings.TrimSpace(normalized.WorkspaceID)
	normalized.ParentTaskID = strings.TrimSpace(normalized.ParentTaskID)
	normalized.NetworkChannel = strings.TrimSpace(normalized.NetworkChannel)
	normalized.Title = strings.TrimSpace(normalized.Title)
	normalized.Description = strings.TrimSpace(normalized.Description)
	if normalized.Owner != nil {
		normalized.Owner = normalizeOwnership(normalized.Owner)
	}
	normalized.Metadata = normalizeRawJSON(normalized.Metadata)
	if err := normalized.Validate("create_task"); err != nil {
		return CreateTask{}, err
	}
	return normalized, nil
}

func normalizeTaskPatch(patch TaskPatch) (TaskPatch, error) {
	normalized := patch
	if normalized.Title != nil {
		title := strings.TrimSpace(*normalized.Title)
		normalized.Title = &title
	}
	if normalized.Description != nil {
		description := strings.TrimSpace(*normalized.Description)
		normalized.Description = &description
	}
	if normalized.Metadata != nil {
		metadata := normalizeRawJSON(*normalized.Metadata)
		normalized.Metadata = &metadata
	}
	if normalized.NetworkChannel != nil {
		channel := strings.TrimSpace(*normalized.NetworkChannel)
		normalized.NetworkChannel = &channel
	}
	if normalized.Owner != nil {
		normalized.Owner = normalizeOwnership(normalized.Owner)
	}
	if err := normalized.Validate("task_patch"); err != nil {
		return TaskPatch{}, err
	}
	return normalized, nil
}

func normalizeAddDependencySpec(spec AddDependency) (AddDependency, error) {
	normalized := spec
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.DependsOnTaskID = strings.TrimSpace(normalized.DependsOnTaskID)
	normalized.Kind = normalized.Kind.Normalize()
	if err := normalized.Validate("add_dependency"); err != nil {
		return AddDependency{}, err
	}
	return normalized, nil
}

func normalizeCancelTask(req CancelTask) (CancelTask, error) {
	normalized := req
	normalized.Reason = strings.TrimSpace(normalized.Reason)
	normalized.Metadata = normalizeRawJSON(normalized.Metadata)
	if err := normalized.Validate("cancel_task"); err != nil {
		return CancelTask{}, err
	}
	return normalized, nil
}

func normalizeEnqueueRunSpec(spec EnqueueRun) (EnqueueRun, error) {
	normalized := spec
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.IdempotencyKey = strings.TrimSpace(normalized.IdempotencyKey)
	normalized.NetworkChannel = strings.TrimSpace(normalized.NetworkChannel)
	if err := normalized.Validate("enqueue_run"); err != nil {
		return EnqueueRun{}, err
	}
	return normalized, nil
}

func normalizeClaimRun(claim ClaimRun) (ClaimRun, error) {
	normalized := claim
	normalized.IdempotencyKey = strings.TrimSpace(normalized.IdempotencyKey)
	if err := normalized.Validate("claim_run"); err != nil {
		return ClaimRun{}, err
	}
	return normalized, nil
}

func normalizeStartRun(req StartRun) (StartRun, error) {
	normalized := req
	normalized.IdempotencyKey = strings.TrimSpace(normalized.IdempotencyKey)
	if err := normalized.Validate("start_run"); err != nil {
		return StartRun{}, err
	}
	return normalized, nil
}

func normalizeCancelRun(req CancelRun) (CancelRun, error) {
	normalized := req
	normalized.Reason = strings.TrimSpace(normalized.Reason)
	normalized.Metadata = normalizeRawJSON(normalized.Metadata)
	if err := normalized.Validate("cancel_run"); err != nil {
		return CancelRun{}, err
	}
	return normalized, nil
}

func normalizeRunResult(result RunResult) (RunResult, error) {
	normalized := result
	normalized.Value = normalizeRawJSON(normalized.Value)
	if err := normalized.Validate("run_result"); err != nil {
		return RunResult{}, err
	}
	return normalized, nil
}

func normalizeRunFailure(failure RunFailure) (RunFailure, error) {
	normalized := failure
	normalized.Error = strings.TrimSpace(normalized.Error)
	normalized.Metadata = normalizeRawJSON(normalized.Metadata)
	if err := normalized.Validate("run_failure"); err != nil {
		return RunFailure{}, err
	}
	return normalized, nil
}

func normalizeRunBootRecovery(recovery RunBootRecovery) (RunBootRecovery, error) {
	normalized := recovery
	normalized.Action = normalized.Action.Normalize()
	normalized.Reason = strings.TrimSpace(normalized.Reason)
	normalized.SessionState = strings.TrimSpace(normalized.SessionState)
	if err := normalized.Validate("run_boot_recovery"); err != nil {
		return RunBootRecovery{}, err
	}
	return normalized, nil
}

func requireLifecycleIdempotency(actor ActorContext, key string, path string) error {
	if actor.Actor.Kind.Normalize() == ActorKindHuman {
		return nil
	}
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("%w: %s.idempotency_key is required for non-human actors", ErrValidation, path)
	}
	return nil
}

func (m *TaskManager) validateParentConstraints(ctx context.Context, spec CreateTask) error {
	if strings.TrimSpace(spec.ParentTaskID) == "" {
		return nil
	}

	parent, err := m.store.GetTask(ctx, spec.ParentTaskID)
	if err != nil {
		return err
	}
	if err := validateParentChildScope(parent, spec.Scope, spec.WorkspaceID); err != nil {
		return err
	}

	childCount, err := m.store.CountDirectChildren(ctx, parent.ID)
	if err != nil {
		return err
	}
	if err := ValidateDirectChildCount(childCount + 1); err != nil {
		return err
	}

	parentDepth, err := m.taskDepth(ctx, parent)
	if err != nil {
		return err
	}
	return ValidateHierarchyDepth(parentDepth + 1)
}

func validateParentChildScope(parent Task, childScope Scope, childWorkspaceID string) error {
	switch parent.Scope.Normalize() {
	case ScopeGlobal:
		return nil
	case ScopeWorkspace:
		if childScope.Normalize() != ScopeWorkspace {
			return fmt.Errorf("%w: workspace-scoped parent tasks require workspace-scoped children", ErrValidation)
		}
		if strings.TrimSpace(parent.WorkspaceID) != strings.TrimSpace(childWorkspaceID) {
			return fmt.Errorf("%w: child workspace_id must match workspace-scoped parent", ErrValidation)
		}
		return nil
	default:
		return fmt.Errorf("%w: parent task has unsupported scope %q", ErrValidation, parent.Scope)
	}
}

func (m *TaskManager) taskDepth(ctx context.Context, record Task) (int, error) {
	depth := 1
	current := record
	seen := map[string]struct{}{strings.TrimSpace(record.ID): {}}

	for strings.TrimSpace(current.ParentTaskID) != "" {
		parentID := strings.TrimSpace(current.ParentTaskID)
		if _, ok := seen[parentID]; ok {
			return 0, fmt.Errorf("%w: task hierarchy contains a cycle at %q", ErrValidation, parentID)
		}
		seen[parentID] = struct{}{}

		parent, err := m.store.GetTask(ctx, parentID)
		if err != nil {
			return 0, err
		}
		depth++
		current = parent
	}

	return depth, nil
}

func (m *TaskManager) reconcileTask(ctx context.Context, taskID string) (Task, error) {
	record, err := m.store.GetTask(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	dependencies, err := m.store.ListDependencies(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	runs, err := m.store.ListTaskRuns(ctx, TaskRunQuery{TaskID: taskID})
	if err != nil {
		return Task{}, err
	}

	canonicalStatus, err := m.canonicalTaskStatus(ctx, record, dependencies, runs)
	if err != nil {
		return Task{}, err
	}
	if record.Status.Normalize() == canonicalStatus.Normalize() {
		return record, nil
	}

	record.Status = canonicalStatus
	record.UpdatedAt = m.now().UTC()
	if isTerminalTaskStatus(record.Status) {
		record.ClosedAt = record.UpdatedAt
	} else {
		record.ClosedAt = time.Time{}
	}
	if err := m.store.UpdateTask(ctx, record); err != nil {
		return Task{}, err
	}
	return record, nil
}

func (m *TaskManager) reconcileTaskCascade(ctx context.Context, taskID string) (Task, error) {
	previous, err := m.store.GetTask(ctx, taskID)
	if err != nil {
		return Task{}, err
	}

	reconciled, err := m.reconcileTask(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	if previous.Status.Normalize() != reconciled.Status.Normalize() {
		if err := m.reconcileDependentTasks(ctx, taskID, map[string]struct{}{taskID: {}}); err != nil {
			return Task{}, err
		}
	}
	return reconciled, nil
}

func (m *TaskManager) canonicalTaskStatus(ctx context.Context, record Task, dependencies []TaskDependency, runs []TaskRun) (TaskStatus, error) {
	unresolvedDependencies, err := m.hasUnresolvedDependencies(ctx, dependencies)
	if err != nil {
		return "", err
	}
	return taskStatusFromSnapshot(record.Status, unresolvedDependencies, runs), nil
}

func hasActiveRun(runs []TaskRun) bool {
	for _, run := range runs {
		switch run.Status.Normalize() {
		case TaskRunStatusStarting, TaskRunStatusRunning:
			return true
		}
	}
	return false
}

func hasOpenRun(runs []TaskRun) bool {
	for _, run := range runs {
		if !isTerminalRunStatus(run.Status) {
			return true
		}
	}
	return false
}

func isTerminalTaskStatus(status TaskStatus) bool {
	switch status.Normalize() {
	case TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled:
		return true
	default:
		return false
	}
}

func isTerminalRunStatus(status TaskRunStatus) bool {
	switch status.Normalize() {
	case TaskRunStatusCompleted, TaskRunStatusFailed, TaskRunStatusCancelled:
		return true
	default:
		return false
	}
}

func taskStatusFromSnapshot(currentStatus TaskStatus, unresolvedDependencies bool, runs []TaskRun) TaskStatus {
	status := currentStatus.Normalize()
	if status == TaskStatusCancelled {
		return status
	}
	if hasActiveRun(runs) {
		return TaskStatusInProgress
	}
	if hasQueuedOrClaimedRun(runs) {
		if unresolvedDependencies {
			return TaskStatusBlocked
		}
		return TaskStatusReady
	}
	if latest := latestTerminalRun(runs); latest != nil {
		switch latest.Status.Normalize() {
		case TaskRunStatusCompleted:
			return TaskStatusCompleted
		case TaskRunStatusFailed:
			return TaskStatusFailed
		case TaskRunStatusCancelled:
			return TaskStatusCancelled
		}
	}

	if isTerminalTaskStatus(status) {
		return status
	}
	if unresolvedDependencies {
		return TaskStatusBlocked
	}
	return TaskStatusReady
}

func hasQueuedOrClaimedRun(runs []TaskRun) bool {
	for _, run := range runs {
		switch run.Status.Normalize() {
		case TaskRunStatusQueued, TaskRunStatusClaimed:
			return true
		}
	}
	return false
}

func latestTerminalRun(runs []TaskRun) *TaskRun {
	var latest *TaskRun
	for idx := range runs {
		run := runs[idx]
		if !isTerminalRunStatus(run.Status) {
			continue
		}
		if latest == nil || runComesAfter(run, *latest) {
			candidate := run
			latest = &candidate
		}
	}
	return latest
}

func runComesAfter(left TaskRun, right TaskRun) bool {
	switch {
	case left.Attempt != right.Attempt:
		return left.Attempt > right.Attempt
	case !left.QueuedAt.Equal(right.QueuedAt):
		return left.QueuedAt.After(right.QueuedAt)
	default:
		return left.ID > right.ID
	}
}

func (m *TaskManager) hasUnresolvedDependencies(ctx context.Context, dependencies []TaskDependency) (bool, error) {
	for _, dependency := range dependencies {
		record, err := m.reconcileTask(ctx, dependency.DependsOnTaskID)
		if err != nil {
			return false, err
		}
		if record.Status.Normalize() != TaskStatusCompleted {
			return true, nil
		}
	}
	return false, nil
}

func (m *TaskManager) reconcileDependentTasks(ctx context.Context, taskID string, visited map[string]struct{}) error {
	dependents, err := m.store.ListDependents(ctx, taskID)
	if err != nil {
		return err
	}

	for _, dependent := range dependents {
		dependentTaskID := strings.TrimSpace(dependent.TaskID)
		if _, seen := visited[dependentTaskID]; seen {
			continue
		}
		visited[dependentTaskID] = struct{}{}

		previous, err := m.store.GetTask(ctx, dependentTaskID)
		if err != nil {
			return err
		}
		reconciled, err := m.reconcileTask(ctx, dependentTaskID)
		if err != nil {
			return err
		}
		if previous.Status.Normalize() != reconciled.Status.Normalize() {
			if err := m.reconcileDependentTasks(ctx, dependentTaskID, visited); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *TaskManager) lookupIdempotentRun(ctx context.Context, key string, origin Origin, taskID string) (*TaskRun, error) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return nil, nil
	}

	run, err := m.store.GetTaskRunByIdempotencyKey(ctx, trimmedKey, origin)
	if err != nil {
		if errors.Is(err, ErrTaskRunIdempotencyNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if strings.TrimSpace(run.TaskID) != strings.TrimSpace(taskID) {
		return nil, fmt.Errorf("%w: idempotency key %q already maps to task %q", ErrValidation, trimmedKey, run.TaskID)
	}
	return &run, nil
}

func (m *TaskManager) saveRunIdempotency(ctx context.Context, run TaskRun, origin Origin) error {
	if strings.TrimSpace(run.IdempotencyKey) == "" {
		return nil
	}
	return m.store.SaveTaskRunIdempotency(ctx, TaskRunIdempotency{
		IdempotencyKey: run.IdempotencyKey,
		RunID:          run.ID,
		Origin:         origin,
		CreatedAt:      m.now().UTC(),
	})
}

func (m *TaskManager) loadRunWithTask(ctx context.Context, runID string) (TaskRun, Task, error) {
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return TaskRun{}, Task{}, fmt.Errorf("%w: task run id is required", ErrValidation)
	}

	run, err := m.store.GetTaskRun(ctx, trimmedRunID)
	if err != nil {
		return TaskRun{}, Task{}, err
	}
	taskRecord, err := m.store.GetTask(ctx, run.TaskID)
	if err != nil {
		return TaskRun{}, Task{}, err
	}
	return run, taskRecord, nil
}

func (m *TaskManager) ensureTaskExecutable(ctx context.Context, record Task) error {
	dependencies, err := m.store.ListDependencies(ctx, record.ID)
	if err != nil {
		return err
	}
	runs, err := m.store.ListTaskRuns(ctx, TaskRunQuery{TaskID: record.ID})
	if err != nil {
		return err
	}
	status, err := m.canonicalTaskStatus(ctx, record, dependencies, runs)
	if err != nil {
		return err
	}

	switch status.Normalize() {
	case TaskStatusBlocked:
		return fmt.Errorf("%w: task %q is blocked", ErrInvalidStatusTransition, record.ID)
	case TaskStatusCancelled:
		return fmt.Errorf("%w: task %q is cancelled", ErrInvalidStatusTransition, record.ID)
	default:
		return nil
	}
}

func (m *TaskManager) requireSessionExecutor(action string) error {
	if m.sessions == nil {
		return fmt.Errorf("%w: session executor is required to %s", ErrValidation, action)
	}
	return nil
}

func (m *TaskManager) collectTaskTree(ctx context.Context, rootTaskID string) ([]Task, error) {
	root, err := m.store.GetTask(ctx, rootTaskID)
	if err != nil {
		return nil, err
	}

	tree := []Task{root}
	queue := []Task{root}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		children, err := m.store.ListTasks(ctx, TaskQuery{ParentTaskID: current.ID})
		if err != nil {
			return nil, err
		}
		for _, child := range children {
			record, err := m.store.GetTask(ctx, child.ID)
			if err != nil {
				return nil, err
			}
			tree = append(tree, record)
			queue = append(queue, record)
		}
	}

	return tree, nil
}

type cancelRunOptions struct {
	propagatedFromTaskID string
	reconcileTask        bool
}

func (m *TaskManager) failRunRecord(ctx context.Context, taskRecord Task, run TaskRun, failure RunFailure, actor ActorContext) (*TaskRun, error) {
	switch run.Status.Normalize() {
	case TaskRunStatusStarting, TaskRunStatusRunning:
	default:
		return nil, requireRunTransition(run, TaskRunStatusFailed)
	}

	run.Status = TaskRunStatusFailed
	run.Error = failure.Error
	run.Result = nil
	run.EndedAt = m.now().UTC()
	if err := m.store.UpdateTaskRun(ctx, run); err != nil {
		return nil, err
	}

	reconciledTask, err := m.reconcileTaskCascade(ctx, taskRecord.ID)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunFailed, actor, failedRunPayload{
		Status:     run.Status,
		TaskStatus: reconciledTask.Status,
		Error:      run.Error,
		Metadata:   cloneRawJSON(failure.Metadata),
	}); err != nil {
		return nil, err
	}

	return &run, nil
}

func (m *TaskManager) cancelRunRecord(ctx context.Context, taskRecord Task, run TaskRun, req CancelRun, actor ActorContext, opts cancelRunOptions) (*TaskRun, error) {
	status := run.Status.Normalize()
	switch status {
	case TaskRunStatusQueued, TaskRunStatusClaimed, TaskRunStatusStarting, TaskRunStatusRunning:
	default:
		return nil, requireRunTransition(run, TaskRunStatusCancelled)
	}

	sessionID := strings.TrimSpace(run.SessionID)
	activeSession := (status == TaskRunStatusStarting || status == TaskRunStatusRunning) && sessionID != ""
	if activeSession {
		if err := m.requireSessionExecutor("cancel active run"); err != nil {
			return nil, err
		}
	}

	run.Status = TaskRunStatusCancelled
	run.Result = nil
	run.Error = ""
	run.EndedAt = m.now().UTC()
	if err := m.store.UpdateTaskRun(ctx, run); err != nil {
		return nil, err
	}

	cooperativeStopRequested := false
	if activeSession {
		if err := m.sessions.RequestTaskStop(ctx, sessionID, StopReasonCancellation); err != nil {
			return nil, fmt.Errorf("task: request stop for session %q: %w", sessionID, err)
		}
		cooperativeStopRequested = true
	}

	reconciledTask := taskRecord
	if opts.reconcileTask {
		var err error
		reconciledTask, err = m.reconcileTaskCascade(ctx, taskRecord.ID)
		if err != nil {
			return nil, err
		}
	}
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunCancelled, actor, cancelledRunPayload{
		Status:                   run.Status,
		TaskStatus:               reconciledTask.Status,
		Reason:                   req.Reason,
		Metadata:                 cloneRawJSON(req.Metadata),
		SessionID:                sessionID,
		PropagatedFromTaskID:     opts.propagatedFromTaskID,
		CooperativeStopRequested: cooperativeStopRequested,
	}); err != nil {
		return nil, err
	}

	if activeSession {
		if err := m.waitAndForceStopRun(ctx, sessionID); err != nil {
			return nil, err
		}
		if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunForceStopped, actor, forceStoppedRunPayload{
			SessionID:            sessionID,
			GraceTimeoutMillis:   m.cancelGracePeriod.Milliseconds(),
			PropagatedFromTaskID: opts.propagatedFromTaskID,
		}); err != nil {
			return nil, err
		}
	}

	return &run, nil
}

func (m *TaskManager) waitAndForceStopRun(ctx context.Context, sessionID string) error {
	if m.cancelGracePeriod > 0 {
		timer := time.NewTimer(m.cancelGracePeriod)
		defer timer.Stop()

		select {
		case <-timer.C:
		case <-ctx.Done():
			return fmt.Errorf("task: wait for force-stop grace period on session %q: %w", sessionID, ctx.Err())
		}
	}
	if err := m.sessions.ForceTaskStop(ctx, sessionID, StopReasonCancellation); err != nil {
		return fmt.Errorf("task: force stop session %q: %w", sessionID, err)
	}
	return nil
}

func requireRunTransition(run TaskRun, next TaskRunStatus) error {
	current := run.Status.Normalize()
	target := next.Normalize()
	if allowsRunTransition(current, target) {
		return nil
	}
	return fmt.Errorf("%w: task run %q cannot transition from %q to %q", ErrInvalidStatusTransition, run.ID, current, target)
}

func allowsRunTransition(current TaskRunStatus, next TaskRunStatus) bool {
	switch current.Normalize() {
	case TaskRunStatusQueued:
		return next.Normalize() == TaskRunStatusClaimed || next.Normalize() == TaskRunStatusCancelled
	case TaskRunStatusClaimed:
		switch next.Normalize() {
		case TaskRunStatusStarting, TaskRunStatusCancelled:
			return true
		}
	case TaskRunStatusStarting:
		switch next.Normalize() {
		case TaskRunStatusRunning, TaskRunStatusFailed, TaskRunStatusCancelled:
			return true
		}
	case TaskRunStatusRunning:
		switch next.Normalize() {
		case TaskRunStatusCompleted, TaskRunStatusFailed, TaskRunStatusCancelled:
			return true
		}
	}
	return false
}

func nextRunAttempt(runs []TaskRun) int {
	maxAttempt := 0
	for _, run := range runs {
		if run.Attempt > maxAttempt {
			maxAttempt = run.Attempt
		}
	}
	return maxAttempt + 1
}

func (m *TaskManager) validateNetworkChannel(path string, channel string) error {
	if m == nil || m.channelValidator == nil {
		return nil
	}

	trimmed := strings.TrimSpace(channel)
	if trimmed == "" {
		return nil
	}
	if err := m.channelValidator(trimmed); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrValidation, path, err)
	}
	return nil
}

func (m *TaskManager) validateRunChannelUsable(ctx context.Context, taskRecord Task, run TaskRun, actor ActorContext, operation string) error {
	channel := resolvedRunChannel(run.NetworkChannel, taskRecord.NetworkChannel)
	if strings.TrimSpace(channel) == "" {
		return nil
	}
	if err := m.validateNetworkChannel("task_run.network_channel", channel); err == nil {
		return nil
	}

	rejectedErr := fmt.Errorf(
		"%w: task %q run %q channel %q is no longer valid",
		ErrStaleNetworkChannel,
		taskRecord.ID,
		run.ID,
		strings.TrimSpace(channel),
	)
	if recordErr := m.recordTaskEvent(ctx, taskRecord.ID, run.ID, taskEventRunRejected, actor, rejectedRunPayload{
		Operation:      strings.TrimSpace(operation),
		Reason:         "stale_network_channel",
		NetworkChannel: strings.TrimSpace(channel),
	}); recordErr != nil {
		return errorsJoin(rejectedErr, recordErr)
	}
	return rejectedErr
}

func resolvedRunChannel(requested string, taskChannel string) string {
	if strings.TrimSpace(requested) != "" {
		return strings.TrimSpace(requested)
	}
	return strings.TrimSpace(taskChannel)
}

func errorsJoin(errs ...error) error {
	return errors.Join(errs...)
}

func runBootRecoveryError(run TaskRun, recovery RunBootRecovery) string {
	sessionID := strings.TrimSpace(run.SessionID)
	switch {
	case sessionID != "" && recovery.SessionState != "":
		return fmt.Sprintf("orphaned on boot: session %q is %s", sessionID, recovery.SessionState)
	case sessionID != "":
		return fmt.Sprintf("orphaned on boot: session %q is not live", sessionID)
	default:
		return "orphaned on boot: run has no live session"
	}
}

func runBootRecoveryMetadata(run TaskRun, recovery RunBootRecovery) json.RawMessage {
	payload, err := marshalTaskEventPayload(map[string]string{
		"reason":          normalizedBootRecoveryReason(recovery.Reason),
		"previous_status": string(run.Status.Normalize()),
		"session_id":      strings.TrimSpace(run.SessionID),
		"session_state":   strings.TrimSpace(recovery.SessionState),
	})
	if err != nil {
		return nil
	}
	return payload
}

func normalizedBootRecoveryReason(reason string) string {
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return "orphaned_on_boot"
	}
	return trimmed
}

func (m *TaskManager) recordTaskEvent(ctx context.Context, taskID string, runID string, eventType string, actor ActorContext, payload any) error {
	rawPayload, err := marshalTaskEventPayload(payload)
	if err != nil {
		return err
	}
	return m.store.CreateTaskEvent(ctx, TaskEvent{
		ID:        m.newID("evt"),
		TaskID:    strings.TrimSpace(taskID),
		RunID:     strings.TrimSpace(runID),
		EventType: strings.TrimSpace(eventType),
		Actor:     actor.Actor,
		Origin:    actor.Origin,
		Payload:   rawPayload,
		Timestamp: m.now().UTC(),
	})
}

func marshalTaskEventPayload(payload any) (json.RawMessage, error) {
	if payload == nil {
		return nil, nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("task: marshal task event payload: %w", err)
	}
	return json.RawMessage(raw), nil
}

func normalizeOwnership(owner *Ownership) *Ownership {
	if owner == nil {
		return nil
	}
	normalized := *owner
	normalized.Kind = normalized.Kind.Normalize()
	normalized.Ref = strings.TrimSpace(normalized.Ref)
	if normalized.IsZero() {
		return nil
	}
	return &normalized
}

func cloneOwnership(owner *Ownership) *Ownership {
	if owner == nil {
		return nil
	}
	cloned := *owner
	return &cloned
}

func sameOwnership(left *Ownership, right *Ownership) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return left.Kind.Normalize() == right.Kind.Normalize() && strings.TrimSpace(left.Ref) == strings.TrimSpace(right.Ref)
	}
}

func normalizeRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil
	}
	return json.RawMessage(trimmed)
}

func cloneRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	cloned := make(json.RawMessage, len(raw))
	copy(cloned, raw)
	return cloned
}

func sameRawJSON(left json.RawMessage, right json.RawMessage) bool {
	return string(normalizeRawJSON(left)) == string(normalizeRawJSON(right))
}

type createdTaskPayload struct {
	Scope          Scope      `json:"scope"`
	WorkspaceID    string     `json:"workspace_id,omitempty"`
	ParentTaskID   string     `json:"parent_task_id,omitempty"`
	Status         TaskStatus `json:"status"`
	NetworkChannel string     `json:"network_channel,omitempty"`
	Owner          *Ownership `json:"owner,omitempty"`
}

type updatedTaskPayload struct {
	ChangedFields []string   `json:"changed_fields"`
	Status        TaskStatus `json:"status"`
}

type childCreatedTaskPayload struct {
	ChildTaskID      string `json:"child_task_id"`
	ChildScope       Scope  `json:"child_scope"`
	ChildWorkspaceID string `json:"child_workspace_id,omitempty"`
}

type dependencyTaskPayload struct {
	DependsOnTaskID string         `json:"depends_on_task_id"`
	Kind            DependencyKind `json:"kind"`
	Status          TaskStatus     `json:"status"`
}

type cancelledTaskPayload struct {
	Reason               string          `json:"reason,omitempty"`
	Metadata             json.RawMessage `json:"metadata,omitempty"`
	Status               TaskStatus      `json:"status"`
	PropagatedFromTaskID string          `json:"propagated_from_task_id,omitempty"`
	CancelledRunIDs      []string        `json:"cancelled_run_ids,omitempty"`
}

type runEnqueuedPayload struct {
	Attempt        int           `json:"attempt"`
	Status         TaskRunStatus `json:"status"`
	TaskStatus     TaskStatus    `json:"task_status"`
	NetworkChannel string        `json:"network_channel,omitempty"`
	IdempotencyKey string        `json:"idempotency_key,omitempty"`
}

type runClaimedPayload struct {
	Status     TaskRunStatus `json:"status"`
	TaskStatus TaskStatus    `json:"task_status"`
	ClaimedBy  ActorIdentity `json:"claimed_by"`
}

type runTransitionPayload struct {
	Status     TaskRunStatus `json:"status"`
	TaskStatus TaskStatus    `json:"task_status"`
	SessionID  string        `json:"session_id,omitempty"`
}

type completedRunPayload struct {
	Status     TaskRunStatus   `json:"status"`
	TaskStatus TaskStatus      `json:"task_status"`
	Result     json.RawMessage `json:"result,omitempty"`
}

type failedRunPayload struct {
	Status     TaskRunStatus   `json:"status"`
	TaskStatus TaskStatus      `json:"task_status"`
	Error      string          `json:"error"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

type cancelledRunPayload struct {
	Status                   TaskRunStatus   `json:"status"`
	TaskStatus               TaskStatus      `json:"task_status,omitempty"`
	Reason                   string          `json:"reason,omitempty"`
	Metadata                 json.RawMessage `json:"metadata,omitempty"`
	SessionID                string          `json:"session_id,omitempty"`
	PropagatedFromTaskID     string          `json:"propagated_from_task_id,omitempty"`
	CooperativeStopRequested bool            `json:"cooperative_stop_requested,omitempty"`
}

type forceStoppedRunPayload struct {
	SessionID            string `json:"session_id"`
	GraceTimeoutMillis   int64  `json:"grace_timeout_ms"`
	PropagatedFromTaskID string `json:"propagated_from_task_id,omitempty"`
}

type rejectedRunPayload struct {
	Operation      string `json:"operation"`
	Reason         string `json:"reason"`
	NetworkChannel string `json:"network_channel,omitempty"`
}

type recoveredRunPayload struct {
	Action         RunBootRecoveryAction `json:"action"`
	PreviousStatus TaskRunStatus         `json:"previous_status"`
	Status         TaskRunStatus         `json:"status"`
	TaskStatus     TaskStatus            `json:"task_status"`
	Reason         string                `json:"reason,omitempty"`
	SessionID      string                `json:"session_id,omitempty"`
	SessionState   string                `json:"session_state,omitempty"`
}
