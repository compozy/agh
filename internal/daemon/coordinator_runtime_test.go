package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/coordinator"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	storepkg "github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	toolspkg "github.com/pedronauck/agh/internal/tools"
)

func TestCoordinatorRuntimeBootstrapsManagedCoordinatorSession(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := newCoordinatorRuntimeStore(coordinatorRuntimeTask(), coordinatorRuntimeRun())
	sessions := &coordinatorRuntimeSessions{}
	hooks := &recordingCoordinatorHooks{}
	runtime := newCoordinatorRuntimeForTest(t, store, sessions, hooks, coordinatorRuntimeConfig(), now)

	info, created, err := runtime.bootstrapRun(
		context.Background(),
		store.tasks["task-1"],
		store.runs["run-1"],
		coordinator.ReasonRunEnqueued,
	)
	if err != nil {
		t.Fatalf("bootstrapRun() error = %v", err)
	}
	if !created {
		t.Fatal("bootstrapRun() created = false, want true")
	}
	if info == nil || info.Type != session.SessionTypeCoordinator {
		t.Fatalf("bootstrapRun() info = %#v, want coordinator info", info)
	}

	call := sessions.createCall(0)
	if call.Type != session.SessionTypeCoordinator {
		t.Fatalf("CreateOpts.Type = %q, want coordinator", call.Type)
	}
	if call.AgentName != "coordinator" || call.Provider != "codex" {
		t.Fatalf("CreateOpts agent/provider = %q/%q, want coordinator/codex", call.AgentName, call.Provider)
	}
	if call.Workspace != "ws-1" || call.Channel != "ch-run-1" {
		t.Fatalf("CreateOpts workspace/channel = %q/%q, want ws-1/ch-run-1", call.Workspace, call.Channel)
	}
	if call.Lineage == nil || call.Lineage.SpawnRole != string(session.SessionTypeCoordinator) {
		t.Fatalf("CreateOpts.Lineage = %#v, want coordinator root lineage", call.Lineage)
	}
	if err := storepkg.ValidateSessionLineage("coord-1", call.Lineage); err != nil {
		t.Fatalf("ValidateSessionLineage(coordinator create) error = %v", err)
	}
	if call.Lineage.TTLExpiresAt == nil || !call.Lineage.TTLExpiresAt.Equal(now.Add(2*time.Hour)) {
		t.Fatalf("Lineage.TTLExpiresAt = %#v, want %s", call.Lineage.TTLExpiresAt, now.Add(2*time.Hour))
	}
	if !coordinator.ToolAllowed(toolspkg.ToolIDTaskRunClaimNext.String()) ||
		coordinator.ToolAllowed(toolspkg.ToolIDTaskCancel.String()) {
		t.Fatal("coordinator tool allowlist is not restricted as expected")
	}
	if got := call.Lineage.PermissionPolicy.NetworkChannels; len(got) != 1 || got[0] != "ch-run-1" {
		t.Fatalf("Lineage.PermissionPolicy.NetworkChannels = %#v, want ch-run-1", got)
	}
	for _, required := range []string{"agh me context", "agh task next", "agh ch", "agh spawn", "ch-run-1"} {
		if !contains(call.PromptOverlay, required) {
			t.Fatalf("PromptOverlay missing %q:\n%s", required, call.PromptOverlay)
		}
	}
	if hooks.preSpawnCount() != 1 || hooks.spawnedCount() != 1 {
		t.Fatalf("hook counts pre_spawn/spawned = %d/%d, want 1/1", hooks.preSpawnCount(), hooks.spawnedCount())
	}
	if got := hooks.spawnedPayload(0).Model; got != "gpt-5" {
		t.Fatalf("spawned hook model = %q, want gpt-5", got)
	}
}

func TestCoordinatorRuntimeSkipsIneligibleRuns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		task       taskpkg.Task
		run        taskpkg.Run
		cfg        aghconfig.CoordinatorConfig
		wantReason string
	}{
		{
			name:       "disabled config",
			task:       coordinatorRuntimeTask(),
			run:        coordinatorRuntimeRun(),
			cfg:        aghconfig.DefaultCoordinatorConfig(),
			wantReason: coordinator.DecisionDisabled,
		},
		{
			name: "global scope",
			task: func() taskpkg.Task {
				task := coordinatorRuntimeTask()
				task.Scope = taskpkg.ScopeGlobal
				task.WorkspaceID = ""
				return task
			}(),
			run:        coordinatorRuntimeRun(),
			cfg:        coordinatorRuntimeConfig(),
			wantReason: coordinator.DecisionGlobalScope,
		},
		{
			name: "missing channel",
			task: coordinatorRuntimeTask(),
			run: func() taskpkg.Run {
				run := coordinatorRuntimeRun()
				run.CoordinationChannelID = ""
				return run
			}(),
			cfg:        coordinatorRuntimeConfig(),
			wantReason: coordinator.DecisionMissingChannel,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := newCoordinatorRuntimeStore(tc.task, tc.run)
			sessions := &coordinatorRuntimeSessions{}
			hooks := &recordingCoordinatorHooks{}
			runtime := newCoordinatorRuntimeForTest(t, store, sessions, hooks, tc.cfg, time.Now().UTC())

			_, created, err := runtime.bootstrapRun(
				context.Background(),
				tc.task,
				tc.run,
				coordinator.ReasonRunEnqueued,
			)
			if err != nil {
				t.Fatalf("bootstrapRun() error = %v", err)
			}
			if created {
				t.Fatal("bootstrapRun() created = true, want false")
			}
			if got := sessions.createCount(); got != 0 {
				t.Fatalf("Create count = %d, want 0", got)
			}
			if got := hooks.lastDecision(); got != tc.wantReason {
				t.Fatalf("last decision = %q, want %q", got, tc.wantReason)
			}
		})
	}
}

func TestCoordinatorRuntimePreventsDuplicateCoordinatorsUnderConcurrentEnqueue(t *testing.T) {
	t.Parallel()

	store := newCoordinatorRuntimeStore(coordinatorRuntimeTask(), coordinatorRuntimeRun())
	sessions := &coordinatorRuntimeSessions{}
	runtime := newCoordinatorRuntimeForTest(
		t,
		store,
		sessions,
		&recordingCoordinatorHooks{},
		coordinatorRuntimeConfig(),
		time.Now().UTC(),
	)

	const attempts = 24
	var wg sync.WaitGroup
	errs := make(chan error, attempts)
	for range attempts {
		wg.Go(func() {
			_, _, err := runtime.bootstrapRun(
				context.Background(),
				store.tasks["task-1"],
				store.runs["run-1"],
				coordinator.ReasonRunEnqueued,
			)
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("bootstrapRun() concurrent error = %v", err)
		}
	}
	if got := sessions.createCount(); got != 1 {
		t.Fatalf("Create count = %d, want 1", got)
	}
}

func TestCoordinatorRuntimeObservesTaskRunEnqueuedButNotTaskCreation(t *testing.T) {
	t.Parallel()

	store := newCoordinatorRuntimeStore(coordinatorRuntimeTask(), coordinatorRuntimeRun())
	sessions := &coordinatorRuntimeSessions{}
	runtime := newCoordinatorRuntimeForTest(
		t,
		store,
		sessions,
		&recordingCoordinatorHooks{},
		coordinatorRuntimeConfig(),
		time.Now().UTC(),
	)

	if got := sessions.createCount(); got != 0 {
		t.Fatalf("Create count before enqueue = %d, want 0", got)
	}
	runtime.OnTaskRunEnqueued(context.Background(), hookspkg.TaskRunEnqueuedPayload{
		TaskRunContext: hookspkg.TaskRunContext{TaskID: "task-1", RunID: "run-1"},
	})
	if got := sessions.createCount(); got != 1 {
		t.Fatalf("Create count after enqueue = %d, want 1", got)
	}
}

func TestCoordinatorRuntimeManualSessionsCoexist(t *testing.T) {
	t.Parallel()

	store := newCoordinatorRuntimeStore(coordinatorRuntimeTask(), coordinatorRuntimeRun())
	sessions := &coordinatorRuntimeSessions{
		infos: []*session.Info{{
			ID:          "manual-1",
			Type:        session.SessionTypeUser,
			WorkspaceID: "ws-1",
			State:       session.StateActive,
		}},
	}
	runtime := newCoordinatorRuntimeForTest(
		t,
		store,
		sessions,
		&recordingCoordinatorHooks{},
		coordinatorRuntimeConfig(),
		time.Now().UTC(),
	)

	_, created, err := runtime.bootstrapRun(
		context.Background(),
		store.tasks["task-1"],
		store.runs["run-1"],
		coordinator.ReasonRunEnqueued,
	)
	if err != nil {
		t.Fatalf("bootstrapRun() error = %v", err)
	}
	if !created {
		t.Fatal("bootstrapRun() created = false, want true")
	}
	if got := sessions.createCount(); got != 1 {
		t.Fatalf("Create count = %d, want 1", got)
	}
}

func TestCoordinatorRuntimeRecoversWhenCoordinatorStopsWithExecutableWork(t *testing.T) {
	t.Parallel()

	store := newCoordinatorRuntimeStore(coordinatorRuntimeTask(), coordinatorRuntimeRun())
	sessions := &coordinatorRuntimeSessions{}
	hooks := &recordingCoordinatorHooks{}
	runtime := newCoordinatorRuntimeForTest(t, store, sessions, hooks, coordinatorRuntimeConfig(), time.Now().UTC())

	runtime.OnSessionStopped(context.Background(), &session.Session{
		ID:          "coord-old",
		AgentName:   "coordinator",
		WorkspaceID: "ws-1",
		Workspace:   "ws-1",
		Type:        session.SessionTypeCoordinator,
		State:       session.StateStopped,
	})
	if got := sessions.createCount(); got != 1 {
		t.Fatalf("Create count after coordinator stop = %d, want 1", got)
	}
	if hooks.stoppedCount() != 1 {
		t.Fatalf("stopped hook count = %d, want 1", hooks.stoppedCount())
	}
}

func TestCoordinatorRuntimePreSpawnDeny(t *testing.T) {
	t.Parallel()

	store := newCoordinatorRuntimeStore(coordinatorRuntimeTask(), coordinatorRuntimeRun())
	sessions := &coordinatorRuntimeSessions{}
	hooks := &recordingCoordinatorHooks{denyPreSpawn: true}
	runtime := newCoordinatorRuntimeForTest(t, store, sessions, hooks, coordinatorRuntimeConfig(), time.Now().UTC())

	_, created, err := runtime.bootstrapRun(
		context.Background(),
		store.tasks["task-1"],
		store.runs["run-1"],
		coordinator.ReasonRunEnqueued,
	)
	if err != nil {
		t.Fatalf("bootstrapRun() error = %v", err)
	}
	if created {
		t.Fatal("bootstrapRun() created = true, want false")
	}
	if got := sessions.createCount(); got != 0 {
		t.Fatalf("Create count = %d, want 0", got)
	}
	if got := hooks.lastDecision(); got != coordinator.DecisionDenied {
		t.Fatalf("last decision = %q, want denied", got)
	}
}

func TestCoordinatorRuntimePreSpawnDenyFromHookError(t *testing.T) {
	t.Parallel()

	store := newCoordinatorRuntimeStore(coordinatorRuntimeTask(), coordinatorRuntimeRun())
	sessions := &coordinatorRuntimeSessions{}
	hooks := &recordingCoordinatorHooks{denyPreSpawn: true, denyWithError: true}
	runtime := newCoordinatorRuntimeForTest(t, store, sessions, hooks, coordinatorRuntimeConfig(), time.Now().UTC())

	_, created, err := runtime.bootstrapRun(
		context.Background(),
		store.tasks["task-1"],
		store.runs["run-1"],
		coordinator.ReasonRunEnqueued,
	)
	if err != nil {
		t.Fatalf("bootstrapRun() error = %v, want denied decision without failure", err)
	}
	if created {
		t.Fatal("bootstrapRun() created = true, want false")
	}
	if got := sessions.createCount(); got != 0 {
		t.Fatalf("Create count = %d, want 0", got)
	}
	if got := hooks.lastDecision(); got != coordinator.DecisionDenied {
		t.Fatalf("last decision = %q, want denied", got)
	}
	if got := hooks.failedCount(); got != 0 {
		t.Fatalf("failed hook count = %d, want 0 for policy denial", got)
	}
}

func TestHooksNotifierTaskRunEnqueuedObserversReceivePayload(t *testing.T) {
	t.Parallel()

	notifier := newHooksNotifier(discardLogger(), func() time.Time {
		return time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	})
	observer := &recordingTaskRunEnqueuedObserver{}
	notifier.AddTaskRunEnqueuedObserver(observer)

	_, err := notifier.DispatchTaskRunEnqueued(context.Background(), hookspkg.TaskRunEnqueuedPayload{
		TaskRunContext: hookspkg.TaskRunContext{
			TaskID:                "task-1",
			RunID:                 "run-1",
			WorkspaceID:           "ws-1",
			CoordinationChannelID: "ch-run-1",
		},
	})
	if err != nil {
		t.Fatalf("DispatchTaskRunEnqueued() error = %v", err)
	}
	if got := observer.count(); got != 1 {
		t.Fatalf("observer count = %d, want 1", got)
	}
	if got := observer.last().CoordinationChannelID; got != "ch-run-1" {
		t.Fatalf("observer channel = %q, want ch-run-1", got)
	}
}

func newCoordinatorRuntimeForTest(
	t *testing.T,
	store *coordinatorRuntimeStore,
	sessions *coordinatorRuntimeSessions,
	hooks *recordingCoordinatorHooks,
	cfg aghconfig.CoordinatorConfig,
	now time.Time,
) *coordinatorRuntime {
	t.Helper()
	runtime, err := newCoordinatorRuntime(
		store,
		sessions,
		&staticCoordinatorConfigResolver{cfg: cfg},
		hooks,
		discardLogger(),
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatalf("newCoordinatorRuntime() error = %v", err)
	}
	return runtime
}

func coordinatorRuntimeConfig() aghconfig.CoordinatorConfig {
	cfg := aghconfig.DefaultCoordinatorConfig()
	cfg.Enabled = true
	cfg.AgentName = "coordinator"
	cfg.Provider = "codex"
	cfg.Model = "gpt-5"
	cfg.DefaultTTL = 2 * time.Hour
	cfg.MaxChildren = 5
	cfg.MaxActivePerWorkspace = 1
	return cfg
}

func coordinatorRuntimeTask() taskpkg.Task {
	return taskpkg.Task{
		ID:          "task-1",
		Scope:       taskpkg.ScopeWorkspace,
		WorkspaceID: "ws-1",
		Status:      taskpkg.TaskStatusReady,
		Title:       "Build the thing",
	}
}

func coordinatorRuntimeRun() taskpkg.Run {
	return taskpkg.Run{
		ID:                    "run-1",
		TaskID:                "task-1",
		Status:                taskpkg.TaskRunStatusQueued,
		CoordinationChannelID: "ch-run-1",
		Metadata:              json.RawMessage(`{"workflow_id":"wf-1"}`),
	}
}

type staticCoordinatorConfigResolver struct {
	cfg aghconfig.CoordinatorConfig
	err error
	mu  sync.Mutex
	got []string
}

func (r *staticCoordinatorConfigResolver) ResolveCoordinatorConfig(
	_ context.Context,
	workspaceID string,
) (aghconfig.CoordinatorConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, workspaceID)
	if r.err != nil {
		return aghconfig.CoordinatorConfig{}, r.err
	}
	return r.cfg, nil
}

type coordinatorRuntimeStore struct {
	mu    sync.Mutex
	tasks map[string]taskpkg.Task
	runs  map[string]taskpkg.Run
}

func newCoordinatorRuntimeStore(tasks taskpkg.Task, runs taskpkg.Run) *coordinatorRuntimeStore {
	return &coordinatorRuntimeStore{
		tasks: map[string]taskpkg.Task{tasks.ID: tasks},
		runs:  map[string]taskpkg.Run{runs.ID: runs},
	}
}

func (s *coordinatorRuntimeStore) GetTask(_ context.Context, id string) (taskpkg.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok {
		return taskpkg.Task{}, taskpkg.ErrTaskNotFound
	}
	return task, nil
}

func (s *coordinatorRuntimeStore) GetTaskRun(_ context.Context, id string) (taskpkg.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[id]
	if !ok {
		return taskpkg.Run{}, taskpkg.ErrTaskRunNotFound
	}
	return run, nil
}

func (s *coordinatorRuntimeStore) ListTaskRunsByStatus(
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

type coordinatorRuntimeSessions struct {
	mu          sync.Mutex
	infos       []*session.Info
	createCalls []session.CreateOpts
	createErr   error
}

func (s *coordinatorRuntimeSessions) Create(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.createErr != nil {
		return nil, s.createErr
	}
	s.createCalls = append(s.createCalls, opts)
	id := fmt.Sprintf("coord-%d", len(s.createCalls))
	info := &session.Info{
		ID:          id,
		Name:        opts.Name,
		AgentName:   opts.AgentName,
		Provider:    opts.Provider,
		WorkspaceID: opts.Workspace,
		Workspace:   opts.Workspace,
		Channel:     opts.Channel,
		Type:        opts.Type,
		Lineage:     opts.Lineage,
		State:       session.StateActive,
		CreatedAt:   time.Now().UTC(),
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
		Lineage:     info.Lineage,
		State:       info.State,
		CreatedAt:   info.CreatedAt,
	}, nil
}

func (s *coordinatorRuntimeSessions) ListAll(context.Context) ([]*session.Info, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	infos := make([]*session.Info, len(s.infos))
	copy(infos, s.infos)
	return infos, nil
}

func (s *coordinatorRuntimeSessions) createCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.createCalls)
}

func (s *coordinatorRuntimeSessions) createCall(index int) session.CreateOpts {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.createCalls) {
		return session.CreateOpts{}
	}
	return s.createCalls[index]
}

type recordingCoordinatorHooks struct {
	mu            sync.Mutex
	denyPreSpawn  bool
	denyWithError bool
	preSpawn      []hookspkg.CoordinatorPreSpawnPayload
	spawned       []hookspkg.CoordinatorSpawnedPayload
	decisions     []hookspkg.CoordinatorDecisionPayload
	stopped       []hookspkg.CoordinatorStoppedPayload
	failed        []hookspkg.CoordinatorFailedPayload
}

type recordingTaskRunEnqueuedObserver struct {
	mu       sync.Mutex
	payloads []hookspkg.TaskRunEnqueuedPayload
}

func (o *recordingTaskRunEnqueuedObserver) OnTaskRunEnqueued(
	_ context.Context,
	payload hookspkg.TaskRunEnqueuedPayload,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.payloads = append(o.payloads, payload)
}

func (o *recordingTaskRunEnqueuedObserver) count() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.payloads)
}

func (o *recordingTaskRunEnqueuedObserver) last() hookspkg.TaskRunEnqueuedPayload {
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.payloads) == 0 {
		return hookspkg.TaskRunEnqueuedPayload{}
	}
	return o.payloads[len(o.payloads)-1]
}

func (h *recordingCoordinatorHooks) DispatchCoordinatorPreSpawn(
	_ context.Context,
	payload hookspkg.CoordinatorPreSpawnPayload,
) (hookspkg.CoordinatorPreSpawnPayload, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.preSpawn = append(h.preSpawn, payload)
	if h.denyPreSpawn {
		payload.Denied = true
		payload.DenyReason = "policy"
	}
	if h.denyWithError {
		return payload, fmt.Errorf("hooks: event %q denied", hookspkg.HookCoordinatorPreSpawn)
	}
	return payload, nil
}

func (h *recordingCoordinatorHooks) DispatchCoordinatorSpawned(
	_ context.Context,
	payload hookspkg.CoordinatorSpawnedPayload,
) (hookspkg.CoordinatorSpawnedPayload, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.spawned = append(h.spawned, payload)
	return payload, nil
}

func (h *recordingCoordinatorHooks) DispatchCoordinatorDecision(
	_ context.Context,
	payload hookspkg.CoordinatorDecisionPayload,
) (hookspkg.CoordinatorDecisionPayload, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.decisions = append(h.decisions, payload)
	return payload, nil
}

func (h *recordingCoordinatorHooks) DispatchCoordinatorStopped(
	_ context.Context,
	payload hookspkg.CoordinatorStoppedPayload,
) (hookspkg.CoordinatorStoppedPayload, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.stopped = append(h.stopped, payload)
	return payload, nil
}

func (h *recordingCoordinatorHooks) DispatchCoordinatorFailed(
	_ context.Context,
	payload hookspkg.CoordinatorFailedPayload,
) (hookspkg.CoordinatorFailedPayload, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failed = append(h.failed, payload)
	return payload, nil
}

func (h *recordingCoordinatorHooks) preSpawnCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.preSpawn)
}

func (h *recordingCoordinatorHooks) spawnedCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.spawned)
}

func (h *recordingCoordinatorHooks) stoppedCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.stopped)
}

func (h *recordingCoordinatorHooks) failedCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.failed)
}

func (h *recordingCoordinatorHooks) spawnedPayload(index int) hookspkg.CoordinatorSpawnedPayload {
	h.mu.Lock()
	defer h.mu.Unlock()
	if index < 0 || index >= len(h.spawned) {
		return hookspkg.CoordinatorSpawnedPayload{}
	}
	return h.spawned[index]
}

func (h *recordingCoordinatorHooks) lastDecision() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.decisions) == 0 {
		return ""
	}
	return h.decisions[len(h.decisions)-1].Decision
}

func contains(value string, needle string) bool {
	return strings.Contains(value, needle)
}
