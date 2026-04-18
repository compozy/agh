import { useCallback, useMemo, useState } from "react";

import { useTask, useTaskRuns, useTaskTimeline, useTaskTree } from "@/systems/tasks";
import type { TaskRunsFilter, TaskTimelineFilter } from "@/systems/tasks";

type TaskDetailPanel = "overview" | "timeline" | "tree" | "runs";

interface UseTaskDetailPageOptions {
  initialPanel?: TaskDetailPanel;
  runFilters?: TaskRunsFilter;
  timelineFilters?: TaskTimelineFilter;
  enableTimeline?: boolean;
  enableTree?: boolean;
  enableRuns?: boolean;
}

function useTaskDetailPage(taskId: string, options: UseTaskDetailPageOptions = {}) {
  const [panel, setPanel] = useState<TaskDetailPanel>(options.initialPanel ?? "overview");

  const hasTaskId = Boolean(taskId);
  const enableTimeline = options.enableTimeline ?? true;
  const enableTree = options.enableTree ?? true;
  const enableRuns = options.enableRuns ?? true;

  const detailQuery = useTask(taskId, { enabled: hasTaskId });
  const timelineQuery = useTaskTimeline(taskId, options.timelineFilters ?? {}, {
    enabled: hasTaskId && enableTimeline,
  });
  const treeQuery = useTaskTree(taskId, { enabled: hasTaskId && enableTree });
  const runsQuery = useTaskRuns(taskId, options.runFilters ?? {}, {
    enabled: hasTaskId && enableRuns,
  });

  const detail = detailQuery.data ?? null;
  const runs = runsQuery.data ?? [];
  const timeline = timelineQuery.data ?? [];
  const tree = treeQuery.data ?? null;

  const activeRun = useMemo(() => detail?.summary?.active_run ?? null, [detail]);

  const fatalError = useMemo(() => {
    if (!hasTaskId) {
      return new Error("Missing task id");
    }

    return detailQuery.error ?? null;
  }, [detailQuery.error, hasTaskId]);

  const handlePanelChange = useCallback((next: TaskDetailPanel) => {
    setPanel(next);
  }, []);

  return {
    activeRun,
    detail,
    detailError: detailQuery.error ?? null,
    detailLoading: detailQuery.isLoading && !detail,
    fatalError,
    handlePanelChange,
    notFound: detailQuery.isError && detailQuery.error?.message?.includes("not found"),
    panel,
    runs,
    runsError: runsQuery.error ?? null,
    runsLoading: runsQuery.isLoading && runs.length === 0,
    taskId,
    timeline,
    timelineError: timelineQuery.error ?? null,
    timelineLoading: timelineQuery.isLoading && timeline.length === 0,
    tree,
    treeError: treeQuery.error ?? null,
    treeLoading: treeQuery.isLoading && !tree,
  };
}

export { useTaskDetailPage };
export type { TaskDetailPanel, UseTaskDetailPageOptions };
