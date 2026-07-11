import { Outlet, useNavigate } from "@tanstack/react-router";
import { AlertCircle, Plus } from "lucide-react";

import { useTasksRoute } from "@/hooks/routes/use-tasks-route";
import {
  TasksDashboardView,
  TasksEmptyState,
  TasksInboxView,
  TasksKanbanBoard,
  TasksListSurface,
} from "@/systems/tasks";
import { BlockLoading, Button, Empty, PillGroup, useTopbarSlot } from "@agh/ui";

export function TasksRoute() {
  const view = useTasksRoute();
  const navigate = useNavigate({ from: "/tasks" });
  const {
    page,
    hasChildMatch,
    routedTaskId,
    isCreateRoute,
    surfaceMode,
    shellCount,
    handleModeSelect,
    openCreateRoute,
  } = view;

  const handleSelectTask = (taskId: string) => {
    void navigate({ params: { id: taskId }, to: "/tasks/$id" });
  };

  useTopbarSlot({
    count: shellCount,
    tabs: (
      <PillGroup
        data-testid="tasks-mode-pills"
        value={surfaceMode}
        onChange={handleModeSelect}
        items={[
          { value: "list", label: "List", testId: "tasks-mode-list" },
          { value: "kanban", label: "Kanban", testId: "tasks-mode-kanban" },
          { value: "dashboard", label: "Dashboard", testId: "tasks-mode-dashboard" },
          {
            value: "inbox",
            label: "Inbox",
            badge: page.inboxUnreadCount,
            testId: "tasks-mode-inbox",
          },
        ]}
      />
    ),
    actions: (
      <Button
        data-testid="tasks-open-create"
        disabled={isCreateRoute || !page.hasActiveTaskScope}
        onClick={openCreateRoute}
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
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="tasks-shell-body">
        {hasChildMatch ? (
          <Outlet />
        ) : page.scopeLoading ? (
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
        ) : surfaceMode === "dashboard" ? (
          <TasksDashboardView
            dashboard={page.dashboard}
            errorMessage={page.dashboardError?.message ?? null}
            isLoading={page.dashboardLoading}
            scheduler={page.schedulerStatus}
            schedulerBacklog={page.schedulerBacklog}
            schedulerBacklogErrorMessage={page.schedulerBacklogError?.message ?? null}
            schedulerBacklogLoading={page.schedulerBacklogLoading}
            schedulerErrorMessage={page.schedulerStatusError?.message ?? null}
            schedulerLoading={page.schedulerStatusLoading}
            isSchedulerDrainPending={page.isSchedulerDrainPending}
            isSchedulerPausePending={page.isSchedulerPausePending}
            isSchedulerResumePending={page.isSchedulerResumePending}
            onDrainScheduler={page.handleDrainScheduler}
            onPauseScheduler={page.handlePauseScheduler}
            onResumeScheduler={page.handleResumeScheduler}
          />
        ) : surfaceMode === "inbox" ? (
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
            onSelectTemplate={templateId => {
              void navigate({
                search: () =>
                  templateId === "one_shot" ? { template: undefined } : { template: templateId },
                to: "/tasks/new",
              });
            }}
            workspaceName={page.activeWorkspaceName}
          />
        ) : surfaceMode === "kanban" ? (
          <TasksKanbanBoard
            columns={page.kanbanColumns}
            errorMessage={page.listError?.message ?? null}
            hasMore={page.hasMoreTasks}
            isLoading={page.listLoading}
            isLoadingMore={page.isLoadingMoreTasks}
            onCreate={openCreateRoute}
            onRetryLoad={page.retryTasks}
            onRetryTask={page.handleRetryRun}
            onSelectTask={handleSelectTask}
            onLoadMore={page.loadMoreTasks}
            selectedTaskId={routedTaskId ?? page.effectiveSelectedTaskId}
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
    </div>
  );
}
