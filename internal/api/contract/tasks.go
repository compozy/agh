package contract

import (
	"encoding/json"
	"time"

	taskpkg "github.com/compozy/agh/internal/task"
)

// TaskReferencePayload is the human-meaningful task identity shared across task read models.
type TaskReferencePayload struct {
	ID              string             `json:"id"`
	Identifier      string             `json:"identifier,omitempty"`
	Title           string             `json:"title"`
	Status          taskpkg.Status     `json:"status"`
	Priority        taskpkg.Priority   `json:"priority,omitempty"`
	Owner           *taskpkg.Ownership `json:"owner,omitempty"`
	Scope           taskpkg.Scope      `json:"scope"`
	WorkspaceID     string             `json:"workspace_id,omitempty"`
	LatestEventSeq  int64              `json:"latest_event_seq"`
	Paused          bool               `json:"paused,omitempty"`
	EffectivePaused bool               `json:"effective_paused,omitempty"`
	PausedByTaskID  string             `json:"paused_by_task_id,omitempty"`
}

// TaskSummaryPayload is the shared list-oriented task response payload.
type TaskSummaryPayload struct {
	ID                 string                           `json:"id"`
	Identifier         string                           `json:"identifier,omitempty"`
	Scope              taskpkg.Scope                    `json:"scope"`
	WorkspaceID        string                           `json:"workspace_id,omitempty"`
	ParentTaskID       string                           `json:"parent_task_id,omitempty"`
	NetworkChannel     string                           `json:"network_channel,omitempty"`
	Title              string                           `json:"title"`
	Priority           taskpkg.Priority                 `json:"priority,omitempty"`
	MaxAttempts        int                              `json:"max_attempts,omitempty"`
	AutoEnqueueOnReady bool                             `json:"auto_enqueue_on_ready,omitempty"`
	Status             taskpkg.Status                   `json:"status"`
	ApprovalPolicy     taskpkg.ApprovalPolicy           `json:"approval_policy,omitempty"`
	ApprovalState      taskpkg.ApprovalState            `json:"approval_state,omitempty"`
	Draft              bool                             `json:"draft,omitempty"`
	Owner              *taskpkg.Ownership               `json:"owner,omitempty"`
	CurrentRunID       string                           `json:"current_run_id,omitempty"`
	LatestEventSeq     int64                            `json:"latest_event_seq"`
	Paused             bool                             `json:"paused,omitempty"`
	PausedBy           string                           `json:"paused_by,omitempty"`
	PausedAt           *time.Time                       `json:"paused_at,omitempty"`
	PausedReason       string                           `json:"paused_reason,omitempty"`
	EffectivePaused    bool                             `json:"effective_paused,omitempty"`
	PausedByTaskID     string                           `json:"paused_by_task_id,omitempty"`
	CreatedBy          taskpkg.ActorIdentity            `json:"created_by"`
	Origin             taskpkg.Origin                   `json:"origin"`
	CreatedAt          time.Time                        `json:"created_at"`
	UpdatedAt          time.Time                        `json:"updated_at"`
	ClosedAt           *time.Time                       `json:"closed_at,omitempty"`
	ChildCount         int                              `json:"child_count,omitempty"`
	DependencyCount    int                              `json:"dependency_count,omitempty"`
	Dependencies       []TaskDependencyReferencePayload `json:"dependencies,omitempty"`
	ActiveRun          *TaskRunSummaryPayload           `json:"active_run,omitempty"`
	LastActivityAt     *time.Time                       `json:"last_activity_at,omitempty"`
}

// TaskPayload is the shared full task response payload.
type TaskPayload struct {
	ID                 string                 `json:"id"`
	Identifier         string                 `json:"identifier,omitempty"`
	Scope              taskpkg.Scope          `json:"scope"`
	WorkspaceID        string                 `json:"workspace_id,omitempty"`
	ParentTaskID       string                 `json:"parent_task_id,omitempty"`
	NetworkChannel     string                 `json:"network_channel,omitempty"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description,omitempty"`
	Priority           taskpkg.Priority       `json:"priority,omitempty"`
	MaxAttempts        int                    `json:"max_attempts,omitempty"`
	AutoEnqueueOnReady bool                   `json:"auto_enqueue_on_ready,omitempty"`
	Status             taskpkg.Status         `json:"status"`
	ApprovalPolicy     taskpkg.ApprovalPolicy `json:"approval_policy,omitempty"`
	ApprovalState      taskpkg.ApprovalState  `json:"approval_state,omitempty"`
	Draft              bool                   `json:"draft,omitempty"`
	Owner              *taskpkg.Ownership     `json:"owner,omitempty"`
	CurrentRunID       string                 `json:"current_run_id,omitempty"`
	LatestEventSeq     int64                  `json:"latest_event_seq"`
	Paused             bool                   `json:"paused,omitempty"`
	PausedBy           string                 `json:"paused_by,omitempty"`
	PausedAt           *time.Time             `json:"paused_at,omitempty"`
	PausedReason       string                 `json:"paused_reason,omitempty"`
	EffectivePaused    bool                   `json:"effective_paused,omitempty"`
	PausedByTaskID     string                 `json:"paused_by_task_id,omitempty"`
	CreatedBy          taskpkg.ActorIdentity  `json:"created_by"`
	Origin             taskpkg.Origin         `json:"origin"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	ClosedAt           *time.Time             `json:"closed_at,omitempty"`
	Metadata           json.RawMessage        `json:"metadata,omitempty"`
}

// TaskExecutionProfilePayload is the task-owned orchestration profile read model.
type TaskExecutionProfilePayload = taskpkg.ExecutionProfile

// SetTaskExecutionProfileRequest captures one profile replacement request.
type SetTaskExecutionProfileRequest = taskpkg.ExecutionProfile

// TaskExecutionProfileResponse wraps one execution profile response.
type TaskExecutionProfileResponse struct {
	Profile TaskExecutionProfilePayload `json:"profile"`
}

// TaskRunReviewPayload is the task-run review gate read model.
type TaskRunReviewPayload = taskpkg.RunReview

// CreateTaskRunReviewRequest captures one request to review a terminal task run.
type CreateTaskRunReviewRequest = taskpkg.RunReviewRequest

// SubmitTaskRunReviewVerdictRequest captures one persisted reviewer verdict write.
type SubmitTaskRunReviewVerdictRequest struct {
	RunID   string                   `json:"run_id"`
	Verdict taskpkg.RunReviewVerdict `json:"verdict"`
}

// TaskRunReviewListQuery captures shared review read filters.
type TaskRunReviewListQuery = taskpkg.RunReviewQuery

// TaskDependencyPayload is the shared dependency-edge response payload.
type TaskDependencyPayload struct {
	TaskID          string                 `json:"task_id"`
	DependsOnTaskID string                 `json:"depends_on_task_id"`
	Kind            taskpkg.DependencyKind `json:"kind"`
	CreatedAt       time.Time              `json:"created_at"`
}

// TaskDependencyReferencePayload enriches one dependency edge with the referenced blocker identity.
type TaskDependencyReferencePayload struct {
	TaskID          string                 `json:"task_id"`
	DependsOnTaskID string                 `json:"depends_on_task_id"`
	Kind            taskpkg.DependencyKind `json:"kind"`
	CreatedAt       time.Time              `json:"created_at"`
	DependsOn       TaskReferencePayload   `json:"depends_on"`
}

// TaskRunPayload is the shared task-run response payload.
type TaskRunPayload struct {
	ID                    string                      `json:"id"`
	TaskID                string                      `json:"task_id"`
	Status                taskpkg.RunStatus           `json:"status"`
	Attempt               int                         `json:"attempt"`
	PreviousRunID         string                      `json:"previous_run_id,omitempty"`
	FailureKind           string                      `json:"failure_kind,omitempty"`
	ClaimedBy             *taskpkg.ActorIdentity      `json:"claimed_by,omitempty"`
	SessionID             string                      `json:"session_id,omitempty"`
	Origin                taskpkg.Origin              `json:"origin"`
	IdempotencyKey        string                      `json:"idempotency_key,omitempty"`
	NetworkChannel        string                      `json:"network_channel,omitempty"`
	ClaimTokenHash        string                      `json:"claim_token_hash,omitempty"`
	LeaseUntil            *time.Time                  `json:"lease_until,omitempty"`
	HeartbeatAt           *time.Time                  `json:"heartbeat_at,omitempty"`
	CoordinationChannelID string                      `json:"coordination_channel_id,omitempty"`
	CoordinationChannel   *CoordinationChannelPayload `json:"coordination_channel,omitempty"`
	QueuedAt              time.Time                   `json:"queued_at"`
	ClaimedAt             *time.Time                  `json:"claimed_at,omitempty"`
	StartedAt             *time.Time                  `json:"started_at,omitempty"`
	EndedAt               *time.Time                  `json:"ended_at,omitempty"`
	Error                 string                      `json:"error,omitempty"`
	Metadata              json.RawMessage             `json:"metadata,omitempty"`
	Result                json.RawMessage             `json:"result,omitempty"`
}

// TaskRunSummaryPayload is the shared run-chip payload reused by enriched task reads.
type TaskRunSummaryPayload struct {
	ID                    string                      `json:"id"`
	TaskID                string                      `json:"task_id"`
	Status                taskpkg.RunStatus           `json:"status"`
	Attempt               int                         `json:"attempt"`
	PreviousRunID         string                      `json:"previous_run_id,omitempty"`
	FailureKind           string                      `json:"failure_kind,omitempty"`
	MaxAttempts           int                         `json:"max_attempts"`
	SessionID             string                      `json:"session_id,omitempty"`
	ClaimedBy             *taskpkg.ActorIdentity      `json:"claimed_by,omitempty"`
	ClaimTokenHash        string                      `json:"claim_token_hash,omitempty"`
	LeaseUntil            *time.Time                  `json:"lease_until,omitempty"`
	HeartbeatAt           *time.Time                  `json:"heartbeat_at,omitempty"`
	CoordinationChannelID string                      `json:"coordination_channel_id,omitempty"`
	CoordinationChannel   *CoordinationChannelPayload `json:"coordination_channel,omitempty"`
	QueuedAt              time.Time                   `json:"queued_at"`
	ClaimedAt             *time.Time                  `json:"claimed_at,omitempty"`
	StartedAt             *time.Time                  `json:"started_at,omitempty"`
	EndedAt               *time.Time                  `json:"ended_at,omitempty"`
	Error                 string                      `json:"error,omitempty"`
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
	Summary              TaskSummaryPayload               `json:"summary"`
	Task                 TaskPayload                      `json:"task"`
	Children             []TaskSummaryPayload             `json:"children,omitempty"`
	Dependencies         []TaskDependencyPayload          `json:"dependencies,omitempty"`
	DependencyReferences []TaskDependencyReferencePayload `json:"dependency_references,omitempty"`
	Runs                 []TaskRunPayload                 `json:"runs,omitempty"`
	Events               []TaskEventPayload               `json:"events,omitempty"`
}

// TaskTimelineItemPayload is the shared task-timeline response payload.
type TaskTimelineItemPayload struct {
	Sequence  int64                  `json:"sequence"`
	EventID   string                 `json:"event_id"`
	Task      TaskReferencePayload   `json:"task"`
	Run       *TaskRunSummaryPayload `json:"run,omitempty"`
	EventType string                 `json:"event_type"`
	Actor     taskpkg.ActorIdentity  `json:"actor"`
	Origin    taskpkg.Origin         `json:"origin"`
	Payload   json.RawMessage        `json:"payload,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// TaskStreamEventPayload is one task-scoped replayable stream event.
type TaskStreamEventPayload struct {
	Sequence int64                   `json:"sequence"`
	Type     string                  `json:"type"`
	Timeline TaskTimelineItemPayload `json:"timeline"`
}

// TaskTreeNodePayload is one node inside a task-tree live snapshot.
type TaskTreeNodePayload struct {
	Task           TaskReferencePayload   `json:"task"`
	ParentTaskID   string                 `json:"parent_task_id,omitempty"`
	Depth          int                    `json:"depth"`
	ChildCount     int                    `json:"child_count,omitempty"`
	ActiveRun      *TaskRunSummaryPayload `json:"active_run,omitempty"`
	LastActivityAt time.Time              `json:"last_activity_at"`
}

// TaskTreePayload is the shared task-tree live snapshot.
type TaskTreePayload struct {
	Root        TaskTreeNodePayload   `json:"root"`
	Descendants []TaskTreeNodePayload `json:"descendants,omitempty"`
}

// TaskRunSessionPayload links one task run to its backing session when available.
type TaskRunSessionPayload struct {
	SessionID   string    `json:"session_id"`
	WorkspaceID string    `json:"workspace_id,omitempty"`
	AgentName   string    `json:"agent_name,omitempty"`
	Name        string    `json:"name,omitempty"`
	Channel     string    `json:"channel,omitempty"`
	State       string    `json:"state,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TaskRunOperationalSummaryPayload captures aggregated runtime metrics for run detail.
type TaskRunOperationalSummaryPayload struct {
	LastActivityAt time.Time `json:"last_activity_at"`
	LastEventType  string    `json:"last_event_type,omitempty"`
	ToolCallCount  *int64    `json:"tool_call_count,omitempty"`
	TurnCount      *int64    `json:"turn_count,omitempty"`
	InputTokens    *int64    `json:"input_tokens,omitempty"`
	OutputTokens   *int64    `json:"output_tokens,omitempty"`
	TotalTokens    *int64    `json:"total_tokens,omitempty"`
	TotalCost      *float64  `json:"total_cost,omitempty"`
	CostCurrency   *string   `json:"cost_currency,omitempty"`
}

// TaskRunDetailPayload is the shared run-detail response payload.
type TaskRunDetailPayload struct {
	Run     TaskRunPayload                   `json:"run"`
	Task    TaskReferencePayload             `json:"task"`
	Session *TaskRunSessionPayload           `json:"session,omitempty"`
	Summary TaskRunOperationalSummaryPayload `json:"summary"`
}

// TaskInspectRunPayload is the redacted run projection returned by task inspect.
type TaskInspectRunPayload struct {
	RunID                   string            `json:"run_id"`
	TaskID                  string            `json:"task_id"`
	Status                  taskpkg.RunStatus `json:"status"`
	ClaimTokenHashTruncated string            `json:"claim_token_hash_truncated,omitempty"`
	LeaseUntil              *time.Time        `json:"lease_until,omitempty"`
	HeartbeatAt             *time.Time        `json:"heartbeat_at,omitempty"`
	HeartbeatAgeSeconds     *int64            `json:"heartbeat_age_seconds,omitempty"`
	Retries                 int               `json:"retries,omitempty"`
	LastErrorSummary        string            `json:"last_error_summary,omitempty"`
	FailureKind             string            `json:"failure_kind,omitempty"`
	BoundSessionID          string            `json:"bound_session_id,omitempty"`
	StartedAt               *time.Time        `json:"started_at,omitempty"`
	EndedAt                 *time.Time        `json:"ended_at,omitempty"`
	PreviousRunID           string            `json:"previous_run_id,omitempty"`
	QueuedAt                time.Time         `json:"queued_at"`
	Attempt                 int               `json:"attempt"`
}

// TaskInspectSessionPayload is the bound-session projection returned by task inspect.
type TaskInspectSessionPayload struct {
	SessionID      string     `json:"session_id"`
	State          string     `json:"state,omitempty"`
	AgentName      string     `json:"agent_name,omitempty"`
	ProviderName   string     `json:"provider_name,omitempty"`
	WorkspaceID    string     `json:"workspace_id,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	LastActivityAt *time.Time `json:"last_activity_at,omitempty"`
	StopReason     string     `json:"stop_reason,omitempty"`
	FailureKind    string     `json:"failure_kind,omitempty"`
}

// TaskInspectEventPayload is one recent event summary returned by task inspect.
type TaskInspectEventPayload struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	SessionID string    `json:"session_id,omitempty"`
	TaskID    string    `json:"task_id,omitempty"`
	RunID     string    `json:"run_id,omitempty"`
	Outcome   string    `json:"outcome,omitempty"`
	Summary   string    `json:"summary,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// TaskInspectSchedulerPayload is the scheduler state used for inspect diagnostics.
type TaskInspectSchedulerPayload struct {
	Paused    bool       `json:"paused"`
	PausedBy  string     `json:"paused_by,omitempty"`
	PausedAt  *time.Time `json:"paused_at,omitempty"`
	Reason    string     `json:"reason,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// TaskInspectPayload is the shared task/run inspection response payload.
type TaskInspectPayload struct {
	Target       string                      `json:"target"`
	Task         TaskSummaryPayload          `json:"task"`
	CurrentRun   *TaskInspectRunPayload      `json:"current_run,omitempty"`
	BoundSession *TaskInspectSessionPayload  `json:"bound_session,omitempty"`
	RecentRuns   []TaskInspectRunPayload     `json:"recent_runs,omitempty"`
	RecentEvents []TaskInspectEventPayload   `json:"recent_events,omitempty"`
	Scheduler    TaskInspectSchedulerPayload `json:"scheduler"`
	Diagnostics  []DiagnosticItem            `json:"diagnostics,omitempty"`
	NextAction   string                      `json:"next_action"`
	AsOf         time.Time                   `json:"as_of"`
}

// TaskInspectResponse wraps one task/run inspect snapshot.
type TaskInspectResponse struct {
	Inspect TaskInspectPayload `json:"inspect"`
}

// TaskDashboardPayload is the observer-backed task dashboard response payload.
type TaskDashboardPayload struct {
	Totals          TaskDashboardTotalsPayload            `json:"totals"`
	Cards           TaskDashboardCardsPayload             `json:"cards"`
	StatusBreakdown []TaskDashboardStatusBreakdownPayload `json:"status_breakdown,omitempty"`
	Queue           TaskDashboardQueuePayload             `json:"queue"`
	Health          TaskDashboardHealthPayload            `json:"health"`
	ActiveRuns      TaskDashboardActiveRunsPayload        `json:"active_runs"`
	Freshness       TaskDashboardFreshnessPayload         `json:"freshness"`
}

// TaskDashboardTotalsPayload collapses current task and run totals into operator-facing counters.
type TaskDashboardTotalsPayload struct {
	TasksTotal             int `json:"tasks_total"`
	RunsTotal              int `json:"runs_total"`
	DraftTasks             int `json:"draft_tasks"`
	PendingTasks           int `json:"pending_tasks"`
	ReadyTasks             int `json:"ready_tasks"`
	InProgressTasks        int `json:"in_progress_tasks"`
	BlockedTasks           int `json:"blocked_tasks"`
	CompletedTasks         int `json:"completed_tasks"`
	FailedTasks            int `json:"failed_tasks"`
	CanceledTasks          int `json:"canceled_tasks"`
	AwaitingApprovalTasks  int `json:"awaiting_approval_tasks"`
	DependencyBlockedTasks int `json:"dependency_blocked_tasks"`
	QueuedRuns             int `json:"queued_runs"`
	ClaimedRuns            int `json:"claimed_runs"`
	StartingRuns           int `json:"starting_runs"`
	RunningRuns            int `json:"running_runs"`
	CompletedRuns          int `json:"completed_runs"`
	FailedRuns             int `json:"failed_runs"`
	CanceledRuns           int `json:"canceled_runs"`
	ActiveRuns             int `json:"active_runs"`
}

// TaskDashboardCardsPayload exposes dashboard-ready card values.
type TaskDashboardCardsPayload struct {
	InProgress TaskDashboardInProgressCardPayload `json:"in_progress"`
	Blocked    TaskDashboardBlockedCardPayload    `json:"blocked"`
	Failed     TaskDashboardFailedCardPayload     `json:"failed"`
	Latency    TaskDashboardLatencyCardPayload    `json:"latency"`
}

// TaskDashboardInProgressCardPayload summarizes active work and live-run pressure.
type TaskDashboardInProgressCardPayload struct {
	Tasks        int    `json:"tasks"`
	ActiveRuns   int    `json:"active_runs"`
	RunningRuns  int    `json:"running_runs"`
	StartingRuns int    `json:"starting_runs"`
	ClaimedRuns  int    `json:"claimed_runs"`
	QueuedRuns   int    `json:"queued_runs"`
	HealthStatus string `json:"health_status"`
}

// TaskDashboardBlockedCardPayload summarizes approval and dependency pressure.
type TaskDashboardBlockedCardPayload struct {
	Tasks                int    `json:"tasks"`
	AwaitingApproval     int    `json:"awaiting_approval"`
	AwaitingDependencies int    `json:"awaiting_dependencies"`
	HealthStatus         string `json:"health_status"`
}

// TaskDashboardFailedCardPayload summarizes failed work and disruptive run outcomes.
type TaskDashboardFailedCardPayload struct {
	Tasks        int    `json:"tasks"`
	FailedRuns   int    `json:"failed_runs"`
	ForcedStops  int    `json:"forced_stops"`
	HealthStatus string `json:"health_status"`
}

// TaskLatencyMetricPayload exposes one task latency metric family.
type TaskLatencyMetricPayload struct {
	Samples       int   `json:"samples"`
	AverageMillis int64 `json:"average_ms"`
	MaximumMillis int64 `json:"maximum_ms"`
}

// TaskDashboardLatencyCardPayload exposes queue and start latency summaries.
type TaskDashboardLatencyCardPayload struct {
	ClaimLatencyMillis TaskLatencyMetricPayload `json:"claim_latency_ms"`
	StartLatencyMillis TaskLatencyMetricPayload `json:"start_latency_ms"`
}

// TaskDashboardStatusBreakdownPayload reports one aggregated task-status bucket.
type TaskDashboardStatusBreakdownPayload struct {
	Status       taskpkg.Status `json:"status"`
	Count        int            `json:"count"`
	SharePercent int            `json:"share_percent"`
}

// TaskDashboardQueuePayload reports current queue backlog state.
type TaskDashboardQueuePayload struct {
	Total                 int                              `json:"total"`
	Depth                 []TaskDashboardQueueDepthPayload `json:"depth,omitempty"`
	OldestQueuedAt        time.Time                        `json:"oldest_queued_at"`
	OldestQueueAgeMilli   int64                            `json:"oldest_queue_age_ms"`
	BacklogWarning        bool                             `json:"backlog_warning"`
	BacklogStatus         string                           `json:"backlog_status"`
	BacklogThresholdMilli int64                            `json:"backlog_threshold_ms"`
}

// TaskDashboardQueueDepthPayload reports queued work by channel.
type TaskDashboardQueueDepthPayload struct {
	NetworkChannel      string    `json:"network_channel,omitempty"`
	Count               int       `json:"count"`
	OldestQueuedAt      time.Time `json:"oldest_queued_at"`
	OldestQueueAgeMilli int64     `json:"oldest_queue_age_ms"`
}

// TaskDashboardHealthPayload exposes warning-oriented dashboard health indicators.
type TaskDashboardHealthPayload struct {
	Status           string `json:"status"`
	StuckRuns        int    `json:"stuck_runs"`
	ActiveOrphanRuns int    `json:"active_orphan_runs"`
	QueueBacklog     bool   `json:"queue_backlog"`
}

// TaskDashboardActiveRunsPayload summarizes the currently active run set and recent cards.
type TaskDashboardActiveRunsPayload struct {
	Total    int                             `json:"total"`
	Running  int                             `json:"running"`
	Starting int                             `json:"starting"`
	Claimed  int                             `json:"claimed"`
	Queued   int                             `json:"queued"`
	Items    []TaskDashboardActiveRunPayload `json:"items,omitempty"`
}

// TaskDashboardActiveRunPayload exposes one recent active-run card payload.
type TaskDashboardActiveRunPayload struct {
	TaskID         string             `json:"task_id"`
	TaskIdentifier string             `json:"task_identifier,omitempty"`
	TaskTitle      string             `json:"task_title"`
	TaskStatus     taskpkg.Status     `json:"task_status"`
	TaskPriority   taskpkg.Priority   `json:"task_priority,omitempty"`
	TaskOwner      *taskpkg.Ownership `json:"task_owner,omitempty"`
	Scope          taskpkg.Scope      `json:"scope"`
	WorkspaceID    string             `json:"workspace_id,omitempty"`
	LatestEventSeq int64              `json:"latest_event_seq"`
	RunID          string             `json:"run_id"`
	RunStatus      taskpkg.RunStatus  `json:"run_status"`
	Attempt        int                `json:"attempt"`
	MaxAttempts    int                `json:"max_attempts"`
	SessionID      string             `json:"session_id,omitempty"`
	NetworkChannel string             `json:"network_channel,omitempty"`
	LastActivityAt time.Time          `json:"last_activity_at"`
	AgeMilli       int64              `json:"age_ms"`
	HealthStatus   string             `json:"health_status"`
	Stuck          bool               `json:"stuck"`
	Error          string             `json:"error,omitempty"`
}

// TaskDashboardFreshnessPayload exposes recency and stale-warning state for the dashboard snapshot.
type TaskDashboardFreshnessPayload struct {
	ObservedAt       time.Time `json:"observed_at"`
	LatestActivityAt time.Time `json:"latest_activity_at"`
	AgeMilli         int64     `json:"age_ms"`
	StaleAfterMilli  int64     `json:"stale_after_ms"`
	HasLiveWork      bool      `json:"has_live_work"`
	Status           string    `json:"status"`
	Stale            bool      `json:"stale"`
}

// TaskTriageStatePayload is the shared actor-scoped task triage state.
type TaskTriageStatePayload struct {
	TaskID             string                `json:"task_id"`
	Actor              taskpkg.ActorIdentity `json:"actor"`
	Read               bool                  `json:"read"`
	Archived           bool                  `json:"archived"`
	Dismissed          bool                  `json:"dismissed"`
	LastSeenActivityAt *time.Time            `json:"last_seen_activity_at,omitempty"`
	UpdatedAt          time.Time             `json:"updated_at"`
}

// TaskInboxLane identifies one inbox grouping lane.
type TaskInboxLane string

const (
	// TaskInboxLaneMyWork identifies directly assigned or actively owned work.
	TaskInboxLaneMyWork TaskInboxLane = "my_work"
	// TaskInboxLaneApprovals identifies approval-gated work awaiting a decision.
	TaskInboxLaneApprovals TaskInboxLane = "approvals"
	// TaskInboxLaneFailedRuns identifies items whose latest execution failed.
	TaskInboxLaneFailedRuns TaskInboxLane = "failed_runs"
	// TaskInboxLaneBlocked identifies blocked work that is not waiting for approval.
	TaskInboxLaneBlocked TaskInboxLane = "blocked"
	// TaskInboxLaneArchived identifies items archived by the current actor context.
	TaskInboxLaneArchived TaskInboxLane = "archived"
)

// TaskInboxItemPayload is one task inbox item with action-ready metadata.
type TaskInboxItemPayload struct {
	Task             TaskReferencePayload   `json:"task"`
	Lane             TaskInboxLane          `json:"lane"`
	ApprovalPolicy   taskpkg.ApprovalPolicy `json:"approval_policy,omitempty"`
	ApprovalState    taskpkg.ApprovalState  `json:"approval_state,omitempty"`
	BlockingReason   string                 `json:"blocking_reason,omitempty"`
	LatestActivityAt time.Time              `json:"latest_activity_at"`
	Run              *TaskRunSummaryPayload `json:"run,omitempty"`
	Triage           TaskTriageStatePayload `json:"triage"`
}

// TaskInboxLaneGroupPayload groups inbox items by lane.
type TaskInboxLaneGroupPayload struct {
	Lane        TaskInboxLane          `json:"lane"`
	Count       int                    `json:"count"`
	UnreadCount int                    `json:"unread_count"`
	Items       []TaskInboxItemPayload `json:"items,omitempty"`
}

// TaskInboxPayload is the observer-backed task inbox response payload.
type TaskInboxPayload struct {
	Total         int                         `json:"total"`
	UnreadTotal   int                         `json:"unread_total"`
	ArchivedTotal int                         `json:"archived_total"`
	Groups        []TaskInboxLaneGroupPayload `json:"groups,omitempty"`
}

// TaskListQuery captures the shared task list filters.
type TaskListQuery struct {
	Scope          taskpkg.Scope         `json:"scope,omitempty"`
	Workspace      string                `json:"workspace,omitempty"`
	Status         taskpkg.Status        `json:"status,omitempty"`
	Priority       taskpkg.Priority      `json:"priority,omitempty"`
	IncludeDrafts  bool                  `json:"include_drafts,omitempty"`
	ApprovalState  taskpkg.ApprovalState `json:"approval_state,omitempty"`
	OwnerKind      taskpkg.OwnerKind     `json:"owner_kind,omitempty"`
	OwnerRef       string                `json:"owner_ref,omitempty"`
	ParentTaskID   string                `json:"parent_task_id,omitempty"`
	NetworkChannel string                `json:"network_channel,omitempty"`
	Query          string                `json:"query,omitempty"`
	Limit          int                   `json:"limit,omitempty"`
}

// TaskRunListQuery captures the shared task-run list filters.
type TaskRunListQuery struct {
	Status    taskpkg.RunStatus `json:"status,omitempty"`
	SessionID string            `json:"session_id,omitempty"`
	Limit     int               `json:"limit,omitempty"`
}

// TaskTimelineQuery captures the shared task timeline filters.
type TaskTimelineQuery struct {
	AfterSequence int64 `json:"after_sequence,omitempty"`
	Limit         int   `json:"limit,omitempty"`
}

// TaskStreamQuery captures the shared task stream replay filters.
type TaskStreamQuery struct {
	AfterSequence int64 `json:"after_sequence,omitempty"`
}

// TaskDashboardQuery captures the shared observer-backed task dashboard filters.
type TaskDashboardQuery struct {
	Scope          taskpkg.Scope      `json:"scope,omitempty"`
	Workspace      string             `json:"workspace,omitempty"`
	OwnerKind      taskpkg.OwnerKind  `json:"owner_kind,omitempty"`
	OwnerRef       string             `json:"owner_ref,omitempty"`
	NetworkChannel string             `json:"network_channel,omitempty"`
	OriginKind     taskpkg.OriginKind `json:"origin_kind,omitempty"`
}

// TaskInboxQuery captures the shared observer-backed task inbox filters.
type TaskInboxQuery struct {
	Scope     taskpkg.Scope     `json:"scope,omitempty"`
	Workspace string            `json:"workspace,omitempty"`
	OwnerKind taskpkg.OwnerKind `json:"owner_kind,omitempty"`
	OwnerRef  string            `json:"owner_ref,omitempty"`
	Lane      TaskInboxLane     `json:"lane,omitempty"`
	Unread    bool              `json:"unread,omitempty"`
	Query     string            `json:"query,omitempty"`
	Limit     int               `json:"limit,omitempty"`
}

// CreateTaskRequest is the shared task-create request payload.
type CreateTaskRequest struct {
	ID                 string                 `json:"id,omitempty"`
	Identifier         string                 `json:"identifier,omitempty"`
	Scope              taskpkg.Scope          `json:"scope"`
	Workspace          string                 `json:"workspace,omitempty"`
	NetworkChannel     string                 `json:"network_channel,omitempty"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description,omitempty"`
	Priority           taskpkg.Priority       `json:"priority,omitempty"`
	MaxAttempts        *int                   `json:"max_attempts,omitempty"`
	AutoEnqueueOnReady bool                   `json:"auto_enqueue_on_ready,omitempty"`
	Draft              bool                   `json:"draft,omitempty"`
	ApprovalPolicy     taskpkg.ApprovalPolicy `json:"approval_policy,omitempty"`
	Owner              *taskpkg.Ownership     `json:"owner,omitempty"`
	Metadata           json.RawMessage        `json:"metadata,omitempty"`
}

// CreateTaskChildRequest is the shared child-task create payload.
type CreateTaskChildRequest struct {
	ID                 string                 `json:"id,omitempty"`
	Identifier         string                 `json:"identifier,omitempty"`
	Scope              taskpkg.Scope          `json:"scope"`
	Workspace          string                 `json:"workspace,omitempty"`
	NetworkChannel     string                 `json:"network_channel,omitempty"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description,omitempty"`
	Priority           taskpkg.Priority       `json:"priority,omitempty"`
	MaxAttempts        *int                   `json:"max_attempts,omitempty"`
	AutoEnqueueOnReady bool                   `json:"auto_enqueue_on_ready,omitempty"`
	Draft              bool                   `json:"draft,omitempty"`
	ApprovalPolicy     taskpkg.ApprovalPolicy `json:"approval_policy,omitempty"`
	Owner              *taskpkg.Ownership     `json:"owner,omitempty"`
	Metadata           json.RawMessage        `json:"metadata,omitempty"`
}

// UpdateTaskRequest is the shared task patch payload.
type UpdateTaskRequest struct {
	Title              *string                 `json:"title,omitempty"`
	Description        *string                 `json:"description,omitempty"`
	Priority           *taskpkg.Priority       `json:"priority,omitempty"`
	MaxAttempts        *int                    `json:"max_attempts,omitempty"`
	AutoEnqueueOnReady *bool                   `json:"auto_enqueue_on_ready,omitempty"`
	ApprovalPolicy     *taskpkg.ApprovalPolicy `json:"approval_policy,omitempty"`
	Metadata           *json.RawMessage        `json:"metadata,omitempty"`
	NetworkChannel     *string                 `json:"network_channel,omitempty"`
	Owner              *taskpkg.Ownership      `json:"owner,omitempty"`
	ClearOwner         bool                    `json:"clear_owner,omitempty"`
}

// HasChanges reports whether the patch includes any mutable task field.
func (r UpdateTaskRequest) HasChanges() bool {
	return r.Title != nil ||
		r.Description != nil ||
		r.Priority != nil ||
		r.MaxAttempts != nil ||
		r.AutoEnqueueOnReady != nil ||
		r.ApprovalPolicy != nil ||
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
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	NetworkChannel string          `json:"network_channel,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

// TaskExecutionRequest is the shared task publish/start/approval execution payload.
type TaskExecutionRequest struct {
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	NetworkChannel string          `json:"network_channel,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
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

// ForceReleaseTaskRunRequest is the shared force-release request payload.
type ForceReleaseTaskRunRequest struct {
	Reason   string          `json:"reason,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// ForceFailTaskRunRequest is the shared forced-failure request payload.
type ForceFailTaskRunRequest struct {
	Reason   string          `json:"reason"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// RetryTaskRunRequest is the shared retry request payload.
type RetryTaskRunRequest struct {
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// RecoverTaskRunRequest is the shared recovery request payload for a needs_attention run.
type RecoverTaskRunRequest struct {
	Reason   string          `json:"reason,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// PauseTaskRequest captures one per-task pause request.
type PauseTaskRequest struct {
	Reason   string          `json:"reason"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// ResumeTaskRequest captures one per-task resume request.
type ResumeTaskRequest struct {
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// BulkForceTaskRunRequest is the shared bounded bulk force-operation payload.
type BulkForceTaskRunRequest struct {
	RunIDs   []string        `json:"run_ids"`
	Reason   string          `json:"reason,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// BulkForceTaskRunItemPayload records one per-row bulk force-operation outcome.
type BulkForceTaskRunItemPayload struct {
	RunID string          `json:"run_id"`
	OK    bool            `json:"ok"`
	Run   *TaskRunPayload `json:"run,omitempty"`
	Error *ErrorPayload   `json:"error,omitempty"`
}

// CancelTaskRunRequest is the shared run-cancel request payload.
type CancelTaskRunRequest struct {
	Reason   string          `json:"reason,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}
