import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TaskCard } from "../task-card";
import type { TaskListItem } from "../../types";

function buildTask(overrides: Partial<TaskListItem> = {}): TaskListItem {
  return {
    id: "task_001",
    title: "Summarize feedback",
    identifier: "TASK-1",
    status: "in_progress",
    scope: "workspace",
    origin: { kind: "web", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    created_by: { kind: "human", ref: "op" },
    owner: { kind: "agent_session", ref: "Coder" },
    priority: "high",
    child_count: 2,
    dependency_count: 1,
    active_run: {
      id: "run_001",
      task_id: "task_001",
      attempt: 2,
      max_attempts: 3,
      status: "running",
      queued_at: "2026-04-11T09:00:00Z",
    },
    ...overrides,
  } as TaskListItem;
}

describe("TaskCard", () => {
  it("renders enriched task data inline through the meta slot", () => {
    const { container } = render(<TaskCard task={buildTask()} />);

    expect(screen.getByTestId("task-card-task_001")).toBeInTheDocument();
    expect(screen.getByText("task-1")).toBeInTheDocument();
    expect(screen.getByText("Summarize feedback")).toBeInTheDocument();
    expect(screen.getByTestId("task-card-owner-task_001")).toHaveTextContent("Coder");
    expect(screen.getByTestId("task-card-attempt-task_001")).toHaveTextContent("attempt 2 of 3");
    expect(screen.getByTestId("task-card-children-task_001")).toHaveTextContent("2 children");
    expect(screen.getByTestId("task-card-deps-task_001")).toHaveTextContent("1 dep");
    expect(container.querySelector('[data-slot="status-dot"]')).toBeNull();
    // Priority pill stays as a textual pill in the trailing slot.
    expect(screen.getByText("High")).toBeInTheDocument();
  });

  it("invokes onSelect when the card is clicked and reflects selection state", () => {
    const onSelect = vi.fn();
    render(<TaskCard onSelect={onSelect} selected task={buildTask()} />);

    const card = screen.getByTestId("task-card-task_001");
    expect(card).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(card);
    expect(onSelect).toHaveBeenCalledTimes(1);
  });

  it("renders the failed-run error inline in the meta row (no inline retry button)", () => {
    render(
      <TaskCard
        task={buildTask({
          status: "failed",
          active_run: {
            id: "run_002",
            task_id: "task_001",
            attempt: 3,
            max_attempts: 3,
            status: "failed",
            queued_at: "2026-04-11T09:00:00Z",
            error: "rate-limited by upstream",
          },
        })}
      />
    );

    expect(screen.getByTestId("task-card-error-task_001")).toHaveTextContent(
      "rate-limited by upstream"
    );
    // Retry control lives on the detail panel (tasks-detail-header), not the row.
    expect(screen.queryByTestId("task-card-retry-task_001")).not.toBeInTheDocument();
  });

  it("does not render a publish button on draft rows (publish lives on the detail header)", () => {
    render(<TaskCard task={buildTask({ status: "draft", draft: true, active_run: null })} />);
    expect(screen.queryByTestId("task-card-publish-task_001")).not.toBeInTheDocument();
  });

  it("renders a Blocked pill in the trailing slot for blocked tasks", () => {
    render(<TaskCard task={buildTask({ status: "blocked", active_run: null })} />);
    expect(screen.getByTestId("task-card-blocked-task_001")).toBeInTheDocument();
  });
});
