import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TasksListPanel } from "./tasks-list-panel";
import type { TaskListItem } from "../types";

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

describe("TasksListPanel", () => {
  it("renders the headline, total count, and one row per task", () => {
    const tasks = [
      buildTask({ id: "a", title: "First", identifier: "TASK-1" }),
      buildTask({ id: "b", title: "Second", identifier: "TASK-2" }),
    ];

    render(
      <TasksListPanel
        onSearchChange={vi.fn()}
        onSelectTask={vi.fn()}
        searchQuery=""
        selectedTaskId={null}
        statusFilter="ready"
        tasks={tasks}
        totalCount={5}
      />
    );

    expect(screen.getByTestId("tasks-list-headline").textContent ?? "").toMatch(/Ready/);
    expect(screen.getByTestId("tasks-list-total")).toHaveTextContent("5 total");
    expect(screen.getByTestId("task-card-a")).toBeInTheDocument();
    expect(screen.getByTestId("task-card-b")).toBeInTheDocument();
  });

  it("forwards search query changes and selection events", () => {
    const onSearchChange = vi.fn();
    const onSelectTask = vi.fn();

    render(
      <TasksListPanel
        onSearchChange={onSearchChange}
        onSelectTask={onSelectTask}
        searchQuery=""
        selectedTaskId={null}
        tasks={[buildTask()]}
        totalCount={1}
      />
    );

    const search = screen.getByTestId("tasks-list-search-input") as HTMLInputElement;
    fireEvent.change(search, { target: { value: "client" } });
    expect(onSearchChange).toHaveBeenCalledWith("client");

    fireEvent.click(screen.getByTestId("task-card-task_001"));
    expect(onSelectTask).toHaveBeenCalledWith("task_001");
  });

  it("renders the lane switcher with All / Mine / Watched and emits lane changes", () => {
    const onLaneChange = vi.fn();
    render(
      <TasksListPanel
        lane="all"
        laneBadges={{ mine: 2 }}
        onLaneChange={onLaneChange}
        onSearchChange={vi.fn()}
        onSelectTask={vi.fn()}
        searchQuery=""
        selectedTaskId={null}
        tasks={[buildTask()]}
        totalCount={1}
      />
    );

    const laneAll = screen.getByTestId("tasks-list-lane-all");
    const laneMine = screen.getByTestId("tasks-list-lane-mine");
    const laneWatched = screen.getByTestId("tasks-list-lane-watched");

    expect(laneAll).toHaveAttribute("aria-selected", "true");
    expect(laneMine).toHaveAttribute("aria-selected", "false");
    expect(laneWatched).toHaveAttribute("aria-selected", "false");

    fireEvent.click(laneMine);
    expect(onLaneChange).toHaveBeenCalledWith("mine");
  });

  it("renders loading, error, and empty states cleanly", () => {
    const { rerender } = render(
      <TasksListPanel
        isLoading
        onSearchChange={vi.fn()}
        onSelectTask={vi.fn()}
        searchQuery=""
        selectedTaskId={null}
        tasks={[]}
        totalCount={0}
      />
    );
    expect(screen.getByTestId("tasks-list-loading")).toBeInTheDocument();

    rerender(
      <TasksListPanel
        errorMessage="boom"
        onSearchChange={vi.fn()}
        onSelectTask={vi.fn()}
        searchQuery=""
        selectedTaskId={null}
        tasks={[]}
        totalCount={0}
      />
    );
    expect(screen.getByTestId("tasks-list-error")).toHaveTextContent("boom");

    rerender(
      <TasksListPanel
        onSearchChange={vi.fn()}
        onSelectTask={vi.fn()}
        searchQuery=""
        selectedTaskId={null}
        tasks={[]}
        totalCount={0}
      />
    );
    expect(screen.getByTestId("tasks-list-empty")).toBeInTheDocument();
  });
});
