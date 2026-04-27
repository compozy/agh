import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { SessionPayload } from "@/systems/session";
import { primarySessionFixture } from "@/systems/session/testing";
import { AgentStatsGrid } from "./agent-stats-grid";

function makeSession(overrides: Partial<SessionPayload>): SessionPayload {
  return {
    ...primarySessionFixture,
    ...overrides,
  };
}

describe("AgentStatsGrid", () => {
  it("counts failures with the shared session-status predicate", () => {
    const sessions = [
      makeSession({
        id: "active_failure_metadata",
        state: "active",
        failure: {
          kind: "agent_crashed",
          summary: "stale failure metadata",
        },
      }),
      makeSession({ id: "stopped_error", state: "stopped", stop_reason: "error" }),
      makeSession({
        id: "stopped_crash",
        state: "stopped",
        stop_reason: "agent_crashed",
      }),
      makeSession({ id: "stopped_done", state: "stopped", stop_reason: "completed" }),
    ];

    render(<AgentStatsGrid sessions={sessions} />);

    expect(within(screen.getByTestId("agent-stat-failed")).getByText("2")).toBeInTheDocument();
  });
});
