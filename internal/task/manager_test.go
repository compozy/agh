package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	storepkg "github.com/pedronauck/agh/internal/store"
)

type inMemoryManagerStore struct {
	tasks             map[string]Task
	dependencies      map[string]map[string]Dependency
	runs              map[string]Run
	triageStates      map[string]TriageState
	events            []Event
	eventSequenceByID map[string]int64
	nextEventSequence int64
	idempotencyByKey  map[string]RunIdempotency
}

type testSessionExecutor struct{}

func (testSessionExecutor) StartTaskSession(context.Context, *StartTaskSession) (*SessionRef, error) {
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

type testRuntimeViewReader struct {
	sessions map[string]*RunSessionRef
	events   map[string][]storepkg.SessionEvent
	stats    map[string][]storepkg.TokenStats
}

func (r testRuntimeViewReader) GetSession(_ context.Context, sessionID string) (*RunSessionRef, error) {
	session, ok := r.sessions[strings.TrimSpace(sessionID)]
	if !ok || session == nil {
		return nil, ErrTaskRunNotFound
	}
	cloned := *session
	return &cloned, nil
}

func (r testRuntimeViewReader) ListSessionEvents(
	_ context.Context,
	sessionID string,
	_ storepkg.EventQuery,
) ([]storepkg.SessionEvent, error) {
	return append([]storepkg.SessionEvent(nil), r.events[strings.TrimSpace(sessionID)]...), nil
}

func (r testRuntimeViewReader) ListSessionTokenStats(
	_ context.Context,
	sessionID string,
) ([]storepkg.TokenStats, error) {
	return append([]storepkg.TokenStats(nil), r.stats[strings.TrimSpace(sessionID)]...), nil
}

func (e *recordingSessionExecutor) StartTaskSession(_ context.Context, spec *StartTaskSession) (*SessionRef, error) {
	if spec == nil {
		return nil, fmtTestError("%w: start task session spec is required", ErrValidation)
	}
	e.startCalls = append(e.startCalls, *spec)
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

func (e *recordingSessionExecutor) AttachTaskSession(
	_ context.Context,
	runID string,
	sessionID string,
) (*SessionRef, error) {
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
		tasks:             make(map[string]Task),
		dependencies:      make(map[string]map[string]Dependency),
		runs:              make(map[string]Run),
		triageStates:      make(map[string]TriageState),
		events:            make([]Event, 0),
		eventSequenceByID: make(map[string]int64),
		idempotencyByKey:  make(map[string]RunIdempotency),
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

func (s *inMemoryManagerStore) ListTasks(_ context.Context, query Query) ([]Summary, error) {
	if err := query.Validate("task_query"); err != nil {
		return nil, err
	}

	normalized := query
	normalized.Scope = normalized.Scope.Normalize()
	normalized.WorkspaceID = strings.TrimSpace(normalized.WorkspaceID)
	normalized.Status = normalized.Status.Normalize()
	normalized.Priority = normalized.Priority.Normalize()
	normalized.ApprovalState = normalized.ApprovalState.Normalize()
	normalized.OwnerKind = normalized.OwnerKind.Normalize()
	normalized.OwnerRef = strings.TrimSpace(normalized.OwnerRef)
	normalized.ParentTaskID = strings.TrimSpace(normalized.ParentTaskID)
	normalized.NetworkChannel = strings.TrimSpace(normalized.NetworkChannel)
	normalized.Search = strings.ToLower(strings.TrimSpace(normalized.Search))

	summaries := make([]Summary, 0)
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
		if normalized.Priority.Normalize() != "" && record.Priority != normalized.Priority {
			continue
		}
		if normalized.ApprovalState.Normalize() != "" && record.ApprovalState != normalized.ApprovalState {
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
		if normalized.Search != "" {
			title := strings.ToLower(strings.TrimSpace(record.Title))
			identifier := strings.ToLower(strings.TrimSpace(record.Identifier))
			if !strings.Contains(title, normalized.Search) && !strings.Contains(identifier, normalized.Search) {
				continue
			}
		}
		summaries = append(summaries, Summary{
			ID:             record.ID,
			Identifier:     record.Identifier,
			Scope:          record.Scope,
			WorkspaceID:    record.WorkspaceID,
			ParentTaskID:   record.ParentTaskID,
			NetworkChannel: record.NetworkChannel,
			Title:          record.Title,
			Priority:       record.Priority,
			MaxAttempts:    record.MaxAttempts,
			Status:         record.Status,
			ApprovalPolicy: record.ApprovalPolicy,
			ApprovalState:  record.ApprovalState,
			Draft:          record.Status.Normalize() == TaskStatusDraft,
			Owner:          cloneOwnership(record.Owner),
			CreatedBy:      record.CreatedBy,
			Origin:         record.Origin,
			CreatedAt:      record.CreatedAt,
			UpdatedAt:      record.UpdatedAt,
			ClosedAt:       record.ClosedAt,
			LastActivityAt: record.UpdatedAt,
		})
	}

	sort.Slice(summaries, func(i int, j int) bool {
		left := inMemoryTaskLatestActivity(summaries[i], s.runs, s.events)
		right := inMemoryTaskLatestActivity(summaries[j], s.runs, s.events)
		if !left.Equal(right) {
			return left.After(right)
		}
		if !summaries[i].UpdatedAt.Equal(summaries[j].UpdatedAt) {
			return summaries[i].UpdatedAt.After(summaries[j].UpdatedAt)
		}
		if !summaries[i].CreatedAt.Equal(summaries[j].CreatedAt) {
			return summaries[i].CreatedAt.After(summaries[j].CreatedAt)
		}
		return summaries[i].ID > summaries[j].ID
	})
	if normalized.Limit > 0 && len(summaries) > normalized.Limit {
		return append([]Summary(nil), summaries[:normalized.Limit]...), nil
	}
	return append([]Summary(nil), summaries...), nil
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

func (s *inMemoryManagerStore) CreateDependency(_ context.Context, dependency Dependency) error {
	if _, ok := s.tasks[dependency.TaskID]; !ok {
		return ErrTaskNotFound
	}
	if _, ok := s.tasks[dependency.DependsOnTaskID]; !ok {
		return ErrTaskNotFound
	}
	if s.dependencies[dependency.TaskID] == nil {
		s.dependencies[dependency.TaskID] = make(map[string]Dependency)
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

func (s *inMemoryManagerStore) ListDependencies(_ context.Context, taskID string) ([]Dependency, error) {
	taskDeps := s.dependencies[strings.TrimSpace(taskID)]
	if len(taskDeps) == 0 {
		return nil, nil
	}

	dependencies := make([]Dependency, 0, len(taskDeps))
	for _, dependency := range taskDeps {
		dependencies = append(dependencies, dependency)
	}
	sort.Slice(dependencies, func(i int, j int) bool {
		return dependencies[i].DependsOnTaskID < dependencies[j].DependsOnTaskID
	})
	return dependencies, nil
}

func (s *inMemoryManagerStore) ListDependents(_ context.Context, dependsOnTaskID string) ([]Dependency, error) {
	dependents := make([]Dependency, 0)
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

func (s *inMemoryManagerStore) CreateTaskRun(_ context.Context, run Run) error {
	s.runs[run.ID] = cloneTaskRun(run)
	return nil
}

func (s *inMemoryManagerStore) UpdateTaskRun(_ context.Context, run Run) error {
	s.runs[run.ID] = cloneTaskRun(run)
	return nil
}

func (s *inMemoryManagerStore) GetTaskRun(_ context.Context, id string) (Run, error) {
	run, ok := s.runs[strings.TrimSpace(id)]
	if !ok {
		return Run{}, ErrTaskRunNotFound
	}
	return cloneTaskRun(run), nil
}

func (s *inMemoryManagerStore) ListTaskRuns(_ context.Context, query RunQuery) ([]Run, error) {
	if err := query.Validate("task_run_query"); err != nil {
		return nil, err
	}

	normalized := query
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.Status = normalized.Status.Normalize()
	normalized.SessionID = strings.TrimSpace(normalized.SessionID)

	runs := make([]Run, 0)
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
		return append([]Run(nil), runs[:normalized.Limit]...), nil
	}
	return append([]Run(nil), runs...), nil
}

func (s *inMemoryManagerStore) ListTaskRunsByStatus(_ context.Context, statuses []RunStatus) ([]Run, error) {
	allowed := make(map[RunStatus]struct{}, len(statuses))
	for _, status := range statuses {
		allowed[status.Normalize()] = struct{}{}
	}
	runs := make([]Run, 0)
	for _, run := range s.runs {
		if _, ok := allowed[run.Status.Normalize()]; ok {
			runs = append(runs, cloneTaskRun(run))
		}
	}
	return runs, nil
}

func (s *inMemoryManagerStore) GetTaskTriageState(
	_ context.Context,
	taskID string,
	actor ActorIdentity,
) (TriageState, error) {
	record, ok := s.triageStates[triageKey(taskID, actor)]
	if !ok {
		return TriageState{}, ErrTaskTriageStateNotFound
	}
	return cloneTriageState(record), nil
}

func (s *inMemoryManagerStore) UpsertTaskTriageState(_ context.Context, state TriageState) error {
	if _, ok := s.tasks[strings.TrimSpace(state.TaskID)]; !ok {
		return ErrTaskNotFound
	}
	s.triageStates[triageKey(state.TaskID, state.Actor)] = cloneTriageState(state)
	return nil
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

func (s *inMemoryManagerStore) ReserveQueuedRun(
	_ context.Context,
	taskID string,
	runID string,
	runIdempotencyKey string,
	origin Origin,
	requestedChannel string,
	metadata json.RawMessage,
	queuedAt time.Time,
) (Task, Run, bool, error) {
	taskRecord, ok := s.tasks[strings.TrimSpace(taskID)]
	if !ok {
		return Task{}, Run{}, false, ErrTaskNotFound
	}
	taskRecord = cloneTask(taskRecord)
	if err := validateTaskForEnqueue(taskRecord); err != nil {
		return Task{}, Run{}, false, err
	}

	trimmedKey := strings.TrimSpace(runIdempotencyKey)
	if trimmedKey != "" {
		if record, ok := s.idempotencyByKey[idempotencyKey(origin, trimmedKey)]; ok {
			existingRun, err := s.GetTaskRun(context.Background(), record.RunID)
			if err != nil {
				return Task{}, Run{}, false, err
			}
			if existingRun.TaskID != taskRecord.ID {
				return Task{}, Run{}, false, fmtTestError(
					"%w: idempotency key %q is already bound to task %q",
					ErrValidation,
					trimmedKey,
					existingRun.TaskID,
				)
			}
			return taskRecord, existingRun, true, nil
		}
	}

	existingRuns, err := s.ListTaskRuns(context.Background(), RunQuery{TaskID: taskRecord.ID})
	if err != nil {
		return Task{}, Run{}, false, err
	}
	nextAttempt := nextRunAttempt(existingRuns)
	maxAttempts := normalizeTaskMaxAttemptsOrDefault(taskRecord.MaxAttempts)
	if nextAttempt > maxAttempts {
		return Task{}, Run{}, false, fmtTestError(
			"%w: task %q exhausted max_attempts=%d",
			ErrInvalidStatusTransition,
			taskRecord.ID,
			maxAttempts,
		)
	}

	run := Run{
		ID:             strings.TrimSpace(runID),
		TaskID:         taskRecord.ID,
		Status:         TaskRunStatusQueued,
		Attempt:        nextAttempt,
		Origin:         origin,
		IdempotencyKey: trimmedKey,
		NetworkChannel: resolvedRunChannel(requestedChannel, taskRecord.NetworkChannel),
		Metadata:       normalizeRawJSON(metadata),
		QueuedAt:       queuedAt.UTC(),
	}
	if err := s.CreateTaskRun(context.Background(), run); err != nil {
		return Task{}, Run{}, false, err
	}
	if trimmedKey != "" {
		if err := s.SaveTaskRunIdempotency(context.Background(), RunIdempotency{
			IdempotencyKey: trimmedKey,
			RunID:          run.ID,
			Origin:         origin,
			CreatedAt:      queuedAt.UTC(),
		}); err != nil {
			return Task{}, Run{}, false, err
		}
	}
	return taskRecord, run, false, nil
}

func (s *inMemoryManagerStore) CreateTaskEvent(_ context.Context, event Event) error {
	if _, ok := s.tasks[event.TaskID]; !ok {
		return ErrTaskNotFound
	}
	s.events = append(s.events, event)
	s.nextEventSequence++
	s.eventSequenceByID[event.ID] = s.nextEventSequence
	sort.Slice(s.events, func(i int, j int) bool {
		if s.events[i].Timestamp.Equal(s.events[j].Timestamp) {
			return s.events[i].ID > s.events[j].ID
		}
		return s.events[i].Timestamp.After(s.events[j].Timestamp)
	})
	return nil
}

func (s *inMemoryManagerStore) ListTaskEvents(_ context.Context, query EventQuery) ([]Event, error) {
	if err := query.Validate("task_event_query"); err != nil {
		return nil, err
	}
	events := make([]Event, 0)
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
		return append([]Event(nil), events[:query.Limit]...), nil
	}
	return append([]Event(nil), events...), nil
}

func (s *inMemoryManagerStore) GetTaskEventRecord(_ context.Context, eventID string) (EventRecord, error) {
	trimmedEventID := strings.TrimSpace(eventID)
	sequence, ok := s.eventSequenceByID[trimmedEventID]
	if !ok {
		return EventRecord{}, ErrTaskEventNotFound
	}
	for _, event := range s.events {
		if event.ID == trimmedEventID {
			return EventRecord{Sequence: sequence, Event: event}, nil
		}
	}
	return EventRecord{}, ErrTaskEventNotFound
}

func (s *inMemoryManagerStore) ListTaskEventRecords(
	_ context.Context,
	query EventRecordQuery,
) ([]EventRecord, error) {
	if err := query.Validate("task_event_record_query"); err != nil {
		return nil, err
	}

	records := make([]EventRecord, 0)
	for _, event := range s.events {
		if event.TaskID != strings.TrimSpace(query.TaskID) {
			continue
		}
		sequence := s.eventSequenceByID[event.ID]
		if sequence <= query.AfterSequence {
			continue
		}
		records = append(records, EventRecord{
			Sequence: sequence,
			Event:    event,
		})
	}

	sort.SliceStable(records, func(i int, j int) bool {
		return records[i].Sequence < records[j].Sequence
	})
	if query.Limit > 0 && len(records) > query.Limit {
		return append([]EventRecord(nil), records[:query.Limit]...), nil
	}
	return append([]EventRecord(nil), records...), nil
}

func (s *inMemoryManagerStore) GetTaskRunByIdempotencyKey(
	_ context.Context,
	key string,
	origin Origin,
) (Run, error) {
	record, ok := s.idempotencyByKey[idempotencyKey(origin, key)]
	if !ok {
		return Run{}, ErrTaskRunIdempotencyNotFound
	}
	return s.GetTaskRun(context.Background(), record.RunID)
}

func (s *inMemoryManagerStore) SaveTaskRunIdempotency(_ context.Context, record RunIdempotency) error {
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
			name: "automation-linked agent session",
			derive: func() (ActorContext, error) {
				return DeriveAutomationLinkedAgentSessionActorContext("sess-1", "run:run-1")
			},
			want: ActorContext{
				Actor:     ActorIdentity{Kind: ActorKindAgentSession, Ref: "sess-1"},
				Origin:    Origin{Kind: OriginKindAutomation, Ref: "run:run-1"},
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

func TestManagerTimelineSupportsStableOrderingAndWindows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor := validActorContext()
	base := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)

	taskRecord := Task{
		ID:             "task-root",
		Scope:          ScopeGlobal,
		Title:          "Root task",
		Priority:       DefaultPriority,
		MaxAttempts:    DefaultTaskMaxAttempts,
		Status:         TaskStatusReady,
		ApprovalPolicy: ApprovalPolicyNone,
		ApprovalState:  ApprovalStateNotRequired,
		CreatedBy:      actor.Actor,
		Origin:         actor.Origin,
		CreatedAt:      base,
		UpdatedAt:      base,
	}
	if err := store.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if err := store.CreateTaskRun(ctx, Run{
		ID:        "run-1",
		TaskID:    taskRecord.ID,
		Status:    TaskRunStatusRunning,
		Attempt:   1,
		SessionID: "sess-1",
		Origin:    actor.Origin,
		QueuedAt:  base,
		StartedAt: base.Add(30 * time.Second),
	}); err != nil {
		t.Fatalf("CreateTaskRun() error = %v", err)
	}

	sameTimestamp := base.Add(time.Minute)
	for _, event := range []Event{
		{
			ID:        "evt-z",
			TaskID:    taskRecord.ID,
			EventType: taskEventUpdated,
			Actor:     actor.Actor,
			Origin:    actor.Origin,
			Timestamp: sameTimestamp,
		},
		{
			ID:        "evt-a",
			TaskID:    taskRecord.ID,
			RunID:     "run-1",
			EventType: taskEventRunEnqueued,
			Actor:     actor.Actor,
			Origin:    actor.Origin,
			Timestamp: sameTimestamp,
		},
		{
			ID:        "evt-b",
			TaskID:    taskRecord.ID,
			RunID:     "run-1",
			EventType: taskEventRunStarted,
			Actor:     actor.Actor,
			Origin:    actor.Origin,
			Timestamp: sameTimestamp,
		},
	} {
		if err := store.CreateTaskEvent(ctx, event); err != nil {
			t.Fatalf("CreateTaskEvent(%q) error = %v", event.ID, err)
		}
	}

	window, err := manager.Timeline(ctx, taskRecord.ID, TimelineQuery{Limit: 2}, actor)
	if err != nil {
		t.Fatalf("Timeline(window) error = %v", err)
	}
	if got, want := len(window), 2; got != want {
		t.Fatalf("len(window) = %d, want %d", got, want)
	}
	if got, want := []string{window[0].EventID, window[1].EventID}, []string{"evt-z", "evt-a"}; !equalStringSlices(
		got,
		want,
	) {
		t.Fatalf("window event ids = %#v, want %#v", got, want)
	}
	if got, want := []int64{window[0].Sequence, window[1].Sequence}, []int64{1, 2}; !equalInt64Slices(got, want) {
		t.Fatalf("window sequences = %#v, want %#v", got, want)
	}
	if window[0].Run != nil {
		t.Fatalf("window[0].Run = %#v, want nil", window[0].Run)
	}
	if window[1].Run == nil || window[1].Run.ID != "run-1" {
		t.Fatalf("window[1].Run = %#v, want run-1 summary", window[1].Run)
	}

	tail, err := manager.Timeline(ctx, taskRecord.ID, TimelineQuery{
		AfterSequence: window[len(window)-1].Sequence,
		Limit:         2,
	}, actor)
	if err != nil {
		t.Fatalf("Timeline(tail) error = %v", err)
	}
	if got, want := len(tail), 1; got != want {
		t.Fatalf("len(tail) = %d, want %d", got, want)
	}
	if got, want := tail[0].EventID, "evt-b"; got != want {
		t.Fatalf("tail[0].EventID = %q, want %q", got, want)
	}
	if got, want := tail[0].Sequence, int64(3); got != want {
		t.Fatalf("tail[0].Sequence = %d, want %d", got, want)
	}

	all, err := manager.Timeline(ctx, taskRecord.ID, TimelineQuery{}, actor)
	if err != nil {
		t.Fatalf("Timeline(all) error = %v", err)
	}
	if got, want := []string{
		all[0].EventID,
		all[1].EventID,
		all[2].EventID,
	}, []string{
		"evt-z",
		"evt-a",
		"evt-b",
	}; !equalStringSlices(
		got,
		want,
	) {
		t.Fatalf("all event ids = %#v, want %#v", got, want)
	}
}

func TestManagerRunDetailAggregatesRuntimeContextAndOmitsOptionalFields(t *testing.T) {
	t.Parallel()

	t.Run("aggregates session and usage data", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		store := newInMemoryManagerStore()
		base := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)
		actor := validActorContext()

		runtimeReader := testRuntimeViewReader{
			sessions: map[string]*RunSessionRef{
				"sess-runtime": {
					SessionID:   "sess-runtime",
					WorkspaceID: "ws-1",
					AgentName:   "codex",
					Name:        "Run session",
					Channel:     "tasks",
					State:       "running",
					CreatedAt:   base,
					UpdatedAt:   base.Add(10 * time.Minute),
				},
			},
			events: map[string][]storepkg.SessionEvent{
				"sess-runtime": {
					{
						ID:        "sess-event-1",
						SessionID: "sess-runtime",
						Sequence:  1,
						TurnID:    "turn-1",
						Type:      "agent_message",
						AgentName: "codex",
						Content:   `{"text":"working"}`,
						Timestamp: base.Add(time.Minute),
					},
					{
						ID:        "sess-event-2",
						SessionID: "sess-runtime",
						Sequence:  2,
						TurnID:    "turn-1",
						Type:      sessionEventTypeToolCall,
						AgentName: "codex",
						Content:   `{"tool_call_id":"tool-1"}`,
						Timestamp: base.Add(2 * time.Minute),
					},
					{
						ID:        "sess-event-3",
						SessionID: "sess-runtime",
						Sequence:  3,
						TurnID:    "turn-1",
						Type:      sessionEventTypeToolResult,
						AgentName: "codex",
						Content:   `{"tool_call_id":"tool-1"}`,
						Timestamp: base.Add(3 * time.Minute),
					},
					{
						ID:        "sess-event-4",
						SessionID: "sess-runtime",
						Sequence:  4,
						TurnID:    "turn-2",
						Type:      sessionEventTypeToolCall,
						AgentName: "codex",
						Content:   `{"toolCallId":"tool-2"}`,
						Timestamp: base.Add(4 * time.Minute),
					},
				},
			},
			stats: map[string][]storepkg.TokenStats{
				"sess-runtime": {
					{
						ID:           "stats-1",
						SessionID:    "sess-runtime",
						AgentName:    "codex",
						InputTokens:  ptr(int64(10)),
						OutputTokens: ptr(int64(5)),
						TotalTokens:  ptr(int64(15)),
						TotalCost:    ptr(0.25),
						CostCurrency: ptr("USD"),
						TurnCount:    1,
						UpdatedAt:    base.Add(5 * time.Minute),
					},
					{
						ID:          "stats-2",
						SessionID:   "sess-runtime",
						AgentName:   "reviewer",
						InputTokens: ptr(int64(3)),
						TotalTokens: ptr(int64(3)),
						TotalCost:   ptr(0.10),
						TurnCount:   2,
						UpdatedAt:   base.Add(6 * time.Minute),
					},
				},
			},
		}
		manager := newTaskManagerForTestWithOptions(t, store, WithRuntimeViewReader(runtimeReader))

		taskRecord := Task{
			ID:             "task-runtime",
			Scope:          ScopeWorkspace,
			WorkspaceID:    "ws-1",
			Title:          "Runtime detail task",
			Priority:       DefaultPriority,
			MaxAttempts:    DefaultTaskMaxAttempts,
			Status:         TaskStatusReady,
			ApprovalPolicy: ApprovalPolicyNone,
			ApprovalState:  ApprovalStateNotRequired,
			CreatedBy:      actor.Actor,
			Origin:         actor.Origin,
			CreatedAt:      base,
			UpdatedAt:      base,
		}
		if err := store.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		run := Run{
			ID:        "run-runtime",
			TaskID:    taskRecord.ID,
			Status:    TaskRunStatusRunning,
			Attempt:   1,
			SessionID: "sess-runtime",
			Origin:    actor.Origin,
			QueuedAt:  base,
			StartedAt: base.Add(30 * time.Second),
		}
		if err := store.CreateTaskRun(ctx, run); err != nil {
			t.Fatalf("CreateTaskRun() error = %v", err)
		}

		detail, err := manager.RunDetail(ctx, run.ID, actor)
		if err != nil {
			t.Fatalf("RunDetail() error = %v", err)
		}
		if got, want := detail.Task.ID, taskRecord.ID; got != want {
			t.Fatalf("detail.Task.ID = %q, want %q", got, want)
		}
		if got, want := detail.Task.Status, TaskStatusInProgress; got != want {
			t.Fatalf("detail.Task.Status = %q, want %q", got, want)
		}
		if detail.Session == nil {
			t.Fatal("detail.Session = nil, want enriched session reference")
		}
		if got, want := detail.Session.AgentName, "codex"; got != want {
			t.Fatalf("detail.Session.AgentName = %q, want %q", got, want)
		}
		if got, want := detail.Session.WorkspaceID, "ws-1"; got != want {
			t.Fatalf("detail.Session.WorkspaceID = %q, want %q", got, want)
		}
		if detail.Summary.ToolCallCount == nil || *detail.Summary.ToolCallCount != 2 {
			t.Fatalf("detail.Summary.ToolCallCount = %#v, want 2", detail.Summary.ToolCallCount)
		}
		if detail.Summary.InputTokens == nil || *detail.Summary.InputTokens != 13 {
			t.Fatalf("detail.Summary.InputTokens = %#v, want 13", detail.Summary.InputTokens)
		}
		if detail.Summary.OutputTokens == nil || *detail.Summary.OutputTokens != 5 {
			t.Fatalf("detail.Summary.OutputTokens = %#v, want 5", detail.Summary.OutputTokens)
		}
		if detail.Summary.TotalTokens == nil || *detail.Summary.TotalTokens != 18 {
			t.Fatalf("detail.Summary.TotalTokens = %#v, want 18", detail.Summary.TotalTokens)
		}
		if detail.Summary.TurnCount == nil || *detail.Summary.TurnCount != 3 {
			t.Fatalf("detail.Summary.TurnCount = %#v, want 3", detail.Summary.TurnCount)
		}
		if detail.Summary.CostCurrency == nil || *detail.Summary.CostCurrency != "USD" {
			t.Fatalf("detail.Summary.CostCurrency = %#v, want USD", detail.Summary.CostCurrency)
		}
		if detail.Summary.TotalCost == nil || math.Abs(*detail.Summary.TotalCost-0.35) > 1e-9 {
			t.Fatalf("detail.Summary.TotalCost = %#v, want 0.35", detail.Summary.TotalCost)
		}
		if got, want := detail.Summary.LastEventType, sessionEventTypeToolCall; got != want {
			t.Fatalf("detail.Summary.LastEventType = %q, want %q", got, want)
		}
		if got, want := detail.Summary.LastActivityAt, base.Add(6*time.Minute); !got.Equal(want) {
			t.Fatalf("detail.Summary.LastActivityAt = %s, want %s", got, want)
		}
	})

	t.Run("keeps optional fields empty when runtime data is absent", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		store := newInMemoryManagerStore()
		manager := newTaskManagerForTest(t, store)
		actor := validActorContext()
		base := time.Date(2026, 4, 17, 11, 0, 0, 0, time.UTC)

		taskRecord := Task{
			ID:             "task-no-runtime",
			Scope:          ScopeGlobal,
			Title:          "No runtime detail task",
			Priority:       DefaultPriority,
			MaxAttempts:    DefaultTaskMaxAttempts,
			Status:         TaskStatusReady,
			ApprovalPolicy: ApprovalPolicyNone,
			ApprovalState:  ApprovalStateNotRequired,
			CreatedBy:      actor.Actor,
			Origin:         actor.Origin,
			CreatedAt:      base,
			UpdatedAt:      base,
		}
		if err := store.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		run := Run{
			ID:       "run-no-runtime",
			TaskID:   taskRecord.ID,
			Status:   TaskRunStatusQueued,
			Attempt:  1,
			Origin:   actor.Origin,
			QueuedAt: base,
		}
		if err := store.CreateTaskRun(ctx, run); err != nil {
			t.Fatalf("CreateTaskRun() error = %v", err)
		}

		detail, err := manager.RunDetail(ctx, run.ID, actor)
		if err != nil {
			t.Fatalf("RunDetail() error = %v", err)
		}
		if detail.Session != nil {
			t.Fatalf("detail.Session = %#v, want nil", detail.Session)
		}
		if !detail.Summary.LastActivityAt.IsZero() {
			t.Fatalf("detail.Summary.LastActivityAt = %s, want zero", detail.Summary.LastActivityAt)
		}
		if detail.Summary.ToolCallCount != nil {
			t.Fatalf("detail.Summary.ToolCallCount = %#v, want nil", detail.Summary.ToolCallCount)
		}
	})
}

func TestManagerTreeIncludesDescendantsActiveRunsAndLatestActivity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor := validActorContext()
	base := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)

	root := Task{
		ID:             "task-root",
		Scope:          ScopeGlobal,
		Title:          "Root",
		Priority:       DefaultPriority,
		MaxAttempts:    DefaultTaskMaxAttempts,
		Status:         TaskStatusReady,
		ApprovalPolicy: ApprovalPolicyNone,
		ApprovalState:  ApprovalStateNotRequired,
		CreatedBy:      actor.Actor,
		Origin:         actor.Origin,
		CreatedAt:      base,
		UpdatedAt:      base,
	}
	childActive := root
	childActive.ID = "task-child-active"
	childActive.ParentTaskID = root.ID
	childActive.Title = "Child active"
	childActive.CreatedAt = base.Add(time.Minute)
	childActive.UpdatedAt = base.Add(time.Minute)

	childIdle := root
	childIdle.ID = "task-child-idle"
	childIdle.ParentTaskID = root.ID
	childIdle.Title = "Child idle"
	childIdle.CreatedAt = base.Add(2 * time.Minute)
	childIdle.UpdatedAt = base.Add(2 * time.Minute)

	grandchild := root
	grandchild.ID = "task-grandchild"
	grandchild.ParentTaskID = childActive.ID
	grandchild.Title = "Grandchild"
	grandchild.CreatedAt = base.Add(3 * time.Minute)
	grandchild.UpdatedAt = base.Add(3 * time.Minute)

	for _, record := range []Task{root, childActive, childIdle, grandchild} {
		if err := store.CreateTask(ctx, record); err != nil {
			t.Fatalf("CreateTask(%q) error = %v", record.ID, err)
		}
	}

	for _, run := range []Run{
		{
			ID:        "run-active",
			TaskID:    childActive.ID,
			Status:    TaskRunStatusRunning,
			Attempt:   1,
			SessionID: "sess-child-active",
			Origin:    actor.Origin,
			QueuedAt:  base.Add(4 * time.Minute),
			StartedAt: base.Add(5 * time.Minute),
		},
		{
			ID:       "run-grandchild-complete",
			TaskID:   grandchild.ID,
			Status:   TaskRunStatusCompleted,
			Attempt:  1,
			Origin:   actor.Origin,
			QueuedAt: base.Add(2 * time.Minute),
			EndedAt:  base.Add(3 * time.Minute),
		},
	} {
		if err := store.CreateTaskRun(ctx, run); err != nil {
			t.Fatalf("CreateTaskRun(%q) error = %v", run.ID, err)
		}
	}

	for _, event := range []Event{
		{
			ID:        "root-event",
			TaskID:    root.ID,
			EventType: taskEventUpdated,
			Actor:     actor.Actor,
			Origin:    actor.Origin,
			Timestamp: base.Add(30 * time.Second),
		},
		{
			ID:        "child-active-event",
			TaskID:    childActive.ID,
			RunID:     "run-active",
			EventType: taskEventRunStarted,
			Actor:     actor.Actor,
			Origin:    actor.Origin,
			Timestamp: base.Add(6 * time.Minute),
		},
		{
			ID:        "child-idle-event",
			TaskID:    childIdle.ID,
			EventType: taskEventUpdated,
			Actor:     actor.Actor,
			Origin:    actor.Origin,
			Timestamp: base.Add(4 * time.Minute),
		},
		{
			ID:        "grandchild-event",
			TaskID:    grandchild.ID,
			RunID:     "run-grandchild-complete",
			EventType: taskEventRunCompleted,
			Actor:     actor.Actor,
			Origin:    actor.Origin,
			Timestamp: base.Add(3 * time.Minute),
		},
	} {
		if err := store.CreateTaskEvent(ctx, event); err != nil {
			t.Fatalf("CreateTaskEvent(%q) error = %v", event.ID, err)
		}
	}

	tree, err := manager.Tree(ctx, root.ID, actor)
	if err != nil {
		t.Fatalf("Tree() error = %v", err)
	}
	if got, want := tree.Root.Task.ID, root.ID; got != want {
		t.Fatalf("tree.Root.Task.ID = %q, want %q", got, want)
	}
	if got, want := tree.Root.ChildCount, 2; got != want {
		t.Fatalf("tree.Root.ChildCount = %d, want %d", got, want)
	}
	if got, want := len(tree.Descendants), 3; got != want {
		t.Fatalf("len(tree.Descendants) = %d, want %d", got, want)
	}
	if got, want := []string{
		tree.Descendants[0].Task.ID,
		tree.Descendants[1].Task.ID,
		tree.Descendants[2].Task.ID,
	}, []string{childActive.ID, childIdle.ID, grandchild.ID}; !equalStringSlices(got, want) {
		t.Fatalf("tree.Descendants order = %#v, want %#v", got, want)
	}

	activeNode := tree.Descendants[0]
	if got, want := activeNode.ParentTaskID, root.ID; got != want {
		t.Fatalf("activeNode.ParentTaskID = %q, want %q", got, want)
	}
	if got, want := activeNode.Depth, 1; got != want {
		t.Fatalf("activeNode.Depth = %d, want %d", got, want)
	}
	if got, want := activeNode.ChildCount, 1; got != want {
		t.Fatalf("activeNode.ChildCount = %d, want %d", got, want)
	}
	if activeNode.ActiveRun == nil || activeNode.ActiveRun.ID != "run-active" {
		t.Fatalf("activeNode.ActiveRun = %#v, want run-active", activeNode.ActiveRun)
	}
	if got, want := activeNode.LastActivityAt, base.Add(6*time.Minute); !got.Equal(want) {
		t.Fatalf("activeNode.LastActivityAt = %s, want %s", got, want)
	}

	grandchildNode := tree.Descendants[2]
	if got, want := grandchildNode.ParentTaskID, childActive.ID; got != want {
		t.Fatalf("grandchildNode.ParentTaskID = %q, want %q", got, want)
	}
	if got, want := grandchildNode.Depth, 2; got != want {
		t.Fatalf("grandchildNode.Depth = %d, want %d", got, want)
	}
	if grandchildNode.ActiveRun != nil {
		t.Fatalf("grandchildNode.ActiveRun = %#v, want nil", grandchildNode.ActiveRun)
	}
}

func TestManagerStreamReplaysStableBacklogAndLiveDescendantEvents(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor := validActorContext()
	base := time.Date(2026, 4, 17, 13, 0, 0, 0, time.UTC)

	root := Task{
		ID:             "task-root",
		Scope:          ScopeGlobal,
		Title:          "Root",
		Priority:       DefaultPriority,
		MaxAttempts:    DefaultTaskMaxAttempts,
		Status:         TaskStatusReady,
		ApprovalPolicy: ApprovalPolicyNone,
		ApprovalState:  ApprovalStateNotRequired,
		CreatedBy:      actor.Actor,
		Origin:         actor.Origin,
		CreatedAt:      base,
		UpdatedAt:      base,
	}
	child := root
	child.ID = "task-child"
	child.ParentTaskID = root.ID
	child.Title = "Child"
	child.CreatedAt = base.Add(time.Minute)
	child.UpdatedAt = child.CreatedAt
	for _, record := range []Task{root, child} {
		if err := store.CreateTask(ctx, record); err != nil {
			t.Fatalf("CreateTask(%q) error = %v", record.ID, err)
		}
	}

	if err := manager.recordTaskEvent(
		ctx,
		root.ID,
		"",
		taskEventUpdated,
		actor,
		map[string]any{"source": "root"},
	); err != nil {
		t.Fatalf("recordTaskEvent(root) error = %v", err)
	}
	if err := manager.recordTaskEvent(
		ctx,
		child.ID,
		"",
		taskEventUpdated,
		actor,
		map[string]any{"source": "child"},
	); err != nil {
		t.Fatalf("recordTaskEvent(child) error = %v", err)
	}

	streamCtx, cancel := context.WithCancel(ctx)
	stream, err := manager.Stream(streamCtx, root.ID, StreamQuery{AfterSequence: 1}, actor)
	if err != nil {
		t.Fatalf("Stream(first) error = %v", err)
	}

	backlog := awaitTaskStreamEvent(t, stream)
	if got, want := backlog.Sequence, int64(2); got != want {
		t.Fatalf("backlog.Sequence = %d, want %d", got, want)
	}
	if got, want := backlog.Timeline.Task.ID, child.ID; got != want {
		t.Fatalf("backlog.Timeline.Task.ID = %q, want %q", got, want)
	}
	if got, want := backlog.Type, taskEventUpdated; got != want {
		t.Fatalf("backlog.Type = %q, want %q", got, want)
	}

	if err := manager.recordTaskEvent(
		ctx,
		child.ID,
		"",
		taskEventRunEnqueued,
		actor,
		map[string]any{"attempt": 1},
	); err != nil {
		t.Fatalf("recordTaskEvent(live child) error = %v", err)
	}
	liveChild := awaitTaskStreamEvent(t, stream)
	if got, want := liveChild.Sequence, int64(3); got != want {
		t.Fatalf("liveChild.Sequence = %d, want %d", got, want)
	}
	if got, want := liveChild.Timeline.Task.ID, child.ID; got != want {
		t.Fatalf("liveChild.Timeline.Task.ID = %q, want %q", got, want)
	}
	if got, want := liveChild.Type, taskEventRunEnqueued; got != want {
		t.Fatalf("liveChild.Type = %q, want %q", got, want)
	}

	cancel()

	reconnectCtx, reconnectCancel := context.WithCancel(ctx)
	defer reconnectCancel()
	reconnected, err := manager.Stream(reconnectCtx, root.ID, StreamQuery{AfterSequence: liveChild.Sequence}, actor)
	if err != nil {
		t.Fatalf("Stream(reconnected) error = %v", err)
	}
	assertNoTaskStreamEvent(t, reconnected, 150*time.Millisecond)

	if err := manager.recordTaskEvent(
		ctx,
		root.ID,
		"",
		taskEventCanceled,
		actor,
		map[string]any{"reason": "done"},
	); err != nil {
		t.Fatalf("recordTaskEvent(root live) error = %v", err)
	}
	liveRoot := awaitTaskStreamEvent(t, reconnected)
	if got, want := liveRoot.Sequence, int64(4); got != want {
		t.Fatalf("liveRoot.Sequence = %d, want %d", got, want)
	}
	if got, want := liveRoot.Timeline.Task.ID, root.ID; got != want {
		t.Fatalf("liveRoot.Timeline.Task.ID = %q, want %q", got, want)
	}
	if got, want := liveRoot.Type, taskEventCanceled; got != want {
		t.Fatalf("liveRoot.Type = %q, want %q", got, want)
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
	if got, want := created.Priority, DefaultPriority; got != want {
		t.Fatalf("created.Priority = %q, want %q", got, want)
	}
	if got, want := created.MaxAttempts, DefaultTaskMaxAttempts; got != want {
		t.Fatalf("created.MaxAttempts = %d, want %d", got, want)
	}
	if got, want := created.ApprovalPolicy, ApprovalPolicyNone; got != want {
		t.Fatalf("created.ApprovalPolicy = %q, want %q", got, want)
	}
	if got, want := created.ApprovalState, ApprovalStateNotRequired; got != want {
		t.Fatalf("created.ApprovalState = %q, want %q", got, want)
	}

	events, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: created.ID})
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

func TestManagerCreateTaskAppliesSemanticDefaultsAndDraftStatus(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTestWithOptions(t, store, WithSessionExecutor(testSessionExecutor{}))
	actor := validActorContext()

	draftCreated, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Draft task",
		Draft: true,
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(draft) error = %v", err)
	}

	if got, want := draftCreated.Status, TaskStatusDraft; got != want {
		t.Fatalf("draftCreated.Status = %q, want %q", got, want)
	}
	if got, want := draftCreated.Priority, DefaultPriority; got != want {
		t.Fatalf("draftCreated.Priority = %q, want %q", got, want)
	}
	if got, want := draftCreated.MaxAttempts, DefaultTaskMaxAttempts; got != want {
		t.Fatalf("draftCreated.MaxAttempts = %d, want %d", got, want)
	}
	if got, want := draftCreated.ApprovalPolicy, ApprovalPolicyNone; got != want {
		t.Fatalf("draftCreated.ApprovalPolicy = %q, want %q", got, want)
	}
	if got, want := draftCreated.ApprovalState, ApprovalStateNotRequired; got != want {
		t.Fatalf("draftCreated.ApprovalState = %q, want %q", got, want)
	}
}

func TestManagerDraftPublicationReconcilesIntoReadyOrBlocked(t *testing.T) {
	t.Parallel()

	t.Run("draft stays non runnable until published", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		manager := newTaskManagerForTest(t, store)
		actor := validActorContext()

		draftTask, err := manager.CreateTask(context.Background(), CreateTask{
			Scope: ScopeGlobal,
			Title: "Draft task",
			Draft: true,
		}, actor)
		if err != nil {
			t.Fatalf("CreateTask(draft) error = %v", err)
		}

		if _, err := manager.EnqueueRun(
			context.Background(),
			EnqueueRun{TaskID: draftTask.ID},
			actor,
		); !errors.Is(err, ErrInvalidStatusTransition) {
			t.Fatalf("EnqueueRun(draft) error = %v, want %v", err, ErrInvalidStatusTransition)
		}

		published, err := manager.PublishTask(context.Background(), draftTask.ID, actor)
		if err != nil {
			t.Fatalf("PublishTask() error = %v", err)
		}
		if got, want := published.Status, TaskStatusReady; got != want {
			t.Fatalf("published.Status = %q, want %q", got, want)
		}

		events, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: draftTask.ID})
		if err != nil {
			t.Fatalf("ListTaskEvents() error = %v", err)
		}
		if !containsEventType(events, taskEventPublished) {
			t.Fatalf("event types = %#v, want %q", sortedEventTypes(events), taskEventPublished)
		}
	})

	t.Run("published draft remains blocked for unresolved dependencies", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		manager := newTaskManagerForTest(t, store)
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
			Title: "Target draft",
			Draft: true,
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
		if got, want := store.tasks[target.ID].Status, TaskStatusDraft; got != want {
			t.Fatalf("target.Status before publish = %q, want %q", got, want)
		}

		published, err := manager.PublishTask(context.Background(), target.ID, actor)
		if err != nil {
			t.Fatalf("PublishTask() error = %v", err)
		}
		if got, want := published.Status, TaskStatusBlocked; got != want {
			t.Fatalf("published.Status = %q, want %q", got, want)
		}
	})
}

func TestManagerPublishTaskRejectsNonDraftTasks(t *testing.T) {
	t.Parallel()

	t.Run("ready task cannot publish", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		manager := newTaskManagerForTest(t, store)
		actor := validActorContext()

		taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
			Scope: ScopeGlobal,
			Title: "Already runnable",
		}, actor)
		if err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}

		if _, err := manager.PublishTask(context.Background(), taskRecord.ID, actor); !errors.Is(
			err,
			ErrInvalidStatusTransition,
		) {
			t.Fatalf("PublishTask(ready) error = %v, want %v", err, ErrInvalidStatusTransition)
		}
	})

	t.Run("already published draft cannot publish again", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		manager := newTaskManagerForTest(t, store)
		actor := validActorContext()

		draftTask, err := manager.CreateTask(context.Background(), CreateTask{
			Scope: ScopeGlobal,
			Title: "Draft once",
			Draft: true,
		}, actor)
		if err != nil {
			t.Fatalf("CreateTask(draft) error = %v", err)
		}

		if _, err := manager.PublishTask(context.Background(), draftTask.ID, actor); err != nil {
			t.Fatalf("PublishTask(first) error = %v", err)
		}
		if _, err := manager.PublishTask(context.Background(), draftTask.ID, actor); !errors.Is(
			err,
			ErrInvalidStatusTransition,
		) {
			t.Fatalf("PublishTask(second) error = %v, want %v", err, ErrInvalidStatusTransition)
		}
	})
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

func TestManagerCreateTaskRejectsInvalidSemanticInputsBeforePersistence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec CreateTask
	}{
		{
			name: "invalid priority",
			spec: CreateTask{Scope: ScopeGlobal, Title: "Bad priority", Priority: Priority("rush")},
		},
		{
			name: "invalid max attempts",
			spec: CreateTask{Scope: ScopeGlobal, Title: "Bad attempts", MaxAttempts: ptr(0)},
		},
		{
			name: "invalid approval policy",
			spec: CreateTask{Scope: ScopeGlobal, Title: "Bad approval", ApprovalPolicy: ApprovalPolicy("auto")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := newInMemoryManagerStore()
			manager := newTaskManagerForTest(t, store)

			_, err := manager.CreateTask(context.Background(), tt.spec, validActorContext())
			if err == nil {
				t.Fatal("CreateTask() error = nil, want non-nil")
			}
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("CreateTask() error = %v, want %v", err, ErrValidation)
			}
			if got := len(store.tasks); got != 0 {
				t.Fatalf("len(store.tasks) = %d, want 0", got)
			}
		})
	}
}

func TestManagerUpdateTaskAllowsMutableOwnershipAndChannelFields(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTestWithOptions(t, store, WithSessionExecutor(testSessionExecutor{}))
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
	metadata := json.RawMessage(`{"source":"ui"}`)
	priority := PriorityUrgent
	maxAttempts := 5
	approvalPolicy := ApprovalPolicyManual
	updated, err := manager.UpdateTask(context.Background(), created.ID, Patch{
		Title:          &title,
		Description:    &description,
		Priority:       &priority,
		MaxAttempts:    &maxAttempts,
		ApprovalPolicy: &approvalPolicy,
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
	if got, want := updated.Priority, priority; got != want {
		t.Fatalf("updated.Priority = %q, want %q", got, want)
	}
	if got, want := updated.MaxAttempts, maxAttempts; got != want {
		t.Fatalf("updated.MaxAttempts = %d, want %d", got, want)
	}
	if got, want := updated.ApprovalPolicy, approvalPolicy; got != want {
		t.Fatalf("updated.ApprovalPolicy = %q, want %q", got, want)
	}
	if got, want := updated.ApprovalState, ApprovalStatePending; got != want {
		t.Fatalf("updated.ApprovalState = %q, want %q", got, want)
	}
	if got, want := updated.Status, TaskStatusBlocked; got != want {
		t.Fatalf("updated.Status = %q, want %q", got, want)
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
	cleared, err := manager.UpdateTask(context.Background(), created.ID, Patch{
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
	updated, err := manager.UpdateTask(context.Background(), taskA.ID, Patch{
		Title: &title,
	}, actor)
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if got, want := updated.Status, TaskStatusBlocked; got != want {
		t.Fatalf("updated.Status = %q, want %q", got, want)
	}
}

func TestManagerApprovalGateBlocksExecutionUntilApproved(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor := validActorContext()

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope:          ScopeGlobal,
		Title:          "Manual approval task",
		ApprovalPolicy: ApprovalPolicyManual,
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if got, want := taskRecord.Status, TaskStatusBlocked; got != want {
		t.Fatalf("taskRecord.Status = %q, want %q", got, want)
	}

	run, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	if _, err := manager.ClaimRun(
		context.Background(),
		run.ID,
		ClaimRun{},
		actor,
	); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("ClaimRun(blocked) error = %v, want %v", err, ErrInvalidStatusTransition)
	}

	approved, err := manager.ApproveTask(context.Background(), taskRecord.ID, actor)
	if err != nil {
		t.Fatalf("ApproveTask() error = %v", err)
	}
	if got, want := approved.ApprovalState, ApprovalStateApproved; got != want {
		t.Fatalf("approved.ApprovalState = %q, want %q", got, want)
	}
	if got, want := approved.Status, TaskStatusReady; got != want {
		t.Fatalf("approved.Status = %q, want %q", got, want)
	}

	claimed, err := manager.ClaimRun(context.Background(), run.ID, ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(approved) error = %v", err)
	}
	if got, want := claimed.Status, TaskRunStatusClaimed; got != want {
		t.Fatalf("claimed.Status = %q, want %q", got, want)
	}
	if got, want := store.tasks[taskRecord.ID].Status, TaskStatusReady; got != want {
		t.Fatalf("task.Status after claim = %q, want %q", got, want)
	}

	events, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: taskRecord.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if !containsEventType(events, taskEventApproved) {
		t.Fatalf("event types = %v, want %q", sortedEventTypes(events), taskEventApproved)
	}
}

func TestManagerRejectTaskKeepsManualApprovalBlocked(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor := validActorContext()

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope:          ScopeGlobal,
		Title:          "Manual rejection task",
		ApprovalPolicy: ApprovalPolicyManual,
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	run, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}

	rejected, err := manager.RejectTask(context.Background(), taskRecord.ID, actor)
	if err != nil {
		t.Fatalf("RejectTask() error = %v", err)
	}
	if got, want := rejected.ApprovalState, ApprovalStateRejected; got != want {
		t.Fatalf("rejected.ApprovalState = %q, want %q", got, want)
	}
	if got, want := rejected.Status, TaskStatusBlocked; got != want {
		t.Fatalf("rejected.Status = %q, want %q", got, want)
	}
	if _, err := manager.ClaimRun(
		context.Background(),
		run.ID,
		ClaimRun{},
		actor,
	); !errors.Is(
		err,
		ErrInvalidStatusTransition,
	) {
		t.Fatalf("ClaimRun(rejected) error = %v, want %v", err, ErrInvalidStatusTransition)
	}

	events, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: taskRecord.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if !containsEventType(events, taskEventRejected) {
		t.Fatalf("event types = %v, want %q", sortedEventTypes(events), taskEventRejected)
	}
}

func TestManagerTaskTriageMutationsPersistActorScopedStateWithoutTaskEvents(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	alice := validActorContext()
	bob := validActorContext()
	bob.Actor.Ref = "user-bob"

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Inbox triage target",
	}, alice)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	eventsBefore, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: taskRecord.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents(before) error = %v", err)
	}
	if got, want := len(eventsBefore), 1; got != want {
		t.Fatalf("len(eventsBefore) = %d, want %d", got, want)
	}

	readState, err := manager.MarkTaskRead(context.Background(), taskRecord.ID, alice)
	if err != nil {
		t.Fatalf("MarkTaskRead() error = %v", err)
	}
	if !readState.Read || readState.Archived || readState.Dismissed {
		t.Fatalf("readState = %#v, want read-only triage state", readState)
	}
	if !readState.LastSeenActivityAt.Equal(taskRecord.UpdatedAt) {
		t.Fatalf("readState.LastSeenActivityAt = %v, want %v", readState.LastSeenActivityAt, taskRecord.UpdatedAt)
	}

	dismissedState, err := manager.DismissTask(context.Background(), taskRecord.ID, bob)
	if err != nil {
		t.Fatalf("DismissTask() error = %v", err)
	}
	if !dismissedState.Read || !dismissedState.Dismissed || dismissedState.Archived {
		t.Fatalf("dismissedState = %#v, want dismissed unread-clearing triage state", dismissedState)
	}

	archivedState, err := manager.ArchiveTask(context.Background(), taskRecord.ID, alice)
	if err != nil {
		t.Fatalf("ArchiveTask() error = %v", err)
	}
	if !archivedState.Read || !archivedState.Archived || archivedState.Dismissed {
		t.Fatalf("archivedState = %#v, want archived triage state", archivedState)
	}

	storedAlice, err := store.GetTaskTriageState(context.Background(), taskRecord.ID, alice.Actor)
	if err != nil {
		t.Fatalf("GetTaskTriageState(alice) error = %v", err)
	}
	if storedAlice != archivedState {
		t.Fatalf("storedAlice = %#v, want %#v", storedAlice, archivedState)
	}
	storedBob, err := store.GetTaskTriageState(context.Background(), taskRecord.ID, bob.Actor)
	if err != nil {
		t.Fatalf("GetTaskTriageState(bob) error = %v", err)
	}
	if storedBob != dismissedState {
		t.Fatalf("storedBob = %#v, want %#v", storedBob, dismissedState)
	}

	eventsAfter, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: taskRecord.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents(after) error = %v", err)
	}
	if got, want := len(eventsAfter), len(eventsBefore); got != want {
		t.Fatalf("len(eventsAfter) = %d, want %d; event types=%v", got, want, sortedEventTypes(eventsAfter))
	}
}

func TestManagerAttemptExhaustionBlocksFurtherRetries(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTestWithOptions(t, store, WithSessionExecutor(testSessionExecutor{}))
	actor := validActorContext()

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope:       ScopeGlobal,
		Title:       "Retry budget task",
		MaxAttempts: ptr(2),
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	firstRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(first) error = %v", err)
	}
	firstRun, err = manager.ClaimRun(context.Background(), firstRun.ID, ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(first) error = %v", err)
	}
	firstRun, err = manager.StartRun(context.Background(), firstRun.ID, StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun(first) error = %v", err)
	}
	if _, err := manager.FailRun(context.Background(), firstRun.ID, RunFailure{
		Error: "boom-1",
	}, actor); err != nil {
		t.Fatalf("FailRun(first) error = %v", err)
	}
	if got, want := store.tasks[taskRecord.ID].Status, TaskStatusReady; got != want {
		t.Fatalf("task.Status after first failure = %q, want %q", got, want)
	}

	secondRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(second) error = %v", err)
	}
	secondRun, err = manager.ClaimRun(context.Background(), secondRun.ID, ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun(second) error = %v", err)
	}
	secondRun, err = manager.StartRun(context.Background(), secondRun.ID, StartRun{}, actor)
	if err != nil {
		t.Fatalf("StartRun(second) error = %v", err)
	}
	if _, err := manager.FailRun(context.Background(), secondRun.ID, RunFailure{
		Error: "boom-2",
	}, actor); err != nil {
		t.Fatalf("FailRun(second) error = %v", err)
	}
	if got, want := store.tasks[taskRecord.ID].Status, TaskStatusFailed; got != want {
		t.Fatalf("task.Status after second failure = %q, want %q", got, want)
	}

	if _, err := manager.EnqueueRun(
		context.Background(),
		EnqueueRun{TaskID: taskRecord.ID},
		actor,
	); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("EnqueueRun(exhausted) error = %v, want %v", err, ErrInvalidStatusTransition)
	}
}

func TestManagerEnqueueRunRejectsDraftTask(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor := validActorContext()

	draftTask, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Draft task",
		Draft: true,
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	run, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: draftTask.ID}, actor)
	if run != nil {
		t.Fatalf("EnqueueRun() run = %#v, want nil", run)
	}
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("EnqueueRun() error = %v, want %v", err, ErrInvalidStatusTransition)
	}
	if got := len(store.runs); got != 0 {
		t.Fatalf("len(store.runs) = %d, want 0", got)
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

		events, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: parent.ID})
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

	events, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: taskA.ID})
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

	store.runs["run-active"] = Run{
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
	if got, want := view.Summary.DependencyCount, 1; got != want {
		t.Fatalf("view.Summary.DependencyCount = %d, want %d", got, want)
	}
	if got, want := view.Summary.ChildCount, 0; got != want {
		t.Fatalf("view.Summary.ChildCount = %d, want %d", got, want)
	}
	if view.Summary.ActiveRun == nil || view.Summary.ActiveRun.ID != "run-active" {
		t.Fatalf("view.Summary.ActiveRun = %#v, want run-active", view.Summary.ActiveRun)
	}
	if view.Summary.LastActivityAt.IsZero() {
		t.Fatal("view.Summary.LastActivityAt is zero, want latest activity timestamp")
	}
	if len(view.DependencyReferences) != 1 {
		t.Fatalf("len(view.DependencyReferences) = %d, want 1", len(view.DependencyReferences))
	}
	if got, want := view.DependencyReferences[0].DependsOn.Title, dependency.Title; got != want {
		t.Fatalf("view.DependencyReferences[0].DependsOn.Title = %q, want %q", got, want)
	}

	summaries, err := manager.ListTasks(context.Background(), Query{ParentTaskID: parent.ID}, actor)
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(summaries) != 1 || summaries[0].ID != child.ID {
		t.Fatalf("ListTasks(parent filter) = %#v, want only child %q", summaries, child.ID)
	}
	if got, want := summaries[0].DependencyCount, 1; got != want {
		t.Fatalf("summaries[0].DependencyCount = %d, want %d", got, want)
	}
	if len(summaries[0].Dependencies) != 1 {
		t.Fatalf("len(summaries[0].Dependencies) = %d, want 1", len(summaries[0].Dependencies))
	}
	if got, want := summaries[0].Dependencies[0].DependsOn.Identifier, dependency.Identifier; got != want {
		t.Fatalf("summaries[0].Dependencies[0].DependsOn.Identifier = %q, want %q", got, want)
	}
	if summaries[0].ActiveRun == nil || summaries[0].ActiveRun.Status != TaskRunStatusRunning {
		t.Fatalf("summaries[0].ActiveRun = %#v, want running summary", summaries[0].ActiveRun)
	}
	runs, err := manager.ListTaskRuns(context.Background(), child.ID, RunQuery{}, actor)
	if err != nil {
		t.Fatalf("ListTaskRuns() error = %v", err)
	}
	if len(runs) != 1 || runs[0].ID != "run-active" {
		t.Fatalf("ListTaskRuns() = %#v, want only run-active", runs)
	}

	noRead := actor
	noRead.Authority.Read = false
	if _, err := manager.GetTask(context.Background(), child.ID, noRead); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("GetTask(no read) error = %v, want %v", err, ErrPermissionDenied)
	}
	if _, err := manager.ListTasks(context.Background(), Query{}, noRead); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("ListTasks(no read) error = %v, want %v", err, ErrPermissionDenied)
	}
	if _, err := manager.ListTaskRuns(
		context.Background(),
		child.ID,
		RunQuery{},
		noRead,
	); !errors.Is(
		err,
		ErrPermissionDenied,
	) {
		t.Fatalf("ListTaskRuns(no read) error = %v, want %v", err, ErrPermissionDenied)
	}
}

func TestManagerListTasksSupportsSearchAndOrdersByLatestActivity(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor := validActorContext()

	first, err := manager.CreateTask(context.Background(), CreateTask{
		Scope:      ScopeGlobal,
		Title:      "Alpha planning",
		Identifier: "OPS-100",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(first) error = %v", err)
	}
	second, err := manager.CreateTask(context.Background(), CreateTask{
		Scope:      ScopeGlobal,
		Title:      "Beta rollout",
		Identifier: "OPS-200",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(second) error = %v", err)
	}

	store.runs["run-second"] = Run{
		ID:        "run-second",
		TaskID:    second.ID,
		Status:    TaskRunStatusRunning,
		Attempt:   1,
		Origin:    Origin{Kind: OriginKindAutomation, Ref: "rule:nightly"},
		QueuedAt:  time.Date(2026, 4, 14, 16, 0, 0, 0, time.UTC),
		StartedAt: time.Date(2026, 4, 14, 16, 5, 0, 0, time.UTC),
	}

	byTitle, err := manager.ListTasks(context.Background(), Query{Search: "alpha"}, actor)
	if err != nil {
		t.Fatalf("ListTasks(search title) error = %v", err)
	}
	if len(byTitle) != 1 || byTitle[0].ID != first.ID {
		t.Fatalf("ListTasks(search title) = %#v, want only %q", byTitle, first.ID)
	}

	byIdentifier, err := manager.ListTasks(context.Background(), Query{Search: "ops-200"}, actor)
	if err != nil {
		t.Fatalf("ListTasks(search identifier) error = %v", err)
	}
	if len(byIdentifier) != 1 || byIdentifier[0].ID != second.ID {
		t.Fatalf("ListTasks(search identifier) = %#v, want only %q", byIdentifier, second.ID)
	}

	all, err := manager.ListTasks(context.Background(), Query{}, actor)
	if err != nil {
		t.Fatalf("ListTasks(all) error = %v", err)
	}
	if got, want := []string{all[0].ID, all[1].ID}, []string{second.ID, first.ID}; !equalStringSlices(got, want) {
		t.Fatalf("ListTasks(all) order = %#v, want %#v", got, want)
	}
}

func TestManagerListTasksCombinedFiltersPreserveEnrichedFields(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor := validActorContext()

	parent, err := manager.CreateTask(context.Background(), CreateTask{
		Scope:       ScopeWorkspace,
		WorkspaceID: "ws-1",
		Title:       "Alpha parent",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}
	blocker, err := manager.CreateTask(context.Background(), CreateTask{
		Scope:       ScopeWorkspace,
		WorkspaceID: "ws-1",
		Title:       "Ops blocker",
		Identifier:  "OPS-999",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask(blocker) error = %v", err)
	}
	matching, err := manager.CreateChildTask(context.Background(), parent.ID, CreateTask{
		Scope:       ScopeWorkspace,
		WorkspaceID: "ws-1",
		Title:       "Alpha child rollout",
		Identifier:  "OPS-300",
	}, actor)
	if err != nil {
		t.Fatalf("CreateChildTask(matching) error = %v", err)
	}
	if err := manager.AddDependency(context.Background(), AddDependency{
		TaskID:          matching.ID,
		DependsOnTaskID: blocker.ID,
		Kind:            DependencyKindBlocks,
	}, actor); err != nil {
		t.Fatalf("AddDependency(matching) error = %v", err)
	}

	if _, err := manager.CreateChildTask(context.Background(), parent.ID, CreateTask{
		Scope:       ScopeWorkspace,
		WorkspaceID: "ws-1",
		Title:       "Alpha child ready",
		Identifier:  "OPS-301",
	}, actor); err != nil {
		t.Fatalf("CreateChildTask(ready sibling) error = %v", err)
	}
	if _, err := manager.CreateTask(context.Background(), CreateTask{
		Scope:       ScopeWorkspace,
		WorkspaceID: "ws-2",
		Title:       "Alpha child rollout",
		Identifier:  "OPS-300",
	}, actor); err != nil {
		t.Fatalf("CreateTask(other workspace) error = %v", err)
	}

	summaries, err := manager.ListTasks(context.Background(), Query{
		WorkspaceID:  "ws-1",
		Status:       TaskStatusBlocked,
		ParentTaskID: parent.ID,
		Search:       "ops-300",
	}, actor)
	if err != nil {
		t.Fatalf("ListTasks(combined filters) error = %v", err)
	}
	if len(summaries) != 1 || summaries[0].ID != matching.ID {
		t.Fatalf("ListTasks(combined filters) = %#v, want only %q", summaries, matching.ID)
	}
	if got, want := summaries[0].Status, TaskStatusBlocked; got != want {
		t.Fatalf("summaries[0].Status = %q, want %q", got, want)
	}
	if got, want := summaries[0].DependencyCount, 1; got != want {
		t.Fatalf("summaries[0].DependencyCount = %d, want %d", got, want)
	}
	if got, want := summaries[0].ChildCount, 0; got != want {
		t.Fatalf("summaries[0].ChildCount = %d, want %d", got, want)
	}
	if len(summaries[0].Dependencies) != 1 {
		t.Fatalf("len(summaries[0].Dependencies) = %d, want 1", len(summaries[0].Dependencies))
	}
	if got, want := summaries[0].Dependencies[0].DependsOn.Title, blocker.Title; got != want {
		t.Fatalf("summaries[0].Dependencies[0].DependsOn.Title = %q, want %q", got, want)
	}
	if summaries[0].LastActivityAt.IsZero() {
		t.Fatal("summaries[0].LastActivityAt is zero, want latest activity timestamp")
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

	if _, err := manager.ClaimRun(
		context.Background(),
		runningRun.ID,
		ClaimRun{},
		actor,
	); !errors.Is(
		err,
		ErrInvalidStatusTransition,
	) {
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
		Scope:       ScopeGlobal,
		Title:       "Failure task",
		MaxAttempts: ptr(1),
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
		Title: "Canceled task",
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
	if got, want := store.tasks[cancelledTask.ID].Status, TaskStatusCanceled; got != want {
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
	if got, want := cancelledParent.Status, TaskStatusCanceled; got != want {
		t.Fatalf("cancelledParent.Status = %q, want %q", got, want)
	}
	if got, want := store.tasks[parent.ID].Status, TaskStatusCanceled; got != want {
		t.Fatalf("parent.Status = %q, want %q", got, want)
	}
	if got, want := store.tasks[queuedChild.ID].Status, TaskStatusCanceled; got != want {
		t.Fatalf("queuedChild.Status = %q, want %q", got, want)
	}
	if got, want := store.tasks[activeChild.ID].Status, TaskStatusCanceled; got != want {
		t.Fatalf("activeChild.Status = %q, want %q", got, want)
	}
	if got, want := store.runs[queuedRun.ID].Status, TaskRunStatusCanceled; got != want {
		t.Fatalf("queuedRun.Status = %q, want %q", got, want)
	}
	if got, want := store.runs[activeRun.ID].Status, TaskRunStatusCanceled; got != want {
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

	parentEvents, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: parent.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents(parent) error = %v", err)
	}
	if !containsEventType(parentEvents, taskEventCanceled) {
		t.Fatalf("parent events = %#v, want %q", sortedEventTypes(parentEvents), taskEventCanceled)
	}

	activeChildEvents, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: activeChild.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents(active child) error = %v", err)
	}
	if !containsEventType(activeChildEvents, taskEventRunCanceled) {
		t.Fatalf("active child events = %#v, want %q", sortedEventTypes(activeChildEvents), taskEventRunCanceled)
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
		MaxAttempts:    ptr(2),
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
	if _, err := manager.AttachRunSession(
		context.Background(),
		retryRun.ID,
		"sess-other",
		actor,
	); !errors.Is(
		err,
		ErrSessionAlreadyBound,
	) {
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

	if _, err := manager.EnqueueRun(
		context.Background(),
		EnqueueRun{TaskID: taskOne.ID},
		automationActor,
	); !errors.Is(
		err,
		ErrValidation,
	) {
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

	if _, err := manager.ClaimRun(
		context.Background(),
		runOne.ID,
		ClaimRun{},
		automationActor,
	); !errors.Is(
		err,
		ErrValidation,
	) {
		t.Fatalf("ClaimRun(no idempotency) error = %v, want %v", err, ErrValidation)
	}
	claimedRun, err := manager.ClaimRun(context.Background(), runOne.ID, ClaimRun{
		IdempotencyKey: "claim-idem",
	}, automationActor)
	if err != nil {
		t.Fatalf("ClaimRun(with idempotency) error = %v", err)
	}
	if _, err := manager.StartRun(
		context.Background(),
		claimedRun.ID,
		StartRun{},
		automationActor,
	); !errors.Is(
		err,
		ErrValidation,
	) {
		t.Fatalf("StartRun(no idempotency) error = %v, want %v", err, ErrValidation)
	}
	if _, err := manager.StartRun(context.Background(), claimedRun.ID, StartRun{
		IdempotencyKey: "start-idem",
	}, automationActor); !errors.Is(err, ErrValidation) {
		t.Fatalf("StartRun(no session executor) error = %v, want %v", err, ErrValidation)
	}
}

func TestManagerEnqueueRunPreservesMetadataAcrossIdempotentDuplicates(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor, err := DeriveAutomationActorContext("rule:harness-detached", "daemon.harness.detached")
	if err != nil {
		t.Fatalf("DeriveAutomationActorContext() error = %v", err)
	}

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Detached metadata task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	metadata := json.RawMessage(
		`{"schema":"agh.harness.detached.v1","owner_session_id":"sess-owner","wake_target":{"session_id":"sess-wake"}}`,
	)
	firstRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{
		TaskID:         taskRecord.ID,
		IdempotencyKey: "detached-metadata-1",
		Metadata:       metadata,
	}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(first) error = %v", err)
	}
	if got, want := string(firstRun.Metadata), string(metadata); got != want {
		t.Fatalf("firstRun.Metadata = %s, want %s", got, want)
	}

	storedRun, err := store.GetTaskRun(context.Background(), firstRun.ID)
	if err != nil {
		t.Fatalf("GetTaskRun() error = %v", err)
	}
	if got, want := string(storedRun.Metadata), string(metadata); got != want {
		t.Fatalf("storedRun.Metadata = %s, want %s", got, want)
	}

	duplicateRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{
		TaskID:         taskRecord.ID,
		IdempotencyKey: "detached-metadata-1",
		Metadata:       metadata,
	}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(duplicate) error = %v", err)
	}
	if got, want := duplicateRun.ID, firstRun.ID; got != want {
		t.Fatalf("duplicateRun.ID = %q, want %q", got, want)
	}
	if got, want := string(duplicateRun.Metadata), string(metadata); got != want {
		t.Fatalf("duplicateRun.Metadata = %s, want %s", got, want)
	}
	if got, want := len(store.runs), 1; got != want {
		t.Fatalf("len(store.runs) = %d, want %d", got, want)
	}
}

func TestManagerNetworkPeerEnqueueRunUsesOriginScopedIdempotency(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor, err := DeriveNetworkPeerActorContext("peer.ops-review", "peer:peer.ops-review/channel:ops")
	if err != nil {
		t.Fatalf("DeriveNetworkPeerActorContext() error = %v", err)
	}

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope:          ScopeGlobal,
		Title:          "Peer-originated task",
		NetworkChannel: "ops",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	firstRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{
		TaskID:         taskRecord.ID,
		IdempotencyKey: "delivery-1",
	}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(first) error = %v", err)
	}
	secondRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{
		TaskID:         taskRecord.ID,
		IdempotencyKey: "delivery-1",
	}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun(duplicate) error = %v", err)
	}

	if got, want := secondRun.ID, firstRun.ID; got != want {
		t.Fatalf("duplicate enqueue run id = %q, want %q", got, want)
	}
	if got, want := len(store.runs), 1; got != want {
		t.Fatalf("len(store.runs) = %d, want %d", got, want)
	}
	if got, want := len(store.idempotencyByKey), 1; got != want {
		t.Fatalf("len(store.idempotencyByKey) = %d, want %d", got, want)
	}
}

func TestManagerStartRunRejectsStaleRunChannelWithoutMutation(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	bootstrap := newTaskManagerForTest(t, store)
	actor, err := DeriveHumanActorContext("user-1", OriginKindCLI, "agh task run start")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}

	taskRecord, err := bootstrap.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Stale run snapshot task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	run, err := bootstrap.EnqueueRun(context.Background(), EnqueueRun{
		TaskID:         taskRecord.ID,
		NetworkChannel: "legacy",
	}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	run, err = bootstrap.ClaimRun(context.Background(), run.ID, ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun() error = %v", err)
	}

	executor := &recordingSessionExecutor{}
	manager := newTaskManagerForTestWithOptions(
		t,
		store,
		WithSessionExecutor(executor),
		WithNetworkChannelValidator(func(channel string) error {
			if channel == "legacy" {
				return fmt.Errorf("channel retired")
			}
			return nil
		}),
	)

	started, err := manager.StartRun(context.Background(), run.ID, StartRun{}, actor)
	if !errors.Is(err, ErrStaleNetworkChannel) {
		t.Fatalf("StartRun() error = %v, want %v", err, ErrStaleNetworkChannel)
	}
	if started != nil {
		t.Fatalf("StartRun() run = %#v, want nil on stale-channel rejection", started)
	}
	if got := len(executor.startCalls); got != 0 {
		t.Fatalf("len(executor.startCalls) = %d, want 0", got)
	}

	storedRun, err := store.GetTaskRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("GetTaskRun() error = %v", err)
	}
	if got, want := storedRun.Status, TaskRunStatusClaimed; got != want {
		t.Fatalf("storedRun.Status = %q, want %q", got, want)
	}
	if !storedRun.StartedAt.IsZero() {
		t.Fatalf("storedRun.StartedAt = %s, want zero", storedRun.StartedAt)
	}
	if got, want := storedRun.NetworkChannel, "legacy"; got != want {
		t.Fatalf("storedRun.NetworkChannel = %q, want %q", got, want)
	}

	events, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: taskRecord.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	rejectedEvents := make([]Event, 0)
	for _, event := range events {
		if event.EventType == taskEventRunRejected {
			rejectedEvents = append(rejectedEvents, event)
		}
	}
	if got, want := len(rejectedEvents), 1; got != want {
		t.Fatalf("len(rejectedEvents) = %d, want %d; event types=%v", got, want, sortedEventTypes(events))
	}

	var payload rejectedRunPayload
	if err := json.Unmarshal(rejectedEvents[0].Payload, &payload); err != nil {
		t.Fatalf("json.Unmarshal(rejected payload) error = %v", err)
	}
	if got, want := payload.Operation, "start"; got != want {
		t.Fatalf("payload.Operation = %q, want %q", got, want)
	}
	if got, want := payload.Reason, "stale_network_channel"; got != want {
		t.Fatalf("payload.Reason = %q, want %q", got, want)
	}
	if got, want := payload.NetworkChannel, "legacy"; got != want {
		t.Fatalf("payload.NetworkChannel = %q, want %q", got, want)
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
	if _, err := manager.ClaimRun(
		context.Background(),
		blockedRun.ID,
		ClaimRun{},
		actor,
	); !errors.Is(
		err,
		ErrInvalidStatusTransition,
	) {
		t.Fatalf("ClaimRun(blocked target) error = %v, want %v", err, ErrInvalidStatusTransition)
	}

	failingTask, err := manager.CreateTask(context.Background(), CreateTask{
		Scope:       ScopeGlobal,
		Title:       "Failing start task",
		MaxAttempts: ptr(1),
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

	if !hasOpenRun([]Run{{Status: TaskRunStatusQueued}}) {
		t.Fatal("hasOpenRun(queued) = false, want true")
	}
	if hasOpenRun([]Run{{Status: TaskRunStatusCompleted}}) {
		t.Fatal("hasOpenRun(completed) = true, want false")
	}
	if !runComesAfter(
		Run{ID: "run-2", Attempt: 2, QueuedAt: time.Date(2026, 4, 14, 16, 0, 0, 0, time.UTC)},
		Run{ID: "run-1", Attempt: 1, QueuedAt: time.Date(2026, 4, 14, 15, 0, 0, 0, time.UTC)},
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
	if err := manager.waitAndForceStopRun(cancelledCtx, "sess-canceled"); err == nil {
		t.Fatal("waitAndForceStopRun(canceled) error = nil, want non-nil")
	}
}

func TestTaskStatusFromSnapshot(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 14, 16, 0, 0, 0, time.UTC)
	tests := []struct {
		name                   string
		currentStatus          Status
		unresolvedDependencies bool
		runs                   []Run
		want                   Status
	}{
		{
			name:          "canceled task stays canceled",
			currentStatus: TaskStatusCanceled,
			runs: []Run{{
				Status: TaskRunStatusRunning,
			}},
			want: TaskStatusCanceled,
		},
		{
			name:          "active run wins immediately",
			currentStatus: TaskStatusReady,
			runs: []Run{
				{Status: TaskRunStatusCompleted, Attempt: 1, QueuedAt: base},
				{Status: TaskRunStatusRunning, Attempt: 2, QueuedAt: base.Add(time.Second)},
			},
			want: TaskStatusInProgress,
		},
		{
			name:                   "queued run with unresolved dependency is blocked",
			currentStatus:          TaskStatusReady,
			unresolvedDependencies: true,
			runs: []Run{{
				Status: TaskRunStatusQueued,
			}},
			want: TaskStatusBlocked,
		},
		{
			name:          "queued run without unresolved dependency is ready",
			currentStatus: TaskStatusBlocked,
			runs: []Run{{
				Status: TaskRunStatusClaimed,
			}},
			want: TaskStatusReady,
		},
		{
			name:          "latest completed terminal run wins",
			currentStatus: TaskStatusReady,
			runs: []Run{
				{ID: "run-1", Status: TaskRunStatusFailed, Attempt: 1, QueuedAt: base},
				{ID: "run-2", Status: TaskRunStatusCompleted, Attempt: 2, QueuedAt: base.Add(time.Second)},
			},
			want: TaskStatusCompleted,
		},
		{
			name:                   "no runs with unresolved dependency is blocked",
			currentStatus:          TaskStatusReady,
			unresolvedDependencies: true,
			want:                   TaskStatusBlocked,
		},
		{
			name:          "terminal status with no runs is preserved",
			currentStatus: TaskStatusFailed,
			want:          TaskStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := taskStatusFromSnapshot(tt.currentStatus, tt.unresolvedDependencies, tt.runs); got != tt.want {
				t.Fatalf("taskStatusFromSnapshot() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeRawJSONAndSameRawJSONTrimWhitespace(t *testing.T) {
	t.Parallel()

	trimmed := json.RawMessage(`{"ok":true}`)
	spaced := json.RawMessage(" \n\t" + string(trimmed) + "\n\t ")

	if got := normalizeRawJSON(nil); got != nil {
		t.Fatalf("normalizeRawJSON(nil) = %q, want nil", string(got))
	}
	if got := normalizeRawJSON(json.RawMessage(" \n\t ")); got != nil {
		t.Fatalf("normalizeRawJSON(blank) = %q, want nil", string(got))
	}
	if got := normalizeRawJSON(spaced); string(got) != string(trimmed) {
		t.Fatalf("normalizeRawJSON(spaced) = %q, want %q", string(got), string(trimmed))
	}
	if !sameRawJSON(spaced, trimmed) {
		t.Fatal("sameRawJSON(spaced, trimmed) = false, want true")
	}
	if sameRawJSON(trimmed, json.RawMessage(`{"ok":false}`)) {
		t.Fatal("sameRawJSON(trimmed, changed) = true, want false")
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
		if _, err := manager.AttachRunSession(
			context.Background(),
			runTwo.ID,
			"sess-shared",
			actor,
		); !errors.Is(
			err,
			ErrSessionAlreadyBound,
		) {
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
		if _, err := managerWithoutExecutor.AttachRunSession(
			context.Background(),
			run.ID,
			"sess-1",
			actor,
		); !errors.Is(
			err,
			ErrValidation,
		) {
			t.Fatalf("AttachRunSession(no executor) error = %v, want %v", err, ErrValidation)
		}

		managerWithExecutor := newTaskManagerForTestWithOptions(
			t,
			newInMemoryManagerStore(),
			WithSessionExecutor(&recordingSessionExecutor{}),
		)
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
		if _, err := managerWithExecutor.AttachRunSession(
			context.Background(),
			run.ID,
			"",
			actor,
		); !errors.Is(
			err,
			ErrValidation,
		) {
			t.Fatalf("AttachRunSession(empty session id) error = %v, want %v", err, ErrValidation)
		}
		if _, err := managerWithExecutor.AttachRunSession(
			context.Background(),
			run.ID,
			"sess-2",
			actor,
		); !errors.Is(
			err,
			ErrSessionAttachNotAllowed,
		) {
			t.Fatalf("AttachRunSession(queued run) error = %v, want %v", err, ErrSessionAttachNotAllowed)
		}
	})
}

func TestManagerRecoverRunOnBoot(t *testing.T) {
	t.Parallel()

	daemonActor, err := DeriveDaemonActorContext("boot-recovery", "daemon.boot")
	if err != nil {
		t.Fatalf("DeriveDaemonActorContext() error = %v", err)
	}

	t.Run("claimed run requeues and records recovery event", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		manager := newTaskManagerForTest(t, store)
		actor := validActorContext()

		taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
			Scope: ScopeGlobal,
			Title: "Claimed recovery",
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

		recovered, err := manager.RecoverRunOnBoot(context.Background(), run.ID, RunBootRecovery{
			Action:       RunBootRecoveryRequeue,
			Reason:       "orphaned_on_boot",
			SessionState: "missing",
		}, daemonActor)
		if err != nil {
			t.Fatalf("RecoverRunOnBoot(requeue) error = %v", err)
		}
		if got, want := recovered.Status, TaskRunStatusQueued; got != want {
			t.Fatalf("recovered.Status = %q, want %q", got, want)
		}
		if recovered.ClaimedBy != nil {
			t.Fatalf("recovered.ClaimedBy = %#v, want nil", recovered.ClaimedBy)
		}
		if !store.runs[run.ID].ClaimedAt.IsZero() {
			t.Fatalf("store.runs[%q].ClaimedAt = %v, want zero", run.ID, store.runs[run.ID].ClaimedAt)
		}

		events, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: taskRecord.ID})
		if err != nil {
			t.Fatalf("ListTaskEvents() error = %v", err)
		}
		if !containsEventType(events, taskEventRunRecovered) {
			t.Fatalf("events = %#v, want %q", sortedEventTypes(events), taskEventRunRecovered)
		}
	})

	t.Run("starting run is promoted to running when session is live", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		executor := &recordingSessionExecutor{}
		manager := newTaskManagerForTestWithOptions(t, store, WithSessionExecutor(executor))
		actor := validActorContext()

		taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
			Scope: ScopeGlobal,
			Title: "Starting recovery",
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
		run, err = manager.AttachRunSession(context.Background(), run.ID, "sess-live", actor)
		if err != nil {
			t.Fatalf("AttachRunSession() error = %v", err)
		}

		recovered, err := manager.RecoverRunOnBoot(context.Background(), run.ID, RunBootRecovery{
			Action:       RunBootRecoveryMarkRunning,
			Reason:       "orphaned_on_boot",
			SessionState: "active",
		}, daemonActor)
		if err != nil {
			t.Fatalf("RecoverRunOnBoot(mark running) error = %v", err)
		}
		if got, want := recovered.Status, TaskRunStatusRunning; got != want {
			t.Fatalf("recovered.Status = %q, want %q", got, want)
		}
		if recovered.StartedAt.IsZero() {
			t.Fatal("recovered.StartedAt = zero, want recovery timestamp")
		}
	})

	t.Run("running run fails closed when the attached session is not live", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		executor := &recordingSessionExecutor{}
		manager := newTaskManagerForTestWithOptions(t, store, WithSessionExecutor(executor))
		actor := validActorContext()

		taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
			Scope: ScopeGlobal,
			Title: "Running recovery",
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
		run, err = manager.StartRun(context.Background(), run.ID, StartRun{}, actor)
		if err != nil {
			t.Fatalf("StartRun() error = %v", err)
		}

		recovered, err := manager.RecoverRunOnBoot(context.Background(), run.ID, RunBootRecovery{
			Action:       RunBootRecoveryFail,
			Reason:       "orphaned_on_boot",
			SessionState: "missing",
		}, daemonActor)
		if err != nil {
			t.Fatalf("RecoverRunOnBoot(fail) error = %v", err)
		}
		if got, want := recovered.Status, TaskRunStatusFailed; got != want {
			t.Fatalf("recovered.Status = %q, want %q", got, want)
		}
		if !strings.Contains(recovered.Error, "orphaned on boot") {
			t.Fatalf("recovered.Error = %q, want orphaned-on-boot detail", recovered.Error)
		}

		events, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: taskRecord.ID})
		if err != nil {
			t.Fatalf("ListTaskEvents() error = %v", err)
		}
		if !containsEventType(events, taskEventRunFailed) || !containsEventType(events, taskEventRunRecovered) {
			t.Fatalf(
				"events = %#v, want %q and %q",
				sortedEventTypes(events),
				taskEventRunFailed,
				taskEventRunRecovered,
			)
		}
	})

	t.Run("claimed run cannot recover to running without a session binding", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		manager := newTaskManagerForTest(t, store)
		actor := validActorContext()

		taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
			Scope: ScopeGlobal,
			Title: "Claimed without session",
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

		if _, err := manager.RecoverRunOnBoot(context.Background(), run.ID, RunBootRecovery{
			Action:       RunBootRecoveryMarkRunning,
			Reason:       "orphaned_on_boot",
			SessionState: "missing",
		}, daemonActor); !errors.Is(err, ErrInvalidStatusTransition) {
			t.Fatalf(
				"RecoverRunOnBoot(mark running without session) error = %v, want %v",
				err,
				ErrInvalidStatusTransition,
			)
		}
	})

	t.Run("running run remains unchanged when recovery confirms it is still live", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		executor := &recordingSessionExecutor{}
		manager := newTaskManagerForTestWithOptions(t, store, WithSessionExecutor(executor))
		actor := validActorContext()

		taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
			Scope: ScopeGlobal,
			Title: "Running still live",
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
		run, err = manager.StartRun(context.Background(), run.ID, StartRun{}, actor)
		if err != nil {
			t.Fatalf("StartRun() error = %v", err)
		}

		eventsBefore, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: taskRecord.ID})
		if err != nil {
			t.Fatalf("ListTaskEvents(before) error = %v", err)
		}

		recovered, err := manager.RecoverRunOnBoot(context.Background(), run.ID, RunBootRecovery{
			Action:       RunBootRecoveryMarkRunning,
			Reason:       "orphaned_on_boot",
			SessionState: "active",
		}, daemonActor)
		if err != nil {
			t.Fatalf("RecoverRunOnBoot(mark running while already running) error = %v", err)
		}
		if got, want := recovered.Status, TaskRunStatusRunning; got != want {
			t.Fatalf("recovered.Status = %q, want %q", got, want)
		}

		eventsAfter, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: taskRecord.ID})
		if err != nil {
			t.Fatalf("ListTaskEvents(after) error = %v", err)
		}
		if got, want := len(eventsAfter), len(eventsBefore); got != want {
			t.Fatalf("event count after noop recovery = %d, want %d", got, want)
		}
	})

	t.Run("starting run cannot be requeued on boot", func(t *testing.T) {
		t.Parallel()

		store := newInMemoryManagerStore()
		executor := &recordingSessionExecutor{}
		manager := newTaskManagerForTestWithOptions(t, store, WithSessionExecutor(executor))
		actor := validActorContext()

		taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
			Scope: ScopeGlobal,
			Title: "Starting cannot requeue",
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
		run, err = manager.AttachRunSession(context.Background(), run.ID, "sess-bound", actor)
		if err != nil {
			t.Fatalf("AttachRunSession() error = %v", err)
		}

		if _, err := manager.RecoverRunOnBoot(context.Background(), run.ID, RunBootRecovery{
			Action:       RunBootRecoveryRequeue,
			Reason:       "orphaned_on_boot",
			SessionState: "missing",
		}, daemonActor); !errors.Is(err, ErrInvalidStatusTransition) {
			t.Fatalf("RecoverRunOnBoot(requeue starting) error = %v, want %v", err, ErrInvalidStatusTransition)
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

func TestRunBootRecoveryHelpersAndWriteAuthority(t *testing.T) {
	t.Parallel()

	t.Run("formats recovery errors for bound and missing sessions", func(t *testing.T) {
		t.Parallel()

		if got, want := runBootRecoveryError(Run{
			ID:        "run-active",
			SessionID: "sess-active",
		}, RunBootRecovery{
			SessionState: "stopped",
		}), `orphaned on boot: session "sess-active" is stopped`; got != want {
			t.Fatalf("runBootRecoveryError(bound+state) = %q, want %q", got, want)
		}
		if got, want := runBootRecoveryError(Run{
			ID:        "run-bound",
			SessionID: "sess-bound",
		}, RunBootRecovery{}), `orphaned on boot: session "sess-bound" is not live`; got != want {
			t.Fatalf("runBootRecoveryError(bound) = %q, want %q", got, want)
		}
		if got, want := runBootRecoveryError(
			Run{ID: "run-missing"},
			RunBootRecovery{},
		), "orphaned on boot: run has no live session"; got != want {
			t.Fatalf("runBootRecoveryError(missing) = %q, want %q", got, want)
		}
	})

	t.Run("normalizes recovery metadata reason", func(t *testing.T) {
		t.Parallel()

		metadata := runBootRecoveryMetadata(Run{
			ID:        "run-meta",
			Status:    TaskRunStatusStarting,
			SessionID: "sess-meta",
		}, RunBootRecovery{
			Reason:       "   ",
			SessionState: "missing",
		})
		if metadata == nil {
			t.Fatal("runBootRecoveryMetadata() = nil, want payload")
		}

		var payload map[string]string
		if err := json.Unmarshal(metadata, &payload); err != nil {
			t.Fatalf("json.Unmarshal(metadata) error = %v", err)
		}
		if got, want := payload["reason"], "orphaned_on_boot"; got != want {
			t.Fatalf("payload[reason] = %q, want %q", got, want)
		}
		if got, want := payload["previous_status"], string(TaskRunStatusStarting); got != want {
			t.Fatalf("payload[previous_status] = %q, want %q", got, want)
		}
	})

	t.Run("write authority rejects read-only actors", func(t *testing.T) {
		t.Parallel()

		actor := validActorContext()
		actor.Authority.Write = false
		actor.Authority.CreateGlobal = false
		actor.Authority.CreateWorkspace = false
		if err := requireWriteAuthority(actor); !errors.Is(err, ErrPermissionDenied) {
			t.Fatalf("requireWriteAuthority(read-only) error = %v, want %v", err, ErrPermissionDenied)
		}
	})
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

func newTaskManagerForTest(t *testing.T, store Store) *Service {
	t.Helper()
	return newTaskManagerForTestWithOptions(t, store)
}

func newTaskManagerForTestWithOptions(t *testing.T, store Store, extraOpts ...Option) *Service {
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

func cloneTaskRun(record Run) Run {
	cloned := record
	if record.ClaimedBy != nil {
		claimedBy := *record.ClaimedBy
		cloned.ClaimedBy = &claimedBy
	}
	cloned.Metadata = cloneRawJSON(record.Metadata)
	cloned.Result = cloneRawJSON(record.Result)
	return cloned
}

func inMemoryTaskLatestActivity(summary Summary, runs map[string]Run, events []Event) time.Time {
	taskRuns := make([]Run, 0)
	for _, run := range runs {
		if run.TaskID == summary.ID {
			taskRuns = append(taskRuns, run)
		}
	}
	taskEvents := make([]Event, 0)
	for _, event := range events {
		if event.TaskID == summary.ID {
			taskEvents = append(taskEvents, event)
		}
	}
	return latestTaskActivityAt(taskRecordFromSummary(summary), taskRuns, taskEvents)
}

func cloneTriageState(record TriageState) TriageState {
	cloned := record
	cloned.Actor = record.Actor
	return cloned
}

func sortedEventTypes(events []Event) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.EventType)
	}
	sort.Strings(types)
	return types
}

func equalStringSlices(left []string, right []string) bool {
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

func equalInt64Slices(left []int64, right []int64) bool {
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

func awaitTaskStreamEvent(t *testing.T, stream <-chan StreamEvent) StreamEvent {
	t.Helper()

	select {
	case event, ok := <-stream:
		if !ok {
			t.Fatal("stream closed before event was available")
		}
		return event
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for task stream event")
		return StreamEvent{}
	}
}

func assertNoTaskStreamEvent(t *testing.T, stream <-chan StreamEvent, wait time.Duration) {
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

func containsEventType(events []Event, want string) bool {
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

func triageKey(taskID string, actor ActorIdentity) string {
	return strings.TrimSpace(taskID) + "|" + string(actor.Kind.Normalize()) + "|" + strings.TrimSpace(actor.Ref)
}

func fmtTestError(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
