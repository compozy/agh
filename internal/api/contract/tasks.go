package contract

import (
	"encoding/json"
	"time"

	taskpkg "github.com/pedronauck/agh/internal/task"
)

// TaskSummaryPayload is the shared list-oriented task response payload.
type TaskSummaryPayload struct {
	ID             string                `json:"id"`
	Identifier     string                `json:"identifier,omitempty"`
	Scope          taskpkg.Scope         `json:"scope"`
	WorkspaceID    string                `json:"workspace_id,omitempty"`
	ParentTaskID   string                `json:"parent_task_id,omitempty"`
	NetworkChannel string                `json:"network_channel,omitempty"`
	Title          string                `json:"title"`
	Status         taskpkg.TaskStatus    `json:"status"`
	Owner          *taskpkg.Ownership    `json:"owner,omitempty"`
	CreatedBy      taskpkg.ActorIdentity `json:"created_by"`
	Origin         taskpkg.Origin        `json:"origin"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
	ClosedAt       time.Time             `json:"closed_at,omitempty"`
}

// TaskPayload is the shared full task response payload.
type TaskPayload struct {
	ID             string                `json:"id"`
	Identifier     string                `json:"identifier,omitempty"`
	Scope          taskpkg.Scope         `json:"scope"`
	WorkspaceID    string                `json:"workspace_id,omitempty"`
	ParentTaskID   string                `json:"parent_task_id,omitempty"`
	NetworkChannel string                `json:"network_channel,omitempty"`
	Title          string                `json:"title"`
	Description    string                `json:"description,omitempty"`
	Status         taskpkg.TaskStatus    `json:"status"`
	Owner          *taskpkg.Ownership    `json:"owner,omitempty"`
	CreatedBy      taskpkg.ActorIdentity `json:"created_by"`
	Origin         taskpkg.Origin        `json:"origin"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
	ClosedAt       time.Time             `json:"closed_at,omitempty"`
	Metadata       json.RawMessage       `json:"metadata,omitempty"`
}

// TaskDependencyPayload is the shared dependency-edge response payload.
type TaskDependencyPayload struct {
	TaskID          string                 `json:"task_id"`
	DependsOnTaskID string                 `json:"depends_on_task_id"`
	Kind            taskpkg.DependencyKind `json:"kind"`
	CreatedAt       time.Time              `json:"created_at"`
}

// TaskRunPayload is the shared task-run response payload.
type TaskRunPayload struct {
	ID             string                 `json:"id"`
	TaskID         string                 `json:"task_id"`
	Status         taskpkg.TaskRunStatus  `json:"status"`
	Attempt        int                    `json:"attempt"`
	ClaimedBy      *taskpkg.ActorIdentity `json:"claimed_by,omitempty"`
	SessionID      string                 `json:"session_id,omitempty"`
	Origin         taskpkg.Origin         `json:"origin"`
	IdempotencyKey string                 `json:"idempotency_key,omitempty"`
	NetworkChannel string                 `json:"network_channel,omitempty"`
	QueuedAt       time.Time              `json:"queued_at"`
	ClaimedAt      time.Time              `json:"claimed_at,omitempty"`
	StartedAt      time.Time              `json:"started_at,omitempty"`
	EndedAt        time.Time              `json:"ended_at,omitempty"`
	Error          string                 `json:"error,omitempty"`
	Result         json.RawMessage        `json:"result,omitempty"`
}

// TaskEventPayload is the shared task audit-event response payload.
type TaskEventPayload struct {
	ID        string                `json:"id"`
	TaskID    string                `json:"task_id"`
	RunID     string                `json:"run_id,omitempty"`
	EventType string                `json:"event_type"`
	Actor     taskpkg.ActorIdentity `json:"actor"`
	Origin    taskpkg.Origin        `json:"origin"`
	Payload   json.RawMessage       `json:"payload,omitempty"`
	Timestamp time.Time             `json:"timestamp"`
}

// TaskDetailPayload is the shared expanded task response payload.
type TaskDetailPayload struct {
	Task         TaskPayload             `json:"task"`
	Children     []TaskSummaryPayload    `json:"children,omitempty"`
	Dependencies []TaskDependencyPayload `json:"dependencies,omitempty"`
	Runs         []TaskRunPayload        `json:"runs,omitempty"`
	Events       []TaskEventPayload      `json:"events,omitempty"`
}

// TaskListQuery captures the shared task list filters.
type TaskListQuery struct {
	Scope          taskpkg.Scope      `json:"scope,omitempty"`
	Workspace      string             `json:"workspace,omitempty"`
	Status         taskpkg.TaskStatus `json:"status,omitempty"`
	OwnerKind      taskpkg.OwnerKind  `json:"owner_kind,omitempty"`
	OwnerRef       string             `json:"owner_ref,omitempty"`
	ParentTaskID   string             `json:"parent_task_id,omitempty"`
	NetworkChannel string             `json:"network_channel,omitempty"`
	Limit          int                `json:"limit,omitempty"`
}

// TaskRunListQuery captures the shared task-run list filters.
type TaskRunListQuery struct {
	Status    taskpkg.TaskRunStatus `json:"status,omitempty"`
	SessionID string                `json:"session_id,omitempty"`
	Limit     int                   `json:"limit,omitempty"`
}

// CreateTaskRequest is the shared task-create request payload.
type CreateTaskRequest struct {
	ID             string             `json:"id,omitempty"`
	Identifier     string             `json:"identifier,omitempty"`
	Scope          taskpkg.Scope      `json:"scope"`
	Workspace      string             `json:"workspace,omitempty"`
	NetworkChannel string             `json:"network_channel,omitempty"`
	Title          string             `json:"title"`
	Description    string             `json:"description,omitempty"`
	Owner          *taskpkg.Ownership `json:"owner,omitempty"`
	Metadata       json.RawMessage    `json:"metadata,omitempty"`
}

// CreateTaskChildRequest is the shared child-task create payload.
type CreateTaskChildRequest struct {
	ID             string             `json:"id,omitempty"`
	Identifier     string             `json:"identifier,omitempty"`
	Scope          taskpkg.Scope      `json:"scope"`
	Workspace      string             `json:"workspace,omitempty"`
	NetworkChannel string             `json:"network_channel,omitempty"`
	Title          string             `json:"title"`
	Description    string             `json:"description,omitempty"`
	Owner          *taskpkg.Ownership `json:"owner,omitempty"`
	Metadata       json.RawMessage    `json:"metadata,omitempty"`
}

// UpdateTaskRequest is the shared task patch payload.
type UpdateTaskRequest struct {
	Title          *string            `json:"title,omitempty"`
	Description    *string            `json:"description,omitempty"`
	Metadata       *json.RawMessage   `json:"metadata,omitempty"`
	NetworkChannel *string            `json:"network_channel,omitempty"`
	Owner          *taskpkg.Ownership `json:"owner,omitempty"`
	ClearOwner     bool               `json:"clear_owner,omitempty"`
}

// HasChanges reports whether the patch includes any mutable task field.
func (r UpdateTaskRequest) HasChanges() bool {
	return r.Title != nil ||
		r.Description != nil ||
		r.Metadata != nil ||
		r.NetworkChannel != nil ||
		r.Owner != nil ||
		r.ClearOwner
}

// CancelTaskRequest is the shared task-cancel request payload.
type CancelTaskRequest struct {
	Reason   string          `json:"reason,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// AddTaskDependencyRequest is the shared dependency-create request payload.
type AddTaskDependencyRequest struct {
	DependsOnTaskID string                 `json:"depends_on_task_id"`
	Kind            taskpkg.DependencyKind `json:"kind,omitempty"`
}

// EnqueueTaskRunRequest is the shared run-enqueue request payload.
type EnqueueTaskRunRequest struct {
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	NetworkChannel string `json:"network_channel,omitempty"`
}

// ClaimTaskRunRequest is the shared run-claim request payload.
type ClaimTaskRunRequest struct {
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// StartTaskRunRequest is the shared run-start request payload.
type StartTaskRunRequest struct {
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// AttachTaskRunSessionRequest is the shared run-session attach request payload.
type AttachTaskRunSessionRequest struct {
	SessionID string `json:"session_id"`
}

// CompleteTaskRunRequest is the shared run-complete request payload.
type CompleteTaskRunRequest struct {
	Result json.RawMessage `json:"result,omitempty"`
}

// FailTaskRunRequest is the shared run-fail request payload.
type FailTaskRunRequest struct {
	Error    string          `json:"error"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// CancelTaskRunRequest is the shared run-cancel request payload.
type CancelTaskRunRequest struct {
	Reason   string          `json:"reason,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}
