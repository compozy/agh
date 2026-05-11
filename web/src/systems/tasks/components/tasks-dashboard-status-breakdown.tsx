import { Eyebrow, type PillTone, Pill } from "@agh/ui";

import { formatPercent, taskStatusLabel, taskStatusTone } from "../lib/task-formatters";
import type { TaskDashboardView } from "../types";
import { TasksDashboardPanel } from "./tasks-dashboard-panel";

export interface TasksDashboardStatusBreakdownProps {
  dashboard: TaskDashboardView;
}

// Tone tokens are exposed via Tailwind `--color-*` namespace only; raw
// `var(--success)` etc. do not resolve on `:root`, which breaks the default
// inline color path inside `<Pill.Dot>` (and every consumer that resolves a
// signal hex via `var(--token)`). The map below paints both the dot (passed
// as the `color` override) and the progress fill from the `--color-*` ladder.
// TODO(escalate): the `<Pill.Dot>` primitive itself should resolve via
// `--color-*` so callers stop having to thread `color=`.
const TONE_COLOR_VAR: Record<PillTone, string> = {
  neutral: "var(--color-neutral)",
  accent: "var(--color-accent)",
  success: "var(--color-success)",
  warning: "var(--color-warning)",
  danger: "var(--color-danger)",
  info: "var(--color-info)",
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
        <ul className="flex flex-col gap-3">
          {entries.map(entry => {
            const tone = taskStatusTone(entry.status);
            const sharePct = Math.max(0, Math.min(100, entry.share_percent ?? 0));
            const toneColor = TONE_COLOR_VAR[tone];
            return (
              <li
                className="flex flex-col gap-1.5"
                data-testid={`tasks-dashboard-status-row-${entry.status}`}
                key={entry.status}
              >
                <div className="flex min-w-0 items-center gap-2 text-[12px]">
                  <Pill.Dot color={toneColor} size="sm" />
                  <span
                    className="min-w-0 flex-1 truncate font-medium text-fg"
                    data-testid={`tasks-dashboard-status-label-${entry.status}`}
                  >
                    {taskStatusLabel(entry.status)}
                  </span>
                  <span
                    className="shrink-0 font-mono text-mono-id tabular-nums text-faint"
                    data-testid={`tasks-dashboard-status-count-${entry.status}`}
                  >
                    {entry.count}
                  </span>
                  <span
                    className="w-12 shrink-0 text-right font-mono text-[11px] font-medium tabular-nums text-muted"
                    data-testid={`tasks-dashboard-status-share-${entry.status}`}
                  >
                    {formatPercent(sharePct)}
                  </span>
                </div>
                <div
                  aria-hidden="true"
                  className="ml-4 h-1 overflow-hidden rounded-xs bg-surface-glaze"
                >
                  <div
                    className="h-full rounded-xs"
                    data-testid={`tasks-dashboard-status-bar-${entry.status}`}
                    style={{
                      backgroundColor: toneColor,
                      width: `${sharePct}%`,
                    }}
                  />
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </TasksDashboardPanel>
  );
}
