import { AlertCircle } from "lucide-react";

import { BlockLoading, Empty, Eyebrow } from "@agh/ui";

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
      <BlockLoading
        label="Loading tasks dashboard"
        size="md"
        surface="bare"
        data-testid="tasks-dashboard-loading"
      />
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

      <div className="flex items-center justify-between gap-2 border-t border-(--color-divider) pt-3">
        <Eyebrow data-testid="tasks-dashboard-freshness">
          {freshness.stale ? "Stale" : "Fresh"} · {freshnessLabel}
        </Eyebrow>
        <Eyebrow>
          {dashboard.totals.tasks_total} tasks · {dashboard.totals.runs_total} runs
        </Eyebrow>
      </div>
    </div>
  );
}
