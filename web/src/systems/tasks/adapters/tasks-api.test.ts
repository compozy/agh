import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { expectFetchRequest, mockJsonResponse } from "@/test/fetch-test-utils";

function mockJsonSequence(body: unknown, status = 200): void {
  vi.mocked(globalThis.fetch).mockImplementation(() =>
    Promise.resolve(
      new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
      })
    )
  );
}
import {
  TasksApiError,
  addTaskDependency,
  approveTask,
  archiveTask,
  attachTaskRunSession,
  cancelTask,
  cancelTaskRun,
  claimTaskRun,
  completeTaskRun,
  createChildTask,
  createTask,
  dismissTask,
  enqueueTaskRun,
  failTaskRun,
  getTask,
  getTaskDashboard,
  getTaskInbox,
  getTaskRun,
  getTaskTimeline,
  getTaskTree,
  listTaskRuns,
  listTasks,
  markTaskRead,
  publishTask,
  rejectTask,
  removeTaskDependency,
  startTaskRun,
  updateTask,
} from "@/systems/tasks/adapters/tasks-api";

const taskFixture = {
  id: "task_001",
  title: "Review changes",
  status: "ready" as const,
  scope: "workspace" as const,
  origin: { kind: "web" as const, ref: "op" },
  created_at: "2026-04-11T09:00:00Z",
  updated_at: "2026-04-11T09:00:00Z",
  created_by: { kind: "human" as const, ref: "op" },
};

const taskDetailFixture = {
  task: taskFixture,
  summary: {
    ...taskFixture,
    active_run: null,
  },
};

const runFixture = {
  id: "run_001",
  task_id: "task_001",
  attempt: 1,
  status: "running" as const,
  queued_at: "2026-04-11T09:00:00Z",
  origin: { kind: "web" as const, ref: "op" },
};

const runDetailFixture = {
  run: runFixture,
  task: {
    id: taskFixture.id,
    title: taskFixture.title,
    status: taskFixture.status,
    scope: taskFixture.scope,
  },
  summary: { last_activity_at: "2026-04-11T09:00:00Z" },
};

const timelineFixture = {
  actor: { kind: "human" as const, ref: "op" },
  event_id: "evt_001",
  event_type: "task.updated",
  origin: { kind: "web" as const, ref: "op" },
  sequence: 1,
  task: {
    id: taskFixture.id,
    title: taskFixture.title,
    status: taskFixture.status,
    scope: taskFixture.scope,
  },
  timestamp: "2026-04-11T09:00:00Z",
};

const treeFixture = {
  root: {
    depth: 0,
    last_activity_at: "2026-04-11T09:00:00Z",
    task: {
      id: taskFixture.id,
      title: taskFixture.title,
      status: taskFixture.status,
      scope: taskFixture.scope,
    },
  },
};

const dashboardFixture = {
  active_runs: { claimed: 0, queued: 0, running: 0, starting: 0, total: 0 },
  cards: {
    blocked: { awaiting_approval: 0, awaiting_dependencies: 0, health_status: "ok", tasks: 0 },
    failed: { failed_runs: 0, forced_stops: 0, health_status: "ok", tasks: 0 },
    in_progress: {
      active_runs: 0,
      claimed_runs: 0,
      health_status: "ok",
      queued_runs: 0,
      running_runs: 0,
      starting_runs: 0,
      tasks: 0,
    },
    latency: {
      claim_latency_ms: { average_ms: 0, maximum_ms: 0, samples: 0 },
      start_latency_ms: { average_ms: 0, maximum_ms: 0, samples: 0 },
    },
  },
  freshness: {
    age_ms: 0,
    has_live_work: false,
    latest_activity_at: "2026-04-11T09:00:00Z",
    observed_at: "2026-04-11T09:00:00Z",
    stale: false,
    stale_after_ms: 10000,
    status: "fresh",
  },
  health: { active_orphan_runs: 0, queue_backlog: false, status: "ok", stuck_runs: 0 },
  queue: {
    backlog_status: "idle",
    backlog_threshold_ms: 0,
    backlog_warning: false,
    oldest_queue_age_ms: 0,
    oldest_queued_at: "2026-04-11T09:00:00Z",
    total: 0,
  },
  totals: {
    active_runs: 0,
    awaiting_approval_tasks: 0,
    blocked_tasks: 0,
    canceled_runs: 0,
    canceled_tasks: 0,
    claimed_runs: 0,
    completed_runs: 0,
    completed_tasks: 0,
    dependency_blocked_tasks: 0,
    draft_tasks: 0,
    failed_runs: 0,
    failed_tasks: 0,
    in_progress_tasks: 0,
    pending_tasks: 0,
    queued_runs: 0,
    ready_tasks: 0,
    running_runs: 0,
    runs_total: 0,
    starting_runs: 0,
    tasks_total: 0,
  },
};

const inboxFixture = {
  archived_total: 0,
  total: 0,
  unread_total: 0,
};

const triageFixture = {
  actor: { kind: "human" as const, ref: "op" },
  archived: false,
  dismissed: false,
  read: true,
  task_id: taskFixture.id,
  updated_at: "2026-04-11T09:00:00Z",
};

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn());
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("listTasks", () => {
  it("calls GET /api/tasks with normalized filters", async () => {
    mockJsonResponse({ tasks: [taskFixture] });

    const result = await listTasks({
      scope: "workspace",
      workspace: "  ws_alpha  ",
      status: "ready",
      priority: "high",
      include_drafts: true,
      query: " review ",
      limit: 25,
    });

    expect(result).toEqual([taskFixture]);
    await expectFetchRequest({
      path: "/api/tasks?scope=workspace&workspace=ws_alpha&status=ready&priority=high&include_drafts=true&query=review&limit=25",
    });
  });

  it("omits empty string filters", async () => {
    mockJsonResponse({ tasks: [] });

    await listTasks({ workspace: "   ", owner_ref: "" });

    await expectFetchRequest({ path: "/api/tasks" });
  });

  it("throws TasksApiError on non-2xx", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(listTasks()).rejects.toThrow(TasksApiError);
    await expect(listTasks()).rejects.toThrow("Failed to fetch tasks: 500");
  });
});

describe("getTask", () => {
  it("fetches task detail by id", async () => {
    mockJsonResponse({ task: taskDetailFixture });

    const result = await getTask("task_001");

    expect(result).toEqual(taskDetailFixture);
    await expectFetchRequest({ path: "/api/tasks/task_001" });
  });

  it("throws not-found for 404", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(getTask("missing")).rejects.toThrow("Task not found: missing");
  });
});

describe("task mutations", () => {
  it("creates a task", async () => {
    mockJsonResponse({ task: taskFixture }, { status: 201 });

    const body = { title: "Review", scope: "workspace" as const };
    const result = await createTask(body);

    expect(result).toEqual(taskFixture);
    await expectFetchRequest({ body, method: "POST", path: "/api/tasks" });
  });

  it("updates a task", async () => {
    mockJsonResponse({ task: { ...taskFixture, title: "Updated" } });

    const result = await updateTask("task_001", { title: "Updated" });

    expect(result.title).toBe("Updated");
    await expectFetchRequest({
      body: { title: "Updated" },
      method: "PATCH",
      path: "/api/tasks/task_001",
    });
  });

  it("publishes a draft task", async () => {
    mockJsonResponse({ task: taskFixture });

    const result = await publishTask("task_001");

    expect(result).toEqual(taskFixture);
    await expectFetchRequest({ method: "POST", path: "/api/tasks/task_001/publish" });
  });

  it("cancels a task with default body", async () => {
    mockJsonResponse({ task: taskFixture });

    const result = await cancelTask("task_001");

    expect(result).toEqual(taskFixture);
    await expectFetchRequest({ body: {}, method: "POST", path: "/api/tasks/task_001/cancel" });
  });

  it("approves and rejects approval-gated tasks", async () => {
    mockJsonSequence({ task: taskFixture });

    await approveTask("task_001");
    await rejectTask("task_001");

    await expectFetchRequest({ method: "POST", path: "/api/tasks/task_001/approve" });
    await expectFetchRequest({
      callIndex: 1,
      method: "POST",
      path: "/api/tasks/task_001/reject",
    });
  });

  it("creates a child task", async () => {
    mockJsonResponse({ task: taskFixture }, { status: 201 });

    const body = { title: "Child", scope: "workspace" as const };
    await createChildTask("task_001", body);

    await expectFetchRequest({
      body,
      method: "POST",
      path: "/api/tasks/task_001/children",
    });
  });

  it("adds and removes task dependencies", async () => {
    mockJsonSequence({ task: taskDetailFixture });

    await addTaskDependency("task_001", { depends_on_task_id: "task_002" });
    await removeTaskDependency("task_001", "task_002");

    await expectFetchRequest({
      body: { depends_on_task_id: "task_002" },
      method: "POST",
      path: "/api/tasks/task_001/dependencies",
    });
    await expectFetchRequest({
      callIndex: 1,
      method: "DELETE",
      path: "/api/tasks/task_001/dependencies/task_002",
    });
  });
});

describe("task runs", () => {
  it("lists task runs with filters", async () => {
    mockJsonResponse({ runs: [runFixture] });

    const result = await listTaskRuns("task_001", {
      status: "running",
      session_id: " sess_a ",
      limit: 5,
    });

    expect(result).toEqual([runFixture]);
    await expectFetchRequest({
      path: "/api/tasks/task_001/runs?status=running&session_id=sess_a&limit=5",
    });
  });

  it("enqueues a task run", async () => {
    mockJsonResponse({ run: runFixture }, { status: 201 });

    await enqueueTaskRun("task_001", { idempotency_key: "idem_1" });

    await expectFetchRequest({
      body: { idempotency_key: "idem_1" },
      method: "POST",
      path: "/api/tasks/task_001/runs",
    });
  });

  it("fetches task timeline", async () => {
    mockJsonResponse({ timeline: [timelineFixture] });

    await getTaskTimeline("task_001", { after_sequence: 10, limit: 20 });

    await expectFetchRequest({
      path: "/api/tasks/task_001/timeline?after_sequence=10&limit=20",
    });
  });

  it("fetches task tree", async () => {
    mockJsonResponse({ tree: treeFixture });

    const result = await getTaskTree("task_001");

    expect(result).toEqual(treeFixture);
    await expectFetchRequest({ path: "/api/tasks/task_001/tree" });
  });

  it("fetches task-run detail", async () => {
    mockJsonResponse({ run: runDetailFixture });

    const result = await getTaskRun("run_001");

    expect(result).toEqual(runDetailFixture);
    await expectFetchRequest({ path: "/api/task-runs/run_001" });
  });

  it("runs lifecycle commands against /api/task-runs/{id}/*", async () => {
    mockJsonSequence({ run: runFixture });

    await claimTaskRun("run_001", { idempotency_key: "c" });
    await startTaskRun("run_001");
    await completeTaskRun("run_001", { result: { ok: true } });
    await failTaskRun("run_001", { error: "boom" });
    await cancelTaskRun("run_001");
    await attachTaskRunSession("run_001", { session_id: "sess_a" });

    await expectFetchRequest({
      body: { idempotency_key: "c" },
      method: "POST",
      path: "/api/task-runs/run_001/claim",
    });
    await expectFetchRequest({
      body: {},
      callIndex: 1,
      method: "POST",
      path: "/api/task-runs/run_001/start",
    });
    await expectFetchRequest({
      body: { result: { ok: true } },
      callIndex: 2,
      method: "POST",
      path: "/api/task-runs/run_001/complete",
    });
    await expectFetchRequest({
      body: { error: "boom" },
      callIndex: 3,
      method: "POST",
      path: "/api/task-runs/run_001/fail",
    });
    await expectFetchRequest({
      body: {},
      callIndex: 4,
      method: "POST",
      path: "/api/task-runs/run_001/cancel",
    });
    await expectFetchRequest({
      body: { session_id: "sess_a" },
      callIndex: 5,
      method: "POST",
      path: "/api/task-runs/run_001/attach-session",
    });
  });
});

describe("dashboard and inbox", () => {
  it("fetches dashboard payload with filter normalization", async () => {
    mockJsonResponse({ dashboard: dashboardFixture });

    await getTaskDashboard({ scope: "workspace", workspace: "  ws_a  " });

    await expectFetchRequest({
      path: "/api/observe/tasks/dashboard?scope=workspace&workspace=ws_a",
    });
  });

  it("fetches inbox payload with filter normalization", async () => {
    mockJsonResponse({ inbox: inboxFixture });

    await getTaskInbox({
      scope: "workspace",
      workspace: "ws_a",
      lane: "my_work",
      unread: true,
      limit: 10,
    });

    await expectFetchRequest({
      path: "/api/observe/tasks/inbox?scope=workspace&workspace=ws_a&lane=my_work&unread=true&limit=10",
    });
  });
});

describe("triage mutations", () => {
  it("marks a task read", async () => {
    mockJsonResponse({ triage: triageFixture });

    const result = await markTaskRead("task_001");

    expect(result).toEqual(triageFixture);
    await expectFetchRequest({
      method: "POST",
      path: "/api/tasks/task_001/triage/read",
    });
  });

  it("archives and dismisses tasks", async () => {
    mockJsonSequence({ triage: triageFixture });

    await archiveTask("task_001");
    await dismissTask("task_001");

    await expectFetchRequest({
      method: "POST",
      path: "/api/tasks/task_001/triage/archive",
    });
    await expectFetchRequest({
      callIndex: 1,
      method: "POST",
      path: "/api/tasks/task_001/triage/dismiss",
    });
  });
});

describe("TasksApiError", () => {
  it("stores status", () => {
    const err = new TasksApiError("boom", 422);

    expect(err.name).toBe("TasksApiError");
    expect(err.status).toBe(422);
    expect(err.message).toBe("boom");
  });
});
