import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/systems/tasks/adapters/tasks-api", () => ({
  listTasks: vi.fn(),
  getTask: vi.fn(),
  listTaskRuns: vi.fn(),
  getTaskTimeline: vi.fn(),
  getTaskTree: vi.fn(),
  getTaskRun: vi.fn(),
  getTaskDashboard: vi.fn(),
  getTaskInbox: vi.fn(),
}));

import {
  getTask,
  getTaskTimeline,
  getTaskTree,
  listTaskRuns,
} from "@/systems/tasks/adapters/tasks-api";

import { useTaskDetailPage } from "./use-task-detail-page";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

const detailFixture = {
  task: { id: "task_001", title: "Review", status: "ready", scope: "workspace" },
  summary: {
    id: "task_001",
    title: "Review",
    status: "ready",
    scope: "workspace",
    active_run: { id: "run_active" },
  },
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(getTask).mockResolvedValue(detailFixture as never);
  vi.mocked(getTaskTimeline).mockResolvedValue([{ event_id: "evt_1", sequence: 1 }] as never);
  vi.mocked(getTaskTree).mockResolvedValue({
    root: { depth: 0, task: { id: "task_001" } },
  } as never);
  vi.mocked(listTaskRuns).mockResolvedValue([{ id: "run_1", status: "running" }] as never);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useTaskDetailPage", () => {
  it("loads detail, timeline, tree, and runs for a task", async () => {
    const { result } = renderHook(() => useTaskDetailPage("task_001"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.detail?.task.id).toBe("task_001");
      expect(result.current.timeline).toHaveLength(1);
      expect(result.current.tree?.root.task.id).toBe("task_001");
      expect(result.current.runs).toHaveLength(1);
    });

    expect(result.current.panel).toBe("overview");
    expect(result.current.activeRun?.id).toBe("run_active");
  });

  it("supports panel switching via handlePanelChange", () => {
    const { result } = renderHook(() => useTaskDetailPage("task_001"), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.handlePanelChange("timeline");
    });

    expect(result.current.panel).toBe("timeline");
  });

  it("honors enable flags for optional live reads", async () => {
    const { result } = renderHook(
      () =>
        useTaskDetailPage("task_001", {
          enableTimeline: false,
          enableTree: false,
          enableRuns: false,
        }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.detail?.task.id).toBe("task_001");
    });

    expect(getTaskTimeline).not.toHaveBeenCalled();
    expect(getTaskTree).not.toHaveBeenCalled();
    expect(listTaskRuns).not.toHaveBeenCalled();
  });

  it("reports a fatal error when no task id is supplied", () => {
    const { result } = renderHook(() => useTaskDetailPage(""), { wrapper: createWrapper() });

    expect(result.current.fatalError).toBeInstanceOf(Error);
    expect(getTask).not.toHaveBeenCalled();
  });

  it("advances the timeline cursor when handleTimelineLoadMore is called", () => {
    const { result } = renderHook(
      () => useTaskDetailPage("task_001", { initialTimelineLimit: 25 }),
      { wrapper: createWrapper() }
    );

    expect(result.current.timelineLimit).toBe(25);

    act(() => {
      result.current.handleTimelineLoadMore();
    });

    expect(result.current.timelineLimit).toBeGreaterThan(25);
  });

  it("derives an isLive flag from the active run status", async () => {
    vi.mocked(getTask).mockResolvedValue({
      task: { id: "task_001", title: "Review", status: "in_progress", scope: "workspace" },
      summary: {
        id: "task_001",
        title: "Review",
        status: "in_progress",
        scope: "workspace",
        active_run: { id: "run_active", status: "running" },
      },
    } as never);

    const { result } = renderHook(() => useTaskDetailPage("task_001"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLive).toBe(true);
    });
  });

  it("derives multi-agent view counts and live states from the tree read", async () => {
    vi.mocked(getTaskTree).mockResolvedValue({
      root: {
        depth: 0,
        task: {
          id: "task_001",
          identifier: "TASK-38",
          status: "in_progress",
          scope: "workspace",
          title: "Epic",
          owner: { kind: "agent_session", ref: "Researcher" },
        },
        active_run: {
          id: "run_a",
          attempt: 1,
          max_attempts: 3,
          queued_at: "2026-04-17T10:00:00Z",
          status: "running",
          task_id: "task_001",
          session_id: "sess_a",
        },
        child_count: 2,
        last_activity_at: "2026-04-17T10:01:00Z",
      },
      descendants: [
        {
          depth: 1,
          parent_task_id: "task_001",
          task: {
            id: "task_002",
            identifier: "TASK-39",
            status: "in_progress",
            scope: "workspace",
            title: "Child 1",
            owner: { kind: "agent_session", ref: "Coder" },
          },
          active_run: {
            id: "run_b",
            attempt: 1,
            max_attempts: 2,
            queued_at: "2026-04-17T10:00:10Z",
            status: "running",
            task_id: "task_002",
            session_id: "sess_b",
          },
          child_count: 0,
          last_activity_at: "2026-04-17T10:01:00Z",
        },
        {
          depth: 1,
          parent_task_id: "task_001",
          task: {
            id: "task_003",
            identifier: "TASK-40",
            status: "ready",
            scope: "workspace",
            title: "Child 2",
            owner: { kind: "agent_session", ref: "Writer" },
          },
          active_run: null,
          child_count: 0,
          last_activity_at: "2026-04-17T10:00:30Z",
        },
      ],
    } as never);

    const { result } = renderHook(() => useTaskDetailPage("task_001"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.multiAgent.state).toBe("ready");
    });

    const { multiAgent } = result.current;
    expect(multiAgent.liveCount).toBeGreaterThanOrEqual(2);
    expect(multiAgent.descendantCount).toBe(2);
    expect(multiAgent.activeDescendants).toBe(1);
    expect(multiAgent.nodes).toHaveLength(3);
    expect(multiAgent.nodes[0].isRoot).toBe(true);
    expect(multiAgent.nodes[0].isPrimary).toBe(true);
    expect(multiAgent.nodes[1].isLive).toBe(true);
    expect(multiAgent.nodes[2].isLive).toBe(false);
  });

  it("reports a loading multi-agent state while the tree is resolving", () => {
    vi.mocked(getTaskTree).mockImplementation(() => new Promise(() => {}));

    const { result } = renderHook(() => useTaskDetailPage("task_001"), {
      wrapper: createWrapper(),
    });

    expect(result.current.multiAgent.state).toBe("loading");
    expect(result.current.multiAgent.nodes).toHaveLength(0);
  });

  it("reports a disconnected multi-agent state when the tree read fails", async () => {
    vi.mocked(getTaskTree).mockRejectedValue(new Error("stream disconnected"));

    const { result } = renderHook(() => useTaskDetailPage("task_001"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.multiAgent.state).toBe("disconnected");
    });
    expect(result.current.treeError?.message).toContain("disconnected");
  });

  it("reports no-descendants when the tree has no children and no live root", async () => {
    vi.mocked(getTaskTree).mockResolvedValue({
      root: {
        depth: 0,
        task: {
          id: "task_001",
          status: "completed",
          scope: "workspace",
          title: "Done",
        },
        active_run: null,
        child_count: 0,
        last_activity_at: "2026-04-17T10:00:00Z",
      },
      descendants: [],
    } as never);

    const { result } = renderHook(() => useTaskDetailPage("task_001"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.multiAgent.state).toBe("no-descendants");
    });
  });

  it("preserves multi-agent state when switching to the agents panel", async () => {
    const { result } = renderHook(() => useTaskDetailPage("task_001"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.tree?.root.task.id).toBe("task_001");
    });

    act(() => {
      result.current.handlePanelChange("agents");
    });

    expect(result.current.panel).toBe("agents");
    expect(result.current.multiAgent).toBeDefined();
  });
});
