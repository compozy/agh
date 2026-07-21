import { JsonViewer } from "@agh/ui";

import type { TaskDetailView } from "../types";

/**
 * Inspect drawer › Raw pane: the task record and (when present) the active
 * run DTO, verbatim. The operator floor — nothing rendered above hides this.
 */
export function TaskRawPane({ detail }: { detail: TaskDetailView }) {
  const activeRun = detail.summary?.active_run ?? null;
  return (
    <div className="flex flex-col gap-4" data-testid="tasks-inspect-raw">
      <section className="flex flex-col gap-2">
        <span className="eyebrow text-subtle">Task</span>
        <JsonViewer value={detail.task} />
      </section>
      {activeRun ? (
        <section className="flex flex-col gap-2">
          <span className="eyebrow text-subtle">Active run</span>
          <JsonViewer value={activeRun} />
        </section>
      ) : null}
    </div>
  );
}
