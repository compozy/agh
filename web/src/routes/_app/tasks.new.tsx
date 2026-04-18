import { createFileRoute } from "@tanstack/react-router";

import { useTaskCreateRouteState } from "@/hooks/routes/use-task-create-route-state";
import type { TaskTemplateId } from "@/systems/tasks/lib/task-templates";
import { TaskEditorSurface } from "@/systems/tasks/components/task-editor-surface";

export const Route = createFileRoute("/_app/tasks/new")({
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
  const page = useTaskCreateRouteState(Route.useSearch());

  return (
    <TaskEditorSurface
      canSubmit={page.draft.title.trim().length > 0}
      draft={page.draft}
      isSubmitting={page.isSubmitting}
      mode="create"
      onDraftChange={page.setDraft}
      onSubmit={page.handleSubmit}
      onTemplateChange={page.handleTemplateChange}
      task={null}
      template={page.template}
      templateId={page.templateId}
      workspaceName={page.workspaceName}
    />
  );
}
