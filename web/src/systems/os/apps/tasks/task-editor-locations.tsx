import { useTopbarSlot } from "@agh/ui";

import {
  TaskEditorSurface,
  useTaskCreateState,
  useTaskEditState,
  type TaskCreateSearch,
} from "@/systems/tasks";

import { useTasksNavigation } from "./hooks/use-tasks-navigation";

export function TaskCreateLocation({ search }: { search: TaskCreateSearch }) {
  const navigate = useTasksNavigation();
  const page = useTaskCreateState(search, navigate);
  useTopbarSlot({
    onBack: () => navigate({ pathname: "/tasks", search: {} }),
    crumbs: [
      { id: "tasks", label: "Tasks", onSelect: () => navigate({ pathname: "/tasks", search: {} }) },
    ],
    crumb: "New",
  });

  return (
    <TaskEditorSurface
      canSubmit={
        page.draft.title.trim().length > 0 &&
        (page.draft.scope === "global" || Boolean(page.draft.workspaceId))
      }
      draft={page.draft}
      isSubmitting={page.isSubmitting}
      mode="new"
      onCancel={() => navigate({ pathname: "/tasks", search: {} })}
      onDraftChange={page.setDraft}
      onSubmit={page.handleSubmit}
      onTemplateChange={page.handleTemplateChange}
      template={page.template}
      templateId={page.templateId}
      workspaces={page.workspaces}
    />
  );
}

export function TaskEditLocation({ taskId }: { taskId: string }) {
  const navigate = useTasksNavigation();
  const page = useTaskEditState(taskId, navigate);
  const backToTask = () =>
    navigate({ pathname: `/tasks/${encodeURIComponent(taskId)}`, search: {} });
  useTopbarSlot({
    onBack: backToTask,
    crumbs: [
      { id: "tasks", label: "Tasks", onSelect: () => navigate({ pathname: "/tasks", search: {} }) },
      { id: "task", label: taskId, onSelect: backToTask },
    ],
    crumb: "Edit",
  });

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
    <TaskEditorSurface
      canSubmit={page.draft.title.trim().length > 0}
      draft={page.draft}
      isSubmitting={page.isSubmitting}
      mode="edit"
      onCancel={backToTask}
      onDraftChange={page.setDraft}
      onSubmit={page.handleSubmit}
    />
  );
}
