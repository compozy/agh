import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";

import { TasksListSurface } from "../tasks-list-surface";
import type { TaskListItem } from "../../types";

function buildTask(overrides: Partial<TaskListItem> = {}): TaskListItem {
  return {
    id: "task_001",
    title: "Generate API client",
    identifier: "TASK-1",
    status: "ready",
    scope: "workspace",
    origin: { kind: "web", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    created_by: { kind: "human", ref: "op" },
    ...overrides,
  } as TaskListItem;
}

interface RenderOptions {
  tasks?: TaskListItem[];
  totalCount?: number;
  onSelectTask?: (taskId: string) => void;
  isLoading?: boolean;
  errorMessage?: string | null;
  statusFilter?: TaskListItem["status"] | null;
  workspaceName?: string | null;
  listUpdatedAt?: number;
}

function renderSurface(options: RenderOptions = {}) {
  return render(
    <UIProvider reducedMotion="always">
      <TasksListSurface
        errorMessage={options.errorMessage ?? null}
        isLoading={options.isLoading}
        listUpdatedAt={options.listUpdatedAt}
        onOwnerChange={() => {}}
        onPriorityChange={() => {}}
        onScopeChange={() => {}}
        onSelectTask={options.onSelectTask ?? (() => {})}
        onSortChange={() => {}}
        onStatusChange={() => {}}
        ownerFilter={null}
        ownerOptions={[]}
        priorityFilter={null}
        scopeFilter="all"
        sortBy="recent"
        statusFilter={options.statusFilter ?? null}
        tasks={options.tasks ?? []}
        totalCount={options.totalCount ?? 0}
        workspaceName={options.workspaceName ?? "agh-runtime"}
      />
    </UIProvider>
  );
}

describe("TasksListSurface", () => {
  it("Should partition tasks into the canonical status group sections", () => {
    const tasks = [
      buildTask({ id: "a", title: "Active task", status: "in_progress" }),
      buildTask({ id: "b", title: "Blocked task", status: "blocked" }),
      buildTask({ id: "c", title: "Queued task", status: "ready" }),
      buildTask({ id: "d", title: "Done task", status: "completed" }),
      buildTask({ id: "e", title: "Failed task", status: "failed" }),
    ];

    renderSurface({ tasks, totalCount: tasks.length });

    expect(screen.getByTestId("task-group-active-label")).toHaveTextContent(/active/i);
    expect(screen.getByTestId("task-group-active-count")).toHaveTextContent("1");
    expect(screen.getByTestId("task-group-blocked")).toBeInTheDocument();
    expect(screen.getByTestId("task-group-queued")).toBeInTheDocument();
    expect(screen.getByTestId("task-group-done")).toBeInTheDocument();
    expect(screen.getByTestId("task-group-failed")).toBeInTheDocument();
  });

  it("Should forward row selection to onSelectTask with the task id", () => {
    const onSelectTask = vi.fn();
    renderSurface({
      tasks: [buildTask({ id: "task_777" })],
      totalCount: 1,
      onSelectTask,
    });

    fireEvent.click(screen.getByTestId("task-card-task_777"));
    expect(onSelectTask).toHaveBeenCalledWith("task_777");
  });

  it("Should render the empty state when the list is empty", () => {
    renderSurface({ tasks: [], totalCount: 0 });
    expect(screen.getByTestId("tasks-list-surface-empty")).toBeInTheDocument();
  });

  it("Should render the loading skeleton when isLoading and no tasks", () => {
    renderSurface({ tasks: [], totalCount: 0, isLoading: true });
    expect(screen.getByTestId("tasks-list-surface-loading")).toBeInTheDocument();
  });

  it("Should render the error state when errorMessage is set and no tasks", () => {
    renderSurface({ tasks: [], totalCount: 0, errorMessage: "Unable to reach the daemon" });
    expect(screen.getByTestId("tasks-list-surface-error")).toBeInTheDocument();
    expect(screen.getByText(/unable to reach the daemon/i)).toBeInTheDocument();
  });

  it("Should render the page header with title, count, and workspace meta", () => {
    renderSurface({
      tasks: [buildTask({ id: "a", status: "ready" })],
      totalCount: 4,
      workspaceName: "agh-runtime",
      listUpdatedAt: Date.now() - 90_000,
    });

    expect(screen.getByTestId("tasks-list-page-title")).toHaveTextContent("Tasks");
    expect(screen.getByTestId("tasks-list-page-count")).toHaveTextContent("1 of 4");
    expect(screen.getByTestId("tasks-list-page-workspace")).toHaveTextContent("agh-runtime");
    expect(screen.getByTestId("tasks-list-page-synced")).toHaveTextContent(/synced/i);
  });
});
