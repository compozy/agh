import { useCallback } from "react";
import { Plus } from "lucide-react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { useTaskCreateRouteState } from "@/hooks/routes/use-task-create-route-state";
import type { TaskTemplateId } from "@/systems/tasks/lib/task-templates";
import { TaskEditorModal } from "@/systems/tasks/components/task-editor-modal";

export const Route = createFileRoute("/_app/tasks/new")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Tasks", icon: Plus },
  }),
  validateSearch: search => ({
    template:
      typeof search.template === "string" &&
      ["one_shot", "recurring", "epic", "remote_peer", "human_in_loop", "blank"].includes(
        search.template
      )
        ? (search.template as TaskTemplateId)
        : undefined,
  }),
  component: TaskCreateRoute,
});

function TaskCreateRoute() {
  const navigate = useNavigate({ from: "/tasks/new" });
  const page = useTaskCreateRouteState(Route.useSearch());

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
      canSubmit={page.draft.title.trim().length > 0}
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
      workspaceName={page.workspaceName}
    />
  );
}
