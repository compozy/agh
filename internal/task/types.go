package task

import (
	"encoding/json"
	"time"
)

// Scope identifies whether a task is daemon-global or workspace-scoped.
type Scope string

const (
	// ScopeGlobal identifies a daemon-wide task with no workspace binding.
	ScopeGlobal Scope = "global"
	// ScopeWorkspace identifies a task bound to one workspace.
	ScopeWorkspace Scope = "workspace"
)

// Status identifies the canonical lifecycle state of a task.
type Status string

const (
	// TaskStatusDraft reports a saved draft that is not yet runnable.
	TaskStatusDraft Status = "draft"
	// TaskStatusPending reports a task that exists but has not yet been reconciled into ready work.
	TaskStatusPending Status = "pending"
	// TaskStatusBlocked reports a task with unresolved dependencies.
	TaskStatusBlocked Status = "blocked"
	// TaskStatusReady reports a task that may execute because dependencies are satisfied.
	TaskStatusReady Status = "ready"
	// TaskStatusInProgress reports a task with an active starting or running run.
	TaskStatusInProgress Status = "in_progress"
	// TaskStatusCompleted reports a task that finished successfully.
	TaskStatusCompleted Status = "completed"
	// TaskStatusFailed reports a task that ended unsuccessfully.
	TaskStatusFailed Status = "failed"
	// TaskStatusCanceled reports a task that was canceled before successful completion.
	TaskStatusCanceled Status = "canceled"
)

// Priority identifies the operator-facing urgency assigned to one task.
type Priority string

const (
	// PriorityLow identifies the lowest urgency.
	PriorityLow Priority = "low"
	// PriorityMedium identifies the default urgency.
	PriorityMedium Priority = "medium"
	// PriorityHigh identifies elevated urgency.
	PriorityHigh Priority = "high"
	// PriorityUrgent identifies the highest urgency.
	PriorityUrgent Priority = "urgent"
	// DefaultPriority is the canonical priority used when callers omit the field.
	DefaultPriority Priority = PriorityMedium
)

// ApprovalPolicy identifies whether a task requires an explicit approval step.
type ApprovalPolicy string

const (
	// ApprovalPolicyNone identifies tasks that do not require approval.
	ApprovalPolicyNone ApprovalPolicy = "none"
	// ApprovalPolicyManual identifies tasks that require an explicit approve or reject action.
	ApprovalPolicyManual ApprovalPolicy = "manual"
	// DefaultApprovalPolicy is the canonical policy used when callers omit approval requirements.
	DefaultApprovalPolicy ApprovalPolicy = ApprovalPolicyNone
)

// ApprovalState identifies the current approval outcome for one task.
type ApprovalState string

const (
	// ApprovalStateNotRequired identifies tasks whose policy does not require approval.
	ApprovalStateNotRequired ApprovalState = "not_required"
	// ApprovalStatePending identifies tasks waiting for approval.
	ApprovalStatePending ApprovalState = "pending"
	// ApprovalStateApproved identifies tasks that were approved.
	ApprovalStateApproved ApprovalState = "approved"
	// ApprovalStateRejected identifies tasks that were rejected.
	ApprovalStateRejected ApprovalState = "rejected"
)

// RunStatus identifies the canonical lifecycle state of a task run.
type RunStatus string

const (
	// TaskRunStatusQueued reports a run that has been accepted but not yet claimed.
	TaskRunStatusQueued RunStatus = "queued"
	// TaskRunStatusClaimed reports a run that has been claimed for execution.
	TaskRunStatusClaimed RunStatus = "claimed"
	// TaskRunStatusStarting reports a run that is starting its execution session.
	TaskRunStatusStarting RunStatus = "starting"
	// TaskRunStatusRunning reports a run that is actively executing.
	TaskRunStatusRunning RunStatus = "running"
	// TaskRunStatusCompleted reports a run that finished successfully.
	TaskRunStatusCompleted RunStatus = "completed"
	// TaskRunStatusFailed reports a run that finished with an error.
	TaskRunStatusFailed RunStatus = "failed"
	// TaskRunStatusCanceled reports a run that was canceled.
	TaskRunStatusCanceled RunStatus = "canceled"
)

// ActorKind identifies the authenticated principal class behind task writes.
type ActorKind string

const (
	// ActorKindHuman identifies a human principal writing through CLI, web, HTTP, or UDS surfaces.
	ActorKindHuman ActorKind = "human"
	// ActorKindAgentSession identifies an AGH agent session principal.
	ActorKindAgentSession ActorKind = "agent_session"
	// ActorKindAutomation identifies daemon-owned automation flows.
	ActorKindAutomation ActorKind = "automation"
	// ActorKindExtension identifies an authenticated extension runtime principal.
	ActorKindExtension ActorKind = "extension"
	// ActorKindNetworkPeer identifies an authenticated network peer principal.
	ActorKindNetworkPeer ActorKind = "network_peer"
	// ActorKindDaemon identifies daemon-owned system work.
	ActorKindDaemon ActorKind = "daemon"
)

// OwnerKind identifies who currently owns a task operationally.
type OwnerKind string

const (
	// OwnerKindHuman identifies a human owner.
	OwnerKindHuman OwnerKind = "human"
	// OwnerKindAgentSession identifies an agent-session owner.
	OwnerKindAgentSession OwnerKind = "agent_session"
	// OwnerKindAutomation identifies an automation owner.
	OwnerKindAutomation OwnerKind = "automation"
	// OwnerKindExtension identifies an extension owner.
	OwnerKindExtension OwnerKind = "extension"
	// OwnerKindNetworkPeer identifies a network-peer owner.
	OwnerKindNetworkPeer OwnerKind = "network_peer"
	// OwnerKindPool identifies pooled ownership without a dedicated assignee.
	OwnerKindPool OwnerKind = "pool"
)

// OriginKind identifies the technical ingress surface that produced a task-domain write.
type OriginKind string

const (
	// OriginKindCLI identifies CLI ingress.
	OriginKindCLI OriginKind = "cli"
	// OriginKindWeb identifies web UI ingress.
	OriginKindWeb OriginKind = "web"
	// OriginKindUDS identifies local UDS ingress.
	OriginKindUDS OriginKind = "uds"
	// OriginKindHTTP identifies HTTP ingress.
	OriginKindHTTP OriginKind = "http"
	// OriginKindAutomation identifies automation ingress.
	OriginKindAutomation OriginKind = "automation"
	// OriginKindExtension identifies extension ingress.
	OriginKindExtension OriginKind = "extension"
	// OriginKindNetwork identifies network ingress.
	OriginKindNetwork OriginKind = "network"
	// OriginKindAgentSession identifies session tool-call ingress.
	OriginKindAgentSession OriginKind = "agent_session"
	// OriginKindDaemon identifies daemon-owned internal ingress.
	OriginKindDaemon OriginKind = "daemon"
)

// DependencyKind identifies the semantic meaning of one dependency edge.
type DependencyKind string

const (
	// DependencyKindBlocks identifies a dependency that must resolve before the task may proceed.
	DependencyKindBlocks DependencyKind = "blocks"
)

// StopReason identifies why the task domain asked the session bridge to stop a session.
type StopReason string

const (
	// StopReasonCompleted identifies successful task-run completion.
	StopReasonCompleted StopReason = "completed"
	// StopReasonFailed identifies failed task-run termination.
	StopReasonFailed StopReason = "failed"
	// StopReasonCancellation identifies explicit task or run cancellation.
	StopReasonCancellation StopReason = "cancellation"
	// StopReasonShutdown identifies daemon shutdown or boot recovery stop requests.
	StopReasonShutdown StopReason = "shutdown"
	// StopReasonOrphanedRun identifies orphaned-run recovery handling.
	StopReasonOrphanedRun StopReason = "orphaned_run"
)

// RunBootRecoveryAction identifies the manager-owned recovery action applied to
// a non-terminal run during daemon startup reconciliation.
type RunBootRecoveryAction string

const (
	// RunBootRecoveryRequeue resets one claimed run back to the durable queue.
	RunBootRecoveryRequeue RunBootRecoveryAction = "requeue"
	// RunBootRecoveryMarkRunning promotes one live attached run into running.
	RunBootRecoveryMarkRunning RunBootRecoveryAction = "mark_running"
	// RunBootRecoveryFail marks one orphaned attached run as failed.
	RunBootRecoveryFail RunBootRecoveryAction = "fail"
)

// ActorIdentity is the immutable server-derived actor identity attached to task and run writes.
type ActorIdentity struct {
	Kind ActorKind `json:"kind"`
	Ref  string    `json:"ref"`
}

// Ownership is the optional mutable operational assignee attached to a task.
type Ownership struct {
	Kind OwnerKind `json:"kind"`
	Ref  string    `json:"ref"`
}

// Origin is the immutable technical ingress context attached to task and run writes.
type Origin struct {
	Kind OriginKind `json:"kind"`
	Ref  string     `json:"ref"`
}

// Authority captures the task-domain permissions resolved for one authenticated principal.
type Authority struct {
	Read            bool `json:"read"`
	Write           bool `json:"write"`
	CreateGlobal    bool `json:"create_global"`
	CreateWorkspace bool `json:"create_workspace"`
}

// ActorContext carries the authenticated principal, ingress origin, and resolved task authority.
type ActorContext struct {
	Actor     ActorIdentity `json:"actor"`
	Origin    Origin        `json:"origin"`
	Authority Authority     `json:"authority"`
}

// Task is the durable coordination record owned by the task domain.
type Task struct {
	ID             string          `json:"id"`
	Identifier     string          `json:"identifier,omitempty"`
	Scope          Scope           `json:"scope"`
	WorkspaceID    string          `json:"workspace_id,omitempty"`
	ParentTaskID   string          `json:"parent_task_id,omitempty"`
	NetworkChannel string          `json:"network_channel,omitempty"`
	Title          string          `json:"title"`
	Description    string          `json:"description,omitempty"`
	Priority       Priority        `json:"priority,omitempty"`
	MaxAttempts    int             `json:"max_attempts,omitempty"`
	Status         Status          `json:"status"`
	ApprovalPolicy ApprovalPolicy  `json:"approval_policy,omitempty"`
	ApprovalState  ApprovalState   `json:"approval_state,omitempty"`
	Owner          *Ownership      `json:"owner,omitempty"`
	CurrentRunID   string          `json:"current_run_id,omitempty"`
	LatestEventSeq int64           `json:"latest_event_seq"`
	CreatedBy      ActorIdentity   `json:"created_by"`
	Origin         Origin          `json:"origin"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	ClosedAt       time.Time       `json:"closed_at"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

// Dependency is the durable edge record connecting one task to a blocking dependency.
type Dependency struct {
	TaskID          string         `json:"task_id"`
	DependsOnTaskID string         `json:"depends_on_task_id"`
	Kind            DependencyKind `json:"kind"`
	CreatedAt       time.Time      `json:"created_at"`
}

// RunReviewLineage captures review-gate fields attached to a task run.
type RunReviewLineage struct {
	Required           bool            `json:"required,omitempty"`
	RequestRound       int             `json:"request_round,omitempty"`
	PolicySnapshot     ReviewPolicy    `json:"policy_snapshot,omitempty"`
	RequestID          string          `json:"request_id,omitempty"`
	ParentRunID        string          `json:"parent_run_id,omitempty"`
	ReviewID           string          `json:"review_id,omitempty"`
	ReviewRound        int             `json:"review_round,omitempty"`
	ContinuationReason string          `json:"continuation_reason,omitempty"`
	MissingWork        json.RawMessage `json:"missing_work,omitempty"`
	NextRoundGuidance  string          `json:"next_round_guidance,omitempty"`
}

// Run is the durable execution record for one task attempt.
type Run struct {
	ID                    string            `json:"id"`
	TaskID                string            `json:"task_id"`
	Status                RunStatus         `json:"status"`
	Attempt               int               `json:"attempt"`
	ClaimedBy             *ActorIdentity    `json:"claimed_by,omitempty"`
	SessionID             string            `json:"session_id,omitempty"`
	Origin                Origin            `json:"origin"`
	IdempotencyKey        string            `json:"idempotency_key,omitempty"`
	NetworkChannel        string            `json:"network_channel,omitempty"`
	ClaimToken            string            `json:"-"`
	ClaimTokenHash        string            `json:"claim_token_hash,omitempty"`
	LeaseUntil            time.Time         `json:"lease_until"`
	HeartbeatAt           time.Time         `json:"heartbeat_at"`
	CoordinationChannelID string            `json:"coordination_channel_id,omitempty"`
	RequiredCapabilities  []string          `json:"required_capabilities,omitempty"`
	PreferredCapabilities []string          `json:"preferred_capabilities,omitempty"`
	Review                *RunReviewLineage `json:"review,omitempty"`
	Metadata              json.RawMessage   `json:"metadata,omitempty"`
	QueuedAt              time.Time         `json:"queued_at"`
	ClaimedAt             time.Time         `json:"claimed_at"`
	StartedAt             time.Time         `json:"started_at"`
	EndedAt               time.Time         `json:"ended_at"`
	Error                 string            `json:"error,omitempty"`
	Result                json.RawMessage   `json:"result,omitempty"`
}

// Event is the immutable audit record emitted for task-domain actions.
type Event struct {
	ID        string          `json:"id"`
	TaskID    string          `json:"task_id"`
	RunID     string          `json:"run_id,omitempty"`
	EventType string          `json:"event_type"`
	Actor     ActorIdentity   `json:"actor"`
	Origin    Origin          `json:"origin"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// RunIdempotency is the durable deduplication record for non-human run ingress.
type RunIdempotency struct {
	IdempotencyKey string    `json:"idempotency_key"`
	RunID          string    `json:"run_id"`
	Origin         Origin    `json:"origin"`
	CreatedAt      time.Time `json:"created_at"`
}

// TriageState is the durable actor-scoped inbox and triage state for one task.
type TriageState struct {
	TaskID             string        `json:"task_id"`
	Actor              ActorIdentity `json:"actor"`
	Read               bool          `json:"read"`
	Archived           bool          `json:"archived"`
	Dismissed          bool          `json:"dismissed"`
	LastSeenActivityAt time.Time     `json:"last_seen_activity_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

// Summary is the lightweight read model returned from list-oriented task queries.
type Summary struct {
	ID              string                `json:"id"`
	Identifier      string                `json:"identifier,omitempty"`
	Scope           Scope                 `json:"scope"`
	WorkspaceID     string                `json:"workspace_id,omitempty"`
	ParentTaskID    string                `json:"parent_task_id,omitempty"`
	NetworkChannel  string                `json:"network_channel,omitempty"`
	Title           string                `json:"title"`
	Priority        Priority              `json:"priority,omitempty"`
	MaxAttempts     int                   `json:"max_attempts,omitempty"`
	Status          Status                `json:"status"`
	ApprovalPolicy  ApprovalPolicy        `json:"approval_policy,omitempty"`
	ApprovalState   ApprovalState         `json:"approval_state,omitempty"`
	Draft           bool                  `json:"draft"`
	Owner           *Ownership            `json:"owner,omitempty"`
	CurrentRunID    string                `json:"current_run_id,omitempty"`
	LatestEventSeq  int64                 `json:"latest_event_seq"`
	CreatedBy       ActorIdentity         `json:"created_by"`
	Origin          Origin                `json:"origin"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	ClosedAt        time.Time             `json:"closed_at"`
	ChildCount      int                   `json:"child_count,omitempty"`
	DependencyCount int                   `json:"dependency_count,omitempty"`
	Dependencies    []DependencyReference `json:"dependencies,omitempty"`
	ActiveRun       *RunSummary           `json:"active_run,omitempty"`
	LastActivityAt  time.Time             `json:"last_activity_at"`
}

// Reference is the human-meaningful task identity used in enriched read models.
type Reference struct {
	ID             string     `json:"id"`
	Identifier     string     `json:"identifier,omitempty"`
	Title          string     `json:"title"`
	Status         Status     `json:"status"`
	Priority       Priority   `json:"priority,omitempty"`
	Owner          *Ownership `json:"owner,omitempty"`
	Scope          Scope      `json:"scope"`
	WorkspaceID    string     `json:"workspace_id,omitempty"`
	LatestEventSeq int64      `json:"latest_event_seq"`
}

// DependencyReference enriches one dependency edge with the referenced blocker identity.
type DependencyReference struct {
	TaskID          string         `json:"task_id"`
	DependsOnTaskID string         `json:"depends_on_task_id"`
	Kind            DependencyKind `json:"kind"`
	CreatedAt       time.Time      `json:"created_at"`
	DependsOn       Reference      `json:"depends_on"`
}

// RunSummary captures the operator-facing run chip data used by enriched task cards.
type RunSummary struct {
	ID                    string         `json:"id"`
	TaskID                string         `json:"task_id"`
	Status                RunStatus      `json:"status"`
	Attempt               int            `json:"attempt"`
	MaxAttempts           int            `json:"max_attempts"`
	SessionID             string         `json:"session_id,omitempty"`
	ClaimedBy             *ActorIdentity `json:"claimed_by,omitempty"`
	ClaimTokenHash        string         `json:"claim_token_hash,omitempty"`
	LeaseUntil            time.Time      `json:"lease_until"`
	HeartbeatAt           time.Time      `json:"heartbeat_at"`
	CoordinationChannelID string         `json:"coordination_channel_id,omitempty"`
	QueuedAt              time.Time      `json:"queued_at"`
	ClaimedAt             time.Time      `json:"claimed_at"`
	StartedAt             time.Time      `json:"started_at"`
	EndedAt               time.Time      `json:"ended_at"`
	Error                 string         `json:"error,omitempty"`
}

// View is the expanded read model returned from single-task lookups.
type View struct {
	Summary              Summary               `json:"summary"`
	Task                 Task                  `json:"task"`
	Children             []Summary             `json:"children,omitempty"`
	Dependencies         []Dependency          `json:"dependencies,omitempty"`
	DependencyReferences []DependencyReference `json:"dependency_references,omitempty"`
	Runs                 []Run                 `json:"runs,omitempty"`
	Events               []Event               `json:"events,omitempty"`
}

// CreateTask captures the mutable inputs accepted when creating a new task.
type CreateTask struct {
	ID             string          `json:"id,omitempty"`
	Identifier     string          `json:"identifier,omitempty"`
	Scope          Scope           `json:"scope"`
	WorkspaceID    string          `json:"workspace_id,omitempty"`
	ParentTaskID   string          `json:"parent_task_id,omitempty"`
	NetworkChannel string          `json:"network_channel,omitempty"`
	Title          string          `json:"title"`
	Description    string          `json:"description,omitempty"`
	Priority       Priority        `json:"priority,omitempty"`
	MaxAttempts    *int            `json:"max_attempts,omitempty"`
	Draft          bool            `json:"draft,omitempty"`
	ApprovalPolicy ApprovalPolicy  `json:"approval_policy,omitempty"`
	Owner          *Ownership      `json:"owner,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

// Patch captures the mutable task fields accepted by update operations.
type Patch struct {
	Title          *string          `json:"title,omitempty"`
	Description    *string          `json:"description,omitempty"`
	Priority       *Priority        `json:"priority,omitempty"`
	MaxAttempts    *int             `json:"max_attempts,omitempty"`
	ApprovalPolicy *ApprovalPolicy  `json:"approval_policy,omitempty"`
	Metadata       *json.RawMessage `json:"metadata,omitempty"`
	NetworkChannel *string          `json:"network_channel,omitempty"`
	Owner          *Ownership       `json:"owner,omitempty"`
	ClearOwner     bool             `json:"clear_owner,omitempty"`
}

// CancelTask captures the task-level cancellation request payload.
type CancelTask struct {
	Reason   string          `json:"reason,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// ExecutionAction identifies the operator action that crossed the
// create-versus-execute lifecycle boundary.
type ExecutionAction string

const (
	// ExecutionActionStart records an explicit operator start request.
	ExecutionActionStart ExecutionAction = "start"
	// ExecutionActionPublish records a draft publish request that also starts execution.
	ExecutionActionPublish ExecutionAction = "publish"
	// ExecutionActionApproval records an approval request that also starts execution.
	ExecutionActionApproval ExecutionAction = "approval"
)

// ExecutionRequest captures the mutable inputs accepted when an operator
// publish, start, or approval action enqueues executable work.
type ExecutionRequest struct {
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	NetworkChannel string          `json:"network_channel,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

// Execution captures the task and run created or resolved at the explicit
// execution boundary.
type Execution struct {
	Task        Task            `json:"task"`
	Run         Run             `json:"run"`
	Action      ExecutionAction `json:"action"`
	ExistingRun bool            `json:"existing_run,omitempty"`
}

// AddDependency captures one dependency-edge creation request.
type AddDependency struct {
	TaskID          string         `json:"task_id"`
	DependsOnTaskID string         `json:"depends_on_task_id"`
	Kind            DependencyKind `json:"kind"`
}

// EnqueueRun captures the mutable inputs accepted when queuing a task run.
type EnqueueRun struct {
	TaskID         string          `json:"task_id"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	NetworkChannel string          `json:"network_channel,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

// ClaimRun captures one run-claim request.
type ClaimRun struct {
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// StartRun captures one run-start request.
type StartRun struct {
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// CancelRun captures one run-cancellation request.
type CancelRun struct {
	Reason   string          `json:"reason,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// RunResult captures the durable JSON result returned by a completed run.
type RunResult struct {
	Value json.RawMessage `json:"value,omitempty"`
}

// RunFailure captures the durable failure payload returned by a failed run.
type RunFailure struct {
	Error    string          `json:"error"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// Query captures the supported list filters for task reads.
type Query struct {
	Scope          Scope         `json:"scope,omitempty"`
	WorkspaceID    string        `json:"workspace_id,omitempty"`
	Status         Status        `json:"status,omitempty"`
	Priority       Priority      `json:"priority,omitempty"`
	ApprovalState  ApprovalState `json:"approval_state,omitempty"`
	OwnerKind      OwnerKind     `json:"owner_kind,omitempty"`
	OwnerRef       string        `json:"owner_ref,omitempty"`
	ParentTaskID   string        `json:"parent_task_id,omitempty"`
	NetworkChannel string        `json:"network_channel,omitempty"`
	Search         string        `json:"search,omitempty"`
	Limit          int           `json:"limit,omitempty"`
}

// RunQuery captures the supported list filters for task-run reads.
type RunQuery struct {
	TaskID                string    `json:"task_id,omitempty"`
	Status                RunStatus `json:"status,omitempty"`
	SessionID             string    `json:"session_id,omitempty"`
	CoordinationChannelID string    `json:"coordination_channel_id,omitempty"`
	Limit                 int       `json:"limit,omitempty"`
}

// EventQuery captures the supported list filters for task-event reads.
type EventQuery struct {
	TaskID    string `json:"task_id,omitempty"`
	RunID     string `json:"run_id,omitempty"`
	EventType string `json:"event_type,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

// StartTaskSession captures the task and run context needed to allocate a dedicated session.
type StartTaskSession struct {
	Task             Task              `json:"task"`
	Run              Run               `json:"run"`
	ExecutionProfile *ExecutionProfile `json:"execution_profile,omitempty"`
	Actor            ActorContext      `json:"actor"`
}

// SessionRef is the task-domain view of a runtime session binding.
type SessionRef struct {
	SessionID   string    `json:"session_id"`
	WorkspaceID string    `json:"workspace_id,omitempty"`
	StartedAt   time.Time `json:"started_at"`
}

// RunBootRecovery captures one daemon-owned recovery decision for an in-flight
// run discovered during boot reconciliation.
type RunBootRecovery struct {
	Action         RunBootRecoveryAction `json:"action"`
	Reason         string                `json:"reason,omitempty"`
	SessionState   string                `json:"session_state,omitempty"`
	Classification string                `json:"classification,omitempty"`
	Detail         string                `json:"detail,omitempty"`
}
