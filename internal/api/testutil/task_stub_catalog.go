package testutil

import (
	"context"

	taskpkg "github.com/compozy/agh/internal/task"
)

// ListTasks exposes the unpaged task-list seam used by internal lifecycle tests.
func (s *StubTaskManager) ListTasks(
	ctx context.Context,
	query taskpkg.Query,
	actor taskpkg.ActorContext,
) ([]taskpkg.Summary, error) {
	if s.ListTasksFn != nil {
		return s.ListTasksFn(ctx, query, actor)
	}
	return nil, nil
}

// ListTaskCatalog exposes the bounded catalog seam through the most specific configured callback.
func (s *StubTaskManager) ListTaskCatalog(
	ctx context.Context,
	query taskpkg.CatalogQuery,
	actor taskpkg.ActorContext,
) (taskpkg.CatalogPage, error) {
	if s.ListTaskCatalogFn != nil {
		return s.ListTaskCatalogFn(ctx, query, actor)
	}
	if s.ListTasksFn == nil {
		return taskpkg.CatalogPage{Limit: query.Limit}, nil
	}
	tasks, err := s.ListTasksFn(ctx, taskpkg.Query{
		Scope:          taskpkg.Scope(query.Scope),
		WorkspaceID:    query.WorkspaceID,
		Status:         query.Status,
		Priority:       query.Priority,
		ApprovalState:  query.ApprovalState,
		OwnerKind:      query.OwnerKind,
		OwnerRef:       query.OwnerRef,
		ParentTaskID:   query.ParentTaskID,
		NetworkChannel: query.NetworkChannel,
		Search:         query.Search,
		Limit:          query.Limit,
	}, actor)
	if err != nil {
		return taskpkg.CatalogPage{}, err
	}
	return taskpkg.CatalogPage{Tasks: tasks, Total: len(tasks), Limit: query.Limit}, nil
}
