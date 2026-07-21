import { Pill, Time } from "@agh/ui";

import { projectTaskExceptionPills } from "../lib/task-detail-pills";
import { taskOwnerLabel } from "../lib/task-formatters";
import type { TaskDetailView } from "../types";

function MetaDot() {
  return (
    <span aria-hidden="true" className="text-faint">
      ·
    </span>
  );
}

/**
 * Demoted meta line under the window head (§4.2): exception pills first, then
 * owner / creator / freshness. The head itself stays crumb + status + actions.
 */
export function TasksDetailSubhead({ detail }: { detail: TaskDetailView }) {
  const record = detail.task;
  const pills = projectTaskExceptionPills(detail);
  const closedAt = record.closed_at ?? null;

  return (
    <div
      className="mb-4 flex min-w-0 flex-wrap items-center gap-2 border-b border-line pb-3.5 text-form-label text-subtle"
      data-testid="tasks-detail-subhead"
    >
      {pills.map(pill => (
        <Pill
          data-testid={`tasks-detail-pill-${pill.key}`}
          key={pill.key}
          title={pill.title}
          tone={pill.tone}
        >
          {pill.label}
        </Pill>
      ))}
      <span>
        Owner <span className="font-medium text-muted">{taskOwnerLabel(record.owner)}</span>
      </span>
      <MetaDot />
      <span>
        Created by{" "}
        <span className="font-medium text-muted">{record.created_by?.ref ?? "unknown"}</span>
      </span>
      <MetaDot />
      {closedAt ? (
        <span className="inline-flex items-center gap-1">
          Closed <Time iso={closedAt} mode="relative" />
        </span>
      ) : (
        <span className="inline-flex items-center gap-1">
          Updated <Time iso={record.updated_at} mode="relative" />
        </span>
      )}
    </div>
  );
}
