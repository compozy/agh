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

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    activeWorkspace: { id: "ws_alpha", name: "Alpha" },
    activeWorkspaceId: "ws_alpha",
  }),
}));

import { getTaskDashboard, getTaskInbox, listTasks } from "@/systems/tasks/adapters/tasks-api";

import { useTasksPage } from "./use-tasks-page";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

const taskFixture = {
  id: "task_001",
  title: "Review PR",
  identifier: "TASK-1",
  status: "ready" as const,
  scope: "workspace" as const,
  origin: { kind: "web" as const, ref: "op" },
  created_at: "2026-04-11T09:00:00Z",
  updated_at: "2026-04-11T09:00:00Z",
  created_by: { kind: "human" as const, ref: "op" },
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(listTasks).mockResolvedValue([
    taskFixture,
    { ...taskFixture, id: "task_002", title: "Fix bug", status: "failed" },
  ] as never);
  vi.mocked(getTaskDashboard).mockResolvedValue({ totals: { tasks_total: 2 } } as never);
  vi.mocked(getTaskInbox).mockResolvedValue({
    total: 0,
    archived_total: 0,
    unread_total: 0,
  } as never);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useTasksPage", () => {
  it("exposes list state, counts, and derived flags", async () => {
    const { result } = renderHook(() => useTasksPage(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.visibleTasks).toHaveLength(2);
    });

    expect(result.current.mode).toBe("list");
    expect(result.current.tasksCount).toBe(2);
    expect(result.current.effectiveSelectedTaskId).toBe("task_001");
    expect(result.current.statusCounts.ready).toBe(1);
    expect(result.current.statusCounts.failed).toBe(1);
    expect(result.current.activeWorkspaceName).toBe("Alpha");
  });

  it("only fetches list reads when the list/kanban tab is active", async () => {
    const { result } = renderHook(() => useTasksPage({ initialMode: "dashboard" }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(getTaskDashboard).toHaveBeenCalled();
    });

    expect(result.current.mode).toBe("dashboard");
    expect(listTasks).not.toHaveBeenCalled();
    expect(getTaskInbox).not.toHaveBeenCalled();
  });

  it("swaps to inbox reads when the inbox tab is active", async () => {
    const { result } = renderHook(() => useTasksPage(), { wrapper: createWrapper() });

    act(() => {
      result.current.handleModeChange("inbox");
    });

    await waitFor(() => {
      expect(getTaskInbox).toHaveBeenCalled();
    });

    expect(result.current.mode).toBe("inbox");
  });

  it("updates scope and search params without losing active workspace id", async () => {
    const { result } = renderHook(() => useTasksPage(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.visibleTasks).toHaveLength(2);
    });

    act(() => {
      result.current.handleScopeChange("workspace");
      result.current.setSearchQuery("Fix");
    });

    await waitFor(() => {
      expect(result.current.scopeFilter).toBe("workspace");
    });

    await waitFor(() => {
      expect(result.current.visibleTasks.map(task => task.id)).toEqual(["task_002"]);
    });

    expect(result.current.effectiveSelectedTaskId).toBe("task_002");
    expect(listTasks).toHaveBeenCalledWith(
      expect.objectContaining({ scope: "workspace", workspace: "ws_alpha" }),
      expect.any(AbortSignal)
    );
  });

  it("exposes create modal open/close handlers", () => {
    const { result } = renderHook(() => useTasksPage(), { wrapper: createWrapper() });

    expect(result.current.isCreateModalOpen).toBe(false);

    act(() => {
      result.current.handleOpenCreateModal();
    });

    expect(result.current.isCreateModalOpen).toBe(true);

    act(() => {
      result.current.handleCloseCreateModal();
    });

    expect(result.current.isCreateModalOpen).toBe(false);
  });
});
