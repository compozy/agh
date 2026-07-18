import { useState, type SetStateAction } from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import { useTask, useTaskExecutionProfile, useUpdateTask } from "@/systems/tasks";
import {
  buildUpdateTaskRequest,
  EMPTY_TASK_EDITOR_DRAFT,
  taskEditorDraftFromTask,
  type TaskEditorDraft,
} from "@/systems/tasks/lib/task-editor";

export function useTaskEditRouteState(id: string | undefined) {
  const navigate = useNavigate({ from: "/tasks/$id/edit" });
  const detailQuery = useTask(id ?? "", { enabled: Boolean(id) });
  const profileQuery = useTaskExecutionProfile(id ?? "", { enabled: Boolean(id) });
  const updateMutation = useUpdateTask();
  const detail = detailQuery.data ?? null;
  const task = detail?.task ?? null;
  const profile = profileQuery.data ?? null;

  const taskKey =
    task && profile ? `${task.id}:${task.updated_at}:${profile.updated_at}` : "pending";
  const sourceDraft =
    task && profile ? taskEditorDraftFromTask(task, profile) : EMPTY_TASK_EDITOR_DRAFT;
  const [draftState, setDraftState] = useState({ draft: sourceDraft, key: taskKey });
  const draft = draftState.key === taskKey ? draftState.draft : sourceDraft;
  const setDraft = (update: SetStateAction<TaskEditorDraft>) => {
    setDraftState(current => {
      const currentDraft = current.key === taskKey ? current.draft : sourceDraft;
      return {
        draft: typeof update === "function" ? update(currentDraft) : update,
        key: taskKey,
      };
    });
  };

  const handleSubmit = async (nextDraft: TaskEditorDraft) => {
    if (!id || !task || !profile) return null;
    try {
      await updateMutation.mutateAsync({ id, data: buildUpdateTaskRequest(nextDraft) });
      toast.success("Task updated.");
      await navigate({ to: "/tasks/$id", params: { id } });
      return true;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to update task");
      return null;
    }
  };

  return {
    draft,
    handleSubmit,
    isInitialized: task !== null && profile !== null,
    isLoading: (detailQuery.isLoading && !task) || (profileQuery.isLoading && !profile),
    isSubmitting: updateMutation.isPending,
    setDraft,
    task,
    workspaceName: task?.scope === "workspace" ? (task.workspace_id ?? null) : null,
  };
}
