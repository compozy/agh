import type { TaskChildSummary, TaskDetailView, TaskListItem, TaskRun } from "@/systems/tasks";

export function buildTaskFixture(overrides: Partial<TaskListItem> = {}): TaskListItem {
  return {
    id: "task_001",
    identifier: "TASK-1",
    title: "Summarize review feedback",
    status: "in_progress",
    scope: "workspace",
    origin: { kind: "web", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-17T10:00:00Z",
    last_activity_at: "2026-04-17T10:00:00Z",
    created_by: { kind: "human", ref: "pedro@" },
    owner: { kind: "agent_session", ref: "Coder" },
    priority: "high",
    child_count: 2,
    dependency_count: 1,
    active_run: {
      id: "run_001",
      task_id: "task_001",
      attempt: 2,
      max_attempts: 3,
      status: "running",
      queued_at: "2026-04-11T09:00:00Z",
    },
    ...overrides,
  } as TaskListItem;
}

export const TASK_FIXTURES: TaskListItem[] = [
  buildTaskFixture({ id: "task_001", identifier: "TASK-1", title: "Refactor event mapper" }),
  buildTaskFixture({
    id: "task_002",
    identifier: "TASK-2",
    title: "Add NATS retry backoff",
    status: "pending",
    active_run: null,
  }),
  buildTaskFixture({
    id: "task_003",
    identifier: "TASK-3",
    title: "Streaming buffer leak",
    status: "failed",
    active_run: {
      id: "run_003",
      task_id: "task_003",
      attempt: 3,
      max_attempts: 3,
      status: "failed",
      queued_at: "2026-04-17T08:00:00Z",
      error: "rate-limited by upstream",
    },
  }),
  buildTaskFixture({
    id: "task_004",
    identifier: "TASK-4",
    title: "Bridge health telemetry",
    status: "completed",
    active_run: null,
    child_count: 0,
    dependency_count: 0,
  }),
  buildTaskFixture({
    id: "task_005",
    identifier: "TASK-5",
    title: "Rewrite permission prompt",
    status: "blocked",
    active_run: null,
  }),
];

export function buildDetailFixture(overrides: Partial<TaskDetailView> = {}): TaskDetailView {
  const base = buildTaskFixture();
  const task = {
    ...base,
    description: "Pull CodeRabbit review on PR 341 and post a summary.",
  } as TaskDetailView["task"];
  return {
    task,
    summary: task as unknown as TaskDetailView["summary"],
    children: [
      {
        id: "child_001",
        identifier: "TASK-43",
        status: "ready",
        scope: "workspace",
        title: "Write migration",
        priority: "medium",
        owner: { kind: "agent_session", ref: "Coder" },
        last_activity_at: "2026-04-17T09:30:00Z",
      } as TaskChildSummary,
    ],
    dependency_references: [],
    runs: [
      {
        id: "run_001",
        task_id: "task_001",
        attempt: 1,
        status: "completed",
        queued_at: "2026-04-17T09:00:00Z",
        started_at: "2026-04-17T09:00:10Z",
        ended_at: "2026-04-17T09:03:00Z",
      } as TaskRun,
    ],
    ...overrides,
  } as TaskDetailView;
}
