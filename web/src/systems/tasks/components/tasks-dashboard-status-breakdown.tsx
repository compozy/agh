import { Eyebrow, type PillTone, Pill } from "@agh/ui";

import { formatPercent, taskStatusLabel, taskStatusTone } from "../lib/task-formatters";
import type { TaskDashboardView } from "../types";
import { TasksDashboardPanel } from "./tasks-dashboard-panel";

export interface TasksDashboardStatusBreakdownProps {
  dashboard: TaskDashboardView;
}

const TONE_BAR_COLOR: Record<PillTone, string> = {
  neutral: "var(--neutral)",
  accent: "var(--accent)",
  success: "var(--success)",
  warning: "var(--warning)",
  danger: "var(--danger)",
  info: "var(--info)",
};

export function TasksDashboardStatusBreakdown({ dashboard }: TasksDashboardStatusBreakdownProps) {
  const entries = dashboard.status_breakdown ?? [];
  const total = entries.reduce((sum, entry) => sum + entry.count, 0);

  return (
    <TasksDashboardPanel
      data-testid="tasks-dashboard-status-breakdown"
      right={
        total > 0 ? (
          <Eyebrow className="text-muted" data-testid="tasks-dashboard-status-breakdown-total">
            total {total}
          </Eyebrow>
        ) : undefined
      }
      title="Status breakdown"
    >
      {entries.length === 0 ? (
        <p className="text-[12px] text-muted" data-testid="tasks-dashboard-status-breakdown-empty">
          No task activity yet.
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {entries.map(entry => {
            const tone = taskStatusTone(entry.status);
            const sharePct = entry.share_percent ?? 0;
            const barColor = TONE_BAR_COLOR[tone];
            return (
              <li
                className="grid grid-cols-[12px_1fr_36px] items-center gap-2 text-[12px]"
                data-testid={`tasks-dashboard-status-row-${entry.status}`}
                key={entry.status}
              >
                <span className="flex shrink-0 items-center justify-center">
                  <Pill.Dot tone={tone} />
                </span>
                <div className="flex min-w-0 flex-col gap-1">
                  <div className="flex min-w-0 items-baseline justify-between gap-2">
                    <span
                      className="min-w-0 truncate text-[12px] font-medium tracking-eyebrow text-fg"
                      data-testid={`tasks-dashboard-status-label-${entry.status}`}
                    >
                      {taskStatusLabel(entry.status)}
                    </span>
                    <span
                      className="shrink-0 font-mono text-[10.5px] tabular-nums text-faint"
                      data-testid={`tasks-dashboard-status-count-${entry.status}`}
                    >
                      {entry.count}
                    </span>
                  </div>
                  <div
                    aria-hidden="true"
                    className="h-[3px] overflow-hidden rounded-[2px] bg-surface-glaze"
                  >
                    <div
                      className="h-full rounded-[2px] opacity-85"
                      data-testid={`tasks-dashboard-status-bar-${entry.status}`}
                      style={{
                        backgroundColor: barColor,
                        width: `${Math.max(0, Math.min(100, sharePct))}%`,
                      }}
                    />
                  </div>
                </div>
                <span
                  className="text-right font-mono text-[11px] font-medium tabular-nums text-muted"
                  data-testid={`tasks-dashboard-status-share-${entry.status}`}
                >
                  {formatPercent(sharePct)}
                </span>
              </li>
            );
          })}
        </ul>
      )}
    </TasksDashboardPanel>
  );
}
