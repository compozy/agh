import { render, screen } from "@testing-library/react";
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

import {
  TaskRunActivityPanel,
  TaskRunIdentityPanel,
  TaskRunProgressPanel,
} from "../task-run-detail-panels";
import type { TaskRunDetailView } from "../../types";

function buildRun(overrides: Partial<TaskRunDetailView> = {}): TaskRunDetailView {
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
      idempotency_key: "pr-341-review",
      claimed_by: { kind: "agent_session", ref: "Coder" },
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
      last_event_type: "task.run_progress",
      tool_call_count: 4,
      input_tokens: 14281,
      output_tokens: 3046,
      total_tokens: 17327,
      turn_count: 6,
      total_cost: 0.18,
      cost_currency: "USD",
    },
    session: {
      session_id: "sess_jf8d21",
      created_at: "2026-04-11T14:30:00Z",
      updated_at: "2026-04-11T14:40:45Z",
      agent_name: "Coder",
    },
    ...overrides,
  } as unknown as TaskRunDetailView;
}

describe("TaskRunIdentityPanel", () => {
  it("renders run metadata with session drill-down link", () => {
    render(<TaskRunIdentityPanel run={buildRun()} />);
    expect(screen.getByTestId("task-run-detail-identity")).toBeInTheDocument();
    expect(screen.getByTestId("task-run-detail-identity-run")).toHaveTextContent("run_7k2m9x");
    expect(screen.getByTestId("task-run-detail-identity-attempt")).toHaveTextContent("2");
    expect(screen.getByTestId("task-run-detail-identity-idempotency")).toHaveTextContent(
      "pr-341-review"
    );
    expect(screen.getByTestId("task-run-detail-session-link")).toHaveTextContent("sess_jf8d21");
    expect(screen.getByTestId("task-run-detail-session-link")).toHaveAttribute(
      "data-to",
      "/agents/$name/sessions/$id"
    );
  });

  it("links to the session permalink when only the run session id is available", () => {
    render(<TaskRunIdentityPanel run={buildRun({ session: null } as never)} />);
    expect(screen.getByTestId("task-run-detail-session-link")).toHaveTextContent("sess_jf8d21");
    expect(screen.getByTestId("task-run-detail-session-link")).toHaveAttribute(
      "data-to",
      "/session/$id"
    );
  });

  it("falls back to a missing-session indicator when no session is attached", () => {
    const run = buildRun({ session: null } as never);
    render(
      <TaskRunIdentityPanel
        run={{ ...run, run: { ...run.run, session_id: undefined } } as TaskRunDetailView}
      />
    );
    expect(screen.getByTestId("task-run-detail-session-missing")).toBeInTheDocument();
  });
});

describe("TaskRunProgressPanel", () => {
  it("renders tool calls, token counts, elapsed, and cost", () => {
    render(<TaskRunProgressPanel run={buildRun()} />);
    expect(screen.getByTestId("task-run-detail-progress")).toBeInTheDocument();
    expect(screen.getByTestId("task-run-detail-progress-tool-calls")).toHaveTextContent("4");
    expect(screen.getByTestId("task-run-detail-progress-input-tokens")).toHaveTextContent("14,281");
    expect(screen.getByTestId("task-run-detail-progress-output-tokens")).toHaveTextContent("3,046");
    expect(screen.getByTestId("task-run-detail-progress-total-tokens")).toHaveTextContent("17,327");
    expect(screen.getByTestId("task-run-detail-progress-cost")).toHaveTextContent("USD");
  });

  it("shows a dash placeholder when metrics are missing", () => {
    const run = buildRun();
    render(<TaskRunProgressPanel run={{ ...run, summary: {} } as TaskRunDetailView} />);
    expect(screen.getByTestId("task-run-detail-progress-tool-calls")).toHaveTextContent("--");
  });
});

describe("TaskRunActivityPanel", () => {
  it("renders last event, activity timestamp, and error payload when present", () => {
    const run = buildRun();
    const withError = {
      ...run,
      run: { ...run.run, error: "rate_limited" },
      summary: { ...run.summary, last_event_type: "task.run_failed" },
    } as TaskRunDetailView;

    render(<TaskRunActivityPanel run={withError} />);
    expect(screen.getByTestId("task-run-detail-activity")).toBeInTheDocument();
    expect(screen.getByTestId("task-run-detail-activity-event")).toHaveTextContent(
      "task.run_failed"
    );
    expect(screen.getByTestId("task-run-detail-activity-error")).toHaveTextContent("rate_limited");
  });
});
