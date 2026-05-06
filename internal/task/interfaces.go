package task

import (
	"context"
	"encoding/json"
	"time"
)

// Manager is the task-domain authority for task and run lifecycle operations.
type Manager interface {
	CreateTask(ctx context.Context, spec CreateTask, actor ActorContext) (*Task, error)
	CreateChildTask(ctx context.Context, parentTaskID string, spec CreateTask, actor ActorContext) (*Task, error)
	DeleteTask(ctx context.Context, id string, actor ActorContext) error
	UpdateTask(ctx context.Context, id string, patch Patch, actor ActorContext) (*Task, error)
	PublishTask(ctx context.Context, id string, req ExecutionRequest, actor ActorContext) (*Execution, error)
	StartTask(ctx context.Context, id string, req ExecutionRequest, actor ActorContext) (*Execution, error)
	ApproveTask(ctx context.Context, id string, req ExecutionRequest, actor ActorContext) (*Execution, error)
	RejectTask(ctx context.Context, id string, actor ActorContext) (*Task, error)
	CancelTask(ctx context.Context, id string, req CancelTask, actor ActorContext) (*Task, error)
	MarkTaskRead(ctx context.Context, id string, actor ActorContext) (TriageState, error)
	ArchiveTask(ctx context.Context, id string, actor ActorContext) (TriageState, error)
	DismissTask(ctx context.Context, id string, actor ActorContext) (TriageState, error)

	GetExecutionProfile(ctx context.Context, taskID string, actor ActorContext) (ExecutionProfile, error)
	SetExecutionProfile(
		ctx context.Context,
		taskID string,
		profile *ExecutionProfile,
		actor ActorContext,
	) (ExecutionProfile, error)
	DeleteExecutionProfile(ctx context.Context, taskID string, actor ActorContext) error

	RequestRunReview(ctx context.Context, req RunReviewRequest, actor ActorContext) (RunReview, bool, error)
	GetRunReview(ctx context.Context, reviewID string, actor ActorContext) (RunReview, error)
	RecordRunReview(ctx context.Context, req RecordRunReviewRequest, actor ActorContext) (RunReviewResult, error)
	BindRunReviewSession(
		ctx context.Context,
		req BindRunReviewSessionRequest,
		actor ActorContext,
	) (RunReviewBinding, error)
	LookupRunReviewForSession(ctx context.Context, sessionID string, actor ActorContext) (RunReviewBinding, error)
	ListRunReviews(ctx context.Context, query RunReviewQuery, actor ActorContext) ([]RunReview, error)

	AddDependency(ctx context.Context, spec AddDependency, actor ActorContext) error
	RemoveDependency(ctx context.Context, taskID string, dependsOnID string, actor ActorContext) error

	EnqueueRun(ctx context.Context, spec EnqueueRun, actor ActorContext) (*Run, error)
	ClaimNextRun(ctx context.Context, criteria ClaimCriteria, actor ActorContext) (*ClaimResult, error)
	ClaimRun(ctx context.Context, runID string, claim ClaimRun, actor ActorContext) (*Run, error)
	StartRun(ctx context.Context, runID string, req StartRun, actor ActorContext) (*Run, error)
	AttachRunSession(ctx context.Context, runID string, sessionID string, actor ActorContext) (*Run, error)
	HeartbeatRunLease(ctx context.Context, heartbeat LeaseHeartbeat, actor ActorContext) (*Run, error)
	ReleaseRunLease(ctx context.Context, release LeaseRelease, actor ActorContext) (*Run, error)
	CompleteRunLease(ctx context.Context, completion LeaseCompletion, actor ActorContext) (*Run, error)
	FailRunLease(ctx context.Context, failure LeaseFailure, actor ActorContext) (*Run, error)
	CompleteRun(ctx context.Context, runID string, result RunResult, actor ActorContext) (*Run, error)
	FailRun(ctx context.Context, runID string, failure RunFailure, actor ActorContext) (*Run, error)
	CancelRun(ctx context.Context, runID string, req CancelRun, actor ActorContext) (*Run, error)
	RecoverExpiredRunLeases(
		ctx context.Context,
		recovery ExpiredLeaseRecovery,
		actor ActorContext,
	) ([]ExpiredLeaseRecoveryResult, error)

	GetTask(ctx context.Context, id string, actor ActorContext) (*View, error)
	ListTaskRuns(ctx context.Context, taskID string, query RunQuery, actor ActorContext) ([]Run, error)
	ListTasks(ctx context.Context, query Query, actor ActorContext) ([]Summary, error)

	LiveService
}

// RecordStore is the persistence surface for durable task records.
type RecordStore interface {
	CreateTask(ctx context.Context, task Task) error
	DeleteTask(ctx context.Context, id string) error
	UpdateTask(ctx context.Context, task Task) error
	GetTask(ctx context.Context, id string) (Task, error)
	ListTasks(ctx context.Context, query Query) ([]Summary, error)
	CountDirectChildren(ctx context.Context, parentTaskID string) (int, error)
}

// DeleteTaskMutationStore is the narrowed persistence surface required to
// execute task deletion and dependent reconciliation as one unit.
type DeleteTaskMutationStore interface {
	GetTask(ctx context.Context, id string) (Task, error)
	UpdateTask(ctx context.Context, task Task) error
	DeleteTask(ctx context.Context, id string) error
	CountDirectChildren(ctx context.Context, parentTaskID string) (int, error)
	ListDependencies(ctx context.Context, taskID string) ([]Dependency, error)
	ListDependents(ctx context.Context, dependsOnTaskID string) ([]Dependency, error)
	ListTaskRuns(ctx context.Context, query RunQuery) ([]Run, error)
}

// DeleteTaskTransactionStore optionally exposes transactional delete-task
// execution so the manager can roll back the primary delete when dependent
// reconciliation fails.
type DeleteTaskTransactionStore interface {
	WithDeleteTaskTransaction(ctx context.Context, fn func(DeleteTaskMutationStore) error) error
}

// DependencyStore is the persistence surface for durable dependency edges.
type DependencyStore interface {
	CreateDependency(ctx context.Context, dependency Dependency) error
	DeleteDependency(ctx context.Context, taskID string, dependsOnID string) error
	ListDependencies(ctx context.Context, taskID string) ([]Dependency, error)
	ListDependents(ctx context.Context, dependsOnTaskID string) ([]Dependency, error)
	CountDependencies(ctx context.Context, taskID string) (int, error)
	HasDependencyPath(ctx context.Context, fromTaskID string, toTaskID string) (bool, error)
}

// RunStore is the persistence surface for durable task-run records.
type RunStore interface {
	CreateTaskRun(ctx context.Context, run Run) error
	UpdateTaskRun(ctx context.Context, run Run) error
	GetTaskRun(ctx context.Context, id string) (Run, error)
	ListTaskRuns(ctx context.Context, query RunQuery) ([]Run, error)
	ListTaskRunsByStatus(ctx context.Context, statuses []RunStatus) ([]Run, error)
	CountActiveSessionBindings(ctx context.Context, sessionID string) (int, error)
	ClaimNextRun(ctx context.Context, criteria ClaimCriteria) (ClaimResult, error)
	HeartbeatRunLease(ctx context.Context, heartbeat LeaseHeartbeat) (Run, error)
	ReleaseRunLease(ctx context.Context, release LeaseRelease) (Run, error)
	CompleteRunLease(ctx context.Context, completion LeaseCompletion) (Run, error)
	FailRunLease(ctx context.Context, failure LeaseFailure) (Run, error)
	RecoverExpiredRunLeases(ctx context.Context, recovery ExpiredLeaseRecovery) ([]ExpiredLeaseRecoveryResult, error)
	ReserveQueuedRun(
		ctx context.Context,
		taskID string,
		runID string,
		idempotencyKey string,
		origin Origin,
		requestedChannel string,
		metadata json.RawMessage,
		queuedAt time.Time,
	) (Task, Run, bool, error)
}

// EventStore is the persistence surface for immutable task audit events.
type EventStore interface {
	CreateTaskEvent(ctx context.Context, event Event) error
	ListTaskEvents(ctx context.Context, query EventQuery) ([]Event, error)
}

// EventSequenceStore is the persistence surface for stable task event sequencing used by live reads.
type EventSequenceStore interface {
	GetTaskEventRecord(ctx context.Context, eventID string) (EventRecord, error)
	ListTaskEventRecords(ctx context.Context, query EventRecordQuery) ([]EventRecord, error)
}

// IdempotencyStore is the persistence surface for non-human run idempotency tracking.
type IdempotencyStore interface {
	GetTaskRunByIdempotencyKey(ctx context.Context, key string, origin Origin) (Run, error)
	SaveTaskRunIdempotency(ctx context.Context, record RunIdempotency) error
}

// TriageStore is the persistence surface for durable actor-scoped task triage state.
type TriageStore interface {
	GetTaskTriageState(ctx context.Context, taskID string, actor ActorIdentity) (TriageState, error)
	UpsertTaskTriageState(ctx context.Context, state TriageState) error
}

// ExecutionProfileStore is the persistence surface for task-owned execution profiles.
type ExecutionProfileStore interface {
	GetExecutionProfile(ctx context.Context, taskID string) (ExecutionProfile, error)
	UpsertExecutionProfile(ctx context.Context, profile *ExecutionProfile) (ExecutionProfile, error)
	DeleteExecutionProfile(ctx context.Context, taskID string) error
}

// RunReviewStore is the persistence surface for task-run review gate records.
type RunReviewStore interface {
	RequestRunReview(ctx context.Context, review *RunReview) (RunReview, bool, error)
	GetRunReview(ctx context.Context, reviewID string) (RunReview, error)
	RecordRunReview(
		ctx context.Context,
		req RecordRunReviewRequest,
		actor ActorContext,
		recordedAt time.Time,
		continuationRunID string,
	) (RunReviewResult, error)
	BindRunReviewSession(ctx context.Context, req BindRunReviewSessionRequest, boundAt time.Time) (RunReview, error)
	LookupRunReviewBySession(ctx context.Context, sessionID string) (RunReview, error)
	ListRunReviews(ctx context.Context, query RunReviewQuery) ([]RunReview, error)
}

// Store composes the task-domain persistence surfaces consumed by the manager.
type Store interface {
	RecordStore
	DependencyStore
	RunStore
	EventStore
	EventSequenceStore
	IdempotencyStore
	TriageStore
	ExecutionProfileStore
	RunReviewStore
}

// SessionExecutor is the injected runtime bridge used to start, attach, and stop task sessions.
type SessionExecutor interface {
	StartTaskSession(ctx context.Context, spec *StartTaskSession) (*SessionRef, error)
	AttachTaskSession(ctx context.Context, runID string, sessionID string) (*SessionRef, error)
	RequestTaskStop(ctx context.Context, sessionID string, reason StopReason) error
	ForceTaskStop(ctx context.Context, sessionID string, reason StopReason) error
}
