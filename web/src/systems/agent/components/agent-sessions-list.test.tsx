import { render, screen, within } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { SessionPayload } from "@/systems/session";
import { primarySessionFixture } from "@/systems/session/testing";
import { AgentSessionsList } from "./agent-sessions-list";

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    to,
    params,
    ...props
  }: {
    children: ReactNode;
    to: string;
    params?: Record<string, string>;
    [key: string]: unknown;
  }) => {
    const href = params
      ? Object.entries(params).reduce((acc, [key, value]) => acc.replace(`$${key}`, value), to)
      : to;
    return (
      <a href={href} {...props}>
        {children}
      </a>
    );
  },
}));

function makeSession(overrides: Partial<SessionPayload>): SessionPayload {
  return {
    ...primarySessionFixture,
    ...overrides,
  };
}

describe("AgentSessionsList", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("formats relative times against one render-pass timestamp", () => {
    vi.spyOn(Date, "now").mockReturnValue(Date.parse("2026-04-17T18:11:00Z"));
    const sessions = [
      makeSession({
        id: "sess_one",
        updated_at: "2026-04-17T18:10:30Z",
        activity: {
          elapsed_ms: 60_000,
          elapsed_seconds: 60,
          idle_seconds: 0,
          iteration_current: 1,
          iteration_max: 2,
          last_activity_at: "2026-04-17T18:10:30Z",
        },
      }),
      makeSession({
        id: "sess_two",
        updated_at: "2026-04-17T18:10:30Z",
        activity: {
          elapsed_ms: 60_000,
          elapsed_seconds: 60,
          idle_seconds: 0,
          iteration_current: 1,
          iteration_max: 2,
          last_activity_at: "2026-04-17T18:10:30Z",
        },
      }),
    ];

    render(
      <AgentSessionsList
        agentName="codex-agent"
        sessions={sessions}
        isLoading={false}
        isError={false}
      />
    );

    expect(
      within(screen.getByTestId("agent-session-row-sess_one")).getByText("just now")
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("agent-session-row-sess_two")).getByText("just now")
    ).toBeInTheDocument();
    expect(Date.now).toHaveBeenCalledTimes(1);
  });

  it("renders zero elapsed duration as zero seconds", () => {
    const sessions = [
      makeSession({
        id: "sess_zero_duration",
        activity: {
          elapsed_ms: 0,
          elapsed_seconds: 0,
          idle_seconds: 0,
          iteration_current: 0,
          iteration_max: 4,
          last_activity_at: "2026-04-17T18:10:30Z",
        },
      }),
    ];

    render(
      <AgentSessionsList
        agentName="codex-agent"
        sessions={sessions}
        isLoading={false}
        isError={false}
      />
    );

    expect(
      within(screen.getByTestId("agent-session-row-sess_zero_duration")).getByText("0s")
    ).toBeInTheDocument();
  });
});
