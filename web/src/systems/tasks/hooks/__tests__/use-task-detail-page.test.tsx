import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/systems/tasks/adapters/tasks-api", () => ({
  listTasks: vi.fn(),
  getTask: vi.fn(),
  listTaskRuns: vi.fn(),
  getTaskTimeline: vi.fn(),
  getTaskRun: vi.fn(),
  inspectTask: vi.fn().mockResolvedValue(null),
  inspectRun: vi.fn().mockResolvedValue(null),
  getTaskExecutionProfile: vi.fn().mockResolvedValue(null),
  listTaskReviews: vi.fn().mockResolvedValue([]),
  getTaskDashboard: vi.fn(),
  getTaskInbox: vi.fn(),
  recoverTask: vi.fn(),
  recoverTaskRun: vi.fn(),
  approveTask: vi.fn(),
  rejectTask: vi.fn(),
  retryTaskRun: vi.fn(),
  clearTaskBlock: vi.fn(),
}));

import {
  approveTask,
  getTask,
  getTaskExecutionProfile,
  getTaskTimeline,
  listTaskReviews,
  listTaskRuns,
  recoverTask,
  recoverTaskRun,
  rejectTask,
  retryTaskRun,
} from "@/systems/tasks/adapters/tasks-api";

import { useTaskDetailPage } from "../use-task-detail-page";

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
  vi.mocked(listTaskRuns).mockResolvedValue([{ id: "run_1", status: "running" }] as never);
  vi.mocked(getTaskExecutionProfile).mockResolvedValue(null as never);
  vi.mocked(listTaskReviews).mockResolvedValue([] as never);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useTaskDetailPage", () => {
  it("Should load detail, timeline, runs, profile, and reviews for a task", async () => {
    const { result } = renderHook(() => useTaskDetailPage("task_001"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.detail?.task.id).toBe("task_001");
      expect(result.current.timeline).toHaveLength(1);
      expect(result.current.runs).toHaveLength(1);
    });

    expect(getTaskExecutionProfile).toHaveBeenCalledWith("task_001", expect.anything());
    expect(listTaskReviews).toHaveBeenCalled();
    expect(result.current.activeRun?.id).toBe("run_active");
  });

  it("Should honor enable flags for optional live reads", async () => {
    const { result } = renderHook(
      () =>
        useTaskDetailPage("task_001", {
          enableTimeline: false,
          enableRuns: false,
        }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.detail?.task.id).toBe("task_001");
    });

    expect(getTaskTimeline).not.toHaveBeenCalled();
    expect(listTaskRuns).not.toHaveBeenCalled();
  });

  it("Should report a fatal error when no task id is supplied", () => {
    const { result } = renderHook(() => useTaskDetailPage(""), { wrapper: createWrapper() });

    expect(result.current.fatalError).toBeInstanceOf(Error);
    expect(getTask).not.toHaveBeenCalled();
  });

  it("Should advance the timeline cursor when handleTimelineLoadMore is called", () => {
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

  it("Should derive an isLive flag from the active run status", async () => {
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

  it("Should approve and reject through their runtime verbs", async () => {
    vi.mocked(approveTask).mockResolvedValue({ id: "task_001" } as never);
    vi.mocked(rejectTask).mockResolvedValue({ id: "task_001" } as never);

    const { result } = renderHook(() => useTaskDetailPage("task_001"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.detail?.task.id).toBe("task_001"));

    await act(async () => {
      await result.current.handleApproveTask();
    });
    expect(approveTask).toHaveBeenCalledWith("task_001");

    await act(async () => {
      await result.current.handleRejectTask();
    });
    expect(rejectTask).toHaveBeenCalledWith("task_001");
  });

  it("Should retry a failed run by run id", async () => {
    vi.mocked(retryTaskRun).mockResolvedValue({
      run: { id: "run_retry", status: "queued" },
    } as never);

    const { result } = renderHook(() => useTaskDetailPage("task_001"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.detail?.task.id).toBe("task_001"));

    await act(async () => {
      await result.current.handleRetryRun("run_failed");
    });

    expect(retryTaskRun).toHaveBeenCalledWith("run_failed", {});
  });

  it("Should recover the active needs_attention run by run id exactly once", async () => {
    vi.mocked(getTask).mockResolvedValue({
      task: {
        id: "task_001",
        title: "Review",
        status: "ready",
        scope: "workspace",
        max_attempts: 2,
      },
      summary: {
        id: "task_001",
        title: "Review",
        status: "ready",
        scope: "workspace",
        active_run: {
          id: "run_attention",
          attempt: 1,
          status: "needs_attention",
        },
      },
    } as never);
    vi.mocked(recoverTaskRun).mockResolvedValue({
      previous_run: { id: "run_attention", status: "canceled" },
      run: { id: "run_continuation", status: "queued" },
    } as never);

    const { result } = renderHook(() => useTaskDetailPage("task_001"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.activeRun?.id).toBe("run_attention"));
    await act(async () => {
      await result.current.handleRecoverTask();
    });

    expect(recoverTaskRun).toHaveBeenCalledTimes(1);
    expect(recoverTaskRun).toHaveBeenCalledWith("run_attention", {});
  });

  it("Should not recover through either mutation when the active run exhausted attempts", async () => {
    vi.mocked(getTask).mockResolvedValue({
      task: {
        id: "task_001",
        title: "Review",
        status: "ready",
        scope: "workspace",
        max_attempts: 1,
      },
      summary: {
        id: "task_001",
        title: "Review",
        status: "ready",
        scope: "workspace",
        active_run: {
          id: "run_exhausted",
          attempt: 1,
          status: "needs_attention",
        },
      },
    } as never);

    const { result } = renderHook(() => useTaskDetailPage("task_001"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.activeRun?.id).toBe("run_exhausted"));
    await act(async () => {
      await result.current.handleRecoverTask();
    });

    expect(recoverTaskRun).not.toHaveBeenCalled();
    expect(recoverTask).not.toHaveBeenCalled();
  });
});
