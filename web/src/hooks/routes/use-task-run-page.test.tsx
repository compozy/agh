import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
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

import { getTask, getTaskRun } from "@/systems/tasks/adapters/tasks-api";

import { useTaskRunPage } from "./use-task-run-page";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

const runDetailFixture = {
  run: { id: "run_001", task_id: "task_001", status: "running" },
  task: { id: "task_001", title: "Review", status: "ready", scope: "workspace" },
  summary: { last_activity_at: "2026-04-11T09:00:00Z" },
  session: {
    session_id: "sess_a",
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
  },
};

const taskDetailFixture = {
  task: { id: "task_001", title: "Review", status: "ready", scope: "workspace" },
  summary: { id: "task_001", title: "Review", status: "ready", scope: "workspace" },
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(getTaskRun).mockResolvedValue(runDetailFixture as never);
  vi.mocked(getTask).mockResolvedValue(taskDetailFixture as never);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useTaskRunPage", () => {
  it("loads run detail and task detail together", async () => {
    const { result } = renderHook(() => useTaskRunPage("task_001", "run_001"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.run?.run.id).toBe("run_001");
      expect(result.current.task?.task.id).toBe("task_001");
    });

    expect(result.current.session?.session_id).toBe("sess_a");
    expect(result.current.summary?.last_activity_at).toBe("2026-04-11T09:00:00Z");
  });

  it("reports fatal error when ids are missing", () => {
    const { result } = renderHook(() => useTaskRunPage("", ""), { wrapper: createWrapper() });

    expect(result.current.fatalError).toBeInstanceOf(Error);
    expect(getTaskRun).not.toHaveBeenCalled();
    expect(getTask).not.toHaveBeenCalled();
  });

  it("skips task detail query when disabled", async () => {
    const { result } = renderHook(
      () => useTaskRunPage("task_001", "run_001", { enableTaskDetail: false }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.run?.run.id).toBe("run_001");
    });

    expect(getTask).not.toHaveBeenCalled();
  });
});
