package testutil

import (
	"context"

	core "github.com/compozy/agh/internal/api/core"
	hookspkg "github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/observe"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

type StubObserver struct {
	QueryEventsFn        func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error)
	QueryHookCatalogFn   func(context.Context, hookspkg.CatalogFilter) ([]hookspkg.CatalogEntry, error)
	QueryHookRunsFn      func(context.Context, store.HookRunQuery) ([]hookspkg.HookRunRecord, error)
	QueryHookEventsFn    func(context.Context, hookspkg.EventFilter) ([]hookspkg.EventDescriptor, error)
	QueryBridgeHealthFn  func(context.Context) ([]observe.BridgeInstanceHealth, error)
	HealthFn             func(context.Context) (observe.Health, error)
	QueryTaskDashboardFn func(context.Context, observe.TaskDashboardQuery) (observe.TaskDashboardView, error)
	QueryTaskInboxFn     func(
		context.Context,
		observe.TaskInboxQuery,
		taskpkg.ActorIdentity,
	) (observe.TaskInboxView, error)
}

func (s StubObserver) QueryEvents(ctx context.Context, query store.EventSummaryQuery) ([]store.EventSummary, error) {
	if s.QueryEventsFn != nil {
		return s.QueryEventsFn(ctx, query)
	}
	return nil, nil
}

func (s StubObserver) QueryTaskDashboard(
	ctx context.Context,
	query observe.TaskDashboardQuery,
) (observe.TaskDashboardView, error) {
	if s.QueryTaskDashboardFn != nil {
		return s.QueryTaskDashboardFn(ctx, query)
	}
	return observe.TaskDashboardView{}, nil
}

func (s StubObserver) QueryTaskInbox(
	ctx context.Context,
	query observe.TaskInboxQuery,
	actor taskpkg.ActorIdentity,
) (observe.TaskInboxView, error) {
	if s.QueryTaskInboxFn != nil {
		return s.QueryTaskInboxFn(ctx, query, actor)
	}
	return observe.TaskInboxView{}, nil
}

func (s StubObserver) Health(ctx context.Context) (observe.Health, error) {
	if s.HealthFn != nil {
		return s.HealthFn(ctx)
	}
	return observe.Health{Status: "ok"}, nil
}

func (s StubObserver) QueryBridgeHealth(ctx context.Context) ([]observe.BridgeInstanceHealth, error) {
	if s.QueryBridgeHealthFn != nil {
		return s.QueryBridgeHealthFn(ctx)
	}
	return nil, nil
}

func (s StubObserver) QueryHookCatalog(
	ctx context.Context,
	filter hookspkg.CatalogFilter,
) ([]hookspkg.CatalogEntry, error) {
	if s.QueryHookCatalogFn != nil {
		return s.QueryHookCatalogFn(ctx, filter)
	}
	return nil, nil
}

func (s StubObserver) QueryHookRuns(ctx context.Context, query store.HookRunQuery) ([]hookspkg.HookRunRecord, error) {
	if s.QueryHookRunsFn != nil {
		return s.QueryHookRunsFn(ctx, query)
	}
	return nil, nil
}

func (s StubObserver) QueryHookEvents(
	ctx context.Context,
	filter hookspkg.EventFilter,
) ([]hookspkg.EventDescriptor, error) {
	if s.QueryHookEventsFn != nil {
		return s.QueryHookEventsFn(ctx, filter)
	}
	return nil, nil
}

var _ core.Observer = (*StubObserver)(nil)
