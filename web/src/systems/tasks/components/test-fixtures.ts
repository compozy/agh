import type { TaskDashboardView, TaskInboxItem, TaskInboxView } from "../types";

export function buildDashboardFixture(
  overrides: Partial<TaskDashboardView> = {}
): TaskDashboardView {
  return {
    active_runs: {
      claimed: 0,
      queued: 0,
      running: 1,
      starting: 0,
      total: 1,
      items: [
        {
          age_ms: 45_000,
          attempt: 1,
          health_status: "ok",
          last_activity_at: "2026-04-17T10:00:00Z",
          max_attempts: 3,
          run_id: "run_101",
          run_status: "running",
          scope: "workspace",
          stuck: false,
          task_id: "task_101",
          task_identifier: "TASK-42",
          task_status: "in_progress",
          task_title: "Summarize review feedback",
        },
      ],
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
        active_runs: 1,
        claimed_runs: 0,
        queued_runs: 0,
        running_runs: 1,
        starting_runs: 0,
        tasks: 1,
        health_status: "ok",
      },
      latency: {
        claim_latency_ms: { average_ms: 1000, maximum_ms: 1800, samples: 4 },
        start_latency_ms: { average_ms: 500, maximum_ms: 1200, samples: 4 },
      },
    },
    freshness: {
      age_ms: 500,
      has_live_work: true,
      latest_activity_at: "2026-04-17T10:00:00Z",
      observed_at: "2026-04-17T10:00:01Z",
      stale: false,
      stale_after_ms: 60_000,
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
      backlog_threshold_ms: 60_000,
      backlog_warning: false,
      oldest_queue_age_ms: 0,
      oldest_queued_at: "2026-04-17T10:00:00Z",
      total: 0,
    },
    status_breakdown: [
      { count: 5, share_percent: 50, status: "completed" },
      { count: 3, share_percent: 30, status: "in_progress" },
      { count: 2, share_percent: 20, status: "blocked" },
    ],
    totals: {
      active_runs: 1,
      awaiting_approval_tasks: 0,
      blocked_tasks: 2,
      canceled_runs: 0,
      canceled_tasks: 0,
      claimed_runs: 0,
      completed_runs: 5,
      completed_tasks: 5,
      dependency_blocked_tasks: 0,
      draft_tasks: 0,
      failed_runs: 0,
      failed_tasks: 0,
      in_progress_tasks: 1,
      pending_tasks: 2,
      queued_runs: 0,
      ready_tasks: 0,
      running_runs: 1,
      runs_total: 6,
      starting_runs: 0,
      tasks_total: 10,
    },
    ...overrides,
  } as TaskDashboardView;
}

export function buildInboxItemFixture(overrides: Partial<TaskInboxItem> = {}): TaskInboxItem {
  return {
    lane: "my_work",
    latest_activity_at: "2026-04-17T10:00:00Z",
    task: {
      id: "task_001",
      identifier: "TASK-1",
      scope: "workspace",
      status: "ready",
      title: "Inbox item",
      owner: { kind: "agent_session", ref: "Coder" },
    },
    triage: {
      actor: { kind: "human", ref: "op" },
      archived: false,
      dismissed: false,
      read: false,
      task_id: "task_001",
      updated_at: "2026-04-17T10:00:00Z",
    },
    ...overrides,
  } as TaskInboxItem;
}

export function buildInboxFixture(overrides: Partial<TaskInboxView> = {}): TaskInboxView {
  return {
    archived_total: 0,
    total: 0,
    unread_total: 0,
    groups: [],
    ...overrides,
  } as TaskInboxView;
}
