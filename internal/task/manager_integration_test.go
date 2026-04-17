//go:build integration

package task_test

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/store/sessiondb"
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

type integrationRuntimeViewReader struct {
	registry     *globaldb.GlobalDB
	sessionStore map[string]*sessiondb.SessionDB
}

func (e *integrationSessionExecutor) StartTaskSession(
	_ context.Context,
	spec *taskpkg.StartTaskSession,
) (*taskpkg.SessionRef, error) {
	if spec == nil {
		return nil, errors.New("task integration session executor requires start spec")
	}
	e.startCalls = append(e.startCalls, *spec)
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

func (r *integrationRuntimeViewReader) GetSession(
	ctx context.Context,
	sessionID string,
) (*taskpkg.RunSessionRef, error) {
	if r == nil || r.registry == nil {
		return nil, taskpkg.ErrTaskRunNotFound
	}

	trimmedSessionID := strings.TrimSpace(sessionID)
	sessions, err := r.registry.ListSessions(ctx, store.SessionListQuery{})
	if err != nil {
		return nil, err
	}
	for _, session := range sessions {
		if session.ID != trimmedSessionID {
			continue
		}
		return &taskpkg.RunSessionRef{
			SessionID:   session.ID,
			WorkspaceID: session.WorkspaceID,
			AgentName:   session.AgentName,
			Name:        session.Name,
			Channel:     session.Channel,
			State:       session.State,
			CreatedAt:   session.CreatedAt,
			UpdatedAt:   session.UpdatedAt,
		}, nil
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (r *integrationRuntimeViewReader) ListSessionEvents(
	ctx context.Context,
	sessionID string,
	query store.EventQuery,
) ([]store.SessionEvent, error) {
	if r == nil {
		return nil, taskpkg.ErrTaskRunNotFound
	}
	sessionDB := r.sessionStore[strings.TrimSpace(sessionID)]
	if sessionDB == nil {
		return nil, taskpkg.ErrTaskRunNotFound
	}
	return sessionDB.Query(ctx, query)
}

func (r *integrationRuntimeViewReader) ListSessionTokenStats(
	ctx context.Context,
	sessionID string,
) ([]store.TokenStats, error) {
	if r == nil || r.registry == nil {
		return nil, taskpkg.ErrTaskRunNotFound
	}
	return r.registry.ListTokenStats(ctx, store.TokenStatsQuery{SessionID: strings.TrimSpace(sessionID)})
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

	events, err := db.ListTaskEvents(ctx, taskpkg.EventQuery{TaskID: stored.ID})
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

func TestTaskManagerRejectsInvalidTaskSemanticsBeforePersistence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec taskpkg.CreateTask
	}{
		{
			name: "invalid priority",
			spec: taskpkg.CreateTask{Scope: taskpkg.ScopeGlobal, Title: "Bad priority", Priority: taskpkg.Priority("rush")},
		},
		{
			name: "invalid max attempts",
			spec: taskpkg.CreateTask{Scope: taskpkg.ScopeGlobal, Title: "Bad attempts", MaxAttempts: intPtr(0)},
		},
		{
			name: "invalid approval policy",
			spec: taskpkg.CreateTask{Scope: taskpkg.ScopeGlobal, Title: "Bad approval", ApprovalPolicy: taskpkg.ApprovalPolicy("auto")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := testutil.Context(t)
			db := openTaskManagerGlobalDB(t)
			manager := newTaskManagerIntegration(t, db)

			actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task create")
			if err != nil {
				t.Fatalf("DeriveHumanActorContext() error = %v", err)
			}

			_, err = manager.CreateTask(ctx, tt.spec, actor)
			if err == nil {
				t.Fatal("CreateTask() error = nil, want non-nil")
			}
			if !errors.Is(err, taskpkg.ErrValidation) {
				t.Fatalf("CreateTask() error = %v, want %v", err, taskpkg.ErrValidation)
			}

			tasks, err := db.ListTasks(ctx, taskpkg.Query{})
			if err != nil {
				t.Fatalf("ListTasks() error = %v", err)
			}
			if got := len(tasks); got != 0 {
				t.Fatalf("len(tasks) = %d, want 0", got)
			}
		})
	}
}

func TestTaskManagerCreateTaskPersistsAutomationLinkedAgentOrigin(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openTaskManagerGlobalDB(t)
	manager := newTaskManagerIntegration(t, db)

	actor, err := taskpkg.DeriveAutomationLinkedAgentSessionActorContext("sess-agent-2", "run:run-2")
	if err != nil {
		t.Fatalf("DeriveAutomationLinkedAgentSessionActorContext() error = %v", err)
	}

	created, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Investigate automation-linked task creation",
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
	if got, want := stored.CreatedBy.Ref, "sess-agent-2"; got != want {
		t.Fatalf("stored.CreatedBy.Ref = %q, want %q", got, want)
	}
	if got, want := stored.Origin.Kind, taskpkg.OriginKindAutomation; got != want {
		t.Fatalf("stored.Origin.Kind = %q, want %q", got, want)
	}
	if got, want := stored.Origin.Ref, "run:run-2"; got != want {
		t.Fatalf("stored.Origin.Ref = %q, want %q", got, want)
	}
}

func TestTaskManagerPublishTaskReconcilesDraftLifecycleIntegration(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openTaskManagerGlobalDB(t)
	executor := &integrationSessionExecutor{}
	manager := newTaskManagerIntegration(t, db, taskpkg.WithSessionExecutor(executor))

	actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task publish")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}

	blocker, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Blocker",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(blocker) error = %v", err)
	}
	target, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Draft target",
		Draft: true,
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(target) error = %v", err)
	}
	if err := manager.AddDependency(ctx, taskpkg.AddDependency{
		TaskID:          target.ID,
		DependsOnTaskID: blocker.ID,
		Kind:            taskpkg.DependencyKindBlocks,
	}, actor); err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}

	if _, err := manager.EnqueueRun(ctx, taskpkg.EnqueueRun{TaskID: target.ID}, actor); !errors.Is(
		err,
		taskpkg.ErrInvalidStatusTransition,
	) {
		t.Fatalf("EnqueueRun(draft) error = %v, want %v", err, taskpkg.ErrInvalidStatusTransition)
	}

	published, err := manager.PublishTask(ctx, target.ID, actor)
	if err != nil {
		t.Fatalf("PublishTask() error = %v", err)
	}
	if got, want := published.Status, taskpkg.TaskStatusBlocked; got != want {
		t.Fatalf("published.Status = %q, want %q", got, want)
	}

	blockerRun, err := manager.EnqueueRun(ctx, taskpkg.EnqueueRun{TaskID: blocker.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(blocker) error = %v", err)
	}
	blockerRun, err = manager.ClaimRun(ctx, blockerRun.ID, taskpkg.ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(blocker) error = %v", err)
	}
	blockerRun, err = manager.StartRun(ctx, blockerRun.ID, taskpkg.StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun(blocker) error = %v", err)
	}
	if _, err := manager.CompleteRun(ctx, blockerRun.ID, taskpkg.RunResult{
		Value: json.RawMessage(`{"ok":true}`),
	}, actor); err != nil {
		t.Fatalf("CompleteRun(blocker) error = %v", err)
	}

	reloadedTarget, err := db.GetTask(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetTask(target) error = %v", err)
	}
	if got, want := reloadedTarget.Status, taskpkg.TaskStatusReady; got != want {
		t.Fatalf("reloadedTarget.Status = %q, want %q", got, want)
	}

	events, err := db.ListTaskEvents(ctx, taskpkg.EventQuery{TaskID: target.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents(target) error = %v", err)
	}
	if !containsEventType(events, "task.published") {
		t.Fatalf("event types = %#v, want task.published", sortedEventTypes(events))
	}
}

func TestTaskManagerPublishTaskReadModelsStayConsistentAfterReload(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	dbPath := filepath.Join(t.TempDir(), "agh.db")

	first, err := globaldb.OpenGlobalDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(first) error = %v", err)
	}

	executor := &integrationSessionExecutor{}
	firstManager := newTaskManagerIntegration(t, first, taskpkg.WithSessionExecutor(executor))
	actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task publish")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}

	blocker, err := firstManager.CreateTask(ctx, taskpkg.CreateTask{
		Scope:      taskpkg.ScopeGlobal,
		Title:      "Release blocker",
		Identifier: "OPS-100",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(blocker) error = %v", err)
	}
	target, err := firstManager.CreateTask(ctx, taskpkg.CreateTask{
		Scope:      taskpkg.ScopeGlobal,
		Title:      "Draft target",
		Identifier: "OPS-300",
		Draft:      true,
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(target) error = %v", err)
	}
	if err := firstManager.AddDependency(ctx, taskpkg.AddDependency{
		TaskID:          target.ID,
		DependsOnTaskID: blocker.ID,
		Kind:            taskpkg.DependencyKindBlocks,
	}, actor); err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}
	published, err := firstManager.PublishTask(ctx, target.ID, actor)
	if err != nil {
		t.Fatalf("PublishTask() error = %v", err)
	}
	if got, want := published.Status, taskpkg.TaskStatusBlocked; got != want {
		t.Fatalf("published.Status = %q, want %q", got, want)
	}

	blockerRun, err := firstManager.EnqueueRun(ctx, taskpkg.EnqueueRun{TaskID: blocker.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(blocker) error = %v", err)
	}
	blockerRun, err = firstManager.ClaimRun(ctx, blockerRun.ID, taskpkg.ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(blocker) error = %v", err)
	}
	blockerRun, err = firstManager.StartRun(ctx, blockerRun.ID, taskpkg.StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun(blocker) error = %v", err)
	}
	if _, err := firstManager.CompleteRun(ctx, blockerRun.ID, taskpkg.RunResult{
		Value: json.RawMessage(`{"ok":true}`),
	}, actor); err != nil {
		t.Fatalf("CompleteRun(blocker) error = %v", err)
	}

	if err := first.Close(ctx); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	second, err := globaldb.OpenGlobalDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(second) error = %v", err)
	}
	t.Cleanup(func() {
		if err := second.Close(ctx); err != nil {
			t.Fatalf("Close(second) error = %v", err)
		}
	})

	secondManager := newTaskManagerIntegration(t, second)

	view, err := secondManager.GetTask(ctx, target.ID, actor)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got, want := view.Task.Status, taskpkg.TaskStatusReady; got != want {
		t.Fatalf("view.Task.Status = %q, want %q", got, want)
	}
	if got, want := view.Summary.Status, taskpkg.TaskStatusReady; got != want {
		t.Fatalf("view.Summary.Status = %q, want %q", got, want)
	}
	if got, want := view.Summary.DependencyCount, 1; got != want {
		t.Fatalf("view.Summary.DependencyCount = %d, want %d", got, want)
	}
	if len(view.DependencyReferences) != 1 {
		t.Fatalf("len(view.DependencyReferences) = %d, want 1", len(view.DependencyReferences))
	}
	if got, want := view.DependencyReferences[0].DependsOn.Identifier, blocker.Identifier; got != want {
		t.Fatalf("view.DependencyReferences[0].DependsOn.Identifier = %q, want %q", got, want)
	}

	summaries, err := secondManager.ListTasks(ctx, taskpkg.Query{
		Status: taskpkg.TaskStatusReady,
		Search: "ops-300",
	}, actor)
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(summaries) != 1 || summaries[0].ID != target.ID {
		t.Fatalf("ListTasks() = %#v, want only %q", summaries, target.ID)
	}
	if got, want := summaries[0].Status, taskpkg.TaskStatusReady; got != want {
		t.Fatalf("summaries[0].Status = %q, want %q", got, want)
	}
	if got, want := summaries[0].Dependencies[0].DependsOn.Title, blocker.Title; got != want {
		t.Fatalf("summaries[0].Dependencies[0].DependsOn.Title = %q, want %q", got, want)
	}
}

func TestTaskManagerTriageMutationsRemainActorScopedAfterReload(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	dbPath := filepath.Join(t.TempDir(), "agh.db")

	first, err := globaldb.OpenGlobalDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(first) error = %v", err)
	}

	firstManager := newTaskManagerIntegration(t, first)
	alice, err := taskpkg.DeriveHumanActorContext("alice", taskpkg.OriginKindCLI, "agh task inbox")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext(alice) error = %v", err)
	}
	bob, err := taskpkg.DeriveHumanActorContext("bob", taskpkg.OriginKindCLI, "agh task inbox")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext(bob) error = %v", err)
	}

	taskRecord, err := firstManager.CreateTask(ctx, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Persist triage state",
	}, alice)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	if _, err := firstManager.MarkTaskRead(ctx, taskRecord.ID, alice); err != nil {
		t.Fatalf("MarkTaskRead(alice) error = %v", err)
	}
	archivedState, err := firstManager.ArchiveTask(ctx, taskRecord.ID, alice)
	if err != nil {
		t.Fatalf("ArchiveTask(alice) error = %v", err)
	}
	dismissedState, err := firstManager.DismissTask(ctx, taskRecord.ID, bob)
	if err != nil {
		t.Fatalf("DismissTask(bob) error = %v", err)
	}
	if !archivedState.Archived || !dismissedState.Dismissed {
		t.Fatalf("triage states = %#v / %#v, want archived and dismissed states", archivedState, dismissedState)
	}

	if err := first.Close(ctx); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	second, err := globaldb.OpenGlobalDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(second) error = %v", err)
	}
	t.Cleanup(func() {
		if err := second.Close(ctx); err != nil {
			t.Fatalf("Close(second) error = %v", err)
		}
	})

	storedAlice, err := second.GetTaskTriageState(ctx, taskRecord.ID, alice.Actor)
	if err != nil {
		t.Fatalf("GetTaskTriageState(alice) error = %v", err)
	}
	if storedAlice != archivedState {
		t.Fatalf("storedAlice = %#v, want %#v", storedAlice, archivedState)
	}
	storedBob, err := second.GetTaskTriageState(ctx, taskRecord.ID, bob.Actor)
	if err != nil {
		t.Fatalf("GetTaskTriageState(bob) error = %v", err)
	}
	if storedBob != dismissedState {
		t.Fatalf("storedBob = %#v, want %#v", storedBob, dismissedState)
	}
}

func TestTaskManagerApprovalGateAndAttemptExhaustionIntegration(t *testing.T) {
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
		Scope:          taskpkg.ScopeGlobal,
		Title:          "Approval-gated task",
		ApprovalPolicy: taskpkg.ApprovalPolicyManual,
		MaxAttempts:    intPtr(1),
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if got, want := taskRecord.Status, taskpkg.TaskStatusBlocked; got != want {
		t.Fatalf("taskRecord.Status = %q, want %q", got, want)
	}

	run, err := manager.EnqueueRun(ctx, taskpkg.EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	if _, err := manager.ClaimRun(ctx, run.ID, taskpkg.ClaimRun{}, actor); !errors.Is(
		err,
		taskpkg.ErrInvalidStatusTransition,
	) {
		t.Fatalf("ClaimRun(blocked) error = %v, want %v", err, taskpkg.ErrInvalidStatusTransition)
	}

	approved, err := manager.ApproveTask(ctx, taskRecord.ID, actor)
	if err != nil {
		t.Fatalf("ApproveTask() error = %v", err)
	}
	if got, want := approved.ApprovalState, taskpkg.ApprovalStateApproved; got != want {
		t.Fatalf("approved.ApprovalState = %q, want %q", got, want)
	}

	run, err = manager.ClaimRun(ctx, run.ID, taskpkg.ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(approved) error = %v", err)
	}
	run, err = manager.StartRun(ctx, run.ID, taskpkg.StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun() error = %v", err)
	}
	if _, err := manager.FailRun(ctx, run.ID, taskpkg.RunFailure{
		Error: "approval path failed",
	}, actor); err != nil {
		t.Fatalf("FailRun() error = %v", err)
	}

	reloaded, err := db.GetTask(ctx, taskRecord.ID)
	if err != nil {
		t.Fatalf("GetTask(reloaded) error = %v", err)
	}
	if got, want := reloaded.Status, taskpkg.TaskStatusFailed; got != want {
		t.Fatalf("reloaded.Status = %q, want %q", got, want)
	}
	if got, want := reloaded.ApprovalState, taskpkg.ApprovalStateApproved; got != want {
		t.Fatalf("reloaded.ApprovalState = %q, want %q", got, want)
	}

	if _, err := manager.EnqueueRun(ctx, taskpkg.EnqueueRun{TaskID: taskRecord.ID}, actor); !errors.Is(
		err,
		taskpkg.ErrInvalidStatusTransition,
	) {
		t.Fatalf("EnqueueRun(exhausted) error = %v, want %v", err, taskpkg.ErrInvalidStatusTransition)
	}

	events, err := db.ListTaskEvents(ctx, taskpkg.EventQuery{TaskID: taskRecord.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if !containsEventType(events, "task.approved") {
		t.Fatalf("event types = %#v, want task.approved", sortedEventTypes(events))
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

	childEvents, err := db.ListTaskEvents(ctx, taskpkg.EventQuery{TaskID: child.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents(child) error = %v", err)
	}
	if !testutil.EqualStringSlices(sortedEventTypes(childEvents), []string{"task.created", "task.dependency_added"}) {
		t.Fatalf("child event types = %#v, want task.created + task.dependency_added", sortedEventTypes(childEvents))
	}

	parentEvents, err := db.ListTaskEvents(ctx, taskpkg.EventQuery{TaskID: parent.ID})
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
	if got, want := view.Summary.DependencyCount, 1; got != want {
		t.Fatalf("view.Summary.DependencyCount = %d, want %d", got, want)
	}
	if got, want := view.Summary.ChildCount, 0; got != want {
		t.Fatalf("view.Summary.ChildCount = %d, want %d", got, want)
	}
	if len(view.DependencyReferences) != 1 {
		t.Fatalf("len(view.DependencyReferences) = %d, want 1", len(view.DependencyReferences))
	}
	if got, want := view.DependencyReferences[0].DependsOn.Title, blocker.Title; got != want {
		t.Fatalf("view.DependencyReferences[0].DependsOn.Title = %q, want %q", got, want)
	}
	if view.Summary.LastActivityAt.IsZero() {
		t.Fatal("view.Summary.LastActivityAt is zero, want latest activity timestamp")
	}
}

func TestTaskManagerListTasksReturnsEnrichedSummariesIntegration(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openTaskManagerGlobalDB(t)
	manager := newTaskManagerIntegration(t, db)
	actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task list")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}

	first, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope:      taskpkg.ScopeGlobal,
		Title:      "Alpha planning",
		Identifier: "OPS-100",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(first) error = %v", err)
	}
	second, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope:      taskpkg.ScopeGlobal,
		Title:      "Beta rollout",
		Identifier: "OPS-200",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(second) error = %v", err)
	}
	if err := manager.AddDependency(ctx, taskpkg.AddDependency{
		TaskID:          second.ID,
		DependsOnTaskID: first.ID,
		Kind:            taskpkg.DependencyKindBlocks,
	}, actor); err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}
	if err := db.CreateTaskRun(ctx, taskpkg.Run{
		ID:        "run-beta",
		TaskID:    second.ID,
		Status:    taskpkg.TaskRunStatusRunning,
		Attempt:   1,
		Origin:    taskpkg.Origin{Kind: taskpkg.OriginKindDaemon, Ref: "scheduler"},
		QueuedAt:  time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC),
		StartedAt: time.Date(2026, 4, 17, 12, 5, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("CreateTaskRun() error = %v", err)
	}

	byTitle, err := manager.ListTasks(ctx, taskpkg.Query{Search: "alpha"}, actor)
	if err != nil {
		t.Fatalf("ListTasks(search title) error = %v", err)
	}
	if len(byTitle) != 1 || byTitle[0].ID != first.ID {
		t.Fatalf("ListTasks(search title) = %#v, want only %q", byTitle, first.ID)
	}

	byIdentifier, err := manager.ListTasks(ctx, taskpkg.Query{Search: "ops-200"}, actor)
	if err != nil {
		t.Fatalf("ListTasks(search identifier) error = %v", err)
	}
	if len(byIdentifier) != 1 || byIdentifier[0].ID != second.ID {
		t.Fatalf("ListTasks(search identifier) = %#v, want only %q", byIdentifier, second.ID)
	}
	if got, want := byIdentifier[0].DependencyCount, 1; got != want {
		t.Fatalf("byIdentifier[0].DependencyCount = %d, want %d", got, want)
	}
	if byIdentifier[0].ActiveRun == nil || byIdentifier[0].ActiveRun.ID != "run-beta" {
		t.Fatalf("byIdentifier[0].ActiveRun = %#v, want run-beta", byIdentifier[0].ActiveRun)
	}
	if len(byIdentifier[0].Dependencies) != 1 {
		t.Fatalf("len(byIdentifier[0].Dependencies) = %d, want 1", len(byIdentifier[0].Dependencies))
	}
	if got, want := byIdentifier[0].Dependencies[0].DependsOn.Identifier, first.Identifier; got != want {
		t.Fatalf("byIdentifier[0].Dependencies[0].DependsOn.Identifier = %q, want %q", got, want)
	}

	all, err := manager.ListTasks(ctx, taskpkg.Query{}, actor)
	if err != nil {
		t.Fatalf("ListTasks(all) error = %v", err)
	}
	if got, want := []string{all[0].ID, all[1].ID}, []string{second.ID, first.ID}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("ListTasks(all) order = %#v, want %#v", got, want)
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

	events, err := db.ListTaskEvents(ctx, taskpkg.EventQuery{TaskID: taskRecord.ID})
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
	if got, want := cancelledParent.Status, taskpkg.TaskStatusCanceled; got != want {
		t.Fatalf("cancelled parent status = %q, want %q", got, want)
	}

	for _, taskID := range []string{parent.ID, queuedChild.ID, activeChild.ID} {
		record, err := db.GetTask(ctx, taskID)
		if err != nil {
			t.Fatalf("GetTask(%q) error = %v", taskID, err)
		}
		if got, want := record.Status, taskpkg.TaskStatusCanceled; got != want {
			t.Fatalf("task %q status = %q, want %q", taskID, got, want)
		}
	}

	storedQueuedRun, err := db.GetTaskRun(ctx, queuedRun.ID)
	if err != nil {
		t.Fatalf("GetTaskRun(queued) error = %v", err)
	}
	if got, want := storedQueuedRun.Status, taskpkg.TaskRunStatusCanceled; got != want {
		t.Fatalf("queued child run status = %q, want %q", got, want)
	}
	storedActiveRun, err := db.GetTaskRun(ctx, activeRun.ID)
	if err != nil {
		t.Fatalf("GetTaskRun(active) error = %v", err)
	}
	if got, want := storedActiveRun.Status, taskpkg.TaskRunStatusCanceled; got != want {
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

	parentEvents, err := db.ListTaskEvents(ctx, taskpkg.EventQuery{TaskID: parent.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents(parent) error = %v", err)
	}
	if !containsEventType(parentEvents, "task.canceled") {
		t.Fatalf("parent event types = %#v, want task.canceled", sortedEventTypes(parentEvents))
	}

	activeChildEvents, err := db.ListTaskEvents(ctx, taskpkg.EventQuery{TaskID: activeChild.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents(active child) error = %v", err)
	}
	if !containsEventType(activeChildEvents, "task.run_canceled") {
		t.Fatalf("active child event types = %#v, want task.run_canceled", sortedEventTypes(activeChildEvents))
	}
	if !containsEventType(activeChildEvents, "task.run_force_stopped") {
		t.Fatalf("active child event types = %#v, want task.run_force_stopped", sortedEventTypes(activeChildEvents))
	}
}

func TestTaskManagerTimelineLiveReadsIntegration(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openTaskManagerGlobalDB(t)
	executor := &integrationSessionExecutor{}
	fixedNow := time.Date(2026, 4, 17, 14, 0, 0, 0, time.UTC)
	counter := 0
	manager := newTaskManagerIntegration(
		t,
		db,
		taskpkg.WithSessionExecutor(executor),
		taskpkg.WithManagerNow(func() time.Time { return fixedNow }),
		taskpkg.WithIDGenerator(func(prefix string) string {
			counter++
			return prefix + "-timeline-" + strconv.Itoa(counter)
		}),
	)
	actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task timeline")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}

	taskRecord, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Timeline detail task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	run, err := manager.EnqueueRun(ctx, taskpkg.EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	run, err = manager.ClaimRun(ctx, run.ID, taskpkg.ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun() error = %v", err)
	}
	run, err = manager.StartRun(ctx, run.ID, taskpkg.StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun() error = %v", err)
	}
	if _, err := manager.CompleteRun(ctx, run.ID, taskpkg.RunResult{
		Value: json.RawMessage(`{"ok":true}`),
	}, actor); err != nil {
		t.Fatalf("CompleteRun() error = %v", err)
	}

	pageOne, err := manager.Timeline(ctx, taskRecord.ID, taskpkg.TimelineQuery{Limit: 3}, actor)
	if err != nil {
		t.Fatalf("Timeline(page one) error = %v", err)
	}
	if got, want := len(pageOne), 3; got != want {
		t.Fatalf("len(pageOne) = %d, want %d", got, want)
	}
	if got, want := []string{
		pageOne[0].EventType,
		pageOne[1].EventType,
		pageOne[2].EventType,
	}, []string{"task.created", "task.run_enqueued", "task.run_claimed"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("pageOne event types = %#v, want %#v", got, want)
	}
	for idx, item := range pageOne {
		if got, want := item.Sequence, int64(idx+1); got != want {
			t.Fatalf("pageOne[%d].Sequence = %d, want %d", idx, got, want)
		}
		if idx == 0 {
			if item.Run != nil {
				t.Fatalf("pageOne[0].Run = %#v, want nil", item.Run)
			}
			continue
		}
		if item.Run == nil || item.Run.ID != run.ID {
			t.Fatalf("pageOne[%d].Run = %#v, want run %q", idx, item.Run, run.ID)
		}
	}

	pageTwo, err := manager.Timeline(ctx, taskRecord.ID, taskpkg.TimelineQuery{
		AfterSequence: pageOne[len(pageOne)-1].Sequence,
		Limit:         3,
	}, actor)
	if err != nil {
		t.Fatalf("Timeline(page two) error = %v", err)
	}
	if got, want := len(pageTwo), 3; got != want {
		t.Fatalf("len(pageTwo) = %d, want %d", got, want)
	}
	if got, want := []string{
		pageTwo[0].EventType,
		pageTwo[1].EventType,
		pageTwo[2].EventType,
	}, []string{"task.run_starting", "task.run_started", "task.run_completed"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("pageTwo event types = %#v, want %#v", got, want)
	}
	for idx, item := range pageTwo {
		if got, want := item.Sequence, int64(idx+4); got != want {
			t.Fatalf("pageTwo[%d].Sequence = %d, want %d", idx, got, want)
		}
		if item.Run == nil || item.Run.ID != run.ID {
			t.Fatalf("pageTwo[%d].Run = %#v, want run %q", idx, item.Run, run.ID)
		}
	}
}

func TestTaskManagerRunDetailUsesPersistedRuntimeDataIntegration(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openTaskManagerGlobalDB(t)
	workspaceID := registerTaskManagerWorkspace(t, db, "runtime-detail", filepath.Join(t.TempDir(), "workspace"))
	executor := &integrationSessionExecutor{}
	runtimeReader := &integrationRuntimeViewReader{registry: db, sessionStore: make(map[string]*sessiondb.SessionDB)}
	fixedNow := time.Date(2026, 4, 17, 15, 0, 0, 0, time.UTC)
	counter := 0
	manager := newTaskManagerIntegration(
		t,
		db,
		taskpkg.WithSessionExecutor(executor),
		taskpkg.WithRuntimeViewReader(runtimeReader),
		taskpkg.WithManagerNow(func() time.Time { return fixedNow }),
		taskpkg.WithIDGenerator(func(prefix string) string {
			counter++
			return prefix + "-detail-" + strconv.Itoa(counter)
		}),
	)
	actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task run-detail")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}

	taskRecord, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope:       taskpkg.ScopeWorkspace,
		WorkspaceID: workspaceID,
		Title:       "Run detail task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	run, err := manager.EnqueueRun(ctx, taskpkg.EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	run, err = manager.ClaimRun(ctx, run.ID, taskpkg.ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun() error = %v", err)
	}
	run, err = manager.StartRun(ctx, run.ID, taskpkg.StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun() error = %v", err)
	}

	if err := db.RegisterSession(ctx, store.SessionInfo{
		ID:          run.SessionID,
		Name:        "Task detail session",
		AgentName:   "codex",
		WorkspaceID: workspaceID,
		Channel:     "tasks",
		SessionType: "task",
		State:       "running",
		CreatedAt:   fixedNow,
		UpdatedAt:   fixedNow.Add(5 * time.Minute),
	}); err != nil {
		t.Fatalf("RegisterSession() error = %v", err)
	}

	sessionDir := filepath.Join(t.TempDir(), "sessions", run.SessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", sessionDir, err)
	}
	sessionDB, err := sessiondb.OpenSessionDB(ctx, run.SessionID, filepath.Join(sessionDir, store.SessionDatabaseName))
	if err != nil {
		t.Fatalf("OpenSessionDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := sessionDB.Close(ctx); err != nil {
			t.Fatalf("SessionDB.Close() error = %v", err)
		}
	})
	runtimeReader.sessionStore[run.SessionID] = sessionDB

	for _, event := range []store.SessionEvent{
		{
			ID:        "event-1",
			TurnID:    "turn-1",
			Type:      "agent_message",
			AgentName: "codex",
			Content:   `{"text":"planning"}`,
			Timestamp: fixedNow.Add(time.Minute),
		},
		{
			ID:        "event-2",
			TurnID:    "turn-1",
			Type:      "tool_call",
			AgentName: "codex",
			Content:   `{"tool_call_id":"call-1"}`,
			Timestamp: fixedNow.Add(2 * time.Minute),
		},
		{
			ID:        "event-3",
			TurnID:    "turn-1",
			Type:      "tool_result",
			AgentName: "codex",
			Content:   `{"tool_call_id":"call-1"}`,
			Timestamp: fixedNow.Add(3 * time.Minute),
		},
		{
			ID:        "event-4",
			TurnID:    "turn-2",
			Type:      "tool_call",
			AgentName: "codex",
			Content:   `{"toolCallId":"call-2"}`,
			Timestamp: fixedNow.Add(4 * time.Minute),
		},
	} {
		if err := sessionDB.Record(ctx, event); err != nil {
			t.Fatalf("SessionDB.Record(%q) error = %v", event.ID, err)
		}
	}

	for _, update := range []store.TokenStatsUpdate{
		{
			SessionID:    run.SessionID,
			AgentName:    "codex",
			InputTokens:  int64Ptr(10),
			OutputTokens: int64Ptr(6),
			TotalTokens:  int64Ptr(16),
			CostAmount:   float64Ptr(0.2),
			CostCurrency: stringPtr("USD"),
			Turns:        1,
			UpdatedAt:    fixedNow.Add(5 * time.Minute),
		},
		{
			SessionID:   run.SessionID,
			AgentName:   "reviewer",
			InputTokens: int64Ptr(4),
			TotalTokens: int64Ptr(4),
			CostAmount:  float64Ptr(0.1),
			Turns:       2,
			UpdatedAt:   fixedNow.Add(6 * time.Minute),
		},
	} {
		if err := db.UpdateTokenStats(ctx, update); err != nil {
			t.Fatalf("UpdateTokenStats(%q) error = %v", update.AgentName, err)
		}
	}

	detail, err := manager.RunDetail(ctx, run.ID, actor)
	if err != nil {
		t.Fatalf("RunDetail() error = %v", err)
	}
	if got, want := detail.Task.ID, taskRecord.ID; got != want {
		t.Fatalf("detail.Task.ID = %q, want %q", got, want)
	}
	if got, want := detail.Task.Status, taskpkg.TaskStatusInProgress; got != want {
		t.Fatalf("detail.Task.Status = %q, want %q", got, want)
	}
	if detail.Session == nil {
		t.Fatal("detail.Session = nil, want session reference")
	}
	if got, want := detail.Session.AgentName, "codex"; got != want {
		t.Fatalf("detail.Session.AgentName = %q, want %q", got, want)
	}
	if got, want := detail.Session.Channel, "tasks"; got != want {
		t.Fatalf("detail.Session.Channel = %q, want %q", got, want)
	}
	if detail.Summary.ToolCallCount == nil || *detail.Summary.ToolCallCount != 2 {
		t.Fatalf("detail.Summary.ToolCallCount = %#v, want 2", detail.Summary.ToolCallCount)
	}
	if detail.Summary.InputTokens == nil || *detail.Summary.InputTokens != 14 {
		t.Fatalf("detail.Summary.InputTokens = %#v, want 14", detail.Summary.InputTokens)
	}
	if detail.Summary.OutputTokens == nil || *detail.Summary.OutputTokens != 6 {
		t.Fatalf("detail.Summary.OutputTokens = %#v, want 6", detail.Summary.OutputTokens)
	}
	if detail.Summary.TotalTokens == nil || *detail.Summary.TotalTokens != 20 {
		t.Fatalf("detail.Summary.TotalTokens = %#v, want 20", detail.Summary.TotalTokens)
	}
	if detail.Summary.TurnCount == nil || *detail.Summary.TurnCount != 3 {
		t.Fatalf("detail.Summary.TurnCount = %#v, want 3", detail.Summary.TurnCount)
	}
	if detail.Summary.TotalCost == nil || math.Abs(*detail.Summary.TotalCost-0.3) > 1e-9 {
		t.Fatalf("detail.Summary.TotalCost = %#v, want 0.3", detail.Summary.TotalCost)
	}
	if detail.Summary.CostCurrency == nil || *detail.Summary.CostCurrency != "USD" {
		t.Fatalf("detail.Summary.CostCurrency = %#v, want USD", detail.Summary.CostCurrency)
	}
	if got, want := detail.Summary.LastEventType, "tool_call"; got != want {
		t.Fatalf("detail.Summary.LastEventType = %q, want %q", got, want)
	}
	if got, want := detail.Summary.LastActivityAt, fixedNow.Add(6*time.Minute); !got.Equal(want) {
		t.Fatalf("detail.Summary.LastActivityAt = %s, want %s", got, want)
	}
}

func TestTaskManagerTreeLiveViewIntegration(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openTaskManagerGlobalDB(t)
	executor := &integrationSessionExecutor{}
	clock := incrementingClock(time.Date(2026, 4, 17, 16, 0, 0, 0, time.UTC), time.Minute)
	counter := 0
	manager := newTaskManagerIntegration(
		t,
		db,
		taskpkg.WithSessionExecutor(executor),
		taskpkg.WithManagerNow(clock),
		taskpkg.WithIDGenerator(func(prefix string) string {
			counter++
			return prefix + "-tree-" + strconv.Itoa(counter)
		}),
	)
	actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task live-tree")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}

	root, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Root live task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(root) error = %v", err)
	}
	childActive, err := manager.CreateChildTask(ctx, root.ID, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Active child",
	}, actor)
	if err != nil {
		t.Fatalf("CreateChildTask(active) error = %v", err)
	}
	childIdle, err := manager.CreateChildTask(ctx, root.ID, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Idle child",
	}, actor)
	if err != nil {
		t.Fatalf("CreateChildTask(idle) error = %v", err)
	}
	grandchild, err := manager.CreateChildTask(ctx, childActive.ID, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Grandchild",
	}, actor)
	if err != nil {
		t.Fatalf("CreateChildTask(grandchild) error = %v", err)
	}

	run, err := manager.EnqueueRun(ctx, taskpkg.EnqueueRun{TaskID: childActive.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	run, err = manager.ClaimRun(ctx, run.ID, taskpkg.ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun() error = %v", err)
	}
	run, err = manager.StartRun(ctx, run.ID, taskpkg.StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun() error = %v", err)
	}

	tree, err := manager.Tree(ctx, root.ID, actor)
	if err != nil {
		t.Fatalf("Tree() error = %v", err)
	}
	if got, want := tree.Root.Task.ID, root.ID; got != want {
		t.Fatalf("tree.Root.Task.ID = %q, want %q", got, want)
	}
	if got, want := len(tree.Descendants), 3; got != want {
		t.Fatalf("len(tree.Descendants) = %d, want %d", got, want)
	}
	if got, want := []string{
		tree.Descendants[0].Task.ID,
		tree.Descendants[1].Task.ID,
		tree.Descendants[2].Task.ID,
	}, []string{childActive.ID, childIdle.ID, grandchild.ID}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("tree.Descendants order = %#v, want %#v", got, want)
	}
	if tree.Descendants[0].ActiveRun == nil || tree.Descendants[0].ActiveRun.ID != run.ID {
		t.Fatalf("tree.Descendants[0].ActiveRun = %#v, want run %q", tree.Descendants[0].ActiveRun, run.ID)
	}
	if got, want := tree.Descendants[0].Depth, 1; got != want {
		t.Fatalf("tree.Descendants[0].Depth = %d, want %d", got, want)
	}
	if got, want := tree.Descendants[0].ChildCount, 1; got != want {
		t.Fatalf("tree.Descendants[0].ChildCount = %d, want %d", got, want)
	}
	if got, want := tree.Descendants[2].Task.ID, grandchild.ID; got != want {
		t.Fatalf("tree.Descendants[2].Task.ID = %q, want %q", got, want)
	}
	if got, want := tree.Descendants[2].ParentTaskID, childActive.ID; got != want {
		t.Fatalf("tree.Descendants[2].ParentTaskID = %q, want %q", got, want)
	}
	if got, want := tree.Descendants[2].Depth, 2; got != want {
		t.Fatalf("tree.Descendants[2].Depth = %d, want %d", got, want)
	}
}

func TestTaskManagerStreamSupportsReplayAndReconnectIntegration(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openTaskManagerGlobalDB(t)
	executor := &integrationSessionExecutor{}
	clock := incrementingClock(time.Date(2026, 4, 17, 17, 0, 0, 0, time.UTC), time.Minute)
	counter := 0
	manager := newTaskManagerIntegration(
		t,
		db,
		taskpkg.WithSessionExecutor(executor),
		taskpkg.WithManagerNow(clock),
		taskpkg.WithIDGenerator(func(prefix string) string {
			counter++
			return prefix + "-stream-" + strconv.Itoa(counter)
		}),
	)
	actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task live-stream")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}

	root, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Stream root",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(root) error = %v", err)
	}
	child, err := manager.CreateChildTask(ctx, root.ID, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Stream child",
	}, actor)
	if err != nil {
		t.Fatalf("CreateChildTask(child) error = %v", err)
	}

	streamCtx, cancel := context.WithCancel(ctx)
	stream, err := manager.Stream(streamCtx, root.ID, taskpkg.StreamQuery{AfterSequence: 1}, actor)
	if err != nil {
		t.Fatalf("Stream(first) error = %v", err)
	}

	backlogChildCreated := awaitIntegrationTaskStreamEvent(t, stream)
	backlogParentJoin := awaitIntegrationTaskStreamEvent(t, stream)
	if got, want := []int64{backlogChildCreated.Sequence, backlogParentJoin.Sequence}, []int64{2, 3}; !equalInt64s(got, want) {
		t.Fatalf("backlog sequences = %#v, want [2 3]", got)
	}
	if got, want := backlogChildCreated.Timeline.Task.ID, child.ID; got != want {
		t.Fatalf("backlogChildCreated.Timeline.Task.ID = %q, want %q", got, want)
	}
	if got, want := backlogChildCreated.Type, "task.created"; got != want {
		t.Fatalf("backlogChildCreated.Type = %q, want %q", got, want)
	}
	if got, want := backlogParentJoin.Timeline.Task.ID, root.ID; got != want {
		t.Fatalf("backlogParentJoin.Timeline.Task.ID = %q, want %q", got, want)
	}
	if got, want := backlogParentJoin.Type, "task.child_created"; got != want {
		t.Fatalf("backlogParentJoin.Type = %q, want %q", got, want)
	}

	run, err := manager.EnqueueRun(ctx, taskpkg.EnqueueRun{TaskID: child.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	liveEnqueued := awaitIntegrationTaskStreamEvent(t, stream)
	if got, want := liveEnqueued.Sequence, int64(4); got != want {
		t.Fatalf("liveEnqueued.Sequence = %d, want %d", got, want)
	}
	if got, want := liveEnqueued.Timeline.Task.ID, child.ID; got != want {
		t.Fatalf("liveEnqueued.Timeline.Task.ID = %q, want %q", got, want)
	}
	if got, want := liveEnqueued.Type, "task.run_enqueued"; got != want {
		t.Fatalf("liveEnqueued.Type = %q, want %q", got, want)
	}

	run, err = manager.ClaimRun(ctx, run.ID, taskpkg.ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun() error = %v", err)
	}
	liveClaimed := awaitIntegrationTaskStreamEvent(t, stream)
	if got, want := liveClaimed.Type, "task.run_claimed"; got != want {
		t.Fatalf("liveClaimed.Type = %q, want %q", got, want)
	}

	run, err = manager.StartRun(ctx, run.ID, taskpkg.StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun() error = %v", err)
	}
	liveStarting := awaitIntegrationTaskStreamEvent(t, stream)
	liveStarted := awaitIntegrationTaskStreamEvent(t, stream)
	if got, want := []string{liveStarting.Type, liveStarted.Type}, []string{"task.run_starting", "task.run_started"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("live start event types = %#v, want %#v", got, want)
	}
	lastSequence := liveStarted.Sequence
	cancel()

	reconnectCtx, reconnectCancel := context.WithCancel(ctx)
	defer reconnectCancel()
	reconnected, err := manager.Stream(reconnectCtx, root.ID, taskpkg.StreamQuery{AfterSequence: lastSequence}, actor)
	if err != nil {
		t.Fatalf("Stream(reconnected) error = %v", err)
	}
	assertNoIntegrationTaskStreamEvent(t, reconnected, 150*time.Millisecond)

	if _, err := manager.CompleteRun(ctx, run.ID, taskpkg.RunResult{
		Value: json.RawMessage(`{"ok":true}`),
	}, actor); err != nil {
		t.Fatalf("CompleteRun() error = %v", err)
	}
	liveCompleted := awaitIntegrationTaskStreamEvent(t, reconnected)
	if liveCompleted.Sequence <= lastSequence {
		t.Fatalf("liveCompleted.Sequence = %d, want > %d", liveCompleted.Sequence, lastSequence)
	}
	if got, want := liveCompleted.Timeline.Task.ID, child.ID; got != want {
		t.Fatalf("liveCompleted.Timeline.Task.ID = %q, want %q", got, want)
	}
	if got, want := liveCompleted.Type, "task.run_completed"; got != want {
		t.Fatalf("liveCompleted.Type = %q, want %q", got, want)
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

func newTaskManagerIntegration(t *testing.T, store taskpkg.Store, extraOpts ...taskpkg.Option) *taskpkg.Service {
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

func intPtr(value int) *int {
	return &value
}

func sortedEventTypes(events []taskpkg.Event) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.EventType)
	}
	sort.Strings(types)
	return types
}

func containsEventType(events []taskpkg.Event, want string) bool {
	for _, event := range events {
		if event.EventType == want {
			return true
		}
	}
	return false
}

func int64Ptr(value int64) *int64 {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func incrementingClock(start time.Time, step time.Duration) func() time.Time {
	current := start.Add(-step)
	return func() time.Time {
		current = current.Add(step)
		return current
	}
}

func awaitIntegrationTaskStreamEvent(
	t *testing.T,
	stream <-chan taskpkg.StreamEvent,
) taskpkg.StreamEvent {
	t.Helper()

	select {
	case event, ok := <-stream:
		if !ok {
			t.Fatal("task stream closed before event was available")
		}
		return event
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for task stream event")
		return taskpkg.StreamEvent{}
	}
}

func assertNoIntegrationTaskStreamEvent(
	t *testing.T,
	stream <-chan taskpkg.StreamEvent,
	wait time.Duration,
) {
	t.Helper()

	select {
	case event, ok := <-stream:
		if !ok {
			return
		}
		t.Fatalf("unexpected task stream event = %#v", event)
	case <-time.After(wait):
	}
}

func equalInt64s(left []int64, right []int64) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
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
