//go:build integration

package automation

import (
	"context"
	"errors"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestManagerIntegrationDirectTaskBackedJobDelegatesIntoTaskDomain(t *testing.T) {
	t.Parallel()

	h := newManagerHarness(t)
	taskManager, err := taskpkg.NewManager(taskpkg.WithStore(h.db))
	if err != nil {
		t.Fatalf("task.NewManager() error = %v", err)
	}

	manager := h.newManager(t, integrationAutomationConfig(), WithTasks(taskManager))
	if err := manager.Start(h.ctx); err != nil {
		t.Fatalf("manager.Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("manager.Shutdown() error = %v", err)
		}
	})

	job, err := manager.CreateJob(h.ctx, Job{
		Scope:       AutomationScopeWorkspace,
		Name:        "direct-task-backed",
		WorkspaceID: h.workspace.ID,
		Schedule: &ScheduleSpec{
			Mode:     ScheduleModeEvery,
			Interval: "1h",
		},
		Task: &JobTaskConfig{
			Title:          "Direct automation review",
			Description:    "Persist a durable review task.",
			NetworkChannel: "ops-automation",
			Owner: &taskpkg.Ownership{
				Kind: taskpkg.OwnerKindAutomation,
				Ref:  "job:direct-task-backed",
			},
		},
		Enabled:   true,
		Retry:     DefaultRetryConfig(),
		FireLimit: DefaultFireLimitConfig(),
	})
	if err != nil {
		t.Fatalf("manager.CreateJob() error = %v", err)
	}

	run, err := manager.TriggerJob(h.ctx, job.ID)
	if err != nil {
		t.Fatalf("manager.TriggerJob() error = %v", err)
	}
	if got, want := run.Status, RunDelegated; got != want {
		t.Fatalf("run.Status = %q, want %q", got, want)
	}
	if got := run.SessionID; got != "" {
		t.Fatalf("run.SessionID = %q, want empty", got)
	}
	if got, want := len(h.sessions.creator.createCalls()), 0; got != want {
		t.Fatalf("len(Create calls) = %d, want %d", got, want)
	}
	if got, want := len(h.sessions.creator.promptCalls()), 0; got != want {
		t.Fatalf("len(Prompt calls) = %d, want %d", got, want)
	}

	storedRun, err := h.db.GetRun(h.ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if got, want := storedRun.TaskID, run.TaskID; got != want {
		t.Fatalf("storedRun.TaskID = %q, want %q", got, want)
	}
	if got, want := storedRun.TaskRunID, run.TaskRunID; got != want {
		t.Fatalf("storedRun.TaskRunID = %q, want %q", got, want)
	}

	taskRecord, err := h.db.GetTask(h.ctx, run.TaskID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got, want := taskRecord.Scope, taskpkg.ScopeWorkspace; got != want {
		t.Fatalf("task.Scope = %q, want %q", got, want)
	}
	if got, want := taskRecord.WorkspaceID, h.workspace.ID; got != want {
		t.Fatalf("task.WorkspaceID = %q, want %q", got, want)
	}
	if got, want := taskRecord.NetworkChannel, "ops-automation"; got != want {
		t.Fatalf("task.NetworkChannel = %q, want %q", got, want)
	}
	if taskRecord.Owner == nil || taskRecord.Owner.Kind != taskpkg.OwnerKindAutomation || taskRecord.Owner.Ref != "job:direct-task-backed" {
		t.Fatalf("task.Owner = %#v, want automation owner", taskRecord.Owner)
	}
	if got, want := taskRecord.CreatedBy.Kind, taskpkg.ActorKindAutomation; got != want {
		t.Fatalf("task.CreatedBy.Kind = %q, want %q", got, want)
	}
	if got, want := taskRecord.CreatedBy.Ref, job.ID; got != want {
		t.Fatalf("task.CreatedBy.Ref = %q, want %q", got, want)
	}
	if got, want := taskRecord.Origin.Kind, taskpkg.OriginKindAutomation; got != want {
		t.Fatalf("task.Origin.Kind = %q, want %q", got, want)
	}
	if got, want := taskRecord.Origin.Ref, "run:"+run.ID; got != want {
		t.Fatalf("task.Origin.Ref = %q, want %q", got, want)
	}

	taskRun, err := h.db.GetTaskRun(h.ctx, run.TaskRunID)
	if err != nil {
		t.Fatalf("GetTaskRun() error = %v", err)
	}
	if got, want := taskRun.Origin.Kind, taskpkg.OriginKindAutomation; got != want {
		t.Fatalf("taskRun.Origin.Kind = %q, want %q", got, want)
	}
	if got, want := taskRun.Origin.Ref, "run:"+run.ID; got != want {
		t.Fatalf("taskRun.Origin.Ref = %q, want %q", got, want)
	}
	if got, want := taskRun.IdempotencyKey, "automation-run:"+run.ID; got != want {
		t.Fatalf("taskRun.IdempotencyKey = %q, want %q", got, want)
	}
	if got, want := taskRun.NetworkChannel, "ops-automation"; got != want {
		t.Fatalf("taskRun.NetworkChannel = %q, want %q", got, want)
	}
}

func TestManagerIntegrationAutomationSessionCanCreateTaskWithAutomationOrigin(t *testing.T) {
	t.Parallel()

	promptStarted := make(chan struct{}, 1)
	promptRelease := make(chan struct{})

	h := newManagerHarness(t)
	h.sessions = newManagerSessionStub(sessionAttemptPlan{
		sessionID:     "sess-automation-agent",
		promptStarted: promptStarted,
		promptRelease: promptRelease,
	})

	taskManager, err := taskpkg.NewManager(taskpkg.WithStore(h.db))
	if err != nil {
		t.Fatalf("task.NewManager() error = %v", err)
	}

	manager := h.newManager(t, integrationAutomationConfig(), WithTasks(taskManager))
	if err := manager.Start(h.ctx); err != nil {
		t.Fatalf("manager.Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("manager.Shutdown() error = %v", err)
		}
	})

	job, err := manager.CreateJob(h.ctx, Job{
		Scope:       AutomationScopeWorkspace,
		Name:        "agent-mediated-task-create",
		AgentName:   "researcher",
		WorkspaceID: h.workspace.ID,
		Prompt:      "Inspect the repo and decide whether to create a task.",
		Schedule: &ScheduleSpec{
			Mode:     ScheduleModeEvery,
			Interval: "1h",
		},
		Enabled:   true,
		Retry:     DefaultRetryConfig(),
		FireLimit: DefaultFireLimitConfig(),
	})
	if err != nil {
		t.Fatalf("manager.CreateJob() error = %v", err)
	}

	runCh := make(chan Run, 1)
	errCh := make(chan error, 1)
	go func() {
		run, err := manager.TriggerJob(h.ctx, job.ID)
		runCh <- run
		errCh <- err
	}()

	select {
	case <-promptStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("automation session did not reach Prompt() in time")
	}

	runs, err := h.db.ListRuns(h.ctx, RunQuery{JobID: job.ID})
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if got, want := len(runs), 1; got != want {
		t.Fatalf("len(runs) = %d, want %d", got, want)
	}
	if got, want := runs[0].Status, RunRunning; got != want {
		t.Fatalf("runs[0].Status = %q, want %q", got, want)
	}
	if got, want := runs[0].SessionID, "sess-automation-agent"; got != want {
		t.Fatalf("runs[0].SessionID = %q, want %q", got, want)
	}

	actor, err := manager.TaskActorContextForSession("sess-automation-agent")
	if err != nil {
		t.Fatalf("TaskActorContextForSession() error = %v", err)
	}
	if got, want := actor.Actor.Kind, taskpkg.ActorKindAgentSession; got != want {
		t.Fatalf("actor.Actor.Kind = %q, want %q", got, want)
	}
	if got, want := actor.Origin.Kind, taskpkg.OriginKindAutomation; got != want {
		t.Fatalf("actor.Origin.Kind = %q, want %q", got, want)
	}
	if got, want := actor.Origin.Ref, "run:"+runs[0].ID; got != want {
		t.Fatalf("actor.Origin.Ref = %q, want %q", got, want)
	}

	createdTask, err := taskManager.CreateTask(h.ctx, taskpkg.CreateTask{
		Scope:       taskpkg.ScopeWorkspace,
		WorkspaceID: h.workspace.ID,
		Title:       "Agent-created follow-up",
	}, actor)
	if err != nil {
		t.Fatalf("taskManager.CreateTask() error = %v", err)
	}

	storedTask, err := h.db.GetTask(h.ctx, createdTask.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got, want := storedTask.CreatedBy.Kind, taskpkg.ActorKindAgentSession; got != want {
		t.Fatalf("storedTask.CreatedBy.Kind = %q, want %q", got, want)
	}
	if got, want := storedTask.CreatedBy.Ref, "sess-automation-agent"; got != want {
		t.Fatalf("storedTask.CreatedBy.Ref = %q, want %q", got, want)
	}
	if got, want := storedTask.Origin.Kind, taskpkg.OriginKindAutomation; got != want {
		t.Fatalf("storedTask.Origin.Kind = %q, want %q", got, want)
	}
	if got, want := storedTask.Origin.Ref, "run:"+runs[0].ID; got != want {
		t.Fatalf("storedTask.Origin.Ref = %q, want %q", got, want)
	}

	close(promptRelease)

	var completedRun Run
	select {
	case completedRun = <-runCh:
	case <-time.After(2 * time.Second):
		t.Fatal("manager.TriggerJob() did not return after prompt release")
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("manager.TriggerJob() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("manager.TriggerJob() error channel did not return")
	}
	if got, want := completedRun.Status, RunCompleted; got != want {
		t.Fatalf("completedRun.Status = %q, want %q", got, want)
	}

	if _, err := manager.TaskActorContextForSession("sess-automation-agent"); !errors.Is(err, ErrSessionTaskActorNotFound) {
		t.Fatalf("TaskActorContextForSession(after completion) error = %v, want ErrSessionTaskActorNotFound", err)
	}
}

func TestManagerIntegrationManualTriggerSurvivesCallerCancellation(t *testing.T) {
	t.Parallel()

	promptStarted := make(chan struct{}, 1)
	promptRelease := make(chan struct{})

	h := newManagerHarness(t)
	h.sessions = newManagerSessionStub(sessionAttemptPlan{
		sessionID:     "sess-trigger-cancel",
		promptStarted: promptStarted,
		promptRelease: promptRelease,
	})

	manager := h.newManager(t, integrationAutomationConfig())
	if err := manager.Start(h.ctx); err != nil {
		t.Fatalf("manager.Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("manager.Shutdown() error = %v", err)
		}
	})

	job, err := manager.CreateJob(h.ctx, Job{
		Scope:       AutomationScopeWorkspace,
		Name:        "manual-trigger-cancel",
		AgentName:   "researcher",
		WorkspaceID: h.workspace.ID,
		Prompt:      "Run the accepted automation.",
		Schedule: &ScheduleSpec{
			Mode:     ScheduleModeEvery,
			Interval: "1h",
		},
		Enabled:   true,
		Retry:     DefaultRetryConfig(),
		FireLimit: DefaultFireLimitConfig(),
	})
	if err != nil {
		t.Fatalf("manager.CreateJob() error = %v", err)
	}

	triggerCtx, cancel := context.WithCancel(h.ctx)
	runCh := make(chan Run, 1)
	errCh := make(chan error, 1)
	go func() {
		run, triggerErr := manager.TriggerJob(triggerCtx, job.ID)
		runCh <- run
		errCh <- triggerErr
	}()

	select {
	case <-promptStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("automation session did not reach Prompt() in time")
	}

	runs, err := h.db.ListRuns(h.ctx, RunQuery{JobID: job.ID})
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if got, want := len(runs), 1; got != want {
		t.Fatalf("len(runs) = %d, want %d", got, want)
	}
	if got, want := runs[0].Status, RunRunning; got != want {
		t.Fatalf("runs[0].Status = %q, want %q", got, want)
	}

	cancel()
	close(promptRelease)

	var completedRun Run
	select {
	case completedRun = <-runCh:
	case <-time.After(2 * time.Second):
		t.Fatal("manager.TriggerJob() did not return after prompt release")
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("manager.TriggerJob() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("manager.TriggerJob() error channel did not return")
	}
	if got, want := completedRun.Status, RunCompleted; got != want {
		t.Fatalf("completedRun.Status = %q, want %q", got, want)
	}
}

func integrationAutomationConfig() aghconfig.AutomationConfig {
	return aghconfig.AutomationConfig{
		Enabled:           true,
		Timezone:          DefaultTimezone,
		MaxConcurrentJobs: DefaultMaxConcurrentJobs,
		DefaultFireLimit:  DefaultFireLimitConfig(),
	}
}
