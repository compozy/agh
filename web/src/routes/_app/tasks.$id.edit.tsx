import { useCallback } from "react";
import { Pencil } from "lucide-react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { useTaskEditRouteState } from "@/hooks/routes/use-task-edit-route-state";
import { TaskEditorModal } from "@/systems/tasks/components/task-editor-modal";

export const Route = createFileRoute("/_app/tasks/$id/edit")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `Task ${params.id}`, icon: Pencil },
  }),
  component: TaskEditRoute,
});

function TaskEditRoute() {
  const { id } = Route.useParams();
  const navigate = useNavigate({ from: "/tasks/$id/edit" });
  const page = useTaskEditRouteState(id);

  const handleOpenChange = useCallback(
    (open: boolean) => {
      if (!open) {
        void navigate({ to: "/tasks/$id", params: { id } });
      }
    },
    [id, navigate]
  );

  if (page.isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="task-edit-loading">
        <span className="text-sm text-muted">Loading task…</span>
      </div>
    );
  }

  if (!page.task || !page.isInitialized) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="task-edit-empty">
        <span className="text-sm text-muted">We couldn&apos;t load this task for editing.</span>
      </div>
    );
  }

  return (
    <TaskEditorModal
      canSubmit={page.draft.title.trim().length > 0}
      draft={page.draft}
      isSubmitting={page.isSubmitting}
      mode="edit"
      onDraftChange={page.setDraft}
      onOpenChange={handleOpenChange}
      onSubmit={page.handleSubmit}
      open
      task={page.task}
      workspaceName={page.workspaceName}
    />
  );
}
