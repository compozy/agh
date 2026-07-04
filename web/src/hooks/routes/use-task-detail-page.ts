import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";

import {
  useCancelTask,
  useEnqueueTaskRun,
  usePauseTask,
  usePublishTask,
  useRecoverTask,
  useResumeTask,
  useTask,
  useTaskInspect,
  useTaskRuns,
  useTaskStream,
  useTaskTimeline,
  useTaskTree,
} from "@/systems/tasks";
import type {
  TaskRunsFilter,
  TaskTimelineFilter,
  TaskTreeNode,
  TaskTreeView,
} from "@/systems/tasks";

type TaskDetailPanel =
  | "overview"
  | "timeline"
  | "runs"
  | "children"
  | "dependencies"
  | "agents"
  | "orchestration";

type MultiAgentLiveState = "loading" | "disconnected" | "no-descendants" | "no-active" | "ready";

interface MultiAgentAgent {
  node: TaskTreeNode;
  isRoot: boolean;
  isPrimary: boolean;
  isLive: boolean;
  label: string;
}

interface MultiAgentView {
  nodes: MultiAgentAgent[];
  liveCount: number;
  descendantCount: number;
  activeDescendants: number;
  hasActiveRoot: boolean;
  state: MultiAgentLiveState;
}

interface UseTaskDetailPageOptions {
  initialPanel?: TaskDetailPanel;
  initialTimelineLimit?: number;
  runFilters?: TaskRunsFilter;
  timelineFilters?: TaskTimelineFilter;
  enableTimeline?: boolean;
  enableTree?: boolean;
  enableRuns?: boolean;
  enableInspect?: boolean;
  enableStream?: boolean;
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
  const enableInspect = options.enableInspect ?? true;
  const enableStream = options.enableStream ?? true;

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
  const inspectQuery = useTaskInspect(taskId, { enabled: hasTaskId && enableInspect });

  const publishMutation = usePublishTask();
  const cancelMutation = useCancelTask();
  const enqueueMutation = useEnqueueTaskRun();
  const pauseMutation = usePauseTask();
  const resumeMutation = useResumeTask();
  const recoverMutation = useRecoverTask();

  const detail = detailQuery.data ?? null;
  const runs = runsQuery.data ?? [];
  const timeline = timelineQuery.data ?? [];
  const tree = treeQuery.data ?? null;
  const inspect = inspectQuery.data ?? null;

  const activeRun = useMemo(() => detail?.summary?.active_run ?? null, [detail]);
  const isLive = useMemo(() => isRunActive(activeRun?.status ?? null), [activeRun]);

  // Keep the always-visible header/overview surfaces (status, blocked-reasons,
  // needs_attention badge, wake indicator) fresh from needs_attention / recover /
  // run-lifecycle SSE events. The orchestration tab owns its own stream + resume-
  // card status, so gate this one out on that panel to avoid a duplicate
  // EventSource. Wait for the detail payload before connecting so we seed from the
  // real cursor instead of after_sequence=0 (a full-history replay + immediate
  // reconnect when latest_event_seq resolves).
  const detailEventSeq = detail?.task?.latest_event_seq;
  const hasEventSeq = typeof detailEventSeq === "number";
  useTaskStream(taskId, {
    enabled: hasTaskId && enableStream && panel !== "orchestration" && hasEventSeq,
    afterSequence: hasEventSeq ? Math.max(0, detailEventSeq) : undefined,
  });

  const multiAgent = useMemo<MultiAgentView>(
    () => deriveMultiAgentView(tree, treeQuery.isLoading, Boolean(treeQuery.error), isLive),
    [isLive, tree, treeQuery.error, treeQuery.isLoading]
  );

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

  const handlePauseTask = useCallback(
    async (reason: string) => {
      if (!hasTaskId) {
        return;
      }

      try {
        await pauseMutation.mutateAsync({ id: taskId, data: { reason } });
        toast.success("Task paused.");
      } catch (error) {
        const message = error instanceof Error ? error.message : "Failed to pause task";
        toast.error(message);
        throw error;
      }
    },
    [hasTaskId, pauseMutation, taskId]
  );

  const handleResumeTask = useCallback(async () => {
    if (!hasTaskId) {
      return;
    }

    try {
      await resumeMutation.mutateAsync({ id: taskId });
      toast.success("Task resumed.");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to resume task";
      toast.error(message);
      throw error;
    }
  }, [hasTaskId, resumeMutation, taskId]);

  const handleRecoverTask = useCallback(async () => {
    if (!hasTaskId) {
      return;
    }

    try {
      await recoverMutation.mutateAsync({ id: taskId });
      toast.success("Task recovered.");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to recover task";
      toast.error(message);
      throw error;
    }
  }, [hasTaskId, recoverMutation, taskId]);

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
    handlePauseTask,
    handlePublishTask,
    handleRecoverTask,
    handleResumeTask,
    handleTimelineLoadMore,
    handleTimelineReset,
    isCancelPending: cancelMutation.isPending,
    isEnqueuePending: enqueueMutation.isPending,
    isLive,
    isPausePending: pauseMutation.isPending,
    isPublishPending: publishMutation.isPending,
    isRecoverPending: recoverMutation.isPending,
    isResumePending: resumeMutation.isPending,
    isTimelineSaturated,
    inspect,
    inspectError: inspectQuery.error ?? null,
    inspectLoading: inspectQuery.isLoading && !inspect,
    multiAgent,
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

function isRunActive(status?: string | null): boolean {
  return (
    status === "running" || status === "claimed" || status === "starting" || status === "queued"
  );
}

function deriveMultiAgentView(
  tree: TaskTreeView | null,
  isLoading: boolean,
  hasError: boolean,
  rootIsLive: boolean
): MultiAgentView {
  if (!tree) {
    if (isLoading) {
      return buildEmptyMultiAgentView("loading");
    }

    if (hasError) {
      return buildEmptyMultiAgentView("disconnected");
    }

    return buildEmptyMultiAgentView("no-descendants");
  }

  const descendants = tree.descendants ?? [];
  const rootNode: MultiAgentAgent = {
    node: tree.root,
    isRoot: true,
    isPrimary: true,
    isLive: rootIsLive || isRunActive(tree.root.active_run?.status ?? null),
    label: agentLabel(tree.root),
  };

  const descendantNodes: MultiAgentAgent[] = descendants.map(node => ({
    node,
    isRoot: false,
    isPrimary: false,
    isLive: isRunActive(node.active_run?.status ?? null),
    label: agentLabel(node),
  }));

  const nodes = [rootNode, ...descendantNodes];
  const liveCount = nodes.reduce((total, item) => total + (item.isLive ? 1 : 0), 0);
  const activeDescendants = descendantNodes.reduce(
    (total, item) => total + (item.isLive ? 1 : 0),
    0
  );

  let state: MultiAgentLiveState = "ready";
  if (descendants.length === 0 && !rootNode.isLive) {
    state = "no-descendants";
  } else if (liveCount === 0) {
    state = "no-active";
  }

  return {
    nodes,
    liveCount,
    descendantCount: descendants.length,
    activeDescendants,
    hasActiveRoot: rootNode.isLive,
    state,
  };
}

function buildEmptyMultiAgentView(state: MultiAgentLiveState): MultiAgentView {
  return {
    nodes: [],
    liveCount: 0,
    descendantCount: 0,
    activeDescendants: 0,
    hasActiveRoot: false,
    state,
  };
}

function agentLabel(node: TaskTreeNode): string {
  const owner = node.task.owner;
  if (owner?.ref) {
    return owner.ref;
  }

  if (owner?.kind) {
    return owner.kind;
  }

  return node.task.identifier ?? node.task.id;
}

export { useTaskDetailPage };
export type {
  MultiAgentAgent,
  MultiAgentLiveState,
  MultiAgentView,
  TaskDetailPanel,
  UseTaskDetailPageOptions,
};
