//go:build integration

package globaldb

import (
	"path/filepath"
	"testing"
	"time"

	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestGlobalDBTaskPersistenceSurvivesReopenWithGlobalAndWorkspaceTasks(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	dbPath := filepath.Join(t.TempDir(), GlobalDatabaseName)

	first, err := OpenGlobalDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(first) error = %v", err)
	}

	workspaceID := registerWorkspaceForGlobalTests(t, first, "task-integration-workspace", filepath.Join(t.TempDir(), "workspace"))
	globalTask := taskRecordForTest("task-integration-global")
	globalTask.Status = taskpkg.TaskStatusDraft
	globalTask.Priority = taskpkg.PriorityLow
	globalTask.MaxAttempts = 2
	workspaceTask := taskRecordForTest("task-integration-workspace-child")
	workspaceTask.Scope = taskpkg.ScopeWorkspace
	workspaceTask.WorkspaceID = workspaceID
	workspaceTask.ParentTaskID = globalTask.ID
	workspaceTask.Priority = taskpkg.PriorityUrgent
	workspaceTask.MaxAttempts = 5
	workspaceTask.ApprovalPolicy = taskpkg.ApprovalPolicyManual
	workspaceTask.ApprovalState = taskpkg.ApprovalStateApproved
	workspaceTask.Owner = ownershipForTest(taskpkg.OwnerKindPool, "backlog")
	workspaceTask.NetworkChannel = "engineering"

	if err := first.CreateTask(ctx, globalTask); err != nil {
		t.Fatalf("CreateTask(global) error = %v", err)
	}
	if err := first.CreateTask(ctx, workspaceTask); err != nil {
		t.Fatalf("CreateTask(workspace) error = %v", err)
	}

	aliceTriage := taskpkg.TriageState{
		TaskID:             workspaceTask.ID,
		Actor:              taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "user:alice"},
		Read:               true,
		Archived:           false,
		Dismissed:          false,
		LastSeenActivityAt: workspaceTask.UpdatedAt.Add(3 * time.Minute),
		UpdatedAt:          workspaceTask.UpdatedAt.Add(4 * time.Minute),
	}
	if err := first.UpsertTaskTriageState(ctx, aliceTriage); err != nil {
		t.Fatalf("UpsertTaskTriageState(alice) error = %v", err)
	}
	bobTriage := taskpkg.TriageState{
		TaskID:    workspaceTask.ID,
		Actor:     taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "user:bob"},
		Read:      false,
		Archived:  true,
		Dismissed: true,
		UpdatedAt: workspaceTask.UpdatedAt.Add(5 * time.Minute),
	}
	if err := first.UpsertTaskTriageState(ctx, bobTriage); err != nil {
		t.Fatalf("UpsertTaskTriageState(bob) error = %v", err)
	}

	if err := first.Close(ctx); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	second, err := OpenGlobalDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(second) error = %v", err)
	}
	t.Cleanup(func() {
		if err := second.Close(ctx); err != nil {
			t.Fatalf("Close(second) error = %v", err)
		}
	})

	globalTasks, err := second.ListTasks(ctx, taskpkg.Query{Scope: taskpkg.ScopeGlobal})
	if err != nil {
		t.Fatalf("ListTasks(global) error = %v", err)
	}
	if got, want := len(globalTasks), 1; got != want {
		t.Fatalf("len(ListTasks(global)) = %d, want %d", got, want)
	}
	assertTaskSummaryMatchesTask(t, globalTasks[0], globalTask)
	if !globalTasks[0].Draft {
		t.Fatalf("globalTasks[0].Draft = false, want true")
	}

	workspaceTasks, err := second.ListTasks(ctx, taskpkg.Query{WorkspaceID: workspaceID})
	if err != nil {
		t.Fatalf("ListTasks(workspace) error = %v", err)
	}
	if got, want := len(workspaceTasks), 1; got != want {
		t.Fatalf("len(ListTasks(workspace)) = %d, want %d", got, want)
	}
	assertTaskSummaryMatchesTask(t, workspaceTasks[0], workspaceTask)

	reloadedTask, err := second.GetTask(ctx, workspaceTask.ID)
	if err != nil {
		t.Fatalf("GetTask(workspace) error = %v", err)
	}
	assertTaskEqual(t, reloadedTask, workspaceTask)

	storedAlice, err := second.GetTaskTriageState(ctx, workspaceTask.ID, aliceTriage.Actor)
	if err != nil {
		t.Fatalf("GetTaskTriageState(alice) error = %v", err)
	}
	if storedAlice != aliceTriage {
		t.Fatalf("storedAlice = %#v, want %#v", storedAlice, aliceTriage)
	}

	storedBob, err := second.GetTaskTriageState(ctx, workspaceTask.ID, bobTriage.Actor)
	if err != nil {
		t.Fatalf("GetTaskTriageState(bob) error = %v", err)
	}
	if storedBob != bobTriage {
		t.Fatalf("storedBob = %#v, want %#v", storedBob, bobTriage)
	}
}

func TestGlobalDBTaskRunSessionAttachmentSurvivesReopen(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	dbPath := filepath.Join(t.TempDir(), GlobalDatabaseName)

	first, err := OpenGlobalDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(first) error = %v", err)
	}

	taskRecord := taskRecordForTest("task-integration-run")
	if err := first.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	run := taskRunForTest("run-integration", taskRecord.ID)
	if err := first.CreateTaskRun(ctx, run); err != nil {
		t.Fatalf("CreateTaskRun(queued) error = %v", err)
	}

	if err := first.Close(ctx); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	second, err := OpenGlobalDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(second) error = %v", err)
	}

	storedQueued, err := second.GetTaskRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetTaskRun(queued) error = %v", err)
	}
	if storedQueued.SessionID != "" {
		t.Fatalf("GetTaskRun(queued).SessionID = %q, want empty", storedQueued.SessionID)
	}

	storedQueued.Status = taskpkg.TaskRunStatusRunning
	storedQueued.SessionID = "sess-persisted"
	storedQueued.StartedAt = storedQueued.QueuedAt.Add(45 * time.Second)
	storedQueued.ClaimedAt = storedQueued.QueuedAt.Add(15 * time.Second)
	storedQueued.ClaimedBy = actorForTest(taskpkg.ActorKindDaemon, "scheduler")
	if err := second.UpdateTaskRun(ctx, storedQueued); err != nil {
		t.Fatalf("UpdateTaskRun(attached) error = %v", err)
	}

	if err := second.Close(ctx); err != nil {
		t.Fatalf("Close(second) error = %v", err)
	}

	third, err := OpenGlobalDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(third) error = %v", err)
	}
	t.Cleanup(func() {
		if err := third.Close(ctx); err != nil {
			t.Fatalf("Close(third) error = %v", err)
		}
	})

	reloadedRun, err := third.GetTaskRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetTaskRun(attached) error = %v", err)
	}
	assertTaskRunEqual(t, reloadedRun, storedQueued)

	runs, err := third.ListTaskRuns(ctx, taskpkg.RunQuery{TaskID: taskRecord.ID})
	if err != nil {
		t.Fatalf("ListTaskRuns() error = %v", err)
	}
	if got, want := len(runs), 1; got != want {
		t.Fatalf("len(ListTaskRuns()) = %d, want %d", got, want)
	}
}
