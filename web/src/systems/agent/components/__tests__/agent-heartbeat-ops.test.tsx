import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import type { SessionPayload } from "@/systems/session";
import { primarySessionFixture } from "@/systems/session/testing";

import type { AgentHeartbeatStatusPayload } from "../../types";
import { AgentHeartbeatOps } from "../agent-heartbeat-ops";

function status(eligible: boolean): AgentHeartbeatStatusPayload {
  return {
    active: true,
    agent_name: "coder",
    enabled: true,
    present: true,
    valid: true,
    validation_status: "valid",
    preferences: { context: {}, min_interval: "30m" },
    session_health: {
      active_prompt: false,
      agent_name: "coder",
      attachable: true,
      eligible_for_wake: eligible,
      health: eligible ? "healthy" : "stale",
      ineligibility_reason: eligible ? undefined : "session_health_stale",
      session_id: "sess-1",
      state: "idle",
      updated_at: "2026-07-11T12:00:00Z",
      workspace_id: "ws-test",
    },
    wake_events: [
      {
        created_at: "2026-07-11T11:00:00Z",
        expires_at: "2026-07-12T11:00:00Z",
        id: "wake-1",
        reason: "wake_sent",
        result: "sent",
        source: "manual",
      },
    ],
  };
}

function activeSession(id = "sess-1"): SessionPayload {
  return { ...primarySessionFixture, id, name: "Release check", state: "active" };
}

function Harness({ eligible = true }: { eligible?: boolean }) {
  const [selected, setSelected] = useState<string | null>("sess-1");
  return (
    <AgentHeartbeatOps
      status={status(eligible)}
      statusLoading={false}
      statusError={false}
      onRetryStatus={vi.fn()}
      activeSessions={[activeSession()]}
      selectedSessionId={selected}
      onSelectSessionId={setSelected}
      onWake={mockWake}
      waking={false}
      wakeDecision={undefined}
      wakeError={null}
      onNewSession={vi.fn()}
    />
  );
}

const mockWake = vi.fn();

describe("AgentHeartbeatOps", () => {
  it("Should wake the selected active session only when health says it is eligible", async () => {
    const user = userEvent.setup();
    mockWake.mockReset();
    render(<Harness />);

    await waitFor(() =>
      expect(screen.getByTestId("agent-heartbeat-wake")).toHaveAttribute("aria-disabled", "false")
    );
    expect(screen.getByText("sent · wake_sent")).toBeInTheDocument();
    await user.click(screen.getByTestId("agent-heartbeat-wake"));
    expect(mockWake).toHaveBeenCalledWith("sess-1");
  });

  it("Should keep Wake now focusable but inert with a truthful reason when session health blocks it", async () => {
    const user = userEvent.setup();
    mockWake.mockReset();
    render(<Harness eligible={false} />);

    await waitFor(() => expect(screen.getByTestId("agent-heartbeat-ineligible")).toBeVisible());
    const wake = screen.getByTestId("agent-heartbeat-wake");
    expect(wake).toHaveAttribute("aria-disabled", "true");
    expect(wake).not.toBeDisabled();
    await user.click(wake);
    expect(mockWake).not.toHaveBeenCalled();
  });

  it("Should expose recovery when heartbeat status fails", async () => {
    const user = userEvent.setup();
    const retry = vi.fn();
    render(
      <AgentHeartbeatOps
        status={undefined}
        statusLoading={false}
        statusError
        onRetryStatus={retry}
        activeSessions={[]}
        selectedSessionId={null}
        onSelectSessionId={vi.fn()}
        onWake={vi.fn()}
        waking={false}
        wakeDecision={undefined}
        wakeError={null}
        onNewSession={vi.fn()}
      />
    );

    expect(screen.getByTestId("agent-heartbeat-status-error")).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(retry).toHaveBeenCalledTimes(1);
  });
});
