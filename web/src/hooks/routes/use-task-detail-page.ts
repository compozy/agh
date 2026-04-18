import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";

import {
  useCancelTask,
  useEnqueueTaskRun,
  usePublishTask,
  useTask,
  useTaskRuns,
  useTaskTimeline,
  useTaskTree,
} from "@/systems/tasks";
import type { TaskRunsFilter, TaskTimelineFilter } from "@/systems/tasks";

type TaskDetailPanel = "overview" | "timeline" | "runs" | "children" | "dependencies";

interface UseTaskDetailPageOptions {
  initialPanel?: TaskDetailPanel;
  initialTimelineLimit?: number;
  runFilters?: TaskRunsFilter;
  timelineFilters?: TaskTimelineFilter;
  enableTimeline?: boolean;
  enableTree?: boolean;
  enableRuns?: boolean;
}

const DEFAULT_TIMELINE_LIMIT = 50;
const TIMELINE_PAGE_SIZE = 50;

function useTaskDetailPage(taskId: string, options: UseTaskDetailPageOptions = {}) {
  const [panel, setPanel] = useState<TaskDetailPanel>(options.initialPanel ?? "overview");
  const [timelineLimit, setTimelineLimit] = useState<number>(
    options.initialTimelineLimit ?? DEFAULT_TIMELINE_LIMIT
  );

  const hasTaskId = Boolean(taskId);
  const enableTimeline = options.enableTimeline ?? true;
  const enableTree = options.enableTree ?? true;
  const enableRuns = options.enableRuns ?? true;

  const timelineFilters: TaskTimelineFilter = useMemo(
    () => ({
      limit: timelineLimit,
      after_sequence: options.timelineFilters?.after_sequence,
    }),
    [options.timelineFilters?.after_sequence, timelineLimit]
  );

  const detailQuery = useTask(taskId, { enabled: hasTaskId });
  const timelineQuery = useTaskTimeline(taskId, timelineFilters, {
    enabled: hasTaskId && enableTimeline,
  });
  const treeQuery = useTaskTree(taskId, { enabled: hasTaskId && enableTree });
  const runsQuery = useTaskRuns(taskId, options.runFilters ?? {}, {
    enabled: hasTaskId && enableRuns,
  });

  const publishMutation = usePublishTask();
  const cancelMutation = useCancelTask();
  const enqueueMutation = useEnqueueTaskRun();

  const detail = detailQuery.data ?? null;
  const runs = runsQuery.data ?? [];
  const timeline = timelineQuery.data ?? [];
  const tree = treeQuery.data ?? null;

  const activeRun = useMemo(() => detail?.summary?.active_run ?? null, [detail]);
  const isLive = useMemo(() => {
    if (!activeRun) {
      return false;
    }

    return (
      activeRun.status === "running" ||
      activeRun.status === "claimed" ||
      activeRun.status === "starting" ||
      activeRun.status === "queued"
    );
  }, [activeRun]);

  const fatalError = useMemo(() => {
    if (!hasTaskId) {
      return new Error("Missing task id");
    }

    return detailQuery.error ?? null;
  }, [detailQuery.error, hasTaskId]);

  const handlePanelChange = useCallback((next: TaskDetailPanel) => {
    setPanel(next);
  }, []);

  const handleTimelineLoadMore = useCallback(() => {
    setTimelineLimit(current => current + TIMELINE_PAGE_SIZE);
  }, []);

  const handleTimelineReset = useCallback(() => {
    setTimelineLimit(options.initialTimelineLimit ?? DEFAULT_TIMELINE_LIMIT);
  }, [options.initialTimelineLimit]);

  const handlePublishTask = useCallback(async () => {
    if (!hasTaskId) {
      return;
    }

    try {
      await publishMutation.mutateAsync({ id: taskId });
      toast.success("Task published.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to publish task");
    }
  }, [hasTaskId, publishMutation, taskId]);

  const handleCancelTask = useCallback(async () => {
    if (!hasTaskId) {
      return;
    }

    try {
      await cancelMutation.mutateAsync({ id: taskId });
      toast.success("Task canceled.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to cancel task");
    }
  }, [cancelMutation, hasTaskId, taskId]);

  const handleEnqueueRun = useCallback(async () => {
    if (!hasTaskId) {
      return;
    }

    try {
      await enqueueMutation.mutateAsync({ id: taskId });
      toast.success("Run enqueued.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to enqueue run");
    }
  }, [enqueueMutation, hasTaskId, taskId]);

  const isTimelineSaturated =
    typeof timelineFilters.limit === "number" && timeline.length >= timelineFilters.limit;

  return {
    activeRun,
    detail,
    detailError: detailQuery.error ?? null,
    detailLoading: detailQuery.isLoading && !detail,
    fatalError,
    handleCancelTask,
    handleEnqueueRun,
    handlePanelChange,
    handlePublishTask,
    handleTimelineLoadMore,
    handleTimelineReset,
    isCancelPending: cancelMutation.isPending,
    isEnqueuePending: enqueueMutation.isPending,
    isLive,
    isPublishPending: publishMutation.isPending,
    isTimelineSaturated,
    notFound: detailQuery.isError && detailQuery.error?.message?.includes("not found"),
    panel,
    runs,
    runsError: runsQuery.error ?? null,
    runsLoading: runsQuery.isLoading && runs.length === 0,
    taskId,
    timeline,
    timelineError: timelineQuery.error ?? null,
    timelineLimit,
    timelineLoading: timelineQuery.isLoading && timeline.length === 0,
    tree,
    treeError: treeQuery.error ?? null,
    treeLoading: treeQuery.isLoading && !tree,
  };
}

export { useTaskDetailPage };
export type { TaskDetailPanel, UseTaskDetailPageOptions };
