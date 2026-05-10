import { Outlet, createFileRoute, useChildMatches, useNavigate } from "@tanstack/react-router";
import { ListChecks, Plus } from "lucide-react";

import { Button, Empty, PillGroup, SplitPane, useTopbarSlot } from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import {
  TasksDashboardView,
  TasksDetailPreviewPanel,
  TasksEmptyState,
  TasksInboxView,
  TasksKanbanBoard,
  TasksListPanel,
  useTask,
} from "@/systems/tasks";
import { useTasksPage } from "@/hooks/routes/use-tasks-page";

export const Route = createFileRoute("/_app/tasks")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Tasks", icon: ListChecks },
  }),
  component: TasksRoute,
});

function TasksRoute() {
  const navigate = useNavigate({ from: "/tasks" });
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;
  const page = useTasksPage({ forceListData: hasChildMatch });
  const currentChildRouteId = String(childMatches.at(-1)?.id ?? "");
  const routedTaskId = extractRoutedTaskId(childMatches);
  const isCreateRoute = currentChildRouteId.includes("/tasks/new");

  const surfaceMode = hasChildMatch ? "list" : page.mode;
  const showDetailPreview = surfaceMode === "list" && !hasChildMatch;

  const detailQuery = useTask(routedTaskId ?? page.effectiveSelectedTaskId ?? "", {
    enabled: showDetailPreview && Boolean(routedTaskId ?? page.effectiveSelectedTaskId),
  });

  const shellCount =
    surfaceMode === "inbox"
      ? (page.inbox?.total ?? 0)
      : surfaceMode === "dashboard"
        ? (page.dashboard?.totals.tasks_total ?? page.tasksCount)
        : page.tasksCount;

  const handleModeSelect = (nextMode: "list" | "kanban" | "dashboard" | "inbox") => {
    page.handleModeChange(nextMode);
    if (hasChildMatch) {
      void navigate({ to: "/tasks" });
    }
  };

  const openCreateRoute = () => {
    void navigate({ search: () => ({ template: undefined }), to: "/tasks/new" });
  };

  const handleCloseDetail = () => {
    page.dismissSelectedTask();
    if (hasChildMatch) {
      void navigate({ to: "/tasks" });
    }
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

  const listNode = (
    <TasksListPanel
      errorMessage={page.listError?.message ?? null}
      isLoading={page.listLoading}
      isPublishPending={page.isPublishPending}
      onCreateTask={openCreateRoute}
      onPublishTask={page.handlePublishTask}
      onSearchChange={page.setSearchQuery}
      onSelectTask={taskId => {
        page.setSelectedTaskId(taskId);
        void navigate({ params: { id: taskId }, to: "/tasks/$id" });
      }}
      searchQuery={page.searchQuery}
      selectedTaskId={routedTaskId ?? page.effectiveSelectedTaskId}
      statusFilter={page.statusFilter}
      tasks={page.visibleTasks}
      totalCount={page.tasksCount}
    />
  );

  const hasSelectedTask = hasChildMatch || page.selectedTask !== null;
  const detailNode = hasChildMatch ? (
    <Outlet />
  ) : page.selectedTask ? (
    <TasksDetailPreviewPanel
      detail={detailQuery.data ?? null}
      errorMessage={detailQuery.error?.message ?? null}
      isDeletePending={page.isDeletePending}
      isLoading={detailQuery.isLoading && !detailQuery.data}
      onDeleteTask={page.handleDeleteTask}
      isPublishPending={page.isPublishPending}
      onPublishTask={page.handlePublishTask}
      task={page.selectedTask}
    />
  ) : null;

  const detailEmpty = (
    <div
      className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
      data-testid="tasks-detail-empty-slot"
    >
      <Empty
        icon={ListChecks}
        title="Select a task"
        description="Pick an item from the list to see its runs, dependencies, and preview."
      />
    </div>
  );

  return (
    <div
      className="flex min-h-0 flex-1 flex-col overflow-hidden"
      data-testid="tasks-shell"
    >
      <div
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
        data-testid="tasks-shell-body"
      >
        {surfaceMode === "dashboard" ? (
          <TasksDashboardView
            dashboard={page.dashboard}
            errorMessage={page.dashboardError?.message ?? null}
            isLoading={page.dashboardLoading}
          />
        ) : surfaceMode === "inbox" ? (
          <TasksInboxView
            errorMessage={page.inboxError?.message ?? null}
            inbox={page.inbox}
            isLoading={page.inboxLoading}
            laneFilter={page.inboxLaneFilter}
            onApprove={page.handleApproveTask}
            onArchive={page.handleArchiveTask}
            onDismiss={page.handleDismissTask}
            onLaneChange={page.handleInboxLaneChange}
            onMarkRead={page.handleMarkTaskRead}
            onReject={page.handleRejectTask}
            onRetry={page.handleRetryTask}
            onSearchChange={page.setInboxSearchQuery}
            onToggleUnread={page.handleInboxUnreadToggle}
            searchQuery={page.inboxSearchQuery}
            unreadOnly={page.inboxUnreadOnly}
          />
        ) : page.isEmpty && !hasChildMatch ? (
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
            onSelectTask={taskId => {
              page.setSelectedTaskId(taskId);
              void navigate({ params: { id: taskId }, to: "/tasks/$id" });
            }}
            selectedTaskId={routedTaskId ?? page.effectiveSelectedTaskId}
          />
        ) : (
          <SplitPane
            data-testid="tasks-split-pane"
            detail={hasSelectedTask ? detailNode : undefined}
            detailEmpty={detailEmpty}
            list={listNode}
            listWidth={340}
            onDetailClose={handleCloseDetail}
          />
        )}
      </div>
    </div>
  );
}

function extractRoutedTaskId(matches: Array<unknown>): string | null {
  for (let index = matches.length - 1; index >= 0; index -= 1) {
    const match = matches[index];
    if (!match || typeof match !== "object" || !("params" in match)) {
      continue;
    }

    const params = (match as { params?: Record<string, unknown> }).params;
    if (!params || typeof params.id !== "string") {
      continue;
    }

    return params.id;
  }

  return null;
}
