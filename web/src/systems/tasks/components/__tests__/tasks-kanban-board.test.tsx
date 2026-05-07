import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TasksKanbanBoard } from "../tasks-kanban-board";
import { groupTasksForKanban } from "../../lib/task-grouping";
import type { TaskListItem } from "../../types";

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
  it("renders exactly four canonical columns labeled Pending, Running, Done, Failed", () => {
    render(
      <TasksKanbanBoard
        columns={groupTasksForKanban([])}
        onSelectTask={vi.fn()}
        selectedTaskId={null}
      />
    );

    expect(screen.getByTestId("tasks-kanban-board")).toBeInTheDocument();
    const columns = screen.getAllByRole("listitem");
    expect(columns).toHaveLength(4);
    expect(screen.getByTestId("tasks-kanban-column-pending")).toHaveTextContent(/Pending/);
    expect(screen.getByTestId("tasks-kanban-column-running")).toHaveTextContent(/Running/);
    expect(screen.getByTestId("tasks-kanban-column-done")).toHaveTextContent(/Done/);
    expect(screen.getByTestId("tasks-kanban-column-failed")).toHaveTextContent(/Failed/);
  });

  it("routes a running task into the Running column and leaves the other three empty", () => {
    const tasks = [buildTask({ id: "live", status: "in_progress" })];

    render(
      <TasksKanbanBoard
        columns={groupTasksForKanban(tasks)}
        onSelectTask={vi.fn()}
        selectedTaskId={null}
      />
    );

    expect(screen.getByTestId("tasks-kanban-card-live")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-kanban-column-running")).toContainElement(
      screen.getByTestId("tasks-kanban-card-live")
    );
    expect(screen.getByTestId("tasks-kanban-column-empty-pending")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-kanban-column-empty-done")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-kanban-column-empty-failed")).toBeInTheDocument();
  });

  it("collapses draft, pending, ready, and blocked tasks into the Pending column", () => {
    const tasks = [
      buildTask({ id: "d", status: "draft" }),
      buildTask({ id: "p", status: "pending" }),
      buildTask({ id: "r", status: "ready" }),
      buildTask({ id: "b", status: "blocked" }),
    ];

    render(
      <TasksKanbanBoard
        columns={groupTasksForKanban(tasks)}
        onSelectTask={vi.fn()}
        selectedTaskId={null}
      />
    );

    const pendingColumn = screen.getByTestId("tasks-kanban-column-pending");
    for (const id of ["d", "p", "r", "b"]) {
      expect(pendingColumn).toContainElement(screen.getByTestId(`tasks-kanban-card-${id}`));
    }
  });

  it("emits selection events when a card is clicked", () => {
    const onSelectTask = vi.fn();
    const tasks = [buildTask({ id: "a", status: "ready" })];

    render(
      <TasksKanbanBoard
        columns={groupTasksForKanban(tasks)}
        onSelectTask={onSelectTask}
        selectedTaskId={null}
      />
    );

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

    fireEvent.click(screen.getByTestId("tasks-kanban-column-add-pending"));
    expect(onCreateInColumn).toHaveBeenCalledWith("pending");
  });

  it("renders live indicators for in-progress runs and a retry affordance for failed cards", () => {
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
