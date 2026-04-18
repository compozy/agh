import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
}));

import { TaskRunDetailSessionLink } from "./task-run-detail-session-link";
import type { TaskRunDetailView } from "../types";

function buildRun(session: TaskRunDetailView["session"]): TaskRunDetailView {
  return {
    run: {
      id: "run_7k2m9x",
      task_id: "task_001",
      attempt: 1,
      status: "running",
      queued_at: "2026-04-11T14:00:00Z",
      origin: { kind: "cli", ref: "op" },
    },
    task: {
      id: "task_001",
      identifier: "TASK-42",
      status: "ready",
      scope: "workspace",
      title: "Review",
    },
    summary: { last_activity_at: "2026-04-11T14:00:00Z" },
    session,
  } as unknown as TaskRunDetailView;
}

describe("TaskRunDetailSessionLink", () => {
  it("renders a drill-down link when the run has an attached session", () => {
    render(
      <TaskRunDetailSessionLink
        run={buildRun({
          session_id: "sess_jf8d21",
          agent_name: "Coder",
          workspace_id: "ws_alpha",
          state: "active",
          created_at: "2026-04-11T14:30:00Z",
          updated_at: "2026-04-11T14:40:45Z",
        })}
      />
    );

    expect(screen.getByTestId("task-run-detail-session-link-panel")).toBeInTheDocument();
    expect(screen.getByTestId("task-run-detail-session-id")).toHaveTextContent("sess_jf8d21");
    expect(screen.getByTestId("task-run-detail-session-agent")).toHaveTextContent("Agent Coder");
    expect(screen.getByTestId("task-run-detail-session-drilldown")).toHaveAttribute(
      "data-testid",
      "task-run-detail-session-drilldown"
    );
  });

  it("shows a placeholder when no session is attached", () => {
    render(<TaskRunDetailSessionLink run={buildRun(null)} />);
    expect(screen.getByTestId("task-run-detail-session-none")).toBeInTheDocument();
  });
});
