import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TasksKanbanBoard } from "./tasks-kanban-board";
import { groupTasksForKanban } from "../lib/task-grouping";
import type { TaskListItem } from "../types";

function buildTask(overrides: Partial<TaskListItem> = {}): TaskListItem {
  return {
    id: overrides.id ?? "task_001",
    title: overrides.title ?? "Generate API client",
    identifier: overrides.identifier ?? "TASK-1",
    status: overrides.status ?? "ready",
    scope: "workspace",
    origin: { kind: "web", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    created_by: { kind: "human", ref: "op" },
    ...overrides,
  } as TaskListItem;
}

describe("TasksKanbanBoard", () => {
  it("renders all canonical kanban columns and emits selection events", () => {
    const tasks = [
      buildTask({ id: "a", status: "ready" }),
      buildTask({ id: "b", status: "in_progress" }),
      buildTask({ id: "c", status: "blocked" }),
    ];
    const onSelectTask = vi.fn();
    const columns = groupTasksForKanban(tasks);

    render(
      <TasksKanbanBoard columns={columns} onSelectTask={onSelectTask} selectedTaskId={null} />
    );

    expect(screen.getByTestId("tasks-kanban-board")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-kanban-column-pending")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-kanban-column-ready")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-kanban-column-empty-pending")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-kanban-card-a")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("tasks-kanban-card-a"));
    expect(onSelectTask).toHaveBeenCalledWith("a");
  });

  it("invokes onCreateInColumn from a column add affordance", () => {
    const onCreateInColumn = vi.fn();
    render(
      <TasksKanbanBoard
        columns={groupTasksForKanban([])}
        onCreateInColumn={onCreateInColumn}
        onSelectTask={vi.fn()}
        selectedTaskId={null}
      />
    );

    fireEvent.click(screen.getByTestId("tasks-kanban-column-add-ready"));
    expect(onCreateInColumn).toHaveBeenCalledWith("ready");
  });

  it("renders live indicators for in-progress runs and retry affordance for failed cards", () => {
    const onRetryTask = vi.fn();
    const tasks = [
      buildTask({
        id: "live",
        status: "in_progress",
        active_run: {
          id: "run_live",
          task_id: "live",
          attempt: 1,
          max_attempts: 3,
          status: "running",
          queued_at: "2026-04-11T09:00:00Z",
        },
      }),
      buildTask({
        id: "fail",
        status: "failed",
        active_run: {
          id: "run_fail",
          task_id: "fail",
          attempt: 3,
          max_attempts: 3,
          status: "failed",
          queued_at: "2026-04-11T09:00:00Z",
          error: "boom",
        },
      }),
    ];

    render(
      <TasksKanbanBoard
        columns={groupTasksForKanban(tasks)}
        onRetryTask={onRetryTask}
        onSelectTask={vi.fn()}
        selectedTaskId={null}
      />
    );

    expect(screen.getByTestId("tasks-kanban-card-live-live")).toHaveTextContent(/LIVE/);
    expect(screen.getByTestId("tasks-kanban-card-error-fail")).toHaveTextContent("boom");

    fireEvent.click(screen.getByTestId("tasks-kanban-card-retry-fail"));
    expect(onRetryTask).toHaveBeenCalledWith("fail");
  });

  it("renders loading and error states without crashing", () => {
    const { rerender } = render(
      <TasksKanbanBoard
        columns={groupTasksForKanban([])}
        isLoading
        onSelectTask={vi.fn()}
        selectedTaskId={null}
      />
    );
    expect(screen.getByTestId("tasks-kanban-loading")).toBeInTheDocument();

    rerender(
      <TasksKanbanBoard
        columns={groupTasksForKanban([])}
        errorMessage="oops"
        onSelectTask={vi.fn()}
        selectedTaskId={null}
      />
    );
    expect(screen.getByTestId("tasks-kanban-error")).toHaveTextContent("oops");
  });
});
