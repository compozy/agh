import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TasksDashboardCards } from "./tasks-dashboard-cards";
import type { TaskDashboardView } from "../types";

function buildDashboard(overrides: Partial<TaskDashboardView> = {}): TaskDashboardView {
  return {
    active_runs: {
      claimed: 1,
      queued: 2,
      running: 3,
      starting: 0,
      total: 6,
      items: [],
    },
    cards: {
      blocked: {
        awaiting_approval: 1,
        awaiting_dependencies: 1,
        health_status: "warning",
        tasks: 2,
      },
      failed: {
        failed_runs: 3,
        forced_stops: 0,
        health_status: "warning",
        tasks: 3,
      },
      in_progress: {
        active_runs: 3,
        claimed_runs: 1,
        queued_runs: 2,
        running_runs: 3,
        starting_runs: 0,
        tasks: 4,
        health_status: "ok",
      },
      latency: {
        claim_latency_ms: { average_ms: 1800, maximum_ms: 4200, samples: 12 },
        start_latency_ms: { average_ms: 900, maximum_ms: 2500, samples: 12 },
      },
    },
    freshness: {
      age_ms: 500,
      has_live_work: true,
      latest_activity_at: "2026-04-17T10:00:00Z",
      observed_at: "2026-04-17T10:00:01Z",
      stale: false,
      stale_after_ms: 60000,
      status: "fresh",
    },
    health: {
      active_orphan_runs: 0,
      queue_backlog: false,
      status: "ok",
      stuck_runs: 0,
    },
    queue: {
      backlog_status: "ok",
      backlog_threshold_ms: 60000,
      backlog_warning: false,
      oldest_queue_age_ms: 0,
      oldest_queued_at: "2026-04-17T10:00:00Z",
      total: 2,
    },
    status_breakdown: [
      { count: 12, share_percent: 43, status: "completed" },
      { count: 6, share_percent: 21, status: "pending" },
    ],
    totals: {
      active_runs: 3,
      awaiting_approval_tasks: 1,
      blocked_tasks: 2,
      canceled_runs: 0,
      canceled_tasks: 0,
      claimed_runs: 1,
      completed_runs: 12,
      completed_tasks: 12,
      dependency_blocked_tasks: 1,
      draft_tasks: 1,
      failed_runs: 3,
      failed_tasks: 3,
      in_progress_tasks: 4,
      pending_tasks: 6,
      queued_runs: 2,
      ready_tasks: 4,
      running_runs: 3,
      runs_total: 92,
      starting_runs: 0,
      tasks_total: 28,
    },
    ...overrides,
  } as TaskDashboardView;
}

describe("TasksDashboardCards", () => {
  it("renders the four summary cards with totals, subtitles, and live badge", () => {
    render(<TasksDashboardCards dashboard={buildDashboard()} />);

    expect(screen.getByTestId("tasks-dashboard-card-in_progress-value")).toHaveTextContent("4");
    expect(screen.getByTestId("tasks-dashboard-card-in_progress-live")).toHaveTextContent("live");
    expect(screen.getByTestId("tasks-dashboard-card-blocked-value")).toHaveTextContent("2");
    expect(screen.getByTestId("tasks-dashboard-card-blocked-detail")).toHaveTextContent(
      /deps unresolved/i
    );
    expect(screen.getByTestId("tasks-dashboard-card-failed-value")).toHaveTextContent("3");
    expect(screen.getByTestId("tasks-dashboard-card-latency-value").textContent).toMatch(/\d/);
  });

  it("hides the live badge when no runs are active", () => {
    const dashboard = buildDashboard({
      active_runs: {
        claimed: 0,
        queued: 0,
        running: 0,
        starting: 0,
        total: 0,
        items: [],
      },
      cards: {
        blocked: {
          awaiting_approval: 0,
          awaiting_dependencies: 0,
          health_status: "ok",
          tasks: 0,
        },
        failed: {
          failed_runs: 0,
          forced_stops: 0,
          health_status: "ok",
          tasks: 0,
        },
        in_progress: {
          active_runs: 0,
          claimed_runs: 0,
          queued_runs: 0,
          running_runs: 0,
          starting_runs: 0,
          tasks: 0,
          health_status: "ok",
        },
        latency: {
          claim_latency_ms: { average_ms: 0, maximum_ms: 0, samples: 0 },
          start_latency_ms: { average_ms: 0, maximum_ms: 0, samples: 0 },
        },
      },
    });

    render(<TasksDashboardCards dashboard={dashboard} />);
    expect(screen.queryByTestId("tasks-dashboard-card-in_progress-live")).not.toBeInTheDocument();
    expect(screen.getByTestId("tasks-dashboard-card-blocked-detail")).toHaveTextContent(
      /no blockers/i
    );
    expect(screen.getByTestId("tasks-dashboard-card-failed-detail")).toHaveTextContent(
      /no failures/i
    );
  });
});
