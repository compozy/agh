import { useMemo } from "react";

import { useTask, useTaskRunDetail } from "@/systems/tasks";

interface UseTaskRunPageOptions {
  enableTaskDetail?: boolean;
}

function useTaskRunPage(taskId: string, runId: string, options: UseTaskRunPageOptions = {}) {
  const hasTaskId = Boolean(taskId);
  const hasRunId = Boolean(runId);
  const enableTaskDetail = options.enableTaskDetail ?? true;

  const runQuery = useTaskRunDetail(runId, { enabled: hasRunId });
  const taskQuery = useTask(taskId, { enabled: hasTaskId && enableTaskDetail });

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

  return {
    fatalError,
    notFound: runQuery.isError && runQuery.error?.message?.includes("not found"),
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
