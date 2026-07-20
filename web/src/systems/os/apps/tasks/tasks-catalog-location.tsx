import { Link } from "@tanstack/react-router";
import { AlertCircle, Plus } from "lucide-react";

import { BlockLoading, Button, Empty, RouteNav, useTopbarSlot } from "@agh/ui";

import {
  TasksDashboardView,
  TasksEmptyState,
  TasksInboxView,
  TasksKanbanBoard,
  TasksListSurface,
  useTasksPage,
  type TaskViewMode,
} from "@/systems/tasks";

import { useOsShell } from "../../hooks/use-os-shell";

const TASK_MODE_ITEMS: ReadonlyArray<{
  value: TaskViewMode;
  label: string;
  testId: string;
}> = [
  { value: "list", label: "List", testId: "tasks-mode-list" },
  { value: "kanban", label: "Kanban", testId: "tasks-mode-kanban" },
  { value: "dashboard", label: "Dashboard", testId: "tasks-mode-dashboard" },
  { value: "inbox", label: "Inbox", testId: "tasks-mode-inbox" },
];

export function TasksCatalogLocation({ mode }: { mode: TaskViewMode }) {
  const { coordinator } = useOsShell();
  const page = useTasksPage({ mode });
  const navigate = (pathname: string, search: Record<string, unknown> = {}) =>
    coordinator.userOpen({ app: "tasks", location: { pathname, search } });
  const openCreate = () => navigate("/tasks/new");

  useTopbarSlot({
    routeNav: (
      <RouteNav aria-label="Tasks views" data-testid="tasks-mode-nav">
        {TASK_MODE_ITEMS.map(item => (
          <RouteNav.Link
            aria-current={item.value === mode ? "page" : undefined}
            data-testid={item.testId}
            key={item.value}
            render={
              <Link
                activeOptions={{ exact: true, includeSearch: true }}
                onClick={() => {
                  if (item.value !== mode) page.setSearchQuery("");
                }}
                search={item.value === "list" ? {} : { mode: item.value }}
                to="/tasks"
              />
            }
          >
            {item.label}
            {item.value === "inbox" && page.inboxUnreadCount ? (
              <RouteNav.Count data-testid="tasks-mode-inbox-count">
                {page.inboxUnreadCount}
              </RouteNav.Count>
            ) : null}
          </RouteNav.Link>
        ))}
      </RouteNav>
    ),
    actions: (
      <Button
        data-testid="tasks-open-create"
        disabled={!page.hasActiveTaskScope}
        onClick={openCreate}
        size="sm"
        type="button"
        variant="outline"
      >
        <Plus className="size-3" />
        Task
      </Button>
    ),
  });

  return (
    <div
      className="flex min-h-0 flex-1 flex-col overflow-hidden"
      data-density="route"
      data-testid="tasks-shell"
    >
      {page.scopeLoading ? (
        <BlockLoading
          data-testid="tasks-scope-loading"
          label="Resolving task scope"
          size="md"
          surface="bare"
        />
      ) : page.scopeError ? (
        <Empty
          data-testid="tasks-scope-error"
          description={page.scopeError.message}
          icon={AlertCircle}
          title="Unable to resolve task scope"
        />
      ) : mode === "dashboard" ? (
        <TasksDashboardView
          dashboard={page.dashboard}
          errorMessage={page.dashboardError?.message ?? null}
          dashboardStatus={page.dashboardLoading ? "loading" : "ready"}
          scheduler={page.schedulerStatus}
          schedulerBacklog={page.schedulerBacklog}
          schedulerBacklogErrorMessage={page.schedulerBacklogError?.message ?? null}
          schedulerBacklogStatus={page.schedulerBacklogLoading ? "loading" : "ready"}
          schedulerErrorMessage={page.schedulerStatusError?.message ?? null}
          schedulerStatus={page.schedulerStatusLoading ? "loading" : "ready"}
          schedulerPendingActions={
            new Set(
              [
                page.isSchedulerDrainPending ? "drain" : null,
                page.isSchedulerPausePending ? "pause" : null,
                page.isSchedulerResumePending ? "resume" : null,
              ].filter((action): action is "drain" | "pause" | "resume" => action !== null)
            )
          }
          onDrainScheduler={page.handleDrainScheduler}
          onPauseScheduler={page.handlePauseScheduler}
          onResumeScheduler={page.handleResumeScheduler}
        />
      ) : mode === "inbox" ? (
        <TasksInboxView
          errorMessage={page.inboxError?.message ?? null}
          inbox={page.inbox}
          inboxUpdatedAt={page.inboxUpdatedAt}
          hasMore={page.hasMoreInbox}
          isLoading={page.inboxLoading}
          isLoadingMore={page.isLoadingMoreInbox}
          laneFilter={page.inboxLaneFilter}
          onApprove={page.handleApproveTask}
          onArchive={page.handleArchiveTask}
          onDismiss={page.handleDismissTask}
          onLaneChange={page.handleInboxLaneChange}
          onMarkRead={page.handleMarkTaskRead}
          onLoadMore={page.loadMoreInbox}
          onPriorityChange={page.handleInboxPriorityChange}
          onReject={page.handleRejectTask}
          onRetry={page.handleRetryRun}
          onRetryQuery={page.retryInbox}
          onSearchChange={page.setInboxSearchQuery}
          onStatusChange={page.handleInboxStatusChange}
          onToggleUnread={page.handleInboxUnreadToggle}
          priorityFilter={page.inboxPriorityFilter}
          searchQuery={page.inboxSearchQuery}
          statusFilter={page.inboxStatusFilter}
          unreadOnly={page.inboxUnreadOnly}
          workspaceName={page.activeWorkspaceName}
          pendingApproveIds={page.pendingApproveIds}
          pendingArchiveIds={page.pendingArchiveIds}
          pendingDismissIds={page.pendingDismissIds}
          pendingMarkReadIds={page.pendingMarkReadIds}
          pendingRejectIds={page.pendingRejectIds}
          pendingRetryIds={page.pendingRetryIds}
        />
      ) : page.isEmpty ? (
        <TasksEmptyState
          onSelectTemplate={templateId =>
            navigate("/tasks/new", templateId === "one_shot" ? {} : { template: templateId })
          }
          workspaceName={page.activeWorkspaceName}
        />
      ) : mode === "kanban" ? (
        <TasksKanbanBoard
          columns={page.kanbanColumns}
          errorMessage={page.listError?.message ?? null}
          hasMore={page.hasMoreTasks}
          isLoading={page.listLoading}
          isLoadingMore={page.isLoadingMoreTasks}
          onCreate={openCreate}
          onRetryLoad={page.retryTasks}
          onRetryTask={page.handleRetryRun}
          onSelectTask={taskId => navigate(`/tasks/${encodeURIComponent(taskId)}`)}
          onLoadMore={page.loadMoreTasks}
          selectedTaskId={page.effectiveSelectedTaskId}
          statusCounts={page.statusCounts}
        />
      ) : (
        <TasksListSurface
          errorMessage={page.listError?.message ?? null}
          isLoading={page.listLoading}
          hasMore={page.hasMoreTasks}
          isLoadingMore={page.isLoadingMoreTasks}
          listUpdatedAt={page.listUpdatedAt}
          onOwnerChange={page.handleOwnerChange}
          onLoadMore={page.loadMoreTasks}
          onRetryLoad={page.retryTasks}
          onPriorityChange={page.handlePriorityChange}
          onSortChange={page.handleSortChange}
          onStatusChange={page.handleStatusChange}
          onSearchQueryChange={page.setSearchQuery}
          ownerFilter={page.ownerFilter}
          ownerOptions={page.ownerOptions}
          priorityFilter={page.priorityFilter}
          searchQuery={page.searchQuery}
          sortBy={page.sortBy}
          statusFilter={page.statusFilter}
          statusCounts={page.statusCounts}
          tasks={page.visibleTasks}
          totalCount={page.tasksCount}
          workspaceName={page.activeWorkspaceName}
        />
      )}
    </div>
  );
}
