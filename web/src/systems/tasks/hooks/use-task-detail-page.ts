import { useState } from "react";
import { toast } from "sonner";

import {
  useApproveTask,
  useCancelTask,
  useClearTaskBlock,
  useEnqueueTaskRun,
  useFanOutTaskRuns,
  usePauseTask,
  usePublishTask,
  useRecoverTask,
  useRejectTask,
  useResumeTask,
  useRetryTaskRun,
} from "./use-task-actions";
import { useTask, useTaskRuns } from "./use-tasks";
import { useTaskInspect, useTaskTimeline } from "./use-task-live";
import { useTaskExecutionProfile } from "./use-task-profile";
import { useTaskReviews } from "./use-task-reviews";
import { useRecoverTaskRun } from "./use-task-run-recovery";
import { useTaskStream } from "./use-task-stream";
import { taskRunCanRecover } from "../lib/task-run-recovery";
import type { FanOutTaskRunsRequest, TaskRunsFilter, TaskTimelineFilter } from "../types";

interface UseTaskDetailPageOptions {
  initialTimelineLimit?: number;
  runFilters?: TaskRunsFilter;
  timelineFilters?: TaskTimelineFilter;
  enableTimeline?: boolean;
  enableRuns?: boolean;
  enableInspect?: boolean;
  enableStream?: boolean;
}

const DEFAULT_TIMELINE_LIMIT = 50;
const TIMELINE_PAGE_SIZE = 50;

async function runAction(
  action: () => Promise<unknown>,
  success: string,
  failure: string
): Promise<void> {
  try {
    await action();
    toast.success(success);
  } catch (error) {
    toast.error(error instanceof Error ? error.message : failure);
  }
}

/**
 * View model for the 3-tab task detail page. Tab state lives in the window
 * location (URL-addressable); this hook owns queries, the SSE stream, and
 * every task-level verb handler.
 */
function useTaskDetailPage(taskId: string, options: UseTaskDetailPageOptions = {}) {
  const [timelineLimit, setTimelineLimit] = useState<number>(
    options.initialTimelineLimit ?? DEFAULT_TIMELINE_LIMIT
  );

  const hasTaskId = Boolean(taskId);
  const enableTimeline = options.enableTimeline ?? true;
  const enableRuns = options.enableRuns ?? true;
  const enableInspect = options.enableInspect ?? true;
  const enableStream = options.enableStream ?? true;

  const timelineFilters: TaskTimelineFilter = {
    limit: timelineLimit,
    after_sequence: options.timelineFilters?.after_sequence,
  };

  const detailQuery = useTask(taskId, { enabled: hasTaskId });
  const timelineQuery = useTaskTimeline(taskId, timelineFilters, {
    enabled: hasTaskId && enableTimeline,
  });
  const runsQuery = useTaskRuns(taskId, options.runFilters ?? {}, {
    enabled: hasTaskId && enableRuns,
  });
  const inspectQuery = useTaskInspect(taskId, { enabled: hasTaskId && enableInspect });
  const profileQuery = useTaskExecutionProfile(taskId, { enabled: hasTaskId });
  const reviewsQuery = useTaskReviews(taskId, {}, { enabled: hasTaskId });

  const publishMutation = usePublishTask();
  const cancelMutation = useCancelTask();
  const enqueueMutation = useEnqueueTaskRun();
  const pauseMutation = usePauseTask();
  const resumeMutation = useResumeTask();
  const recoverTaskMutation = useRecoverTask();
  const recoverRunMutation = useRecoverTaskRun();
  const approveMutation = useApproveTask();
  const rejectMutation = useRejectTask();
  const retryRunMutation = useRetryTaskRun();
  const clearBlockMutation = useClearTaskBlock();
  const fanOutMutation = useFanOutTaskRuns();

  const detail = detailQuery.data ?? null;
  const runs = runsQuery.data ?? [];
  const timeline = timelineQuery.data ?? [];
  const inspect = inspectQuery.data ?? null;
  const profile = profileQuery.data ?? null;
  const reviews = reviewsQuery.data ?? [];

  const activeRun = detail?.summary?.active_run ?? null;
  const activeRunNeedsAttention = activeRun?.status === "needs_attention";
  const recoverableRunId =
    activeRun && taskRunCanRecover(activeRun, detail?.task.max_attempts) ? activeRun.id : null;
  const isLive = isRunActive(activeRun?.status ?? null);

  // Keep the page fresh from run-lifecycle SSE events. Wait for the detail
  // payload before connecting so we seed from the real cursor instead of
  // after_sequence=0 (a full-history replay + immediate reconnect when
  // latest_event_seq resolves).
  const detailEventSeq = detail?.task?.latest_event_seq;
  const hasEventSeq = typeof detailEventSeq === "number";
  useTaskStream(taskId, {
    enabled: hasTaskId && enableStream && hasEventSeq,
    afterSequence: hasEventSeq ? Math.max(0, detailEventSeq) : undefined,
  });

  const fatalError = hasTaskId ? (detailQuery.error ?? null) : new Error("Missing task id");

  const handleTimelineLoadMore = () => {
    setTimelineLimit(current => current + TIMELINE_PAGE_SIZE);
  };

  const handlePublishTask = () =>
    hasTaskId
      ? runAction(
          () => publishMutation.mutateAsync({ id: taskId }),
          "Task published.",
          "Failed to publish task"
        )
      : Promise.resolve();

  const handleCancelTask = () =>
    hasTaskId
      ? runAction(
          () => cancelMutation.mutateAsync({ id: taskId }),
          "Task canceled.",
          "Failed to cancel task"
        )
      : Promise.resolve();

  const handleEnqueueRun = () =>
    hasTaskId
      ? runAction(
          () => enqueueMutation.mutateAsync({ id: taskId }),
          "Run queued.",
          "Failed to queue run"
        )
      : Promise.resolve();

  const handleApproveTask = () =>
    hasTaskId
      ? runAction(
          () => approveMutation.mutateAsync({ id: taskId }),
          "Task approved.",
          "Failed to approve task"
        )
      : Promise.resolve();

  const handleRejectTask = () =>
    hasTaskId
      ? runAction(
          () => rejectMutation.mutateAsync({ id: taskId }),
          "Task rejected.",
          "Failed to reject task"
        )
      : Promise.resolve();

  const handleRetryRun = (runId: string) =>
    runAction(
      () => retryRunMutation.mutateAsync({ runId }),
      "Retry queued.",
      "Failed to retry run"
    );

  const handleClearBlock = (blockId: string) =>
    hasTaskId
      ? runAction(
          () => clearBlockMutation.mutateAsync({ id: taskId, blockId }),
          "Block cleared.",
          "Failed to clear block"
        )
      : Promise.resolve();

  const handleFanOutRuns = async (data: FanOutTaskRunsRequest) => {
    if (!hasTaskId) return;
    try {
      const result = await fanOutMutation.mutateAsync({ id: taskId, data });
      const count = result?.runs?.length ?? 0;
      toast.success(count > 0 ? `Created ${count} run${count === 1 ? "" : "s"}.` : "Runs created.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to fan out runs");
      throw error;
    }
  };

  const handlePauseTask = async (reason: string) => {
    if (!hasTaskId) return;
    try {
      await pauseMutation.mutateAsync({ id: taskId, data: { reason } });
      toast.success("Task paused.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to pause task");
      throw error;
    }
  };

  const handleResumeTask = async () => {
    if (!hasTaskId) return;
    try {
      await resumeMutation.mutateAsync({ id: taskId });
      toast.success("Task resumed.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to resume task");
      throw error;
    }
  };

  const handleRecoverTask = async () => {
    if (
      !hasTaskId ||
      (activeRunNeedsAttention && !recoverableRunId) ||
      recoverTaskMutation.isPending ||
      recoverRunMutation.isPending
    ) {
      return;
    }

    try {
      if (recoverableRunId) {
        await recoverRunMutation.mutateAsync({ runId: recoverableRunId, taskId });
        toast.success("Run recovered.");
      } else {
        await recoverTaskMutation.mutateAsync({ id: taskId });
        toast.success("Task recovered.");
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to recover task");
      throw error;
    }
  };

  const isTimelineSaturated =
    typeof timelineFilters.limit === "number" && timeline.length >= timelineFilters.limit;

  return {
    activeRun,
    detail,
    detailError: detailQuery.error ?? null,
    detailLoading: detailQuery.isLoading && !detail,
    fatalError,
    handleApproveTask,
    handleCancelTask,
    handleClearBlock,
    handleEnqueueRun,
    handleFanOutRuns,
    handlePauseTask,
    handlePublishTask,
    handleRecoverTask,
    handleRejectTask,
    handleResumeTask,
    handleRetryRun,
    handleTimelineLoadMore,
    inspect,
    inspectError: inspectQuery.error ?? null,
    inspectLoading: inspectQuery.isLoading && !inspect,
    isApprovePending: approveMutation.isPending,
    isCancelPending: cancelMutation.isPending,
    isClearBlockPending: clearBlockMutation.isPending,
    isEnqueuePending: enqueueMutation.isPending,
    isFanOutPending: fanOutMutation.isPending,
    isLive,
    isPausePending: pauseMutation.isPending,
    isPublishPending: publishMutation.isPending,
    isRecoverPending: recoverTaskMutation.isPending || recoverRunMutation.isPending,
    isRejectPending: rejectMutation.isPending,
    isResumePending: resumeMutation.isPending,
    isRetryPending: retryRunMutation.isPending,
    isTimelineSaturated,
    notFound: detailQuery.isError && detailQuery.error?.message?.includes("not found"),
    profile,
    profileError: profileQuery.error ?? null,
    profileLoading: profileQuery.isLoading && !profile,
    reviews,
    reviewsError: reviewsQuery.error ?? null,
    reviewsLoading: reviewsQuery.isLoading && reviews.length === 0,
    runs,
    runsError: runsQuery.error ?? null,
    runsLoading: runsQuery.isLoading && runs.length === 0,
    taskId,
    timeline,
    timelineError: timelineQuery.error ?? null,
    timelineLimit,
    timelineLoading: timelineQuery.isLoading && timeline.length === 0,
  };
}

function isRunActive(status?: string | null): boolean {
  return (
    status === "running" || status === "claimed" || status === "starting" || status === "queued"
  );
}

export { useTaskDetailPage };
export type { UseTaskDetailPageOptions };
