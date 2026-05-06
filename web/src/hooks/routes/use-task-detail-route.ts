import { useCallback } from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import { useDeleteTask } from "@/systems/tasks";

import { useTaskDetailOrchestrationTab } from "./use-task-detail-orchestration-tab";
import { useTaskDetailPage } from "./use-task-detail-page";

function useTaskDetailRoute(taskId: string) {
  const navigate = useNavigate({ from: "/tasks/$id" });
  const page = useTaskDetailPage(taskId);
  const deleteMutation = useDeleteTask();
  const latestEventSeq = page.detail?.task?.latest_event_seq ?? null;
  const orchestration = useTaskDetailOrchestrationTab(taskId, {
    enabled: page.panel === "orchestration",
    latestEventSeq,
  });

  const handleDeleteTask = useCallback(
    async (id: string) => {
      void navigate({ to: "/tasks", replace: true });
      try {
        await deleteMutation.mutateAsync({ id });
        toast.success("Task deleted.");
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to delete task");
      }
    },
    [deleteMutation, navigate]
  );

  return {
    deleteMutation,
    handleDeleteTask,
    latestEventSeq,
    orchestration,
    page,
  };
}

export { useTaskDetailRoute };
