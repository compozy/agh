import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TasksPageShell } from "./tasks-page-shell";

describe("TasksPageShell", () => {
  it("renders the shared shell with the Tasks title, icon, and default count", () => {
    render(
      <TasksPageShell>
        <div data-testid="tasks-shell-content" />
      </TasksPageShell>
    );

    expect(screen.getByRole("heading", { level: 1, name: "Tasks" })).toBeInTheDocument();
    expect(screen.getByTestId("tasks-shell-icon")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-shell-body")).toContainElement(
      screen.getByTestId("tasks-shell-content")
    );
    expect(screen.getByText("0")).toBeInTheDocument();
  });

  it("shows a custom count badge when provided", () => {
    render(
      <TasksPageShell count={12}>
        <div />
      </TasksPageShell>
    );
    expect(screen.getByText("12")).toBeInTheDocument();
  });

  it("renders controls and meta slots when provided", () => {
    render(
      <TasksPageShell
        controls={<button data-testid="tasks-shell-control">control</button>}
        meta={<span data-testid="tasks-shell-meta">meta</span>}
      >
        <div />
      </TasksPageShell>
    );
    expect(screen.getByTestId("tasks-shell-control")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-shell-meta")).toBeInTheDocument();
  });
});
