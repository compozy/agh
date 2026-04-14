package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type inMemoryManagerStore struct {
	tasks            map[string]Task
	dependencies     map[string]map[string]TaskDependency
	runs             map[string]TaskRun
	events           []TaskEvent
	idempotencyByKey map[string]TaskRunIdempotency
}

type testSessionExecutor struct{}

func (testSessionExecutor) StartTaskSession(context.Context, StartTaskSession) (*SessionRef, error) {
	return &SessionRef{SessionID: "sess-test"}, nil
}

func (testSessionExecutor) AttachTaskSession(context.Context, string, string) (*SessionRef, error) {
	return &SessionRef{SessionID: "sess-test"}, nil
}

func (testSessionExecutor) RequestTaskStop(context.Context, string, StopReason) error {
	return nil
}

func (testSessionExecutor) ForceTaskStop(context.Context, string, StopReason) error {
	return nil
}

type sessionStopCall struct {
	SessionID string
	Reason    StopReason
}

type attachSessionCall struct {
	RunID     string
	SessionID string
}

type recordingSessionExecutor struct {
	startCalls       []StartTaskSession
	attachCalls      []attachSessionCall
	requestStopCalls []sessionStopCall
	forceStopCalls   []sessionStopCall
	startRef         *SessionRef
	returnNilStart   bool
	startErr         error
	attachErr        error
	requestStopErr   error
	forceStopErr     error
}

func (e *recordingSessionExecutor) StartTaskSession(_ context.Context, spec StartTaskSession) (*SessionRef, error) {
	e.startCalls = append(e.startCalls, spec)
	if e.startErr != nil {
		return nil, e.startErr
	}
	if e.returnNilStart {
		return nil, nil
	}
	if e.startRef != nil {
		ref := *e.startRef
		return &ref, nil
	}
	return &SessionRef{SessionID: "sess-start-" + strconv.Itoa(len(e.startCalls))}, nil
}

func (e *recordingSessionExecutor) AttachTaskSession(_ context.Context, runID string, sessionID string) (*SessionRef, error) {
	e.attachCalls = append(e.attachCalls, attachSessionCall{RunID: runID, SessionID: sessionID})
	if e.attachErr != nil {
		return nil, e.attachErr
	}
	return &SessionRef{SessionID: sessionID}, nil
}

func (e *recordingSessionExecutor) RequestTaskStop(_ context.Context, sessionID string, reason StopReason) error {
	e.requestStopCalls = append(e.requestStopCalls, sessionStopCall{SessionID: sessionID, Reason: reason})
	return e.requestStopErr
}

func (e *recordingSessionExecutor) ForceTaskStop(_ context.Context, sessionID string, reason StopReason) error {
	e.forceStopCalls = append(e.forceStopCalls, sessionStopCall{SessionID: sessionID, Reason: reason})
	return e.forceStopErr
}

func newInMemoryManagerStore() *inMemoryManagerStore {
	return &inMemoryManagerStore{
		tasks:            make(map[string]Task),
		dependencies:     make(map[string]map[string]TaskDependency),
		runs:             make(map[string]TaskRun),
		events:           make([]TaskEvent, 0),
		idempotencyByKey: make(map[string]TaskRunIdempotency),
	}
}

func (s *inMemoryManagerStore) CreateTask(_ context.Context, taskRecord Task) error {
	if _, exists := s.tasks[taskRecord.ID]; exists {
		return fmtTestError("%w: duplicate task %q", ErrValidation, taskRecord.ID)
	}
	s.tasks[taskRecord.ID] = cloneTask(taskRecord)
	return nil
}

func (s *inMemoryManagerStore) UpdateTask(_ context.Context, taskRecord Task) error {
	if _, exists := s.tasks[taskRecord.ID]; !exists {
		return ErrTaskNotFound
	}
	s.tasks[taskRecord.ID] = cloneTask(taskRecord)
	return nil
}

func (s *inMemoryManagerStore) GetTask(_ context.Context, id string) (Task, error) {
	record, ok := s.tasks[strings.TrimSpace(id)]
	if !ok {
		return Task{}, ErrTaskNotFound
	}
	return cloneTask(record), nil
}

func (s *inMemoryManagerStore) ListTasks(_ context.Context, query TaskQuery) ([]TaskSummary, error) {
	if err := query.Validate("task_query"); err != nil {
		return nil, err
	}

	normalized := query
	normalized.Scope = normalized.Scope.Normalize()
	normalized.WorkspaceID = strings.TrimSpace(normalized.WorkspaceID)
	normalized.Status = normalized.Status.Normalize()
	normalized.OwnerKind = normalized.OwnerKind.Normalize()
	normalized.OwnerRef = strings.TrimSpace(normalized.OwnerRef)
	normalized.ParentTaskID = strings.TrimSpace(normalized.ParentTaskID)
	normalized.NetworkChannel = strings.TrimSpace(normalized.NetworkChannel)

	summaries := make([]TaskSummary, 0)
	for _, record := range s.tasks {
		if normalized.Scope.Normalize() != "" && record.Scope != normalized.Scope {
			continue
		}
		if normalized.WorkspaceID != "" && record.WorkspaceID != normalized.WorkspaceID {
			continue
		}
		if normalized.Status.Normalize() != "" && record.Status != normalized.Status {
			continue
		}
		if normalized.OwnerKind.Normalize() != "" {
			if record.Owner == nil || record.Owner.Kind != normalized.OwnerKind {
				continue
			}
		}
		if normalized.OwnerRef != "" {
			if record.Owner == nil || record.Owner.Ref != normalized.OwnerRef {
				continue
			}
		}
		if normalized.ParentTaskID != "" && record.ParentTaskID != normalized.ParentTaskID {
			continue
		}
		if normalized.NetworkChannel != "" && record.NetworkChannel != normalized.NetworkChannel {
			continue
		}
		summaries = append(summaries, TaskSummary{
			ID:             record.ID,
			Identifier:     record.Identifier,
			Scope:          record.Scope,
			WorkspaceID:    record.WorkspaceID,
			ParentTaskID:   record.ParentTaskID,
			NetworkChannel: record.NetworkChannel,
			Title:          record.Title,
			Status:         record.Status,
			Owner:          cloneOwnership(record.Owner),
			CreatedBy:      record.CreatedBy,
			Origin:         record.Origin,
			CreatedAt:      record.CreatedAt,
			UpdatedAt:      record.UpdatedAt,
			ClosedAt:       record.ClosedAt,
		})
	}

	sort.Slice(summaries, func(i int, j int) bool {
		return summaries[i].ID < summaries[j].ID
	})
	if normalized.Limit > 0 && len(summaries) > normalized.Limit {
		return append([]TaskSummary(nil), summaries[:normalized.Limit]...), nil
	}
	return append([]TaskSummary(nil), summaries...), nil
}

func (s *inMemoryManagerStore) CountDirectChildren(_ context.Context, parentTaskID string) (int, error) {
	count := 0
	for _, record := range s.tasks {
		if record.ParentTaskID == strings.TrimSpace(parentTaskID) {
			count++
		}
	}
	return count, nil
}

func (s *inMemoryManagerStore) CreateDependency(_ context.Context, dependency TaskDependency) error {
	if _, ok := s.tasks[dependency.TaskID]; !ok {
		return ErrTaskNotFound
	}
	if _, ok := s.tasks[dependency.DependsOnTaskID]; !ok {
		return ErrTaskNotFound
	}
	if s.dependencies[dependency.TaskID] == nil {
		s.dependencies[dependency.TaskID] = make(map[string]TaskDependency)
	}
	s.dependencies[dependency.TaskID][dependency.DependsOnTaskID] = dependency
	return nil
}

func (s *inMemoryManagerStore) DeleteDependency(_ context.Context, taskID string, dependsOnID string) error {
	taskDeps := s.dependencies[strings.TrimSpace(taskID)]
	if taskDeps == nil {
		return ErrTaskDependencyNotFound
	}
	if _, ok := taskDeps[strings.TrimSpace(dependsOnID)]; !ok {
		return ErrTaskDependencyNotFound
	}
	delete(taskDeps, strings.TrimSpace(dependsOnID))
	return nil
}

func (s *inMemoryManagerStore) ListDependencies(_ context.Context, taskID string) ([]TaskDependency, error) {
	taskDeps := s.dependencies[strings.TrimSpace(taskID)]
	if len(taskDeps) == 0 {
		return nil, nil
	}

	dependencies := make([]TaskDependency, 0, len(taskDeps))
	for _, dependency := range taskDeps {
		dependencies = append(dependencies, dependency)
	}
	sort.Slice(dependencies, func(i int, j int) bool {
		return dependencies[i].DependsOnTaskID < dependencies[j].DependsOnTaskID
	})
	return dependencies, nil
}

func (s *inMemoryManagerStore) ListDependents(_ context.Context, dependsOnTaskID string) ([]TaskDependency, error) {
	dependents := make([]TaskDependency, 0)
	for _, taskDeps := range s.dependencies {
		if dependency, ok := taskDeps[strings.TrimSpace(dependsOnTaskID)]; ok {
			dependents = append(dependents, dependency)
		}
	}
	sort.Slice(dependents, func(i int, j int) bool {
		return dependents[i].TaskID < dependents[j].TaskID
	})
	return dependents, nil
}

func (s *inMemoryManagerStore) CountDependencies(_ context.Context, taskID string) (int, error) {
	return len(s.dependencies[strings.TrimSpace(taskID)]), nil
}

func (s *inMemoryManagerStore) HasDependencyPath(_ context.Context, fromTaskID string, toTaskID string) (bool, error) {
	visited := make(map[string]struct{})
	var walk func(string) bool
	walk = func(current string) bool {
		if current == strings.TrimSpace(toTaskID) {
			return true
		}
		if _, seen := visited[current]; seen {
			return false
		}
		visited[current] = struct{}{}
		for _, dependency := range s.dependencies[current] {
			if walk(dependency.DependsOnTaskID) {
				return true
			}
		}
		return false
	}
	return walk(strings.TrimSpace(fromTaskID)), nil
}

func (s *inMemoryManagerStore) CreateTaskRun(_ context.Context, run TaskRun) error {
	s.runs[run.ID] = cloneTaskRun(run)
	return nil
}

func (s *inMemoryManagerStore) UpdateTaskRun(_ context.Context, run TaskRun) error {
	s.runs[run.ID] = cloneTaskRun(run)
	return nil
}

func (s *inMemoryManagerStore) GetTaskRun(_ context.Context, id string) (TaskRun, error) {
	run, ok := s.runs[strings.TrimSpace(id)]
	if !ok {
		return TaskRun{}, ErrTaskRunNotFound
	}
	return cloneTaskRun(run), nil
}

func (s *inMemoryManagerStore) ListTaskRuns(_ context.Context, query TaskRunQuery) ([]TaskRun, error) {
	if err := query.Validate("task_run_query"); err != nil {
		return nil, err
	}

	normalized := query
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.Status = normalized.Status.Normalize()
	normalized.SessionID = strings.TrimSpace(normalized.SessionID)

	runs := make([]TaskRun, 0)
	for _, run := range s.runs {
		if normalized.TaskID != "" && run.TaskID != normalized.TaskID {
			continue
		}
		if normalized.Status.Normalize() != "" && run.Status != normalized.Status {
			continue
		}
		if normalized.SessionID != "" && run.SessionID != normalized.SessionID {
			continue
		}
		runs = append(runs, cloneTaskRun(run))
	}
	sort.Slice(runs, func(i int, j int) bool {
		return runs[i].ID < runs[j].ID
	})
	if normalized.Limit > 0 && len(runs) > normalized.Limit {
		return append([]TaskRun(nil), runs[:normalized.Limit]...), nil
	}
	return append([]TaskRun(nil), runs...), nil
}

func (s *inMemoryManagerStore) ListTaskRunsByStatus(_ context.Context, statuses []TaskRunStatus) ([]TaskRun, error) {
	allowed := make(map[TaskRunStatus]struct{}, len(statuses))
	for _, status := range statuses {
		allowed[status.Normalize()] = struct{}{}
	}
	runs := make([]TaskRun, 0)
	for _, run := range s.runs {
		if _, ok := allowed[run.Status.Normalize()]; ok {
			runs = append(runs, cloneTaskRun(run))
		}
	}
	return runs, nil
}

func (s *inMemoryManagerStore) CountActiveSessionBindings(_ context.Context, sessionID string) (int, error) {
	count := 0
	for _, run := range s.runs {
		if run.SessionID == strings.TrimSpace(sessionID) && run.EndedAt.IsZero() {
			count++
		}
	}
	return count, nil
}

func (s *inMemoryManagerStore) CreateTaskEvent(_ context.Context, event TaskEvent) error {
	if _, ok := s.tasks[event.TaskID]; !ok {
		return ErrTaskNotFound
	}
	s.events = append(s.events, event)
	sort.Slice(s.events, func(i int, j int) bool {
		if s.events[i].Timestamp.Equal(s.events[j].Timestamp) {
			return s.events[i].ID > s.events[j].ID
		}
		return s.events[i].Timestamp.After(s.events[j].Timestamp)
	})
	return nil
}

func (s *inMemoryManagerStore) ListTaskEvents(_ context.Context, query TaskEventQuery) ([]TaskEvent, error) {
	if err := query.Validate("task_event_query"); err != nil {
		return nil, err
	}
	events := make([]TaskEvent, 0)
	for _, event := range s.events {
		if query.TaskID != "" && event.TaskID != strings.TrimSpace(query.TaskID) {
			continue
		}
		if query.RunID != "" && event.RunID != strings.TrimSpace(query.RunID) {
			continue
		}
		if query.EventType != "" && event.EventType != strings.TrimSpace(query.EventType) {
			continue
		}
		events = append(events, event)
	}
	if query.Limit > 0 && len(events) > query.Limit {
		return append([]TaskEvent(nil), events[:query.Limit]...), nil
	}
	return append([]TaskEvent(nil), events...), nil
}

func (s *inMemoryManagerStore) GetTaskRunByIdempotencyKey(_ context.Context, key string, origin Origin) (TaskRun, error) {
	record, ok := s.idempotencyByKey[idempotencyKey(origin, key)]
	if !ok {
		return TaskRun{}, ErrTaskRunIdempotencyNotFound
	}
	return s.GetTaskRun(context.Background(), record.RunID)
}

func (s *inMemoryManagerStore) SaveTaskRunIdempotency(_ context.Context, record TaskRunIdempotency) error {
	s.idempotencyByKey[idempotencyKey(record.Origin, record.IdempotencyKey)] = record
	return nil
}

func TestDeriveActorContextsForSupportedSurfaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		derive  func() (ActorContext, error)
		want    ActorContext
		wantErr error
	}{
		{
			name: "human cli",
			derive: func() (ActorContext, error) {
				return DeriveHumanActorContext("user-1", OriginKindCLI, "agh task create")
			},
			want: ActorContext{
				Actor:     ActorIdentity{Kind: ActorKindHuman, Ref: "user-1"},
				Origin:    Origin{Kind: OriginKindCLI, Ref: "agh task create"},
				Authority: FullAccessAuthority(),
			},
		},
		{
			name: "agent session",
			derive: func() (ActorContext, error) {
				return DeriveAgentSessionActorContext("sess-1")
			},
			want: ActorContext{
				Actor:     ActorIdentity{Kind: ActorKindAgentSession, Ref: "sess-1"},
				Origin:    Origin{Kind: OriginKindAgentSession, Ref: "sess-1"},
				Authority: FullAccessAuthority(),
			},
		},
		{
			name: "automation",
			derive: func() (ActorContext, error) {
				return DeriveAutomationActorContext("rule:nightly", "")
			},
			want: ActorContext{
				Actor:     ActorIdentity{Kind: ActorKindAutomation, Ref: "rule:nightly"},
				Origin:    Origin{Kind: OriginKindAutomation, Ref: "rule:nightly"},
				Authority: FullAccessAuthority(),
			},
		},
		{
			name: "extension",
			derive: func() (ActorContext, error) {
				return DeriveExtensionActorContext("ext.telegram", "cap.task.write")
			},
			want: ActorContext{
				Actor:     ActorIdentity{Kind: ActorKindExtension, Ref: "ext.telegram"},
				Origin:    Origin{Kind: OriginKindExtension, Ref: "cap.task.write"},
				Authority: FullAccessAuthority(),
			},
		},
		{
			name: "network peer",
			derive: func() (ActorContext, error) {
				return DeriveNetworkPeerActorContext("peer:finance", "peer:finance/ops")
			},
			want: ActorContext{
				Actor:     ActorIdentity{Kind: ActorKindNetworkPeer, Ref: "peer:finance"},
				Origin:    Origin{Kind: OriginKindNetwork, Ref: "peer:finance/ops"},
				Authority: FullAccessAuthority(),
			},
		},
		{
			name: "human invalid origin",
			derive: func() (ActorContext, error) {
				return DeriveHumanActorContext("user-1", OriginKindAutomation, "rule:nightly")
			},
			wantErr: ErrValidation,
		},
		{
			name: "manual actor origin mismatch rejected",
			derive: func() (ActorContext, error) {
				ctx := validActorContext()
				ctx.Actor.Kind = ActorKindHuman
				ctx.Origin.Kind = OriginKindAutomation
				return ctx, ctx.Validate()
			},
			wantErr: ErrValidation,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.derive()
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("derive() error = %v", err)
				}
				if got != tt.want {
					t.Fatalf("derive() = %#v, want %#v", got, tt.want)
				}
				return
			}
			if err == nil {
				t.Fatal("derive() error = nil, want non-nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("derive() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestManagerCreateTaskUsesTrustedActorContext(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor, err := DeriveAgentSessionActorContext("sess-123")
	if err != nil {
		t.Fatalf("DeriveAgentSessionActorContext() error = %v", err)
	}

	created, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Investigate task manager",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	if got, want := created.CreatedBy, actor.Actor; got != want {
		t.Fatalf("created.CreatedBy = %#v, want %#v", got, want)
	}
	if got, want := created.Origin, actor.Origin; got != want {
		t.Fatalf("created.Origin = %#v, want %#v", got, want)
	}
	if created.Owner != nil {
		t.Fatalf("created.Owner = %#v, want nil", created.Owner)
	}
	if got, want := created.Status, TaskStatusReady; got != want {
		t.Fatalf("created.Status = %q, want %q", got, want)
	}

	events, err := store.ListTaskEvents(context.Background(), TaskEventQuery{TaskID: created.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if got, want := events[0].EventType, taskEventCreated; got != want {
		t.Fatalf("events[0].EventType = %q, want %q", got, want)
	}
	if got, want := events[0].Actor, actor.Actor; got != want {
		t.Fatalf("events[0].Actor = %#v, want %#v", got, want)
	}
	if got, want := events[0].Origin, actor.Origin; got != want {
		t.Fatalf("events[0].Origin = %#v, want %#v", got, want)
	}
}

func TestManagerCreateTaskEnforcesScopeAuthority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		spec  CreateTask
		actor ActorContext
	}{
		{
			name: "global create denied without global authority",
			spec: CreateTask{Scope: ScopeGlobal, Title: "Global task"},
			actor: ActorContext{
				Actor:     ActorIdentity{Kind: ActorKindHuman, Ref: "user-1"},
				Origin:    Origin{Kind: OriginKindCLI, Ref: "agh task create"},
				Authority: Authority{Read: true, Write: true, CreateWorkspace: true},
			},
		},
		{
			name: "workspace create denied without workspace authority",
			spec: CreateTask{Scope: ScopeWorkspace, WorkspaceID: "ws-1", Title: "Workspace task"},
			actor: ActorContext{
				Actor:     ActorIdentity{Kind: ActorKindHuman, Ref: "user-1"},
				Origin:    Origin{Kind: OriginKindCLI, Ref: "agh task create"},
				Authority: Authority{Read: true, Write: true, CreateGlobal: true},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			manager := newTaskManagerForTest(t, newInMemoryManagerStore())
			_, err := manager.CreateTask(context.Background(), tt.spec, tt.actor)
			if !errors.Is(err, ErrPermissionDenied) {
				t.Fatalf("CreateTask() error = %v, want %v", err, ErrPermissionDenied)
			}
		})
	}
}

func TestManagerUpdateTaskAllowsMutableOwnershipAndChannelFields(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor := validActorContext()

	created, err := manager.CreateTask(context.Background(), CreateTask{
		Scope:       ScopeWorkspace,
		WorkspaceID: "ws-1",
		Title:       "Queue task",
		Description: "Unassigned",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	title := "Claimed task"
	description := "Assigned to triage"
	channel := "ops"
	metadata := json.RawMessage(`{"priority":"high"}`)
	updated, err := manager.UpdateTask(context.Background(), created.ID, TaskPatch{
		Title:          &title,
		Description:    &description,
		NetworkChannel: &channel,
		Owner:          &Ownership{Kind: OwnerKindPool, Ref: "triage"},
		Metadata:       &metadata,
	}, actor)
	if err != nil {
		t.Fatalf("UpdateTask(assign) error = %v", err)
	}

	if got, want := updated.Title, title; got != want {
		t.Fatalf("updated.Title = %q, want %q", got, want)
	}
	if got, want := updated.Description, description; got != want {
		t.Fatalf("updated.Description = %q, want %q", got, want)
	}
	if got, want := updated.NetworkChannel, channel; got != want {
		t.Fatalf("updated.NetworkChannel = %q, want %q", got, want)
	}
	if updated.Owner == nil || updated.Owner.Kind != OwnerKindPool || updated.Owner.Ref != "triage" {
		t.Fatalf("updated.Owner = %#v, want pool/triage", updated.Owner)
	}
	if got, want := updated.Scope, created.Scope; got != want {
		t.Fatalf("updated.Scope = %q, want %q", got, want)
	}
	if got, want := updated.WorkspaceID, created.WorkspaceID; got != want {
		t.Fatalf("updated.WorkspaceID = %q, want %q", got, want)
	}
	if got, want := updated.ParentTaskID, created.ParentTaskID; got != want {
		t.Fatalf("updated.ParentTaskID = %q, want %q", got, want)
	}
	if got, want := updated.CreatedBy, created.CreatedBy; got != want {
		t.Fatalf("updated.CreatedBy = %#v, want %#v", got, want)
	}
	if got, want := updated.Origin, created.Origin; got != want {
		t.Fatalf("updated.Origin = %#v, want %#v", got, want)
	}

	clearChannel := ""
	cleared, err := manager.UpdateTask(context.Background(), created.ID, TaskPatch{
		NetworkChannel: &clearChannel,
		ClearOwner:     true,
	}, actor)
	if err != nil {
		t.Fatalf("UpdateTask(clear) error = %v", err)
	}
	if cleared.Owner != nil {
		t.Fatalf("cleared.Owner = %#v, want nil", cleared.Owner)
	}
	if got := cleared.NetworkChannel; got != "" {
		t.Fatalf("cleared.NetworkChannel = %q, want empty", got)
	}
}

func TestManagerUpdateTaskPreservesCanonicalBlockedStatus(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor := validActorContext()

	taskA, err := manager.CreateTask(context.Background(), CreateTask{Scope: ScopeGlobal, Title: "task A"}, actor)
	if err != nil {
		t.Fatalf("CreateTask(taskA) error = %v", err)
	}
	taskB, err := manager.CreateTask(context.Background(), CreateTask{Scope: ScopeGlobal, Title: "task B"}, actor)
	if err != nil {
		t.Fatalf("CreateTask(taskB) error = %v", err)
	}
	if err := manager.AddDependency(context.Background(), AddDependency{
		TaskID:          taskA.ID,
		DependsOnTaskID: taskB.ID,
		Kind:            DependencyKindBlocks,
	}, actor); err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}

	title := "task A renamed"
	updated, err := manager.UpdateTask(context.Background(), taskA.ID, TaskPatch{
		Title: &title,
	}, actor)
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if got, want := updated.Status, TaskStatusBlocked; got != want {
		t.Fatalf("updated.Status = %q, want %q", got, want)
	}
}

func TestManagerCreateChildTaskEnforcesParentRulesAndEmitsAudit(t *testing.T) {
	t.Parallel()

	t.Run("global parent allows workspace child and emits parent event", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		manager := newTaskManagerForTest(t, store)
		actor := validActorContext()

		parent, err := manager.CreateTask(context.Background(), CreateTask{
			Scope: ScopeGlobal,
			Title: "Coordinator",
		}, actor)
		if err != nil {
			t.Fatalf("CreateTask(parent) error = %v", err)
		}

		child, err := manager.CreateChildTask(context.Background(), parent.ID, CreateTask{
			Scope:       ScopeWorkspace,
			WorkspaceID: "ws-1",
			Title:       "Workspace child",
		}, actor)
		if err != nil {
			t.Fatalf("CreateChildTask() error = %v", err)
		}
		if got, want := child.ParentTaskID, parent.ID; got != want {
			t.Fatalf("child.ParentTaskID = %q, want %q", got, want)
		}

		events, err := store.ListTaskEvents(context.Background(), TaskEventQuery{TaskID: parent.ID})
		if err != nil {
			t.Fatalf("ListTaskEvents(parent) error = %v", err)
		}
		if len(events) != 2 {
			t.Fatalf("len(parent events) = %d, want 2", len(events))
		}
		if got, want := events[0].EventType, taskEventChildCreated; got != want {
			t.Fatalf("parent event type = %q, want %q", got, want)
		}
	})

	t.Run("workspace parent rejects cross scope or cross workspace children", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		manager := newTaskManagerForTest(t, store)
		actor := validActorContext()

		parent, err := manager.CreateTask(context.Background(), CreateTask{
			Scope:       ScopeWorkspace,
			WorkspaceID: "ws-parent",
			Title:       "Workspace parent",
		}, actor)
		if err != nil {
			t.Fatalf("CreateTask(parent) error = %v", err)
		}

		_, err = manager.CreateChildTask(context.Background(), parent.ID, CreateTask{
			Scope: ScopeGlobal,
			Title: "Invalid global child",
		}, actor)
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("CreateChildTask(global child) error = %v, want %v", err, ErrValidation)
		}

		_, err = manager.CreateChildTask(context.Background(), parent.ID, CreateTask{
			Scope:       ScopeWorkspace,
			WorkspaceID: "ws-other",
			Title:       "Wrong workspace child",
		}, actor)
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("CreateChildTask(other workspace child) error = %v, want %v", err, ErrValidation)
		}
	})
}

func TestManagerAddAndRemoveDependencyReconcileStatusAndEvents(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor := validActorContext()

	taskA, err := manager.CreateTask(context.Background(), CreateTask{Scope: ScopeGlobal, Title: "task A"}, actor)
	if err != nil {
		t.Fatalf("CreateTask(taskA) error = %v", err)
	}
	taskB, err := manager.CreateTask(context.Background(), CreateTask{Scope: ScopeGlobal, Title: "task B"}, actor)
	if err != nil {
		t.Fatalf("CreateTask(taskB) error = %v", err)
	}

	if err := manager.AddDependency(context.Background(), AddDependency{
		TaskID:          taskA.ID,
		DependsOnTaskID: taskB.ID,
		Kind:            DependencyKindBlocks,
	}, actor); err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}

	blocked, err := store.GetTask(context.Background(), taskA.ID)
	if err != nil {
		t.Fatalf("GetTask(blocked) error = %v", err)
	}
	if got, want := blocked.Status, TaskStatusBlocked; got != want {
		t.Fatalf("blocked.Status = %q, want %q", got, want)
	}

	if err := manager.RemoveDependency(context.Background(), taskA.ID, taskB.ID, actor); err != nil {
		t.Fatalf("RemoveDependency() error = %v", err)
	}

	ready, err := store.GetTask(context.Background(), taskA.ID)
	if err != nil {
		t.Fatalf("GetTask(ready) error = %v", err)
	}
	if got, want := ready.Status, TaskStatusReady; got != want {
		t.Fatalf("ready.Status = %q, want %q", got, want)
	}

	events, err := store.ListTaskEvents(context.Background(), TaskEventQuery{TaskID: taskA.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents(taskA) error = %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("len(taskA events) = %d, want 3", len(events))
	}
	if got, want := events[0].EventType, taskEventDependencyRemoved; got != want {
		t.Fatalf("events[0].EventType = %q, want %q", got, want)
	}
	if got, want := events[1].EventType, taskEventDependencyAdded; got != want {
		t.Fatalf("events[1].EventType = %q, want %q", got, want)
	}
}

func TestManagerGetAndListTasksRequireReadAuthorityAndBuildView(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor := validActorContext()

	parent, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Parent task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}
	child, err := manager.CreateChildTask(context.Background(), parent.ID, CreateTask{
		Scope: ScopeGlobal,
		Title: "Child task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateChildTask() error = %v", err)
	}
	dependency, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Dependency",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(dependency) error = %v", err)
	}
	if err := manager.AddDependency(context.Background(), AddDependency{
		TaskID:          child.ID,
		DependsOnTaskID: dependency.ID,
		Kind:            DependencyKindBlocks,
	}, actor); err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}

	store.runs["run-active"] = TaskRun{
		ID:       "run-active",
		TaskID:   child.ID,
		Status:   TaskRunStatusRunning,
		Attempt:  1,
		Origin:   Origin{Kind: OriginKindAutomation, Ref: "rule:nightly"},
		QueuedAt: time.Date(2026, 4, 14, 13, 0, 0, 0, time.UTC),
	}

	view, err := manager.GetTask(context.Background(), child.ID, actor)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got, want := view.Task.Status, TaskStatusInProgress; got != want {
		t.Fatalf("view.Task.Status = %q, want %q", got, want)
	}
	if len(view.Dependencies) != 1 {
		t.Fatalf("len(view.Dependencies) = %d, want 1", len(view.Dependencies))
	}
	if len(view.Runs) != 1 {
		t.Fatalf("len(view.Runs) = %d, want 1", len(view.Runs))
	}
	if len(view.Events) < 2 {
		t.Fatalf("len(view.Events) = %d, want at least 2", len(view.Events))
	}

	summaries, err := manager.ListTasks(context.Background(), TaskQuery{ParentTaskID: parent.ID}, actor)
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(summaries) != 1 || summaries[0].ID != child.ID {
		t.Fatalf("ListTasks(parent filter) = %#v, want only child %q", summaries, child.ID)
	}

	noRead := actor
	noRead.Authority.Read = false
	if _, err := manager.GetTask(context.Background(), child.ID, noRead); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("GetTask(no read) error = %v, want %v", err, ErrPermissionDenied)
	}
	if _, err := manager.ListTasks(context.Background(), TaskQuery{}, noRead); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("ListTasks(no read) error = %v, want %v", err, ErrPermissionDenied)
	}
}

func TestManagerRunLifecycleRejectsInvalidTransitions(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	executor := &recordingSessionExecutor{}
	manager := newTaskManagerForTestWithOptions(t, store, WithSessionExecutor(executor))
	actor := validActorContext()

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Lifecycle transitions",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	queuedRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}

	if _, err := manager.CompleteRun(context.Background(), queuedRun.ID, RunResult{
		Value: json.RawMessage(`{"ok":true}`),
	}, actor); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("CompleteRun(queued) error = %v, want %v", err, ErrInvalidStatusTransition)
	}

	claimedRun, err := manager.ClaimRun(context.Background(), queuedRun.ID, ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun() error = %v", err)
	}
	runningRun, err := manager.StartRun(context.Background(), claimedRun.ID, StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun() error = %v", err)
	}

	if _, err := manager.ClaimRun(context.Background(), runningRun.ID, ClaimRun{}, actor); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("ClaimRun(running) error = %v, want %v", err, ErrInvalidStatusTransition)
	}
}

func TestManagerTaskReconciliationAcrossDependenciesAndRuns(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	executor := &recordingSessionExecutor{}
	manager := newTaskManagerForTestWithOptions(t, store, WithSessionExecutor(executor))
	actor := validActorContext()

	blocker, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Blocking task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(blocker) error = %v", err)
	}
	target, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Target task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(target) error = %v", err)
	}

	if err := manager.AddDependency(context.Background(), AddDependency{
		TaskID:          target.ID,
		DependsOnTaskID: blocker.ID,
		Kind:            DependencyKindBlocks,
	}, actor); err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}
	if got, want := store.tasks[target.ID].Status, TaskStatusBlocked; got != want {
		t.Fatalf("target.Status after blocker add = %q, want %q", got, want)
	}

	blockerRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: blocker.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(blocker) error = %v", err)
	}
	blockerRun, err = manager.ClaimRun(context.Background(), blockerRun.ID, ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(blocker) error = %v", err)
	}
	blockerRun, err = manager.StartRun(context.Background(), blockerRun.ID, StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun(blocker) error = %v", err)
	}
	if _, err := manager.CompleteRun(context.Background(), blockerRun.ID, RunResult{
		Value: json.RawMessage(`{"state":"done"}`),
	}, actor); err != nil {
		t.Fatalf("CompleteRun(blocker) error = %v", err)
	}
	if got, want := store.tasks[blocker.ID].Status, TaskStatusCompleted; got != want {
		t.Fatalf("blocker.Status after complete = %q, want %q", got, want)
	}
	if got, want := store.tasks[target.ID].Status, TaskStatusReady; got != want {
		t.Fatalf("target.Status after blocker complete = %q, want %q", got, want)
	}

	targetRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: target.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(target) error = %v", err)
	}
	targetRun, err = manager.ClaimRun(context.Background(), targetRun.ID, ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(target) error = %v", err)
	}
	targetRun, err = manager.StartRun(context.Background(), targetRun.ID, StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun(target) error = %v", err)
	}
	if got, want := store.tasks[target.ID].Status, TaskStatusInProgress; got != want {
		t.Fatalf("target.Status after start = %q, want %q", got, want)
	}
	if _, err := manager.CompleteRun(context.Background(), targetRun.ID, RunResult{
		Value: json.RawMessage(`{"state":"done"}`),
	}, actor); err != nil {
		t.Fatalf("CompleteRun(target) error = %v", err)
	}
	if got, want := store.tasks[target.ID].Status, TaskStatusCompleted; got != want {
		t.Fatalf("target.Status after complete = %q, want %q", got, want)
	}

	failedTask, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Failure task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(failedTask) error = %v", err)
	}
	failedRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: failedTask.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(failedTask) error = %v", err)
	}
	failedRun, err = manager.ClaimRun(context.Background(), failedRun.ID, ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(failedTask) error = %v", err)
	}
	failedRun, err = manager.StartRun(context.Background(), failedRun.ID, StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun(failedTask) error = %v", err)
	}
	if _, err := manager.FailRun(context.Background(), failedRun.ID, RunFailure{
		Error: "boom",
	}, actor); err != nil {
		t.Fatalf("FailRun() error = %v", err)
	}
	if got, want := store.tasks[failedTask.ID].Status, TaskStatusFailed; got != want {
		t.Fatalf("failedTask.Status = %q, want %q", got, want)
	}

	cancelledTask, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Cancelled task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(cancelledTask) error = %v", err)
	}
	cancelledRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: cancelledTask.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(cancelledTask) error = %v", err)
	}
	cancelledRun, err = manager.ClaimRun(context.Background(), cancelledRun.ID, ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(cancelledTask) error = %v", err)
	}
	cancelledRun, err = manager.StartRun(context.Background(), cancelledRun.ID, StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun(cancelledTask) error = %v", err)
	}
	if _, err := manager.CancelRun(context.Background(), cancelledRun.ID, CancelRun{
		Reason: "stop",
	}, actor); err != nil {
		t.Fatalf("CancelRun() error = %v", err)
	}
	if got, want := store.tasks[cancelledTask.ID].Status, TaskStatusCancelled; got != want {
		t.Fatalf("cancelledTask.Status = %q, want %q", got, want)
	}
}

func TestManagerCancelTaskPropagatesAcrossTree(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	executor := &recordingSessionExecutor{}
	manager := newTaskManagerForTestWithOptions(
		t,
		store,
		WithSessionExecutor(executor),
		WithCancelGracePeriod(0),
	)
	actor := validActorContext()

	parent, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Parent task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}
	queuedChild, err := manager.CreateChildTask(context.Background(), parent.ID, CreateTask{
		Scope: ScopeGlobal,
		Title: "Queued child",
	}, actor)
	if err != nil {
		t.Fatalf("CreateChildTask(queued child) error = %v", err)
	}
	activeChild, err := manager.CreateChildTask(context.Background(), parent.ID, CreateTask{
		Scope: ScopeGlobal,
		Title: "Active child",
	}, actor)
	if err != nil {
		t.Fatalf("CreateChildTask(active child) error = %v", err)
	}

	queuedRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: queuedChild.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(queued child) error = %v", err)
	}
	activeRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: activeChild.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(active child) error = %v", err)
	}
	activeRun, err = manager.ClaimRun(context.Background(), activeRun.ID, ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(active child) error = %v", err)
	}
	activeRun, err = manager.StartRun(context.Background(), activeRun.ID, StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun(active child) error = %v", err)
	}

	cancelledParent, err := manager.CancelTask(context.Background(), parent.ID, CancelTask{
		Reason: "parent requested stop",
	}, actor)
	if err != nil {
		t.Fatalf("CancelTask() error = %v", err)
	}
	if got, want := cancelledParent.Status, TaskStatusCancelled; got != want {
		t.Fatalf("cancelledParent.Status = %q, want %q", got, want)
	}
	if got, want := store.tasks[parent.ID].Status, TaskStatusCancelled; got != want {
		t.Fatalf("parent.Status = %q, want %q", got, want)
	}
	if got, want := store.tasks[queuedChild.ID].Status, TaskStatusCancelled; got != want {
		t.Fatalf("queuedChild.Status = %q, want %q", got, want)
	}
	if got, want := store.tasks[activeChild.ID].Status, TaskStatusCancelled; got != want {
		t.Fatalf("activeChild.Status = %q, want %q", got, want)
	}
	if got, want := store.runs[queuedRun.ID].Status, TaskRunStatusCancelled; got != want {
		t.Fatalf("queuedRun.Status = %q, want %q", got, want)
	}
	if got, want := store.runs[activeRun.ID].Status, TaskRunStatusCancelled; got != want {
		t.Fatalf("activeRun.Status = %q, want %q", got, want)
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
	if got, want := executor.forceStopCalls[0].Reason, StopReasonCancellation; got != want {
		t.Fatalf("forceStopCalls[0].Reason = %q, want %q", got, want)
	}

	parentEvents, err := store.ListTaskEvents(context.Background(), TaskEventQuery{TaskID: parent.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents(parent) error = %v", err)
	}
	if !containsEventType(parentEvents, taskEventCancelled) {
		t.Fatalf("parent events = %#v, want %q", sortedEventTypes(parentEvents), taskEventCancelled)
	}

	activeChildEvents, err := store.ListTaskEvents(context.Background(), TaskEventQuery{TaskID: activeChild.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents(active child) error = %v", err)
	}
	if !containsEventType(activeChildEvents, taskEventRunCancelled) {
		t.Fatalf("active child events = %#v, want %q", sortedEventTypes(activeChildEvents), taskEventRunCancelled)
	}
}

func TestManagerAttachRunSessionAndRetryLatestRunOutcome(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	executor := &recordingSessionExecutor{}
	manager := newTaskManagerForTestWithOptions(t, store, WithSessionExecutor(executor))
	actor := validActorContext()

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope:          ScopeGlobal,
		Title:          "Attach and retry",
		NetworkChannel: "ops",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	firstRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(first) error = %v", err)
	}
	if got, want := firstRun.NetworkChannel, "ops"; got != want {
		t.Fatalf("firstRun.NetworkChannel = %q, want %q", got, want)
	}
	firstRun, err = manager.ClaimRun(context.Background(), firstRun.ID, ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(first) error = %v", err)
	}
	firstRun, err = manager.StartRun(context.Background(), firstRun.ID, StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun(first) error = %v", err)
	}
	if _, err := manager.CompleteRun(context.Background(), firstRun.ID, RunResult{
		Value: json.RawMessage(`{"result":"ok"}`),
	}, actor); err != nil {
		t.Fatalf("CompleteRun(first) error = %v", err)
	}
	if got, want := store.tasks[taskRecord.ID].Status, TaskStatusCompleted; got != want {
		t.Fatalf("task.Status after first completion = %q, want %q", got, want)
	}

	retryRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{
		TaskID:         taskRecord.ID,
		NetworkChannel: "custom",
	}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(retry) error = %v", err)
	}
	if got, want := retryRun.Attempt, 2; got != want {
		t.Fatalf("retryRun.Attempt = %d, want %d", got, want)
	}
	if got, want := retryRun.NetworkChannel, "custom"; got != want {
		t.Fatalf("retryRun.NetworkChannel = %q, want %q", got, want)
	}
	if got, want := store.tasks[taskRecord.ID].Status, TaskStatusReady; got != want {
		t.Fatalf("task.Status after retry enqueue = %q, want %q", got, want)
	}

	retryRun, err = manager.ClaimRun(context.Background(), retryRun.ID, ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(retry) error = %v", err)
	}
	retryRun, err = manager.AttachRunSession(context.Background(), retryRun.ID, "sess-resume", actor)
	if err != nil {
		t.Fatalf("AttachRunSession() error = %v", err)
	}
	if got, want := retryRun.Status, TaskRunStatusStarting; got != want {
		t.Fatalf("retryRun.Status after attach = %q, want %q", got, want)
	}
	if got, want := retryRun.SessionID, "sess-resume"; got != want {
		t.Fatalf("retryRun.SessionID after attach = %q, want %q", got, want)
	}
	if got, want := store.tasks[taskRecord.ID].Status, TaskStatusInProgress; got != want {
		t.Fatalf("task.Status after attach = %q, want %q", got, want)
	}

	retryRun, err = manager.StartRun(context.Background(), retryRun.ID, StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun(retry) error = %v", err)
	}
	if got, want := len(executor.attachCalls), 1; got != want {
		t.Fatalf("len(attachCalls) = %d, want %d", got, want)
	}
	if got, want := len(executor.startCalls), 1; got != want {
		t.Fatalf("len(startCalls) = %d, want %d", got, want)
	}
	if _, err := manager.AttachRunSession(context.Background(), retryRun.ID, "sess-other", actor); !errors.Is(err, ErrSessionAlreadyBound) {
		t.Fatalf("AttachRunSession(running) error = %v, want %v", err, ErrSessionAlreadyBound)
	}

	if _, err := manager.FailRun(context.Background(), retryRun.ID, RunFailure{
		Error: "resume failed",
	}, actor); err != nil {
		t.Fatalf("FailRun(retry) error = %v", err)
	}
	if got, want := store.tasks[taskRecord.ID].Status, TaskStatusFailed; got != want {
		t.Fatalf("task.Status after retry failure = %q, want %q", got, want)
	}
}

func TestManagerNonHumanIdempotencyAndExecutionGuards(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	automationActor, err := DeriveAutomationActorContext("rule:nightly", "")
	if err != nil {
		t.Fatalf("DeriveAutomationActorContext() error = %v", err)
	}

	taskOne, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Idempotent task one",
	}, automationActor)
	if err != nil {
		t.Fatalf("CreateTask(taskOne) error = %v", err)
	}
	taskTwo, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Idempotent task two",
	}, automationActor)
	if err != nil {
		t.Fatalf("CreateTask(taskTwo) error = %v", err)
	}

	if _, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskOne.ID}, automationActor); !errors.Is(err, ErrValidation) {
		t.Fatalf("EnqueueRun(no idempotency) error = %v, want %v", err, ErrValidation)
	}

	runOne, err := manager.EnqueueRun(context.Background(), EnqueueRun{
		TaskID:         taskOne.ID,
		IdempotencyKey: "idem-1",
	}, automationActor)
	if err != nil {
		t.Fatalf("EnqueueRun(taskOne) error = %v", err)
	}
	runAgain, err := manager.EnqueueRun(context.Background(), EnqueueRun{
		TaskID:         taskOne.ID,
		IdempotencyKey: "idem-1",
	}, automationActor)
	if err != nil {
		t.Fatalf("EnqueueRun(taskOne duplicate) error = %v", err)
	}
	if got, want := runAgain.ID, runOne.ID; got != want {
		t.Fatalf("duplicate enqueue run id = %q, want %q", got, want)
	}
	if got, want := len(store.runs), 1; got != want {
		t.Fatalf("len(store.runs) = %d, want %d", got, want)
	}

	if _, err := manager.EnqueueRun(context.Background(), EnqueueRun{
		TaskID:         taskTwo.ID,
		IdempotencyKey: "idem-1",
	}, automationActor); !errors.Is(err, ErrValidation) {
		t.Fatalf("EnqueueRun(taskTwo duplicate key) error = %v, want %v", err, ErrValidation)
	}

	if _, err := manager.ClaimRun(context.Background(), runOne.ID, ClaimRun{}, automationActor); !errors.Is(err, ErrValidation) {
		t.Fatalf("ClaimRun(no idempotency) error = %v, want %v", err, ErrValidation)
	}
	claimedRun, err := manager.ClaimRun(context.Background(), runOne.ID, ClaimRun{
		IdempotencyKey: "claim-idem",
	}, automationActor)
	if err != nil {
		t.Fatalf("ClaimRun(with idempotency) error = %v", err)
	}
	if _, err := manager.StartRun(context.Background(), claimedRun.ID, StartRun{}, automationActor); !errors.Is(err, ErrValidation) {
		t.Fatalf("StartRun(no idempotency) error = %v, want %v", err, ErrValidation)
	}
	if _, err := manager.StartRun(context.Background(), claimedRun.ID, StartRun{
		IdempotencyKey: "start-idem",
	}, automationActor); !errors.Is(err, ErrValidation) {
		t.Fatalf("StartRun(no session executor) error = %v, want %v", err, ErrValidation)
	}
}

func TestManagerBlockedExecutionAndFailureGuardrails(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	executor := &recordingSessionExecutor{startErr: errors.New("boot failed")}
	manager := newTaskManagerForTestWithOptions(
		t,
		store,
		WithSessionExecutor(executor),
		WithCancelGracePeriod(time.Millisecond),
	)
	actor := validActorContext()

	blocker, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Blocker",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(blocker) error = %v", err)
	}
	target, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Target",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(target) error = %v", err)
	}
	if err := manager.AddDependency(context.Background(), AddDependency{
		TaskID:          target.ID,
		DependsOnTaskID: blocker.ID,
		Kind:            DependencyKindBlocks,
	}, actor); err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}

	blockedRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: target.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(blocked target) error = %v", err)
	}
	if _, err := manager.ClaimRun(context.Background(), blockedRun.ID, ClaimRun{}, actor); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("ClaimRun(blocked target) error = %v, want %v", err, ErrInvalidStatusTransition)
	}

	failingTask, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Failing start task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(failingTask) error = %v", err)
	}
	failingRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: failingTask.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(failingTask) error = %v", err)
	}
	failingRun, err = manager.ClaimRun(context.Background(), failingRun.ID, ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(failingTask) error = %v", err)
	}
	failedRun, err := manager.StartRun(context.Background(), failingRun.ID, StartRun{}, actor)
	if err == nil {
		t.Fatal("StartRun(failingTask) error = nil, want non-nil")
	}
	if failedRun == nil || failedRun.Status != TaskRunStatusFailed {
		t.Fatalf("failedRun = %#v, want failed status", failedRun)
	}
	if got, want := store.tasks[failingTask.ID].Status, TaskStatusFailed; got != want {
		t.Fatalf("failingTask.Status = %q, want %q", got, want)
	}

	completedTask, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Completed task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(completedTask) error = %v", err)
	}
	completedRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: completedTask.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(completedTask) error = %v", err)
	}
	completedRun, err = manager.ClaimRun(context.Background(), completedRun.ID, ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(completedTask) error = %v", err)
	}
	executor.startErr = nil
	completedRun, err = manager.StartRun(context.Background(), completedRun.ID, StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun(completedTask) error = %v", err)
	}
	if _, err := manager.CompleteRun(context.Background(), completedRun.ID, RunResult{
		Value: json.RawMessage(`{"ok":true}`),
	}, actor); err != nil {
		t.Fatalf("CompleteRun(completedTask) error = %v", err)
	}
	if _, err := manager.CancelTask(context.Background(), completedTask.ID, CancelTask{
		Reason: "too late",
	}, actor); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("CancelTask(completedTask) error = %v, want %v", err, ErrInvalidStatusTransition)
	}
}

func TestManagerHelperCoverage(t *testing.T) {
	t.Parallel()

	if !hasOpenRun([]TaskRun{{Status: TaskRunStatusQueued}}) {
		t.Fatal("hasOpenRun(queued) = false, want true")
	}
	if hasOpenRun([]TaskRun{{Status: TaskRunStatusCompleted}}) {
		t.Fatal("hasOpenRun(completed) = true, want false")
	}
	if !runComesAfter(
		TaskRun{ID: "run-2", Attempt: 2, QueuedAt: time.Date(2026, 4, 14, 16, 0, 0, 0, time.UTC)},
		TaskRun{ID: "run-1", Attempt: 1, QueuedAt: time.Date(2026, 4, 14, 15, 0, 0, 0, time.UTC)},
	) {
		t.Fatal("runComesAfter(later, earlier) = false, want true")
	}
	if allowsRunTransition(TaskRunStatusCompleted, TaskRunStatusRunning) {
		t.Fatal("allowsRunTransition(completed, running) = true, want false")
	}

	joined := errorsJoin(nil, ErrValidation)
	if !errors.Is(joined, ErrValidation) {
		t.Fatalf("errorsJoin(nil, ErrValidation) = %v, want ErrValidation", joined)
	}

	executor := &recordingSessionExecutor{}
	manager := newTaskManagerForTestWithOptions(
		t,
		newInMemoryManagerStore(),
		WithSessionExecutor(executor),
		WithCancelGracePeriod(time.Millisecond),
	)

	if err := manager.waitAndForceStopRun(context.Background(), "sess-helper"); err != nil {
		t.Fatalf("waitAndForceStopRun() error = %v", err)
	}
	if len(executor.forceStopCalls) != 1 {
		t.Fatalf("len(forceStopCalls) = %d, want 1", len(executor.forceStopCalls))
	}

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := manager.waitAndForceStopRun(cancelledCtx, "sess-cancelled"); err == nil {
		t.Fatal("waitAndForceStopRun(cancelled) error = nil, want non-nil")
	}
}

func TestManagerStartRunAndAttachErrorBranches(t *testing.T) {
	t.Parallel()

	t.Run("start run fails closed when executor returns nil session ref", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		executor := &recordingSessionExecutor{returnNilStart: true}
		manager := newTaskManagerForTestWithOptions(t, store, WithSessionExecutor(executor))
		actor := validActorContext()

		taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
			Scope: ScopeGlobal,
			Title: "Nil session ref task",
		}, actor)
		if err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		run, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor)
		if err != nil {
			t.Fatalf("EnqueueRun() error = %v", err)
		}
		run, err = manager.ClaimRun(context.Background(), run.ID, ClaimRun{}, actor)
		if err != nil {
			t.Fatalf("ClaimRun() error = %v", err)
		}

		failedRun, err := manager.StartRun(context.Background(), run.ID, StartRun{}, actor)
		if err == nil {
			t.Fatal("StartRun() error = nil, want non-nil")
		}
		if failedRun == nil || failedRun.Status != TaskRunStatusFailed {
			t.Fatalf("failedRun = %#v, want failed run", failedRun)
		}
	})

	t.Run("attach run rejects active session reuse", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		executor := &recordingSessionExecutor{}
		manager := newTaskManagerForTestWithOptions(t, store, WithSessionExecutor(executor))
		actor := validActorContext()

		taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
			Scope: ScopeGlobal,
			Title: "Shared session guard",
		}, actor)
		if err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		runOne, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor)
		if err != nil {
			t.Fatalf("EnqueueRun(runOne) error = %v", err)
		}
		runOne, err = manager.ClaimRun(context.Background(), runOne.ID, ClaimRun{}, actor)
		if err != nil {
			t.Fatalf("ClaimRun(runOne) error = %v", err)
		}
		if _, err := manager.AttachRunSession(context.Background(), runOne.ID, "sess-shared", actor); err != nil {
			t.Fatalf("AttachRunSession(runOne) error = %v", err)
		}

		runTwo, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor)
		if err != nil {
			t.Fatalf("EnqueueRun(runTwo) error = %v", err)
		}
		runTwo, err = manager.ClaimRun(context.Background(), runTwo.ID, ClaimRun{}, actor)
		if err != nil {
			t.Fatalf("ClaimRun(runTwo) error = %v", err)
		}
		if _, err := manager.AttachRunSession(context.Background(), runTwo.ID, "sess-shared", actor); !errors.Is(err, ErrSessionAlreadyBound) {
			t.Fatalf("AttachRunSession(runTwo shared session) error = %v, want %v", err, ErrSessionAlreadyBound)
		}
	})

	t.Run("attach run validates executor state and session id", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		managerWithoutExecutor := newTaskManagerForTest(t, store)
		actor := validActorContext()

		taskRecord, err := managerWithoutExecutor.CreateTask(context.Background(), CreateTask{
			Scope: ScopeGlobal,
			Title: "Attach validation task",
		}, actor)
		if err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		run, err := managerWithoutExecutor.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor)
		if err != nil {
			t.Fatalf("EnqueueRun() error = %v", err)
		}
		run, err = managerWithoutExecutor.ClaimRun(context.Background(), run.ID, ClaimRun{}, actor)
		if err != nil {
			t.Fatalf("ClaimRun() error = %v", err)
		}
		if _, err := managerWithoutExecutor.AttachRunSession(context.Background(), run.ID, "sess-1", actor); !errors.Is(err, ErrValidation) {
			t.Fatalf("AttachRunSession(no executor) error = %v, want %v", err, ErrValidation)
		}

		managerWithExecutor := newTaskManagerForTestWithOptions(t, newInMemoryManagerStore(), WithSessionExecutor(&recordingSessionExecutor{}))
		taskRecord, err = managerWithExecutor.CreateTask(context.Background(), CreateTask{
			Scope: ScopeGlobal,
			Title: "Attach session id validation",
		}, actor)
		if err != nil {
			t.Fatalf("CreateTask(with executor) error = %v", err)
		}
		run, err = managerWithExecutor.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor)
		if err != nil {
			t.Fatalf("EnqueueRun(with executor) error = %v", err)
		}
		if _, err := managerWithExecutor.AttachRunSession(context.Background(), run.ID, "", actor); !errors.Is(err, ErrValidation) {
			t.Fatalf("AttachRunSession(empty session id) error = %v, want %v", err, ErrValidation)
		}
		if _, err := managerWithExecutor.AttachRunSession(context.Background(), run.ID, "sess-2", actor); !errors.Is(err, ErrSessionAttachNotAllowed) {
			t.Fatalf("AttachRunSession(queued run) error = %v, want %v", err, ErrSessionAttachNotAllowed)
		}
	})
}

func TestManagerGetTaskAndFailRunGuardrails(t *testing.T) {
	t.Parallel()

	manager := newTaskManagerForTest(t, newInMemoryManagerStore())
	actor := validActorContext()

	if _, err := manager.GetTask(context.Background(), "", actor); !errors.Is(err, ErrValidation) {
		t.Fatalf("GetTask(empty id) error = %v, want %v", err, ErrValidation)
	}
	if _, err := manager.GetTask(context.Background(), "missing-task", actor); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("GetTask(missing) error = %v, want %v", err, ErrTaskNotFound)
	}

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Queued fail guard",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	run, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	if _, err := manager.FailRun(context.Background(), run.ID, RunFailure{
		Error: "cannot fail queued run",
	}, actor); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("FailRun(queued) error = %v, want %v", err, ErrInvalidStatusTransition)
	}
}

func TestManagerAdditionalBranchCoverage(t *testing.T) {
	t.Parallel()

	t.Run("constructor validates required dependencies", func(t *testing.T) {
		t.Parallel()

		if _, err := NewManager(); err == nil {
			t.Fatal("NewManager() error = nil, want non-nil")
		}
		if _, err := NewManager(WithStore(newInMemoryManagerStore()), WithManagerNow(nil)); err == nil {
			t.Fatal("NewManager(nil clock) error = nil, want non-nil")
		}
		if _, err := NewManager(WithStore(newInMemoryManagerStore()), WithIDGenerator(nil)); err == nil {
			t.Fatal("NewManager(nil generator) error = nil, want non-nil")
		}

		manager, err := NewManager(
			WithStore(newInMemoryManagerStore()),
			WithSessionExecutor(testSessionExecutor{}),
		)
		if err != nil {
			t.Fatalf("NewManager(with session executor) error = %v", err)
		}
		if manager.sessions == nil {
			t.Fatal("manager.sessions = nil, want non-nil")
		}
	})

	t.Run("surface helpers default origin refs", func(t *testing.T) {
		t.Parallel()

		extension, err := DeriveExtensionActorContext("ext-1", "")
		if err != nil {
			t.Fatalf("DeriveExtensionActorContext() error = %v", err)
		}
		if got, want := extension.Origin.Ref, "ext-1"; got != want {
			t.Fatalf("extension.Origin.Ref = %q, want %q", got, want)
		}

		network, err := DeriveNetworkPeerActorContext("peer-1", "")
		if err != nil {
			t.Fatalf("DeriveNetworkPeerActorContext() error = %v", err)
		}
		if got, want := network.Origin.Ref, "peer-1"; got != want {
			t.Fatalf("network.Origin.Ref = %q, want %q", got, want)
		}

		daemon, err := DeriveDaemonActorContext("scheduler", "")
		if err != nil {
			t.Fatalf("DeriveDaemonActorContext() error = %v", err)
		}
		if got, want := daemon.Origin.Ref, "scheduler"; got != want {
			t.Fatalf("daemon.Origin.Ref = %q, want %q", got, want)
		}
	})

	t.Run("task depth detects cycles", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		manager := newTaskManagerForTest(t, store)
		taskA := validTask()
		taskA.ID = "task-a"
		taskA.Status = TaskStatusReady
		taskA.ParentTaskID = "task-b"
		taskB := validTask()
		taskB.ID = "task-b"
		taskB.Status = TaskStatusReady
		taskB.ParentTaskID = "task-a"
		store.tasks[taskA.ID] = taskA
		store.tasks[taskB.ID] = taskB

		_, err := manager.taskDepth(context.Background(), taskA)
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("taskDepth(cycle) error = %v, want %v", err, ErrValidation)
		}
	})

	t.Run("create child rejects mismatched parent field", func(t *testing.T) {
		t.Parallel()

		manager := newTaskManagerForTest(t, newInMemoryManagerStore())
		actor := validActorContext()
		parent, err := manager.CreateTask(context.Background(), CreateTask{
			Scope: ScopeGlobal,
			Title: "Parent",
		}, actor)
		if err != nil {
			t.Fatalf("CreateTask(parent) error = %v", err)
		}

		_, err = manager.CreateChildTask(context.Background(), parent.ID, CreateTask{
			Scope:        ScopeGlobal,
			ParentTaskID: "different-parent",
			Title:        "Child",
		}, actor)
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("CreateChildTask(mismatch) error = %v, want %v", err, ErrValidation)
		}
	})

	t.Run("remove dependency validates ids and nil payload marshals cleanly", func(t *testing.T) {
		t.Parallel()

		manager := newTaskManagerForTest(t, newInMemoryManagerStore())
		actor := validActorContext()
		if err := manager.RemoveDependency(context.Background(), "", "task-b", actor); !errors.Is(err, ErrValidation) {
			t.Fatalf("RemoveDependency(empty task) error = %v, want %v", err, ErrValidation)
		}
		if err := manager.RemoveDependency(context.Background(), "task-a", "", actor); !errors.Is(err, ErrValidation) {
			t.Fatalf("RemoveDependency(empty depends_on) error = %v, want %v", err, ErrValidation)
		}

		raw, err := marshalTaskEventPayload(nil)
		if err != nil {
			t.Fatalf("marshalTaskEventPayload(nil) error = %v", err)
		}
		if raw != nil {
			t.Fatalf("marshalTaskEventPayload(nil) = %q, want nil", string(raw))
		}
	})

	t.Run("ownership helpers cover nil and zero values", func(t *testing.T) {
		t.Parallel()

		if got := normalizeOwnership(nil); got != nil {
			t.Fatalf("normalizeOwnership(nil) = %#v, want nil", got)
		}
		if got := normalizeOwnership(&Ownership{}); got != nil {
			t.Fatalf("normalizeOwnership(zero) = %#v, want nil", got)
		}
		if sameOwnership(nil, &Ownership{Kind: OwnerKindPool, Ref: "triage"}) {
			t.Fatal("sameOwnership(nil, owner) = true, want false")
		}
		if !isTerminalTaskStatus(TaskStatusFailed) {
			t.Fatal("isTerminalTaskStatus(failed) = false, want true")
		}
	})
}

func newTaskManagerForTest(t *testing.T, store Store) *TaskManager {
	t.Helper()
	return newTaskManagerForTestWithOptions(t, store)
}

func newTaskManagerForTestWithOptions(t *testing.T, store Store, extraOpts ...Option) *TaskManager {
	t.Helper()

	now := time.Date(2026, 4, 14, 15, 0, 0, 0, time.UTC)
	counter := 0
	options := []Option{
		WithStore(store),
		WithManagerNow(func() time.Time { return now }),
		WithIDGenerator(func(prefix string) string {
			counter++
			return prefix + "-test-" + strconv.Itoa(counter)
		}),
	}
	options = append(options, extraOpts...)
	manager, err := NewManager(options...)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	return manager
}

func cloneTask(record Task) Task {
	cloned := record
	cloned.Owner = cloneOwnership(record.Owner)
	cloned.Metadata = cloneRawJSON(record.Metadata)
	return cloned
}

func cloneTaskRun(record TaskRun) TaskRun {
	cloned := record
	if record.ClaimedBy != nil {
		claimedBy := *record.ClaimedBy
		cloned.ClaimedBy = &claimedBy
	}
	cloned.Result = cloneRawJSON(record.Result)
	return cloned
}

func sortedEventTypes(events []TaskEvent) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.EventType)
	}
	sort.Strings(types)
	return types
}

func containsEventType(events []TaskEvent, want string) bool {
	for _, event := range events {
		if event.EventType == want {
			return true
		}
	}
	return false
}

func idempotencyKey(origin Origin, key string) string {
	return string(origin.Kind.Normalize()) + "|" + strings.TrimSpace(origin.Ref) + "|" + strings.TrimSpace(key)
}

func fmtTestError(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
