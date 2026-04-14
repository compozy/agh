package observe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	taskEventCancelled            = "task.cancelled"
	taskEventRunEnqueued          = "task.run_enqueued"
	taskEventRunForceStopped      = "task.run_force_stopped"
	taskEventRunRecovered         = "task.run_recovered"
)

// TaskSummaryQuery filters the current task summary view.
type TaskSummaryQuery struct {
	Scope          taskpkg.Scope      `json:"scope,omitempty"`
	WorkspaceID    string             `json:"workspace_id,omitempty"`
	OwnerKind      taskpkg.OwnerKind  `json:"owner_kind,omitempty"`
	OwnerRef       string             `json:"owner_ref,omitempty"`
	NetworkChannel string             `json:"network_channel,omitempty"`
	OriginKind     taskpkg.OriginKind `json:"origin_kind,omitempty"`
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
	Since          time.Time          `json:"since,omitempty"`
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

// TaskStatusTotal reports one current task-count bucket.
type TaskStatusTotal struct {
	Scope          taskpkg.Scope      `json:"scope"`
	Status         taskpkg.TaskStatus `json:"status"`
	NetworkChannel string             `json:"network_channel,omitempty"`
	Count          int                `json:"count"`
}

// TaskOriginTotal reports one current task-origin bucket.
type TaskOriginTotal struct {
	OriginKind     taskpkg.OriginKind `json:"origin_kind"`
	NetworkChannel string             `json:"network_channel,omitempty"`
	Count          int                `json:"count"`
}

// TaskRunTotal reports one current task-run bucket.
type TaskRunTotal struct {
	Status         taskpkg.TaskRunStatus `json:"status"`
	OriginKind     taskpkg.OriginKind    `json:"origin_kind"`
	NetworkChannel string                `json:"network_channel,omitempty"`
	Count          int                   `json:"count"`
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
	OldestQueuedAt      time.Time `json:"oldest_queued_at,omitempty"`
	OldestQueueAgeMilli int64     `json:"oldest_queue_age_ms"`
}

// TaskSummary exposes the current read-side task summary buckets.
type TaskSummary struct {
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
	TaskID         string                `json:"task_id"`
	RunID          string                `json:"run_id"`
	Status         taskpkg.TaskRunStatus `json:"status"`
	OriginKind     taskpkg.OriginKind    `json:"origin_kind"`
	NetworkChannel string                `json:"network_channel,omitempty"`
	SessionID      string                `json:"session_id,omitempty"`
	AgeMillis      int64                 `json:"age_ms"`
}

// TaskHealth exposes the current operational task-health view.
type TaskHealth struct {
	Status                     string             `json:"status"`
	QueueDepthTotal            int                `json:"queue_depth_total"`
	OldestQueuedAt             time.Time          `json:"oldest_queued_at,omitempty"`
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

type taskSnapshot struct {
	tasks     []taskpkg.TaskSummary
	runs      []taskpkg.TaskRun
	events    []taskpkg.TaskEvent
	audits    []store.NetworkAuditEntry
	tasksByID map[string]taskpkg.TaskSummary
	runsByID  map[string]taskpkg.TaskRun
}

type taskRecoveryPayload struct {
	Action taskpkg.RunBootRecoveryAction `json:"action"`
}

// QueryTaskSummary returns the current task summary buckets filtered by the supplied view.
func (o *Observer) QueryTaskSummary(ctx context.Context, query TaskSummaryQuery) (TaskSummary, error) {
	snapshot, err := o.loadTaskSnapshot(ctx, query)
	if err != nil {
		return TaskSummary{}, err
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

	status := "ok"
	if len(stuckRuns) > 0 || activeOrphans > 0 {
		status = "warn"
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

func taskSummaryFromSnapshot(snapshot taskSnapshot, now func() time.Time) TaskSummary {
	return TaskSummary{
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
	networkEnqueueEvents := filterNetworkEnqueueEvents(events)

	duplicateIngress := len(filterAcceptedEnqueueAudits(audits)) - len(networkEnqueueEvents)
	if duplicateIngress < 0 {
		duplicateIngress = 0
	}

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

func (o *Observer) loadTaskSnapshot(ctx context.Context, query TaskSummaryQuery) (taskSnapshot, error) {
	if ctx == nil {
		return taskSnapshot{}, errors.New("observe: task summary context is required")
	}
	if err := query.Validate(); err != nil {
		return taskSnapshot{}, err
	}

	tasks, err := o.registry.ListTasks(ctx, taskpkg.TaskQuery{
		Scope:          query.Scope,
		WorkspaceID:    strings.TrimSpace(query.WorkspaceID),
		OwnerKind:      query.OwnerKind.Normalize(),
		OwnerRef:       strings.TrimSpace(query.OwnerRef),
		NetworkChannel: strings.TrimSpace(query.NetworkChannel),
	})
	if err != nil {
		return taskSnapshot{}, fmt.Errorf("observe: list tasks for summary: %w", err)
	}
	tasks = filterTasksByOrigin(tasks, query.OriginKind)

	tasksByID := make(map[string]taskpkg.TaskSummary, len(tasks))
	taskIDs := make(map[string]struct{}, len(tasks))
	for _, item := range tasks {
		taskID := strings.TrimSpace(item.ID)
		if taskID == "" {
			continue
		}
		tasksByID[taskID] = item
		taskIDs[taskID] = struct{}{}
	}

	runs, err := o.registry.ListTaskRuns(ctx, taskpkg.TaskRunQuery{})
	if err != nil {
		return taskSnapshot{}, fmt.Errorf("observe: list task runs for summary: %w", err)
	}
	runs = filterRuns(runs, taskIDs, query)

	runsByID := make(map[string]taskpkg.TaskRun, len(runs))
	for _, item := range runs {
		runID := strings.TrimSpace(item.ID)
		if runID == "" {
			continue
		}
		runsByID[runID] = item
	}

	events, err := o.registry.ListTaskEvents(ctx, taskpkg.TaskEventQuery{})
	if err != nil {
		return taskSnapshot{}, fmt.Errorf("observe: list task events for summary: %w", err)
	}
	events = filterEventsForTasks(events, taskIDs)

	audits, err := o.registry.ListNetworkAudit(ctx, store.NetworkAuditQuery{Channel: strings.TrimSpace(query.NetworkChannel)})
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

func summarizeTasks(tasks []taskpkg.TaskSummary) []TaskStatusTotal {
	counts := make(map[string]TaskStatusTotal)
	for _, item := range tasks {
		key := string(item.Scope.Normalize()) + "\x00" + string(item.Status.Normalize()) + "\x00" + strings.TrimSpace(item.NetworkChannel)
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

func summarizeTaskOrigins(tasks []taskpkg.TaskSummary) []TaskOriginTotal {
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

func summarizeRuns(runs []taskpkg.TaskRun) []TaskRunTotal {
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

func summarizeOwners(tasks []taskpkg.TaskSummary) []TaskOwnerTotal {
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

func summarizeQueueDepth(runs []taskpkg.TaskRun, now func() time.Time) []TaskQueueDepth {
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
			age := currentTime.Sub(item.QueuedAt)
			if age < 0 {
				age = 0
			}
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

func summarizeCancelRequests(events []taskpkg.TaskEvent) []TaskCancelRequestTotal {
	counts := make(map[string]TaskCancelRequestTotal)
	for _, item := range events {
		if item.EventType != taskEventCancelled {
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

func summarizeClaimLatency(runs []taskpkg.TaskRun) LatencyMetric {
	return summarizeLatency(runs, func(run taskpkg.TaskRun) (time.Duration, bool) {
		if run.ClaimedAt.IsZero() {
			return 0, false
		}
		duration := run.ClaimedAt.Sub(run.QueuedAt)
		if duration < 0 {
			duration = 0
		}
		return duration, true
	})
}

func summarizeStartLatency(runs []taskpkg.TaskRun) LatencyMetric {
	return summarizeLatency(runs, func(run taskpkg.TaskRun) (time.Duration, bool) {
		if run.StartedAt.IsZero() {
			return 0, false
		}
		base := run.ClaimedAt
		if base.IsZero() {
			base = run.QueuedAt
		}
		duration := run.StartedAt.Sub(base)
		if duration < 0 {
			duration = 0
		}
		return duration, true
	})
}

func summarizeLatency(runs []taskpkg.TaskRun, measure func(taskpkg.TaskRun) (time.Duration, bool)) LatencyMetric {
	var total time.Duration
	var max time.Duration
	samples := 0
	for _, item := range runs {
		duration, ok := measure(item)
		if !ok {
			continue
		}
		samples++
		total += duration
		if duration > max {
			max = duration
		}
	}
	if samples == 0 {
		return LatencyMetric{}
	}
	return LatencyMetric{
		Samples:       samples,
		AverageMillis: (total / time.Duration(samples)).Milliseconds(),
		MaximumMillis: max.Milliseconds(),
	}
}

func summarizeRecovery(events []taskpkg.TaskEvent) TaskRecoveryTotals {
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

func countEventsByType(events []taskpkg.TaskEvent, eventType string) int {
	count := 0
	for _, item := range events {
		if item.EventType == eventType {
			count++
		}
	}
	return count
}

func findStuckRuns(runs []taskpkg.TaskRun, now time.Time, cfg TaskHealthConfig) []StuckTaskRun {
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

func runStuckAge(run taskpkg.TaskRun, now time.Time, cfg TaskHealthConfig) (time.Duration, time.Duration, bool) {
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

func (o *Observer) countActiveOrphanRuns(ctx context.Context, runs []taskpkg.TaskRun) (int, error) {
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

func filterTasksByOrigin(tasks []taskpkg.TaskSummary, origin taskpkg.OriginKind) []taskpkg.TaskSummary {
	if origin.Normalize() == "" {
		return tasks
	}
	filtered := make([]taskpkg.TaskSummary, 0, len(tasks))
	for _, item := range tasks {
		if item.Origin.Kind.Normalize() == origin.Normalize() {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterRuns(runs []taskpkg.TaskRun, taskIDs map[string]struct{}, query TaskSummaryQuery) []taskpkg.TaskRun {
	filtered := make([]taskpkg.TaskRun, 0, len(runs))
	for _, item := range runs {
		if _, ok := taskIDs[strings.TrimSpace(item.TaskID)]; !ok {
			continue
		}
		if query.OriginKind.Normalize() != "" && item.Origin.Kind.Normalize() != query.OriginKind.Normalize() {
			continue
		}
		if channel := strings.TrimSpace(query.NetworkChannel); channel != "" && strings.TrimSpace(item.NetworkChannel) != channel {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func filterRunsByOrigin(runs []taskpkg.TaskRun, origin taskpkg.OriginKind) []taskpkg.TaskRun {
	if origin.Normalize() == "" {
		return runs
	}
	filtered := make([]taskpkg.TaskRun, 0, len(runs))
	for _, item := range runs {
		if item.Origin.Kind.Normalize() == origin.Normalize() {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterEventsForTasks(events []taskpkg.TaskEvent, taskIDs map[string]struct{}) []taskpkg.TaskEvent {
	filtered := make([]taskpkg.TaskEvent, 0, len(events))
	for _, item := range events {
		if _, ok := taskIDs[strings.TrimSpace(item.TaskID)]; !ok {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func filterTaskEvents(
	events []taskpkg.TaskEvent,
	tasksByID map[string]taskpkg.TaskSummary,
	runsByID map[string]taskpkg.TaskRun,
	query TaskMetricsQuery,
) []taskpkg.TaskEvent {
	filtered := make([]taskpkg.TaskEvent, 0, len(events))
	for _, item := range events {
		if !query.Since.IsZero() && item.Timestamp.Before(query.Since) {
			continue
		}
		if query.OriginKind.Normalize() != "" && item.Origin.Kind.Normalize() != query.OriginKind.Normalize() {
			continue
		}
		if channel := strings.TrimSpace(query.NetworkChannel); channel != "" && eventChannel(item, tasksByID, runsByID) != channel {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func eventChannel(event taskpkg.TaskEvent, tasksByID map[string]taskpkg.TaskSummary, runsByID map[string]taskpkg.TaskRun) string {
	if run, ok := runsByID[strings.TrimSpace(event.RunID)]; ok {
		return strings.TrimSpace(run.NetworkChannel)
	}
	if taskItem, ok := tasksByID[strings.TrimSpace(event.TaskID)]; ok {
		return strings.TrimSpace(taskItem.NetworkChannel)
	}
	return ""
}

func filterTaskIngressAudits(audits []store.NetworkAuditEntry, query TaskMetricsQuery) []store.NetworkAuditEntry {
	filtered := make([]store.NetworkAuditEntry, 0, len(audits))
	for _, item := range audits {
		if !isTaskIngressAudit(item) {
			continue
		}
		if !query.Since.IsZero() && item.Timestamp.Before(query.Since) {
			continue
		}
		if channel := strings.TrimSpace(query.NetworkChannel); channel != "" && strings.TrimSpace(item.Channel) != channel {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func isTaskIngressAudit(entry store.NetworkAuditEntry) bool {
	return strings.HasPrefix(strings.TrimSpace(entry.Kind), "task.")
}

func filterNetworkEnqueueEvents(events []taskpkg.TaskEvent) []taskpkg.TaskEvent {
	filtered := make([]taskpkg.TaskEvent, 0, len(events))
	for _, item := range events {
		if item.EventType == taskEventRunEnqueued && item.Origin.Kind.Normalize() == taskpkg.OriginKindNetwork {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterAcceptedEnqueueAudits(audits []store.NetworkAuditEntry) []store.NetworkAuditEntry {
	filtered := make([]store.NetworkAuditEntry, 0, len(audits))
	for _, item := range audits {
		if strings.TrimSpace(item.Kind) != taskIngressAuditEnqueueAction {
			continue
		}
		if strings.TrimSpace(item.Direction) != "received" {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
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
