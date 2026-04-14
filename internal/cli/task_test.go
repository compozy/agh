package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	taskpkg "github.com/pedronauck/agh/internal/task"
)

func TestTaskCreateAndUpdateRejectInvalidFlagCombos(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "create requires workspace for workspace scope",
			args:    []string{"task", "create", "--scope", "workspace", "--title", "Investigate"},
			wantErr: "--workspace is required when --scope is workspace",
		},
		{
			name:    "create forbids workspace for global scope",
			args:    []string{"task", "create", "--scope", "global", "--workspace", "alpha", "--title", "Investigate"},
			wantErr: "--workspace must be empty when --scope is global",
		},
		{
			name:    "update requires change flags",
			args:    []string{"task", "update", "task-1"},
			wantErr: "task update requires at least one change flag",
		},
		{
			name:    "update rejects clear owner with owner mutation",
			args:    []string{"task", "update", "task-1", "--clear-owner", "--owner-kind", "pool", "--owner-ref", "triage"},
			wantErr: "--clear-owner cannot be combined with --owner-kind or --owner-ref",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := executeRootCommand(t, newTestDeps(t, stubClient{}), tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("executeRootCommand(%v) error = %v, want %q", tt.args, err, tt.wantErr)
			}
		})
	}
}

func TestTaskCreateAndListCommandsParseTaskFields(t *testing.T) {
	t.Parallel()

	var (
		createRequest CreateTaskRequest
		listQuery     TaskListQuery
	)

	deps := newTestDeps(t, stubClient{
		createTaskFn: func(_ context.Context, got CreateTaskRequest) (TaskRecord, error) {
			createRequest = got
			return sampleTaskRecord(), nil
		},
		listTasksFn: func(_ context.Context, query TaskListQuery) ([]TaskSummaryRecord, error) {
			listQuery = query
			return []TaskSummaryRecord{sampleTaskSummaryRecord()}, nil
		},
	})

	createJSON, _, err := executeRootCommand(
		t,
		deps,
		"task", "create",
		"--id", "task-1",
		"--identifier", "OPS-42",
		"--scope", "workspace",
		"--workspace", "alpha",
		"--channel", "builders",
		"--title", "Investigate flaky task runs",
		"--description", "Capture root cause",
		"--owner-kind", "pool",
		"--owner-ref", "triage",
		"--metadata", `{"priority":"high"}`,
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("task create error = %v", err)
	}

	if createRequest.Scope != taskpkg.ScopeWorkspace ||
		createRequest.Workspace != "alpha" ||
		createRequest.NetworkChannel != "builders" ||
		createRequest.Title != "Investigate flaky task runs" ||
		createRequest.Owner == nil ||
		createRequest.Owner.Kind != taskpkg.OwnerKindPool ||
		createRequest.Owner.Ref != "triage" ||
		string(createRequest.Metadata) != `{"priority":"high"}` {
		t.Fatalf("createRequest = %#v, want parsed workspace/channel/owner/metadata", createRequest)
	}

	var created TaskRecord
	if err := json.Unmarshal([]byte(createJSON), &created); err != nil {
		t.Fatalf("json.Unmarshal(task create) error = %v", err)
	}
	if created.ID != "task-1" || created.Title != "Investigate flaky task runs" {
		t.Fatalf("created task = %#v, want sample task output", created)
	}

	listJSON, _, err := executeRootCommand(
		t,
		deps,
		"task", "list",
		"--scope", "workspace",
		"--workspace", "alpha",
		"--status", "ready",
		"--owner-kind", "pool",
		"--owner-ref", "triage",
		"--parent", "task-root",
		"--channel", "builders",
		"--last", "3",
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("task list error = %v", err)
	}

	if listQuery.Scope != taskpkg.ScopeWorkspace ||
		listQuery.Workspace != "alpha" ||
		listQuery.Status != taskpkg.TaskStatusReady ||
		listQuery.OwnerKind != taskpkg.OwnerKindPool ||
		listQuery.OwnerRef != "triage" ||
		listQuery.ParentTaskID != "task-root" ||
		listQuery.NetworkChannel != "builders" ||
		listQuery.Limit != 3 {
		t.Fatalf("listQuery = %#v, want parsed filters", listQuery)
	}

	var listed []TaskSummaryRecord
	if err := json.Unmarshal([]byte(listJSON), &listed); err != nil {
		t.Fatalf("json.Unmarshal(task list) error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != "task-1" {
		t.Fatalf("listed tasks = %#v, want one task summary", listed)
	}
}

func TestTaskRunCommandsMapLifecycleRequests(t *testing.T) {
	t.Parallel()

	var (
		runListQuery    TaskRunListQuery
		enqueueRequest  EnqueueTaskRunRequest
		claimRequest    ClaimTaskRunRequest
		startRequest    StartTaskRunRequest
		attachRequest   AttachTaskRunSessionRequest
		completeRequest CompleteTaskRunRequest
		failRequest     FailTaskRunRequest
		cancelRequest   CancelTaskRunRequest
	)

	deps := newTestDeps(t, stubClient{
		listTaskRunsFn: func(_ context.Context, _ string, query TaskRunListQuery) ([]TaskRunRecord, error) {
			runListQuery = query
			return []TaskRunRecord{sampleTaskRunRecord(taskpkg.TaskRunStatusRunning)}, nil
		},
		enqueueTaskRunFn: func(_ context.Context, _ string, request EnqueueTaskRunRequest) (TaskRunRecord, error) {
			enqueueRequest = request
			return sampleTaskRunRecord(taskpkg.TaskRunStatusQueued), nil
		},
		claimTaskRunFn: func(_ context.Context, _ string, request ClaimTaskRunRequest) (TaskRunRecord, error) {
			claimRequest = request
			return sampleTaskRunRecord(taskpkg.TaskRunStatusClaimed), nil
		},
		startTaskRunFn: func(_ context.Context, _ string, request StartTaskRunRequest) (TaskRunRecord, error) {
			startRequest = request
			return sampleTaskRunRecord(taskpkg.TaskRunStatusRunning), nil
		},
		attachTaskRunSessionFn: func(_ context.Context, _ string, request AttachTaskRunSessionRequest) (TaskRunRecord, error) {
			attachRequest = request
			return sampleTaskRunRecord(taskpkg.TaskRunStatusStarting), nil
		},
		completeTaskRunFn: func(_ context.Context, _ string, request CompleteTaskRunRequest) (TaskRunRecord, error) {
			completeRequest = request
			return sampleTaskRunRecord(taskpkg.TaskRunStatusCompleted), nil
		},
		failTaskRunFn: func(_ context.Context, _ string, request FailTaskRunRequest) (TaskRunRecord, error) {
			failRequest = request
			return sampleTaskRunRecord(taskpkg.TaskRunStatusFailed), nil
		},
		cancelTaskRunFn: func(_ context.Context, _ string, request CancelTaskRunRequest) (TaskRunRecord, error) {
			cancelRequest = request
			return sampleTaskRunRecord(taskpkg.TaskRunStatusCancelled), nil
		},
	})

	if _, _, err := executeRootCommand(t, deps, "task", "run", "list", "task-1", "--status", "running", "--session", "sess-1", "--last", "2", "-o", "json"); err != nil {
		t.Fatalf("task run list error = %v", err)
	}
	if runListQuery.Status != taskpkg.TaskRunStatusRunning || runListQuery.SessionID != "sess-1" || runListQuery.Limit != 2 {
		t.Fatalf("runListQuery = %#v, want parsed run filters", runListQuery)
	}

	if _, _, err := executeRootCommand(t, deps, "task", "run", "enqueue", "task-1", "--idempotency-key", "idem-1", "--channel", "builders", "-o", "json"); err != nil {
		t.Fatalf("task run enqueue error = %v", err)
	}
	if enqueueRequest.IdempotencyKey != "idem-1" || enqueueRequest.NetworkChannel != "builders" {
		t.Fatalf("enqueueRequest = %#v, want idempotency key and channel", enqueueRequest)
	}

	if _, _, err := executeRootCommand(t, deps, "task", "run", "claim", "run-1", "--idempotency-key", "idem-claim", "-o", "json"); err != nil {
		t.Fatalf("task run claim error = %v", err)
	}
	if claimRequest.IdempotencyKey != "idem-claim" {
		t.Fatalf("claimRequest = %#v, want idempotency key", claimRequest)
	}

	if _, _, err := executeRootCommand(t, deps, "task", "run", "start", "run-1", "--idempotency-key", "idem-start", "-o", "json"); err != nil {
		t.Fatalf("task run start error = %v", err)
	}
	if startRequest.IdempotencyKey != "idem-start" {
		t.Fatalf("startRequest = %#v, want idempotency key", startRequest)
	}

	if _, _, err := executeRootCommand(t, deps, "task", "run", "attach-session", "run-1", "--session", "sess-attach", "-o", "json"); err != nil {
		t.Fatalf("task run attach-session error = %v", err)
	}
	if attachRequest.SessionID != "sess-attach" {
		t.Fatalf("attachRequest = %#v, want session id", attachRequest)
	}

	if _, _, err := executeRootCommand(t, deps, "task", "run", "complete", "run-1", "--result", `{"ok":true}`, "-o", "json"); err != nil {
		t.Fatalf("task run complete error = %v", err)
	}
	if string(completeRequest.Result) != `{"ok":true}` {
		t.Fatalf("completeRequest = %#v, want JSON result", completeRequest)
	}

	if _, _, err := executeRootCommand(t, deps, "task", "run", "fail", "run-1", "--error", "boom", "--metadata", `{"code":"E_TASK"}`, "-o", "json"); err != nil {
		t.Fatalf("task run fail error = %v", err)
	}
	if failRequest.Error != "boom" || string(failRequest.Metadata) != `{"code":"E_TASK"}` {
		t.Fatalf("failRequest = %#v, want error and metadata", failRequest)
	}

	if _, _, err := executeRootCommand(t, deps, "task", "run", "cancel", "run-1", "--reason", "operator-request", "--metadata", `{"source":"cli"}`, "-o", "json"); err != nil {
		t.Fatalf("task run cancel error = %v", err)
	}
	if cancelRequest.Reason != "operator-request" || string(cancelRequest.Metadata) != `{"source":"cli"}` {
		t.Fatalf("cancelRequest = %#v, want reason and metadata", cancelRequest)
	}
}

func TestTaskMutationCommandsMapRequests(t *testing.T) {
	t.Parallel()

	var (
		updateTaskID       string
		updateRequest      UpdateTaskRequest
		cancelTaskID       string
		cancelRequest      CancelTaskRequest
		childParentID      string
		childCreateRequest CreateTaskChildRequest
		dependencyTaskID   string
		dependencyRequest  AddTaskDependencyRequest
		removeTaskID       string
		removeDependsOnID  string
	)

	deps := newTestDeps(t, stubClient{
		updateTaskFn: func(_ context.Context, taskID string, request UpdateTaskRequest) (TaskRecord, error) {
			updateTaskID = taskID
			updateRequest = request
			return sampleTaskRecord(), nil
		},
		cancelTaskFn: func(_ context.Context, taskID string, request CancelTaskRequest) (TaskRecord, error) {
			cancelTaskID = taskID
			cancelRequest = request
			return sampleTaskRecord(), nil
		},
		createChildTaskFn: func(_ context.Context, parentID string, request CreateTaskChildRequest) (TaskRecord, error) {
			childParentID = parentID
			childCreateRequest = request
			return sampleTaskRecord(), nil
		},
		addTaskDependencyFn: func(_ context.Context, taskID string, request AddTaskDependencyRequest) (TaskDetailRecord, error) {
			dependencyTaskID = taskID
			dependencyRequest = request
			return sampleTaskDetailRecord(), nil
		},
		removeTaskDependencyFn: func(_ context.Context, taskID string, dependsOnID string) (TaskDetailRecord, error) {
			removeTaskID = taskID
			removeDependsOnID = dependsOnID
			return sampleTaskDetailRecord(), nil
		},
	})

	if _, _, err := executeRootCommand(
		t,
		deps,
		"task", "update", "task-1",
		"--title", "Retitle triage task",
		"--description", "Refined scope",
		"--channel", "builders",
		"--owner-kind", "pool",
		"--owner-ref", "triage",
		"--metadata", `{"priority":"low"}`,
		"-o", "json",
	); err != nil {
		t.Fatalf("task update error = %v", err)
	}
	if updateTaskID != "task-1" ||
		updateRequest.Title == nil || *updateRequest.Title != "Retitle triage task" ||
		updateRequest.Description == nil || *updateRequest.Description != "Refined scope" ||
		updateRequest.NetworkChannel == nil || *updateRequest.NetworkChannel != "builders" ||
		updateRequest.Owner == nil || updateRequest.Owner.Kind != taskpkg.OwnerKindPool || updateRequest.Owner.Ref != "triage" ||
		updateRequest.ClearOwner ||
		updateRequest.Metadata == nil || string(*updateRequest.Metadata) != `{"priority":"low"}` {
		t.Fatalf("update request = %#v, want parsed task mutation payload", updateRequest)
	}

	if _, _, err := executeRootCommand(t, deps, "task", "cancel", "task-1", "--reason", "operator-request", "--metadata", `{"source":"cli"}`, "-o", "json"); err != nil {
		t.Fatalf("task cancel error = %v", err)
	}
	if cancelTaskID != "task-1" || cancelRequest.Reason != "operator-request" || string(cancelRequest.Metadata) != `{"source":"cli"}` {
		t.Fatalf("cancel request = %#v, want parsed cancel payload", cancelRequest)
	}

	if _, _, err := executeRootCommand(
		t,
		deps,
		"task", "child", "create", "task-root",
		"--id", "task-child",
		"--identifier", "OPS-43",
		"--scope", "workspace",
		"--workspace", "alpha",
		"--channel", "builders",
		"--title", "Check runtime logs",
		"--description", "Focus on worker output",
		"--owner-kind", "human",
		"--owner-ref", "alice",
		"--metadata", `{"phase":"two"}`,
		"-o", "json",
	); err != nil {
		t.Fatalf("task child create error = %v", err)
	}
	if childParentID != "task-root" ||
		childCreateRequest.ID != "task-child" ||
		childCreateRequest.Identifier != "OPS-43" ||
		childCreateRequest.Scope != taskpkg.ScopeWorkspace ||
		childCreateRequest.Workspace != "alpha" ||
		childCreateRequest.NetworkChannel != "builders" ||
		childCreateRequest.Title != "Check runtime logs" ||
		childCreateRequest.Description != "Focus on worker output" ||
		childCreateRequest.Owner == nil || childCreateRequest.Owner.Kind != taskpkg.OwnerKindHuman || childCreateRequest.Owner.Ref != "alice" ||
		string(childCreateRequest.Metadata) != `{"phase":"two"}` {
		t.Fatalf("childCreateRequest = %#v, want parsed child task payload", childCreateRequest)
	}

	if _, _, err := executeRootCommand(t, deps, "task", "dependency", "add", "task-1", "--depends-on", "task-root", "--kind", "blocks", "-o", "json"); err != nil {
		t.Fatalf("task dependency add error = %v", err)
	}
	if dependencyTaskID != "task-1" || dependencyRequest.DependsOnTaskID != "task-root" || dependencyRequest.Kind != taskpkg.DependencyKindBlocks {
		t.Fatalf("dependencyRequest = %#v, want dependency payload", dependencyRequest)
	}

	if _, _, err := executeRootCommand(t, deps, "task", "dependency", "remove", "task-1", "task-root", "-o", "json"); err != nil {
		t.Fatalf("task dependency remove error = %v", err)
	}
	if removeTaskID != "task-1" || removeDependsOnID != "task-root" {
		t.Fatalf("remove dependency args = (%q, %q), want task-1/task-root", removeTaskID, removeDependsOnID)
	}
}

func TestTaskCommandsSupportDetailAndToonOutput(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, stubClient{
		getTaskFn: func(context.Context, string) (TaskDetailRecord, error) {
			return sampleTaskDetailRecord(), nil
		},
		listTasksFn: func(context.Context, TaskListQuery) ([]TaskSummaryRecord, error) {
			return []TaskSummaryRecord{sampleTaskSummaryRecord()}, nil
		},
	})

	humanOut, _, err := executeRootCommand(t, deps, "task", "get", "task-1", "-o", "human")
	if err != nil {
		t.Fatalf("task get human error = %v", err)
	}
	if !strings.Contains(humanOut, "Task") || !strings.Contains(humanOut, "Dependencies") || !strings.Contains(humanOut, "Task Runs") {
		t.Fatalf("task get human output = %q, want detail sections", humanOut)
	}

	toonOut, _, err := executeRootCommand(t, deps, "task", "list", "-o", "toon")
	if err != nil {
		t.Fatalf("task list toon error = %v", err)
	}
	if !strings.Contains(toonOut, "tasks[1]{id,identifier,scope,workspace_id,parent_task_id,status,owner,network_channel,title}:") {
		t.Fatalf("task list toon output = %q, want tasks TOON array", toonOut)
	}
}

func TestTaskBundlesRenderTaskRunAndDetailSections(t *testing.T) {
	t.Parallel()

	detail := sampleTaskDetailRecord()
	detailToon, err := taskDetailBundle(detail).toon()
	if err != nil {
		t.Fatalf("taskDetailBundle().toon() error = %v", err)
	}
	if !strings.Contains(detailToon, "task_children[1]{id,identifier,scope,workspace_id,status,owner,title}:") ||
		!strings.Contains(detailToon, "task_dependencies[1]{task_id,depends_on_task_id,kind,created_at}:") ||
		!strings.Contains(detailToon, "task_runs[1]{id,status,attempt,session_id,claimed_by,network_channel,queued_at,started_at,ended_at,error}:") ||
		!strings.Contains(detailToon, "task_events[1]{id,event_type,run_id,actor,origin,timestamp}:") {
		t.Fatalf("task detail toon output = %q, want child/dependency/run/event sections", detailToon)
	}

	runHuman, err := taskRunBundle(sampleTaskRunRecord(taskpkg.TaskRunStatusCompleted)).human()
	if err != nil {
		t.Fatalf("taskRunBundle().human() error = %v", err)
	}
	if !strings.Contains(runHuman, "Task Run") || !strings.Contains(runHuman, "Idempotency Key") || !strings.Contains(runHuman, "Result") {
		t.Fatalf("task run human output = %q, want task run detail section", runHuman)
	}

	runToon, err := taskRunListBundle([]TaskRunRecord{sampleTaskRunRecord(taskpkg.TaskRunStatusCompleted)}).toon()
	if err != nil {
		t.Fatalf("taskRunListBundle().toon() error = %v", err)
	}
	if !strings.Contains(runToon, "task_runs[1]{id,status,attempt,session_id,claimed_by,network_channel,queued_at,started_at,ended_at,error}:") {
		t.Fatalf("task run toon output = %q, want task run TOON array", runToon)
	}

	if kind, err := parseOptionalTaskDependencyKind("blocks"); err != nil || kind != taskpkg.DependencyKindBlocks {
		t.Fatalf("parseOptionalTaskDependencyKind(blocks) = (%q, %v), want blocks", kind, err)
	}
	if _, err := parseOptionalTaskDependencyKind("relates"); err == nil || !strings.Contains(err.Error(), "unsupported value") {
		t.Fatalf("parseOptionalTaskDependencyKind(relates) error = %v, want unsupported value validation", err)
	}
}

func sampleTaskSummaryRecord() TaskSummaryRecord {
	return TaskSummaryRecord{
		ID:             "task-1",
		Identifier:     "OPS-42",
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    "ws-alpha",
		ParentTaskID:   "task-root",
		NetworkChannel: "builders",
		Title:          "Investigate flaky task runs",
		Status:         taskpkg.TaskStatusReady,
		Owner:          &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: "triage"},
		CreatedBy:      taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
		Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindCLI, Ref: "tasks.create"},
		CreatedAt:      fixedTestNow,
		UpdatedAt:      fixedTestNow,
	}
}

func sampleTaskRecord() TaskRecord {
	return TaskRecord{
		ID:             "task-1",
		Identifier:     "OPS-42",
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    "ws-alpha",
		ParentTaskID:   "task-root",
		NetworkChannel: "builders",
		Title:          "Investigate flaky task runs",
		Description:    "Capture root cause",
		Status:         taskpkg.TaskStatusReady,
		Owner:          &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: "triage"},
		CreatedBy:      taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
		Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindCLI, Ref: "tasks.create"},
		CreatedAt:      fixedTestNow,
		UpdatedAt:      fixedTestNow,
		Metadata:       json.RawMessage(`{"priority":"high"}`),
	}
}

func sampleTaskRunRecord(status taskpkg.TaskRunStatus) TaskRunRecord {
	claimedAt := fixedTestNow.Add(time.Minute)
	startedAt := fixedTestNow.Add(2 * time.Minute)
	endedAt := fixedTestNow.Add(3 * time.Minute)
	return TaskRunRecord{
		ID:             "run-1",
		TaskID:         "task-1",
		Status:         status,
		Attempt:        1,
		ClaimedBy:      &taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
		SessionID:      "sess-1",
		Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindCLI, Ref: "tasks.run.start"},
		IdempotencyKey: "idem-run",
		NetworkChannel: "builders",
		QueuedAt:       fixedTestNow,
		ClaimedAt:      claimedAt,
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		Error:          "boom",
		Result:         json.RawMessage(`{"ok":true}`),
	}
}

func sampleTaskDetailRecord() TaskDetailRecord {
	return TaskDetailRecord{
		Task: sampleTaskRecord(),
		Children: []TaskSummaryRecord{
			{
				ID:          "task-child",
				Identifier:  "OPS-43",
				Scope:       taskpkg.ScopeWorkspace,
				WorkspaceID: "ws-alpha",
				Title:       "Check runtime logs",
				Status:      taskpkg.TaskStatusInProgress,
				Owner:       &taskpkg.Ownership{Kind: taskpkg.OwnerKindHuman, Ref: "alice"},
			},
		},
		Dependencies: []TaskDependencyRecord{
			{
				TaskID:          "task-1",
				DependsOnTaskID: "task-blocker",
				Kind:            taskpkg.DependencyKindBlocks,
				CreatedAt:       fixedTestNow,
			},
		},
		Runs: []TaskRunRecord{
			sampleTaskRunRecord(taskpkg.TaskRunStatusRunning),
		},
		Events: []TaskEventRecord{
			{
				ID:        "evt-1",
				TaskID:    "task-1",
				RunID:     "run-1",
				EventType: "task.run_started",
				Actor:     taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
				Origin:    taskpkg.Origin{Kind: taskpkg.OriginKindCLI, Ref: "tasks.run.start"},
				Timestamp: fixedTestNow,
			},
		},
	}
}
