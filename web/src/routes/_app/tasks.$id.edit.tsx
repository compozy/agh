import { Pencil } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { useTaskEditRouteState } from "@/hooks/routes/use-task-edit-route-state";
import { TaskEditorSurface } from "@/systems/tasks/components/task-editor-surface";

export const Route = createFileRoute("/_app/tasks/$id/edit")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `Edit task ${params.id}`, icon: Pencil },
  }),
  component: TaskEditRoute,
});

function TaskEditRoute() {
  const { id } = Route.useParams();
  const page = useTaskEditRouteState(id);

  if (page.isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="task-edit-loading">
        <span className="text-sm text-(--muted)">Loading task…</span>
      </div>
    );
  }

  if (!page.task || !page.isInitialized) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="task-edit-empty">
        <span className="text-sm text-(--muted)">We couldn&apos;t load this task for editing.</span>
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
