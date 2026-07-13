package testutil

import (
	"context"
	"strings"

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
	limit := query.Limit
	if strings.TrimSpace(query.NetworkChannel) != "" {
		limit = 0
	}
	tasks, err := s.ListTasksFn(ctx, taskpkg.Query{
		Scope:         taskpkg.Scope(query.Scope),
		WorkspaceID:   query.WorkspaceID,
		Status:        query.Status,
		Priority:      query.Priority,
		ApprovalState: query.ApprovalState,
		OwnerKind:     query.OwnerKind,
		OwnerRef:      query.OwnerRef,
		ParentTaskID:  query.ParentTaskID,
		Search:        query.Search,
		Limit:         limit,
	}, actor)
	if err != nil {
		return taskpkg.CatalogPage{}, err
	}
	channel := strings.TrimSpace(query.NetworkChannel)
	if channel != "" {
		filtered := make([]taskpkg.Summary, 0, len(tasks))
		for index := range tasks {
			run := tasks[index].ActiveRun
			if run != nil && run.ResolvedNetworkParticipation != nil &&
				strings.TrimSpace(run.ResolvedNetworkParticipation.ChannelID) == channel {
				filtered = append(filtered, tasks[index])
			}
		}
		tasks = filtered
	}
	total := len(tasks)
	if query.Limit > 0 && len(tasks) > query.Limit {
		tasks = tasks[:query.Limit]
	}
	return taskpkg.CatalogPage{Tasks: tasks, Total: total, Limit: query.Limit}, nil
}
