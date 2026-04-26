package task

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/store"
)

const (
	taskEventCreated           = "task.created"
	taskEventUpdated           = "task.updated"
	taskEventPublished         = "task.published"
	taskEventApproved          = "task.approved"
	taskEventRejected          = "task.rejected"
	taskEventCanceled          = "task.canceled"
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
	taskEventRunCanceled       = "task.run_canceled"
	taskEventRunForceStopped   = "task.run_force_stopped"
	taskEventRunRecovered      = "task.run_recovered"
	taskEventRunRejected       = "task.run_rejected"
)

// Option customizes Service construction.
type Option func(*managerOptions)

type managerOptions struct {
	store             Store
	sessions          SessionExecutor
	runtimeViews      RuntimeViewReader
	eventObserver     EventObserver
	taskHooks         RunHookDispatcher
	channelValidator  func(string) error
	now               func() time.Time
	newID             func(prefix string) string
	cancelGracePeriod time.Duration
}

// Service centralizes canonical task-domain creation, mutation, read, and
// graph-management rules above the persistence layer.
type Service struct {
	store             Store
	sessions          SessionExecutor
	runtimeViews      RuntimeViewReader
	eventObserver     EventObserver
	taskHooks         RunHookDispatcher
	channelValidator  func(string) error
	now               func() time.Time
	newID             func(prefix string) string
	cancelGracePeriod time.Duration
	liveMu            sync.Mutex
	liveSubscribers   map[uint64]*taskStreamSubscriber
	nextSubscriberID  uint64
}

var _ Manager = (*Service)(nil)

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

// WithRuntimeViewReader injects optional session telemetry enrichment for task live reads.
func WithRuntimeViewReader(reader RuntimeViewReader) Option {
	return func(opts *managerOptions) {
		opts.runtimeViews = reader
	}
}

// WithEventObserver injects a best-effort observer for immutable task events.
func WithEventObserver(observer EventObserver) Option {
	return func(opts *managerOptions) {
		opts.eventObserver = observer
	}
}

// WithTaskRunHooks injects the task-run hook bridge used at authoritative run transitions.
func WithTaskRunHooks(hooks RunHookDispatcher) Option {
	return func(opts *managerOptions) {
		opts.taskHooks = hooks
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
func NewManager(opts ...Option) (*Service, error) {
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

	return &Service{
		store:             options.store,
		sessions:          options.sessions,
		runtimeViews:      options.runtimeViews,
		eventObserver:     options.eventObserver,
		taskHooks:         defaultTaskRunHooks(options.taskHooks),
		channelValidator:  options.channelValidator,
		now:               options.now,
		newID:             options.newID,
		cancelGracePeriod: options.cancelGracePeriod,
		liveSubscribers:   make(map[uint64]*taskStreamSubscriber),
	}, nil
}

// CreateTask derives one canonical task record from trusted actor context and
// persists the corresponding immutable audit event.
func (m *Service) CreateTask(ctx context.Context, spec CreateTask, actor ActorContext) (*Task, error) {
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
		Priority:       normalizedSpec.Priority,
		MaxAttempts:    createTaskMaxAttempts(normalizedSpec),
		Status:         createdTaskStatus(normalizedSpec),
		ApprovalPolicy: normalizedSpec.ApprovalPolicy,
		ApprovalState:  defaultApprovalStateForPolicy(normalizedSpec.ApprovalPolicy),
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
func (m *Service) CreateChildTask(
	ctx context.Context,
	parentTaskID string,
	spec CreateTask,
	actor ActorContext,
) (*Task, error) {
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

// DeleteTask removes one task after verifying it is not still in use by child
// tasks or non-terminal runs, then reconciles any dependents unblocked by the
// cascade-deleted dependency edges.
func (m *Service) DeleteTask(ctx context.Context, id string, actor ActorContext) error {
	if err := requireWriteAuthority(actor); err != nil {
		return err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return fmt.Errorf("%w: task id is required", ErrValidation)
	}

	if txStore, ok := m.store.(DeleteTaskTransactionStore); ok {
		return txStore.WithDeleteTaskTransaction(ctx, func(store DeleteTaskMutationStore) error {
			return m.deleteTaskWithStore(ctx, store, trimmedID)
		})
	}

	return m.deleteTaskWithStore(ctx, m.store, trimmedID)
}

func (m *Service) deleteTaskWithStore(
	ctx context.Context,
	store DeleteTaskMutationStore,
	trimmedID string,
) error {
	record, err := store.GetTask(ctx, trimmedID)
	if err != nil {
		return fmt.Errorf("task: load task %q for delete: %w", trimmedID, err)
	}
	if err := m.ensureTaskDeleteAllowedWithStore(ctx, store, record); err != nil {
		return err
	}

	dependents, err := store.ListDependents(ctx, trimmedID)
	if err != nil {
		return fmt.Errorf("task: list dependents for task %q delete: %w", trimmedID, err)
	}
	dependentIDs := uniqueDependentTaskIDs(dependents)

	if err := store.DeleteTask(ctx, trimmedID); err != nil {
		return fmt.Errorf("task: delete task %q: %w", trimmedID, err)
	}

	for _, dependentID := range dependentIDs {
		if _, err := m.reconcileTaskCascadeWithStore(ctx, store, dependentID); err != nil {
			return fmt.Errorf(
				"task: reconcile dependent task %q after deleting %q: %w",
				dependentID,
				trimmedID,
				err,
			)
		}
	}

	return nil
}

// UpdateTask applies one mutable patch while preserving immutable identity and
// structural fields under manager control.
func (m *Service) UpdateTask(ctx context.Context, id string, patch Patch, actor ActorContext) (*Task, error) {
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

	updated, changedFields := applyTaskPatch(current, normalizedPatch)
	if len(changedFields) == 0 {
		return &current, nil
	}

	dependencies, err := m.store.ListDependencies(ctx, trimmedID)
	if err != nil {
		return nil, err
	}
	runs, err := m.store.ListTaskRuns(ctx, RunQuery{TaskID: trimmedID})
	if err != nil {
		return nil, err
	}

	canonicalStatus, err := m.canonicalTaskStatus(ctx, updated, dependencies, runs)
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
func (m *Service) CancelTask(ctx context.Context, id string, req CancelTask, actor ActorContext) (*Task, error) {
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

	tree, root, err := m.loadCancellationTree(ctx, trimmedID)
	if err != nil {
		return nil, err
	}
	if err := m.ensureTaskCancelable(ctx, root); err != nil {
		return nil, err
	}

	cancelledRoot := root
	for idx, record := range tree {
		record, err = m.cancelTaskTreeRecord(ctx, trimmedID, idx, record, normalizedReq, actor)
		if err != nil {
			return nil, err
		}
		if record.ID == trimmedID {
			cancelledRoot = record
		}
	}

	return &cancelledRoot, nil
}

func applyTaskPatch(current Task, patch Patch) (Task, []string) {
	updated := current
	changedFields := make([]string, 0, len(MutableTaskFields()))

	if patch.Title != nil && updated.Title != *patch.Title {
		updated.Title = *patch.Title
		changedFields = append(changedFields, TaskFieldTitle)
	}
	if patch.Description != nil && updated.Description != *patch.Description {
		updated.Description = *patch.Description
		changedFields = append(changedFields, TaskFieldDescription)
	}
	if patch.Priority != nil && updated.Priority != *patch.Priority {
		updated.Priority = *patch.Priority
		changedFields = append(changedFields, TaskFieldPriority)
	}
	if patch.MaxAttempts != nil && updated.MaxAttempts != *patch.MaxAttempts {
		updated.MaxAttempts = *patch.MaxAttempts
		changedFields = append(changedFields, TaskFieldMaxAttempts)
	}
	if patch.ApprovalPolicy != nil && updated.ApprovalPolicy != *patch.ApprovalPolicy {
		updated.ApprovalPolicy = *patch.ApprovalPolicy
		updated.ApprovalState = defaultApprovalStateForPolicy(*patch.ApprovalPolicy)
		changedFields = append(changedFields, TaskFieldApprovalPolicy)
	}
	if patch.Metadata != nil && !sameRawJSON(updated.Metadata, *patch.Metadata) {
		updated.Metadata = cloneRawJSON(*patch.Metadata)
		changedFields = append(changedFields, TaskFieldMetadata)
	}
	if patch.NetworkChannel != nil && updated.NetworkChannel != *patch.NetworkChannel {
		updated.NetworkChannel = *patch.NetworkChannel
		changedFields = append(changedFields, TaskFieldNetworkChannel)
	}
	if patch.Owner != nil && !sameOwnership(updated.Owner, patch.Owner) {
		updated.Owner = cloneOwnership(patch.Owner)
		changedFields = append(changedFields, TaskFieldOwner)
	}
	if patch.ClearOwner && updated.Owner != nil {
		updated.Owner = nil
		changedFields = append(changedFields, TaskFieldOwner)
	}

	return updated, changedFields
}

func createTaskMaxAttempts(spec CreateTask) int {
	if spec.MaxAttempts == nil {
		return DefaultTaskMaxAttempts
	}
	return normalizeTaskMaxAttemptsOrDefault(*spec.MaxAttempts)
}

func createdTaskStatus(spec CreateTask) Status {
	if spec.Draft {
		return TaskStatusDraft
	}
	if approvalStateBlocksExecution(
		normalizeApprovalPolicyOrDefault(spec.ApprovalPolicy),
		defaultApprovalStateForPolicy(spec.ApprovalPolicy),
	) {
		return TaskStatusBlocked
	}
	return TaskStatusReady
}

// PublishTask transitions one durable draft into manager-owned runnable reconciliation.
func (m *Service) PublishTask(ctx context.Context, id string, actor ActorContext) (*Task, error) {
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
	if record.Status.Normalize() != TaskStatusDraft {
		return nil, fmt.Errorf(
			"%w: task %q cannot publish from %q",
			ErrInvalidStatusTransition,
			record.ID,
			record.Status,
		)
	}

	record.Status = TaskStatusPending
	record.UpdatedAt = m.now().UTC()
	record.ClosedAt = time.Time{}
	if err := m.store.UpdateTask(ctx, record); err != nil {
		return nil, err
	}

	reconciled, err := m.reconcileTaskCascade(ctx, record.ID)
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

// ApproveTask records one approval decision for a manual-approval task that is
// currently awaiting a decision and reconciles the resulting task status.
func (m *Service) ApproveTask(ctx context.Context, id string, actor ActorContext) (*Task, error) {
	return m.transitionTaskApproval(ctx, id, ApprovalStateApproved, taskEventApproved, actor)
}

// RejectTask records one rejection decision for a manual-approval task that is
// currently awaiting a decision and reconciles the resulting task status.
func (m *Service) RejectTask(ctx context.Context, id string, actor ActorContext) (*Task, error) {
	return m.transitionTaskApproval(ctx, id, ApprovalStateRejected, taskEventRejected, actor)
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

	previousApprovalState := normalizeApprovalStateOrDefault(record.ApprovalPolicy, record.ApprovalState)
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
	if err := m.store.UpdateTask(ctx, record); err != nil {
		return nil, err
	}

	reconciled, err := m.reconcileTaskCascade(ctx, record.ID)
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
	return normalizeApprovalStateOrDefault(normalizedPolicy, record.ApprovalState) == ApprovalStatePending
}

// MarkTaskRead persists the current actor-scoped triage state as read for the
// task's latest known activity snapshot.
func (m *Service) MarkTaskRead(ctx context.Context, id string, actor ActorContext) (TriageState, error) {
	return m.mutateTaskTriage(ctx, id, actor, func(state *TriageState, latestActivity time.Time) {
		state.Read = true
		state.Dismissed = false
		state.LastSeenActivityAt = latestActivity
	})
}

// ArchiveTask persists the current actor-scoped triage state as archived for
// the task's latest known activity snapshot.
func (m *Service) ArchiveTask(ctx context.Context, id string, actor ActorContext) (TriageState, error) {
	return m.mutateTaskTriage(ctx, id, actor, func(state *TriageState, latestActivity time.Time) {
		state.Read = true
		state.Archived = true
		state.Dismissed = false
		state.LastSeenActivityAt = latestActivity
	})
}

// DismissTask persists the current actor-scoped triage state as dismissed for
// the task's latest known activity snapshot.
func (m *Service) DismissTask(ctx context.Context, id string, actor ActorContext) (TriageState, error) {
	return m.mutateTaskTriage(ctx, id, actor, func(state *TriageState, latestActivity time.Time) {
		state.Read = true
		state.Dismissed = true
		state.LastSeenActivityAt = latestActivity
	})
}

func (m *Service) mutateTaskTriage(
	ctx context.Context,
	id string,
	actor ActorContext,
	mutate func(state *TriageState, latestActivity time.Time),
) (TriageState, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return TriageState{}, err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return TriageState{}, fmt.Errorf("%w: task id is required", ErrValidation)
	}

	record, err := m.store.GetTask(ctx, trimmedID)
	if err != nil {
		return TriageState{}, err
	}
	runs, err := m.store.ListTaskRuns(ctx, RunQuery{TaskID: trimmedID})
	if err != nil {
		return TriageState{}, err
	}
	events, err := m.store.ListTaskEvents(ctx, EventQuery{TaskID: trimmedID, Limit: 1})
	if err != nil {
		return TriageState{}, err
	}

	state, err := m.loadTaskTriageState(ctx, trimmedID, actor.Actor)
	if err != nil {
		return TriageState{}, err
	}
	mutate(&state, latestTaskActivityAt(record, runs, events))
	state.TaskID = trimmedID
	state.Actor = actor.Actor
	state.UpdatedAt = m.now().UTC()
	if err := m.store.UpsertTaskTriageState(ctx, state); err != nil {
		return TriageState{}, err
	}

	return state, nil
}

func (m *Service) loadTaskTriageState(
	ctx context.Context,
	taskID string,
	actor ActorIdentity,
) (TriageState, error) {
	state, err := m.store.GetTaskTriageState(ctx, taskID, actor)
	if err == nil {
		return state, nil
	}
	if errors.Is(err, ErrTaskTriageStateNotFound) {
		return TriageState{TaskID: taskID, Actor: actor}, nil
	}
	return TriageState{}, err
}

func (m *Service) loadCancellationTree(ctx context.Context, taskID string) ([]Task, Task, error) {
	tree, err := m.collectTaskTree(ctx, taskID)
	if err != nil {
		return nil, Task{}, err
	}
	if len(tree) == 0 {
		return nil, Task{}, ErrTaskNotFound
	}
	return tree, tree[0], nil
}

func (m *Service) ensureTaskCancelable(ctx context.Context, root Task) error {
	rootRuns, rootStatus, err := m.loadTaskRuntimeState(ctx, root)
	if err != nil {
		return err
	}
	if isTerminalTaskStatus(rootStatus) && rootStatus != TaskStatusCanceled && !hasOpenRun(rootRuns) {
		return fmt.Errorf(
			"%w: task %q cannot transition from %q to %q",
			ErrInvalidStatusTransition,
			root.ID,
			rootStatus,
			TaskStatusCanceled,
		)
	}
	return nil
}

func (m *Service) cancelTaskTreeRecord(
	ctx context.Context,
	rootTaskID string,
	idx int,
	record Task,
	req CancelTask,
	actor ActorContext,
) (Task, error) {
	runs, status, err := m.loadTaskRuntimeState(ctx, record)
	if err != nil {
		return Task{}, err
	}
	record.Status = status
	if idx > 0 && isTerminalTaskStatus(status) {
		return record, nil
	}

	propagatedFromTaskID := cancellationPropagationRoot(rootTaskID, idx)
	cancelledRunIDs, err := m.cancelOpenTaskRuns(ctx, record, runs, req, actor, propagatedFromTaskID)
	if err != nil {
		return Task{}, err
	}
	if status.Normalize() == TaskStatusCanceled && len(cancelledRunIDs) == 0 {
		return record, nil
	}
	return m.persistCancelledTask(ctx, record, req, actor, propagatedFromTaskID, cancelledRunIDs)
}

func (m *Service) loadTaskRuntimeState(
	ctx context.Context,
	record Task,
) ([]Run, Status, error) {
	runs, err := m.store.ListTaskRuns(ctx, RunQuery{TaskID: record.ID})
	if err != nil {
		return nil, "", err
	}
	dependencies, err := m.store.ListDependencies(ctx, record.ID)
	if err != nil {
		return nil, "", err
	}
	status, err := m.canonicalTaskStatus(ctx, record, dependencies, runs)
	if err != nil {
		return nil, "", err
	}
	return runs, status, nil
}

func cancellationPropagationRoot(rootTaskID string, idx int) string {
	if idx == 0 {
		return ""
	}
	return rootTaskID
}

func (m *Service) cancelOpenTaskRuns(
	ctx context.Context,
	record Task,
	runs []Run,
	req CancelTask,
	actor ActorContext,
	propagatedFromTaskID string,
) ([]string, error) {
	cancelledRunIDs := make([]string, 0)
	for _, run := range runs {
		if isTerminalRunStatus(run.Status) {
			continue
		}
		cancelledRun, err := m.cancelRunRecord(ctx, record, run, CancelRun(req), actor, cancelRunOptions{
			propagatedFromTaskID: propagatedFromTaskID,
			reconcileTask:        false,
		})
		if err != nil {
			return nil, err
		}
		cancelledRunIDs = append(cancelledRunIDs, cancelledRun.ID)
	}
	return cancelledRunIDs, nil
}

func (m *Service) persistCancelledTask(
	ctx context.Context,
	record Task,
	req CancelTask,
	actor ActorContext,
	propagatedFromTaskID string,
	cancelledRunIDs []string,
) (Task, error) {
	record.Status = TaskStatusCanceled
	record.UpdatedAt = m.now().UTC()
	record.ClosedAt = record.UpdatedAt
	if err := m.store.UpdateTask(ctx, record); err != nil {
		return Task{}, err
	}
	if err := m.recordTaskEvent(ctx, record.ID, "", taskEventCanceled, actor, cancelledTaskPayload{
		Reason:               req.Reason,
		Metadata:             cloneRawJSON(req.Metadata),
		Status:               record.Status,
		PropagatedFromTaskID: propagatedFromTaskID,
		CancelledRunIDs:      append([]string(nil), cancelledRunIDs...),
	}); err != nil {
		return Task{}, err
	}
	if err := m.reconcileDependentTasks(ctx, record.ID, map[string]struct{}{record.ID: {}}); err != nil {
		return Task{}, err
	}
	return record, nil
}

func (m *Service) transitionClaimedRunToStarting(
	ctx context.Context,
	taskRecord Task,
	run Run,
	actor ActorContext,
) (Run, *Run, error) {
	if err := m.requireSessionExecutor("start run"); err != nil {
		return Run{}, nil, err
	}

	run.Status = TaskRunStatusStarting
	if err := m.store.UpdateTaskRun(ctx, run); err != nil {
		return Run{}, nil, err
	}

	startingTask, err := m.reconcileTaskCascade(ctx, run.TaskID)
	if err != nil {
		return Run{}, nil, err
	}
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunStarting, actor, runTransitionPayload{
		Status:     run.Status,
		TaskStatus: startingTask.Status,
		SessionID:  run.SessionID,
	}); err != nil {
		return Run{}, nil, err
	}

	sessionID, failedRun, err := m.startRunSession(ctx, taskRecord, startingTask, run, actor)
	if err != nil {
		return Run{}, failedRun, err
	}
	run.SessionID = sessionID
	return run, nil, nil
}

func (m *Service) startRunSession(
	ctx context.Context,
	taskRecord Task,
	startingTask Task,
	run Run,
	actor ActorContext,
) (string, *Run, error) {
	sessionRef, err := m.sessions.StartTaskSession(ctx, &StartTaskSession{
		Task:  startingTask,
		Run:   run,
		Actor: actor,
	})
	if err != nil {
		message := fmt.Sprintf("start task session: %v", err)
		failedRun, failErr := m.failRunAfterSessionStartError(ctx, taskRecord, run, actor, message)
		if failErr != nil {
			return "", nil, errorsJoin(err, failErr)
		}
		return "", failedRun, fmt.Errorf("task: start task session for run %q: %w", run.ID, err)
	}
	if sessionRef == nil {
		failedRun, failErr := m.failRunAfterSessionStartError(
			ctx,
			taskRecord,
			run,
			actor,
			"start task session: nil session reference",
		)
		if failErr != nil {
			return "", nil, failErr
		}
		return "", failedRun, fmt.Errorf("%w: start_task_session returned nil session reference", ErrValidation)
	}
	if err := sessionRef.Validate(); err != nil {
		message := fmt.Sprintf("start task session: %v", err)
		failedRun, failErr := m.failRunAfterSessionStartError(ctx, taskRecord, run, actor, message)
		if failErr != nil {
			return "", nil, errorsJoin(err, failErr)
		}
		return "", failedRun, err
	}
	return strings.TrimSpace(sessionRef.SessionID), nil, nil
}

func (m *Service) failRunAfterSessionStartError(
	ctx context.Context,
	taskRecord Task,
	run Run,
	actor ActorContext,
	message string,
) (*Run, error) {
	return m.failRunRecord(ctx, taskRecord, run, RunFailure{Error: message}, actor)
}

func validateRunningSessionBinding(run Run) error {
	if strings.TrimSpace(run.SessionID) == "" {
		return fmt.Errorf(
			"%w: task run %q cannot transition from %q to %q without a session binding",
			ErrInvalidStatusTransition,
			run.ID,
			run.Status,
			TaskRunStatusRunning,
		)
	}
	return nil
}

func (m *Service) recoverRunByRequeue(
	ctx context.Context,
	taskRecord Task,
	run Run,
	recovery RunBootRecovery,
	actor ActorContext,
	previousStatus RunStatus,
	previousSessionID string,
) (*Run, error) {
	if previousStatus != TaskRunStatusClaimed {
		return nil, invalidRunRecoveryTransition(run, previousStatus, recovery.Action)
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
	reconciledTask, err := m.recordRecoveredRun(
		ctx,
		taskRecord.ID,
		run,
		recovery,
		actor,
		previousStatus,
		previousSessionID,
	)
	if err != nil {
		return nil, err
	}
	if err := m.dispatchTaskRunLeaseRecovered(
		ctx,
		run,
		reconciledTask,
		actor,
		previousStatus,
		previousSessionID,
		recovery,
	); err != nil {
		return nil, err
	}
	return &run, nil
}

func (m *Service) recoverRunByMarkRunning(
	ctx context.Context,
	taskRecord Task,
	run Run,
	recovery RunBootRecovery,
	actor ActorContext,
	previousStatus RunStatus,
	previousSessionID string,
) (*Run, error) {
	switch previousStatus {
	case TaskRunStatusClaimed, TaskRunStatusStarting:
	case TaskRunStatusRunning:
		return &run, nil
	default:
		return nil, invalidRunRecoveryTransition(run, previousStatus, recovery.Action)
	}
	if previousSessionID == "" {
		return nil, fmt.Errorf(
			"%w: task run %q cannot recover to running without a session binding",
			ErrInvalidStatusTransition,
			run.ID,
		)
	}

	run.Status = TaskRunStatusRunning
	if run.StartedAt.IsZero() {
		run.StartedAt = m.now().UTC()
	}
	if err := m.store.UpdateTaskRun(ctx, run); err != nil {
		return nil, err
	}
	reconciledTask, err := m.recordRecoveredRun(
		ctx,
		taskRecord.ID,
		run,
		recovery,
		actor,
		previousStatus,
		previousSessionID,
	)
	if err != nil {
		return nil, err
	}
	if err := m.dispatchTaskRunLeaseRecovered(
		ctx,
		run,
		reconciledTask,
		actor,
		previousStatus,
		previousSessionID,
		recovery,
	); err != nil {
		return nil, err
	}
	return &run, nil
}

func (m *Service) recoverRunByFailure(
	ctx context.Context,
	taskRecord Task,
	run Run,
	recovery RunBootRecovery,
	actor ActorContext,
	previousStatus RunStatus,
	previousSessionID string,
) (*Run, error) {
	failedRun, err := m.failRunRecord(ctx, taskRecord, run, RunFailure{
		Error:    runBootRecoveryError(run, recovery),
		Metadata: runBootRecoveryMetadata(run, recovery),
	}, actor)
	if err != nil {
		return nil, err
	}
	reconciledTask, err := m.recordRecoveredRun(
		ctx,
		taskRecord.ID,
		*failedRun,
		recovery,
		actor,
		previousStatus,
		previousSessionID,
	)
	if err != nil {
		return nil, err
	}
	if err := m.dispatchTaskRunLeaseRecovered(
		ctx,
		*failedRun,
		reconciledTask,
		actor,
		previousStatus,
		previousSessionID,
		recovery,
	); err != nil {
		return nil, err
	}
	return failedRun, nil
}

func invalidRunRecoveryTransition(run Run, previousStatus RunStatus, action RunBootRecoveryAction) error {
	return fmt.Errorf(
		"%w: task run %q cannot recover from %q via %q",
		ErrInvalidStatusTransition,
		run.ID,
		previousStatus,
		action,
	)
}

func (m *Service) recordRecoveredRun(
	ctx context.Context,
	taskID string,
	run Run,
	recovery RunBootRecovery,
	actor ActorContext,
	previousStatus RunStatus,
	previousSessionID string,
) (Task, error) {
	reconciledTask, err := m.reconcileTaskCascade(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunRecovered, actor, recoveredRunPayload{
		Action:         recovery.Action,
		PreviousStatus: previousStatus,
		Status:         run.Status,
		TaskStatus:     reconciledTask.Status,
		Reason:         recovery.Reason,
		SessionID:      previousSessionID,
		SessionState:   recovery.SessionState,
		Classification: recovery.Classification,
		Detail:         recovery.Detail,
	}); err != nil {
		return Task{}, err
	}
	return reconciledTask, nil
}

// GetTask returns one expanded task view after enforcing read authority.
func (m *Service) GetTask(ctx context.Context, id string, actor ActorContext) (*View, error) {
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

	children, err := m.listTaskSummaries(ctx, Query{ParentTaskID: trimmedID})
	if err != nil {
		return nil, err
	}
	dependencies, err := m.store.ListDependencies(ctx, trimmedID)
	if err != nil {
		return nil, err
	}
	runs, err := m.store.ListTaskRuns(ctx, RunQuery{TaskID: trimmedID})
	if err != nil {
		return nil, err
	}
	events, err := m.store.ListTaskEvents(ctx, EventQuery{TaskID: trimmedID})
	if err != nil {
		return nil, err
	}

	summary, err := m.enrichTaskSummaryFromState(ctx, record, len(children), dependencies, runs, events)
	if err != nil {
		return nil, err
	}
	dependencyReferences, err := m.buildDependencyReferences(ctx, dependencies)
	if err != nil {
		return nil, err
	}

	view := &View{
		Summary:              summary,
		Task:                 record,
		Children:             children,
		Dependencies:         dependencies,
		DependencyReferences: dependencyReferences,
		Runs:                 runs,
		Events:               events,
	}
	view.Task.Status = summary.Status
	return view, nil
}

// ListTaskRuns returns task runs for one task after enforcing read authority and
// task existence.
func (m *Service) ListTaskRuns(
	ctx context.Context,
	taskID string,
	query RunQuery,
	actor ActorContext,
) ([]Run, error) {
	if err := requireReadAuthority(actor); err != nil {
		return nil, err
	}

	trimmedID := strings.TrimSpace(taskID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: task id is required", ErrValidation)
	}

	if _, err := m.store.GetTask(ctx, trimmedID); err != nil {
		return nil, err
	}

	normalizedQuery := query
	normalizedQuery.TaskID = trimmedID
	return m.store.ListTaskRuns(ctx, normalizedQuery)
}

// ListTasks returns task summaries that satisfy the supplied query filters
// after enforcing read authority.
func (m *Service) ListTasks(ctx context.Context, query Query, actor ActorContext) ([]Summary, error) {
	if err := requireReadAuthority(actor); err != nil {
		return nil, err
	}
	return m.listTaskSummaries(ctx, query)
}

func (m *Service) listTaskSummaries(ctx context.Context, query Query) ([]Summary, error) {
	summaries, err := m.store.ListTasks(ctx, query)
	if err != nil {
		return nil, err
	}

	enriched := make([]Summary, 0, len(summaries))
	for _, summary := range summaries {
		item, err := m.enrichTaskSummary(ctx, summary)
		if err != nil {
			return nil, err
		}
		enriched = append(enriched, item)
	}
	return enriched, nil
}

func (m *Service) enrichTaskSummary(ctx context.Context, summary Summary) (Summary, error) {
	childCount, err := m.store.CountDirectChildren(ctx, summary.ID)
	if err != nil {
		return Summary{}, err
	}
	dependencies, err := m.store.ListDependencies(ctx, summary.ID)
	if err != nil {
		return Summary{}, err
	}
	runs, err := m.store.ListTaskRuns(ctx, RunQuery{TaskID: summary.ID})
	if err != nil {
		return Summary{}, err
	}
	events, err := m.store.ListTaskEvents(ctx, EventQuery{TaskID: summary.ID, Limit: 1})
	if err != nil {
		return Summary{}, err
	}
	return m.enrichTaskSummaryFromState(ctx, taskRecordFromSummary(summary), childCount, dependencies, runs, events)
}

func (m *Service) enrichTaskSummaryFromState(
	ctx context.Context,
	record Task,
	childCount int,
	dependencies []Dependency,
	runs []Run,
	events []Event,
) (Summary, error) {
	status, err := m.canonicalTaskStatus(ctx, record, dependencies, runs)
	if err != nil {
		return Summary{}, err
	}

	summary := summaryFromTaskRecord(record)
	summary.Status = status
	summary.Draft = status == TaskStatusDraft
	summary.ChildCount = childCount
	summary.DependencyCount = len(dependencies)
	summary.ActiveRun = activeRunSummary(runs, record.MaxAttempts)
	summary.LastActivityAt = latestTaskActivityAt(record, runs, events)
	summary.Dependencies, err = m.buildDependencyReferences(ctx, dependencies)
	if err != nil {
		return Summary{}, err
	}
	return summary, nil
}

func (m *Service) buildDependencyReferences(
	ctx context.Context,
	dependencies []Dependency,
) ([]DependencyReference, error) {
	refs := make([]DependencyReference, 0, len(dependencies))
	for _, dependency := range dependencies {
		dependsOn, err := m.taskReference(ctx, dependency.DependsOnTaskID)
		if err != nil {
			return nil, err
		}
		refs = append(refs, DependencyReference{
			TaskID:          dependency.TaskID,
			DependsOnTaskID: dependency.DependsOnTaskID,
			Kind:            dependency.Kind,
			CreatedAt:       dependency.CreatedAt,
			DependsOn:       dependsOn,
		})
	}
	return refs, nil
}

// AddDependency adds one dependency edge through the manager, reconciles the
// task status, and records the canonical audit event.
func (m *Service) AddDependency(ctx context.Context, spec AddDependency, actor ActorContext) error {
	if err := requireWriteAuthority(actor); err != nil {
		return err
	}

	normalizedSpec, err := normalizeAddDependencySpec(spec)
	if err != nil {
		return err
	}
	if err := m.store.CreateDependency(ctx, Dependency{
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
func (m *Service) RemoveDependency(
	ctx context.Context,
	taskID string,
	dependsOnID string,
	actor ActorContext,
) error {
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
func (m *Service) EnqueueRun(ctx context.Context, spec EnqueueRun, actor ActorContext) (*Run, error) {
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

	_, run, existing, err := m.store.ReserveQueuedRun(
		ctx,
		normalizedSpec.TaskID,
		m.newID("run"),
		normalizedSpec.IdempotencyKey,
		actor.Origin,
		normalizedSpec.NetworkChannel,
		normalizedSpec.Metadata,
		m.now().UTC(),
	)
	if err != nil {
		return nil, err
	}
	if existing {
		return &run, nil
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
	if err := m.dispatchTaskRunEnqueued(ctx, run, reconciledTask, actor, normalizedSpec.IdempotencyKey); err != nil {
		return nil, err
	}

	return &run, nil
}

func validateTaskForEnqueue(taskRecord Task) error {
	switch taskRecord.Status.Normalize() {
	case TaskStatusDraft:
		return fmt.Errorf("%w: task %q is draft", ErrInvalidStatusTransition, taskRecord.ID)
	case TaskStatusCanceled:
		return fmt.Errorf("%w: task %q is canceled", ErrInvalidStatusTransition, taskRecord.ID)
	default:
		return nil
	}
}

// ClaimRun transitions one queued run into the claimed state.
func (m *Service) ClaimRun(
	ctx context.Context,
	runID string,
	claim ClaimRun,
	actor ActorContext,
) (*Run, error) {
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
	if err := m.dispatchTaskRunPreClaim(ctx, run, taskRecord, actor); err != nil {
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
	if err := m.dispatchTaskRunPostClaim(ctx, run, reconciledTask, actor); err != nil {
		return nil, err
	}

	return &run, nil
}

// StartRun transitions one claimed or starting run into active execution.
func (m *Service) StartRun(ctx context.Context, runID string, req StartRun, actor ActorContext) (*Run, error) {
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
		var failedRun *Run
		run, failedRun, err = m.transitionClaimedRunToStarting(ctx, taskRecord, run, actor)
		if err != nil {
			if failedRun != nil {
				return failedRun, err
			}
			return nil, err
		}
	case TaskRunStatusStarting:
		if err := validateRunningSessionBinding(run); err != nil {
			return nil, err
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
func (m *Service) AttachRunSession(
	ctx context.Context,
	runID string,
	sessionID string,
	actor ActorContext,
) (*Run, error) {
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
func (m *Service) CompleteRun(
	ctx context.Context,
	runID string,
	result RunResult,
	actor ActorContext,
) (*Run, error) {
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
func (m *Service) FailRun(
	ctx context.Context,
	runID string,
	failure RunFailure,
	actor ActorContext,
) (*Run, error) {
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
func (m *Service) CancelRun(
	ctx context.Context,
	runID string,
	req CancelRun,
	actor ActorContext,
) (*Run, error) {
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
func (m *Service) RecoverRunOnBoot(
	ctx context.Context,
	runID string,
	recovery RunBootRecovery,
	actor ActorContext,
) (*Run, error) {
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
		return m.recoverRunByRequeue(
			ctx,
			taskRecord,
			run,
			normalizedRecovery,
			actor,
			previousStatus,
			previousSessionID,
		)
	case RunBootRecoveryMarkRunning:
		return m.recoverRunByMarkRunning(
			ctx,
			taskRecord,
			run,
			normalizedRecovery,
			actor,
			previousStatus,
			previousSessionID,
		)
	case RunBootRecoveryFail:
		return m.recoverRunByFailure(
			ctx,
			taskRecord,
			run,
			normalizedRecovery,
			actor,
			previousStatus,
			previousSessionID,
		)
	default:
		return nil, fmt.Errorf(
			"%w: run boot recovery action %q is not supported",
			ErrValidation,
			normalizedRecovery.Action,
		)
	}
}

func (m *Service) dispatchTaskRunEnqueued(
	ctx context.Context,
	run Run,
	taskRecord Task,
	actor ActorContext,
	idempotencyKey string,
) error {
	payload := hookspkg.TaskRunEnqueuedPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskRunEnqueued,
			Timestamp: m.now().UTC(),
		},
		TaskRunContext: m.taskRunHookContext(run, taskRecord, actor),
		IdempotencyKey: strings.TrimSpace(idempotencyKey),
	}
	_, err := m.taskHooks.DispatchTaskRunEnqueued(ctx, payload)
	return err
}

func (m *Service) dispatchTaskRunPreClaim(
	ctx context.Context,
	run Run,
	taskRecord Task,
	actor ActorContext,
) error {
	contextPayload := m.taskRunHookContext(run, taskRecord, actor)
	payload := hookspkg.TaskRunPreClaimPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskRunPreClaim,
			Timestamp: m.now().UTC(),
		},
		TaskRunContext: contextPayload,
		Criteria: hookspkg.TaskRunClaimCriteria{
			WorkspaceID:           contextPayload.WorkspaceID,
			ClaimerSessionID:      taskRunHookClaimerSessionID(run, actor),
			AgentName:             contextPayload.AgentName,
			RequiredCapabilities:  taskRunMetadataStringList(run.Metadata, "required_capabilities"),
			PriorityMin:           taskPriorityMin(taskRecord.Priority),
			CoordinationChannelID: contextPayload.CoordinationChannelID,
		},
	}
	result, err := m.taskHooks.DispatchTaskRunPreClaim(ctx, payload)
	if err != nil {
		return err
	}
	if result.Denied {
		reason := strings.TrimSpace(result.DenyReason)
		if reason == "" {
			reason = "task run claim denied by hook"
		}
		return fmt.Errorf("%w: %s", ErrPermissionDenied, reason)
	}
	return nil
}

func (m *Service) dispatchTaskRunPostClaim(
	ctx context.Context,
	run Run,
	taskRecord Task,
	actor ActorContext,
) error {
	payload := hookspkg.TaskRunPostClaimPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskRunPostClaim,
			Timestamp: m.now().UTC(),
		},
		TaskRunContext: m.taskRunHookContext(run, taskRecord, actor),
		ClaimedAt:      run.ClaimedAt,
	}
	_, err := m.taskHooks.DispatchTaskRunPostClaim(ctx, payload)
	return err
}

func (m *Service) dispatchTaskRunLeaseRecovered(
	ctx context.Context,
	run Run,
	taskRecord Task,
	actor ActorContext,
	previousStatus RunStatus,
	previousSessionID string,
	recovery RunBootRecovery,
) error {
	payload := hookspkg.TaskRunLeaseRecoveredPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskRunLeaseRecovered,
			Timestamp: m.now().UTC(),
		},
		TaskRunContext:    m.taskRunHookContext(run, taskRecord, actor),
		PreviousRunStatus: string(previousStatus.Normalize()),
		PreviousSessionID: strings.TrimSpace(previousSessionID),
		RecoveryAction:    string(recovery.Action.Normalize()),
		RecoveryReason:    strings.TrimSpace(recovery.Reason),
	}
	_, err := m.taskHooks.DispatchTaskRunLeaseRecovered(ctx, payload)
	return err
}

func (m *Service) taskRunHookContext(run Run, taskRecord Task, actor ActorContext) hookspkg.TaskRunContext {
	coordinationChannelID := taskRunCoordinationChannelID(run)
	return hookspkg.TaskRunContext{
		TaskID:                strings.TrimSpace(run.TaskID),
		RunID:                 strings.TrimSpace(run.ID),
		WorkspaceID:           strings.TrimSpace(taskRecord.WorkspaceID),
		WorkflowID:            taskRunMetadataString(run.Metadata, "workflow_id"),
		CoordinationChannelID: coordinationChannelID,
		NetworkChannel:        strings.TrimSpace(run.NetworkChannel),
		AgentName:             taskRunHookAgentName(run, actor),
		SessionID:             strings.TrimSpace(run.SessionID),
		ActorKind:             string(actor.Actor.Kind.Normalize()),
		ActorRef:              strings.TrimSpace(actor.Actor.Ref),
		TaskStatus:            string(taskRecord.Status.Normalize()),
		RunStatus:             string(run.Status.Normalize()),
		Attempt:               run.Attempt,
		Error:                 strings.TrimSpace(run.Error),
	}
}

func taskRunCoordinationChannelID(run Run) string {
	if value := taskRunMetadataString(run.Metadata, "coordination_channel_id"); value != "" {
		return value
	}
	return strings.TrimSpace(run.NetworkChannel)
}

func taskRunHookAgentName(run Run, actor ActorContext) string {
	if value := taskRunMetadataString(run.Metadata, "agent_name"); value != "" {
		return value
	}
	if actor.Actor.Kind.Normalize() == ActorKindAgentSession {
		return strings.TrimSpace(actor.Actor.Ref)
	}
	return ""
}

func taskRunHookClaimerSessionID(run Run, actor ActorContext) string {
	if strings.TrimSpace(run.SessionID) != "" {
		return strings.TrimSpace(run.SessionID)
	}
	if actor.Actor.Kind.Normalize() == ActorKindAgentSession {
		return strings.TrimSpace(actor.Actor.Ref)
	}
	return ""
}

func taskPriorityMin(priority Priority) int {
	switch priority.Normalize() {
	case PriorityLow:
		return 10
	case PriorityHigh:
		return 30
	case PriorityUrgent:
		return 40
	default:
		return 20
	}
}

func taskRunMetadataString(raw json.RawMessage, key string) string {
	var data map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &data) != nil {
		return ""
	}
	value, ok := data[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func taskRunMetadataStringList(raw json.RawMessage, key string) []string {
	var data map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &data) != nil {
		return nil
	}
	value, ok := data[key]
	if !ok {
		return nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			continue
		}
		if trimmed := strings.TrimSpace(text); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
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
	normalized.Priority = normalizePriorityOrDefault(normalized.Priority)
	if normalized.MaxAttempts != nil {
		maxAttempts := *normalized.MaxAttempts
		normalized.MaxAttempts = &maxAttempts
	}
	normalized.ApprovalPolicy = normalizeApprovalPolicyOrDefault(normalized.ApprovalPolicy)
	if normalized.Owner != nil {
		normalized.Owner = normalizeOwnership(normalized.Owner)
	}
	normalized.Metadata = normalizeRawJSON(normalized.Metadata)
	if err := normalized.Validate("create_task"); err != nil {
		return CreateTask{}, err
	}
	return normalized, nil
}

func normalizeTaskPatch(patch Patch) (Patch, error) {
	normalized := patch
	if normalized.Title != nil {
		title := strings.TrimSpace(*normalized.Title)
		normalized.Title = &title
	}
	if normalized.Description != nil {
		description := strings.TrimSpace(*normalized.Description)
		normalized.Description = &description
	}
	if normalized.Priority != nil {
		priority := normalized.Priority.Normalize()
		normalized.Priority = &priority
	}
	if normalized.MaxAttempts != nil {
		maxAttempts := *normalized.MaxAttempts
		normalized.MaxAttempts = &maxAttempts
	}
	if normalized.ApprovalPolicy != nil {
		approvalPolicy := normalized.ApprovalPolicy.Normalize()
		normalized.ApprovalPolicy = &approvalPolicy
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
		return Patch{}, err
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
	normalized.Metadata = normalizeRawJSON(normalized.Metadata)
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
	normalized.Classification = strings.TrimSpace(normalized.Classification)
	normalized.Detail = strings.TrimSpace(normalized.Detail)
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

func (m *Service) validateParentConstraints(ctx context.Context, spec CreateTask) error {
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

func (m *Service) taskDepth(ctx context.Context, record Task) (int, error) {
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

func (m *Service) reconcileTaskWithStore(
	ctx context.Context,
	store DeleteTaskMutationStore,
	taskID string,
) (Task, error) {
	record, err := store.GetTask(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	dependencies, err := store.ListDependencies(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	runs, err := store.ListTaskRuns(ctx, RunQuery{TaskID: taskID})
	if err != nil {
		return Task{}, err
	}

	canonicalStatus, err := m.canonicalTaskStatusWithStore(ctx, store, record, dependencies, runs)
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
	if err := store.UpdateTask(ctx, record); err != nil {
		return Task{}, err
	}
	return record, nil
}

func (m *Service) reconcileTaskCascade(ctx context.Context, taskID string) (Task, error) {
	return m.reconcileTaskCascadeWithStore(ctx, m.store, taskID)
}

func (m *Service) reconcileTaskCascadeWithStore(
	ctx context.Context,
	store DeleteTaskMutationStore,
	taskID string,
) (Task, error) {
	previous, err := store.GetTask(ctx, taskID)
	if err != nil {
		return Task{}, err
	}

	reconciled, err := m.reconcileTaskWithStore(ctx, store, taskID)
	if err != nil {
		return Task{}, err
	}
	if previous.Status.Normalize() != reconciled.Status.Normalize() {
		if err := m.reconcileDependentTasksWithStore(ctx, store, taskID, map[string]struct{}{taskID: {}}); err != nil {
			return Task{}, err
		}
	}
	return reconciled, nil
}

func (m *Service) canonicalTaskStatus(
	ctx context.Context,
	record Task,
	dependencies []Dependency,
	runs []Run,
) (Status, error) {
	return m.canonicalTaskStatusWithStore(ctx, m.store, record, dependencies, runs)
}

func (m *Service) canonicalTaskStatusWithStore(
	ctx context.Context,
	store DeleteTaskMutationStore,
	record Task,
	dependencies []Dependency,
	runs []Run,
) (Status, error) {
	return m.canonicalTaskStatusReadOnlyWithStore(
		ctx,
		store,
		record,
		dependencies,
		runs,
		make(map[string]struct{}, len(dependencies)+1),
	)
}

func (m *Service) canonicalTaskStatusReadOnlyWithStore(
	ctx context.Context,
	store DeleteTaskMutationStore,
	record Task,
	dependencies []Dependency,
	runs []Run,
	visited map[string]struct{},
) (Status, error) {
	taskID := strings.TrimSpace(record.ID)
	if taskID != "" {
		if _, seen := visited[taskID]; seen {
			// Defensive termination guard: dependency cycles are invalid, but read
			// paths should still terminate without mutating persisted state.
			return taskStatusFromPolicySnapshot(
				record.Status,
				true,
				taskApprovalBlocked(record),
				taskAttemptsExhausted(record, runs),
				runs,
			), nil
		}
		visited[taskID] = struct{}{}
		defer delete(visited, taskID)
	}

	unresolvedDependencies, err := m.hasUnresolvedDependenciesReadOnlyWithStore(ctx, store, dependencies, visited)
	if err != nil {
		return "", err
	}
	return taskStatusFromPolicySnapshot(
		record.Status,
		unresolvedDependencies,
		taskApprovalBlocked(record),
		taskAttemptsExhausted(record, runs),
		runs,
	), nil
}

func hasOpenRun(runs []Run) bool {
	for _, run := range runs {
		if !isTerminalRunStatus(run.Status) {
			return true
		}
	}
	return false
}

func isTerminalTaskStatus(status Status) bool {
	switch status.Normalize() {
	case TaskStatusCompleted, TaskStatusFailed, TaskStatusCanceled:
		return true
	default:
		return false
	}
}

func isTerminalRunStatus(status RunStatus) bool {
	switch status.Normalize() {
	case TaskRunStatusCompleted, TaskRunStatusFailed, TaskRunStatusCanceled:
		return true
	default:
		return false
	}
}

func taskStatusFromSnapshot(currentStatus Status, unresolvedDependencies bool, runs []Run) Status {
	return taskStatusFromPolicySnapshot(currentStatus, unresolvedDependencies, false, false, runs)
}

func taskStatusFromPolicySnapshot(
	currentStatus Status,
	unresolvedDependencies bool,
	approvalBlocked bool,
	attemptsExhausted bool,
	runs []Run,
) Status {
	status := currentStatus.Normalize()
	if status == TaskStatusCanceled || status == TaskStatusDraft {
		return status
	}

	runnableBlocked := unresolvedDependencies || approvalBlocked
	hasQueuedOrClaimed := false
	var latestTerminal Run
	hasLatestTerminal := false
	for idx := range runs {
		run := runs[idx]
		switch run.Status.Normalize() {
		case TaskRunStatusStarting, TaskRunStatusRunning:
			return TaskStatusInProgress
		case TaskRunStatusQueued, TaskRunStatusClaimed:
			hasQueuedOrClaimed = true
		case TaskRunStatusCompleted, TaskRunStatusFailed, TaskRunStatusCanceled:
			if !hasLatestTerminal || runComesAfter(run, latestTerminal) {
				latestTerminal = run
				hasLatestTerminal = true
			}
		}
	}

	if hasQueuedOrClaimed {
		if runnableBlocked {
			return TaskStatusBlocked
		}
		return TaskStatusReady
	}

	if hasLatestTerminal {
		switch latestTerminal.Status.Normalize() {
		case TaskRunStatusCompleted:
			return TaskStatusCompleted
		case TaskRunStatusFailed:
			if attemptsExhausted {
				return TaskStatusFailed
			}
			if runnableBlocked {
				return TaskStatusBlocked
			}
			return TaskStatusReady
		case TaskRunStatusCanceled:
			return TaskStatusCanceled
		}
	}

	if isTerminalTaskStatus(status) {
		return status
	}
	if runnableBlocked {
		return TaskStatusBlocked
	}
	return TaskStatusReady
}

func taskApprovalBlocked(record Task) bool {
	return approvalStateBlocksExecution(record.ApprovalPolicy, record.ApprovalState)
}

func approvalStateBlocksExecution(policy ApprovalPolicy, state ApprovalState) bool {
	normalizedPolicy := normalizeApprovalPolicyOrDefault(policy)
	if normalizedPolicy != ApprovalPolicyManual {
		return false
	}
	return normalizeApprovalStateOrDefault(normalizedPolicy, state) != ApprovalStateApproved
}

func taskAttemptsExhausted(record Task, runs []Run) bool {
	return nextRunAttempt(runs) > normalizeTaskMaxAttemptsOrDefault(record.MaxAttempts)
}

func runComesAfter(left Run, right Run) bool {
	switch {
	case left.Attempt != right.Attempt:
		return left.Attempt > right.Attempt
	case !left.QueuedAt.Equal(right.QueuedAt):
		return left.QueuedAt.After(right.QueuedAt)
	default:
		return left.ID > right.ID
	}
}

func (m *Service) ensureTaskDeleteAllowedWithStore(
	ctx context.Context,
	store DeleteTaskMutationStore,
	record Task,
) error {
	childCount, err := store.CountDirectChildren(ctx, record.ID)
	if err != nil {
		return fmt.Errorf("task: count child tasks for delete %q: %w", record.ID, err)
	}
	if childCount > 0 {
		return fmt.Errorf(
			"%w: task %q has %d child tasks; delete children first",
			ErrValidation,
			record.ID,
			childCount,
		)
	}

	runs, err := store.ListTaskRuns(ctx, RunQuery{TaskID: record.ID})
	if err != nil {
		return fmt.Errorf("task: list runs for delete %q: %w", record.ID, err)
	}
	if hasOpenRun(runs) {
		return fmt.Errorf(
			"%w: task %q has active or queued runs; cancel or finish them first",
			ErrValidation,
			record.ID,
		)
	}

	return nil
}

func uniqueDependentTaskIDs(dependents []Dependency) []string {
	if len(dependents) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(dependents))
	ids := make([]string, 0, len(dependents))
	for _, dependent := range dependents {
		taskID := strings.TrimSpace(dependent.TaskID)
		if taskID == "" {
			continue
		}
		if _, ok := seen[taskID]; ok {
			continue
		}
		seen[taskID] = struct{}{}
		ids = append(ids, taskID)
	}

	sort.Strings(ids)
	return ids
}

func (m *Service) hasUnresolvedDependenciesReadOnlyWithStore(
	ctx context.Context,
	store DeleteTaskMutationStore,
	dependencies []Dependency,
	visited map[string]struct{},
) (bool, error) {
	for _, dependency := range dependencies {
		dependencyTaskID := strings.TrimSpace(dependency.DependsOnTaskID)
		record, err := store.GetTask(ctx, dependencyTaskID)
		if err != nil {
			return false, err
		}
		nestedDependencies, err := store.ListDependencies(ctx, dependencyTaskID)
		if err != nil {
			return false, err
		}
		nestedRuns, err := store.ListTaskRuns(ctx, RunQuery{TaskID: dependencyTaskID})
		if err != nil {
			return false, err
		}
		status, err := m.canonicalTaskStatusReadOnlyWithStore(
			ctx,
			store,
			record,
			nestedDependencies,
			nestedRuns,
			visited,
		)
		if err != nil {
			return false, err
		}
		if status.Normalize() != TaskStatusCompleted {
			return true, nil
		}
	}
	return false, nil
}

func (m *Service) reconcileDependentTasks(ctx context.Context, taskID string, visited map[string]struct{}) error {
	return m.reconcileDependentTasksWithStore(ctx, m.store, taskID, visited)
}

func (m *Service) reconcileDependentTasksWithStore(
	ctx context.Context,
	store DeleteTaskMutationStore,
	taskID string,
	visited map[string]struct{},
) error {
	dependents, err := store.ListDependents(ctx, taskID)
	if err != nil {
		return err
	}

	for _, dependent := range dependents {
		dependentTaskID := strings.TrimSpace(dependent.TaskID)
		if _, seen := visited[dependentTaskID]; seen {
			continue
		}
		visited[dependentTaskID] = struct{}{}

		previous, err := store.GetTask(ctx, dependentTaskID)
		if err != nil {
			return err
		}
		reconciled, err := m.reconcileTaskWithStore(ctx, store, dependentTaskID)
		if err != nil {
			return err
		}
		if previous.Status.Normalize() != reconciled.Status.Normalize() {
			if err := m.reconcileDependentTasksWithStore(ctx, store, dependentTaskID, visited); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Service) loadRunWithTask(ctx context.Context, runID string) (Run, Task, error) {
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return Run{}, Task{}, fmt.Errorf("%w: task run id is required", ErrValidation)
	}

	run, err := m.store.GetTaskRun(ctx, trimmedRunID)
	if err != nil {
		return Run{}, Task{}, err
	}
	taskRecord, err := m.store.GetTask(ctx, run.TaskID)
	if err != nil {
		return Run{}, Task{}, err
	}
	return run, taskRecord, nil
}

func (m *Service) ensureTaskExecutable(ctx context.Context, record Task) error {
	dependencies, err := m.store.ListDependencies(ctx, record.ID)
	if err != nil {
		return err
	}
	runs, err := m.store.ListTaskRuns(ctx, RunQuery{TaskID: record.ID})
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
	case TaskStatusDraft:
		return fmt.Errorf("%w: task %q is draft", ErrInvalidStatusTransition, record.ID)
	case TaskStatusCanceled:
		return fmt.Errorf("%w: task %q is canceled", ErrInvalidStatusTransition, record.ID)
	default:
		return nil
	}
}

func (m *Service) requireSessionExecutor(action string) error {
	if m.sessions == nil {
		return fmt.Errorf("%w: session executor is required to %s", ErrValidation, action)
	}
	return nil
}

func (m *Service) collectTaskTree(ctx context.Context, rootTaskID string) ([]Task, error) {
	root, err := m.store.GetTask(ctx, rootTaskID)
	if err != nil {
		return nil, err
	}

	tree := []Task{root}
	queue := []Task{root}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		children, err := m.store.ListTasks(ctx, Query{ParentTaskID: current.ID})
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

func (m *Service) failRunRecord(
	ctx context.Context,
	taskRecord Task,
	run Run,
	failure RunFailure,
	actor ActorContext,
) (*Run, error) {
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

func (m *Service) cancelRunRecord(
	ctx context.Context,
	taskRecord Task,
	run Run,
	req CancelRun,
	actor ActorContext,
	opts cancelRunOptions,
) (*Run, error) {
	status := run.Status.Normalize()
	switch status {
	case TaskRunStatusQueued, TaskRunStatusClaimed, TaskRunStatusStarting, TaskRunStatusRunning:
	default:
		return nil, requireRunTransition(run, TaskRunStatusCanceled)
	}

	sessionID := strings.TrimSpace(run.SessionID)
	activeSession := (status == TaskRunStatusStarting || status == TaskRunStatusRunning) && sessionID != ""
	if activeSession {
		if err := m.requireSessionExecutor("cancel active run"); err != nil {
			return nil, err
		}
	}

	run.Status = TaskRunStatusCanceled
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
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunCanceled, actor, cancelledRunPayload{
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

func (m *Service) waitAndForceStopRun(ctx context.Context, sessionID string) error {
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

func requireRunTransition(run Run, next RunStatus) error {
	current := run.Status.Normalize()
	target := next.Normalize()
	if allowsRunTransition(current, target) {
		return nil
	}
	return fmt.Errorf(
		"%w: task run %q cannot transition from %q to %q",
		ErrInvalidStatusTransition,
		run.ID,
		current,
		target,
	)
}

func allowsRunTransition(current RunStatus, next RunStatus) bool {
	switch current.Normalize() {
	case TaskRunStatusQueued:
		return next.Normalize() == TaskRunStatusClaimed || next.Normalize() == TaskRunStatusCanceled
	case TaskRunStatusClaimed:
		switch next.Normalize() {
		case TaskRunStatusStarting, TaskRunStatusCanceled:
			return true
		}
	case TaskRunStatusStarting:
		switch next.Normalize() {
		case TaskRunStatusRunning, TaskRunStatusFailed, TaskRunStatusCanceled:
			return true
		}
	case TaskRunStatusRunning:
		switch next.Normalize() {
		case TaskRunStatusCompleted, TaskRunStatusFailed, TaskRunStatusCanceled:
			return true
		}
	}
	return false
}

func nextRunAttempt(runs []Run) int {
	maxAttempt := 0
	for _, run := range runs {
		if run.Attempt > maxAttempt {
			maxAttempt = run.Attempt
		}
	}
	return maxAttempt + 1
}

func (m *Service) validateNetworkChannel(path string, channel string) error {
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

func (m *Service) validateRunChannelUsable(
	ctx context.Context,
	taskRecord Task,
	run Run,
	actor ActorContext,
	operation string,
) error {
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

func runBootRecoveryError(run Run, recovery RunBootRecovery) string {
	sessionID := strings.TrimSpace(run.SessionID)
	switch {
	case sessionID != "" && recovery.Classification != "" && recovery.Detail != "":
		return fmt.Sprintf(
			"orphaned on boot: session %q classified as %s (%s)",
			sessionID,
			recovery.Classification,
			recovery.Detail,
		)
	case sessionID != "" && recovery.Classification != "":
		return fmt.Sprintf(
			"orphaned on boot: session %q classified as %s",
			sessionID,
			recovery.Classification,
		)
	case sessionID != "" && recovery.SessionState != "":
		return fmt.Sprintf("orphaned on boot: session %q is %s", sessionID, recovery.SessionState)
	case sessionID != "":
		return fmt.Sprintf("orphaned on boot: session %q is not live", sessionID)
	default:
		return "orphaned on boot: run has no live session"
	}
}

func runBootRecoveryMetadata(run Run, recovery RunBootRecovery) json.RawMessage {
	payload, err := marshalTaskEventPayload(map[string]string{
		"reason":          normalizedBootRecoveryReason(recovery.Reason),
		"previous_status": string(run.Status.Normalize()),
		"session_id":      strings.TrimSpace(run.SessionID),
		"session_state":   strings.TrimSpace(recovery.SessionState),
		"classification":  strings.TrimSpace(recovery.Classification),
		"detail":          strings.TrimSpace(recovery.Detail),
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

func (m *Service) recordTaskEvent(
	ctx context.Context,
	taskID string,
	runID string,
	eventType string,
	actor ActorContext,
	payload any,
) error {
	rawPayload, err := marshalTaskEventPayload(payload)
	if err != nil {
		return err
	}
	event := Event{
		ID:        m.newID("evt"),
		TaskID:    strings.TrimSpace(taskID),
		RunID:     strings.TrimSpace(runID),
		EventType: strings.TrimSpace(eventType),
		Actor:     actor.Actor,
		Origin:    actor.Origin,
		Payload:   rawPayload,
		Timestamp: m.now().UTC(),
	}
	if err := m.store.CreateTaskEvent(ctx, event); err != nil {
		return err
	}

	postCommitCtx := context.Background()
	if ctx != nil {
		postCommitCtx = context.WithoutCancel(ctx)
	}

	record, err := m.store.GetTaskEventRecord(postCommitCtx, event.ID)
	if err != nil {
		m.emitTaskLiveEventBestEffort(postCommitCtx, event.ID)
		return nil
	}
	m.notifyTaskObserverBestEffort(postCommitCtx, record)
	m.emitTaskLiveRecordBestEffort(postCommitCtx, record)
	return nil
}

func (m *Service) notifyTaskObserverBestEffort(ctx context.Context, record EventRecord) {
	if m == nil || m.eventObserver == nil {
		return
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error(
				"task: task event observer panicked during post-commit notification",
				"panic", recovered,
				"event_id", record.Event.ID,
				"task_id", record.Event.TaskID,
				"run_id", record.Event.RunID,
				"event_type", record.Event.EventType,
			)
		}
	}()

	m.eventObserver.OnTaskEvent(ctx, record)
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

func summaryFromTaskRecord(record Task) Summary {
	return Summary{
		ID:             record.ID,
		Identifier:     record.Identifier,
		Scope:          record.Scope,
		WorkspaceID:    record.WorkspaceID,
		ParentTaskID:   record.ParentTaskID,
		NetworkChannel: record.NetworkChannel,
		Title:          record.Title,
		Priority:       record.Priority,
		MaxAttempts:    record.MaxAttempts,
		Status:         record.Status,
		ApprovalPolicy: record.ApprovalPolicy,
		ApprovalState:  record.ApprovalState,
		Draft:          record.Status.Normalize() == TaskStatusDraft,
		Owner:          cloneOwnership(record.Owner),
		CreatedBy:      record.CreatedBy,
		Origin:         record.Origin,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
		ClosedAt:       record.ClosedAt,
		LastActivityAt: record.UpdatedAt,
	}
}

func taskRecordFromSummary(summary Summary) Task {
	return Task{
		ID:             summary.ID,
		Identifier:     summary.Identifier,
		Scope:          summary.Scope,
		WorkspaceID:    summary.WorkspaceID,
		ParentTaskID:   summary.ParentTaskID,
		NetworkChannel: summary.NetworkChannel,
		Title:          summary.Title,
		Priority:       summary.Priority,
		MaxAttempts:    summary.MaxAttempts,
		Status:         summary.Status,
		ApprovalPolicy: summary.ApprovalPolicy,
		ApprovalState:  summary.ApprovalState,
		Owner:          cloneOwnership(summary.Owner),
		CreatedBy:      summary.CreatedBy,
		Origin:         summary.Origin,
		CreatedAt:      summary.CreatedAt,
		UpdatedAt:      summary.UpdatedAt,
		ClosedAt:       summary.ClosedAt,
	}
}

func (m *Service) taskReference(ctx context.Context, taskID string) (Reference, error) {
	record, err := m.store.GetTask(ctx, taskID)
	if err != nil {
		return Reference{}, err
	}
	dependencies, err := m.store.ListDependencies(ctx, record.ID)
	if err != nil {
		return Reference{}, err
	}
	runs, err := m.store.ListTaskRuns(ctx, RunQuery{TaskID: record.ID})
	if err != nil {
		return Reference{}, err
	}
	status, err := m.canonicalTaskStatus(ctx, record, dependencies, runs)
	if err != nil {
		return Reference{}, err
	}
	return taskReferenceFromTask(record, status), nil
}

func taskReferenceFromTask(record Task, status Status) Reference {
	return Reference{
		ID:          record.ID,
		Identifier:  record.Identifier,
		Title:       record.Title,
		Status:      status,
		Priority:    record.Priority,
		Owner:       cloneOwnership(record.Owner),
		Scope:       record.Scope,
		WorkspaceID: record.WorkspaceID,
	}
}

func activeRunSummary(runs []Run, maxAttempts int) *RunSummary {
	var current *Run
	for idx := range runs {
		run := runs[idx]
		if isTerminalRunStatus(run.Status) {
			continue
		}
		if current == nil || prefersActiveRun(run, *current) {
			candidate := run
			current = &candidate
		}
	}
	if current == nil {
		return nil
	}
	return &RunSummary{
		ID:          current.ID,
		TaskID:      current.TaskID,
		Status:      current.Status,
		Attempt:     current.Attempt,
		MaxAttempts: maxAttempts,
		SessionID:   current.SessionID,
		ClaimedBy:   cloneActorIdentity(current.ClaimedBy),
		QueuedAt:    current.QueuedAt,
		ClaimedAt:   current.ClaimedAt,
		StartedAt:   current.StartedAt,
		EndedAt:     current.EndedAt,
		Error:       current.Error,
	}
}

func prefersActiveRun(candidate Run, current Run) bool {
	candidateRank := activeRunRank(candidate.Status)
	currentRank := activeRunRank(current.Status)
	if candidateRank != currentRank {
		return candidateRank > currentRank
	}

	candidateAt := latestRunActivityAt(candidate)
	currentAt := latestRunActivityAt(current)
	if !candidateAt.Equal(currentAt) {
		return candidateAt.After(currentAt)
	}
	return candidate.ID > current.ID
}

func activeRunRank(status RunStatus) int {
	switch status.Normalize() {
	case TaskRunStatusRunning:
		return 4
	case TaskRunStatusStarting:
		return 3
	case TaskRunStatusClaimed:
		return 2
	case TaskRunStatusQueued:
		return 1
	default:
		return 0
	}
}

func latestTaskActivityAt(record Task, runs []Run, events []Event) time.Time {
	latest := record.UpdatedAt
	if latest.IsZero() || (!record.CreatedAt.IsZero() && record.CreatedAt.After(latest)) {
		latest = record.CreatedAt
	}
	for _, run := range runs {
		runAt := latestRunActivityAt(run)
		if runAt.After(latest) {
			latest = runAt
		}
	}
	for _, event := range events {
		if event.Timestamp.After(latest) {
			latest = event.Timestamp
		}
	}
	return latest
}

func latestRunActivityAt(run Run) time.Time {
	latest := run.QueuedAt
	for _, candidate := range []time.Time{run.ClaimedAt, run.StartedAt, run.EndedAt} {
		if candidate.After(latest) {
			latest = candidate
		}
	}
	return latest
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

func cloneActorIdentity(actor *ActorIdentity) *ActorIdentity {
	if actor == nil {
		return nil
	}
	cloned := *actor
	return &cloned
}

func sameOwnership(left *Ownership, right *Ownership) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return left.Kind.Normalize() == right.Kind.Normalize() &&
			strings.TrimSpace(left.Ref) == strings.TrimSpace(right.Ref)
	}
}

func normalizeRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
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
	return bytes.Equal(normalizeRawJSON(left), normalizeRawJSON(right))
}

type createdTaskPayload struct {
	Scope          Scope      `json:"scope"`
	WorkspaceID    string     `json:"workspace_id,omitempty"`
	ParentTaskID   string     `json:"parent_task_id,omitempty"`
	Status         Status     `json:"status"`
	NetworkChannel string     `json:"network_channel,omitempty"`
	Owner          *Ownership `json:"owner,omitempty"`
}

type updatedTaskPayload struct {
	ChangedFields []string `json:"changed_fields"`
	Status        Status   `json:"status"`
}

type publishedTaskPayload struct {
	PreviousStatus Status        `json:"previous_status"`
	Status         Status        `json:"status"`
	ApprovalState  ApprovalState `json:"approval_state"`
}

type approvalDecisionTaskPayload struct {
	PreviousApprovalState ApprovalState `json:"previous_approval_state"`
	ApprovalState         ApprovalState `json:"approval_state"`
	Status                Status        `json:"status"`
}

type childCreatedTaskPayload struct {
	ChildTaskID      string `json:"child_task_id"`
	ChildScope       Scope  `json:"child_scope"`
	ChildWorkspaceID string `json:"child_workspace_id,omitempty"`
}

type dependencyTaskPayload struct {
	DependsOnTaskID string         `json:"depends_on_task_id"`
	Kind            DependencyKind `json:"kind"`
	Status          Status         `json:"status"`
}

type cancelledTaskPayload struct {
	Reason               string          `json:"reason,omitempty"`
	Metadata             json.RawMessage `json:"metadata,omitempty"`
	Status               Status          `json:"status"`
	PropagatedFromTaskID string          `json:"propagated_from_task_id,omitempty"`
	CancelledRunIDs      []string        `json:"canceled_run_ids,omitempty"`
}

type runEnqueuedPayload struct {
	Attempt        int       `json:"attempt"`
	Status         RunStatus `json:"status"`
	TaskStatus     Status    `json:"task_status"`
	NetworkChannel string    `json:"network_channel,omitempty"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
}

type runClaimedPayload struct {
	Status     RunStatus     `json:"status"`
	TaskStatus Status        `json:"task_status"`
	ClaimedBy  ActorIdentity `json:"claimed_by"`
}

type runTransitionPayload struct {
	Status     RunStatus `json:"status"`
	TaskStatus Status    `json:"task_status"`
	SessionID  string    `json:"session_id,omitempty"`
}

type completedRunPayload struct {
	Status     RunStatus       `json:"status"`
	TaskStatus Status          `json:"task_status"`
	Result     json.RawMessage `json:"result,omitempty"`
}

type failedRunPayload struct {
	Status     RunStatus       `json:"status"`
	TaskStatus Status          `json:"task_status"`
	Error      string          `json:"error"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

type cancelledRunPayload struct {
	Status                   RunStatus       `json:"status"`
	TaskStatus               Status          `json:"task_status,omitempty"`
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
	PreviousStatus RunStatus             `json:"previous_status"`
	Status         RunStatus             `json:"status"`
	TaskStatus     Status                `json:"task_status"`
	Reason         string                `json:"reason,omitempty"`
	SessionID      string                `json:"session_id,omitempty"`
	SessionState   string                `json:"session_state,omitempty"`
	Classification string                `json:"classification,omitempty"`
	Detail         string                `json:"detail,omitempty"`
}
