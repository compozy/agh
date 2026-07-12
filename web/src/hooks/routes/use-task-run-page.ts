import { toast } from "sonner";

import {
  useCancelTaskRun,
  useForceFailTaskRun,
  useForceReleaseTaskRun,
  useRetryTaskRun,
  useTask,
  useTaskRunDetail,
  useTaskRunInspect,
  useTaskRunReviews,
} from "@/systems/tasks";

interface UseTaskRunPageOptions {
  enableTaskDetail?: boolean;
  enableInspect?: boolean;
  enableRunReviews?: boolean;
}

function useTaskRunPage(taskId: string, runId: string, options: UseTaskRunPageOptions = {}) {
  const hasTaskId = Boolean(taskId);
  const hasRunId = Boolean(runId);
  const enableTaskDetail = options.enableTaskDetail ?? true;
  const enableInspect = options.enableInspect ?? true;
  const enableRunReviews = options.enableRunReviews ?? true;

  const runQuery = useTaskRunDetail(runId, { enabled: hasRunId });
  const inspectQuery = useTaskRunInspect(runId, { enabled: hasRunId && enableInspect });
  const taskQuery = useTask(taskId, { enabled: hasTaskId && enableTaskDetail });
  const reviewsQuery = useTaskRunReviews(runId, {}, { enabled: hasRunId && enableRunReviews });
  const cancelMutation = useCancelTaskRun();
  const forceReleaseMutation = useForceReleaseTaskRun();
  const forceFailMutation = useForceFailTaskRun();
  const retryMutation = useRetryTaskRun();

  const run = runQuery.data ?? null;
  const inspect = inspectQuery.data ?? null;
  const task = taskQuery.data ?? null;
  const session = run?.session ?? null;
  const summary = run?.summary ?? null;

  const fatalError =
    hasRunId && hasTaskId ? (runQuery.error ?? null) : new Error("Missing task or run id");

  const runStatus = run?.run?.status;
  const isLive =
    runStatus === "running" ||
    runStatus === "starting" ||
    runStatus === "claimed" ||
    runStatus === "queued";

  const handleCancelRun = async () => {
    if (!hasRunId) {
      return;
    }

    try {
      await cancelMutation.mutateAsync({ runId });
      toast.success("Run canceled.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to cancel run");
    }
  };

  const handleForceReleaseRun = async (reason?: string) => {
    if (!hasRunId) return;
    try {
      await forceReleaseMutation.mutateAsync({
        runId,
        data: { reason: reason?.trim() || undefined },
      });
      toast.success("Run released.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to release run");
    }
  };

  const handleForceFailRun = async (reason: string) => {
    if (!hasRunId) return;
    try {
      await forceFailMutation.mutateAsync({ runId, data: { reason: reason.trim() } });
      toast.success("Run failed.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to fail run");
    }
  };

  const handleRetryRun = async () => {
    if (!hasRunId) {
      return;
    }

    try {
      await retryMutation.mutateAsync({ runId });
      toast.success("Retry queued.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to retry run");
    }
  };

  const reviews = reviewsQuery.data ?? [];

  return {
    fatalError,
    handleCancelRun,
    handleForceFailRun,
    handleForceReleaseRun,
    handleRetryRun,
    isCancelPending: cancelMutation.isPending,
    isForceFailPending: forceFailMutation.isPending,
    isForceReleasePending: forceReleaseMutation.isPending,
    isLive,
    isRetryPending: retryMutation.isPending,
    inspect,
    inspectError: inspectQuery.error ?? null,
    inspectLoading: inspectQuery.isLoading && !inspect,
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
