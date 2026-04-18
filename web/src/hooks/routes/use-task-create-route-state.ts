import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import { useCreateTask, useEnqueueTaskRun } from "@/systems/tasks";
import {
  buildCreateTaskRequest,
  createTaskEditorDraft,
  type TaskEditorDraft,
} from "@/systems/tasks/lib/task-editor";
import {
  DEFAULT_TASK_TEMPLATE_ID,
  getTaskTemplate,
  type TaskTemplateId,
} from "@/systems/tasks/lib/task-templates";
import { useActiveWorkspace } from "@/systems/workspace";

export function useTaskCreateRouteState(search: { template?: TaskTemplateId }) {
  const navigate = useNavigate({ from: "/tasks/new" });
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();
  const createMutation = useCreateTask();
  const enqueueMutation = useEnqueueTaskRun();

  const templateId = search.template ?? DEFAULT_TASK_TEMPLATE_ID;
  const [draft, setDraft] = useState<TaskEditorDraft>(() =>
    createTaskEditorDraft(templateId, activeWorkspaceId)
  );

  useEffect(() => {
    setDraft(createTaskEditorDraft(templateId, activeWorkspaceId));
  }, [activeWorkspaceId, templateId]);

  const handleTemplateChange = useCallback(
    (nextTemplateId: TaskTemplateId) => {
      void navigate({
        to: "/tasks/new",
        search: () =>
          nextTemplateId === DEFAULT_TASK_TEMPLATE_ID
            ? { template: undefined }
            : { template: nextTemplateId },
      });
    },
    [navigate]
  );

  const handleSubmit = useCallback(
    async (nextDraft: TaskEditorDraft, asDraft: boolean) => {
      const trimmedTitle = nextDraft.title.trim();
      if (!trimmedTitle) {
        toast.error("Provide a title before creating the task.");
        return null;
      }

      if (nextDraft.scope === "workspace" && !activeWorkspaceId) {
        toast.error("Select an active workspace before creating a workspace task.");
        return null;
      }

      const payload = buildCreateTaskRequest(nextDraft, {
        activeWorkspaceId,
        asDraft,
        templateId,
      });

      try {
        const created = await createMutation.mutateAsync(payload);
        const wantsImmediateRun =
          !payload.draft && getTaskTemplate(templateId).preview.enqueueOnSubmit;
        if (wantsImmediateRun && created.id) {
          try {
            await enqueueMutation.mutateAsync({ id: created.id });
          } catch (runError) {
            const message =
              runError instanceof Error ? runError.message : "Failed to enqueue first run";
            toast.error(`Task created, but enqueue failed: ${message}`);
          }
        }

        toast.success(
          payload.draft ? `Saved draft "${trimmedTitle}".` : `Created task "${trimmedTitle}".`
        );

        if (created.id) {
          await navigate({ to: "/tasks/$id", params: { id: created.id } });
        }

        return created;
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to create task");
        return null;
      }
    },
    [activeWorkspaceId, createMutation, enqueueMutation, navigate, templateId]
  );

  return {
    draft,
    handleSubmit,
    handleTemplateChange,
    isSubmitting: createMutation.isPending || enqueueMutation.isPending,
    setDraft,
    template: getTaskTemplate(templateId),
    templateId,
    workspaceName: activeWorkspace?.name ?? null,
  };
}
