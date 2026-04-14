//go:build integration

package observe

import (
	"context"
	"strconv"
	"testing"
	"time"

	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
)

type observeTaskClock struct {
	current time.Time
}

func (c *observeTaskClock) Now() time.Time {
	return c.current
}

func (c *observeTaskClock) Advance(duration time.Duration) {
	c.current = c.current.Add(duration)
}

type observeSessionExecutor struct {
	nextSessionID    string
	startCalls       []taskpkg.StartTaskSession
	attachCalls      []string
	requestStopCalls []taskStopCall
	forceStopCalls   []taskStopCall
}

type taskStopCall struct {
	SessionID string
	Reason    taskpkg.StopReason
}

func (e *observeSessionExecutor) StartTaskSession(_ context.Context, spec taskpkg.StartTaskSession) (*taskpkg.SessionRef, error) {
	e.startCalls = append(e.startCalls, spec)
	if e.nextSessionID == "" {
		e.nextSessionID = "sess-observe-1"
	}
	return &taskpkg.SessionRef{SessionID: e.nextSessionID, StartedAt: spec.Run.StartedAt}, nil
}

func (e *observeSessionExecutor) AttachTaskSession(_ context.Context, _ string, sessionID string) (*taskpkg.SessionRef, error) {
	e.attachCalls = append(e.attachCalls, sessionID)
	return &taskpkg.SessionRef{SessionID: sessionID}, nil
}

func (e *observeSessionExecutor) RequestTaskStop(_ context.Context, sessionID string, reason taskpkg.StopReason) error {
	e.requestStopCalls = append(e.requestStopCalls, taskStopCall{SessionID: sessionID, Reason: reason})
	return nil
}

func (e *observeSessionExecutor) ForceTaskStop(_ context.Context, sessionID string, reason taskpkg.StopReason) error {
	e.forceStopCalls = append(e.forceStopCalls, taskStopCall{SessionID: sessionID, Reason: reason})
	return nil
}

func TestObserveTaskLifecycleSummaryAndMetrics(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	clock := &observeTaskClock{current: h.now.Add(30 * time.Minute)}
	executor := &observeSessionExecutor{nextSessionID: "sess-observe-lifecycle"}
	manager := newObserveTaskManager(t, h, executor, clock)

	networkActor, err := taskpkg.DeriveNetworkPeerActorContext("peer-build", "peer:peer-build/channel:engineering")
	if err != nil {
		t.Fatalf("DeriveNetworkPeerActorContext() error = %v", err)
	}
	daemonActor, err := taskpkg.DeriveDaemonActorContext("scheduler", "daemon.scheduler")
	if err != nil {
		t.Fatalf("DeriveDaemonActorContext() error = %v", err)
	}

	created, err := manager.CreateTask(testutil.Context(t), taskpkg.CreateTask{
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    h.workspaceID,
		Title:          "Implement observe lifecycle coverage",
		NetworkChannel: "engineering",
	}, networkActor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	clock.Advance(2 * time.Minute)
	run, err := manager.EnqueueRun(testutil.Context(t), taskpkg.EnqueueRun{
		TaskID:         created.ID,
		IdempotencyKey: "idem-observe-1",
	}, networkActor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}

	clock.Advance(2 * time.Minute)
	if _, err := manager.ClaimRun(testutil.Context(t), run.ID, taskpkg.ClaimRun{IdempotencyKey: "claim-observe-1"}, daemonActor); err != nil {
		t.Fatalf("ClaimRun() error = %v", err)
	}

	clock.Advance(3 * time.Minute)
	if _, err := manager.StartRun(testutil.Context(t), run.ID, taskpkg.StartRun{IdempotencyKey: "start-observe-1"}, daemonActor); err != nil {
		t.Fatalf("StartRun() error = %v", err)
	}

	clock.Advance(4 * time.Minute)
	if _, err := manager.CompleteRun(testutil.Context(t), run.ID, taskpkg.RunResult{}, daemonActor); err != nil {
		t.Fatalf("CompleteRun() error = %v", err)
	}

	summary, err := h.observer.QueryTaskSummary(testutil.Context(t), TaskSummaryQuery{})
	if err != nil {
		t.Fatalf("QueryTaskSummary() error = %v", err)
	}
	if !containsTaskTotal(summary.TaskTotals, taskpkg.ScopeWorkspace, taskpkg.TaskStatusCompleted, "engineering", 1) {
		t.Fatalf("summary.TaskTotals = %#v, want workspace/completed/engineering count 1", summary.TaskTotals)
	}
	if !containsTaskOriginTotal(summary.TaskOrigins, taskpkg.OriginKindNetwork, "engineering", 1) {
		t.Fatalf("summary.TaskOrigins = %#v, want network/engineering count 1", summary.TaskOrigins)
	}
	if !containsRunTotal(summary.RunTotals, taskpkg.TaskRunStatusCompleted, taskpkg.OriginKindNetwork, "engineering", 1) {
		t.Fatalf("summary.RunTotals = %#v, want completed/network/engineering count 1", summary.RunTotals)
	}

	metrics, err := h.observer.QueryTaskMetrics(testutil.Context(t), TaskMetricsQuery{})
	if err != nil {
		t.Fatalf("QueryTaskMetrics() error = %v", err)
	}
	if !containsRunTotal(metrics.TaskRunsTotal, taskpkg.TaskRunStatusCompleted, taskpkg.OriginKindNetwork, "engineering", 1) {
		t.Fatalf("metrics.TaskRunsTotal = %#v, want completed/network/engineering count 1", metrics.TaskRunsTotal)
	}
	if got, want := metrics.TaskClaimLatencyMillis.Samples, 1; got != want {
		t.Fatalf("metrics.TaskClaimLatencyMillis.Samples = %d, want %d", got, want)
	}
	if got, want := metrics.TaskClaimLatencyMillis.AverageMillis, int64((2 * time.Minute).Milliseconds()); got != want {
		t.Fatalf("metrics.TaskClaimLatencyMillis.AverageMillis = %d, want %d", got, want)
	}
	if got, want := metrics.TaskStartLatencyMillis.AverageMillis, int64((3 * time.Minute).Milliseconds()); got != want {
		t.Fatalf("metrics.TaskStartLatencyMillis.AverageMillis = %d, want %d", got, want)
	}
}

func TestObserveHealthReflectsRecoveryAndForcedStopOutcomes(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	clock := &observeTaskClock{current: h.now.Add(45 * time.Minute)}
	executor := &observeSessionExecutor{nextSessionID: "sess-observe-cancel"}
	manager := newObserveTaskManager(t, h, executor, clock)

	humanActor, err := taskpkg.DeriveHumanActorContext("user-ops", taskpkg.OriginKindCLI, "agh task")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}
	daemonActor, err := taskpkg.DeriveDaemonActorContext("scheduler", "daemon.scheduler")
	if err != nil {
		t.Fatalf("DeriveDaemonActorContext() error = %v", err)
	}
	recoveryActor, err := taskpkg.DeriveDaemonActorContext("boot-recovery", "daemon.boot")
	if err != nil {
		t.Fatalf("DeriveDaemonActorContext(boot) error = %v", err)
	}

	cancelTask, err := manager.CreateTask(testutil.Context(t), taskpkg.CreateTask{
		Scope:       taskpkg.ScopeWorkspace,
		WorkspaceID: h.workspaceID,
		Title:       "Cancel running work",
	}, humanActor)
	if err != nil {
		t.Fatalf("CreateTask(cancelTask) error = %v", err)
	}
	recoveryTask, err := manager.CreateTask(testutil.Context(t), taskpkg.CreateTask{
		Scope:       taskpkg.ScopeWorkspace,
		WorkspaceID: h.workspaceID,
		Title:       "Recover orphaned run",
	}, humanActor)
	if err != nil {
		t.Fatalf("CreateTask(recoveryTask) error = %v", err)
	}

	clock.Advance(time.Minute)
	runningRun, err := manager.EnqueueRun(testutil.Context(t), taskpkg.EnqueueRun{TaskID: cancelTask.ID}, humanActor)
	if err != nil {
		t.Fatalf("EnqueueRun(running) error = %v", err)
	}
	clock.Advance(time.Minute)
	if _, err := manager.ClaimRun(testutil.Context(t), runningRun.ID, taskpkg.ClaimRun{IdempotencyKey: "claim-cancel-1"}, daemonActor); err != nil {
		t.Fatalf("ClaimRun(running) error = %v", err)
	}
	clock.Advance(time.Minute)
	if _, err := manager.StartRun(testutil.Context(t), runningRun.ID, taskpkg.StartRun{IdempotencyKey: "start-cancel-1"}, daemonActor); err != nil {
		t.Fatalf("StartRun(running) error = %v", err)
	}
	clock.Advance(time.Minute)
	if _, err := manager.CancelTask(testutil.Context(t), cancelTask.ID, taskpkg.CancelTask{Reason: "shutdown"}, humanActor); err != nil {
		t.Fatalf("CancelTask() error = %v", err)
	}

	executor.nextSessionID = "sess-observe-attach"
	clock.Advance(time.Minute)
	recoveredRun, err := manager.EnqueueRun(testutil.Context(t), taskpkg.EnqueueRun{TaskID: recoveryTask.ID}, humanActor)
	if err != nil {
		t.Fatalf("EnqueueRun(recovery) error = %v", err)
	}
	clock.Advance(time.Minute)
	if _, err := manager.ClaimRun(testutil.Context(t), recoveredRun.ID, taskpkg.ClaimRun{IdempotencyKey: "claim-recovery-1"}, daemonActor); err != nil {
		t.Fatalf("ClaimRun(recovery) error = %v", err)
	}
	clock.Advance(time.Minute)
	if _, err := manager.AttachRunSession(testutil.Context(t), recoveredRun.ID, "sess-missing-on-boot", daemonActor); err != nil {
		t.Fatalf("AttachRunSession() error = %v", err)
	}
	clock.Advance(time.Minute)
	if _, err := manager.RecoverRunOnBoot(testutil.Context(t), recoveredRun.ID, taskpkg.RunBootRecovery{
		Action:       taskpkg.RunBootRecoveryFail,
		Reason:       "orphaned_on_boot",
		SessionState: "missing",
	}, recoveryActor); err != nil {
		t.Fatalf("RecoverRunOnBoot() error = %v", err)
	}

	health, err := h.observer.Health(testutil.Context(t))
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	if got, want := len(executor.forceStopCalls), 1; got != want {
		t.Fatalf("len(forceStopCalls) = %d, want %d", got, want)
	}
	if got, want := health.Tasks.ForcedStopsSinceStart, 1; got != want {
		t.Fatalf("health.Tasks.ForcedStopsSinceStart = %d, want %d", got, want)
	}
	if got, want := health.Tasks.RecoverySinceStart.Failed, 1; got != want {
		t.Fatalf("health.Tasks.RecoverySinceStart.Failed = %d, want %d", got, want)
	}
	if !containsTaskTotal(health.Tasks.TaskTotals, taskpkg.ScopeWorkspace, taskpkg.TaskStatusCancelled, "", 1) {
		t.Fatalf("health.Tasks.TaskTotals = %#v, want cancelled task count 1", health.Tasks.TaskTotals)
	}
	if !containsTaskTotal(health.Tasks.TaskTotals, taskpkg.ScopeWorkspace, taskpkg.TaskStatusFailed, "", 1) {
		t.Fatalf("health.Tasks.TaskTotals = %#v, want failed task count 1", health.Tasks.TaskTotals)
	}
	if !containsRunTotal(health.Tasks.RunTotals, taskpkg.TaskRunStatusCancelled, taskpkg.OriginKindCLI, "", 1) {
		t.Fatalf("health.Tasks.RunTotals = %#v, want cancelled/cli run count 1", health.Tasks.RunTotals)
	}
	if !containsRunTotal(health.Tasks.RunTotals, taskpkg.TaskRunStatusFailed, taskpkg.OriginKindCLI, "", 1) {
		t.Fatalf("health.Tasks.RunTotals = %#v, want failed/cli run count 1", health.Tasks.RunTotals)
	}
}

func newObserveTaskManager(t *testing.T, h *harness, executor *observeSessionExecutor, clock *observeTaskClock) *taskpkg.TaskManager {
	t.Helper()

	sequence := 0
	manager, err := taskpkg.NewManager(
		taskpkg.WithStore(h.registry),
		taskpkg.WithSessionExecutor(executor),
		taskpkg.WithManagerNow(clock.Now),
		taskpkg.WithIDGenerator(func(prefix string) string {
			sequence++
			return prefix + "-observe-" + strconv.FormatInt(clock.current.UnixNano(), 10) + "-" + strconv.Itoa(sequence)
		}),
		taskpkg.WithCancelGracePeriod(0),
	)
	if err != nil {
		t.Fatalf("task.NewManager() error = %v", err)
	}
	return manager
}
