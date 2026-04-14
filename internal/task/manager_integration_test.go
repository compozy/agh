//go:build integration

package task_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store/globaldb"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

type integrationStopCall struct {
	SessionID string
	Reason    taskpkg.StopReason
}

type integrationSessionExecutor struct {
	startCalls       []taskpkg.StartTaskSession
	requestStopCalls []integrationStopCall
	forceStopCalls   []integrationStopCall
}

func (e *integrationSessionExecutor) StartTaskSession(_ context.Context, spec taskpkg.StartTaskSession) (*taskpkg.SessionRef, error) {
	e.startCalls = append(e.startCalls, spec)
	return &taskpkg.SessionRef{SessionID: "sess-int-" + strconv.Itoa(len(e.startCalls))}, nil
}

func (e *integrationSessionExecutor) AttachTaskSession(_ context.Context, runID string, sessionID string) (*taskpkg.SessionRef, error) {
	return &taskpkg.SessionRef{SessionID: sessionID}, nil
}

func (e *integrationSessionExecutor) RequestTaskStop(_ context.Context, sessionID string, reason taskpkg.StopReason) error {
	e.requestStopCalls = append(e.requestStopCalls, integrationStopCall{SessionID: sessionID, Reason: reason})
	return nil
}

func (e *integrationSessionExecutor) ForceTaskStop(_ context.Context, sessionID string, reason taskpkg.StopReason) error {
	e.forceStopCalls = append(e.forceStopCalls, integrationStopCall{SessionID: sessionID, Reason: reason})
	return nil
}

func TestTaskManagerCreateTaskPersistsAgentSessionIdentity(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openTaskManagerGlobalDB(t)
	manager := newTaskManagerIntegration(t, db)

	actor, err := taskpkg.DeriveAgentSessionActorContext("sess-agent-1")
	if err != nil {
		t.Fatalf("DeriveAgentSessionActorContext() error = %v", err)
	}

	created, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Investigate task manager",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	stored, err := db.GetTask(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got, want := stored.CreatedBy.Kind, taskpkg.ActorKindAgentSession; got != want {
		t.Fatalf("stored.CreatedBy.Kind = %q, want %q", got, want)
	}
	if got, want := stored.CreatedBy.Ref, "sess-agent-1"; got != want {
		t.Fatalf("stored.CreatedBy.Ref = %q, want %q", got, want)
	}
	if got, want := stored.Origin.Kind, taskpkg.OriginKindAgentSession; got != want {
		t.Fatalf("stored.Origin.Kind = %q, want %q", got, want)
	}
	if got, want := stored.Origin.Ref, "sess-agent-1"; got != want {
		t.Fatalf("stored.Origin.Ref = %q, want %q", got, want)
	}
	if stored.Owner != nil {
		t.Fatalf("stored.Owner = %#v, want nil", stored.Owner)
	}
	if got, want := stored.Status, taskpkg.TaskStatusReady; got != want {
		t.Fatalf("stored.Status = %q, want %q", got, want)
	}

	events, err := db.ListTaskEvents(ctx, taskpkg.TaskEventQuery{TaskID: stored.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if got, want := events[0].EventType, "task.created"; got != want {
		t.Fatalf("events[0].EventType = %q, want %q", got, want)
	}
	if got, want := events[0].Actor.Kind, taskpkg.ActorKindAgentSession; got != want {
		t.Fatalf("events[0].Actor.Kind = %q, want %q", got, want)
	}
}

func TestTaskManagerChildAndDependencyFlowsPersistAudit(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openTaskManagerGlobalDB(t)
	manager := newTaskManagerIntegration(t, db)
	workspaceID := registerTaskManagerWorkspace(t, db, "task-manager-integration", filepath.Join(t.TempDir(), "workspace"))

	actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task create")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}

	parent, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Coordinator",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}
	child, err := manager.CreateChildTask(ctx, parent.ID, taskpkg.CreateTask{
		Scope:       taskpkg.ScopeWorkspace,
		WorkspaceID: workspaceID,
		Title:       "Workspace child",
		Owner:       &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: "triage"},
	}, actor)
	if err != nil {
		t.Fatalf("CreateChildTask() error = %v", err)
	}
	blocker, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    workspaceID,
		Title:          "Blocking task",
		NetworkChannel: "ops",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(blocker) error = %v", err)
	}

	if err := manager.AddDependency(ctx, taskpkg.AddDependency{
		TaskID:          child.ID,
		DependsOnTaskID: blocker.ID,
		Kind:            taskpkg.DependencyKindBlocks,
	}, actor); err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}

	storedChild, err := db.GetTask(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetTask(child) error = %v", err)
	}
	if got, want := storedChild.ParentTaskID, parent.ID; got != want {
		t.Fatalf("storedChild.ParentTaskID = %q, want %q", got, want)
	}
	if got, want := storedChild.Status, taskpkg.TaskStatusBlocked; got != want {
		t.Fatalf("storedChild.Status = %q, want %q", got, want)
	}

	dependencies, err := db.ListDependencies(ctx, child.ID)
	if err != nil {
		t.Fatalf("ListDependencies(child) error = %v", err)
	}
	if len(dependencies) != 1 {
		t.Fatalf("len(dependencies) = %d, want 1", len(dependencies))
	}
	if got, want := dependencies[0].DependsOnTaskID, blocker.ID; got != want {
		t.Fatalf("dependencies[0].DependsOnTaskID = %q, want %q", got, want)
	}

	childEvents, err := db.ListTaskEvents(ctx, taskpkg.TaskEventQuery{TaskID: child.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents(child) error = %v", err)
	}
	if !testutil.EqualStringSlices(sortedEventTypes(childEvents), []string{"task.created", "task.dependency_added"}) {
		t.Fatalf("child event types = %#v, want task.created + task.dependency_added", sortedEventTypes(childEvents))
	}

	parentEvents, err := db.ListTaskEvents(ctx, taskpkg.TaskEventQuery{TaskID: parent.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents(parent) error = %v", err)
	}
	if !containsEventType(parentEvents, "task.child_created") {
		t.Fatalf("parent events = %#v, want task.child_created", sortedEventTypes(parentEvents))
	}

	view, err := manager.GetTask(ctx, child.ID, actor)
	if err != nil {
		t.Fatalf("GetTask(view) error = %v", err)
	}
	if got, want := len(view.Dependencies), 1; got != want {
		t.Fatalf("len(view.Dependencies) = %d, want %d", got, want)
	}
	if got, want := view.Task.Status, taskpkg.TaskStatusBlocked; got != want {
		t.Fatalf("view.Task.Status = %q, want %q", got, want)
	}
}

func TestTaskManagerRunLifecyclePersistsAndReconcilesAgainstStorage(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openTaskManagerGlobalDB(t)
	executor := &integrationSessionExecutor{}
	manager := newTaskManagerIntegration(t, db, taskpkg.WithSessionExecutor(executor))
	actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task run")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}

	taskRecord, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Lifecycle integration",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	run, err := manager.EnqueueRun(ctx, taskpkg.EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	storedRun, err := db.GetTaskRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetTaskRun(queued) error = %v", err)
	}
	if got, want := storedRun.Status, taskpkg.TaskRunStatusQueued; got != want {
		t.Fatalf("queued run status = %q, want %q", got, want)
	}

	run, err = manager.ClaimRun(ctx, run.ID, taskpkg.ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun() error = %v", err)
	}
	if got, want := run.Status, taskpkg.TaskRunStatusClaimed; got != want {
		t.Fatalf("claimed run status = %q, want %q", got, want)
	}
	storedTask, err := db.GetTask(ctx, taskRecord.ID)
	if err != nil {
		t.Fatalf("GetTask(claimed) error = %v", err)
	}
	if got, want := storedTask.Status, taskpkg.TaskStatusReady; got != want {
		t.Fatalf("task status after claim = %q, want %q", got, want)
	}

	run, err = manager.StartRun(ctx, run.ID, taskpkg.StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun() error = %v", err)
	}
	if got, want := run.Status, taskpkg.TaskRunStatusRunning; got != want {
		t.Fatalf("running run status = %q, want %q", got, want)
	}
	if got := run.SessionID; got == "" {
		t.Fatal("run.SessionID = empty, want dedicated session id")
	}
	storedTask, err = db.GetTask(ctx, taskRecord.ID)
	if err != nil {
		t.Fatalf("GetTask(running) error = %v", err)
	}
	if got, want := storedTask.Status, taskpkg.TaskStatusInProgress; got != want {
		t.Fatalf("task status after start = %q, want %q", got, want)
	}

	run, err = manager.CompleteRun(ctx, run.ID, taskpkg.RunResult{
		Value: json.RawMessage(`{"result":"ok"}`),
	}, actor)
	if err != nil {
		t.Fatalf("CompleteRun() error = %v", err)
	}
	if got, want := run.Status, taskpkg.TaskRunStatusCompleted; got != want {
		t.Fatalf("completed run status = %q, want %q", got, want)
	}
	storedTask, err = db.GetTask(ctx, taskRecord.ID)
	if err != nil {
		t.Fatalf("GetTask(completed) error = %v", err)
	}
	if got, want := storedTask.Status, taskpkg.TaskStatusCompleted; got != want {
		t.Fatalf("task status after complete = %q, want %q", got, want)
	}

	events, err := db.ListTaskEvents(ctx, taskpkg.TaskEventQuery{TaskID: taskRecord.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	wantTypes := []string{
		"task.created",
		"task.run_claimed",
		"task.run_completed",
		"task.run_enqueued",
		"task.run_started",
		"task.run_starting",
	}
	if !testutil.EqualStringSlices(sortedEventTypes(events), wantTypes) {
		t.Fatalf("event types = %#v, want %#v", sortedEventTypes(events), wantTypes)
	}
}

func TestTaskManagerCancelTaskTreePersistsCancellationAudit(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openTaskManagerGlobalDB(t)
	executor := &integrationSessionExecutor{}
	manager := newTaskManagerIntegration(
		t,
		db,
		taskpkg.WithSessionExecutor(executor),
		taskpkg.WithCancelGracePeriod(0),
	)
	actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task cancel")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}

	parent, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Cancellation parent",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}
	queuedChild, err := manager.CreateChildTask(ctx, parent.ID, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Queued child",
	}, actor)
	if err != nil {
		t.Fatalf("CreateChildTask(queued child) error = %v", err)
	}
	activeChild, err := manager.CreateChildTask(ctx, parent.ID, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Active child",
	}, actor)
	if err != nil {
		t.Fatalf("CreateChildTask(active child) error = %v", err)
	}

	queuedRun, err := manager.EnqueueRun(ctx, taskpkg.EnqueueRun{TaskID: queuedChild.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(queued child) error = %v", err)
	}
	activeRun, err := manager.EnqueueRun(ctx, taskpkg.EnqueueRun{TaskID: activeChild.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(active child) error = %v", err)
	}
	activeRun, err = manager.ClaimRun(ctx, activeRun.ID, taskpkg.ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(active child) error = %v", err)
	}
	activeRun, err = manager.StartRun(ctx, activeRun.ID, taskpkg.StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun(active child) error = %v", err)
	}

	cancelledParent, err := manager.CancelTask(ctx, parent.ID, taskpkg.CancelTask{
		Reason: "stop tree",
	}, actor)
	if err != nil {
		t.Fatalf("CancelTask() error = %v", err)
	}
	if got, want := cancelledParent.Status, taskpkg.TaskStatusCancelled; got != want {
		t.Fatalf("cancelled parent status = %q, want %q", got, want)
	}

	for _, taskID := range []string{parent.ID, queuedChild.ID, activeChild.ID} {
		record, err := db.GetTask(ctx, taskID)
		if err != nil {
			t.Fatalf("GetTask(%q) error = %v", taskID, err)
		}
		if got, want := record.Status, taskpkg.TaskStatusCancelled; got != want {
			t.Fatalf("task %q status = %q, want %q", taskID, got, want)
		}
	}

	storedQueuedRun, err := db.GetTaskRun(ctx, queuedRun.ID)
	if err != nil {
		t.Fatalf("GetTaskRun(queued) error = %v", err)
	}
	if got, want := storedQueuedRun.Status, taskpkg.TaskRunStatusCancelled; got != want {
		t.Fatalf("queued child run status = %q, want %q", got, want)
	}
	storedActiveRun, err := db.GetTaskRun(ctx, activeRun.ID)
	if err != nil {
		t.Fatalf("GetTaskRun(active) error = %v", err)
	}
	if got, want := storedActiveRun.Status, taskpkg.TaskRunStatusCancelled; got != want {
		t.Fatalf("active child run status = %q, want %q", got, want)
	}

	if len(executor.requestStopCalls) != 1 {
		t.Fatalf("len(requestStopCalls) = %d, want 1", len(executor.requestStopCalls))
	}
	if got, want := executor.requestStopCalls[0].SessionID, activeRun.SessionID; got != want {
		t.Fatalf("requestStopCalls[0].SessionID = %q, want %q", got, want)
	}
	if len(executor.forceStopCalls) != 1 {
		t.Fatalf("len(forceStopCalls) = %d, want 1", len(executor.forceStopCalls))
	}

	parentEvents, err := db.ListTaskEvents(ctx, taskpkg.TaskEventQuery{TaskID: parent.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents(parent) error = %v", err)
	}
	if !containsEventType(parentEvents, "task.cancelled") {
		t.Fatalf("parent event types = %#v, want task.cancelled", sortedEventTypes(parentEvents))
	}

	activeChildEvents, err := db.ListTaskEvents(ctx, taskpkg.TaskEventQuery{TaskID: activeChild.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents(active child) error = %v", err)
	}
	if !containsEventType(activeChildEvents, "task.run_cancelled") {
		t.Fatalf("active child event types = %#v, want task.run_cancelled", sortedEventTypes(activeChildEvents))
	}
	if !containsEventType(activeChildEvents, "task.run_force_stopped") {
		t.Fatalf("active child event types = %#v, want task.run_force_stopped", sortedEventTypes(activeChildEvents))
	}
}

func openTaskManagerGlobalDB(t *testing.T) *globaldb.GlobalDB {
	t.Helper()

	ctx := testutil.Context(t)
	dbPath := filepath.Join(t.TempDir(), "agh.db")
	db, err := globaldb.OpenGlobalDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(ctx); err != nil {
			t.Fatalf("GlobalDB.Close() error = %v", err)
		}
	})
	return db
}

func newTaskManagerIntegration(t *testing.T, store taskpkg.Store, extraOpts ...taskpkg.Option) *taskpkg.TaskManager {
	t.Helper()

	options := []taskpkg.Option{taskpkg.WithStore(store)}
	options = append(options, extraOpts...)
	manager, err := taskpkg.NewManager(options...)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	return manager
}

func registerTaskManagerWorkspace(t *testing.T, db *globaldb.GlobalDB, name string, rootDir string) string {
	t.Helper()

	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", rootDir, err)
	}

	workspace := aghworkspace.Workspace{
		ID:        "ws-" + strings.ReplaceAll(name, " ", "-"),
		RootDir:   rootDir,
		Name:      name,
		CreatedAt: time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC),
	}
	if err := db.InsertWorkspace(testutil.Context(t), workspace); err != nil {
		t.Fatalf("InsertWorkspace() error = %v", err)
	}
	return workspace.ID
}

func sortedEventTypes(events []taskpkg.TaskEvent) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.EventType)
	}
	sort.Strings(types)
	return types
}

func containsEventType(events []taskpkg.TaskEvent, want string) bool {
	for _, event := range events {
		if event.EventType == want {
			return true
		}
	}
	return false
}

func TestTaskManagerGetTaskRequiresReadAuthorityIntegration(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openTaskManagerGlobalDB(t)
	manager := newTaskManagerIntegration(t, db)
	actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task create")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}
	taskRecord, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Read auth check",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	denied := actor
	denied.Authority.Read = false
	_, err = manager.GetTask(ctx, taskRecord.ID, denied)
	if !errors.Is(err, taskpkg.ErrPermissionDenied) {
		t.Fatalf("GetTask(no read) error = %v, want %v", err, taskpkg.ErrPermissionDenied)
	}
}
