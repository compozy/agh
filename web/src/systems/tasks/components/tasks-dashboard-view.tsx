import { AlertCircle, Loader2 } from "lucide-react";

import { Empty } from "@agh/ui";

import { formatRelativeTime } from "../lib/task-formatters";
import type { TaskDashboardView } from "../types";
import { TasksDashboardActiveRuns } from "./tasks-dashboard-active-runs";
import { TasksDashboardCards } from "./tasks-dashboard-cards";
import { TasksDashboardQueueHealth } from "./tasks-dashboard-queue-health";
import { TasksDashboardStatusBreakdown } from "./tasks-dashboard-status-breakdown";

export interface TasksDashboardViewProps {
  dashboard: TaskDashboardView | null;
  isLoading?: boolean;
  errorMessage?: string | null;
}

export function TasksDashboardView({
  dashboard,
  isLoading = false,
  errorMessage = null,
}: TasksDashboardViewProps) {
  if (isLoading && !dashboard) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center py-10"
        data-testid="tasks-dashboard-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (errorMessage && !dashboard) {
    return (
      <Empty
        icon={AlertCircle}
        title="Unable to load dashboard"
        description={errorMessage}
        data-testid="tasks-dashboard-error"
      />
    );
  }

  if (!dashboard) {
    return (
      <Empty
        description="Create or run tasks to see queue depth, health, freshness, and live work in one place."
        icon={AlertCircle}
        title="No dashboard data yet"
        data-testid="tasks-dashboard-empty"
      />
    );
  }

  const freshness = dashboard.freshness;
  const freshnessLabel =
    freshness.stale || !freshness.has_live_work
      ? `Updated ${formatRelativeTime(freshness.observed_at)} ago`
      : `Live · updated ${formatRelativeTime(freshness.observed_at)} ago`;

  return (
    <div
      className="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-4 py-4"
      data-testid="tasks-dashboard-view"
    >
      <TasksDashboardCards dashboard={dashboard} />

      <div className="grid grid-cols-1 gap-5 xl:grid-cols-3">
        <div className="xl:col-span-2">
          <TasksDashboardQueueHealth dashboard={dashboard} />
        </div>
        <TasksDashboardStatusBreakdown dashboard={dashboard} />
      </div>

      <TasksDashboardActiveRuns dashboard={dashboard} />

      <div className="flex items-center justify-between gap-2 border-t border-[color:var(--color-divider)] pt-3 font-mono text-[11px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]">
        <span data-testid="tasks-dashboard-freshness">
          {freshness.stale ? "Stale" : "Fresh"} · {freshnessLabel}
        </span>
        <span>
          {dashboard.totals.tasks_total} tasks · {dashboard.totals.runs_total} runs
        </span>
      </div>
    </div>
  );
}
