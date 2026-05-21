import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  useTask,
  useTaskDashboard,
  useTaskInbox,
  useTaskRunDetail,
  useTaskRuns,
  useTaskTimeline,
  useTaskTree,
  useTasks,
} from "@/systems/tasks";

vi.mock("@/systems/tasks/adapters/tasks-api", () => ({
  listTasks: vi.fn(),
  getTask: vi.fn(),
  listTaskRuns: vi.fn(),
  getTaskTimeline: vi.fn(),
  getTaskTree: vi.fn(),
  getTaskRun: vi.fn(),
  inspectTask: vi.fn().mockResolvedValue(null),
  inspectRun: vi.fn().mockResolvedValue(null),
  getTaskDashboard: vi.fn(),
  getTaskInbox: vi.fn(),
}));

import {
  getTask,
  getTaskDashboard,
  getTaskInbox,
  getTaskRun,
  getTaskTimeline,
  getTaskTree,
  listTaskRuns,
  listTasks,
} from "@/systems/tasks/adapters/tasks-api";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

const taskFixture = { id: "task_001", title: "Review", status: "ready" };
const runFixture = { id: "run_001", task_id: "task_001", status: "running" };
const timelineFixture = { event_id: "evt_001", sequence: 1 };
const treeFixture = { root: { depth: 0, task: { id: "task_001" } } };
const runDetailFixture = { run: runFixture, task: { id: "task_001" }, summary: {} };
const dashboardFixture = { totals: { tasks_total: 0 } };
const inboxFixture = { total: 0, archived_total: 0, unread_total: 0 };

describe("tasks read hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads tasks list with filters", async () => {
    vi.mocked(listTasks).mockResolvedValue([taskFixture] as never);

    const { result } = renderHook(() => useTasks({ scope: "workspace", workspace: "ws_alpha" }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toHaveLength(1);
    });

    expect(listTasks).toHaveBeenCalledWith(
      { scope: "workspace", workspace: "ws_alpha" },
      expect.any(AbortSignal)
    );
  });

  it("respects explicit disable flag for tasks list", () => {
    renderHook(() => useTasks({}, { enabled: false }), { wrapper: createWrapper() });
    expect(listTasks).not.toHaveBeenCalled();
  });

  it("loads task detail and guards empty ids", async () => {
    vi.mocked(getTask).mockResolvedValue({ task: taskFixture, summary: {} } as never);

    const { result } = renderHook(() => useTask("task_001"), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.data?.task.id).toBe("task_001");
    });

    renderHook(() => useTask(""), { wrapper: createWrapper() });
    expect(getTask).toHaveBeenCalledTimes(1);
  });

  it("loads task runs with filters and respects enabled flag", async () => {
    vi.mocked(listTaskRuns).mockResolvedValue([runFixture] as never);

    const { result } = renderHook(() => useTaskRuns("task_001", { status: "running", limit: 5 }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toHaveLength(1);
    });

    expect(listTaskRuns).toHaveBeenCalledWith(
      "task_001",
      { status: "running", limit: 5 },
      expect.any(AbortSignal)
    );

    renderHook(() => useTaskRuns("task_001", {}, { enabled: false }), {
      wrapper: createWrapper(),
    });
    expect(listTaskRuns).toHaveBeenCalledTimes(1);
  });

  it("loads live reads (timeline, tree, run detail)", async () => {
    vi.mocked(getTaskTimeline).mockResolvedValue([timelineFixture] as never);
    vi.mocked(getTaskTree).mockResolvedValue(treeFixture as never);
    vi.mocked(getTaskRun).mockResolvedValue(runDetailFixture as never);

    const timeline = renderHook(() => useTaskTimeline("task_001", { limit: 20 }), {
      wrapper: createWrapper(),
    });
    const tree = renderHook(() => useTaskTree("task_001"), { wrapper: createWrapper() });
    const runDetail = renderHook(() => useTaskRunDetail("run_001"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(timeline.result.current.data).toHaveLength(1);
      expect(tree.result.current.data?.root.task.id).toBe("task_001");
      expect(runDetail.result.current.data?.run.id).toBe("run_001");
    });

    expect(getTaskTimeline).toHaveBeenCalledWith(
      "task_001",
      { limit: 20 },
      expect.any(AbortSignal)
    );
    expect(getTaskTree).toHaveBeenCalledWith("task_001", expect.any(AbortSignal));
    expect(getTaskRun).toHaveBeenCalledWith("run_001", expect.any(AbortSignal));
  });

  it("loads dashboard and inbox aggregates", async () => {
    vi.mocked(getTaskDashboard).mockResolvedValue(dashboardFixture as never);
    vi.mocked(getTaskInbox).mockResolvedValue(inboxFixture as never);

    const dashboard = renderHook(() => useTaskDashboard({ scope: "workspace" }), {
      wrapper: createWrapper(),
    });
    const inbox = renderHook(() => useTaskInbox({ lane: "approvals" }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(dashboard.result.current.data?.totals.tasks_total).toBe(0);
      expect(inbox.result.current.data?.total).toBe(0);
    });

    expect(getTaskDashboard).toHaveBeenCalledWith({ scope: "workspace" }, expect.any(AbortSignal));
    expect(getTaskInbox).toHaveBeenCalledWith({ lane: "approvals" }, expect.any(AbortSignal));
  });
});
