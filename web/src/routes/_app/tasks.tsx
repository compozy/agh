import { Outlet, createFileRoute, useChildMatches } from "@tanstack/react-router";
import { Plus } from "lucide-react";

import { PillButton } from "@/components/design-system";
import { Button } from "@agh/ui";
import {
  TasksCreateModal,
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
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;
  const page = useTasksPage();

  const showDetailPreview = page.mode === "list" && !hasChildMatch;

  const detailQuery = useTask(page.effectiveSelectedTaskId ?? "", {
    enabled: showDetailPreview && Boolean(page.effectiveSelectedTaskId),
  });

  if (hasChildMatch) {
    return (
      <TasksPageShell count={page.tasksCount}>
        <Outlet />
      </TasksPageShell>
    );
  }

  const showCreateButton = page.mode !== "inbox";
  const shellCount =
    page.mode === "inbox"
      ? (page.inbox?.total ?? 0)
      : page.mode === "dashboard"
        ? (page.dashboard?.totals.tasks_total ?? page.tasksCount)
        : page.tasksCount;

  return (
    <>
      <TasksPageShell
        controls={
          <div className="flex items-center gap-1.5" data-testid="tasks-mode-pills">
            <PillButton
              active={page.mode === "list"}
              data-testid="tasks-mode-list"
              onClick={() => page.handleModeChange("list")}
            >
              List
            </PillButton>
            <PillButton
              active={page.mode === "kanban"}
              data-testid="tasks-mode-kanban"
              onClick={() => page.handleModeChange("kanban")}
            >
              Kanban
            </PillButton>
            <PillButton
              active={page.mode === "dashboard"}
              data-testid="tasks-mode-dashboard"
              onClick={() => page.handleModeChange("dashboard")}
            >
              Dashboard
            </PillButton>
            <PillButton
              active={page.mode === "inbox"}
              data-testid="tasks-mode-inbox"
              onClick={() => page.handleModeChange("inbox")}
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
            {showCreateButton ? (
              <Button
                className="border-[color:var(--color-accent)] bg-transparent text-[color:var(--color-accent)] hover:bg-[color:var(--color-accent-tint)] hover:text-[color:var(--color-accent)]"
                data-testid="tasks-open-create"
                onClick={() => page.handleOpenCreateModal()}
                size="lg"
                type="button"
                variant="outline"
              >
                <Plus className="size-4" />
                Task
              </Button>
            ) : null}
          </div>
        }
      >
        {page.mode === "dashboard" ? (
          <TasksDashboardView
            dashboard={page.dashboard}
            errorMessage={page.dashboardError?.message ?? null}
            isLoading={page.dashboardLoading}
          />
        ) : page.mode === "inbox" ? (
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
        ) : page.isEmpty ? (
          <TasksEmptyState
            onSelectTemplate={page.handleOpenCreateModal}
            workspaceName={page.activeWorkspaceName}
          />
        ) : page.mode === "kanban" ? (
          <TasksKanbanBoard
            columns={page.kanbanColumns}
            errorMessage={page.listError?.message ?? null}
            isLoading={page.listLoading}
            onCreateInColumn={() => page.handleOpenCreateModal()}
            onSelectTask={page.setSelectedTaskId}
            selectedTaskId={page.effectiveSelectedTaskId}
          />
        ) : (
          <div className="flex min-h-0 flex-1 overflow-hidden">
            <TasksListPanel
              errorMessage={page.listError?.message ?? null}
              isLoading={page.listLoading}
              isPublishPending={page.isPublishPending}
              onPublishTask={page.handlePublishTask}
              onSearchChange={page.setSearchQuery}
              onSelectTask={page.setSelectedTaskId}
              searchQuery={page.searchQuery}
              selectedTaskId={page.effectiveSelectedTaskId}
              statusFilter={page.statusFilter}
              tasks={page.visibleTasks}
              totalCount={page.tasksCount}
            />
            <TasksDetailPreviewPanel
              detail={detailQuery.data ?? null}
              errorMessage={detailQuery.error?.message ?? null}
              isLoading={detailQuery.isLoading && !detailQuery.data}
              isPublishPending={page.isPublishPending}
              onPublishTask={page.handlePublishTask}
              task={page.selectedTask}
            />
          </div>
        )}
      </TasksPageShell>

      <TasksCreateModal
        canSubmit={page.canSubmitCreate}
        draft={page.createDraft}
        isSubmitting={page.isCreatePending}
        onDraftChange={page.setCreateDraft}
        onOpenChange={open =>
          open ? page.handleOpenCreateModal(page.createTemplateId) : page.handleCloseCreateModal()
        }
        onSubmit={page.submitCreateTask}
        onTemplateChange={page.handleTemplateChange}
        open={page.isCreateModalOpen}
        template={page.createTemplate}
        templateId={page.createTemplateId}
        workspaceName={page.activeWorkspaceName}
      />
    </>
  );
}
