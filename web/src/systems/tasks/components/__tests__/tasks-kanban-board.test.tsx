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
    owner: { kind: "agent_session", ref: "claude" },
    ...overrides,
  } as TaskListItem;
}

describe("TasksKanbanBoard", () => {
  it("Should render exactly four canonical columns labeled Pending, In progress, Blocked, Done", () => {
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
    expect(screen.getByTestId("tasks-kanban-column-in_progress")).toHaveTextContent(/In progress/);
    expect(screen.getByTestId("tasks-kanban-column-blocked")).toHaveTextContent(/Blocked/);
    expect(screen.getByTestId("tasks-kanban-column-done")).toHaveTextContent(/Done/);
  });

  it("Should route an in-progress task into the In progress column", () => {
    const tasks = [buildTask({ id: "live", status: "in_progress" })];

    render(
      <TasksKanbanBoard
        columns={groupTasksForKanban(tasks)}
        onSelectTask={vi.fn()}
        selectedTaskId={null}
      />
    );

    expect(screen.getByTestId("tasks-kanban-card-live")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-kanban-column-in_progress")).toContainElement(
      screen.getByTestId("tasks-kanban-card-live")
    );
    expect(screen.getByTestId("tasks-kanban-column-empty-pending")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-kanban-column-empty-blocked")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-kanban-column-empty-done")).toBeInTheDocument();
  });

  it("Should collapse draft, pending, and ready into Pending; blocked into Blocked", () => {
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
    for (const id of ["d", "p", "r"]) {
      expect(pendingColumn).toContainElement(screen.getByTestId(`tasks-kanban-card-${id}`));
    }
    expect(screen.getByTestId("tasks-kanban-column-blocked")).toContainElement(
      screen.getByTestId("tasks-kanban-card-b")
    );
  });

  it("Should collapse terminal statuses (completed, failed, canceled) into Done", () => {
    const tasks = [
      buildTask({ id: "c", status: "completed" }),
      buildTask({ id: "f", status: "failed" }),
      buildTask({ id: "x", status: "canceled" }),
    ];

    render(
      <TasksKanbanBoard
        columns={groupTasksForKanban(tasks)}
        onSelectTask={vi.fn()}
        selectedTaskId={null}
      />
    );

    const doneColumn = screen.getByTestId("tasks-kanban-column-done");
    for (const id of ["c", "f", "x"]) {
      expect(doneColumn).toContainElement(screen.getByTestId(`tasks-kanban-card-${id}`));
    }
  });

  it("Should emit selection events when a card is clicked", () => {
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

  it("Should invoke onCreateInColumn from a column add affordance", () => {
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

  it("Should render a retry affordance for failed cards and surface their error", () => {
    const onRetryTask = vi.fn();
    const tasks = [
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

    expect(screen.getByTestId("tasks-kanban-card-error-fail")).toHaveTextContent("boom");
    fireEvent.click(screen.getByTestId("tasks-kanban-card-retry-fail"));
    expect(onRetryTask).toHaveBeenCalledWith("fail");
  });

  it("Should render loading and error states without crashing", () => {
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

describe("TaskKanbanCard", () => {
  it("Should render the OwnerAvatar primitive (no plain text owner fallback alone)", () => {
    const tasks = [
      buildTask({
        id: "owned",
        owner: { kind: "agent_session", ref: "claude" },
      }),
    ];

    render(
      <TasksKanbanBoard
        columns={groupTasksForKanban(tasks)}
        onSelectTask={vi.fn()}
        selectedTaskId={null}
      />
    );

    expect(screen.getByTestId("tasks-kanban-card-avatar-owned")).toHaveAttribute(
      "data-slot",
      "owner-avatar"
    );
  });

  it("Should paint the card with an inset ring instead of a border class", () => {
    const tasks = [buildTask({ id: "ring" })];
    render(
      <TasksKanbanBoard
        columns={groupTasksForKanban(tasks)}
        onSelectTask={vi.fn()}
        selectedTaskId={null}
      />
    );

    const card = screen.getByTestId("tasks-kanban-card-ring");
    expect(card.className).toContain("shadow-focus-ring-inset-soft");
    expect(card.className).not.toContain("border-line");
  });

  it("Should not render an accent rail when the card is selected", () => {
    const tasks = [buildTask({ id: "sel" })];
    const { container } = render(
      <TasksKanbanBoard
        columns={groupTasksForKanban(tasks)}
        onSelectTask={vi.fn()}
        selectedTaskId="sel"
      />
    );

    const card = screen.getByTestId("tasks-kanban-card-sel");
    expect(card).toHaveAttribute("data-selected", "true");
    expect(card.querySelector("[class*='bg-accent']")).toBeNull();
    // Ensure no accent rail wrapper anywhere inside the card.
    expect(container.querySelectorAll(".bg-\\(--color-accent\\)").length).toBe(0);
  });
});
