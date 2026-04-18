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
});
