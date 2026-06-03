import { getRouteApi, useNavigate } from "@tanstack/react-router";
import { useCallback } from "react";

import { useTaskCreateRouteState } from "@/hooks/routes/use-task-create-route-state";
import { TaskEditorModal } from "@/systems/tasks/components/task-editor-modal";

const routeApi = getRouteApi("/_app/tasks/new");

export function TaskCreateRoute() {
  const navigate = useNavigate({ from: "/tasks/new" });
  const page = useTaskCreateRouteState(routeApi.useSearch());

  const handleOpenChange = useCallback(
    (open: boolean) => {
      if (!open) {
        void navigate({ to: "/tasks" });
      }
    },
    [navigate]
  );

  return (
    <TaskEditorModal
      canSubmit={
        page.draft.title.trim().length > 0 &&
        (page.draft.scope === "global" || Boolean(page.draft.workspaceId))
      }
      draft={page.draft}
      isSubmitting={page.isSubmitting}
      mode="new"
      onDraftChange={page.setDraft}
      onOpenChange={handleOpenChange}
      onSubmit={page.handleSubmit}
      onTemplateChange={page.handleTemplateChange}
      open
      task={null}
      template={page.template}
      templateId={page.templateId}
      workspaces={page.workspaces}
    />
  );
}
