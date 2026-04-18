import { Outlet, createFileRoute, useChildMatches } from "@tanstack/react-router";
import { Plus } from "lucide-react";

import { PillButton } from "@/components/design-system";
import { Button } from "@agh/ui";
import { useTask } from "@/systems/tasks";
import {
  TasksCreateModal,
  TasksDetailPreviewPanel,
  TasksEmptyState,
  TasksKanbanBoard,
  TasksListPanel,
  TasksPageShell,
} from "@/systems/tasks";
import { useTasksPage } from "@/hooks/routes/use-tasks-page";

export const Route = createFileRoute("/_app/tasks")({
  component: TasksRoute,
});

function TasksRoute() {
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;
  const page = useTasksPage();

  const detailQuery = useTask(page.effectiveSelectedTaskId ?? "", {
    enabled: !hasChildMatch && page.mode === "list" && Boolean(page.effectiveSelectedTaskId),
  });

  if (hasChildMatch) {
    return (
      <TasksPageShell count={page.tasksCount}>
        <Outlet />
      </TasksPageShell>
    );
  }

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
          </div>
        }
        count={page.tasksCount}
        meta={
          <div className="flex items-center gap-1.5">
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
          </div>
        }
      >
        {page.isEmpty ? (
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
