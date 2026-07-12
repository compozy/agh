import { useReducer, useRef } from "react";
import { toast } from "sonner";

import {
  useApproveTask,
  useArchiveTask,
  useDismissTask,
  useMarkTaskRead,
  useRejectTask,
  useRetryTaskRun,
} from "@/systems/tasks";

type PendingActionKind = "approve" | "archive" | "dismiss" | "markRead" | "reject" | "retry";
type PendingActionIds = Record<PendingActionKind, Set<string>>;

function createPendingActionIds(): PendingActionIds {
  return {
    approve: new Set(),
    archive: new Set(),
    dismiss: new Set(),
    markRead: new Set(),
    reject: new Set(),
    retry: new Set(),
  };
}

export function useTasksPageActions() {
  const retryMutation = useRetryTaskRun();
  const approveMutation = useApproveTask();
  const rejectMutation = useRejectTask();
  const markReadMutation = useMarkTaskRead();
  const archiveMutation = useArchiveTask();
  const dismissMutation = useDismissTask();
  const pendingIdsRef = useRef<PendingActionIds | null>(null);
  if (pendingIdsRef.current === null) {
    pendingIdsRef.current = createPendingActionIds();
  }
  const pendingIds = pendingIdsRef.current;
  const [, renderPendingState] = useReducer((version: number) => version + 1, 0);

  const runOnce = async (
    kind: PendingActionKind,
    id: string,
    operation: () => Promise<unknown>
  ) => {
    const pending = pendingIds[kind];
    if (pending.has(id)) return false;
    pending.add(id);
    renderPendingState();
    try {
      await operation();
      return true;
    } finally {
      pending.delete(id);
      renderPendingState();
    }
  };

  const handleApproveTask = async (taskId: string) => {
    try {
      const executed = await runOnce("approve", taskId, () =>
        approveMutation.mutateAsync({ id: taskId })
      );
      if (executed) {
        toast.success("Task approved.");
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to approve task");
    }
  };
  const handleRejectTask = async (taskId: string) => {
    try {
      const executed = await runOnce("reject", taskId, () =>
        rejectMutation.mutateAsync({ id: taskId })
      );
      if (executed) {
        toast.success("Task rejected.");
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to reject task");
    }
  };
  const handleMarkTaskRead = async (taskId: string) => {
    try {
      await runOnce("markRead", taskId, () => markReadMutation.mutateAsync({ id: taskId }));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to mark task read");
    }
  };
  const handleArchiveTask = async (taskId: string) => {
    try {
      const executed = await runOnce("archive", taskId, () =>
        archiveMutation.mutateAsync({ id: taskId })
      );
      if (executed) {
        toast.success("Task archived.");
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to archive task");
    }
  };
  const handleDismissTask = async (taskId: string) => {
    try {
      await runOnce("dismiss", taskId, () => dismissMutation.mutateAsync({ id: taskId }));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to dismiss task");
    }
  };
  const handleRetryRun = async (runId: string) => {
    try {
      const executed = await runOnce("retry", runId, () => retryMutation.mutateAsync({ runId }));
      if (executed) {
        toast.success("Run retry requested.");
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to retry run");
    }
  };

  return {
    handleApproveTask,
    handleArchiveTask,
    handleDismissTask,
    handleMarkTaskRead,
    handleRejectTask,
    handleRetryRun,
    pendingApproveIds: pendingIds.approve as ReadonlySet<string>,
    pendingArchiveIds: pendingIds.archive as ReadonlySet<string>,
    pendingDismissIds: pendingIds.dismiss as ReadonlySet<string>,
    pendingMarkReadIds: pendingIds.markRead as ReadonlySet<string>,
    pendingRejectIds: pendingIds.reject as ReadonlySet<string>,
    pendingRetryIds: pendingIds.retry as ReadonlySet<string>,
  };
}
