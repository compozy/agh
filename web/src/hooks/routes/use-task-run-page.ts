import { useCallback, useMemo } from "react";
import { toast } from "sonner";

import { useCancelTaskRun, useTask, useTaskRunDetail, useTaskRunReviews } from "@/systems/tasks";

interface UseTaskRunPageOptions {
  enableTaskDetail?: boolean;
  enableRunReviews?: boolean;
}

function useTaskRunPage(taskId: string, runId: string, options: UseTaskRunPageOptions = {}) {
  const hasTaskId = Boolean(taskId);
  const hasRunId = Boolean(runId);
  const enableTaskDetail = options.enableTaskDetail ?? true;
  const enableRunReviews = options.enableRunReviews ?? true;

  const runQuery = useTaskRunDetail(runId, { enabled: hasRunId });
  const taskQuery = useTask(taskId, { enabled: hasTaskId && enableTaskDetail });
  const reviewsQuery = useTaskRunReviews(runId, {}, { enabled: hasRunId && enableRunReviews });
  const cancelMutation = useCancelTaskRun();

  const run = runQuery.data ?? null;
  const task = taskQuery.data ?? null;
  const session = run?.session ?? null;
  const summary = run?.summary ?? null;

  const fatalError = useMemo(() => {
    if (!hasRunId || !hasTaskId) {
      return new Error("Missing task or run id");
    }

    return runQuery.error ?? null;
  }, [hasRunId, hasTaskId, runQuery.error]);

  const isLive = useMemo(() => {
    const status = run?.run?.status;
    return (
      status === "running" || status === "starting" || status === "claimed" || status === "queued"
    );
  }, [run]);

  const handleCancelRun = useCallback(async () => {
    if (!hasRunId) {
      return;
    }

    try {
      await cancelMutation.mutateAsync({ runId });
      toast.success("Run canceled.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to cancel run");
    }
  }, [cancelMutation, hasRunId, runId]);

  const reviews = reviewsQuery.data ?? [];

  return {
    fatalError,
    handleCancelRun,
    isCancelPending: cancelMutation.isPending,
    isLive,
    notFound: runQuery.isError && runQuery.error?.message?.includes("not found"),
    reviews,
    reviewsError: reviewsQuery.error ?? null,
    reviewsLoading: reviewsQuery.isLoading && reviews.length === 0,
    run,
    runError: runQuery.error ?? null,
    runId,
    runLoading: runQuery.isLoading && !run,
    session,
    summary,
    task,
    taskError: taskQuery.error ?? null,
    taskId,
    taskLoading: taskQuery.isLoading && !task,
  };
}

export { useTaskRunPage };
export type { UseTaskRunPageOptions };
