package daemon

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

func TestTaskRoleRuntimeActivatesPoolOwnerSessions(t *testing.T) {
	t.Parallel()

	t.Run("Should start the pool owner agent session when a run is enqueued", func(t *testing.T) {
		t.Parallel()

		taskRecord := taskRoleRuntimeTask("task-frontend", "frontend-engineer-agent", "design-review")
		run := taskRoleRuntimeRun("run-frontend", taskRecord.ID, "design-review")
		store := newTaskRoleRuntimeStore(taskRecord, run)
		sessions := &taskRoleRuntimeSessions{}
		runtime := newTaskRoleRuntimeForTest(t, store, sessions)

		runtime.OnTaskRunEnqueued(context.Background(), hookspkg.TaskRunEnqueuedPayload{
			TaskRunContext: hookspkg.TaskRunContext{TaskID: taskRecord.ID, RunID: run.ID},
		})

		if got, want := sessions.createCount(), 1; got != want {
			t.Fatalf("create count = %d, want %d", got, want)
		}
		call := sessions.createCall(0)
		if got, want := call.AgentName, "frontend-engineer-agent"; got != want {
			t.Fatalf("CreateOpts.AgentName = %q, want %q", got, want)
		}
		if got, want := call.Workspace, taskRecord.WorkspaceID; got != want {
			t.Fatalf("CreateOpts.Workspace = %q, want %q", got, want)
		}
		if got, want := call.Channel, "design-review"; got != want {
			t.Fatalf("CreateOpts.Channel = %q, want %q", got, want)
		}
		if got, want := call.Type, session.SessionTypeSystem; got != want {
			t.Fatalf("CreateOpts.Type = %q, want %q", got, want)
		}
		if got := call.Provider; got != "" {
			t.Fatalf("CreateOpts.Provider = %q, want default provider resolution", got)
		}
		for _, required := range []string{"agh task next", "agh task run claim", run.ID, "design-review"} {
			if !strings.Contains(call.PromptOverlay, required) {
				t.Fatalf("PromptOverlay missing %q:\n%s", required, call.PromptOverlay)
			}
		}
	})

	t.Run("Should reuse an active matching role session for duplicate queued runs", func(t *testing.T) {
		t.Parallel()

		firstTask := taskRoleRuntimeTask("task-frontend-a", "frontend-engineer-agent", "design-review")
		firstRun := taskRoleRuntimeRun("run-frontend-a", firstTask.ID, "design-review")
		secondTask := taskRoleRuntimeTask("task-frontend-b", "frontend-engineer-agent", "design-review")
		secondRun := taskRoleRuntimeRun("run-frontend-b", secondTask.ID, "design-review")
		store := newTaskRoleRuntimeStore(firstTask, secondTask, firstRun, secondRun)
		sessions := &taskRoleRuntimeSessions{}
		runtime := newTaskRoleRuntimeForTest(t, store, sessions)

		runtime.OnTaskRunEnqueued(context.Background(), hookspkg.TaskRunEnqueuedPayload{
			TaskRunContext: hookspkg.TaskRunContext{TaskID: firstTask.ID, RunID: firstRun.ID},
		})
		runtime.OnTaskRunEnqueued(context.Background(), hookspkg.TaskRunEnqueuedPayload{
			TaskRunContext: hookspkg.TaskRunContext{TaskID: secondTask.ID, RunID: secondRun.ID},
		})

		if got, want := sessions.createCount(), 1; got != want {
			t.Fatalf("create count = %d, want %d", got, want)
		}
	})

	t.Run("Should recover queued pool-owned runs on boot", func(t *testing.T) {
		t.Parallel()

		frontendTask := taskRoleRuntimeTask("task-frontend", "frontend-engineer-agent", "design-review")
		frontendRun := taskRoleRuntimeRun("run-frontend", frontendTask.ID, "design-review")
		analyticsTask := taskRoleRuntimeTask("task-analytics", "analytics-engineer-agent", "data-watch")
		analyticsRun := taskRoleRuntimeRun("run-analytics", analyticsTask.ID, "data-watch")
		humanTask := taskRoleRuntimeTask("task-human", "human-owner", "ops")
		humanTask.Owner = &taskpkg.Ownership{Kind: taskpkg.OwnerKindHuman, Ref: "local-user"}
		humanRun := taskRoleRuntimeRun("run-human", humanTask.ID, "ops")
		store := newTaskRoleRuntimeStore(frontendTask, frontendRun, analyticsTask, analyticsRun, humanTask, humanRun)
		sessions := &taskRoleRuntimeSessions{}
		runtime := newTaskRoleRuntimeForTest(t, store, sessions)

		runtime.Recover(context.Background())

		if got, want := sessions.createCount(), 2; got != want {
			t.Fatalf("create count = %d, want %d", got, want)
		}
		gotAgents := []string{sessions.createCall(0).AgentName, sessions.createCall(1).AgentName}
		if !slices.Contains(gotAgents, "frontend-engineer-agent") ||
			!slices.Contains(gotAgents, "analytics-engineer-agent") {
			t.Fatalf("created agents = %#v, want frontend and analytics role sessions", gotAgents)
		}
	})
}

func newTaskRoleRuntimeForTest(
	t *testing.T,
	store *taskRoleRuntimeStore,
	sessions *taskRoleRuntimeSessions,
) *taskRoleRuntime {
	t.Helper()

	runtime, err := newTaskRoleRuntime(store, sessions, t.TempDir(), discardLogger())
	if err != nil {
		t.Fatalf("newTaskRoleRuntime() error = %v", err)
	}
	return runtime
}

func taskRoleRuntimeTask(id string, ownerRef string, channel string) taskpkg.Task {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	return taskpkg.Task{
		ID:             id,
		Scope:          taskpkg.ScopeWorkspace,
		WorkspaceID:    "ws-growth",
		NetworkChannel: channel,
		Title:          "Task " + id,
		Status:         taskpkg.TaskStatusReady,
		Priority:       taskpkg.PriorityMedium,
		Owner:          &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: ownerRef},
		CreatedBy:      taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
		Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindCLI, Ref: "task-role-test"},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func taskRoleRuntimeRun(id string, taskID string, channel string) taskpkg.Run {
	return taskpkg.Run{
		ID:                    id,
		TaskID:                taskID,
		Status:                taskpkg.TaskRunStatusQueued,
		Attempt:               1,
		Origin:                taskpkg.Origin{Kind: taskpkg.OriginKindCLI, Ref: "task-role-test"},
		NetworkChannel:        channel,
		CoordinationChannelID: channel,
		QueuedAt:              time.Date(2026, 5, 6, 12, 1, 0, 0, time.UTC),
	}
}

type taskRoleRuntimeStore struct {
	mu    sync.Mutex
	tasks map[string]taskpkg.Task
	runs  map[string]taskpkg.Run
}

func newTaskRoleRuntimeStore(records ...any) *taskRoleRuntimeStore {
	store := &taskRoleRuntimeStore{
		tasks: make(map[string]taskpkg.Task),
		runs:  make(map[string]taskpkg.Run),
	}
	for _, record := range records {
		switch value := record.(type) {
		case taskpkg.Task:
			store.tasks[value.ID] = value
		case taskpkg.Run:
			store.runs[value.ID] = value
		}
	}
	return store
}

func (s *taskRoleRuntimeStore) GetTask(_ context.Context, id string) (taskpkg.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	taskRecord, ok := s.tasks[strings.TrimSpace(id)]
	if !ok {
		return taskpkg.Task{}, taskpkg.ErrTaskNotFound
	}
	return taskRecord, nil
}

func (s *taskRoleRuntimeStore) GetTaskRun(_ context.Context, id string) (taskpkg.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[strings.TrimSpace(id)]
	if !ok {
		return taskpkg.Run{}, taskpkg.ErrTaskRunNotFound
	}
	return run, nil
}

func (s *taskRoleRuntimeStore) ListTaskRunsByStatus(
	_ context.Context,
	statuses []taskpkg.RunStatus,
) ([]taskpkg.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	allowed := make(map[taskpkg.RunStatus]struct{}, len(statuses))
	for _, status := range statuses {
		allowed[status.Normalize()] = struct{}{}
	}
	runs := make([]taskpkg.Run, 0, len(s.runs))
	for _, run := range s.runs {
		if _, ok := allowed[run.Status.Normalize()]; ok {
			runs = append(runs, run)
		}
	}
	return runs, nil
}

type taskRoleRuntimeSessions struct {
	mu          sync.Mutex
	infos       []*session.Info
	createCalls []session.CreateOpts
}

func (s *taskRoleRuntimeSessions) Create(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.createCalls = append(s.createCalls, opts)
	id := fmt.Sprintf("role-%d", len(s.createCalls))
	info := &session.Info{
		ID:          id,
		Name:        opts.Name,
		AgentName:   opts.AgentName,
		Provider:    opts.Provider,
		WorkspaceID: opts.Workspace,
		Workspace:   firstNonEmpty(opts.Workspace, opts.WorkspacePath),
		Channel:     opts.Channel,
		Type:        opts.Type,
		State:       session.StateActive,
		CreatedAt:   time.Date(2026, 5, 6, 12, 2, 0, len(s.createCalls), time.UTC),
	}
	s.infos = append(s.infos, info)
	return &session.Session{
		ID:          info.ID,
		Name:        info.Name,
		AgentName:   info.AgentName,
		Provider:    info.Provider,
		WorkspaceID: info.WorkspaceID,
		Workspace:   info.Workspace,
		Channel:     info.Channel,
		Type:        info.Type,
		State:       info.State,
		CreatedAt:   info.CreatedAt,
	}, nil
}

func (s *taskRoleRuntimeSessions) ListAll(context.Context) ([]*session.Info, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	infos := make([]*session.Info, len(s.infos))
	copy(infos, s.infos)
	return infos, nil
}

func (s *taskRoleRuntimeSessions) createCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.createCalls)
}

func (s *taskRoleRuntimeSessions) createCall(index int) session.CreateOpts {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.createCalls) {
		return session.CreateOpts{}
	}
	return s.createCalls[index]
}
