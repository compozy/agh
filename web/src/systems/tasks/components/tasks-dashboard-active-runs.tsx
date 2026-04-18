import { AlertCircle, ArrowRight } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { Pill } from "@/components/design-system";

import { formatAttemptLabel, formatDurationMs, taskRunStatusTone } from "../lib/task-formatters";
import type { TaskDashboardView } from "../types";

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
    <section
      className="flex flex-col gap-3 rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] p-4"
      data-testid="tasks-dashboard-active-runs"
    >
      <div className="flex items-center justify-between gap-2">
        <p className="font-mono text-[0.62rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
          Active Runs · {dashboard.active_runs.total}
        </p>
        <div className="flex items-center gap-2 text-[0.64rem] font-mono uppercase tracking-[0.14em] text-[color:var(--color-text-tertiary)]">
          <span>{dashboard.active_runs.running} running</span>
          <span>·</span>
          <span>{dashboard.active_runs.queued} queued</span>
          <span>·</span>
          <span>{dashboard.active_runs.claimed} claimed</span>
        </div>
      </div>

      {visible.length === 0 ? (
        <p
          className="text-sm text-[color:var(--color-text-secondary)]"
          data-testid="tasks-dashboard-active-runs-empty"
        >
          No active runs right now.
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {visible.map(run => (
            <li
              className="flex flex-col gap-2 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-3 py-2.5"
              data-testid={`tasks-dashboard-active-run-${run.run_id}`}
              key={run.run_id}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="flex min-w-0 flex-col gap-1">
                  <div className="flex items-center gap-2 text-xs text-[color:var(--color-text-tertiary)]">
                    {run.task_identifier ? (
                      <span className="font-mono uppercase tracking-[0.12em]">
                        {run.task_identifier}
                      </span>
                    ) : null}
                    <Pill kind="state" tone={taskRunStatusTone(run.run_status)}>
                      {run.run_status}
                    </Pill>
                    {run.stuck ? (
                      <span
                        className="font-mono text-[0.6rem] uppercase tracking-[0.12em] text-[color:var(--color-danger)]"
                        data-testid={`tasks-dashboard-active-run-stuck-${run.run_id}`}
                      >
                        stuck
                      </span>
                    ) : null}
                  </div>
                  <p className="truncate text-sm font-medium text-[color:var(--color-text-primary)]">
                    {run.task_title}
                  </p>
                  <div className="flex items-center gap-3 text-xs text-[color:var(--color-text-secondary)]">
                    <span>{formatAttemptLabel(run.attempt, run.max_attempts) ?? "—"}</span>
                    <span>age {formatDurationMs(run.age_ms)}</span>
                  </div>
                </div>
                <Link
                  className="inline-flex items-center gap-1 font-mono text-[0.64rem] uppercase tracking-[0.14em] text-[color:var(--color-accent)] hover:underline"
                  data-testid={`tasks-dashboard-active-run-link-${run.run_id}`}
                  params={{ id: run.task_id, runId: run.run_id }}
                  to="/tasks/$id/runs/$runId"
                >
                  Open <ArrowRight className="size-3" />
                </Link>
              </div>
              {run.error ? (
                <p
                  className="flex items-start gap-1 text-xs text-[color:var(--color-danger)]"
                  data-testid={`tasks-dashboard-active-run-error-${run.run_id}`}
                >
                  <AlertCircle className="mt-0.5 size-3 shrink-0" />
                  <span className="truncate">{run.error}</span>
                </p>
              ) : null}
            </li>
          ))}
        </ul>
      )}

      {hidden > 0 ? (
        <p
          className="font-mono text-[0.62rem] uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]"
          data-testid="tasks-dashboard-active-runs-more"
        >
          +{hidden} more active runs
        </p>
      ) : null}
    </section>
  );
}
