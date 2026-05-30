package daemon

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	aghconfig "github.com/compozy/agh/internal/config"
	hookspkg "github.com/compozy/agh/internal/hooks"
	schedulerpkg "github.com/compozy/agh/internal/scheduler"
	"github.com/compozy/agh/internal/session"
	taskpkg "github.com/compozy/agh/internal/task"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func TestShellQuoteSimpleAlwaysSingleQuotes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty", value: "", want: "''"},
		{name: "simple", value: "frontend", want: "'frontend'"},
		{name: "metacharacters", value: "frontend; rm -rf /", want: "'frontend; rm -rf /'"},
		{name: "single quote", value: "owner's-tool", want: "'owner'\\''s-tool'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := shellQuoteSimple(tt.value); got != tt.want {
				t.Fatalf("shellQuoteSimple(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

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

	t.Run("Should start the selected execution-profile worker when no owner is set", func(t *testing.T) {
		t.Parallel()

		taskRecord := taskRoleRuntimeTask("task-profile-worker", "", "design-review")
		taskRecord.Owner = nil
		run := taskRoleRuntimeRun("run-profile-worker", taskRecord.ID, "design-review")
		profile := taskpkg.ExecutionProfile{
			TaskID: taskRecord.ID,
			Worker: taskpkg.WorkerProfile{
				Mode:                 taskpkg.WorkerModeSelect,
				AgentName:            "frontend-engineer",
				Provider:             "claude",
				Model:                "sonnet",
				RequiredCapabilities: []string{"frontend"},
			},
			Runtime: taskpkg.RuntimePolicy{Mode: taskpkg.RuntimeModeEvidence},
		}
		store := newTaskRoleRuntimeStore(taskRecord, run, profile)
		sessions := &taskRoleRuntimeSessions{}
		runtime := newTaskRoleRuntimeForTest(t, store, sessions)

		runtime.OnTaskRunEnqueued(context.Background(), hookspkg.TaskRunEnqueuedPayload{
			TaskRunContext: hookspkg.TaskRunContext{TaskID: taskRecord.ID, RunID: run.ID},
		})

		if got, want := sessions.createCount(), 1; got != want {
			t.Fatalf("create count = %d, want %d", got, want)
		}
		call := sessions.createCall(0)
		if got, want := call.AgentName, "frontend-engineer"; got != want {
			t.Fatalf("CreateOpts.AgentName = %q, want %q", got, want)
		}
		if got, want := call.Provider, "claude"; got != want {
			t.Fatalf("CreateOpts.Provider = %q, want %q", got, want)
		}
		if got, want := call.Model, "sonnet"; got != want {
			t.Fatalf("CreateOpts.Model = %q, want %q", got, want)
		}
		if got, want := call.Permissions, aghconfig.PermissionModeApproveAll; got != want {
			t.Fatalf("CreateOpts.Permissions = %q, want %q", got, want)
		}
		if !strings.Contains(call.PromptOverlay, "Runtime evidence mode is enabled") {
			t.Fatalf("PromptOverlay missing runtime evidence guidance:\n%s", call.PromptOverlay)
		}
		if !strings.Contains(call.PromptOverlay, "--capability 'frontend'") {
			t.Fatalf("PromptOverlay missing required capability claim:\n%s", call.PromptOverlay)
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

var taskRoleRuntimeClock = time.Date(2026, 5, 6, 12, 5, 0, 0, time.UTC)

func TestTaskRoleRuntimeActivateForStarvation(t *testing.T) {
	t.Parallel()

	t.Run("Should spawn the pool owner with a TTL-bounded spawn budget lineage", func(t *testing.T) {
		t.Parallel()

		taskRecord := taskRoleRuntimeTask("task-starved", "frontend-engineer-agent", "design-review")
		run := taskRoleRuntimeRun("run-starved", taskRecord.ID, "design-review")
		store := newTaskRoleRuntimeStore(taskRecord, run)
		sessions := &taskRoleRuntimeSessions{}
		runtime := newTaskRoleRuntimeForTest(t, store, sessions)

		if err := runtime.activateForStarvation(
			context.Background(),
			taskRecord,
			run,
			starvationSpawner{},
		); err != nil {
			t.Fatalf("activateForStarvation() error = %v", err)
		}
		if got, want := sessions.createCount(), 1; got != want {
			t.Fatalf("create count = %d, want %d", got, want)
		}
		call := sessions.createCall(0)
		if got, want := call.AgentName, "frontend-engineer-agent"; got != want {
			t.Fatalf("CreateOpts.AgentName = %q, want %q", got, want)
		}
		if got, want := call.Workspace, taskRecord.WorkspaceID; got != want {
			t.Fatalf("CreateOpts.Workspace = %q, want run workspace %q", got, want)
		}
		if got, want := call.Type, session.SessionTypeSystem; got != want {
			t.Fatalf("CreateOpts.Type = %q, want %q", got, want)
		}
		if call.Lineage == nil {
			t.Fatal("CreateOpts.Lineage = nil, want TTL + spawn budget")
		}
		if call.Lineage.ParentSessionID != "" {
			t.Fatalf(
				"CreateOpts.Lineage.ParentSessionID = %q, want empty (parent-less worker)",
				call.Lineage.ParentSessionID,
			)
		}
		if call.Lineage.TTLExpiresAt == nil || !call.Lineage.TTLExpiresAt.After(taskRoleRuntimeClock) {
			t.Fatalf("CreateOpts.Lineage.TTLExpiresAt = %v, want a future deadline", call.Lineage.TTLExpiresAt)
		}
		if call.Lineage.SpawnBudget.TTLSeconds <= 0 || call.Lineage.SpawnBudget.MaxChildren <= 0 {
			t.Fatalf("CreateOpts.Lineage.SpawnBudget = %#v, want positive ttl + children", call.Lineage.SpawnBudget)
		}
	})

	t.Run(
		"Should prefer a capability-matched agent over the pool owner when the run requires capabilities",
		func(t *testing.T) {
			t.Parallel()

			taskRecord := taskRoleRuntimeTask("task-cap-owner", "frontend-engineer-agent", "design-review")
			run := taskRoleRuntimeRun("run-cap-owner", taskRecord.ID, "design-review")
			run.RequiredCapabilities = []string{"sqlite"}
			store := newTaskRoleRuntimeStore(taskRecord, run)
			sessions := &taskRoleRuntimeSessions{}
			runtime := newTaskRoleRuntimeForTest(t, store, sessions)
			spawner := starvationSpawner{
				workspaces: &fakeSpawnWorkspaceResolver{
					resolved: workspacepkg.ResolvedWorkspace{Agents: []aghconfig.AgentDef{
						spawnAgentDef("frontend-engineer-agent", "typescript"),
						spawnAgentDef("storage-agent", "sqlite"),
					}},
				},
				agents: reviewRouterAgentResolverStub{
					"frontend-engineer-agent": spawnAgentDef("frontend-engineer-agent", "typescript"),
					"storage-agent":           spawnAgentDef("storage-agent", "sqlite"),
				},
			}

			if err := runtime.activateForStarvation(context.Background(), taskRecord, run, spawner); err != nil {
				t.Fatalf("activateForStarvation() error = %v", err)
			}
			if got, want := sessions.createCount(), 1; got != want {
				t.Fatalf("create count = %d, want %d", got, want)
			}
			if got, want := sessions.createCall(0).AgentName, "storage-agent"; got != want {
				t.Fatalf("CreateOpts.AgentName = %q, want capability-matched %q", got, want)
			}
		},
	)

	t.Run(
		"Should spawn an eligible workspace agent when the run has no required capabilities or owner",
		func(t *testing.T) {
			t.Parallel()

			taskRecord := taskRoleRuntimeTask("task-no-cap-owner", "", "design-review")
			taskRecord.Owner = nil
			run := taskRoleRuntimeRun("run-no-cap-owner", taskRecord.ID, "design-review")
			store := newTaskRoleRuntimeStore(taskRecord, run)
			sessions := &taskRoleRuntimeSessions{}
			runtime := newTaskRoleRuntimeForTest(t, store, sessions)
			spawner := starvationSpawner{
				workspaces: &fakeSpawnWorkspaceResolver{
					resolved: workspacepkg.ResolvedWorkspace{Agents: []aghconfig.AgentDef{
						spawnAgentDef("zeta-agent"),
						spawnAgentDef("alpha-agent"),
					}},
				},
				agents: reviewRouterAgentResolverStub{
					"zeta-agent":  spawnAgentDef("zeta-agent"),
					"alpha-agent": spawnAgentDef("alpha-agent"),
				},
			}

			if err := runtime.activateForStarvation(context.Background(), taskRecord, run, spawner); err != nil {
				t.Fatalf("activateForStarvation() error = %v", err)
			}
			if got, want := sessions.createCount(), 1; got != want {
				t.Fatalf("create count = %d, want %d", got, want)
			}
			if got, want := sessions.createCall(0).AgentName, "alpha-agent"; got != want {
				t.Fatalf("CreateOpts.AgentName = %q, want eligible workspace agent %q", got, want)
			}
		},
	)

	t.Run("Should skip spawning when no agent covers the required capabilities", func(t *testing.T) {
		t.Parallel()

		taskRecord := taskRoleRuntimeTask("task-cap", "", "design-review")
		taskRecord.Owner = nil
		run := taskRoleRuntimeRun("run-cap", taskRecord.ID, "design-review")
		run.RequiredCapabilities = []string{"go"}
		store := newTaskRoleRuntimeStore(taskRecord, run)
		sessions := &taskRoleRuntimeSessions{}
		runtime := newTaskRoleRuntimeForTest(t, store, sessions)
		spawner := starvationSpawner{
			workspaces: &fakeSpawnWorkspaceResolver{
				resolved: workspacepkg.ResolvedWorkspace{
					Agents: []aghconfig.AgentDef{spawnAgentDef("docs-agent", "docs")},
				},
			},
			agents: reviewRouterAgentResolverStub{"docs-agent": spawnAgentDef("docs-agent", "docs")},
		}

		err := runtime.activateForStarvation(context.Background(), taskRecord, run, spawner)
		if !errors.Is(err, errStarvationSpawnUnresolvable) {
			t.Fatalf("activateForStarvation() error = %v, want errStarvationSpawnUnresolvable", err)
		}
		if got := sessions.createCount(); got != 0 {
			t.Fatalf("create count = %d, want 0 (no capable agent)", got)
		}
	})

	t.Run("Should reuse an already-active role session instead of spawning a duplicate", func(t *testing.T) {
		t.Parallel()

		taskRecord := taskRoleRuntimeTask("task-dup", "frontend-engineer-agent", "design-review")
		run := taskRoleRuntimeRun("run-dup", taskRecord.ID, "design-review")
		store := newTaskRoleRuntimeStore(taskRecord, run)
		sessions := &taskRoleRuntimeSessions{}
		runtime := newTaskRoleRuntimeForTest(t, store, sessions)

		if err := runtime.activateForStarvation(
			context.Background(),
			taskRecord,
			run,
			starvationSpawner{},
		); err != nil {
			t.Fatalf("first activateForStarvation() error = %v", err)
		}
		if err := runtime.activateForStarvation(
			context.Background(),
			taskRecord,
			run,
			starvationSpawner{},
		); err != nil {
			t.Fatalf("second activateForStarvation() error = %v", err)
		}
		if got, want := sessions.createCount(), 1; got != want {
			t.Fatalf("create count = %d, want %d (dedup is the per-role cap)", got, want)
		}
	})
}

func TestEscalationActorAdapterRequestWorkerSpawn(t *testing.T) {
	t.Parallel()

	t.Run("Should retry later instead of coalescing when task roles are not booted", func(t *testing.T) {
		t.Parallel()

		adapter := escalationActorAdapter{tasks: &taskRuntime{}}
		err := adapter.RequestWorkerSpawn(context.Background(), &schedulerpkg.RunSnapshot{})
		if !errors.Is(err, schedulerpkg.ErrSpawnUnresolvable) {
			t.Fatalf("RequestWorkerSpawn() error = %v, want %v", err, schedulerpkg.ErrSpawnUnresolvable)
		}
	})
}

func newTaskRoleRuntimeForTest(
	t *testing.T,
	store *taskRoleRuntimeStore,
	sessions *taskRoleRuntimeSessions,
) *taskRoleRuntime {
	t.Helper()

	runtime, err := newTaskRoleRuntime(
		store,
		sessions,
		t.TempDir(),
		discardLogger(),
		func() time.Time { return taskRoleRuntimeClock },
	)
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
	mu       sync.Mutex
	tasks    map[string]taskpkg.Task
	runs     map[string]taskpkg.Run
	profiles map[string]taskpkg.ExecutionProfile
}

func newTaskRoleRuntimeStore(records ...any) *taskRoleRuntimeStore {
	store := &taskRoleRuntimeStore{
		tasks:    make(map[string]taskpkg.Task),
		runs:     make(map[string]taskpkg.Run),
		profiles: make(map[string]taskpkg.ExecutionProfile),
	}
	for _, record := range records {
		switch value := record.(type) {
		case taskpkg.Task:
			store.tasks[value.ID] = value
		case taskpkg.Run:
			store.runs[value.ID] = value
		case taskpkg.ExecutionProfile:
			store.profiles[value.TaskID] = value
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

func (s *taskRoleRuntimeStore) GetExecutionProfile(
	_ context.Context,
	taskID string,
) (taskpkg.ExecutionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	profile, ok := s.profiles[strings.TrimSpace(taskID)]
	if !ok {
		return taskpkg.ExecutionProfile{}, taskpkg.ErrExecutionProfileNotFound
	}
	return profile, nil
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
