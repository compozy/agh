//go:build integration

package task_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	taskpkg "github.com/pedronauck/agh/internal/task"
)

type fakeStore struct{}

func (fakeStore) CreateTask(context.Context, taskpkg.Task) error { return nil }

func (fakeStore) UpdateTask(context.Context, taskpkg.Task) error { return nil }

func (fakeStore) GetTask(context.Context, string) (taskpkg.Task, error) { return taskpkg.Task{}, nil }

func (fakeStore) ListTasks(context.Context, taskpkg.Query) ([]taskpkg.Summary, error) {
	return []taskpkg.Summary{{
		ID:             "task-1",
		Title:          "bootstrap",
		Scope:          taskpkg.ScopeGlobal,
		Priority:       taskpkg.PriorityMedium,
		MaxAttempts:    taskpkg.DefaultTaskMaxAttempts,
		ApprovalPolicy: taskpkg.ApprovalPolicyManual,
		ApprovalState:  taskpkg.ApprovalStatePending,
	}}, nil
}

func (fakeStore) CountDirectChildren(context.Context, string) (int, error) { return 0, nil }

func (fakeStore) CreateDependency(context.Context, taskpkg.Dependency) error { return nil }

func (fakeStore) DeleteDependency(context.Context, string, string) error { return nil }

func (fakeStore) ListDependencies(context.Context, string) ([]taskpkg.Dependency, error) {
	return []taskpkg.Dependency{{TaskID: "task-1", DependsOnTaskID: "task-0", Kind: taskpkg.DependencyKindBlocks}}, nil
}

func (fakeStore) ListDependents(context.Context, string) ([]taskpkg.Dependency, error) {
	return nil, nil
}

func (fakeStore) CountDependencies(context.Context, string) (int, error) { return 1, nil }

func (fakeStore) HasDependencyPath(context.Context, string, string) (bool, error) { return false, nil }

func (fakeStore) CreateTaskRun(context.Context, taskpkg.Run) error { return nil }

func (fakeStore) UpdateTaskRun(context.Context, taskpkg.Run) error { return nil }

func (fakeStore) GetTaskRun(context.Context, string) (taskpkg.Run, error) {
	return taskpkg.Run{}, nil
}

func (fakeStore) ListTaskRuns(context.Context, taskpkg.RunQuery) ([]taskpkg.Run, error) {
	return []taskpkg.Run{{ID: "run-1", TaskID: "task-1", Status: taskpkg.TaskRunStatusQueued, Attempt: 1}}, nil
}

func (fakeStore) ListTaskRunsByStatus(context.Context, []taskpkg.RunStatus) ([]taskpkg.Run, error) {
	return []taskpkg.Run{{ID: "run-1", TaskID: "task-1", Status: taskpkg.TaskRunStatusQueued, Attempt: 1}}, nil
}

func (fakeStore) ReserveQueuedRun(
	context.Context,
	string,
	string,
	string,
	taskpkg.Origin,
	string,
	json.RawMessage,
	time.Time,
) (taskpkg.Task, taskpkg.Run, bool, error) {
	return taskpkg.Task{
			ID:             "task-1",
			Scope:          taskpkg.ScopeGlobal,
			Title:          "bootstrap",
			Priority:       taskpkg.PriorityMedium,
			MaxAttempts:    taskpkg.DefaultTaskMaxAttempts,
			Status:         taskpkg.TaskStatusReady,
			ApprovalPolicy: taskpkg.ApprovalPolicyManual,
			ApprovalState:  taskpkg.ApprovalStatePending,
		}, taskpkg.Run{
			ID:      "run-1",
			TaskID:  "task-1",
			Status:  taskpkg.TaskRunStatusQueued,
			Attempt: 1,
		}, false, nil
}

func (fakeStore) GetTaskTriageState(context.Context, string, taskpkg.ActorIdentity) (taskpkg.TriageState, error) {
	return taskpkg.TriageState{}, taskpkg.ErrTaskTriageStateNotFound
}

func (fakeStore) UpsertTaskTriageState(context.Context, taskpkg.TriageState) error { return nil }

func (fakeStore) CountActiveSessionBindings(context.Context, string) (int, error) { return 0, nil }

func (fakeStore) CreateTaskEvent(context.Context, taskpkg.Event) error { return nil }

func (fakeStore) ListTaskEvents(context.Context, taskpkg.EventQuery) ([]taskpkg.Event, error) {
	return []taskpkg.Event{{ID: "evt-1", TaskID: "task-1", EventType: "task.created"}}, nil
}

func (fakeStore) GetTaskEventRecord(context.Context, string) (taskpkg.EventRecord, error) {
	return taskpkg.EventRecord{
		Sequence: 1,
		Event:    taskpkg.Event{ID: "evt-1", TaskID: "task-1", EventType: "task.created"},
	}, nil
}

func (fakeStore) ListTaskEventRecords(
	context.Context,
	taskpkg.EventRecordQuery,
) ([]taskpkg.EventRecord, error) {
	return []taskpkg.EventRecord{{
		Sequence: 1,
		Event:    taskpkg.Event{ID: "evt-1", TaskID: "task-1", EventType: "task.created"},
	}}, nil
}

func (fakeStore) GetTaskRunByIdempotencyKey(context.Context, string, taskpkg.Origin) (taskpkg.Run, error) {
	return taskpkg.Run{}, nil
}

func (fakeStore) SaveTaskRunIdempotency(context.Context, taskpkg.RunIdempotency) error {
	return nil
}

type fakeSessionExecutor struct{}

func (fakeSessionExecutor) StartTaskSession(context.Context, *taskpkg.StartTaskSession) (*taskpkg.SessionRef, error) {
	return &taskpkg.SessionRef{SessionID: "sess-1"}, nil
}

func (fakeSessionExecutor) AttachTaskSession(context.Context, string, string) (*taskpkg.SessionRef, error) {
	return &taskpkg.SessionRef{SessionID: "sess-1"}, nil
}

func (fakeSessionExecutor) RequestTaskStop(context.Context, string, taskpkg.StopReason) error {
	return nil
}

func (fakeSessionExecutor) ForceTaskStop(context.Context, string, taskpkg.StopReason) error {
	return nil
}

type fakeCoordinator struct {
	store    taskpkg.Store
	sessions taskpkg.SessionExecutor
}

func (c fakeCoordinator) compose(ctx context.Context) error {
	if _, err := c.store.ListTasks(ctx, taskpkg.Query{
		Limit:         1,
		Priority:      taskpkg.PriorityMedium,
		ApprovalState: taskpkg.ApprovalStatePending,
	}); err != nil {
		return err
	}
	if err := c.sessions.RequestTaskStop(ctx, "sess-1", taskpkg.StopReasonCancellation); err != nil {
		return err
	}
	return nil
}

var _ taskpkg.Store = (*fakeStore)(nil)
var _ taskpkg.SessionExecutor = (*fakeSessionExecutor)(nil)

func TestTaskDomainInterfacesComposeWithoutSessionImport(t *testing.T) {
	t.Parallel()

	coordinator := fakeCoordinator{
		store:    fakeStore{},
		sessions: fakeSessionExecutor{},
	}
	if err := coordinator.compose(context.Background()); err != nil {
		t.Fatalf("compose() error = %v", err)
	}
}
