import { Time } from "@agh/ui";

import { computeElapsed } from "../lib/task-formatters";
import type { TaskRun, TaskRunDetailView } from "../types";
import { TaskStateBand } from "./task-state-band";

/**
 * Terminal outcome band for the run page (§4.9): failed, completed, or stuck.
 * Renders nothing while the run is live — the head status already says so.
 */
export function TaskRunOutcome({
  run,
  nextAttempt,
}: {
  run: TaskRunDetailView;
  nextAttempt: TaskRun | null;
}) {
  const record = run.run;
  const duration = computeElapsed(record);

  if (record.status === "failed") {
    return (
      <TaskStateBand
        body={
          <>
            {record.error ?? "The run ended before reaching a checkpoint."}
            {nextAttempt ? ` Attempt ${nextAttempt.attempt} was queued from this run.` : null}
          </>
        }
        data-testid="tasks-run-outcome-failed"
        micro={`task.run_failed · ${record.id}`}
        title={duration ? `Failed after ${duration}` : "Failed"}
        tone="danger"
      />
    );
  }

  if (record.status === "completed") {
    return (
      <TaskStateBand
        body={
          record.ended_at ? (
            <>
              Finished <Time iso={record.ended_at} mode="relative" />
              {duration ? ` in ${duration}` : null}. The result is ready below.
            </>
          ) : (
            "The result is ready below."
          )
        }
        data-testid="tasks-run-outcome-completed"
        title="Completed"
        tone="success"
      />
    );
  }

  if (record.status === "needs_attention") {
    return (
      <TaskStateBand
        body={
          record.error ??
          "The agent stopped responding. Recover requeues the work as a fresh attempt."
        }
        data-testid="tasks-run-outcome-stuck"
        micro={`task_run_stuck · ${record.id}`}
        title="This attempt looks stuck"
        tone="danger"
      />
    );
  }

  return null;
}
