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
	workspaceTask := taskRecordForTest("task-integration-workspace-child")
	workspaceTask.Scope = taskpkg.ScopeWorkspace
	workspaceTask.WorkspaceID = workspaceID
	workspaceTask.ParentTaskID = globalTask.ID
	workspaceTask.Owner = ownershipForTest(taskpkg.OwnerKindPool, "backlog")
	workspaceTask.NetworkChannel = "engineering"

	if err := first.CreateTask(ctx, globalTask); err != nil {
		t.Fatalf("CreateTask(global) error = %v", err)
	}
	if err := first.CreateTask(ctx, workspaceTask); err != nil {
		t.Fatalf("CreateTask(workspace) error = %v", err)
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

	globalTasks, err := second.ListTasks(ctx, taskpkg.TaskQuery{Scope: taskpkg.ScopeGlobal})
	if err != nil {
		t.Fatalf("ListTasks(global) error = %v", err)
	}
	if got, want := len(globalTasks), 1; got != want {
		t.Fatalf("len(ListTasks(global)) = %d, want %d", got, want)
	}
	assertTaskSummaryMatchesTask(t, globalTasks[0], globalTask)

	workspaceTasks, err := second.ListTasks(ctx, taskpkg.TaskQuery{WorkspaceID: workspaceID})
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

	runs, err := third.ListTaskRuns(ctx, taskpkg.TaskRunQuery{TaskID: taskRecord.ID})
	if err != nil {
		t.Fatalf("ListTaskRuns() error = %v", err)
	}
	if got, want := len(runs), 1; got != want {
		t.Fatalf("len(ListTaskRuns()) = %d, want %d", got, want)
	}
}
