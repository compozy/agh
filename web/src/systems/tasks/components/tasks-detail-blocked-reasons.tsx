import { Pill } from "@agh/ui";

import { projectBlockedReasonChips } from "../lib/task-formatters";
import type { TaskBlockedReason } from "../types";

export interface TasksDetailBlockedReasonsProps {
  reasons?: TaskBlockedReason[] | null;
}

/**
 * Renders the read-only `blocked_reasons` projection as one chip per blocking
 * cause (source + kind + reason). Truthful UI: only payload-carried data is
 * shown, and the component renders nothing when the task carries no open causes
 * so the parent `Section` stays out of the layout entirely.
 */
export function TasksDetailBlockedReasons({ reasons }: TasksDetailBlockedReasonsProps) {
  const chips = projectBlockedReasonChips(reasons);
  if (chips.length === 0) {
    return null;
  }

  return (
    <ul className="flex flex-col gap-2" data-testid="tasks-detail-blocked-reasons">
      {chips.map(chip => (
        <li
          key={chip.key}
          className="flex min-w-0 flex-wrap items-center gap-2"
          data-source={chip.source}
          data-testid="tasks-detail-blocked-reason"
        >
          <Pill size="sm" tone={chip.tone}>
            {chip.kindLabel ? `${chip.sourceLabel} · ${chip.kindLabel}` : chip.sourceLabel}
          </Pill>
          {chip.reason ? (
            <span className="min-w-0 text-small-body text-muted">{chip.reason}</span>
          ) : null}
          {chip.dependsOnTaskIds ? (
            <span className="min-w-0 text-small-body text-muted">
              Waiting on {chip.dependsOnTaskIds.join(", ")}
            </span>
          ) : null}
        </li>
      ))}
    </ul>
  );
}
