import { createFileRoute } from "@tanstack/react-router";

import { useTaskEditRouteState } from "@/hooks/routes/use-task-edit-route-state";
import { TaskEditorSurface } from "@/systems/tasks/components/task-editor-surface";

export const Route = createFileRoute("/_app/tasks/$id/edit")({
  component: TaskEditRoute,
});

function TaskEditRoute() {
  const { id } = Route.useParams();
  const page = useTaskEditRouteState(id);

  if (page.isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="task-edit-loading">
        <span className="text-sm text-(--color-text-secondary)">Loading task…</span>
      </div>
    );
  }

  if (!page.task || !page.isInitialized) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="task-edit-empty">
        <span className="text-sm text-(--color-text-secondary)">
          We couldn&apos;t load this task for editing.
        </span>
      </div>
    );
  }

  return (
    <TaskEditorSurface
      canSubmit={page.draft.title.trim().length > 0}
      draft={page.draft}
      isSubmitting={page.isSubmitting}
      mode="edit"
      onDraftChange={page.setDraft}
      onSubmit={page.handleSubmit}
      task={page.task}
      workspaceName={page.workspaceName}
    />
  );
}
