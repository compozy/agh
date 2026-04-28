import { AlertCircle, ArrowRight } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { Pill, Section } from "@agh/ui";

import { pillToneFromLegacyTone } from "@/lib/pill-variant";
import {
  formatAttemptLabel,
  formatDurationMs,
  taskRunStatusTone,
  taskStatusSignal,
} from "../lib/task-formatters";
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
    <Section
      data-testid="tasks-dashboard-active-runs"
      label={`Active runs · ${dashboard.active_runs.total}`}
      right={
        <span className="font-mono text-[11px] uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]">
          {dashboard.active_runs.running} running · {dashboard.active_runs.queued} queued ·{" "}
          {dashboard.active_runs.claimed} claimed
        </span>
      }
    >
      {visible.length === 0 ? (
        <p
          className="px-1 py-6 text-sm text-[color:var(--color-text-secondary)]"
          data-testid="tasks-dashboard-active-runs-empty"
        >
          No active runs right now.
        </p>
      ) : (
        <ul className="flex flex-col">
          {visible.map(run => {
            const signal = taskStatusSignal(run.task_status);
            return (
              <li
                className="flex flex-col gap-2 border-b border-[color:var(--color-divider)] py-3 last:border-b-0"
                data-testid={`tasks-dashboard-active-run-${run.run_id}`}
                key={run.run_id}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex min-w-0 flex-col gap-1.5">
                    <div className="flex min-w-0 items-center gap-2">
                      <Pill.Dot pulse={signal.pulse} tone={signal.tone} />
                      <span className="min-w-0 truncate text-[13px] font-medium text-[color:var(--color-text-primary)]">
                        {run.task_title}
                      </span>
                    </div>
                    <div className="flex flex-wrap items-center gap-2 text-[11px]">
                      {run.task_identifier ? <Pill mono>{run.task_identifier}</Pill> : null}
                      <Pill tone={pillToneFromLegacyTone(taskRunStatusTone(run.run_status))}>
                        {run.run_status}
                      </Pill>
                      {run.stuck ? (
                        <Pill
                          data-testid={`tasks-dashboard-active-run-stuck-${run.run_id}`}
                          tone="danger"
                        >
                          stuck
                        </Pill>
                      ) : null}
                      <span className="font-mono text-[10px] text-[color:var(--color-text-tertiary)]">
                        {formatAttemptLabel(run.attempt, run.max_attempts) ?? "—"}
                      </span>
                      <span className="font-mono text-[10px] text-[color:var(--color-text-tertiary)]">
                        age {formatDurationMs(run.age_ms)}
                      </span>
                    </div>
                  </div>
                  <Link
                    className="inline-flex items-center gap-1 font-mono text-[11px] uppercase tracking-[0.12em] text-[color:var(--color-accent)] hover:underline"
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
            );
          })}
        </ul>
      )}

      {hidden > 0 ? (
        <p
          className="pt-2 font-mono text-[11px] uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]"
          data-testid="tasks-dashboard-active-runs-more"
        >
          +{hidden} more active runs
        </p>
      ) : null}
    </Section>
  );
}
