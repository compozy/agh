import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TasksDashboardQueueHealth, type QueueBucket } from "../tasks-dashboard-queue-health";
import { buildDashboardFixture } from "../test-fixtures";

function buildBuckets(count: number): QueueBucket[] {
  return Array.from({ length: count }, (_unused, index) => ({
    label: `${count - index}h`,
    value: (index + 1) * 2,
    warn: index >= count - 2,
  }));
}

describe("TasksDashboardQueueHealth", () => {
  it("renders a 24-bar chart when supplied 24 hourly buckets", () => {
    const buckets = buildBuckets(24);
    render(<TasksDashboardQueueHealth buckets={buckets} dashboard={buildDashboardFixture()} />);

    expect(screen.getByTestId("tasks-dashboard-queue-chart")).toBeInTheDocument();
    const bars = screen
      .getByTestId("tasks-dashboard-queue-chart")
      .querySelectorAll("[data-testid^=tasks-dashboard-queue-bar-]");
    expect(bars).toHaveLength(24);
  });

  it("renders the Empty primitive when no buckets are available", () => {
    const dashboard = buildDashboardFixture();
    Object.assign(dashboard, {
      active_runs: { ...dashboard.active_runs, running: 0, total: 0 },
      queue: { ...dashboard.queue, total: 0 },
      totals: { ...dashboard.totals, runs_total: 0 },
    });

    render(<TasksDashboardQueueHealth buckets={[]} dashboard={dashboard} />);

    expect(screen.getByTestId("tasks-dashboard-queue-chart-empty")).toBeInTheDocument();
  });

  it("surfaces the queue warning banner when backlog_warning is true", () => {
    const dashboard = buildDashboardFixture({
      queue: {
        backlog_status: "warning",
        backlog_threshold_ms: 60000,
        backlog_warning: true,
        oldest_queue_age_ms: 180000,
        oldest_queued_at: "2026-04-17T09:57:00Z",
        total: 4,
      },
    });

    render(<TasksDashboardQueueHealth buckets={buildBuckets(24)} dashboard={dashboard} />);

    expect(screen.getByTestId("tasks-dashboard-warning")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-dashboard-queue-total")).toHaveTextContent("4");
  });
});
