import { Outlet, createFileRoute, useChildMatches, useNavigate } from "@tanstack/react-router";
import { Plus } from "lucide-react";

import { PillButton } from "@/components/design-system";
import { Button } from "@agh/ui";
import {
  TasksDashboardView,
  TasksDetailPreviewPanel,
  TasksEmptyState,
  TasksInboxView,
  TasksKanbanBoard,
  TasksListPanel,
  TasksPageShell,
  useTask,
} from "@/systems/tasks";
import { useTasksPage } from "@/hooks/routes/use-tasks-page";

export const Route = createFileRoute("/_app/tasks")({
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

  return (
    <TasksPageShell
      controls={
        <div className="flex items-center gap-1.5" data-testid="tasks-mode-pills">
          <PillButton
            active={surfaceMode === "list"}
            data-testid="tasks-mode-list"
            onClick={() => handleModeSelect("list")}
          >
            List
          </PillButton>
          <PillButton
            active={surfaceMode === "kanban"}
            data-testid="tasks-mode-kanban"
            onClick={() => handleModeSelect("kanban")}
          >
            Kanban
          </PillButton>
          <PillButton
            active={surfaceMode === "dashboard"}
            data-testid="tasks-mode-dashboard"
            onClick={() => handleModeSelect("dashboard")}
          >
            Dashboard
          </PillButton>
          <PillButton
            active={surfaceMode === "inbox"}
            data-testid="tasks-mode-inbox"
            onClick={() => handleModeSelect("inbox")}
          >
            Inbox
            {page.inbox && page.inbox.unread_total > 0 ? (
              <span
                className="ml-1.5 inline-flex size-4 items-center justify-center rounded-full bg-[color:var(--color-warning)] text-[0.58rem] font-semibold text-[color:var(--color-accent-ink)]"
                data-testid="tasks-mode-inbox-unread"
              >
                {page.inbox.unread_total}
              </span>
            ) : null}
          </PillButton>
        </div>
      }
      count={shellCount}
      meta={
        <div className="flex items-center gap-1.5">
          <Button
            className="border-[color:var(--color-divider)] bg-transparent text-[color:var(--color-text-primary)] hover:bg-[color:var(--color-hover)]"
            data-testid="tasks-open-create"
            disabled={isCreateRoute}
            onClick={openCreateRoute}
            size="lg"
            type="button"
            variant="outline"
          >
            <Plus className="size-4" />
            Task
          </Button>
        </div>
      }
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
        <div className="flex min-h-0 flex-1 overflow-hidden">
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
          {hasChildMatch ? (
            <Outlet />
          ) : (
            <TasksDetailPreviewPanel
              detail={detailQuery.data ?? null}
              errorMessage={detailQuery.error?.message ?? null}
              isLoading={detailQuery.isLoading && !detailQuery.data}
              isPublishPending={page.isPublishPending}
              onPublishTask={page.handlePublishTask}
              task={page.selectedTask}
            />
          )}
        </div>
      )}
    </TasksPageShell>
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
