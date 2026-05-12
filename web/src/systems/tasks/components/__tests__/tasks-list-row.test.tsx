import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TasksListRow } from "../tasks-list-row";
import type { TaskListItem } from "../../types";

function buildTask(overrides: Partial<TaskListItem> = {}): TaskListItem {
  return {
    id: "task_abcdef0_tail",
    title: "Summarize feedback",
    identifier: "TASK-1",
    status: "in_progress",
    scope: "workspace",
    origin: { kind: "web", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    created_by: { kind: "human", ref: "op" },
    ...overrides,
  } as TaskListItem;
}

function queryDot(container: HTMLElement): HTMLElement | null {
  return container.querySelector('[data-slot="status-dot"]');
}

function getDot(container: HTMLElement): HTMLElement {
  const dot = queryDot(container);
  expect(dot).not.toBeNull();
  return dot as HTMLElement;
}

describe("TasksListRow", () => {
  it("reserves the dot column without decoration for terminal + normal statuses", () => {
    const { container, rerender } = render(
      <TasksListRow task={buildTask({ status: "completed" })} />
    );
    expect(queryDot(container)).toBeNull();

    rerender(<TasksListRow task={buildTask({ status: "ready" })} />);
    expect(queryDot(container)).toBeNull();

    rerender(<TasksListRow task={buildTask({ status: "pending" })} />);
    expect(queryDot(container)).toBeNull();

    // Accepts the mock shorthand ("done") for cross-layer interop.
    rerender(<TasksListRow task={buildTask({ status: "done" as never })} />);
    expect(queryDot(container)).toBeNull();
  });

  it("renders StatusDot with tone=accent and ring variant when task.status is the running equivalent", () => {
    const { container, rerender } = render(
      <TasksListRow task={buildTask({ status: "in_progress" })} />
    );
    const dot = getDot(container);
    expect(dot).toHaveAttribute("data-tone", "accent");
    expect(dot).toHaveAttribute("data-variant", "ring");

    rerender(<TasksListRow task={buildTask({ status: "running" as never })} />);
    const dot2 = getDot(container);
    expect(dot2).toHaveAttribute("data-tone", "accent");
    expect(dot2).toHaveAttribute("data-variant", "ring");
  });

  it("renders attention-demanding tones only for statuses that actually demand attention", () => {
    const { container, rerender } = render(
      <TasksListRow task={buildTask({ status: "blocked" })} />
    );
    expect(getDot(container)).toHaveAttribute("data-tone", "warning");

    rerender(<TasksListRow task={buildTask({ status: "failed" })} />);
    expect(getDot(container)).toHaveAttribute("data-tone", "danger");

    rerender(<TasksListRow task={buildTask({ status: "canceled" })} />);
    expect(getDot(container)).toHaveAttribute("data-tone", "danger");
  });

  it("renders the identifier as bare mono text (proposal `.task-row__id`, not a Pill)", () => {
    render(<TasksListRow task={buildTask({ identifier: "TASK-42" })} />);
    const id = screen.getByText("task-42").closest('[data-slot="tasks-list-row-id"]');
    expect(id).not.toBeNull();
    expect(id).toHaveAttribute("data-slot", "tasks-list-row-id");
  });

  it("falls back to the 7-character short id when the identifier is absent", () => {
    render(<TasksListRow task={buildTask({ identifier: undefined })} />);
    // id = "task_abcdef0_tail" → short id "task_ab"
    const id = screen.getByText("task_ab").closest('[data-slot="tasks-list-row-id"]');
    expect(id).not.toBeNull();
    expect(id).toHaveAttribute("data-slot", "tasks-list-row-id");
  });

  it("invokes onSelect(task.id) when the row is clicked", () => {
    const onSelect = vi.fn();
    render(<TasksListRow onSelect={onSelect} task={buildTask({ id: "task_xyz" })} />);

    fireEvent.click(screen.getByTestId("task-card-task_xyz"));
    expect(onSelect).toHaveBeenCalledWith("task_xyz");
  });

  it("shows a lane pill when lane is provided", () => {
    render(<TasksListRow lane="approvals" task={buildTask()} />);
    expect(screen.getByText("Approvals")).toBeInTheDocument();
  });

  it("reflects selection state via aria-pressed + data-selected", () => {
    render(<TasksListRow selected task={buildTask()} />);
    const row = screen.getByTestId("task-card-task_abcdef0_tail");
    expect(row).toHaveAttribute("aria-pressed", "true");
    expect(row).toHaveAttribute("data-selected", "true");
  });
});
