package observe

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

const (
	taskIngressAuditEnqueueAction = "task.run.enqueue"
	taskIngressChannelMismatch    = "channel_mismatch"
	taskEventCanceled             = "task.canceled"
	taskEventRunEnqueued          = "task.run_enqueued"
	taskEventRunForceStopped      = "task.run_force_stopped"
	taskEventRunRecovered         = "task.run_recovered"
	taskHealthStatusOK            = "ok"
	taskHealthStatusWarn          = "warn"
)

// TaskSummaryQuery filters the current task summary view.
type TaskSummaryQuery struct {
	Scope          taskpkg.Scope      `json:"scope,omitempty"`
	WorkspaceID    string             `json:"workspace_id,omitempty"`
	OwnerKind      taskpkg.OwnerKind  `json:"owner_kind,omitempty"`
	OwnerRef       string             `json:"owner_ref,omitempty"`
	NetworkChannel string             `json:"network_channel,omitempty"`
	OriginKind     taskpkg.OriginKind `json:"origin_kind,omitempty"`
	Search         string             `json:"search,omitempty"`
}

// Validate ensures the summary query uses supported filters.
func (q TaskSummaryQuery) Validate() error {
	if q.Scope.Normalize() != "" {
		if err := q.Scope.Validate("task_summary_query.scope"); err != nil {
			return err
		}
	}
	if q.OwnerKind.Normalize() != "" {
		if err := q.OwnerKind.Validate("task_summary_query.owner_kind"); err != nil {
			return err
		}
	}
	if q.OriginKind.Normalize() != "" {
		if err := q.OriginKind.Validate("task_summary_query.origin_kind"); err != nil {
			return err
		}
	}
	return nil
}

// TaskMetricsQuery filters audit-derived metrics and current queue metrics.
type TaskMetricsQuery struct {
	Since          time.Time          `json:"since"`
	NetworkChannel string             `json:"network_channel,omitempty"`
	OriginKind     taskpkg.OriginKind `json:"origin_kind,omitempty"`
}

// Validate ensures the metrics query uses supported filters.
func (q TaskMetricsQuery) Validate() error {
	if q.OriginKind.Normalize() != "" {
		if err := q.OriginKind.Validate("task_metrics_query.origin_kind"); err != nil {
			return err
		}
	}
	return nil
}

// TaskDashboardQuery filters the observer-backed task dashboard read model.
type TaskDashboardQuery struct {
	Scope          taskpkg.Scope      `json:"scope,omitempty"`
	WorkspaceID    string             `json:"workspace_id,omitempty"`
	OwnerKind      taskpkg.OwnerKind  `json:"owner_kind,omitempty"`
	OwnerRef       string             `json:"owner_ref,omitempty"`
	NetworkChannel string             `json:"network_channel,omitempty"`
	OriginKind     taskpkg.OriginKind `json:"origin_kind,omitempty"`
}

// Validate ensures the dashboard query uses supported filters.
func (q TaskDashboardQuery) Validate() error {
	if q.Scope.Normalize() != "" {
		if err := taskpkg.ValidateScopeBinding(
			q.Scope,
			q.WorkspaceID,
			"task_dashboard_query",
			"workspace_id",
		); err != nil {
			return err
		}
	}
	return q.summaryQuery().Validate()
}

func (q TaskDashboardQuery) summaryQuery() TaskSummaryQuery {
	return TaskSummaryQuery{
		Scope:          q.Scope,
		WorkspaceID:    q.WorkspaceID,
		OwnerKind:      q.OwnerKind,
		OwnerRef:       q.OwnerRef,
		NetworkChannel: q.NetworkChannel,
		OriginKind:     q.OriginKind,
	}
}

func (q TaskDashboardQuery) metricsQuery(since time.Time) TaskMetricsQuery {
	return TaskMetricsQuery{
		Since:          since,
		NetworkChannel: q.NetworkChannel,
		OriginKind:     q.OriginKind,
	}
}

// TaskStatusTotal reports one current task-count bucket.
type TaskStatusTotal struct {
	Scope          taskpkg.Scope  `json:"scope"`
	Status         taskpkg.Status `json:"status"`
	NetworkChannel string         `json:"network_channel,omitempty"`
	Count          int            `json:"count"`
}

// TaskOriginTotal reports one current task-origin bucket.
type TaskOriginTotal struct {
	OriginKind     taskpkg.OriginKind `json:"origin_kind"`
	NetworkChannel string             `json:"network_channel,omitempty"`
	Count          int                `json:"count"`
}

// TaskRunTotal reports one current task-run bucket.
type TaskRunTotal struct {
	Status         taskpkg.RunStatus  `json:"status"`
	OriginKind     taskpkg.OriginKind `json:"origin_kind"`
	NetworkChannel string             `json:"network_channel,omitempty"`
	Count          int                `json:"count"`
}

// TaskOwnerTotal reports one current ownership bucket.
type TaskOwnerTotal struct {
	OwnerKind taskpkg.OwnerKind `json:"owner_kind"`
	OwnerRef  string            `json:"owner_ref"`
	Count     int               `json:"count"`
}

// TaskQueueDepth reports queued work by channel.
type TaskQueueDepth struct {
	NetworkChannel      string    `json:"network_channel,omitempty"`
	Count               int       `json:"count"`
	OldestQueuedAt      time.Time `json:"oldest_queued_at"`
	OldestQueueAgeMilli int64     `json:"oldest_queue_age_ms"`
}

// Summary exposes the current read-side task summary buckets.
type Summary struct {
	TotalTasks  int               `json:"total_tasks"`
	TotalRuns   int               `json:"total_runs"`
	TaskTotals  []TaskStatusTotal `json:"task_totals,omitempty"`
	TaskOrigins []TaskOriginTotal `json:"task_origins,omitempty"`
	RunTotals   []TaskRunTotal    `json:"run_totals,omitempty"`
	OwnerTotals []TaskOwnerTotal  `json:"owner_totals,omitempty"`
	QueueDepth  []TaskQueueDepth  `json:"queue_depth,omitempty"`
}

// LatencyMetric summarizes one task-run latency family in milliseconds.
type LatencyMetric struct {
	Samples       int   `json:"samples"`
	AverageMillis int64 `json:"average_ms"`
	MaximumMillis int64 `json:"maximum_ms"`
}

// TaskCancelRequestTotal reports cancellation requests grouped by origin.
type TaskCancelRequestTotal struct {
	OriginKind taskpkg.OriginKind `json:"origin_kind"`
	Count      int                `json:"count"`
}

// TaskRecoveryTotals reports boot-recovery outcomes grouped by manager action.
type TaskRecoveryTotals struct {
	Requeued      int `json:"requeued"`
	MarkedRunning int `json:"marked_running"`
	Failed        int `json:"failed"`
}

// TaskMetrics exposes current counters and latency summaries for the task domain.
type TaskMetrics struct {
	TasksTotal              []TaskStatusTotal        `json:"tasks_total,omitempty"`
	TaskRunsTotal           []TaskRunTotal           `json:"task_runs_total,omitempty"`
	TaskQueueDepth          []TaskQueueDepth         `json:"task_queue_depth,omitempty"`
	TaskCancelRequestsTotal []TaskCancelRequestTotal `json:"task_cancel_requests_total,omitempty"`
	TaskForcedStopsTotal    int                      `json:"task_forced_stops_total"`
	TaskClaimLatencyMillis  LatencyMetric            `json:"task_claim_latency_ms"`
	TaskStartLatencyMillis  LatencyMetric            `json:"task_start_latency_ms"`
	DuplicateIngressTotal   int                      `json:"duplicate_ingress_total"`
	ChannelMismatchTotal    int                      `json:"channel_mismatch_total"`
	RecoveryTotals          TaskRecoveryTotals       `json:"recovery_totals"`
}

// StuckTaskRun reports one run that exceeded the configured claimed/starting/running threshold.
type StuckTaskRun struct {
	TaskID         string             `json:"task_id"`
	RunID          string             `json:"run_id"`
	Status         taskpkg.RunStatus  `json:"status"`
	OriginKind     taskpkg.OriginKind `json:"origin_kind"`
	NetworkChannel string             `json:"network_channel,omitempty"`
	SessionID      string             `json:"session_id,omitempty"`
	AgeMillis      int64              `json:"age_ms"`
}

// TaskHealth exposes the current operational task-health view.
type TaskHealth struct {
	Status                     string             `json:"status"`
	QueueDepthTotal            int                `json:"queue_depth_total"`
	OldestQueuedAt             time.Time          `json:"oldest_queued_at"`
	OldestQueueAgeMilli        int64              `json:"oldest_queue_age_ms"`
	QueueDepth                 []TaskQueueDepth   `json:"queue_depth,omitempty"`
	StuckRuns                  []StuckTaskRun     `json:"stuck_runs,omitempty"`
	ActiveOrphanRuns           int                `json:"active_orphan_runs"`
	TaskTotals                 []TaskStatusTotal  `json:"task_totals,omitempty"`
	RunTotals                  []TaskRunTotal     `json:"run_totals,omitempty"`
	OwnerTotals                []TaskOwnerTotal   `json:"owner_totals,omitempty"`
	ForcedStopsSinceStart      int                `json:"forced_stops_since_start"`
	DuplicateIngressSinceStart int                `json:"duplicate_ingress_since_start"`
	ChannelMismatchSinceStart  int                `json:"channel_mismatch_since_start"`
	RecoverySinceStart         TaskRecoveryTotals `json:"recovery_since_start"`
}

// TaskDashboardView exposes the observer-owned aggregate payload for the Paper task dashboard.
type TaskDashboardView struct {
	Totals          TaskDashboardTotals            `json:"totals"`
	Cards           TaskDashboardCards             `json:"cards"`
	StatusBreakdown []TaskDashboardStatusBreakdown `json:"status_breakdown,omitempty"`
	Queue           TaskDashboardQueue             `json:"queue"`
	Health          TaskDashboardHealth            `json:"health"`
	ActiveRuns      TaskDashboardActiveRuns        `json:"active_runs"`
	Freshness       TaskDashboardFreshness         `json:"freshness"`
}

// TaskDashboardTotals collapses the current task and run totals into chart-friendly counters.
type TaskDashboardTotals struct {
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

// TaskDashboardCards exposes dashboard-ready card values without leaking raw summary buckets.
type TaskDashboardCards struct {
	InProgress TaskDashboardInProgressCard `json:"in_progress"`
	Blocked    TaskDashboardBlockedCard    `json:"blocked"`
	Failed     TaskDashboardFailedCard     `json:"failed"`
	Latency    TaskDashboardLatencyCard    `json:"latency"`
}

// TaskDashboardInProgressCard summarizes active work and live run pressure.
type TaskDashboardInProgressCard struct {
	Tasks        int    `json:"tasks"`
	ActiveRuns   int    `json:"active_runs"`
	RunningRuns  int    `json:"running_runs"`
	StartingRuns int    `json:"starting_runs"`
	ClaimedRuns  int    `json:"claimed_runs"`
	QueuedRuns   int    `json:"queued_runs"`
	HealthStatus string `json:"health_status"`
}

// TaskDashboardBlockedCard summarizes blocked work and approval/dependency pressure.
type TaskDashboardBlockedCard struct {
	Tasks                int    `json:"tasks"`
	AwaitingApproval     int    `json:"awaiting_approval"`
	AwaitingDependencies int    `json:"awaiting_dependencies"`
	HealthStatus         string `json:"health_status"`
}

// TaskDashboardFailedCard summarizes failed work and disruptive run outcomes.
type TaskDashboardFailedCard struct {
	Tasks        int    `json:"tasks"`
	FailedRuns   int    `json:"failed_runs"`
	ForcedStops  int    `json:"forced_stops"`
	HealthStatus string `json:"health_status"`
}

// TaskDashboardLatencyCard exposes current run-queue latency summaries for operator cards.
type TaskDashboardLatencyCard struct {
	ClaimLatencyMillis LatencyMetric `json:"claim_latency_ms"`
	StartLatencyMillis LatencyMetric `json:"start_latency_ms"`
}

// TaskDashboardStatusBreakdown reports one aggregated task status bucket for chart rendering.
type TaskDashboardStatusBreakdown struct {
	Status       taskpkg.Status `json:"status"`
	Count        int            `json:"count"`
	SharePercent int            `json:"share_percent"`
}

// TaskDashboardQueue reports backlog state for queued task work.
type TaskDashboardQueue struct {
	Total                 int              `json:"total"`
	Depth                 []TaskQueueDepth `json:"depth,omitempty"`
	OldestQueuedAt        time.Time        `json:"oldest_queued_at"`
	OldestQueueAgeMilli   int64            `json:"oldest_queue_age_ms"`
	BacklogWarning        bool             `json:"backlog_warning"`
	BacklogStatus         string           `json:"backlog_status"`
	BacklogThresholdMilli int64            `json:"backlog_threshold_ms"`
}

// TaskDashboardHealth reports warning-oriented dashboard health indicators.
type TaskDashboardHealth struct {
	Status           string `json:"status"`
	StuckRuns        int    `json:"stuck_runs"`
	ActiveOrphanRuns int    `json:"active_orphan_runs"`
	QueueBacklog     bool   `json:"queue_backlog"`
}

// TaskDashboardActiveRuns summarizes the currently active run set and exposes recent cards.
type TaskDashboardActiveRuns struct {
	Total    int                      `json:"total"`
	Running  int                      `json:"running"`
	Starting int                      `json:"starting"`
	Claimed  int                      `json:"claimed"`
	Queued   int                      `json:"queued"`
	Items    []TaskDashboardActiveRun `json:"items,omitempty"`
}

// TaskDashboardActiveRun exposes one recent active-run card payload.
type TaskDashboardActiveRun struct {
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

// TaskDashboardFreshness exposes the recency and stale-warning state of the dashboard snapshot.
type TaskDashboardFreshness struct {
	ObservedAt       time.Time `json:"observed_at"`
	LatestActivityAt time.Time `json:"latest_activity_at"`
	AgeMilli         int64     `json:"age_ms"`
	StaleAfterMilli  int64     `json:"stale_after_ms"`
	HasLiveWork      bool      `json:"has_live_work"`
	Status           string    `json:"status"`
	Stale            bool      `json:"stale"`
}

// TaskInboxLane identifies one observer-backed inbox grouping.
type TaskInboxLane string

const (
	// TaskInboxLaneMyWork identifies directly assigned or actively owned work.
	TaskInboxLaneMyWork TaskInboxLane = "my_work"
	// TaskInboxLaneApprovals identifies approval-gated work awaiting a decision.
	TaskInboxLaneApprovals TaskInboxLane = "approvals"
	// TaskInboxLaneFailedRuns identifies tasks whose latest execution failed.
	TaskInboxLaneFailedRuns TaskInboxLane = "failed_runs"
	// TaskInboxLaneBlocked identifies blocked tasks that are not awaiting approval.
	TaskInboxLaneBlocked TaskInboxLane = "blocked"
	// TaskInboxLaneArchived identifies tasks archived by the current actor.
	TaskInboxLaneArchived TaskInboxLane = "archived"
)

const (
	taskInboxBlockingReasonAwaitingApproval = "awaiting_approval"
	taskInboxBlockingReasonApprovalRejected = "approval_rejected"
	taskInboxBlockingReasonAwaitingDeps     = "awaiting_dependencies"
	taskInboxBlockingReasonLatestRunFailed  = "latest_run_failed"
)

// Normalize returns the normalized inbox lane value.
func (l TaskInboxLane) Normalize() TaskInboxLane {
	return TaskInboxLane(strings.TrimSpace(strings.ToLower(string(l))))
}

// Validate ensures the inbox lane is one of the supported values.
func (l TaskInboxLane) Validate(path string) error {
	switch l.Normalize() {
	case "":
		return nil
	case TaskInboxLaneMyWork,
		TaskInboxLaneApprovals,
		TaskInboxLaneFailedRuns,
		TaskInboxLaneBlocked,
		TaskInboxLaneArchived:
		return nil
	default:
		return fmt.Errorf("%w: %s has unsupported lane %q", taskpkg.ErrValidation, path, l)
	}
}

// TaskInboxQuery filters the observer-backed task inbox read model.
type TaskInboxQuery struct {
	Scope       taskpkg.Scope     `json:"scope,omitempty"`
	WorkspaceID string            `json:"workspace_id,omitempty"`
	OwnerKind   taskpkg.OwnerKind `json:"owner_kind,omitempty"`
	OwnerRef    string            `json:"owner_ref,omitempty"`
	Lane        TaskInboxLane     `json:"lane,omitempty"`
	Unread      bool              `json:"unread,omitempty"`
	Search      string            `json:"search,omitempty"`
	Limit       int               `json:"limit,omitempty"`
}

// Validate ensures the inbox query uses supported filters.
func (q TaskInboxQuery) Validate() error {
	if q.Scope.Normalize() != "" {
		if err := taskpkg.ValidateScopeBinding(q.Scope, q.WorkspaceID, "task_inbox_query", "workspace_id"); err != nil {
			return err
		}
	}
	if q.OwnerKind.Normalize() != "" {
		if err := q.OwnerKind.Validate("task_inbox_query.owner_kind"); err != nil {
			return err
		}
	}
	if err := q.Lane.Validate("task_inbox_query.lane"); err != nil {
		return err
	}
	if q.Limit < 0 {
		return fmt.Errorf("%w: task_inbox_query.limit must be zero or positive: %d", taskpkg.ErrValidation, q.Limit)
	}
	return nil
}

func (q TaskInboxQuery) summaryQuery() TaskSummaryQuery {
	return TaskSummaryQuery{
		Scope:       q.Scope,
		WorkspaceID: q.WorkspaceID,
		OwnerKind:   q.OwnerKind,
		OwnerRef:    q.OwnerRef,
		Search:      q.Search,
	}
}

// TaskInboxItem is one triage-oriented task inbox item with action-ready metadata.
type TaskInboxItem struct {
	Task             taskpkg.Reference      `json:"task"`
	Lane             TaskInboxLane          `json:"lane"`
	ApprovalPolicy   taskpkg.ApprovalPolicy `json:"approval_policy,omitempty"`
	ApprovalState    taskpkg.ApprovalState  `json:"approval_state,omitempty"`
	BlockingReason   string                 `json:"blocking_reason,omitempty"`
	LatestActivityAt time.Time              `json:"latest_activity_at"`
	Run              *taskpkg.RunSummary    `json:"run,omitempty"`
	Triage           taskpkg.TriageState    `json:"triage"`
}

// TaskInboxLaneGroup groups inbox items by lane while preserving full counts.
type TaskInboxLaneGroup struct {
	Lane        TaskInboxLane   `json:"lane"`
	Count       int             `json:"count"`
	UnreadCount int             `json:"unread_count"`
	Items       []TaskInboxItem `json:"items,omitempty"`
}

// TaskInboxView exposes the observer-backed aggregate payload for the Paper inbox.
type TaskInboxView struct {
	Total         int                  `json:"total"`
	UnreadTotal   int                  `json:"unread_total"`
	ArchivedTotal int                  `json:"archived_total"`
	Groups        []TaskInboxLaneGroup `json:"groups,omitempty"`
}

type taskSnapshot struct {
	tasks     []taskpkg.Summary
	runs      []taskpkg.Run
	events    []taskpkg.Event
	audits    []store.NetworkAuditEntry
	tasksByID map[string]taskpkg.Summary
	runsByID  map[string]taskpkg.Run
}

type taskRecoveryPayload struct {
	Action taskpkg.RunBootRecoveryAction `json:"action"`
}

// QueryTaskSummary returns the current task summary buckets filtered by the supplied view.
func (o *Observer) QueryTaskSummary(ctx context.Context, query TaskSummaryQuery) (Summary, error) {
	snapshot, err := o.loadTaskSnapshot(ctx, query)
	if err != nil {
		return Summary{}, err
	}
	return taskSummaryFromSnapshot(snapshot, o.now), nil
}

// QueryTaskMetrics returns task-domain counters and latency summaries derived from durable state and audit rows.
func (o *Observer) QueryTaskMetrics(ctx context.Context, query TaskMetricsQuery) (TaskMetrics, error) {
	if ctx == nil {
		return TaskMetrics{}, errors.New("observe: task metrics context is required")
	}
	if err := query.Validate(); err != nil {
		return TaskMetrics{}, err
	}

	snapshot, err := o.loadTaskSnapshot(ctx, TaskSummaryQuery{NetworkChannel: query.NetworkChannel})
	if err != nil {
		return TaskMetrics{}, err
	}
	return taskMetricsFromSnapshot(snapshot, query, o.now), nil
}

// QueryTaskDashboard returns the observer-backed aggregate task dashboard view.
func (o *Observer) QueryTaskDashboard(ctx context.Context, query TaskDashboardQuery) (TaskDashboardView, error) {
	if ctx == nil {
		return TaskDashboardView{}, errors.New("observe: task dashboard context is required")
	}
	if err := query.Validate(); err != nil {
		return TaskDashboardView{}, err
	}

	snapshot, err := o.loadTaskSnapshot(ctx, query.summaryQuery())
	if err != nil {
		return TaskDashboardView{}, err
	}

	summary := taskSummaryFromSnapshot(snapshot, o.now)
	metrics := taskMetricsFromSnapshot(snapshot, query.metricsQuery(o.startedAt), o.now)
	health, err := o.taskHealthFromSnapshot(ctx, snapshot, summary, metrics)
	if err != nil {
		return TaskDashboardView{}, err
	}

	return o.taskDashboardFromSnapshot(snapshot, summary, metrics, health), nil
}

// QueryTaskInbox returns the observer-backed aggregate task inbox view for one actor.
func (o *Observer) QueryTaskInbox(
	ctx context.Context,
	query TaskInboxQuery,
	actor taskpkg.ActorIdentity,
) (TaskInboxView, error) {
	if ctx == nil {
		return TaskInboxView{}, errors.New("observe: task inbox context is required")
	}
	if err := query.Validate(); err != nil {
		return TaskInboxView{}, err
	}

	normalizedActor, err := normalizeTaskInboxActor(actor)
	if err != nil {
		return TaskInboxView{}, err
	}

	snapshot, err := o.loadTaskSnapshot(ctx, query.summaryQuery())
	if err != nil {
		return TaskInboxView{}, err
	}
	triageStates, err := o.registry.ListTaskTriageStates(ctx, normalizedActor)
	if err != nil {
		return TaskInboxView{}, fmt.Errorf("observe: list task triage states: %w", err)
	}

	return taskInboxFromSnapshot(snapshot, triageStates, query, normalizedActor), nil
}

func (o *Observer) collectTaskHealth(ctx context.Context) (TaskHealth, error) {
	if ctx == nil {
		return TaskHealth{}, errors.New("observe: task health context is required")
	}

	snapshot, err := o.loadTaskSnapshot(ctx, TaskSummaryQuery{})
	if err != nil {
		return TaskHealth{}, err
	}
	summary := taskSummaryFromSnapshot(snapshot, o.now)
	metrics := taskMetricsFromSnapshot(snapshot, TaskMetricsQuery{Since: o.startedAt}, o.now)
	return o.taskHealthFromSnapshot(ctx, snapshot, summary, metrics)
}

func (o *Observer) taskHealthFromSnapshot(
	ctx context.Context,
	snapshot taskSnapshot,
	summary Summary,
	metrics TaskMetrics,
) (TaskHealth, error) {
	stuckRuns := findStuckRuns(snapshot.runs, o.now(), o.taskHealthConfig)
	sortStuckRuns(stuckRuns)
	activeOrphans, err := o.countActiveOrphanRuns(ctx, snapshot.runs)
	if err != nil {
		return TaskHealth{}, err
	}

	queueDepthTotal := 0
	var oldestQueuedAt time.Time
	var oldestQueuedAge int64
	for _, item := range summary.QueueDepth {
		queueDepthTotal += item.Count
		if item.OldestQueuedAt.IsZero() {
			continue
		}
		if oldestQueuedAt.IsZero() || item.OldestQueuedAt.Before(oldestQueuedAt) {
			oldestQueuedAt = item.OldestQueuedAt
			oldestQueuedAge = item.OldestQueueAgeMilli
		}
	}

	status := taskHealthStatusOK
	if len(stuckRuns) > 0 || activeOrphans > 0 {
		status = taskHealthStatusWarn
	}

	return TaskHealth{
		Status:                     status,
		QueueDepthTotal:            queueDepthTotal,
		OldestQueuedAt:             oldestQueuedAt,
		OldestQueueAgeMilli:        oldestQueuedAge,
		QueueDepth:                 summary.QueueDepth,
		StuckRuns:                  stuckRuns,
		ActiveOrphanRuns:           activeOrphans,
		TaskTotals:                 summary.TaskTotals,
		RunTotals:                  summary.RunTotals,
		OwnerTotals:                summary.OwnerTotals,
		ForcedStopsSinceStart:      metrics.TaskForcedStopsTotal,
		DuplicateIngressSinceStart: metrics.DuplicateIngressTotal,
		ChannelMismatchSinceStart:  metrics.ChannelMismatchTotal,
		RecoverySinceStart:         metrics.RecoveryTotals,
	}, nil
}

func taskSummaryFromSnapshot(snapshot taskSnapshot, now func() time.Time) Summary {
	return Summary{
		TotalTasks:  len(snapshot.tasks),
		TotalRuns:   len(snapshot.runs),
		TaskTotals:  summarizeTasks(snapshot.tasks),
		TaskOrigins: summarizeTaskOrigins(snapshot.tasks),
		RunTotals:   summarizeRuns(snapshot.runs),
		OwnerTotals: summarizeOwners(snapshot.tasks),
		QueueDepth:  summarizeQueueDepth(snapshot.runs, now),
	}
}

func taskMetricsFromSnapshot(snapshot taskSnapshot, query TaskMetricsQuery, now func() time.Time) TaskMetrics {
	runs := filterRunsByOrigin(snapshot.runs, query.OriginKind)
	events := filterTaskEvents(snapshot.events, snapshot.tasksByID, snapshot.runsByID, query)
	audits := filterTaskIngressAudits(snapshot.audits, query)
	duplicateIngress := max(countAcceptedEnqueueAudits(audits)-countNetworkEnqueueEvents(events), 0)

	return TaskMetrics{
		TasksTotal:              summarizeTasks(filterTasksByOrigin(snapshot.tasks, query.OriginKind)),
		TaskRunsTotal:           summarizeRuns(runs),
		TaskQueueDepth:          summarizeQueueDepth(runs, now),
		TaskCancelRequestsTotal: summarizeCancelRequests(events),
		TaskForcedStopsTotal:    countEventsByType(events, taskEventRunForceStopped),
		TaskClaimLatencyMillis:  summarizeClaimLatency(runs),
		TaskStartLatencyMillis:  summarizeStartLatency(runs),
		DuplicateIngressTotal:   duplicateIngress,
		ChannelMismatchTotal:    countChannelMismatchAudits(audits),
		RecoveryTotals:          summarizeRecovery(events),
	}
}

func (o *Observer) taskDashboardFromSnapshot(
	snapshot taskSnapshot,
	summary Summary,
	metrics TaskMetrics,
	health TaskHealth,
) TaskDashboardView {
	totals := taskDashboardTotalsFromSnapshot(snapshot, summary, metrics)
	queue := taskDashboardQueueFromRows(summary.QueueDepth, o.taskDashboardConfig, o.now)
	healthSummary := taskDashboardHealthFromHealth(health, queue.BacklogWarning)

	return TaskDashboardView{
		Totals:          totals,
		Cards:           taskDashboardCardsFromTotals(totals, metrics, healthSummary, health.ForcedStopsSinceStart),
		StatusBreakdown: taskDashboardStatusBreakdownFromTotals(totals),
		Queue:           queue,
		Health:          healthSummary,
		ActiveRuns: taskDashboardActiveRunsFromSnapshot(
			snapshot,
			o.now,
			o.taskDashboardConfig,
			o.taskHealthConfig,
		),
		Freshness: taskDashboardFreshnessFromSnapshot(snapshot, o.now, o.taskDashboardConfig),
	}
}

func taskDashboardTotalsFromSnapshot(
	snapshot taskSnapshot,
	summary Summary,
	metrics TaskMetrics,
) TaskDashboardTotals {
	awaitingApproval := countAwaitingApprovalTasks(snapshot.tasks)
	dependencyBlocked := countDependencyBlockedTasks(snapshot.tasks)

	totals := TaskDashboardTotals{
		TasksTotal:            summary.TotalTasks,
		RunsTotal:             summary.TotalRuns,
		DraftTasks:            countTaskStatus(summary.TaskTotals, taskpkg.TaskStatusDraft),
		PendingTasks:          countTaskStatus(summary.TaskTotals, taskpkg.TaskStatusPending),
		ReadyTasks:            countTaskStatus(summary.TaskTotals, taskpkg.TaskStatusReady),
		InProgressTasks:       countTaskStatus(summary.TaskTotals, taskpkg.TaskStatusInProgress),
		BlockedTasks:          countTaskStatus(summary.TaskTotals, taskpkg.TaskStatusBlocked),
		CompletedTasks:        countTaskStatus(summary.TaskTotals, taskpkg.TaskStatusCompleted),
		FailedTasks:           countTaskStatus(summary.TaskTotals, taskpkg.TaskStatusFailed),
		CanceledTasks:         countTaskStatus(summary.TaskTotals, taskpkg.TaskStatusCanceled),
		AwaitingApprovalTasks: awaitingApproval,
		QueuedRuns:            countRunStatus(metrics.TaskRunsTotal, taskpkg.TaskRunStatusQueued),
		ClaimedRuns:           countRunStatus(metrics.TaskRunsTotal, taskpkg.TaskRunStatusClaimed),
		StartingRuns:          countRunStatus(metrics.TaskRunsTotal, taskpkg.TaskRunStatusStarting),
		RunningRuns:           countRunStatus(metrics.TaskRunsTotal, taskpkg.TaskRunStatusRunning),
		CompletedRuns:         countRunStatus(metrics.TaskRunsTotal, taskpkg.TaskRunStatusCompleted),
		FailedRuns:            countRunStatus(metrics.TaskRunsTotal, taskpkg.TaskRunStatusFailed),
		CanceledRuns:          countRunStatus(metrics.TaskRunsTotal, taskpkg.TaskRunStatusCanceled),
	}
	totals.DependencyBlockedTasks = dependencyBlocked
	totals.ActiveRuns = totals.QueuedRuns + totals.ClaimedRuns + totals.StartingRuns + totals.RunningRuns
	return totals
}

func taskDashboardCardsFromTotals(
	totals TaskDashboardTotals,
	metrics TaskMetrics,
	health TaskDashboardHealth,
	forcedStops int,
) TaskDashboardCards {
	return TaskDashboardCards{
		InProgress: TaskDashboardInProgressCard{
			Tasks:        totals.InProgressTasks,
			ActiveRuns:   totals.ActiveRuns,
			RunningRuns:  totals.RunningRuns,
			StartingRuns: totals.StartingRuns,
			ClaimedRuns:  totals.ClaimedRuns,
			QueuedRuns:   totals.QueuedRuns,
			HealthStatus: health.Status,
		},
		Blocked: TaskDashboardBlockedCard{
			Tasks:                totals.BlockedTasks,
			AwaitingApproval:     totals.AwaitingApprovalTasks,
			AwaitingDependencies: totals.DependencyBlockedTasks,
			HealthStatus:         dashboardStatusForCount(totals.BlockedTasks),
		},
		Failed: TaskDashboardFailedCard{
			Tasks:        totals.FailedTasks,
			FailedRuns:   totals.FailedRuns,
			ForcedStops:  forcedStops,
			HealthStatus: dashboardStatusForAny(totals.FailedTasks > 0 || totals.FailedRuns > 0 || forcedStops > 0),
		},
		Latency: TaskDashboardLatencyCard{
			ClaimLatencyMillis: metrics.TaskClaimLatencyMillis,
			StartLatencyMillis: metrics.TaskStartLatencyMillis,
		},
	}
}

func taskDashboardStatusBreakdownFromTotals(totals TaskDashboardTotals) []TaskDashboardStatusBreakdown {
	type statusCount struct {
		status taskpkg.Status
		count  int
	}

	rows := []statusCount{
		{status: taskpkg.TaskStatusCompleted, count: totals.CompletedTasks},
		{status: taskpkg.TaskStatusPending, count: totals.PendingTasks},
		{status: taskpkg.TaskStatusInProgress, count: totals.InProgressTasks},
		{status: taskpkg.TaskStatusReady, count: totals.ReadyTasks},
		{status: taskpkg.TaskStatusBlocked, count: totals.BlockedTasks},
		{status: taskpkg.TaskStatusFailed, count: totals.FailedTasks},
		{status: taskpkg.TaskStatusCanceled, count: totals.CanceledTasks},
		{status: taskpkg.TaskStatusDraft, count: totals.DraftTasks},
	}

	breakdown := make([]TaskDashboardStatusBreakdown, 0, len(rows))
	for _, row := range rows {
		if row.count <= 0 || totals.TasksTotal <= 0 {
			continue
		}
		breakdown = append(breakdown, TaskDashboardStatusBreakdown{
			Status:       row.status,
			Count:        row.count,
			SharePercent: int(math.Round(float64(row.count) * 100 / float64(totals.TasksTotal))),
		})
	}
	return breakdown
}

func taskDashboardQueueFromRows(
	rows []TaskQueueDepth,
	cfg taskDashboardConfig,
	now func() time.Time,
) TaskDashboardQueue {
	queue := TaskDashboardQueue{
		Depth: rows,
	}
	threshold := max(cfg.backlogWarnAfter, 0)
	queue.BacklogThresholdMilli = threshold.Milliseconds()

	for _, item := range rows {
		queue.Total += item.Count
		if item.OldestQueuedAt.IsZero() {
			continue
		}
		if queue.OldestQueuedAt.IsZero() || item.OldestQueuedAt.Before(queue.OldestQueuedAt) {
			queue.OldestQueuedAt = item.OldestQueuedAt
			queue.OldestQueueAgeMilli = item.OldestQueueAgeMilli
		}
	}

	if queue.Total > 0 && threshold > 0 && time.Duration(queue.OldestQueueAgeMilli)*time.Millisecond >= threshold {
		queue.BacklogWarning = true
		queue.BacklogStatus = taskHealthStatusWarn
	} else {
		queue.BacklogStatus = taskHealthStatusOK
	}
	if now != nil && !queue.OldestQueuedAt.IsZero() {
		queue.OldestQueueAgeMilli = safeSince(now(), queue.OldestQueuedAt).Milliseconds()
		if queue.Total > 0 && threshold > 0 && time.Duration(queue.OldestQueueAgeMilli)*time.Millisecond >= threshold {
			queue.BacklogWarning = true
			queue.BacklogStatus = taskHealthStatusWarn
		}
	}

	return queue
}

func taskDashboardHealthFromHealth(health TaskHealth, queueBacklog bool) TaskDashboardHealth {
	status := health.Status
	if queueBacklog {
		status = taskHealthStatusWarn
	}
	if strings.TrimSpace(status) == "" {
		status = taskHealthStatusOK
	}

	return TaskDashboardHealth{
		Status:           status,
		StuckRuns:        len(health.StuckRuns),
		ActiveOrphanRuns: health.ActiveOrphanRuns,
		QueueBacklog:     queueBacklog,
	}
}

func taskDashboardActiveRunsFromSnapshot(
	snapshot taskSnapshot,
	now func() time.Time,
	cfg taskDashboardConfig,
	healthCfg TaskHealthConfig,
) TaskDashboardActiveRuns {
	currentTime := dashboardNow(now)
	items := dashboardActiveRuns(snapshot.runs)
	stuckByID := dashboardStuckRunSet(snapshot.runs, currentTime, healthCfg)

	activeRuns := taskDashboardActiveRunCounts(items)
	activeRuns.Items = taskDashboardActiveRunItems(
		items,
		snapshot.tasksByID,
		currentTime,
		cfg.activeRunLimit,
		stuckByID,
	)
	return activeRuns
}

func taskDashboardFreshnessFromSnapshot(
	snapshot taskSnapshot,
	now func() time.Time,
	cfg taskDashboardConfig,
) TaskDashboardFreshness {
	observedAt := time.Now().UTC()
	if now != nil {
		observedAt = now().UTC()
	}

	latestActivity := latestTaskSnapshotActivityAt(snapshot)
	staleAfter := max(cfg.staleAfter, 0)
	hasLiveWork := snapshotHasLiveWork(snapshot)
	age := safeSince(observedAt, latestActivity)

	freshness := TaskDashboardFreshness{
		ObservedAt:       observedAt,
		LatestActivityAt: latestActivity,
		AgeMilli:         age.Milliseconds(),
		StaleAfterMilli:  staleAfter.Milliseconds(),
		HasLiveWork:      hasLiveWork,
		Status:           "current",
	}

	switch {
	case latestActivity.IsZero():
		freshness.Status = "empty"
	case hasLiveWork && staleAfter > 0 && age > staleAfter:
		freshness.Status = "stale"
		freshness.Stale = true
	}

	return freshness
}

func taskInboxFromSnapshot(
	snapshot taskSnapshot,
	triageStates []taskpkg.TriageState,
	query TaskInboxQuery,
	actor taskpkg.ActorIdentity,
) TaskInboxView {
	triageByTaskID := taskInboxTriageByTaskID(triageStates)
	runsByTaskID := taskRunsByTaskID(snapshot.runs)
	eventsByTaskID := taskEventsByTaskID(snapshot.events)
	lanes := taskInboxLanes(query.Lane.Normalize())
	itemsByLane := make(map[TaskInboxLane][]TaskInboxItem, len(lanes))
	countsByLane := make(map[TaskInboxLane]int, len(lanes))
	unreadByLane := make(map[TaskInboxLane]int, len(lanes))

	view := TaskInboxView{}
	selectedLane := query.Lane.Normalize()
	for _, summary := range snapshot.tasks {
		taskID := strings.TrimSpace(summary.ID)
		if taskID == "" {
			continue
		}

		taskRuns := runsByTaskID[taskID]
		taskEvents := eventsByTaskID[taskID]
		latestActivityAt := taskInboxLatestActivityAt(summary, taskRuns, taskEvents)
		triage, hasTriage := triageByTaskID[taskID]
		effectiveTriage := taskInboxEffectiveTriage(taskID, actor, latestActivityAt, triage, hasTriage)
		if taskInboxSuppressedByDismissal(effectiveTriage) {
			continue
		}

		latestRun := latestTaskInboxRun(taskRuns)
		lane, include := taskInboxLaneForTask(summary, latestRun, taskRuns, effectiveTriage, actor)
		if !include {
			continue
		}
		if selectedLane != "" && lane != selectedLane {
			continue
		}

		item := TaskInboxItem{
			Task:             taskInboxReference(summary),
			Lane:             lane,
			ApprovalPolicy:   summary.ApprovalPolicy,
			ApprovalState:    summary.ApprovalState,
			BlockingReason:   taskInboxBlockingReason(summary, latestRun),
			LatestActivityAt: latestActivityAt,
			Run:              taskInboxRunSummary(latestRun, summary.MaxAttempts),
			Triage:           effectiveTriage,
		}
		if query.Unread && !taskInboxItemUnread(item) {
			continue
		}

		countsByLane[lane]++
		if taskInboxItemUnread(item) {
			unreadByLane[lane]++
			view.UnreadTotal++
		}
		if lane == TaskInboxLaneArchived {
			view.ArchivedTotal++
		}
		view.Total++
		itemsByLane[lane] = append(itemsByLane[lane], item)
	}

	view.Groups = taskInboxGroups(lanes, itemsByLane, countsByLane, unreadByLane, query.Limit)
	return view
}

func taskInboxTriageByTaskID(states []taskpkg.TriageState) map[string]taskpkg.TriageState {
	items := make(map[string]taskpkg.TriageState, len(states))
	for _, state := range states {
		taskID := strings.TrimSpace(state.TaskID)
		if taskID == "" {
			continue
		}
		items[taskID] = state
	}
	return items
}

func normalizeTaskInboxActor(actor taskpkg.ActorIdentity) (taskpkg.ActorIdentity, error) {
	normalized := taskpkg.ActorIdentity{
		Kind: actor.Kind.Normalize(),
		Ref:  strings.TrimSpace(actor.Ref),
	}
	if err := normalized.Validate("task_inbox.actor"); err != nil {
		return taskpkg.ActorIdentity{}, err
	}
	return normalized, nil
}

func taskInboxLanes(filter TaskInboxLane) []TaskInboxLane {
	if filter.Normalize() != "" {
		return []TaskInboxLane{filter.Normalize()}
	}
	return []TaskInboxLane{
		TaskInboxLaneMyWork,
		TaskInboxLaneApprovals,
		TaskInboxLaneFailedRuns,
		TaskInboxLaneBlocked,
		TaskInboxLaneArchived,
	}
}

func taskRunsByTaskID(runs []taskpkg.Run) map[string][]taskpkg.Run {
	items := make(map[string][]taskpkg.Run)
	for _, run := range runs {
		taskID := strings.TrimSpace(run.TaskID)
		if taskID == "" {
			continue
		}
		items[taskID] = append(items[taskID], run)
	}
	return items
}

func taskEventsByTaskID(events []taskpkg.Event) map[string][]taskpkg.Event {
	items := make(map[string][]taskpkg.Event)
	for _, event := range events {
		taskID := strings.TrimSpace(event.TaskID)
		if taskID == "" {
			continue
		}
		items[taskID] = append(items[taskID], event)
	}
	return items
}

func taskInboxLatestActivityAt(
	summary taskpkg.Summary,
	runs []taskpkg.Run,
	events []taskpkg.Event,
) time.Time {
	latest := summary.UpdatedAt
	for _, candidate := range []time.Time{summary.CreatedAt, summary.ClosedAt, summary.LastActivityAt} {
		if candidate.After(latest) {
			latest = candidate
		}
	}
	for _, run := range runs {
		if runAt := dashboardRunActivityAt(run); runAt.After(latest) {
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

func latestTaskInboxRun(runs []taskpkg.Run) *taskpkg.Run {
	var latest *taskpkg.Run
	for idx := range runs {
		run := runs[idx]
		if latest == nil || taskInboxRunComesAfter(run, *latest) {
			candidate := run
			latest = &candidate
		}
	}
	return latest
}

func taskInboxRunComesAfter(left, right taskpkg.Run) bool {
	switch {
	case left.Attempt != right.Attempt:
		return left.Attempt > right.Attempt
	case !left.QueuedAt.Equal(right.QueuedAt):
		return left.QueuedAt.After(right.QueuedAt)
	default:
		return left.ID > right.ID
	}
}

func taskInboxEffectiveTriage(
	taskID string,
	actor taskpkg.ActorIdentity,
	latestActivityAt time.Time,
	record taskpkg.TriageState,
	ok bool,
) taskpkg.TriageState {
	if !ok {
		return taskpkg.TriageState{TaskID: taskID, Actor: actor}
	}

	effective := record
	effective.TaskID = taskID
	effective.Actor = actor
	if effective.Archived {
		effective.Read = true
		effective.Dismissed = false
		return effective
	}
	if !latestActivityAt.IsZero() &&
		(effective.LastSeenActivityAt.IsZero() || latestActivityAt.After(effective.LastSeenActivityAt)) {
		effective.Read = false
		effective.Dismissed = false
	}
	return effective
}

func taskInboxSuppressedByDismissal(triage taskpkg.TriageState) bool {
	return triage.Dismissed && !triage.Archived
}

func taskInboxLaneForTask(
	summary taskpkg.Summary,
	latestRun *taskpkg.Run,
	runs []taskpkg.Run,
	triage taskpkg.TriageState,
	actor taskpkg.ActorIdentity,
) (TaskInboxLane, bool) {
	if triage.Archived {
		return TaskInboxLaneArchived, true
	}
	if summary.Status.Normalize() == taskpkg.TaskStatusBlocked &&
		summary.ApprovalPolicy.Normalize() == taskpkg.ApprovalPolicyManual &&
		summary.ApprovalState.Normalize() == taskpkg.ApprovalStatePending {
		return TaskInboxLaneApprovals, true
	}
	if latestRun != nil && latestRun.Status.Normalize() == taskpkg.TaskRunStatusFailed {
		return TaskInboxLaneFailedRuns, true
	}
	if summary.Status.Normalize() == taskpkg.TaskStatusBlocked {
		return TaskInboxLaneBlocked, true
	}
	if taskInboxIsMyWork(summary, runs, actor) && taskInboxEligibleForMyWork(summary.Status) {
		return TaskInboxLaneMyWork, true
	}
	return "", false
}

func taskInboxBlockingReason(summary taskpkg.Summary, latestRun *taskpkg.Run) string {
	if summary.ApprovalPolicy.Normalize() == taskpkg.ApprovalPolicyManual {
		switch summary.ApprovalState.Normalize() {
		case taskpkg.ApprovalStatePending:
			return taskInboxBlockingReasonAwaitingApproval
		case taskpkg.ApprovalStateRejected:
			return taskInboxBlockingReasonApprovalRejected
		}
	}
	if latestRun != nil && latestRun.Status.Normalize() == taskpkg.TaskRunStatusFailed {
		return taskInboxBlockingReasonLatestRunFailed
	}
	if summary.Status.Normalize() == taskpkg.TaskStatusBlocked {
		return taskInboxBlockingReasonAwaitingDeps
	}
	return ""
}

func taskInboxIsMyWork(
	summary taskpkg.Summary,
	runs []taskpkg.Run,
	actor taskpkg.ActorIdentity,
) bool {
	if ownerKind, ok := taskInboxOwnerKindForActor(actor.Kind); ok && summary.Owner != nil &&
		summary.Owner.Kind.Normalize() == ownerKind &&
		strings.TrimSpace(summary.Owner.Ref) == strings.TrimSpace(actor.Ref) {
		return true
	}

	for _, run := range runs {
		if !isDashboardActiveRunStatus(run.Status) || run.ClaimedBy == nil {
			continue
		}
		if run.ClaimedBy.Kind.Normalize() == actor.Kind.Normalize() &&
			strings.TrimSpace(run.ClaimedBy.Ref) == strings.TrimSpace(actor.Ref) {
			return true
		}
	}
	return false
}

func taskInboxOwnerKindForActor(kind taskpkg.ActorKind) (taskpkg.OwnerKind, bool) {
	switch kind.Normalize() {
	case taskpkg.ActorKindHuman:
		return taskpkg.OwnerKindHuman, true
	case taskpkg.ActorKindAgentSession:
		return taskpkg.OwnerKindAgentSession, true
	case taskpkg.ActorKindAutomation:
		return taskpkg.OwnerKindAutomation, true
	case taskpkg.ActorKindExtension:
		return taskpkg.OwnerKindExtension, true
	case taskpkg.ActorKindNetworkPeer:
		return taskpkg.OwnerKindNetworkPeer, true
	default:
		return "", false
	}
}

func taskInboxEligibleForMyWork(status taskpkg.Status) bool {
	switch status.Normalize() {
	case taskpkg.TaskStatusDraft,
		taskpkg.TaskStatusCompleted,
		taskpkg.TaskStatusFailed,
		taskpkg.TaskStatusCanceled:
		return false
	default:
		return true
	}
}

func taskInboxReference(summary taskpkg.Summary) taskpkg.Reference {
	return taskpkg.Reference{
		ID:             summary.ID,
		Identifier:     summary.Identifier,
		Title:          summary.Title,
		Status:         summary.Status,
		Priority:       summary.Priority,
		Owner:          cloneOwnership(summary.Owner),
		Scope:          summary.Scope,
		WorkspaceID:    summary.WorkspaceID,
		LatestEventSeq: summary.LatestEventSeq,
	}
}

func taskInboxRunSummary(run *taskpkg.Run, maxAttempts int) *taskpkg.RunSummary {
	if run == nil {
		return nil
	}
	return &taskpkg.RunSummary{
		ID:          run.ID,
		TaskID:      run.TaskID,
		Status:      run.Status,
		Attempt:     run.Attempt,
		MaxAttempts: maxAttempts,
		SessionID:   run.SessionID,
		ClaimedBy:   cloneActorIdentity(run.ClaimedBy),
		QueuedAt:    run.QueuedAt,
		ClaimedAt:   run.ClaimedAt,
		StartedAt:   run.StartedAt,
		EndedAt:     run.EndedAt,
		Error:       strings.TrimSpace(run.Error),
	}
}

func taskInboxItemUnread(item TaskInboxItem) bool {
	return !item.Triage.Read
}

func taskInboxGroups(
	lanes []TaskInboxLane,
	itemsByLane map[TaskInboxLane][]TaskInboxItem,
	countsByLane map[TaskInboxLane]int,
	unreadByLane map[TaskInboxLane]int,
	limit int,
) []TaskInboxLaneGroup {
	groups := make([]TaskInboxLaneGroup, 0, len(lanes))
	for _, lane := range lanes {
		items := itemsByLane[lane]
		sortTaskInboxItems(items)
		if limit > 0 && len(items) > limit {
			items = append([]TaskInboxItem(nil), items[:limit]...)
		} else if len(items) > 0 {
			items = append([]TaskInboxItem(nil), items...)
		}
		groups = append(groups, TaskInboxLaneGroup{
			Lane:        lane,
			Count:       countsByLane[lane],
			UnreadCount: unreadByLane[lane],
			Items:       items,
		})
	}
	return groups
}

func sortTaskInboxItems(items []TaskInboxItem) {
	slices.SortFunc(items, compareTaskInboxItems)
}

func compareTaskInboxItems(left, right TaskInboxItem) int {
	if leftUnread, rightUnread := taskInboxItemUnread(left), taskInboxItemUnread(right); leftUnread != rightUnread {
		if leftUnread {
			return -1
		}
		return 1
	}
	if !left.LatestActivityAt.Equal(right.LatestActivityAt) {
		if left.LatestActivityAt.After(right.LatestActivityAt) {
			return -1
		}
		return 1
	}
	leftPriority := taskInboxPriorityRank(left.Task.Priority)
	rightPriority := taskInboxPriorityRank(right.Task.Priority)
	if leftPriority != rightPriority {
		if leftPriority > rightPriority {
			return -1
		}
		return 1
	}
	return strings.Compare(left.Task.ID, right.Task.ID)
}

func taskInboxPriorityRank(priority taskpkg.Priority) int {
	switch priority.Normalize() {
	case taskpkg.PriorityUrgent:
		return 4
	case taskpkg.PriorityHigh:
		return 3
	case taskpkg.PriorityMedium:
		return 2
	case taskpkg.PriorityLow:
		return 1
	default:
		return 0
	}
}

func countTaskStatus(rows []TaskStatusTotal, status taskpkg.Status) int {
	count := 0
	for _, item := range rows {
		if item.Status.Normalize() == status.Normalize() {
			count += item.Count
		}
	}
	return count
}

func countRunStatus(rows []TaskRunTotal, status taskpkg.RunStatus) int {
	count := 0
	for _, item := range rows {
		if item.Status.Normalize() == status.Normalize() {
			count += item.Count
		}
	}
	return count
}

func countAwaitingApprovalTasks(tasks []taskpkg.Summary) int {
	count := 0
	for _, item := range tasks {
		if item.Status.Normalize() == taskpkg.TaskStatusBlocked &&
			item.ApprovalState.Normalize() == taskpkg.ApprovalStatePending {
			count++
		}
	}
	return count
}

func countDependencyBlockedTasks(tasks []taskpkg.Summary) int {
	count := 0
	for _, item := range tasks {
		if item.Status.Normalize() != taskpkg.TaskStatusBlocked {
			continue
		}
		if item.DependencyCount <= 0 && len(item.Dependencies) == 0 {
			continue
		}
		count++
	}
	return count
}

func dashboardStatusForCount(count int) string {
	return dashboardStatusForAny(count > 0)
}

func dashboardStatusForAny(warn bool) string {
	if warn {
		return taskHealthStatusWarn
	}
	return taskHealthStatusOK
}

func dashboardActiveRunRank(status taskpkg.RunStatus) int {
	switch status.Normalize() {
	case taskpkg.TaskRunStatusRunning:
		return 4
	case taskpkg.TaskRunStatusStarting:
		return 3
	case taskpkg.TaskRunStatusClaimed:
		return 2
	case taskpkg.TaskRunStatusQueued:
		return 1
	default:
		return 0
	}
}

func dashboardRunActivityAt(run taskpkg.Run) time.Time {
	latest := run.QueuedAt
	for _, candidate := range []time.Time{run.ClaimedAt, run.StartedAt, run.EndedAt} {
		if candidate.After(latest) {
			latest = candidate
		}
	}
	return latest
}

func dashboardRunAge(run taskpkg.Run, now time.Time) time.Duration {
	switch run.Status.Normalize() {
	case taskpkg.TaskRunStatusRunning:
		if !run.StartedAt.IsZero() {
			return safeSince(now, run.StartedAt)
		}
	case taskpkg.TaskRunStatusStarting, taskpkg.TaskRunStatusClaimed:
		if !run.ClaimedAt.IsZero() {
			return safeSince(now, run.ClaimedAt)
		}
	}
	return safeSince(now, run.QueuedAt)
}

func latestTaskSnapshotActivityAt(snapshot taskSnapshot) time.Time {
	var latest time.Time
	for _, item := range snapshot.tasks {
		for _, candidate := range []time.Time{item.CreatedAt, item.UpdatedAt, item.ClosedAt} {
			if candidate.After(latest) {
				latest = candidate
			}
		}
	}
	for _, item := range snapshot.runs {
		if activityAt := dashboardRunActivityAt(item); activityAt.After(latest) {
			latest = activityAt
		}
	}
	for _, item := range snapshot.events {
		if item.Timestamp.After(latest) {
			latest = item.Timestamp
		}
	}
	return latest
}

func snapshotHasLiveWork(snapshot taskSnapshot) bool {
	for _, item := range snapshot.runs {
		if isDashboardActiveRunStatus(item.Status) {
			return true
		}
	}
	return false
}

func dashboardNow(now func() time.Time) time.Time {
	if now == nil {
		return time.Now().UTC()
	}
	return now().UTC()
}

func dashboardActiveRuns(runs []taskpkg.Run) []taskpkg.Run {
	items := make([]taskpkg.Run, 0, len(runs))
	for _, item := range runs {
		if isDashboardActiveRunStatus(item.Status) {
			items = append(items, item)
		}
	}
	slices.SortFunc(items, compareDashboardActiveRuns)
	return items
}

func compareDashboardActiveRuns(left, right taskpkg.Run) int {
	leftRank := dashboardActiveRunRank(left.Status)
	rightRank := dashboardActiveRunRank(right.Status)
	if leftRank != rightRank {
		if leftRank > rightRank {
			return -1
		}
		return 1
	}

	leftAt := dashboardRunActivityAt(left)
	rightAt := dashboardRunActivityAt(right)
	if !leftAt.Equal(rightAt) {
		if leftAt.After(rightAt) {
			return -1
		}
		return 1
	}
	return strings.Compare(right.ID, left.ID)
}

func dashboardStuckRunSet(
	runs []taskpkg.Run,
	currentTime time.Time,
	healthCfg TaskHealthConfig,
) map[string]struct{} {
	stuckByID := make(map[string]struct{})
	for _, item := range findStuckRuns(runs, currentTime, healthCfg) {
		stuckByID[item.RunID] = struct{}{}
	}
	return stuckByID
}

func taskDashboardActiveRunCounts(items []taskpkg.Run) TaskDashboardActiveRuns {
	activeRuns := TaskDashboardActiveRuns{Total: len(items)}
	for _, item := range items {
		switch item.Status.Normalize() {
		case taskpkg.TaskRunStatusRunning:
			activeRuns.Running++
		case taskpkg.TaskRunStatusStarting:
			activeRuns.Starting++
		case taskpkg.TaskRunStatusClaimed:
			activeRuns.Claimed++
		case taskpkg.TaskRunStatusQueued:
			activeRuns.Queued++
		}
	}
	return activeRuns
}

func taskDashboardActiveRunItems(
	items []taskpkg.Run,
	tasksByID map[string]taskpkg.Summary,
	currentTime time.Time,
	limit int,
	stuckByID map[string]struct{},
) []TaskDashboardActiveRun {
	if limit <= 0 {
		limit = 4
	}
	if limit > len(items) {
		limit = len(items)
	}

	activeRunItems := make([]TaskDashboardActiveRun, 0, limit)
	for _, run := range items[:limit] {
		taskItem, ok := tasksByID[strings.TrimSpace(run.TaskID)]
		if !ok {
			continue
		}
		_, stuck := stuckByID[run.ID]
		activeRunItems = append(activeRunItems, TaskDashboardActiveRun{
			TaskID:         taskItem.ID,
			TaskIdentifier: taskItem.Identifier,
			TaskTitle:      taskItem.Title,
			TaskStatus:     taskItem.Status.Normalize(),
			TaskPriority:   taskItem.Priority,
			TaskOwner:      cloneOwnership(taskItem.Owner),
			Scope:          taskItem.Scope.Normalize(),
			WorkspaceID:    taskItem.WorkspaceID,
			LatestEventSeq: taskItem.LatestEventSeq,
			RunID:          run.ID,
			RunStatus:      run.Status.Normalize(),
			Attempt:        run.Attempt,
			MaxAttempts:    taskItem.MaxAttempts,
			SessionID:      strings.TrimSpace(run.SessionID),
			NetworkChannel: strings.TrimSpace(run.NetworkChannel),
			LastActivityAt: dashboardRunActivityAt(run),
			AgeMilli:       dashboardRunAge(run, currentTime).Milliseconds(),
			HealthStatus:   dashboardStatusForAny(stuck),
			Stuck:          stuck,
			Error:          strings.TrimSpace(run.Error),
		})
	}
	return activeRunItems
}

func isDashboardActiveRunStatus(status taskpkg.RunStatus) bool {
	switch status.Normalize() {
	case taskpkg.TaskRunStatusQueued,
		taskpkg.TaskRunStatusClaimed,
		taskpkg.TaskRunStatusStarting,
		taskpkg.TaskRunStatusRunning:
		return true
	default:
		return false
	}
}

func cloneOwnership(owner *taskpkg.Ownership) *taskpkg.Ownership {
	if owner == nil {
		return nil
	}
	cloned := *owner
	return &cloned
}

func cloneActorIdentity(actor *taskpkg.ActorIdentity) *taskpkg.ActorIdentity {
	if actor == nil {
		return nil
	}
	cloned := *actor
	return &cloned
}

func (o *Observer) loadTaskSnapshot(ctx context.Context, query TaskSummaryQuery) (taskSnapshot, error) {
	if ctx == nil {
		return taskSnapshot{}, errors.New("observe: task summary context is required")
	}
	if err := query.Validate(); err != nil {
		return taskSnapshot{}, err
	}

	tasks, err := o.registry.ListTasks(ctx, taskpkg.Query{
		Scope:          query.Scope,
		WorkspaceID:    strings.TrimSpace(query.WorkspaceID),
		OwnerKind:      query.OwnerKind.Normalize(),
		OwnerRef:       strings.TrimSpace(query.OwnerRef),
		NetworkChannel: strings.TrimSpace(query.NetworkChannel),
		Search:         strings.TrimSpace(query.Search),
	})
	if err != nil {
		return taskSnapshot{}, fmt.Errorf("observe: list tasks for summary: %w", err)
	}
	tasks = filterTasksByOrigin(tasks, query.OriginKind)
	dependencyCounts, err := o.loadTaskDependencyCounts(ctx, tasks)
	if err != nil {
		return taskSnapshot{}, err
	}
	for idx := range tasks {
		taskID := strings.TrimSpace(tasks[idx].ID)
		if taskID == "" {
			continue
		}
		tasks[idx].DependencyCount = dependencyCounts[taskID]
	}

	tasksByID, taskIDs := taskSummaryIndex(tasks)

	runs, err := o.registry.ListTaskRuns(ctx, taskpkg.RunQuery{})
	if err != nil {
		return taskSnapshot{}, fmt.Errorf("observe: list task runs for summary: %w", err)
	}
	runs = filterRuns(runs, taskIDs, query)

	runsByID := make(map[string]taskpkg.Run, len(runs))
	for _, item := range runs {
		runID := strings.TrimSpace(item.ID)
		if runID == "" {
			continue
		}
		runsByID[runID] = item
	}

	events, err := o.registry.ListTaskEvents(ctx, taskpkg.EventQuery{})
	if err != nil {
		return taskSnapshot{}, fmt.Errorf("observe: list task events for summary: %w", err)
	}
	events = filterEventsForTasks(events, taskIDs)

	workspaceID := strings.TrimSpace(query.WorkspaceID)
	audits, err := o.registry.ListNetworkAudit(ctx, store.NetworkAuditQuery{
		WorkspaceID: workspaceID,
		Global:      workspaceID == "",
		Channel:     strings.TrimSpace(query.NetworkChannel),
	})
	if err != nil {
		return taskSnapshot{}, fmt.Errorf("observe: list network audit for summary: %w", err)
	}

	return taskSnapshot{
		tasks:     tasks,
		runs:      runs,
		events:    events,
		audits:    audits,
		tasksByID: tasksByID,
		runsByID:  runsByID,
	}, nil
}

func taskSummaryIndex(
	tasks []taskpkg.Summary,
) (map[string]taskpkg.Summary, map[string]struct{}) {
	tasksByID := make(map[string]taskpkg.Summary, len(tasks))
	taskIDs := make(map[string]struct{}, len(tasks))
	for _, item := range tasks {
		taskID := strings.TrimSpace(item.ID)
		if taskID == "" {
			continue
		}
		tasksByID[taskID] = item
		taskIDs[taskID] = struct{}{}
	}
	return tasksByID, taskIDs
}

func (o *Observer) loadTaskDependencyCounts(
	ctx context.Context,
	tasks []taskpkg.Summary,
) (map[string]int, error) {
	taskIDs := make([]string, 0, len(tasks))
	seen := make(map[string]struct{}, len(tasks))
	for _, item := range tasks {
		taskID := strings.TrimSpace(item.ID)
		if taskID == "" {
			continue
		}
		if _, ok := seen[taskID]; ok {
			continue
		}
		seen[taskID] = struct{}{}
		taskIDs = append(taskIDs, taskID)
	}
	if len(taskIDs) == 0 {
		return map[string]int{}, nil
	}

	path := strings.TrimSpace(o.registry.Path())
	if path == "" {
		return o.loadTaskDependencyCountsIndividually(ctx, taskIDs)
	}

	db, err := store.OpenSQLiteDatabase(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("observe: open registry database for dependency counts: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	counts, err := queryTaskDependencyCounts(ctx, db, taskIDs)
	if err != nil {
		return nil, err
	}
	return counts, nil
}

func (o *Observer) loadTaskDependencyCountsIndividually(
	ctx context.Context,
	taskIDs []string,
) (map[string]int, error) {
	counts := make(map[string]int, len(taskIDs))
	for _, taskID := range taskIDs {
		count, err := o.registry.CountDependencies(ctx, taskID)
		if err != nil {
			return nil, fmt.Errorf("observe: count dependencies for task %q: %w", taskID, err)
		}
		counts[taskID] = count
	}
	return counts, nil
}

func queryTaskDependencyCounts(
	ctx context.Context,
	db *sql.DB,
	taskIDs []string,
) (map[string]int, error) {
	taskIDsJSON, err := json.Marshal(taskIDs)
	if err != nil {
		return nil, fmt.Errorf("observe: encode dependency count task ids: %w", err)
	}

	rows, err := db.QueryContext(
		ctx,
		`SELECT dep.task_id, COUNT(1)
		FROM task_dependencies dep
		JOIN json_each(?) requested ON requested.value = dep.task_id
		GROUP BY dep.task_id`,
		string(taskIDsJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("observe: query dependency counts: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	counts := make(map[string]int, len(taskIDs))
	for rows.Next() {
		var taskID string
		var count int
		if err := rows.Scan(&taskID, &count); err != nil {
			return nil, fmt.Errorf("observe: scan dependency counts: %w", err)
		}
		counts[strings.TrimSpace(taskID)] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("observe: iterate dependency counts: %w", err)
	}
	return counts, nil
}

func summarizeTasks(tasks []taskpkg.Summary) []TaskStatusTotal {
	counts := make(map[string]TaskStatusTotal)
	for _, item := range tasks {
		key := string(
			item.Scope.Normalize(),
		) + "\x00" + string(
			item.Status.Normalize(),
		) + "\x00" + strings.TrimSpace(
			item.NetworkChannel,
		)
		current := counts[key]
		current.Scope = item.Scope.Normalize()
		current.Status = item.Status.Normalize()
		current.NetworkChannel = strings.TrimSpace(item.NetworkChannel)
		current.Count++
		counts[key] = current
	}
	rows := make([]TaskStatusTotal, 0, len(counts))
	for _, item := range counts {
		rows = append(rows, item)
	}
	slices.SortFunc(rows, func(left, right TaskStatusTotal) int {
		if cmp := strings.Compare(string(left.Scope), string(right.Scope)); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(string(left.Status), string(right.Status)); cmp != 0 {
			return cmp
		}
		return strings.Compare(left.NetworkChannel, right.NetworkChannel)
	})
	return rows
}

func summarizeTaskOrigins(tasks []taskpkg.Summary) []TaskOriginTotal {
	counts := make(map[string]TaskOriginTotal)
	for _, item := range tasks {
		key := string(item.Origin.Kind.Normalize()) + "\x00" + strings.TrimSpace(item.NetworkChannel)
		current := counts[key]
		current.OriginKind = item.Origin.Kind.Normalize()
		current.NetworkChannel = strings.TrimSpace(item.NetworkChannel)
		current.Count++
		counts[key] = current
	}
	rows := make([]TaskOriginTotal, 0, len(counts))
	for _, item := range counts {
		rows = append(rows, item)
	}
	slices.SortFunc(rows, func(left, right TaskOriginTotal) int {
		if cmp := strings.Compare(string(left.OriginKind), string(right.OriginKind)); cmp != 0 {
			return cmp
		}
		return strings.Compare(left.NetworkChannel, right.NetworkChannel)
	})
	return rows
}

func summarizeRuns(runs []taskpkg.Run) []TaskRunTotal {
	counts := make(map[string]TaskRunTotal)
	for _, item := range runs {
		channel := strings.TrimSpace(item.NetworkChannel)
		key := string(item.Status.Normalize()) + "\x00" + string(item.Origin.Kind.Normalize()) + "\x00" + channel
		current := counts[key]
		current.Status = item.Status.Normalize()
		current.OriginKind = item.Origin.Kind.Normalize()
		current.NetworkChannel = channel
		current.Count++
		counts[key] = current
	}
	rows := make([]TaskRunTotal, 0, len(counts))
	for _, item := range counts {
		rows = append(rows, item)
	}
	slices.SortFunc(rows, func(left, right TaskRunTotal) int {
		if cmp := strings.Compare(string(left.Status), string(right.Status)); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(string(left.OriginKind), string(right.OriginKind)); cmp != 0 {
			return cmp
		}
		return strings.Compare(left.NetworkChannel, right.NetworkChannel)
	})
	return rows
}

func summarizeOwners(tasks []taskpkg.Summary) []TaskOwnerTotal {
	counts := make(map[string]TaskOwnerTotal)
	for _, item := range tasks {
		if item.Owner == nil {
			continue
		}
		key := string(item.Owner.Kind.Normalize()) + "\x00" + strings.TrimSpace(item.Owner.Ref)
		current := counts[key]
		current.OwnerKind = item.Owner.Kind.Normalize()
		current.OwnerRef = strings.TrimSpace(item.Owner.Ref)
		current.Count++
		counts[key] = current
	}
	rows := make([]TaskOwnerTotal, 0, len(counts))
	for _, item := range counts {
		rows = append(rows, item)
	}
	slices.SortFunc(rows, func(left, right TaskOwnerTotal) int {
		if cmp := strings.Compare(string(left.OwnerKind), string(right.OwnerKind)); cmp != 0 {
			return cmp
		}
		return strings.Compare(left.OwnerRef, right.OwnerRef)
	})
	return rows
}

func summarizeQueueDepth(runs []taskpkg.Run, now func() time.Time) []TaskQueueDepth {
	counts := make(map[string]TaskQueueDepth)
	currentTime := time.Now().UTC()
	if now != nil {
		currentTime = now().UTC()
	}
	for _, item := range runs {
		if item.Status.Normalize() != taskpkg.TaskRunStatusQueued {
			continue
		}
		channel := strings.TrimSpace(item.NetworkChannel)
		current := counts[channel]
		current.NetworkChannel = channel
		current.Count++
		if current.OldestQueuedAt.IsZero() || item.QueuedAt.Before(current.OldestQueuedAt) {
			current.OldestQueuedAt = item.QueuedAt
			age := max(currentTime.Sub(item.QueuedAt), 0)
			current.OldestQueueAgeMilli = age.Milliseconds()
		}
		counts[channel] = current
	}
	rows := make([]TaskQueueDepth, 0, len(counts))
	for _, item := range counts {
		rows = append(rows, item)
	}
	slices.SortFunc(rows, func(left, right TaskQueueDepth) int {
		return strings.Compare(left.NetworkChannel, right.NetworkChannel)
	})
	return rows
}

func summarizeCancelRequests(events []taskpkg.Event) []TaskCancelRequestTotal {
	counts := make(map[string]TaskCancelRequestTotal)
	for _, item := range events {
		if item.EventType != taskEventCanceled {
			continue
		}
		key := string(item.Origin.Kind.Normalize())
		current := counts[key]
		current.OriginKind = item.Origin.Kind.Normalize()
		current.Count++
		counts[key] = current
	}
	rows := make([]TaskCancelRequestTotal, 0, len(counts))
	for _, item := range counts {
		rows = append(rows, item)
	}
	slices.SortFunc(rows, func(left, right TaskCancelRequestTotal) int {
		return strings.Compare(string(left.OriginKind), string(right.OriginKind))
	})
	return rows
}

func summarizeClaimLatency(runs []taskpkg.Run) LatencyMetric {
	return summarizeLatency(runs, func(run taskpkg.Run) (time.Duration, bool) {
		if run.ClaimedAt.IsZero() {
			return 0, false
		}
		duration := max(run.ClaimedAt.Sub(run.QueuedAt), 0)
		return duration, true
	})
}

func summarizeStartLatency(runs []taskpkg.Run) LatencyMetric {
	return summarizeLatency(runs, func(run taskpkg.Run) (time.Duration, bool) {
		if run.StartedAt.IsZero() {
			return 0, false
		}
		base := run.ClaimedAt
		if base.IsZero() {
			base = run.QueuedAt
		}
		duration := max(run.StartedAt.Sub(base), 0)
		return duration, true
	})
}

func summarizeLatency(runs []taskpkg.Run, measure func(taskpkg.Run) (time.Duration, bool)) LatencyMetric {
	var total time.Duration
	var maxDuration time.Duration
	samples := 0
	for _, item := range runs {
		duration, ok := measure(item)
		if !ok {
			continue
		}
		samples++
		total += duration
		if duration > maxDuration {
			maxDuration = duration
		}
	}
	if samples == 0 {
		return LatencyMetric{}
	}
	return LatencyMetric{
		Samples:       samples,
		AverageMillis: (total / time.Duration(samples)).Milliseconds(),
		MaximumMillis: maxDuration.Milliseconds(),
	}
}

func summarizeRecovery(events []taskpkg.Event) TaskRecoveryTotals {
	totals := TaskRecoveryTotals{}
	for _, item := range events {
		if item.EventType != taskEventRunRecovered {
			continue
		}
		var payload taskRecoveryPayload
		if len(item.Payload) > 0 {
			if err := json.Unmarshal(item.Payload, &payload); err != nil {
				continue
			}
		}
		switch payload.Action.Normalize() {
		case taskpkg.RunBootRecoveryRequeue:
			totals.Requeued++
		case taskpkg.RunBootRecoveryMarkRunning:
			totals.MarkedRunning++
		case taskpkg.RunBootRecoveryFail:
			totals.Failed++
		}
	}
	return totals
}

func countEventsByType(events []taskpkg.Event, eventType string) int {
	count := 0
	for _, item := range events {
		if item.EventType == eventType {
			count++
		}
	}
	return count
}

func findStuckRuns(runs []taskpkg.Run, now time.Time, cfg TaskHealthConfig) []StuckTaskRun {
	stuck := make([]StuckTaskRun, 0)
	for _, item := range runs {
		threshold, age, ok := runStuckAge(item, now, cfg)
		if !ok || threshold <= 0 || age < threshold {
			continue
		}
		stuck = append(stuck, StuckTaskRun{
			TaskID:         strings.TrimSpace(item.TaskID),
			RunID:          strings.TrimSpace(item.ID),
			Status:         item.Status.Normalize(),
			OriginKind:     item.Origin.Kind.Normalize(),
			NetworkChannel: strings.TrimSpace(item.NetworkChannel),
			SessionID:      strings.TrimSpace(item.SessionID),
			AgeMillis:      age.Milliseconds(),
		})
	}
	return stuck
}

func runStuckAge(run taskpkg.Run, now time.Time, cfg TaskHealthConfig) (time.Duration, time.Duration, bool) {
	switch run.Status.Normalize() {
	case taskpkg.TaskRunStatusClaimed:
		base := run.ClaimedAt
		if base.IsZero() {
			base = run.QueuedAt
		}
		return cfg.ClaimedStuckAfter, safeSince(now, base), true
	case taskpkg.TaskRunStatusStarting:
		base := run.ClaimedAt
		if base.IsZero() {
			base = run.QueuedAt
		}
		return cfg.StartingStuckAfter, safeSince(now, base), true
	case taskpkg.TaskRunStatusRunning:
		base := run.StartedAt
		if base.IsZero() {
			base = run.ClaimedAt
		}
		if base.IsZero() {
			base = run.QueuedAt
		}
		return cfg.RunningStuckAfter, safeSince(now, base), true
	default:
		return 0, 0, false
	}
}

func safeSince(now time.Time, started time.Time) time.Duration {
	if started.IsZero() {
		return 0
	}
	duration := now.Sub(started)
	if duration < 0 {
		return 0
	}
	return duration
}

func sortStuckRuns(runs []StuckTaskRun) {
	slices.SortFunc(runs, func(left, right StuckTaskRun) int {
		switch {
		case left.AgeMillis > right.AgeMillis:
			return -1
		case left.AgeMillis < right.AgeMillis:
			return 1
		default:
			return strings.Compare(left.RunID, right.RunID)
		}
	})
}

func (o *Observer) countActiveOrphanRuns(ctx context.Context, runs []taskpkg.Run) (int, error) {
	liveSessions, err := o.liveSessionIDs(ctx)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, item := range runs {
		status := item.Status.Normalize()
		if status != taskpkg.TaskRunStatusStarting && status != taskpkg.TaskRunStatusRunning {
			continue
		}
		sessionID := strings.TrimSpace(item.SessionID)
		if sessionID == "" {
			count++
			continue
		}
		if _, ok := liveSessions[sessionID]; !ok {
			count++
		}
	}
	return count, nil
}

func (o *Observer) liveSessionIDs(ctx context.Context) (map[string]struct{}, error) {
	live := make(map[string]struct{})
	if o.sessionSource != nil {
		for _, info := range o.sessionSource.List() {
			if info == nil || !isLiveSessionState(string(info.State)) {
				continue
			}
			live[strings.TrimSpace(info.ID)] = struct{}{}
		}
		return live, nil
	}

	sessions, err := o.registry.ListSessions(ctx, store.SessionListQuery{})
	if err != nil {
		return nil, fmt.Errorf("observe: list sessions for task health: %w", err)
	}
	for _, info := range sessions {
		if !isLiveSessionState(info.State) {
			continue
		}
		live[strings.TrimSpace(info.ID)] = struct{}{}
	}
	return live, nil
}

func isLiveSessionState(state string) bool {
	normalized := strings.TrimSpace(state)
	return normalized != "" && normalized != string(session.StateStopped) && normalized != "orphaned"
}

func filterTasksByOrigin(tasks []taskpkg.Summary, origin taskpkg.OriginKind) []taskpkg.Summary {
	normalizedOrigin := origin.Normalize()
	if normalizedOrigin == "" {
		return tasks
	}
	filtered := make([]taskpkg.Summary, 0, len(tasks))
	for _, item := range tasks {
		if item.Origin.Kind.Normalize() == normalizedOrigin {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterRuns(runs []taskpkg.Run, taskIDs map[string]struct{}, query TaskSummaryQuery) []taskpkg.Run {
	normalizedOrigin := query.OriginKind.Normalize()
	channel := strings.TrimSpace(query.NetworkChannel)
	filtered := make([]taskpkg.Run, 0, len(runs))
	for _, item := range runs {
		if _, ok := taskIDs[strings.TrimSpace(item.TaskID)]; !ok {
			continue
		}
		if normalizedOrigin != "" && item.Origin.Kind.Normalize() != normalizedOrigin {
			continue
		}
		if channel != "" && strings.TrimSpace(item.NetworkChannel) != channel {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func filterRunsByOrigin(runs []taskpkg.Run, origin taskpkg.OriginKind) []taskpkg.Run {
	normalizedOrigin := origin.Normalize()
	if normalizedOrigin == "" {
		return runs
	}
	filtered := make([]taskpkg.Run, 0, len(runs))
	for _, item := range runs {
		if item.Origin.Kind.Normalize() == normalizedOrigin {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterEventsForTasks(events []taskpkg.Event, taskIDs map[string]struct{}) []taskpkg.Event {
	filtered := make([]taskpkg.Event, 0, len(events))
	for _, item := range events {
		if _, ok := taskIDs[strings.TrimSpace(item.TaskID)]; !ok {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func filterTaskEvents(
	events []taskpkg.Event,
	tasksByID map[string]taskpkg.Summary,
	runsByID map[string]taskpkg.Run,
	query TaskMetricsQuery,
) []taskpkg.Event {
	normalizedOrigin := query.OriginKind.Normalize()
	channel := strings.TrimSpace(query.NetworkChannel)
	var filtered []taskpkg.Event
	for i, item := range events {
		accepted := true
		if !query.Since.IsZero() && item.Timestamp.Before(query.Since) {
			accepted = false
		}
		if accepted && normalizedOrigin != "" && item.Origin.Kind.Normalize() != normalizedOrigin {
			accepted = false
		}
		if accepted && channel != "" && eventChannel(item, tasksByID, runsByID) != channel {
			accepted = false
		}
		if accepted {
			if filtered != nil {
				filtered = append(filtered, item)
			}
			continue
		}
		if filtered == nil {
			filtered = make([]taskpkg.Event, 0, len(events)-1)
			filtered = append(filtered, events[:i]...)
		}
	}
	if filtered == nil {
		return events
	}
	return filtered
}

func eventChannel(
	event taskpkg.Event,
	tasksByID map[string]taskpkg.Summary,
	runsByID map[string]taskpkg.Run,
) string {
	if run, ok := runsByID[strings.TrimSpace(event.RunID)]; ok {
		return strings.TrimSpace(run.NetworkChannel)
	}
	if taskItem, ok := tasksByID[strings.TrimSpace(event.TaskID)]; ok {
		return strings.TrimSpace(taskItem.NetworkChannel)
	}
	return ""
}

func filterTaskIngressAudits(audits []store.NetworkAuditEntry, query TaskMetricsQuery) []store.NetworkAuditEntry {
	channel := strings.TrimSpace(query.NetworkChannel)
	normalizedOrigin := query.OriginKind.Normalize()
	var filtered []store.NetworkAuditEntry
	for i, item := range audits {
		accepted := isTaskIngressAudit(item)
		if accepted && normalizedOrigin != "" && normalizedOrigin != taskpkg.OriginKindNetwork {
			accepted = false
		}
		if accepted && !query.Since.IsZero() && item.Timestamp.Before(query.Since) {
			accepted = false
		}
		if accepted && channel != "" && strings.TrimSpace(item.Channel) != channel {
			accepted = false
		}
		if accepted {
			if filtered != nil {
				filtered = append(filtered, item)
			}
			continue
		}
		if filtered == nil {
			filtered = make([]store.NetworkAuditEntry, 0, len(audits)-1)
			filtered = append(filtered, audits[:i]...)
		}
	}
	if filtered == nil {
		return audits
	}
	return filtered
}

func isTaskIngressAudit(entry store.NetworkAuditEntry) bool {
	return strings.HasPrefix(strings.TrimSpace(entry.Kind), "task.")
}

func countNetworkEnqueueEvents(events []taskpkg.Event) int {
	count := 0
	for _, item := range events {
		if item.EventType == taskEventRunEnqueued && item.Origin.Kind.Normalize() == taskpkg.OriginKindNetwork {
			count++
		}
	}
	return count
}

func countAcceptedEnqueueAudits(audits []store.NetworkAuditEntry) int {
	count := 0
	for _, item := range audits {
		if strings.TrimSpace(item.Kind) != taskIngressAuditEnqueueAction {
			continue
		}
		if strings.TrimSpace(item.Direction) != "received" {
			continue
		}
		count++
	}
	return count
}

func countChannelMismatchAudits(audits []store.NetworkAuditEntry) int {
	count := 0
	for _, item := range audits {
		if strings.TrimSpace(item.Direction) != "rejected" {
			continue
		}
		if strings.TrimSpace(item.Reason) == taskIngressChannelMismatch {
			count++
		}
	}
	return count
}
