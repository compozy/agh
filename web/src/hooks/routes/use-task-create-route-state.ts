import { useNavigate } from "@tanstack/react-router";
import { useRef, useState, type SetStateAction } from "react";
import { toast } from "sonner";

import { useCreateChildTask, useCreateTask, useEnqueueTaskRun } from "@/systems/tasks";
import {
  buildCreateChildTaskRequest,
  buildCreateTaskRequest,
  createTaskEditorDraft,
  type TaskEditorDraft,
} from "@/systems/tasks/lib/task-editor";
import {
  DEFAULT_TASK_TEMPLATE_ID,
  getTaskTemplate,
  type TaskTemplateId,
} from "@/systems/tasks/lib/task-templates";
import {
  toWorkspaceCommandSelectOptions,
  useActiveWorkspace,
  useUserHomeDir,
} from "@/systems/workspace";
import { taskScopeForActiveWorkspace } from "./workspace-scope-filter";

export function useTaskCreateRouteState(search: { template?: TaskTemplateId }) {
  const navigate = useNavigate({ from: "/tasks/new" });
  const { activeWorkspace, workspaces } = useActiveWorkspace();
  const userHomeDir = useUserHomeDir();
  const createMutation = useCreateTask();
  const createChildMutation = useCreateChildTask();
  const enqueueMutation = useEnqueueTaskRun();
  const submitInFlightRef = useRef(false);

  const templateId = search.template ?? DEFAULT_TASK_TEMPLATE_ID;
  const activeTaskScope = taskScopeForActiveWorkspace(activeWorkspace, userHomeDir);
  const createDraftWorkspaceId =
    activeTaskScope?.scope === "workspace" ? activeTaskScope.workspace : undefined;
  const draftKey = `${templateId}:${createDraftWorkspaceId ?? "global"}`;
  const [draftState, setDraftState] = useState(() => ({
    draft: createTaskEditorDraft(templateId, createDraftWorkspaceId),
    key: draftKey,
  }));
  const draft =
    draftState.key === draftKey
      ? draftState.draft
      : createTaskEditorDraft(templateId, createDraftWorkspaceId);
  const setDraft = (update: SetStateAction<TaskEditorDraft>) => {
    setDraftState(current => {
      const currentDraft =
        current.key === draftKey
          ? current.draft
          : createTaskEditorDraft(templateId, createDraftWorkspaceId);
      return {
        draft: typeof update === "function" ? update(currentDraft) : update,
        key: draftKey,
      };
    });
  };

  const handleTemplateChange = (nextTemplateId: TaskTemplateId) => {
    void navigate({
      to: "/tasks/new",
      search: () =>
        nextTemplateId === DEFAULT_TASK_TEMPLATE_ID
          ? { template: undefined }
          : { template: nextTemplateId },
    });
  };

  const handleSubmit = async (nextDraft: TaskEditorDraft, asDraft: boolean) => {
    if (submitInFlightRef.current) {
      return null;
    }

    const trimmedTitle = nextDraft.title.trim();
    if (!trimmedTitle) {
      toast.error("Provide a title before creating the task.");
      return null;
    }

    if (nextDraft.scope === "workspace" && !nextDraft.workspaceId) {
      toast.error("Select a workspace before creating a workspace task.");
      return null;
    }

    const parentTaskId = nextDraft.parentTaskId.trim();
    const isChildTask = parentTaskId.length > 0;

    submitInFlightRef.current = true;
    try {
      const created = isChildTask
        ? await createChildMutation.mutateAsync({
            parentId: parentTaskId,
            data: buildCreateChildTaskRequest(nextDraft, {
              asDraft,
              templateId,
            }),
          })
        : await createMutation.mutateAsync(
            buildCreateTaskRequest(nextDraft, {
              asDraft,
              templateId,
            })
          );
      const wantsImmediateRun =
        !created.draft && getTaskTemplate(templateId).preview.enqueueOnSubmit;
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
        created.draft ? `Saved draft "${trimmedTitle}".` : `Created task "${trimmedTitle}".`
      );

      if (created.id) {
        await navigate({ to: "/tasks/$id", params: { id: created.id } });
      }

      submitInFlightRef.current = false;
      return created;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to create task");
      submitInFlightRef.current = false;
      return null;
    }
  };

  return {
    draft,
    handleSubmit,
    handleTemplateChange,
    isSubmitting:
      createMutation.isPending || createChildMutation.isPending || enqueueMutation.isPending,
    setDraft,
    template: getTaskTemplate(templateId),
    templateId,
    workspaces: toWorkspaceCommandSelectOptions(workspaces),
  };
}
