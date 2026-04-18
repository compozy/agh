import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
}));

import { TasksDetailRunsPanel } from "./tasks-detail-runs-panel";
import type { TaskRun } from "../types";

function buildRun(overrides: Partial<TaskRun> = {}): TaskRun {
  return {
    id: "run_001",
    attempt: 1,
    status: "running",
    queued_at: "2026-04-11T09:00:00Z",
    started_at: "2026-04-11T09:00:30Z",
    task_id: "task_001",
    origin: { kind: "cli", ref: "op" },
    session_id: "sess_123",
    ...overrides,
  } as TaskRun;
}

describe("TasksDetailRunsPanel", () => {
  it("renders loading, error, and empty states", () => {
    const { rerender } = render(<TasksDetailRunsPanel isLoading runs={[]} taskId="task_001" />);
    expect(screen.getByTestId("tasks-detail-runs-loading")).toBeInTheDocument();

    rerender(<TasksDetailRunsPanel errorMessage="boom" runs={[]} taskId="task_001" />);
    expect(screen.getByTestId("tasks-detail-runs-error")).toHaveTextContent("boom");

    rerender(<TasksDetailRunsPanel runs={[]} taskId="task_001" />);
    expect(screen.getByTestId("tasks-detail-runs-empty")).toBeInTheDocument();
  });

  it("renders run rows with deep-link to run detail", () => {
    render(
      <TasksDetailRunsPanel
        runs={[
          buildRun(),
          buildRun({
            id: "run_002",
            status: "failed",
            error: "rate-limited",
            attempt: 2,
            ended_at: "2026-04-11T09:05:00Z",
          }),
        ]}
        taskId="task_001"
      />
    );

    expect(screen.getByTestId("tasks-detail-runs-panel")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-detail-runs-item-run_001")).toHaveTextContent("run_001");
    expect(screen.getByTestId("tasks-detail-runs-item-run_002")).toHaveTextContent("attempt 2");
    expect(screen.getByTestId("tasks-detail-runs-error-run_002")).toHaveTextContent("rate-limited");
    expect(screen.getByTestId("tasks-detail-runs-link-run_001")).toBeInTheDocument();
  });
});
