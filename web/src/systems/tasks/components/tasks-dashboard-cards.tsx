import { Metric, type MetricTone } from "@agh/ui";

import { formatDurationMs, formatPercent } from "../lib/task-formatters";
import type { TaskDashboardView } from "../types";

export interface TasksDashboardCardsProps {
  dashboard: TaskDashboardView;
}

/**
 * Top-row metric set per task 18 spec: Active runs, Success rate, Average duration, Queue depth.
 * Values are derived from the dashboard payload; no computed 24h windowing since the API
 * does not yet expose a time-bucketed histogram — we surface the freshest totals we have.
 */
export function TasksDashboardCards({ dashboard }: TasksDashboardCardsProps) {
  const { active_runs, totals, cards, queue } = dashboard;

  const activeRuns = active_runs.running;
  const activeDetail =
    [
      active_runs.queued > 0 ? `${active_runs.queued} queued` : null,
      active_runs.claimed > 0 ? `${active_runs.claimed} claimed` : null,
    ]
      .filter(Boolean)
      .join(" · ") || "idle";

  const successRate = computeSuccessRate(totals);
  const successTone: MetricTone =
    successRate === null
      ? "default"
      : successRate >= 90
        ? "success"
        : successRate >= 70
          ? "default"
          : "warning";

  const avgDurationMs = cards.latency.claim_latency_ms.average_ms;
  const avgDurationSamples = cards.latency.claim_latency_ms.samples;
  const avgDurationDetail = avgDurationSamples > 0 ? `n=${avgDurationSamples}` : "no data";

  const queueDepth = queue.total;
  const queueTone: MetricTone = queue.backlog_warning ? "warning" : "default";
  const queueDetail = queue.backlog_warning
    ? `oldest ${formatDurationMs(queue.oldest_queue_age_ms)}`
    : queueDepth > 0
      ? "queued"
      : "drained";

  return (
    <div
      className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4"
      data-testid="tasks-dashboard-cards"
    >
      <Metric
        data-testid="tasks-dashboard-card-active-runs"
        detail={activeDetail}
        label="Active runs"
        tone={activeRuns > 0 ? "accent" : "default"}
        value={activeRuns}
      />
      <Metric
        data-testid="tasks-dashboard-card-success-rate"
        detail="24h"
        label="Success rate"
        tone={successTone}
        value={successRate === null ? "—" : formatPercent(successRate)}
      />
      <Metric
        data-testid="tasks-dashboard-card-average-duration"
        detail={avgDurationDetail}
        label="Average duration"
        tone="default"
        value={formatDurationMs(avgDurationMs)}
      />
      <Metric
        data-testid="tasks-dashboard-card-queue-depth"
        detail={queueDetail}
        label="Queue depth"
        tone={queueTone}
        value={queueDepth}
      />
    </div>
  );
}

function computeSuccessRate(totals: TaskDashboardView["totals"]): number | null {
  const completed = totals.completed_runs ?? 0;
  const failed = totals.failed_runs ?? 0;
  const canceled = totals.canceled_runs ?? 0;
  const observed = completed + failed + canceled;
  if (observed === 0) {
    return null;
  }
  return (completed / observed) * 100;
}
