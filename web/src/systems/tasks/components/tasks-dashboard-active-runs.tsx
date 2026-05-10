import { AlertCircle, ArrowRight } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { Eyebrow, Item, ItemGroup, ItemTitle, Pill, Section } from "@agh/ui";

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
        <Eyebrow>
          {dashboard.active_runs.running} running · {dashboard.active_runs.queued} queued ·{" "}
          {dashboard.active_runs.claimed} claimed
        </Eyebrow>
      }
    >
      {visible.length === 0 ? (
        <p
          className="px-1 py-6 text-sm text-(--muted)"
          data-testid="tasks-dashboard-active-runs-empty"
        >
          No active runs right now.
        </p>
      ) : (
        <ItemGroup className="gap-0">
          {visible.map(run => {
            const signal = taskStatusSignal(run.task_status);
            return (
              <Item
                className="flex-col gap-2 rounded-none border-x-0 border-t-0 border-b border-(--line) px-0 py-3 last:border-b-0"
                data-testid={`tasks-dashboard-active-run-${run.run_id}`}
                key={run.run_id}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex min-w-0 flex-col gap-1.5">
                    <div className="flex min-w-0 items-center gap-2">
                      <Pill.Dot pulse={signal.pulse} tone={signal.tone} />
                      <ItemTitle className="min-w-0 truncate text-small-body font-medium text-(--fg)">
                        {run.task_title}
                      </ItemTitle>
                    </div>
                    <div className="flex flex-wrap items-center gap-2 text-eyebrow">
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
                      <span className="font-mono text-badge text-(--subtle)">
                        {formatAttemptLabel(run.attempt, run.max_attempts) ?? "--"}
                      </span>
                      <span className="font-mono text-badge text-(--subtle)">
                        age {formatDurationMs(run.age_ms)}
                      </span>
                    </div>
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
                </div>
                {run.error ? (
                  <p
                    className="flex items-start gap-1 text-xs text-(--danger)"
                    data-testid={`tasks-dashboard-active-run-error-${run.run_id}`}
                  >
                    <AlertCircle className="mt-0.5 size-3 shrink-0" />
                    <span className="truncate">{run.error}</span>
                  </p>
                ) : null}
              </Item>
            );
          })}
        </ItemGroup>
      )}

      {hidden > 0 ? (
        <Eyebrow className="pt-2" data-testid="tasks-dashboard-active-runs-more">
          +{hidden} more active runs
        </Eyebrow>
      ) : null}
    </Section>
  );
}
