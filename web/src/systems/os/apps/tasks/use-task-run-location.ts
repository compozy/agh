import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { toast } from "sonner";

import {
  taskRunCanRecover,
  useForceFailDialog,
  useTaskRunPage,
  useTaskRuns,
  useTaskStream,
  useTaskTimeline,
} from "@/systems/tasks";

/**
 * Controller for the run-detail window location: run page data, sibling runs
 * for lineage, run-scoped timeline slice, live stream attach (§11.4), and
 * WM-location navigation.
 */
export function useTaskRunLocation(taskId: string, runId: string) {
  const navigate = useNavigate();
  const page = useTaskRunPage(taskId, runId);
  const [inspectOpen, setInspectOpen] = useState(false);
  const forceFailDialog = useForceFailDialog(page.handleForceFailRun);

  const authoritativeTaskId = page.run?.task?.id ?? "";
  const timelineQuery = useTaskTimeline(
    authoritativeTaskId,
    {},
    { enabled: Boolean(authoritativeTaskId) }
  );
  const runsQuery = useTaskRuns(authoritativeTaskId, {}, { enabled: Boolean(authoritativeTaskId) });

  // §11.4: attach the task stream so this page is genuinely live before it
  // advertises live state.
  const latestEventSeq = page.task?.task?.latest_event_seq;
  const hasEventSeq = typeof latestEventSeq === "number";
  useTaskStream(authoritativeTaskId, {
    enabled: Boolean(authoritativeTaskId) && hasEventSeq,
    afterSequence: hasEventSeq ? Math.max(0, latestEventSeq) : undefined,
  });

  const record = page.run?.run ?? null;

  const backToTask = () => {
    void navigate({ to: "/tasks/$id", params: { id: authoritativeTaskId || taskId } });
  };

  const backToTasks = () => {
    void navigate({ to: "/tasks" });
  };

  const openSession = (sessionId: string) => {
    void navigate({ to: "/session/$id", params: { id: sessionId } });
  };

  const copyRunId = () => {
    if (!record || !navigator.clipboard) return;
    navigator.clipboard
      .writeText(record.id)
      .then(() => toast.success("Run id copied."))
      .catch(() => toast.error("Couldn't copy the run id"));
  };

  const canRecover = record
    ? record.status === "needs_attention" && taskRunCanRecover(record, page.task?.task.max_attempts)
    : false;

  return {
    authoritativeTaskId,
    backToTask,
    backToTasks,
    canRecover,
    copyRunId,
    forceFailDialog,
    inspectOpen,
    openSession,
    page,
    record,
    runId,
    setInspectOpen,
    taskId,
    taskRuns: runsQuery.data ?? [],
    timelineItems: timelineQuery.data ?? [],
  };
}

export type TaskRunLocationController = ReturnType<typeof useTaskRunLocation>;
