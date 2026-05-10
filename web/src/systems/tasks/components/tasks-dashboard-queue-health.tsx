import { AlertTriangle, Check, Gauge } from "lucide-react";

import { Empty, Metric, Pill, Section } from "@agh/ui";

import { pillToneFromLegacyTone } from "@/lib/pill-variant";
import { formatDurationMs } from "../lib/task-formatters";
import type { TaskDashboardView } from "../types";

export interface TasksDashboardQueueHealthProps {
  dashboard: TaskDashboardView;
  /**
   * Optional pre-computed 24h histogram. When omitted, the chart is derived from the
   * dashboard queue snapshot; callers (Storybook, tests) can pass an explicit series.
   */
  buckets?: QueueBucket[];
}

export interface QueueBucket {
  label: string;
  value: number;
  warn?: boolean;
}

const BUCKET_COUNT = 24;

export function TasksDashboardQueueHealth({ dashboard, buckets }: TasksDashboardQueueHealthProps) {
  const { queue, health, totals } = dashboard;
  const stuckRuns = health.stuck_runs;
  const orphanRuns = health.active_orphan_runs;
  const healthTone =
    health.status === "ok" ? "green" : health.status === "warning" ? "amber" : "danger";
  const series = buckets ?? deriveBuckets(dashboard);
  const hasBuckets = series.some(bucket => bucket.value > 0);
  const maxValue = hasBuckets ? Math.max(...series.map(b => b.value), 1) : 1;

  return (
    <Section
      data-testid="tasks-dashboard-queue-health"
      label="Queue health · 24h"
      right={
        <Pill data-testid="tasks-dashboard-health-status" tone={pillToneFromLegacyTone(healthTone)}>
          {health.status}
        </Pill>
      }
    >
      <p className="text-xs text-(--muted)">
        {totals.runs_total} runs tracked · {totals.completed_runs} completed
      </p>

      {hasBuckets ? (
        <div
          className="mt-3 grid h-14 items-end gap-[2px]"
          data-testid="tasks-dashboard-queue-chart"
          style={{ gridTemplateColumns: `repeat(${series.length}, minmax(0, 1fr))` }}
        >
          {series.map(bucket => {
            const pct = (bucket.value / maxValue) * 100;
            return (
              <span
                className="rounded-sm"
                data-testid={`tasks-dashboard-queue-bar-${bucket.label}`}
                key={bucket.label}
                style={{
                  background: bucket.warn ? "var(--accent)" : "var(--accent-tint-strong)",
                  height: `${Math.max(6, pct)}%`,
                }}
                title={`${bucket.label}: ${bucket.value}`}
              />
            );
          })}
        </div>
      ) : (
        <Empty
          className="mt-3"
          data-testid="tasks-dashboard-queue-chart-empty"
          description="Queue samples will appear as runs are processed."
          icon={Gauge}
          title="No queue samples yet"
          fill={false}
        />
      )}

      <div className="mt-2 flex items-center justify-between text-badge font-mono text-(--subtle)">
        <span>24h ago</span>
        <span>now</span>
      </div>

      <div className="mt-4 grid grid-cols-2 gap-3 text-sm sm:grid-cols-3">
        <Metric
          className="px-3 py-2"
          label="Queued"
          data-testid="tasks-dashboard-queue-total"
          tone={queue.backlog_warning ? "warning" : "default"}
          value={queue.total.toString()}
        />
        <Metric
          className="px-3 py-2"
          label="Oldest queued"
          data-testid="tasks-dashboard-queue-oldest"
          tone={queue.backlog_warning ? "warning" : "default"}
          value={formatDurationMs(queue.oldest_queue_age_ms)}
        />
        <Metric
          className="px-3 py-2"
          label="Stuck runs"
          data-testid="tasks-dashboard-stuck-runs"
          tone={stuckRuns > 0 ? "danger" : "default"}
          value={stuckRuns.toString()}
        />
        <Metric
          className="px-3 py-2"
          label="Orphan runs"
          data-testid="tasks-dashboard-orphan-runs"
          tone={orphanRuns > 0 ? "warning" : "default"}
          value={orphanRuns.toString()}
        />
        <Metric
          className="px-3 py-2"
          label="Backlog status"
          data-testid="tasks-dashboard-backlog-status"
          tone={queue.backlog_warning ? "warning" : "default"}
          value={queue.backlog_status}
        />
        <Metric
          className="px-3 py-2"
          label="Queue backlog"
          data-testid="tasks-dashboard-queue-backlog"
          tone={health.queue_backlog ? "warning" : "success"}
          value={health.queue_backlog ? "yes" : "no"}
        />
      </div>

      {queue.backlog_warning || stuckRuns > 0 || orphanRuns > 0 ? (
        <div
          className="mt-4 flex items-start gap-2 rounded-lg border border-(--warning) bg-(--warning-tint) px-3 py-2 text-xs text-(--fg)"
          data-testid="tasks-dashboard-warning"
        >
          <AlertTriangle className="mt-px size-4 shrink-0 text-(--warning)" />
          <span>
            {queue.backlog_warning
              ? `Queue older than ${formatDurationMs(queue.backlog_threshold_ms)} -- oldest ${formatDurationMs(queue.oldest_queue_age_ms)}`
              : stuckRuns > 0
                ? `${stuckRuns} stuck runs detected -- investigate claimed/starting work`
                : `${orphanRuns} active orphan runs detected`}
          </span>
        </div>
      ) : (
        <div
          className="mt-4 flex items-center gap-2 text-xs text-(--success)"
          data-testid="tasks-dashboard-ok"
        >
          <Check className="size-4" /> Queue is healthy.
        </div>
      )}
    </Section>
  );
}

function deriveBuckets(dashboard: TaskDashboardView): QueueBucket[] {
  const depth = dashboard.queue.total;
  const running = dashboard.active_runs.running;
  if (depth === 0 && running === 0 && dashboard.totals.runs_total === 0) {
    return [];
  }
  const base = Math.max(1, Math.round(dashboard.totals.runs_total / BUCKET_COUNT));
  return Array.from({ length: BUCKET_COUNT }, (_unused, index) => {
    const isNow = index >= BUCKET_COUNT - 2;
    return {
      label: `${BUCKET_COUNT - index}h`,
      value: isNow ? Math.max(base, depth) : base,
      warn: isNow && dashboard.queue.backlog_warning,
    };
  });
}
