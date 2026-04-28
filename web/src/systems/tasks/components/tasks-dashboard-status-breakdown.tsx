import { Pill, Section } from "@agh/ui";

import { pillToneFromLegacyTone } from "@/lib/pill-variant";
import { formatPercent, taskStatusLabel, taskStatusTone } from "../lib/task-formatters";
import type { TaskDashboardView } from "../types";

export interface TasksDashboardStatusBreakdownProps {
  dashboard: TaskDashboardView;
}

export function TasksDashboardStatusBreakdown({ dashboard }: TasksDashboardStatusBreakdownProps) {
  const entries = dashboard.status_breakdown ?? [];
  const total = entries.reduce((sum, entry) => sum + entry.count, 0);

  return (
    <Section
      data-testid="tasks-dashboard-status-breakdown"
      label="Status breakdown"
      right={
        total > 0 ? (
          <span
            className="font-mono text-[11px] uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]"
            data-testid="tasks-dashboard-status-breakdown-total"
          >
            total {total}
          </span>
        ) : undefined
      }
    >
      {entries.length === 0 ? (
        <p
          className="px-1 py-6 text-sm text-[color:var(--color-text-secondary)]"
          data-testid="tasks-dashboard-status-breakdown-empty"
        >
          No task activity yet.
        </p>
      ) : (
        <ul className="flex flex-col gap-2 pt-2">
          {entries.map(entry => (
            <li
              className="flex items-center justify-between gap-3"
              data-testid={`tasks-dashboard-status-row-${entry.status}`}
              key={entry.status}
            >
              <Pill
                data-testid={`tasks-dashboard-status-pill-${entry.status}`}
                size="sm"
                tone={pillToneFromLegacyTone(taskStatusTone(entry.status))}
              >
                {taskStatusLabel(entry.status)}
                <span
                  className="ml-1 font-mono text-[10px] tracking-[0.06em] opacity-80"
                  data-testid={`tasks-dashboard-status-count-${entry.status}`}
                >
                  {entry.count}
                </span>
              </Pill>
              <span className="font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
                {formatPercent(entry.share_percent)}
              </span>
            </li>
          ))}
        </ul>
      )}
    </Section>
  );
}
