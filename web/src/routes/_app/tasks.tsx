import { Outlet, createFileRoute, useNavigate } from "@tanstack/react-router";
import { ListChecks, Plus } from "lucide-react";

import { Button, PillGroup, SearchInput, useTopbarSlot } from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import {
  TasksDashboardView,
  TasksEmptyState,
  TasksInboxView,
  TasksKanbanBoard,
  TasksListSurface,
} from "@/systems/tasks";
import { useTasksRoute } from "@/hooks/routes/use-tasks-route";

export const Route = createFileRoute("/_app/tasks")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Tasks", icon: ListChecks },
  }),
  component: TasksRoute,
});

function TasksRoute() {
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
            badge: page.inbox?.unread_total ?? 0,
            testId: "tasks-mode-inbox",
          },
        ]}
      />
    ),
    search:
      surfaceMode === "list" && !hasChildMatch ? (
        <SearchInput
          data-testid="tasks-list-search-input"
          onChange={page.setSearchQuery}
          placeholder="Search tasks..."
          value={page.searchQuery}
        />
      ) : undefined,
    actions: (
      <Button
        data-testid="tasks-open-create"
        disabled={isCreateRoute}
        onClick={openCreateRoute}
        size="sm"
        type="button"
        variant="outline"
      >
        <Plus className="size-3.5" />
        Task
      </Button>
    ),
  });

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="tasks-shell">
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="tasks-shell-body">
        {hasChildMatch ? (
          <Outlet />
        ) : surfaceMode === "dashboard" ? (
          <TasksDashboardView
            dashboard={page.dashboard}
            errorMessage={page.dashboardError?.message ?? null}
            isLoading={page.dashboardLoading}
          />
        ) : surfaceMode === "inbox" ? (
          <TasksInboxView
            errorMessage={page.inboxError?.message ?? null}
            inbox={page.inbox}
            inboxUpdatedAt={page.inboxUpdatedAt}
            isLoading={page.inboxLoading}
            laneFilter={page.inboxLaneFilter}
            onApprove={page.handleApproveTask}
            onArchive={page.handleArchiveTask}
            onDismiss={page.handleDismissTask}
            onLaneChange={page.handleInboxLaneChange}
            onMarkRead={page.handleMarkTaskRead}
            onPriorityChange={page.handleInboxPriorityChange}
            onReject={page.handleRejectTask}
            onRetry={page.handleRetryTask}
            onSearchChange={page.setInboxSearchQuery}
            onStatusChange={page.handleInboxStatusChange}
            onToggleUnread={page.handleInboxUnreadToggle}
            priorityFilter={page.inboxPriorityFilter}
            searchQuery={page.inboxSearchQuery}
            statusFilter={page.inboxStatusFilter}
            unreadOnly={page.inboxUnreadOnly}
            workspaceName={page.activeWorkspaceName}
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
            isLoading={page.listLoading}
            onCreateInColumn={openCreateRoute}
            onSelectTask={handleSelectTask}
            selectedTaskId={routedTaskId ?? page.effectiveSelectedTaskId}
          />
        ) : (
          <TasksListSurface
            errorMessage={page.listError?.message ?? null}
            isLoading={page.listLoading}
            listUpdatedAt={page.listUpdatedAt}
            onOwnerChange={page.handleOwnerChange}
            onPriorityChange={page.handlePriorityChange}
            onScopeChange={page.handleScopeChange}
            onSelectTask={handleSelectTask}
            onSortChange={page.handleSortChange}
            onStatusChange={page.handleStatusChange}
            ownerFilter={page.ownerFilter}
            ownerOptions={page.ownerOptions}
            priorityFilter={page.priorityFilter}
            scopeFilter={page.scopeFilter}
            sortBy={page.sortBy}
            statusFilter={page.statusFilter}
            tasks={page.visibleTasks}
            totalCount={page.tasksCount}
            workspaceName={page.activeWorkspaceName}
          />
        )}
      </div>
    </div>
  );
}
