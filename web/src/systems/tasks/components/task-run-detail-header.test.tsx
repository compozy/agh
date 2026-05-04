import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params, to, ...domRest } = rest as Record<string, unknown>;
    return (
      <a data-params={JSON.stringify(params ?? {})} data-to={String(to ?? "")} {...domRest}>
        {children}
      </a>
    );
  },
}));

import { TaskRunDetailHeader } from "./task-run-detail-header";
import type { TaskRunDetailView } from "../types";

function buildRun(overrides: Partial<TaskRunDetailView["run"]> = {}): TaskRunDetailView {
  return {
    run: {
      id: "run_7k2m9x",
      task_id: "task_001",
      attempt: 2,
      status: "running",
      queued_at: "2026-04-11T14:30:00Z",
      started_at: "2026-04-11T14:37:45Z",
      origin: { kind: "cli", ref: "op" },
      session_id: "sess_jf8d21",
      claimed_by: { kind: "agent_session", ref: "Coder" },
      ...overrides,
    },
    task: {
      id: "task_001",
      identifier: "TASK-42",
      status: "ready",
      scope: "workspace",
      title: "Summarize review feedback",
    },
    summary: {
      last_activity_at: "2026-04-11T14:40:45Z",
    },
    session: null,
  } as unknown as TaskRunDetailView;
}

describe("TaskRunDetailHeader", () => {
  it("renders breadcrumb, title, and run meta", () => {
    render(<TaskRunDetailHeader run={buildRun()} />);
    expect(screen.getByTestId("task-run-detail-breadcrumb")).toHaveTextContent("TASK-42");
    expect(screen.getByTestId("task-run-detail-title")).toHaveTextContent("Run run_7k2m9x");
    expect(screen.getByTestId("task-run-detail-meta")).toHaveTextContent("attempt 2");
    expect(screen.getByTestId("task-run-detail-meta")).toHaveTextContent("session sess_jf8d21");
  });

  it("links to the session permalink when the run lacks hydrated agent metadata", () => {
    render(<TaskRunDetailHeader run={buildRun()} />);
    const link = screen.getByTestId("task-run-detail-open-session");
    expect(link).toHaveAttribute("data-to", "/session/$id");
    expect(link).toHaveAttribute("data-params", JSON.stringify({ id: "sess_jf8d21" }));
  });

  it("links to the canonical agent session route when hydrated metadata is available", () => {
    render(
      <TaskRunDetailHeader
        run={
          {
            ...buildRun(),
            session: {
              session_id: "sess_jf8d21",
              agent_name: "Coder",
            },
          } as unknown as TaskRunDetailView
        }
      />
    );
    const link = screen.getByTestId("task-run-detail-open-session");
    expect(link).toHaveAttribute("data-to", "/agents/$name/sessions/$id");
    expect(link).toHaveAttribute(
      "data-params",
      JSON.stringify({ name: "Coder", id: "sess_jf8d21" })
    );
  });

  it("fires cancel callback when the Kill run button is clicked", () => {
    const onCancelRun = vi.fn();
    render(<TaskRunDetailHeader onCancelRun={onCancelRun} run={buildRun()} />);
    fireEvent.click(screen.getByTestId("task-run-detail-cancel"));
    expect(onCancelRun).toHaveBeenCalledTimes(1);
  });

  it("hides cancel action when the run has already finished", () => {
    render(
      <TaskRunDetailHeader
        onCancelRun={() => {}}
        run={buildRun({ status: "completed", ended_at: "2026-04-11T14:45:00Z" })}
      />
    );
    expect(screen.queryByTestId("task-run-detail-cancel")).not.toBeInTheDocument();
  });
});
