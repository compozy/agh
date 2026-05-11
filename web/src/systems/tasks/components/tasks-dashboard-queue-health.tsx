import { AlertTriangle, Check, Gauge } from "lucide-react";

import {
  Empty,
  Pill,
  type PillTone,
  QueueHealthSparkline,
  type QueueHealthSparklineBucket,
} from "@agh/ui";

import { formatDurationMs } from "../lib/task-formatters";
import type { TaskDashboardView } from "../types";
import { TasksDashboardPanel } from "./tasks-dashboard-panel";

export interface TasksDashboardQueueHealthProps {
  dashboard: TaskDashboardView;
  /**
   * Optional pre-computed 24h histogram. When omitted, the chart is derived
   * from the dashboard queue snapshot; callers (Storybook, tests) can pass
   * an explicit series. Shape matches `<QueueHealthSparkline>` per ADR-010 §3.
   */
  buckets?: QueueHealthSparklineBucket[];
}

/** Re-exported for legacy callers that built fixtures around the local shape. */
export type QueueBucket = QueueHealthSparklineBucket;

const BUCKET_COUNT = 24;
const SPARKLINE_HEIGHT = 96;

export function TasksDashboardQueueHealth({ dashboard, buckets }: TasksDashboardQueueHealthProps) {
  const { queue, health, totals } = dashboard;
  const stuckRuns = health.stuck_runs;
  const orphanRuns = health.active_orphan_runs;
  const healthTone: PillTone =
    health.status === "ok" ? "success" : health.status === "warning" ? "warning" : "danger";
  const series = buckets ?? deriveBuckets(dashboard);
  const hasBuckets = series.some(bucket => bucket.value > 0);

  return (
    <TasksDashboardPanel
      data-testid="tasks-dashboard-queue-health"
      right={
        <Pill data-testid="tasks-dashboard-health-status" tone={healthTone}>
          {health.status}
        </Pill>
      }
      title="Queue health · 24h"
    >
      <p className="text-[12px] text-(--muted)">
        {totals.runs_total} runs tracked · {totals.completed_runs} completed
      </p>

      {hasBuckets ? (
        <div className="mt-3 flex flex-col gap-1.5" data-testid="tasks-dashboard-queue-chart">
          <QueueHealthSparkline
            ariaLabel="Queue depth over the last 24 hours"
            data={series}
            height={SPARKLINE_HEIGHT}
          />
          <div className="flex items-center justify-between font-mono text-[10px] text-(--faint)">
            <span>24h ago</span>
            <span>now</span>
          </div>
        </div>
      ) : (
        <Empty
          className="mt-3"
          data-testid="tasks-dashboard-queue-chart-empty"
          description="Queue samples will appear as runs are processed."
          fill={false}
          icon={Gauge}
          title="No queue samples yet"
        />
      )}

      {queue.backlog_warning || stuckRuns > 0 || orphanRuns > 0 ? (
        <div
          className="mt-4 flex items-start gap-2 rounded-(--radius-lg) bg-(--warning-tint) px-3 py-2 text-xs text-(--fg)"
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
    </TasksDashboardPanel>
  );
}

function deriveBuckets(dashboard: TaskDashboardView): QueueHealthSparklineBucket[] {
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
      stuck: isNow && dashboard.queue.backlog_warning,
    };
  });
}
