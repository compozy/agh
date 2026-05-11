import { AlertCircle, ArrowRight } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { Eyebrow, Pill } from "@agh/ui";

import {
  formatAttemptLabel,
  formatDurationMs,
  taskRunStatusTone,
  taskStatusSignal,
} from "../lib/task-formatters";
import type { TaskDashboardView } from "../types";
import { TasksDashboardPanel } from "./tasks-dashboard-panel";

export interface TasksDashboardActiveRunsProps {
  dashboard: TaskDashboardView;
  maxItems?: number;
}

export function TasksDashboardActiveRuns({
  dashboard,
  maxItems = 6,
}: TasksDashboardActiveRunsProps) {
  const runs = dashboard.active_runs.items ?? [];
  const visible = runs.slice(0, maxItems);
  const hidden = runs.length - visible.length;

  return (
    <TasksDashboardPanel
      bodyClassName="p-0"
      data-testid="tasks-dashboard-active-runs"
      right={
        <Eyebrow className="text-muted">
          {dashboard.active_runs.running} running · {dashboard.active_runs.queued} queued ·{" "}
          {dashboard.active_runs.claimed} claimed
        </Eyebrow>
      }
      title={`Active runs · ${dashboard.active_runs.total}`}
    >
      {visible.length === 0 ? (
        <p
          className="px-4 py-6 text-[12px] text-muted"
          data-testid="tasks-dashboard-active-runs-empty"
        >
          No active runs right now.
        </p>
      ) : (
        <ul className="divide-y divide-line-soft" data-testid="tasks-dashboard-active-runs-list">
          {visible.map(run => {
            const signal = taskStatusSignal(run.task_status);
            return (
              <li
                className="grid grid-cols-[14px_minmax(0,1fr)_96px_100px_84px_84px] items-center gap-3 px-4 py-[9px] text-[12px] hover:bg-row-hover"
                data-testid={`tasks-dashboard-active-run-${run.run_id}`}
                key={run.run_id}
              >
                <span className="flex shrink-0 items-center justify-center">
                  <Pill.Dot pulse={signal.pulse} tone={signal.tone} />
                </span>
                <div className="flex min-w-0 items-center gap-2">
                  <span className="min-w-0 truncate text-[12.5px] font-medium tracking-section-head text-fg-strong">
                    {run.task_title}
                  </span>
                  {run.task_identifier ? (
                    <span className="shrink-0 font-mono text-[10.5px] text-faint">
                      {run.task_identifier}
                    </span>
                  ) : null}
                </div>
                <span className="font-mono text-[11px] tabular-nums text-muted">
                  {formatAttemptLabel(run.attempt, run.max_attempts) ?? "—"}
                </span>
                <span className="font-mono text-[11px] tabular-nums text-muted">
                  age {formatDurationMs(run.age_ms)}
                </span>
                <div className="flex min-w-0 items-center gap-1">
                  <Pill size="sm" tone={taskRunStatusTone(run.run_status)}>
                    {run.run_status}
                  </Pill>
                  {run.stuck ? (
                    <Pill
                      data-testid={`tasks-dashboard-active-run-stuck-${run.run_id}`}
                      size="sm"
                      tone="danger"
                    >
                      stuck
                    </Pill>
                  ) : null}
                </div>
                <Pill.Link
                  data-testid={`tasks-dashboard-active-run-link-${run.run_id}`}
                  render={
                    <Link
                      params={{ id: run.task_id, runId: run.run_id }}
                      to="/tasks/$id/runs/$runId"
                    />
                  }
                >
                  Open <ArrowRight className="size-3" />
                </Pill.Link>
                {run.error ? (
                  <p
                    className="col-span-6 flex items-start gap-1 pt-1 text-[11.5px] text-danger"
                    data-testid={`tasks-dashboard-active-run-error-${run.run_id}`}
                  >
                    <AlertCircle className="mt-px size-3 shrink-0" />
                    <span className="min-w-0 truncate">{run.error}</span>
                  </p>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}

      {hidden > 0 ? (
        <Eyebrow
          className="text-muted block px-4 pt-3 pb-3"
          data-testid="tasks-dashboard-active-runs-more"
        >
          +{hidden} more active runs
        </Eyebrow>
      ) : null}
    </TasksDashboardPanel>
  );
}
