package globaldb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

func TestOpenGlobalDBCreatesTaskSchemaAndIndexes(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	assertTablesPresent(
		t,
		globalDB.db,
		"tasks",
		"task_triage_state",
		"task_runs",
		"task_dependencies",
		"task_events",
		"task_run_idempotency",
	)
	assertTableColumns(t, globalDB.db, "tasks", []string{
		"id",
		"identifier",
		"scope",
		"workspace_id",
		"parent_task_id",
		"network_channel",
		"title",
		"description",
		"priority",
		"max_attempts",
		"status",
		"approval_policy",
		"approval_state",
		"owner_kind",
		"owner_ref",
		"created_by_kind",
		"created_by_ref",
		"origin_kind",
		"origin_ref",
		"created_at",
		"updated_at",
		"closed_at",
		"metadata_json",
	})
	assertTableColumns(t, globalDB.db, "task_triage_state", []string{
		"task_id",
		"actor_kind",
		"actor_ref",
		"is_read",
		"archived",
		"dismissed",
		"last_seen_activity_at",
		"updated_at",
	})
	assertTableColumns(t, globalDB.db, "task_runs", []string{
		"id",
		"task_id",
		"status",
		"attempt",
		"claimed_by_kind",
		"claimed_by_ref",
		"session_id",
		"origin_kind",
		"origin_ref",
		"idempotency_key",
		"network_channel",
		"queued_at",
		"claimed_at",
		"started_at",
		"ended_at",
		"error",
		"metadata_json",
		"result_json",
	})
	assertTableColumns(t, globalDB.db, "task_dependencies", []string{
		"task_id",
		"depends_on_task_id",
		"kind",
		"created_at",
	})
	assertTableColumns(t, globalDB.db, "task_events", []string{
		"id",
		"event_seq",
		"task_id",
		"run_id",
		"event_type",
		"actor_kind",
		"actor_ref",
		"origin_kind",
		"origin_ref",
		"payload_json",
		"timestamp",
	})
	assertTableColumns(t, globalDB.db, "task_run_idempotency", []string{
		"idempotency_key",
		"origin_kind",
		"origin_ref",
		"run_id",
		"created_at",
	})
	assertIndexesPresent(t, globalDB.db, "tasks",
		"idx_tasks_scope",
		"idx_tasks_workspace",
		"idx_tasks_status",
		"idx_tasks_priority",
		"idx_tasks_approval_state",
		"idx_tasks_parent",
		"idx_tasks_owner",
		"idx_tasks_channel",
	)
	assertIndexesPresent(t, globalDB.db, "task_triage_state",
		"idx_task_triage_task",
		"idx_task_triage_actor",
	)
	assertIndexesPresent(t, globalDB.db, "task_runs",
		"idx_task_runs_task",
		"idx_task_runs_task_status",
		"idx_task_runs_status",
		"idx_task_runs_session",
		"idx_task_runs_channel",
	)
	assertIndexesPresent(t, globalDB.db, "task_dependencies",
		"idx_task_dependencies_task",
		"idx_task_dependencies_depends_on",
	)
	assertIndexesPresent(t, globalDB.db, "task_events",
		"idx_task_events_task",
		"idx_task_events_run",
		"idx_task_events_type",
		"uq_task_events_event_seq",
		"idx_task_events_task_seq",
	)
	assertIndexesPresent(t, globalDB.db, "task_run_idempotency",
		"idx_task_run_idempotency_run",
	)
}

func TestGlobalDBTaskRoundTripPreservesNullableFields(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"task-roundtrip-workspace",
		filepath.Join(t.TempDir(), "workspace"),
	)

	parent := taskRecordForTest("task-parent")
	parent.Metadata = json.RawMessage(`{"kind":"global"}`)
	if err := globalDB.CreateTask(testutil.Context(t), parent); err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}

	child := taskRecordForTest("task-child")
	child.Scope = taskpkg.ScopeWorkspace
	child.WorkspaceID = workspaceID
	child.ParentTaskID = parent.ID
	child.NetworkChannel = "finance"
	child.Priority = taskpkg.PriorityUrgent
	child.MaxAttempts = 5
	child.ApprovalPolicy = taskpkg.ApprovalPolicyManual
	child.ApprovalState = taskpkg.ApprovalStateApproved
	child.Owner = ownershipForTest(taskpkg.OwnerKindHuman, "alice")
	child.Metadata = json.RawMessage(`{"kind":"workspace"}`)
	if err := globalDB.CreateTask(testutil.Context(t), child); err != nil {
		t.Fatalf("CreateTask(child) error = %v", err)
	}

	gotParent, err := globalDB.GetTask(testutil.Context(t), parent.ID)
	if err != nil {
		t.Fatalf("GetTask(parent) error = %v", err)
	}
	assertTaskEqual(t, gotParent, parent)
	if gotParent.WorkspaceID != "" {
		t.Fatalf("GetTask(parent).WorkspaceID = %q, want empty", gotParent.WorkspaceID)
	}
	if gotParent.ParentTaskID != "" {
		t.Fatalf("GetTask(parent).ParentTaskID = %q, want empty", gotParent.ParentTaskID)
	}
	if gotParent.Owner != nil {
		t.Fatalf("GetTask(parent).Owner = %#v, want nil", gotParent.Owner)
	}
	if gotParent.NetworkChannel != "" {
		t.Fatalf("GetTask(parent).NetworkChannel = %q, want empty", gotParent.NetworkChannel)
	}

	gotChild, err := globalDB.GetTask(testutil.Context(t), child.ID)
	if err != nil {
		t.Fatalf("GetTask(child) error = %v", err)
	}
	assertTaskEqual(t, gotChild, child)

	child.Title = "Updated child"
	child.Description = "Updated description"
	child.Priority = taskpkg.PriorityHigh
	child.MaxAttempts = 4
	child.Status = taskpkg.TaskStatusInProgress
	child.ApprovalPolicy = taskpkg.ApprovalPolicyNone
	child.ApprovalState = taskpkg.ApprovalStateNotRequired
	child.Owner = ownershipForTest(taskpkg.OwnerKindAgentSession, "sess-1")
	child.Metadata = json.RawMessage(`{"kind":"updated"}`)
	child.UpdatedAt = child.UpdatedAt.Add(2 * time.Minute)
	if err := globalDB.UpdateTask(testutil.Context(t), child); err != nil {
		t.Fatalf("UpdateTask(child) error = %v", err)
	}
	gotChild, err = globalDB.GetTask(testutil.Context(t), child.ID)
	if err != nil {
		t.Fatalf("GetTask(updated child) error = %v", err)
	}
	assertTaskEqual(t, gotChild, child)

	summaries, err := globalDB.ListTasks(testutil.Context(t), taskpkg.Query{ParentTaskID: parent.ID})
	if err != nil {
		t.Fatalf("ListTasks(parent filter) error = %v", err)
	}
	if got, want := len(summaries), 1; got != want {
		t.Fatalf("len(ListTasks(parent filter)) = %d, want %d", got, want)
	}
	assertTaskSummaryMatchesTask(t, summaries[0], child)

	children, err := globalDB.CountDirectChildren(testutil.Context(t), parent.ID)
	if err != nil {
		t.Fatalf("CountDirectChildren() error = %v", err)
	}
	if got, want := children, 1; got != want {
		t.Fatalf("CountDirectChildren() = %d, want %d", got, want)
	}
}

func TestGlobalDBDeleteTaskMapsChildConstraintToValidationError(t *testing.T) {
	t.Parallel()

	t.Run("ShouldMapChildConstraintFailuresToTaskValidationErrors", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)

		parent := taskRecordForTest("task-parent-delete")
		if err := globalDB.CreateTask(testutil.Context(t), parent); err != nil {
			t.Fatalf("CreateTask(parent) error = %v", err)
		}

		child := taskRecordForTest("task-child-delete")
		child.ParentTaskID = parent.ID
		if err := globalDB.CreateTask(testutil.Context(t), child); err != nil {
			t.Fatalf("CreateTask(child) error = %v", err)
		}

		err := globalDB.DeleteTask(testutil.Context(t), parent.ID)
		if !errors.Is(err, taskpkg.ErrValidation) {
			t.Fatalf("DeleteTask(parent) error = %v, want %v", err, taskpkg.ErrValidation)
		}
		if strings.Contains(strings.ToLower(err.Error()), "foreign key constraint failed") {
			t.Fatalf("DeleteTask(parent) error = %q, want mapped task validation error", err.Error())
		}
	})
}

func TestGlobalDBCreateAndUpdateTaskRejectInvalidScopeBindings(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"invalid-scope-workspace",
		filepath.Join(t.TempDir(), "workspace"),
	)

	t.Run("create rejects global task with workspace", func(t *testing.T) {
		t.Parallel()

		record := taskRecordForTest("task-invalid-create-global")
		record.WorkspaceID = workspaceID

		err := globalDB.CreateTask(testutil.Context(t), record)
		if !errors.Is(err, taskpkg.ErrInvalidScopeBinding) {
			t.Fatalf("CreateTask(global with workspace) error = %v, want ErrInvalidScopeBinding", err)
		}
	})

	t.Run("create rejects workspace task without workspace", func(t *testing.T) {
		t.Parallel()

		record := taskRecordForTest("task-invalid-create-workspace")
		record.Scope = taskpkg.ScopeWorkspace

		err := globalDB.CreateTask(testutil.Context(t), record)
		if !errors.Is(err, taskpkg.ErrInvalidScopeBinding) {
			t.Fatalf("CreateTask(workspace without workspace_id) error = %v, want ErrInvalidScopeBinding", err)
		}
	})

	t.Run("update rejects global task with workspace", func(t *testing.T) {
		t.Parallel()

		record := taskRecordForTest("task-invalid-update-global")
		if err := globalDB.CreateTask(testutil.Context(t), record); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}

		record.WorkspaceID = workspaceID
		record.UpdatedAt = record.UpdatedAt.Add(time.Minute)
		err := globalDB.UpdateTask(testutil.Context(t), record)
		if !errors.Is(err, taskpkg.ErrInvalidScopeBinding) {
			t.Fatalf("UpdateTask(global with workspace) error = %v, want ErrInvalidScopeBinding", err)
		}
	})

	t.Run("update rejects workspace task without workspace", func(t *testing.T) {
		t.Parallel()

		record := taskRecordForTest("task-invalid-update-workspace")
		record.Scope = taskpkg.ScopeWorkspace
		record.WorkspaceID = workspaceID
		if err := globalDB.CreateTask(testutil.Context(t), record); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}

		record.WorkspaceID = ""
		record.UpdatedAt = record.UpdatedAt.Add(time.Minute)
		err := globalDB.UpdateTask(testutil.Context(t), record)
		if !errors.Is(err, taskpkg.ErrInvalidScopeBinding) {
			t.Fatalf("UpdateTask(workspace without workspace_id) error = %v, want ErrInvalidScopeBinding", err)
		}
	})
}

func TestGlobalDBListTasksFilters(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceA := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"task-filter-a",
		filepath.Join(t.TempDir(), "workspace-a"),
	)
	workspaceB := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"task-filter-b",
		filepath.Join(t.TempDir(), "workspace-b"),
	)

	globalTask := taskRecordForTest("task-filter-global")
	globalTask.Status = taskpkg.TaskStatusPending

	readyTask := taskRecordForTest("task-filter-ready")
	readyTask.CreatedAt = readyTask.CreatedAt.Add(time.Minute)
	readyTask.UpdatedAt = readyTask.UpdatedAt.Add(time.Minute)
	readyTask.Scope = taskpkg.ScopeWorkspace
	readyTask.WorkspaceID = workspaceA
	readyTask.Status = taskpkg.TaskStatusReady
	readyTask.Priority = taskpkg.PriorityHigh
	readyTask.ApprovalPolicy = taskpkg.ApprovalPolicyManual
	readyTask.ApprovalState = taskpkg.ApprovalStateApproved
	readyTask.Owner = ownershipForTest(taskpkg.OwnerKindHuman, "alice")
	readyTask.NetworkChannel = "finance"

	childTask := taskRecordForTest("task-filter-child")
	childTask.CreatedAt = childTask.CreatedAt.Add(2 * time.Minute)
	childTask.UpdatedAt = childTask.UpdatedAt.Add(2 * time.Minute)
	childTask.Scope = taskpkg.ScopeWorkspace
	childTask.WorkspaceID = workspaceB
	childTask.ParentTaskID = globalTask.ID
	childTask.Status = taskpkg.TaskStatusBlocked
	childTask.Priority = taskpkg.PriorityUrgent
	childTask.ApprovalPolicy = taskpkg.ApprovalPolicyManual
	childTask.ApprovalState = taskpkg.ApprovalStatePending
	childTask.Owner = ownershipForTest(taskpkg.OwnerKindPool, "backlog")
	childTask.NetworkChannel = "engineering"

	for _, record := range []taskpkg.Task{globalTask, readyTask, childTask} {
		if err := globalDB.CreateTask(testutil.Context(t), record); err != nil {
			t.Fatalf("CreateTask(%q) error = %v", record.ID, err)
		}
	}

	for _, tc := range []struct {
		name  string
		query taskpkg.Query
		want  []string
	}{
		{
			name:  "scope",
			query: taskpkg.Query{Scope: taskpkg.ScopeGlobal},
			want:  []string{globalTask.ID},
		},
		{
			name:  "workspace",
			query: taskpkg.Query{WorkspaceID: workspaceA},
			want:  []string{readyTask.ID},
		},
		{
			name:  "status",
			query: taskpkg.Query{Status: taskpkg.TaskStatusReady},
			want:  []string{readyTask.ID},
		},
		{
			name:  "priority",
			query: taskpkg.Query{Priority: taskpkg.PriorityUrgent},
			want:  []string{childTask.ID},
		},
		{
			name:  "approval state",
			query: taskpkg.Query{ApprovalState: taskpkg.ApprovalStatePending},
			want:  []string{childTask.ID},
		},
		{
			name:  "parent",
			query: taskpkg.Query{ParentTaskID: globalTask.ID},
			want:  []string{childTask.ID},
		},
		{
			name:  "owner kind",
			query: taskpkg.Query{OwnerKind: taskpkg.OwnerKindHuman},
			want:  []string{readyTask.ID},
		},
		{
			name:  "owner ref",
			query: taskpkg.Query{OwnerRef: "alice"},
			want:  []string{readyTask.ID},
		},
		{
			name:  "channel",
			query: taskpkg.Query{NetworkChannel: "engineering"},
			want:  []string{childTask.ID},
		},
		{
			name:  "limit",
			query: taskpkg.Query{Limit: 2},
			want:  []string{childTask.ID, readyTask.ID},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			summaries, err := globalDB.ListTasks(testutil.Context(t), tc.query)
			if err != nil {
				t.Fatalf("ListTasks(%s) error = %v", tc.name, err)
			}
			gotIDs := taskSummaryIDs(summaries)
			if !testutil.EqualStringSlices(gotIDs, tc.want) {
				t.Fatalf("ListTasks(%s) ids = %#v, want %#v", tc.name, gotIDs, tc.want)
			}
		})
	}
}

func TestGlobalDBListTasksSearchAndActivityOrdering(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	alpha := taskRecordForTest("task-search-alpha")
	alpha.Title = "Alpha planning"
	alpha.Identifier = "OPS-100"
	alpha.UpdatedAt = alpha.UpdatedAt.Add(time.Minute)

	beta := taskRecordForTest("task-search-beta")
	beta.Title = "Beta rollout"
	beta.Identifier = "OPS-200"
	beta.CreatedAt = beta.CreatedAt.Add(2 * time.Minute)
	beta.UpdatedAt = beta.UpdatedAt.Add(2 * time.Minute)

	for _, record := range []taskpkg.Task{alpha, beta} {
		if err := globalDB.CreateTask(testutil.Context(t), record); err != nil {
			t.Fatalf("CreateTask(%q) error = %v", record.ID, err)
		}
	}
	if err := globalDB.CreateTaskRun(testutil.Context(t), taskpkg.Run{
		ID:        "run-search-beta",
		TaskID:    beta.ID,
		Status:    taskpkg.TaskRunStatusRunning,
		Attempt:   1,
		Origin:    taskpkg.Origin{Kind: taskpkg.OriginKindDaemon, Ref: "scheduler"},
		QueuedAt:  time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC),
		StartedAt: time.Date(2026, 4, 17, 12, 5, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("CreateTaskRun() error = %v", err)
	}

	byTitle, err := globalDB.ListTasks(testutil.Context(t), taskpkg.Query{Search: "alpha"})
	if err != nil {
		t.Fatalf("ListTasks(search title) error = %v", err)
	}
	if got, want := orderedTaskSummaryIDs(byTitle), []string{alpha.ID}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("ListTasks(search title) ids = %#v, want %#v", got, want)
	}

	byIdentifier, err := globalDB.ListTasks(testutil.Context(t), taskpkg.Query{Search: "ops-200"})
	if err != nil {
		t.Fatalf("ListTasks(search identifier) error = %v", err)
	}
	if got, want := orderedTaskSummaryIDs(byIdentifier), []string{beta.ID}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("ListTasks(search identifier) ids = %#v, want %#v", got, want)
	}

	all, err := globalDB.ListTasks(testutil.Context(t), taskpkg.Query{})
	if err != nil {
		t.Fatalf("ListTasks(all) error = %v", err)
	}
	if got, want := orderedTaskSummaryIDs(all), []string{beta.ID, alpha.ID}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("ListTasks(all) order = %#v, want %#v", got, want)
	}
}

func TestGlobalDBTaskRunRoundTripAndFilters(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	taskRecord := taskRecordForTest("task-run-roundtrip")
	if err := globalDB.CreateTask(testutil.Context(t), taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	queuedRun := taskRunForTest("run-queued", taskRecord.ID)
	queuedRun.Metadata = json.RawMessage(`{"schema":"agh.harness.detached.v1","owner_session_id":"sess-owner"}`)
	if err := globalDB.CreateTaskRun(testutil.Context(t), queuedRun); err != nil {
		t.Fatalf("CreateTaskRun() error = %v", err)
	}

	storedQueued, err := globalDB.GetTaskRun(testutil.Context(t), queuedRun.ID)
	if err != nil {
		t.Fatalf("GetTaskRun(queued) error = %v", err)
	}
	if storedQueued.SessionID != "" {
		t.Fatalf("GetTaskRun(queued).SessionID = %q, want empty", storedQueued.SessionID)
	}
	if storedQueued.ClaimedBy != nil {
		t.Fatalf("GetTaskRun(queued).ClaimedBy = %#v, want nil", storedQueued.ClaimedBy)
	}

	runningRun := queuedRun
	runningRun.Status = taskpkg.TaskRunStatusRunning
	runningRun.ClaimedBy = actorForTest(taskpkg.ActorKindDaemon, "scheduler")
	runningRun.SessionID = "sess-task-run"
	runningRun.NetworkChannel = "finance"
	runningRun.ClaimedAt = queuedRun.QueuedAt.Add(30 * time.Second)
	runningRun.StartedAt = queuedRun.QueuedAt.Add(time.Minute)
	if err := globalDB.UpdateTaskRun(testutil.Context(t), runningRun); err != nil {
		t.Fatalf("UpdateTaskRun(running) error = %v", err)
	}

	runsByTask, err := globalDB.ListTaskRuns(testutil.Context(t), taskpkg.RunQuery{TaskID: taskRecord.ID})
	if err != nil {
		t.Fatalf("ListTaskRuns(task) error = %v", err)
	}
	if got, want := len(runsByTask), 1; got != want {
		t.Fatalf("len(ListTaskRuns(task)) = %d, want %d", got, want)
	}
	assertTaskRunEqual(t, runsByTask[0], runningRun)

	runsBySession, err := globalDB.ListTaskRuns(testutil.Context(t), taskpkg.RunQuery{SessionID: "sess-task-run"})
	if err != nil {
		t.Fatalf("ListTaskRuns(session) error = %v", err)
	}
	if got, want := len(runsBySession), 1; got != want {
		t.Fatalf("len(ListTaskRuns(session)) = %d, want %d", got, want)
	}

	runsByStatus, err := globalDB.ListTaskRunsByStatus(
		testutil.Context(t),
		[]taskpkg.RunStatus{taskpkg.TaskRunStatusRunning},
	)
	if err != nil {
		t.Fatalf("ListTaskRunsByStatus() error = %v", err)
	}
	if got, want := len(runsByStatus), 1; got != want {
		t.Fatalf("len(ListTaskRunsByStatus()) = %d, want %d", got, want)
	}

	activeBindings, err := globalDB.CountActiveSessionBindings(testutil.Context(t), "sess-task-run")
	if err != nil {
		t.Fatalf("CountActiveSessionBindings(running) error = %v", err)
	}
	if got, want := activeBindings, 1; got != want {
		t.Fatalf("CountActiveSessionBindings(running) = %d, want %d", got, want)
	}

	completedRun := runningRun
	completedRun.Status = taskpkg.TaskRunStatusCompleted
	completedRun.EndedAt = runningRun.StartedAt.Add(5 * time.Minute)
	completedRun.Result = json.RawMessage(`{"ok":true}`)
	if err := globalDB.UpdateTaskRun(testutil.Context(t), completedRun); err != nil {
		t.Fatalf("UpdateTaskRun(completed) error = %v", err)
	}

	storedCompleted, err := globalDB.GetTaskRun(testutil.Context(t), completedRun.ID)
	if err != nil {
		t.Fatalf("GetTaskRun(completed) error = %v", err)
	}
	assertTaskRunEqual(t, storedCompleted, completedRun)

	activeBindings, err = globalDB.CountActiveSessionBindings(testutil.Context(t), "sess-task-run")
	if err != nil {
		t.Fatalf("CountActiveSessionBindings(completed) error = %v", err)
	}
	if got, want := activeBindings, 0; got != want {
		t.Fatalf("CountActiveSessionBindings(completed) = %d, want %d", got, want)
	}
}

func TestGlobalDBReserveQueuedRunDeduplicatesConcurrentIdempotentRequests(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	ctx := testutil.Context(t)
	taskRecord := taskRecordForTest("task-run-reserve-idempotent")
	if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	origin := taskpkg.Origin{Kind: taskpkg.OriginKindDaemon, Ref: "scheduler"}
	queuedAt := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	metadata := json.RawMessage(`{"schema":"agh.harness.detached.v1","wake_target":{"session_id":"sess-wake"}}`)
	type reserveResult struct {
		task     taskpkg.Task
		run      taskpkg.Run
		existing bool
		err      error
	}

	results := make([]reserveResult, 2)
	runIDs := []string{"run-reserved-a", "run-reserved-b"}
	var wg sync.WaitGroup
	wg.Add(len(results))
	for idx := range results {
		go func(i int) {
			defer wg.Done()
			taskCopy, runCopy, existing, err := globalDB.ReserveQueuedRun(
				ctx,
				taskRecord.ID,
				runIDs[i],
				"dup-key",
				origin,
				"ops",
				metadata,
				queuedAt,
			)
			results[i] = reserveResult{
				task:     taskCopy,
				run:      runCopy,
				existing: existing,
				err:      err,
			}
		}(idx)
	}
	wg.Wait()

	for idx, result := range results {
		if result.err != nil {
			t.Fatalf("ReserveQueuedRun(%d) error = %v", idx, result.err)
		}
		if got, want := result.task.ID, taskRecord.ID; got != want {
			t.Fatalf("ReserveQueuedRun(%d) task id = %q, want %q", idx, got, want)
		}
		if got, want := result.run.TaskID, taskRecord.ID; got != want {
			t.Fatalf("ReserveQueuedRun(%d) run task id = %q, want %q", idx, got, want)
		}
		if got, want := result.run.IdempotencyKey, "dup-key"; got != want {
			t.Fatalf("ReserveQueuedRun(%d) idempotency key = %q, want %q", idx, got, want)
		}
		if got, want := result.run.Attempt, 1; got != want {
			t.Fatalf("ReserveQueuedRun(%d) attempt = %d, want %d", idx, got, want)
		}
		if got, want := string(result.run.Metadata), string(metadata); got != want {
			t.Fatalf("ReserveQueuedRun(%d) metadata = %s, want %s", idx, got, want)
		}
	}

	if results[0].run.ID != results[1].run.ID {
		t.Fatalf("ReserveQueuedRun() run ids = [%q %q], want same run", results[0].run.ID, results[1].run.ID)
	}

	existingCount := 0
	for _, result := range results {
		if result.existing {
			existingCount++
		}
	}
	if got, want := existingCount, 1; got != want {
		t.Fatalf("existing result count = %d, want %d", got, want)
	}

	runs, err := globalDB.ListTaskRuns(ctx, taskpkg.RunQuery{TaskID: taskRecord.ID})
	if err != nil {
		t.Fatalf("ListTaskRuns() error = %v", err)
	}
	if got, want := len(runs), 1; got != want {
		t.Fatalf("len(ListTaskRuns()) = %d, want %d", got, want)
	}
	if got, want := runs[0].ID, results[0].run.ID; got != want {
		t.Fatalf("stored run id = %q, want %q", got, want)
	}

	storedRun, err := globalDB.GetTaskRunByIdempotencyKey(ctx, "dup-key", origin)
	if err != nil {
		t.Fatalf("GetTaskRunByIdempotencyKey() error = %v", err)
	}
	if got, want := storedRun.ID, results[0].run.ID; got != want {
		t.Fatalf("GetTaskRunByIdempotencyKey() id = %q, want %q", got, want)
	}
	if got, want := string(storedRun.Metadata), string(metadata); got != want {
		t.Fatalf("GetTaskRunByIdempotencyKey() metadata = %s, want %s", got, want)
	}
}

func TestGlobalDBReserveQueuedRunRejectsConcurrentOpenRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    taskpkg.RunStatus
		configure func(taskpkg.Run) taskpkg.Run
	}{
		{
			name:   "Should reject another queued reservation while a queued run exists",
			status: taskpkg.TaskRunStatusQueued,
			configure: func(run taskpkg.Run) taskpkg.Run {
				return run
			},
		},
		{
			name:   "Should reject another queued reservation while a claimed run exists",
			status: taskpkg.TaskRunStatusClaimed,
			configure: func(run taskpkg.Run) taskpkg.Run {
				run.ClaimedBy = actorForTest(taskpkg.ActorKindDaemon, "scheduler")
				run.ClaimedAt = run.QueuedAt.Add(30 * time.Second)
				return run
			},
		},
		{
			name:   "Should reject another queued reservation while a starting run exists",
			status: taskpkg.TaskRunStatusStarting,
			configure: func(run taskpkg.Run) taskpkg.Run {
				run.ClaimedBy = actorForTest(taskpkg.ActorKindDaemon, "scheduler")
				run.ClaimedAt = run.QueuedAt.Add(30 * time.Second)
				run.SessionID = "sess-open-starting"
				run.StartedAt = run.QueuedAt.Add(time.Minute)
				return run
			},
		},
		{
			name:   "Should reject another queued reservation while a running run exists",
			status: taskpkg.TaskRunStatusRunning,
			configure: func(run taskpkg.Run) taskpkg.Run {
				run.ClaimedBy = actorForTest(taskpkg.ActorKindDaemon, "scheduler")
				run.ClaimedAt = run.QueuedAt.Add(30 * time.Second)
				run.SessionID = "sess-open-running"
				run.StartedAt = run.QueuedAt.Add(time.Minute)
				return run
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			globalDB := openTestGlobalDB(t)
			ctx := testutil.Context(t)
			taskRecord := taskRecordForTest("task-run-reserve-open-guard")
			if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
				t.Fatalf("CreateTask() error = %v", err)
			}

			origin := taskpkg.Origin{Kind: taskpkg.OriginKindDaemon, Ref: "scheduler"}
			queuedAt := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
			_, firstRun, existing, err := globalDB.ReserveQueuedRun(
				ctx,
				taskRecord.ID,
				"run-reserved-open-a",
				"open-key",
				origin,
				"ops",
				nil,
				queuedAt,
			)
			if err != nil {
				t.Fatalf("ReserveQueuedRun(first) error = %v", err)
			}
			if existing {
				t.Fatal("ReserveQueuedRun(first) existing = true, want false")
			}

			storedFirstRun, err := globalDB.GetTaskRun(ctx, firstRun.ID)
			if err != nil {
				t.Fatalf("GetTaskRun(first) error = %v", err)
			}
			storedFirstRun.Status = tt.status
			storedFirstRun = tt.configure(storedFirstRun)
			if err := globalDB.UpdateTaskRun(ctx, storedFirstRun); err != nil {
				t.Fatalf("UpdateTaskRun(%s) error = %v", tt.status, err)
			}

			_, duplicateRun, duplicateExisting, err := globalDB.ReserveQueuedRun(
				ctx,
				taskRecord.ID,
				"run-reserved-open-duplicate",
				"open-key",
				origin,
				"ops",
				nil,
				queuedAt.Add(time.Second),
			)
			if err != nil {
				t.Fatalf("ReserveQueuedRun(idempotent duplicate) error = %v", err)
			}
			if !duplicateExisting {
				t.Fatal("ReserveQueuedRun(idempotent duplicate) existing = false, want true")
			}
			if got, want := duplicateRun.ID, firstRun.ID; got != want {
				t.Fatalf("ReserveQueuedRun(idempotent duplicate).ID = %q, want %q", got, want)
			}

			_, secondRun, secondExisting, err := globalDB.ReserveQueuedRun(
				ctx,
				taskRecord.ID,
				"run-reserved-open-b",
				"new-open-key",
				origin,
				"ops",
				nil,
				queuedAt.Add(2*time.Second),
			)
			if secondExisting {
				t.Fatal("ReserveQueuedRun(second) existing = true, want false")
			}
			if secondRun.ID != "" {
				t.Fatalf("ReserveQueuedRun(second) run = %#v, want zero value", secondRun)
			}
			if !errors.Is(err, taskpkg.ErrInvalidStatusTransition) {
				t.Fatalf("ReserveQueuedRun(second) error = %v, want %v", err, taskpkg.ErrInvalidStatusTransition)
			}

			runs, err := globalDB.ListTaskRuns(ctx, taskpkg.RunQuery{TaskID: taskRecord.ID})
			if err != nil {
				t.Fatalf("ListTaskRuns() error = %v", err)
			}
			if got, want := len(runs), 1; got != want {
				t.Fatalf("len(ListTaskRuns()) = %d, want %d", got, want)
			}
			if got, want := runs[0].ID, firstRun.ID; got != want {
				t.Fatalf("stored run id = %q, want %q", got, want)
			}
		})
	}
}

func TestGlobalDBUpdateTaskRunRejectsSessionRebinding(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	taskRecord := taskRecordForTest("task-run-rebinding")
	if err := globalDB.CreateTask(testutil.Context(t), taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	run := taskRunForTest("run-rebinding", taskRecord.ID)
	run.Status = taskpkg.TaskRunStatusRunning
	run.SessionID = "sess-1"
	run.StartedAt = run.QueuedAt.Add(time.Minute)
	if err := globalDB.CreateTaskRun(testutil.Context(t), run); err != nil {
		t.Fatalf("CreateTaskRun() error = %v", err)
	}

	run.SessionID = "sess-2"
	err := globalDB.UpdateTaskRun(testutil.Context(t), run)
	if !errors.Is(err, taskpkg.ErrSessionAlreadyBound) {
		t.Fatalf("UpdateTaskRun(rebind) error = %v, want ErrSessionAlreadyBound", err)
	}
}

func TestGlobalDBTaskAndRunReferenceErrors(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	_, err := globalDB.GetTask(testutil.Context(t), "missing-task")
	if !errors.Is(err, taskpkg.ErrTaskNotFound) {
		t.Fatalf("GetTask(missing) error = %v, want ErrTaskNotFound", err)
	}

	_, err = globalDB.GetTaskRun(testutil.Context(t), "missing-run")
	if !errors.Is(err, taskpkg.ErrTaskRunNotFound) {
		t.Fatalf("GetTaskRun(missing) error = %v, want ErrTaskRunNotFound", err)
	}

	workspaceTask := taskRecordForTest("task-missing-workspace")
	workspaceTask.Scope = taskpkg.ScopeWorkspace
	workspaceTask.WorkspaceID = "ws-missing"
	err = globalDB.CreateTask(testutil.Context(t), workspaceTask)
	if !errors.Is(err, aghworkspace.ErrWorkspaceNotFound) {
		t.Fatalf("CreateTask(missing workspace) error = %v, want ErrWorkspaceNotFound", err)
	}

	childTask := taskRecordForTest("task-missing-parent")
	childTask.ParentTaskID = "missing-parent"
	err = globalDB.CreateTask(testutil.Context(t), childTask)
	if !errors.Is(err, taskpkg.ErrTaskNotFound) {
		t.Fatalf("CreateTask(missing parent) error = %v, want ErrTaskNotFound", err)
	}

	run := taskRunForTest("run-missing-task", "missing-task")
	err = globalDB.CreateTaskRun(testutil.Context(t), run)
	if !errors.Is(err, taskpkg.ErrTaskNotFound) {
		t.Fatalf("CreateTaskRun(missing task) error = %v, want ErrTaskNotFound", err)
	}
}

func TestTaskNormalizationDefaultsAndHelpers(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	globalDB.now = func() time.Time {
		return time.Date(2026, 4, 14, 15, 0, 0, 0, time.UTC)
	}

	record := taskRecordForTest("task-defaults")
	record.CreatedAt = time.Time{}
	record.UpdatedAt = time.Time{}
	record.Owner = ownershipForTest(taskpkg.OwnerKindHuman, " alice ")
	normalizedTask, err := globalDB.normalizeTaskForCreate(record)
	if err != nil {
		t.Fatalf("normalizeTaskForCreate() error = %v", err)
	}
	if !normalizedTask.CreatedAt.Equal(globalDB.now()) {
		t.Fatalf("normalizeTaskForCreate().CreatedAt = %v, want %v", normalizedTask.CreatedAt, globalDB.now())
	}
	if !normalizedTask.UpdatedAt.Equal(globalDB.now()) {
		t.Fatalf("normalizeTaskForCreate().UpdatedAt = %v, want %v", normalizedTask.UpdatedAt, globalDB.now())
	}
	if normalizedTask.Owner == nil || normalizedTask.Owner.Ref != "alice" {
		t.Fatalf("normalizeTaskForCreate().Owner = %#v, want trimmed owner", normalizedTask.Owner)
	}

	updateRecord := taskRecordForTest("task-update-default")
	updateRecord.UpdatedAt = time.Time{}
	normalizedUpdate, err := globalDB.normalizeTaskForUpdate(updateRecord)
	if err != nil {
		t.Fatalf("normalizeTaskForUpdate() error = %v", err)
	}
	if !normalizedUpdate.UpdatedAt.Equal(globalDB.now()) {
		t.Fatalf("normalizeTaskForUpdate().UpdatedAt = %v, want %v", normalizedUpdate.UpdatedAt, globalDB.now())
	}

	run := taskRunForTest("run-defaults", "task-defaults")
	run.Attempt = 0
	run.QueuedAt = time.Time{}
	normalizedRun, err := globalDB.normalizeTaskRunForCreate(run)
	if err != nil {
		t.Fatalf("normalizeTaskRunForCreate() error = %v", err)
	}
	if got, want := normalizedRun.Attempt, 1; got != want {
		t.Fatalf("normalizeTaskRunForCreate().Attempt = %d, want %d", got, want)
	}
	if !normalizedRun.QueuedAt.Equal(globalDB.now()) {
		t.Fatalf("normalizeTaskRunForCreate().QueuedAt = %v, want %v", normalizedRun.QueuedAt, globalDB.now())
	}

	runs, err := globalDB.ListTaskRunsByStatus(testutil.Context(t), nil)
	if err != nil {
		t.Fatalf("ListTaskRunsByStatus(nil) error = %v", err)
	}
	if got := len(runs); got != 0 {
		t.Fatalf("len(ListTaskRunsByStatus(nil)) = %d, want 0", got)
	}

	if _, err := requireTaskValue("", "task id"); err == nil {
		t.Fatal("requireTaskValue(empty) error = nil, want non-nil")
	}

	decoded, err := decodeTaskJSON(sqlNullStringForTest(`{"ok":true}`), "test")
	if err != nil {
		t.Fatalf("decodeTaskJSON(valid) error = %v", err)
	}
	if got, want := string(decoded), `{"ok":true}`; got != want {
		t.Fatalf("decodeTaskJSON(valid) = %q, want %q", got, want)
	}
	if _, err := decodeTaskJSON(sqlNullStringForTest(`{"ok":`), "test"); err == nil {
		t.Fatal("decodeTaskJSON(invalid) error = nil, want non-nil")
	}
}

func TestGlobalDBTaskTriageStateRoundTripAndActorIsolation(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	taskRecord := taskRecordForTest("task-triage-roundtrip")
	if err := globalDB.CreateTask(testutil.Context(t), taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	aliceState := taskpkg.TriageState{
		TaskID:             taskRecord.ID,
		Actor:              taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "user:alice"},
		Read:               true,
		Archived:           true,
		Dismissed:          false,
		LastSeenActivityAt: taskRecord.UpdatedAt.Add(5 * time.Minute),
		UpdatedAt:          taskRecord.UpdatedAt.Add(6 * time.Minute),
	}
	if err := globalDB.UpsertTaskTriageState(testutil.Context(t), aliceState); err != nil {
		t.Fatalf("UpsertTaskTriageState(alice) error = %v", err)
	}

	bobState := taskpkg.TriageState{
		TaskID:    taskRecord.ID,
		Actor:     taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "user:bob"},
		Read:      false,
		Archived:  false,
		Dismissed: true,
		UpdatedAt: taskRecord.UpdatedAt.Add(7 * time.Minute),
	}
	if err := globalDB.UpsertTaskTriageState(testutil.Context(t), bobState); err != nil {
		t.Fatalf("UpsertTaskTriageState(bob) error = %v", err)
	}

	storedAlice, err := globalDB.GetTaskTriageState(testutil.Context(t), taskRecord.ID, aliceState.Actor)
	if err != nil {
		t.Fatalf("GetTaskTriageState(alice) error = %v", err)
	}
	if storedAlice != aliceState {
		t.Fatalf("storedAlice = %#v, want %#v", storedAlice, aliceState)
	}

	storedBob, err := globalDB.GetTaskTriageState(testutil.Context(t), taskRecord.ID, bobState.Actor)
	if err != nil {
		t.Fatalf("GetTaskTriageState(bob) error = %v", err)
	}
	if storedBob != bobState {
		t.Fatalf("storedBob = %#v, want %#v", storedBob, bobState)
	}

	aliceState.Archived = false
	aliceState.Dismissed = true
	aliceState.UpdatedAt = aliceState.UpdatedAt.Add(time.Minute)
	if err := globalDB.UpsertTaskTriageState(testutil.Context(t), aliceState); err != nil {
		t.Fatalf("UpsertTaskTriageState(alice update) error = %v", err)
	}

	updatedAlice, err := globalDB.GetTaskTriageState(testutil.Context(t), taskRecord.ID, aliceState.Actor)
	if err != nil {
		t.Fatalf("GetTaskTriageState(updated alice) error = %v", err)
	}
	if updatedAlice != aliceState {
		t.Fatalf("updatedAlice = %#v, want %#v", updatedAlice, aliceState)
	}

	if _, err := globalDB.GetTaskTriageState(
		testutil.Context(t),
		taskRecord.ID,
		taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "user:charlie"},
	); !errors.Is(err, taskpkg.ErrTaskTriageStateNotFound) {
		t.Fatalf("GetTaskTriageState(missing) error = %v, want %v", err, taskpkg.ErrTaskTriageStateNotFound)
	}
}

func TestGlobalDBListTaskTriageStatesFiltersByActorAndOrdersByUpdate(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	firstTask := taskRecordForTest("task-triage-list-first")
	secondTask := taskRecordForTest("task-triage-list-second")
	secondTask.UpdatedAt = secondTask.UpdatedAt.Add(2 * time.Minute)
	if err := globalDB.CreateTask(testutil.Context(t), firstTask); err != nil {
		t.Fatalf("CreateTask(firstTask) error = %v", err)
	}
	if err := globalDB.CreateTask(testutil.Context(t), secondTask); err != nil {
		t.Fatalf("CreateTask(secondTask) error = %v", err)
	}

	alice := taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "user:alice"}
	aliceFirst := taskpkg.TriageState{
		TaskID:             firstTask.ID,
		Actor:              alice,
		Read:               true,
		LastSeenActivityAt: firstTask.UpdatedAt,
		UpdatedAt:          firstTask.UpdatedAt.Add(5 * time.Minute),
	}
	aliceSecond := taskpkg.TriageState{
		TaskID:             secondTask.ID,
		Actor:              alice,
		Archived:           true,
		LastSeenActivityAt: secondTask.UpdatedAt,
		UpdatedAt:          secondTask.UpdatedAt.Add(8 * time.Minute),
	}
	bob := taskpkg.TriageState{
		TaskID:    secondTask.ID,
		Actor:     taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "user:bob"},
		Dismissed: true,
		UpdatedAt: secondTask.UpdatedAt.Add(9 * time.Minute),
	}
	for _, state := range []taskpkg.TriageState{aliceFirst, aliceSecond, bob} {
		if err := globalDB.UpsertTaskTriageState(testutil.Context(t), state); err != nil {
			t.Fatalf("UpsertTaskTriageState(%q/%q) error = %v", state.Actor.Kind, state.Actor.Ref, err)
		}
	}

	aliceStates, err := globalDB.ListTaskTriageStates(testutil.Context(t), alice)
	if err != nil {
		t.Fatalf("ListTaskTriageStates(alice) error = %v", err)
	}
	if got, want := len(aliceStates), 2; got != want {
		t.Fatalf("len(ListTaskTriageStates(alice)) = %d, want %d", got, want)
	}
	if got, want := []string{
		aliceStates[0].TaskID,
		aliceStates[1].TaskID,
	}, []string{
		secondTask.ID,
		firstTask.ID,
	}; !testutil.EqualStringSlices(
		got,
		want,
	) {
		t.Fatalf("alice task ids = %#v, want %#v", got, want)
	}
	if aliceStates[0] != aliceSecond {
		t.Fatalf("aliceStates[0] = %#v, want %#v", aliceStates[0], aliceSecond)
	}
	if aliceStates[1] != aliceFirst {
		t.Fatalf("aliceStates[1] = %#v, want %#v", aliceStates[1], aliceFirst)
	}

	bobStates, err := globalDB.ListTaskTriageStates(
		testutil.Context(t),
		taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "user:bob"},
	)
	if err != nil {
		t.Fatalf("ListTaskTriageStates(bob) error = %v", err)
	}
	if got, want := len(bobStates), 1; got != want {
		t.Fatalf("len(ListTaskTriageStates(bob)) = %d, want %d", got, want)
	}
	if bobStates[0] != bob {
		t.Fatalf("bobStates[0] = %#v, want %#v", bobStates[0], bob)
	}
}

func TestOpenGlobalDBMigratesLegacyTaskSchemaAndPreservesRows(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	dbPath := filepath.Join(t.TempDir(), GlobalDatabaseName)

	legacyDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	if _, err := legacyDB.ExecContext(ctx, `CREATE TABLE tasks (
		id              TEXT PRIMARY KEY,
		identifier      TEXT,
		scope           TEXT NOT NULL CHECK (scope IN ('global', 'workspace')),
		workspace_id    TEXT,
		parent_task_id  TEXT,
		network_channel TEXT,
		title           TEXT NOT NULL,
		description     TEXT,
		status          TEXT NOT NULL CHECK (
			status IN ('pending', 'blocked', 'ready', 'in_progress', 'completed', 'failed', 'canceled')
		),
		owner_kind      TEXT,
		owner_ref       TEXT,
		created_by_kind TEXT NOT NULL,
		created_by_ref  TEXT NOT NULL,
		origin_kind     TEXT NOT NULL,
		origin_ref      TEXT NOT NULL,
		created_at      TEXT NOT NULL,
		updated_at      TEXT NOT NULL,
		closed_at       TEXT,
		metadata_json   TEXT
	)`); err != nil {
		t.Fatalf("create legacy tasks table error = %v", err)
	}
	if _, err := legacyDB.ExecContext(ctx, `INSERT INTO tasks (
		id, identifier, scope, workspace_id, parent_task_id, network_channel, title, description, status,
		owner_kind, owner_ref, created_by_kind, created_by_ref, origin_kind, origin_ref,
		created_at, updated_at, closed_at, metadata_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"legacy-task-1",
		"identifier-legacy-task-1",
		string(taskpkg.ScopeGlobal),
		nil,
		nil,
		nil,
		"Legacy task",
		"Legacy description",
		string(taskpkg.TaskStatusPending),
		nil,
		nil,
		string(taskpkg.ActorKindHuman),
		"user:alice",
		string(taskpkg.OriginKindCLI),
		"cli",
		"2026-04-14T12:00:00.000000000Z",
		"2026-04-14T12:00:00.000000000Z",
		nil,
		`{"legacy":true}`,
	); err != nil {
		t.Fatalf("insert legacy task error = %v", err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("legacyDB.Close() error = %v", err)
	}

	globalDB, err := OpenGlobalDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := globalDB.Close(ctx); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	assertIndexesPresent(t, globalDB.db, "tasks",
		"idx_tasks_scope",
		"idx_tasks_workspace",
		"idx_tasks_status",
		"idx_tasks_priority",
		"idx_tasks_approval_state",
		"idx_tasks_parent",
		"idx_tasks_owner",
		"idx_tasks_channel",
	)

	stored, err := globalDB.GetTask(ctx, "legacy-task-1")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got, want := stored.Priority, taskpkg.DefaultPriority; got != want {
		t.Fatalf("stored.Priority = %q, want %q", got, want)
	}
	if got, want := stored.MaxAttempts, taskpkg.DefaultTaskMaxAttempts; got != want {
		t.Fatalf("stored.MaxAttempts = %d, want %d", got, want)
	}
	if got, want := stored.ApprovalPolicy, taskpkg.ApprovalPolicyNone; got != want {
		t.Fatalf("stored.ApprovalPolicy = %q, want %q", got, want)
	}
	if got, want := stored.ApprovalState, taskpkg.ApprovalStateNotRequired; got != want {
		t.Fatalf("stored.ApprovalState = %q, want %q", got, want)
	}

	stored.Priority = taskpkg.PriorityUrgent
	stored.MaxAttempts = 5
	stored.ApprovalPolicy = taskpkg.ApprovalPolicyManual
	stored.ApprovalState = taskpkg.ApprovalStateApproved
	stored.UpdatedAt = stored.UpdatedAt.Add(time.Minute)
	if err := globalDB.UpdateTask(ctx, stored); err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}

	updated, err := globalDB.GetTask(ctx, stored.ID)
	if err != nil {
		t.Fatalf("GetTask(updated) error = %v", err)
	}
	assertTaskEqual(t, updated, stored)
}

func TestOpenGlobalDBMigratesLegacyTaskEventsToStableSequences(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	dbPath := filepath.Join(t.TempDir(), GlobalDatabaseName)

	legacyDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	if _, err := legacyDB.ExecContext(ctx, `CREATE TABLE tasks (
		id TEXT PRIMARY KEY,
		identifier TEXT,
		scope TEXT NOT NULL,
		workspace_id TEXT,
		parent_task_id TEXT,
		network_channel TEXT,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL,
		owner_kind TEXT,
		owner_ref TEXT,
		created_by_kind TEXT NOT NULL,
		created_by_ref TEXT NOT NULL,
		origin_kind TEXT NOT NULL,
		origin_ref TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		closed_at TEXT,
		metadata_json TEXT
	)`); err != nil {
		t.Fatalf("create legacy tasks table error = %v", err)
	}
	if _, err := legacyDB.ExecContext(ctx, `CREATE TABLE task_events (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL,
		run_id TEXT,
		event_type TEXT NOT NULL,
		actor_kind TEXT NOT NULL,
		actor_ref TEXT NOT NULL,
		origin_kind TEXT NOT NULL,
		origin_ref TEXT NOT NULL,
		payload_json TEXT,
		timestamp TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy task_events table error = %v", err)
	}
	if _, err := legacyDB.ExecContext(ctx, `INSERT INTO tasks (
		id, identifier, scope, workspace_id, parent_task_id, network_channel, title, description, status,
		owner_kind, owner_ref, created_by_kind, created_by_ref, origin_kind, origin_ref,
		created_at, updated_at, closed_at, metadata_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"legacy-task-events",
		nil,
		string(taskpkg.ScopeGlobal),
		nil,
		nil,
		nil,
		"Legacy event task",
		nil,
		string(taskpkg.TaskStatusReady),
		nil,
		nil,
		string(taskpkg.ActorKindHuman),
		"user:alice",
		string(taskpkg.OriginKindCLI),
		"cli",
		"2026-04-14T12:00:00.000000000Z",
		"2026-04-14T12:00:00.000000000Z",
		nil,
		nil,
	); err != nil {
		t.Fatalf("insert legacy task error = %v", err)
	}
	for _, args := range [][]any{
		{"evt-1", "legacy-task-events", nil, "task.created", string(taskpkg.ActorKindHuman), "user:alice", string(taskpkg.OriginKindCLI), "cli", nil, "2026-04-14T12:00:00.000000000Z"},
		{"evt-2", "legacy-task-events", nil, "task.updated", string(taskpkg.ActorKindHuman), "user:alice", string(taskpkg.OriginKindCLI), "cli", nil, "2026-04-14T12:05:00.000000000Z"},
	} {
		if _, err := legacyDB.ExecContext(ctx, `INSERT INTO task_events (
			id, task_id, run_id, event_type, actor_kind, actor_ref, origin_kind, origin_ref, payload_json, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, args...); err != nil {
			t.Fatalf("insert legacy task event error = %v", err)
		}
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("legacyDB.Close() error = %v", err)
	}

	globalDB, err := OpenGlobalDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := globalDB.Close(ctx); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	assertTableColumns(t, globalDB.db, "task_events", []string{
		"id",
		"task_id",
		"run_id",
		"event_type",
		"actor_kind",
		"actor_ref",
		"origin_kind",
		"origin_ref",
		"payload_json",
		"timestamp",
		"event_seq",
	})
	assertIndexesPresent(t, globalDB.db, "task_events",
		"uq_task_events_event_seq",
		"idx_task_events_task_seq",
	)

	record, err := globalDB.GetTaskEventRecord(ctx, "evt-2")
	if err != nil {
		t.Fatalf("GetTaskEventRecord() error = %v", err)
	}
	if got, want := record.Sequence, int64(2); got != want {
		t.Fatalf("record.Sequence = %d, want %d", got, want)
	}

	records, err := globalDB.ListTaskEventRecords(ctx, taskpkg.EventRecordQuery{
		TaskID:        "legacy-task-events",
		AfterSequence: 0,
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListTaskEventRecords() error = %v", err)
	}
	if got, want := []int64{
		records[0].Sequence,
		records[1].Sequence,
	}, []int64{
		1,
		2,
	}; got[0] != want[0] ||
		got[1] != want[1] {
		t.Fatalf("record sequences = %#v, want %#v", got, want)
	}
}

func taskRecordForTest(id string) taskpkg.Task {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	return taskpkg.Task{
		ID:             id,
		Identifier:     "identifier-" + id,
		Scope:          taskpkg.ScopeGlobal,
		Title:          "Task " + id,
		Description:    "Description for " + id,
		Priority:       taskpkg.DefaultPriority,
		MaxAttempts:    taskpkg.DefaultTaskMaxAttempts,
		Status:         taskpkg.TaskStatusPending,
		ApprovalPolicy: taskpkg.ApprovalPolicyNone,
		ApprovalState:  taskpkg.ApprovalStateNotRequired,
		CreatedBy: taskpkg.ActorIdentity{
			Kind: taskpkg.ActorKindHuman,
			Ref:  "user:alice",
		},
		Origin: taskpkg.Origin{
			Kind: taskpkg.OriginKindCLI,
			Ref:  "cli",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func taskRunForTest(id string, taskID string) taskpkg.Run {
	queuedAt := time.Date(2026, 4, 14, 13, 0, 0, 0, time.UTC)
	return taskpkg.Run{
		ID:       id,
		TaskID:   taskID,
		Status:   taskpkg.TaskRunStatusQueued,
		Attempt:  1,
		Origin:   taskpkg.Origin{Kind: taskpkg.OriginKindDaemon, Ref: "scheduler"},
		QueuedAt: queuedAt,
	}
}

func ownershipForTest(kind taskpkg.OwnerKind, ref string) *taskpkg.Ownership {
	return &taskpkg.Ownership{Kind: kind, Ref: ref}
}

func actorForTest(kind taskpkg.ActorKind, ref string) *taskpkg.ActorIdentity {
	return &taskpkg.ActorIdentity{Kind: kind, Ref: ref}
}

func assertTaskEqual(t *testing.T, got taskpkg.Task, want taskpkg.Task) {
	t.Helper()

	if got.ID != want.ID ||
		got.Identifier != want.Identifier ||
		got.Scope != want.Scope ||
		got.WorkspaceID != want.WorkspaceID ||
		got.ParentTaskID != want.ParentTaskID ||
		got.NetworkChannel != want.NetworkChannel ||
		got.Title != want.Title ||
		got.Description != want.Description ||
		got.Priority != want.Priority ||
		got.MaxAttempts != want.MaxAttempts ||
		got.Status != want.Status ||
		got.ApprovalPolicy != want.ApprovalPolicy ||
		got.ApprovalState != want.ApprovalState ||
		got.CreatedBy != want.CreatedBy ||
		got.Origin != want.Origin ||
		!got.CreatedAt.Equal(want.CreatedAt) ||
		!got.UpdatedAt.Equal(want.UpdatedAt) ||
		!got.ClosedAt.Equal(want.ClosedAt) ||
		string(got.Metadata) != string(want.Metadata) {
		t.Fatalf("task = %#v, want %#v", got, want)
	}
	assertOwnershipEqual(t, got.Owner, want.Owner)
}

func assertTaskSummaryMatchesTask(t *testing.T, got taskpkg.Summary, want taskpkg.Task) {
	t.Helper()

	if got.ID != want.ID ||
		got.Identifier != want.Identifier ||
		got.Scope != want.Scope ||
		got.WorkspaceID != want.WorkspaceID ||
		got.ParentTaskID != want.ParentTaskID ||
		got.NetworkChannel != want.NetworkChannel ||
		got.Title != want.Title ||
		got.Priority != want.Priority ||
		got.MaxAttempts != want.MaxAttempts ||
		got.Status != want.Status ||
		got.ApprovalPolicy != want.ApprovalPolicy ||
		got.ApprovalState != want.ApprovalState ||
		got.Draft != (want.Status == taskpkg.TaskStatusDraft) ||
		got.CreatedBy != want.CreatedBy ||
		got.Origin != want.Origin ||
		!got.CreatedAt.Equal(want.CreatedAt) ||
		!got.UpdatedAt.Equal(want.UpdatedAt) ||
		!got.ClosedAt.Equal(want.ClosedAt) {
		t.Fatalf("task summary = %#v, want task %#v", got, want)
	}
	assertOwnershipEqual(t, got.Owner, want.Owner)
}

func assertTaskRunEqual(t *testing.T, got taskpkg.Run, want taskpkg.Run) {
	t.Helper()

	if got.ID != want.ID ||
		got.TaskID != want.TaskID ||
		got.Status != want.Status ||
		got.Attempt != want.Attempt ||
		got.SessionID != want.SessionID ||
		got.Origin != want.Origin ||
		got.IdempotencyKey != want.IdempotencyKey ||
		got.NetworkChannel != want.NetworkChannel ||
		!got.QueuedAt.Equal(want.QueuedAt) ||
		!got.ClaimedAt.Equal(want.ClaimedAt) ||
		!got.StartedAt.Equal(want.StartedAt) ||
		!got.EndedAt.Equal(want.EndedAt) ||
		got.Error != want.Error ||
		string(got.Metadata) != string(want.Metadata) ||
		string(got.Result) != string(want.Result) {
		t.Fatalf("task run = %#v, want %#v", got, want)
	}
	assertActorEqual(t, got.ClaimedBy, want.ClaimedBy)
}

func assertOwnershipEqual(t *testing.T, got *taskpkg.Ownership, want *taskpkg.Ownership) {
	t.Helper()

	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("ownership = %#v, want %#v", got, want)
	case *got != *want:
		t.Fatalf("ownership = %#v, want %#v", *got, *want)
	}
}

func assertActorEqual(t *testing.T, got *taskpkg.ActorIdentity, want *taskpkg.ActorIdentity) {
	t.Helper()

	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("actor = %#v, want %#v", got, want)
	case *got != *want:
		t.Fatalf("actor = %#v, want %#v", *got, *want)
	}
}

func taskSummaryIDs(summaries []taskpkg.Summary) []string {
	ids := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		ids = append(ids, summary.ID)
	}
	sort.Strings(ids)
	return ids
}

func orderedTaskSummaryIDs(summaries []taskpkg.Summary) []string {
	ids := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		ids = append(ids, summary.ID)
	}
	return ids
}

func sqlNullStringForTest(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
