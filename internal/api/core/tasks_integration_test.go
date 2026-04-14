//go:build integration

package core_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/testutil"
	taskpkg "github.com/pedronauck/agh/internal/task"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestTaskHandlersCreateTaskAndListFiltersReachManagerIntegration(t *testing.T) {
	t.Parallel()

	var capturedCreate taskpkg.CreateTask
	var capturedCreateActor taskpkg.ActorContext
	var capturedList taskpkg.TaskQuery
	var capturedListActor taskpkg.ActorContext

	tasks := testutil.StubTaskManager{
		CreateTaskFn: func(_ context.Context, spec taskpkg.CreateTask, actor taskpkg.ActorContext) (*taskpkg.Task, error) {
			capturedCreate = spec
			capturedCreateActor = actor
			return &taskpkg.Task{
				ID:             "task-1",
				Identifier:     spec.Identifier,
				Scope:          spec.Scope,
				WorkspaceID:    spec.WorkspaceID,
				NetworkChannel: spec.NetworkChannel,
				Title:          spec.Title,
				Description:    spec.Description,
				Status:         taskpkg.TaskStatusPending,
				Owner:          spec.Owner,
				CreatedBy:      actor.Actor,
				Origin:         actor.Origin,
				CreatedAt:      time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC),
				UpdatedAt:      time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC),
				Metadata:       spec.Metadata,
			}, nil
		},
		ListTasksFn: func(_ context.Context, query taskpkg.TaskQuery, actor taskpkg.ActorContext) ([]taskpkg.TaskSummary, error) {
			capturedList = query
			capturedListActor = actor
			return []taskpkg.TaskSummary{{
				ID:        "task-1",
				Scope:     query.Scope,
				Title:     "Review task API",
				Status:    query.Status,
				CreatedBy: actor.Actor,
				Origin:    actor.Origin,
				CreatedAt: time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC),
			}}, nil
		},
	}
	workspaces := testutil.StubWorkspaceService{
		GetFn: func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
			if ref != "alpha" {
				t.Fatalf("workspace ref = %q, want %q", ref, "alpha")
			}
			return workspacepkg.Workspace{ID: "ws-alpha", Name: "alpha"}, nil
		},
	}

	fixture := newHandlerFixtureWithTasks(t, testutil.StubSessionManager{}, testutil.StubObserver{}, tasks, workspaces, nil, nil)
	fixture.Handlers.TaskActorContextResolver = func(_ *gin.Context, action string) (taskpkg.ActorContext, error) {
		return taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindHTTP, "tasks."+action)
	}

	createResp := performRequest(t, fixture.Engine, "POST", "/tasks", []byte(`{"scope":"workspace","workspace":"alpha","identifier":"TASK-1","network_channel":"builders","title":"Review task API","description":"Check handler wiring","owner":{"kind":"pool","ref":"reviewers"},"metadata":{"priority":"high"}}`))
	if createResp.Code != 201 {
		t.Fatalf("create status = %d, want %d; body=%s", createResp.Code, 201, createResp.Body.String())
	}

	if capturedCreate.Scope != taskpkg.ScopeWorkspace || capturedCreate.WorkspaceID != "ws-alpha" {
		t.Fatalf("create spec = %#v", capturedCreate)
	}
	if capturedCreate.NetworkChannel != "builders" || capturedCreate.Owner == nil || capturedCreate.Owner.Ref != "reviewers" {
		t.Fatalf("create spec = %#v", capturedCreate)
	}
	if capturedCreateActor.Actor.Ref != "user-1" || capturedCreateActor.Origin.Ref != "tasks.create" {
		t.Fatalf("create actor = %#v", capturedCreateActor)
	}

	listResp := performRequest(t, fixture.Engine, "GET", "/tasks?scope=workspace&workspace=alpha&status=ready&owner_kind=pool&owner_ref=reviewers&parent_task_id=task-root&network_channel=builders&limit=5", nil)
	if listResp.Code != 200 {
		t.Fatalf("list status = %d, want %d; body=%s", listResp.Code, 200, listResp.Body.String())
	}

	if capturedList.Scope != taskpkg.ScopeWorkspace || capturedList.WorkspaceID != "ws-alpha" {
		t.Fatalf("list query = %#v", capturedList)
	}
	if capturedList.Status != taskpkg.TaskStatusReady || capturedList.OwnerKind != taskpkg.OwnerKindPool || capturedList.OwnerRef != "reviewers" {
		t.Fatalf("list query = %#v", capturedList)
	}
	if capturedList.ParentTaskID != "task-root" || capturedList.NetworkChannel != "builders" || capturedList.Limit != 5 {
		t.Fatalf("list query = %#v", capturedList)
	}
	if capturedListActor.Actor.Ref != "user-1" || capturedListActor.Origin.Ref != "tasks.list" {
		t.Fatalf("list actor = %#v", capturedListActor)
	}
}

func TestTaskRunHandlersDelegateLifecycleSequenceIntegration(t *testing.T) {
	t.Parallel()

	calls := make([]string, 0, 4)
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)

	tasks := testutil.StubTaskManager{
		EnqueueRunFn: func(_ context.Context, spec taskpkg.EnqueueRun, actor taskpkg.ActorContext) (*taskpkg.TaskRun, error) {
			calls = append(calls, "enqueue")
			return &taskpkg.TaskRun{
				ID:             "run-1",
				TaskID:         spec.TaskID,
				Status:         taskpkg.TaskRunStatusQueued,
				Attempt:        1,
				Origin:         actor.Origin,
				IdempotencyKey: spec.IdempotencyKey,
				NetworkChannel: spec.NetworkChannel,
				QueuedAt:       now,
			}, nil
		},
		ClaimRunFn: func(_ context.Context, runID string, claim taskpkg.ClaimRun, actor taskpkg.ActorContext) (*taskpkg.TaskRun, error) {
			calls = append(calls, "claim")
			return &taskpkg.TaskRun{
				ID:        runID,
				TaskID:    "task-1",
				Status:    taskpkg.TaskRunStatusClaimed,
				Attempt:   1,
				ClaimedBy: &actor.Actor,
				Origin:    actor.Origin,
				QueuedAt:  now,
				ClaimedAt: now.Add(time.Minute),
			}, nil
		},
		StartRunFn: func(_ context.Context, runID string, _ taskpkg.StartRun, actor taskpkg.ActorContext) (*taskpkg.TaskRun, error) {
			calls = append(calls, "start")
			return &taskpkg.TaskRun{
				ID:        runID,
				TaskID:    "task-1",
				Status:    taskpkg.TaskRunStatusRunning,
				Attempt:   1,
				SessionID: "sess-1",
				Origin:    actor.Origin,
				QueuedAt:  now,
				StartedAt: now.Add(2 * time.Minute),
			}, nil
		},
		CompleteRunFn: func(_ context.Context, runID string, result taskpkg.RunResult, actor taskpkg.ActorContext) (*taskpkg.TaskRun, error) {
			calls = append(calls, "complete")
			return &taskpkg.TaskRun{
				ID:       runID,
				TaskID:   "task-1",
				Status:   taskpkg.TaskRunStatusCompleted,
				Attempt:  1,
				Origin:   actor.Origin,
				QueuedAt: now,
				EndedAt:  now.Add(3 * time.Minute),
				Result:   result.Value,
			}, nil
		},
	}

	fixture := newHandlerFixtureWithTasks(t, testutil.StubSessionManager{}, testutil.StubObserver{}, tasks, testutil.StubWorkspaceService{}, nil, nil)
	fixture.Handlers.TaskActorContextResolver = func(_ *gin.Context, action string) (taskpkg.ActorContext, error) {
		return taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindHTTP, "tasks."+action)
	}

	resp := performRequest(t, fixture.Engine, "POST", "/tasks/task-1/runs", []byte(`{"idempotency_key":"key-1","network_channel":"builders"}`))
	if resp.Code != 201 {
		t.Fatalf("enqueue status = %d, want %d; body=%s", resp.Code, 201, resp.Body.String())
	}

	resp = performRequest(t, fixture.Engine, "POST", "/task-runs/run-1/claim", []byte(`{}`))
	if resp.Code != 200 {
		t.Fatalf("claim status = %d, want %d; body=%s", resp.Code, 200, resp.Body.String())
	}

	resp = performRequest(t, fixture.Engine, "POST", "/task-runs/run-1/start", []byte(`{}`))
	if resp.Code != 200 {
		t.Fatalf("start status = %d, want %d; body=%s", resp.Code, 200, resp.Body.String())
	}

	resp = performRequest(t, fixture.Engine, "POST", "/task-runs/run-1/complete", []byte(`{"result":{"ok":true}}`))
	if resp.Code != 200 {
		t.Fatalf("complete status = %d, want %d; body=%s", resp.Code, 200, resp.Body.String())
	}

	if want := []string{"enqueue", "claim", "start", "complete"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("call order = %#v, want %#v", calls, want)
	}
}
