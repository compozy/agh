import { formatPercent, taskStatusLabel, taskStatusTone } from "../lib/task-formatters";
import type { TaskDashboardView } from "../types";

export interface TasksDashboardStatusBreakdownProps {
  dashboard: TaskDashboardView;
}

const TONE_BAR_CLASSES: Record<string, string> = {
  neutral: "bg-[color:var(--color-text-tertiary)]",
  amber: "bg-[color:var(--color-warning)]",
  green: "bg-[color:var(--color-success)]",
  violet: "bg-[color:var(--color-info)]",
  danger: "bg-[color:var(--color-danger)]",
};

export function TasksDashboardStatusBreakdown({ dashboard }: TasksDashboardStatusBreakdownProps) {
  const entries = dashboard.status_breakdown ?? [];

  return (
    <section
      className="flex flex-col gap-3 rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] p-4"
      data-testid="tasks-dashboard-status-breakdown"
    >
      <p className="font-mono text-[0.62rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
        Status Breakdown
      </p>
      {entries.length === 0 ? (
        <p
          className="text-sm text-[color:var(--color-text-secondary)]"
          data-testid="tasks-dashboard-status-breakdown-empty"
        >
          No task activity yet.
        </p>
      ) : (
        <ul className="flex flex-col gap-3">
          {entries.map(entry => {
            const tone = taskStatusTone(entry.status);
            const barClass = TONE_BAR_CLASSES[tone] ?? TONE_BAR_CLASSES.neutral;
            const width = Math.max(0, Math.min(100, Math.round(entry.share_percent)));

            return (
              <li
                className="flex flex-col gap-1.5"
                data-testid={`tasks-dashboard-status-row-${entry.status}`}
                key={entry.status}
              >
                <div className="flex items-center justify-between gap-2 text-xs">
                  <span className="text-[color:var(--color-text-secondary)]">
                    {taskStatusLabel(entry.status)}
                  </span>
                  <span className="font-mono text-[0.62rem] uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]">
                    {entry.count} · {formatPercent(entry.share_percent)}
                  </span>
                </div>
                <div className="h-1.5 w-full overflow-hidden rounded-full bg-[color:var(--color-surface-panel)]">
                  <div
                    className={`h-full ${barClass}`}
                    data-testid={`tasks-dashboard-status-bar-${entry.status}`}
                    style={{ width: `${width}%` }}
                  />
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
