import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import { useTask, useUpdateTask } from "@/systems/tasks";
import {
  buildUpdateTaskRequest,
  EMPTY_TASK_EDITOR_DRAFT,
  taskEditorDraftFromTask,
  type TaskEditorDraft,
} from "@/systems/tasks/lib/task-editor";

export function useTaskEditRouteState(id: string | undefined) {
  const navigate = useNavigate({ from: "/tasks/$id/edit" });
  const detailQuery = useTask(id ?? "", { enabled: Boolean(id) });
  const updateMutation = useUpdateTask();
  const detail = detailQuery.data ?? null;
  const task = detail?.task ?? null;

  const [draft, setDraft] = useState<TaskEditorDraft>(EMPTY_TASK_EDITOR_DRAFT);
  const [isInitialized, setInitialized] = useState(false);

  useEffect(() => {
    if (task) {
      setDraft(taskEditorDraftFromTask(task));
      setInitialized(true);
    }
  }, [task]);

  const handleSubmit = useCallback(
    async (nextDraft: TaskEditorDraft) => {
      if (!id) {
        return null;
      }

      try {
        await updateMutation.mutateAsync({
          id,
          data: buildUpdateTaskRequest(nextDraft),
        });
        toast.success("Task updated.");
        await navigate({ to: "/tasks/$id", params: { id } });
        return true;
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to update task");
        return null;
      }
    },
    [id, navigate, updateMutation]
  );

  return {
    draft,
    handleSubmit,
    isInitialized,
    isLoading: detailQuery.isLoading && !task,
    isSubmitting: updateMutation.isPending,
    setDraft,
    task,
    workspaceName: task?.scope === "workspace" ? (task.workspace_id ?? null) : null,
  };
}
