package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/session"
	taskpkg "github.com/pedronauck/agh/internal/task"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestTaskSessionBridgeStartTaskSessionUsesDedicatedSystemSessions(t *testing.T) {
	t.Parallel()

	globalPath := t.TempDir()
	testCases := []struct {
		name          string
		taskRecord    taskpkg.Task
		wantWorkspace string
		wantPath      string
	}{
		{
			name: "Should use the workspace identifier for workspace-scoped tasks",
			taskRecord: taskpkg.Task{
				ID:          "task-workspace",
				Scope:       taskpkg.ScopeWorkspace,
				WorkspaceID: "ws-123",
				Title:       "Workspace Task",
			},
			wantWorkspace: "ws-123",
		},
		{
			name: "Should use the global workspace path for global tasks",
			taskRecord: taskpkg.Task{
				ID:    "task-global",
				Scope: taskpkg.ScopeGlobal,
				Title: "Global Task",
			},
			wantPath: globalPath,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sessions := &fakeSessionManager{}
			bridge, err := newTaskSessionBridge(sessions, globalPath, discardLogger())
			if err != nil {
				t.Fatalf("newTaskSessionBridge() error = %v", err)
			}

			ref, err := bridge.StartTaskSession(context.Background(), &taskpkg.StartTaskSession{
				Task: tc.taskRecord,
				Run: taskpkg.Run{
					ID:             "run-1",
					TaskID:         tc.taskRecord.ID,
					Status:         taskpkg.TaskRunStatusStarting,
					Attempt:        2,
					Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindCLI, Ref: "agh task run"},
					NetworkChannel: "builders",
					QueuedAt:       time.Date(2026, 4, 14, 18, 0, 0, 0, time.UTC),
				},
			})
			if err != nil {
				t.Fatalf("StartTaskSession() error = %v", err)
			}

			if ref == nil || strings.TrimSpace(ref.SessionID) == "" {
				t.Fatalf("StartTaskSession() ref = %#v, want non-empty session id", ref)
			}
			if got, want := sessions.createCount(), 1; got != want {
				t.Fatalf("createCount() = %d, want %d", got, want)
			}

			createCall := sessions.createCall(0)
			if got, want := createCall.Type, session.SessionTypeSystem; got != want {
				t.Fatalf("createCall.Type = %q, want %q", got, want)
			}
			if got, want := createCall.Channel, "builders"; got != want {
				t.Fatalf("createCall.Channel = %q, want %q", got, want)
			}
			if got, want := createCall.Workspace, tc.wantWorkspace; got != want {
				t.Fatalf("createCall.Workspace = %q, want %q", got, want)
			}
			if got, want := createCall.WorkspacePath, tc.wantPath; got != want {
				t.Fatalf("createCall.WorkspacePath = %q, want %q", got, want)
			}
			if !strings.Contains(createCall.Name, tc.taskRecord.Title) {
				t.Fatalf("createCall.Name = %q, want task title %q", createCall.Name, tc.taskRecord.Title)
			}
		})
	}
}

func TestTaskSessionBridgeAttachTaskSessionRejectsStoppedSessions(t *testing.T) {
	t.Parallel()

	sessions := &fakeSessionManager{
		infos: []*session.Info{
			{
				ID:          "sess-active",
				State:       session.StateActive,
				WorkspaceID: "ws-active",
				CreatedAt:   time.Date(2026, 4, 14, 18, 0, 0, 0, time.UTC),
			},
			{
				ID:          "sess-stopped",
				State:       session.StateStopped,
				WorkspaceID: "ws-stopped",
				CreatedAt:   time.Date(2026, 4, 14, 17, 0, 0, 0, time.UTC),
			},
		},
	}
	bridge, err := newTaskSessionBridge(sessions, t.TempDir(), discardLogger())
	if err != nil {
		t.Fatalf("newTaskSessionBridge() error = %v", err)
	}

	ref, err := bridge.AttachTaskSession(context.Background(), "run-1", "sess-active")
	if err != nil {
		t.Fatalf("AttachTaskSession(active) error = %v", err)
	}
	if got, want := ref.SessionID, "sess-active"; got != want {
		t.Fatalf("AttachTaskSession(active).SessionID = %q, want %q", got, want)
	}

	if _, err := bridge.AttachTaskSession(
		context.Background(),
		"run-1",
		"sess-stopped",
	); !errors.Is(
		err,
		taskpkg.ErrSessionAttachNotAllowed,
	) {
		t.Fatalf("AttachTaskSession(stopped) error = %v, want %v", err, taskpkg.ErrSessionAttachNotAllowed)
	}
}

func TestTaskSessionBridgeStopPathsUseCooperativeThenForcedCalls(t *testing.T) {
	t.Parallel()

	sessions := &fakeSessionManager{}
	bridge, err := newTaskSessionBridge(sessions, t.TempDir(), discardLogger())
	if err != nil {
		t.Fatalf("newTaskSessionBridge() error = %v", err)
	}

	if err := bridge.RequestTaskStop(context.Background(), "sess-1", taskpkg.StopReasonCancellation); err != nil {
		t.Fatalf("RequestTaskStop() error = %v", err)
	}
	if err := bridge.ForceTaskStop(context.Background(), "sess-1", taskpkg.StopReasonCancellation); err != nil {
		t.Fatalf("ForceTaskStop() error = %v", err)
	}

	if got, want := len(sessions.requestStopCalls), 1; got != want {
		t.Fatalf("len(requestStopCalls) = %d, want %d", got, want)
	}
	if got, want := sessions.requestStopCalls[0].cause, session.CauseUserRequested; got != want {
		t.Fatalf("requestStopCalls[0].cause = %v, want %v", got, want)
	}
	if got, want := sessions.requestStopCalls[0].detail, "task cancellation"; got != want {
		t.Fatalf("requestStopCalls[0].detail = %q, want %q", got, want)
	}
	if got, want := len(sessions.stopWithCauseCalls), 1; got != want {
		t.Fatalf("len(stopWithCauseCalls) = %d, want %d", got, want)
	}
}

func TestPlanTaskRunRecoveryClassifiesClaimedStartingRunning(t *testing.T) {
	t.Parallel()

	sessions := &fakeSessionManager{
		infos: []*session.Info{
			{ID: "sess-active", State: session.StateActive},
			{ID: "sess-stopping", State: session.StateStopping},
			{ID: "sess-stopped", State: session.StateStopped},
		},
	}

	testCases := []struct {
		name       string
		run        taskpkg.Run
		wantAction taskpkg.RunBootRecoveryAction
		wantState  string
		wantNil    bool
	}{
		{
			name: "Should requeue claimed runs without a bound session",
			run: taskpkg.Run{
				ID:       "run-claimed",
				TaskID:   "task-1",
				Status:   taskpkg.TaskRunStatusClaimed,
				Attempt:  1,
				Origin:   taskpkg.Origin{Kind: taskpkg.OriginKindCLI, Ref: "agh task run"},
				QueuedAt: time.Now().UTC(),
			},
			wantAction: taskpkg.RunBootRecoveryRequeue,
			wantState:  taskRecoverySessionMissing,
		},
		{
			name: "Should resume starting runs when the bound session is active",
			run: taskpkg.Run{
				ID:        "run-starting",
				TaskID:    "task-2",
				Status:    taskpkg.TaskRunStatusStarting,
				Attempt:   1,
				SessionID: "sess-active",
				Origin:    taskpkg.Origin{Kind: taskpkg.OriginKindCLI, Ref: "agh task run"},
				QueuedAt:  time.Now().UTC(),
			},
			wantAction: taskpkg.RunBootRecoveryMarkRunning,
			wantState:  string(session.StateActive),
		},
		{
			name: "Should keep running runs live while the bound session is stopping",
			run: taskpkg.Run{
				ID:        "run-running",
				TaskID:    "task-3",
				Status:    taskpkg.TaskRunStatusRunning,
				Attempt:   1,
				SessionID: "sess-stopping",
				Origin:    taskpkg.Origin{Kind: taskpkg.OriginKindCLI, Ref: "agh task run"},
				QueuedAt:  time.Now().UTC(),
			},
			wantNil: true,
		},
		{
			name: "Should fail starting runs when the bound session is stopped",
			run: taskpkg.Run{
				ID:        "run-orphaned-starting",
				TaskID:    "task-4",
				Status:    taskpkg.TaskRunStatusStarting,
				Attempt:   1,
				SessionID: "sess-stopped",
				Origin:    taskpkg.Origin{Kind: taskpkg.OriginKindCLI, Ref: "agh task run"},
				QueuedAt:  time.Now().UTC(),
			},
			wantAction: taskpkg.RunBootRecoveryFail,
			wantState:  string(session.StateStopped),
		},
		{
			name: "Should fail running runs when the bound session is missing",
			run: taskpkg.Run{
				ID:        "run-orphaned-running",
				TaskID:    "task-5",
				Status:    taskpkg.TaskRunStatusRunning,
				Attempt:   1,
				SessionID: "sess-missing",
				Origin:    taskpkg.Origin{Kind: taskpkg.OriginKindCLI, Ref: "agh task run"},
				QueuedAt:  time.Now().UTC(),
			},
			wantAction: taskpkg.RunBootRecoveryFail,
			wantState:  taskRecoverySessionMissing,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			recovery, err := planTaskRunRecovery(context.Background(), sessions, tc.run)
			if err != nil {
				t.Fatalf("planTaskRunRecovery() error = %v", err)
			}
			if tc.wantNil {
				if recovery != nil {
					t.Fatalf("planTaskRunRecovery() = %#v, want nil", recovery)
				}
				return
			}
			if recovery == nil {
				t.Fatal("planTaskRunRecovery() = nil, want recovery action")
			}
			if got, want := recovery.Action, tc.wantAction; got != want {
				t.Fatalf("recovery.Action = %q, want %q", got, want)
			}
			if got, want := recovery.SessionState, tc.wantState; got != want {
				t.Fatalf("recovery.SessionState = %q, want %q", got, want)
			}
		})
	}
}

func TestTaskSessionBridgeGuardsAndFallbackStopPaths(t *testing.T) {
	t.Parallel()

	if _, err := newTaskSessionBridge(nil, t.TempDir(), discardLogger()); err == nil {
		t.Fatal("newTaskSessionBridge(nil) error = nil, want validation error")
	}

	bridge, err := newTaskSessionBridge(&fakeSessionManager{}, "", discardLogger())
	if err != nil {
		t.Fatalf("newTaskSessionBridge() error = %v", err)
	}

	if _, err := bridge.StartTaskSession(nilTaskRuntimeContext(), &taskpkg.StartTaskSession{}); err == nil {
		t.Fatal("StartTaskSession(nil ctx) error = nil, want validation error")
	}
	if _, err := bridge.StartTaskSession(context.Background(), &taskpkg.StartTaskSession{
		Task: taskpkg.Task{
			ID:    "task-global",
			Scope: taskpkg.ScopeGlobal,
		},
		Run: taskpkg.Run{
			ID:      "run-global",
			Attempt: 1,
		},
	}); err == nil {
		t.Fatal("StartTaskSession(global without workspace path) error = nil, want validation error")
	}
	if _, err := bridge.StartTaskSession(context.Background(), &taskpkg.StartTaskSession{
		Task: taskpkg.Task{
			ID:    "task-invalid",
			Scope: taskpkg.Scope("invalid"),
		},
		Run: taskpkg.Run{
			ID:      "run-invalid",
			Attempt: 1,
		},
	}); !errors.Is(err, taskpkg.ErrValidation) {
		t.Fatalf("StartTaskSession(invalid scope) error = %v, want %v", err, taskpkg.ErrValidation)
	}
	if _, err := bridge.AttachTaskSession(nilTaskRuntimeContext(), "run-1", "sess-1"); err == nil {
		t.Fatal("AttachTaskSession(nil ctx) error = nil, want validation error")
	}
	if err := bridge.RequestTaskStop(nilTaskRuntimeContext(), "sess-1", taskpkg.StopReasonCancellation); err == nil {
		t.Fatal("RequestTaskStop(nil ctx) error = nil, want validation error")
	}
	if err := bridge.ForceTaskStop(
		context.Background(),
		"   ",
		taskpkg.StopReasonCancellation,
	); !errors.Is(
		err,
		taskpkg.ErrValidation,
	) {
		t.Fatalf("ForceTaskStop(blank id) error = %v, want %v", err, taskpkg.ErrValidation)
	}

	sessions := &fakeSessionManager{
		requestStopErr: func(string, session.StopCause, string) error {
			return session.ErrSessionNotFound
		},
		stopWithCauseErr: func(string, session.StopCause, string) error {
			return session.ErrSessionNotFound
		},
	}
	bridge, err = newTaskSessionBridge(sessions, t.TempDir(), discardLogger())
	if err != nil {
		t.Fatalf("newTaskSessionBridge() error = %v", err)
	}
	if err := bridge.RequestTaskStop(context.Background(), "sess-missing", taskpkg.StopReasonShutdown); err != nil {
		t.Fatalf("RequestTaskStop(missing) error = %v, want nil", err)
	}
	if err := bridge.ForceTaskStop(context.Background(), "sess-missing", taskpkg.StopReasonOrphanedRun); err != nil {
		t.Fatalf("ForceTaskStop(missing) error = %v, want nil", err)
	}

	stopOnlyBridge, err := newTaskSessionBridge(&taskBridgeStopOnlySessionManager{}, t.TempDir(), discardLogger())
	if err != nil {
		t.Fatalf("newTaskSessionBridge(stop-only) error = %v", err)
	}
	if err := stopOnlyBridge.RequestTaskStop(
		context.Background(),
		"sess-fallback",
		taskpkg.StopReasonShutdown,
	); err != nil {
		t.Fatalf("RequestTaskStop(fallback) error = %v", err)
	}

	stopOnly := stopOnlyBridge.sessions.(*taskBridgeStopOnlySessionManager)
	if got, want := len(stopOnly.stopCalls), 1; got != want {
		t.Fatalf("len(stopCalls) = %d, want %d", got, want)
	}
	if got, want := stopOnly.stopCalls[0].cause, session.CauseShutdown; got != want {
		t.Fatalf("stopCalls[0].cause = %v, want %v", got, want)
	}
	if got, want := stopOnly.stopCalls[0].detail, "task shutdown"; got != want {
		t.Fatalf("stopCalls[0].detail = %q, want %q", got, want)
	}
}

func TestTaskRuntimeHelpers(t *testing.T) {
	t.Parallel()

	if got, want := taskSessionName(&taskpkg.StartTaskSession{
		Task: taskpkg.Task{
			Identifier: "build-index",
		},
		Run: taskpkg.Run{
			ID:      "run-identifier",
			Attempt: 3,
		},
	}), "task:build-index#3"; got != want {
		t.Fatalf("taskSessionName(identifier) = %q, want %q", got, want)
	}
	if got, want := taskSessionName(&taskpkg.StartTaskSession{
		Run: taskpkg.Run{
			ID:      "run-fallback",
			Attempt: 4,
		},
	}), "task:run-fallback#4"; got != want {
		t.Fatalf("taskSessionName(run fallback) = %q, want %q", got, want)
	}

	if got, want := taskStopCause(taskpkg.StopReasonShutdown), session.CauseShutdown; got != want {
		t.Fatalf("taskStopCause(shutdown) = %v, want %v", got, want)
	}
	if got, want := taskStopCause(taskpkg.StopReasonOrphanedRun), session.CauseFailed; got != want {
		t.Fatalf("taskStopCause(orphaned) = %v, want %v", got, want)
	}
	if got, want := taskStopCause(taskpkg.StopReasonCancellation), session.CauseUserRequested; got != want {
		t.Fatalf("taskStopCause(cancellation) = %v, want %v", got, want)
	}
	if got, want := taskStopDetail(taskpkg.StopReasonShutdown), "task shutdown"; got != want {
		t.Fatalf("taskStopDetail(shutdown) = %q, want %q", got, want)
	}
	if got, want := taskStopDetail(taskpkg.StopReasonOrphanedRun), "task run orphaned"; got != want {
		t.Fatalf("taskStopDetail(orphaned) = %q, want %q", got, want)
	}
	if got, want := taskStopDetail(taskpkg.StopReasonCancellation), "task cancellation"; got != want {
		t.Fatalf("taskStopDetail(cancellation) = %q, want %q", got, want)
	}

	live, state, err := taskSessionRuntimeState(context.Background(), &taskBridgeStopOnlySessionManager{}, "")
	if err != nil {
		t.Fatalf("taskSessionRuntimeState(blank id) error = %v", err)
	}
	if live {
		t.Fatal("taskSessionRuntimeState(blank id) live = true, want false")
	}
	if got, want := state, taskRecoverySessionMissing; got != want {
		t.Fatalf("taskSessionRuntimeState(blank id) state = %q, want %q", got, want)
	}

	if _, err := planTaskRunRecovery(context.Background(), nil, taskpkg.Run{
		ID:     "run-1",
		Status: taskpkg.TaskRunStatusClaimed,
	}); err == nil {
		t.Fatal("planTaskRunRecovery(nil sessions) error = nil, want validation error")
	}
}

func TestTaskRuntimeDetachedHarnessSubmissionPersistsMetadataAndReusesIdempotency(t *testing.T) {
	t.Parallel()

	sessions := &fakeSessionManager{}
	runtime, resolver, _ := newDetachedHarnessTaskRuntimeForTest(t, sessions)
	workspace := resolveDaemonWorkspace(t, resolver, filepath.Join(t.TempDir(), "workspace"))
	sessions.infos = []*session.Info{
		{
			ID:          "sess-owner",
			Type:        session.SessionTypeSystem,
			State:       session.StateActive,
			WorkspaceID: workspace.ID,
			Workspace:   workspace.RootDir,
			Channel:     "builders",
		},
		{
			ID:          "sess-wake",
			Type:        session.SessionTypeSystem,
			State:       session.StateActive,
			WorkspaceID: workspace.ID,
			Workspace:   workspace.RootDir,
			Channel:     "builders",
		},
	}

	req := detachedHarnessSubmitRequest{
		SubmissionKey:  "detached-work-1",
		OwnerSessionID: "sess-owner",
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    workspace.ID,
		Summary:        "Workspace detached audit",
		Description:    "Review the queued harness work.",
		NetworkChannel: "builders",
		TurnSource:     session.TurnSourceNetwork,
		WakeTarget: detachedHarnessWakeTargetInput{
			SessionID: "sess-wake",
		},
	}

	first, err := runtime.submitDetachedHarnessWork(context.Background(), req)
	if err != nil {
		t.Fatalf("submitDetachedHarnessWork(first) error = %v", err)
	}
	if first == nil {
		t.Fatal("submitDetachedHarnessWork(first) = nil, want submission")
	}
	if first.ExistingTask {
		t.Fatal("submitDetachedHarnessWork(first).ExistingTask = true, want false")
	}
	if first.ExistingRun {
		t.Fatal("submitDetachedHarnessWork(first).ExistingRun = true, want false")
	}

	second, err := runtime.submitDetachedHarnessWork(context.Background(), req)
	if err != nil {
		t.Fatalf("submitDetachedHarnessWork(duplicate) error = %v", err)
	}
	if second == nil {
		t.Fatal("submitDetachedHarnessWork(duplicate) = nil, want submission")
	}
	if !second.ExistingTask || !second.ExistingRun {
		t.Fatalf("duplicate submission flags = task:%v run:%v, want both true", second.ExistingTask, second.ExistingRun)
	}
	if got, want := second.Task.ID, first.Task.ID; got != want {
		t.Fatalf("duplicate task id = %q, want %q", got, want)
	}
	if got, want := second.Run.ID, first.Run.ID; got != want {
		t.Fatalf("duplicate run id = %q, want %q", got, want)
	}

	actor, err := detachedHarnessActorContext("sess-owner")
	if err != nil {
		t.Fatalf("detachedHarnessActorContext() error = %v", err)
	}
	storedTask, err := runtime.store.GetTask(context.Background(), first.Task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got, want := storedTask.Scope, taskpkg.ScopeWorkspace; got != want {
		t.Fatalf("storedTask.Scope = %q, want %q", got, want)
	}
	if got, want := storedTask.WorkspaceID, workspace.ID; got != want {
		t.Fatalf("storedTask.WorkspaceID = %q, want %q", got, want)
	}
	if storedTask.Owner == nil {
		t.Fatal("storedTask.Owner = nil, want owner session")
	}
	if got, want := storedTask.Owner.Kind, taskpkg.OwnerKindAgentSession; got != want {
		t.Fatalf("storedTask.Owner.Kind = %q, want %q", got, want)
	}
	if got, want := storedTask.Owner.Ref, "sess-owner"; got != want {
		t.Fatalf("storedTask.Owner.Ref = %q, want %q", got, want)
	}
	if got, want := storedTask.CreatedBy, actor.Actor; got != want {
		t.Fatalf("storedTask.CreatedBy = %#v, want %#v", got, want)
	}
	if got, want := storedTask.Origin, actor.Origin; got != want {
		t.Fatalf("storedTask.Origin = %#v, want %#v", got, want)
	}

	taskMetadata, err := decodeDetachedHarnessTaskMetadata(storedTask.Metadata)
	if err != nil {
		t.Fatalf("decodeDetachedHarnessTaskMetadata() error = %v", err)
	}
	if got, want := taskMetadata, (detachedHarnessTaskMetadata{
		Schema:               harnessDetachedMetadataSchema,
		Kind:                 harnessDetachedTaskMetadataKey,
		SubmissionKey:        "detached-work-1",
		Summary:              "Workspace detached audit",
		SubmissionTurnSource: string(session.TurnSourceNetwork),
		OwnerSessionID:       "sess-owner",
		OwnerSessionType:     string(session.SessionTypeSystem),
		OwnerWorkspaceID:     workspace.ID,
		OwnerChannel:         "builders",
		WakeTarget: detachedHarnessWakeTarget{
			SessionID:   "sess-wake",
			SessionType: string(session.SessionTypeSystem),
			WorkspaceID: workspace.ID,
			Channel:     "builders",
		},
	}); got != want {
		t.Fatalf("task metadata = %#v, want %#v", got, want)
	}

	storedRun, err := runtime.store.GetTaskRun(context.Background(), first.Run.ID)
	if err != nil {
		t.Fatalf("GetTaskRun() error = %v", err)
	}
	if got, want := storedRun.TaskID, storedTask.ID; got != want {
		t.Fatalf("storedRun.TaskID = %q, want %q", got, want)
	}
	if got, want := storedRun.Status, taskpkg.TaskRunStatusQueued; got != want {
		t.Fatalf("storedRun.Status = %q, want %q", got, want)
	}
	if got, want := storedRun.Origin, actor.Origin; got != want {
		t.Fatalf("storedRun.Origin = %#v, want %#v", got, want)
	}
	if got, want := storedRun.IdempotencyKey, "detached-work-1"; got != want {
		t.Fatalf("storedRun.IdempotencyKey = %q, want %q", got, want)
	}

	runMetadata, err := decodeDetachedHarnessRunMetadata(storedRun.Metadata)
	if err != nil {
		t.Fatalf("decodeDetachedHarnessRunMetadata() error = %v", err)
	}
	if got, want := runMetadata, (detachedHarnessRunMetadata{
		Schema:               harnessDetachedMetadataSchema,
		Kind:                 harnessDetachedRunMetadataKey,
		SubmissionKey:        "detached-work-1",
		Summary:              "Workspace detached audit",
		SubmissionTurnSource: string(session.TurnSourceNetwork),
		OwnerSessionID:       "sess-owner",
		OwnerSessionType:     string(session.SessionTypeSystem),
		OwnerWorkspaceID:     workspace.ID,
		OwnerChannel:         "builders",
		WakeTarget: detachedHarnessWakeTarget{
			SessionID:   "sess-wake",
			SessionType: string(session.SessionTypeSystem),
			WorkspaceID: workspace.ID,
			Channel:     "builders",
		},
	}); got != want {
		t.Fatalf("run metadata = %#v, want %#v", got, want)
	}

	readActor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task inspect")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}
	view, err := runtime.manager.GetTask(context.Background(), storedTask.ID, readActor)
	if err != nil {
		t.Fatalf("manager.GetTask() error = %v", err)
	}
	if got, want := len(view.Runs), 1; got != want {
		t.Fatalf("len(view.Runs) = %d, want %d", got, want)
	}
	runs, err := runtime.manager.ListTaskRuns(context.Background(), storedTask.ID, taskpkg.RunQuery{}, readActor)
	if err != nil {
		t.Fatalf("manager.ListTaskRuns() error = %v", err)
	}
	if got, want := len(runs), 1; got != want {
		t.Fatalf("len(runs) = %d, want %d", got, want)
	}
}

func TestTaskRuntimeDetachedHarnessSubmissionValidationErrors(t *testing.T) {
	t.Parallel()

	sessions := &fakeSessionManager{
		infos: []*session.Info{
			{
				ID:          "sess-owner",
				Type:        session.SessionTypeSystem,
				State:       session.StateActive,
				WorkspaceID: "ws-owner",
				Workspace:   "/tmp/ws-owner",
				Channel:     "builders",
			},
			{
				ID:          "sess-other-workspace",
				Type:        session.SessionTypeSystem,
				State:       session.StateActive,
				WorkspaceID: "ws-other",
				Workspace:   "/tmp/ws-other",
				Channel:     "builders",
			},
		},
	}
	runtime, _, _ := newDetachedHarnessTaskRuntimeForTest(t, sessions)

	testCases := []struct {
		name string
		req  detachedHarnessSubmitRequest
	}{
		{
			name: "Should reject blank wake target session id",
			req: detachedHarnessSubmitRequest{
				SubmissionKey:  "detached-invalid-blank-wake",
				OwnerSessionID: "sess-owner",
				Scope:          taskpkg.ScopeGlobal,
				WakeTarget:     detachedHarnessWakeTargetInput{},
			},
		},
		{
			name: "Should reject unsupported scope",
			req: detachedHarnessSubmitRequest{
				SubmissionKey:  "detached-invalid-scope",
				OwnerSessionID: "sess-owner",
				Scope:          taskpkg.Scope("invalid"),
				WakeTarget: detachedHarnessWakeTargetInput{
					SessionID: "sess-owner",
				},
			},
		},
		{
			name: "Should reject workspace mismatch between owner and wake target",
			req: detachedHarnessSubmitRequest{
				SubmissionKey:  "detached-invalid-workspace",
				OwnerSessionID: "sess-owner",
				Scope:          taskpkg.ScopeWorkspace,
				WakeTarget: detachedHarnessWakeTargetInput{
					SessionID: "sess-other-workspace",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := runtime.submitDetachedHarnessWork(
				context.Background(),
				tc.req,
			); !errors.Is(
				err,
				taskpkg.ErrValidation,
			) {
				t.Fatalf("submitDetachedHarnessWork() error = %v, want %v", err, taskpkg.ErrValidation)
			}
		})
	}
}

func TestRecoverTaskRunsOnBootPreservesDetachedHarnessMetadata(t *testing.T) {
	t.Parallel()

	sessions := &fakeSessionManager{}
	runtime, resolver, _ := newDetachedHarnessTaskRuntimeForTest(t, sessions)
	workspace := resolveDaemonWorkspace(t, resolver, filepath.Join(t.TempDir(), "workspace"))
	sessions.infos = []*session.Info{
		{
			ID:          "sess-owner",
			Type:        session.SessionTypeSystem,
			State:       session.StateActive,
			WorkspaceID: workspace.ID,
			Workspace:   workspace.RootDir,
			Channel:     "builders",
		},
		{
			ID:          "sess-wake",
			Type:        session.SessionTypeSystem,
			State:       session.StateActive,
			WorkspaceID: workspace.ID,
			Workspace:   workspace.RootDir,
			Channel:     "builders",
		},
		{
			ID:          "sess-runtime",
			Type:        session.SessionTypeSystem,
			State:       session.StateActive,
			WorkspaceID: workspace.ID,
			Workspace:   workspace.RootDir,
			Channel:     "builders",
		},
	}

	submission, err := runtime.submitDetachedHarnessWork(context.Background(), detachedHarnessSubmitRequest{
		SubmissionKey:  "detached-recovery-1",
		OwnerSessionID: "sess-owner",
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    workspace.ID,
		Summary:        "Recover detached harness run",
		NetworkChannel: "builders",
		TurnSource:     session.TurnSourceSynthetic,
		WakeTarget: detachedHarnessWakeTargetInput{
			SessionID: "sess-wake",
		},
	})
	if err != nil {
		t.Fatalf("submitDetachedHarnessWork() error = %v", err)
	}

	actor, err := detachedHarnessActorContext("sess-owner")
	if err != nil {
		t.Fatalf("detachedHarnessActorContext() error = %v", err)
	}
	claimed, err := runtime.manager.ClaimRun(context.Background(), submission.Run.ID, taskpkg.ClaimRun{
		IdempotencyKey: "claim-detached-recovery-1",
	}, actor)
	if err != nil {
		t.Fatalf("ClaimRun() error = %v", err)
	}
	starting, err := runtime.manager.AttachRunSession(context.Background(), claimed.ID, "sess-runtime", actor)
	if err != nil {
		t.Fatalf("AttachRunSession() error = %v", err)
	}
	if got, want := starting.Status, taskpkg.TaskRunStatusStarting; got != want {
		t.Fatalf("starting.Status = %q, want %q", got, want)
	}

	bootActor, err := taskpkg.DeriveDaemonActorContext("boot-recovery", "daemon.boot")
	if err != nil {
		t.Fatalf("DeriveDaemonActorContext() error = %v", err)
	}
	stats, err := recoverTaskRunsOnBoot(context.Background(), runtime.manager, runtime.store, sessions, bootActor)
	if err != nil {
		t.Fatalf("recoverTaskRunsOnBoot() error = %v", err)
	}
	if got, want := stats.markedRunning, 1; got != want {
		t.Fatalf("stats.markedRunning = %d, want %d", got, want)
	}

	recovered, err := runtime.store.GetTaskRun(context.Background(), submission.Run.ID)
	if err != nil {
		t.Fatalf("GetTaskRun(recovered) error = %v", err)
	}
	if got, want := recovered.Status, taskpkg.TaskRunStatusRunning; got != want {
		t.Fatalf("recovered.Status = %q, want %q", got, want)
	}
	metadata, err := decodeDetachedHarnessRunMetadata(recovered.Metadata)
	if err != nil {
		t.Fatalf("decodeDetachedHarnessRunMetadata(recovered) error = %v", err)
	}
	if got, want := metadata.SubmissionKey, "detached-recovery-1"; got != want {
		t.Fatalf("recovered metadata submission key = %q, want %q", got, want)
	}
	if got, want := metadata.OwnerSessionID, "sess-owner"; got != want {
		t.Fatalf("recovered metadata owner session id = %q, want %q", got, want)
	}
	if got, want := metadata.WakeTarget.SessionID, "sess-wake"; got != want {
		t.Fatalf("recovered metadata wake target session id = %q, want %q", got, want)
	}
}

func TestDetachedHarnessWorkBridgeHelperValidation(t *testing.T) {
	t.Parallel()

	if _, err := newHarnessDetachedWorkBridge(nil, openDaemonTestGlobalDB(t), &fakeSessionManager{}); err == nil {
		t.Fatal("newHarnessDetachedWorkBridge(nil tasks) error = nil, want validation error")
	}
	if _, err := newHarnessDetachedWorkBridge(&taskpkg.Service{}, nil, &fakeSessionManager{}); err == nil {
		t.Fatal("newHarnessDetachedWorkBridge(nil store) error = nil, want validation error")
	}
	if _, err := newHarnessDetachedWorkBridge(&taskpkg.Service{}, openDaemonTestGlobalDB(t), nil); err == nil {
		t.Fatal("newHarnessDetachedWorkBridge(nil sessions) error = nil, want validation error")
	}

	if _, err := decodeDetachedHarnessTaskMetadata(
		json.RawMessage(`{"schema":"bad","kind":"other"}`),
	); !errors.Is(
		err,
		taskpkg.ErrValidation,
	) {
		t.Fatalf("decodeDetachedHarnessTaskMetadata(wrong schema) error = %v, want %v", err, taskpkg.ErrValidation)
	}
	if _, err := decodeDetachedHarnessTaskMetadata(json.RawMessage(`{`)); !errors.Is(err, taskpkg.ErrValidation) {
		t.Fatalf("decodeDetachedHarnessTaskMetadata(invalid json) error = %v, want %v", err, taskpkg.ErrValidation)
	}
	if _, err := decodeDetachedHarnessRunMetadata(
		json.RawMessage(`{"schema":"bad","kind":"other"}`),
	); !errors.Is(
		err,
		taskpkg.ErrValidation,
	) {
		t.Fatalf("decodeDetachedHarnessRunMetadata(wrong schema) error = %v, want %v", err, taskpkg.ErrValidation)
	}
	if _, err := decodeDetachedHarnessRunMetadata(json.RawMessage(`{`)); !errors.Is(err, taskpkg.ErrValidation) {
		t.Fatalf("decodeDetachedHarnessRunMetadata(invalid json) error = %v, want %v", err, taskpkg.ErrValidation)
	}

	if got, want := detachedHarnessSummary("   "), defaultDetachedHarnessSummary; got != want {
		t.Fatalf("detachedHarnessSummary(blank) = %q, want %q", got, want)
	}
	if got, want := detachedHarnessChannel("", " owners "), "owners"; got != want {
		t.Fatalf("detachedHarnessChannel(owner fallback) = %q, want %q", got, want)
	}
	if got, want := normalizeDetachedHarnessTurnSource(
		session.TurnSource("unexpected"),
	), session.TurnSourceUser; got != want {
		t.Fatalf("normalizeDetachedHarnessTurnSource(unexpected) = %q, want %q", got, want)
	}
}

func TestTaskRuntimeDetachedHarnessSubmissionRejectsExistingMismatches(t *testing.T) {
	t.Parallel()

	sessions := &fakeSessionManager{}
	runtime, resolver, _ := newDetachedHarnessTaskRuntimeForTest(t, sessions)
	workspace := resolveDaemonWorkspace(t, resolver, filepath.Join(t.TempDir(), "workspace"))
	sessions.infos = []*session.Info{
		{
			ID:          "sess-owner",
			Type:        session.SessionTypeSystem,
			State:       session.StateActive,
			WorkspaceID: workspace.ID,
			Workspace:   workspace.RootDir,
			Channel:     "builders",
		},
		{
			ID:          "sess-wake",
			Type:        session.SessionTypeSystem,
			State:       session.StateActive,
			WorkspaceID: workspace.ID,
			Workspace:   workspace.RootDir,
			Channel:     "builders",
		},
	}

	baseReq := detachedHarnessSubmitRequest{
		SubmissionKey:  "detached-mismatch",
		OwnerSessionID: "sess-owner",
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    workspace.ID,
		Summary:        "Original detached work",
		NetworkChannel: "builders",
		WakeTarget: detachedHarnessWakeTargetInput{
			SessionID: "sess-wake",
		},
	}
	if _, err := runtime.submitDetachedHarnessWork(context.Background(), baseReq); err != nil {
		t.Fatalf("submitDetachedHarnessWork(base) error = %v", err)
	}

	if _, err := runtime.submitDetachedHarnessWork(context.Background(), detachedHarnessSubmitRequest{
		SubmissionKey:  "detached-mismatch",
		OwnerSessionID: "sess-owner",
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    workspace.ID,
		Summary:        "Changed detached work",
		NetworkChannel: "builders",
		WakeTarget: detachedHarnessWakeTargetInput{
			SessionID: "sess-wake",
		},
	}); !errors.Is(err, taskpkg.ErrValidation) {
		t.Fatalf("submitDetachedHarnessWork(run mismatch) error = %v, want %v", err, taskpkg.ErrValidation)
	}

	actor, err := detachedHarnessActorContext("sess-owner")
	if err != nil {
		t.Fatalf("detachedHarnessActorContext() error = %v", err)
	}
	conflictMetadata, err := marshalDetachedHarnessMetadata(detachedHarnessTaskMetadata{
		Schema:               harnessDetachedMetadataSchema,
		Kind:                 harnessDetachedTaskMetadataKey,
		SubmissionKey:        "detached-conflict",
		Summary:              "Conflicting stored task",
		SubmissionTurnSource: string(session.TurnSourceUser),
		OwnerSessionID:       "sess-owner",
		OwnerSessionType:     string(session.SessionTypeSystem),
		OwnerWorkspaceID:     workspace.ID,
		OwnerChannel:         "builders",
		WakeTarget: detachedHarnessWakeTarget{
			SessionID:   "sess-wake",
			SessionType: string(session.SessionTypeSystem),
			WorkspaceID: workspace.ID,
			Channel:     "builders",
		},
	})
	if err != nil {
		t.Fatalf("marshalDetachedHarnessMetadata() error = %v", err)
	}
	conflictingTask := taskpkg.Task{
		ID:             detachedHarnessTaskID("sess-owner", "detached-conflict"),
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    workspace.ID,
		Title:          "Conflicting stored task",
		Status:         taskpkg.TaskStatusPending,
		MaxAttempts:    taskpkg.DefaultTaskMaxAttempts,
		ApprovalPolicy: taskpkg.ApprovalPolicyNone,
		ApprovalState:  taskpkg.ApprovalStateNotRequired,
		Owner: &taskpkg.Ownership{
			Kind: taskpkg.OwnerKindAgentSession,
			Ref:  "sess-owner",
		},
		CreatedBy: actor.Actor,
		Origin:    actor.Origin,
		CreatedAt: time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC),
		Metadata:  conflictMetadata,
	}
	if err := runtime.store.CreateTask(context.Background(), conflictingTask); err != nil {
		t.Fatalf("CreateTask(conflictingTask) error = %v", err)
	}

	if _, err := runtime.submitDetachedHarnessWork(context.Background(), detachedHarnessSubmitRequest{
		SubmissionKey:  "detached-conflict",
		OwnerSessionID: "sess-owner",
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    workspace.ID,
		Summary:        "Expected detached work",
		NetworkChannel: "builders",
		WakeTarget: detachedHarnessWakeTargetInput{
			SessionID: "sess-wake",
		},
	}); !errors.Is(err, taskpkg.ErrValidation) {
		t.Fatalf("submitDetachedHarnessWork(task mismatch) error = %v, want %v", err, taskpkg.ErrValidation)
	}

	if _, err := runtime.submitDetachedHarnessWork(context.Background(), detachedHarnessSubmitRequest{
		SubmissionKey:  "missing-session",
		OwnerSessionID: "sess-owner",
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    workspace.ID,
		Summary:        "Missing wake target",
		NetworkChannel: "builders",
		WakeTarget: detachedHarnessWakeTargetInput{
			SessionID: "sess-missing",
		},
	}); !errors.Is(err, taskpkg.ErrValidation) {
		t.Fatalf("submitDetachedHarnessWork(missing session) error = %v, want %v", err, taskpkg.ErrValidation)
	}
}

func TestTaskRuntimeSubmitDetachedHarnessWorkGuards(t *testing.T) {
	t.Parallel()

	var nilRuntime *taskRuntime
	if _, err := nilRuntime.submitDetachedHarnessWork(
		context.Background(),
		detachedHarnessSubmitRequest{},
	); err == nil {
		t.Fatal("nil runtime submit error = nil, want validation error")
	}

	runtime := &taskRuntime{}
	if _, err := runtime.submitDetachedHarnessWork(context.Background(), detachedHarnessSubmitRequest{}); err == nil {
		t.Fatal("runtime without detached bridge error = nil, want validation error")
	}

	sessions := &fakeSessionManager{
		infos: []*session.Info{
			{ID: "sess-owner", Type: session.SessionTypeSystem, State: session.StateActive},
			{ID: "sess-wake", Type: session.SessionTypeSystem, State: session.StateActive},
		},
	}
	readyRuntime, _, _ := newDetachedHarnessTaskRuntimeForTest(t, sessions)
	if _, err := readyRuntime.submitDetachedHarnessWork(nilTaskRuntimeContext(), detachedHarnessSubmitRequest{
		SubmissionKey:  "detached-guard",
		OwnerSessionID: "sess-owner",
		Scope:          taskpkg.ScopeGlobal,
		WakeTarget: detachedHarnessWakeTargetInput{
			SessionID: "sess-wake",
		},
	}); err == nil {
		t.Fatal("submitDetachedHarnessWork(nil ctx) error = nil, want validation error")
	}
}

func TestDetachedHarnessMatchValidatorsRejectConflicts(t *testing.T) {
	t.Parallel()

	req := normalizedDetachedHarnessSubmitRequest{
		TaskID:           detachedHarnessTaskID("sess-owner", "validator"),
		SubmissionKey:    "validator",
		Scope:            taskpkg.ScopeWorkspace,
		WorkspaceID:      "ws-1",
		Summary:          "Validator task",
		Description:      "Ensure helper coverage",
		NetworkChannel:   "builders",
		TurnSource:       session.TurnSourceSynthetic,
		OwnerSessionID:   "sess-owner",
		OwnerSessionType: string(session.SessionTypeSystem),
		OwnerWorkspaceID: "ws-1",
		OwnerChannel:     "builders",
		WakeTarget: detachedHarnessWakeTarget{
			SessionID:   "sess-wake",
			SessionType: string(session.SessionTypeSystem),
			WorkspaceID: "ws-1",
			Channel:     "builders",
		},
	}
	actor, err := detachedHarnessActorContext(req.OwnerSessionID)
	if err != nil {
		t.Fatalf("detachedHarnessActorContext() error = %v", err)
	}
	taskMetadata := buildDetachedHarnessTaskMetadata(req)
	taskMetadataJSON, err := marshalDetachedHarnessMetadata(taskMetadata)
	if err != nil {
		t.Fatalf("marshalDetachedHarnessMetadata(task) error = %v", err)
	}
	runMetadata := buildDetachedHarnessRunMetadata(req)
	runMetadataJSON, err := marshalDetachedHarnessMetadata(runMetadata)
	if err != nil {
		t.Fatalf("marshalDetachedHarnessMetadata(run) error = %v", err)
	}

	matchingTask := taskpkg.Task{
		ID:             req.TaskID,
		Scope:          req.Scope,
		WorkspaceID:    req.WorkspaceID,
		NetworkChannel: req.NetworkChannel,
		Title:          req.Summary,
		Description:    req.Description,
		Status:         taskpkg.TaskStatusPending,
		MaxAttempts:    taskpkg.DefaultTaskMaxAttempts,
		ApprovalPolicy: taskpkg.ApprovalPolicyNone,
		ApprovalState:  taskpkg.ApprovalStateNotRequired,
		Owner: &taskpkg.Ownership{
			Kind: taskpkg.OwnerKindAgentSession,
			Ref:  req.OwnerSessionID,
		},
		CreatedBy: actor.Actor,
		Origin:    actor.Origin,
		CreatedAt: time.Date(2026, 4, 18, 11, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 18, 11, 0, 0, 0, time.UTC),
		Metadata:  taskMetadataJSON,
	}
	if err := validateDetachedHarnessTaskMatch(matchingTask, req, actor, taskMetadata); err != nil {
		t.Fatalf("validateDetachedHarnessTaskMatch(match) error = %v", err)
	}

	missingOwnerTask := matchingTask
	missingOwnerTask.Owner = nil
	if err := validateDetachedHarnessTaskMatch(
		missingOwnerTask,
		req,
		actor,
		taskMetadata,
	); !errors.Is(
		err,
		taskpkg.ErrValidation,
	) {
		t.Fatalf("validateDetachedHarnessTaskMatch(missing owner) error = %v, want %v", err, taskpkg.ErrValidation)
	}

	matchingRun := taskpkg.Run{
		ID:             "run-validator",
		TaskID:         req.TaskID,
		Status:         taskpkg.TaskRunStatusQueued,
		Attempt:        1,
		Origin:         actor.Origin,
		IdempotencyKey: req.SubmissionKey,
		NetworkChannel: req.NetworkChannel,
		Metadata:       runMetadataJSON,
		QueuedAt:       time.Date(2026, 4, 18, 11, 5, 0, 0, time.UTC),
	}
	if err := validateDetachedHarnessRunMatch(matchingRun, req, actor.Origin, runMetadata); err != nil {
		t.Fatalf("validateDetachedHarnessRunMatch(match) error = %v", err)
	}

	wrongOriginRun := matchingRun
	wrongOriginRun.Origin = taskpkg.Origin{Kind: taskpkg.OriginKindCLI, Ref: "agh task run"}
	if err := validateDetachedHarnessRunMatch(
		wrongOriginRun,
		req,
		actor.Origin,
		runMetadata,
	); !errors.Is(
		err,
		taskpkg.ErrValidation,
	) {
		t.Fatalf("validateDetachedHarnessRunMatch(wrong origin) error = %v, want %v", err, taskpkg.ErrValidation)
	}
}

type taskBridgeStopOnlySessionManager struct {
	stopCalls []fakeStopWithCauseCall
}

func (m *taskBridgeStopOnlySessionManager) Create(context.Context, session.CreateOpts) (*session.Session, error) {
	return nil, nil
}

func (m *taskBridgeStopOnlySessionManager) Status(context.Context, string) (*session.Info, error) {
	return nil, session.ErrSessionNotFound
}

func (m *taskBridgeStopOnlySessionManager) StopWithCause(
	_ context.Context,
	id string,
	cause session.StopCause,
	detail string,
) error {
	m.stopCalls = append(m.stopCalls, fakeStopWithCauseCall{id: id, cause: cause, detail: detail})
	return nil
}

func nilTaskRuntimeContext() context.Context {
	return nil
}

func newDetachedHarnessTaskRuntimeForTest(
	t *testing.T,
	sessions *fakeSessionManager,
) (*taskRuntime, workspacepkg.RuntimeResolver, aghconfig.HomePaths) {
	t.Helper()

	if sessions == nil {
		sessions = &fakeSessionManager{}
	}

	db := openDaemonTestGlobalDB(t)
	homePaths := testHomePaths(t)
	resolver, err := workspacepkg.NewResolver(
		db,
		workspacepkg.WithHomePaths(homePaths),
		workspacepkg.WithLogger(discardLogger()),
		workspacepkg.WithConfigLoader(func(rootDir string) (aghconfig.Config, error) {
			return aghconfig.LoadForHome(homePaths, aghconfig.WithWorkspaceRoot(rootDir))
		}),
	)
	if err != nil {
		t.Fatalf("workspace.NewResolver() error = %v", err)
	}

	sessionBridge, err := newTaskSessionBridge(sessions, homePaths.HomeDir, discardLogger())
	if err != nil {
		t.Fatalf("newTaskSessionBridge() error = %v", err)
	}
	manager, err := taskpkg.NewManager(
		taskpkg.WithStore(db),
		taskpkg.WithSessionExecutor(sessionBridge),
		taskpkg.WithNetworkChannelValidator(network.ValidateChannel),
		taskpkg.WithCancelGracePeriod(defaultTaskCancelGrace),
	)
	if err != nil {
		t.Fatalf("task.NewManager() error = %v", err)
	}
	detached, err := newHarnessDetachedWorkBridge(manager, db, sessions)
	if err != nil {
		t.Fatalf("newHarnessDetachedWorkBridge() error = %v", err)
	}

	return &taskRuntime{
		manager:  manager,
		store:    db,
		detached: detached,
	}, resolver, homePaths
}
