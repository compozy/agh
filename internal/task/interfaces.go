package task

import (
	"context"
)

// Manager is the task-domain authority for task and run lifecycle operations.
type Manager interface {
	CreateTask(ctx context.Context, spec CreateTask, actor ActorContext) (*Task, error)
	CreateChildTask(ctx context.Context, parentTaskID string, spec CreateTask, actor ActorContext) (*Task, error)
	UpdateTask(ctx context.Context, id string, patch TaskPatch, actor ActorContext) (*Task, error)
	CancelTask(ctx context.Context, id string, req CancelTask, actor ActorContext) (*Task, error)

	AddDependency(ctx context.Context, spec AddDependency, actor ActorContext) error
	RemoveDependency(ctx context.Context, taskID string, dependsOnID string, actor ActorContext) error

	EnqueueRun(ctx context.Context, spec EnqueueRun, actor ActorContext) (*TaskRun, error)
	ClaimRun(ctx context.Context, runID string, claim ClaimRun, actor ActorContext) (*TaskRun, error)
	StartRun(ctx context.Context, runID string, req StartRun, actor ActorContext) (*TaskRun, error)
	AttachRunSession(ctx context.Context, runID string, sessionID string, actor ActorContext) (*TaskRun, error)
	CompleteRun(ctx context.Context, runID string, result RunResult, actor ActorContext) (*TaskRun, error)
	FailRun(ctx context.Context, runID string, failure RunFailure, actor ActorContext) (*TaskRun, error)
	CancelRun(ctx context.Context, runID string, req CancelRun, actor ActorContext) (*TaskRun, error)

	GetTask(ctx context.Context, id string, actor ActorContext) (*TaskView, error)
	ListTasks(ctx context.Context, query TaskQuery, actor ActorContext) ([]TaskSummary, error)
}

// TaskStore is the persistence surface for durable task records.
type TaskStore interface {
	CreateTask(ctx context.Context, task Task) error
	UpdateTask(ctx context.Context, task Task) error
	GetTask(ctx context.Context, id string) (Task, error)
	ListTasks(ctx context.Context, query TaskQuery) ([]TaskSummary, error)
	CountDirectChildren(ctx context.Context, parentTaskID string) (int, error)
}

// DependencyStore is the persistence surface for durable dependency edges.
type DependencyStore interface {
	CreateDependency(ctx context.Context, dependency TaskDependency) error
	DeleteDependency(ctx context.Context, taskID string, dependsOnID string) error
	ListDependencies(ctx context.Context, taskID string) ([]TaskDependency, error)
	ListDependents(ctx context.Context, dependsOnTaskID string) ([]TaskDependency, error)
	CountDependencies(ctx context.Context, taskID string) (int, error)
	HasDependencyPath(ctx context.Context, fromTaskID string, toTaskID string) (bool, error)
}

// RunStore is the persistence surface for durable task-run records.
type RunStore interface {
	CreateTaskRun(ctx context.Context, run TaskRun) error
	UpdateTaskRun(ctx context.Context, run TaskRun) error
	GetTaskRun(ctx context.Context, id string) (TaskRun, error)
	ListTaskRuns(ctx context.Context, query TaskRunQuery) ([]TaskRun, error)
	ListTaskRunsByStatus(ctx context.Context, statuses []TaskRunStatus) ([]TaskRun, error)
	CountActiveSessionBindings(ctx context.Context, sessionID string) (int, error)
}

// EventStore is the persistence surface for immutable task audit events.
type EventStore interface {
	CreateTaskEvent(ctx context.Context, event TaskEvent) error
	ListTaskEvents(ctx context.Context, query TaskEventQuery) ([]TaskEvent, error)
}

// IdempotencyStore is the persistence surface for non-human run idempotency tracking.
type IdempotencyStore interface {
	GetTaskRunByIdempotencyKey(ctx context.Context, key string, origin Origin) (TaskRun, error)
	SaveTaskRunIdempotency(ctx context.Context, record TaskRunIdempotency) error
}

// Store composes the task-domain persistence surfaces consumed by the manager.
type Store interface {
	TaskStore
	DependencyStore
	RunStore
	EventStore
	IdempotencyStore
}

// SessionExecutor is the injected runtime bridge used to start, attach, and stop task sessions.
type SessionExecutor interface {
	StartTaskSession(ctx context.Context, spec StartTaskSession) (*SessionRef, error)
	AttachTaskSession(ctx context.Context, runID string, sessionID string) (*SessionRef, error)
	RequestTaskStop(ctx context.Context, sessionID string, reason StopReason) error
	ForceTaskStop(ctx context.Context, sessionID string, reason StopReason) error
}
