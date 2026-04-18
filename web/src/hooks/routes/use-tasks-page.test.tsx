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
  createTask: vi.fn(),
  updateTask: vi.fn(),
  publishTask: vi.fn(),
  cancelTask: vi.fn(),
  approveTask: vi.fn(),
  rejectTask: vi.fn(),
  createChildTask: vi.fn(),
  addTaskDependency: vi.fn(),
  removeTaskDependency: vi.fn(),
  enqueueTaskRun: vi.fn(),
  attachTaskRunSession: vi.fn(),
  cancelTaskRun: vi.fn(),
  claimTaskRun: vi.fn(),
  startTaskRun: vi.fn(),
  completeTaskRun: vi.fn(),
  failTaskRun: vi.fn(),
  markTaskRead: vi.fn(),
  archiveTask: vi.fn(),
  dismissTask: vi.fn(),
}));

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    activeWorkspace: { id: "ws_alpha", name: "Alpha" },
    activeWorkspaceId: "ws_alpha",
  }),
}));

import {
  createTask,
  enqueueTaskRun,
  getTaskDashboard,
  getTaskInbox,
  listTasks,
  publishTask,
} from "@/systems/tasks/adapters/tasks-api";

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
    {
      ...taskFixture,
      id: "task_003",
      title: "Draft proposal",
      status: "draft",
      draft: true,
    },
  ] as never);
  vi.mocked(getTaskDashboard).mockResolvedValue({ totals: { tasks_total: 3 } } as never);
  vi.mocked(getTaskInbox).mockResolvedValue({
    total: 0,
    archived_total: 0,
    unread_total: 0,
  } as never);
  vi.mocked(createTask).mockResolvedValue({ id: "task_999", title: "Generated" } as never);
  vi.mocked(publishTask).mockResolvedValue({ id: "task_003", title: "Draft" } as never);
  vi.mocked(enqueueTaskRun).mockResolvedValue({ id: "run_001" } as never);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useTasksPage", () => {
  it("exposes list state, counts, draft tasks, and derived flags", async () => {
    const { result } = renderHook(() => useTasksPage(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.visibleTasks).toHaveLength(3);
    });

    expect(result.current.mode).toBe("list");
    expect(result.current.tasksCount).toBe(3);
    expect(result.current.effectiveSelectedTaskId).toBe("task_001");
    expect(result.current.statusCounts.ready).toBe(1);
    expect(result.current.statusCounts.failed).toBe(1);
    expect(result.current.statusCounts.draft).toBe(1);
    expect(result.current.draftTasks.map(task => task.id)).toEqual(["task_003"]);
    expect(result.current.kanbanColumns.find(c => c.column.id === "pending")?.tasks).toHaveLength(
      1
    );
    expect(result.current.activeWorkspaceName).toBe("Alpha");
    expect(result.current.isEmpty).toBe(false);
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
      expect(result.current.visibleTasks).toHaveLength(3);
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

  it("opens the create modal with template defaults applied to the draft", () => {
    const { result } = renderHook(() => useTasksPage(), { wrapper: createWrapper() });

    expect(result.current.isCreateModalOpen).toBe(false);

    act(() => {
      result.current.handleOpenCreateModal("human_in_loop");
    });

    expect(result.current.isCreateModalOpen).toBe(true);
    expect(result.current.createTemplateId).toBe("human_in_loop");
    expect(result.current.createDraft.priority).toBe("high");
    expect(result.current.createDraft.approvalPolicy).toBe("manual");
    expect(result.current.createDraft.scope).toBe("workspace");

    act(() => {
      result.current.handleCloseCreateModal();
    });
    expect(result.current.isCreateModalOpen).toBe(false);
  });

  it("submits the create payload, enqueues the first run, and closes the modal", async () => {
    const { result } = renderHook(() => useTasksPage(), { wrapper: createWrapper() });

    act(() => {
      result.current.handleOpenCreateModal("one_shot");
    });

    act(() => {
      result.current.setCreateDraft(current => ({ ...current, title: "New thing" }));
    });

    await act(async () => {
      await result.current.submitCreateTask(result.current.createDraft, false);
    });

    expect(createTask).toHaveBeenCalledTimes(1);
    expect(createTask).toHaveBeenCalledWith(
      expect.objectContaining({
        title: "New thing",
        scope: "workspace",
        priority: "medium",
        max_attempts: 1,
        draft: false,
      })
    );
    expect(enqueueTaskRun).toHaveBeenCalledWith("task_999", {});
    expect(result.current.isCreateModalOpen).toBe(false);
  });

  it("save-draft submissions never enqueue a run", async () => {
    const { result } = renderHook(() => useTasksPage(), { wrapper: createWrapper() });
    act(() => {
      result.current.handleOpenCreateModal("one_shot");
    });
    act(() => {
      result.current.setCreateDraft(current => ({ ...current, title: "Drafted" }));
    });

    await act(async () => {
      await result.current.submitCreateTask(result.current.createDraft, true);
    });

    expect(createTask).toHaveBeenCalledWith(
      expect.objectContaining({ title: "Drafted", draft: true })
    );
    expect(enqueueTaskRun).not.toHaveBeenCalled();
  });

  it("recurring template always saves as draft even when submit triggers create", async () => {
    const { result } = renderHook(() => useTasksPage(), { wrapper: createWrapper() });
    act(() => {
      result.current.handleOpenCreateModal("recurring");
    });
    act(() => {
      result.current.setCreateDraft(current => ({ ...current, title: "Recurring" }));
    });

    await act(async () => {
      await result.current.submitCreateTask(result.current.createDraft, false);
    });

    expect(createTask).toHaveBeenCalledWith(expect.objectContaining({ draft: true }));
    expect(enqueueTaskRun).not.toHaveBeenCalled();
  });

  it("publishTask delegates to the publish mutation", async () => {
    const { result } = renderHook(() => useTasksPage(), { wrapper: createWrapper() });
    await act(async () => {
      await result.current.handlePublishTask("task_003");
    });

    expect(publishTask).toHaveBeenCalledWith("task_003");
  });
});
