package observe

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestQueryTaskSummaryAggregatesByScopeOriginChannelAndOwner(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	createObserveTask(t, h, taskpkg.Task{
		ID:        "task-global-ready",
		Scope:     taskpkg.ScopeGlobal,
		Title:     "Global ready",
		Status:    taskpkg.TaskStatusReady,
		CreatedBy: taskActor(taskpkg.ActorKindHuman, "user-1"),
		Origin:    taskOrigin(taskpkg.OriginKindCLI, "agh task create"),
		CreatedAt: h.now.Add(time.Minute),
		UpdatedAt: h.now.Add(time.Minute),
	})
	createObserveTask(t, h, taskpkg.Task{
		ID:             "task-workspace-blocked",
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    h.workspaceID,
		Title:          "Workspace blocked",
		Status:         taskpkg.TaskStatusBlocked,
		Owner:          taskOwner(taskpkg.OwnerKindHuman, "alice"),
		NetworkChannel: "ops",
		CreatedBy:      taskActor(taskpkg.ActorKindNetworkPeer, "peer-ops"),
		Origin:         taskOrigin(taskpkg.OriginKindNetwork, "peer:peer-ops/channel:ops"),
		CreatedAt:      h.now.Add(2 * time.Minute),
		UpdatedAt:      h.now.Add(2 * time.Minute),
	})
	createObserveTask(t, h, taskpkg.Task{
		ID:             "task-workspace-completed",
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    h.workspaceID,
		Title:          "Workspace completed",
		Status:         taskpkg.TaskStatusCompleted,
		Owner:          taskOwner(taskpkg.OwnerKindPool, "backlog"),
		NetworkChannel: "eng",
		CreatedBy:      taskActor(taskpkg.ActorKindAutomation, "rule-1"),
		Origin:         taskOrigin(taskpkg.OriginKindAutomation, "run:rule-1"),
		CreatedAt:      h.now.Add(3 * time.Minute),
		UpdatedAt:      h.now.Add(3 * time.Minute),
		ClosedAt:       h.now.Add(4 * time.Minute),
	})

	createObserveRun(t, h, taskpkg.Run{
		ID:       "run-global-queued",
		TaskID:   "task-global-ready",
		Status:   taskpkg.TaskRunStatusQueued,
		Attempt:  1,
		Origin:   taskOrigin(taskpkg.OriginKindCLI, "agh task run"),
		QueuedAt: h.now.Add(10 * time.Minute),
	})
	createObserveRun(t, h, taskpkg.Run{
		ID:             "run-workspace-running",
		TaskID:         "task-workspace-blocked",
		Status:         taskpkg.TaskRunStatusRunning,
		Attempt:        1,
		ClaimedBy:      taskActorPtr(taskpkg.ActorKindDaemon, "scheduler"),
		SessionID:      "sess-ops-live",
		Origin:         taskOrigin(taskpkg.OriginKindNetwork, "peer:peer-ops/channel:ops"),
		NetworkChannel: "ops",
		QueuedAt:       h.now.Add(11 * time.Minute),
		ClaimedAt:      h.now.Add(12 * time.Minute),
		StartedAt:      h.now.Add(13 * time.Minute),
	})
	createObserveRun(t, h, taskpkg.Run{
		ID:             "run-workspace-completed",
		TaskID:         "task-workspace-completed",
		Status:         taskpkg.TaskRunStatusCompleted,
		Attempt:        1,
		ClaimedBy:      taskActorPtr(taskpkg.ActorKindDaemon, "scheduler"),
		SessionID:      "sess-eng-done",
		Origin:         taskOrigin(taskpkg.OriginKindAutomation, "run:rule-1"),
		NetworkChannel: "eng",
		QueuedAt:       h.now.Add(14 * time.Minute),
		ClaimedAt:      h.now.Add(15 * time.Minute),
		StartedAt:      h.now.Add(16 * time.Minute),
		EndedAt:        h.now.Add(18 * time.Minute),
	})

	summary, err := h.observer.QueryTaskSummary(testutil.Context(t), TaskSummaryQuery{})
	if err != nil {
		t.Fatalf("QueryTaskSummary() error = %v", err)
	}

	if got, want := summary.TotalTasks, 3; got != want {
		t.Fatalf("summary.TotalTasks = %d, want %d", got, want)
	}
	if got, want := summary.TotalRuns, 3; got != want {
		t.Fatalf("summary.TotalRuns = %d, want %d", got, want)
	}
	if !containsTaskTotal(summary.TaskTotals, taskpkg.ScopeGlobal, taskpkg.TaskStatusReady, "", 1) {
		t.Fatalf("summary.TaskTotals = %#v, want global/ready/unbound count 1", summary.TaskTotals)
	}
	if !containsTaskTotal(summary.TaskTotals, taskpkg.ScopeWorkspace, taskpkg.TaskStatusBlocked, "ops", 1) {
		t.Fatalf("summary.TaskTotals = %#v, want workspace/blocked/ops count 1", summary.TaskTotals)
	}
	if !containsTaskOriginTotal(summary.TaskOrigins, taskpkg.OriginKindNetwork, "ops", 1) {
		t.Fatalf("summary.TaskOrigins = %#v, want network/ops count 1", summary.TaskOrigins)
	}
	if !containsRunTotal(summary.RunTotals, taskpkg.TaskRunStatusCompleted, taskpkg.OriginKindAutomation, "eng", 1) {
		t.Fatalf("summary.RunTotals = %#v, want completed/automation/eng count 1", summary.RunTotals)
	}
	if !containsOwnerTotal(summary.OwnerTotals, taskpkg.OwnerKindHuman, "alice", 1) {
		t.Fatalf("summary.OwnerTotals = %#v, want human/alice count 1", summary.OwnerTotals)
	}
	if !containsQueueDepth(summary.QueueDepth, "", 1) {
		t.Fatalf("summary.QueueDepth = %#v, want unbound queue depth 1", summary.QueueDepth)
	}

	filtered, err := h.observer.QueryTaskSummary(testutil.Context(t), TaskSummaryQuery{
		OwnerKind:      taskpkg.OwnerKindHuman,
		OwnerRef:       "alice",
		NetworkChannel: "ops",
	})
	if err != nil {
		t.Fatalf("QueryTaskSummary(filtered) error = %v", err)
	}
	if got, want := filtered.TotalTasks, 1; got != want {
		t.Fatalf("filtered.TotalTasks = %d, want %d", got, want)
	}
	if got, want := filtered.TotalRuns, 1; got != want {
		t.Fatalf("filtered.TotalRuns = %d, want %d", got, want)
	}
	if !containsRunTotal(filtered.RunTotals, taskpkg.TaskRunStatusRunning, taskpkg.OriginKindNetwork, "ops", 1) {
		t.Fatalf("filtered.RunTotals = %#v, want running/network/ops count 1", filtered.RunTotals)
	}
}

func TestTaskHealthFlagsStuckRunsByConfiguredThresholds(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.observer.taskHealthConfig = TaskHealthConfig{
		ClaimedStuckAfter:  5 * time.Minute,
		StartingStuckAfter: 10 * time.Minute,
		RunningStuckAfter:  15 * time.Minute,
	}

	liveStartedAt := h.observer.now().Add(-5 * time.Minute)
	h.source.sessions = []*session.Info{
		{
			ID:           "sess-live-running",
			Name:         "LIVE",
			AgentName:    "coder",
			WorkspaceID:  h.workspaceID,
			Workspace:    h.workspace,
			State:        session.StateActive,
			ACPSessionID: "acp-live-running",
			CreatedAt:    liveStartedAt,
			UpdatedAt:    liveStartedAt,
		},
		{
			ID:           "sess-live-starting",
			Name:         "LIVE2",
			AgentName:    "coder",
			WorkspaceID:  h.workspaceID,
			Workspace:    h.workspace,
			State:        session.StateActive,
			ACPSessionID: "acp-live-starting",
			CreatedAt:    liveStartedAt,
			UpdatedAt:    liveStartedAt,
		},
	}

	taskIDs := []string{"task-claimed", "task-starting-recent", "task-starting-stale", "task-running-stale"}
	for _, id := range taskIDs {
		createObserveTask(t, h, taskpkg.Task{
			ID:          id,
			Scope:       taskpkg.ScopeWorkspace,
			WorkspaceID: h.workspaceID,
			Title:       id,
			Status:      taskpkg.TaskStatusInProgress,
			CreatedBy:   taskActor(taskpkg.ActorKindHuman, "user"),
			Origin:      taskOrigin(taskpkg.OriginKindCLI, "agh task"),
			CreatedAt:   h.now,
			UpdatedAt:   h.now,
		})
	}

	now := h.observer.now()
	createObserveRun(t, h, taskpkg.Run{
		ID:        "run-claimed-stale",
		TaskID:    "task-claimed",
		Status:    taskpkg.TaskRunStatusClaimed,
		Attempt:   1,
		Origin:    taskOrigin(taskpkg.OriginKindCLI, "agh task"),
		QueuedAt:  now.Add(-40 * time.Minute),
		ClaimedAt: now.Add(-20 * time.Minute),
	})
	createObserveRun(t, h, taskpkg.Run{
		ID:             "run-starting-fresh",
		TaskID:         "task-starting-recent",
		Status:         taskpkg.TaskRunStatusStarting,
		Attempt:        1,
		SessionID:      "sess-live-starting",
		Origin:         taskOrigin(taskpkg.OriginKindCLI, "agh task"),
		QueuedAt:       now.Add(-15 * time.Minute),
		ClaimedAt:      now.Add(-4 * time.Minute),
		NetworkChannel: "ops",
	})
	createObserveRun(t, h, taskpkg.Run{
		ID:             "run-starting-stale",
		TaskID:         "task-starting-stale",
		Status:         taskpkg.TaskRunStatusStarting,
		Attempt:        1,
		SessionID:      "sess-live-starting",
		Origin:         taskOrigin(taskpkg.OriginKindNetwork, "peer:peer-1/channel:ops"),
		QueuedAt:       now.Add(-25 * time.Minute),
		ClaimedAt:      now.Add(-12 * time.Minute),
		NetworkChannel: "ops",
	})
	createObserveRun(t, h, taskpkg.Run{
		ID:             "run-running-stale",
		TaskID:         "task-running-stale",
		Status:         taskpkg.TaskRunStatusRunning,
		Attempt:        1,
		SessionID:      "sess-live-running",
		Origin:         taskOrigin(taskpkg.OriginKindAutomation, "run:auto-1"),
		QueuedAt:       now.Add(-30 * time.Minute),
		ClaimedAt:      now.Add(-28 * time.Minute),
		StartedAt:      now.Add(-20 * time.Minute),
		NetworkChannel: "eng",
	})

	health, err := h.observer.Health(testutil.Context(t))
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	if got, want := len(health.Tasks.StuckRuns), 3; got != want {
		t.Fatalf("len(health.Tasks.StuckRuns) = %d, want %d", got, want)
	}
	if !containsStuckRun(health.Tasks.StuckRuns, "run-claimed-stale", taskpkg.TaskRunStatusClaimed) {
		t.Fatalf("health.Tasks.StuckRuns = %#v, want claimed stale run", health.Tasks.StuckRuns)
	}
	if !containsStuckRun(health.Tasks.StuckRuns, "run-starting-stale", taskpkg.TaskRunStatusStarting) {
		t.Fatalf("health.Tasks.StuckRuns = %#v, want starting stale run", health.Tasks.StuckRuns)
	}
	if !containsStuckRun(health.Tasks.StuckRuns, "run-running-stale", taskpkg.TaskRunStatusRunning) {
		t.Fatalf("health.Tasks.StuckRuns = %#v, want running stale run", health.Tasks.StuckRuns)
	}
	if containsStuckRun(health.Tasks.StuckRuns, "run-starting-fresh", taskpkg.TaskRunStatusStarting) {
		t.Fatalf("health.Tasks.StuckRuns = %#v, fresh starting run should not be flagged", health.Tasks.StuckRuns)
	}
	if got, want := health.Tasks.ActiveOrphanRuns, 0; got != want {
		t.Fatalf("health.Tasks.ActiveOrphanRuns = %d, want %d", got, want)
	}
	if got, want := health.Tasks.Status, "warn"; got != want {
		t.Fatalf("health.Tasks.Status = %q, want %q", got, want)
	}
}

func TestQueryTaskMetricsCountsDuplicateIngressAndChannelMismatch(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	createObserveTask(t, h, taskpkg.Task{
		ID:             "task-net",
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    h.workspaceID,
		Title:          "Network task",
		Status:         taskpkg.TaskStatusReady,
		NetworkChannel: "ops",
		CreatedBy:      taskActor(taskpkg.ActorKindNetworkPeer, "peer-ops"),
		Origin:         taskOrigin(taskpkg.OriginKindNetwork, "peer:peer-ops/channel:ops"),
		CreatedAt:      h.now,
		UpdatedAt:      h.now,
	})
	createObserveRun(t, h, taskpkg.Run{
		ID:             "run-net",
		TaskID:         "task-net",
		Status:         taskpkg.TaskRunStatusQueued,
		Attempt:        1,
		Origin:         taskOrigin(taskpkg.OriginKindNetwork, "peer:peer-ops/channel:ops"),
		NetworkChannel: "ops",
		IdempotencyKey: "idem-1",
		QueuedAt:       h.now.Add(time.Minute),
	})
	createObserveEvent(t, h, taskpkg.Event{
		ID:        "evt-run-enqueued",
		TaskID:    "task-net",
		RunID:     "run-net",
		EventType: taskEventRunEnqueued,
		Actor:     taskActor(taskpkg.ActorKindNetworkPeer, "peer-ops"),
		Origin:    taskOrigin(taskpkg.OriginKindNetwork, "peer:peer-ops/channel:ops"),
		Timestamp: h.now.Add(2 * time.Minute),
		Payload:   mustJSON(t, map[string]any{"network_channel": "ops", "idempotency_key": "idem-1"}),
	})
	createObserveAudit(t, h, store.NetworkAuditEntry{
		ID:        "naud-accepted-1",
		SessionID: "netpeer:peer-ops",
		Direction: "received",
		Kind:      taskIngressAuditEnqueueAction,
		Channel:   "ops",
		PeerFrom:  "peer-ops",
		MessageID: "req-1",
		Size:      32,
		Timestamp: h.now.Add(2 * time.Minute),
	})
	createObserveAudit(t, h, store.NetworkAuditEntry{
		ID:        "naud-accepted-2",
		SessionID: "netpeer:peer-ops",
		Direction: "received",
		Kind:      taskIngressAuditEnqueueAction,
		Channel:   "ops",
		PeerFrom:  "peer-ops",
		MessageID: "req-2",
		Size:      32,
		Timestamp: h.now.Add(3 * time.Minute),
	})
	createObserveAudit(t, h, store.NetworkAuditEntry{
		ID:        "naud-rejected-mismatch",
		SessionID: "netpeer:peer-ops",
		Direction: "rejected",
		Kind:      taskIngressAuditEnqueueAction,
		Channel:   "ops",
		PeerFrom:  "peer-ops",
		MessageID: "req-3",
		Reason:    taskIngressChannelMismatch,
		Size:      32,
		Timestamp: h.now.Add(4 * time.Minute),
	})
	createObserveAudit(t, h, store.NetworkAuditEntry{
		ID:        "naud-rejected-stale",
		SessionID: "netpeer:peer-ops",
		Direction: "rejected",
		Kind:      taskIngressAuditEnqueueAction,
		Channel:   "ops",
		PeerFrom:  "peer-ops",
		MessageID: "req-4",
		Reason:    "stale_channel",
		Size:      32,
		Timestamp: h.now.Add(5 * time.Minute),
	})

	metrics, err := h.observer.QueryTaskMetrics(testutil.Context(t), TaskMetricsQuery{
		Since:          h.now,
		NetworkChannel: "ops",
	})
	if err != nil {
		t.Fatalf("QueryTaskMetrics() error = %v", err)
	}

	if got, want := metrics.DuplicateIngressTotal, 1; got != want {
		t.Fatalf("metrics.DuplicateIngressTotal = %d, want %d", got, want)
	}
	if got, want := metrics.ChannelMismatchTotal, 1; got != want {
		t.Fatalf("metrics.ChannelMismatchTotal = %d, want %d", got, want)
	}
	if !containsRunTotal(metrics.TaskRunsTotal, taskpkg.TaskRunStatusQueued, taskpkg.OriginKindNetwork, "ops", 1) {
		t.Fatalf("metrics.TaskRunsTotal = %#v, want queued/network/ops count 1", metrics.TaskRunsTotal)
	}
	if !containsQueueDepth(metrics.TaskQueueDepth, "ops", 1) {
		t.Fatalf("metrics.TaskQueueDepth = %#v, want ops queue depth 1", metrics.TaskQueueDepth)
	}

	cliMetrics, err := h.observer.QueryTaskMetrics(testutil.Context(t), TaskMetricsQuery{
		Since:          h.now,
		NetworkChannel: "ops",
		OriginKind:     taskpkg.OriginKindCLI,
	})
	if err != nil {
		t.Fatalf("QueryTaskMetrics(cli filter) error = %v", err)
	}
	if got := cliMetrics.DuplicateIngressTotal; got != 0 {
		t.Fatalf("cliMetrics.DuplicateIngressTotal = %d, want 0", got)
	}
	if got := cliMetrics.ChannelMismatchTotal; got != 0 {
		t.Fatalf("cliMetrics.ChannelMismatchTotal = %d, want 0", got)
	}
}

func TestTaskObserveQueryValidationAndConfigOption(t *testing.T) {
	t.Parallel()

	cfg := TaskHealthConfig{
		ClaimedStuckAfter:  time.Minute,
		StartingStuckAfter: 2 * time.Minute,
		RunningStuckAfter:  3 * time.Minute,
	}

	observer := &Observer{}
	WithTaskHealthConfig(cfg)(observer)

	if observer.taskHealthConfig != cfg {
		t.Fatalf("observer.taskHealthConfig = %#v, want %#v", observer.taskHealthConfig, cfg)
	}

	if err := (TaskSummaryQuery{Scope: taskpkg.Scope("bogus")}).Validate(); !errors.Is(err, taskpkg.ErrValidation) ||
		!strings.Contains(err.Error(), "scope") {
		t.Fatalf("TaskSummaryQuery.Validate(invalid scope) error = %v, want ErrValidation mentioning scope", err)
	}
	if err := (TaskSummaryQuery{OwnerKind: taskpkg.OwnerKind("bogus")}).Validate(); !errors.Is(
		err,
		taskpkg.ErrValidation,
	) ||
		!strings.Contains(err.Error(), "owner_kind") {
		t.Fatalf(
			"TaskSummaryQuery.Validate(invalid owner kind) error = %v, want ErrValidation mentioning owner_kind",
			err,
		)
	}
	if err := (TaskMetricsQuery{OriginKind: taskpkg.OriginKind("bogus")}).Validate(); !errors.Is(
		err,
		taskpkg.ErrValidation,
	) ||
		!strings.Contains(err.Error(), "origin_kind") {
		t.Fatalf(
			"TaskMetricsQuery.Validate(invalid origin kind) error = %v, want ErrValidation mentioning origin_kind",
			err,
		)
	}
}

func TestObserverHealthWrapsTaskHealthErrors(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.observer.bridgeSource = nil
	if err := h.registry.Close(testutil.Context(t)); err != nil {
		t.Fatalf("registry.Close() error = %v", err)
	}

	_, err := h.observer.Health(testutil.Context(t))
	if err == nil {
		t.Fatal("Health() error = nil, want wrapped task health failure")
	}
	if !strings.Contains(err.Error(), "observe: collect task health") {
		t.Fatalf("Health() error = %v, want collect task health context", err)
	}
}

func createObserveTask(t *testing.T, h *harness, record taskpkg.Task) {
	t.Helper()
	if err := h.registry.CreateTask(testutil.Context(t), record); err != nil {
		t.Fatalf("CreateTask(%q) error = %v", record.ID, err)
	}
}

func createObserveRun(t *testing.T, h *harness, run taskpkg.Run) {
	t.Helper()
	if err := h.registry.CreateTaskRun(testutil.Context(t), run); err != nil {
		t.Fatalf("CreateTaskRun(%q) error = %v", run.ID, err)
	}
}

func createObserveEvent(t *testing.T, h *harness, event taskpkg.Event) {
	t.Helper()
	if err := h.registry.CreateTaskEvent(testutil.Context(t), event); err != nil {
		t.Fatalf("CreateTaskEvent(%q) error = %v", event.ID, err)
	}
}

func createObserveAudit(t *testing.T, h *harness, entry store.NetworkAuditEntry) {
	t.Helper()
	if err := h.registry.WriteNetworkAudit(testutil.Context(t), entry); err != nil {
		t.Fatalf("WriteNetworkAudit(%q) error = %v", entry.ID, err)
	}
}

func taskActor(kind taskpkg.ActorKind, ref string) taskpkg.ActorIdentity {
	return taskpkg.ActorIdentity{Kind: kind, Ref: ref}
}

func taskActorPtr(kind taskpkg.ActorKind, ref string) *taskpkg.ActorIdentity {
	actor := taskActor(kind, ref)
	return &actor
}

func taskOrigin(kind taskpkg.OriginKind, ref string) taskpkg.Origin {
	return taskpkg.Origin{Kind: kind, Ref: ref}
}

func taskOwner(kind taskpkg.OwnerKind, ref string) *taskpkg.Ownership {
	return &taskpkg.Ownership{Kind: kind, Ref: ref}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return raw
}

func containsTaskTotal(
	rows []TaskStatusTotal,
	scope taskpkg.Scope,
	status taskpkg.Status,
	channel string,
	count int,
) bool {
	for _, item := range rows {
		if item.Scope == scope && item.Status == status && item.NetworkChannel == channel && item.Count == count {
			return true
		}
	}
	return false
}

func containsTaskOriginTotal(rows []TaskOriginTotal, origin taskpkg.OriginKind, channel string, count int) bool {
	for _, item := range rows {
		if item.OriginKind == origin && item.NetworkChannel == channel && item.Count == count {
			return true
		}
	}
	return false
}

func containsRunTotal(
	rows []TaskRunTotal,
	status taskpkg.RunStatus,
	origin taskpkg.OriginKind,
	channel string,
	count int,
) bool {
	for _, item := range rows {
		if item.Status == status && item.OriginKind == origin && item.NetworkChannel == channel && item.Count == count {
			return true
		}
	}
	return false
}

func containsOwnerTotal(rows []TaskOwnerTotal, ownerKind taskpkg.OwnerKind, ownerRef string, count int) bool {
	for _, item := range rows {
		if item.OwnerKind == ownerKind && item.OwnerRef == ownerRef && item.Count == count {
			return true
		}
	}
	return false
}

func containsQueueDepth(rows []TaskQueueDepth, channel string, count int) bool {
	for _, item := range rows {
		if item.NetworkChannel == channel && item.Count == count {
			return true
		}
	}
	return false
}

func containsStuckRun(rows []StuckTaskRun, runID string, status taskpkg.RunStatus) bool {
	for _, item := range rows {
		if item.RunID == runID && item.Status == status {
			return true
		}
	}
	return false
}
