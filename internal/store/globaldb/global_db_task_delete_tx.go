package globaldb

import (
	"context"

	taskpkg "github.com/compozy/agh/internal/task"
)

type deleteTaskTxStore struct {
	global *GlobalDB
	exec   taskSQLExecutor
}

var _ taskpkg.DeleteTaskMutationStore = (*deleteTaskTxStore)(nil)

func (s *deleteTaskTxStore) GetTask(ctx context.Context, id string) (taskpkg.Task, error) {
	return s.global.getTaskWithExecutor(ctx, s.exec, id)
}

func (s *deleteTaskTxStore) UpdateTask(ctx context.Context, record taskpkg.Task, actor taskpkg.ActorContext) error {
	return s.global.updateTaskWithExecutor(ctx, s.exec, record, actor)
}

func (s *deleteTaskTxStore) DeleteTask(ctx context.Context, id string) error {
	return s.global.deleteTaskWithExecutor(ctx, s.exec, id)
}

func (s *deleteTaskTxStore) CountDirectChildren(
	ctx context.Context,
	parentTaskID string,
) (int, error) {
	return s.global.countDirectChildrenWithExecutor(ctx, s.exec, parentTaskID)
}

func (s *deleteTaskTxStore) ListDependencies(
	ctx context.Context,
	taskID string,
) ([]taskpkg.Dependency, error) {
	return s.global.listDependenciesWithExecutor(ctx, s.exec, taskID)
}

func (s *deleteTaskTxStore) ListTaskBlocks(
	ctx context.Context,
	taskID string,
	includeCleared bool,
) ([]taskpkg.TaskBlock, error) {
	return s.global.listTaskBlocksWithExecutor(ctx, s.exec, taskID, includeCleared, s.global.now())
}

func (s *deleteTaskTxStore) HasOpenTaskBlocks(ctx context.Context, taskID string) (bool, error) {
	return s.global.hasOpenTaskBlocksWithExecutor(ctx, s.exec, taskID, s.global.now())
}

func (s *deleteTaskTxStore) ListDependents(
	ctx context.Context,
	dependsOnTaskID string,
) ([]taskpkg.Dependency, error) {
	return s.global.listDependentsWithExecutor(ctx, s.exec, dependsOnTaskID)
}

func (s *deleteTaskTxStore) ListTaskRuns(
	ctx context.Context,
	query taskpkg.RunQuery,
) ([]taskpkg.Run, error) {
	return s.global.listTaskRunsWithExecutor(ctx, s.exec, query)
}
