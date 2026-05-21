import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TaskInspectDiagnosticsCard } from "../task-inspect-diagnostics-card";
import type { TaskInspectView } from "../../types";

function buildInspect(overrides: Partial<TaskInspectView> = {}): TaskInspectView {
  return {
    target: "task",
    task: {
      id: "task_001",
      title: "Review launch blockers",
      status: "in_progress",
      scope: "workspace",
      created_at: "2026-04-17T09:00:00Z",
      updated_at: "2026-04-17T09:00:00Z",
      created_by: { kind: "human", ref: "pedro@" },
      origin: { kind: "web", ref: "op" },
      latest_event_seq: 7,
    },
    current_run: {
      run_id: "run_001",
      task_id: "task_001",
      status: "claimed",
      claim_token_hash_truncated: "abcdef12",
      queued_at: "2026-04-17T09:00:00Z",
      attempt: 2,
      retries: 1,
      heartbeat_age_seconds: 420,
      bound_session_id: "sess_a",
    },
    bound_session: {
      session_id: "sess_a",
      state: "stopped",
      agent_name: "Coder",
      provider_name: "codex",
    },
    scheduler: {
      paused: false,
    },
    diagnostics: [
      {
        id: "task.inspect.task_run_stuck.run_001",
        code: "task_run_stuck",
        severity: "warn",
        category: "task",
        title: "Run heartbeat is stale",
        message: "The claimed run has not reported a heartbeat inside the expected window.",
        suggested_command: 'agh task release run_001 --reason "stale heartbeat"',
        data_freshness: "live",
        evidence: {
          run_id: "run_001",
          heartbeat_age_seconds: 420,
          claim_token_hash_truncated: "abcdef12",
        },
      },
    ],
    next_action: "recovery_required",
    as_of: "2026-04-17T10:00:00Z",
    ...overrides,
  } as TaskInspectView;
}

describe("TaskInspectDiagnosticsCard", () => {
  it("renders inspect diagnostics with suggested recovery command", () => {
    render(<TaskInspectDiagnosticsCard inspect={buildInspect()} />);

    expect(screen.getByTestId("task-inspect-diagnostics-card")).toBeInTheDocument();
    expect(screen.getByTestId("task-inspect-diagnostics-card-next-action")).toHaveTextContent(
      "recovery required"
    );
    expect(screen.getByTestId("task-inspect-diagnostics-card-current-run")).toHaveTextContent(
      "run_001"
    );
    expect(
      screen.getByTestId("task-inspect-diagnostics-card-item-task_run_stuck")
    ).toHaveTextContent("Run heartbeat is stale");
    expect(
      screen.getByTestId("task-inspect-diagnostics-card-item-task_run_stuck-command")
    ).toHaveTextContent("agh task release run_001");
    expect(
      screen.getByTestId("task-inspect-diagnostics-card-item-task_run_stuck-evidence")
    ).toHaveTextContent("abcdef12");
  });

  it("renders an empty diagnostic state for clean inspect snapshots", () => {
    render(<TaskInspectDiagnosticsCard inspect={buildInspect({ diagnostics: [] })} />);

    expect(screen.getByTestId("task-inspect-diagnostics-card-no-diagnostics")).toHaveTextContent(
      "No diagnostics"
    );
  });

  it("renders an error state when inspect cannot load", () => {
    render(<TaskInspectDiagnosticsCard errorMessage="inspect failed" inspect={null} />);

    expect(screen.getByTestId("task-inspect-diagnostics-card-error")).toHaveTextContent(
      "inspect failed"
    );
  });
});
