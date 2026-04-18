import { AlertTriangle, Check } from "lucide-react";

import { Pill } from "@/components/design-system";

import { formatDurationMs } from "../lib/task-formatters";
import type { TaskDashboardView } from "../types";

export interface TasksDashboardQueueHealthProps {
  dashboard: TaskDashboardView;
}

export function TasksDashboardQueueHealth({ dashboard }: TasksDashboardQueueHealthProps) {
  const { queue, health } = dashboard;
  const { totals } = dashboard;
  const stuckRuns = health.stuck_runs;
  const orphanRuns = health.active_orphan_runs;
  const healthTone =
    health.status === "ok" ? "green" : health.status === "warning" ? "amber" : "danger";

  return (
    <section
      className="flex flex-col gap-4 rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] p-4"
      data-testid="tasks-dashboard-queue-health"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex flex-col gap-1">
          <p className="font-mono text-[0.62rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
            Queue &amp; Health
          </p>
          <p className="text-sm text-[color:var(--color-text-secondary)]">
            {totals.runs_total} runs tracked · {totals.completed_runs} completed
          </p>
        </div>
        <Pill kind="state" tone={healthTone} data-testid="tasks-dashboard-health-status">
          {health.status}
        </Pill>
      </div>

      <dl className="grid grid-cols-2 gap-3 text-sm sm:grid-cols-3">
        <Metric
          label="Queued"
          value={queue.total.toString()}
          testId="tasks-dashboard-queue-total"
          tone={queue.backlog_warning ? "amber" : "neutral"}
        />
        <Metric
          label="Oldest queued"
          value={formatDurationMs(queue.oldest_queue_age_ms)}
          testId="tasks-dashboard-queue-oldest"
          tone={queue.backlog_warning ? "amber" : "neutral"}
        />
        <Metric
          label="Stuck runs"
          value={stuckRuns.toString()}
          testId="tasks-dashboard-stuck-runs"
          tone={stuckRuns > 0 ? "danger" : "neutral"}
        />
        <Metric
          label="Orphan runs"
          value={orphanRuns.toString()}
          testId="tasks-dashboard-orphan-runs"
          tone={orphanRuns > 0 ? "amber" : "neutral"}
        />
        <Metric
          label="Backlog status"
          value={queue.backlog_status}
          testId="tasks-dashboard-backlog-status"
          tone={queue.backlog_warning ? "amber" : "neutral"}
        />
        <Metric
          label="Queue backlog"
          value={health.queue_backlog ? "yes" : "no"}
          testId="tasks-dashboard-queue-backlog"
          tone={health.queue_backlog ? "amber" : "green"}
        />
      </dl>

      {queue.backlog_warning || stuckRuns > 0 || orphanRuns > 0 ? (
        <div
          className="flex items-start gap-2 rounded-xl border border-[color:var(--color-warning)] bg-[color:var(--color-accent-tint)] px-3 py-2 text-xs text-[color:var(--color-text-primary)]"
          data-testid="tasks-dashboard-warning"
        >
          <AlertTriangle className="mt-[1px] size-4 shrink-0 text-[color:var(--color-warning)]" />
          <span>
            {queue.backlog_warning
              ? `Queue older than ${formatDurationMs(queue.backlog_threshold_ms)} — oldest ${formatDurationMs(queue.oldest_queue_age_ms)}`
              : stuckRuns > 0
                ? `${stuckRuns} stuck runs detected — investigate claimed/starting work`
                : `${orphanRuns} active orphan runs detected`}
          </span>
        </div>
      ) : (
        <div
          className="flex items-center gap-2 text-xs text-[color:var(--color-success)]"
          data-testid="tasks-dashboard-ok"
        >
          <Check className="size-4" /> Queue is healthy.
        </div>
      )}
    </section>
  );
}

interface MetricProps {
  label: string;
  value: string;
  testId: string;
  tone: "neutral" | "amber" | "danger" | "green";
}

function Metric({ label, value, testId, tone }: MetricProps) {
  const valueTone =
    tone === "amber"
      ? "text-[color:var(--color-warning)]"
      : tone === "danger"
        ? "text-[color:var(--color-danger)]"
        : tone === "green"
          ? "text-[color:var(--color-success)]"
          : "text-[color:var(--color-text-primary)]";

  return (
    <div className="flex flex-col gap-0.5">
      <dt className="font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        {label}
      </dt>
      <dd className={`text-sm font-medium ${valueTone}`} data-testid={testId}>
        {value}
      </dd>
    </div>
  );
}
