package daemon

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/session"
	taskpkg "github.com/pedronauck/agh/internal/task"
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
			name: "workspace task uses workspace id",
			taskRecord: taskpkg.Task{
				ID:          "task-workspace",
				Scope:       taskpkg.ScopeWorkspace,
				WorkspaceID: "ws-123",
				Title:       "Workspace Task",
			},
			wantWorkspace: "ws-123",
		},
		{
			name: "global task uses global workspace path",
			taskRecord: taskpkg.Task{
				ID:    "task-global",
				Scope: taskpkg.ScopeGlobal,
				Title: "Global Task",
			},
			wantPath: globalPath,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sessions := &fakeSessionManager{}
			bridge, err := newTaskSessionBridge(sessions, globalPath, discardLogger())
			if err != nil {
				t.Fatalf("newTaskSessionBridge() error = %v", err)
			}

			ref, err := bridge.StartTaskSession(context.Background(), taskpkg.StartTaskSession{
				Task: tc.taskRecord,
				Run: taskpkg.TaskRun{
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
		infos: []*session.SessionInfo{
			{ID: "sess-active", State: session.StateActive, WorkspaceID: "ws-active", CreatedAt: time.Date(2026, 4, 14, 18, 0, 0, 0, time.UTC)},
			{ID: "sess-stopped", State: session.StateStopped, WorkspaceID: "ws-stopped", CreatedAt: time.Date(2026, 4, 14, 17, 0, 0, 0, time.UTC)},
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

	if _, err := bridge.AttachTaskSession(context.Background(), "run-1", "sess-stopped"); !errors.Is(err, taskpkg.ErrSessionAttachNotAllowed) {
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
		infos: []*session.SessionInfo{
			{ID: "sess-active", State: session.StateActive},
			{ID: "sess-stopping", State: session.StateStopping},
			{ID: "sess-stopped", State: session.StateStopped},
		},
	}

	testCases := []struct {
		name       string
		run        taskpkg.TaskRun
		wantAction taskpkg.RunBootRecoveryAction
		wantState  string
		wantNil    bool
	}{
		{
			name: "claimed without session requeues",
			run: taskpkg.TaskRun{
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
			name: "starting with active session resumes running",
			run: taskpkg.TaskRun{
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
			name: "running with stopping session is kept live",
			run: taskpkg.TaskRun{
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
			name: "starting with stopped session fails",
			run: taskpkg.TaskRun{
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
			name: "running with missing session fails",
			run: taskpkg.TaskRun{
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
		tc := tc
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

	if _, err := bridge.StartTaskSession(nilTaskRuntimeContext(), taskpkg.StartTaskSession{}); err == nil {
		t.Fatal("StartTaskSession(nil ctx) error = nil, want validation error")
	}
	if _, err := bridge.StartTaskSession(context.Background(), taskpkg.StartTaskSession{
		Task: taskpkg.Task{
			ID:    "task-global",
			Scope: taskpkg.ScopeGlobal,
		},
		Run: taskpkg.TaskRun{
			ID:      "run-global",
			Attempt: 1,
		},
	}); err == nil {
		t.Fatal("StartTaskSession(global without workspace path) error = nil, want validation error")
	}
	if _, err := bridge.StartTaskSession(context.Background(), taskpkg.StartTaskSession{
		Task: taskpkg.Task{
			ID:    "task-invalid",
			Scope: taskpkg.Scope("invalid"),
		},
		Run: taskpkg.TaskRun{
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
	if err := bridge.ForceTaskStop(context.Background(), "   ", taskpkg.StopReasonCancellation); !errors.Is(err, taskpkg.ErrValidation) {
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
	if err := stopOnlyBridge.RequestTaskStop(context.Background(), "sess-fallback", taskpkg.StopReasonShutdown); err != nil {
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

	if got, want := taskSessionName(taskpkg.StartTaskSession{
		Task: taskpkg.Task{
			Identifier: "build-index",
		},
		Run: taskpkg.TaskRun{
			ID:      "run-identifier",
			Attempt: 3,
		},
	}), "task:build-index#3"; got != want {
		t.Fatalf("taskSessionName(identifier) = %q, want %q", got, want)
	}
	if got, want := taskSessionName(taskpkg.StartTaskSession{
		Run: taskpkg.TaskRun{
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

	if _, err := planTaskRunRecovery(context.Background(), nil, taskpkg.TaskRun{
		ID:     "run-1",
		Status: taskpkg.TaskRunStatusClaimed,
	}); err == nil {
		t.Fatal("planTaskRunRecovery(nil sessions) error = nil, want validation error")
	}
}

type taskBridgeStopOnlySessionManager struct {
	stopCalls []fakeStopWithCauseCall
}

func (m *taskBridgeStopOnlySessionManager) Create(context.Context, session.CreateOpts) (*session.Session, error) {
	return nil, nil
}

func (m *taskBridgeStopOnlySessionManager) Status(context.Context, string) (*session.SessionInfo, error) {
	return nil, session.ErrSessionNotFound
}

func (m *taskBridgeStopOnlySessionManager) StopWithCause(_ context.Context, id string, cause session.StopCause, detail string) error {
	m.stopCalls = append(m.stopCalls, fakeStopWithCauseCall{id: id, cause: cause, detail: detail})
	return nil
}

func nilTaskRuntimeContext() context.Context {
	return nil
}
