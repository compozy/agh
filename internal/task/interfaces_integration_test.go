//go:build integration

package task_test

import (
	"context"
	"testing"

	taskpkg "github.com/pedronauck/agh/internal/task"
)

type fakeStore struct{}

func (fakeStore) CreateTask(context.Context, taskpkg.Task) error { return nil }

func (fakeStore) UpdateTask(context.Context, taskpkg.Task) error { return nil }

func (fakeStore) GetTask(context.Context, string) (taskpkg.Task, error) { return taskpkg.Task{}, nil }

func (fakeStore) ListTasks(context.Context, taskpkg.TaskQuery) ([]taskpkg.TaskSummary, error) {
	return []taskpkg.TaskSummary{{ID: "task-1", Title: "bootstrap", Scope: taskpkg.ScopeGlobal}}, nil
}

func (fakeStore) CountDirectChildren(context.Context, string) (int, error) { return 0, nil }

func (fakeStore) CreateDependency(context.Context, taskpkg.TaskDependency) error { return nil }

func (fakeStore) DeleteDependency(context.Context, string, string) error { return nil }

func (fakeStore) ListDependencies(context.Context, string) ([]taskpkg.TaskDependency, error) {
	return []taskpkg.TaskDependency{{TaskID: "task-1", DependsOnTaskID: "task-0", Kind: taskpkg.DependencyKindBlocks}}, nil
}

func (fakeStore) CountDependencies(context.Context, string) (int, error) { return 1, nil }

func (fakeStore) HasDependencyPath(context.Context, string, string) (bool, error) { return false, nil }

func (fakeStore) CreateTaskRun(context.Context, taskpkg.TaskRun) error { return nil }

func (fakeStore) UpdateTaskRun(context.Context, taskpkg.TaskRun) error { return nil }

func (fakeStore) GetTaskRun(context.Context, string) (taskpkg.TaskRun, error) {
	return taskpkg.TaskRun{}, nil
}

func (fakeStore) ListTaskRuns(context.Context, taskpkg.TaskRunQuery) ([]taskpkg.TaskRun, error) {
	return []taskpkg.TaskRun{{ID: "run-1", TaskID: "task-1", Status: taskpkg.TaskRunStatusQueued, Attempt: 1}}, nil
}

func (fakeStore) ListTaskRunsByStatus(context.Context, []taskpkg.TaskRunStatus) ([]taskpkg.TaskRun, error) {
	return []taskpkg.TaskRun{{ID: "run-1", TaskID: "task-1", Status: taskpkg.TaskRunStatusQueued, Attempt: 1}}, nil
}

func (fakeStore) CountActiveSessionBindings(context.Context, string) (int, error) { return 0, nil }

func (fakeStore) CreateTaskEvent(context.Context, taskpkg.TaskEvent) error { return nil }

func (fakeStore) ListTaskEvents(context.Context, taskpkg.TaskEventQuery) ([]taskpkg.TaskEvent, error) {
	return []taskpkg.TaskEvent{{ID: "evt-1", TaskID: "task-1", EventType: "task.created"}}, nil
}

func (fakeStore) GetTaskRunByIdempotencyKey(context.Context, string, taskpkg.Origin) (taskpkg.TaskRun, error) {
	return taskpkg.TaskRun{}, nil
}

func (fakeStore) SaveTaskRunIdempotency(context.Context, taskpkg.TaskRunIdempotency) error {
	return nil
}

type fakeSessionExecutor struct{}

func (fakeSessionExecutor) StartTaskSession(context.Context, taskpkg.StartTaskSession) (*taskpkg.SessionRef, error) {
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
	if _, err := c.store.ListTasks(ctx, taskpkg.TaskQuery{Limit: 1}); err != nil {
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
