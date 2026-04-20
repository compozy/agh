import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TasksListRow } from "./tasks-list-row";
import type { TaskListItem } from "../types";

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

function getDot(container: HTMLElement): HTMLElement {
  const dot = container.querySelector('[data-slot="status-dot"]');
  expect(dot).not.toBeNull();
  return dot as HTMLElement;
}

describe("TasksListRow", () => {
  it("renders StatusDot with tone=success when task.status is the done equivalent", () => {
    const { container, rerender } = render(
      <TasksListRow task={buildTask({ status: "completed" })} />
    );
    expect(getDot(container)).toHaveAttribute("data-tone", "success");

    // Accepts the mock shorthand ("done") for cross-layer interop.
    rerender(<TasksListRow task={buildTask({ status: "done" as never })} />);
    expect(getDot(container)).toHaveAttribute("data-tone", "success");
  });

  it("renders StatusDot with tone=accent and pulse=true when task.status is the running equivalent", () => {
    const { container, rerender } = render(
      <TasksListRow task={buildTask({ status: "in_progress" })} />
    );
    const dot = getDot(container);
    expect(dot).toHaveAttribute("data-tone", "accent");
    expect(dot).toHaveAttribute("data-pulse", "true");

    rerender(<TasksListRow task={buildTask({ status: "running" as never })} />);
    const dot2 = getDot(container);
    expect(dot2).toHaveAttribute("data-tone", "accent");
    expect(dot2).toHaveAttribute("data-pulse", "true");
  });

  it("renders tone=warning for blocked, tone=danger for failed, tone=info for pending", () => {
    const { container, rerender } = render(
      <TasksListRow task={buildTask({ status: "blocked" })} />
    );
    expect(getDot(container)).toHaveAttribute("data-tone", "warning");

    rerender(<TasksListRow task={buildTask({ status: "failed" })} />);
    expect(getDot(container)).toHaveAttribute("data-tone", "danger");

    rerender(<TasksListRow task={buildTask({ status: "pending" })} />);
    expect(getDot(container)).toHaveAttribute("data-tone", "info");
  });

  it("renders a MonoBadge showing the identifier when present", () => {
    render(<TasksListRow task={buildTask({ identifier: "TASK-42" })} />);
    const badge = screen.getByText("TASK-42");
    expect(badge).toHaveAttribute("data-slot", "mono-badge");
  });

  it("falls back to the 7-character short id when the identifier is absent", () => {
    render(<TasksListRow task={buildTask({ identifier: undefined })} />);
    // id = "task_abcdef0_tail" → short id "task_ab"
    const badge = screen.getByText("task_ab");
    expect(badge).toHaveAttribute("data-slot", "mono-badge");
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
